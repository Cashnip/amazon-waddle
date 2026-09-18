package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

	quero := textoDe(t, pool, `SELECT id::text FROM catalogo.produto_visivel ORDER BY id`)
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

	// Paginação completa: cada id visível exatamente uma vez, na ordem do id.
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
		t.Errorf("a paginação por 7 devolveu %d ids, quero os %d da VIEW na ordem do id", len(ids), len(quero))
	}
}
