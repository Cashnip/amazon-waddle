package pedido

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Estes testes rodam sem banco. O que provam é a tabela e o que Transicionar
// decide antes e depois do compare-and-swap; o efeito sobre o Estoque, os
// gatilhos e a releitura contra Postgres de verdade estão em
// api/maquina_test.go.

const pedidoDeTeste = "018f2b00-0000-7000-8000-000000000000"

// txFalso responde ao CAS com `linhas` e à releitura do Status com
// `statusAtual` (vazio: o Pedido não existe). Conta cada comando, e é isso que
// prova que a recusa pela tabela nunca chega ao banco.
type txFalso struct {
	pgx.Tx
	linhas      string
	statusAtual string
	comandos    *int
}

func (t txFalso) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	*t.comandos++
	return pgconn.NewCommandTag(t.linhas), nil
}

func (t txFalso) QueryRow(context.Context, string, ...any) pgx.Row {
	*t.comandos++
	return linhaFalsa{t.statusAtual}
}

type linhaFalsa struct{ status string }

func (l linhaFalsa) Scan(destino ...any) error {
	if l.status == "" {
		return pgx.ErrNoRows
	}
	*destino[0].(*string) = l.status
	return nil
}

func novoTx(linhas, statusAtual string) (txFalso, *int) {
	n := 0
	return txFalso{linhas: linhas, statusAtual: statusAtual, comandos: &n}, &n
}

var todosOsStatus = []Status{
	StatusAguardandoPagamento, StatusPagamentoRecusado, StatusPago, StatusSeparando,
	StatusEnviado, StatusEntregue, StatusCancelado,
}

var todosOsAtores = []Ator{AtorComprador, AtorProvedor, AtorVarredura, AtorAdministrador, AtorSimulacao}

// A tabela do AD-3 reescrita à mão, como texto: se alguém acrescentar,
// remover ou alargar uma linha de `transicoes`, este teste é que diz.
var tabelaDoAD3 = []string{
	">AGUARDANDO_PAGAMENTO:COMPRADOR",
	"AGUARDANDO_PAGAMENTO>PAGO:PROVEDOR",
	"AGUARDANDO_PAGAMENTO>PAGAMENTO_RECUSADO:PROVEDOR",
	"AGUARDANDO_PAGAMENTO>PAGAMENTO_RECUSADO:VARREDURA:TEMPO_ESGOTADO",
	"PAGAMENTO_RECUSADO>AGUARDANDO_PAGAMENTO:COMPRADOR",
	"PAGO>SEPARANDO:ADMINISTRADOR,SIMULACAO",
	"SEPARANDO>ENVIADO:ADMINISTRADOR,SIMULACAO",
	"ENVIADO>ENTREGUE:ADMINISTRADOR,SIMULACAO",
	"AGUARDANDO_PAGAMENTO,PAGAMENTO_RECUSADO,PAGO,SEPARANDO>CANCELADO:COMPRADOR",
}

func TestATabelaEhADoAD3(t *testing.T) {
	var tem []string
	for _, tr := range transicoes {
		linha := fmt.Sprintf("%s>%s:%s", juntar(tr.de), tr.para, juntar(tr.atores))
		if tr.motivo != "" {
			linha += ":" + tr.motivo
		}
		tem = append(tem, linha)
	}
	if !slices.Equal(tem, tabelaDoAD3) {
		t.Errorf("tabela =\n%v\nquero\n%v", tem, tabelaDoAD3)
	}
	if len(transicoes) != 9 {
		t.Errorf("%d transições; o AD-3 declara exatamente nove", len(transicoes))
	}
	// O efeito da terceira coluna, linha a linha.
	efeitos := []efeito{semEfeito, semEfeito, efeitoLiberar, efeitoLiberar, efeitoReservar,
		semEfeito, efeitoConsolidar, semEfeito, efeitoLiberar}
	for i, tr := range transicoes {
		if tr.efeito != efeitos[i] {
			t.Errorf("linha %d (%s → %s): efeito = %d, quero %d", i, juntar(tr.de), tr.para, tr.efeito, efeitos[i])
		}
	}
}

