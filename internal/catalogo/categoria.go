package catalogo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Cashnip/amazon-waddle/internal/catalogo/db/gerado"
)

// ErrCategoriaJaCadastrada é o UNIQUE de `categoria.nome` (23505), no criar e
// no renomear.
var ErrCategoriaJaCadastrada = errors.New("Já existe uma Categoria com este nome.")

// ErrCategoriaComProdutos é a FK `produto.categoria_id` (23503). Quem precisa
// da contagem usa errors.As com CategoriaComProdutos: a mensagem que nomeia o
// número é escrita em `plataforma/erro`, como a do teto de Endereços.
var ErrCategoriaComProdutos = errors.New("Esta Categoria tem Produtos e não pode ser removida.")

// CategoriaComProdutos carrega quantos Produtos estão vinculados.
type CategoriaComProdutos struct{ Produtos int64 }

func (e CategoriaComProdutos) Error() string { return ErrCategoriaComProdutos.Error() }
func (e CategoriaComProdutos) Unwrap() error { return ErrCategoriaComProdutos }

// Categoria é plana: `categoria_pai_id` existe, nulo e não exposto.
type Categoria struct {
	ID   string
	Nome string
}

// ListarCategorias devolve todas, por nome; sem nenhuma, a fatia vem vazia.
func ListarCategorias(ctx context.Context, bd gerado.DBTX) ([]Categoria, error) {
	linhas, err := gerado.New(bd).ListarCategorias(ctx)
	if err != nil {
		return nil, err
	}
	categorias := make([]Categoria, 0, len(linhas))
	for _, l := range linhas {
		categorias = append(categorias, Categoria{ID: l.ID.String(), Nome: l.Nome})
	}
	return categorias, nil
}

// CriarCategoria grava a Categoria. Quem apara e valida o nome é `api/`.
func CriarCategoria(ctx context.Context, bd gerado.DBTX, nome string) (Categoria, error) {
	l, err := gerado.New(bd).CriarCategoria(ctx, nome)
	if err != nil {
		return Categoria{}, traduzirCategoria(err)
	}
	return Categoria{ID: l.ID.String(), Nome: l.Nome}, nil
}

// RenomearCategoria: inexistente e uuid malformado saem como pgx.ErrNoRows.
func RenomearCategoria(ctx context.Context, bd gerado.DBTX, id, nome string) (Categoria, error) {
	chave, err := uuidDe(id)
	if err != nil {
		return Categoria{}, err
	}
	l, err := gerado.New(bd).AtualizarCategoria(ctx, gerado.AtualizarCategoriaParams{ID: chave, Nome: nome})
	if err != nil {
		return Categoria{}, traduzirCategoria(err)
	}
	return Categoria{ID: l.ID.String(), Nome: l.Nome}, nil
}

// RemoverCategoria apaga a Categoria sem Produto. Com Produto, a FK recusa, e
// só então os Produtos são contados: sai CategoriaComProdutos. Zero linhas
// afetadas é pgx.ErrNoRows.
func RemoverCategoria(ctx context.Context, bd gerado.DBTX, id string) error {
	chave, err := uuidDe(id)
	if err != nil {
		return err
	}
	consultas := gerado.New(bd)
	n, err := consultas.RemoverCategoria(ctx, chave)
	if err != nil {
		if !ehReferencia(err) {
			return err
		}
		produtos, errContagem := consultas.ContarProdutosDaCategoria(ctx, chave)
		if errContagem != nil {
			return errContagem
		}
		return CategoriaComProdutos{Produtos: produtos}
	}
	if n == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func traduzirCategoria(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrCategoriaJaCadastrada
	}
	return err
}

func ehReferencia(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
