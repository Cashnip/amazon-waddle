package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
)

// chaveDoSinal é o nome do campo da FR-26 nas duas respostas administrativas.
const chaveDoSinal = "pagamento_aprovado_sobre_cancelado"

// aprovadoSobreCancelado é a matriz de servidor da 6.5 contra o banco de
// verdade (FR-26, AD-7, AD-18): o sinal de pagamento aprovado sobre Pedido
// cancelado, derivado na leitura, nas duas respostas administrativas — o
// Detalhe e a linha da Tabela filtrada pelo Status. O predicado é exatamente
// "CANCELADO e uma confirmação APROVADO em NAO_APLICAVEL_SINALIZADA numa
// Tentativa dele", e cada subteste prende uma metade: a recusa sinalizada
// morre se `resultado = 'APROVADO'` sair da consulta, e o PAGO sinalizado
// morre se a condição CANCELADO sair do Go.
//
// Tudo é próprio: conta, Vendedor, Categoria e Produtos. O Produto de `,00`
// fecha o total em `,00` com o Frete (reais inteiros), a faixa que o Provedor
// Simulado aprova; o de `,90`, a que ele recusa na primeira Tentativa. Roda
// depois do painel da 6.4, cuja Tabela sem filtro confere que os Pedidos dele
// são os mais recentes da loja.
//
// A ordem importa só onde um subteste anda com o Pedido do anterior, e isso
// está dito onde acontece.
func aprovadoSobreCancelado(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Estela Estorno","email":"estela.estorno@exemplo.br","senha":"senha-da-estela-1"}`), http.StatusCreated)

	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja do Estorno"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Estorno"}`, admin), http.StatusCreated)
	novoProduto := func(nome string, preco, estoque int) string {
		t.Helper()
		corpo, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": preco, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": estoque, "ativo": true,
		})
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(corpo), admin), http.StatusCreated)
	}
	aprovada := novoProduto("Poltrona do Estorno", 5000, 20)
	recusada := novoProduto("Estante da Recusa Tardia", 4990, 5)

	// O tique do Provedor Simulado com o transporte de verdade, entregando só
	// as Tentativas destes Pedidos — as dos outros subtestes ficam onde estão.
	emitirPara := func(t *testing.T, pedidos ...string) {
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
	varrer := func(t *testing.T) {
		t.Helper()
		if err := pedido.Varrer(ctx, pool); err != nil {
			t.Fatalf("varrer: %v", err)
		}
	}
	aguardando := func(t *testing.T, produto string) string {
		t.Helper()
		return idDe(t, pedidoPeloCheckout(t, rotas, comprador, produto, 1), http.StatusCreated)
	}
	// pago leva o Pedido a PAGO pelo caminho de produção: a aprovação do
	// Provedor aplicada pela varredura, e nada de UPDATE à mão.
	pago := func(t *testing.T) string {
		t.Helper()
		id := aguardando(t, aprovada)
		emitirPara(t, id)
		varrer(t)
		if s := statusDe(t, pool, id); s != string(pedido.StatusPago) {
			t.Fatalf("status = %s; o Pedido de `,00` tinha de ir a PAGO", s)
		}
		return id
	}
	cancelar := func(t *testing.T, id string) {
		t.Helper()
		if resp := postarCancelamento(t, rotas, id, comprador); resp.Code != http.StatusOK {
			t.Fatalf("cancelar = %d (%s), quero 200", resp.Code, resp.Body.String())
		}
	}
	// sinalDe lê o campo exigindo que ele exista e seja booleano: sem o `ok`,
	// a chave ausente leria `false` e todo caso negativo passaria sem conferir.
	sinalDe := func(t *testing.T, onde string, corpo map[string]any) bool {
		t.Helper()
		v, ok := corpo[chaveDoSinal].(bool)
		if !ok {
			t.Fatalf("%s: `%s` ausente ou não booleano: %v", onde, chaveDoSinal, corpo)
		}
		return v
	}
	// linhaDaTabela é a linha do Pedido na Tabela filtrada pelo Status dele —
	// como o Administrador a vê em `/admin/pedidos?status=…`.
	linhaDaTabela := func(t *testing.T, status pedido.Status, id string) map[string]any {
		t.Helper()
		resp := pegarCom(t, rotas, "/api/v1/admin/pedidos?status="+string(status)+"&por_pagina=100", admin)
		if resp.Code != http.StatusOK {
			t.Fatalf("listar %s = %d (%s), quero 200", status, resp.Code, resp.Body.String())
		}
		cru, ok := decodificar(t, resp)["itens"].([]any)
		if !ok {
			t.Fatalf("a listagem não trouxe `itens`")
		}
		for _, i := range cru {
			linha, ok := i.(map[string]any)
			if !ok {
				t.Fatalf("linha da Tabela não é objeto: %v", i)
			}
			if linha["id"] == id {
				return linha
			}
		}
		t.Fatalf("o Pedido %s não está na primeira página da Tabela filtrada por %s", id, status)
		return nil
	}
	// confere é a matriz numa chamada: o Detalhe e a linha da Tabela dizem o
	// mesmo Status e o mesmo sinal.
	confere := func(t *testing.T, id string, status pedido.Status, quer bool) {
		t.Helper()
		resp := pegarPedidoAdmin(t, rotas, id, admin)
		if resp.Code != http.StatusOK {
			t.Fatalf("Detalhe = %d (%s), quero 200", resp.Code, resp.Body.String())
		}
		detalhe := decodificar(t, resp)
		if detalhe["status"] != string(status) {
			t.Errorf("Detalhe: status = %v, quero %s", detalhe["status"], status)
		}
		if got := sinalDe(t, "Detalhe", detalhe); got != quer {
			t.Errorf("Detalhe: %s = %v, quero %v", chaveDoSinal, got, quer)
		}
		linha := linhaDaTabela(t, status, id)
		if got := sinalDe(t, "Tabela", linha); got != quer {
			t.Errorf("Tabela: %s = %v, quero %v", chaveDoSinal, got, quer)
		}
	}
	chaveDa := func(id string, numero int32) string {
		return pagamento.ChaveIdempotencia(pagamento.IDExterno(id, numero))
	}

	var corrida string
	t.Run("a corrida clássica: a aprovação que chega depois do cancelamento acende o sinal", func(t *testing.T) {
		corrida = aguardando(t, aprovada)
		cancelar(t, corrida)
		// O Provedor aprova assim mesmo: `pagamento` não conhece `pedido`.
		emitirPara(t, corrida)
		varrer(t)
		if tem := confirmacoesDe(t, pool, corrida); !slices.Equal(tem, []string{chaveDa(corrida, 1) + "|APROVADO|NAO_APLICAVEL_SINALIZADA"}) {
			t.Fatalf("inbox = %v; quero a aprovação sinalizada", tem)
		}

		confere(t, corrida, pedido.StatusCancelado, true)

		// A leitura não escreve: o Status, o histórico e a Reserva ficam como
		// o cancelamento os deixou.
		if s := statusDe(t, pool, corrida); s != string(pedido.StatusCancelado) {
			t.Errorf("status = %s; o sinal não move o Pedido", s)
		}
		quer := []string{"|AGUARDANDO_PAGAMENTO|COMPRADOR", "AGUARDANDO_PAGAMENTO|CANCELADO|COMPRADOR"}
		if h := transicoesDe(t, pool, corrida); !slices.Equal(h, quer) {
			t.Errorf("histórico = %v, quero %v", h, quer)
		}
		if r := reservasDe(t, pool, corrida); !slices.Equal(r, []string{"LIBERADA"}) {
			t.Errorf("reservas = %v; a aprovação sobre cancelado não reserva nada", r)
		}
	})

	t.Run("a resposta do Comprador não carrega o sinal", func(t *testing.T) {
		if corrida == "" {
			t.Skip("a corrida clássica não chegou a criar o Pedido")
		}
		detalhe := pegarCom(t, rotas, "/api/v1/pedidos/"+corrida, comprador)
		if detalhe.Code != http.StatusOK {
			t.Fatalf("Detalhe do Comprador = %d (%s), quero 200", detalhe.Code, detalhe.Body.String())
		}
		if slices.Contains(chavesDe(decodificar(t, detalhe)), chaveDoSinal) {
			t.Errorf("o Detalhe do Comprador carrega `%s`", chaveDoSinal)
		}
		lista := pegarCom(t, rotas, "/api/v1/pedidos?por_pagina=100", comprador)
		if lista.Code != http.StatusOK {
			t.Fatalf("Meus pedidos = %d (%s), quero 200", lista.Code, lista.Body.String())
		}
		corpo := decodificar(t, lista)
		if cru, ok := corpo["itens"].([]any); !ok || len(cru) == 0 {
			t.Fatalf("Meus pedidos sem `itens`: %v", corpo)
		}
		if slices.Contains(chavesDe(corpo), chaveDoSinal) {
			t.Errorf("Meus pedidos carrega `%s`", chaveDoSinal)
		}
	})

	var semSinal string
	t.Run("cancelado depois de PAGO, com a aprovação aplicada, não acende", func(t *testing.T) {
		semSinal = pago(t)
		cancelar(t, semSinal)
		if tem := confirmacoesDe(t, pool, semSinal); !slices.Equal(tem, []string{chaveDa(semSinal, 1) + "|APROVADO|APLICADA"}) {
			t.Fatalf("inbox = %v; quero só a aprovação aplicada", tem)
		}
		confere(t, semSinal, pedido.StatusCancelado, false)
	})

	var recusa string
	t.Run("a recusa sinalizada não é dinheiro aprovado", func(t *testing.T) {
		recusa = aguardando(t, recusada)
		cancelar(t, recusa)
		emitirPara(t, recusa)
		varrer(t)
		if tem := confirmacoesDe(t, pool, recusa); !slices.Equal(tem, []string{chaveDa(recusa, 1) + "|RECUSADO|NAO_APLICAVEL_SINALIZADA"}) {
			t.Fatalf("inbox = %v; quero a recusa sinalizada", tem)
		}
		confere(t, recusa, pedido.StatusCancelado, false)
	})

	// Os dois próximos andam com o mesmo Pedido: a segunda aprovação é
	// sinalizada com ele em PAGO, e só o cancelamento a transforma no sinal.
	var duplicada string
	t.Run("a segunda aprovação sinalizada sem cancelamento não acende", func(t *testing.T) {
		duplicada = pago(t)
		// Uma segunda Tentativa aprovada sobre o Pedido já PAGO: não é
		// alcançável pela tela, então a Tentativa nasce no banco e a
		// confirmação entra pelo webhook de verdade.
		criarTentativa(t, pool, duplicada, 2)
		confirmar(t, rotas, duplicada, 2)
		varrer(t)
		quer := []string{
			chaveDa(duplicada, 1) + "|APROVADO|APLICADA",
			chaveDa(duplicada, 2) + "|APROVADO|NAO_APLICAVEL_SINALIZADA",
		}
		if tem := confirmacoesDe(t, pool, duplicada); !slices.Equal(tem, quer) {
			t.Fatalf("inbox = %v, quero %v", tem, quer)
		}
		confere(t, duplicada, pedido.StatusPago, false)
	})

	t.Run("cobrança duplicada: o mesmo Pedido, cancelado, acende", func(t *testing.T) {
		if duplicada == "" {
			t.Skip("o subteste anterior não chegou a criar o Pedido")
		}
		cancelar(t, duplicada)
		confere(t, duplicada, pedido.StatusCancelado, true)
		quer := []string{
			"|AGUARDANDO_PAGAMENTO|COMPRADOR",
			"AGUARDANDO_PAGAMENTO|PAGO|PROVEDOR",
			"PAGO|CANCELADO|COMPRADOR",
		}
		if h := transicoesDe(t, pool, duplicada); !slices.Equal(h, quer) {
			t.Errorf("histórico = %v, quero %v", h, quer)
		}
	})

	t.Run("fora de CANCELADO o campo existe e é false, também no 200 da transição", func(t *testing.T) {
		id := pago(t)
		resp := postarTransicao(t, rotas, id, string(pedido.StatusPago), string(pedido.StatusSeparando), admin)
		if resp.Code != http.StatusOK {
			t.Fatalf("transição = %d (%s), quero 200", resp.Code, resp.Body.String())
		}
		if sinalDe(t, "transição", decodificar(t, resp)) {
			t.Errorf("o 200 da transição acendeu o sinal")
		}
		confere(t, id, pedido.StatusSeparando, false)
	})

	// A AC da Tabela: filtrada por CANCELADO, com linhas com e sem o sinal,
	// só as com o sinal acendem — numa listagem só, como a tela a lê.
	t.Run("a Tabela filtrada por CANCELADO acende só as linhas com o sinal", func(t *testing.T) {
		resp := pegarCom(t, rotas, "/api/v1/admin/pedidos?status=CANCELADO&por_pagina=100", admin)
		if resp.Code != http.StatusOK {
			t.Fatalf("listar = %d (%s), quero 200", resp.Code, resp.Body.String())
		}
		cru, ok := decodificar(t, resp)["itens"].([]any)
		if !ok {
			t.Fatalf("a listagem não trouxe `itens`")
		}
		// Um subteste anterior que falhou antes de criar o seu Pedido deixaria
		// o id vazio, e as chaves vazias colidiriam no mapa: a contagem abaixo
		// acusaria um erro em cascata que não é desta asserção.
		if corrida == "" || semSinal == "" || recusa == "" || duplicada == "" {
			t.Skip("um subteste anterior não chegou a criar o seu Pedido")
		}
		meus := map[string]bool{corrida: true, semSinal: false, recusa: false, duplicada: true}
		vistos := 0
		for _, i := range cru {
			linha, ok := i.(map[string]any)
			if !ok {
				t.Fatalf("linha da Tabela não é objeto: %v", i)
			}
			quer, meu := meus[fmt.Sprint(linha["id"])]
			if !meu {
				continue
			}
			vistos++
			if got := sinalDe(t, "Tabela", linha); got != quer {
				t.Errorf("Pedido %v: %s = %v, quero %v", linha["numero"], chaveDoSinal, got, quer)
			}
		}
		if vistos != len(meus) {
			t.Errorf("vi %d dos %d Pedidos cancelados desta matriz na Tabela", vistos, len(meus))
		}
	})
}
