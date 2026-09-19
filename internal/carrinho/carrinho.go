// Package carrinho é dono de Carrinho e Item de Carrinho.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1). O dono vem sempre de fora, como texto: `carrinho` não conhece
// `identidade` (AD-11), e quem resolve a Sessão é `api/`.
package carrinho

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/carrinho/db/gerado"
	"github.com/Cashnip/amazon-waddle/internal/catalogo"
)

// ErrTetoPorItem recusa a adição cuja soma passaria do teto por Item (FR-17).
// Quem conhece o teto é quem chama, e é ele que o nomeia na mensagem.
var ErrTetoPorItem = errors.New("a quantidade passaria do teto por Item")

// Item é o Item de Carrinho como a adição o devolve.
type Item struct {
	ID         string
	ProdutoID  string
	Quantidade int32
}

// Adicionar põe q unidades do Produto no Carrinho do Comprador, criando o
// Carrinho na primeira vez. O Produto repetido soma ao Item existente.
//
// Visibilidade e preço vêm só de catalogo.BuscarProduto (AD-19): Produto
// invisível, inexistente ou de uuid malformado sai como pgx.ErrNoRows. A soma
// acima do teto sai como ErrTetoPorItem, sem gravar nada no Item. O teto da
// entrada (1 ≤ q ≤ teto) é de quem chama, na fronteira (NFR-14).
//
// Garantir o Carrinho e gravar o Item são duas idas sem transação: se a
// segunda falhar, sobra um Carrinho vazio — que é o estado de todo Comprador
// que ainda não adicionou nada.
func Adicionar(ctx context.Context, bd gerado.DBTX, compradorID, produtoID string, q, teto int) (Item, error) {
	dono, err := uuidDe(compradorID)
	if err != nil {
		return Item{}, err
	}
	produto, err := catalogo.BuscarProduto(ctx, bd, produtoID)
	if err != nil {
		return Item{}, err
	}
	var chave pgtype.UUID
	if err := chave.Scan(produto.ID); err != nil {
		return Item{}, err
	}
	consultas := gerado.New(bd)
	carrinhoID, err := consultas.GarantirCarrinho(ctx, dono)
	if err != nil {
		return Item{}, err
	}
	linha, err := consultas.AdicionarItem(ctx, gerado.AdicionarItemParams{
		CarrinhoID:         carrinhoID,
		ProdutoID:          chave,
		Quantidade:         int32(q),
		PrecoVistoCentavos: produto.PrecoCentavos,
		Teto:               int32(teto),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, ErrTetoPorItem
	}
	if err != nil {
		return Item{}, err
	}
	return Item{ID: linha.ID.String(), ProdutoID: linha.ProdutoID.String(), Quantidade: linha.Quantidade}, nil
}

// RemoverItem apaga o Item do dono. A posse está no WHERE (AD-11): Item
// alheio, inexistente e de uuid malformado saem os três como pgx.ErrNoRows.
func RemoverItem(ctx context.Context, bd gerado.DBTX, itemID, compradorID string) error {
	chave, err := uuidDe(itemID)
	if err != nil {
		return err
	}
	dono, err := uuidDe(compradorID)
	if err != nil {
		return err
	}
	linhas, err := gerado.New(bd).RemoverItem(ctx, gerado.RemoverItemParams{ID: chave, CompradorID: dono})
	if err != nil {
		return err
	}
	if linhas == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// uuidDe: o identificador malformado vira pgx.ErrNoRows, e não um 500 — para
// quem chama, "não é um uuid" e "não existe" são a mesma coisa.
func uuidDe(texto string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(texto); err != nil {
		return u, pgx.ErrNoRows
	}
	return u, nil
}
