package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/carrinho"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// entradaItemCarrinho é o corpo da adição. A quantidade chega crua: um
// `int` recusaria "1.5" e `"3"` no decodificador, com o 400 genérico, e a
// matriz quer o erro nomeando o campo.
type entradaItemCarrinho struct {
	ProdutoID  string          `json:"produto_id"`
	Quantidade json.RawMessage `json:"quantidade"`
}

// entradaQuantidade é o corpo da alteração, crua pelo mesmo motivo.
type entradaQuantidade struct {
	Quantidade json.RawMessage `json:"quantidade"`
}

type saidaItemCarrinho struct {
	ID         string `json:"id"`
	ProdutoID  string `json:"produto_id"`
	Quantidade int32  `json:"quantidade"`
}

// saidaLinhaCarrinho é a linha do Carrinho aberto. Produto que saiu da
// visibilidade vai com `visivel` falso e nome, imagem, preço e Estoque vazios.
// `preco_mudou` e `bloqueio` são a revalidação da FR-19, decidida em `carrinho`:
// `bloqueio` vem vazio, `indisponivel` ou `acima_do_estoque`.
type saidaLinhaCarrinho struct {
	ID                 string `json:"id"`
	ProdutoID          string `json:"produto_id"`
	Quantidade         int32  `json:"quantidade"`
	Visivel            bool   `json:"visivel"`
	Nome               string `json:"nome"`
	ImagemURL          string `json:"imagem_url"`
	PrecoCentavos      int64  `json:"preco_centavos"`
	PrecoVistoCentavos int64  `json:"preco_visto_centavos"`
	EstoqueDisponivel  int32  `json:"estoque_disponivel"`
	PrecoMudou         bool   `json:"preco_mudou"`
	Bloqueio           string `json:"bloqueio"`
}

type saidaCarrinho struct {
	Itens            []saidaLinhaCarrinho `json:"itens"`
	Unidades         int64                `json:"unidades"`
	SubtotalCentavos int64                `json:"subtotal_centavos"`
}

// quantidadeDaEntrada confere minimo ≤ q ≤ teto e escreve o 400 no campo
// `quantidade` quando falha (NFR-14). O teto vem da Config, e o mínimo é de
// quem chama: 1 na adição, 0 na alteração, onde zero remove.
func quantidadeDaEntrada(w http.ResponseWriter, r *http.Request, bruto json.RawMessage, minimo, teto int) (int, bool) {
	q, err := strconv.Atoi(string(bruto))
	if err != nil || q < minimo || q > teto {
		erro.EscreverCampo(r.Context(), w, "quantidade", fmt.Sprintf("A quantidade vai de %d a %d.", minimo, teto))
		return 0, false
	}
	return q, true
}

// recusaDoCarrinho traduz as recusas de domínio de adicionar e alterar. Devolve
// falso quando o erro é nil — quem chama segue para a resposta de sucesso.
func (s *servidor) recusaDoCarrinho(w http.ResponseWriter, r *http.Request, err error) bool {
	var acima carrinho.AcimaDoEstoque
	switch {
	case err == nil:
		return false
	case errors.Is(err, carrinho.ErrTetoPorItem):
		erro.EscreverCampo(r.Context(), w, "quantidade",
			fmt.Sprintf("O Carrinho aceita no máximo %d unidades de cada Produto.", s.cfg.CarrinhoUnidadesMax))
	case errors.As(err, &acima):
		erro.EscreverEstoqueInsuficiente(r.Context(), w, acima.ProdutoID, acima.Nome, acima.Disponivel, acima.Solicitado)
	case errors.Is(err, pgx.ErrNoRows):
		// Produto invisível, Item alheio, inexistente e uuid malformado: o
		// mesmo 404 — responder diferente vazaria a existência da linha (AD-11).
		erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
	default:
		erro.Escrever(r.Context(), w, err, nil)
	}
	return true
}