// Toda combinação de Status de origem, destino, ator e motivo que não está na
// tabela é recusada com erro explícito, seja qual for o ator (FR-28) — e
// nenhuma delas encosta no banco. As que estão, a tabela acha.
func TestSoATabelaAutoriza(t *testing.T) {
	ctx := context.Background()
	autorizadas := 0
	for _, de := range append([]Status{""}, todosOsStatus...) {
		for _, para := range todosOsStatus {
			for _, ator := range todosOsAtores {
				for _, motivo := range []string{"", "OUTRO", MotivoTempoEsgotado} {
					if declarada(de, para, ator, motivo) {
						autorizadas++
						// A declarada é achada na tabela; o CAS dela está em
						// TestTransicionarEhCompareAndSwap, e o efeito, que
						// lê o banco, em api/maquina_test.go.
						if _, ok := regra(de, para, ator, motivo); !ok {
							t.Errorf("%s → %s por %s (%q) está no AD-3 e a tabela a recusa", de, para, ator, motivo)
						}
						continue
					}
					tx, comandos := novoTx("UPDATE 1", "")
					err := Transicionar(ctx, tx, pedidoDeTeste, de, para, ator, motivo)
					if !errors.Is(err, ErrTransicaoInvalida) && !errors.Is(err, ErrForaDaJanelaDeCancelamento) {
						t.Errorf("%s → %s por %s (%q) = %v; quero recusa explícita", de, para, ator, motivo, err)
					}
					if *comandos != 0 {
						t.Errorf("%s → %s por %s (%q): %d comandos no banco; a recusa pela tabela não chega a ele", de, para, ator, motivo, *comandos)
					}
				}
			}
		}
	}
	// A contagem fecha com a tabela. O nascimento fica fora (não é CAS). Toda
	// linha sem motivo exigido aceita "" e "OUTRO", nunca TEMPO_ESGOTADO: a
	// aprovação (2), a recusa pelo Provedor (2), a nova Tentativa (2), as três
	// etapas com dois atores (3×2×2) e as quatro origens do cancelamento (4×2).
	// A expiração aceita só o motivo dela (1).
	if quer := 2 + 2 + 1 + 2 + 3*2*2 + 4*2; autorizadas != quer {
		t.Errorf("%d combinações autorizadas, quero %d", autorizadas, quer)
	}
}

// declarada lê a tabela do AD-3 em texto, e não `transicoes`: é a segunda
// escrita da mesma regra que torna o teste independente do código.
func declarada(de, para Status, ator Ator, motivo string) bool {
	if de == "" {
		return false // o nascimento não passa por Transicionar
	}
	for _, linha := range tabelaDoAD3 {
		origens, resto, _ := strings.Cut(linha, ">")
		partes := strings.Split(resto, ":")
		destino, atores, exigido := partes[0], partes[1], ""
		if len(partes) == 3 {
			exigido = partes[2]
		}
		motivoCabe := exigido == motivo || (exigido == "" && motivo != MotivoTempoEsgotado)
		if destino == string(para) && slices.Contains(strings.Split(origens, ","), string(de)) &&
			slices.Contains(strings.Split(atores, ","), string(ator)) && motivoCabe {
			return true
		}
	}
	return false
}

func TestPermitidas(t *testing.T) {
	casos := []struct {
		de    Status
		ator  Ator
		quero []Status
	}{
		{StatusPago, AtorAdministrador, []Status{StatusSeparando}},
		{StatusPago, AtorComprador, []Status{StatusCancelado}},
		{StatusAguardandoPagamento, AtorProvedor, []Status{StatusPago, StatusPagamentoRecusado}},
		{StatusAguardandoPagamento, AtorVarredura, []Status{StatusPagamentoRecusado}},
		{StatusPagamentoRecusado, AtorComprador, []Status{StatusAguardandoPagamento, StatusCancelado}},
		{StatusCancelado, AtorAdministrador, []Status{}},
		{StatusEntregue, AtorComprador, []Status{}},
		{"", AtorComprador, []Status{}},
	}
	for _, c := range casos {
		tem := Permitidas(c.de, c.ator)
		if tem == nil || !slices.Equal(tem, c.quero) {
			t.Errorf("Permitidas(%s, %s) = %#v, quero %v", c.de, c.ator, tem, c.quero)
		}
	}
}

// A recusa carrega o dado que a tela usa, e o sentinela continua casando por
// errors.Is — o registro de erro.go traduz pelo sentinela.
func TestTransicaoInvalidaCarregaPermitidas(t *testing.T) {
	tx, _ := novoTx("UPDATE 1", "")
	err := Transicionar(context.Background(), tx, pedidoDeTeste, StatusCancelado, StatusSeparando, AtorAdministrador, "")
	var invalida TransicaoInvalida
	if !errors.As(err, &invalida) || !errors.Is(err, ErrTransicaoInvalida) {
		t.Fatalf("CANCELADO → SEPARANDO = %v; quero TransicaoInvalida", err)
	}
	if invalida.De != StatusCancelado || invalida.Para != StatusSeparando || len(invalida.Permitidas) != 0 {
		t.Errorf("TransicaoInvalida = %+v", invalida)
	}

	// O ator errado para uma transição que existe: também inválida, e as
	// permitidas são as dele.
	err = Transicionar(context.Background(), tx, pedidoDeTeste, StatusPago, StatusSeparando, AtorComprador, "")
	if !errors.As(err, &invalida) || !slices.Equal(invalida.Permitidas, []Status{StatusCancelado}) {
		t.Errorf("PAGO → SEPARANDO pelo Comprador = %+v; quero TransicaoInvalida com [CANCELADO]", err)
	}
}

