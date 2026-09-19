package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// O teto do termo, no padrão da Config.
const buscaTermoMaxDeTeste = 100

// envelopeDaVitrine é o AD-18 com o item da loja.
type envelopeDaVitrine struct {
	Itens     []map[string]any `json:"itens"`
	Pagina    int              `json:"pagina"`
	PorPagina int              `json:"por_pagina"`
	Total     int              `json:"total"`
}

// vitrine é a matriz da 3.5 num subteste só. O `total` é conferido contra um
// count(*) da VIEW, e o invisível é um Produto próprio desativado.
func vitrine(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja da Vitrine"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Vitrine Nova"}`, admin), http.StatusCreated)
	corpoDe := func(ativo bool) string {
		b, _ := json.Marshal(map[string]any{
			"nome": "Luminária Escondida", "descricao": "", "preco_centavos": 1990, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": 3, "ativo": ativo,
		})
		return string(b)
	}
	invisivel := idDe(t, produtoCom(t, rotas, http.MethodPost, "", corpoDe(true), admin), http.StatusCreated)
	if resp := produtoCom(t, rotas, http.MethodPut, "/"+invisivel, corpoDe(false), admin); resp.Code != http.StatusOK {
		t.Fatalf("desativar = %d (%s)", resp.Code, resp.Body.String())
	}

	// O outro caso do AD-19: Produto ativo de Vendedor desativado.
	lojaFechada := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja Fechada da Vitrine"}`, admin), http.StatusCreated)
	b, _ := json.Marshal(map[string]any{
		"nome": "Abajur da Loja Fechada", "descricao": "", "preco_centavos": 2990, "imagem_url": "",
		"vendedor_id": lojaFechada, "categoria_id": categoria, "estoque_total": 3, "ativo": true,
	})
	deVendedorInativo := idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(b), admin), http.StatusCreated)
	if resp := vendedorCom(t, rotas, http.MethodPut, "/"+lojaFechada, `{"nome":"Loja Fechada da Vitrine","ativo":false}`, admin); resp.Code != http.StatusOK {
		t.Fatalf("desativar Vendedor = %d (%s)", resp.Code, resp.Body.String())
	}

	// A ordem padrão é "mais recentes" (3.9), com o desempate por id.
	quero := textoDe(t, pool, `SELECT id::text FROM catalogo.produto_visivel ORDER BY criado_em DESC, id`)
	total, _ := strconv.Atoi(textoDe(t, pool, `SELECT count(*)::text FROM catalogo.produto_visivel`)[0])
	if slices.Contains(quero, invisivel) || slices.Contains(quero, deVendedorInativo) || total != len(quero) || total == 0 {
		t.Fatalf("a VIEW tem %d linhas; desativado dentro: %v; de Vendedor desativado dentro: %v",
			total, slices.Contains(quero, invisivel), slices.Contains(quero, deVendedorInativo))
	}
	listar := func(consulta string) envelopeDaVitrine {
		t.Helper()
		resp := pegar(t, rotas, "/api/v1/produtos"+consulta)
		var env envelopeDaVitrine
		if err := json.Unmarshal(resp.Body.Bytes(), &env); resp.Code != http.StatusOK || err != nil {
			t.Fatalf("%s = %d (%s)", consulta, resp.Code, resp.Body.String())
		}
		if env.Total != total || env.Itens == nil {
			t.Fatalf("%s: envelope = %+v, quero total %d e itens não nulos", consulta, env, total)
		}
		return env
	}

	// Padrão: página 1, o tamanho da Config, e os cinco campos do cartão.
	env := listar("")
	if env.Pagina != 1 || env.PorPagina != paginaTamanhoDeTeste || len(env.Itens) != min(total, paginaTamanhoDeTeste) {
		t.Errorf("padrão = pagina %d, por_pagina %d, %d itens", env.Pagina, env.PorPagina, len(env.Itens))
	}
	for _, campo := range []string{"id", "nome", "preco_centavos", "imagem_url", "estoque_disponivel"} {
		if _, ok := env.Itens[0][campo]; !ok {
			t.Errorf("item sem %s: %v", campo, env.Itens[0])
		}
	}
	if len(env.Itens[0]) != 5 {
		t.Errorf("item com campos a mais: %v", env.Itens[0])
	}

	// Teto: rebaixado, e não recusado.
	if env := listar("?por_pagina=100"); env.PorPagina != paginaTamanhoMaxDeTeste || len(env.Itens) != min(total, paginaTamanhoMaxDeTeste) {
		t.Errorf("teto = por_pagina %d, %d itens", env.PorPagina, len(env.Itens))
	}
	// Além do total: vazio tratado, com o total real.
	if env := listar("?pagina=999"); env.Pagina != 999 || len(env.Itens) != 0 {
		t.Errorf("além do total = %+v", env)
	}
	// Inválido: erro em linha no campo.
	for _, campo := range []string{"pagina", "por_pagina"} {
		for _, valor := range []string{"0", "-1", "abc"} {
			confereErro(t, pegar(t, rotas, "/api/v1/produtos?"+campo+"="+valor), http.StatusBadRequest, "CAMPO_INVALIDO", campo)
		}
	}

	// Paginação completa: cada id visível exatamente uma vez, na ordem padrão.
	var ids []string
	for pagina := 1; ; pagina++ {
		env := listar(fmt.Sprintf("?pagina=%d&por_pagina=7", pagina))
		if len(env.Itens) == 0 {
			break
		}
		for _, item := range env.Itens {
			ids = append(ids, fmt.Sprint(item["id"]))
		}
	}
	if !slices.Equal(ids, quero) {
		t.Errorf("a paginação por 7 devolveu %d ids, quero os %d da VIEW na ordem padrão", len(ids), len(quero))
	}
}

