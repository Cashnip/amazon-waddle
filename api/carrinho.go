package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/carrinho"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// entradaItemCarrinho é o corpo da adição. A quantidade chega crua: um
// `int` recusaria "1.5" e `"3"` no decodificador, com o 400 genérico, e a
// matriz quer o erro nomeando o campo.
type entradaItemCarrinho struct {
	ProdutoID  string          `json:"produto_id"`
	Quantidade json.RawMessage `json:"quantidade"`
}

type saidaItemCarrinho struct {
	ID         string `json:"id"`
	ProdutoID  string `json:"produto_id"`
	Quantidade int32  `json:"quantidade"`
}

// adicionarAoCarrinho é a FR-16/FR-17: o dono vem da Sessão, e o Produto
// repetido soma ao Item existente. O teto é conferido duas vezes — aqui a
// entrada, e dentro da consulta a soma.
func (s *servidor) adicionarAoCarrinho(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	var entrada entradaItemCarrinho
	if err := decodificarCorpo(w, r, &entrada); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	teto := s.cfg.CarrinhoUnidadesMax
	q, err := strconv.Atoi(string(entrada.Quantidade))
	if err != nil || q < 1 || q > teto {
		erro.EscreverCampo(r.Context(), w, "quantidade", fmt.Sprintf("A quantidade vai de 1 a %d.", teto))
		return
	}
	item, err := carrinho.Adicionar(r.Context(), s.pool, comprador.ID, entrada.ProdutoID, q, teto)
	switch {
	case errors.Is(err, carrinho.ErrTetoPorItem):
		erro.EscreverCampo(r.Context(), w, "quantidade",
			fmt.Sprintf("O Carrinho aceita no máximo %d unidades de cada Produto.", teto))
		return
	case errors.Is(err, pgx.ErrNoRows):
		// Produto invisível, inexistente e uuid malformado: o mesmo 404.
		erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
		return
	case err != nil:
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusCreated, saidaItemCarrinho{ID: item.ID, ProdutoID: item.ProdutoID, Quantidade: item.Quantidade})
}

// removerItemDoCarrinho: Item alheio, inexistente e uuid malformado são o mesmo
// 404 — responder diferente vazaria a existência da linha (AD-11).
func (s *servidor) removerItemDoCarrinho(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	if err := carrinho.RemoverItem(r.Context(), s.pool, r.PathValue("id"), comprador.ID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
