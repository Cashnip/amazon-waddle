package api

import (
	"net/http"

	"github.com/Cashnip/amazon-waddle/internal/busca"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

type saidaProdutoVitrine struct {
	ID                string `json:"id"`
	Nome              string `json:"nome"`
	PrecoCentavos     int64  `json:"preco_centavos"`
	ImagemURL         string `json:"imagem_url"`
	EstoqueDisponivel int    `json:"estoque_disponivel"`
}

// listarProdutos é a listagem da loja (AD-16): de `busca`, só Produto visível.
// Sem cache: Produto criado ou desativado aparece ou some no próximo pedido.
func (s *servidor) listarProdutos(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	pagina, porPagina, ok := paginacaoDe(w, r, s.cfg)
	if !ok {
		return
	}
	produtos, total, err := busca.Listar(r.Context(), s.pool, pagina, porPagina)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	itens := make([]saidaProdutoVitrine, 0, len(produtos))
	for _, p := range produtos {
		itens = append(itens, saidaProdutoVitrine(p))
	}
	escreverJSON(w, http.StatusOK, listagem[saidaProdutoVitrine]{itens, pagina, porPagina, total})
}
