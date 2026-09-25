package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
)

// postarTransicao é o botão do painel: POST com `de` e `para` no corpo. O `de`
// é o que a TELA leu, e é isso que o teste da corrida explora.
func postarTransicao(t *testing.T, rotas http.Handler, pedidoID, de, para string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	corpo := fmt.Sprintf(`{"de":%q,"para":%q}`, de, para)
	return comCorpo(t, rotas, http.MethodPost,
		"/api/v1/admin/pedidos/"+pedidoID+"/transicoes", corpo, "application/json", cookie)
}

func pegarPedidoAdmin(t *testing.T, rotas http.Handler, pedidoID string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return pegarCom(t, rotas, "/api/v1/admin/pedidos/"+pedidoID, cookie)
}

// painelDePedidosDoAdministrador é a matriz de servidor da 6.4 contra o banco
// de verdade (FR-32): a Tabela sem filtro e com filtro, a ordenação pelas duas
// datas, o Detalhe com Comprador e Vendedores congelados, as três transições
// andando com ator ADMINISTRADOR, a consolidação do Estoque no ENVIADO, a
// transição fora da tabela, o cancelamento recusado ao Administrador e a
// corrida contra a simulação.
//
// Tudo é próprio: contas, Vendedor, Categoria e Produtos. O Produto de `,00`
// fecha o total em `,00` com o Frete (reais inteiros), a faixa que o Provedor
// Simulado aprova — é assim que o Pedido chega a PAGO pelo caminho de
// produção, que é de onde as três transições partem.
//
// A Tabela sem filtro é global, e o que ela precisa é que os Pedidos desta
// estória sejam os mais recentes: são, em qualquer posição da suíte, porque
// nascem aqui, logo antes da leitura.
func painelDePedidosDoAdministrador(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Painel Comprador","email":"painel.comprador@exemplo.br","senha":"senha-do-painel-1"}`), http.StatusCreated)
	segundo := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Beatriz Balcão","email":"beatriz.balcao@exemplo.br","senha":"senha-da-beatriz-1"}`), http.StatusCreated)

	const nomeDaLoja = "Loja do Painel"
	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"`+nomeDaLoja+`"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Painel"}`, admin), http.StatusCreated)
	novoProduto := func(nome string, preco, estoque int) string {
		t.Helper()
		corpo, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": preco, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": estoque, "ativo": true,
		})
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(corpo), admin), http.StatusCreated)
	}
	const nomeDoProduto = "Escrivaninha do Painel"
	escrivaninha := novoProduto(nomeDoProduto, 5000, 20)

	// O tique do Provedor Simulado com o transporte de verdade, entregando só
	// as Tentativas destes Pedidos — as dos outros subtestes ficam onde estão.
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
	// pago cria um Pedido de uma Escrivaninha e o leva a PAGO pelo caminho de
	// produção: a aprovação do Provedor, e nada de UPDATE à mão.
	pago := func(cookie *http.Cookie) string {
		t.Helper()
		id := idDe(t, pedidoPeloCheckout(t, rotas, cookie, escrivaninha, 1), http.StatusCreated)
		emitirPara(id)
		if err := pedido.Varrer(ctx, pool); err != nil {
			t.Fatalf("varrer: %v", err)
		}
		if s := statusDe(t, pool, id); s != string(pedido.StatusPago) {
			t.Fatalf("status = %s; o Pedido de `,00` tinha de ir a PAGO", s)
		}
		return id
	}
	// simular é o outro ator das três mesmas linhas da tabela: é com ele que a
	// corrida da FR-32 é montada, pelo caminho de produção.
	simular := func(pedidoID string, de, para pedido.Status) {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação: %v", err)
		}
		defer tx.Rollback(ctx)
		if err := pedido.Transicionar(ctx, tx, pedidoID, de, para, pedido.AtorSimulacao, ""); err != nil {
			t.Fatalf("a simulação avançar de %s a %s: %v", de, para, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit da simulação: %v", err)
		}
	}
	listar := func(consulta string) map[string]any {
		t.Helper()
		resp := pegarCom(t, rotas, "/api/v1/admin/pedidos"+consulta, admin)
		if resp.Code != http.StatusOK {
			t.Fatalf("listar%s = %d (%s), quero 200", consulta, resp.Code, resp.Body.String())
		}
		return decodificar(t, resp)
	}
	itensDe := func(lista map[string]any) []map[string]any {
		t.Helper()
		cru, ok := lista["itens"].([]any)
		if !ok {
			t.Fatalf("a listagem não trouxe `itens`: %v", lista)
		}
		itens := make([]map[string]any, 0, len(cru))
		for _, i := range cru {
			itens = append(itens, i.(map[string]any))
		}
		return itens
	}
	contarNoBanco := func(sql string, args ...any) int64 {
		t.Helper()
		var n int64
		if err := pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
			t.Fatalf("contar: %v", err)
		}
		return n
	}
	estoqueTotalDe := func(produto string) int64 {
		t.Helper()
		return contarNoBanco(`SELECT estoque_total FROM catalogo.produto WHERE id = $1::uuid`, produto)
	}
	// fotografia é tudo que uma transição poderia gravar: Status, histórico,
	// Reservas com o Estoque total do Produto. A recusa tem de deixá-la
	// idêntica — 409 sem gravar nada não se prova pelo status da resposta.
	fotografia := func(pedidoID string) []string {
		t.Helper()
		f := []string{"status=" + statusDe(t, pool, pedidoID)}
		f = append(f, transicoesDe(t, pool, pedidoID)...)
		return append(f, textoDe(t, pool, `
			SELECT r.produto_id::text || '|' || r.quantidade || '|' || r.estado || '|' || p.estoque_total
			FROM catalogo.reserva_estoque r JOIN catalogo.produto p ON p.id = r.produto_id
			WHERE r.pedido_id = $1::uuid ORDER BY r.id`, pedidoID)...)
	}
	// permitidasDaResposta lê a lista que a tela usa para montar os botões.
	// Sem o `ok`, um campo renomeado leria a fatia vazia e toda asserção de
	// "nenhum botão" passaria sem conferir coisa alguma.
	permitidasDaResposta := func(corpo map[string]any) []string {
		t.Helper()
		cru, ok := corpo["permitidas"].([]any)
		if !ok {
			t.Fatalf("`permitidas` ausente na resposta: %v", corpo)
		}
		destinos := make([]string, 0, len(cru))
		for _, d := range cru {
			destinos = append(destinos, fmt.Sprint(d))
		}
		return destinos
	}

	// Dois Compradores, para a Tabela provar que ela não tem dono no WHERE.
	daBeatriz := pago(segundo)
	doPainel := pago(comprador)

	t.Run("a Tabela lista Pedido de todo Comprador, com o total real e permitidas pronto", func(t *testing.T) {
		lista := listar("?por_pagina=100")
		if total := int64(lista["total"].(float64)); total != contarNoBanco(`SELECT count(*) FROM pedido.pedido`) {
			t.Errorf("total = %d; quero o número de Pedidos do banco", total)
		}
		itens := itensDe(lista)
		vistos := map[string]map[string]any{}
		for _, i := range itens {
			vistos[fmt.Sprint(i["id"])] = i
		}
		// Os dois Compradores na mesma lista: é o que distingue esta rota da
		// listagem do Comprador, cujo dono entra no WHERE.
		for _, id := range []string{daBeatriz, doPainel} {
			if vistos[id] == nil {
				t.Fatalf("o Pedido %s não está na primeira página da Tabela", id)
			}
		}
		// Mais recentes primeiro, sem filtro: os desta estória são os últimos
		// criados, e o da Beatriz nasceu antes do do Painel.
		if i, j := slices.IndexFunc(itens, func(m map[string]any) bool { return m["id"] == doPainel }),
			slices.IndexFunc(itens, func(m map[string]any) bool { return m["id"] == daBeatriz }); i > j {
			t.Errorf("o Pedido mais novo apareceu depois do mais velho (%d > %d) sem filtro", i, j)
		}
		// A linha carrega o que a tela deriva, numa chamada só (AD-18).
		linha := vistos[doPainel]
		if linha["status"] != "PAGO" || linha["terminal"] != false {
			t.Errorf("linha = %v; quero PAGO e terminal falso", linha)
		}
		if p := permitidasDaResposta(linha); !slices.Equal(p, []string{"SEPARANDO"}) {
			t.Errorf("permitidas de PAGO = %v, quero [SEPARANDO]", p)
		}
		if linha["criado_em"] == nil {
			t.Errorf("a linha da Tabela veio sem data: %v", linha)
		}
	})

	t.Run("o filtro e a ordenação vivem na URL, e o que não é da lista é 400", func(t *testing.T) {
		lista := listar("?status=PAGO&ordenacao=antigos&por_pagina=100")
		// A contagem repete o mesmo WHERE da lista.
		if total := int64(lista["total"].(float64)); total != contarNoBanco(`SELECT count(*) FROM pedido.pedido WHERE status = 'PAGO'`) {
			t.Errorf("total filtrado = %d; quero o número de PAGO do banco", total)
		}
		itens := itensDe(lista)
		if len(itens) == 0 {
			t.Fatal("o filtro por PAGO não trouxe nenhum Pedido; os dois criados acima estão em PAGO")
		}
		anterior := ""
		for _, i := range itens {
			if i["status"] != "PAGO" {
				t.Fatalf("o filtro deixou passar %v", i["status"])
			}
			// Do mais antigo ao mais novo: a data nunca anda para trás.
			if data := fmt.Sprint(i["criado_em"]); anterior != "" && data < anterior {
				t.Errorf("ordenação `antigos` fora de ordem: %s veio depois de %s", data, anterior)
			} else {
				anterior = data
			}
		}

		// Os SETE valem, um a um, nas duas ordenações: é a lista que o Select
		// da tela oferece, tirada do mesmo mapa de rótulos. Sem este laço,
		// tirar um Status de `pedido.Statuses` passaria calado aqui enquanto a
		// tela levaria 400 ao escolhê-lo.
		for _, s := range pedido.Statuses {
			for _, ordem := range []string{pedido.OrdenacaoRecentes, pedido.OrdenacaoAntigos} {
				consulta := "?status=" + string(s) + "&ordenacao=" + ordem
				lista := listar(consulta)
				quer := contarNoBanco(`SELECT count(*) FROM pedido.pedido WHERE status = $1`, string(s))
				if total := int64(lista["total"].(float64)); total != quer {
					t.Errorf("%s: total = %d, quero %d", consulta, total, quer)
				}
				for _, i := range itensDe(lista) {
					if i["status"] != string(s) {
						t.Fatalf("%s: o filtro deixou passar %v", consulta, i["status"])
					}
				}
			}
		}

		for _, caso := range []struct{ nome, consulta, campo string }{
			{"Status fora dos sete", "?status=SEPARANDO_TALVEZ", "status"},
			{"ordenação da loja", "?ordenacao=preco_asc", "ordenacao"},
		} {
			resp := pegarCom(t, rotas, "/api/v1/admin/pedidos"+caso.consulta, admin)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("%s: status = %d (%s), quero 400", caso.nome, resp.Code, resp.Body.String())
			}
			corpo := decodificar(t, resp)
			if corpo["codigo"] != "CAMPO_INVALIDO" {
				t.Errorf("%s: codigo = %v, quero CAMPO_INVALIDO", caso.nome, corpo["codigo"])
			}
			if dados, _ := corpo["dados"].(map[string]any); dados["campo"] != caso.campo {
				t.Errorf("%s: dados = %v, quero o campo %q", caso.nome, corpo["dados"], caso.campo)
			}
		}
	})

	t.Run("o Detalhe traz o Comprador, o Vendedor congelado e nada da decisão do Comprador", func(t *testing.T) {
		resp := pegarPedidoAdmin(t, rotas, doPainel, admin)
		if resp.Code != http.StatusOK {
			t.Fatalf("Detalhe = %d (%s), quero 200", resp.Code, resp.Body.String())
		}
		d := decodificar(t, resp)

		quem, _ := d["comprador"].(map[string]any)
		if quem["nome"] != "Painel Comprador" || quem["email"] != "painel.comprador@exemplo.br" {
			t.Errorf("comprador = %v; quero o nome e o e-mail de quem comprou", d["comprador"])
		}
		itens, _ := d["itens"].([]any)
		if len(itens) != 1 {
			t.Fatalf("itens = %v, quero um", d["itens"])
		}
		item := itens[0].(map[string]any)
		if item["nome"] != nomeDoProduto || item["vendedor_nome"] != nomeDaLoja {
			t.Errorf("item = %v; quero o Produto e o Vendedor congelados", item)
		}
		if item["preco_praticado_centavos"] != 5000.0 {
			t.Errorf("preço praticado = %v, quero 5000", item["preco_praticado_centavos"])
		}
		if d["subtotal_centavos"] != 5000.0 || d["endereco"] == nil {
			t.Errorf("valores/Endereço = %v / %v; quero os congelados", d["subtotal_centavos"], d["endereco"])
		}
		if h, _ := d["historico"].([]any); len(h) < 2 {
			t.Errorf("histórico = %v; quero ao menos o nascimento e o pagamento", d["historico"])
		}
		if p := permitidasDaResposta(d); !slices.Equal(p, []string{"SEPARANDO"}) {
			t.Errorf("permitidas do Detalhe = %v, quero [SEPARANDO]", p)
		}

		// O Detalhe do Administrador NÃO é o do Comprador: `expira_em`,
		// `tentativas_restantes`, `pode_cancelar` e `disponivel` são da
		// decisão de quem comprou, e reusar aquela saída os traria de carona.
		for _, proibida := range []string{"expira_em", "tentativas_restantes", "pode_cancelar", "disponivel"} {
			if slices.Contains(chavesDe(d), proibida) {
				t.Errorf("o Detalhe administrativo carrega %q, que é do Comprador", proibida)
			}
		}
	})

	t.Run("Pedido inexistente, malformado ou com Sessão de Comprador é 404", func(t *testing.T) {
		for _, caso := range []struct {
			nome, id string
			cookie   *http.Cookie
		}{
			{"inexistente", uuidNuncaUsado, admin},
			{"malformado", "nao-e-uuid", admin},
			{"Sessão de Comprador", doPainel, comprador},
			{"sem Sessão", doPainel, nil},
		} {
			if resp := pegarPedidoAdmin(t, rotas, caso.id, caso.cookie); resp.Code != http.StatusNotFound {
				t.Errorf("%s: status = %d (%s), quero 404", caso.nome, resp.Code, resp.Body.String())
			}
		}
	})

	t.Run("as três transições andam com ator ADMINISTRADOR, e o ENVIADO consolida o Estoque", func(t *testing.T) {
		antes := estoqueTotalDe(escrivaninha)
		for _, passo := range []struct {
			de, para   string
			permitidas []string
			terminal   bool
		}{
			{"PAGO", "SEPARANDO", []string{"ENVIADO"}, false},
			{"SEPARANDO", "ENVIADO", []string{"ENTREGUE"}, false},
			{"ENVIADO", "ENTREGUE", []string{}, true},
		} {
			resp := postarTransicao(t, rotas, doPainel, passo.de, passo.para, admin)
			if resp.Code != http.StatusOK {
				t.Fatalf("%s → %s = %d (%s), quero 200", passo.de, passo.para, resp.Code, resp.Body.String())
			}
			corpo := decodificar(t, resp)
			if corpo["status"] != passo.para || corpo["terminal"] != passo.terminal {
				t.Errorf("%s → %s: corpo = %v", passo.de, passo.para, corpo)
			}
			// Os botões da linha se remontam da própria resposta (AD-18).
			if p := permitidasDaResposta(corpo); !slices.Equal(p, passo.permitidas) {
				t.Errorf("permitidas depois de %s = %v, quero %v", passo.para, p, passo.permitidas)
			}
			if s := statusDe(t, pool, doPainel); s != passo.para {
				t.Fatalf("no banco = %s, quero %s", s, passo.para)
			}
			quer := passo.de + "|" + passo.para + "|ADMINISTRADOR"
			if h := transicoesDe(t, pool, doPainel); h[len(h)-1] != quer {
				t.Errorf("última linha do histórico = %q, quero %q", h[len(h)-1], quer)
			}
			// O efeito é de Transicionar, e é só o ENVIADO que baixa o total.
			if passo.para == "ENVIADO" {
				if depois := estoqueTotalDe(escrivaninha); depois != antes-1 {
					t.Errorf("estoque_total = %d, quero %d: SEPARANDO → ENVIADO consolida", depois, antes-1)
				}
				if r := reservasDe(t, pool, doPainel); !slices.Equal(r, []string{"CONSOLIDADA"}) {
					t.Errorf("reservas = %v; a consolidação encerra a Reserva", r)
				}
			}
		}
		// ENTREGUE é terminal: nenhum botão, e a lista vazia, nunca `null`.
		if p := permitidasDaResposta(decodificar(t, pegarPedidoAdmin(t, rotas, doPainel, admin))); len(p) != 0 {
			t.Errorf("permitidas de ENTREGUE = %v, quero nenhuma", p)
		}
	})

	t.Run("a transição fora da tabela é 409 com as permitidas, e o Administrador não cancela", func(t *testing.T) {
		id := pago(comprador)
		antes := fotografia(id)
		for _, caso := range []struct {
			nome, de, para string
			permitidas     []any
		}{
			// Pular uma etapa: a linha PAGO → ENTREGUE não existe.
			{"pulando uma etapa", "PAGO", "ENTREGUE", []any{"SEPARANDO"}},
			// FR-32: o Administrador não cancela. Não há `if` que o recuse —
			// a tabela do AD-3 simplesmente não tem essa linha para ele.
			{"cancelando pela API", "PAGO", "CANCELADO", []any{"SEPARANDO"}},
			// E a volta: nenhuma transição do Administrador anda para trás.
			{"voltando", "PAGO", "AGUARDANDO_PAGAMENTO", []any{"SEPARANDO"}},
		} {
			resp := postarTransicao(t, rotas, id, caso.de, caso.para, admin)
			if resp.Code != http.StatusConflict {
				t.Fatalf("%s: status = %d (%s), quero 409", caso.nome, resp.Code, resp.Body.String())
			}
			corpo := decodificar(t, resp)
			if corpo["codigo"] != "TRANSICAO_INVALIDA" {
				t.Errorf("%s: codigo = %v, quero TRANSICAO_INVALIDA", caso.nome, corpo["codigo"])
			}
			dados, _ := corpo["dados"].(map[string]any)
			if !slices.Equal(dados["permitidas"].([]any), caso.permitidas) {
				t.Errorf("%s: dados.permitidas = %v, quero %v", caso.nome, dados["permitidas"], caso.permitidas)
			}
		}
		if depois := fotografia(id); !slices.Equal(depois, antes) {
			t.Errorf("a transição recusada gravou algo:\nantes  %v\ndepois %v", antes, depois)
		}
		// E não existe rota administrativa de cancelamento (FR-32): o que
		// responde ali é o 404 de rota inexistente do prefixo.
		semRota := comCorpo(t, rotas, http.MethodPost,
			"/api/v1/admin/pedidos/"+id+"/cancelamento", "", "", admin)
		if semRota.Code != http.StatusNotFound {
			t.Errorf("cancelamento administrativo = %d (%s), quero 404: a rota não existe", semRota.Code, semRota.Body.String())
		}
	})

	t.Run("a transição de Pedido inexistente ou malformado é 404, e não 500", func(t *testing.T) {
		// A leitura travada é quem distingue os dois do CAS vazio, e o
		// ErrNoRows dela precisa de tradução: sem ela a rota sai 500, e é o
		// que este subteste existe para ver.
		for _, caso := range []struct{ nome, id string }{
			{"inexistente", uuidNuncaUsado},
			{"malformado", "nao-e-uuid"},
		} {
			resp := postarTransicao(t, rotas, caso.id, "PAGO", "SEPARANDO", admin)
			if resp.Code != http.StatusNotFound {
				t.Fatalf("%s: status = %d (%s), quero 404", caso.nome, resp.Code, resp.Body.String())
			}
			if codigo := decodificar(t, resp)["codigo"]; codigo != "NAO_ENCONTRADO" {
				t.Errorf("%s: codigo = %v, quero NAO_ENCONTRADO", caso.nome, codigo)
			}
		}
	})

	t.Run("`de` vazio ou fora dos sete é transição inválida, com permitidas vazio e nada gravado", func(t *testing.T) {
		// A rota não tem validação própria de `de`: quem decide é a tabela do
		// AD-3, e `Permitidas` de um Status que não existe é a lista vazia.
		// Um 400 aqui seria uma segunda declaração de quais Status existem.
		id := pago(comprador)
		antes := fotografia(id)
		for _, caso := range []struct{ nome, de string }{
			{"vazio", ""},
			{"fora dos sete", "SEPARANDO_TALVEZ"},
		} {
			resp := postarTransicao(t, rotas, id, caso.de, "SEPARANDO", admin)
			if resp.Code != http.StatusConflict {
				t.Fatalf("%s: status = %d (%s), quero 409", caso.nome, resp.Code, resp.Body.String())
			}
			corpo := decodificar(t, resp)
			if corpo["codigo"] != "TRANSICAO_INVALIDA" {
				t.Errorf("%s: codigo = %v, quero TRANSICAO_INVALIDA", caso.nome, corpo["codigo"])
			}
			// A lista vazia, e nunca `null`: é o contrato que a tela lê.
			dados, _ := corpo["dados"].(map[string]any)
			if p, ok := dados["permitidas"].([]any); !ok || len(p) != 0 {
				t.Errorf("%s: dados.permitidas = %v, quero a lista vazia", caso.nome, dados["permitidas"])
			}
		}
		if depois := fotografia(id); !slices.Equal(depois, antes) {
			t.Errorf("a transição de `de` inválido gravou algo:\nantes  %v\ndepois %v", antes, depois)
		}
	})

	t.Run("o Pedido que a simulação já levou adiante sai como já avançado, e não como inválido", func(t *testing.T) {
		id := pago(comprador)
		// A tela leu PAGO; a simulação move o Pedido enquanto ela está aberta.
		simular(id, pedido.StatusPago, pedido.StatusSeparando)
		antes := fotografia(id)

		// O Administrador aciona a ação que VIU. É o `de` da tela que faz o
		// compare-and-swap distinguir os dois desfechos: lido no servidor,
		// este POST viraria SEPARANDO → SEPARANDO, que não está na tabela, e
		// sairia TRANSICAO_INVALIDA — a causa falsa que a UX proíbe nomear.
		resp := postarTransicao(t, rotas, id, "PAGO", "SEPARANDO", admin)
		if resp.Code != http.StatusConflict {
			t.Fatalf("status = %d (%s), quero 409", resp.Code, resp.Body.String())
		}
		corpo := decodificar(t, resp)
		if corpo["codigo"] != "ESTADO_JA_AVANCADO" {
			t.Fatalf("codigo = %v, quero ESTADO_JA_AVANCADO (a corrida não é transição inválida)", corpo["codigo"])
		}
		dados, _ := corpo["dados"].(map[string]any)
		if dados["status"] != "SEPARANDO" {
			t.Errorf("dados.status = %v, quero SEPARANDO: é o Status que o Alert informativo nomeia", dados["status"])
		}
		if depois := fotografia(id); !slices.Equal(depois, antes) {
			t.Errorf("a corrida perdida gravou algo:\nantes  %v\ndepois %v", antes, depois)
		}
		// E a linha da simulação continua sendo a única: o Administrador não
		// duplicou a transição que perdeu.
		if h := transicoesDe(t, pool, id); h[len(h)-1] != "PAGO|SEPARANDO|SIMULACAO" {
			t.Errorf("última linha = %q, quero a da simulação", h[len(h)-1])
		}
	})

	t.Run("a Sessão de Comprador não alcança a transição", func(t *testing.T) {
		id := pago(comprador)
		antes := fotografia(id)
		for _, caso := range []struct {
			nome   string
			cookie *http.Cookie
		}{
			{"Comprador", comprador},
			{"sem Sessão", nil},
		} {
			resp := postarTransicao(t, rotas, id, "PAGO", "SEPARANDO", caso.cookie)
			if resp.Code != http.StatusNotFound {
				t.Errorf("%s: status = %d (%s), quero o 404 do prefixo administrativo",
					caso.nome, resp.Code, resp.Body.String())
			}
		}
		if depois := fotografia(id); !slices.Equal(depois, antes) {
			t.Errorf("a transição negada gravou algo:\nantes  %v\ndepois %v", antes, depois)
		}
	})

	// A Tabela consultada em intervalo, e o corte é o `terminal` do Go: com o
	// Pedido em ENTREGUE a lista para de consultar. O teste prende o campo,
	// que é o que a tela lê — a decisão de parar é dela.
	t.Run("terminal acompanha a máquina, e não uma cópia da lista de Status", func(t *testing.T) {
		d := decodificar(t, pegarPedidoAdmin(t, rotas, doPainel, admin))
		if d["status"] != "ENTREGUE" || d["terminal"] != true {
			t.Errorf("ENTREGUE: status/terminal = %v/%v, quero ENTREGUE e true", d["status"], d["terminal"])
		}
		// E o outro lado, para a asserção acima não passar por um `true` fixo.
		emAndamento := decodificar(t, pegarPedidoAdmin(t, rotas, daBeatriz, admin))
		if emAndamento["terminal"] != false {
			t.Errorf("PAGO: terminal = %v, quero false", emAndamento["terminal"])
		}
	})

	// A guarda que a UX-DR9 pede é do prefixo, e está provada em
	// autorizacao_test.go; o que fica aqui é a forma do corpo, que não pode
	// nomear a Reserva de Estoque a ninguém.
	t.Run("nenhuma chave da resposta administrativa nomeia a Reserva", func(t *testing.T) {
		for _, chave := range chavesDe(decodificar(t, pegarPedidoAdmin(t, rotas, doPainel, admin))) {
			if strings.Contains(strings.ToLower(chave), "reserv") {
				t.Errorf("a chave %q nomeia a Reserva", chave)
			}
		}
	})
}
