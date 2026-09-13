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

// O caminho da entrega é uma fila fechada: três avanços, e mais nada. O que
// este teste protege é o "mais nada" — acrescentar uma quarta entrada ao mapa
// é o jeito fácil de a simulação passar a mexer num Status que não é dela.
func TestProximoDaSimulacao(t *testing.T) {
	avanca := map[string]string{
		StatusPago:        StatusEmSeparacao,
		StatusEmSeparacao: StatusEnviado,
		StatusEnviado:     StatusEntregue,
	}
	for de, quero := range avanca {
		proximo, ok := proximoDaSimulacao(de)
		if !ok || proximo != quero {
			t.Errorf("proximoDaSimulacao(%s) = %s, %v; quero %s, true", de, proximo, ok, quero)
		}
	}
	// Nenhum destes é movido pela simulação, por mais tempo que passe.
	for _, parado := range []string{StatusInicial, "PAGAMENTO_RECUSADO", StatusEntregue, StatusCancelado} {
		if proximo, ok := proximoDaSimulacao(parado); ok {
			t.Errorf("proximoDaSimulacao(%s) = %s; a simulação não move este Status", parado, proximo)
		}
	}
}

// Terminal é ENTREGUE ou CANCELADO e mais nada (AD-18). A tela para de
// consultar por esta resposta, então um Status a mais aqui congelaria a
// interface antes do fim do ciclo.
func TestEstadoTerminal(t *testing.T) {
	for _, status := range []string{StatusEntregue, StatusCancelado} {
		if !EstadoTerminal(status) {
			t.Errorf("EstadoTerminal(%s) = false; quero true", status)
		}
	}
	for _, status := range []string{StatusInicial, "PAGAMENTO_RECUSADO", StatusPago, StatusEmSeparacao, StatusEnviado} {
		if EstadoTerminal(status) {
			t.Errorf("EstadoTerminal(%s) = true; quero false", status)
		}
	}
}
