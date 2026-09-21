package api

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/carrinho"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// entrarNoCheckout é a entrada no checkout (FR-19): revalida o Carrinho do
// dono, devolve o que mudou e, na mesma transação, persiste a ciência dos
// preços que acabou de devolver (AD-17). É a única rota que escreve
// `preco_visto_centavos` sem o Comprador ter alterado o Item.
//
// **A ordem é o contrato**: `pedido.EntrarNoCheckout` lê, monta o relatório e
// só então confirma; o `Commit` vem depois; e o JSON sai **só depois do
// Commit**. Um Commit que falha não reporta nada, e o aviso volta na entrada
// seguinte.
//
// O que isto garante é a ordem **dentro da transação** — nunca gravar uma
// ciência que a resposta não chegou a montar. Ciência gravada com a resposta
// perdida continua possível um nível acima, no HTTP: a conexão pode cair
// depois do Commit e antes do último byte. É perda aceita e conhecida, e é o
// lado barato — quem recarrega vê o preço de agora sem aviso, em vez de
// confirmar um Pedido por um preço que nunca viu.
//
// Sem corpo, de propósito: o cookie de Sessão é `SameSite=Lax`, então um
// `<form>` cross-site não o envia, e o precedente é `DELETE
// /api/v1/carrinho/itens`. Não há nada a pedir ao navegador — o dono vem da
// Sessão (AD-11), e quem decide bloqueio e mudança é o Go (AD-10).
//
// A resposta é **o mesmo envelope** de `GET /api/v1/carrinho`: a tela do passo
// Endereço reaproveita a revalidação que o Carrinho já mostrava. Criar o
// Pedido, congelar o Frete e a `Idempotency-Key` são da 5.6 — esta rota não
// toca em nada disso.
func (s *servidor) entrarNoCheckout(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	// O molde de emTransacao (AD-4), com a repetição única em impasse: duas
	// entradas simultâneas do mesmo Comprador confirmam as mesmas linhas, e o
	// Postgres pode escolher uma delas como vítima. `conteudo` é reescrito a
	// cada passada, então o que sai é o relatório da passada que comitou.
	var conteudo carrinho.Conteudo
	err = emTransacao(r.Context(), s.pool, func(tx pgx.Tx) (bool, error) {
		var err error
		conteudo, err = pedido.EntrarNoCheckout(r.Context(), tx, comprador.ID)
		return err == nil, err
	})
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	// Só aqui: antes do Commit, a resposta prometeria uma ciência que o banco
	// ainda podia recusar.
	escreverJSON(w, http.StatusOK, s.envelopeDoCarrinho(conteudo))
}
