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

	servindo, parar := context.WithCancel(ctx)
	fim := make(chan error, 1)
	go func() { fim <- executar(servindo, io.Discard) }()
	defer func() { parar(); <-fim }()

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

	varreduraLevaOPedidoAPago(t, ctx, conexao)
}

// varreduraLevaOPedidoAPago é o comportamento-título da estória 1.7 medido no
// binário inteiro: sem isto, apagar o `go varrer(...)` de executar() ou trocar
// o caminho do webhook deixaria `go test ./...` verde.
//
// O Pedido e a Tentativa entram direto no banco — o que se exercita aqui é o
// relógio e o transporte, não a compra, que api/ já cobre. O total tem centavos
// em `,00`, que é a faixa que o Simulado aprova.
func varreduraLevaOPedidoAPago(t *testing.T, ctx context.Context, conexao *pgx.Conn) {
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

	// Mesmo estilo de espera do arranque: o tique é de 1 s e o atraso da
	// confirmação, de 100 ms.
	prazo := time.Now().Add(30 * time.Second)
	for {
		var status string
		if err := conexao.QueryRow(ctx,
			`SELECT status FROM pedido.pedido WHERE id = $1::uuid`, pedidoID).Scan(&status); err != nil {
			t.Fatalf("ler o Status: %v", err)
		}
		if status == "PAGO" {
			break
		}
		if time.Now().After(prazo) {
			t.Fatalf("o Pedido ficou em %s; a varredura do binário não o levou a PAGO", status)
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
}
