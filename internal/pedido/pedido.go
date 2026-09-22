// Package pedido é dono de Pedido, Item de Pedido, máquina de estados, Frete e varredura.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1), junto com maquina.go, que guarda a tabela de transições do
// AD-3 e o Transicionar, e frete.go, que guarda a Regra de Frete (AD-17).
// Aqui moram o nascimento do Pedido a partir do Carrinho, as leituras e os dois passos da varredura — aplicar a
// confirmação e simular a entrega até ENTREGUE.
package pedido

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/carrinho"
	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/identidade"
	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/Cashnip/amazon-waddle/internal/pedido/db/gerado"
)

// simulacao é a lista inteira de avanços que a entrega simulada conhece, e é
// uma só: dela saem tanto o próximo Status quanto os candidatos que a consulta
// procura. Duas listas seriam duas oportunidades de divergirem.
var simulacao = map[Status]Status{
	StatusPago:      StatusSeparando,
	StatusSeparando: StatusEnviado,
	StatusEnviado:   StatusEntregue,
}

// proximoDaSimulacao devolve para onde a simulação leva este Status, e false
// para todo Status que ela não move — AGUARDANDO_PAGAMENTO, PAGAMENTO_RECUSADO,
// CANCELADO e o próprio ENTREGUE, independentemente do tempo decorrido.
func proximoDaSimulacao(status Status) (Status, bool) {
	proximo, ok := simulacao[status]
	return proximo, ok
}

// Pedido é o que a criação devolve e as leituras mostram.
// O total é coluna, nunca derivação de leitura (AD-3).
type Pedido struct {
	ID            string
	Numero        string
	Status        Status
	TotalCentavos int64
	// AtualizadoEm é o instante da última transição, e só a leitura o
	// preenche: é ele que a tela de acompanhamento exibe, absoluto e vindo do
	// servidor — o navegador nunca conta duração.
	AtualizadoEm time.Time
}

// ErrTotalDivergente recusa a criação cujo total não é mais o que a Revisão
// exibiu: um preço ou o Frete mudou entre a Revisão e a confirmação. Conferido
// duas vezes — antes do INSERT e de novo com os Produtos travados (AD-5) —, e a
// recusa não grava nada, nem prende a chave de idempotência.
var ErrTotalDivergente = errors.New("O total do Pedido mudou desde a Revisão. Confira os valores e confirme de novo.")

// ErrCarrinhoVazio recusa a criação sem Item de Carrinho nenhum: não há Pedido
// a fechar.
var ErrCarrinhoVazio = errors.New("O Carrinho está vazio.")

// ErrChaveReutilizada recusa a `Idempotency-Key` que já criou um Pedido deste
// Comprador com outro corpo (AD-7). Criar devolve esse Pedido original junto
// com o erro: é ele que a tela mostra. A chave só se prende quando o Pedido
// existe — recusa por Estoque, por total ou por Carrinho desfaz a transação e
// a deixa livre.
var ErrChaveReutilizada = errors.New("Esta confirmação já criou um Pedido.")

// NovoPedido é o corpo da confirmação: o Endereço escolhido e o total que a
// Revisão exibiu, mais a chave de idempotência do cabeçalho. Itens, preços e
// Frete não vêm do navegador — vêm do Carrinho e da Regra de Frete (NFR-13).
type NovoPedido struct {
	EnderecoID    string
	TotalCentavos int64
	Chave         string
}

// digestDe é a impressão do corpo que a chave protege: mesma chave com o mesmo
// digest é reenvio, com outro digest é chave reaproveitada. O Endereço entra
// em minúscula porque o uuid em maiúscula é o mesmo Endereço.
func digestDe(novo NovoPedido) string {
	soma := sha256.Sum256([]byte(strings.ToLower(novo.EnderecoID) + "\n" + strconv.FormatInt(novo.TotalCentavos, 10)))
	return hex.EncodeToString(soma[:])
}

