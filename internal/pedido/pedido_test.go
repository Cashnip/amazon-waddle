package pedido

import (
	"maps"
	"testing"
	"time"
)

// Os dois mapas do tempo são listas fechadas: a entrega é uma fila de três
// avanços, a expiração é uma linha só, e o que este teste protege é o "e mais
// nada" — acrescentar uma entrada é o jeito fácil de um passo do tempo passar
// a mexer num Status que não é dele. `avancarUm` lê os mapas direto, então é
// deles que a prova tem de sair.
func TestOsMapasDoTempo(t *testing.T) {
	quero := map[Status]Status{
		StatusPago:      StatusSeparando,
		StatusSeparando: StatusEnviado,
		StatusEnviado:   StatusEntregue,
	}
	if !maps.Equal(simulacao, quero) {
		t.Errorf("simulacao = %v; quero %v", simulacao, quero)
	}
	// A expiração move um Status só, e é o único em que há Tentativa correndo.
	if !maps.Equal(expiracao, map[Status]Status{StatusAguardandoPagamento: StatusPagamentoRecusado}) {
		t.Errorf("expiracao = %v; quero só AGUARDANDO_PAGAMENTO -> PAGAMENTO_RECUSADO", expiracao)
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
	// O mesmo para a expiração, com o ator e o motivo dela: TEMPO_ESGOTADO é
	// motivo reservado, e só a linha da FR-34 o aceita.
	for de, para := range expiracao {
		if _, ok := regra(de, para, AtorVarredura, MotivoTempoEsgotado); !ok {
			t.Errorf("a expiração leva %s a %s, e a tabela não deixa a VARREDURA fazer isso com TEMPO_ESGOTADO", de, para)
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

// O prazo sai do histórico (AD-6): a última entrada *em* AGUARDANDO_PAGAMENTO
// mais o prazo. "Última" importa — a nova Tentativa (5.10) volta a esse Status
// e recomeça o prazo; ler a primeira faria a segunda Tentativa nascer vencida.
// Fora de AGUARDANDO_PAGAMENTO não há prazo correndo.
func TestExpiraEmDerivaDaUltimaEntradaEmAguardando(t *testing.T) {
	const prazo = time.Minute
	nascimento := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	recusa := nascimento.Add(prazo)
	novaTentativa := recusa.Add(10 * time.Second)

	historico := []Transicao{
		{Para: StatusAguardandoPagamento, Ator: AtorComprador, Em: nascimento},
	}
	if tem, ok := ExpiraEm(StatusAguardandoPagamento, historico, prazo); !ok || !tem.Equal(nascimento.Add(prazo)) {
		t.Errorf("recém-criado: ExpiraEm = %v, %v; quero %v", tem, ok, nascimento.Add(prazo))
	}

	historico = append(historico,
		Transicao{De: StatusAguardandoPagamento, Para: StatusPagamentoRecusado, Ator: AtorVarredura, Motivo: MotivoTempoEsgotado, Em: recusa},
	)
	if tem, ok := ExpiraEm(StatusPagamentoRecusado, historico, prazo); ok {
		t.Errorf("recusado: ExpiraEm = %v; fora de AGUARDANDO_PAGAMENTO não há prazo", tem)
	}

	historico = append(historico,
		Transicao{De: StatusPagamentoRecusado, Para: StatusAguardandoPagamento, Ator: AtorComprador, Em: novaTentativa},
	)
	if tem, ok := ExpiraEm(StatusAguardandoPagamento, historico, prazo); !ok || !tem.Equal(novaTentativa.Add(prazo)) {
		t.Errorf("nova Tentativa: ExpiraEm = %v, %v; quero %v", tem, ok, novaTentativa.Add(prazo))
	}

	for _, s := range []Status{StatusPago, StatusSeparando, StatusEnviado, StatusEntregue, StatusCancelado} {
		if tem, ok := ExpiraEm(s, historico, prazo); ok {
			t.Errorf("%s: ExpiraEm = %v; fora de AGUARDANDO_PAGAMENTO não há prazo", s, tem)
		}
	}
}
