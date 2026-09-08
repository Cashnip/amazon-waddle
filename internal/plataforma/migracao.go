package plataforma

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // driver "pgx" do database/sql, para o goose
	"github.com/pressly/goose/v3"
)

// tentativasDeConexao cobre o intervalo entre o healthcheck do Postgres passar
// e ele aceitar conexão de fato. Uma por segundo. Não é limiar do NFR-16 — é
// orçamento de espera por contêiner, e por isso não vira variável AZAMON_.
const tentativasDeConexao = 15

// Migrar aplica as migrações embutidas antes de qualquer requisição ser
// servida. Nunca há auto-migração e nunca se serve com schema errado: todo erro
// daqui derruba o arranque (AD-16, convenção de Migrações).
func Migrar(ctx context.Context, dsn string, arquivos fs.FS, dir string) error {
	goose.SetBaseFS(arquivos)
	goose.SetLogger(goose.NopLogger()) // o log do sistema é o JSON do slog (AD-15)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("dialeto do goose: %w", err)
	}

	// Valida os arquivos antes de tocar no banco: arquivo inválido falha alto.
	// Nenhuma migração é o caso normal enquanto db/migracoes/ está vazio.
	_, err := goose.CollectMigrations(dir, 0, math.MaxInt64)
	vazio := errors.Is(err, goose.ErrNoMigrationFiles)
	if err != nil && !vazio {
		return fmt.Errorf("migrações inválidas: %w", err)
	}

	// A conexão é exigida mesmo sem migração a aplicar: subir alegre com o
	// banco inalcançável só adiaria a descoberta para a primeira consulta.
	bd, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("abrir conexão: %w", err)
	}
	defer bd.Close()

	if err := aguardar(ctx, bd); err != nil {
		return err
	}
	if vazio {
		return nil
	}
	if err := goose.UpContext(ctx, bd, dir); err != nil {
		return fmt.Errorf("aplicar migrações: %w", err)
	}
	return nil
}

func aguardar(ctx context.Context, bd *sql.DB) error {
	var err error
	for i := range tentativasDeConexao {
		if err = bd.PingContext(ctx); err == nil {
			return nil
		}
		if i == tentativasDeConexao-1 {
			break
		}
		// A espera obedece ao contexto: um SIGTERM no meio do arranque não
		// fica preso um segundo por tentativa. `time.After` está fora de
		// questão (AD-6), então quem conta o segundo é o próprio contexto.
		pausa, fim := context.WithTimeout(ctx, time.Second)
		<-pausa.Done()
		fim()
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return fmt.Errorf("banco indisponível após %d tentativas: %w", tentativasDeConexao, err)
}
