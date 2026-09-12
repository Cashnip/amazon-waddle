// Package pedido é dono de Pedido, Item de Pedido, máquina de estados, Frete e varredura.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1). O que mora aqui na Épica 1 é o nascimento do Pedido em
// AGUARDANDO_PAGAMENTO e o compare-and-swap que é o único ponto de mutação do
// Status. As nove transições nomeadas e as recusas são da Épica 5.
package pedido

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

// StatusPago é o único avanço que a Épica 1 conhece. Dele para EM_SEPARACAO é
// a 1.8; a Reserva de Estoque continua ATIVA aqui — consolidar é da 1.8.
const StatusPago = "PAGO"

// autorDaVarredura é o que vai para a coluna `autor` do histórico quando quem
// avança não é uma pessoa. O Comprador é autor do nascimento; o Provedor, da
// confirmação.
const autorDaVarredura = "provedor-pagamento"

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

	status, err := gerado.New(tx).TravarPedido(ctx, chave)
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
	if c.Corrente && c.Resultado == pagamento.Aprovado && status == StatusInicial {
		err := Transicionar(ctx, tx, c.PedidoID, StatusInicial, StatusPago, autorDaVarredura)
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
