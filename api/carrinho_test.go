package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// O teto por Item, no padrão da Config.
const carrinhoUnidadesMaxDeTeste = 10

// carrinhoDoComprador é a matriz de API da 4.1 num subteste só, com conta
// própria.
func carrinhoDoComprador(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	// A contagem é relativa: o negacaoPorDono, acima, deixa um Item do dono.
	total := func() int {
		n, _ := strconv.Atoi(textoDe(t, pool, `SELECT count(*)::text FROM carrinho.item_carrinho`)[0])
		return n
	}
	antes := total()
	contar := func() int { return total() - antes }
	quantidadeNoBanco := func(itemID string) []string {
		return textoDe(t, pool, `SELECT quantidade::text FROM carrinho.item_carrinho WHERE id = $1`, itemID)
	}

	if resp := postarItem(t, rotas, corpoItem(produtoSemeado, "3"), nil); resp.Code != http.StatusUnauthorized {
		t.Fatalf("sem Sessão = %d (%s), quero 401", resp.Code, resp.Body.String())
	} else if codigo := decodificar(t, resp)["codigo"]; codigo != "SESSAO_INVALIDA" {
		t.Errorf("sem Sessão: codigo = %v, quero SESSAO_INVALIDA", codigo)
	}
	if n := contar(); n != 0 {
		t.Fatalf("sem Sessão gravou: %d Itens", n)
	}

	cookie := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Lia Carrinho","email":"lia.carrinho@exemplo.br","senha":"senha-da-lia-1"}`), http.StatusCreated)

	// Primeira adição: o Carrinho e o Item nascem, com o preço atual visto.
	primeira := postarItem(t, rotas, corpoItem(produtoSemeado, "3"), cookie)
	item := idDe(t, primeira, http.StatusCreated)
	if corpo := decodificar(t, primeira); corpo["quantidade"] != 3.0 || corpo["produto_id"] != produtoSemeado {
		t.Errorf("primeira adição = %v, quero quantidade 3 do Produto semeado", corpo)
	}
	if v := primeira.Header().Get("Cache-Control"); v != "no-store" {
		t.Errorf("Cache-Control = %q, quero no-store", v)
	}
	visto := textoDe(t, pool, `SELECT preco_visto_centavos::text FROM carrinho.item_carrinho WHERE id = $1`, item)
	preco := textoDe(t, pool, `SELECT preco_centavos::text FROM catalogo.produto WHERE id = $1`, produtoSemeado)
	if len(visto) != 1 || visto[0] != preco[0] {
		t.Errorf("preco_visto_centavos = %v, quero o preço atual %v", visto, preco)
	}

	// Repetida: soma ao mesmo Item, uma linha só.
	repetida := postarItem(t, rotas, corpoItem(produtoSemeado, "2"), cookie)
	if id := idDe(t, repetida, http.StatusCreated); id != item {
		t.Errorf("a repetida criou outro Item: %s, quero %s", id, item)
	}
	if q := decodificar(t, repetida)["quantidade"]; q != 5.0 {
		t.Errorf("quantidade = %v, quero 5", q)
	}
	if n := contar(); n != 1 {
		t.Errorf("%d Itens no banco, quero 1", n)
	}

	// Soma acima do teto: chega a 9, e +2 não grava.
	idDe(t, postarItem(t, rotas, corpoItem(produtoSemeado, "4"), cookie), http.StatusCreated)
	acima := postarItem(t, rotas, corpoItem(produtoSemeado, "2"), cookie)
	querCampoQuantidade(t, "soma acima do teto", acima)
	if q := quantidadeNoBanco(item); len(q) != 1 || q[0] != "9" {
		t.Errorf("a soma recusada gravou: quantidade = %v, quero 9", q)
	}

	// Quantidade fora da faixa, ausente ou não inteira.
	for _, q := range []string{"0", "11", "-1", "1.5", `"3"`, "null"} {
		querCampoQuantidade(t, "quantidade "+q, postarItem(t, rotas, corpoItem(produtoSemeado, q), cookie))
	}
	querCampoQuantidade(t, "quantidade ausente",
		postarItem(t, rotas, `{"produto_id":"`+produtoSemeado+`"}`, cookie))

	// Produto invisível: desativado, inexistente e uuid malformado.
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja do Carrinho"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Categoria do Carrinho"}`, admin), http.StatusCreated)
	corpoDe := func(ativo bool) string {
		b, _ := json.Marshal(map[string]any{
			"nome": "Chaleira Recolhida", "descricao": "", "preco_centavos": 4990, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": 3, "ativo": ativo,
		})
		return string(b)
	}
	desativado := idDe(t, produtoCom(t, rotas, http.MethodPost, "", corpoDe(true), admin), http.StatusCreated)
	if resp := produtoCom(t, rotas, http.MethodPut, "/"+desativado, corpoDe(false), admin); resp.Code != http.StatusOK {
		t.Fatalf("desativar = %d (%s)", resp.Code, resp.Body.String())
	}
	for _, id := range []string{desativado, uuidNuncaUsado, "nao-e-uuid"} {
		resp := postarItem(t, rotas, corpoItem(id, "1"), cookie)
		if resp.Code != http.StatusNotFound {
			t.Errorf("Produto %s = %d (%s), quero 404", id, resp.Code, resp.Body.String())
		} else if codigo := decodificar(t, resp)["codigo"]; codigo != "NAO_ENCONTRADO" {
			t.Errorf("Produto %s: codigo = %v, quero NAO_ENCONTRADO", id, codigo)
		}
	}
	if n := contar(); n != 1 {
		t.Errorf("a recusa gravou: %d Itens, quero 1", n)
	}

	// Remover o próprio: 204, e a linha some.
	if resp := deletarItem(t, rotas, item, cookie); resp.Code != http.StatusNoContent {
		t.Fatalf("remover o próprio = %d (%s), quero 204", resp.Code, resp.Body.String())
	}
	if q := quantidadeNoBanco(item); len(q) != 0 {
		t.Errorf("o Item removido continua no banco: %v", q)
	}
	if resp := deletarItem(t, rotas, item, cookie); resp.Code != http.StatusNotFound {
		t.Errorf("remover de novo = %d (%s), quero 404", resp.Code, resp.Body.String())
	}
}

func querCampoQuantidade(t *testing.T, caso string, resp *httptest.ResponseRecorder) {
	t.Helper()
	if resp.Code != http.StatusBadRequest {
		t.Errorf("%s = %d (%s), quero 400", caso, resp.Code, resp.Body.String())
		return
	}
	dados, _ := decodificar(t, resp)["dados"].(map[string]any)
	if dados["campo"] != "quantidade" {
		t.Errorf("%s: dados = %v, quero o campo quantidade", caso, dados)
	}
}

func corpoItem(produtoID, quantidade string) string {
	return `{"produto_id":"` + produtoID + `","quantidade":` + quantidade + `}`
}

func postarItem(t *testing.T, rotas http.Handler, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodPost, "/api/v1/carrinho/itens", corpo, "application/json", cookie)
}

// deletarItem tem a assinatura de pegarPedido, para entrar na tabela de
// negacaoPorDono.
func deletarItem(t *testing.T, rotas http.Handler, id string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodDelete, "/api/v1/carrinho/itens/"+id, "", "", cookie)
}
