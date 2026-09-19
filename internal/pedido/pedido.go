// Package pedido é dono de Pedido, Item de Pedido, máquina de estados, Frete e varredura.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1), junto com maquina.go, que guarda a tabela de transições do
// AD-3 e o Transicionar, e frete.go, que guarda a Regra de Frete (AD-17).
// Aqui moram o nascimento do Pedido em AGUARDANDO_PAGAMENTO, as leituras e os dois passos da varredura — aplicar a
// confirmação e simular a entrega até ENTREGUE.
package pedido

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
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

// unidade: a estória compra uma unidade por Pedido. Escolha de quantidade e
// Carrinho são das Épicas 4 e 5 — o campo no banco já é `quantidade`, então
// alargar é trocar esta constante por parâmetro.
const unidade = 1

// Pedido é o que a Página de Produto mostra assim que a compra é confirmada.
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
		Status:        string(StatusAguardandoPagamento),
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

	// A primeira linha do histórico (NFR-9), que é a primeira linha da tabela
	// do AD-3. O estado anterior é vazio porque não havia estado antes — é o
	// nascimento, e não um avanço, e por isso não passa pelo CAS.
	if err := q.RegistrarTransicao(ctx, gerado.RegistrarTransicaoParams{
		PedidoID:       linha.ID,
		StatusAnterior: "",
		StatusNovo:     string(StatusAguardandoPagamento),
		Ator:           string(AtorComprador),
	}); err != nil {
		return Pedido{}, fmt.Errorf("registrar a transição: %w", err)
	}

	pedido := Pedido{
		ID:            linha.ID.String(),
		Numero:        linha.Numero,
		Status:        Status(linha.Status),
		TotalCentavos: linha.TotalCentavos,
	}
	if err := catalogo.Reservar(ctx, tx, pedido.ID, []catalogo.ItemReserva{{ProdutoID: produto.ID, Quantidade: unidade}}); err != nil {
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
		Status:        Status(linha.Status),
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

	// Só é aplicada a confirmação que pertence à Tentativa corrente, que é
	// aprovada e cujo Pedido ainda aguarda pagamento. Todo o resto é sinalizado
	// e sai da fila — recusa e expiração são da Épica 5.
	estado := pagamento.NaoAplicavelSinalizada
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
