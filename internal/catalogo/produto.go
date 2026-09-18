package catalogo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/catalogo/db/gerado"
)

// ReferenciaInvalida é o Vendedor ou a Categoria que não existe (ou nem é
// uuid) no criar e no editar Produto. Campo é o nome no corpo da requisição,
// e `api/` o devolve como erro em linha.
type ReferenciaInvalida struct{ Campo string }

func (e ReferenciaInvalida) Error() string {
	if e.Campo == "categoria_id" {
		return "Escolha uma Categoria cadastrada."
	}
	return "Escolha um Vendedor cadastrado."
}

// ProdutoAdmin é a linha da tela do Administrador: ativos e inativos, com o
// Estoque total — a loja nunca o vê (AD-16).
type ProdutoAdmin struct {
	ID            string
	Nome          string
	Descricao     string
	PrecoCentavos int64
	ImagemURL     string
	EstoqueTotal  int64
	Ativo         bool
	VendedorID    string
	VendedorNome  string
	CategoriaID   string
	CategoriaNome string
}

// DadosDoProduto é o que o Administrador escreve. EstoqueTotal só vale no
// criar, e Ativo só no editar: o Produto nasce ativo.
type DadosDoProduto struct {
	Nome          string
	Descricao     string
	PrecoCentavos int64
	ImagemURL     string
	VendedorID    string
	CategoriaID   string
	EstoqueTotal  int32
	Ativo         bool
}

// ListarProdutosAdmin devolve a página pedida e o total. Página além do total
// é fatia vazia, sem consulta à lista.
func ListarProdutosAdmin(ctx context.Context, bd gerado.DBTX, pagina, porPagina int) ([]ProdutoAdmin, int64, error) {
	consultas := gerado.New(bd)
	total, err := consultas.ContarProdutos(ctx)
	if err != nil {
		return nil, 0, err
	}
	produtos := []ProdutoAdmin{}
	// Comparado antes de multiplicar: uma `pagina` enorme não transborda.
	if int64(pagina-1) >= (total+int64(porPagina)-1)/int64(porPagina) {
		return produtos, total, nil
	}
	linhas, err := consultas.ListarProdutosAdmin(ctx, gerado.ListarProdutosAdminParams{
		Limite:       int32(porPagina),
		Deslocamento: int32((pagina - 1) * porPagina),
	})
	if err != nil {
		return nil, 0, err
	}
	for _, l := range linhas {
		produtos = append(produtos, produtoAdminDe(gerado.BuscarProdutoAdminRow(l)))
	}
	return produtos, total, nil
}

// CriarProduto grava o Produto ativo, com `busca_normalizada` escrita aqui.
// Quem valida os limites é `api/`.
func CriarProduto(ctx context.Context, bd gerado.DBTX, d DadosDoProduto) (ProdutoAdmin, error) {
	vendedor, categoria, err := referenciasDe(d)
	if err != nil {
		return ProdutoAdmin{}, err
	}
	consultas := gerado.New(bd)
	id, err := consultas.CriarProduto(ctx, gerado.CriarProdutoParams{
		Nome:             d.Nome,
		Descricao:        d.Descricao,
		PrecoCentavos:    d.PrecoCentavos,
		ImagemUrl:        d.ImagemURL,
		VendedorID:       vendedor,
		CategoriaID:      categoria,
		EstoqueTotal:     d.EstoqueTotal,
		BuscaNormalizada: NormalizarBusca(d.Nome, d.Descricao),
	})
	if err != nil {
		return ProdutoAdmin{}, traduzirProduto(err)
	}
	linha, err := consultas.BuscarProdutoAdmin(ctx, id)
	if err != nil {
		return ProdutoAdmin{}, err
	}
	return produtoAdminDe(linha), nil
}

// AtualizarProduto reescreve tudo menos o Estoque total (o ajuste é da 3.4).
// Desativar é só `Ativo` falso: quem esconde é a VIEW produto_visivel, e o
// Item de Pedido já congelou o preço. Inexistente e uuid malformado saem como
// pgx.ErrNoRows.
func AtualizarProduto(ctx context.Context, bd gerado.DBTX, id string, d DadosDoProduto) (ProdutoAdmin, error) {
	chave, err := uuidDe(id)
	if err != nil {
		return ProdutoAdmin{}, err
	}
	vendedor, categoria, err := referenciasDe(d)
	if err != nil {
		return ProdutoAdmin{}, err
	}
	consultas := gerado.New(bd)
	chave, err = consultas.AtualizarProduto(ctx, gerado.AtualizarProdutoParams{
		ID:               chave,
		Nome:             d.Nome,
		Descricao:        d.Descricao,
		PrecoCentavos:    d.PrecoCentavos,
		ImagemUrl:        d.ImagemURL,
		VendedorID:       vendedor,
		CategoriaID:      categoria,
		Ativo:            d.Ativo,
		BuscaNormalizada: NormalizarBusca(d.Nome, d.Descricao),
	})
	if err != nil {
		return ProdutoAdmin{}, traduzirProduto(err)
	}
	linha, err := consultas.BuscarProdutoAdmin(ctx, chave)
	if err != nil {
		return ProdutoAdmin{}, err
	}
	return produtoAdminDe(linha), nil
}

// referenciasDe: uuid malformado é o mesmo erro de campo do inexistente.
func referenciasDe(d DadosDoProduto) (vendedor, categoria pgtype.UUID, err error) {
	if vendedor, err = uuidDe(d.VendedorID); err != nil {
		return vendedor, categoria, ReferenciaInvalida{Campo: "vendedor_id"}
	}
	if categoria, err = uuidDe(d.CategoriaID); err != nil {
		return vendedor, categoria, ReferenciaInvalida{Campo: "categoria_id"}
	}
	return vendedor, categoria, nil
}

// traduzirProduto: o 23503 diz qual referência falhou pelo nome da constraint.
func traduzirProduto(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		switch pgErr.ConstraintName {
		case "produto_vendedor_id_fkey":
			return ReferenciaInvalida{Campo: "vendedor_id"}
		case "produto_categoria_id_fkey":
			return ReferenciaInvalida{Campo: "categoria_id"}
		}
	}
	return err
}

func produtoAdminDe(l gerado.BuscarProdutoAdminRow) ProdutoAdmin {
	return ProdutoAdmin{
		ID:            l.ID.String(),
		Nome:          l.Nome,
		Descricao:     l.Descricao,
		PrecoCentavos: l.PrecoCentavos,
		ImagemURL:     l.ImagemUrl,
		EstoqueTotal:  int64(l.EstoqueTotal),
		Ativo:         l.Ativo,
		VendedorID:    l.VendedorID.String(),
		VendedorNome:  l.VendedorNome,
		CategoriaID:   l.CategoriaID.String(),
		CategoriaNome: l.CategoriaNome,
	}
}
