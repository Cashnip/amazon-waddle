// Package pagamento é dono da porta do Provedor de Pagamento, do Provedor Simulado e da inbox de confirmação.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1). O módulo NÃO conhece `pedido` (AD-7): quem lê a inbox e
// aplica é a varredura de `pedido`, e a aresta contrária não existe. Também
// não conhece `net/http` — a função que leva a confirmação até o webhook entra
// como parâmetro, e quem a constrói é `cmd/azamon`.
//
// Nada aqui pede, gera ou guarda dado de cartão, nem no Simulado.
package pagamento

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/pagamento/db/gerado"
)

// Os três desfechos da regra do §7.1. SemConfirmacao não é estado no banco: é
// a faixa em que o Provedor simplesmente não responde, e a Tentativa fica
// esperando a expiração — que é da Épica 5.
const (
	Aprovado       = "APROVADO"
	Recusado       = "RECUSADO"
	SemConfirmacao = "SEM_CONFIRMACAO"
)

// Os dois estados terminais da confirmação. Toda linha da inbox chega a um
// deles: aplicada ao Pedido, ou sinalizada como não aplicável.
const (
	Aplicada               = "APLICADA"
	NaoAplicavelSinalizada = "NAO_APLICAVEL_SINALIZADA"
)

// primeiraTentativa é o número da Tentativa que nasce com o Pedido, e o limiar
// de Decidir: a faixa do §7.1 vale só para ela.
const primeiraTentativa = 1

// ErrTetoDeTentativas recusa a Tentativa além do teto do §7.1 (FR-27). É
// daqui, e não de `pedido`, porque o dono da Tentativa é dono do teto (AD-8):
// `pedido` o traduz na recusa da transição em que a pediu, e é essa tradução
// que chega à borda HTTP.
var ErrTetoDeTentativas = errors.New("o teto de Tentativas de Pagamento do Pedido foi atingido")

// proximaTentativa é o número da Tentativa seguinte às `feitas`, ou
// ErrTetoDeTentativas quando ela passaria do teto. Contar e somar um só é
// seguro porque quem chama segura a linha do Pedido: a primeira Tentativa
// nasce no INSERT dele, e a nova, depois do compare-and-swap da transição.
func proximaTentativa(feitas, teto int) (int32, error) {
	if feitas >= teto {
		return 0, ErrTetoDeTentativas
	}
	return int32(feitas + 1), nil
}

// Simulado é o Provedor Simulado: decide pelos centavos do total e mais nada.
// As duas faixas vêm da configuração (AD-13), e não de constante aqui.
type Simulado struct {
	AprovadoAteCentavos int64
	RecusadoAteCentavos int64
}

// Decidir lê a faixa do §7.1 sobre `total_centavos % 100` — é o que "pelos
// centavos do total" quer dizer, e é a leitura que resolve a ambiguidade que a
// 1.1 deixou registrada em `deferred-work.md`: o limiar é o resto dos
// centavos, nunca um valor em reais.
//
// A faixa vale só para a primeira Tentativa (AD-8): da segunda em diante o
// Simulado aprova, senão a nova Tentativa da FR-27 cairia na mesma recusa do
// mesmo total e não teria o que exercitar.
func (s Simulado) Decidir(totalCentavos int64, numero int32) string {
	if numero > primeiraTentativa {
		return Aprovado
	}
	centavos := totalCentavos % 100
	switch {
	case centavos <= s.AprovadoAteCentavos:
		return Aprovado
	case centavos <= s.RecusadoAteCentavos:
		return Recusado
	default:
		return SemConfirmacao
	}
}

// Confirmacao é o que atravessa a rede entre o Provedor e o webhook. Mora
// aqui, e não em `api/`, porque quem emite e quem recebe têm de concordar
// sobre o corpo — duas definições seriam duas verdades. As etiquetas `json`
// não fazem deste módulo um conhecedor de HTTP: quem serializa é quem envia.
type Confirmacao struct {
	IDExterno string `json:"id_externo"`
	Resultado string `json:"resultado"`
}

