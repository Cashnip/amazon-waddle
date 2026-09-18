package catalogo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
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

// AtualizarProduto reescreve tudo menos o Estoque total, que tem o
// AjustarEstoque com a guarda das Reservas.
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

// ErrEstoqueComprometido recusa o ajuste que deixaria o Estoque total abaixo
// das Reservas ativas. Quem precisa do número usa errors.As com
// EstoqueComprometido.
var ErrEstoqueComprometido = errors.New("O Estoque total não pode ficar abaixo das unidades comprometidas em Pedidos abertos.")

// EstoqueComprometido carrega quantas unidades as Reservas ativas seguram,
// contadas sob a trava, até `api/`, que as nomeia na mensagem.
type EstoqueComprometido struct{ N int64 }

func (e EstoqueComprometido) Error() string { return ErrEstoqueComprometido.Error() }
func (e EstoqueComprometido) Unwrap() error { return ErrEstoqueComprometido }

// AjustarEstoque é o ajuste do Estoque total pelo Administrador, que vale para
// Produto ativo e inativo. A ordem é a do AD-5, a mesma do Reservar: primeiro a
// trava do Produto (sem a VIEW, ou o inativo seria 404), só depois a soma das
// Reservas ativas — invertida, uma compra simultânea entraria entre a soma e o
// UPDATE e o total ficaria abaixo do comprometido. Total abaixo das Reservas
// sai como EstoqueComprometido; inexistente e uuid malformado, como
// pgx.ErrNoRows. Quem valida o teto é `api/`.
//
// tx é parâmetro nomeado (AD-4): a trava só vale até o fim da transação.
func AjustarEstoque(ctx context.Context, tx pgx.Tx, id string, total int32) (ProdutoAdmin, error) {
	chave, err := uuidDe(id)
	if err != nil {
		return ProdutoAdmin{}, err
	}
	consultas := gerado.New(tx)
	if _, err := consultas.TravarProdutoParaAjuste(ctx, []pgtype.UUID{chave}); err != nil {
		return ProdutoAdmin{}, err
	}
	somas, err := consultas.SomarReservasAtivas(ctx, []pgtype.UUID{chave})
	if err != nil {
		return ProdutoAdmin{}, err
	}
	if len(somas) == 1 && int64(total) < somas[0].Reservado {
		return ProdutoAdmin{}, EstoqueComprometido{N: somas[0].Reservado}
	}
	if err := consultas.AjustarEstoqueTotal(ctx, gerado.AjustarEstoqueTotalParams{ID: chave, EstoqueTotal: total}); err != nil {
		return ProdutoAdmin{}, err
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
