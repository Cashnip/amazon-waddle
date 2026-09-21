package api

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// entradaPedido é o corpo da confirmação na Revisão: o Endereço escolhido e o
// total que a Revisão exibiu. Os Itens vêm do Carrinho, e o Frete, da Regra de
// Frete — nada disso viaja do navegador (NFR-13, AD-10). O total vai para que
// o Go recuse com TOTAL_DIVERGENTE o Pedido cujo valor não é mais o que o
// Comprador viu; quem compara é o Go.
type entradaPedido struct {
	EnderecoID    string `json:"endereco_id"`
	TotalCentavos int64  `json:"total_centavos"`
}

// formaDaChave é a forma de uuid do cabeçalho `Idempotency-Key`: a tela gera
// um UUIDv4, e qualquer uuid serve ao banco. Fora dela é 400, antes de abrir
// transação.
var formaDaChave = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// saidaPedido é o que a criação e as leituras devolvem. O total sai cru, em
// centavos int64 (AD-9), e o `numero` é o legível do AD-6 — o uuid vai junto
// porque é dele que a tela do Pedido parte.
type saidaPedido struct {
	ID            string `json:"id"`
	Numero        string `json:"numero"`
	Status        string `json:"status"`
	TotalCentavos int64  `json:"total_centavos"`
}

func saidaDoPedido(p pedido.Pedido) saidaPedido {
	return saidaPedido{ID: p.ID, Numero: p.Numero, Status: string(p.Status), TotalCentavos: p.TotalCentavos}
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

// criarPedido é o Confirmar Pedido da Revisão (FR-23, FR-24, NFR-12): o Pedido
// nasce do Carrinho, com Reserva de Estoque atômica, numa transação só, e a
// `Idempotency-Key` faz da confirmação repetida o mesmo Pedido.
//
// 201 com o Pedido novo; 200 com o original quando a chave já o criou com o
// mesmo corpo — o reenvio **desfaz** a transação, então nada novo é gravado,
// nem o número do contador. O JSON sai só depois do Commit (ou do Rollback
// do reenvio): a transação pode ser repetida uma vez em impasse (emTransacao),
// e uma resposta escrita dentro dela sairia duas vezes.
func (s *servidor) criarPedido(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	chave := r.Header.Get("Idempotency-Key")
	if !formaDaChave.MatchString(chave) {
		erro.Escrever(r.Context(), w, erro.ErrEntradaInvalida, nil)
		return
	}

	var entrada entradaPedido
	if err := decodificarCorpo(w, r, &entrada); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	var novo pedido.Pedido
	var reenvio bool
	err = emTransacao(r.Context(), s.pool, func(tx pgx.Tx) (bool, error) {
		var err error
		novo, reenvio, err = pedido.Criar(r.Context(), tx, comprador.ID, pedido.NovoPedido{
			EnderecoID:    entrada.EnderecoID,
			TotalCentavos: entrada.TotalCentavos,
			Chave:         strings.ToLower(chave),
		}, s.cfg.FreteIsencaoCentavos)
		return err == nil && !reenvio, err
	})
	if err != nil {
		// Endereço de outro Comprador, inexistente e malformado são o mesmo
		// 404 (AD-11); nada disso é 500.
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		// A mensagem publicada continua sendo a do sentinela (AD-14); o
		// Produto que faltou e o disponível dele viajam em `dados`.
		var falta catalogo.EstoqueInsuficiente
		if errors.As(err, &falta) {
			erro.Escrever(r.Context(), w, err, map[string]any{"produto_id": falta.ProdutoID, "disponivel": falta.Disponivel})
			return
		}
		// A chave que já criou um Pedido com outro corpo: a tela mostra esse
		// Pedido e apaga a chave.
		if errors.Is(err, pedido.ErrChaveReutilizada) {
			erro.Escrever(r.Context(), w, err, saidaDoPedido(novo))
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	status := http.StatusCreated
	if reenvio {
		status = http.StatusOK
	}
	escreverJSON(w, status, saidaDoPedido(novo))
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
		saidaPedido: saidaDoPedido(p),
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
		saidas = append(saidas, saidaDoPedido(p))
	}
	escreverJSON(w, http.StatusOK, saidas)
}
