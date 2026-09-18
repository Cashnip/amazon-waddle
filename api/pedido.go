package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// entradaPedido é o corpo da compra. Um Produto, uma unidade: escolha de
// quantidade, Endereço e Frete são das Épicas 4 e 5.
type entradaPedido struct {
	ProdutoID string `json:"produto_id"`
}

// saidaPedido é o que a Página de Produto mostra assim que o Pedido nasce. O
// total sai cru, em centavos int64 (AD-3), e o `numero` é o legível do AD-6 —
// o uuid vai junto porque é dele que a tela de acompanhamento da 1.7 parte.
type saidaPedido struct {
	ID            string `json:"id"`
	Numero        string `json:"numero"`
	Status        string `json:"status"`
	TotalCentavos int64  `json:"total_centavos"`
}

// saidaPedidoDetalhe é o que a tela do Pedido em processamento consulta em
// intervalo. O instante é absoluto e em RFC 3339, vindo do servidor: o
// navegador exibe, e nunca conta. `terminal` vem de pedido.EstadoTerminal, a
// única declaração de "acabou" do sistema (AD-18): é ele que manda a tela
// parar de consultar, e a regra não é redeclarada em JavaScript.
type saidaPedidoDetalhe struct {
	saidaPedido
	AtualizadoEm string `json:"atualizado_em"`
	Terminal     bool   `json:"terminal"`
}

// criarPedido é o primeiro pgx.Tx do repositório: uma transação por caso de
// uso, aberta e fechada aqui, passada adiante como parâmetro nomeado (AD-4).
// `api/` é quem já segura o pool, e um ajudante em `plataforma` seria uma
// camada a mais para embrulhar três chamadas do pgx.
func (s *servidor) criarPedido(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	var entrada entradaPedido
	if err := decodificarCorpo(w, r, &entrada); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	// Rollback depois de um Commit bem-sucedido é no-op no pgx: o que o defer
	// garante é que nenhum caminho de erro deixe a transação aberta.
	// WithoutCancel porque o cliente pode desistir no meio, e desfazer
	// precisa de um contexto vivo — o que sobra é a correlação, para o log.
	defer tx.Rollback(context.WithoutCancel(r.Context()))

	novo, err := pedido.Criar(r.Context(), tx, comprador.ID, entrada.ProdutoID)
	if err != nil {
		// Produto inexistente e identificador malformado são o mesmo 404;
		// nada disso é 500.
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		// A mensagem publicada continua sendo a do sentinela (AD-14); o
		// número que o Comprador precisa ver viaja em `dados`.
		var falta catalogo.EstoqueInsuficiente
		if errors.As(err, &falta) {
			erro.Escrever(r.Context(), w, err, map[string]int64{"disponivel": falta.Disponivel})
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	// WithoutCancel pelo mesmo motivo do Rollback, e com mais razão: um
	// Commit que morre porque o cliente desistiu deixaria o Pedido por
	// gravar e a resposta sairia em 500 sobre trabalho que estava pronto.
	if err := tx.Commit(context.WithoutCancel(r.Context())); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusCreated, saidaPedido{
		ID:            novo.ID,
		Numero:        novo.Numero,
		Status:        novo.Status,
		TotalCentavos: novo.TotalCentavos,
	})
}

// lerPedido é a terceira rota autenticada. Não abre transação: é uma consulta
// só, e o pool é a DBTX que `pedido.Buscar` espera.
func (s *servidor) lerPedido(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	p, err := pedido.Buscar(r.Context(), s.pool, r.PathValue("id"), comprador.ID)
	if err != nil {
		// Pedido de outro Comprador, Pedido inexistente e uuid malformado são
		// o mesmo 404: responder diferente vazaria a existência do Pedido.
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusOK, saidaPedidoDetalhe{
		saidaPedido: saidaPedido{
			ID:            p.ID,
			Numero:        p.Numero,
			Status:        p.Status,
			TotalCentavos: p.TotalCentavos,
		},
		// Nano, e não segundos: duas transições do mesmo Pedido cabem no mesmo
		// segundo, e a tela que compara instantes não teria como distingui-las.
		// RFC3339Nano continua sendo RFC 3339.
		AtualizadoEm: p.AtualizadoEm.Format(time.RFC3339Nano),
		Terminal:     pedido.EstadoTerminal(p.Status),
	})
}

// listarPedidos é a tela "Meus pedidos" (2.6) — o esboço que a Estória 6.1
// substitui: sem filtro por Status, sem paginação, sem Skeleton. O dono entra
// na consulta (AD-11), e não numa checagem depois.
func (s *servidor) listarPedidos(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	pedidos, err := pedido.Listar(r.Context(), s.pool, comprador.ID)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	// Fatia vazia, e nunca `null`: o mesmo contrato de Meus Endereços (2.5) —
	// o estado vazio é da tela.
	saidas := make([]saidaPedido, 0, len(pedidos))
	for _, p := range pedidos {
		saidas = append(saidas, saidaPedido{
			ID:            p.ID,
			Numero:        p.Numero,
			Status:        p.Status,
			TotalCentavos: p.TotalCentavos,
		})
	}
	escreverJSON(w, http.StatusOK, saidas)
}
