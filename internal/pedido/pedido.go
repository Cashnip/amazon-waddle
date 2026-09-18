// Package pedido é dono de Pedido, Item de Pedido, máquina de estados, Frete e varredura.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1). O que mora aqui na Épica 1 é o nascimento do Pedido em
// AGUARDANDO_PAGAMENTO, o compare-and-swap que é o único ponto de mutação do
// Status e os dois passos da varredura — aplicar a confirmação e simular a
// entrega até ENTREGUE. As nove transições nomeadas, as recusas e a expiração
// da Tentativa são da Épica 5.
package pedido

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/Cashnip/amazon-waddle/internal/pedido/db/gerado"
)

// ErrEstadoJaAvancado é o desfecho de toda transição que chega tarde: o
// compare-and-swap afetou zero linhas porque outro caminho já moveu o Pedido.
// Não é falha do sistema — a varredura conta com ele como resultado normal.
var ErrEstadoJaAvancado = errors.New("O Pedido já avançou de estado.")

// StatusInicial é onde todo Pedido nasce. Os outros seis chegam na Épica 5,
// mas o CHECK da migração já os conhece.
const StatusInicial = "AGUARDANDO_PAGAMENTO"

// StatusPago é onde a confirmação do Provedor deixa o Pedido, e de onde a
// simulação de entrega parte. A Reserva de Estoque continua ATIVA aqui: quem a
// consolida é a passagem EM_SEPARACAO → ENVIADO.
const StatusPago = "PAGO"

// Os três Status que a simulação de entrega percorre, mais o CANCELADO que ela
// nunca alcança e que EstadoTerminal precisa nomear. Os sete do CHECK da
// migração estão declarados desde a 1.6 — usá-los não pede migração nova.
const (
	StatusEmSeparacao = "EM_SEPARACAO"
	StatusEnviado     = "ENVIADO"
	StatusEntregue    = "ENTREGUE"
	StatusCancelado   = "CANCELADO"
)

// Os dois autores não-humanos do histórico. O Comprador é autor do nascimento;
// o Provedor, da confirmação; a simulação, dos três avanços da entrega.
const (
	autorDoProvedor  = "provedor-pagamento"
	autorDaSimulacao = "simulacao-entrega"
)

// simulacao é a lista inteira de avanços que a entrega simulada conhece, e é
// uma só: dela saem tanto o próximo Status quanto os candidatos que a consulta
// procura. Duas listas seriam duas oportunidades de divergirem.
var simulacao = map[string]string{
	StatusPago:        StatusEmSeparacao,
	StatusEmSeparacao: StatusEnviado,
	StatusEnviado:     StatusEntregue,
}

// proximoDaSimulacao devolve para onde a simulação leva este Status, e false
// para todo Status que ela não move — AGUARDANDO_PAGAMENTO, PAGAMENTO_RECUSADO,
// CANCELADO e o próprio ENTREGUE, independentemente do tempo decorrido.
func proximoDaSimulacao(status string) (string, bool) {
	proximo, ok := simulacao[status]
	return proximo, ok
}

// EstadoTerminal é a única declaração de "acabou" do sistema (AD-18): terminal
// é ENTREGUE ou CANCELADO, e mais nada. A resposta do Pedido carrega o
// resultado dela para que a tela não redeclare a regra em JavaScript.
func EstadoTerminal(status string) bool {
	return status == StatusEntregue || status == StatusCancelado
}

// unidade: a estória compra uma unidade por Pedido. Escolha de quantidade e
// Carrinho são das Épicas 4 e 5 — o campo no banco já é `quantidade`, então
// alargar é trocar esta constante por parâmetro.
const unidade = 1

// Pedido é o que a Página de Produto mostra assim que a compra é confirmada.
// O total é coluna, nunca derivação de leitura (AD-3).
type Pedido struct {
	ID            string
	Numero        string
	Status        string
	TotalCentavos int64
	// AtualizadoEm é o instante da última transição, e só a leitura o
	// preenche: é ele que a tela de acompanhamento exibe, absoluto e vindo do
	// servidor — o navegador nunca conta duração.
	AtualizadoEm time.Time
}

