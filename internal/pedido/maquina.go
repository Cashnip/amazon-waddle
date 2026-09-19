package pedido

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/pedido/db/gerado"
)

// Status é o Status do Pedido. São sete, e o CHECK de pedido.pedido conhece
// os mesmos sete: um valor fora desta lista não chega ao banco.
type Status string

const (
	StatusAguardandoPagamento Status = "AGUARDANDO_PAGAMENTO"
	StatusPagamentoRecusado   Status = "PAGAMENTO_RECUSADO"
	StatusPago                Status = "PAGO"
	StatusSeparando           Status = "SEPARANDO"
	StatusEnviado             Status = "ENVIADO"
	StatusEntregue            Status = "ENTREGUE"
	StatusCancelado           Status = "CANCELADO"
)

// Ator é quem provocou a transição, como o histórico a grava. São cinco, e o
// CHECK de pedido.transicao_status conhece os mesmos cinco.
type Ator string

const (
	AtorComprador     Ator = "COMPRADOR"
	AtorProvedor      Ator = "PROVEDOR"
	AtorVarredura     Ator = "VARREDURA"
	AtorAdministrador Ator = "ADMINISTRADOR"
	AtorSimulacao     Ator = "SIMULACAO"
)

// MotivoTempoEsgotado é o motivo da expiração da Tentativa de Pagamento
// (FR-34): o caminho é o da recusa, e é o motivo que os distingue.
const MotivoTempoEsgotado = "TEMPO_ESGOTADO"

// ErrEstadoJaAvancado é o desfecho de toda transição que chega tarde: o
// compare-and-swap afetou zero linhas porque outro caminho já moveu o Pedido.
// Não é falha do sistema — a varredura conta com ele como resultado normal.
var ErrEstadoJaAvancado = errors.New("O Pedido já avançou de estado.")

// ErrTransicaoInvalida é a transição que não está na tabela, ou que está mas
// não para este ator. É defeito de quem chamou, e nunca chega ao banco. O
// erro concreto é TransicaoInvalida, que carrega as transições legais.
var ErrTransicaoInvalida = errors.New("Esta mudança de Status não é permitida a partir do Status atual.")

// ErrForaDaJanelaDeCancelamento é o cancelamento de um Pedido que já saiu da
// janela — ENVIADO, ENTREGUE ou já CANCELADO. É também o que uma corrida
// perdida vira quando foi ela que tirou o Pedido da janela (AD-3).
var ErrForaDaJanelaDeCancelamento = errors.New("Este Pedido não pode mais ser cancelado.")

// TransicaoInvalida é o ErrTransicaoInvalida com o dado que a tela usa:
// Permitidas são os destinos legais a partir de De para este Ator — é com
// eles que a FR-32 monta os botões do Administrador.
type TransicaoInvalida struct {
	De, Para   Status
	Ator       Ator
	Permitidas []Status
}

func (e TransicaoInvalida) Error() string { return ErrTransicaoInvalida.Error() }
func (e TransicaoInvalida) Unwrap() error { return ErrTransicaoInvalida }

// efeito é a terceira coluna da tabela do AD-3: o que a transição faz sobre o
// Estoque, na mesma transação e depois do compare-and-swap.
type efeito int

const (
	semEfeito efeito = iota
	efeitoReservar
	efeitoLiberar
	efeitoConsolidar
)

// transicao é uma linha da tabela. `de` é uma lista porque o cancelamento é
// uma transição só com quatro origens; `motivo` preenchido exige exatamente
// ele, e vazio aceita qualquer motivo menos os reservados — TEMPO_ESGOTADO só
// existe na linha da expiração, ou a recusa do Provedor poderia se passar por
// ela e apagar a distinção que a FR-34 faz.
type transicao struct {
	de     []Status
	para   Status
	atores []Ator
	motivo string
	efeito efeito
}

// janelaDeCancelamento são os Status de onde o Comprador cancela (FR-31).
// ENVIADO fica de fora porque é nele que a Reserva consolida: daí em diante
// não há unidade a devolver.
var janelaDeCancelamento = []Status{
	StatusAguardandoPagamento, StatusPagamentoRecusado, StatusPago, StatusSeparando,
}

