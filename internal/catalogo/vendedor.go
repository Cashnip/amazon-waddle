package catalogo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/catalogo/db/gerado"
)

// ErrVendedorJaCadastrado é o UNIQUE de `vendedor.nome` (23505), no criar e no
// editar. A duplicidade é do banco, nunca de um SELECT antes.
var ErrVendedorJaCadastrado = errors.New("Já existe um Vendedor com este nome.")

// ErrVendedorComProdutos é a FK `produto.vendedor_id` (23503): o Vendedor com
// Produto não some, porque os Produtos dele continuam referenciados. A saída é
// desativar, e a mensagem diz isso.
var ErrVendedorComProdutos = errors.New("Este Vendedor tem Produtos e não pode ser removido. Desative-o no lugar.")

// Vendedor é a linha inteira: a tela do Administrador lista ativos e inativos.
type Vendedor struct {
	ID    string
	Nome  string
	Ativo bool
}

// ListarVendedores devolve todos, por nome; sem nenhum, a fatia vem vazia.
func ListarVendedores(ctx context.Context, bd gerado.DBTX) ([]Vendedor, error) {
	linhas, err := gerado.New(bd).ListarVendedores(ctx)
	if err != nil {
		return nil, err
	}
	vendedores := make([]Vendedor, 0, len(linhas))
	for _, l := range linhas {
		vendedores = append(vendedores, vendedorDe(l))
	}
	return vendedores, nil
}

// CriarVendedor grava o Vendedor ativo. Quem apara e valida o nome é `api/`.
func CriarVendedor(ctx context.Context, bd gerado.DBTX, nome string) (Vendedor, error) {
	linha, err := gerado.New(bd).CriarVendedor(ctx, nome)
	if err != nil {
		return Vendedor{}, traduzirVendedor(err)
	}
	return vendedorDe(linha), nil
}

// AtualizarVendedor reescreve nome e situação. Desativar é só isto: um UPDATE
// de `ativo`, e nenhum Pedido é tocado — quem esconde os Produtos é a VIEW
// `produto_visivel`. Inexistente e uuid malformado saem como pgx.ErrNoRows.
func AtualizarVendedor(ctx context.Context, bd gerado.DBTX, id, nome string, ativo bool) (Vendedor, error) {
	chave, err := uuidDe(id)
	if err != nil {
		return Vendedor{}, err
	}
	linha, err := gerado.New(bd).AtualizarVendedor(ctx, gerado.AtualizarVendedorParams{ID: chave, Nome: nome, Ativo: ativo})
	if err != nil {
		return Vendedor{}, traduzirVendedor(err)
	}
	return vendedorDe(linha), nil
}

// RemoverVendedor apaga o Vendedor sem Produto. Com Produto, a FK recusa e sai
// ErrVendedorComProdutos; zero linhas afetadas é pgx.ErrNoRows.
func RemoverVendedor(ctx context.Context, bd gerado.DBTX, id string) error {
	chave, err := uuidDe(id)
	if err != nil {
		return err
	}
	n, err := gerado.New(bd).RemoverVendedor(ctx, chave)
	if err != nil {
		return traduzirVendedor(err)
	}
	if n == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func traduzirVendedor(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrVendedorJaCadastrado
		case "23503":
			return ErrVendedorComProdutos
		}
	}
	return err
}

func vendedorDe(l gerado.CatalogoVendedor) Vendedor {
	return Vendedor{ID: l.ID.String(), Nome: l.Nome, Ativo: l.Ativo}
}

// uuidDe: identificador malformado e inexistente são a mesma coisa para quem
// chama — pgx.ErrNoRows, que `api/` traduz em 404.
func uuidDe(texto string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(texto); err != nil {
		return u, pgx.ErrNoRows
	}
	return u, nil
}
