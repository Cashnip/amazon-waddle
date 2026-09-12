package pedido

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// O que este teste prova é o ramo que separa "avançou" de "chegou tarde": o
// compare-and-swap afetou zero linhas, e isso é ErrEstadoJaAvancado e não um
// erro de banco. A cláusula WHERE em si é provada contra Postgres de verdade
// em api/pedido_test.go — aqui não sobe contêiner nenhum.
type txComLinhas struct {
	pgx.Tx
	linhas string
}

func (t txComLinhas) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(t.linhas), nil
}

func TestTransicionarEhCompareAndSwap(t *testing.T) {
	ctx := context.Background()

	if err := Transicionar(ctx, txComLinhas{linhas: "UPDATE 1"},
		"018f2b00-0000-7000-8000-000000000000", StatusInicial, "PAGO", "teste"); err != nil {
		t.Fatalf("uma linha afetada = %v; quero nil", err)
	}

	err := Transicionar(ctx, txComLinhas{linhas: "UPDATE 0"},
		"018f2b00-0000-7000-8000-000000000000", StatusInicial, "PAGO", "teste")
	if !errors.Is(err, ErrEstadoJaAvancado) {
		t.Errorf("zero linhas afetadas = %v; quero ErrEstadoJaAvancado", err)
	}
}
