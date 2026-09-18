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

// entradaVendedor é o corpo do POST e do PUT. `ativo` é ponteiro porque só o
// PUT o lê, e lá ele é obrigatório: um campo esquecido não pode virar um
// Vendedor desativado por acidente.
type entradaVendedor struct {
	Nome  string `json:"nome"`
	Ativo *bool  `json:"ativo"`
}

type saidaVendedor struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Ativo bool   `json:"ativo"`
}

// Todas as rotas daqui moram no mux `admin` e herdam a guarda do prefixo: não
// há checagem de papel nos handlers, e nenhum deles precisa da Conta.

func (s *servidor) listarVendedores(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	vendedores, err := catalogo.ListarVendedores(r.Context(), s.pool)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	saidas := make([]saidaVendedor, 0, len(vendedores))
	for _, v := range vendedores {
		saidas = append(saidas, saidaVendedor(v))
	}
	escreverJSON(w, http.StatusOK, saidas)
}

func (s *servidor) criarVendedor(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	entrada, ok := s.vendedorDaRequisicao(w, r, false)
	if !ok {
		return
	}
	novo, err := catalogo.CriarVendedor(r.Context(), s.pool, entrada.Nome)
	if err != nil {
		escreverErroDeVendedor(w, r, err)
		return
	}
	escreverJSON(w, http.StatusCreated, saidaVendedor(novo))
}

// atualizarVendedor é editar, desativar e reativar: o corpo é a linha inteira.
func (s *servidor) atualizarVendedor(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	entrada, ok := s.vendedorDaRequisicao(w, r, true)
	if !ok {
		return
	}
	atualizado, err := catalogo.AtualizarVendedor(r.Context(), s.pool, r.PathValue("id"), entrada.Nome, *entrada.Ativo)
	if err != nil {
		escreverErroDeVendedor(w, r, err)
		return
	}
	escreverJSON(w, http.StatusOK, saidaVendedor(atualizado))
}

func (s *servidor) removerVendedor(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	if err := catalogo.RemoverVendedor(r.Context(), s.pool, r.PathValue("id")); err != nil {
		escreverErroDeVendedor(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// escreverErroDeVendedor: inexistente e malformado são o mesmo 404, e o nome
// duplicado é erro em linha — o 409 leva `dados.campo` como o do e-mail.
func escreverErroDeVendedor(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
	case errors.Is(err, catalogo.ErrVendedorJaCadastrado):
		erro.Escrever(r.Context(), w, err, map[string]string{"campo": "nome"})
	default:
		erro.Escrever(r.Context(), w, err, nil)
	}
}

// vendedorDaRequisicao lê o corpo, apara o nome e valida contra o teto da
// Config, contado em runas — "Livraria São João" tem os caracteres que se vê.
func (s *servidor) vendedorDaRequisicao(w http.ResponseWriter, r *http.Request, exigeAtivo bool) (entradaVendedor, bool) {
	var entrada entradaVendedor
	if err := decodificarCorpo(w, r, &entrada); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return entrada, false
	}
	entrada.Nome = strings.TrimSpace(entrada.Nome)
	maximo := s.cfg.VendedorNomeMax
	switch {
	case entrada.Nome == "":
		erro.EscreverCampo(r.Context(), w, "nome", fmt.Sprintf("Informe o nome do Vendedor, com até %d caracteres.", maximo))
		return entrada, false
	case utf8.RuneCountInString(entrada.Nome) > maximo:
		erro.EscreverCampo(r.Context(), w, "nome", fmt.Sprintf("O nome pode ter no máximo %d caracteres.", maximo))
		return entrada, false
	case exigeAtivo && entrada.Ativo == nil:
		erro.EscreverCampo(r.Context(), w, "ativo", "Informe se o Vendedor está ativo.")
		return entrada, false
	}
	return entrada, true
}
