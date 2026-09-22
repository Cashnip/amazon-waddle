package pagamento

import (
	"errors"
	"testing"
)

// A regra do §7.1 é lida sobre os centavos do total, e não sobre o total: um
// Pedido de R$ 1.234,50 decide pelo `50`. Este teste é a fronteira das três
// faixas, que é onde o erro de `<` contra `<=` mora.
func TestSimuladoDecidePelosCentavosDoTotal(t *testing.T) {
	s := Simulado{AprovadoAteCentavos: 89, RecusadoAteCentavos: 94}

	for _, caso := range []struct {
		total int64
		quer  string
	}{
		{100, Aprovado},    // ,00 — o piso da faixa aprovada
		{24990, Recusado},  // ,90 — o Produto semeado de R$ 249,90 não atravessa
		{123489, Aprovado}, // ,89 — o teto da faixa aprovada
		{1090, Recusado},   // ,90 — o piso da recusa
		{1094, Recusado},   // ,94 — o teto da recusa
		{1095, SemConfirmacao},
		{1099, SemConfirmacao},
	} {
		if tem := s.Decidir(caso.total, 1); tem != caso.quer {
			t.Errorf("Decidir(%d, 1) = %s, quero %s", caso.total, tem, caso.quer)
		}
	}
}

// A faixa do §7.1 vale só para a primeira Tentativa (AD-8). Da segunda em
// diante o Simulado aprova — inclusive nos centavos que recusam e nos que
// nunca confirmam —, senão a nova Tentativa da FR-27 repetiria a recusa do
// mesmo total para sempre.
func TestSimuladoAprovaDaSegundaTentativaEmDiante(t *testing.T) {
	s := Simulado{AprovadoAteCentavos: 89, RecusadoAteCentavos: 94}

	for _, total := range []int64{100, 1090, 1094, 1095, 1099} {
		for _, numero := range []int32{2, 3} {
			if tem := s.Decidir(total, numero); tem != Aprovado {
				t.Errorf("Decidir(%d, %d) = %s, quero %s", total, numero, tem, Aprovado)
			}
		}
	}
}

// O número da Tentativa sai das que o Pedido já teve, e o teto do §7.1 é
// conferido antes de gravar: com 3, a quarta é recusada — e não numerada.
func TestProximaTentativaRespeitaOTeto(t *testing.T) {
	for _, caso := range []struct {
		feitas, teto int
		quer         int32
	}{
		{0, 3, 1},
		{1, 3, 2},
		{2, 3, 3},
	} {
		numero, err := proximaTentativa(caso.feitas, caso.teto)
		if err != nil || numero != caso.quer {
			t.Errorf("proximaTentativa(%d, %d) = %d, %v; quero %d", caso.feitas, caso.teto, numero, err, caso.quer)
		}
	}
	for _, feitas := range []int{3, 4} {
		if _, err := proximaTentativa(feitas, 3); !errors.Is(err, ErrTetoDeTentativas) {
			t.Errorf("proximaTentativa(%d, 3) = %v; quero ErrTetoDeTentativas", feitas, err)
		}
	}
}

// A chave é determinística por Tentativa: é ela que transforma a confirmação
// repetida — e a reemissão depois de um reinício — em um único efeito. Um
// sorteio aqui faria a restrição única não encontrar nada com que colidir.
func TestChaveDeIdempotenciaEhDeterministicaPorTentativa(t *testing.T) {
	const pedido = "018f2b00-0000-7000-8000-000000000000"

	primeira := IDExterno(pedido, 1)
	if primeira != IDExterno(pedido, 1) {
		t.Error("o identificador externo mudou entre duas chamadas iguais")
	}
	if primeira == IDExterno(pedido, 2) {
		t.Error("duas Tentativas do mesmo Pedido têm o mesmo identificador externo")
	}
	if ChaveIdempotencia(primeira) != ChaveIdempotencia(primeira) {
		t.Error("a chave de idempotência mudou entre duas chamadas iguais")
	}
	if ChaveIdempotencia(primeira) == ChaveIdempotencia(IDExterno(pedido, 2)) {
		t.Error("duas Tentativas compartilham a chave de idempotência")
	}
}
