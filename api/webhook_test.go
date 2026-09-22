package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// Teclado Mecânico Compacto Tucano, R$ 329,00: os centavos caem em `,00`, que
// é a faixa aprovada do §7.1. O Produto de R$ 249,90 dos outros subtestes cai
// em `,90` e fica esperando a Épica 5 — é o recorte da estória, e não um
// descuido do teste.
const produtoAprovado = "b489768a-4430-5081-b888-35f6dec41790"

// simuladoDeTeste são as faixas do `.env` da demonstração.
var simuladoDeTeste = pagamento.Simulado{AprovadoAteCentavos: 89, RecusadoAteCentavos: 94}

// confirmacaoAprovadaLevaOPedidoAPago percorre a estória inteira contra
// Postgres de verdade: a Tentativa que nasceu junto com o Pedido, a emissão
// derivada, o webhook, a chave única absorvendo a repetição, a varredura
// aplicando, e a Reserva que continua ATIVA porque consolidar é da 1.8.
func confirmacaoAprovadaLevaOPedidoAPago(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie) {
	if cookie == nil {
		t.Fatal("sem cookie: o login falhou antes")
	}
	ctx := context.Background()

	resp := pedidoPeloCheckout(t, rotas, cookie, produtoAprovado, 1)
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d (%s), quero 201", resp.Code, resp.Body.String())
	}
	pedidoID, _ := decodificar(t, resp)["id"].(string)
	if pedidoID == "" {
		t.Fatal("o 201 não devolveu o identificador do Pedido")
	}

	// A Tentativa nasceu na mesma transação do Pedido, com o total congelado.
	tentativas := textoDe(t, pool, `
		SELECT id_externo || '|' || total_centavos || '|' || numero
		FROM pagamento.tentativa_pagamento WHERE pedido_id = $1::uuid`, pedidoID)
	quer := pagamento.IDExterno(pedidoID, 1) + "|32900|1"
	if len(tentativas) != 1 || tentativas[0] != quer {
		t.Fatalf("tentativa = %v, quero [%q]", tentativas, quer)
	}

	// A emissão é derivada: atraso zero vence na hora, e o envio entra pela
	// rota de verdade — o mesmo caminho que o `http.Post` do binário usa.
	var enviadas []pagamento.Confirmacao
	enviar := func(_ context.Context, c pagamento.Confirmacao) error {
		enviadas = append(enviadas, c)
		if r := postarWebhook(t, rotas, corpoDe(t, c)); r.Code != http.StatusOK {
			t.Errorf("webhook = %d (%s), quero 200", r.Code, r.Body.String())
		}
		return nil
	}
	if err := pagamento.EmitirConfirmacoesDevidas(ctx, pool, simuladoDeTeste, 0, enviar); err != nil {
		t.Fatalf("emitir = %v", err)
	}
	// Só o caminho aprovado atravessa: os Pedidos de R$ 249,90 dos subtestes
	// anteriores têm Tentativa vencida e não emitem nada.
	if len(enviadas) != 1 || enviadas[0].Resultado != pagamento.Aprovado {
		t.Fatalf("emitidas = %v; quero só a confirmação aprovada desta Tentativa", enviadas)
	}

	chave := pagamento.ChaveIdempotencia(pagamento.IDExterno(pedidoID, 1))
	if tem := confirmacoesDe(t, pool, pedidoID); len(tem) != 1 || tem[0] != chave+"|APROVADO|PENDENTE" {
		t.Fatalf("inbox = %v, quero [%q]", tem, chave+"|APROVADO|PENDENTE")
	}

	// A mesma confirmação entregue de novo: 200 outra vez, e uma linha só. O
	// 23505 é sinal para o código decidir, e nunca resposta ao cliente.
	repetida := postarWebhook(t, rotas, corpoDe(t, enviadas[0]))
	if repetida.Code != http.StatusOK {
		t.Errorf("repetição = %d (%s), quero 200", repetida.Code, repetida.Body.String())
	}
	if strings.Contains(repetida.Body.String(), "23505") {
		t.Error("o 23505 vazou para o cliente")
	}
	if tem := confirmacoesDe(t, pool, pedidoID); len(tem) != 1 {
		t.Fatalf("inbox = %v; a confirmação repetida tinha de ser no-op", tem)
	}

	// Com linha na inbox, a emissão derivada para sozinha — é o que impede o
	// tique de 1 s de reenviar para sempre.
	enviadas = nil
	if err := pagamento.EmitirConfirmacoesDevidas(ctx, pool, simuladoDeTeste, 0, enviar); err != nil {
		t.Fatalf("segunda emissão = %v", err)
	}
	if len(enviadas) != 0 {
		t.Errorf("reemitiu %v; a Tentativa já tem linha na inbox", enviadas)
	}

	// O que a tela lê antes da varredura, pela rota de verdade — é contra este
	// instante que o avanço é comparado depois.
	antesDaVarredura := pegarPedido(t, rotas, pedidoID, cookie)
	if antesDaVarredura.Code != http.StatusOK {
		t.Fatalf("leitura antes da varredura = %d (%s)", antesDaVarredura.Code, antesDaVarredura.Body.String())
	}
	instanteAntes := instanteDe(t, decodificar(t, antesDaVarredura))

	if err := pedido.Varrer(ctx, pool); err != nil {
		t.Fatalf("varrer = %v", err)
	}
	// A mesma rota, depois: o Status virou PAGO e o instante avançou — é
	// exatamente o que o Comprador vê a tela fazer sozinha.
	depoisDaVarredura := pegarPedido(t, rotas, pedidoID, cookie)
	if depoisDaVarredura.Code != http.StatusOK {
		t.Fatalf("leitura depois da varredura = %d (%s)", depoisDaVarredura.Code, depoisDaVarredura.Body.String())
	}
	lido := decodificar(t, depoisDaVarredura)
	if lido["status"] != "PAGO" {
		t.Errorf("status lido pela rota = %v, quero PAGO", lido["status"])
	}
	if instanteDepois := instanteDe(t, lido); !instanteDepois.After(instanteAntes) {
		t.Errorf("atualizado_em = %v; quero estritamente depois de %v", instanteDepois, instanteAntes)
	}

	if tem := confirmacoesDe(t, pool, pedidoID); len(tem) != 1 || !strings.HasSuffix(tem[0], "|APLICADA") {
		t.Errorf("inbox depois da varredura = %v; quero APLICADA", tem)
	}
	if s := statusDe(t, pool, pedidoID); s != "PAGO" {
		t.Errorf("status = %s, quero PAGO", s)
	}
	// NFR-9: o avanço deixa a sua linha, com o Provedor como ator.
	historico := transicoesDe(t, pool, pedidoID)
	if len(historico) != 2 || historico[1] != "AGUARDANDO_PAGAMENTO|PAGO|PROVEDOR" {
		t.Errorf("histórico = %v; quero o nascimento e o avanço para PAGO", historico)
	}

	// PAGO mantém a Reserva ATIVA: consolidar é da 1.8, e nenhum total de
	// Estoque muda aqui.
	reservas := reservasDe(t, pool, pedidoID)
	if len(reservas) != 1 || reservas[0] != "ATIVA" {
		t.Errorf("reserva = %v, quero [ATIVA]", reservas)
	}

	// Varrer de novo sobre o que já está terminal não faz nada: a inbox não
	// devolve mais esta linha, e o Pedido fica onde está.
	if err := pedido.Varrer(ctx, pool); err != nil {
		t.Fatalf("varredura repetida = %v", err)
	}
	if n := len(transicoesDe(t, pool, pedidoID)); n != 2 {
		t.Errorf("%d transições depois da segunda varredura, quero 2", n)
	}
}

