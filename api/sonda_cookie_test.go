package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Linha da matriz de I/O da estória 1.2, lado Go: a sonda precisa emitir um
// Set-Cookie completo, senão o `curl` da verificação 1 mede a coisa errada e
// o cookie de Sessão da 1.5 herda uma prova que nunca existiu.
func TestSondaCookieEmiteSetCookieCompleto(t *testing.T) {
	resp := httptest.NewRecorder()
	Rotas().ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/sonda-cookie", nil))

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, quero 200", resp.Code)
	}

	cookies := resp.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("quero exatamente um cookie, vieram %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != "azamon_sonda" || c.Value == "" {
		t.Errorf("cookie = %q=%q", c.Name, c.Value)
	}
	// Os mesmos atributos que o cookie de Sessão da 1.5 vai carregar — é isso
	// que a verificação 1 precisa ver atravessar o rewrites() inteiro.
	if !c.HttpOnly {
		t.Error("cookie sem HttpOnly")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, quero Lax", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, quero /", c.Path)
	}
	// Descartável é o ponto: a sonda não pode deixar cookie de sessão do
	// navegador para trás depois que a 1.5 apagar a rota.
	if c.MaxAge <= 0 {
		t.Errorf("MaxAge = %d, quero um prazo curto e positivo", c.MaxAge)
	}
}
