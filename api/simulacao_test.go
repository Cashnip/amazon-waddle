package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/pedido"
)

// passoDoHistorico é uma linha de pedido.transicao_status montada à mão: de,
// para e ator. O Status do Pedido é o `para` da última.
type passoDoHistorico struct{ de, para, ator string }

// simulacaoDeEntregaFR33 é a matriz de servidor da 6.6: cada condição da FR-33
// e do AD-6 que a simulação de entrega afirma em comentário e que o caminho
// feliz de `simulacaoDeEntrega` não alcança — o que ela nunca move, o Pedido
// que outra transação segura e o Pedido cujo avanço falha. `SimularEntrega` é
// chamada direto, como o tique a chama, com o intervalo como parâmetro.
//
// O estado é montado por INSERT, no molde de `cmd/azamon/main_test.go`: os
// Status de partida não são alcançáveis por rota sem esperar o relógio, e o
// Pedido que falha precisa de uma Reserva que `Reservar` recusaria. Nenhum
// UPDATE de Status: todo avanço aqui é o da própria simulação, por
// `Transicionar`. O histórico nasce com uma hora e o intervalo é de meia —
// todo Pedido montado já venceu, e o que a simulação avança recomeça a contar
// de agora, então só se move uma etapa por chamada.
//
// Conta, Vendedor, Categoria e Produtos próprios. A posição em
// `TestSessaoEProduto` é indiferente: `SimularEntrega` é global, mas o Pedido
// que falha é desarmado no fim do subteste que o prova.
func simulacaoDeEntregaFR33(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	const intervalo = 30 * time.Minute

	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	const email = "sofia.simulacao@exemplo.br"
	cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Sofia Simulação","email":"`+email+`","senha":"senha-da-sofia-1"}`), http.StatusCreated)
	donos := textoDe(t, pool, `SELECT id::text FROM identidade.comprador WHERE email = $1`, email)
	if len(donos) != 1 {
		t.Fatalf("a conta da matriz não foi gravada: %v", donos)
	}
	comprador := donos[0]

	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja da Simulação"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Simulação"}`, admin), http.StatusCreated)
	novoProduto := func(nome string, estoque int) string {
		t.Helper()
		corpo, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": 5000, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": estoque, "ativo": true,
		})
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(corpo), admin), http.StatusCreated)
	}

	estoqueDe := func(t *testing.T, produto string) int {
		t.Helper()
		v := textoDe(t, pool, `SELECT estoque_total::text FROM catalogo.produto WHERE id = $1::uuid`, produto)
		if len(v) != 1 {
			t.Fatalf("ler o Estoque de %s: %v", produto, v)
		}
		n, err := strconv.Atoi(v[0])
		if err != nil {
			t.Fatalf("Estoque ilegível: %v", err)
		}
		return n
	}
	// retratoDe é o que "não mexeu" confere, numa linha: o Status, quantas
	// linhas o histórico tem, quantas delas são da simulação, e o estado da
	// Reserva. Só o Status deixaria passar a transição que gravou o histórico
	// e voltou, ou a que consolidou sem avançar.
	retratoDe := func(t *testing.T, id string) string {
		t.Helper()
		v := textoDe(t, pool, `
			SELECT p.status
			    || '|' || (SELECT count(*) FROM pedido.transicao_status h WHERE h.pedido_id = p.id)
			    || '|' || (SELECT count(*) FROM pedido.transicao_status h WHERE h.pedido_id = p.id AND h.ator = 'SIMULACAO')
			    || '|' || coalesce((SELECT string_agg(r.estado, ',' ORDER BY r.id)
			                        FROM catalogo.reserva_estoque r WHERE r.pedido_id = p.id), '')
			FROM pedido.pedido p WHERE p.id = $1::uuid`, id)
		if len(v) != 1 {
			t.Fatalf("Pedido %s não encontrado", id)
		}
		return v[0]
	}

	// numa abre, entrega e comita uma transação: o Pedido só vira candidato
	// quando a linha do histórico comita, e até lá a Reserva já existe.
	numa := func(t *testing.T, montar func(tx pgx.Tx)) {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação: %v", err)
		}
		defer tx.Rollback(ctx)
		montar(tx)
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit da montagem: %v", err)
		}
	}
	// montar grava o Pedido no Status do último passo, a Reserva (se houver)
	// e o histórico, com a última linha de uma hora atrás e as anteriores um
	// minuto mais velhas cada.
	numeros := 0
	montar := func(t *testing.T, tx pgx.Tx, produto, reserva string, quantidade int, passos ...passoDoHistorico) string {
		t.Helper()
		numeros++
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO pedido.pedido (numero, comprador_id, status, subtotal_centavos, frete_centavos, total_centavos)
			VALUES ($1, $2::uuid, $3, 5000, 0, 5000)
			RETURNING id::text`, fmt.Sprintf("AZ-FR33-%06d", numeros), comprador, passos[len(passos)-1].para).Scan(&id); err != nil {
			t.Fatalf("criar o Pedido: %v", err)
		}
		if reserva != "" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
				VALUES ($1::uuid, $2::uuid, $3, $4)`, produto, id, quantidade, reserva); err != nil {
				t.Fatalf("criar a Reserva: %v", err)
			}
		}
		for i, p := range passos {
			if _, err := tx.Exec(ctx, `
				INSERT INTO pedido.transicao_status (pedido_id, status_anterior, status_novo, ator, ocorrido_em)
				VALUES ($1::uuid, $2, $3, $4, now() - make_interval(mins => $5))`,
				id, p.de, p.para, p.ator, 60+len(passos)-1-i); err != nil {
				t.Fatalf("registrar %s → %s: %v", p.de, p.para, err)
			}
		}
		return id
	}
	nascimento := passoDoHistorico{"", "AGUARDANDO_PAGAMENTO", "COMPRADOR"}
	aprovado := passoDoHistorico{"AGUARDANDO_PAGAMENTO", "PAGO", "PROVEDOR"}
	separado := passoDoHistorico{"PAGO", "SEPARANDO", "ADMINISTRADOR"}

	// simular é um tique da simulação com prazo curto: trocado o SKIP LOCKED
	// por uma trava que espera, a chamada penduraria na linha presa até o
	// `context` estourar — e isso é falha, e não lentidão.
	simular := func(t *testing.T) error {
		t.Helper()
		curto, cancelar := context.WithTimeout(ctx, 5*time.Second)
		defer cancelar()
		err := pedido.SimularEntrega(curto, pool, intervalo)
		if curto.Err() != nil {
			t.Fatalf("a simulação estourou o prazo (%v): esperou uma trava em vez de pular o Pedido", err)
		}
		return err
	}

	t.Run("nunca move CANCELADO, AGUARDANDO_PAGAMENTO, PAGAMENTO_RECUSADO nem ENTREGUE", func(t *testing.T) {
		produto := novoProduto("Estante da Simulação Parada", 20)
		var parados []string
		numa(t, func(tx pgx.Tx) {
			parados = []string{
				montar(t, tx, produto, "LIBERADA", 1, nascimento,
					passoDoHistorico{"AGUARDANDO_PAGAMENTO", "CANCELADO", "COMPRADOR"}),
				montar(t, tx, produto, "ATIVA", 1, nascimento),
				montar(t, tx, produto, "LIBERADA", 1, nascimento,
					passoDoHistorico{"AGUARDANDO_PAGAMENTO", "PAGAMENTO_RECUSADO", "PROVEDOR"}),
				montar(t, tx, produto, "CONSOLIDADA", 1, nascimento, aprovado,
					passoDoHistorico{"PAGO", "SEPARANDO", "SIMULACAO"},
					passoDoHistorico{"SEPARANDO", "ENVIADO", "SIMULACAO"},
					passoDoHistorico{"ENVIADO", "ENTREGUE", "SIMULACAO"}),
			}
		})
		antes := make([]string, len(parados))
		for i, id := range parados {
			antes[i] = retratoDe(t, id)
		}
		estoque := estoqueDe(t, produto)

		if err := simular(t); err != nil {
			t.Fatalf("simular = %v; quero nil — nenhum destes Status tem passagem na simulação", err)
		}
		for i, id := range parados {
			if agora := retratoDe(t, id); agora != antes[i] {
				t.Errorf("Pedido em %s mexido: %s → %s", strings.SplitN(antes[i], "|", 2)[0], antes[i], agora)
			}
		}
		if e := estoqueDe(t, produto); e != estoque {
			t.Errorf("estoque_total = %d, quero %d intacto", e, estoque)
		}
	})

	t.Run("o Pedido travado por outra transação é pulado, e o do lado avança", func(t *testing.T) {
		produto := novoProduto("Mesa da Trava", 20)
		// O travado nasce primeiro: com a trava que espera, é nele que a
		// chamada penduraria antes de chegar ao outro.
		var travado, livre string
		numa(t, func(tx pgx.Tx) {
			travado = montar(t, tx, produto, "ATIVA", 1, nascimento, aprovado)
			livre = montar(t, tx, produto, "ATIVA", 1, nascimento, aprovado)
		})
		const pago, separando = "PAGO|2|0|ATIVA", "SEPARANDO|3|1|ATIVA"
		for _, id := range []string{travado, livre} {
			if r := retratoDe(t, id); r != pago {
				t.Fatalf("montagem = %s, quero %s", r, pago)
			}
		}

		// O Administrador no meio da transição: a linha do Pedido presa por
		// uma transação aberta, como a do `TravarPedidoParaAdministrador`.
		trava, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação da trava: %v", err)
		}
		defer trava.Rollback(ctx)
		if _, err := trava.Exec(ctx, `SELECT 1 FROM pedido.pedido WHERE id = $1::uuid FOR UPDATE`, travado); err != nil {
			t.Fatalf("travar o Pedido: %v", err)
		}

		if err := simular(t); err != nil {
			t.Fatalf("simular com a linha presa = %v; quero nil — a linha travada não é falha", err)
		}
		if r := retratoDe(t, travado); r != pago {
			t.Errorf("travado = %s com a trava presa; quero %s", r, pago)
		}
		if r := retratoDe(t, livre); r != separando {
			t.Errorf("o do lado = %s; quero %s na mesma chamada", r, separando)
		}

		if err := trava.Rollback(ctx); err != nil {
			t.Fatalf("soltar a trava: %v", err)
		}
		if err := simular(t); err != nil {
			t.Fatalf("simular com a trava solta = %v; quero nil", err)
		}
		if r := retratoDe(t, travado); r != separando {
			t.Errorf("travado = %s com a trava solta; quero %s — o tique seguinte o encontra", r, separando)
		}
		// O do lado acabou de avançar e o relógio dele recomeçou: não anda de novo.
		if r := retratoDe(t, livre); r != separando {
			t.Errorf("o do lado = %s na chamada seguinte; quero %s parado", r, separando)
		}
	})

	t.Run("o Pedido que falha não derruba o outro e é o único nomeado no erro", func(t *testing.T) {
		// O `CHECK (estoque_total >= 0)` derruba o `Consolidar` do Pedido cuja
		// Reserva passa do Estoque total — a falha natural, sem gancho.
		curto := novoProduto("Cadeira da Falha", 2)
		farto := novoProduto("Cadeira da Sobra", 20)
		// Na mesma transação e nesta ordem: `PedidosParaAvancar` ordena por id
		// (uuidv7, monotônico na sessão), então a falha vem antes do avanço do
		// outro — é o que distingue juntar as falhas de desistir na primeira.
		const reservado = 3
		var falho, saudavel string
		numa(t, func(tx pgx.Tx) {
			falho = montar(t, tx, curto, "ATIVA", reservado, nascimento, aprovado, separado)
			saudavel = montar(t, tx, farto, "ATIVA", 1, nascimento, aprovado, separado)
		})
		// Terminada a prova, o Pedido que falha deixa de falhar: `SimularEntrega`
		// é global, e ele derrubaria toda chamada seguinte da suíte, em
		// qualquer subteste registrado depois deste. Com o Estoque total
		// coberto, a Reserva consolida e ele anda como qualquer outro.
		t.Cleanup(func() {
			if _, err := pool.Exec(ctx, `UPDATE catalogo.produto SET estoque_total = $2 WHERE id = $1::uuid`, curto, reservado); err != nil {
				t.Errorf("desarmar o Pedido que falha: %v", err)
			}
		})
		if v := textoDe(t, pool, `SELECT ($1::uuid < $2::uuid)::text`, falho, saudavel); len(v) != 1 || v[0] != "true" {
			t.Fatalf("o Pedido que falha não nasceu antes do saudável: %v", v)
		}
		const separando = "SEPARANDO|3|0|ATIVA"
		for _, id := range []string{falho, saudavel} {
			if r := retratoDe(t, id); r != separando {
				t.Fatalf("montagem = %s, quero %s", r, separando)
			}
		}
		estoqueCurto, estoqueFarto := estoqueDe(t, curto), estoqueDe(t, farto)

		err := simular(t)
		if err == nil {
			t.Fatal("simular = nil; quero o erro do Pedido cuja consolidação falhou")
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
			t.Errorf("erro = %v; quero a check_violation (23514) do estoque_total", err)
		}
		if n := strings.Count(err.Error(), falho); n != 1 {
			t.Errorf("o erro nomeia o Pedido que falhou %d vez(es); quero 1: %v", n, err)
		}
		if strings.Contains(err.Error(), saudavel) {
			t.Errorf("o erro nomeia o Pedido saudável: %v", err)
		}

		if r := retratoDe(t, falho); r != separando {
			t.Errorf("o que falhou = %s; quero %s — a transição dele é desfeita inteira", r, separando)
		}
		if e := estoqueDe(t, curto); e != estoqueCurto {
			t.Errorf("estoque_total do que falhou = %d, quero %d intacto", e, estoqueCurto)
		}
		if r := retratoDe(t, saudavel); r != "ENVIADO|4|1|CONSOLIDADA" {
			t.Errorf("o saudável = %s; quero ENVIADO|4|1|CONSOLIDADA na mesma chamada", r)
		}
		if e := estoqueDe(t, farto); e != estoqueFarto-1 {
			t.Errorf("estoque_total do saudável = %d, quero %d", e, estoqueFarto-1)
		}
	})
}
