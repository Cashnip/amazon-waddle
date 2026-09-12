package api

import (
	"encoding/json"
	"net/http"

	"github.com/Cashnip/amazon-waddle/internal/identidade"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// corpoMaximo é higiene de entrada (NFR-14), não limiar de negócio: o login
// tem dois campos curtos, e nada justifica ler mais do que isto de um corpo
// que ainda não foi autenticado.
const corpoMaximo = 4 << 10

type entradaSessao struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

// saidaSessao é o que a casca precisa para saudar o Comprador. O identificador
// fica na Sessão, dentro do Redis — o navegador não tem o que fazer com ele.
type saidaSessao struct {
	Nome string `json:"nome"`
}

// criarSessao autentica e emite o cookie opaco. O cookie é `HttpOnly` para
// que script nenhum o leia, e `SameSite=Lax` porque a casca do Next e o Go
// vivem na mesma origem — quem atravessa é o rewrites(), não o navegador.
func (s *servidor) criarSessao(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	var entrada entradaSessao
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, corpoMaximo)).Decode(&entrada); err != nil {
		erro.Escrever(r.Context(), w, erro.ErrEntradaInvalida, nil)
		return
	}

	comprador, err := identidade.Autenticar(r.Context(), s.pool, entrada.Email, entrada.Senha)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	token, err := identidade.CriarSessao(r.Context(), s.rdb, comprador, s.cfg.SessaoExpiracao)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     identidade.NomeCookieSessao,
		Value:    token,
		Path:     "/",
		MaxAge:   int(s.cfg.SessaoExpiracao.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	escreverJSON(w, saidaSessao{Nome: comprador.Nome})
}

// semCache: as duas respostas de Sessão são por Comprador. Sem a diretiva, um
// intermediário pode guardá-las por heurística e servir o nome de um a outro.
func semCache(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }

// lerSessao resolve o cookie. Ausente, adulterado ou fora do Redis saem todos
// como 401 — para quem chama, expirada e inexistente são a mesma coisa.
func (s *servidor) lerSessao(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	cookie, err := r.Cookie(identidade.NomeCookieSessao)
	if err != nil {
		erro.Escrever(r.Context(), w, identidade.ErrSessaoInvalida, nil)
		return
	}
	comprador, err := identidade.LerSessao(r.Context(), s.rdb, cookie.Value)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, saidaSessao{Nome: comprador.Nome})
}
