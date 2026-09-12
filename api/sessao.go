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
	escreverJSON(w, http.StatusOK, saidaSessao{Nome: comprador.Nome})
}

// semCache: as respostas por Comprador não podem ser guardadas. Sem a
// diretiva, um intermediário pode fazê-lo por heurística e servir o nome — ou
// o Pedido — de um a outro.
func semCache(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }

// compradorDaRequisicao resolve o cookie de Sessão. Ausente, adulterado ou
// fora do Redis saem todos como ErrSessaoInvalida — para quem chama, expirada
// e inexistente são a mesma coisa. São duas rotas autenticadas em todo o
// sistema, e por isso isto é uma função, e não um middleware.
func (s *servidor) compradorDaRequisicao(r *http.Request) (identidade.Comprador, error) {
	cookie, err := r.Cookie(identidade.NomeCookieSessao)
	if err != nil {
		return identidade.Comprador{}, identidade.ErrSessaoInvalida
	}
	return identidade.LerSessao(r.Context(), s.rdb, cookie.Value)
}

func (s *servidor) lerSessao(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusOK, saidaSessao{Nome: comprador.Nome})
}