// confirmacaoForaDeAguardandoPagamento: a confirmação que chega para um Pedido
// que já avançou vira NAO_APLICAVEL_SINALIZADA, e o Pedido fica intacto. Aqui
// o Pedido é o mesmo do subteste anterior, já em PAGO, e a Tentativa é uma
// segunda — a corrente, portanto.
func confirmacaoForaDeAguardandoPagamento(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	pedidoID := umPedidoEm(t, pool, "PAGO")

	criarTentativa(t, pool, pedidoID, 2)
	confirmar(t, rotas, pedidoID, 2)
	if err := pedido.Varrer(ctx, pool); err != nil {
		t.Fatalf("varrer = %v", err)
	}

	chave := pagamento.ChaveIdempotencia(pagamento.IDExterno(pedidoID, 2))
	if tem := confirmacoesDe(t, pool, pedidoID); !slices.Contains(tem, chave+"|APROVADO|NAO_APLICAVEL_SINALIZADA") {
		t.Errorf("inbox = %v; quero a segunda confirmação sinalizada", tem)
	}
	if s := statusDe(t, pool, pedidoID); s != "PAGO" {
		t.Errorf("status = %s; o Pedido tinha de ficar intacto", s)
	}
}

// confirmacaoDeTentativaSuperada: só é aplicada a confirmação da Tentativa
// corrente. Aqui o Pedido ainda aguarda pagamento, mas a confirmação é da
// Tentativa 1 e já existe uma 2 — ela não se aplica, e também não pode ficar
// PENDENTE para sempre.
func confirmacaoDeTentativaSuperada(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	pedidoID := umPedidoEm(t, pool, "AGUARDANDO_PAGAMENTO")

	criarTentativa(t, pool, pedidoID, 2)
	confirmar(t, rotas, pedidoID, 1)
	if err := pedido.Varrer(ctx, pool); err != nil {
		t.Fatalf("varrer = %v", err)
	}

	chave := pagamento.ChaveIdempotencia(pagamento.IDExterno(pedidoID, 1))
	if tem := confirmacoesDe(t, pool, pedidoID); !slices.Contains(tem, chave+"|APROVADO|NAO_APLICAVEL_SINALIZADA") {
		t.Errorf("inbox = %v; quero a confirmação da Tentativa superada sinalizada", tem)
	}
	if s := statusDe(t, pool, pedidoID); s != "AGUARDANDO_PAGAMENTO" {
		t.Errorf("status = %s; a Tentativa superada não avança o Pedido", s)
	}
}

