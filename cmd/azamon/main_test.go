package main

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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
}
