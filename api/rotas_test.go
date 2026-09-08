package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSaudeResponde200(t *testing.T) {
	resp := httptest.NewRecorder()
	Rotas().ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/saude", nil))

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, quero 200", resp.Code)
	}
	if resp.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Errorf("corpo = %q", resp.Body.String())
	}
	if resp.Header().Get("X-Correlation-Id") == "" {
		t.Error("resposta sem X-Correlation-Id")
	}
}

// Rota inexistente sai no envelope do AD-14, e não no texto puro do ServeMux.
func TestRotaInexistenteSaiNoEnvelope(t *testing.T) {
	resp := httptest.NewRecorder()
	Rotas().ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/nao-existe", nil))

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, quero 404", resp.Code)
	}
	var env struct {
		Erro struct {
			Codigo     string `json:"codigo"`
			Correlacao string `json:"correlacao"`
		} `json:"erro"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("corpo não é o envelope: %v", err)
	}
	if env.Erro.Codigo != "NAO_ENCONTRADO" {
		t.Errorf("codigo = %q", env.Erro.Codigo)
	}
	if env.Erro.Correlacao == "" {
		t.Error("envelope sem correlação")
	}
}

// Linha da matriz de I/O: dois handlers no mesmo padrão derrubam o arranque —
// falha de subida, e não de comportamento (AD-16).
func TestPadraoDuplicadoEntraEmPanico(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("registrar o mesmo padrão duas vezes devia entrar em pânico")
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/saude", saude)
	mux.HandleFunc("GET /api/v1/saude", saude)
}
