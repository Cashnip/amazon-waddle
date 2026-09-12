// Package api é só tradução: rota, middleware, DTO e os limites do NFR-14.
// Nenhuma regra de domínio mora aqui, e nenhum módulo de domínio conhece HTTP.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Cashnip/amazon-waddle/internal/plataforma"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// servidor carrega as dependências que os handlers precisam. É struct, e não
// variável de pacote, para que o teste monte a sua sem tocar em estado global.
type servidor struct {
	cfg  plataforma.Config
	pool *pgxpool.Pool
	rdb  *redis.Client
}

// Rotas monta o ServeMux da biblioteca padrão. Dois handlers no mesmo padrão
// fazem o binário entrar em pânico no arranque — falha de subida, e não de
// comportamento, que é como o AD-16 quer a posse de rota.
func Rotas(cfg plataforma.Config, pool *pgxpool.Pool, rdb *redis.Client) http.Handler {
	s := &servidor{cfg: cfg, pool: pool, rdb: rdb}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/saude", saude)
	mux.HandleFunc("GET /api/v1/media/{arquivo}", midia)
	// A Sessão é o recurso criado, e por isso o plural do POST. O singular lê
	// a Sessão corrente — é o que o menu da conta da Épica 2 reaproveita.
	mux.HandleFunc("POST /api/v1/sessoes", s.criarSessao)
	mux.HandleFunc("GET /api/v1/sessao", s.lerSessao)
	// Só o detalhe: a listagem (GET /api/v1/produtos) é de `busca`, na Épica 3.
	mux.HandleFunc("GET /api/v1/produtos/{id}", s.detalheDoProduto)
	// Sem isto o ServeMux responderia "404 page not found" em texto puro, e a
	// API teria dois contratos de erro conforme a rota exista ou não (AD-14).
	mux.HandleFunc("/", naoEncontrado)
	return plataforma.Correlacao(mux)
}

func naoEncontrado(w http.ResponseWriter, r *http.Request) {
	erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
}

// escreverJSON é o único lugar que serializa resposta de sucesso. O erro de
// codificação não vira envelope: o cabeçalho já foi, e o cliente já está lendo.
func escreverJSON(w http.ResponseWriter, valor any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(valor)
}
