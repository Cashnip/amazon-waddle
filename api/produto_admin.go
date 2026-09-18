package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// entradaProduto é o corpo do POST e do PUT. Ponteiro onde a ausência não
// pode virar zero em silêncio: `estoque_total` só no POST, `ativo` só no PUT,
// e nos dois ele é obrigatório. O preço é inteiro de centavos: quem converte
// "12,90" em 1290 é o navegador, por texto (AD-3).
type entradaProduto struct {
	Nome          string `json:"nome"`
	Descricao     string `json:"descricao"`
	PrecoCentavos int64  `json:"preco_centavos"`
	ImagemURL     string `json:"imagem_url"`
	VendedorID    string `json:"vendedor_id"`
	CategoriaID   string `json:"categoria_id"`
	EstoqueTotal  *int64 `json:"estoque_total"`
	Ativo         *bool  `json:"ativo"`
}

type saidaProdutoAdmin struct {
	ID            string          `json:"id"`
	Nome          string          `json:"nome"`
	Descricao     string          `json:"descricao"`
	PrecoCentavos int64           `json:"preco_centavos"`
	ImagemURL     string          `json:"imagem_url"`
	EstoqueTotal  int64           `json:"estoque_total"`
	Ativo         bool            `json:"ativo"`
	Vendedor      saidaReferencia `json:"vendedor"`
	Categoria     saidaReferencia `json:"categoria"`
}

type saidaReferencia struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
}

// listagem é o envelope do AD-18.
type listagem[T any] struct {
	Itens     []T   `json:"itens"`
	Pagina    int   `json:"pagina"`
	PorPagina int   `json:"por_pagina"`
	Total     int64 `json:"total"`
}

// listarProdutosAdmin é a listagem administrativa (AD-16 emendado): do
// `catalogo`, com ativos e inativos, por nome e id.
func (s *servidor) listarProdutosAdmin(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	pagina := 1
	if texto := r.URL.Query().Get("pagina"); texto != "" {
		n, err := strconv.Atoi(texto)
		if err != nil || n < 1 {
			erro.EscreverCampo(r.Context(), w, "pagina", "A página é um número inteiro a partir de 1.")
			return
		}
		pagina = n
	}
	produtos, total, err := catalogo.ListarProdutosAdmin(r.Context(), s.pool, pagina, s.cfg.PaginaTamanho)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	itens := make([]saidaProdutoAdmin, 0, len(produtos))
	for _, p := range produtos {
		itens = append(itens, saidaProdutoAdminDe(p))
	}
	escreverJSON(w, http.StatusOK, listagem[saidaProdutoAdmin]{itens, pagina, s.cfg.PaginaTamanho, total})
}

func (s *servidor) criarProduto(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	dados, ok := s.produtoDaRequisicao(w, r, true)
	if !ok {
		return
	}
	novo, err := catalogo.CriarProduto(r.Context(), s.pool, dados)
	if err != nil {
		escreverErroDeProduto(w, r, err)
		return
	}
	escreverJSON(w, http.StatusCreated, saidaProdutoAdminDe(novo))
}

// atualizarProduto é editar, desativar e reativar: o corpo é a linha inteira,
// menos o Estoque total.
func (s *servidor) atualizarProduto(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	dados, ok := s.produtoDaRequisicao(w, r, false)
	if !ok {
		return
	}
	atualizado, err := catalogo.AtualizarProduto(r.Context(), s.pool, r.PathValue("id"), dados)
	if err != nil {
		escreverErroDeProduto(w, r, err)
		return
	}
	escreverJSON(w, http.StatusOK, saidaProdutoAdminDe(atualizado))
}

// ajustarEstoque é a rota própria do Estoque total (3.4), e não o PUT: a tela
// reenvia a linha lida ao desativar e reativar, e um total junto regravaria um
// valor velho por cima de uma consolidação. Aqui cada escrita do total é
// intencional. A transação é daqui (AD-4): a trava do AD-5 dura até o Commit.
func (s *servidor) ajustarEstoque(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	var e struct {
		EstoqueTotal *int64 `json:"estoque_total"`
	}
	if err := decodificarCorpo(w, r, &e); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	if e.EstoqueTotal == nil || *e.EstoqueTotal < 0 || *e.EstoqueTotal > int64(s.cfg.ProdutoEstoqueMax) {
		erro.EscreverCampo(r.Context(), w, "estoque_total", mensagemDoEstoque(s.cfg.ProdutoEstoqueMax))
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	// O mesmo par do criarPedido: Rollback depois do Commit é no-op, e os
	// dois com WithoutCancel porque o cliente pode desistir no meio.
	defer tx.Rollback(context.WithoutCancel(r.Context()))

	ajustado, err := catalogo.AjustarEstoque(r.Context(), tx, r.PathValue("id"), int32(*e.EstoqueTotal))
	if err != nil {
		var comprometido catalogo.EstoqueComprometido
		if errors.As(err, &comprometido) {
			erro.EscreverEstoqueComprometido(r.Context(), w, comprometido.N)
			return
		}
		escreverErroDeProduto(w, r, err)
		return
	}
	if err := tx.Commit(context.WithoutCancel(r.Context())); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusOK, saidaProdutoAdminDe(ajustado))
}

