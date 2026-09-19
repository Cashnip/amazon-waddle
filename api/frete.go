package api

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/pedido"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// saidaFrete é a cotação que a Revisão mostra: as três parcelas, com o total
// já somado pelo Go (NFR-13). O navegador não soma nada.
type saidaFrete struct {
	Regiao           string `json:"regiao"`
	SubtotalCentavos int64  `json:"subtotal_centavos"`
	FreteCentavos    int64  `json:"frete_centavos"`
	TotalCentavos    int64  `json:"total_centavos"`
}

// lerFrete calcula o Frete do Carrinho do dono para um Endereço dele (FR-21).
// Leitura pura, perguntada a cada abertura da Revisão: é isso que faz a troca
// de Endereço recalcular o Frete (FR-20). Congelar o Frete é da criação do
// Pedido (5.6).
func (s *servidor) lerFrete(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	enderecoID := r.URL.Query().Get("endereco_id")
	if enderecoID == "" {
		erro.Escrever(r.Context(), w, erro.ErrEntradaInvalida, nil)
		return
	}
	cotacao, err := pedido.CotarFrete(r.Context(), s.pool, comprador.ID, enderecoID, s.cfg.FreteIsencaoCentavos)
	if err != nil {
		// Endereço de outro Comprador, inexistente e uuid malformado são o
		// mesmo 404, como nas rotas de Endereço: não vaza existência.
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusOK, saidaFrete{
		Regiao:           cotacao.Regiao,
		SubtotalCentavos: cotacao.SubtotalCentavos,
		FreteCentavos:    cotacao.FreteCentavos,
		TotalCentavos:    cotacao.TotalCentavos,
	})
}
