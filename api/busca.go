package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

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
// Vitrine e Resultados são a mesma rota: termo, Categoria, faixa de preço e
// ordenação são opcionais e combináveis. Sem cache: Produto criado ou
// desativado aparece ou some no próximo pedido.
func (s *servidor) listarProdutos(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	pagina, porPagina, ok := paginacaoDe(w, r, s.cfg)
	if !ok {
		return
	}
	filtro, ok := s.filtroDe(w, r)
	if !ok {
		return
	}
	produtos, total, err := busca.Listar(r.Context(), s.pool, filtro, pagina, porPagina)
	if errors.Is(err, busca.ErrCategoriaMalformada) {
		erro.EscreverCampo(r.Context(), w, "categoria", err.Error())
		return
	}
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

// filtroDe lê termo, Categoria, faixa de preço (centavos) e ordenação da URL.
// Inválido escreve o erro em linha no campo e devolve ok falso.
func (s *servidor) filtroDe(w http.ResponseWriter, r *http.Request) (busca.Filtro, bool) {
	q := r.URL.Query()
	var f busca.Filtro

	// Só espaços é "sem termo"; o teto conta runas, não bytes.
	if termo := strings.TrimSpace(q.Get("termo")); termo != "" {
		// O Postgres recusa 0x00 em `text`: sem isto, o termo viraria 500.
		if strings.ContainsRune(termo, 0) {
			erro.EscreverCampo(r.Context(), w, "termo", "O termo de busca tem um caractere inválido.")
			return f, false
		}
		if utf8.RuneCountInString(termo) > s.cfg.BuscaTermoMax {
			erro.EscreverCampo(r.Context(), w, "termo",
				fmt.Sprintf("O termo de busca tem no máximo %s caracteres.", erro.Milhar(int64(s.cfg.BuscaTermoMax))))
			return f, false
		}
		f.Termo = &termo
	}
	if categoria := q.Get("categoria"); categoria != "" {
		f.CategoriaID = &categoria
	}
	preco := func(campo, nome string) (*int64, bool) {
		texto := q.Get(campo)
		if texto == "" {
			return nil, true
		}
		n, err := strconv.ParseInt(texto, 10, 64)
		if err != nil || n < 0 {
			erro.EscreverCampo(r.Context(), w, campo, "O preço "+nome+" é um número inteiro de centavos, a partir de 0.")
			return nil, false
		}
		return &n, true
	}
	var ok bool
	if f.PrecoMin, ok = preco("preco_min", "mínimo"); !ok {
		return f, false
	}
	if f.PrecoMax, ok = preco("preco_max", "máximo"); !ok {
		return f, false
	}
	if f.PrecoMin != nil && f.PrecoMax != nil && *f.PrecoMin > *f.PrecoMax {
		erro.EscreverCampo(r.Context(), w, "preco_min", "O preço mínimo não pode ser maior que o máximo.")
		return f, false
	}
	switch f.Ordenacao = q.Get("ordenacao"); f.Ordenacao {
	case "", busca.OrdenacaoRecentes, busca.OrdenacaoPrecoAsc, busca.OrdenacaoPrecoDesc:
	default:
		erro.EscreverCampo(r.Context(), w, "ordenacao", "A ordenação é recentes, preco_asc ou preco_desc.")
		return f, false
	}
	return f, true
}
