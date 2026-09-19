package main

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// A ordem do arranque é normativa: sem banco, o processo não chega a escutar.
func TestExecutarNaoServeSemBanco(t *testing.T) {
	const endereco = "127.0.0.1:8099"
	t.Setenv("AZAMON_HTTP_ADDR", endereco)
	t.Setenv("AZAMON_POSTGRES_DSN", "postgres://azamon@127.0.0.1:1/azamon")
	t.Setenv("AZAMON_REDIS_URL", "redis://127.0.0.1:1/0")
	t.Setenv("AZAMON_WEBHOOK_SEGREDO", "segredo-de-teste")

	ctx, cancelar := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelar()

	if err := executar(ctx, io.Discard); err == nil {
		t.Fatal("executar devia falhar sem banco")
	}
	c, err := net.DialTimeout("tcp", endereco, 100*time.Millisecond)
	if err == nil {
		c.Close()
		t.Fatalf("%s ficou escutando; a migração tem de barrar o servidor", endereco)
	}
}

// Sem este teste, apagar a chamada a Semear de executar() deixa `go test ./...`
// inteiro verde: main_test falha dentro de Migrar e nunca chega lá, e
// db/schema_test chama Semear direto. A demonstração é que subiria com a loja
// vazia. Aqui o arranque roda inteiro, contra um Postgres de verdade.
func TestExecutarDeixaOCatalogoSemeado(t *testing.T) {
	const endereco = "127.0.0.1:8098"
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx, "postgres:18.6",
		tcpostgres.WithDatabase("azamon"),
		tcpostgres.WithUsername("azamon"),
		tcpostgres.WithPassword("azamon"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(2*time.Minute)),
	)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(ctr) })
	if err != nil {
		t.Fatalf("subir postgres: %v", err)
	}
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("DSN do contêiner: %v", err)
	}

	t.Setenv("AZAMON_HTTP_ADDR", endereco)
	t.Setenv("AZAMON_POSTGRES_DSN", dsn)
	t.Setenv("AZAMON_REDIS_URL", "redis://127.0.0.1:1/0")
	t.Setenv("AZAMON_WEBHOOK_SEGREDO", "segredo-de-teste")
	// O webhook do Provedor Simulado volta para o próprio binário, que é como
	// ele roda em compose. Com isso o tique, o transporte HTTP de verdade, a
	// rota e a varredura entram todos no caminho do teste.
	t.Setenv("AZAMON_WEBHOOK_BASE_URL", "http://"+endereco)
	t.Setenv("AZAMON_CONFIRMACAO_ATRASO", "100ms")
	// Em demonstração a etapa é de 30 s; aqui o que importa é que ela exista e
	// seja lida da configuração, então encolhe para o teste caber num tique.
	t.Setenv("AZAMON_ENTREGA_INTERVALO", "200ms")

	servindo, parar := context.WithCancel(ctx)
	fim := make(chan error, 1)
	go func() { fim <- executar(servindo, io.Discard) }()
	// Desligar é explícito e não só adiado: o subteste do interruptor precisa
	// que este binário já tenha parado antes de o próximo subir, senão a
	// varredura dele avançaria o Pedido que o outro espera parado.
	parado := false
	pararEsperando := func() {
		if parado {
			return
		}
		parado = true
		parar()
		<-fim
	}
	defer pararEsperando()

	// Escutar é o sinal de que migração e semente passaram: a ordem do
	// arranque é config → migrações → semente → servir.
	prazo := time.Now().Add(time.Minute)
	for {
		if c, err := net.DialTimeout("tcp", endereco, time.Second); err == nil {
			c.Close()
			break
		}
		if time.Now().After(prazo) {
			t.Fatal("executar não chegou a escutar")
		}
		time.Sleep(50 * time.Millisecond)
	}

	conexao, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	defer conexao.Close(ctx)
	var n int
	if err := conexao.QueryRow(ctx, `SELECT count(*) FROM catalogo.produto`).Scan(&n); err != nil {
		t.Fatalf("contar Produtos: %v", err)
	}
	if n == 0 {
		t.Error("o arranque terminou com o catálogo vazio; Semear não rodou")
	}

	pedidoID, produtoID, estoqueAntes := varreduraLevaOPedidoAPago(t, ctx, conexao)
	simulacaoLevaOPedidoAEntregue(t, ctx, conexao, pedidoID, produtoID, estoqueAntes)

	pararEsperando()
	interruptorDesligaASimulacao(t, ctx, conexao)
}