// confirmacaoAprovadaSobrePedidoCancelado é a corrida clássica do checkout
// (FR-26, invariante 7 do addendum §2): o Provedor aprova um pagamento cujo
// Pedido o Comprador já cancelou. O Status não muda e a Reserva continua
// liberada, mas a informação de que o pagamento foi aprovado fica registrada
// na Tentativa — é dela que a 6.5 deriva o sinal ao Administrador, e é ela que
// torna possível o estorno da fase 2 sem arqueologia de log.
//
// A emissão derivada não entra aqui de propósito: ela é provada no caminho
// feliz, e rodá-la de novo varreria as Tentativas que os subtestes anteriores
// inventaram. A confirmação entra pela rota de verdade, que é o que importa.
func confirmacaoAprovadaSobrePedidoCancelado(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie) {
	if cookie == nil {
		t.Fatal("sem cookie: o login falhou antes")
	}
	ctx := context.Background()

	pedidoID := idDe(t, pedidoPeloCheckout(t, rotas, cookie, produtoAprovado, 1), http.StatusCreated)

	// O Comprador cancela. A tela do cancelamento é da 6.3, então o caminho é
	// `Transicionar` — e não um UPDATE à mão, que deixaria o histórico e a
	// Reserva do teste divergirem dos de produção.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir a transação: %v", err)
	}
	defer tx.Rollback(ctx)
	if err := pedido.Transicionar(ctx, tx, pedidoID,
		pedido.StatusAguardandoPagamento, pedido.StatusCancelado, pedido.AtorComprador, ""); err != nil {
		t.Fatalf("cancelar o Pedido: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit do cancelamento: %v", err)
	}
	// O cancelamento liberou a Reserva: é este estado que a confirmação
	// aprovada não pode mexer.
	if r := reservasDe(t, pool, pedidoID); len(r) != 1 || r[0] != "LIBERADA" {
		t.Fatalf("reserva depois do cancelamento = %v, quero [LIBERADA]", r)
	}
	// Nascimento e cancelamento, com o ator que o NFR-9 exige. É contra estas
	// duas linhas que a ausência de transição nova é conferida depois — uma
	// contagem sozinha passaria mesmo se o cancelamento não gravasse nada.
	transicoesAntes := transicoesDe(t, pool, pedidoID)
	quer := "AGUARDANDO_PAGAMENTO|CANCELADO|COMPRADOR"
	if len(transicoesAntes) != 2 || transicoesAntes[1] != quer {
		t.Fatalf("histórico depois do cancelamento = %v, quero o nascimento e %q", transicoesAntes, quer)
	}

	// O Provedor aprova assim mesmo: `pagamento` não conhece `pedido` (AD-7) e
	// não tem como saber que o Pedido foi cancelado. Quem decide não aplicar é
	// a varredura, com a trava do Pedido na mão.
	confirmar(t, rotas, pedidoID, 1)
	// A mesma confirmação de novo: 200 outra vez, e uma linha só.
	repetida := postarWebhook(t, rotas, corpoDe(t, pagamento.Confirmacao{
		IDExterno: pagamento.IDExterno(pedidoID, 1), Resultado: pagamento.Aprovado,
	}))
	if repetida.Code != http.StatusOK {
		t.Errorf("repetição = %d (%s), quero 200", repetida.Code, repetida.Body.String())
	}

	// O aviso é a única saída em tempo real deste caso, e some sem ruído se
	// alguém apagar o `if`: o estado terminal já seria o mesmo sem ele.
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(plataforma.NovoLogger(&registro, "varredura"))
	err = pedido.Varrer(ctx, pool)
	slog.SetDefault(anterior)
	if err != nil {
		t.Fatalf("varrer = %v", err)
	}
	if aviso := registro.String(); !strings.Contains(aviso, "pagamento aprovado sobre Pedido cancelado") ||
		!strings.Contains(aviso, pedidoID) {
		t.Errorf("log da varredura = %q; quero o aviso nomeando o Pedido", aviso)
	}

	if s := statusDe(t, pool, pedidoID); s != "CANCELADO" {
		t.Errorf("status = %s; a aprovação tardia não ressuscita o Pedido", s)
	}
	chave := pagamento.ChaveIdempotencia(pagamento.IDExterno(pedidoID, 1))
	if tem := confirmacoesDe(t, pool, pedidoID); len(tem) != 1 || tem[0] != chave+"|APROVADO|NAO_APLICAVEL_SINALIZADA" {
		t.Errorf("inbox = %v, quero [%q]", tem, chave+"|APROVADO|NAO_APLICAVEL_SINALIZADA")
	}
	if n := len(transicoesDe(t, pool, pedidoID)); n != len(transicoesAntes) {
		t.Errorf("%d transições depois da varredura, quero as %d de antes", n, len(transicoesAntes))
	}
	if r := reservasDe(t, pool, pedidoID); len(r) != 1 || r[0] != "LIBERADA" {
		t.Errorf("reserva = %v; a aprovação sobre Pedido cancelado não reserva nada", r)
	}

	// Varrer de novo não encontra mais a linha: o estado é terminal.
	if err := pedido.Varrer(ctx, pool); err != nil {
		t.Fatalf("varredura repetida = %v", err)
	}
	if n := len(transicoesDe(t, pool, pedidoID)); n != len(transicoesAntes) {
		t.Errorf("%d transições depois da segunda varredura, quero as %d de antes", n, len(transicoesAntes))
	}

	// O sinal que a 6.5 vai mostrar é alcançável sem ler log: metade em
	// `pagamento` (aprovada e sinalizada), metade no Status do Pedido. A
	// junção das duas é em Go, e não em SQL, porque `pedido` só fala com
	// `pagamento` pela porta (AD-1).
	sinalizada := textoDe(t, pool, `
		SELECT c.id::text
		FROM pagamento.confirmacao_recebida c
		JOIN pagamento.tentativa_pagamento t ON t.id = c.tentativa_id
		WHERE t.pedido_id = $1::uuid
		  AND c.resultado = 'APROVADO' AND c.estado = 'NAO_APLICAVEL_SINALIZADA'`, pedidoID)
	if len(sinalizada) != 1 || statusDe(t, pool, pedidoID) != "CANCELADO" {
		t.Errorf("o predicado da 6.5 não encontra o Pedido: confirmações = %v", sinalizada)
	}
}

