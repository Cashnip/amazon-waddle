package api

import (
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// cabecalhoSegredo carrega o segredo compartilhado com o Provedor. É o que
// autentica a rota: a chave de idempotência não serve para isso, porque é
// derivada do identificador do Pedido — que volta no 201 e vai na URL da tela
// de acompanhamento, ao alcance do próprio Comprador.
const cabecalhoSegredo = "X-Azamon-Segredo"

// saidaConfirmacao é o 200 do webhook. O corpo é mínimo de propósito: quem
// chama é o Provedor, e nada do estado interno do Pedido lhe diz respeito.
type saidaConfirmacao struct {
	Recebida bool `json:"recebida"`
}

// receberConfirmacao é a única rota que escreve sem Sessão. Quem a autentica é
// o segredo compartilhado no cabeçalho, comparado em tempo constante; sem ele
// o Comprador forjaria a própria confirmação e levaria o Pedido a `PAGO` sem
// pagar, porque o identificador de que a chave de idempotência é derivada
// volta no 201 da compra.
//
// A mesma confirmação recebida duas vezes sai 200 nas duas, com uma linha só
// na inbox: o 23505 da restrição única é absorvido por `pagamento`, e nunca
// vaza para o cliente.
func (s *servidor) receberConfirmacao(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	// Antes de ler o corpo: cabeçalho ausente e cabeçalho errado saem no mesmo
	// 401, sem dizer qual dos dois foi, e nada é gravado.
	if subtle.ConstantTimeCompare([]byte(r.Header.Get(cabecalhoSegredo)), []byte(s.cfg.WebhookSegredo)) != 1 {
		erro.Escrever(r.Context(), w, erro.ErrNaoAutorizado, nil)
		return
	}

	var c pagamento.Confirmacao
	if err := decodificarCorpo(w, r, &c); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	// Validação na fronteira de confiança: sem isto um resultado desconhecido
	// só falharia lá no CHECK da migração, e um 400 sairia como 500.
	if c.Resultado != pagamento.Aprovado && c.Resultado != pagamento.Recusado {
		erro.Escrever(r.Context(), w, erro.ErrEntradaInvalida, nil)
		return
	}

	if err := pagamento.RegistrarConfirmacao(r.Context(), s.pool, c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusOK, saidaConfirmacao{Recebida: true})
}
