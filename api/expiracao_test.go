package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// prazoLargo é o prazo que TODA chamada de Expirar deste arquivo usa, e é
// largo de propósito. `Expirar` varre o banco inteiro, e `envelhecerHistorico`
// recua só o Pedido do subteste: com um prazo curto, os Pedidos em
// AGUARDANDO_PAGAMENTO dos outros subtestes expirariam junto assim que a suíte
// passasse deles, devolvendo Estoque e derrubando as contas que eles conferem.
// Meia hora é mais que qualquer execução plausível, e quem vence aqui vence
// porque foi envelhecido em uma hora.
const prazoLargo = 30 * time.Minute

// expiracaoDaTentativa é a matriz de servidor da 5.11 contra o banco de
// verdade (FR-34): a Tentativa de Pagamento que passa do prazo sem confirmação
// sai sozinha de AGUARDANDO_PAGAMENTO, com ator VARREDURA e motivo
// TEMPO_ESGOTADO, e o Estoque volta — mais as bordas que dizem que a decisão é
// do histórico, e não de um relógio em memória.
//
// Tudo é próprio: conta, Vendedor, Categoria e Produtos. O total precisa cair
// na faixa `,95`–`,99`, a que nunca confirma: R$ 49,95 mais o Frete do Sudeste
// (reais inteiros) fecha em `,95`.
//
// A ordem dos subtestes importa: eles andam com os mesmos Pedidos.
func expiracaoDaTentativa(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Eva Expiração","email":"eva.expiracao@exemplo.br","senha":"senha-da-eva-1"}`), http.StatusCreated)

	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja da Expiração"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Expiração"}`, admin), http.StatusCreated)
	novoProduto := func(nome string, preco, estoque int) string {
		t.Helper()
		corpo, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": preco, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": estoque, "ativo": true,
		})
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(corpo), admin), http.StatusCreated)
	}
	// `,95`: a faixa em que o Provedor Simulado nunca confirma, e a única razão
	// de a expiração existir. Estoque 1 para o disponível ser observável.
	mudo := novoProduto("Abajur do Silêncio", 4995, 1)

	expirar := func() {
		t.Helper()
		if err := pedido.Expirar(ctx, pool, prazoLargo); err != nil {
			t.Fatalf("expirar: %v", err)
		}
	}
	// envelhecer recua o histórico de UM Pedido uma hora — mais que o
	// `prazoLargo` —, que é o que substitui esperar o prazo correr.
	envelhecer := func(pedidoID string) {
		t.Helper()
		envelhecerHistorico(t, pool, pedidoID, time.Hour)
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
	disponivelDe := func(produto string) float64 {
		t.Helper()
		n, ok := decodificar(t, pegar(t, rotas, "/api/v1/produtos/"+produto))["estoque_disponivel"].(float64)
		if !ok {
			t.Fatalf("estoque_disponivel ausente na Página de Produto %s", produto)
		}
		return n
	}

	pedidoID := idDe(t, pedidoPeloCheckout(t, rotas, comprador, mudo, 1), http.StatusCreated)

	t.Run("dentro do prazo, nada muda", func(t *testing.T) {
		if d := disponivelDe(mudo); d != 0 {
			t.Fatalf("disponível antes da expiração = %v, quero 0 (a Reserva levou a unidade)", d)
		}
		antes := transicoesDe(t, pool, pedidoID)
		expirar()
		if s := statusDe(t, pool, pedidoID); s != "AGUARDANDO_PAGAMENTO" {
			t.Errorf("status = %s com o prazo por vencer; quero AGUARDANDO_PAGAMENTO", s)
		}
		if depois := transicoesDe(t, pool, pedidoID); !slices.Equal(depois, antes) {
			t.Errorf("histórico mudou dentro do prazo:\nantes  %v\ndepois %v", antes, depois)
		}
	})

	t.Run("passado o prazo, a Tentativa expira e o Estoque volta", func(t *testing.T) {
		envelhecer(pedidoID)
		// O instante da varredura é o mesmo que a tela mostra: `expira_em` é a
		// última transição para AGUARDANDO_PAGAMENTO mais o prazo, e é o mesmo
		// `max(ocorrido_em)` que o TravarPedido lê. Envelhecido, já passou.
		venceEm := instanteDoCampo(t, decodificar(t, pegarPedido(t, rotas, pedidoID, comprador)), "expira_em")
		if quer := ultimaTransicao(t, pool, pedidoID).Add(expiracaoDaTentativaDeTeste); !venceEm.Equal(quer) {
			t.Errorf("expira_em = %v, quero %v", venceEm, quer)
		}
		if !venceEm.Before(time.Now()) {
			t.Fatalf("expira_em = %v ainda no futuro; o envelhecimento não pegou", venceEm)
		}

		expirar()
		if s := statusDe(t, pool, pedidoID); s != "PAGAMENTO_RECUSADO" {
			t.Fatalf("status = %s, quero PAGAMENTO_RECUSADO", s)
		}
		// O literal, e não a constante: é ele que a tela procura em
		// `frasesDoMotivo` (web/lib/pedido.ts), e mudar só a constante passaria
		// por este teste e calaria a frase na tela.
		if l := ultimaLinha(pedidoID); l != "AGUARDANDO_PAGAMENTO|PAGAMENTO_RECUSADO|VARREDURA|TEMPO_ESGOTADO" {
			t.Errorf("última transição = %s", l)
		}
		if r := reservasDe(t, pool, pedidoID); !slices.Equal(r, []string{"LIBERADA"}) {
			t.Errorf("reservas = %v; a expiração libera a Reserva", r)
		}
		// Roteiro B, passo 5: a Página de Produto em outra aba.
		if d := disponivelDe(mudo); d != 1 {
			t.Errorf("disponível depois da expiração = %v, quero 1", d)
		}
		if c := confirmacoesDe(t, pool, pedidoID); len(c) != 0 {
			t.Errorf("inbox = %v; a expiração não emite confirmação nenhuma", c)
		}

		// Expirar de novo não faz nada: PAGAMENTO_RECUSADO nunca é candidato.
		expirar()
		if n := len(transicoesDe(t, pool, pedidoID)); n != 2 {
			t.Errorf("histórico com %d linhas depois da segunda expiração, quero 2", n)
		}
	})

	t.Run("a nova Tentativa recomeça o prazo inteiro", func(t *testing.T) {
		resp := comCorpo(t, rotas, http.MethodPost, "/api/v1/pedidos/"+pedidoID+"/tentativas", "", "", comprador)
		if resp.Code != http.StatusCreated {
			t.Fatalf("nova Tentativa = %d (%s), quero 201", resp.Code, resp.Body.String())
		}
		antes := len(transicoesDe(t, pool, pedidoID))

		// O histórico do Pedido continua envelhecido uma hora; o que não está
		// é a linha nova. Se a decisão saísse da criação do Pedido, e não da
		// última entrada em AGUARDANDO_PAGAMENTO, este tique o expiraria já.
		expirar()
		if s := statusDe(t, pool, pedidoID); s != "AGUARDANDO_PAGAMENTO" {
			t.Fatalf("status = %s; a nova Tentativa tem um prazo inteiro pela frente", s)
		}
		if n := len(transicoesDe(t, pool, pedidoID)); n != antes {
			t.Errorf("histórico com %d linhas, quero %d: nada a gravar", n, antes)
		}

		// E passado o prazo novo, expira de novo — pelo mesmo caminho.
		envelhecer(pedidoID)
		expirar()
		if l := ultimaLinha(pedidoID); l != "AGUARDANDO_PAGAMENTO|PAGAMENTO_RECUSADO|VARREDURA|TEMPO_ESGOTADO" {
			t.Errorf("última transição = %s", l)
		}
		if d := disponivelDe(mudo); d != 1 {
			t.Errorf("disponível = %v, quero 1: a Reserva da segunda Tentativa também voltou", d)
		}
	})

	// A ordem normativa do AD-6 é `aplicar` → `expirar`, e é ela que este
	// subteste prende: um total de `,00`, aprovado pelo Provedor, com o prazo
	// já vencido quando o tique corre.
	t.Run("a aprovação que chegou no prazo vence a expiração", func(t *testing.T) {
		// R$ 40,00 mais o Frete do Sudeste fecha em `,00`: a faixa aprovada.
		aprovado := novoProduto("Luminária Pontual", 4000, 1)
		noPrazo := idDe(t, pedidoPeloCheckout(t, rotas, comprador, aprovado, 1), http.StatusCreated)

		// A confirmação é emitida e entra na inbox; só então o prazo é dado
		// por vencido. É a corrida real: o Comprador pagou a tempo e o webhook
		// chegou, mas o tique que aplica é o mesmo em que o relógio zera.
		enviar := func(_ context.Context, c pagamento.Confirmacao) error {
			if c.IDExterno != pagamento.IDExterno(noPrazo, 1) {
				return nil // as Tentativas dos outros subtestes ficam onde estão
			}
			if r := postarWebhook(t, rotas, corpoDe(t, c)); r.Code != http.StatusOK {
				t.Errorf("webhook = %d (%s), quero 200", r.Code, r.Body.String())
			}
			return nil
		}
		if err := pagamento.EmitirConfirmacoesDevidas(ctx, pool, simuladoDeTeste, atrasoDeTeste, janelaDeEmissaoDeTeste, enviar); err != nil {
			t.Fatalf("emitir: %v", err)
		}
		envelhecer(noPrazo)

		// O tique, na ordem do AD-6.
		if err := pedido.Varrer(ctx, pool); err != nil {
			t.Fatalf("varrer: %v", err)
		}
		expirar()

		if s := statusDe(t, pool, noPrazo); s != "PAGO" {
			t.Fatalf("status = %s, quero PAGO: a aprovação chegou primeiro", s)
		}
		if l := ultimaLinha(noPrazo); l != "AGUARDANDO_PAGAMENTO|PAGO|PROVEDOR|" {
			t.Errorf("última transição = %s", l)
		}
		if r := reservasDe(t, pool, noPrazo); !slices.Equal(r, []string{"ATIVA"}) {
			t.Errorf("reservas = %v; PAGO mantém a Reserva", r)
		}

		// E fora de AGUARDANDO_PAGAMENTO o Pedido nunca volta a ser candidato,
		// por mais tempo que passe.
		envelhecer(noPrazo)
		expirar()
		if s := statusDe(t, pool, noPrazo); s != "PAGO" {
			t.Errorf("status = %s; PAGO não expira", s)
		}
	})

	// O que a ordem do tique NÃO cobre, e a inbox sob a trava cobre: a
	// aprovação que `aplicar` não aplicou — porque `Varrer` falhou, porque o
	// SKIP LOCKED pulou a linha, ou porque a confirmação chegou entre os dois
	// passos. Aqui o `Varrer` simplesmente não roda, que é o mesmo estado do
	// banco nos três casos.
	t.Run("a aprovação pendente que aplicar não aplicou segura a expiração", func(t *testing.T) {
		tardio := novoProduto("Rádio do Atraso", 4000, 1)
		alvo := idDe(t, pedidoPeloCheckout(t, rotas, comprador, tardio, 1), http.StatusCreated)
		enviar := func(_ context.Context, c pagamento.Confirmacao) error {
			if c.IDExterno != pagamento.IDExterno(alvo, 1) {
				return nil
			}
			if r := postarWebhook(t, rotas, corpoDe(t, c)); r.Code != http.StatusOK {
				t.Errorf("webhook = %d (%s), quero 200", r.Code, r.Body.String())
			}
			return nil
		}
		if err := pagamento.EmitirConfirmacoesDevidas(ctx, pool, simuladoDeTeste, atrasoDeTeste, janelaDeEmissaoDeTeste, enviar); err != nil {
			t.Fatalf("emitir: %v", err)
		}
		chave := pagamento.ChaveIdempotencia(pagamento.IDExterno(alvo, 1))
		if tem := confirmacoesDe(t, pool, alvo); !slices.Equal(tem, []string{chave + "|APROVADO|PENDENTE"}) {
			t.Fatalf("inbox = %v; quero a aprovação pendente", tem)
		}
		envelhecer(alvo)

		// A expiração sozinha, sem `aplicar` antes: com a linha do Pedido na
		// mão ela vê a confirmação por aplicar e sai calada. Sem a conferência,
		// o Comprador que pagou no prazo perderia a Reserva aqui.
		antes := transicoesDe(t, pool, alvo)
		expirar()
		if s := statusDe(t, pool, alvo); s != "AGUARDANDO_PAGAMENTO" {
			t.Fatalf("status = %s; a confirmação por aplicar segura a expiração", s)
		}
		if depois := transicoesDe(t, pool, alvo); !slices.Equal(depois, antes) {
			t.Errorf("histórico mudou:\nantes  %v\ndepois %v", antes, depois)
		}

		// O tique seguinte aplica, e aí sim o Pedido sai — para PAGO.
		if err := pedido.Varrer(ctx, pool); err != nil {
			t.Fatalf("varrer: %v", err)
		}
		if s := statusDe(t, pool, alvo); s != "PAGO" {
			t.Errorf("status = %s depois de aplicar; quero PAGO", s)
		}
	})

	// A aprovação que chega depois de uma expiração legítima: o Status não
	// muda e a linha da inbox é o registro que a 6.5 lê, mas isso não pode ser
	// perda silenciosa — é dinheiro aprovado sobre Reserva já devolvida.
	t.Run("a aprovação sobre Tentativa expirada é sinalizada e avisada", func(t *testing.T) {
		expirado := novoProduto("Ampulheta Vazia", 4000, 1)
		alvo := idDe(t, pedidoPeloCheckout(t, rotas, comprador, expirado, 1), http.StatusCreated)
		envelhecer(alvo)
		expirar()
		if l := ultimaLinha(alvo); l != "AGUARDANDO_PAGAMENTO|PAGAMENTO_RECUSADO|VARREDURA|TEMPO_ESGOTADO" {
			t.Fatalf("última transição = %s; o Pedido tinha de estar expirado", l)
		}
		antes := transicoesDe(t, pool, alvo)

		// A aprovação chega tarde, pelo webhook, como um Provedor real faria.
		c := pagamento.Confirmacao{IDExterno: pagamento.IDExterno(alvo, 1), Resultado: pagamento.Aprovado}
		if r := postarWebhook(t, rotas, corpoDe(t, c)); r.Code != http.StatusOK {
			t.Fatalf("webhook = %d (%s), quero 200", r.Code, r.Body.String())
		}

		// O aviso é a única saída em tempo real deste caso: o estado terminal
		// da inbox seria o mesmo sem ele.
		var registro bytes.Buffer
		anterior := slog.Default()
		slog.SetDefault(plataforma.NovoLogger(&registro, "varredura"))
		err := pedido.Varrer(ctx, pool)
		slog.SetDefault(anterior)
		if err != nil {
			t.Fatalf("varrer: %v", err)
		}
		if aviso := registro.String(); !strings.Contains(aviso, "pagamento aprovado sobre Tentativa de Pagamento expirada") ||
			!strings.Contains(aviso, alvo) {
			t.Errorf("log da varredura = %q; quero o aviso nomeando o Pedido", aviso)
		}
		if s := statusDe(t, pool, alvo); s != "PAGAMENTO_RECUSADO" {
			t.Errorf("status = %s; a aprovação tardia não desfaz a expiração", s)
		}
		chave := pagamento.ChaveIdempotencia(pagamento.IDExterno(alvo, 1))
		if tem := confirmacoesDe(t, pool, alvo); !slices.Equal(tem, []string{chave + "|APROVADO|NAO_APLICAVEL_SINALIZADA"}) {
			t.Errorf("inbox = %v; quero a aprovação sinalizada", tem)
		}
		if depois := transicoesDe(t, pool, alvo); !slices.Equal(depois, antes) {
			t.Errorf("histórico mudou:\nantes  %v\ndepois %v", antes, depois)
		}
		if d := disponivelDe(expirado); d != 1 {
			t.Errorf("disponível = %v, quero 1: a Reserva continua liberada", d)
		}
	})

	// O `FOR UPDATE SKIP LOCKED` do TravarPedido: o tique que encontra o Pedido
	// já travado não espera nem falha — pula, e o tique seguinte o encontra.
	// Sem isto, trocar o SKIP LOCKED por uma espera deixaria a suíte verde, e a
	// varredura inteira passaria a serializar num Pedido preso. Esse desfecho
	// aparece como travamento, e não como falha rápida: a trava é desta
	// goroutine, então `expirar()` esperaria por si mesmo até o prazo do
	// `go test`. Quem vir a suíte pendurar aqui procure o SKIP LOCKED.
	t.Run("o Pedido travado é pulado, e o tique seguinte o encontra", func(t *testing.T) {
		travavel := novoProduto("Cofre Trancado", 4995, 1)
		preso := idDe(t, pedidoPeloCheckout(t, rotas, comprador, travavel, 1), http.StatusCreated)
		envelhecer(preso)

		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação: %v", err)
		}
		if _, err := tx.Exec(ctx, `SELECT id FROM pedido.pedido WHERE id = $1::uuid FOR UPDATE`, preso); err != nil {
			tx.Rollback(ctx)
			t.Fatalf("travar a linha do Pedido: %v", err)
		}
		// Com a linha presa, o passo do tempo tem de devolver nil: o Pedido
		// vencido está ali e é candidato, mas a trava é de outro.
		antes := transicoesDe(t, pool, preso)
		expirar()
		if s := statusDe(t, pool, preso); s != "AGUARDANDO_PAGAMENTO" {
			t.Errorf("status = %s com a linha travada; quero AGUARDANDO_PAGAMENTO", s)
		}
		if depois := transicoesDe(t, pool, preso); !slices.Equal(depois, antes) {
			t.Errorf("histórico mudou com a linha travada:\nantes  %v\ndepois %v", antes, depois)
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Fatalf("soltar a trava: %v", err)
		}

		// Solta a trava, o tique seguinte o encontra — e não o perde.
		expirar()
		if s := statusDe(t, pool, preso); s != "PAGAMENTO_RECUSADO" {
			t.Fatalf("status = %s depois de soltar a trava; quero PAGAMENTO_RECUSADO", s)
		}
		if l := ultimaLinha(preso); l != "AGUARDANDO_PAGAMENTO|PAGAMENTO_RECUSADO|VARREDURA|TEMPO_ESGOTADO" {
			t.Errorf("última transição = %s", l)
		}
		if d := disponivelDe(travavel); d != 1 {
			t.Errorf("disponível = %v, quero 1: a Reserva do Pedido pulado também voltou", d)
		}
	})

	// A outra metade da janela (AD-7): passado o prazo, a Tentativa está morta
	// e emitir sobre ela não teria efeito, então ela sai da lista. Sem o piso,
	// a faixa que nunca confirma seria relida a cada tique para sempre.
	t.Run("a Tentativa passada do prazo sai da lista de emissão", func(t *testing.T) {
		// `,90`: a faixa recusada, que emite de verdade — com `,95` a Tentativa
		// sairia da lista sem emitir nada, e o teste não distinguiria as duas
		// causas. O transporte só anota: a inbox fica como está.
		recusado := novoProduto("Espelho Atrasado", 4990, 1)
		alvo := idDe(t, pedidoPeloCheckout(t, rotas, comprador, recusado, 1), http.StatusCreated)
		esperado := pagamento.IDExterno(alvo, 1)

		emitidas := map[string]string{}
		enviar := func(_ context.Context, c pagamento.Confirmacao) error {
			emitidas[c.IDExterno] = c.Resultado
			return nil
		}
		emitirCom := func(prazo time.Duration) {
			t.Helper()
			clear(emitidas)
			if err := pagamento.EmitirConfirmacoesDevidas(ctx, pool, simuladoDeTeste, atrasoDeTeste, prazo, enviar); err != nil {
				t.Fatalf("emitir: %v", err)
			}
		}

		emitirCom(janelaDeEmissaoDeTeste)
		if r := emitidas[esperado]; r != pagamento.Recusado {
			t.Fatalf("dentro da janela a Tentativa emitiu %q; quero %s", r, pagamento.Recusado)
		}

		// Duas horas para trás, contra uma janela de uma: a Tentativa passa a
		// ser mais velha que o prazo. `criada_em` não tem gatilho de
		// imutabilidade — quem o tem é o histórico do Pedido —, então o UPDATE
		// direto basta.
		if _, err := pool.Exec(ctx, `
			UPDATE pagamento.tentativa_pagamento SET criada_em = criada_em - interval '2 hours'
			WHERE pedido_id = $1::uuid`, alvo); err != nil {
			t.Fatalf("envelhecer a Tentativa: %v", err)
		}
		emitirCom(janelaDeEmissaoDeTeste)
		if r, tem := emitidas[esperado]; tem {
			t.Errorf("passada do prazo a Tentativa emitiu %s; quero que saia da lista", r)
		}
	})
}