// adicionarAoCarrinho é a FR-16/FR-17: o dono vem da Sessão, e o Produto
// repetido soma ao Item existente. O teto é conferido duas vezes — aqui a
// entrada, e dentro da consulta a soma. Acima do Estoque disponível, 409 com o
// número.
func (s *servidor) adicionarAoCarrinho(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	var entrada entradaItemCarrinho
	if err := decodificarCorpo(w, r, &entrada); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	teto := s.cfg.CarrinhoUnidadesMax
	q, ok := quantidadeDaEntrada(w, r, entrada.Quantidade, 1, teto)
	if !ok {
		return
	}
	item, err := carrinho.Adicionar(r.Context(), s.pool, comprador.ID, entrada.ProdutoID, q, teto)
	if s.recusaDoCarrinho(w, r, err) {
		return
	}
	escreverJSON(w, http.StatusCreated, saidaItemCarrinho{ID: item.ID, ProdutoID: item.ProdutoID, Quantidade: item.Quantidade})
}

// alterarItemDoCarrinho é a FR-18: a quantidade absoluta do Item, de 0 ao teto.
// Zero remove o Item — 204, como o DELETE —, e o resto devolve o Item.
func (s *servidor) alterarItemDoCarrinho(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	var entrada entradaQuantidade
	if err := decodificarCorpo(w, r, &entrada); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	q, ok := quantidadeDaEntrada(w, r, entrada.Quantidade, 0, s.cfg.CarrinhoUnidadesMax)
	if !ok {
		return
	}
	if q == 0 {
		if s.recusaDoCarrinho(w, r, carrinho.RemoverItem(r.Context(), s.pool, r.PathValue("id"), comprador.ID)) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	item, err := carrinho.AlterarQuantidade(r.Context(), s.pool, r.PathValue("id"), comprador.ID, q)
	if s.recusaDoCarrinho(w, r, err) {
		return
	}
	escreverJSON(w, http.StatusOK, saidaItemCarrinho{ID: item.ID, ProdutoID: item.ProdutoID, Quantidade: item.Quantidade})
}

// lerCarrinho abre o Carrinho do dono: os Itens, as unidades e o subtotal pelos
// preços de agora (FR-18), com a revalidação da FR-19 em cada linha. Leitura
// pura (AD-17): nada é gravado.
func (s *servidor) lerCarrinho(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	conteudo, err := carrinho.Itens(r.Context(), s.pool, comprador.ID)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	saida := saidaCarrinho{
		Itens:            make([]saidaLinhaCarrinho, len(conteudo.Itens)),
		Unidades:         conteudo.Unidades,
		SubtotalCentavos: conteudo.SubtotalCentavos,
	}
	for i, it := range conteudo.Itens {
		saida.Itens[i] = saidaLinhaCarrinho{
			ID: it.ID, ProdutoID: it.ProdutoID, Quantidade: it.Quantidade, Visivel: it.Visivel,
			Nome: it.Nome, ImagemURL: it.ImagemURL, PrecoCentavos: it.PrecoCentavos,
			PrecoVistoCentavos: it.PrecoVistoCentavos, EstoqueDisponivel: it.EstoqueDisponivel,
			PrecoMudou: it.PrecoMudou, Bloqueio: it.Bloqueio,
		}
	}
	escreverJSON(w, http.StatusOK, saida)
}

// esvaziarOCarrinho é o "Esvaziar" que o Comprador confirma na tela (FR-18).
// A rota fica sob `/carrinho/itens` de propósito: o AD-3 proíbe uma rota para
// esvaziar o Carrinho na criação do Pedido, e esta é outra coisa.
func (s *servidor) esvaziarOCarrinho(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	if err := carrinho.Limpar(r.Context(), s.pool, comprador.ID); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// removerItemDoCarrinho: Item alheio, inexistente e uuid malformado são o mesmo
// 404 — responder diferente vazaria a existência da linha (AD-11).
func (s *servidor) removerItemDoCarrinho(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	if s.recusaDoCarrinho(w, r, carrinho.RemoverItem(r.Context(), s.pool, r.PathValue("id"), comprador.ID)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