// O ramo que separa "avançou" de "chegou tarde": o compare-and-swap afetou
// zero linhas, e isso é ErrEstadoJaAvancado e não um erro de banco.
func TestTransicionarEhCompareAndSwap(t *testing.T) {
	ctx := context.Background()

	tx, comandos := novoTx("UPDATE 1", "")
	if err := Transicionar(ctx, tx, pedidoDeTeste, StatusAguardandoPagamento, StatusPago, AtorProvedor, ""); err != nil {
		t.Fatalf("uma linha afetada = %v; quero nil", err)
	}
	if *comandos != 2 {
		t.Errorf("%d comandos; quero o CAS e a linha do histórico", *comandos)
	}

	tx, comandos = novoTx("UPDATE 0", string(StatusPago))
	err := Transicionar(ctx, tx, pedidoDeTeste, StatusAguardandoPagamento, StatusPago, AtorProvedor, "")
	if !errors.Is(err, ErrEstadoJaAvancado) {
		t.Errorf("zero linhas afetadas = %v; quero ErrEstadoJaAvancado", err)
	}
	if *comandos != 1 {
		t.Errorf("%d comandos; quem perdeu o CAS não registra histórico nem relê o Status", *comandos)
	}
}

// Quem perde o CAS não aplica o efeito: é o que impede liberar ou consolidar
// uma Reserva duas vezes. O falso não tem Query, então um Consolidar ou um
// Reservar que rodasse aqui entraria em pânico em vez de passar.
func TestEfeitoSoParaQuemGanhouOCAS(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		de, para Status
		ator     Ator
	}{
		{StatusSeparando, StatusEnviado, AtorSimulacao},                     // Consolidar
		{StatusPagamentoRecusado, StatusAguardandoPagamento, AtorComprador}, // Reservar
		{StatusAguardandoPagamento, StatusPagamentoRecusado, AtorProvedor},  // Liberar
	} {
		tx, comandos := novoTx("UPDATE 0", string(c.para))
		if err := Transicionar(ctx, tx, pedidoDeTeste, c.de, c.para, c.ator, ""); !errors.Is(err, ErrEstadoJaAvancado) {
			t.Errorf("%s → %s com o CAS perdido = %v; quero ErrEstadoJaAvancado", c.de, c.para, err)
		}
		if *comandos != 1 {
			t.Errorf("%s → %s: %d comandos; quem perdeu o CAS não escreve histórico nem efeito", c.de, c.para, *comandos)
		}
	}
}

// TEMPO_ESGOTADO é da expiração, e de mais ninguém: a recusa do Provedor com
// esse motivo seria indistinguível dela no histórico (FR-34).
func TestTempoEsgotadoEhSoDaVarredura(t *testing.T) {
	tx, comandos := novoTx("UPDATE 1", "")
	err := Transicionar(context.Background(), tx, pedidoDeTeste, StatusAguardandoPagamento,
		StatusPagamentoRecusado, AtorProvedor, MotivoTempoEsgotado)
	if !errors.Is(err, ErrTransicaoInvalida) || *comandos != 0 {
		t.Errorf("recusa do Provedor com TEMPO_ESGOTADO = %v (%d comandos); quero TransicaoInvalida sem tocar o banco", err, *comandos)
	}
}

// O cancelamento reclassifica: fora da janela antes do CAS, ou depois dele
// quando a releitura mostra que foi a corrida que tirou o Pedido de lá.
func TestCancelamentoReclassifica(t *testing.T) {
	ctx := context.Background()
	for _, fora := range []Status{StatusEnviado, StatusEntregue, StatusCancelado} {
		tx, comandos := novoTx("UPDATE 1", "")
		if err := Transicionar(ctx, tx, pedidoDeTeste, fora, StatusCancelado, AtorComprador, ""); !errors.Is(err, ErrForaDaJanelaDeCancelamento) {
			t.Errorf("cancelar em %s = %v; quero ErrForaDaJanelaDeCancelamento", fora, err)
		}
		if *comandos != 0 {
			t.Errorf("cancelar em %s tocou o banco", fora)
		}
	}

	casos := []struct {
		atual Status
		quero error
	}{
		{StatusEnviado, ErrForaDaJanelaDeCancelamento},
		{StatusEntregue, ErrForaDaJanelaDeCancelamento},
		{StatusCancelado, ErrForaDaJanelaDeCancelamento},
		{StatusPago, ErrEstadoJaAvancado},
		{"", ErrEstadoJaAvancado}, // o Pedido não existe
	}
	for _, c := range casos {
		tx, _ := novoTx("UPDATE 0", string(c.atual))
		err := Transicionar(ctx, tx, pedidoDeTeste, StatusSeparando, StatusCancelado, AtorComprador, "")
		if !errors.Is(err, c.quero) {
			t.Errorf("CAS perdido com o Pedido em %q = %v; quero %v", c.atual, err, c.quero)
		}
	}

	// Quem não é Comprador não cancela, dentro ou fora da janela: é
	// transição inválida, e não explicação de janela.
	tx, _ := novoTx("UPDATE 1", "")
	if err := Transicionar(ctx, tx, pedidoDeTeste, StatusEnviado, StatusCancelado, AtorAdministrador, ""); !errors.Is(err, ErrTransicaoInvalida) {
		t.Errorf("Administrador cancelando = %v; quero ErrTransicaoInvalida", err)
	}
}

func juntar[T ~string](v []T) string {
	textos := make([]string, len(v))
	for i, x := range v {
		textos[i] = string(x)
	}
	return strings.Join(textos, ",")
}
