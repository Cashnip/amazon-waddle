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

// vendedorNomeMaxDeTeste é o padrão de AZAMON_VENDEDOR_NOME_MAX: o que se
// confere é que o limiar configurado chega inteiro à mensagem.
const vendedorNomeMaxDeTeste = 120

// categoriaSemeada é "Eletrônicos" da semente: o Produto do Vendedor novo
// precisa de uma Categoria, e o teste não cria nenhuma.
const categoriaSemeada = "cc9fbd39-f581-57bb-9a51-2b1047f864cc"

// gestaoDeVendedores é a matriz da 3.1 num subteste só: a ordem importa (o
// duplicado precisa do criado, o Pedido antigo precisa vir antes de desativar).
// Nenhum Vendedor semeado é desativado — os outros subtestes dependem deles.
func gestaoDeVendedores(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Vera Lima","email":"vera@exemplo.br","senha":"senha-da-vera-1"}`), http.StatusCreated)

	// Sem Sessão de Administrador, toda rota nova é o 404 da guarda.
	for _, cookie := range []*http.Cookie{nil, comprador} {
		for _, resp := range []*httptest.ResponseRecorder{
			pegarCom(t, rotas, "/api/v1/admin/vendedores", cookie),
			vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Intruso"}`, cookie),
			vendedorCom(t, rotas, http.MethodPut, "/33cb4ea4-41d3-5042-814a-3dc2513ea349", `{"nome":"Intruso","ativo":false}`, cookie),
			vendedorCom(t, rotas, http.MethodDelete, "/33cb4ea4-41d3-5042-814a-3dc2513ea349", "", cookie),
		} {
			if resp.Code != http.StatusNotFound {
				t.Fatalf("sem Sessão de Administrador = %d (%s), quero 404", resp.Code, resp.Body.String())
			}
		}
	}

	// Listar: os cinco semeados, por nome, com ativo.
	lista := pegarCom(t, rotas, "/api/v1/admin/vendedores", admin)
	if v := lista.Header().Get("Cache-Control"); v != "no-store" {
		t.Errorf("Cache-Control = %q, quero no-store", v)
	}
	nomes := nomesDe(t, lista)
	// A ordem esperada é literal, e não slices.IsSorted: o Go compara bytes e o
	// ORDER BY do Postgres segue a collation, que divergem em nome acentuado.
	semeados := []string{"Atlântico Importados", "Casa Boa Utilidades", "Pampa Esportes", "Sertão Livraria", "Vale do Sol Distribuidora"}
	if !slices.Equal(nomes, semeados) {
		t.Errorf("lista = %v, quero os cinco semeados por nome", nomes)
	}

	// Criar apara as bordas e nasce ativo.
	criado := vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"  Loja X  "}`, admin)
	if criado.Code != http.StatusCreated {
		t.Fatalf("criar = %d (%s), quero 201", criado.Code, criado.Body.String())
	}
	if v := decodificar(t, criado); v["nome"] != "Loja X" || v["ativo"] != true {
		t.Errorf("criado = %v, quero nome aparado e ativo", v)
	}
	lojaX := idDe(t, criado, http.StatusCreated)
	lojaY := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja Y"}`, admin), http.StatusCreated)

	// Nome vazio e acima do teto: 400 em linha, com o limite na mensagem. O
	// teto é em runas: 120 "é" entram, 121 não.
	for _, nome := range []string{"   ", strings.Repeat("é", vendedorNomeMaxDeTeste+1)} {
		resp := vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"`+nome+`"}`, admin)
		confereErro(t, resp, http.StatusBadRequest, "CAMPO_INVALIDO", "nome")
		if msg := fmt.Sprint(decodificar(t, resp)["mensagem"]); !strings.Contains(msg, fmt.Sprint(vendedorNomeMaxDeTeste)) {
			t.Errorf("mensagem = %q, quero o limite %d", msg, vendedorNomeMaxDeTeste)
		}
	}
	noTeto := vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"`+strings.Repeat("é", vendedorNomeMaxDeTeste)+`"}`, admin)
	if resp := vendedorCom(t, rotas, http.MethodDelete, "/"+idDe(t, noTeto, http.StatusCreated), "", admin); resp.Code != http.StatusNoContent {
		t.Fatalf("remover sem Produto = %d (%s), quero 204", resp.Code, resp.Body.String())
	}

	// Duplicado, no criar e no editar: 409 com o campo.
	confereErro(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja X"}`, admin),
		http.StatusConflict, "VENDEDOR_JA_CADASTRADO", "nome")
	confereErro(t, vendedorCom(t, rotas, http.MethodPut, "/"+lojaY, `{"nome":"Loja X","ativo":true}`, admin),
		http.StatusConflict, "VENDEDOR_JA_CADASTRADO", "nome")

	// Inexistente e malformado: 404 no PUT e no DELETE.
	for _, id := range []string{"/00000000-0000-7000-8000-000000000000", "/nao-e-uuid"} {
		confereErro(t, vendedorCom(t, rotas, http.MethodPut, id, `{"nome":"Qualquer","ativo":true}`, admin),
			http.StatusNotFound, "NAO_ENCONTRADO", "")
		confereErro(t, vendedorCom(t, rotas, http.MethodDelete, id, "", admin),
			http.StatusNotFound, "NAO_ENCONTRADO", "")
	}

	// Remover sem Produto: 204, e de novo é 404.
	if resp := vendedorCom(t, rotas, http.MethodDelete, "/"+lojaY, "", admin); resp.Code != http.StatusNoContent {
		t.Fatalf("remover = %d (%s), quero 204", resp.Code, resp.Body.String())
	}
	confereErro(t, vendedorCom(t, rotas, http.MethodDelete, "/"+lojaY, "", admin), http.StatusNotFound, "NAO_ENCONTRADO", "")

	// O Produto da Loja X entra pelo banco, e o Pedido dele pela API, antes
	// de desativar.
	produto := textoDe(t, pool, `
		INSERT INTO catalogo.produto (nome, descricao, preco_centavos, imagem_url, vendedor_id, categoria_id)
		VALUES ('Produto da Loja X', 'Descrição.', 1000, '/api/v1/media/x.svg', $1::uuid, $2::uuid)
		RETURNING id::text`, lojaX, categoriaSemeada)[0]
	pedidoID := idDe(t, pedidoPeloCheckout(t, rotas, comprador, produto, 1), http.StatusCreated)
	// De volta ao Carrinho antes de desativar: é a criação do Pedido que o
	// recusa, e não a adição.
	if resp := postarItem(t, rotas, corpoItem(produto, "1"), comprador); resp.Code != http.StatusCreated {
		t.Fatalf("adicionar antes de desativar = %d (%s)", resp.Code, resp.Body.String())
	}
	pedidoAntes := corpoSemCorrelacao(pegarPedido(t, rotas, pedidoID, comprador))

	// Editar e desativar: 200 com a linha nova.
	desativado := vendedorCom(t, rotas, http.MethodPut, "/"+lojaX, `{"nome":"Loja X2","ativo":false}`, admin)
	if desativado.Code != http.StatusOK {
		t.Fatalf("desativar = %d (%s), quero 200", desativado.Code, desativado.Body.String())
	}
	if v := decodificar(t, desativado); v["nome"] != "Loja X2" || v["ativo"] != false || v["id"] != lojaX {
		t.Errorf("desativado = %v", v)
	}

	// Produto de inativo: a Página de Produto é 404, e a compra também, sem
	// Reserva gravada.
	confereErro(t, pegar(t, rotas, "/api/v1/produtos/"+produto), http.StatusNotFound, "NAO_ENCONTRADO", "")
	reservasAntes := len(textoDe(t, pool, `SELECT id::text FROM catalogo.reserva_estoque WHERE produto_id = $1::uuid`, produto))
	confereErro(t, confirmarCarrinho(t, rotas, comprador), http.StatusConflict, "ESTOQUE_INSUFICIENTE", "")
	if n := len(textoDe(t, pool, `SELECT id::text FROM catalogo.reserva_estoque WHERE produto_id = $1::uuid`, produto)); n != reservasAntes {
		t.Errorf("reservas = %d depois da compra recusada, eram %d", n, reservasAntes)
	}

	// O Pedido antigo é lido idêntico.
	if depois := corpoSemCorrelacao(pegarPedido(t, rotas, pedidoID, comprador)); depois != pedidoAntes {
		t.Errorf("o Pedido mudou com a desativação:\n%s\n%s", pedidoAntes, depois)
	}

	// Remover com Produto: 409, e a mensagem oferece desativar.
	comProdutos := vendedorCom(t, rotas, http.MethodDelete, "/"+lojaX, "", admin)
	confereErro(t, comProdutos, http.StatusConflict, "VENDEDOR_COM_PRODUTOS", "")
	if msg := fmt.Sprint(decodificar(t, comProdutos)["mensagem"]); !strings.Contains(msg, "Desative-o no lugar.") {
		t.Errorf("mensagem = %q", msg)
	}

	// A lista traz ativos e inativos.
	if nomes := nomesDe(t, pegarCom(t, rotas, "/api/v1/admin/vendedores", admin)); !slices.Contains(nomes, "Loja X2") {
		t.Errorf("lista = %v, quero o inativo nela", nomes)
	}

	// Reativar devolve a Página de Produto.
	if resp := vendedorCom(t, rotas, http.MethodPut, "/"+lojaX, `{"nome":"Loja X2","ativo":true}`, admin); resp.Code != http.StatusOK {
		t.Fatalf("reativar = %d (%s)", resp.Code, resp.Body.String())
	}
	if resp := pegar(t, rotas, "/api/v1/produtos/"+produto); resp.Code != http.StatusOK {
		t.Errorf("Produto depois de reativar = %d, quero 200", resp.Code)
	}
}