// Pendente é uma linha da inbox que a varredura de `pedido` ainda não aplicou.
type Pendente struct {
	ID        string
	PedidoID  string
	Resultado string
	// Corrente é falso quando a confirmação pertence a uma Tentativa que já
	// foi superada. Ela não se aplica, mas também não pode ficar PENDENTE para
	// sempre: vira NAO_APLICAVEL_SINALIZADA.
	Corrente bool
}

// Enviar é o transporte da confirmação simulada, injetado por `cmd/azamon`.
type Enviar func(context.Context, Confirmacao) error

// IDExterno é o identificador que o Provedor conhece, derivado do Pedido e do
// número da Tentativa. Determinístico de propósito: é dele que sai a chave de
// idempotência, e um sorteio faria da reemissão um efeito novo em vez de um
// no-op.
func IDExterno(pedidoID string, numero int32) string {
	return fmt.Sprintf("sim-%s-%d", pedidoID, numero)
}

// ChaveIdempotencia é determinística por Tentativa: a mesma confirmação
// recebida duas vezes casa na restrição única e produz um único efeito.
func ChaveIdempotencia(idExterno string) string { return "confirmacao:" + idExterno }

// IniciarTentativa é a primeira das duas operações da porta. Roda dentro da
// transação de quem a pede (AD-4) — a que faz o Pedido nascer, ou a da nova
// Tentativa (FR-27) —, e se ela for revertida a Tentativa vai junto. O total
// chega como valor — o adapter nunca consulta `pedido` para saber quanto
// cobrar —, e o teto também, da configuração (AD-13).
//
// O número é derivado das Tentativas que o Pedido já tem, e o teto é
// conferido aqui: além dele, ErrTetoDeTentativas, sem gravar nada.
func IniciarTentativa(ctx context.Context, tx pgx.Tx, pedidoID string, totalCentavos int64, teto int) error {
	var chave pgtype.UUID
	if err := chave.Scan(pedidoID); err != nil {
		return fmt.Errorf("identificador de Pedido inválido: %w", err)
	}
	q := gerado.New(tx)
	feitas, err := q.ContarTentativas(ctx, chave)
	if err != nil {
		return fmt.Errorf("contar as Tentativas do Pedido: %w", err)
	}
	numero, err := proximaTentativa(int(feitas), teto)
	if err != nil {
		return err
	}
	if err := q.CriarTentativa(ctx, gerado.CriarTentativaParams{
		PedidoID:      chave,
		TotalCentavos: totalCentavos,
		IDExterno:     IDExterno(pedidoID, numero),
		Numero:        numero,
	}); err != nil {
		return fmt.Errorf("criar a Tentativa de Pagamento: %w", err)
	}
	return nil
}

// RegistrarConfirmacao é a segunda operação da porta: a confirmação chega e
// vira linha na inbox, nada mais. Aplicar é da varredura.
//
// Chave sem Tentativa sai como pgx.ErrNoRows, que `api/` traduz em 404 sem
// gravar nada. Chave repetida é no-op: o 23505 da restrição única é sinal para
// decidir aqui, e nunca resposta ao cliente.
func RegistrarConfirmacao(ctx context.Context, bd gerado.DBTX, c Confirmacao) error {
	q := gerado.New(bd)
	tentativa, err := q.BuscarTentativa(ctx, c.IDExterno)
	if err != nil {
		return err
	}
	err = q.GravarConfirmacao(ctx, gerado.GravarConfirmacaoParams{
		TentativaID:       tentativa,
		ChaveIdempotencia: ChaveIdempotencia(c.IDExterno),
		Resultado:         c.Resultado,
	})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return nil
	}
	if err != nil {
		return fmt.Errorf("gravar a confirmação: %w", err)
	}
	return nil
}

