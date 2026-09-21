package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/carrinho"
	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
)

// As funções abaixo são chamadas por TestSessaoEProduto: Postgres e Redis
// sobem uma vez só para api/, e não um par por arquivo de teste.
//
// Este Produto é semeado e tem estoque próprio — o de 409 precisa de um
// Produto que possa ser esgotado sem estragar os outros subtestes.
const produtoParaEsgotar = "a0ae8ff1-da13-5591-9291-4a4f1ce15383"

// freteSudeste é o Frete da faixa de SP (AD-17): o Endereço que
// enderecoDeSP garante cai nela, e todo Pedido abaixo do limiar de isenção de
// teste (R$ 250,00) paga isto.
const freteSudeste = 1500

// pedidoNasceAguardandoPagamento cobre de uma vez as linhas da matriz que só
// um Pedido de verdade produz: o 201, a numeração por ano, o histórico da
// transição, o Item congelado, o Frete e o Endereço congelados, a Reserva
// ATIVA e o Carrinho vazio. Devolve o uuid do primeiro, que o subteste do
// compare-and-swap reaproveita.
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
		resp := pedidoPeloCheckout(t, rotas, cookie, produtoSemeado, 1)
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
		// O total é o preço do Produto mais o Frete de SP, em centavos
		// inteiros: não há divisão no caminho monetário (AD-9), e o JSON não
		// pode trazer 264.9.
		if total, ok := corpo["total_centavos"].(float64); !ok || total != 24990+freteSudeste {
			t.Errorf("total_centavos = %v, quero %d inteiro", corpo["total_centavos"], 24990+freteSudeste)
		}
		if strings.Contains(resp.Body.String(), "264.9") {
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
	// estado antes; o Comprador como ator, e o instante gravado.
	transicoes := textoDe(t, pool, `
		SELECT status_anterior || '|' || status_novo || '|' || ator || '|' ||
		       (ocorrido_em IS NOT NULL)::text
		FROM pedido.transicao_status WHERE pedido_id = $1::uuid`, primeiro)
	if len(transicoes) != 1 {
		t.Fatalf("%d linhas de transição, quero 1: %v", len(transicoes), transicoes)
	}
	if transicoes[0] != "|AGUARDANDO_PAGAMENTO|COMPRADOR|true" {
		t.Errorf("transição = %q; quero anterior vazio, novo AGUARDANDO_PAGAMENTO, ator COMPRADOR e instante", transicoes[0])
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

	// Subtotal, Frete e Endereço congelados em colunas do próprio Pedido.
	congelado := textoDe(t, pool, `
		SELECT subtotal_centavos || '|' || frete_centavos || '|' || total_centavos || '|' ||
		       endereco_cep || '|' || endereco_uf
		FROM pedido.pedido WHERE id = $1::uuid`, primeiro)
	quer = fmt.Sprintf("24990|%d|%d|01310100|SP", freteSudeste, 24990+freteSudeste)
	if len(congelado) != 1 || congelado[0] != quer {
		t.Errorf("Pedido = %v, quero [%q]", congelado, quer)
	}

	// A Reserva nasce ATIVA, na mesma transação, com a quantidade comprada.
	reservas := textoDe(t, pool, `
		SELECT estado || '|' || quantidade || '|' || produto_id
		FROM catalogo.reserva_estoque WHERE pedido_id = $1::uuid`, primeiro)
	quer = "ATIVA|1|" + produtoSemeado
	if len(reservas) != 1 || reservas[0] != quer {
		t.Errorf("reserva = %v, quero [%q]", reservas, quer)
	}

	// O Carrinho sai vazio da criação: os Itens viraram Itens de Pedido.
	if n := len(decodificar(t, pegarCarrinho(t, rotas, cookie))["itens"].([]any)); n != 0 {
		t.Errorf("%d Itens no Carrinho depois do Pedido, quero 0", n)
	}
	return primeiro
}

// pedidoSemSessaoDa401: sem cookie, ou com cookie fora do Redis, nenhuma linha
// entra em pedido.pedido — a Sessão é conferida antes de tudo, até da chave.
func pedidoSemSessaoDa401(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	antes := contarPedidos(t, pool)
	for _, cookie := range []*http.Cookie{
		nil,
		{Name: "azamon_sessao", Value: strings.Repeat("0", 64)},
	} {
		resp := postarPedido(t, rotas, chaveNova(t), corpoPedido("00000000-0000-7000-8000-000000000000", 100), cookie)
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

// pedidoComEntradaRuim: sem `Idempotency-Key`, ou com ela fora da forma de
// uuid, é 400 antes de abrir transação; Endereço inexistente, malformado e de
// outro Comprador são o mesmo 404; corpo malformado ou grande demais é 400.
func pedidoComEntradaRuim(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie) {
	outro := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Ulisses Alheio","email":"ulisses.alheio@exemplo.br","senha":"senha-do-ulisses-1"}`), http.StatusCreated)
	alheio := enderecoDeSP(t, rotas, outro)
	// O Carrinho tem Item, para que o 404 seja do Endereço, e não de outra
	// recusa que viesse antes dele.
	esvaziarCarrinho(t, rotas, cookie)
	if resp := postarItem(t, rotas, corpoItem(produtoSemeado, "1"), cookie); resp.Code != http.StatusCreated {
		t.Fatalf("adicionar = %d (%s)", resp.Code, resp.Body.String())
	}
	antes := contarPedidos(t, pool)
	for _, caso := range []struct {
		nome   string
		chave  string
		corpo  string
		status int
		codigo string
	}{
		{"sem chave", "", corpoPedido(alheio, 100), http.StatusBadRequest, "ENTRADA_INVALIDA"},
		{"chave fora da forma", "nao-e-uuid", corpoPedido(alheio, 100), http.StatusBadRequest, "ENTRADA_INVALIDA"},
		{"chave com sobra", chaveNova(t) + "x", corpoPedido(alheio, 100), http.StatusBadRequest, "ENTRADA_INVALIDA"},
		{"chave com quebra de linha", chaveNova(t)[:18] + "\r\n" + chaveNova(t)[18:], corpoPedido(alheio, 100), http.StatusBadRequest, "ENTRADA_INVALIDA"},
		{"Endereço de outro Comprador", chaveNova(t), corpoPedido(alheio, 100), http.StatusNotFound, "NAO_ENCONTRADO"},
		{"Endereço inexistente", chaveNova(t), corpoPedido("00000000-0000-7000-8000-000000000000", 100), http.StatusNotFound, "NAO_ENCONTRADO"},
		{"Endereço malformado", chaveNova(t), corpoPedido("abc", 100), http.StatusNotFound, "NAO_ENCONTRADO"},
		{"JSON malformado", chaveNova(t), `{isso não é json`, http.StatusBadRequest, "ENTRADA_INVALIDA"},
		{"corpo grande demais", chaveNova(t), `{"endereco_id":"` + strings.Repeat("a", corpoMaximo) + `"}`, http.StatusBadRequest, "ENTRADA_INVALIDA"},
	} {
		resp := postarPedido(t, rotas, caso.chave, caso.corpo, cookie)
		if resp.Code != caso.status {
			t.Fatalf("%s: status = %d (%s), quero %d", caso.nome, resp.Code, resp.Body.String(), caso.status)
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != caso.codigo {
			t.Errorf("%s: codigo = %v, quero %q", caso.nome, codigo, caso.codigo)
		}
	}
	if depois := contarPedidos(t, pool); depois != antes {
		t.Errorf("%d Pedidos depois das recusas, quero os %d de antes", depois, antes)
	}
	if n := len(decodificar(t, pegarCarrinho(t, rotas, cookie))["itens"].([]any)); n != 1 {
		t.Errorf("%d Itens no Carrinho depois das recusas, quero o 1 de antes", n)
	}
	esvaziarCarrinho(t, rotas, cookie)
}

// pedidoSemEstoqueDa409 é a primeira condição de aceite da 5.6: dois Produtos
// no Carrinho, um sem Estoque — nenhum Pedido nem Reserva nasce, o disponível
// do outro não muda, e o Carrinho fica intacto. O esgotamento é por Reserva
// ATIVA direta, depois de o Item já estar no Carrinho: é a Reserva, sob a
// trava, quem recusa, e não o conselho da adição.
func pedidoSemEstoqueDa409(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie) {
	ctx := context.Background()
	esvaziarCarrinho(t, rotas, cookie)
	for _, produto := range []string{produtoSemeado, produtoParaEsgotar} {
		if resp := postarItem(t, rotas, corpoItem(produto, "1"), cookie); resp.Code != http.StatusCreated {
			t.Fatalf("adicionar %s = %d (%s)", produto, resp.Code, resp.Body.String())
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
		SELECT id, uuidv7(), estoque_total, 'ATIVA' FROM catalogo.produto WHERE id = $1::uuid`,
		produtoParaEsgotar); err != nil {
		t.Fatalf("esgotar o Estoque: %v", err)
	}
	antes := contarPedidos(t, pool)
	reservasDoOutro := func() []string {
		return textoDe(t, pool, `SELECT id::text FROM catalogo.reserva_estoque WHERE produto_id = $1::uuid ORDER BY id`, produtoSemeado)
	}
	disponivelDoOutro := func() string {
		return textoDe(t, pool, `SELECT estoque_disponivel::text FROM catalogo.produto_visivel WHERE id = $1::uuid`, produtoSemeado)[0]
	}
	reservasAntes, disponivelAntes := reservasDoOutro(), disponivelDoOutro()

	resp := confirmarCarrinho(t, rotas, cookie)
	if resp.Code != http.StatusConflict {
		t.Fatalf("status = %d (%s), quero 409", resp.Code, resp.Body.String())
	}
	envelope := decodificar(t, resp)
	if envelope["codigo"] != "ESTOQUE_INSUFICIENTE" {
		t.Errorf("codigo = %v", envelope["codigo"])
	}
	dados, ok := envelope["dados"].(map[string]any)
	if !ok || dados["disponivel"] != float64(0) || dados["produto_id"] != produtoParaEsgotar {
		t.Errorf("dados = %v, quero o Produto que faltou e o disponível em zero", envelope["dados"])
	}
	if depois := contarPedidos(t, pool); depois != antes {
		t.Errorf("%d Pedidos depois do 409, quero os %d de antes; a transação tinha de ser revertida", depois, antes)
	}
	// Tudo ou nada: o Produto que tinha Estoque não ganhou Reserva, e o
	// disponível dele é o mesmo de antes.
	if depois := reservasDoOutro(); !slices.Equal(depois, reservasAntes) {
		t.Errorf("Reservas do outro Produto = %v, eram %v", depois, reservasAntes)
	}
	if depois := disponivelDoOutro(); depois != disponivelAntes {
		t.Errorf("disponível do outro Produto = %s, era %s", depois, disponivelAntes)
	}
	// A Tentativa de Pagamento nasce na mesma transação do Pedido, e por isso
	// é revertida junto: uma Tentativa órfã aqui seria emitida pela varredura
	// para um Pedido que nunca existiu.
	if n := contarTentativas(t, pool); n != antes {
		t.Errorf("%d Tentativas depois do 409, quero uma por Pedido (%d)", n, antes)
	}
	// O Carrinho fica intacto: o esvaziar é o último passo, e foi desfeito.
	if n := len(decodificar(t, pegarCarrinho(t, rotas, cookie))["itens"].([]any)); n != 2 {
		t.Errorf("%d Itens no Carrinho depois do 409, quero os 2 de antes", n)
	}

	// A recusa não queima um número: o contador é incrementado dentro da
	// transação revertida, que é a justificativa escrita de ele ser tabela e
	// não SEQUENCE. O Pedido seguinte é o consecutivo, sem buraco.
	seguinte := pedidoPeloCheckout(t, rotas, cookie, produtoSemeado, 1)
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

	if err := pedido.Transicionar(ctx, tx, pedidoID, pedido.StatusAguardandoPagamento, pedido.StatusPago, pedido.AtorProvedor, ""); err != nil {
		t.Fatalf("primeira transição = %v; quero nil", err)
	}
	// O NFR-9 vale para todo avanço, e não só para o nascimento: a transição
	// que aconteceu deixa a sua linha, na mesma transação.
	historico := textoDe(t, tx, `
		SELECT status_anterior || '|' || status_novo || '|' || ator
		FROM pedido.transicao_status WHERE pedido_id = $1::uuid ORDER BY ocorrido_em`, pedidoID)
	if len(historico) != 2 {
		t.Fatalf("%d linhas de histórico depois do avanço, quero 2: %v", len(historico), historico)
	}
	if quer := "AGUARDANDO_PAGAMENTO|PAGO|PROVEDOR"; historico[1] != quer {
		t.Errorf("segunda linha = %q, quero %q", historico[1], quer)
	}

	err = pedido.Transicionar(ctx, tx, pedidoID, pedido.StatusAguardandoPagamento, pedido.StatusPago, pedido.AtorProvedor, "")
	if !errors.Is(err, pedido.ErrEstadoJaAvancado) {
		t.Errorf("segunda transição = %v; quero ErrEstadoJaAvancado", err)
	}
}

// leituraDoPedido cobre as três linhas da matriz da tela de acompanhamento: o
// dono lê, quem não é dono recebe o mesmo 404 de quem pede um Pedido que não
// existe, e sem Sessão é 401.
func leituraDoPedido(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie, pedidoID string) {
	if pedidoID == "" {
		t.Skip("sem Pedido: a criação falhou antes")
	}

	resp := pegarPedido(t, rotas, pedidoID, cookie)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d (%s), quero 200", resp.Code, resp.Body.String())
	}
	if v := resp.Header().Get("Cache-Control"); v != "no-store" {
		t.Errorf("Cache-Control = %q, quero no-store", v)
	}
	corpo := decodificar(t, resp)
	if corpo["status"] != "AGUARDANDO_PAGAMENTO" {
		t.Errorf("status = %v", corpo["status"])
	}
	if total, ok := corpo["total_centavos"].(float64); !ok || total != 24990+freteSudeste {
		t.Errorf("total_centavos = %v, quero %d inteiro", corpo["total_centavos"], 24990+freteSudeste)
	}
	if !strings.HasPrefix(fmt.Sprint(corpo["numero"]), "AZ-") {
		t.Errorf("numero = %v", corpo["numero"])
	}
	// Instante absoluto, RFC 3339, vindo do servidor: é o que a tela exibe, e
	// o navegador nunca conta duração. Analisar não basta — trocar o max por
	// min na consulta, ou devolver o ano 0001, passaria. O que se compara é o
	// instante contra a última transição que o Pedido de fato tem.
	if quer := ultimaTransicao(t, pool, pedidoID); !instanteDe(t, corpo).Equal(quer) {
		t.Errorf("atualizado_em = %v, quero %v (o max de ocorrido_em)", instanteDe(t, corpo), quer)
	}

	// Pedido de outro Comprador entra pelo banco: a semente tem uma conta só,
	// e o que se prova aqui é o dono no WHERE, não a segunda Sessão.
	deOutro := textoDe(t, pool, `
		INSERT INTO pedido.pedido (numero, comprador_id, status, subtotal_centavos, frete_centavos, total_centavos)
		VALUES ('AZ-0000-000001', uuidv7(), 'AGUARDANDO_PAGAMENTO', 100, 0, 100)
		RETURNING id::text`)
	if len(deOutro) != 1 {
		t.Fatalf("criar o Pedido de outro Comprador: %v", deOutro)
	}
	for _, id := range []string{
		deOutro[0],
		"00000000-0000-7000-8000-000000000000",
		"nao-e-uuid",
	} {
		resp := pegarPedido(t, rotas, id, cookie)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d (%s), quero 404", id, resp.Code, resp.Body.String())
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != "NAO_ENCONTRADO" {
			t.Errorf("%s: codigo = %v", id, codigo)
		}
	}

	semCookie := pegarPedido(t, rotas, pedidoID, nil)
	if semCookie.Code != http.StatusUnauthorized {
		t.Fatalf("sem cookie: status = %d, quero 401", semCookie.Code)
	}
	if codigo := decodificar(t, semCookie)["codigo"]; codigo != "SESSAO_INVALIDA" {
		t.Errorf("sem cookie: codigo = %v", codigo)
	}
}

// instanteDe extrai o `atualizado_em` do corpo e exige que ele seja RFC 3339.
func instanteDe(t *testing.T, corpo map[string]any) time.Time {
	t.Helper()
	texto, ok := corpo["atualizado_em"].(string)
	if !ok {
		t.Fatalf("atualizado_em = %v; quero o instante da última transição", corpo["atualizado_em"])
	}
	instante, err := time.Parse(time.RFC3339Nano, texto)
	if err != nil {
		t.Fatalf("atualizado_em = %q não é RFC 3339: %v", texto, err)
	}
	return instante
}

// ultimaTransicao é a resposta certa vinda do banco.
func ultimaTransicao(t *testing.T, pool *pgxpool.Pool, pedidoID string) time.Time {
	t.Helper()
	var quando time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT max(ocorrido_em) FROM pedido.transicao_status WHERE pedido_id = $1::uuid`,
		pedidoID).Scan(&quando); err != nil {
		t.Fatalf("ler a última transição: %v", err)
	}
	return quando.UTC()
}

// meusPedidosListaPorDono é a 2.6, no molde de enderecosDoComprador
// (endereco_test.go): conta própria, para o vazio da matriz ser observável
// antes de qualquer Pedido existir — cookieValido, reaproveitado pelos
// subtestes acima, já chega aqui com Pedidos seus. O AD-11 entra na própria
// consulta (ListarPedidosDoComprador), e a ordem é `id DESC`: a chave é
// uuidv7(), ordenada no tempo por construção, então o Pedido criado por
// último sai primeiro sem depender de relógio nenhum.
func meusPedidosListaPorDono(t *testing.T, rotas http.Handler) {
	cookie := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Nara Bastos","email":"nara-pedidos@exemplo.br","senha":"senha-da-nara-1"}`), http.StatusCreated)

	// Listar sem nenhum: 200 com `[]`, e não `null` — o mesmo contrato de
	// Meus Endereços (2.5): o estado vazio é da tela.
	vazia := pegarPedidos(t, rotas, cookie)
	if vazia.Code != http.StatusOK {
		t.Fatalf("listar sem nenhum = %d (%s), quero 200", vazia.Code, vazia.Body.String())
	}
	if v := vazia.Header().Get("Cache-Control"); v != "no-store" {
		t.Errorf("Cache-Control da listagem = %q, quero no-store", v)
	}
	if corpo := strings.TrimSpace(vazia.Body.String()); corpo != "[]" {
		t.Errorf("lista vazia = %s, quero []", corpo)
	}

	primeiro := idDe(t, pedidoPeloCheckout(t, rotas, cookie, produtoSemeado, 1), http.StatusCreated)
	segundo := idDe(t, pedidoPeloCheckout(t, rotas, cookie, produtoSemeado, 1), http.StatusCreated)

	// Outro Comprador cria um terceiro Pedido: a lista de Nara não pode
	// trazê-lo — o dono entra na própria consulta, e não numa checagem
	// depois (AD-11).
	outro := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Otelo Farias","email":"otelo-pedidos@exemplo.br","senha":"senha-do-otelo-1"}`), http.StatusCreated)
	if resp := pedidoPeloCheckout(t, rotas, outro, produtoSemeado, 1); resp.Code != http.StatusCreated {
		t.Fatalf("Pedido do outro Comprador = %d (%s), quero 201", resp.Code, resp.Body.String())
	}

	lista := pegarPedidos(t, rotas, cookie)
	if lista.Code != http.StatusOK {
		t.Fatalf("listar com dois = %d (%s), quero 200", lista.Code, lista.Body.String())
	}
	var ids []string
	for _, p := range decodificarLista(t, lista) {
		ids = append(ids, fmt.Sprint(p["id"]))
	}
	if !slices.Equal(ids, []string{segundo, primeiro}) {
		t.Errorf("ids = %v, quero [%s, %s] — mais recente primeiro, só os de Nara", ids, segundo, primeiro)
	}

	// Sem Sessão: 401 SESSAO_INVALIDA, a mesma guarda das outras rotas
	// autenticadas do Comprador.
	semCookie := pegarPedidos(t, rotas, nil)
	if semCookie.Code != http.StatusUnauthorized {
		t.Fatalf("sem cookie: status = %d, quero 401", semCookie.Code)
	}
	if codigo := decodificar(t, semCookie)["codigo"]; codigo != "SESSAO_INVALIDA" {
		t.Errorf("sem cookie: codigo = %v, quero SESSAO_INVALIDA", codigo)
	}
}

func pegarPedidos(t *testing.T, rotas http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return pegarCom(t, rotas, "/api/v1/pedidos", cookie)
}

func pegarPedido(t *testing.T, rotas http.Handler, id string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pedidos/"+id, nil)
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

func contarTentativas(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM pagamento.tentativa_pagamento`).Scan(&n); err != nil {
		t.Fatalf("contar Tentativas: %v", err)
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

// simulacaoDeEntrega cobre as linhas da matriz da 1.8 que só um Pedido de
// verdade produz: a Reserva nasce pelo caminho real da compra, e é ela que a
// consolidação encerra. O relógio entra como parâmetro e não como tique, e é
// isso que permite provar o "cedo demais" sem esperar por ele.
//
// Roda por último: é o único subteste que baixa o `estoque_total` de
// produtoSemeado, e os que compram esse mesmo Produto contam com o total
// intacto para o disponível bater.
func simulacaoDeEntrega(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie) {
	ctx := context.Background()

	resp := pedidoPeloCheckout(t, rotas, cookie, produtoSemeado, 1)
	if resp.Code != http.StatusCreated {
		t.Fatalf("criar o Pedido: status = %d (%s)", resp.Code, resp.Body.String())
	}
	pedidoID := fmt.Sprint(decodificar(t, resp)["id"])

	estoque := func() int32 {
		var n int32
		if err := pool.QueryRow(ctx,
			`SELECT estoque_total FROM catalogo.produto WHERE id = $1::uuid`, produtoSemeado).Scan(&n); err != nil {
			t.Fatalf("ler o Estoque: %v", err)
		}
		return n
	}
	statusDo := func() string {
		v := textoDe(t, pool, `SELECT status FROM pedido.pedido WHERE id = $1::uuid`, pedidoID)
		if len(v) != 1 {
			t.Fatalf("ler o Status: %v", v)
		}
		return v[0]
	}
	// Envelhecer o histórico é o que substitui a espera: o avanço deriva de
	// "está neste estado desde quando", então recuar o instante é a mesma coisa
	// que deixar o intervalo vencer — e prova que a decisão vem da tabela, e
	// não de um relógio em memória que um reinício zeraria.
	envelhecer := func() {
		t.Helper()
		envelhecerHistorico(t, pool, pedidoID, time.Hour)
	}
	antes := estoque()

	reserva := func() string {
		v := textoDe(t, pool,
			`SELECT estado FROM catalogo.reserva_estoque WHERE pedido_id = $1::uuid`, pedidoID)
		if len(v) != 1 {
			t.Fatalf("ler a Reserva: %v", v)
		}
		return v[0]
	}

	// O Pedido entra em PAGO pelo caminho da 1.7 — o que se prova neste
	// subteste é o passo seguinte. A transição vai por `pedido.Transicionar`, e
	// não por um UPDATE à mão: o `AD-6` diz que esse UPDATE não existe, e
	// escrevê-lo aqui deixaria o histórico do teste divergir do de produção.
	paraPago, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir a transação: %v", err)
	}
	defer paraPago.Rollback(ctx)
	if err := pedido.Transicionar(ctx, paraPago, pedidoID, pedido.StatusAguardandoPagamento, pedido.StatusPago, pedido.AtorProvedor, ""); err != nil {
		t.Fatalf("pôr o Pedido em PAGO: %v", err)
	}
	if err := paraPago.Commit(ctx); err != nil {
		t.Fatalf("commit do avanço para PAGO: %v", err)
	}
	if corpo := decodificar(t, pegarPedido(t, rotas, pedidoID, cookie)); corpo["terminal"] != false {
		t.Errorf("terminal = %v em PAGO; quero false", corpo["terminal"])
	}

	// Cedo demais: a última transição é de agora e o intervalo, de uma hora.
	if err := pedido.SimularEntrega(ctx, pool, time.Hour); err != nil {
		t.Fatalf("simular antes do intervalo: %v", err)
	}
	if s := statusDo(); s != "PAGO" {
		t.Errorf("status = %s com o intervalo por vencer; quero PAGO", s)
	}

	// Uma etapa por intervalo vencido, e nenhuma pulada. Cada chamada é um
	// tique: o Status novo reinicia a contagem, então só a próxima o move.
	//
	// O intervalo é largo de propósito. `SimularEntrega` avança TODO Pedido
	// elegível do banco, e `envelhecer` só recua o histórico deste: um intervalo
	// curto deixaria um Pedido de outro subteste entrar na varredura assim que a
	// suíte passasse dele, baixando o Estoque do mesmo Produto e quebrando a
	// conta. Meia hora é mais que qualquer execução plausível, e `envelhecer`
	// recua uma hora — todo passo daqui segue vencendo.
	const intervaloLargo = 30 * time.Minute

	// conferirEstoque prende a baixa a UMA transição: enquanto o Pedido não
	// chega a ENVIADO a Reserva está ATIVA e o total intacto; de lá em diante,
	// CONSOLIDADA e um a menos. Sem esta conferência passo a passo, mover
	// `Consolidar` para qualquer outro avanço deixaria o teste verde — ela é
	// idempotente, e o total no fim seria o mesmo.
	conferirEstoque := func(momento string, consolidado bool) {
		t.Helper()
		queroReserva, queroEstoque := "ATIVA", antes
		if consolidado {
			queroReserva, queroEstoque = "CONSOLIDADA", antes-1
		}
		if r, e := reserva(), estoque(); r != queroReserva || e != queroEstoque {
			t.Fatalf("%s: Reserva = %s e estoque_total = %d; quero %s e %d",
				momento, r, e, queroReserva, queroEstoque)
		}
	}

	for _, quero := range []string{"SEPARANDO", "ENVIADO", "ENTREGUE"} {
		conferirEstoque("antes de "+quero, quero == "ENTREGUE")
		envelhecer()
		if err := pedido.SimularEntrega(ctx, pool, intervaloLargo); err != nil {
			t.Fatalf("simular até %s: %v", quero, err)
		}
		if s := statusDo(); s != quero {
			t.Fatalf("status = %s, quero %s", s, quero)
		}
		conferirEstoque("em "+quero, quero != "SEPARANDO")
	}

	if corpo := decodificar(t, pegarPedido(t, rotas, pedidoID, cookie)); corpo["terminal"] != true {
		t.Errorf("terminal = %v em ENTREGUE; quero true — é ele que manda a tela parar de consultar", corpo["terminal"])
	}

	// Consolidar de novo é no-op: zero linhas ATIVAS, nil, e o Estoque não
	// baixa duas vezes. Tratado como erro, um tique repetido derrubaria a
	// transação inteira do Pedido já consolidado.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir a transação: %v", err)
	}
	defer tx.Rollback(ctx)
	if err := catalogo.Consolidar(ctx, tx, pedidoID); err != nil {
		t.Errorf("consolidar sobre Reserva já consolidada = %v; quero nil", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit da segunda consolidação: %v", err)
	}
	if depois := estoque(); depois != antes-1 {
		t.Errorf("estoque_total = %d depois da segunda consolidação, quero %d", depois, antes-1)
	}
}

// postarPedido é o Confirmar Pedido da Revisão: o corpo e o cabeçalho
// `Idempotency-Key`, que sai da requisição quando a chave é vazia.
func postarPedido(t *testing.T, rotas http.Handler, chave, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pedidos", strings.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	if chave != "" {
		req.Header.Set("Idempotency-Key", chave)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
}

func corpoPedido(enderecoID string, total int64) string {
	b, _ := json.Marshal(map[string]any{"endereco_id": enderecoID, "total_centavos": total})
	return string(b)
}

// chaveNova é o UUIDv4 que a Revisão gera ao abrir: um por tentativa de
// checkout.
func chaveNova(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("sortear a chave: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// enderecoDeSP garante um Endereço de SP (CEP 01310-100, faixa Sudeste) na
// conta, e devolve o id dele: o primeiro da lista com esse CEP, ou um novo.
func enderecoDeSP(t *testing.T, rotas http.Handler, cookie *http.Cookie) string {
	t.Helper()
	for _, e := range decodificarLista(t, pegarEnderecos(t, rotas, cookie)) {
		if e["cep"] == "01310100" {
			return fmt.Sprint(e["id"])
		}
	}
	return idDe(t, postarEndereco(t, rotas, corpoEnderecoValido, cookie), http.StatusCreated)
}

// totalDaRevisao é o total que a Revisão exibiria para este Endereço: o da
// cotação do Go, e nunca uma soma feita no teste.
func totalDaRevisao(t *testing.T, rotas http.Handler, cookie *http.Cookie, enderecoID string) int64 {
	t.Helper()
	resp := pegarFrete(t, rotas, enderecoID, cookie)
	if resp.Code != http.StatusOK {
		t.Fatalf("cotar o Frete = %d (%s)", resp.Code, resp.Body.String())
	}
	total, _ := decodificar(t, resp)["total_centavos"].(float64)
	return int64(total)
}

// confirmarCarrinho é o passeio do checkout sobre o Carrinho como está:
// Endereço de SP garantido, a cotação da Revisão, e o Confirmar Pedido com uma
// chave nova.
func confirmarCarrinho(t *testing.T, rotas http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	endereco := enderecoDeSP(t, rotas, cookie)
	return postarPedido(t, rotas, chaveNova(t), corpoPedido(endereco, totalDaRevisao(t, rotas, cookie, endereco)), cookie)
}

// pedidoPeloCheckout é o caminho de um Pedido de um Produto só: limpa o
// Carrinho, adiciona, e confirma pelo checkout. É o molde de todo subteste que
// precisa de um Pedido e não é sobre a criação dele.
func pedidoPeloCheckout(t *testing.T, rotas http.Handler, cookie *http.Cookie, produto string, quantidade int) *httptest.ResponseRecorder {
	t.Helper()
	if resp := esvaziarCarrinho(t, rotas, cookie); resp.Code != http.StatusNoContent {
		t.Fatalf("esvaziar o Carrinho = %d (%s)", resp.Code, resp.Body.String())
	}
	if resp := postarItem(t, rotas, corpoItem(produto, strconv.Itoa(quantidade)), cookie); resp.Code != http.StatusCreated {
		t.Fatalf("adicionar ao Carrinho = %d (%s)", resp.Code, resp.Body.String())
	}
	return confirmarCarrinho(t, rotas, cookie)
}

// criacaoPeloCheckout é a matriz de servidor da 5.6 contra o banco de verdade,
// com conta, Produtos e Endereços próprios: a criação a partir do Carrinho, o
// reenvio, a chave reaproveitada, o duplo clique concorrente, o preço que muda
// depois da Revisão, o Carrinho vazio e o Endereço alheio — mais as três
// condições de aceite que não cabem noutro subteste.
func criacaoPeloCheckout(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	loja := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja da Criação"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Categoria da Criação"}`, admin), http.StatusCreated)
	corpoDe := func(nome string, preco int) string {
		b, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": preco, "imagem_url": "",
			"vendedor_id": loja, "categoria_id": categoria, "estoque_total": 20, "ativo": true,
		})
		return string(b)
	}
	novoProduto := func(nome string, preco int) string {
		t.Helper()
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", corpoDe(nome, preco), admin), http.StatusCreated)
	}
	publicar := func(id, nome string, preco int) {
		t.Helper()
		if resp := produtoCom(t, rotas, http.MethodPut, "/"+id, corpoDe(nome, preco), admin); resp.Code != http.StatusOK {
			t.Fatalf("republicar %s = %d (%s)", nome, resp.Code, resp.Body.String())
		}
	}
	adicionar := func(cookie *http.Cookie, produto, quantidade string) string {
		t.Helper()
		return idDe(t, postarItem(t, rotas, corpoItem(produto, quantidade), cookie), http.StatusCreated)
	}
	itensNoCarrinho := func(cookie *http.Cookie) int {
		t.Helper()
		itens, _ := decodificar(t, pegarCarrinho(t, rotas, cookie))["itens"].([]any)
		return len(itens)
	}
	contar := func(tabela string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM `+tabela).Scan(&n); err != nil {
			t.Fatalf("contar %s: %v", tabela, err)
		}
		return n
	}
	// contagens são Pedidos, Itens de Pedido, transições, Reservas e
	// Tentativas, nessa ordem; fotografia é o que "nada gravado" quer dizer.
	contagens := func() []int {
		t.Helper()
		return []int{contar("pedido.pedido"), contar("pedido.item_pedido"), contar("pedido.transicao_status"),
			contar("catalogo.reserva_estoque"), contar("pagamento.tentativa_pagamento")}
	}
	fotografia := func() string {
		t.Helper()
		return fmt.Sprint(contagens())
	}

	chaleira := novoProduto("Chaleira da Criação", 5000)
	bule := novoProduto("Bule da Criação", 3000)
	lia := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Lia Criação","email":"lia.criacao@exemplo.br","senha":"senha-da-lia-1"}`), http.StatusCreated)
	sp := enderecoDeSP(t, rotas, lia)

	// Criação: dois Produtos, Endereço de SP, o total da Revisão.
	adicionar(lia, chaleira, "2")
	adicionar(lia, bule, "1")
	total := totalDaRevisao(t, rotas, lia, sp)
	if total != 13000+freteSudeste {
		t.Fatalf("total da Revisão = %d, quero %d", total, 13000+freteSudeste)
	}
	chave := chaveNova(t)
	criado := postarPedido(t, rotas, chave, corpoPedido(sp, total), lia)
	if criado.Code != http.StatusCreated {
		t.Fatalf("criar = %d (%s), quero 201", criado.Code, criado.Body.String())
	}
	original := decodificar(t, criado)
	pedidoID := fmt.Sprint(original["id"])
	if original["status"] != "AGUARDANDO_PAGAMENTO" || original["total_centavos"] != float64(total) {
		t.Errorf("Pedido = %v; quero AGUARDANDO_PAGAMENTO com o total da Revisão", original)
	}
	congeladoNoPedido := func() string {
		t.Helper()
		return strings.Join(textoDe(t, pool, `
			SELECT subtotal_centavos || '|' || frete_centavos || '|' || total_centavos || '|' ||
			       endereco_destinatario || '|' || endereco_cep || '|' || endereco_logradouro || '|' ||
			       endereco_numero || '|' || endereco_complemento || '|' || endereco_bairro || '|' ||
			       endereco_cidade || '|' || endereco_uf
			FROM pedido.pedido WHERE id = $1::uuid`, pedidoID), "")
	}
	itensDoPedido := func() string {
		t.Helper()
		return strings.Join(textoDe(t, pool, `
			SELECT nome || '|' || vendedor_nome || '|' || preco_praticado_centavos || '|' || quantidade
			FROM pedido.item_pedido WHERE pedido_id = $1::uuid ORDER BY nome`, pedidoID), ",")
	}
	querCongelado := fmt.Sprintf("13000|%d|%d|Joana Ribeiro|01310100|Avenida Paulista|1578|Apto 12|Bela Vista|São Paulo|SP",
		freteSudeste, total)
	if c := congeladoNoPedido(); c != querCongelado {
		t.Errorf("Pedido congelado = %q, quero %q", c, querCongelado)
	}
	querItens := "Bule da Criação|Loja da Criação|3000|1,Chaleira da Criação|Loja da Criação|5000|2"
	if i := itensDoPedido(); i != querItens {
		t.Errorf("Itens de Pedido = %q, quero %q", i, querItens)
	}
	if r := strings.Join(textoDe(t, pool, `
		SELECT estado || '|' || quantidade FROM catalogo.reserva_estoque
		WHERE pedido_id = $1::uuid ORDER BY quantidade`, pedidoID), ","); r != "ATIVA|1,ATIVA|2" {
		t.Errorf("Reservas = %q, quero as duas ATIVAS", r)
	}
	if n := itensNoCarrinho(lia); n != 0 {
		t.Errorf("%d Itens no Carrinho depois do Pedido, quero 0", n)
	}
	if tentativas := textoDe(t, pool, `
		SELECT total_centavos::text FROM pagamento.tentativa_pagamento WHERE pedido_id = $1::uuid`, pedidoID); !slices.Equal(tentativas, []string{fmt.Sprint(total)}) {
		t.Errorf("Tentativas = %v, quero uma, com o total %d", tentativas, total)
	}

	// Reenvio: a mesma chave e o mesmo corpo devolvem o mesmo Pedido em 200,
	// e nada novo nasce — nem o número do contador.
	antes := fotografia()
	reenvio := postarPedido(t, rotas, chave, corpoPedido(sp, total), lia)
	if reenvio.Code != http.StatusOK || decodificar(t, reenvio)["id"] != pedidoID {
		t.Fatalf("reenvio = %d (%s), quero 200 com o Pedido %s", reenvio.Code, reenvio.Body.String(), pedidoID)
	}
	// O uuid em maiúscula é a mesma chave e o mesmo Endereço.
	if maiuscula := postarPedido(t, rotas, strings.ToUpper(chave), corpoPedido(strings.ToUpper(sp), total), lia); maiuscula.Code != http.StatusOK {
		t.Errorf("reenvio em maiúscula = %d (%s), quero 200", maiuscula.Code, maiuscula.Body.String())
	}
	if depois := fotografia(); depois != antes {
		t.Errorf("o reenvio gravou: %s, era %s", depois, antes)
	}

	// Chave reaproveitada: outro corpo sob a chave que já criou um Pedido é
	// 409, com o Pedido original em `dados`.
	reaproveitada := postarPedido(t, rotas, chave, corpoPedido(sp, total+100), lia)
	confereErro(t, reaproveitada, http.StatusConflict, "CHAVE_REUTILIZADA", "")
	if dados, _ := decodificar(t, reaproveitada)["dados"].(map[string]any); dados["id"] != pedidoID || dados["numero"] != original["numero"] {
		t.Errorf("dados = %v, quero o Pedido original", dados)
	}
	if depois := fotografia(); depois != antes {
		t.Errorf("a chave reaproveitada gravou: %s, era %s", depois, antes)
	}
	// A chave é do Comprador: a mesma chave, noutra conta, é chave livre.
	outro := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Téo Criação","email":"teo.criacao@exemplo.br","senha":"senha-do-teo-1"}`), http.StatusCreated)
	adicionar(outro, bule, "1")
	enderecoDoOutro := enderecoDeSP(t, rotas, outro)
	if resp := postarPedido(t, rotas, chave, corpoPedido(enderecoDoOutro, totalDaRevisao(t, rotas, outro, enderecoDoOutro)), outro); resp.Code != http.StatusCreated {
		t.Errorf("a mesma chave noutra conta = %d (%s), quero 201", resp.Code, resp.Body.String())
	}

	// Segunda condição de aceite: o preço do Produto muda e o Endereço some,
	// e o Pedido continua o mesmo — Itens, Frete, Endereço e total.
	publicar(chaleira, "Chaleira da Criação", 5500)
	if resp := deletarEndereco(t, rotas, sp, lia); resp.Code != http.StatusNoContent {
		t.Fatalf("remover o Endereço = %d (%s)", resp.Code, resp.Body.String())
	}
	if c := congeladoNoPedido(); c != querCongelado {
		t.Errorf("Pedido depois de mudar o preço e remover o Endereço = %q, quero %q", c, querCongelado)
	}
	if i := itensDoPedido(); i != querItens {
		t.Errorf("Itens depois de mudar o preço = %q, quero %q", i, querItens)
	}
	sp = enderecoDeSP(t, rotas, lia)

	// O gatilho da 5.6: Item de Pedido depois do histórico é recusado pelo
	// banco, e o CHECK do AD-9 recusa total que não é a soma das parcelas.
	var pgErr *pgconn.PgError
	_, err := pool.Exec(ctx, `
		INSERT INTO pedido.item_pedido (pedido_id, produto_id, nome, vendedor_nome, preco_praticado_centavos, quantidade)
		VALUES ($1::uuid, $2::uuid, 'Tardio', 'Loja', 100, 1)`, pedidoID, bule)
	if !errors.As(err, &pgErr) || pgErr.Code != "23001" {
		t.Errorf("Item tardio = %v, quero restrict_violation (23001)", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO pedido.pedido (numero, comprador_id, status, subtotal_centavos, frete_centavos, total_centavos)
		VALUES ('AZ-SOMA-000001', uuidv7(), 'AGUARDANDO_PAGAMENTO', 1000, 1500, 2600)`)
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Errorf("total fora da soma = %v, quero check_violation (23514)", err)
	}

	// Carrinho vazio: 409 CARRINHO_VAZIO, e nada gravado.
	antes = fotografia()
	confereErro(t, postarPedido(t, rotas, chaveNova(t), corpoPedido(sp, freteSudeste), lia), http.StatusConflict, "CARRINHO_VAZIO", "")
	if depois := fotografia(); depois != antes {
		t.Errorf("o Carrinho vazio gravou: %s, era %s", depois, antes)
	}

	// Endereço alheio: 404, nada gravado e o Carrinho intacto.
	adicionar(lia, bule, "1")
	confereErro(t, postarPedido(t, rotas, chaveNova(t), corpoPedido(enderecoDoOutro, 3000+freteSudeste), lia), http.StatusNotFound, "NAO_ENCONTRADO", "")
	if depois := fotografia(); depois != antes {
		t.Errorf("o Endereço alheio gravou: %s, era %s", depois, antes)
	}
	if n := itensNoCarrinho(lia); n != 1 {
		t.Errorf("%d Itens no Carrinho depois do 404, quero 1", n)
	}

	// Preço mudou depois da Revisão: 409 TOTAL_DIVERGENTE, nada gravado, e o
	// Carrinho intacto.
	daRevisao := totalDaRevisao(t, rotas, lia, sp)
	publicar(bule, "Bule da Criação", 3200)
	chaveDaRevisao := chaveNova(t)
	confereErro(t, postarPedido(t, rotas, chaveDaRevisao, corpoPedido(sp, daRevisao), lia), http.StatusConflict, "TOTAL_DIVERGENTE", "")
	if depois := fotografia(); depois != antes {
		t.Errorf("o total divergente gravou: %s, era %s", depois, antes)
	}
	if n := itensNoCarrinho(lia); n != 1 {
		t.Errorf("%d Itens no Carrinho depois do TOTAL_DIVERGENTE, quero 1", n)
	}
	// Terceira condição de aceite: a recusa não consome a chave. A mesma
	// chave, com o total novo que a Revisão relida mostra, cria o Pedido.
	novoTotal := totalDaRevisao(t, rotas, lia, sp)
	if novoTotal != 3200+freteSudeste {
		t.Fatalf("total relido = %d, quero %d", novoTotal, 3200+freteSudeste)
	}
	if resp := postarPedido(t, rotas, chaveDaRevisao, corpoPedido(sp, novoTotal), lia); resp.Code != http.StatusCreated {
		t.Errorf("a mesma chave com o total novo = %d (%s), quero 201", resp.Code, resp.Body.String())
	}

	// Duplo clique concorrente: duas requisições iguais em paralelo, um
	// Pedido, e as duas respostas com o mesmo id.
	adicionar(lia, chaleira, "1")
	adicionar(lia, bule, "1")
	corpo := corpoPedido(sp, totalDaRevisao(t, rotas, lia, sp))
	dupla := chaveNova(t)
	antesDoDuplo := contagens()
	respostas := make([]*httptest.ResponseRecorder, 2)
	var grupo sync.WaitGroup
	for i := range respostas {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			respostas[i] = postarPedido(t, rotas, dupla, corpo, lia)
		}()
	}
	grupo.Wait()
	var ids, codigos []string
	for _, r := range respostas {
		codigos = append(codigos, strconv.Itoa(r.Code))
		ids = append(ids, fmt.Sprint(decodificar(t, r)["id"]))
	}
	slices.Sort(codigos)
	if !slices.Equal(codigos, []string{"200", "201"}) || ids[0] != ids[1] {
		t.Errorf("duplo clique: status %v e ids %v; quero um 201 e um 200 com o mesmo id", codigos, ids)
	}
	// O delta inteiro, e não só o número de Pedidos: um Pedido, os seus dois
	// Itens, uma transição, as suas duas Reservas e uma Tentativa. O gêmeo
	// desfez tudo, inclusive o número do contador.
	depoisDoDuplo := contagens()
	quero := []int{antesDoDuplo[0] + 1, antesDoDuplo[1] + 2, antesDoDuplo[2] + 1, antesDoDuplo[3] + 2, antesDoDuplo[4] + 1}
	if !slices.Equal(depoisDoDuplo, quero) {
		t.Errorf("duplo clique: Pedidos, Itens, transições, Reservas e Tentativas = %v, quero %v", depoisDoDuplo, quero)
	}
	if n := itensNoCarrinho(lia); n != 0 {
		t.Errorf("%d Itens no Carrinho depois do duplo clique, quero 0", n)
	}

	// Carrinho só de Produtos desativados: 409 ESTOQUE_INSUFICIENTE nomeando o
	// Produto, e nada gravado — nem o CHECK do subtotal chega a ser tentado.
	jarra := novoProduto("Jarra da Criação", 2000)
	adicionar(lia, jarra, "1")
	desativado, _ := json.Marshal(map[string]any{
		"nome": "Jarra da Criação", "descricao": "", "preco_centavos": 2000, "imagem_url": "",
		"vendedor_id": loja, "categoria_id": categoria, "estoque_total": 20, "ativo": false,
	})
	if resp := produtoCom(t, rotas, http.MethodPut, "/"+jarra, string(desativado), admin); resp.Code != http.StatusOK {
		t.Fatalf("desativar a Jarra = %d (%s)", resp.Code, resp.Body.String())
	}
	antes = fotografia()
	invisiveis := confirmarCarrinho(t, rotas, lia)
	confereErro(t, invisiveis, http.StatusConflict, "ESTOQUE_INSUFICIENTE", "")
	if dados, _ := decodificar(t, invisiveis)["dados"].(map[string]any); dados["produto_id"] != jarra || dados["disponivel"] != float64(0) {
		t.Errorf("dados = %v, quero a Jarra com disponível 0", dados)
	}
	if depois := fotografia(); depois != antes {
		t.Errorf("o Carrinho só de invisíveis gravou: %s, era %s", depois, antes)
	}
	if resp := esvaziarCarrinho(t, rotas, lia); resp.Code != http.StatusNoContent {
		t.Fatalf("esvaziar = %d", resp.Code)
	}

	// Item adicionado noutra aba entre a leitura e o fim: Esvaziar apaga só os
	// Itens lidos, e o novo fica no Carrinho. Sem gancho no meio da criação,
	// a prova é da própria porta, na transação que a criação usaria.
	lido := adicionar(lia, chaleira, "1")
	daOutraAba := adicionar(lia, bule, "1")
	compradora := textoDe(t, pool, `SELECT id::text FROM identidade.comprador WHERE email = 'lia.criacao@exemplo.br'`)[0]
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir a transação: %v", err)
	}
	defer tx.Rollback(ctx)
	if err := carrinho.Esvaziar(ctx, tx, compradora, []string{lido}); err != nil {
		t.Fatalf("Esvaziar só o lido = %v", err)
	}
	if ficou := textoDe(t, tx, `SELECT id::text FROM carrinho.item_carrinho WHERE id = ANY($1::uuid[])`, []string{lido, daOutraAba}); !slices.Equal(ficou, []string{daOutraAba}) {
		t.Errorf("Itens depois de Esvaziar = %v, quero só o da outra aba", ficou)
	}
	// O Item lido que sumiu debaixo da transação é ErrCarrinhoMudou: linha a
	// menos desfaz tudo.
	if err := carrinho.Esvaziar(ctx, tx, compradora, []string{lido}); !errors.Is(err, carrinho.ErrCarrinhoMudou) {
		t.Errorf("Esvaziar de Item que sumiu = %v, quero ErrCarrinhoMudou", err)
	}
	// A escrita parcial da entrada no checkout (5.5) é o mesmo sentinela: o
	// preço a confirmar é de um Item que sumiu na mesma transação.
	err = carrinho.ConfirmarPrecoVisto(ctx, tx, compradora, []carrinho.PrecoVisto{
		{ItemID: daOutraAba, PrecoCentavos: 3200},
		{ItemID: lido, PrecoCentavos: 5500},
	})
	if !errors.Is(err, carrinho.ErrCarrinhoMudou) {
		t.Errorf("ConfirmarPrecoVisto com Item que sumiu = %v, quero ErrCarrinhoMudou", err)
	}
}