// Criar faz o Pedido nascer em AGUARDANDO_PAGAMENTO, dentro da transação que
// `api/` abriu (AD-4). A ordem é a do AD-6: o Pedido e a primeira linha do
// histórico primeiro, o efeito sobre o Estoque depois — os dois na mesma
// transação, e uma Reserva recusada reverte o Pedido junto.
//
// Produto inexistente e identificador malformado saem como pgx.ErrNoRows, que
// `api/` traduz em 404 — nunca em 500.
func Criar(ctx context.Context, tx pgx.Tx, compradorID, produtoID string) (Pedido, error) {
	var comprador pgtype.UUID
	if err := comprador.Scan(compradorID); err != nil {
		return Pedido{}, fmt.Errorf("identificador de Comprador inválido: %w", err)
	}
	// O Item congela o Produto: daqui em diante o Pedido não depende mais da
	// linha de catalogo.produto continuar como estava.
	produto, err := catalogo.BuscarProduto(ctx, tx, produtoID)
	if err != nil {
		return Pedido{}, err
	}

	q := gerado.New(tx)
	ano := time.Now().UTC().Year()
	sequencial, err := q.ProximoNumeroDoAno(ctx, int32(ano))
	if err != nil {
		return Pedido{}, fmt.Errorf("reservar o número do Pedido: %w", err)
	}

	// Multiplicação de inteiros: não existe divisão no caminho monetário (AD-3).
	linha, err := q.CriarPedido(ctx, gerado.CriarPedidoParams{
		Numero:        fmt.Sprintf("AZ-%d-%06d", ano, sequencial),
		CompradorID:   comprador,
		Status:        StatusInicial,
		TotalCentavos: produto.PrecoCentavos * unidade,
	})
	if err != nil {
		return Pedido{}, fmt.Errorf("criar o Pedido: %w", err)
	}

	var produtoChave pgtype.UUID
	_ = produtoChave.Scan(produto.ID) // veio do banco, já é canônico
	if err := q.CriarItemPedido(ctx, gerado.CriarItemPedidoParams{
		PedidoID:               linha.ID,
		ProdutoID:              produtoChave,
		Nome:                   produto.Nome,
		VendedorNome:           produto.VendedorNome,
		PrecoPraticadoCentavos: produto.PrecoCentavos,
		Quantidade:             unidade,
	}); err != nil {
		return Pedido{}, fmt.Errorf("criar o Item do Pedido: %w", err)
	}

	// A primeira linha do histórico (NFR-9). O estado anterior é vazio porque
	// não havia estado antes — é o nascimento, e não um avanço.
	if err := q.RegistrarTransicao(ctx, gerado.RegistrarTransicaoParams{
		PedidoID:       linha.ID,
		StatusAnterior: "",
		StatusNovo:     StatusInicial,
		Autor:          compradorID,
	}); err != nil {
		return Pedido{}, fmt.Errorf("registrar a transição: %w", err)
	}

	pedido := Pedido{
		ID:            linha.ID.String(),
		Numero:        linha.Numero,
		Status:        linha.Status,
		TotalCentavos: linha.TotalCentavos,
	}
	if err := catalogo.Reservar(ctx, tx, produto.ID, pedido.ID, unidade); err != nil {
		return Pedido{}, err
	}
	// A Tentativa nasce na mesma transação (AD-7): o Pedido revertido não
	// deixa Tentativa órfã, e a emissão derivada não tem o que reemitir.
	// O total vai como valor — `pagamento` não consulta `pedido`.
	if err := pagamento.IniciarTentativa(ctx, tx, pedido.ID, pedido.TotalCentavos); err != nil {
		return Pedido{}, err
	}
	return pedido, nil
}

