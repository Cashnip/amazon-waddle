package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/pedido"
)

// As funções abaixo são chamadas por TestSessaoEProduto: Postgres e Redis
// sobem uma vez só para api/, e não um par por arquivo de teste.
//
// Este Produto é semeado e tem estoque próprio — o de 409 precisa de um
// Produto que possa ser esgotado sem estragar os outros subtestes.
const produtoParaEsgotar = "a0ae8ff1-da13-5591-9291-4a4f1ce15383"

// pedidoNasceAguardandoPagamento cobre de uma vez as linhas da matriz que só
// um Pedido de verdade produz: o 201, a numeração por ano, o histórico da
// transição, o Item congelado e a Reserva ATIVA. Devolve o uuid do primeiro,
// que o subteste do compare-and-swap reaproveita.
func pedidoNasceAguardandoPagamento(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie) string {
	if cookie == nil {
		t.Fatal("sem cookie: o login falhou antes")
	}
	ano := time.Now().UTC().Year()

	var primeiro string
	for i, quer := range []string{
		fmt.Sprintf("AZ-%d-%06d", ano, 1),
		fmt.Sprintf("AZ-%d-%06d", ano, 2),
	} {
		resp := postarPedido(t, rotas, `{"produto_id":"`+produtoSemeado+`"}`, cookie)
		if resp.Code != http.StatusCreated {
			t.Fatalf("Pedido %d: status = %d (%s), quero 201", i+1, resp.Code, resp.Body.String())
		}
		// A resposta é por Comprador: sem a diretiva, um intermediário pode
		// guardá-la por heurística e servir o Pedido de um a outro.
		if v := resp.Header().Get("Cache-Control"); v != "no-store" {
			t.Errorf("Cache-Control = %q, quero no-store", v)
		}
		// escreverJSON põe o cabeçalho antes do WriteHeader, e a ordem é
		// carga: invertida, o 201 sairia em text/plain.
		if v := resp.Header().Get("Content-Type"); v != "application/json; charset=utf-8" {
			t.Errorf("Content-Type = %q", v)
		}
		corpo := decodificar(t, resp)
		// Número legível e separado do uuid (AD-6), e sem colisão entre dois
		// Pedidos seguidos — é o contador por ano, não o identificador.
		if corpo["numero"] != quer {
			t.Errorf("numero = %v, quero %q", corpo["numero"], quer)
		}
		if corpo["status"] != "AGUARDANDO_PAGAMENTO" {
			t.Errorf("status = %v", corpo["status"])
		}
		// O total é o preço do Produto, em centavos inteiros: não há divisão
		// no caminho monetário (AD-3), e o JSON não pode trazer 249.90.
		if total, ok := corpo["total_centavos"].(float64); !ok || total != 24990 {
			t.Errorf("total_centavos = %v, quero 24990 inteiro", corpo["total_centavos"])
		}
		if strings.Contains(resp.Body.String(), "249.9") {
			t.Error("o total saiu em ponto flutuante")
		}
		if i == 0 {
			primeiro, _ = corpo["id"].(string)
		}
	}
	if primeiro == "" {
		t.Fatal("o 201 não devolveu o identificador do Pedido")
	}

	// A primeira linha do histórico (NFR-9): anterior vazio, porque não havia
	// estado antes; autor e instante gravados.
	transicoes := textoDe(t, pool, `
		SELECT status_anterior || '|' || status_novo || '|' || autor || '|' ||
		       (ocorrido_em IS NOT NULL)::text
		FROM pedido.transicao_status WHERE pedido_id = $1::uuid`, primeiro)
	if len(transicoes) != 1 {
		t.Fatalf("%d linhas de transição, quero 1: %v", len(transicoes), transicoes)
	}
	if !strings.HasPrefix(transicoes[0], "|AGUARDANDO_PAGAMENTO|") || !strings.HasSuffix(transicoes[0], "|true") {
		t.Errorf("transição = %q; quero anterior vazio, novo AGUARDANDO_PAGAMENTO, autor e instante", transicoes[0])
	}

	// O Item congela: nome, preço praticado e Vendedor são cópia, não
	// referência — mudar o Produto amanhã não muda o Pedido de hoje.
	itens := textoDe(t, pool, `
		SELECT nome || '|' || vendedor_nome || '|' || preco_praticado_centavos || '|' || quantidade
		FROM pedido.item_pedido WHERE pedido_id = $1::uuid`, primeiro)
	quer := "Fone de Ouvido Bluetooth Aurora|Atlântico Importados|24990|1"
	if len(itens) != 1 || itens[0] != quer {
		t.Errorf("item = %v, quero [%q]", itens, quer)
	}

	// A Reserva nasce ATIVA, na mesma transação, com a quantidade comprada.
	reservas := textoDe(t, pool, `
		SELECT estado || '|' || quantidade || '|' || produto_id
		FROM catalogo.reserva_estoque WHERE pedido_id = $1::uuid`, primeiro)
	quer = "ATIVA|1|" + produtoSemeado
	if len(reservas) != 1 || reservas[0] != quer {
		t.Errorf("reserva = %v, quero [%q]", reservas, quer)
	}
	return primeiro
}

