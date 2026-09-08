package plataforma

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A correlação recebida atravessa cabeçalho, contexto e log — o mesmo valor nos três.
func TestCorrelacaoRecebidaAtravessaTudo(t *testing.T) {
	var saida bytes.Buffer
	logger := NovoLogger(&saida, "api")

	var doContexto string
	h := Correlacao(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		doContexto = CorrelacaoDe(r.Context())
		logger.InfoContext(r.Context(), "requisição")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/saude", nil)
	req.Header.Set(CabecalhoCorrelacao, "abc")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if got := resp.Header().Get(CabecalhoCorrelacao); got != "abc" {
		t.Errorf("cabeçalho da resposta = %q, quero %q", got, "abc")
	}
	if doContexto != "abc" {
		t.Errorf("contexto = %q, quero %q", doContexto, "abc")
	}
	linha := decodificar(t, saida.String())
	if linha["correlacao"] != "abc" {
		t.Errorf("log correlacao = %v, quero abc", linha["correlacao"])
	}
	if linha["modulo"] != "api" {
		t.Errorf("log modulo = %v, quero api", linha["modulo"])
	}
	if n := strings.Count(saida.String(), `"correlacao"`); n != 1 {
		t.Errorf("correlacao aparece %d vezes na linha; quero 1", n)
	}
}

// Sem cabeçalho, o middleware gera um identificador e o devolve.
func TestCorrelacaoAusenteEhGerada(t *testing.T) {
	h := Correlacao(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if CorrelacaoDe(r.Context()) != w.Header().Get(CabecalhoCorrelacao) {
			t.Error("contexto e cabeçalho divergem")
		}
	}))
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/saude", nil))

	id := resp.Header().Get(CabecalhoCorrelacao)
	if len(id) != 36 || strings.Count(id, "-") != 4 {
		t.Errorf("identificador gerado = %q, não parece um UUID", id)
	}
	outro := httptest.NewRecorder()
	h.ServeHTTP(outro, httptest.NewRequest(http.MethodGet, "/api/v1/saude", nil))
	if outro.Header().Get(CabecalhoCorrelacao) == id {
		t.Error("duas requisições receberam a mesma correlação")
	}
}

// Fora de requisição a linha continua tendo correlação — nenhuma linha sem ela.
func TestLogForaDeRequisicaoTemCorrelacao(t *testing.T) {
	var saida bytes.Buffer
	NovoLogger(&saida, "arranque").Info("servindo")
	if linha := decodificar(t, saida.String()); linha["correlacao"] != SemCorrelacao {
		t.Errorf("correlacao = %v, quero %q", linha["correlacao"], SemCorrelacao)
	}
}

func TestComModuloNaoDuplicaChave(t *testing.T) {
	var saida bytes.Buffer
	ComModulo(NovoLogger(&saida, "plataforma"), "pedido").Info("oi")
	if n := strings.Count(saida.String(), `"modulo"`); n != 1 {
		t.Fatalf("modulo aparece %d vezes; quero 1: %s", n, saida.String())
	}
	if linha := decodificar(t, saida.String()); linha["modulo"] != "pedido" {
		t.Errorf("modulo = %v, quero pedido", linha["modulo"])
	}
}

// O valor do cliente é ecoado e logado: o que não serve é trocado por um novo.
func TestCorrelacaoInvalidaEhTrocada(t *testing.T) {
	casos := map[string]string{
		"longa demais":          strings.Repeat("a", tamanhoMaxCorrelacao+1),
		"caractere de controle": "abc\ndef",
		"espaço":                "abc def",
	}
	for nome, valor := range casos {
		t.Run(nome, func(t *testing.T) {
			h := Correlacao(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/saude", nil)
			req.Header[CabecalhoCorrelacao] = []string{valor} // sem a higiene do Set
			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, req)

			if got := resp.Header().Get(CabecalhoCorrelacao); got == valor {
				t.Errorf("correlação inválida foi ecoada: %q", got)
			} else if len(got) != 36 {
				t.Errorf("não veio um identificador gerado: %q", got)
			}
		})
	}
}

func decodificar(t *testing.T, linha string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(linha)), &m); err != nil {
		t.Fatalf("log não é JSON (%v): %s", err, linha)
	}
	return m
}