// buscaFiltrosEOrdenacao é a matriz da API da 3.7–3.9 num subteste só. O
// oráculo de cada termo é um `strpos` sobre a VIEW — casamento literal, sem
// curinga — ordenado como a ordenação declara.
func buscaFiltrosEOrdenacao(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja da Busca"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Busca Própria"}`, admin), http.StatusCreated)
	criar := func(nome, descricao string, preco int) string {
		b, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": descricao, "preco_centavos": preco, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": 3, "ativo": true,
		})
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(b), admin), http.StatusCreated)
	}
	cafe := criar("Café Torrado Zeta", "", 1500)
	cha := criar("Chá Zeta", "Moído junto com CAFÉ", 2500)
	criar("Cupom Zeta", "desconto de 50% literal", 3000)
	criar("Cupom Zeta Falso", "desconto de 50x", 3100)
	ab := criar("Peça Zeta a_b", "", 4000)
	axb := criar("Peça Zeta axb", "", 4500)
	ultimo := criar("Peça Zeta cara", "", 5000)

	listar := func(consulta string) envelopeDaVitrine {
		t.Helper()
		resp := pegar(t, rotas, "/api/v1/produtos"+consulta)
		var env envelopeDaVitrine
		if err := json.Unmarshal(resp.Body.Bytes(), &env); resp.Code != http.StatusOK || err != nil {
			t.Fatalf("%s = %d (%s)", consulta, resp.Code, resp.Body.String())
		}
		return env
	}
	idsDe := func(env envelopeDaVitrine) []string {
		ids := []string{}
		for _, item := range env.Itens {
			ids = append(ids, fmt.Sprint(item["id"]))
		}
		return ids
	}
	literal := func(trecho string) []string {
		return textoDe(t, pool, `SELECT id::text FROM catalogo.produto_visivel
			WHERE strpos(busca_normalizada, $1) > 0 ORDER BY criado_em DESC, id`, trecho)
	}

	// Acento e caixa: "CAFE" casa com "Café" no nome e "CAFÉ" na descrição.
	if got := idsDe(listar("?termo=CAFE&categoria=" + categoria)); !slices.Equal(got, []string{cha, cafe}) {
		t.Errorf("CAFE na Categoria = %v, quero [%s %s]", got, cha, cafe)
	}
	// Curinga literal, sem Categoria: o oráculo é o Catálogo inteiro.
	for termo, trecho := range map[string]string{"50%25": "50%", "a_b": "a_b", "CAFE": "cafe", "%20%20zeta%20": "zeta"} {
		env := listar("?por_pagina=60&termo=" + termo)
		if quero := literal(trecho); !slices.Equal(idsDe(env), quero) || env.Total != len(quero) {
			t.Errorf("termo %q = %v (total %d), quero %v", termo, idsDe(env), env.Total, quero)
		}
	}
	if got := idsDe(listar("?termo=a_b&categoria=" + categoria)); !slices.Equal(got, []string{ab}) {
		t.Errorf("a_b = %v, quero só %s (e não %s)", got, ab, axb)
	}
	// Só espaços é "sem termo".
	semTermo := listar("?termo=%20%20")
	if todos := listar(""); semTermo.Total != todos.Total || semTermo.Total == 0 {
		t.Errorf("termo só com espaços: total %d, quero %d", semTermo.Total, todos.Total)
	}
	// O teto conta runas: 100 "é" passam, 101 não.
	listar("?termo=" + strings.Repeat("%C3%A9", 100))
	confereErro(t, pegar(t, rotas, "/api/v1/produtos?termo="+strings.Repeat("%C3%A9", 101)), http.StatusBadRequest, "CAMPO_INVALIDO", "termo")
	confereErro(t, pegar(t, rotas, "/api/v1/produtos?termo=caf%00e"), http.StatusBadRequest, "CAMPO_INVALIDO", "termo")

	// Combinação: a interseção, com total coerente.
	env := listar("?termo=peca&categoria=" + categoria + "&preco_min=4000&preco_max=4500&ordenacao=preco_asc")
	if got := idsDe(env); !slices.Equal(got, []string{ab, axb}) || env.Total != 2 {
		t.Errorf("combinação = %v (total %d), quero [%s %s]", got, env.Total, ab, axb)
	}
	// Categoria bem formada e inexistente é vazio; malformada é 400.
	if env := listar("?categoria=00000000-0000-7000-8000-000000000000"); env.Total != 0 || len(env.Itens) != 0 {
		t.Errorf("Categoria inexistente = %+v", env)
	}
	confereErro(t, pegar(t, rotas, "/api/v1/produtos?categoria=abc"), http.StatusBadRequest, "CAMPO_INVALIDO", "categoria")
	for _, campo := range []string{"preco_min", "preco_max"} {
		for _, valor := range []string{"-1", "1.5", "abc"} {
			confereErro(t, pegar(t, rotas, "/api/v1/produtos?"+campo+"="+valor), http.StatusBadRequest, "CAMPO_INVALIDO", campo)
		}
	}
	invertida := pegar(t, rotas, "/api/v1/produtos?preco_min=5000&preco_max=4000")
	confereErro(t, invertida, http.StatusBadRequest, "CAMPO_INVALIDO", "preco_min")
	if m := decodificar(t, invertida)["mensagem"]; m != "O preço mínimo não pode ser maior que o máximo." {
		t.Errorf("faixa invertida: mensagem %q", m)
	}
	confereErro(t, pegar(t, rotas, "/api/v1/produtos?ordenacao=xyz"), http.StatusBadRequest, "CAMPO_INVALIDO", "ordenacao")

	// "Mais recentes" é o padrão: o último Produto criado vem primeiro.
	if got := idsDe(listar("")); len(got) == 0 || got[0] != ultimo {
		t.Errorf("padrão começa em %v, quero o recém-criado %s", got, ultimo)
	}

	// Paginação estável nas três ordenações: cada id visível exatamente uma
	// vez, na ordem declarada.
	for ordenacao, ordem := range map[string]string{
		"recentes":   "criado_em DESC, id",
		"preco_asc":  "preco_centavos, id",
		"preco_desc": "preco_centavos DESC, id",
	} {
		quero := textoDe(t, pool, `SELECT id::text FROM catalogo.produto_visivel ORDER BY `+ordem)
		var ids []string
		for pagina := 1; ; pagina++ {
			env := listar(fmt.Sprintf("?ordenacao=%s&pagina=%d&por_pagina=7", ordenacao, pagina))
			if len(env.Itens) == 0 {
				break
			}
			ids = append(ids, idsDe(env)...)
		}
		if !slices.Equal(ids, quero) {
			t.Errorf("%s por 7 devolveu %d ids, quero os %d da VIEW em %s", ordenacao, len(ids), len(quero), ordem)
		}
	}

	// As Categorias da loja: sem Sessão.
	resp := pegar(t, rotas, "/api/v1/categorias")
	if !strings.Contains(resp.Body.String(), categoria) {
		t.Errorf("GET /api/v1/categorias = %d (%s), quero a Categoria %s", resp.Code, resp.Body.String(), categoria)
	}
	if len(decodificarLista(t, resp)[0]) != 2 {
		t.Errorf("Categoria com campos a mais: %s", resp.Body.String())
	}
}