// pedidoSemSessaoDa401: sem cookie, ou com cookie fora do Redis, nenhuma linha
// entra em pedido.pedido — a rota é a segunda (e última) autenticada.
func pedidoSemSessaoDa401(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	antes := contarPedidos(t, pool)
	for _, cookie := range []*http.Cookie{
		nil,
		{Name: "azamon_sessao", Value: strings.Repeat("0", 64)},
	} {
		resp := postarPedido(t, rotas, `{"produto_id":"`+produtoSemeado+`"}`, cookie)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("cookie %v: status = %d, quero 401", cookie, resp.Code)
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != "SESSAO_INVALIDA" {
			t.Errorf("cookie %v: codigo = %v", cookie, codigo)
		}
	}
	if depois := contarPedidos(t, pool); depois != antes {
		t.Errorf("%d Pedidos depois do 401, quero os %d de antes", depois, antes)
	}
}

// pedidoComEntradaRuim: Produto inexistente e identificador malformado são o
// mesmo 404 — nunca 500; corpo malformado ou grande demais é 400.
func pedidoComEntradaRuim(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie) {
	antes := contarPedidos(t, pool)
	for _, caso := range []struct {
		corpo  string
		status int
		codigo string
	}{
		{`{"produto_id":"00000000-0000-5000-8000-000000000000"}`, http.StatusNotFound, "NAO_ENCONTRADO"},
		{`{"produto_id":"abc"}`, http.StatusNotFound, "NAO_ENCONTRADO"},
		{`{isso não é json`, http.StatusBadRequest, "ENTRADA_INVALIDA"},
		{`{"produto_id":"` + strings.Repeat("a", corpoMaximo) + `"}`, http.StatusBadRequest, "ENTRADA_INVALIDA"},
	} {
		resp := postarPedido(t, rotas, caso.corpo, cookie)
		if resp.Code != caso.status {
			t.Fatalf("%.40s: status = %d (%s), quero %d", caso.corpo, resp.Code, resp.Body.String(), caso.status)
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != caso.codigo {
			t.Errorf("%.40s: codigo = %v, quero %q", caso.corpo, codigo, caso.codigo)
		}
	}
	// Nenhum dos quatro casos deixa Pedido para trás: os dois de 400 voltam
	// antes do Begin, e os dois de 404 voltam na busca do Produto, que é a
	// primeira coisa que a transação faz.
	if depois := contarPedidos(t, pool); depois != antes {
		t.Errorf("%d Pedidos depois das recusas, quero os %d de antes", depois, antes)
	}
}

// pedidoSemEstoqueDa409 esgota um Produto por Reserva ATIVA direta e confere
// que a compra é recusada com o disponível em `dados`.
func pedidoSemEstoqueDa409(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie) {
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
		SELECT id, uuidv7(), estoque_total, 'ATIVA' FROM catalogo.produto WHERE id = $1::uuid`,
		produtoParaEsgotar); err != nil {
		t.Fatalf("esgotar o Estoque: %v", err)
	}
	antes := contarPedidos(t, pool)

	resp := postarPedido(t, rotas, `{"produto_id":"`+produtoParaEsgotar+`"}`, cookie)
	if resp.Code != http.StatusConflict {
		t.Fatalf("status = %d (%s), quero 409", resp.Code, resp.Body.String())
	}
	envelope := decodificar(t, resp)
	if envelope["codigo"] != "ESTOQUE_INSUFICIENTE" {
		t.Errorf("codigo = %v", envelope["codigo"])
	}
	dados, ok := envelope["dados"].(map[string]any)
	if !ok || dados["disponivel"] != float64(0) {
		t.Errorf("dados = %v, quero o disponível em zero", envelope["dados"])
	}
	if depois := contarPedidos(t, pool); depois != antes {
		t.Errorf("%d Pedidos depois do 409, quero os %d de antes; a transação tinha de ser revertida", depois, antes)
	}

	// A recusa não queima um número: o contador é incrementado dentro da
	// transação revertida, que é a justificativa escrita de ele ser tabela e
	// não SEQUENCE. O Pedido seguinte é o consecutivo, sem buraco.
	seguinte := postarPedido(t, rotas, `{"produto_id":"`+produtoSemeado+`"}`, cookie)
	if seguinte.Code != http.StatusCreated {
		t.Fatalf("Pedido seguinte: status = %d (%s), quero 201", seguinte.Code, seguinte.Body.String())
	}
	quer := fmt.Sprintf("AZ-%d-%06d", time.Now().UTC().Year(), antes+1)
	if numero := decodificar(t, seguinte)["numero"]; numero != quer {
		t.Errorf("numero = %v, quero %q; a recusa queimou um número", numero, quer)
	}
}

// transicaoRepetidaNaoAvanca prova o compare-and-swap contra Postgres de
// verdade: a segunda chamada com o mesmo estado de origem afeta zero linhas.
// Tudo dentro de uma transação revertida no fim — o Pedido continua onde está.
func transicaoRepetidaNaoAvanca(t *testing.T, pool *pgxpool.Pool, pedidoID string) {
	if pedidoID == "" {
		t.Skip("sem Pedido: a criação falhou antes")
	}
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir a transação: %v", err)
	}
	defer tx.Rollback(ctx)

	if err := pedido.Transicionar(ctx, tx, pedidoID, "AGUARDANDO_PAGAMENTO", "PAGO", "teste"); err != nil {
		t.Fatalf("primeira transição = %v; quero nil", err)
	}
	// O NFR-9 vale para todo avanço, e não só para o nascimento: a transição
	// que aconteceu deixa a sua linha, na mesma transação.
	historico := textoDe(t, tx, `
		SELECT status_anterior || '|' || status_novo || '|' || autor
		FROM pedido.transicao_status WHERE pedido_id = $1::uuid ORDER BY ocorrido_em`, pedidoID)
	if len(historico) != 2 {
		t.Fatalf("%d linhas de histórico depois do avanço, quero 2: %v", len(historico), historico)
	}
	if quer := "AGUARDANDO_PAGAMENTO|PAGO|teste"; historico[1] != quer {
		t.Errorf("segunda linha = %q, quero %q", historico[1], quer)
	}

	err = pedido.Transicionar(ctx, tx, pedidoID, "AGUARDANDO_PAGAMENTO", "PAGO", "teste")
	if !errors.Is(err, pedido.ErrEstadoJaAvancado) {
		t.Errorf("segunda transição = %v; quero ErrEstadoJaAvancado", err)
	}
}

func postarPedido(t *testing.T, rotas http.Handler, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pedidos", strings.NewReader(corpo))
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
}

func contarPedidos(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM pedido.pedido`).Scan(&n); err != nil {
		t.Fatalf("contar Pedidos: %v", err)
	}
	return n
}

// consultavel é o que *pgxpool.Pool e pgx.Tx têm em comum: ler de dentro da
// transação aberta é a única forma de observar o que ela ainda não gravou.
type consultavel interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func textoDe(t *testing.T, bd consultavel, sql string, args ...any) []string {
	t.Helper()
	linhas, err := bd.Query(context.Background(), sql, args...)
	if err != nil {
		t.Fatalf("consultar: %v", err)
	}
	defer linhas.Close()
	var valores []string
	for linhas.Next() {
		var v string
		if err := linhas.Scan(&v); err != nil {
			t.Fatalf("ler resultado: %v", err)
		}
		valores = append(valores, v)
	}
	if err := linhas.Err(); err != nil {
		t.Fatalf("percorrer resultado: %v", err)
	}
	return valores
}
