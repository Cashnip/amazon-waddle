package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
)

// maquinaDeEstados prova contra Postgres de verdade o que o teste em memória
// de internal/pedido não alcança: o efeito de cada transição sobre a Reserva e
// o Estoque total na mesma transação, a releitura do Status que reclassifica o
// cancelamento, o histórico consultável e os gatilhos que tornam conteúdo e
// histórico imutáveis (5.1). Tem conta, Vendedor e Produto próprios, e por
// isso não mexe no Estoque de ninguém.
func maquinaDeEstados(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Marina Estados","email":"marina.estados@exemplo.br","senha":"senha-da-marina-1"}`), http.StatusCreated)

	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja da Máquina"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Máquina de Estados"}`, admin), http.StatusCreated)
	corpo, _ := json.Marshal(map[string]any{
		"nome": "Relógio de Estados", "descricao": "", "preco_centavos": 4900, "imagem_url": "",
		"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": 3, "ativo": true,
	})
	produto := idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(corpo), admin), http.StatusCreated)

	novoPedido := func() string {
		t.Helper()
		resp := postarPedido(t, rotas, `{"produto_id":"`+produto+`"}`, comprador)
		if resp.Code != http.StatusCreated {
			t.Fatalf("criar o Pedido: status = %d (%s)", resp.Code, resp.Body.String())
		}
		id, _ := decodificar(t, resp)["id"].(string)
		return id
	}
	// Cada transição na sua transação, como faz quem a chama de verdade:
	// sucesso comita, recusa reverte.
	transicionar := func(pedidoID string, de, para pedido.Status, ator pedido.Ator, motivo string) error {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação: %v", err)
		}
		defer tx.Rollback(ctx)
		if err := pedido.Transicionar(ctx, tx, pedidoID, de, para, ator, motivo); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit de %s → %s: %v", de, para, err)
		}
		return nil
	}
	avancar := func(pedidoID string, de, para pedido.Status, ator pedido.Ator, motivo string) {
		t.Helper()
		if err := transicionar(pedidoID, de, para, ator, motivo); err != nil {
			t.Fatalf("%s → %s por %s = %v; quero nil", de, para, ator, err)
		}
	}
	// reservas são os estados das Reservas do Pedido, na ordem em que nasceram.
	reservas := func(pedidoID string) string {
		t.Helper()
		return strings.Join(textoDe(t, pool, `
			SELECT estado FROM catalogo.reserva_estoque WHERE pedido_id = $1::uuid ORDER BY id`, pedidoID), ",")
	}
	total := func() string {
		t.Helper()
		return textoDe(t, pool, `SELECT estoque_total::text FROM catalogo.produto WHERE id = $1::uuid`, produto)[0]
	}
	ultima := func(pedidoID string) string {
		t.Helper()
		linhas := textoDe(t, pool, `
			SELECT status_anterior || '|' || status_novo || '|' || ator || '|' || coalesce(motivo, '∅')
			FROM pedido.transicao_status WHERE pedido_id = $1::uuid ORDER BY ocorrido_em, id`, pedidoID)
		if len(linhas) == 0 {
			t.Fatalf("o Pedido %s não tem histórico", pedidoID)
		}
		return linhas[len(linhas)-1]
	}
	linhasDoHistorico := func(pedidoID string) int {
		t.Helper()
		return len(textoDe(t, pool, `SELECT id::text FROM pedido.transicao_status WHERE pedido_id = $1::uuid`, pedidoID))
	}

	// Recusa, nova Tentativa, expiração e cancelamento sobre o mesmo Pedido:
	// cada passo confere o efeito da terceira coluna do AD-3.
	recusado := novoPedido()
	t.Run("recusa libera, nova Tentativa reserva de novo, expiração libera com motivo", func(t *testing.T) {
		avancar(recusado, pedido.StatusAguardandoPagamento, pedido.StatusPagamentoRecusado, pedido.AtorProvedor, "")
		if r, e := reservas(recusado), total(); r != "LIBERADA" || e != "3" {
			t.Errorf("depois da recusa: Reservas = %s e total = %s; quero LIBERADA e 3", r, e)
		}
		if u := ultima(recusado); u != "AGUARDANDO_PAGAMENTO|PAGAMENTO_RECUSADO|PROVEDOR|∅" {
			t.Errorf("última transição = %q", u)
		}

		avancar(recusado, pedido.StatusPagamentoRecusado, pedido.StatusAguardandoPagamento, pedido.AtorComprador, "")
		if r := reservas(recusado); r != "LIBERADA,ATIVA" {
			t.Errorf("depois da nova Tentativa: Reservas = %s; quero LIBERADA,ATIVA", r)
		}

		avancar(recusado, pedido.StatusAguardandoPagamento, pedido.StatusPagamentoRecusado, pedido.AtorVarredura, pedido.MotivoTempoEsgotado)
		if r := reservas(recusado); r != "LIBERADA,LIBERADA" {
			t.Errorf("depois da expiração: Reservas = %s; quero LIBERADA,LIBERADA", r)
		}
		if u := ultima(recusado); u != "AGUARDANDO_PAGAMENTO|PAGAMENTO_RECUSADO|VARREDURA|TEMPO_ESGOTADO" {
			t.Errorf("última transição = %q; o motivo da expiração tem de ficar no histórico", u)
		}

		// Sem Reserva ativa, cancelar não falha: Liberar é no-op (AD-5).
		avancar(recusado, pedido.StatusPagamentoRecusado, pedido.StatusCancelado, pedido.AtorComprador, "")
		if s := statusDe(t, pool, recusado); s != "CANCELADO" {
			t.Errorf("Status = %s, quero CANCELADO", s)
		}
	})

	t.Run("o histórico é consultável, em ordem, com ator e motivo", func(t *testing.T) {
		historico, err := pedido.Historico(ctx, pool, recusado)
		if err != nil {
			t.Fatalf("Historico = %v", err)
		}
		var tem []string
		for _, h := range historico {
			tem = append(tem, fmt.Sprintf("%s>%s:%s:%s", h.De, h.Para, h.Ator, h.Motivo))
			if h.Em.IsZero() || h.Em.Location() != time.UTC {
				t.Errorf("instante %v; quero absoluto em UTC", h.Em)
			}
		}
		quer := []string{
			">AGUARDANDO_PAGAMENTO:COMPRADOR:",
			"AGUARDANDO_PAGAMENTO>PAGAMENTO_RECUSADO:PROVEDOR:",
			"PAGAMENTO_RECUSADO>AGUARDANDO_PAGAMENTO:COMPRADOR:",
			"AGUARDANDO_PAGAMENTO>PAGAMENTO_RECUSADO:VARREDURA:TEMPO_ESGOTADO",
			"PAGAMENTO_RECUSADO>CANCELADO:COMPRADOR:",
		}
		if !slices.Equal(tem, quer) {
			t.Errorf("histórico = %v\nquero      %v", tem, quer)
		}

		// Pedido que não existe não tem histórico vazio: é o mesmo "nenhuma
		// linha" do identificador malformado.
		for _, id := range []string{"018f2b00-0000-7000-8000-000000000000", "nao-e-uuid"} {
			if _, err := pedido.Historico(ctx, pool, id); !errors.Is(err, pgx.ErrNoRows) {
				t.Errorf("Historico(%q) = %v; quero pgx.ErrNoRows", id, err)
			}
		}
	})

	entregue := novoPedido()
	t.Run("a entrega: a Reserva segura até ENVIADO, e só ali o total baixa", func(t *testing.T) {
		passos := []struct {
			de, para pedido.Status
			ator     pedido.Ator
			reserva  string
			total    string
		}{
			{pedido.StatusAguardandoPagamento, pedido.StatusPago, pedido.AtorProvedor, "ATIVA", "3"},
			{pedido.StatusPago, pedido.StatusSeparando, pedido.AtorAdministrador, "ATIVA", "3"},
			{pedido.StatusSeparando, pedido.StatusEnviado, pedido.AtorSimulacao, "CONSOLIDADA", "2"},
			{pedido.StatusEnviado, pedido.StatusEntregue, pedido.AtorAdministrador, "CONSOLIDADA", "2"},
		}
		for _, p := range passos {
			avancar(entregue, p.de, p.para, p.ator, "")
			if r, e := reservas(entregue), total(); r != p.reserva || e != p.total {
				t.Errorf("em %s: Reserva = %s e total = %s; quero %s e %s", p.para, r, e, p.reserva, p.total)
			}
		}
		if u := ultima(entregue); u != "ENVIADO|ENTREGUE|ADMINISTRADOR|∅" {
			t.Errorf("última transição = %q", u)
		}
	})

	t.Run("cancelar com Reserva ativa a libera, sem tocar no total", func(t *testing.T) {
		cancelado := novoPedido()
		avancar(cancelado, pedido.StatusAguardandoPagamento, pedido.StatusCancelado, pedido.AtorComprador, "")
		if r, e := reservas(cancelado), total(); r != "LIBERADA" || e != "2" {
			t.Errorf("Reserva = %s e total = %s; quero LIBERADA e 2", r, e)
		}
	})

	t.Run("nova Tentativa sem Estoque falha e o Pedido fica recusado", func(t *testing.T) {
		semEstoque := novoPedido()
		avancar(semEstoque, pedido.StatusAguardandoPagamento, pedido.StatusPagamentoRecusado, pedido.AtorProvedor, "")
		// Outro Pedido leva todo o disponível enquanto este está recusado.
		var outra string
		if err := pool.QueryRow(ctx, `
			INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
			SELECT id, uuidv7(), estoque_total, 'ATIVA' FROM catalogo.produto WHERE id = $1::uuid
			RETURNING id::text`, produto).Scan(&outra); err != nil {
			t.Fatalf("esgotar o Estoque: %v", err)
		}
		defer func() {
			if _, err := pool.Exec(ctx, `UPDATE catalogo.reserva_estoque SET estado = 'LIBERADA' WHERE id = $1::uuid`, outra); err != nil {
				t.Errorf("devolver o Estoque: %v", err)
			}
		}()
		antes := linhasDoHistorico(semEstoque)

		err := transicionar(semEstoque, pedido.StatusPagamentoRecusado, pedido.StatusAguardandoPagamento, pedido.AtorComprador, "")
		var falta catalogo.EstoqueInsuficiente
		if !errors.As(err, &falta) || falta.Disponivel != 0 {
			t.Fatalf("nova Tentativa sem Estoque = %v; quero EstoqueInsuficiente com disponível 0", err)
		}
		if s := statusDe(t, pool, semEstoque); s != "PAGAMENTO_RECUSADO" {
			t.Errorf("Status = %s; o CAS tinha de ser revertido com o efeito", s)
		}
		if depois := linhasDoHistorico(semEstoque); depois != antes {
			t.Errorf("%d linhas de histórico, quero as %d de antes", depois, antes)
		}
		if r := reservas(semEstoque); r != "LIBERADA" {
			t.Errorf("Reservas = %s; quero só a LIBERADA da recusa", r)
		}
	})

	t.Run("as três recusas contra o banco", func(t *testing.T) {
		antes := linhasDoHistorico(recusado)

		// Fora da tabela: nada é escrito, e Permitidas é a lista vazia.
		err := transicionar(recusado, pedido.StatusCancelado, pedido.StatusSeparando, pedido.AtorAdministrador, "")
		var invalida pedido.TransicaoInvalida
		if !errors.As(err, &invalida) || invalida.Permitidas == nil || len(invalida.Permitidas) != 0 {
			t.Errorf("CANCELADO → SEPARANDO = %#v; quero TransicaoInvalida com Permitidas vazia", err)
		}
		if depois := linhasDoHistorico(recusado); depois != antes {
			t.Errorf("a transição inválida escreveu %d linhas de histórico", depois-antes)
		}

		// Cancelar o que já foi entregue.
		if err := transicionar(entregue, pedido.StatusEntregue, pedido.StatusCancelado, pedido.AtorComprador, ""); !errors.Is(err, pedido.ErrForaDaJanelaDeCancelamento) {
			t.Errorf("cancelar ENTREGUE = %v; quero ErrForaDaJanelaDeCancelamento", err)
		}
		// A tela ainda mostrava SEPARANDO, mas o Pedido já saiu da janela: a
		// corrida perdida volta reclassificada, nunca como corrida.
		if err := transicionar(entregue, pedido.StatusSeparando, pedido.StatusCancelado, pedido.AtorComprador, ""); !errors.Is(err, pedido.ErrForaDaJanelaDeCancelamento) {
			t.Errorf("cancelar com SEPARANDO já superado = %v; quero ErrForaDaJanelaDeCancelamento", err)
		}
		// A corrida comum continua sendo corrida.
		if err := transicionar(entregue, pedido.StatusEnviado, pedido.StatusEntregue, pedido.AtorSimulacao, ""); !errors.Is(err, pedido.ErrEstadoJaAvancado) {
			t.Errorf("ENVIADO → ENTREGUE já feito = %v; quero ErrEstadoJaAvancado", err)
		}
		if r, e := reservas(entregue), total(); r != "CONSOLIDADA" || e != "2" {
			t.Errorf("depois das recusas: Reserva = %s e total = %s; nada devia ter mudado", r, e)
		}
	})

	t.Run("conteúdo e histórico são imutáveis no banco", func(t *testing.T) {
		for _, comando := range []string{
			`UPDATE pedido.pedido SET total_centavos = total_centavos + 1 WHERE id = $1::uuid`,
			`UPDATE pedido.pedido SET numero = numero || '-X' WHERE id = $1::uuid`,
			// O Status junto com o conteúdo: o Status não serve de carona.
			`UPDATE pedido.pedido SET status = 'CANCELADO', total_centavos = 1 WHERE id = $1::uuid`,
			`DELETE FROM pedido.pedido WHERE id = $1::uuid`,
			`UPDATE pedido.item_pedido SET preco_praticado_centavos = 1 WHERE pedido_id = $1::uuid`,
			`DELETE FROM pedido.item_pedido WHERE pedido_id = $1::uuid`,
			`UPDATE pedido.transicao_status SET motivo = 'OUTRO' WHERE pedido_id = $1::uuid`,
			`DELETE FROM pedido.transicao_status WHERE pedido_id = $1::uuid`,
		} {
			_, err := pool.Exec(ctx, comando, entregue)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23001" {
				t.Errorf("%s = %v; quero restrict_violation (23001)", comando, err)
			}
		}
		// TRUNCATE não dispara gatilho de linha, e tem o seu. Dentro de uma
		// transação revertida: se o gatilho faltasse, nada se perderia.
		for _, tabela := range []string{"pedido.transicao_status", "pedido.item_pedido", "pedido.pedido"} {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatalf("abrir a transação: %v", err)
			}
			_, err = tx.Exec(ctx, "TRUNCATE "+tabela+" CASCADE")
			_ = tx.Rollback(ctx)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23001" {
				t.Errorf("TRUNCATE %s = %v; quero restrict_violation (23001)", tabela, err)
			}
		}
	})
}

// envelhecerHistorico recua os instantes do histórico do Pedido, que é o que
// substitui esperar o intervalo da varredura. O histórico é imutável (5.1), e
// o gatilho que o protege é desligado só nesta transação, pelo papel de
// réplica — o usuário do contêiner de teste é superusuário. Em produção não
// existe caminho equivalente.
func envelhecerHistorico(t *testing.T, pool *pgxpool.Pool, pedidoID string, quanto time.Duration) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir a transação: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SET LOCAL session_replication_role = replica`); err != nil {
		t.Fatalf("desligar os gatilhos nesta transação: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pedido.transicao_status SET ocorrido_em = ocorrido_em - make_interval(secs => $2)
		WHERE pedido_id = $1::uuid`, pedidoID, quanto.Seconds()); err != nil {
		t.Fatalf("envelhecer o histórico: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit do envelhecimento: %v", err)
	}
}
