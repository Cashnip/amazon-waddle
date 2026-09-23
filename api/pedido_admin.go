package api

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/pedido"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// A área administrativa de Pedidos (6.4, FR-32): a Tabela filtrável e
// ordenável, o Detalhe de qualquer Pedido e as três transições que cabem ao
// Administrador. Todas as rotas moram no mux `admin` e herdam a guarda do
// prefixo — não há `if` de papel aqui dentro.
//
// Nenhuma rota de cancelamento: o Administrador não cancela (FR-32). Isso não
// é conferido por um `if` — sai da própria tabela do AD-3, que não tem linha
// de AtorAdministrador para CANCELADO, e por isso `{para: CANCELADO}` chega
// aqui e sai TRANSICAO_INVALIDA como qualquer outra transição inexistente.

// saidaPedidoAdmin é a linha da Tabela de Pedidos. Carrega `permitidas` pronto
// — os destinos que o Administrador alcança a partir do Status atual, tirados
// de pedido.Permitidas com AtorAdministrador —, e é dela que os botões são
// montados: a tela nunca redeclara a tabela de transições em JavaScript
// (AD-10), como `pode_cancelar` do Comprador (6.3).
//
// `terminal` vem de pedido.EstadoTerminal, a única declaração de "acabou" do
// sistema: é ele que manda a lista parar de consultar em intervalo.
type saidaPedidoAdmin struct {
	saidaPedido
	Terminal   bool     `json:"terminal"`
	Permitidas []string `json:"permitidas"`
}

// saidaPedidoAdminDetalhe é o Detalhe administrativo. Não é
// `saidaPedidoDetalhe`: aquela carrega `expira_em`, `tentativas_restantes`,
// `disponivel` por Item e `pode_cancelar`, que são da decisão do Comprador. O
// que esta tem a mais é o Comprador e o Vendedor congelado de cada Item — o
// que o Administrador precisa para operar a loja.
type saidaPedidoAdminDetalhe struct {
	saidaPedidoAdmin
	AtualizadoEm     string                  `json:"atualizado_em"`
	Comprador        saidaCompradorDoPedido  `json:"comprador"`
	SubtotalCentavos int64                   `json:"subtotal_centavos"`
	FreteCentavos    int64                   `json:"frete_centavos"`
	Endereco         *saidaEnderecoCongelado `json:"endereco"`
	Itens            []saidaItemPedidoAdmin  `json:"itens"`
	Historico        []saidaTransicao        `json:"historico"`
}

// saidaCompradorDoPedido é quem comprou, como a Tabela do Administrador o
// mostra: nome e e-mail, e nada de credencial. Campos vazios quando o Pedido é
// o do esqueleto, cujo dono não é uma Conta de verdade.
type saidaCompradorDoPedido struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

// saidaItemPedidoAdmin: o preço e o Vendedor são os congelados na compra —
// nunca os de hoje. Sem `disponivel`: o Estoque de agora não entra em decisão
// nenhuma desta tela.
type saidaItemPedidoAdmin struct {
	ProdutoID              string `json:"produto_id"`
	Nome                   string `json:"nome"`
	VendedorNome           string `json:"vendedor_nome"`
	Quantidade             int32  `json:"quantidade"`
	PrecoPraticadoCentavos int64  `json:"preco_praticado_centavos"`
}

// permitidasDe é o ponto único de onde saem os destinos do Administrador, nas
// duas respostas. Nunca `null`: o contrato é a lista, vazia quando o Status não
// tem saída nenhuma para este ator.
func permitidasDe(s pedido.Status) []string {
	destinos := pedido.Permitidas(s, pedido.AtorAdministrador)
	saida := make([]string, 0, len(destinos))
	for _, d := range destinos {
		saida = append(saida, string(d))
	}
	return saida
}

func saidaAdminDoPedido(p pedido.Pedido) saidaPedidoAdmin {
	return saidaPedidoAdmin{
		saidaPedido: saidaDoPedido(p),
		Terminal:    pedido.EstadoTerminal(p.Status),
		Permitidas:  permitidasDe(p.Status),
	}
}

func saidaAdminDoDetalhe(d pedido.DetalheAdmin) saidaPedidoAdminDetalhe {
	saida := saidaPedidoAdminDetalhe{
		saidaPedidoAdmin: saidaAdminDoPedido(d.Pedido),
		AtualizadoEm:     instante(d.AtualizadoEm),
		Comprador: saidaCompradorDoPedido{
			ID: d.Comprador.ID, Nome: d.Comprador.Nome, Email: d.Comprador.Email,
		},
		SubtotalCentavos: d.SubtotalCentavos,
		FreteCentavos:    d.FreteCentavos,
		// Fatias vazias, e nunca `null`: o mesmo contrato das listas.
		Itens:     make([]saidaItemPedidoAdmin, 0, len(d.Itens)),
		Historico: make([]saidaTransicao, 0, len(d.Historico)),
	}
	if e := d.Endereco; e != nil {
		saida.Endereco = &saidaEnderecoCongelado{
			Destinatario: e.Destinatario, CEP: e.CEP, Logradouro: e.Logradouro, Numero: e.Numero,
			Complemento: e.Complemento, Bairro: e.Bairro, Cidade: e.Cidade, UF: e.UF,
		}
	}
	for _, it := range d.Itens {
		saida.Itens = append(saida.Itens, saidaItemPedidoAdmin{
			ProdutoID:              it.ProdutoID,
			Nome:                   it.Nome,
			VendedorNome:           it.VendedorNome,
			Quantidade:             it.Quantidade,
			PrecoPraticadoCentavos: it.PrecoPraticadoCentavos,
		})
	}
	for _, t := range d.Historico {
		saida.Historico = append(saida.Historico, saidaTransicao{
			De:     textoOuNulo(string(t.De)),
			Para:   string(t.Para),
			Ator:   string(t.Ator),
			Motivo: textoOuNulo(t.Motivo),
			Em:     instante(t.Em),
		})
	}
	return saida
}

