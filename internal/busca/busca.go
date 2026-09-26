// Package busca é dono da única consulta de Produto da loja — a costura do
// Elasticsearch. Não tem tabelas: lê o Catálogo só pela VIEW produto_visivel,
// que já aplica o predicado de visibilidade (AD-2, AD-16, AD-19).
//
// Este arquivo abre a interface pública do módulo, mas a porta é o pacote: o
// que os arquivos irmãos exportam também é interface (AD-1).
package busca

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/busca/db/gerado"
	"github.com/Cashnip/amazon-waddle/internal/catalogo"
)

// As ordenações da loja (FR-14). Recentes é o padrão.
const (
	OrdenacaoRecentes  = "recentes"
	OrdenacaoPrecoAsc  = "preco_asc"
	OrdenacaoPrecoDesc = "preco_desc"
)

// ErrCategoriaMalformada é o `categoria` que não é uuid. Categoria bem formada
// e inexistente não é erro: é listagem vazia.
var ErrCategoriaMalformada = errors.New("A Categoria informada não é válida.")

// Produto é o cartão da Vitrine: o Estoque disponível, e nunca o total.
type Produto struct {
	ID                string
	Nome              string
	PrecoCentavos     int64
	ImagemURL         string
	EstoqueDisponivel int
}

// Filtro é o que a loja pede além da página. Nil é "sem filtro". Os tetos e a
// faixa invertida são de `api/`; aqui o termo só é normalizado e escapado.
type Filtro struct {
	Termo       *string
	CategoriaID *string
	PrecoMin    *int64
	PrecoMax    *int64
	Ordenacao   string // vazio vale OrdenacaoRecentes
}

// escapeLike torna `\`, `%` e `_` texto literal no LIKE (o escape padrão do
// PostgreSQL é a barra invertida).
var escapeLike = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Listar devolve a página pedida dos Produtos visíveis que casam com o filtro,
// e o total deles. Página além do total é fatia vazia, sem consulta à lista.
func Listar(ctx context.Context, bd gerado.DBTX, f Filtro, pagina, porPagina int) ([]Produto, int64, error) {
	var termo pgtype.Text
	if f.Termo != nil {
		// A mesma normalização da escrita: "CAFÉ" casa com "cafe".
		if n := catalogo.Normalizar(*f.Termo); n != "" {
			termo = pgtype.Text{String: escapeLike.Replace(n), Valid: true}
		}
	}
	var categoria pgtype.UUID
	if f.CategoriaID != nil {
		if err := categoria.Scan(*f.CategoriaID); err != nil {
			return nil, 0, ErrCategoriaMalformada
		}
	}
	precoMin, precoMax := int8De(f.PrecoMin), int8De(f.PrecoMax)
	ordenacao := f.Ordenacao
	if ordenacao == "" {
		ordenacao = OrdenacaoRecentes
	}

	consultas := gerado.New(bd)
	total, err := consultas.ContarVisiveis(ctx, gerado.ContarVisiveisParams{
		Termo: termo, CategoriaID: categoria, PrecoMin: precoMin, PrecoMax: precoMax,
	})
	if err != nil {
		return nil, 0, err
	}
	produtos := []Produto{}
	// Comparado antes de multiplicar: uma `pagina` enorme não transborda.
	if int64(pagina-1) >= (total+int64(porPagina)-1)/int64(porPagina) {
		return produtos, total, nil
	}
	linhas, err := consultas.ListarVisiveis(ctx, gerado.ListarVisiveisParams{
		Termo: termo, CategoriaID: categoria, PrecoMin: precoMin, PrecoMax: precoMax,
		Ordenacao:    ordenacao,
		Limite:       int32(porPagina),
		Deslocamento: int32((pagina - 1) * porPagina),
	})
	if err != nil {
		return nil, 0, err
	}
	for _, l := range linhas {
		produtos = append(produtos, Produto{
			ID:                l.ID.String(),
			Nome:              l.Nome,
			PrecoCentavos:     l.PrecoCentavos,
			ImagemURL:         l.ImagemUrl,
			EstoqueDisponivel: int(l.EstoqueDisponivel),
		})
	}
	return produtos, total, nil
}

func int8De(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}
