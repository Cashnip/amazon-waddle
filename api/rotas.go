// Package api é só tradução: rota, middleware, DTO e os limites do NFR-14.
// Nenhuma regra de domínio mora aqui, e nenhum módulo de domínio conhece HTTP.
package api

import (
	"net/http"

	"github.com/Cashnip/amazon-waddle/internal/plataforma"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// Rotas monta o ServeMux da biblioteca padrão. Dois handlers no mesmo padrão
// fazem o binário entrar em pânico no arranque — falha de subida, e não de
// comportamento, que é como o AD-16 quer a posse de rota.
func Rotas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/saude", saude)
	// Sem isto o ServeMux responderia "404 page not found" em texto puro, e a
	// API teria dois contratos de erro conforme a rota exista ou não (AD-14).
	mux.HandleFunc("/", naoEncontrado)
	return plataforma.Correlacao(mux)
}

func naoEncontrado(w http.ResponseWriter, r *http.Request) {
	erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
}
