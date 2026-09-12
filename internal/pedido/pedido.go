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
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/pedido/db/gerado"
)

// ErrEstadoJaAvancado é o desfecho de toda transição que chega tarde: o
// compare-and-swap afetou zero linhas porque outro caminho já moveu o Pedido.
// Não é falha do sistema — a varredura conta com ele como resultado normal.
var ErrEstadoJaAvancado = errors.New("O Pedido já avançou de estado.")

// StatusInicial é onde todo Pedido nasce. Os outros seis chegam na Épica 5,
// mas o CHECK da migração já os conhece.
const StatusInicial = "AGUARDANDO_PAGAMENTO"

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
	return pedido, nil
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