// confirmacaoDePedidoTravadoFicaPendente é a última linha da matriz: o tique
// que encontra o Pedido travado por outro caminho não espera nem falha — o
// `FOR UPDATE SKIP LOCKED` do `TravarPedido` o deixa passar, a confirmação
// continua `PENDENTE`, e o tique seguinte a aplica. É o que garante que uma
// confirmação não se perde por chegar no instante errado.
func confirmacaoDePedidoTravadoFicaPendente(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie) {
	if cookie == nil {
		t.Fatal("sem cookie: o login falhou antes")
	}
	ctx := context.Background()

	pedidoID := idDe(t, pedidoPeloCheckout(t, rotas, cookie, produtoAprovado, 1), http.StatusCreated)
	confirmar(t, rotas, pedidoID, 1)

	// Outro caminho segura a linha do Pedido. Não é encenação: é exatamente o
	// que dois tiques sobrepostos, ou o tique e uma requisição, fazem.
	travada, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir a transação que trava: %v", err)
	}
	// `defer`, e não Rollback em cada saída: qualquer t.Fatalf de dentro de um
	// helper deixaria a trava de pé sobre a linha do Pedido e uma conexão presa
	// pelo resto do pacote. O Rollback explícito lá embaixo é quem solta a
	// trava no caminho feliz; este é a rede.
	defer travada.Rollback(context.WithoutCancel(ctx))
	if _, err := travada.Exec(ctx, `SELECT id FROM pedido.pedido WHERE id = $1::uuid FOR UPDATE`, pedidoID); err != nil {
		t.Fatalf("travar a linha do Pedido: %v", err)
	}

	if err := pedido.Varrer(ctx, pool); err != nil {
		t.Fatalf("varrer com o Pedido travado = %v; quero nil", err)
	}
	chave := pagamento.ChaveIdempotencia(pagamento.IDExterno(pedidoID, 1))
	if tem := confirmacoesDe(t, pool, pedidoID); len(tem) != 1 || tem[0] != chave+"|APROVADO|PENDENTE" {
		t.Fatalf("inbox = %v, quero [%q]", tem, chave+"|APROVADO|PENDENTE")
	}
	if s := statusDe(t, pool, pedidoID); s != "AGUARDANDO_PAGAMENTO" {
		t.Fatalf("status = %s; o tique travado não avança nada", s)
	}

	if err := travada.Rollback(ctx); err != nil {
		t.Fatalf("soltar a trava: %v", err)
	}
	// Solta a trava, o tique seguinte aplica: nada se perdeu.
	if err := pedido.Varrer(ctx, pool); err != nil {
		t.Fatalf("varrer depois da trava = %v", err)
	}
	if tem := confirmacoesDe(t, pool, pedidoID); len(tem) != 1 || tem[0] != chave+"|APROVADO|APLICADA" {
		t.Errorf("inbox = %v, quero [%q]", tem, chave+"|APROVADO|APLICADA")
	}
	if s := statusDe(t, pool, pedidoID); s != "PAGO" {
		t.Errorf("status = %s, quero PAGO", s)
	}
}

