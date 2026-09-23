// Package pedido é dono de Pedido, Item de Pedido, máquina de estados, Frete e varredura.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1), junto com maquina.go, que guarda a tabela de transições do
// AD-3 e o Transicionar, e frete.go, que guarda a Regra de Frete (AD-17).
// Aqui moram o nascimento do Pedido a partir do Carrinho, as leituras e os três
// passos da varredura que são de `pedido` — aplicar a confirmação, expirar a
// Tentativa vencida e simular a entrega até ENTREGUE.
package pedido

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
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

// expiracao é o avanço do passo `expirar` (FR-34), e é uma linha só: a
// Tentativa vencida leva o Pedido pelo caminho da recusa. Mora ao lado da
// `simulacao` pelo mesmo motivo dela — é a lista única de onde saem tanto os
// candidatos da consulta quanto o Status de destino.
var expiracao = map[Status]Status{
	StatusAguardandoPagamento: StatusPagamentoRecusado,
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

// ErrTentativasEsgotadas recusa a nova Tentativa de Pagamento do Pedido que
// já usou o teto do §7.1 (FR-27). Quem confere o teto é `pagamento`, dono da
// Tentativa, com pagamento.ErrTetoDeTentativas; `pedido` o traduz nesta
// recusa da transição (AD-8), e é esta que a borda HTTP conhece — o AD-1 não
// tem aresta de `plataforma` para `pagamento`. O erro devolvido embrulha os
// dois, então errors.Is casa com qualquer um.
var ErrTentativasEsgotadas = errors.New("Este Pedido já usou todas as Tentativas de Pagamento.")

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
func Criar(ctx context.Context, tx pgx.Tx, compradorID string, novo NovoPedido, isencaoCentavos int64, teto int) (Pedido, bool, error) {
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
	// `pedido`. O teto também vai como valor: quem o confere é `pagamento`.
	if err := pagamento.IniciarTentativa(ctx, tx, pedido.ID, pedido.TotalCentavos, teto); err != nil {
		return Pedido{}, false, err
	}
	return pedido, false, nil
}

// NovaTentativa é a Tentativa de Pagamento que o Comprador inicia a partir do
// próprio Pedido recusado (FR-27), na transação que `api/` abriu (AD-4). Nada
// é remontado: os Itens, o total e o Endereço são os do Pedido.
//
// **A ordem é o contrato:**
//
//  1. o Pedido do dono — de outro Comprador, inexistente ou malformado sai
//     como pgx.ErrNoRows, o mesmo 404 (AD-11);
//  2. Transicionar de PAGAMENTO_RECUSADO a AGUARDANDO_PAGAMENTO, que faz o
//     compare-and-swap, a linha do histórico — é dela que ExpiraEm recomeça o
//     prazo — e a nova Reserva sobre os Itens do Pedido, falhando com
//     EstoqueInsuficiente se faltar;
//  3. só então pagamento.IniciarTentativa, que numera e confere o teto —
//     além dele, ErrTentativasEsgotadas. Com a linha do Pedido presa pelo CAS, dois
//     cliques contam uma vez só: o segundo espera e sai ErrEstadoJaAvancado.
//     Antes do CAS, os dois contariam o mesmo número e colidiriam no índice
//     único de `id_externo`, num 500.
//
// Qualquer recusa volta para quem abriu a transação desfazer: o Pedido
// permanece em PAGAMENTO_RECUSADO, sem Reserva, sem Tentativa e sem linha nova.
// O Pedido devolvido já está no Status novo.
func NovaTentativa(ctx context.Context, tx pgx.Tx, pedidoID, compradorID string, teto int) (Pedido, error) {
	var chave, comprador pgtype.UUID
	if err := chave.Scan(pedidoID); err != nil {
		return Pedido{}, pgx.ErrNoRows
	}
	if err := comprador.Scan(compradorID); err != nil {
		return Pedido{}, pgx.ErrNoRows
	}
	linha, err := gerado.New(tx).BuscarPedidoDoComprador(ctx, gerado.BuscarPedidoDoCompradorParams{
		PedidoID:    chave,
		CompradorID: comprador,
	})
	if err != nil {
		return Pedido{}, err
	}
	p := Pedido{
		ID:            linha.ID.String(),
		Numero:        linha.Numero,
		Status:        StatusAguardandoPagamento,
		TotalCentavos: linha.TotalCentavos,
	}
	if err := Transicionar(ctx, tx, p.ID, StatusPagamentoRecusado, StatusAguardandoPagamento, AtorComprador, ""); err != nil {
		return Pedido{}, err
	}
	if err := pagamento.IniciarTentativa(ctx, tx, p.ID, p.TotalCentavos, teto); err != nil {
		if errors.Is(err, pagamento.ErrTetoDeTentativas) {
			return Pedido{}, fmt.Errorf("%w: %w", ErrTentativasEsgotadas, err)
		}
		return Pedido{}, err
	}
	return p, nil
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

	// O gêmeo da FR-34: a aprovação que chega depois de uma expiração legítima.
	// O Comprador pagou tarde, a Reserva já voltou para a prateleira, e o
	// dinheiro foi aprovado assim mesmo — pelo mesmo motivo de cima, não pode
	// ser perda silenciosa. Só a expiração conta: a recusa do Provedor não é
	// dinheiro aprovado que se perdeu, é dinheiro que nunca entrou.
	aprovadaSobreExpirado := false
	if c.Resultado == pagamento.Aprovado && Status(travado.Status) == StatusPagamentoRecusado {
		aprovadaSobreExpirado, err = expirouPorTempoEsgotado(ctx, tx, chave)
		if err != nil {
			return err
		}
	}

	// Só é aplicada a confirmação que pertence à Tentativa corrente e cujo
	// Pedido ainda aguarda pagamento: a aprovada leva a PAGO, e a recusada, a
	// PAGAMENTO_RECUSADO, liberando a Reserva dentro de Transicionar. Todo o
	// resto é sinalizado e sai da fila — sem o `Corrente`, uma recusa atrasada
	// da Tentativa 1 derrubaria a Reserva que a 2 acabou de criar (AD-7).
	estado := pagamento.NaoAplicavelSinalizada
	destino, motivo := StatusPago, ""
	if c.Resultado == pagamento.Recusado {
		destino, motivo = StatusPagamentoRecusado, MotivoRecusadoPeloProvedor
	}
	// Quem protege o Status do Pedido é o CAS de Transicionar, e não esta
	// condição: sem ela, o UPDATE … WHERE status = 'AGUARDANDO_PAGAMENTO'
	// casaria zero linhas e sairia ErrEstadoJaAvancado. Ela existe para nomear
	// o caso e não gastar uma transição perdida.
	if c.Corrente && Status(travado.Status) == StatusAguardandoPagamento {
		err := Transicionar(ctx, tx, c.PedidoID, StatusAguardandoPagamento, destino, AtorProvedor, motivo)
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
	if aprovadaSobreExpirado {
		slog.WarnContext(ctx, "pagamento aprovado sobre Tentativa de Pagamento expirada",
			"pedido", c.PedidoID, "confirmacao", c.ID)
	}
	return nil
}

// expirouPorTempoEsgotado diz se a última transição do Pedido é a da FR-34 —
// PAGAMENTO_RECUSADO por TEMPO_ESGOTADO, e não pela recusa do Provedor. Lê o
// histórico que já existe, sob a mesma trava: consulta nova para isto seria
// pagar por uma linha que `HistoricoDoPedido` já traz, e a lista é curta por
// construção (o teto de Tentativas a limita).
func expirouPorTempoEsgotado(ctx context.Context, tx pgx.Tx, id pgtype.UUID) (bool, error) {
	historico, err := gerado.New(tx).HistoricoDoPedido(ctx, id)
	if err != nil {
		return false, fmt.Errorf("ler o histórico do Pedido: %w", err)
	}
	if len(historico) == 0 {
		return false, nil
	}
	ultima := historico[len(historico)-1]
	return Status(ultima.StatusNovo) == StatusPagamentoRecusado && ultima.Motivo.String == MotivoTempoEsgotado, nil
}

// SimularEntrega é o passo "simular" do tique: leva o Pedido PAGO até ENTREGUE,
// uma etapa por intervalo vencido. Não tem relógio próprio — quem a chama é
// `cmd/azamon`, e o interruptor que a desliga é dele (AD-6).
//
// O que decide não é estado em memória: é o histórico de pedido.transicao_status
// dizendo desde quando o Pedido está neste Status. Por isso o contêiner
// derrubado no meio retoma cada Pedido de onde parou, sem intervenção.
func SimularEntrega(ctx context.Context, pool *pgxpool.Pool, intervalo time.Duration) error {
	return avancarPeloTempo(ctx, pool, intervalo, passoDoTempo{
		rotulo:  "intervalo da entrega simulada",
		avancos: simulacao,
		ator:    AtorSimulacao,
	})
}

// Expirar é o passo "expirar" do tique (FR-34): a Tentativa de Pagamento que
// passou do prazo sem confirmação leva o Pedido a PAGAMENTO_RECUSADO com
// TEMPO_ESGOTADO. A Reserva de Estoque é liberada pelo efeito da linha do AD-3,
// dentro de Transicionar — quem chama aqui não invoca catalogo.Liberar.
//
// O instante é o mesmo que a tela mostra em `ExpiraEm`, e não uma segunda
// conta: para um Pedido em AGUARDANDO_PAGAMENTO, o `desde` do TravarPedido
// (max(ocorrido_em)) É a última transição *para* esse Status, porque qualquer
// linha posterior o teria tirado de lá. Por isso a nova Tentativa (5.10)
// recomeça o prazo sozinha, sem código novo aqui.
//
// Não é desligável: o interruptor de `cmd/azamon` é da simulação de entrega, e
// a FR-34 não tem um.
//
// O que protege quem pagou no prazo não é a ordem do tique, e sim a inbox lida
// com a linha do Pedido já presa (`conferirInbox` abaixo): a ordem sozinha
// deixa três furos — `Varrer` que falhou e não aplicou nada, a linha que o
// SKIP LOCKED pulou, e a confirmação que chegou entre os dois passos.
func Expirar(ctx context.Context, pool *pgxpool.Pool, prazo time.Duration) error {
	return avancarPeloTempo(ctx, pool, prazo, passoDoTempo{
		rotulo:        "prazo da Tentativa de Pagamento",
		avancos:       expiracao,
		ator:          AtorVarredura,
		motivo:        MotivoTempoEsgotado,
		conferirInbox: true,
	})
}

// passoDoTempo é o que distingue os dois passos da varredura que decidem pelo
// relógio — `expirar` e `simular` (AD-6). A casca é a mesma e mora abaixo;
// muda só de onde para onde se avança, com que ator e motivo o histórico
// registra, e se a inbox é conferida antes de transitar.
type passoDoTempo struct {
	// rotulo nomeia o relógio deste passo nas mensagens de erro: quem falhou
	// foi o corte, e dizer "VARREDURA" mandaria procurar o ator.
	rotulo  string
	avancos map[Status]Status
	ator    Ator
	motivo  string
	// conferirInbox é só da expiração. A simulação de entrega não o usa: ela
	// move Pedido já pago, onde não há confirmação por aplicar que se perca.
	conferirInbox bool
}

// avancarPeloTempo lê os candidatos FORA da transação e os reconfere dentro
// dela, no molde do Varrer: uma transação por Pedido, e não uma por tique.
func avancarPeloTempo(ctx context.Context, pool *pgxpool.Pool, decorrido time.Duration, passo passoDoTempo) error {
	var ate pgtype.Timestamptz
	if err := ate.Scan(time.Now().Add(-decorrido)); err != nil {
		return fmt.Errorf("calcular o corte do %s: %w", passo.rotulo, err)
	}
	origens := make([]string, 0, len(passo.avancos))
	for de := range passo.avancos {
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
		if err := avancarUm(ctx, pool, id, ate, passo); err != nil {
			falhas = append(falhas, fmt.Errorf("avançar o Pedido %s: %w", id.String(), err))
		}
	}
	return errors.Join(falhas...)
}

// avancarUm é a transação de um Pedido só. A releitura travada é o que
// torna inofensiva a corrida entre dois tiques sobrepostos: quem chega depois
// ou não trava a linha, ou reencontra um Status que já não é candidato.
func avancarUm(ctx context.Context, pool *pgxpool.Pool, id pgtype.UUID, ate pgtype.Timestamptz, passo passoDoTempo) error {
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
	proximo, ok := passo.avancos[atual]
	// A decisão é reconferida com a trava na mão, e não só na seleção: entre a
	// leitura dos candidatos e esta linha, outro caminho pode ter avançado o
	// Pedido — a aprovação aplicada pelo passo `aplicar` do mesmo tique, por
	// exemplo — e aí o prazo recomeça a contar do Status novo.
	//
	// `Desde` é anulável, e um NULL lê como instante zero, que nunca é posterior
	// ao corte: sem o teste de validade, Pedido sem histórico nenhum passaria
	// por "vencido há muito" e avançaria. A consulta de candidatos já o exclui,
	// e é aqui que as duas deixam de concordar por acaso.
	if !ok || !travado.Desde.Valid || travado.Desde.Time.After(ate.Time) {
		return nil
	}

	// A inbox, já com a linha do Pedido presa: havendo confirmação por
	// aplicar, quem pagou pagou dentro do prazo e o tique seguinte a aplica —
	// expirar aqui tiraria a Reserva de Estoque de quem pagou. É esta leitura,
	// e não a ordem `aplicar` → `expirar`, que dá a garantia: a ordem não cobre
	// o `Varrer` que falhou, a linha que o SKIP LOCKED pulou nem a confirmação
	// que chegou entre os dois passos.
	if passo.conferirInbox {
		pendente, err := pagamento.TemConfirmacaoPendente(ctx, tx, id.String())
		if err != nil {
			return err
		}
		if pendente {
			slog.InfoContext(ctx, "expiração adiada: confirmação por aplicar",
				"pedido", id.String())
			return nil
		}
	}

	// O efeito sobre o Estoque — a consolidação em SEPARANDO → ENVIADO, a
	// liberação da Reserva na expiração — é do próprio Transicionar, que o
	// aplica depois do CAS e na mesma transação (AD-3): o passo do tempo não
	// tem como esquecê-lo.
	switch err := Transicionar(ctx, tx, id.String(), atual, proximo, passo.ator, passo.motivo); {
	case errors.Is(err, ErrEstadoJaAvancado):
		// Desfecho esperado, e não erro: alguém chegou primeiro.
		slog.InfoContext(ctx, "o passo do tempo chegou depois do avanço",
			"pedido", id.String(), "de", string(atual), "ator", string(passo.ator))
		return nil
	case err != nil:
		return err
	}
	return tx.Commit(context.WithoutCancel(ctx))
}
