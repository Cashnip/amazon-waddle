package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// categoriaNomeMaxDeTeste é o padrão de AZAMON_CATEGORIA_NOME_MAX.
const categoriaNomeMaxDeTeste = 120

// gestaoDeCategorias é a matriz da 3.2 num subteste só. As Categorias são
// criadas aqui: nenhuma semeada é renomeada nem removida.
func gestaoDeCategorias(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Caio Prado","email":"caio@exemplo.br","senha":"senha-do-caio-1"}`), http.StatusCreated)

	// Sem Sessão de Administrador, toda rota nova é o 404 da guarda.
	for _, cookie := range []*http.Cookie{nil, comprador} {
		for _, resp := range []*httptest.ResponseRecorder{
			pegarCom(t, rotas, "/api/v1/admin/categorias", cookie),
			categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Intrusa"}`, cookie),
			categoriaCom(t, rotas, http.MethodPut, "/"+categoriaSemeada, `{"nome":"Intrusa"}`, cookie),
			categoriaCom(t, rotas, http.MethodDelete, "/"+categoriaSemeada, "", cookie),
		} {
			if resp.Code != http.StatusNotFound {
				t.Fatalf("sem Sessão de Administrador = %d (%s), quero 404", resp.Code, resp.Body.String())
			}
		}
	}

	// Listar: as semeadas, na ordem do banco, só com id e nome.
	lista := pegarCom(t, rotas, "/api/v1/admin/categorias", admin)
	if v := lista.Header().Get("Cache-Control"); v != "no-store" {
		t.Errorf("Cache-Control = %q, quero no-store", v)
	}
	itens := decodificarLista(t, lista)
	if len(itens) == 0 || len(itens[0]) != 2 {
		t.Fatalf("lista = %v, quero só {id,nome}", itens)
	}
	if nomes, quero := nomesDe(t, pegarCom(t, rotas, "/api/v1/admin/categorias", admin)),
		textoDe(t, pool, `SELECT nome FROM catalogo.categoria ORDER BY nome, id`); !slices.Equal(nomes, quero) {
		t.Errorf("lista = %v, quero %v", nomes, quero)
	}

	// Criar apara as bordas e não expõe `categoria_pai_id`.
	criada := categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"  Jardim  "}`, admin)
	if criada.Code != http.StatusCreated {
		t.Fatalf("criar = %d (%s), quero 201", criada.Code, criada.Body.String())
	}
	if c := decodificar(t, criada); c["nome"] != "Jardim" || len(c) != 2 {
		t.Errorf("criada = %v, quero {id,nome:Jardim}", c)
	}
	jardim := idDe(t, criada, http.StatusCreated)
	pesca := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Pesca"}`, admin), http.StatusCreated)

	// Vazio e acima do teto (em runas): 400 em linha, com o limite.
	for _, nome := range []string{"   ", strings.Repeat("é", categoriaNomeMaxDeTeste+1)} {
		resp := categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"`+nome+`"}`, admin)
		confereErro(t, resp, http.StatusBadRequest, "CAMPO_INVALIDO", "nome")
		if msg := fmt.Sprint(decodificar(t, resp)["mensagem"]); !strings.Contains(msg, fmt.Sprint(categoriaNomeMaxDeTeste)) {
			t.Errorf("mensagem = %q, quero o limite %d", msg, categoriaNomeMaxDeTeste)
		}
	}
	confereErro(t, categoriaCom(t, rotas, http.MethodPut, "/"+jardim, `{"nome":""}`, admin),
		http.StatusBadRequest, "CAMPO_INVALIDO", "nome")

	// Duplicado, no criar e no renomear: 409 com o campo.
	confereErro(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Jardim"}`, admin),
		http.StatusConflict, "CATEGORIA_JA_CADASTRADA", "nome")
	confereErro(t, categoriaCom(t, rotas, http.MethodPut, "/"+pesca, `{"nome":"Jardim"}`, admin),
		http.StatusConflict, "CATEGORIA_JA_CADASTRADA", "nome")

	// Renomear: 200 com a linha nova.
	renomeada := categoriaCom(t, rotas, http.MethodPut, "/"+jardim, `{"nome":"Jardim e Varanda"}`, admin)
	if c := decodificar(t, renomeada); renomeada.Code != http.StatusOK || c["nome"] != "Jardim e Varanda" || c["id"] != jardim {
		t.Errorf("renomear = %d %v", renomeada.Code, c)
	}

	// Inexistente e malformado: 404 no PUT e no DELETE.
	for _, id := range []string{"/00000000-0000-7000-8000-000000000000", "/nao-e-uuid"} {
		confereErro(t, categoriaCom(t, rotas, http.MethodPut, id, `{"nome":"Qualquer"}`, admin),
			http.StatusNotFound, "NAO_ENCONTRADO", "")
		confereErro(t, categoriaCom(t, rotas, http.MethodDelete, id, "", admin),
			http.StatusNotFound, "NAO_ENCONTRADO", "")
	}

	// Remover sem Produto: 204, e de novo é 404.
	if resp := categoriaCom(t, rotas, http.MethodDelete, "/"+pesca, "", admin); resp.Code != http.StatusNoContent {
		t.Fatalf("remover = %d (%s), quero 204", resp.Code, resp.Body.String())
	}
	confereErro(t, categoriaCom(t, rotas, http.MethodDelete, "/"+pesca, "", admin), http.StatusNotFound, "NAO_ENCONTRADO", "")

	// Com N Produtos: 409, a mensagem nomeia N e `dados.produtos = N`.
	for n := 1; n <= 2; n++ {
		textoDe(t, pool, `
			INSERT INTO catalogo.produto (nome, descricao, preco_centavos, imagem_url, vendedor_id, categoria_id)
			SELECT 'Produto do Jardim', 'Descrição.', 1000, '', id, $1::uuid FROM catalogo.vendedor ORDER BY nome LIMIT 1
			RETURNING id::text`, jardim)
		resp := categoriaCom(t, rotas, http.MethodDelete, "/"+jardim, "", admin)
		confereErro(t, resp, http.StatusConflict, "CATEGORIA_COM_PRODUTOS", "")
		envelope := decodificar(t, resp)
		quero := "1 Produto vinculado"
		if n == 2 {
			quero = "2 Produtos vinculados"
		}
		if msg := fmt.Sprint(envelope["mensagem"]); !strings.Contains(msg, quero) {
			t.Errorf("mensagem = %q, quero %q", msg, quero)
		}
		if dados, _ := envelope["dados"].(map[string]any); dados["produtos"] != float64(n) {
			t.Errorf("dados = %v, quero produtos %d", envelope["dados"], n)
		}
	}
}

func categoriaCom(t *testing.T, rotas http.Handler, metodo, sufixo, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	tipo := ""
	if corpo != "" {
		tipo = "application/json"
	}
	return comCorpo(t, rotas, metodo, "/api/v1/admin/categorias"+sufixo, corpo, tipo, cookie)
}