// TentativasRestantes diz quantas Tentativas o Pedido ainda pode abrir, com
// piso zero. O teto chega como valor, da configuração (AD-13), mas a conta é
// daqui: `pagamento` é dono da Tentativa e do teto (AD-8), e `pedido` só
// repassa o número à tela (AD-18).
func TentativasRestantes(ctx context.Context, bd gerado.DBTX, pedidoID string, teto int) (int, error) {
	var chave pgtype.UUID
	if err := chave.Scan(pedidoID); err != nil {
		return 0, fmt.Errorf("identificador de Pedido inválido: %w", err)
	}
	feitas, err := gerado.New(bd).ContarTentativas(ctx, chave)
	if err != nil {
		return 0, fmt.Errorf("contar as Tentativas do Pedido: %w", err)
	}
	return max(teto-int(feitas), 0), nil
}

// ConfirmacoesNaoAplicadas é a porta por onde a varredura de `pedido` lê a
// inbox. É esta função que mantém a aresta na direção certa: `pedido` chama
// `pagamento`, e `pagamento` não sabe que `pedido` existe.
func ConfirmacoesNaoAplicadas(ctx context.Context, bd gerado.DBTX) ([]Pendente, error) {
	linhas, err := gerado.New(bd).ConfirmacoesNaoAplicadas(ctx)
	if err != nil {
		return nil, fmt.Errorf("ler a inbox de confirmação: %w", err)
	}
	pendentes := make([]Pendente, 0, len(linhas))
	for _, l := range linhas {
		pendentes = append(pendentes, Pendente{
			ID:        l.ID.String(),
			PedidoID:  l.PedidoID.String(),
			Resultado: l.Resultado,
			Corrente:  l.Corrente,
		})
	}
	return pendentes, nil
}

// TemConfirmacaoPendente é a outra porta por onde `pedido` lê a inbox, e
// existe para a FR-34: antes de expirar, a varredura pergunta se ainda há
// confirmação por aplicar para aquele Pedido. Quem chama já segura a linha do
// Pedido, então a resposta não envelhece entre a pergunta e a transição.
//
// A aresta continua na direção do AD-7: `pedido` pergunta a `pagamento`, e
// `pagamento` segue sem saber que `pedido` existe.
func TemConfirmacaoPendente(ctx context.Context, bd gerado.DBTX, pedidoID string) (bool, error) {
	var chave pgtype.UUID
	if err := chave.Scan(pedidoID); err != nil {
		return false, fmt.Errorf("identificador de Pedido inválido: %w", err)
	}
	tem, err := gerado.New(bd).TemConfirmacaoPendente(ctx, chave)
	if err != nil {
		return false, fmt.Errorf("ler a inbox do Pedido: %w", err)
	}
	return tem, nil
}

// PedidosComAprovacaoSinalizada é a metade de `pagamento` do sinal da FR-26
// (6.5): dos Pedidos pedidos, quais têm uma confirmação APROVADO que a
// varredura sinalizou em vez de aplicar (NAO_APLICAVEL_SINALIZADA), em qualquer
// Tentativa — superada inclusive. A outra metade, o Pedido estar CANCELADO, é
// de `pedido`, que junta as duas em Go: esta porta não sabe o que é Status.
//
// Em lote, e não um `bool` por Pedido: a Tabela do Administrador pergunta pela
// página inteira numa ida só. O mapa vem indexado pelo id canônico — o que o
// Postgres devolve, e o que quem chama já tem quando o id saiu de uma coluna
// uuid —, e só com os Pedidos sinalizados: o ausente lê `false`. Lista vazia
// não vai ao banco.
func PedidosComAprovacaoSinalizada(ctx context.Context, bd gerado.DBTX, pedidoIDs []string) (map[string]bool, error) {
	sinalizados := map[string]bool{}
	if len(pedidoIDs) == 0 {
		return sinalizados, nil
	}
	chaves := make([]pgtype.UUID, 0, len(pedidoIDs))
	for _, id := range pedidoIDs {
		var chave pgtype.UUID
		if err := chave.Scan(id); err != nil {
			return nil, fmt.Errorf("identificador de Pedido inválido: %w", err)
		}
		chaves = append(chaves, chave)
	}
	linhas, err := gerado.New(bd).PedidosComAprovacaoSinalizada(ctx, chaves)
	if err != nil {
		return nil, fmt.Errorf("ler as aprovações sinalizadas dos Pedidos: %w", err)
	}
	for _, l := range linhas {
		sinalizados[l.String()] = true
	}
	return sinalizados, nil
}

