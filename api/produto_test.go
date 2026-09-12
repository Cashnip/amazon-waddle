package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// As duas funções abaixo são chamadas por TestSessaoEProduto: Postgres e
// Redis sobem uma vez só para api/, e não um par por arquivo de teste.

func produtoSemeadoSai(t *testing.T, rotas http.Handler) {
	resp := pegar(t, rotas, "/api/v1/produtos/"+produtoSemeado)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d (%s), quero 200", resp.Code, resp.Body.String())
	}
	corpo := decodificar(t, resp)

	if corpo["nome"] != "Fone de Ouvido Bluetooth Aurora" {
		t.Errorf("nome = %v", corpo["nome"])
	}
	// O preço sai cru, em centavos: não há divisão no caminho monetário
	// (AD-3), e o JSON não pode trazer 249.90.
	if preco, ok := corpo["preco_centavos"].(float64); !ok || preco != 24990 {
		t.Errorf("preco_centavos = %v, quero 24990 inteiro", corpo["preco_centavos"])
	}
	if strings.Contains(resp.Body.String(), ".") && strings.Contains(resp.Body.String(), "249.9") {
		t.Error("o preço saiu em ponto flutuante")
	}
	// Relativa: quem serve o byte é GET /api/v1/media/{arquivo}, do próprio
	// binário — nada é buscado em rede (AD-12).
	if url, _ := corpo["imagem_url"].(string); !strings.HasPrefix(url, "/api/v1/media/") {
		t.Errorf("imagem_url = %v, quero relativa sob /api/v1/media/", corpo["imagem_url"])
	}
	// O nome do Vendedor é o motivo do JOIN: ele mora em catalogo.vendedor.
	if corpo["vendedor"] != "Atlântico Importados" {
		t.Errorf("vendedor = %v", corpo["vendedor"])
	}
	// O Postgres aceita o uuid em maiúsculas, mas o DTO devolve a forma
	// canônica da linha — nunca o texto que veio na rota.
	maiusculas := pegar(t, rotas, "/api/v1/produtos/"+strings.ToUpper(produtoSemeado))
	if maiusculas.Code != http.StatusOK {
		t.Fatalf("uuid em maiúsculas: status = %d, quero 200", maiusculas.Code)
	}
	if id := decodificar(t, maiusculas)["id"]; id != produtoSemeado {
		t.Errorf("id = %v, quero %q em forma canônica", id, produtoSemeado)
	}

	// busca_normalizada é dado de índice e nunca vai no DTO.
	if _, tem := corpo["busca_normalizada"]; tem {
		t.Error("busca_normalizada vazou no DTO")
	}
}

func produtoAusenteDa404(t *testing.T, rotas http.Handler) {
	// Identificador malformado é o mesmo 404 do Produto inexistente — e
	// nunca 500, que é o que um Scan de uuid sem guarda produziria.
	for _, caminho := range []string{
		"/api/v1/produtos/abc",
		"/api/v1/produtos/00000000-0000-5000-8000-000000000000",
	} {
		resp := pegar(t, rotas, caminho)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, quero 404", caminho, resp.Code)
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != "NAO_ENCONTRADO" {
			t.Errorf("%s: codigo = %v", caminho, codigo)
		}
	}
}

func pegar(t *testing.T, rotas http.Handler, caminho string) *httptest.ResponseRecorder {
	t.Helper()
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, caminho, nil))
	return resp
}