// transicoes é a tabela do AD-3, exaustiva e obrigatória: nove linhas, na
// ordem da espinha. Nenhuma transição fora dela existe.
//
// A primeira é o nascimento: `de` vazio, porque não havia estado antes. Ela
// não passa pelo compare-and-swap — quem a faz é Criar, com INSERT — e está
// aqui para que a tabela seja a do AD-3 inteira, e não oito nonas dela. O
// efeito dela (Reservar e depois esvaziar o Carrinho) é de Criar.
var transicoes = []transicao{
	{de: nil, para: StatusAguardandoPagamento, atores: []Ator{AtorComprador}},
	{de: []Status{StatusAguardandoPagamento}, para: StatusPago, atores: []Ator{AtorProvedor}},
	{de: []Status{StatusAguardandoPagamento}, para: StatusPagamentoRecusado, atores: []Ator{AtorProvedor}, efeito: efeitoLiberar},
	{de: []Status{StatusAguardandoPagamento}, para: StatusPagamentoRecusado, atores: []Ator{AtorVarredura}, motivo: MotivoTempoEsgotado, efeito: efeitoLiberar},
	{de: []Status{StatusPagamentoRecusado}, para: StatusAguardandoPagamento, atores: []Ator{AtorComprador}, efeito: efeitoReservar},
	{de: []Status{StatusPago}, para: StatusSeparando, atores: []Ator{AtorAdministrador, AtorSimulacao}},
	{de: []Status{StatusSeparando}, para: StatusEnviado, atores: []Ator{AtorAdministrador, AtorSimulacao}, efeito: efeitoConsolidar},
	{de: []Status{StatusEnviado}, para: StatusEntregue, atores: []Ator{AtorAdministrador, AtorSimulacao}},
	{de: janelaDeCancelamento, para: StatusCancelado, atores: []Ator{AtorComprador}, efeito: efeitoLiberar},
}

// regra acha a linha que autoriza a transição. `de` vazio nunca casa: o
// nascimento não é transição que se peça a Transicionar.
func regra(de, para Status, ator Ator, motivo string) (transicao, bool) {
	if de == "" {
		return transicao{}, false
	}
	for _, t := range transicoes {
		if t.para == para && slices.Contains(t.de, de) && slices.Contains(t.atores, ator) &&
			motivoServe(t.motivo, motivo) {
			return t, true
		}
	}
	return transicao{}, false
}

// motivoReservado são os motivos que só a linha que os exige aceita.
var motivoReservado = []string{MotivoTempoEsgotado}

func motivoServe(exigido, motivo string) bool {
	if exigido != "" {
		return motivo == exigido
	}
	return !slices.Contains(motivoReservado, motivo)
}

// Permitidas devolve os destinos legais a partir de `de` para este ator, na
// ordem da tabela e sem repetição. Nunca nil: um Status sem saída serializa
// em `[]`, que é o que a tela exibe como "nenhuma".
func Permitidas(de Status, ator Ator) []Status {
	destinos := []Status{}
	if de == "" {
		return destinos
	}
	for _, t := range transicoes {
		if slices.Contains(t.de, de) && slices.Contains(t.atores, ator) && !slices.Contains(destinos, t.para) {
			destinos = append(destinos, t.para)
		}
	}
	return destinos
}

// foraDaJanela é o Pedido que o Comprador já não cancela. O Status
// desconhecido não está fora da janela: é transição inválida, e não esta.
func foraDaJanela(s Status) bool {
	return s == StatusEnviado || s == StatusEntregue || s == StatusCancelado
}

// EstadoTerminal é a única declaração de "acabou" do sistema (AD-18): terminal
// é ENTREGUE ou CANCELADO, e mais nada — os dois únicos que não são origem de
// linha nenhuma da tabela. A resposta do Pedido carrega o resultado dela para
// que a tela não redeclare a regra em JavaScript.
func EstadoTerminal(s Status) bool {
	return s == StatusEntregue || s == StatusCancelado
}

// Transicionar é o único ponto de mutação do Status (AD-3), e a ordem dentro
// dele é normativa: validar contra a tabela, o compare-and-swap, a linha do
// histórico, e só então o efeito sobre o Estoque — o efeito roda só para
// quem ganhou o CAS, então ninguém libera uma Reserva duas vezes.
//
// Três recusas, nunca uma só:
//   - fora da tabela, ou fora dela para este ator ou motivo: TransicaoInvalida,
//     sem tocar o banco;
//   - zero linhas no CAS: ErrEstadoJaAvancado — outro ator chegou antes;
//   - cancelar fora da janela, inclusive quando foi a corrida que tirou o
//     Pedido de lá: ErrForaDaJanelaDeCancelamento.
//
// O efeito que falha (a nova Tentativa sem Estoque) devolve o erro de
// `catalogo`; o CAS já rodou, e quem desfaz é o rollback de quem abriu a
// transação (AD-4).
func Transicionar(ctx context.Context, tx pgx.Tx, pedidoID string, esperado, novo Status, ator Ator, motivo string) error {
	t, ok := regra(esperado, novo, ator, motivo)
	if !ok {
		if novo == StatusCancelado && ator == AtorComprador && foraDaJanela(esperado) {
			return ErrForaDaJanelaDeCancelamento
		}
		return TransicaoInvalida{De: esperado, Para: novo, Ator: ator, Permitidas: Permitidas(esperado, ator)}
	}

	var chave pgtype.UUID
	if err := chave.Scan(pedidoID); err != nil {
		// Não é ErrEstadoJaAvancado: a varredura trata esse sentinela como
		// desfecho normal e engoliria em silêncio um identificador que quem
		// chamou montou errado.
		return fmt.Errorf("identificador de Pedido inválido: %w", err)
	}
	q := gerado.New(tx)
	linhas, err := q.AvancarStatus(ctx, gerado.AvancarStatusParams{PedidoID: chave, De: string(esperado), Para: string(novo)})
	if err != nil {
		return fmt.Errorf("avançar o Status: %w", err)
	}
	if linhas == 0 {
		return corridaPerdida(ctx, q, chave, novo)
	}

	// O histórico vem depois do avanço, e na mesma transação: o que não
	// aconteceu não é registrado, e o que aconteceu não fica sem linha.
	if err := q.RegistrarTransicao(ctx, gerado.RegistrarTransicaoParams{
		PedidoID:       chave,
		StatusAnterior: string(esperado),
		StatusNovo:     string(novo),
		Ator:           string(ator),
		Motivo:         pgtype.Text{String: motivo, Valid: motivo != ""},
	}); err != nil {
		return fmt.Errorf("registrar a transição: %w", err)
	}

	switch t.efeito {
	case efeitoLiberar:
		// Incondicional (AD-5): sem Reserva ativa, Liberar é no-op e devolve
		// nil — cancelar um Pedido já recusado não pode falhar por isso.
		return catalogo.Liberar(ctx, tx, pedidoID)
	case efeitoConsolidar:
		return catalogo.Consolidar(ctx, tx, pedidoID)
	case efeitoReservar:
		return reservarDeNovo(ctx, q, tx, chave, pedidoID)
	}
	return nil
}