// interruptorDesligaASimulacao prova a consequência da FR-33 que nenhuma outra
// verificação alcança: a simulação pode ser desligada por configuração, para o
// Administrador conduzir a apresentação manualmente. Sem isto, apagar o
// `if cfg.EntregaSimulacaoAtiva` do tique deixa tudo verde.
//
// A configuração é lida uma vez, no arranque, então o interruptor só é
// observável subindo o binário de novo — no mesmo Postgres, que é a parte cara.
func interruptorDesligaASimulacao(t *testing.T, ctx context.Context, conexao *pgx.Conn) {
	t.Helper()
	const endereco = "127.0.0.1:8097"
	t.Setenv("AZAMON_HTTP_ADDR", endereco)
	t.Setenv("AZAMON_WEBHOOK_BASE_URL", "http://"+endereco)
	t.Setenv("AZAMON_VARREDURA_INTERVALO", "50ms")
	t.Setenv("AZAMON_ENTREGA_SIMULACAO_ATIVA", "false")

	// Um Pedido em PAGO com o histórico já vencido: com a simulação ligada,
	// este é exatamente o Pedido que o primeiro tique moveria.
	var pedidoID string
	if err := conexao.QueryRow(ctx, `
		INSERT INTO pedido.pedido (numero, comprador_id, status, total_centavos)
		VALUES ('AZ-INTERRUPTOR-0001', uuidv7(), 'PAGO', 32900)
		RETURNING id::text`).Scan(&pedidoID); err != nil {
		t.Fatalf("criar o Pedido: %v", err)
	}
	if _, err := conexao.Exec(ctx, `
		INSERT INTO pedido.transicao_status (pedido_id, status_anterior, status_novo, ator, ocorrido_em)
		VALUES ($1::uuid, 'AGUARDANDO_PAGAMENTO', 'PAGO', 'PROVEDOR', now() - interval '1 hour')`,
		pedidoID); err != nil {
		t.Fatalf("registrar a transição: %v", err)
	}

	servindo, parar := context.WithCancel(ctx)
	fim := make(chan error, 1)
	go func() { fim <- executar(servindo, io.Discard) }()
	defer func() { parar(); <-fim }()

	prazo := time.Now().Add(time.Minute)
	for {
		if c, err := net.DialTimeout("tcp", endereco, time.Second); err == nil {
			c.Close()
			break
		}
		if time.Now().After(prazo) {
			t.Fatal("o segundo arranque não chegou a escutar")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Vários tiques de 50 ms: tempo de sobra para a simulação agir, se
	// estivesse ligada.
	time.Sleep(2 * time.Second)
	var status string
	if err := conexao.QueryRow(ctx,
		`SELECT status FROM pedido.pedido WHERE id = $1::uuid`, pedidoID).Scan(&status); err != nil {
		t.Fatalf("ler o Status: %v", err)
	}
	if status != "PAGO" {
		t.Errorf("status = %s com a simulação desligada; quero PAGO parado", status)
	}
}

// varreduraLevaOPedidoAPago é o comportamento-título da estória 1.7 medido no
// binário inteiro: sem isto, apagar o `go varrer(...)` de executar() ou trocar
// o caminho do webhook deixaria `go test ./...` verde.
//
// O Pedido, a Tentativa e a Reserva entram direto no banco — o que se exercita
// aqui é o relógio e o transporte, não a compra, que api/ já cobre. O total tem
// centavos em `,00`, que é a faixa que o Simulado aprova. A Reserva existe para
// a continuação: é ela que a simulação consolida em SEPARANDO → ENVIADO.
func varreduraLevaOPedidoAPago(t *testing.T, ctx context.Context, conexao *pgx.Conn) (string, string, int32) {
	t.Helper()
	var pedidoID string
	if err := conexao.QueryRow(ctx, `
		INSERT INTO pedido.pedido (numero, comprador_id, status, total_centavos)
		VALUES ('AZ-VARREDURA-000001', uuidv7(), 'AGUARDANDO_PAGAMENTO', 32900)
		RETURNING id::text`).Scan(&pedidoID); err != nil {
		t.Fatalf("criar o Pedido: %v", err)
	}
	if _, err := conexao.Exec(ctx, `
		INSERT INTO pagamento.tentativa_pagamento (pedido_id, total_centavos, id_externo, numero)
		VALUES ($1::uuid, 32900, $2, 1)`, pedidoID, pagamento.IDExterno(pedidoID, 1)); err != nil {
		t.Fatalf("criar a Tentativa: %v", err)
	}
	var produtoID string
	var estoqueAntes int32
	if err := conexao.QueryRow(ctx, `
		SELECT id::text, estoque_total FROM catalogo.produto ORDER BY id LIMIT 1`).
		Scan(&produtoID, &estoqueAntes); err != nil {
		t.Fatalf("escolher o Produto: %v", err)
	}
	if _, err := conexao.Exec(ctx, `
		INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
		VALUES ($1::uuid, $2::uuid, 1, 'ATIVA')`, produtoID, pedidoID); err != nil {
		t.Fatalf("criar a Reserva: %v", err)
	}

	// Mesmo estilo de espera do arranque: o tique é de 1 s e o atraso da
	// confirmação, de 100 ms.
	//
	// Quem responde é o histórico, e não o Status corrente: a simulação de
	// entrega roda no mesmo binário com intervalo de 200 ms, então o Pedido não
	// fica em PAGO — e uma espera que só quebrasse em `status == "PAGO"`
	// perderia a janela numa máquina carregada e acusaria regressão onde não há.
	prazo := time.Now().Add(30 * time.Second)
	for {
		var chegou bool
		if err := conexao.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pedido.transicao_status
				WHERE pedido_id = $1::uuid AND status_novo = 'PAGO'
			)`, pedidoID).Scan(&chegou); err != nil {
			t.Fatalf("ler o histórico: %v", err)
		}
		if chegou {
			break
		}
		if time.Now().After(prazo) {
			t.Fatal("o Pedido não registrou a passagem por PAGO; a varredura do binário não a aplicou")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// E a confirmação atravessou a rota de verdade, com o segredo no
	// cabeçalho: uma linha na inbox, aplicada.
	var estado string
	if err := conexao.QueryRow(ctx, `
		SELECT c.estado FROM pagamento.confirmacao_recebida c
		JOIN pagamento.tentativa_pagamento t ON t.id = c.tentativa_id
		WHERE t.pedido_id = $1::uuid`, pedidoID).Scan(&estado); err != nil {
		t.Fatalf("ler a confirmação: %v", err)
	}
	if estado != "APLICADA" {
		t.Errorf("estado da confirmação = %s, quero APLICADA", estado)
	}
	return pedidoID, produtoID, estoqueAntes
}

// simulacaoLevaOPedidoAEntregue é o comportamento-título da estória 1.8 medido
// no binário inteiro. Sem isto, apagar o passo `simular` do tique deixaria
// `go test ./...` verde com o Pedido parado em PAGO para sempre — que é
// exatamente o fim que a épica existe para não ter.
//
// Ninguém empurra nada: o Pedido chega a PAGO pela varredura e daí em diante só
// o relógio o move, uma etapa por AZAMON_ENTREGA_INTERVALO vencido.
func simulacaoLevaOPedidoAEntregue(t *testing.T, ctx context.Context, conexao *pgx.Conn, pedidoID, produtoID string, estoqueAntes int32) {
	t.Helper()
	prazo := time.Now().Add(30 * time.Second)
	for {
		var status string
		if err := conexao.QueryRow(ctx,
			`SELECT status FROM pedido.pedido WHERE id = $1::uuid`, pedidoID).Scan(&status); err != nil {
			t.Fatalf("ler o Status: %v", err)
		}
		if status == "ENTREGUE" {
			break
		}
		if time.Now().After(prazo) {
			t.Fatalf("o Pedido ficou em %s; a simulação não o levou a ENTREGUE", status)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Uma linha de histórico por avanço, com a simulação como ator: PAGO →
	// SEPARANDO → ENVIADO → ENTREGUE são três, e nenhuma etapa foi pulada.
	var avancos int
	if err := conexao.QueryRow(ctx, `
		SELECT count(*) FROM pedido.transicao_status
		WHERE pedido_id = $1::uuid AND ator = 'SIMULACAO'`, pedidoID).Scan(&avancos); err != nil {
		t.Fatalf("contar as transições: %v", err)
	}
	if avancos != 3 {
		t.Errorf("transições da simulação = %d, quero 3", avancos)
	}

	// A consolidação aconteceu uma vez só: a Reserva saiu de ATIVA e o Estoque
	// total baixou exatamente a quantidade reservada — nem duas vezes, nem em
	// nenhuma outra transição.
	var reserva string
	if err := conexao.QueryRow(ctx, `
		SELECT estado FROM catalogo.reserva_estoque WHERE pedido_id = $1::uuid`, pedidoID).Scan(&reserva); err != nil {
		t.Fatalf("ler a Reserva: %v", err)
	}
	if reserva != "CONSOLIDADA" {
		t.Errorf("estado da Reserva = %s, quero CONSOLIDADA", reserva)
	}
	var estoqueDepois int32
	if err := conexao.QueryRow(ctx, `
		SELECT estoque_total FROM catalogo.produto WHERE id = $1::uuid`, produtoID).Scan(&estoqueDepois); err != nil {
		t.Fatalf("ler o Estoque: %v", err)
	}
	if estoqueDepois != estoqueAntes-1 {
		t.Errorf("estoque_total = %d, quero %d", estoqueDepois, estoqueAntes-1)
	}
}