// Criar faz o Pedido nascer em AGUARDANDO_PAGAMENTO a partir do Carrinho do
// Comprador, dentro da transação que `api/` abriu (AD-4). O `bool` diz
// reenvio: a chave já tinha criado um Pedido com o mesmo corpo, e o Pedido
// devolvido é o original — quem chama desfaz a transação e responde 200.
//
// **A ordem é o contrato**, e cada passo existe por causa do seguinte:
//
//  1. a chave já usada por este Comprador decide antes de tudo: mesmo digest
//     é reenvio, digest diferente é ErrChaveReutilizada (com o original);
//  2. Endereço do dono, Carrinho, Frete e total — ErrTotalDivergente se o
//     total já difere do que a Revisão exibiu;
//  3. o INSERT do Pedido com a chave, que é a reivindicação dela: o gêmeo do
//     duplo clique espera antes de tocar Estoque, e cai no passo 1 quando o
//     primeiro comita;
//  4. catalogo.Reservar, tudo ou nada, sob a trava `ORDER BY id FOR UPDATE`;
//  5. **só então** a releitura dos preços, com os Produtos travados, e o
//     Frete recalculado sobre eles (a Regra de Frete não é travada: só muda
//     por migração) — o total foi ao INSERT calculado antes da trava, e é aqui
//     que se confirma que nada mudou; o gatilho da 5.1 impede corrigir depois;
//  6. Itens de Pedido (preço praticado e Vendedor congelados), a primeira
//     linha do histórico, carrinho.Esvaziar sobre os Itens lidos e a Tentativa.
//
// Endereço de outro Comprador, inexistente ou malformado sai como
// pgx.ErrNoRows, que `api/` traduz no mesmo 404.
func Criar(ctx context.Context, tx pgx.Tx, compradorID string, novo NovoPedido, isencaoCentavos int64) (Pedido, bool, error) {
	var comprador, chave pgtype.UUID
	if err := comprador.Scan(compradorID); err != nil {
		return Pedido{}, false, fmt.Errorf("identificador de Comprador inválido: %w", err)
	}
	if err := chave.Scan(novo.Chave); err != nil {
		return Pedido{}, false, fmt.Errorf("chave de idempotência inválida: %w", err)
	}
	digest := digestDe(novo)
	q := gerado.New(tx)

	// Passo 1. `decidirPelaChave` é chamado de novo em toda recusa do passo 2
	// que depende do Carrinho — vazio, só de invisíveis, total divergente — e
	// no conflito do passo 3: entre a primeira leitura e a recusa, o gêmeo pode
	// ter comitado e esvaziado o Carrinho, e a resposta certa para ele é o
	// Pedido original, nunca CARRINHO_VAZIO. O 404 do Endereço não passa por
	// aqui: o gêmeo não apaga Endereço, então a mesma chave e o mesmo corpo não
	// têm como virar Endereço alheio entre as duas leituras.
	if p, achou, err := decidirPelaChave(ctx, q, comprador, chave, digest); achou || err != nil {
		return p, achou && err == nil, err
	}
	recusar := func(motivo error) (Pedido, bool, error) {
		if p, achou, err := decidirPelaChave(ctx, q, comprador, chave, digest); achou || err != nil {
			return p, achou && err == nil, err
		}
		return Pedido{}, false, motivo
	}

	// Passo 2.
	endereco, err := identidade.BuscarEndereco(ctx, tx, novo.EnderecoID, compradorID)
	if err != nil {
		return Pedido{}, false, err
	}
	conteudo, err := carrinho.Itens(ctx, tx, compradorID)
	if err != nil {
		return Pedido{}, false, err
	}
	if len(conteudo.Itens) == 0 {
		return recusar(ErrCarrinhoVazio)
	}
	// Carrinho só de Produtos invisíveis: não há subtotal, e o Pedido sem
	// parcela de Produto seria recusado pelo CHECK do banco num 500. É a mesma
	// resposta que Reservar daria para o primeiro deles.
	if conteudo.SubtotalCentavos == 0 {
		return recusar(catalogo.EstoqueInsuficiente{ProdutoID: conteudo.Itens[0].ProdutoID})
	}
	faixas, err := faixasDeFrete(ctx, tx)
	if err != nil {
		return Pedido{}, false, err
	}
	frete, err := CalcularFrete(faixas, endereco.CEP, conteudo.SubtotalCentavos, isencaoCentavos)
	if err != nil {
		return Pedido{}, false, err
	}
	total := conteudo.SubtotalCentavos + frete.ValorCentavos
	if total != novo.TotalCentavos {
		return recusar(ErrTotalDivergente)
	}

	// Passo 3. O contador por ano vem antes, porque o número vai no INSERT; é
	// ele, na prática, o primeiro ponto de espera de dois Pedidos do mesmo ano
	// — o gêmeo espera ali, e o índice da chave o recusa em seguida.
	ano := time.Now().UTC().Year()
	sequencial, err := q.ProximoNumeroDoAno(ctx, int32(ano))
	if err != nil {
		return Pedido{}, false, fmt.Errorf("reservar o número do Pedido: %w", err)
	}
	texto := func(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }
	linha, err := q.CriarPedido(ctx, gerado.CriarPedidoParams{
		Numero:               fmt.Sprintf("AZ-%d-%06d", ano, sequencial),
		CompradorID:          comprador,
		Status:               string(StatusAguardandoPagamento),
		SubtotalCentavos:     conteudo.SubtotalCentavos,
		FreteCentavos:        frete.ValorCentavos,
		TotalCentavos:        total,
		EnderecoDestinatario: texto(endereco.Destinatario),
		EnderecoCep:          texto(endereco.CEP),
		EnderecoLogradouro:   texto(endereco.Logradouro),
		EnderecoNumero:       texto(endereco.Numero),
		EnderecoComplemento:  texto(endereco.Complemento),
		EnderecoBairro:       texto(endereco.Bairro),
		EnderecoCidade:       texto(endereco.Cidade),
		EnderecoUf:           texto(endereco.UF),
		ChaveIdempotencia:    chave,
		DigestCorpo:          texto(digest),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// O DO NOTHING: o gêmeo comitou o Pedido desta chave enquanto este
		// esperava. A releitura o encontra — é um comando novo, e em READ
		// COMMITTED ele já vê o que o outro gravou.
		if p, achou, err := decidirPelaChave(ctx, q, comprador, chave, digest); achou || err != nil {
			return p, achou && err == nil, err
		}
		return Pedido{}, false, errors.New("criar o Pedido: a chave colidiu e o Pedido dela não foi encontrado")
	}
	if err != nil {
		return Pedido{}, false, fmt.Errorf("criar o Pedido: %w", err)
	}
	pedido := Pedido{
		ID:            linha.ID.String(),
		Numero:        linha.Numero,
		Status:        Status(linha.Status),
		TotalCentavos: linha.TotalCentavos,
	}

	// Passo 4. Todos os Itens, inclusive o de Produto invisível: é Reservar
	// quem o recusa, sob a trava, com EstoqueInsuficiente de disponível zero.
	reserva := make([]catalogo.ItemReserva, 0, len(conteudo.Itens))
	lidos := make([]string, 0, len(conteudo.Itens))
	for _, item := range conteudo.Itens {
		reserva = append(reserva, catalogo.ItemReserva{ProdutoID: item.ProdutoID, Quantidade: item.Quantidade})
		lidos = append(lidos, item.ID)
	}
	if err := catalogo.Reservar(ctx, tx, pedido.ID, reserva); err != nil {
		return Pedido{}, false, err
	}

	// Passo 5. Com os Produtos travados, o preço não muda até o Commit: o
	// UPDATE do Administrador espera pela mesma linha. O que se lê aqui é o que
	// o Item de Pedido congela. A Regra de Frete é relida, mas **não** travada:
	// pedido.faixa_frete só muda por migração, e a releitura existe para o
	// Frete ser recalculado sobre o subtotal sob a trava, não para protegê-la.
	produtos := make([]catalogo.Produto, 0, len(conteudo.Itens))
	var subtotal int64
	for _, item := range conteudo.Itens {
		produto, err := catalogo.BuscarProduto(ctx, tx, item.ProdutoID)
		if errors.Is(err, pgx.ErrNoRows) {
			// O Vendedor desativado não trava a linha do Produto: sumiu da
			// visibilidade entre a Reserva e aqui, e é a mesma falta.
			return Pedido{}, false, catalogo.EstoqueInsuficiente{ProdutoID: item.ProdutoID}
		}
		if err != nil {
			return Pedido{}, false, err
		}
		produtos = append(produtos, produto)
		// Multiplicação de inteiros: não existe divisão no caminho monetário (AD-9).
		subtotal += produto.PrecoCentavos * int64(item.Quantidade)
	}
	faixas, err = faixasDeFrete(ctx, tx)
	if err != nil {
		return Pedido{}, false, err
	}
	freteSobTrava, err := CalcularFrete(faixas, endereco.CEP, subtotal, isencaoCentavos)
	if err != nil {
		return Pedido{}, false, err
	}
	// As duas parcelas, e não só o total: um preço que cai e cruza o limiar de
	// isenção pode fechar o mesmo total com outra divisão, e o Pedido gravado
	// mentiria sobre o subtotal.
	if subtotal != conteudo.SubtotalCentavos || freteSobTrava.ValorCentavos != frete.ValorCentavos {
		return Pedido{}, false, ErrTotalDivergente
	}

	// Passo 6. Os Itens antes do histórico: depois da primeira transição o
	// banco recusa Item novo (gatilho da 5.6).
	for i, item := range conteudo.Itens {
		var produtoChave pgtype.UUID
		_ = produtoChave.Scan(produtos[i].ID) // veio do banco, já é canônico
		if err := q.CriarItemPedido(ctx, gerado.CriarItemPedidoParams{
			PedidoID:               linha.ID,
			ProdutoID:              produtoChave,
			Nome:                   produtos[i].Nome,
			VendedorNome:           produtos[i].VendedorNome,
			PrecoPraticadoCentavos: produtos[i].PrecoCentavos,
			Quantidade:             item.Quantidade,
		}); err != nil {
			return Pedido{}, false, fmt.Errorf("criar o Item do Pedido: %w", err)
		}
	}

	// A primeira linha do histórico (NFR-9), que é a primeira linha da tabela
	// do AD-3. O estado anterior é vazio porque não havia estado antes — é o
	// nascimento, e não um avanço, e por isso não passa pelo CAS.
	if err := q.RegistrarTransicao(ctx, gerado.RegistrarTransicaoParams{
		PedidoID:       linha.ID,
		StatusAnterior: "",
		StatusNovo:     string(StatusAguardandoPagamento),
		Ator:           string(AtorComprador),
	}); err != nil {
		return Pedido{}, false, fmt.Errorf("registrar a transição: %w", err)
	}

	// Só os Itens lidos: o que outra aba acrescentou depois fica no Carrinho.
	if err := carrinho.Esvaziar(ctx, tx, compradorID, lidos); err != nil {
		return Pedido{}, false, err
	}

	// A Tentativa nasce na mesma transação (AD-7): o Pedido revertido não
	// deixa Tentativa órfã. O total vai como valor — `pagamento` não consulta
	// `pedido`.
	if err := pagamento.IniciarTentativa(ctx, tx, pedido.ID, pedido.TotalCentavos); err != nil {
		return Pedido{}, false, err
	}
	return pedido, false, nil
}

// decidirPelaChave é o passo 1 de Criar. `achou` falso é chave livre; `achou`
// verdadeiro devolve o Pedido original, com ErrChaveReutilizada quando o corpo
// difere.
func decidirPelaChave(ctx context.Context, q *gerado.Queries, comprador, chave pgtype.UUID, digest string) (Pedido, bool, error) {
	linha, err := q.PedidoPorChave(ctx, gerado.PedidoPorChaveParams{CompradorID: comprador, ChaveIdempotencia: chave})
	if errors.Is(err, pgx.ErrNoRows) {
		return Pedido{}, false, nil
	}
	if err != nil {
		return Pedido{}, false, fmt.Errorf("ler o Pedido da chave: %w", err)
	}
	original := Pedido{
		ID:            linha.ID.String(),
		Numero:        linha.Numero,
		Status:        Status(linha.Status),
		TotalCentavos: linha.TotalCentavos,
	}
	if linha.DigestCorpo.String != digest {
		return original, true, ErrChaveReutilizada
	}
	return original, true, nil
}

// Endereco é o Endereço congelado no Pedido na criação (5.6): cópia, e não
// referência — o Endereço do Comprador pode mudar ou sumir depois.
type Endereco struct {
	Destinatario, CEP, Logradouro, Numero, Complemento, Bairro, Cidade, UF string
}

// ItemDoPedido é o Item de Pedido como a tela o mostra: o que foi congelado na
// compra, mais `Disponivel`, que diz se o Estoque disponível de agora cobre a
// quantidade. Enquanto a Reserva do próprio Pedido está ativa ela conta contra
// ele; só `PAGAMENTO_RECUSADO`, com a Reserva já liberada, usa o campo para
// derivar ação (EXPERIENCE, a tripla).
type ItemDoPedido struct {
	ProdutoID              string
	Nome                   string
	Quantidade             int32
	PrecoPraticadoCentavos int64
	Disponivel             bool
}

// Detalhe é tudo que a tela do Pedido deriva, numa leitura só (AD-18): a
// tripla (Status, tentativas restantes, disponível por Item), o instante de
// expiração, os valores e o Endereço congelados, e o histórico. Nenhum campo
// nomeia a Reserva de Estoque.
type Detalhe struct {
	Pedido
	SubtotalCentavos int64
	FreteCentavos    int64
	// Endereco é nil no Pedido do esqueleto, que nasceu sem Endereço.
	Endereco *Endereco
	// ExpiraEm é nil fora de AGUARDANDO_PAGAMENTO: só a Tentativa aberta tem
	// prazo correndo.
	ExpiraEm            *time.Time
	TentativasRestantes int
	Itens               []ItemDoPedido
	Historico           []Transicao
}

// ExpiraEm é o instante em que a Tentativa aberta vence: a última transição
// *para* AGUARDANDO_PAGAMENTO mais o prazo. Sai do histórico, e não de relógio
// em memória (AD-6) — reiniciar o contêiner ou recarregar a tela não recomeça
// a contagem. A nova Tentativa (5.10) volta a AGUARDANDO_PAGAMENTO e, com
// isso, recomeça o prazo sem código novo aqui.
//
// Fora de AGUARDANDO_PAGAMENTO não há prazo: devolve false.
func ExpiraEm(status Status, historico []Transicao, prazo time.Duration) (time.Time, bool) {
	if status != StatusAguardandoPagamento {
		return time.Time{}, false
	}
	for i := len(historico) - 1; i >= 0; i-- {
		if historico[i].Para == StatusAguardandoPagamento {
			return historico[i].Em.Add(prazo), true
		}
	}
	return time.Time{}, false
}

// Detalhar é a leitura da tela do Pedido. O dono entra no WHERE: Pedido de
// outro Comprador, Pedido inexistente e identificador malformado saem os três
// como pgx.ErrNoRows, que `api/` traduz no mesmo 404 — não vaza existência.
//
// A tripla é montada aqui, a partir de `pagamento` (tentativas restantes, cujo
// teto é dele — AD-8) e de `catalogo` (disponível), para que a tela nunca faça
// mais de uma chamada para derivar uma ação (AD-18). São várias consultas: quem
// quer que elas vejam o mesmo instante passa uma transação em `bd`.
func Detalhar(ctx context.Context, bd gerado.DBTX, pedidoID, compradorID string, prazo time.Duration, teto int) (Detalhe, error) {
	var chave, comprador pgtype.UUID
	if err := chave.Scan(pedidoID); err != nil {
		return Detalhe{}, pgx.ErrNoRows
	}
	if err := comprador.Scan(compradorID); err != nil {
		return Detalhe{}, pgx.ErrNoRows
	}
	q := gerado.New(bd)
	linha, err := q.BuscarPedidoDoComprador(ctx, gerado.BuscarPedidoDoCompradorParams{
		PedidoID:    chave,
		CompradorID: comprador,
	})
	if err != nil {
		return Detalhe{}, err
	}
	d := Detalhe{
		Pedido: Pedido{
			ID:            linha.ID.String(),
			Numero:        linha.Numero,
			Status:        Status(linha.Status),
			TotalCentavos: linha.TotalCentavos,
			AtualizadoEm:  linha.AtualizadoEm.Time.UTC(),
		},
		SubtotalCentavos: linha.SubtotalCentavos,
		FreteCentavos:    linha.FreteCentavos,
	}
	// Tudo ou nada, pelo CHECK `pedido_criacao_completa`: basta uma coluna.
	if linha.EnderecoCep.Valid {
		d.Endereco = &Endereco{
			Destinatario: linha.EnderecoDestinatario.String,
			CEP:          linha.EnderecoCep.String,
			Logradouro:   linha.EnderecoLogradouro.String,
			Numero:       linha.EnderecoNumero.String,
			Complemento:  linha.EnderecoComplemento.String,
			Bairro:       linha.EnderecoBairro.String,
			Cidade:       linha.EnderecoCidade.String,
			UF:           linha.EnderecoUf.String,
		}
	}

	if d.Historico, err = Historico(ctx, bd, d.ID); err != nil {
		return Detalhe{}, fmt.Errorf("ler o histórico do Pedido: %w", err)
	}
	if expira, ok := ExpiraEm(d.Status, d.Historico, prazo); ok {
		d.ExpiraEm = &expira
	}
	if d.TentativasRestantes, err = pagamento.TentativasRestantes(ctx, bd, d.ID, teto); err != nil {
		return Detalhe{}, err
	}

	itens, err := q.ItensDoPedido(ctx, chave)
	if err != nil {
		return Detalhe{}, fmt.Errorf("ler os Itens do Pedido: %w", err)
	}
	ids := make([]string, len(itens))
	for i, it := range itens {
		ids[i] = it.ProdutoID.String()
	}
	disponivel, err := catalogo.Disponivel(ctx, bd, ids)
	if err != nil {
		return Detalhe{}, fmt.Errorf("ler o Estoque disponível dos Itens: %w", err)
	}
	// Um Item por Produto: o Pedido nasce do Carrinho, que tem UNIQUE
	// (carrinho_id, produto_id), então comparar cada Item sozinho com o
	// disponível do Produto não esquece unidade de outra linha.
	d.Itens = make([]ItemDoPedido, len(itens))
	for i, it := range itens {
		d.Itens[i] = ItemDoPedido{
			ProdutoID:              ids[i],
			Nome:                   it.Nome,
			Quantidade:             it.Quantidade,
			PrecoPraticadoCentavos: it.PrecoPraticadoCentavos,
			Disponivel:             disponivel[ids[i]] >= int(it.Quantidade),
		}
	}
	return d, nil
}

// Listar é a leitura da tela "Meus pedidos" (2.6) — o esboço que a Estória 6.1
// substitui (filtro por Status, ordenação e Skeleton ficam para ela). O dono
// entra no WHERE, e não numa checagem depois (AD-11): a consulta nunca traz
// Pedido de outro Comprador para descartar depois. A ordem é `id DESC`: a
// chave é uuidv7(), ordenada no tempo por construção, e por isso o mais
// recente já sai no topo sem somar um JOIN em transicao_status.
func Listar(ctx context.Context, bd gerado.DBTX, compradorID string) ([]Pedido, error) {
	var comprador pgtype.UUID
	if err := comprador.Scan(compradorID); err != nil {
		return nil, fmt.Errorf("identificador de Comprador inválido: %w", err)
	}
	linhas, err := gerado.New(bd).ListarPedidosDoComprador(ctx, comprador)
	if err != nil {
		return nil, err
	}
	// Fatia vazia, e não nil: a lista sem nenhum Pedido serializa em `[]`, e
	// não em `null` — o estado vazio é da tela, e um `null` a obrigaria a
	// defender-se do formato (o mesmo contrato de Meus Endereços, 2.5).
	pedidos := make([]Pedido, 0, len(linhas))
	for _, linha := range linhas {
		pedidos = append(pedidos, Pedido{
			ID:            linha.ID.String(),
			Numero:        linha.Numero,
			Status:        Status(linha.Status),
			TotalCentavos: linha.TotalCentavos,
		})
	}
	return pedidos, nil
}

// Varrer é o passo "aplicar" do tique: lê a inbox de `pagamento` e aplica o
// que chegou. Não tem relógio próprio — quem o chama é `cmd/azamon`, e esta
// função não decide quando roda (AD-6).
//
// Uma transação por Pedido, nunca uma por tique: um Pedido que falha não
// desfaz o que os outros já aplicaram, e o `FOR UPDATE SKIP LOCKED` deixa o
// tique seguinte passar por cima do que já está travado.
func Varrer(ctx context.Context, pool *pgxpool.Pool) error {
	pendentes, err := pagamento.ConfirmacoesNaoAplicadas(ctx, pool)
	if err != nil {
		return err
	}
	var falhas []error
	for _, c := range pendentes {
		if err := aplicarConfirmacao(ctx, pool, c); err != nil {
			falhas = append(falhas, fmt.Errorf("aplicar a confirmação %s: %w", c.ID, err))
		}
	}
	return errors.Join(falhas...)
}

// aplicarConfirmacao é a transação de um Pedido só. O desfecho é sempre
// terminal para a confirmação: ou APLICADA, ou NAO_APLICAVEL_SINALIZADA.
func aplicarConfirmacao(ctx context.Context, pool *pgxpool.Pool, c pagamento.Pendente) error {
	var chave pgtype.UUID
	if err := chave.Scan(c.PedidoID); err != nil {
		return fmt.Errorf("identificador de Pedido inválido: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	// WithoutCancel pelo mesmo motivo de `api/`: desfazer precisa de contexto
	// vivo, e o encerramento do processo cancela o da varredura.
	defer tx.Rollback(context.WithoutCancel(ctx))

	travado, err := gerado.New(tx).TravarPedido(ctx, chave)
	if errors.Is(err, pgx.ErrNoRows) {
		// Travado por outro tique — ou por outro processo. Não é falha: a
		// confirmação continua PENDENTE e o próximo tique a encontra.
		return nil
	}
	if err != nil {
		return fmt.Errorf("travar o Pedido: %w", err)
	}

	// A aprovação que chega para um Pedido já CANCELADO é a corrida clássica
	// do checkout (FR-26): o Status não muda, mas a informação de que o
	// pagamento foi aprovado não pode virar perda silenciosa (addendum §2,
	// invariante 7). O registro que a 6.5 lê é a própria linha da inbox —
	// APROVADO em NAO_APLICAVEL_SINALIZADA numa Tentativa de um Pedido
	// CANCELADO —, e não o aviso lá embaixo: log não é caminho de leitura.
	//
	// `Corrente` não entra de propósito, e é a diferença para o `if` que
	// aplica: uma aprovação de Tentativa superada também é dinheiro aprovado,
	// e o Pedido está cancelado do mesmo jeito. Confirmação RECUSADA não avisa
	// — não há pagamento aprovado a perder.
	aprovadaSobreCancelado := c.Resultado == pagamento.Aprovado && Status(travado.Status) == StatusCancelado

	// Só é aplicada a confirmação que pertence à Tentativa corrente, que é
	// aprovada e cujo Pedido ainda aguarda pagamento. Todo o resto é sinalizado
	// e sai da fila — aplicar a recusa é da 5.10, e expirar, da 5.11.
	estado := pagamento.NaoAplicavelSinalizada
	// Quem protege o Status do Pedido é o CAS de Transicionar, e não esta
	// condição: sem ela, o UPDATE … WHERE status = 'AGUARDANDO_PAGAMENTO'
	// casaria zero linhas e sairia ErrEstadoJaAvancado. Ela existe para nomear
	// o caso e não gastar uma transição perdida.
	if c.Corrente && c.Resultado == pagamento.Aprovado && Status(travado.Status) == StatusAguardandoPagamento {
		err := Transicionar(ctx, tx, c.PedidoID, StatusAguardandoPagamento, StatusPago, AtorProvedor, "")
		switch {
		case err == nil:
			estado = pagamento.Aplicada
		case errors.Is(err, ErrEstadoJaAvancado):
			// Desfecho esperado, e não erro: alguém chegou primeiro. Fica
			// registrado e a varredura segue em frente.
			slog.InfoContext(ctx, "confirmação chegou depois do avanço",
				"pedido", c.PedidoID, "confirmacao", c.ID)
		default:
			return err
		}
	}
	if err := pagamento.Marcar(ctx, tx, c.ID, estado); err != nil {
		return err
	}
	if err := tx.Commit(context.WithoutCancel(ctx)); err != nil {
		return err
	}
	// Depois do commit, e não antes: commit que falha devolve a confirmação a
	// PENDENTE, e o aviso emitido antes se repetiria a cada tique sobre uma
	// sinalização que ainda não existe.
	if aprovadaSobreCancelado {
		slog.WarnContext(ctx, "pagamento aprovado sobre Pedido cancelado",
			"pedido", c.PedidoID, "confirmacao", c.ID)
	}
	return nil
}

// SimularEntrega é o passo "simular" do tique: leva o Pedido PAGO até ENTREGUE,
// uma etapa por intervalo vencido. Não tem relógio próprio — quem a chama é
// `cmd/azamon`, e o interruptor que a desliga é dele (AD-6).
//
// O que decide não é estado em memória: é o histórico de pedido.transicao_status
// dizendo desde quando o Pedido está neste Status. Por isso o contêiner
// derrubado no meio retoma cada Pedido de onde parou, sem intervenção.
//
// Os candidatos são lidos fora da transação e reconferidos dentro dela, no
// molde do Varrer: uma transação por Pedido, e não uma por tique.
func SimularEntrega(ctx context.Context, pool *pgxpool.Pool, intervalo time.Duration) error {
	var ate pgtype.Timestamptz
	if err := ate.Scan(time.Now().Add(-intervalo)); err != nil {
		return fmt.Errorf("calcular o corte do intervalo de entrega: %w", err)
	}
	var origens []string
	for de := range maps.Keys(simulacao) {
		origens = append(origens, string(de))
	}
	candidatos, err := gerado.New(pool).PedidosParaAvancar(ctx, gerado.PedidosParaAvancarParams{
		Status: origens,
		Ate:    ate,
	})
	if err != nil {
		return fmt.Errorf("listar os Pedidos para avançar: %w", err)
	}
	var falhas []error
	for _, id := range candidatos {
		if err := avancarEntrega(ctx, pool, id, ate); err != nil {
			falhas = append(falhas, fmt.Errorf("avançar o Pedido %s: %w", id.String(), err))
		}
	}
	return errors.Join(falhas...)
}

// avancarEntrega é a transação de um Pedido só. A releitura travada é o que
// torna inofensiva a corrida entre dois tiques sobrepostos: quem chega depois
// ou não trava a linha, ou reencontra um Status que já não é candidato.
func avancarEntrega(ctx context.Context, pool *pgxpool.Pool, id pgtype.UUID, ate pgtype.Timestamptz) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	travado, err := gerado.New(tx).TravarPedido(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		// Travado por outro tique. Não é falha: o próximo o encontra.
		return nil
	}
	if err != nil {
		return fmt.Errorf("travar o Pedido: %w", err)
	}
	atual := Status(travado.Status)
	proximo, ok := proximoDaSimulacao(atual)
	// A decisão é reconferida com a trava na mão, e não só na seleção: entre a
	// leitura dos candidatos e esta linha, outro caminho pode ter avançado o
	// Pedido — e aí o intervalo recomeça a contar do Status novo.
	//
	// `Desde` é anulável, e um NULL lê como instante zero, que nunca é posterior
	// ao corte: sem o teste de validade, Pedido sem histórico nenhum passaria
	// por "vencido há muito" e avançaria. A consulta de candidatos já o exclui,
	// e é aqui que as duas deixam de concordar por acaso.
	if !ok || !travado.Desde.Valid || travado.Desde.Time.After(ate.Time) {
		return nil
	}

	// O efeito sobre o Estoque — a consolidação em SEPARANDO → ENVIADO — é
	// do próprio Transicionar, que o aplica depois do CAS e na mesma
	// transação (AD-3): a simulação não tem como esquecê-lo.
	switch err := Transicionar(ctx, tx, id.String(), atual, proximo, AtorSimulacao, ""); {
	case errors.Is(err, ErrEstadoJaAvancado):
		// Desfecho esperado, e não erro: alguém chegou primeiro.
		slog.InfoContext(ctx, "a simulação chegou depois do avanço",
			"pedido", id.String(), "de", string(atual))
		return nil
	case err != nil:
		return err
	}
	return tx.Commit(context.WithoutCancel(ctx))
}
