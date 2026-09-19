package pedido

import "testing"

// O caminho da entrega é uma fila fechada: três avanços, e mais nada. O que
// este teste protege é o "mais nada" — acrescentar uma quarta entrada ao mapa
// é o jeito fácil de a simulação passar a mexer num Status que não é dela.
func TestProximoDaSimulacao(t *testing.T) {
	avanca := map[Status]Status{
		StatusPago:      StatusSeparando,
		StatusSeparando: StatusEnviado,
		StatusEnviado:   StatusEntregue,
	}
	for de, quero := range avanca {
		proximo, ok := proximoDaSimulacao(de)
		if !ok || proximo != quero {
			t.Errorf("proximoDaSimulacao(%s) = %s, %v; quero %s, true", de, proximo, ok, quero)
		}
	}
	// Nenhum destes é movido pela simulação, por mais tempo que passe.
	for _, parado := range []Status{StatusAguardandoPagamento, StatusPagamentoRecusado, StatusEntregue, StatusCancelado} {
		if proximo, ok := proximoDaSimulacao(parado); ok {
			t.Errorf("proximoDaSimulacao(%s) = %s; a simulação não move este Status", parado, proximo)
		}
	}
}

// Cada passo da simulação é transição legal para o ator dela: um passo fora
// da tabela do AD-3 devolveria TransicaoInvalida em todo tique, e o Pedido
// pararia ali para sempre.
func TestSimulacaoCabeNaTabela(t *testing.T) {
	for de, para := range simulacao {
		if _, ok := regra(de, para, AtorSimulacao, ""); !ok {
			t.Errorf("a simulação leva %s a %s, e a tabela não deixa a SIMULACAO fazer isso", de, para)
		}
	}
}

// Terminal é ENTREGUE ou CANCELADO e mais nada (AD-18). A tela para de
// consultar por esta resposta, então um Status a mais aqui congelaria a
// interface antes do fim do ciclo.
func TestEstadoTerminal(t *testing.T) {
	for _, status := range []Status{StatusEntregue, StatusCancelado} {
		if !EstadoTerminal(status) {
			t.Errorf("EstadoTerminal(%s) = false; quero true", status)
		}
	}
	for _, status := range []Status{StatusAguardandoPagamento, StatusPagamentoRecusado, StatusPago, StatusSeparando, StatusEnviado} {
		if EstadoTerminal(status) {
			t.Errorf("EstadoTerminal(%s) = true; quero false", status)
		}
	}
}
