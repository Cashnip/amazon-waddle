package api

import (
	"context"
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

// saidaPedidoDetalhe é o que a tela do Pedido consulta em intervalo, e carrega
// tudo que ela deriva (AD-18): a tripla (`status`, `tentativas_restantes`,
// `disponivel` por Item), `expira_em`, os valores e o Endereço congelados e o
// histórico. Todo instante é absoluto e em RFC 3339, vindo do servidor: o
// navegador exibe, e nunca conta — uma duração recomeçaria no recarregamento.
// `terminal` vem de pedido.EstadoTerminal, a única declaração de "acabou" do
// sistema: é ele que manda a tela parar de consultar.
//
// Nenhuma chave nomeia a Reserva de Estoque: é conceito de domínio, e a
// EXPERIENCE proíbe expô-lo ao Comprador.
type saidaPedidoDetalhe struct {
	saidaPedido
	AtualizadoEm        string                  `json:"atualizado_em"`
	Terminal            bool                    `json:"terminal"`
	SubtotalCentavos    int64                   `json:"subtotal_centavos"`
	FreteCentavos       int64                   `json:"frete_centavos"`
	ExpiraEm            *string                 `json:"expira_em"`
	TentativasRestantes int                     `json:"tentativas_restantes"`
	Endereco            *saidaEnderecoCongelado `json:"endereco"`
	Itens               []saidaItemPedido       `json:"itens"`
	Historico           []saidaTransicao        `json:"historico"`
}

// saidaEnderecoCongelado é o Endereço copiado no Pedido na criação: cópia, e
// não referência, então sem id. Tem os campos da etiqueta de "Meus endereços",
// mas tipo próprio — reusar o corpo de entrada daquela rota faria uma mudança
// nele alterar esta resposta sem ninguém ver. `null` no Pedido do esqueleto.
type saidaEnderecoCongelado struct {
	Destinatario string `json:"destinatario"`
	CEP          string `json:"cep"`
	Logradouro   string `json:"logradouro"`
	Numero       string `json:"numero"`
	Complemento  string `json:"complemento"`
	Bairro       string `json:"bairro"`
	Cidade       string `json:"cidade"`
	UF           string `json:"uf"`
}

// saidaItemPedido: o preço é o praticado, congelado na compra — nunca o de hoje.
type saidaItemPedido struct {
	ProdutoID              string `json:"produto_id"`
	Nome                   string `json:"nome"`
	Quantidade             int32  `json:"quantidade"`
	PrecoPraticadoCentavos int64  `json:"preco_praticado_centavos"`
	Disponivel             bool   `json:"disponivel"`
}

// saidaTransicao é uma linha do histórico. `de` é null no nascimento e
// `motivo` é null quando a transição não tem um — texto vazio seria um motivo
// sem conteúdo.
type saidaTransicao struct {
	De     *string `json:"de"`
	Para   string  `json:"para"`
	Ator   string  `json:"ator"`
	Motivo *string `json:"motivo"`
	Em     string  `json:"em"`
}

// instante é a única forma de instante nesta resposta. Nano, e não segundos:
// duas transições do mesmo Pedido cabem no mesmo segundo, e a tela que compara
// instantes não teria como distingui-las. RFC3339Nano continua sendo RFC 3339.
func instante(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// textoOuNulo: o vazio do domínio vira `null` no JSON.
func textoOuNulo(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func saidaDoDetalhe(d pedido.Detalhe) saidaPedidoDetalhe {
	saida := saidaPedidoDetalhe{
		saidaPedido:         saidaDoPedido(d.Pedido),
		AtualizadoEm:        instante(d.AtualizadoEm),
		Terminal:            pedido.EstadoTerminal(d.Status),
		SubtotalCentavos:    d.SubtotalCentavos,
		FreteCentavos:       d.FreteCentavos,
		TentativasRestantes: d.TentativasRestantes,
		// Fatias vazias, e nunca `null`: o mesmo contrato das listas.
		Itens:     make([]saidaItemPedido, 0, len(d.Itens)),
		Historico: make([]saidaTransicao, 0, len(d.Historico)),
	}
	if d.ExpiraEm != nil {
		expira := instante(*d.ExpiraEm)
		saida.ExpiraEm = &expira
	}
	if e := d.Endereco; e != nil {
		saida.Endereco = &saidaEnderecoCongelado{
			Destinatario: e.Destinatario, CEP: e.CEP, Logradouro: e.Logradouro, Numero: e.Numero,
			Complemento: e.Complemento, Bairro: e.Bairro, Cidade: e.Cidade, UF: e.UF,
		}
	}
	for _, it := range d.Itens {
		saida.Itens = append(saida.Itens, saidaItemPedido{
			ProdutoID:              it.ProdutoID,
			Nome:                   it.Nome,
			Quantidade:             it.Quantidade,
			PrecoPraticadoCentavos: it.PrecoPraticadoCentavos,
			Disponivel:             it.Disponivel,
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
		}, s.cfg.FreteIsencaoCentavos, s.cfg.PagamentoTentativasMax)
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

// novaTentativa é o "Tentar pagar de novo" da tela do Pedido (FR-27): a
// nova Tentativa de Pagamento a partir do próprio Pedido recusado, numa
// transação só — nova Reserva sobre os Itens do Pedido, e o teto conferido por
// `pagamento` (AD-8). Sem corpo: nada viaja do navegador além do identificador
// na rota, e nada é remontado a partir do Carrinho.
//
// 201 com o Pedido de volta a AGUARDANDO_PAGAMENTO. Recusas, todas 409 e sem
// gravar nada: ESTOQUE_INSUFICIENTE (com o Produto em `dados`),
// TETO_DE_TENTATIVAS e ESTADO_JA_AVANCADO — este último também é o segundo
// clique, que espera o primeiro na linha do Pedido e o encontra já aguardando.
// A tela relê o Pedido nas três. Pedido alheio, inexistente e malformado são o
// mesmo 404 (AD-11).
//
// Sem corpo, como a entrada no checkout: o cookie de Sessão é `SameSite=Lax`,
// então um formulário de outro site que poste aqui chega sem Sessão e sai 401.
func (s *servidor) novaTentativa(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	var p pedido.Pedido
	err = emTransacao(r.Context(), s.pool, func(tx pgx.Tx) (bool, error) {
		var err error
		p, err = pedido.NovaTentativa(r.Context(), tx, r.PathValue("id"), comprador.ID, s.cfg.PagamentoTentativasMax)
		return err == nil, err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		var falta catalogo.EstoqueInsuficiente
		if errors.As(err, &falta) {
			erro.Escrever(r.Context(), w, err, map[string]any{"produto_id": falta.ProdutoID, "disponivel": falta.Disponivel})
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	// A forma curta da criação, e só depois do Commit (a transação pode ser
	// repetida uma vez em impasse). O prazo novo e as Tentativas restantes a
	// tela lê do Detalhe, que ela relê em seguida.
	escreverJSON(w, http.StatusCreated, saidaDoPedido(p))
}

// lerPedido é a tela do Pedido (5.8): a consulta que ela repete a cada 3 s
// enquanto AGUARDANDO_PAGAMENTO e a cada 10 s depois.
//
// O Detalhe são várias consultas — Pedido, histórico, Tentativas, Itens,
// Estoque disponível —, e elas rodam numa transação só leitura em REPEATABLE
// READ: todas veem o mesmo instante, então o Status nunca chega de um lado da
// transição e o histórico (de onde sai `expira_em`) do outro. Só leitura não
// trava ninguém nem sofre falha de serialização; não há o que repetir.
func (s *servidor) lerPedido(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	tx, err := s.pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	defer tx.Rollback(context.WithoutCancel(r.Context()))

	d, err := pedido.Detalhar(r.Context(), tx, r.PathValue("id"), comprador.ID,
		s.cfg.PagamentoTentativaExpiracao, s.cfg.PagamentoTentativasMax)
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
	escreverJSON(w, http.StatusOK, saidaDoDetalhe(d))
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