// webhookRecusaEntradaRuim cobre as duas linhas de recusa da matriz: corpo que
// nem chega ao domínio é 400, e chave sem Tentativa é 404 sem gravar nada.
func webhookRecusaEntradaRuim(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	antes := contarConfirmacoes(t, pool)
	for _, caso := range []string{
		`{isso não é json`,
		`{"id_externo":"` + strings.Repeat("a", corpoMaximo) + `","resultado":"APROVADO"}`,
		// Resultado fora do CHECK é 400, e não o 500 que a violação da
		// restrição produziria se a validação não existisse aqui.
		`{"id_externo":"sim-x-1","resultado":"TALVEZ"}`,
	} {
		resp := postarWebhook(t, rotas, caso)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("%.40s: status = %d (%s), quero 400", caso, resp.Code, resp.Body.String())
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != "ENTRADA_INVALIDA" {
			t.Errorf("%.40s: codigo = %v", caso, codigo)
		}
	}

	resp := postarWebhook(t, rotas, `{"id_externo":"sim-nao-existe-1","resultado":"APROVADO"}`)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("chave desconhecida: status = %d (%s), quero 404", resp.Code, resp.Body.String())
	}
	if codigo := decodificar(t, resp)["codigo"]; codigo != "NAO_ENCONTRADO" {
		t.Errorf("chave desconhecida: codigo = %v", codigo)
	}
	if depois := contarConfirmacoes(t, pool); depois != antes {
		t.Errorf("%d confirmações depois das recusas, quero as %d de antes", depois, antes)
	}
}

