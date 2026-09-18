package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// saidaCategoria não tem `categoria_pai_id`: a Categoria é plana.
type saidaCategoria struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
}

// As rotas daqui moram no mux `admin` e herdam a guarda do prefixo.

func (s *servidor) listarCategorias(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	categorias, err := catalogo.ListarCategorias(r.Context(), s.pool)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	saidas := make([]saidaCategoria, 0, len(categorias))
	for _, c := range categorias {
		saidas = append(saidas, saidaCategoria(c))
	}
	escreverJSON(w, http.StatusOK, saidas)
}

func (s *servidor) criarCategoria(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	nome, ok := s.nomeDaCategoria(w, r)
	if !ok {
		return
	}
	nova, err := catalogo.CriarCategoria(r.Context(), s.pool, nome)
	if err != nil {
		escreverErroDeCategoria(w, r, err)
		return
	}
	escreverJSON(w, http.StatusCreated, saidaCategoria(nova))
}

func (s *servidor) renomearCategoria(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	nome, ok := s.nomeDaCategoria(w, r)
	if !ok {
		return
	}
	renomeada, err := catalogo.RenomearCategoria(r.Context(), s.pool, r.PathValue("id"), nome)
	if err != nil {
		escreverErroDeCategoria(w, r, err)
		return
	}
	escreverJSON(w, http.StatusOK, saidaCategoria(renomeada))
}

func (s *servidor) removerCategoria(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	if err := catalogo.RemoverCategoria(r.Context(), s.pool, r.PathValue("id")); err != nil {
		escreverErroDeCategoria(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func escreverErroDeCategoria(w http.ResponseWriter, r *http.Request, err error) {
	var comProdutos catalogo.CategoriaComProdutos
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
	case errors.As(err, &comProdutos):
		erro.EscreverCategoriaComProdutos(r.Context(), w, comProdutos.Produtos)
	case errors.Is(err, catalogo.ErrCategoriaJaCadastrada):
		erro.Escrever(r.Context(), w, err, map[string]string{"campo": "nome"})
	default:
		erro.Escrever(r.Context(), w, err, nil)
	}
}

// nomeDaCategoria lê o corpo, apara o nome e o valida contra o teto da Config,
// contado em runas.
func (s *servidor) nomeDaCategoria(w http.ResponseWriter, r *http.Request) (string, bool) {
	var entrada struct {
		Nome string `json:"nome"`
	}
	if err := decodificarCorpo(w, r, &entrada); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return "", false
	}
	nome := strings.TrimSpace(entrada.Nome)
	maximo := erro.Milhar(int64(s.cfg.CategoriaNomeMax))
	switch {
	case nome == "":
		erro.EscreverCampo(r.Context(), w, "nome", fmt.Sprintf("Informe o nome da Categoria, com até %s caracteres.", maximo))
		return "", false
	case utf8.RuneCountInString(nome) > s.cfg.CategoriaNomeMax:
		erro.EscreverCampo(r.Context(), w, "nome", fmt.Sprintf("O nome da Categoria tem no máximo %s caracteres.", maximo))
		return "", false
	}
	return nome, true
}
