// Package busca é dono da única consulta de Produto da loja — a costura do
// Elasticsearch. Não tem tabelas: lê o Catálogo só pela VIEW produto_visivel,
// que já aplica o predicado de visibilidade (AD-2, AD-16, AD-19).
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1).
package busca

import (
	"context"

	"github.com/Cashnip/amazon-waddle/internal/busca/db/gerado"
)

// Produto é o cartão da Vitrine: o Estoque disponível, e nunca o total.
type Produto struct {
	ID                string
	Nome              string
	PrecoCentavos     int64
	ImagemURL         string
	EstoqueDisponivel int
}

// Listar devolve a página pedida dos Produtos visíveis, por id, e o total.
// Página além do total é fatia vazia, sem consulta à lista.
func Listar(ctx context.Context, bd gerado.DBTX, pagina, porPagina int) ([]Produto, int64, error) {
	consultas := gerado.New(bd)
	total, err := consultas.ContarVisiveis(ctx)
	if err != nil {
		return nil, 0, err
	}
	produtos := []Produto{}
	// Comparado antes de multiplicar: uma `pagina` enorme não transborda.
	if int64(pagina-1) >= (total+int64(porPagina)-1)/int64(porPagina) {
		return produtos, total, nil
	}
	linhas, err := consultas.ListarVisiveis(ctx, gerado.ListarVisiveisParams{
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
