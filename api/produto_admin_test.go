package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
)

// Os tetos de Produto e o tamanho da página, nos padrões da Config: o que se
// confere é que o limiar configurado chega inteiro à mensagem.
const (
	produtoNomeMaxDeTeste      = 200
	produtoDescricaoMaxDeTeste = 4000
	produtoPrecoMaxDeTeste     = 100000000
	produtoEstoqueMaxDeTeste   = 100000
	paginaTamanhoDeTeste       = 20
	paginaTamanhoMaxDeTeste    = 60
)

// gestaoDeProdutos é a matriz da 3.3 num subteste só. Vendedor, Categoria e
// Produtos são criados aqui: nenhum semeado é alterado.
func gestaoDeProdutos(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Lia Souza","email":"lia@exemplo.br","senha":"senha-da-lia-1"}`), http.StatusCreated)

	// Sem Sessão de Administrador, toda rota nova é o 404 da guarda.
	for _, cookie := range []*http.Cookie{nil, comprador} {
		for _, resp := range []*httptest.ResponseRecorder{
			pegarCom(t, rotas, "/api/v1/admin/produtos", cookie),
			pegarCom(t, rotas, "/api/v1/admin/midias", cookie),
			produtoCom(t, rotas, http.MethodPost, "", `{}`, cookie),
			produtoCom(t, rotas, http.MethodPut, "/"+produtoSemeado, `{}`, cookie),
		} {
			if resp.Code != http.StatusNotFound {
				t.Fatalf("sem Sessão de Administrador = %d (%s), quero 404", resp.Code, resp.Body.String())
			}
		}
	}

	// Mídias: as URLs de media/, em ordem.
	resp := pegarCom(t, rotas, "/api/v1/admin/midias", admin)
	var midias []string
	if err := json.Unmarshal(resp.Body.Bytes(), &midias); resp.Code != http.StatusOK || err != nil {
		t.Fatalf("mídias = %d (%s)", resp.Code, resp.Body.String())
	}
	imagem := "/api/v1/media/air-fryer-4-litros.svg"
	if !slices.IsSorted(midias) || !slices.Contains(midias, imagem) {
		t.Errorf("mídias = %v, quero em ordem e com %s", midias, imagem)
	}

	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja do Produto"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Cozinha Nova"}`, admin), http.StatusCreated)
	valido := func(mudar func(map[string]any)) string {
		c := map[string]any{
			"nome": "  Panela de Pressão  ", "descricao": " Aço inox, 6 litros. ", "preco_centavos": 12990,
			"imagem_url": imagem, "vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": 7,
		}
		if mudar != nil {
			mudar(c)
		}
		b, _ := json.Marshal(c)
		return string(b)
	}

	// Criar: 201 com Vendedor e Categoria, e a Página de Produto já responde.
	criado := produtoCom(t, rotas, http.MethodPost, "", valido(nil), admin)
	produto := idDe(t, criado, http.StatusCreated)
	p := decodificar(t, criado)
	if p["nome"] != "Panela de Pressão" || p["descricao"] != "Aço inox, 6 litros." || p["preco_centavos"] != float64(12990) ||
		p["estoque_total"] != float64(7) || p["ativo"] != true || p["imagem_url"] != imagem {
		t.Errorf("criado = %v", p)
	}
	if v, _ := p["vendedor"].(map[string]any); v["id"] != vendedor || v["nome"] != "Loja do Produto" {
		t.Errorf("vendedor = %v", p["vendedor"])
	}
	if c, _ := p["categoria"].(map[string]any); c["id"] != categoria || c["nome"] != "Cozinha Nova" {
		t.Errorf("categoria = %v", p["categoria"])
	}
	if resp := pegar(t, rotas, "/api/v1/produtos/"+produto); resp.Code != http.StatusOK {
		t.Fatalf("Página de Produto = %d, quero 200", resp.Code)
	}
	normalizada := func() string {
		return textoDe(t, pool, `SELECT busca_normalizada FROM catalogo.produto WHERE id = $1::uuid`, produto)[0]
	}
	if got := normalizada(); got != catalogo.NormalizarBusca("Panela de Pressão", "Aço inox, 6 litros.") {
		t.Errorf("busca_normalizada = %q", got)
	}

	// "Sem imagem" e os limites exatos entram.
	noTeto := valido(func(c map[string]any) {
		c["imagem_url"], c["nome"], c["descricao"] = "", strings.Repeat("é", produtoNomeMaxDeTeste), strings.Repeat("é", produtoDescricaoMaxDeTeste)
		c["preco_centavos"], c["estoque_total"] = produtoPrecoMaxDeTeste, produtoEstoqueMaxDeTeste
	})
	idDe(t, produtoCom(t, rotas, http.MethodPost, "", noTeto, admin), http.StatusCreated)

	// Cada campo fora do limite: 400 em linha, com o campo e — quando há — o limite.
	inexistente := "00000000-0000-7000-8000-000000000000"
	casos := []struct {
		campo, limite string
		mudar         func(map[string]any)
	}{
		{"nome", "200", func(c map[string]any) { c["nome"] = "   " }},
		{"nome", "200", func(c map[string]any) { c["nome"] = strings.Repeat("é", produtoNomeMaxDeTeste+1) }},
		{"descricao", "4.000", func(c map[string]any) { c["descricao"] = strings.Repeat("é", produtoDescricaoMaxDeTeste+1) }},
		{"preco_centavos", "1.000.000,00", func(c map[string]any) { c["preco_centavos"] = 0 }},
		{"preco_centavos", "1.000.000,00", func(c map[string]any) { c["preco_centavos"] = -5 }},
		{"preco_centavos", "1.000.000,00", func(c map[string]any) { c["preco_centavos"] = produtoPrecoMaxDeTeste + 1 }},
		{"estoque_total", "100.000", func(c map[string]any) { c["estoque_total"] = -1 }},
		{"estoque_total", "100.000", func(c map[string]any) { c["estoque_total"] = produtoEstoqueMaxDeTeste + 1 }},
		{"estoque_total", "100.000", func(c map[string]any) { delete(c, "estoque_total") }},
		{"imagem_url", "", func(c map[string]any) { c["imagem_url"] = "/api/v1/media/nao-existe.svg" }},
		{"imagem_url", "", func(c map[string]any) { c["imagem_url"] = "http://exemplo.br/x.svg" }},
		{"vendedor_id", "", func(c map[string]any) { c["vendedor_id"] = inexistente }},
		{"vendedor_id", "", func(c map[string]any) { c["vendedor_id"] = "nao-e-uuid" }},
		{"categoria_id", "", func(c map[string]any) { c["categoria_id"] = inexistente }},
		{"categoria_id", "", func(c map[string]any) { c["categoria_id"] = "nao-e-uuid" }},
	}
	for _, caso := range casos {
		resp := produtoCom(t, rotas, http.MethodPost, "", valido(caso.mudar), admin)
		confereErro(t, resp, http.StatusBadRequest, "CAMPO_INVALIDO", caso.campo)
		if msg := fmt.Sprint(decodificar(t, resp)["mensagem"]); !strings.Contains(msg, caso.limite) {
			t.Errorf("%s: mensagem = %q, quero o limite %s", caso.campo, msg, caso.limite)
		}
	}
	// As mesmas validações no editar, e o `ativo` é obrigatório.
	for _, caso := range casos {
		if caso.campo == "estoque_total" {
			continue // o editar ignora o Estoque total
		}
		corpo := valido(func(c map[string]any) { c["ativo"] = true; caso.mudar(c) })
		confereErro(t, produtoCom(t, rotas, http.MethodPut, "/"+produto, corpo, admin), http.StatusBadRequest, "CAMPO_INVALIDO", caso.campo)
	}
	confereErro(t, produtoCom(t, rotas, http.MethodPut, "/"+produto, valido(nil), admin), http.StatusBadRequest, "CAMPO_INVALIDO", "ativo")
	for _, id := range []string{"/" + inexistente, "/nao-e-uuid"} {
		confereErro(t, produtoCom(t, rotas, http.MethodPut, id, valido(func(c map[string]any) { c["ativo"] = true }), admin),
			http.StatusNotFound, "NAO_ENCONTRADO", "")
	}

	// Preço congelado: o Pedido feito antes da edição é lido idêntico.
	pedidoID := idDe(t, pedidoPeloCheckout(t, rotas, comprador, produto, 1), http.StatusCreated)
	pedidoAntes := corpoSemCorrelacao(pegarPedido(t, rotas, pedidoID, comprador))
	estoqueAntes := textoDe(t, pool, `SELECT estoque_total::text FROM catalogo.produto WHERE id = $1::uuid`, produto)[0]
	editado := produtoCom(t, rotas, http.MethodPut, "/"+produto, valido(func(c map[string]any) {
		c["nome"], c["preco_centavos"], c["ativo"], c["estoque_total"] = "Panela Elétrica", 15990, true, 99
	}), admin)
	if e := decodificar(t, editado); editado.Code != http.StatusOK || e["nome"] != "Panela Elétrica" || e["preco_centavos"] != float64(15990) {
		t.Fatalf("editar = %d %v", editado.Code, e)
	}
	if depois := corpoSemCorrelacao(pegarPedido(t, rotas, pedidoID, comprador)); depois != pedidoAntes {
		t.Errorf("o Pedido mudou com a edição do preço:\n%s\n%s", pedidoAntes, depois)
	}
	// O Estoque total não é ajustável no editar (é da 3.4).
	if depois := textoDe(t, pool, `SELECT estoque_total::text FROM catalogo.produto WHERE id = $1::uuid`, produto)[0]; depois != estoqueAntes {
		t.Errorf("estoque_total = %s depois do editar, era %s", depois, estoqueAntes)
	}
	if got := normalizada(); got != catalogo.NormalizarBusca("Panela Elétrica", "Aço inox, 6 litros.") {
		t.Errorf("busca_normalizada depois do editar = %q", got)
	}

	// Desativar: a Página de Produto é 404, e o Produto que já estava no
	// Carrinho é recusado na criação do Pedido, sem Reserva — é a mesma falta
	// de Estoque de disponível zero (AD-19).
	if resp := postarItem(t, rotas, corpoItem(produto, "1"), comprador); resp.Code != http.StatusCreated {
		t.Fatalf("adicionar antes de desativar = %d (%s)", resp.Code, resp.Body.String())
	}
	desativar := func(ativo bool) {
		t.Helper()
		resp := produtoCom(t, rotas, http.MethodPut, "/"+produto, valido(func(c map[string]any) { c["ativo"] = ativo }), admin)
		if e := decodificar(t, resp); resp.Code != http.StatusOK || e["ativo"] != ativo {
			t.Fatalf("ativo=%v: %d %v", ativo, resp.Code, e)
		}
	}
	desativar(false)
	confereErro(t, pegar(t, rotas, "/api/v1/produtos/"+produto), http.StatusNotFound, "NAO_ENCONTRADO", "")
	reservas := func() int {
		return len(textoDe(t, pool, `SELECT id::text FROM catalogo.reserva_estoque WHERE produto_id = $1::uuid`, produto))
	}
	antes := reservas()
	confereErro(t, confirmarCarrinho(t, rotas, comprador), http.StatusConflict, "ESTOQUE_INSUFICIENTE", "")
	if n := reservas(); n != antes {
		t.Errorf("reservas = %d depois da compra recusada, eram %d", n, antes)
	}

	// Listar: o envelope, página a página, na ordem do banco e com os inativos.
	for _, pagina := range []string{"abc", "0", "-1", "1.5"} {
		confereErro(t, pegarCom(t, rotas, "/api/v1/admin/produtos?pagina="+pagina, admin), http.StatusBadRequest, "CAMPO_INVALIDO", "pagina")
	}
	quero := textoDe(t, pool, `SELECT id::text FROM catalogo.produto ORDER BY nome, id`)
	var ids []string
	var inativo map[string]any
	for pagina := 1; ; pagina++ {
		resp := pegarCom(t, rotas, fmt.Sprintf("/api/v1/admin/produtos?pagina=%d", pagina), admin)
		var env struct {
			Itens     []map[string]any `json:"itens"`
			Pagina    int              `json:"pagina"`
			PorPagina int              `json:"por_pagina"`
			Total     int              `json:"total"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &env); resp.Code != http.StatusOK || err != nil {
			t.Fatalf("listar página %d = %d (%s)", pagina, resp.Code, resp.Body.String())
		}
		if env.Pagina != pagina || env.PorPagina != paginaTamanhoDeTeste || env.Total != len(quero) || env.Itens == nil {
			t.Fatalf("envelope da página %d = %+v, total no banco %d", pagina, env, len(quero))
		}
		if len(env.Itens) == 0 {
			break
		}
		for _, item := range env.Itens {
			ids = append(ids, fmt.Sprint(item["id"]))
			if item["id"] == produto {
				inativo = item
			}
		}
	}
	if !slices.Equal(ids, quero) {
		t.Errorf("a listagem paginada não é a ordem nome, id do banco")
	}
	if inativo == nil || inativo["ativo"] != false || inativo["descricao"] == nil || inativo["estoque_total"] == nil {
		t.Errorf("o Produto inativo na lista = %v", inativo)
	}
	if c, _ := inativo["categoria"].(map[string]any); c["nome"] != "Cozinha Nova" {
		t.Errorf("categoria na lista = %v", inativo["categoria"])
	}

	// Reativar devolve a Página de Produto.
	desativar(true)
	if resp := pegar(t, rotas, "/api/v1/produtos/"+produto); resp.Code != http.StatusOK {
		t.Errorf("Produto depois de reativar = %d, quero 200", resp.Code)
	}
}

func produtoCom(t *testing.T, rotas http.Handler, metodo, sufixo, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, metodo, "/api/v1/admin/produtos"+sufixo, corpo, "application/json", cookie)
}