// filtroAdminDe lê `status` e `ordenacao` da URL, cada um contra a sua lista
// fechada — a dos sete Status é a da própria máquina (`pedido.Statuses`), e
// não uma oitava cópia aqui. Ausente é "sem filtro" e "recentes". Fora da
// lista escreve o erro em linha e devolve ok falso, no molde de `api/busca.go`.
func filtroAdminDe(w http.ResponseWriter, r *http.Request) (pedido.FiltroAdmin, bool) {
	var f pedido.FiltroAdmin
	q := r.URL.Query()
	if s := q.Get("status"); s != "" {
		if !slices.Contains(pedido.Statuses, pedido.Status(s)) {
			nomes := make([]string, 0, len(pedido.Statuses))
			for _, v := range pedido.Statuses {
				nomes = append(nomes, string(v))
			}
			erro.EscreverCampo(r.Context(), w, "status", "O Status é um destes: "+strings.Join(nomes, ", ")+".")
			return f, false
		}
		f.Status = pedido.Status(s)
	}
	switch f.Ordenacao = q.Get("ordenacao"); f.Ordenacao {
	case "", pedido.OrdenacaoRecentes, pedido.OrdenacaoAntigos:
	default:
		erro.EscreverCampo(r.Context(), w, "ordenacao", "A ordenação é recentes ou antigos.")
		return f, false
	}
	return f, true
}

// listarPedidosAdmin é a Tabela de Pedidos (FR-32), no envelope único do AD-18:
// todos os Compradores, filtrável por Status e ordenável por data. O filtro e a
// ordenação são conferidos antes da paginação, e os dois vivem na URL — a tela
// recarregada reproduz a mesma lista.
func (s *servidor) listarPedidosAdmin(w http.ResponseWriter, r *http.Request) {
	semCache(w)
	filtro, ok := filtroAdminDe(w, r)
	if !ok {
		return
	}
	pagina, porPagina, ok := paginacaoDe(w, r, s.cfg)
	if !ok {
		return
	}
	pedidos, total, err := pedido.ListarParaAdministrador(r.Context(), s.pool, filtro, pagina, porPagina)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	itens := make([]saidaPedidoAdmin, 0, len(pedidos))
	for _, p := range pedidos {
		itens = append(itens, saidaAdminDoPedido(p))
	}
	escreverJSON(w, http.StatusOK, listagem[saidaPedidoAdmin]{itens, pagina, porPagina, total})
}

// lerPedidoAdmin é o Detalhe administrativo, consultado em intervalo como o do
// Comprador. As várias consultas rodam numa transação só leitura em REPEATABLE
// READ: todas veem o mesmo instante, então o Status nunca chega de um lado da
// transição e o histórico do outro. Pedido inexistente e uuid malformado são o
// mesmo 404 — não há dono a proteger aqui, mas a resposta é a de sempre.
func (s *servidor) lerPedidoAdmin(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	tx, err := s.pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	defer tx.Rollback(context.WithoutCancel(r.Context()))

	d, err := pedido.DetalharParaAdministrador(r.Context(), tx, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusOK, saidaAdminDoDetalhe(d))
}

// transicionarPedidoAdmin é o botão da Tabela e do Detalhe (FR-32): o corpo
// traz `de` e `para`, e a transição passa por pedido.Transicionar como toda
// mudança de Status (AD-3) — o efeito sobre o Estoque é dele, e é por isso que
// `SEPARANDO → ENVIADO` consolida sem uma linha a mais aqui.
//
// `de` vem da TELA, e é isso que faz o compare-and-swap distinguir os dois
// desfechos: lido no servidor, o Pedido que a simulação já moveu sairia como
// transição inválida, que é causa falsa. As duas recusas são 409 e não gravam
// nada: TRANSICAO_INVALIDA leva `dados.permitidas` (preenchido em
// erro.Escrever, num ponto só), e ESTADO_JA_AVANCADO leva em `dados.status` o
// Status lido sob a trava — o que o Pedido tem agora.
//
// 200 com a linha nova e `permitidas` já recalculado do Status novo, para quem
// chama a API direto ter o desfecho completo numa resposta só (AD-18). A tela
// não o usa para remontar os botões: nas três saídas — o 200 e as duas recusas
// — ela relê a superfície, que é o que traz também a linha do histórico.
func (s *servidor) transicionarPedidoAdmin(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	var entrada struct {
		De   string `json:"de"`
		Para string `json:"para"`
	}
	if err := decodificarCorpo(w, r, &entrada); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	var p pedido.Pedido
	err := emTransacao(r.Context(), s.pool, func(tx pgx.Tx) (bool, error) {
		var err error
		p, err = pedido.TransicionarPeloAdministrador(r.Context(), tx,
			r.PathValue("id"), pedido.Status(entrada.De), pedido.Status(entrada.Para))
		return err == nil, err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		// O Status lido sob a trava: a recusa não gravou nada, então é o
		// Status em que o Pedido está. É ele que a tela nomeia no Alert
		// informativo, em vez de dizer "transição inválida".
		if errors.Is(err, pedido.ErrEstadoJaAvancado) {
			erro.Escrever(r.Context(), w, err, map[string]any{"status": string(p.Status)})
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	// Só depois do Commit: a transação pode ser repetida uma vez em impasse.
	escreverJSON(w, http.StatusOK, saidaAdminDoPedido(p))
}
