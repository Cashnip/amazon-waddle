package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
)

// recusaENovaTentativa é a matriz de servidor da 5.10 contra o banco de
// verdade: a recusa emitida pelo Provedor Simulado e aplicada pela varredura,
// a nova Tentativa de Pagamento a partir do próprio Pedido, e as recusas dela
// — sem Estoque, teto, duplo clique e Pedido alheio —, cada uma conferindo que
// nada foi gravado.
//
// Tudo é próprio: conta, Produtos e Estoque. O Produto de R$ 249,90 dos
// subtestes da 1.7 também cai em `,90`, mas os Pedidos dele precisam continuar
// aguardando pagamento para os subtestes que vêm depois, e a Reserva que a nova
// Tentativa deixa ativa mexeria no disponível que eles conferem.
//
// A ordem dos subtestes importa: eles andam com os mesmos Pedidos.
func recusaENovaTentativa(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Rita Recusa","email":"rita.recusa@exemplo.br","senha":"senha-da-rita-1"}`), http.StatusCreated)
	outro := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Olavo Outro","email":"olavo.outro@exemplo.br","senha":"senha-do-olavo-1"}`), http.StatusCreated)

	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja da Recusa"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Nova Tentativa"}`, admin), http.StatusCreated)
	// R$ 49,90 mais o Frete do Sudeste (reais inteiros) fecha em `,90`: a
	// faixa que o Simulado recusa na primeira Tentativa (§7.1).
	novoProduto := func(nome string, estoque int) string {
		t.Helper()
		corpo, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": 4990, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": estoque, "ativo": true,
		})
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(corpo), admin), http.StatusCreated)
	}
	ultimo := novoProduto("Relógio da Última Unidade", 1)
	caneca := novoProduto("Caneca da Nova Tentativa", 10)

	// emitirPara é o tique do Provedor Simulado com o transporte de verdade,
	// mas entregando só as Tentativas destes Pedidos: as dos outros subtestes
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
		if err := pagamento.EmitirConfirmacoesDevidas(ctx, pool, simuladoDeTeste, 0, enviar); err != nil {
			t.Fatalf("emitir: %v", err)
		}
	}
	varrer := func() {
		t.Helper()
		if err := pedido.Varrer(ctx, pool); err != nil {
			t.Fatalf("varrer: %v", err)
		}
	}
	recusar := func(pedidoID string, numero int32) {
		t.Helper()
		c := pagamento.Confirmacao{IDExterno: pagamento.IDExterno(pedidoID, numero), Resultado: pagamento.Recusado}
		if r := postarWebhook(t, rotas, corpoDe(t, c)); r.Code != http.StatusOK {
			t.Fatalf("webhook = %d (%s), quero 200", r.Code, r.Body.String())
		}
	}
	tentarDeNovo := func(pedidoID string, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		return comCorpo(t, rotas, http.MethodPost, "/api/v1/pedidos/"+pedidoID+"/tentativas", "", "", cookie)
	}
	lerDetalhe := func(pedidoID string) map[string]any {
		t.Helper()
		resp := pegarPedido(t, rotas, pedidoID, comprador)
		if resp.Code != http.StatusOK {
			t.Fatalf("ler o Pedido = %d (%s)", resp.Code, resp.Body.String())
		}
		return decodificar(t, resp)
	}
	// numero lê um campo que tem de existir: sem o `ok`, um campo renomeado
	// leria zero, e as asserções de "quero 0" passariam sem conferir nada.
	numero := func(corpo map[string]any, campo string) float64 {
		t.Helper()
		n, ok := corpo[campo].(float64)
		if !ok {
			t.Fatalf("%s = %v; quero um número", campo, corpo[campo])
		}
		return n
	}
	disponivelDe := func(produto string) float64 {
		t.Helper()
		return numero(decodificar(t, pegar(t, rotas, "/api/v1/produtos/"+produto)), "estoque_disponivel")
	}
	// oItem é o único Item de Pedido do Detalhe; qualquer outra forma falha
	// aqui, e não num índice fora do alcance.
	oItem := func(d map[string]any) map[string]any {
		t.Helper()
		itens, _ := d["itens"].([]any)
		if len(itens) != 1 {
			t.Fatalf("itens = %v, quero um", d["itens"])
		}
		item, ok := itens[0].(map[string]any)
		if !ok {
			t.Fatalf("item = %v", itens[0])
		}
		return item
	}
	tentativasDe := func(pedidoID string) []string {
		t.Helper()
		return textoDe(t, pool, `
			SELECT numero::text FROM pagamento.tentativa_pagamento
			WHERE pedido_id = $1::uuid ORDER BY numero`, pedidoID)
	}
	contarReservas := func(pedidoID, estado string) int {
		t.Helper()
		n := 0
		for _, r := range reservasDe(t, pool, pedidoID) {
			if r == estado {
				n++
			}
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
	// fotografia é tudo que a nova Tentativa poderia gravar sobre o Pedido:
	// Status, Tentativas, histórico, Reservas (com Produto, quantidade e
	// estado) e inbox. A recusa tem de deixá-la idêntica.
	fotografia := func(pedidoID string) []string {
		t.Helper()
		f := []string{"status=" + statusDe(t, pool, pedidoID)}
		f = append(f, tentativasDe(pedidoID)...)
		f = append(f, transicoesDe(t, pool, pedidoID)...)
		f = append(f, textoDe(t, pool, `
			SELECT produto_id::text || '|' || quantidade || '|' || estado
			FROM catalogo.reserva_estoque WHERE pedido_id = $1::uuid ORDER BY id`, pedidoID)...)
		return append(f, confirmacoesDe(t, pool, pedidoID)...)
	}
	nadaGravado := func(pedidoID string, antes []string) {
		t.Helper()
		if s := statusDe(t, pool, pedidoID); s != "PAGAMENTO_RECUSADO" {
			t.Errorf("status = %s; o Pedido tinha de permanecer recusado", s)
		}
		if depois := fotografia(pedidoID); !slices.Equal(depois, antes) {
			t.Errorf("a recusa gravou algo:\nantes  %v\ndepois %v", antes, depois)
		}
	}

	pedidoID := idDe(t, pedidoPeloCheckout(t, rotas, comprador, ultimo, 1), http.StatusCreated)
	var dele string // o Pedido do outro Comprador, que o subteste do Estoque cria

	t.Run("a recusa do Provedor é aplicada e o Estoque volta", func(t *testing.T) {
		if d := disponivelDe(ultimo); d != 0 {
			t.Fatalf("disponível antes da recusa = %v, quero 0 (a Reserva levou a unidade)", d)
		}
		emitirPara(pedidoID)
		chave := pagamento.ChaveIdempotencia(pagamento.IDExterno(pedidoID, 1))
		if tem := confirmacoesDe(t, pool, pedidoID); !slices.Equal(tem, []string{chave + "|RECUSADO|PENDENTE"}) {
			t.Fatalf("inbox = %v; quero a recusa emitida e pendente", tem)
		}
		varrer()

		if s := statusDe(t, pool, pedidoID); s != "PAGAMENTO_RECUSADO" {
			t.Fatalf("status = %s, quero PAGAMENTO_RECUSADO", s)
		}
		// O literal, e não a constante: é ele que a tela procura em
		// `frasesDoMotivo` (web/lib/pedido.ts), e mudar só a constante
		// passaria por este teste e calaria a frase na tela.
		if l := ultimaLinha(pedidoID); l != "AGUARDANDO_PAGAMENTO|PAGAMENTO_RECUSADO|PROVEDOR|RECUSADO_PELO_PROVEDOR" {
			t.Errorf("última transição = %s", l)
		}
		if tem := confirmacoesDe(t, pool, pedidoID); !slices.Equal(tem, []string{chave + "|RECUSADO|APLICADA"}) {
			t.Errorf("inbox = %v; quero a recusa aplicada", tem)
		}
		if r := reservasDe(t, pool, pedidoID); !slices.Equal(r, []string{"LIBERADA"}) {
			t.Errorf("reservas = %v; a recusa libera a Reserva", r)
		}
		// Roteiro B, passo 3: a Página de Produto em outra aba.
		if d := disponivelDe(ultimo); d != 1 {
			t.Errorf("disponível depois da recusa = %v, quero 1", d)
		}
		d := lerDetalhe(pedidoID)
		if n := numero(d, "tentativas_restantes"); n != tentativasMaxDeTeste-1 {
			t.Errorf("tentativas_restantes = %v, quero %d", n, tentativasMaxDeTeste-1)
		}
		if d["expira_em"] != nil {
			t.Errorf("expira_em = %v; recusado não tem prazo correndo", d["expira_em"])
		}
		if item := oItem(d); item["disponivel"] != true {
			t.Errorf("item = %v; o Item de Pedido segue no Pedido, e disponível", item)
		}

		// Varrer de novo não faz nada: a recusa já é terminal.
		varrer()
		if n := len(transicoesDe(t, pool, pedidoID)); n != 2 {
			t.Errorf("histórico com %d linhas depois da segunda varredura, quero 2", n)
		}
	})

	t.Run("sem Estoque, a nova Tentativa é recusada e nada é gravado", func(t *testing.T) {
		// Outro Comprador leva a única unidade enquanto o Pedido está recusado.
		dele = idDe(t, pedidoPeloCheckout(t, rotas, outro, ultimo, 1), http.StatusCreated)
		antes := fotografia(pedidoID)

		resp := tentarDeNovo(pedidoID, comprador)
		if resp.Code != http.StatusConflict {
			t.Fatalf("status = %d (%s), quero 409", resp.Code, resp.Body.String())
		}
		env := decodificar(t, resp)
		dados, _ := env["dados"].(map[string]any)
		if env["codigo"] != "ESTOQUE_INSUFICIENTE" || dados["produto_id"] != ultimo {
			t.Errorf("resposta = %v; quero ESTOQUE_INSUFICIENTE nomeando o Produto", env)
		}
		nadaGravado(pedidoID, antes)
		if item := oItem(lerDetalhe(pedidoID)); item["disponivel"] != false {
			t.Errorf("item = %v; sem Estoque o Item lê indisponível, e a tela esconde a ação", item)
		}

		// O Pedido do outro é recusado também, e a unidade volta.
		emitirPara(dele)
		varrer()
		if s := statusDe(t, pool, dele); s != "PAGAMENTO_RECUSADO" {
			t.Fatalf("status do outro = %s, quero PAGAMENTO_RECUSADO", s)
		}
	})

	t.Run("a nova Tentativa reserva de novo, recomeça o prazo e o Provedor aprova", func(t *testing.T) {
		resp := tentarDeNovo(pedidoID, comprador)
		if resp.Code != http.StatusCreated {
			t.Fatalf("status = %d (%s), quero 201", resp.Code, resp.Body.String())
		}
		if corpo := decodificar(t, resp); corpo["id"] != pedidoID || corpo["status"] != "AGUARDANDO_PAGAMENTO" {
			t.Errorf("corpo = %v", corpo)
		}
		if tem := tentativasDe(pedidoID); !slices.Equal(tem, []string{"1", "2"}) {
			t.Errorf("Tentativas = %v, quero [1 2]", tem)
		}
		if l := ultimaLinha(pedidoID); l != "PAGAMENTO_RECUSADO|AGUARDANDO_PAGAMENTO|COMPRADOR|" {
			t.Errorf("última transição = %s", l)
		}
		if contarReservas(pedidoID, "ATIVA") != 1 || contarReservas(pedidoID, "LIBERADA") != 1 {
			t.Errorf("reservas = %v; quero a liberada da recusa e uma nova ativa", reservasDe(t, pool, pedidoID))
		}
		if d := disponivelDe(ultimo); d != 0 {
			t.Errorf("disponível = %v, quero 0: a nova Reserva levou a unidade", d)
		}
		d := lerDetalhe(pedidoID)
		if quer := ultimaTransicao(t, pool, pedidoID).Add(expiracaoDaTentativaDeTeste); !instanteDoCampo(t, d, "expira_em").Equal(quer) {
			t.Errorf("expira_em = %v, quero %v (a volta a AGUARDANDO_PAGAMENTO + prazo)", d["expira_em"], quer)
		}
		if n := numero(d, "tentativas_restantes"); n != tentativasMaxDeTeste-2 {
			t.Errorf("tentativas_restantes = %v, quero %d", n, tentativasMaxDeTeste-2)
		}

		// Roteiro B, passo 4: da segunda Tentativa em diante o Simulado aprova.
		emitirPara(pedidoID)
		varrer()
		if s := statusDe(t, pool, pedidoID); s != "PAGO" {
			t.Errorf("status = %s, quero PAGO", s)
		}
		if contarReservas(pedidoID, "ATIVA") != 1 {
			t.Errorf("reservas = %v; PAGO mantém a Reserva", reservasDe(t, pool, pedidoID))
		}
	})

	t.Run("fora de PAGAMENTO_RECUSADO a nova Tentativa sai como estado já avançado", func(t *testing.T) {
		resp := tentarDeNovo(pedidoID, comprador)
		if resp.Code != http.StatusConflict || decodificar(t, resp)["codigo"] != "ESTADO_JA_AVANCADO" {
			t.Fatalf("status = %d (%s), quero 409 ESTADO_JA_AVANCADO", resp.Code, resp.Body.String())
		}
		if tem := tentativasDe(pedidoID); !slices.Equal(tem, []string{"1", "2"}) {
			t.Errorf("Tentativas = %v; nada novo", tem)
		}
	})

	// Um Pedido novo, de Produto com Estoque de sobra, para o duplo clique e o
	// teto.
	segundo := idDe(t, pedidoPeloCheckout(t, rotas, comprador, caneca, 1), http.StatusCreated)

	t.Run("o duplo clique abre uma Tentativa só", func(t *testing.T) {
		emitirPara(segundo)
		varrer()
		if s := statusDe(t, pool, segundo); s != "PAGAMENTO_RECUSADO" {
			t.Fatalf("status = %s, quero PAGAMENTO_RECUSADO", s)
		}

		// Os dois ao mesmo tempo, e de verdade: disparados soltos, cada um
		// termina em milissegundos e o segundo chega depois do commit do
		// primeiro — a suíte passaria com a Tentativa contada antes da
		// transição (verificado por mutação). Então o teste segura a linha do
		// Pedido, espera os dois ficarem presos em trava, e só então solta.
		//
		// Na ordem certa, os dois esperam no compare-and-swap: um ganha, e o
		// outro relê o Status e sai ESTADO_JA_AVANCADO. Com a Tentativa antes
		// do CAS, os dois contariam uma Tentativa, gravariam o número 2, e o
		// segundo sairia em 500 pelo índice único de `id_externo`.
		travada, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação que trava: %v", err)
		}
		defer travada.Rollback(context.WithoutCancel(ctx))
		if _, err := travada.Exec(ctx, `SELECT id FROM pedido.pedido WHERE id = $1::uuid FOR UPDATE`, segundo); err != nil {
			t.Fatalf("travar a linha do Pedido: %v", err)
		}

		var espera sync.WaitGroup
		codigos := make([]string, 2)
		status := make([]int, 2)
		for i := range 2 {
			espera.Go(func() {
				resp := tentarDeNovo(segundo, comprador)
				status[i] = resp.Code
				// `decodificar` chama t.Fatalf, que não pode sair de outra
				// goroutine: o envelope é lido à mão.
				var env struct {
					Erro struct {
						Codigo string `json:"codigo"`
					} `json:"erro"`
				}
				_ = json.Unmarshal(resp.Body.Bytes(), &env)
				codigos[i] = env.Erro.Codigo
			})
		}
		presos := 0
		for prazo := time.Now().Add(10 * time.Second); presos < 2 && time.Now().Before(prazo); {
			if err := pool.QueryRow(ctx, `
				SELECT count(*) FROM pg_stat_activity
				WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&presos); err != nil {
				t.Fatalf("ler as esperas: %v", err)
			}
			if presos < 2 {
				time.Sleep(20 * time.Millisecond)
			}
		}
		if err := travada.Rollback(ctx); err != nil {
			t.Fatalf("soltar a trava: %v", err)
		}
		espera.Wait()
		if presos < 2 {
			t.Fatalf("só %d requisição presa em trava; o teste não provou a sobreposição", presos)
		}
		slices.Sort(status)
		if !slices.Equal(status, []int{http.StatusCreated, http.StatusConflict}) {
			t.Fatalf("status = %v, quero um 201 e um 409", status)
		}
		if !slices.Contains(codigos, "ESTADO_JA_AVANCADO") {
			t.Errorf("códigos = %v; o perdedor sai em ESTADO_JA_AVANCADO", codigos)
		}
		if tem := tentativasDe(segundo); !slices.Equal(tem, []string{"1", "2"}) {
			t.Errorf("Tentativas = %v, quero [1 2]", tem)
		}
		if n := contarReservas(segundo, "ATIVA"); n != 1 {
			t.Errorf("%d Reservas ativas, quero 1", n)
		}
	})

	t.Run("no teto, a nova Tentativa é recusada por pagamento e nada é gravado", func(t *testing.T) {
		// A Tentativa 2 aprovaria sozinha; a recusa entra pelo webhook, que é
		// o que um Provedor real faria.
		recusar(segundo, 2)
		varrer()
		if resp := tentarDeNovo(segundo, comprador); resp.Code != http.StatusCreated {
			t.Fatalf("terceira Tentativa = %d (%s), quero 201", resp.Code, resp.Body.String())
		}
		recusar(segundo, 3)
		varrer()
		if n := numero(lerDetalhe(segundo), "tentativas_restantes"); n != 0 {
			t.Fatalf("tentativas_restantes = %v, quero 0", n)
		}
		antes := fotografia(segundo)

		resp := tentarDeNovo(segundo, comprador)
		if resp.Code != http.StatusConflict || decodificar(t, resp)["codigo"] != "TETO_DE_TENTATIVAS" {
			t.Fatalf("status = %d (%s), quero 409 TETO_DE_TENTATIVAS", resp.Code, resp.Body.String())
		}
		nadaGravado(segundo, antes)
	})

	t.Run("a recusa de uma Tentativa superada não derruba a Reserva da nova", func(t *testing.T) {
		terceiro := idDe(t, pedidoPeloCheckout(t, rotas, comprador, caneca, 1), http.StatusCreated)
		// Recusado sem a confirmação da Tentativa 1 ter entrado na inbox: é o
		// estado de que a recusa atrasada precisa para existir. `Transicionar`,
		// e não um UPDATE à mão, para histórico e Reserva serem os de produção
		// — com o mesmo ator e motivo que a varredura gravaria.
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação: %v", err)
		}
		defer tx.Rollback(ctx)
		if err := pedido.Transicionar(ctx, tx, terceiro, pedido.StatusAguardandoPagamento,
			pedido.StatusPagamentoRecusado, pedido.AtorProvedor, pedido.MotivoRecusadoPeloProvedor); err != nil {
			t.Fatalf("recusar o Pedido: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit da recusa: %v", err)
		}
		if resp := tentarDeNovo(terceiro, comprador); resp.Code != http.StatusCreated {
			t.Fatalf("nova Tentativa = %d (%s), quero 201", resp.Code, resp.Body.String())
		}

		// A recusa da Tentativa 1 chega atrasada, com a 2 já aberta.
		recusar(terceiro, 1)
		varrer()
		if s := statusDe(t, pool, terceiro); s != "AGUARDANDO_PAGAMENTO" {
			t.Errorf("status = %s; a recusa superada não move o Pedido", s)
		}
		if n := contarReservas(terceiro, "ATIVA"); n != 1 {
			t.Errorf("reservas = %v; a Reserva da Tentativa 2 continua ativa", reservasDe(t, pool, terceiro))
		}
		chave := pagamento.ChaveIdempotencia(pagamento.IDExterno(terceiro, 1))
		if tem := confirmacoesDe(t, pool, terceiro); !slices.Contains(tem, chave+"|RECUSADO|NAO_APLICAVEL_SINALIZADA") {
			t.Errorf("inbox = %v; quero a recusa superada sinalizada", tem)
		}
	})

	// A entrada da 5.1 que esta estória fecha: o Produto desativado depois de o
	// Pedido nascer sai igual ao esgotado (FR-12), e o Item de Pedido segue
	// com o nome congelado para a tela nomeá-lo.
	t.Run("Produto desativado depois do Pedido recusa a nova Tentativa como esgotado", func(t *testing.T) {
		const nome = "Luminária Desativada"
		luminaria := novoProduto(nome, 5)
		quarto := idDe(t, pedidoPeloCheckout(t, rotas, comprador, luminaria, 1), http.StatusCreated)
		emitirPara(quarto)
		varrer()
		if s := statusDe(t, pool, quarto); s != "PAGAMENTO_RECUSADO" {
			t.Fatalf("status = %s, quero PAGAMENTO_RECUSADO", s)
		}
		corpo, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": 4990, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": 5, "ativo": false,
		})
		if resp := produtoCom(t, rotas, http.MethodPut, "/"+luminaria, string(corpo), admin); resp.Code != http.StatusOK {
			t.Fatalf("desativar o Produto = %d (%s)", resp.Code, resp.Body.String())
		}
		antes := fotografia(quarto)

		resp := tentarDeNovo(quarto, comprador)
		env := decodificar(t, resp)
		dados, _ := env["dados"].(map[string]any)
		if resp.Code != http.StatusConflict || env["codigo"] != "ESTOQUE_INSUFICIENTE" || dados["produto_id"] != luminaria {
			t.Fatalf("status = %d (%v); quero 409 ESTOQUE_INSUFICIENTE nomeando o Produto", resp.Code, env)
		}
		nadaGravado(quarto, antes)
		if item := oItem(lerDetalhe(quarto)); item["disponivel"] != false || item["nome"] != nome {
			t.Errorf("item = %v; quero indisponível e com o nome congelado", item)
		}
	})

	// O alheio é o Pedido recusado do outro Comprador, com Tentativas
	// restantes: um Status de onde a nova Tentativa sai, e não um PAGO que
	// recusaria sozinho. A posse é conferida antes do Status e do Estoque, e o
	// 404 é o mesmo do inexistente (AD-11).
	t.Run("Pedido alheio, inexistente ou malformado é 404, e sem Sessão é 401", func(t *testing.T) {
		if s := statusDe(t, pool, dele); s != "PAGAMENTO_RECUSADO" {
			t.Fatalf("o Pedido do outro está em %s; quero PAGAMENTO_RECUSADO", s)
		}
		antes := fotografia(dele)
		for _, caso := range []struct {
			nome, id string
			cookie   *http.Cookie
			quer     int
		}{
			{"alheio", dele, comprador, http.StatusNotFound},
			{"inexistente", "00000000-0000-7000-8000-000000000000", comprador, http.StatusNotFound},
			{"malformado", "nao-e-uuid", comprador, http.StatusNotFound},
			{"sem Sessão", dele, nil, http.StatusUnauthorized},
		} {
			if resp := tentarDeNovo(caso.id, caso.cookie); resp.Code != caso.quer {
				t.Errorf("%s: status = %d (%s), quero %d", caso.nome, resp.Code, resp.Body.String(), caso.quer)
			}
		}
		nadaGravado(dele, antes)
	})
}