// corridaPerdida separa as duas leituras de um CAS que afetou zero linhas.
// Para o cancelamento, relê o Status: se quem chegou antes tirou o Pedido da
// janela, a resposta é essa, e não "corrida" — a tela explica por que não dá
// mais, em vez de só recarregar (EXPERIENCE, "cancelamento recusado porque o
// estado mudou"). Pedido inexistente continua sendo ErrEstadoJaAvancado.
func corridaPerdida(ctx context.Context, q *gerado.Queries, chave pgtype.UUID, novo Status) error {
	if novo != StatusCancelado {
		return ErrEstadoJaAvancado
	}
	atual, err := q.StatusDoPedido(ctx, chave)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ErrEstadoJaAvancado
	case err != nil:
		return fmt.Errorf("reler o Status: %w", err)
	case foraDaJanela(Status(atual)):
		return ErrForaDaJanelaDeCancelamento
	}
	return ErrEstadoJaAvancado
}

// reservarDeNovo é o efeito da nova Tentativa (FR-27): a Reserva sai dos
// Itens do próprio Pedido, e nada é remontado a partir do Carrinho. A ordem
// de bloqueio do AD-4 se mantém — a linha do Pedido já está presa pelo CAS,
// e `Reservar` trava os Produtos em ordem de id.
func reservarDeNovo(ctx context.Context, q *gerado.Queries, tx pgx.Tx, chave pgtype.UUID, pedidoID string) error {
	linhas, err := q.ItensParaReserva(ctx, chave)
	if err != nil {
		return fmt.Errorf("ler os Itens do Pedido: %w", err)
	}
	// Reservar com lista vazia é no-op (AD-5), e aqui isso devolveria o
	// Pedido a AGUARDANDO_PAGAMENTO sem nada seguro. Pedido sem Item é dado
	// quebrado, e a transição não passa por cima dele.
	if len(linhas) == 0 {
		return fmt.Errorf("o Pedido %s não tem Itens para reservar", pedidoID)
	}
	itens := make([]catalogo.ItemReserva, len(linhas))
	for i, l := range linhas {
		itens[i] = catalogo.ItemReserva{ProdutoID: l.ProdutoID.String(), Quantidade: l.Quantidade}
	}
	return catalogo.Reservar(ctx, tx, pedidoID, itens)
}

// Transicao é uma linha do histórico, como a linha do tempo da FR-30 a
// exibe. De é vazio no nascimento; Motivo é vazio quando não há motivo.
type Transicao struct {
	De, Para Status
	Ator     Ator
	Motivo   string
	Em       time.Time
}

// Historico devolve as transições do Pedido na ordem em que aconteceram — o
// histórico consultável da FR-28. A posse não entra aqui: quem expõe o
// histórico a um Comprador lê o Pedido pelo dono antes (AD-11).
//
// Todo Pedido tem ao menos a linha do nascimento, então histórico vazio é
// Pedido inexistente, e sai como pgx.ErrNoRows — o mesmo do identificador
// malformado, e o mesmo 404 de Buscar.
func Historico(ctx context.Context, bd gerado.DBTX, pedidoID string) ([]Transicao, error) {
	var chave pgtype.UUID
	if err := chave.Scan(pedidoID); err != nil {
		return nil, pgx.ErrNoRows
	}
	linhas, err := gerado.New(bd).HistoricoDoPedido(ctx, chave)
	if err != nil {
		return nil, err
	}
	if len(linhas) == 0 {
		return nil, pgx.ErrNoRows
	}
	historico := make([]Transicao, 0, len(linhas))
	for _, l := range linhas {
		historico = append(historico, Transicao{
			De:     Status(l.StatusAnterior),
			Para:   Status(l.StatusNovo),
			Ator:   Ator(l.Ator),
			Motivo: l.Motivo.String,
			Em:     l.OcorridoEm.Time.UTC(),
		})
	}
	return historico, nil
}
