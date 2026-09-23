package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
)

// postarCancelamento é o "Cancelar Pedido" confirmado no Dialog: POST sem
// corpo sobre o próprio Pedido. Tem a forma de `bate` de negacaoPorDono, que o
// usa para provar o AD-11 por igualdade.
func postarCancelamento(t *testing.T, rotas http.Handler, pedidoID string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodPost, "/api/v1/pedidos/"+pedidoID+"/cancelamento", "", "", cookie)
}

// cancelamentoPeloComprador é a matriz de servidor da 6.3 contra o banco de
// verdade (FR-31): o cancelamento nos quatro Status da janela, o Estoque
// disponível voltando (ou não mudando, quando já não havia Reserva), o segundo
// cancelamento como sucesso sem efeito, a recusa fora da janela sem gravar
// nada, as duas corridas que a leitura travada resolve, a aprovação que chega
// depois do cancelamento (FR-26) e o Pedido alheio.
//
// Tudo é próprio: contas, Vendedor, Categoria e Produtos. O Produto de `,00`
// fecha o total em `,00` com o Frete (reais inteiros), a faixa que o Provedor
// Simulado aprova; o de `,90`, a que ele recusa na primeira Tentativa.
//
// A ordem dos subtestes importa só onde um anda com o Pedido do anterior, e
// isso está dito onde acontece.
func cancelamentoPeloComprador(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Caio Cancela","email":"caio.cancela@exemplo.br","senha":"senha-do-caio-1"}`), http.StatusCreated)
	outro := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Otávia Outra","email":"otavia.outra@exemplo.br","senha":"senha-da-otavia-1"}`), http.StatusCreated)

	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja do Cancelamento"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Cancelamento"}`, admin), http.StatusCreated)
	novoProduto := func(nome string, preco, estoque int) string {
		t.Helper()
		corpo, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": preco, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": estoque, "ativo": true,
		})
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(corpo), admin), http.StatusCreated)
	}
	cadeira := novoProduto("Cadeira do Arrependimento", 5000, 10)
	recusada := novoProduto("Mesa da Recusa", 4990, 5)

	// emitirPara é o tique do Provedor Simulado com o transporte de verdade,
	// entregando só as Tentativas destes Pedidos — as dos outros subtestes
	// ficam onde estão.
	emitirPara := func(pedidos ...string) {
		t.Helper()
		enviar := func(_ context.Context, c pagamento.Confirmacao) error {
			for _, p := range pedidos {
				for n := int32(1); n <= tentativasMaxDeTeste; n++ {
					if c.IDExterno == pagamento.IDExterno(p, n) {
						if r := postarWebhook(t, rotas, corpoDe(t, c)); r.Code != http.StatusOK {
							t.Errorf("webhook = %d (%s), quero 200", r.Code, r.Body.String())
						}
					}
				}
			}
			return nil
		}
		if err := pagamento.EmitirConfirmacoesDevidas(ctx, pool, simuladoDeTeste, atrasoDeTeste, janelaDeEmissaoDeTeste, enviar); err != nil {
			t.Fatalf("emitir: %v", err)
		}
	}
	varrer := func() {
		t.Helper()
		if err := pedido.Varrer(ctx, pool); err != nil {
			t.Fatalf("varrer: %v", err)
		}
	}
	// avancar é o Administrador (ou a simulação) movendo o Pedido: a rota é da
	// 6.4, então o caminho é `Transicionar` — com o histórico e o efeito de
	// produção, e não um UPDATE à mão.
	avancar := func(pedidoID string, de, para pedido.Status) {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação: %v", err)
		}
		defer tx.Rollback(ctx)
		if err := pedido.Transicionar(ctx, tx, pedidoID, de, para, pedido.AtorAdministrador, ""); err != nil {
			t.Fatalf("avançar de %s a %s: %v", de, para, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit do avanço: %v", err)
		}
	}
	// pedidoEm cria um Pedido da Cadeira e o leva até o Status pedido pelo
	// caminho de produção: a aprovação do Provedor até PAGO, e o Administrador
	// daí em diante.
	pedidoEm := func(cookie *http.Cookie, status pedido.Status) string {
		t.Helper()
		id := idDe(t, pedidoPeloCheckout(t, rotas, cookie, cadeira, 1), http.StatusCreated)
		if status == pedido.StatusAguardandoPagamento {
			return id
		}
		emitirPara(id)
		varrer()
		if s := statusDe(t, pool, id); s != string(pedido.StatusPago) {
			t.Fatalf("status = %s; o Pedido de `,00` tinha de ir a PAGO", s)
		}
		for _, passo := range [][2]pedido.Status{
			{pedido.StatusPago, pedido.StatusSeparando},
			{pedido.StatusSeparando, pedido.StatusEnviado},
			{pedido.StatusEnviado, pedido.StatusEntregue},
		} {
			if pedido.Status(statusDe(t, pool, id)) == status {
				break
			}
			avancar(id, passo[0], passo[1])
		}
		if s := statusDe(t, pool, id); s != string(status) {
			t.Fatalf("status = %s, quero %s", s, status)
		}
		return id
	}
	disponivelDe := func(produto string) float64 {
		t.Helper()
		n, ok := decodificar(t, pegar(t, rotas, "/api/v1/produtos/"+produto))["estoque_disponivel"].(float64)
		if !ok {
			t.Fatalf("estoque_disponivel ausente na Página de Produto %s", produto)
		}
		return n
	}
	ultimaLinha := func(pedidoID string) string {
		t.Helper()
		linhas := textoDe(t, pool, `
			SELECT status_anterior || '|' || status_novo || '|' || ator || '|' || coalesce(motivo, '')
			FROM pedido.transicao_status WHERE pedido_id = $1::uuid
			ORDER BY ocorrido_em DESC, id DESC LIMIT 1`, pedidoID)
		if len(linhas) != 1 {
			t.Fatalf("Pedido %s sem histórico", pedidoID)
		}
		return linhas[0]
	}
	cancelamentos := func(pedidoID string) int {
		t.Helper()
		n := 0
		for _, l := range transicoesDe(t, pool, pedidoID) {
			if strings.Contains(l, "|CANCELADO|") {
				n++
			}
		}
		return n
	}
	// fotografia é tudo que o cancelamento poderia gravar: Status, histórico,
	// Reservas (com Produto, quantidade e estado), Estoque total do Produto e
	// inbox. A recusa e o cancelamento sem efeito têm de deixá-la idêntica.
	fotografia := func(pedidoID string) []string {
		t.Helper()
		f := []string{"status=" + statusDe(t, pool, pedidoID)}
		f = append(f, transicoesDe(t, pool, pedidoID)...)
		f = append(f, textoDe(t, pool, `
			SELECT r.produto_id::text || '|' || r.quantidade || '|' || r.estado || '|' || p.estoque_total
			FROM catalogo.reserva_estoque r JOIN catalogo.produto p ON p.id = r.produto_id
			WHERE r.pedido_id = $1::uuid ORDER BY r.id`, pedidoID)...)
		return append(f, confirmacoesDe(t, pool, pedidoID)...)
	}
	// podeCancelar lê o `pode_cancelar` do Detalhe, que a tela usa para mostrar
	// o botão. Sem o `ok`, um campo renomeado leria false e as asserções de
	// "fora da janela" passariam sem conferir nada.
	podeCancelar := func(pedidoID string) bool {
		t.Helper()
		resp := pegarPedido(t, rotas, pedidoID, comprador)
		if resp.Code != http.StatusOK {
			t.Fatalf("ler o Pedido = %d (%s)", resp.Code, resp.Body.String())
		}
		pode, ok := decodificar(t, resp)["pode_cancelar"].(bool)
		if !ok {
			t.Fatalf("pode_cancelar ausente no Detalhe do Pedido %s", pedidoID)
		}
		return pode
	}
	// oCorpo confere o 200 do cancelamento: o próprio Pedido, em CANCELADO.
	oCorpo := func(resp *httptest.ResponseRecorder, pedidoID string) {
		t.Helper()
		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d (%s), quero 200", resp.Code, resp.Body.String())
		}
		if corpo := decodificar(t, resp); corpo["id"] != pedidoID || corpo["status"] != "CANCELADO" {
			t.Errorf("corpo = %v; quero o Pedido %s em CANCELADO", corpo, pedidoID)
		}
	}
	// foraDaJanela confere o 409 da recusa, com o Status atual em `dados`.
	foraDaJanela := func(resp *httptest.ResponseRecorder, status string) {
		t.Helper()
		if resp.Code != http.StatusConflict {
			t.Fatalf("status = %d (%s), quero 409", resp.Code, resp.Body.String())
		}
		env := decodificar(t, resp)
		dados, _ := env["dados"].(map[string]any)
		if env["codigo"] != "FORA_DA_JANELA_DE_CANCELAMENTO" || dados["status"] != status {
			t.Errorf("resposta = %v; quero FORA_DA_JANELA_DE_CANCELAMENTO com dados.status = %s", env, status)
		}
	}

	var cancelado string // o Pedido que o primeiro subteste cancela em PAGO

	t.Run("na janela, cancela, grava a linha do Comprador e o Estoque volta", func(t *testing.T) {
		for _, status := range []pedido.Status{pedido.StatusAguardandoPagamento, pedido.StatusPago, pedido.StatusSeparando} {
			id := pedidoEm(comprador, status)
			antes := disponivelDe(cadeira)
			if !podeCancelar(id) {
				t.Errorf("%s: pode_cancelar = false; o Status está na janela", status)
			}

			oCorpo(postarCancelamento(t, rotas, id, comprador), id)

			if s := statusDe(t, pool, id); s != "CANCELADO" {
				t.Errorf("%s: status = %s, quero CANCELADO", status, s)
			}
			if l := ultimaLinha(id); l != string(status)+"|CANCELADO|COMPRADOR|" {
				t.Errorf("%s: última transição = %s", status, l)
			}
			if r := reservasDe(t, pool, id); !slices.Equal(r, []string{"LIBERADA"}) {
				t.Errorf("%s: reservas = %v; o cancelamento libera a Reserva", status, r)
			}
			// A outra aba na Página de Produto: a unidade voltou.
			if d := disponivelDe(cadeira); d != antes+1 {
				t.Errorf("%s: disponível = %v, quero %v", status, d, antes+1)
			}
			// E a tela relida: Status novo, terminal, e a linha no histórico.
			d := decodificar(t, pegarPedido(t, rotas, id, comprador))
			if d["status"] != "CANCELADO" || d["terminal"] != true || d["pode_cancelar"] != false {
				t.Errorf("%s: Detalhe = %v; quero CANCELADO, terminal e sem pode_cancelar", status, d)
			}
			if status == pedido.StatusPago {
				cancelado = id
			}
		}
	})

	t.Run("recusado, sem Reserva ativa, cancela e o disponível não muda", func(t *testing.T) {
		id := idDe(t, pedidoPeloCheckout(t, rotas, comprador, recusada, 1), http.StatusCreated)
		emitirPara(id)
		varrer()
		if s := statusDe(t, pool, id); s != "PAGAMENTO_RECUSADO" {
			t.Fatalf("status = %s, quero PAGAMENTO_RECUSADO", s)
		}
		antes := disponivelDe(recusada)
		if !podeCancelar(id) {
			t.Error("pode_cancelar = false em PAGAMENTO_RECUSADO; o Status está na janela")
		}

		// Liberar sem Reserva ativa é nil (AD-5): tratado como erro, este
		// cancelamento falharia sempre, com rollback.
		oCorpo(postarCancelamento(t, rotas, id, comprador), id)

		if l := ultimaLinha(id); l != "PAGAMENTO_RECUSADO|CANCELADO|COMPRADOR|" {
			t.Errorf("última transição = %s", l)
		}
		if r := reservasDe(t, pool, id); !slices.Equal(r, []string{"LIBERADA"}) {
			t.Errorf("reservas = %v; só a que a recusa já tinha liberado", r)
		}
		if d := disponivelDe(recusada); d != antes {
			t.Errorf("disponível = %v, quero %v: não havia unidade presa", d, antes)
		}
	})

	// Anda com o Pedido que o primeiro subteste cancelou.
	t.Run("já cancelado, o segundo cancelamento é 200 e não grava nada", func(t *testing.T) {
		if cancelado == "" {
			t.Fatal("sem Pedido cancelado: o subteste da janela falhou antes")
		}
		antes := fotografia(cancelado)
		disponivel := disponivelDe(cadeira)

		oCorpo(postarCancelamento(t, rotas, cancelado, comprador), cancelado)

		if depois := fotografia(cancelado); !slices.Equal(depois, antes) {
			t.Errorf("o cancelamento repetido gravou algo:\nantes  %v\ndepois %v", antes, depois)
		}
		if d := disponivelDe(cadeira); d != disponivel {
			t.Errorf("disponível = %v, quero %v", d, disponivel)
		}
	})

	t.Run("ENVIADO e ENTREGUE recusam com o Status atual, sem gravar nada", func(t *testing.T) {
		id := pedidoEm(comprador, pedido.StatusEnviado)
		for _, status := range []pedido.Status{pedido.StatusEnviado, pedido.StatusEntregue} {
			if status == pedido.StatusEntregue {
				avancar(id, pedido.StatusEnviado, pedido.StatusEntregue)
			}
			if podeCancelar(id) {
				t.Errorf("%s: pode_cancelar = true fora da janela", status)
			}
			antes := fotografia(id)
			foraDaJanela(postarCancelamento(t, rotas, id, comprador), string(status))
			if depois := fotografia(id); !slices.Equal(depois, antes) {
				t.Errorf("%s: a recusa gravou algo:\nantes  %v\ndepois %v", status, antes, depois)
			}
		}
	})

	// esperarPresos espera `n` conexões presas em trava. O teste só prova a
	// sobreposição se as requisições estão de fato esperando: disparadas
	// soltas, cada uma termina em milissegundos e a segunda chega depois do
	// commit da primeira — a suíte passaria sem a trava que se quer provar.
	esperarPresos := func(n int) int {
		t.Helper()
		presos := 0
		for prazo := time.Now().Add(10 * time.Second); presos < n && time.Now().Before(prazo); {
			if err := pool.QueryRow(ctx, `
				SELECT count(*) FROM pg_stat_activity
				WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&presos); err != nil {
				t.Fatalf("ler as esperas: %v", err)
			}
			if presos < n {
				time.Sleep(20 * time.Millisecond)
			}
		}
		return presos
	}
	// resultado é o que uma goroutine devolve: `decodificar` chama t.Fatalf,
	// que não pode sair de outra goroutine, então o envelope é lido à mão.
	type resultado struct {
		status int
		corpo  struct {
			Status string `json:"status"`
			Erro   struct {
				Codigo string         `json:"codigo"`
				Dados  map[string]any `json:"dados"`
			} `json:"erro"`
		}
	}
	cancelarEmParalelo := func(pedidoID string, r *resultado) {
		resp := postarCancelamento(t, rotas, pedidoID, comprador)
		r.status = resp.Code
		_ = json.Unmarshal(resp.Body.Bytes(), &r.corpo)
	}

	t.Run("o duplo clique preso na trava: dois 200 e uma linha só", func(t *testing.T) {
		id := pedidoEm(comprador, pedido.StatusPago)
		antes := disponivelDe(cadeira)

		// O teste segura a linha do Pedido, espera os dois cancelamentos
		// ficarem presos, e só então solta. Com a leitura travada, o segundo
		// espera o primeiro comitar e lê CANCELADO: sucesso sem efeito. Sem o
		// FOR UPDATE, os dois leem PAGO, esperam no compare-and-swap, e o
		// perdedor relê CANCELADO e sai FORA_DA_JANELA_DE_CANCELAMENTO de um
		// cancelamento que deu certo.
		travada, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação que trava: %v", err)
		}
		defer travada.Rollback(context.WithoutCancel(ctx))
		if _, err := travada.Exec(ctx, `SELECT id FROM pedido.pedido WHERE id = $1::uuid FOR UPDATE`, id); err != nil {
			t.Fatalf("travar a linha do Pedido: %v", err)
		}

		var espera sync.WaitGroup
		resultados := make([]resultado, 2)
		for i := range 2 {
			espera.Go(func() { cancelarEmParalelo(id, &resultados[i]) })
		}
		presos := esperarPresos(2)
		if err := travada.Rollback(ctx); err != nil {
			t.Fatalf("soltar a trava: %v", err)
		}
		espera.Wait()
		if presos < 2 {
			t.Fatalf("só %d requisição presa em trava; o teste não provou a sobreposição", presos)
		}

		for i, r := range resultados {
			if r.status != http.StatusOK || r.corpo.Status != "CANCELADO" {
				t.Errorf("clique %d: status = %d, corpo = %+v; quero 200 e CANCELADO", i+1, r.status, r.corpo)
			}
		}
		if n := cancelamentos(id); n != 1 {
			t.Errorf("%d linhas de cancelamento no histórico, quero 1: %v", n, transicoesDe(t, pool, id))
		}
		if r := reservasDe(t, pool, id); !slices.Equal(r, []string{"LIBERADA"}) {
			t.Errorf("reservas = %v", r)
		}
		if d := disponivelDe(cadeira); d != antes+1 {
			t.Errorf("disponível = %v, quero %v: a unidade volta uma vez", d, antes+1)
		}
	})

	t.Run("o envio comitado enquanto o cancelamento espera recusa com ENVIADO", func(t *testing.T) {
		id := pedidoEm(comprador, pedido.StatusSeparando)

		// O Administrador (ou a simulação) leva o Pedido a ENVIADO numa
		// transação que ainda não comitou: o CAS dela já prendeu a linha, e o
		// Consolidar já baixou o Estoque total.
		envio, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação do envio: %v", err)
		}
		defer envio.Rollback(context.WithoutCancel(ctx))
		if err := pedido.Transicionar(ctx, envio, id, pedido.StatusSeparando, pedido.StatusEnviado, pedido.AtorAdministrador, ""); err != nil {
			t.Fatalf("enviar: %v", err)
		}

		var espera sync.WaitGroup
		var r resultado
		espera.Go(func() { cancelarEmParalelo(id, &r) })
		presos := esperarPresos(1)
		if err := envio.Commit(ctx); err != nil {
			t.Fatalf("commit do envio: %v", err)
		}
		espera.Wait()
		if presos < 1 {
			t.Fatal("o cancelamento não ficou preso na linha do Pedido; o teste não provou a sobreposição")
		}

		if r.status != http.StatusConflict || r.corpo.Erro.Codigo != "FORA_DA_JANELA_DE_CANCELAMENTO" ||
			r.corpo.Erro.Dados["status"] != "ENVIADO" {
			t.Errorf("status = %d, corpo = %+v; quero 409 FORA_DA_JANELA_DE_CANCELAMENTO com dados.status = ENVIADO", r.status, r.corpo)
		}
		if s := statusDe(t, pool, id); s != "ENVIADO" {
			t.Errorf("status = %s, quero ENVIADO", s)
		}
		if n := cancelamentos(id); n != 0 {
			t.Errorf("%d linhas de cancelamento; o envio ganhou", n)
		}
		if r := reservasDe(t, pool, id); !slices.Equal(r, []string{"CONSOLIDADA"}) {
			t.Errorf("reservas = %v; a Reserva do Pedido enviado consolida, e não volta", r)
		}
	})

	t.Run("a aprovação comitada enquanto o cancelamento espera: o PAGO é cancelado", func(t *testing.T) {
		id := pedidoEm(comprador, pedido.StatusAguardandoPagamento)
		antes := disponivelDe(cadeira)

		// A varredura aplica a aprovação numa transação que ainda não comitou:
		// o CAS dela já prendeu a linha do Pedido. Com a leitura travada, o
		// cancelamento espera, lê PAGO — ainda na janela — e cancela. Sem o
		// FOR UPDATE, ele leria AGUARDANDO_PAGAMENTO, perderia o CAS e sairia
		// ESTADO_JA_AVANCADO de um Pedido que ainda cancela.
		aprovacao, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação da aprovação: %v", err)
		}
		defer aprovacao.Rollback(context.WithoutCancel(ctx))
		if err := pedido.Transicionar(ctx, aprovacao, id, pedido.StatusAguardandoPagamento, pedido.StatusPago, pedido.AtorProvedor, ""); err != nil {
			t.Fatalf("aprovar: %v", err)
		}

		var espera sync.WaitGroup
		var r resultado
		espera.Go(func() { cancelarEmParalelo(id, &r) })
		presos := esperarPresos(1)
		if err := aprovacao.Commit(ctx); err != nil {
			t.Fatalf("commit da aprovação: %v", err)
		}
		espera.Wait()
		if presos < 1 {
			t.Fatal("o cancelamento não ficou preso na linha do Pedido; o teste não provou a sobreposição")
		}

		if r.status != http.StatusOK || r.corpo.Status != "CANCELADO" {
			t.Fatalf("status = %d, corpo = %+v; quero 200 e CANCELADO", r.status, r.corpo)
		}
		historico := transicoesDe(t, pool, id)
		quer := []string{
			"|AGUARDANDO_PAGAMENTO|COMPRADOR",
			"AGUARDANDO_PAGAMENTO|PAGO|PROVEDOR",
			"PAGO|CANCELADO|COMPRADOR",
		}
		if !slices.Equal(historico, quer) {
			t.Errorf("histórico = %v, quero %v", historico, quer)
		}
		if r := reservasDe(t, pool, id); !slices.Equal(r, []string{"LIBERADA"}) {
			t.Errorf("reservas = %v; o cancelamento do PAGO libera a Reserva", r)
		}
		if d := disponivelDe(cadeira); d != antes+1 {
			t.Errorf("disponível = %v, quero %v: a unidade volta", d, antes+1)
		}
	})

	t.Run("a aprovação que chega depois do cancelamento não ressuscita o Pedido", func(t *testing.T) {
		id := pedidoEm(comprador, pedido.StatusAguardandoPagamento)
		oCorpo(postarCancelamento(t, rotas, id, comprador), id)

		// O Provedor aprova assim mesmo: `pagamento` não conhece `pedido`.
		emitirPara(id)
		varrer()

		if s := statusDe(t, pool, id); s != "CANCELADO" {
			t.Errorf("status = %s; a aprovação tardia não move o Pedido cancelado", s)
		}
		if r := reservasDe(t, pool, id); !slices.Equal(r, []string{"LIBERADA"}) {
			t.Errorf("reservas = %v; a aprovação sobre cancelado não reserva nada", r)
		}
		chave := pagamento.ChaveIdempotencia(pagamento.IDExterno(id, 1))
		if tem := confirmacoesDe(t, pool, id); !slices.Equal(tem, []string{chave + "|APROVADO|NAO_APLICAVEL_SINALIZADA"}) {
			t.Errorf("inbox = %v; quero a aprovação sinalizada", tem)
		}
		if n := len(transicoesDe(t, pool, id)); n != 2 {
			t.Errorf("histórico com %d linhas, quero o nascimento e o cancelamento", n)
		}
	})

	// O alheio está em PAGO, um Status de onde o cancelamento sai: a posse é
	// conferida antes do Status, e o 404 é o mesmo do inexistente (AD-11). A
	// igualdade byte a byte está em negacaoPorDono.
	t.Run("Pedido alheio, inexistente ou malformado é 404, e sem Sessão é 401", func(t *testing.T) {
		dele := pedidoEm(outro, pedido.StatusPago)
		antes := fotografia(dele)
		for _, caso := range []struct {
			nome, id string
			cookie   *http.Cookie
			quer     int
		}{
			{"alheio", dele, comprador, http.StatusNotFound},
			{"inexistente", uuidNuncaUsado, comprador, http.StatusNotFound},
			{"malformado", "nao-e-uuid", comprador, http.StatusNotFound},
			{"sem Sessão", dele, nil, http.StatusUnauthorized},
		} {
			resp := postarCancelamento(t, rotas, caso.id, caso.cookie)
			if resp.Code != caso.quer {
				t.Errorf("%s: status = %d (%s), quero %d", caso.nome, resp.Code, resp.Body.String(), caso.quer)
			}
			if caso.quer == http.StatusNotFound && decodificar(t, resp)["codigo"] != "NAO_ENCONTRADO" {
				t.Errorf("%s: corpo = %s; quero NAO_ENCONTRADO", caso.nome, resp.Body.String())
			}
		}
		if depois := fotografia(dele); !slices.Equal(depois, antes) {
			t.Errorf("o cancelamento negado gravou algo:\nantes  %v\ndepois %v", antes, depois)
		}
	})
}