// webhookExigeOSegredo é a linha que separa "o Provedor confirmou" de "o
// Comprador confirmou a si mesmo": a chave de idempotência é derivada do
// identificador do Pedido, que volta no 201 e vai na URL da tela de
// acompanhamento — sem o segredo no cabeçalho, qualquer Comprador levaria o
// próprio Pedido a PAGO sem pagar, inclusive um da faixa que o Simulado nunca
// aprova.
func webhookExigeOSegredo(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie) {
	resp := pedidoPeloCheckout(t, rotas, cookie, produtoSemeado, 1)
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d (%s), quero 201", resp.Code, resp.Body.String())
	}
	pedidoID, _ := decodificar(t, resp)["id"].(string)
	corpo := corpoDe(t, pagamento.Confirmacao{
		IDExterno: pagamento.IDExterno(pedidoID, 1),
		Resultado: pagamento.Aprovado,
	})

	antes := contarConfirmacoes(t, pool)
	// Cabeçalho ausente e cabeçalho errado saem no mesmo 401, sem dizer qual
	// dos dois foi.
	var envelopes []string
	for _, segredo := range []string{"", "nao-e-o-segredo"} {
		resp := postarWebhookCom(t, rotas, corpo, segredo)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("segredo %q: status = %d (%s), quero 401", segredo, resp.Code, resp.Body.String())
		}
		envelope := decodificar(t, resp)
		if envelope["codigo"] != "NAO_AUTORIZADO" {
			t.Errorf("segredo %q: codigo = %v", segredo, envelope["codigo"])
		}
		delete(envelope, "correlacao")
		envelopes = append(envelopes, fmt.Sprint(envelope))
	}
	if envelopes[0] != envelopes[1] {
		t.Errorf("as duas respostas diferem: %s vs %s", envelopes[0], envelopes[1])
	}
	if depois := contarConfirmacoes(t, pool); depois != antes {
		t.Errorf("%d confirmações depois dos 401, quero as %d de antes", depois, antes)
	}
	if s := statusDe(t, pool, pedidoID); s != "AGUARDANDO_PAGAMENTO" {
		t.Errorf("status = %s; o 401 não pode ter avançado nada", s)
	}
}