// mensagemDoEstoque é a mesma na criação e no ajuste.
func mensagemDoEstoque(maximo int) string {
	return fmt.Sprintf("O Estoque total tem de ficar entre 0 e %s unidades.", erro.Milhar(int64(maximo)))
}

func escreverErroDeProduto(w http.ResponseWriter, r *http.Request, err error) {
	var referencia catalogo.ReferenciaInvalida
	switch {
	case errors.As(err, &referencia):
		erro.EscreverCampo(r.Context(), w, referencia.Campo, referencia.Error())
	case errors.Is(err, pgx.ErrNoRows):
		erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
	default:
		erro.Escrever(r.Context(), w, err, nil)
	}
}

// produtoDaRequisicao apara o texto, conta em runas e confere cada campo
// contra o teto da Config. Vendedor e Categoria ficam para o banco: é a FK
// que decide se existem.
func (s *servidor) produtoDaRequisicao(w http.ResponseWriter, r *http.Request, criando bool) (catalogo.DadosDoProduto, bool) {
	var e entradaProduto
	// O teto do corpo sai dos tetos de texto da Config: seis bytes por
	// caractere é o pior caso do JSON (`\uXXXX`), e o resto é folga para os
	// outros campos.
	maximo := 6*int64(s.cfg.ProdutoNomeMax+s.cfg.ProdutoDescricaoMax) + corpoMaximo
	if err := decodificarCorpoAte(w, r, &e, maximo); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return catalogo.DadosDoProduto{}, false
	}
	e.Nome = strings.TrimSpace(e.Nome)
	e.Descricao = strings.TrimSpace(e.Descricao)
	cfg := s.cfg
	campo, mensagem := "", ""
	switch {
	case e.Nome == "":
		campo, mensagem = "nome", fmt.Sprintf("Informe o nome do Produto, com até %s caracteres.", erro.Milhar(int64(cfg.ProdutoNomeMax)))
	case utf8.RuneCountInString(e.Nome) > cfg.ProdutoNomeMax:
		campo, mensagem = "nome", fmt.Sprintf("O nome do Produto tem no máximo %s caracteres.", erro.Milhar(int64(cfg.ProdutoNomeMax)))
	case utf8.RuneCountInString(e.Descricao) > cfg.ProdutoDescricaoMax:
		campo, mensagem = "descricao", fmt.Sprintf("A descrição do Produto tem no máximo %s caracteres.", erro.Milhar(int64(cfg.ProdutoDescricaoMax)))
	case e.PrecoCentavos <= 0 || e.PrecoCentavos > cfg.ProdutoPrecoMaxCentavos:
		campo, mensagem = "preco_centavos", fmt.Sprintf("O preço do Produto tem de ser maior que zero e de no máximo R$ %s,%02d.",
			erro.Milhar(cfg.ProdutoPrecoMaxCentavos/100), cfg.ProdutoPrecoMaxCentavos%100)
	case criando && (e.EstoqueTotal == nil || *e.EstoqueTotal < 0 || *e.EstoqueTotal > int64(cfg.ProdutoEstoqueMax)):
		campo, mensagem = "estoque_total", mensagemDoEstoque(cfg.ProdutoEstoqueMax)
	case e.ImagemURL != "" && !slices.Contains(midias(), e.ImagemURL):
		campo, mensagem = "imagem_url", "Escolha uma imagem da lista, ou nenhuma."
	case !criando && e.Ativo == nil:
		campo, mensagem = "ativo", "Informe se o Produto está ativo."
	}
	if campo != "" {
		erro.EscreverCampo(r.Context(), w, campo, mensagem)
		return catalogo.DadosDoProduto{}, false
	}
	dados := catalogo.DadosDoProduto{
		Nome:          e.Nome,
		Descricao:     e.Descricao,
		PrecoCentavos: e.PrecoCentavos,
		ImagemURL:     e.ImagemURL,
		VendedorID:    e.VendedorID,
		CategoriaID:   e.CategoriaID,
		Ativo:         true,
	}
	if criando {
		dados.EstoqueTotal = int32(*e.EstoqueTotal)
	} else {
		dados.Ativo = *e.Ativo
	}
	return dados, true
}

func saidaProdutoAdminDe(p catalogo.ProdutoAdmin) saidaProdutoAdmin {
	return saidaProdutoAdmin{
		ID:            p.ID,
		Nome:          p.Nome,
		Descricao:     p.Descricao,
		PrecoCentavos: p.PrecoCentavos,
		ImagemURL:     p.ImagemURL,
		EstoqueTotal:  p.EstoqueTotal,
		Ativo:         p.Ativo,
		Vendedor:      saidaReferencia{p.VendedorID, p.VendedorNome},
		Categoria:     saidaReferencia{p.CategoriaID, p.CategoriaNome},
	}
}
