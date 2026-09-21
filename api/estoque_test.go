package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
)

// estoqueEReservas é a matriz da 3.4 num subteste só: o ajuste pela rota, e
// Disponivel, Visiveis, Reservar e Liberar chamados direto. Vendedor,
// Categoria e Produtos são criados aqui; as Reservas ATIVA entram por SQL.
func estoqueEReservas(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Rui Matos","email":"rui@exemplo.br","senha":"senha-do-rui-1"}`), http.StatusCreated)

	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja do Estoque"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Estoque Novo"}`, admin), http.StatusCreated)
	corpoDe := func(nome string, ativo bool) string {
		b, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": 4990, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": 7, "ativo": ativo,
		})
		return string(b)
	}
	produto := idDe(t, produtoCom(t, rotas, http.MethodPost, "", corpoDe("Garrafa Térmica", true), admin), http.StatusCreated)
	invisivel := idDe(t, produtoCom(t, rotas, http.MethodPost, "", corpoDe("Garrafa Antiga", true), admin), http.StatusCreated)
	if resp := produtoCom(t, rotas, http.MethodPut, "/"+invisivel, corpoDe("Garrafa Antiga", false), admin); resp.Code != http.StatusOK {
		t.Fatalf("desativar = %d (%s)", resp.Code, resp.Body.String())
	}

	ajustar := func(id, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		return produtoCom(t, rotas, http.MethodPut, "/"+id+"/estoque", corpo, cookie)
	}
	reservar := func(quantidade string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
			VALUES ($1::uuid, uuidv7(), $2::integer, 'ATIVA')`, produto, quantidade); err != nil {
			t.Fatalf("reservar: %v", err)
		}
	}
	total := func() string {
		t.Helper()
		return textoDe(t, pool, `SELECT estoque_total::text FROM catalogo.produto WHERE id = $1::uuid`, produto)[0]
	}
	disponivelNaView := func() string {
		t.Helper()
		return textoDe(t, pool, `SELECT estoque_disponivel::text FROM catalogo.produto_visivel WHERE id = $1::uuid`, produto)[0]
	}
	recusa := func(corpo, frase string, n float64) {
		t.Helper()
		resp := ajustar(produto, corpo, admin)
		confereErro(t, resp, http.StatusConflict, "ESTOQUE_COMPROMETIDO", "estoque_total")
		e := decodificar(t, resp)
		if e["mensagem"] != frase+" O Estoque total não pode ficar abaixo disso." {
			t.Errorf("mensagem = %q", e["mensagem"])
		}
		if d, _ := e["dados"].(map[string]any); d["comprometidas"] != n {
			t.Errorf("dados = %v, quero comprometidas %v", e["dados"], n)
		}
		if got := total(); got != "7" {
			t.Errorf("estoque_total = %s depois da recusa, quero 7", got)
		}
	}

	// Sem Sessão de Administrador, a rota nova é o 404 da guarda.
	for _, cookie := range []*http.Cookie{nil, comprador} {
		confereErro(t, ajustar(produto, `{"estoque_total":5}`, cookie), http.StatusNotFound, "NAO_ENCONTRADO", "")
	}

	// Abaixo das Reservas: nada muda, com a contagem no singular e no plural.
	reservar("1")
	recusa(`{"estoque_total":0}`, "Há 1 unidade comprometida em Pedidos abertos.", 1)
	reservar("3")
	recusa(`{"estoque_total":2}`, "Há 4 unidades comprometidas em Pedidos abertos.", 4)

	// Fora do teto: a mesma mensagem da criação.
	for _, corpo := range []string{`{"estoque_total":-1}`, `{"estoque_total":100001}`, `{}`} {
		resp := ajustar(produto, corpo, admin)
		confereErro(t, resp, http.StatusBadRequest, "CAMPO_INVALIDO", "estoque_total")
		if msg, _ := decodificar(t, resp)["mensagem"].(string); !strings.Contains(msg, "100.000") {
			t.Errorf("%s: mensagem = %q", corpo, msg)
		}
	}
	for _, id := range []string{"00000000-0000-7000-8000-000000000000", "nao-e-uuid"} {
		confereErro(t, ajustar(id, `{"estoque_total":5}`, admin), http.StatusNotFound, "NAO_ENCONTRADO", "")
	}

	// Aceito: a linha administrativa, e a VIEW já mostra o disponível derivado.
	resp := ajustar(produto, `{"estoque_total":5}`, admin)
	if e := decodificar(t, resp); resp.Code != http.StatusOK || e["estoque_total"] != float64(5) || e["id"] != produto {
		t.Fatalf("ajuste = %d %v", resp.Code, e)
	}
	if got := disponivelNaView(); got != "1" {
		t.Errorf("estoque_disponivel = %s, quero 1", got)
	}
	// Vale também para Produto inativo — a trava não passa pela VIEW.
	if resp := ajustar(invisivel, `{"estoque_total":3}`, admin); resp.Code != http.StatusOK {
		t.Errorf("ajuste do inativo = %d (%s), quero 200", resp.Code, resp.Body.String())
	}

	// O novo total vale na hora: sobra uma unidade, a primeira compra leva, a
	// segunda é recusada.
	if resp := pegar(t, rotas, "/api/v1/produtos/"+produto); resp.Code != http.StatusOK {
		t.Fatalf("Página de Produto = %d", resp.Code)
	}
	// O segundo Comprador põe a unidade no Carrinho antes: é a Reserva, sob a
	// trava, quem o recusa, e não o conselho da adição.
	segundo := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Sara Matos","email":"sara@exemplo.br","senha":"senha-da-sara-1"}`), http.StatusCreated)
	if resp := postarItem(t, rotas, corpoItem(produto, "1"), segundo); resp.Code != http.StatusCreated {
		t.Fatalf("adicionar = %d (%s)", resp.Code, resp.Body.String())
	}
	idDe(t, pedidoPeloCheckout(t, rotas, comprador, produto, 1), http.StatusCreated)
	confereErro(t, confirmarCarrinho(t, rotas, segundo), http.StatusConflict, "ESTOQUE_INSUFICIENTE", "")
	if got := disponivelNaView(); got != "0" {
		t.Errorf("estoque_disponivel = %s depois de esgotar, quero 0", got)
	}

	// A VIEW não expõe o total (AD-16).
	if cols := textoDe(t, pool, `SELECT column_name::text FROM information_schema.columns
		WHERE table_schema = 'catalogo' AND table_name = 'produto_visivel' AND column_name = 'estoque_total'`); len(cols) != 0 {
		t.Errorf("a VIEW produto_visivel ainda expõe estoque_total")
	}

	// Disponivel e Visiveis: uma chave por id recebido, com a mesma grafia.
	// Libera as reservas do Produto para que o disponível seja 5.
	if _, err := pool.Exec(ctx, `UPDATE catalogo.reserva_estoque SET estado = 'LIBERADA'
		WHERE produto_id = $1::uuid AND estado = 'ATIVA'`, produto); err != nil {
		t.Fatalf("liberar por SQL: %v", err)
	}
	inexistente, malformado, maiusculo := "00000000-0000-7000-8000-000000000000", "nao-e-uuid", strings.ToUpper(produto)
	ids := []string{produto, maiusculo, invisivel, inexistente, malformado}
	disponivel, err := catalogo.Disponivel(ctx, pool, ids)
	if err != nil {
		t.Fatalf("Disponivel: %v", err)
	}
	querDisponivel := map[string]int{produto: 5, maiusculo: 5, invisivel: 0, inexistente: 0, malformado: 0}
	visiveis, err := catalogo.Visiveis(ctx, pool, ids)
	if err != nil {
		t.Fatalf("Visiveis: %v", err)
	}
	querVisivel := map[string]bool{produto: true, maiusculo: true, invisivel: false, inexistente: false, malformado: false}
	for _, id := range ids {
		if n, ok := disponivel[id]; !ok || n != querDisponivel[id] {
			t.Errorf("Disponivel[%s] = %d, %v; quero %d", id, n, ok, querDisponivel[id])
		}
		if v, ok := visiveis[id]; !ok || v != querVisivel[id] {
			t.Errorf("Visiveis[%s] = %v, %v; quero %v", id, v, ok, querVisivel[id])
		}
	}
	if len(disponivel) != len(ids) || len(visiveis) != len(ids) {
		t.Errorf("chaves a mais: %v %v", disponivel, visiveis)
	}

	// Reservar e Liberar, numa transação revertida.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback(ctx)
	pedidoID := textoDe(t, tx, `SELECT uuidv7()::text`)[0]
	doPedido := func() []string {
		t.Helper()
		return textoDe(t, tx, `SELECT estado FROM catalogo.reserva_estoque WHERE pedido_id = $1::uuid`, pedidoID)
	}
	if err := catalogo.Reservar(ctx, tx, pedidoID, nil); err != nil {
		t.Errorf("Reservar([]) = %v, quero nil", err)
	}
	for _, id := range []string{invisivel, inexistente, malformado} {
		err := catalogo.Reservar(ctx, tx, pedidoID, []catalogo.ItemReserva{{ProdutoID: produto, Quantidade: 1}, {ProdutoID: id, Quantidade: 1}})
		var falta catalogo.EstoqueInsuficiente
		if !errors.As(err, &falta) || falta.ProdutoID != id || falta.Disponivel != 0 {
			t.Errorf("Reservar com %s = %v (%+v), quero EstoqueInsuficiente com disponível 0", id, err, falta)
		}
	}
	if r := doPedido(); len(r) != 0 {
		t.Errorf("Reservas gravadas depois das recusas: %v", r)
	}
	if err := catalogo.Reservar(ctx, tx, pedidoID, []catalogo.ItemReserva{{ProdutoID: produto, Quantidade: 2}}); err != nil {
		t.Fatalf("Reservar: %v", err)
	}
	if n, _ := catalogo.Disponivel(ctx, tx, []string{produto}); n[produto] != 3 {
		t.Errorf("Disponivel depois de reservar 2 = %d, quero 3", n[produto])
	}
	for i := range 2 {
		if err := catalogo.Liberar(ctx, tx, pedidoID); err != nil {
			t.Errorf("Liberar #%d = %v, quero nil", i+1, err)
		}
	}
	if r := doPedido(); len(r) != 1 || r[0] != "LIBERADA" {
		t.Errorf("Reserva depois de Liberar = %v, quero [LIBERADA]", r)
	}
	if got := textoDe(t, tx, `SELECT estoque_total::text FROM catalogo.produto WHERE id = $1::uuid`, produto)[0]; got != "5" {
		t.Errorf("estoque_total = %s depois de Liberar, quero 5 intacto", got)
	}
}