// --- ajudantes ---

func postarWebhook(t *testing.T, rotas http.Handler, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	return postarWebhookCom(t, rotas, corpo, segredoDeTeste)
}

func postarWebhookCom(t *testing.T, rotas http.Handler, corpo, segredo string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/pagamento", strings.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	if segredo != "" {
		req.Header.Set("X-Azamon-Segredo", segredo)
	}
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
}

func corpoDe(t *testing.T, c pagamento.Confirmacao) string {
	t.Helper()
	corpo, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("serializar a confirmação: %v", err)
	}
	return string(corpo)
}

// confirmar entrega ao webhook a confirmação aprovada de uma Tentativa.
func confirmar(t *testing.T, rotas http.Handler, pedidoID string, numero int32) {
	t.Helper()
	c := pagamento.Confirmacao{IDExterno: pagamento.IDExterno(pedidoID, numero), Resultado: pagamento.Aprovado}
	if r := postarWebhook(t, rotas, corpoDe(t, c)); r.Code != http.StatusOK {
		t.Fatalf("webhook = %d (%s), quero 200", r.Code, r.Body.String())
	}
}

// criarTentativa abre uma Tentativa direto no banco: nova Tentativa é da
// Épica 5, e o teste precisa do estado que ela vai produzir.
func criarTentativa(t *testing.T, pool *pgxpool.Pool, pedidoID string, numero int32) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO pagamento.tentativa_pagamento (pedido_id, total_centavos, id_externo, numero)
		SELECT $1::uuid, total_centavos, $2, $3 FROM pedido.pedido WHERE id = $1::uuid`,
		pedidoID, pagamento.IDExterno(pedidoID, numero), numero); err != nil {
		t.Fatalf("criar a Tentativa %d: %v", numero, err)
	}
}

func umPedidoEm(t *testing.T, pool *pgxpool.Pool, status string) string {
	t.Helper()
	ids := textoDe(t, pool, `SELECT id::text FROM pedido.pedido WHERE status = $1 ORDER BY numero LIMIT 1`, status)
	if len(ids) != 1 {
		t.Fatalf("nenhum Pedido em %s; os subtestes anteriores falharam", status)
	}
	return ids[0]
}

func statusDe(t *testing.T, pool *pgxpool.Pool, pedidoID string) string {
	t.Helper()
	linhas := textoDe(t, pool, `SELECT status FROM pedido.pedido WHERE id = $1::uuid`, pedidoID)
	if len(linhas) != 1 {
		t.Fatalf("Pedido %s não encontrado", pedidoID)
	}
	return linhas[0]
}

func reservasDe(t *testing.T, pool *pgxpool.Pool, pedidoID string) []string {
	t.Helper()
	return textoDe(t, pool, `SELECT estado FROM catalogo.reserva_estoque WHERE pedido_id = $1::uuid`, pedidoID)
}

func transicoesDe(t *testing.T, pool *pgxpool.Pool, pedidoID string) []string {
	t.Helper()
	return textoDe(t, pool, `
		SELECT status_anterior || '|' || status_novo || '|' || ator
		FROM pedido.transicao_status WHERE pedido_id = $1::uuid ORDER BY ocorrido_em`, pedidoID)
}

func confirmacoesDe(t *testing.T, pool *pgxpool.Pool, pedidoID string) []string {
	t.Helper()
	return textoDe(t, pool, `
		SELECT c.chave_idempotencia || '|' || c.resultado || '|' || c.estado
		FROM pagamento.confirmacao_recebida c
		JOIN pagamento.tentativa_pagamento t ON t.id = c.tentativa_id
		WHERE t.pedido_id = $1::uuid ORDER BY t.numero`, pedidoID)
}

func contarConfirmacoes(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM pagamento.confirmacao_recebida`).Scan(&n); err != nil {
		t.Fatalf("contar confirmações: %v", err)
	}
	return n
}