// Marcar leva a confirmação ao estado terminal, na mesma transação em que a
// varredura aplicou (ou decidiu não aplicar) o efeito.
func Marcar(ctx context.Context, bd gerado.DBTX, confirmacaoID, estado string) error {
	var chave pgtype.UUID
	if err := chave.Scan(confirmacaoID); err != nil {
		return fmt.Errorf("identificador de confirmação inválido: %w", err)
	}
	if err := gerado.New(bd).MarcarConfirmacao(ctx, gerado.MarcarConfirmacaoParams{
		ID:     chave,
		Estado: estado,
	}); err != nil {
		return fmt.Errorf("marcar a confirmação como %s: %w", estado, err)
	}
	return nil
}

// EmitirConfirmacoesDevidas é o Provedor Simulado respondendo. Não tem relógio
// próprio: quem chama é o tique da varredura, e o que "venceu" significa entra
// como `atraso`. A lista é derivada — Tentativa cujo atraso venceu e que não
// tem linha na inbox —, então reemitir depois de um reinício é normal e a
// chave única faz do reenvio um no-op.
//
// Nenhuma chamada de rede dentro de transação aberta (AD-4): a leitura fecha
// antes de o primeiro envio sair.
//
// Emite o aprovado e o recusado (5.10). A faixa em que a confirmação nunca
// chega não emite nada, de propósito: é ela que a expiração existe para
// resolver (FR-34), e a Tentativa fica na lista até o `prazo` vencer.
//
// O `prazo` é o mesmo da expiração, e fecha o outro lado da janela: passado
// ele a varredura já encerrou o Pedido, e emitir sobre uma Tentativa morta não
// teria efeito. Sem ele a faixa `,95`–`,99` seria relida a cada tique para
// sempre.
func EmitirConfirmacoesDevidas(ctx context.Context, bd gerado.DBTX, s Simulado, atraso, prazo time.Duration, enviar Enviar) error {
	var ate, desde pgtype.Timestamptz
	agora := time.Now()
	if err := ate.Scan(agora.Add(-atraso)); err != nil {
		return fmt.Errorf("calcular o vencimento do atraso: %w", err)
	}
	if err := desde.Scan(agora.Add(-prazo)); err != nil {
		return fmt.Errorf("calcular o piso da janela de emissão: %w", err)
	}
	devidas, err := gerado.New(bd).TentativasSemConfirmacao(ctx, gerado.TentativasSemConfirmacaoParams{
		Ate:   ate,
		Desde: desde,
	})
	if err != nil {
		return fmt.Errorf("listar as Tentativas sem confirmação: %w", err)
	}
	var falhas []error
	for _, t := range devidas {
		resultado := s.Decidir(t.TotalCentavos, t.Numero)
		if resultado == SemConfirmacao {
			continue
		}
		if err := enviar(ctx, Confirmacao{IDExterno: t.IDExterno, Resultado: resultado}); err != nil {
			// Uma Tentativa que falhou não impede as outras: a emissão é
			// derivada, então o próximo tique tenta de novo sozinho.
			falhas = append(falhas, fmt.Errorf("emitir %s: %w", t.IDExterno, err))
		}
	}
	return errors.Join(falhas...)
}
