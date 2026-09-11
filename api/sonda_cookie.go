package api

import "net/http"

// sondaCookie existe por um motivo só: provar a verificação 1 da épica — que
// o `Set-Cookie` do Go atravessa o `rewrites()` do Next íntegro, e que a saída
// declarada (proxy explícito no lugar do rewrite) não precisa ser acionada.
// O cookie é descartável, não carrega nada e não é lido por ninguém.
//
// A estória 1.5 REMOVE este arquivo e o registro dele em rotas.go ao entregar
// a Sessão de verdade, que passa a ser quem exercita o mesmo caminho.
func sondaCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "azamon_sonda",
		Value:    "atravessou",
		Path:     "/",
		MaxAge:   60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(`{"sonda":"cookie emitido"}` + "\n"))
}