// confereErro confere status, código e — quando pedido — o `dados.campo`.
func confereErro(t *testing.T, resp *httptest.ResponseRecorder, status int, codigo, campo string) {
	t.Helper()
	if resp.Code != status {
		t.Fatalf("status = %d (%s), quero %d", resp.Code, resp.Body.String(), status)
	}
	envelope := decodificar(t, resp)
	if envelope["codigo"] != codigo {
		t.Errorf("codigo = %v, quero %s", envelope["codigo"], codigo)
	}
	if campo == "" {
		return
	}
	if dados, _ := envelope["dados"].(map[string]any); dados["campo"] != campo {
		t.Errorf("dados = %v, quero campo %q", envelope["dados"], campo)
	}
}

func nomesDe(t *testing.T, resp *httptest.ResponseRecorder) []string {
	t.Helper()
	var nomes []string
	for _, v := range decodificarLista(t, resp) {
		nomes = append(nomes, fmt.Sprint(v["nome"]))
	}
	return nomes
}

func vendedorCom(t *testing.T, rotas http.Handler, metodo, sufixo, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	tipo := ""
	if corpo != "" {
		tipo = "application/json"
	}
	return comCorpo(t, rotas, metodo, "/api/v1/admin/vendedores"+sufixo, corpo, tipo, cookie)
}