// Buscar é a leitura da tela de acompanhamento. O dono entra no WHERE: Pedido
// de outro Comprador, Pedido inexistente e identificador malformado saem os
// três como pgx.ErrNoRows, que `api/` traduz no mesmo 404 — não vaza
// existência.
func Buscar(ctx context.Context, bd gerado.DBTX, pedidoID, compradorID string) (Pedido, error) {
	var chave, comprador pgtype.UUID
	if err := chave.Scan(pedidoID); err != nil {
		return Pedido{}, pgx.ErrNoRows
	}
	if err := comprador.Scan(compradorID); err != nil {
		return Pedido{}, pgx.ErrNoRows
	}
	linha, err := gerado.New(bd).BuscarPedidoDoComprador(ctx, gerado.BuscarPedidoDoCompradorParams{
		PedidoID:    chave,
		CompradorID: comprador,
	})
	if err != nil {
		return Pedido{}, err
	}
	return Pedido{
		ID:            linha.ID.String(),
		Numero:        linha.Numero,
		Status:        linha.Status,
		TotalCentavos: linha.TotalCentavos,
		AtualizadoEm:  linha.AtualizadoEm.Time.UTC(),
	}, nil
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
			Status:        linha.Status,
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

	// Só é aplicada a confirmação que pertence à Tentativa corrente, que é
	// aprovada e cujo Pedido ainda aguarda pagamento. Todo o resto é sinalizado
	// e sai da fila — recusa e expiração são da Épica 5.
	estado := pagamento.NaoAplicavelSinalizada
	if c.Corrente && c.Resultado == pagamento.Aprovado && travado.Status == StatusInicial {
		err := Transicionar(ctx, tx, c.PedidoID, StatusInicial, StatusPago, autorDoProvedor)
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
	return tx.Commit(context.WithoutCancel(ctx))
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
	candidatos, err := gerado.New(pool).PedidosParaAvancar(ctx, gerado.PedidosParaAvancarParams{
		Status: slices.Collect(maps.Keys(simulacao)),
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
	proximo, ok := proximoDaSimulacao(travado.Status)
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

	switch err := Transicionar(ctx, tx, id.String(), travado.Status, proximo, autorDaSimulacao); {
	case errors.Is(err, ErrEstadoJaAvancado):
		// Desfecho esperado, e não erro: alguém chegou primeiro.
		slog.InfoContext(ctx, "a simulação chegou depois do avanço",
			"pedido", id.String(), "de", travado.Status)
		return nil
	case err != nil:
		return err
	}

	// A transição vem antes do efeito sobre o Estoque, na mesma transação
	// (AD-6). EM_SEPARACAO → ENVIADO é a única passagem que altera o Estoque
	// total: é nela que o cancelamento deixa de ser possível e a Reserva não
	// tem mais o que segurar.
	if proximo == StatusEnviado {
		if err := catalogo.Consolidar(ctx, tx, id.String()); err != nil {
			return err
		}
	}
	return tx.Commit(context.WithoutCancel(ctx))
}

// Transicionar é o único ponto de mutação do Status (AD-6), e é
// compare-and-swap: o estado de origem entra no WHERE. Zero linhas afetadas
// devolve ErrEstadoJaAvancado — outro caminho chegou primeiro, e insistir é
// que seria o defeito.
func Transicionar(ctx context.Context, tx pgx.Tx, pedidoID, de, para, autor string) error {
	var chave pgtype.UUID
	if err := chave.Scan(pedidoID); err != nil {
		// Não é ErrEstadoJaAvancado: a varredura da 1.8 trata esse sentinela
		// como desfecho normal e engoliria em silêncio um identificador que
		// quem chamou montou errado.
		return fmt.Errorf("identificador de Pedido inválido: %w", err)
	}
	q := gerado.New(tx)
	linhas, err := q.AvancarStatus(ctx, gerado.AvancarStatusParams{PedidoID: chave, De: de, Para: para})
	if err != nil {
		return fmt.Errorf("avançar o Status: %w", err)
	}
	if linhas == 0 {
		return ErrEstadoJaAvancado
	}
	// O histórico vem depois do avanço, e na mesma transação: o que não
	// aconteceu não é registrado, e o que aconteceu não fica sem linha.
	if err := q.RegistrarTransicao(ctx, gerado.RegistrarTransicaoParams{
		PedidoID:       chave,
		StatusAnterior: de,
		StatusNovo:     para,
		Autor:          autor,
	}); err != nil {
		return fmt.Errorf("registrar a transição: %w", err)
	}
	return nil
}
