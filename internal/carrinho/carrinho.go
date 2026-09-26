// Package carrinho é dono de Carrinho e Item de Carrinho.
//
// Este arquivo abre a interface pública do módulo, mas a porta é o pacote: o
// que os arquivos irmãos exportam também é interface (AD-1). O dono vem sempre
// de fora, como texto: `carrinho` não conhece `identidade` (AD-11), e quem
// resolve a Sessão é `api/`.
package carrinho

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/carrinho/db/gerado"
	"github.com/Cashnip/amazon-waddle/internal/catalogo"
)

// ErrTetoPorItem recusa a adição cuja soma passaria do teto por Item (FR-17).
// Quem conhece o teto é quem chama, e é ele que o nomeia na mensagem.
var ErrTetoPorItem = errors.New("a quantidade passaria do teto por Item")

// AcimaDoEstoque recusa a quantidade que o Item passaria a pedir acima do
// Estoque disponível (FR-17). Embrulha catalogo.ErrEstoqueInsuficiente, então
// `erro` a traduz no mesmo 409 do Pedido; `api/` lê os números por errors.As e
// os publica em `dados`. É a recusa de um conselho (AD-5): quem decide de
// verdade é Reservar, sob bloqueio, na criação do Pedido.
type AcimaDoEstoque struct {
	ProdutoID  string
	Nome       string
	Disponivel int
	// Solicitado é a quantidade que o Item passaria a ter — a soma, e não só o
	// acréscimo.
	Solicitado int
}

func (e AcimaDoEstoque) Error() string { return catalogo.ErrEstoqueInsuficiente.Error() }
func (e AcimaDoEstoque) Unwrap() error { return catalogo.ErrEstoqueInsuficiente }

// ErrCarrinhoMudou recusa a escrita de fora que encontrou menos Itens do que
// leu: um Item removido noutra aba entre a leitura e a escrita, em READ
// COMMITTED. É erro, e nunca silêncio — quem chama desfaz a transação inteira,
// e o Comprador confere o Carrinho de novo. Sai de Esvaziar (criação do
// Pedido) e da escrita parcial de ConfirmarPrecoVisto (entrada no checkout).
var ErrCarrinhoMudou = errors.New("O Carrinho mudou. Abra-o de novo e confira antes de continuar.")

// Item é o Item de Carrinho como a adição o devolve.
type Item struct {
	ID         string
	ProdutoID  string
	Quantidade int32
}

// O que impede o Item de seguir para o Pedido (FR-19). É o Go que decide, e a
// tela só mostra: a 5.5 reaproveita a mesma regra na entrada do checkout.
const (
	// BloqueioIndisponivel: o Produto saiu da visibilidade (AD-19) — desativado,
	// ou de Vendedor desativado, que a interface não distingue (FR-12) — ou não
	// tem nenhuma unidade disponível. Só remover resolve.
	BloqueioIndisponivel = "indisponivel"
	// BloqueioAcimaDoEstoque: há disponível, mas menos que a quantidade do Item.
	// Remover ou ajustar para o disponível resolve.
	BloqueioAcimaDoEstoque = "acima_do_estoque"
)

// ItemDoCarrinho é uma linha do Carrinho aberto. Um Produto que saiu da
// visibilidade (AD-19) volta com Visivel falso e sem nome, imagem, preço nem
// Estoque: a tela o mostra como indisponível e o Comprador ainda pode removê-lo.
type ItemDoCarrinho struct {
	ID            string
	ProdutoID     string
	Quantidade    int32
	Visivel       bool
	Nome          string
	ImagemURL     string
	PrecoCentavos int64
	// PrecoVistoCentavos é o preço da última alteração do Item: o "de" do "de X
	// para Y". Só se lê aqui (AD-17).
	PrecoVistoCentavos int64
	// EstoqueDisponivel é conselho (AD-5), o mesmo que a Caixa de compra mostra.
	EstoqueDisponivel int32
	// PrecoMudou: o preço de agora difere do visto. Exige confirmação (FR-19).
	PrecoMudou bool
	// Bloqueio vem vazio quando nada impede o Item; senão, um dos Bloqueio*.
	Bloqueio string
}

// bloqueioDe é a regra da FR-19: Produto invisível ou sem nenhuma unidade é
// indisponível, e o disponível abaixo da quantidade é acima do Estoque.
func bloqueioDe(visivel bool, disponivel, quantidade int32) string {
	switch {
	case !visivel || disponivel <= 0:
		return BloqueioIndisponivel
	case disponivel < quantidade:
		return BloqueioAcimaDoEstoque
	}
	return ""
}

// Conteudo é o Carrinho aberto. O subtotal soma só os Itens visíveis, pelos
// preços de agora (FR-18); as unidades somam todos os Itens, porque todos
// aparecem como linha.
type Conteudo struct {
	Itens            []ItemDoCarrinho
	Unidades         int64
	SubtotalCentavos int64
}

// Adicionar põe q unidades do Produto no Carrinho do Comprador, criando o
// Carrinho na primeira vez. O Produto repetido soma ao Item existente.
//
// Visibilidade, preço e Estoque disponível vêm só de catalogo.BuscarProduto
// (AD-19): Produto invisível, inexistente ou de uuid malformado sai como
// pgx.ErrNoRows. A soma acima do teto sai como ErrTetoPorItem, e acima do
// Estoque disponível como AcimaDoEstoque — nessa ordem, porque o teto vale sem
// olhar o Estoque. Nas duas recusas nada grava, nem o Carrinho. O teto da
// entrada (1 ≤ q ≤ teto) é de quem chama, na fronteira (NFR-14).
//
// A soma é lida e depois gravada, em duas idas: o Estoque é conselho e a
// divergência entre o lido e o aceito é projetada (AD-5), mas o teto continua
// atômico, porque a consulta de gravação o confere de novo.
//
// Garantir o Carrinho e gravar o Item são duas idas sem transação: se a
// segunda falhar, sobra um Carrinho vazio — que é o estado de todo Comprador
// que ainda não adicionou nada.
func Adicionar(ctx context.Context, bd gerado.DBTX, compradorID, produtoID string, q, teto int) (Item, error) {
	dono, err := uuidDe(compradorID)
	if err != nil {
		return Item{}, err
	}
	produto, err := catalogo.BuscarProduto(ctx, bd, produtoID)
	if err != nil {
		return Item{}, err
	}
	var chave pgtype.UUID
	if err := chave.Scan(produto.ID); err != nil {
		return Item{}, err
	}
	consultas := gerado.New(bd)
	atual, err := consultas.QuantidadeDoProduto(ctx, gerado.QuantidadeDoProdutoParams{CompradorID: dono, ProdutoID: chave})
	if err != nil {
		return Item{}, err
	}
	soma := int(atual) + q
	if soma > teto {
		return Item{}, ErrTetoPorItem
	}
	if soma > int(produto.EstoqueDisponivel) {
		return Item{}, AcimaDoEstoque{
			ProdutoID: produto.ID, Nome: produto.Nome, Disponivel: int(produto.EstoqueDisponivel), Solicitado: soma,
		}
	}
	carrinhoID, err := consultas.GarantirCarrinho(ctx, dono)
	if err != nil {
		return Item{}, err
	}
	linha, err := consultas.AdicionarItem(ctx, gerado.AdicionarItemParams{
		CarrinhoID:         carrinhoID,
		ProdutoID:          chave,
		Quantidade:         int32(q),
		PrecoVistoCentavos: produto.PrecoCentavos,
		Teto:               int32(teto),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, ErrTetoPorItem
	}
	if err != nil {
		return Item{}, err
	}
	return Item{ID: linha.ID.String(), ProdutoID: linha.ProdutoID.String(), Quantidade: linha.Quantidade}, nil
}

// AlterarQuantidade põe o Item do dono em q unidades, com q ≥ 1: zero remove, e
// quem chama decide isso chamando RemoverItem. O teto de q é de quem chama, na
// fronteira (NFR-14); o Estoque é conferido aqui, como em Adicionar, contra o
// que o Item passaria a pedir. O preço visto passa a ser o de agora.
//
// A posse está nas duas consultas (AD-11). Item alheio, inexistente, de uuid
// malformado ou de Produto que saiu da visibilidade saem como pgx.ErrNoRows.
func AlterarQuantidade(ctx context.Context, bd gerado.DBTX, itemID, compradorID string, q int) (Item, error) {
	chave, err := uuidDe(itemID)
	if err != nil {
		return Item{}, err
	}
	dono, err := uuidDe(compradorID)
	if err != nil {
		return Item{}, err
	}
	consultas := gerado.New(bd)
	do, err := consultas.BuscarItem(ctx, gerado.BuscarItemParams{ID: chave, CompradorID: dono})
	if err != nil {
		return Item{}, err
	}
	produto, err := catalogo.BuscarProduto(ctx, bd, do.ProdutoID.String())
	if err != nil {
		return Item{}, err
	}
	if q > int(produto.EstoqueDisponivel) {
		return Item{}, AcimaDoEstoque{
			ProdutoID: produto.ID, Nome: produto.Nome, Disponivel: int(produto.EstoqueDisponivel), Solicitado: q,
		}
	}
	linha, err := consultas.AlterarItem(ctx, gerado.AlterarItemParams{
		ID:                 chave,
		CompradorID:        dono,
		Quantidade:         int32(q),
		PrecoVistoCentavos: produto.PrecoCentavos,
	})
	if err != nil {
		return Item{}, err
	}
	return Item{ID: linha.ID.String(), ProdutoID: linha.ProdutoID.String(), Quantidade: linha.Quantidade}, nil
}

// Itens abre o Carrinho do Comprador e o revalida (FR-19): leitura pura
// (AD-17), que nunca grava `preco_visto_centavos`. Comprador sem Carrinho, ou
// com o Carrinho vazio, recebe Itens vazio — nunca nil, para o JSON sair `[]`.
//
// Preço, Estoque e visibilidade vêm de catalogo.Resumos, em lote (AD-19). Cada
// linha sai com o que mudou (PrecoMudou) e o que a impede (Bloqueio).
func Itens(ctx context.Context, bd gerado.DBTX, compradorID string) (Conteudo, error) {
	dono, err := uuidDe(compradorID)
	if err != nil {
		return Conteudo{}, err
	}
	linhas, err := gerado.New(bd).ListarItens(ctx, dono)
	if err != nil {
		return Conteudo{}, err
	}
	ids := make([]string, len(linhas))
	for i, l := range linhas {
		ids[i] = l.ProdutoID.String()
	}
	resumos, err := catalogo.Resumos(ctx, bd, ids)
	if err != nil {
		return Conteudo{}, err
	}
	conteudo := Conteudo{Itens: make([]ItemDoCarrinho, 0, len(linhas))}
	for i, l := range linhas {
		item := ItemDoCarrinho{ID: l.ID.String(), ProdutoID: ids[i], Quantidade: l.Quantidade}
		r, visivel := resumos[ids[i]]
		if visivel {
			item.Visivel, item.Nome, item.ImagemURL, item.PrecoCentavos = true, r.Nome, r.ImagemURL, r.PrecoCentavos
			item.PrecoVistoCentavos, item.EstoqueDisponivel = l.PrecoVistoCentavos, r.EstoqueDisponivel
			item.PrecoMudou = r.PrecoCentavos != l.PrecoVistoCentavos
			conteudo.SubtotalCentavos += r.PrecoCentavos * int64(l.Quantidade)
		}
		item.Bloqueio = bloqueioDe(visivel, r.EstoqueDisponivel, l.Quantidade)
		conteudo.Unidades += int64(l.Quantidade)
		conteudo.Itens = append(conteudo.Itens, item)
	}
	return conteudo, nil
}

// Limpar tira todos os Itens do Carrinho do Comprador: o esvaziar que ele pede
// na tela (FR-18). Não é o Esvaziar da criação do Pedido, que recebe a
// transação e só os Itens lidos (AD-3). Carrinho sem Itens é sucesso.
func Limpar(ctx context.Context, bd gerado.DBTX, compradorID string) error {
	dono, err := uuidDe(compradorID)
	if err != nil {
		return err
	}
	return gerado.New(bd).LimparItens(ctx, dono)
}

// RemoverItem apaga o Item do dono. A posse está no WHERE (AD-11): Item
// alheio, inexistente e de uuid malformado saem os três como pgx.ErrNoRows.
func RemoverItem(ctx context.Context, bd gerado.DBTX, itemID, compradorID string) error {
	chave, err := uuidDe(itemID)
	if err != nil {
		return err
	}
	dono, err := uuidDe(compradorID)
	if err != nil {
		return err
	}
	linhas, err := gerado.New(bd).RemoverItem(ctx, gerado.RemoverItemParams{ID: chave, CompradorID: dono})
	if err != nil {
		return err
	}
	if linhas == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// PrecoVisto é uma linha a confirmar: o Item e o preço que o Comprador acabou
// de ver reportado. O preço vem de quem reportou, e nunca de uma releitura.
type PrecoVisto struct {
	ItemID        string
	PrecoCentavos int64
}

// ConfirmarPrecoVisto persiste a ciência de uma mudança de preço que o
// Comprador NÃO provocou — a única escrita de `preco_visto_centavos` fora de
// Adicionar e AlterarQuantidade, que gravam o preço de uma alteração que ele
// mesmo fez e que já é ciência. É a única porta dessa ciência (AD-17), e quem
// a abre é só `pedido`, na transação de entrada no checkout, **depois** de
// montar a resposta que reporta a diferença.
//
// Grava exatamente os preços recebidos: confirmar uma releitura apagaria uma
// mudança que ninguém chegou a ver. Lista vazia é no-op com nil — um Carrinho
// sem preço mudado não é erro —, Item alheio e inexistente não casam, porque a
// posse está no WHERE (AD-11), e o uuid malformado sai como pgx.ErrNoRows,
// como no resto do módulo.
//
// Os vistos são ordenados por ItemID antes de virarem parâmetro, e isso é só
// determinismo: a mesma entrada produz sempre o mesmo comando, o que torna
// falha reproduzível e diff de log comparável. **Não** é ordem de travas — num
// UPDATE ... FROM (unnest ...) quem decide em que ordem as linhas são
// bloqueadas é o plano do executor, e não a posição no vetor. A garantia de
// ordem do AD-5 vem de `SELECT ... ORDER BY id FOR UPDATE`, que é outra
// construção e que esta confirmação não precisa: ela não decide Estoque, e
// todas as linhas que toca são do mesmo dono.
//
// O número de linhas afetadas é conferido contra o número de vistos: uma
// escrita parcial significa que o Carrinho mudou debaixo da transação (um Item
// removido noutra aba, em READ COMMITTED), e deixá-la passar calada devolveria
// um "de X para Y" que não foi gravado — o aviso voltaria para sempre, que é o
// defeito que esta função existe para matar. Divergência é ErrCarrinhoMudou
// (409 CARRINHO_MUDOU, e não mais o 500 genérico), e quem chama desfaz a
// transação: nada reportado, nada gravado, e a entrada seguinte avisa
// de novo.
func ConfirmarPrecoVisto(ctx context.Context, bd gerado.DBTX, compradorID string, vistos []PrecoVisto) error {
	if len(vistos) == 0 {
		return nil
	}
	dono, err := uuidDe(compradorID)
	if err != nil {
		return err
	}
	ordenados := slices.Clone(vistos)
	slices.SortFunc(ordenados, func(a, b PrecoVisto) int { return strings.Compare(a.ItemID, b.ItemID) })
	ids := make([]pgtype.UUID, 0, len(ordenados))
	precos := make([]int64, 0, len(ordenados))
	for _, v := range ordenados {
		chave, err := uuidDe(v.ItemID)
		if err != nil {
			return err
		}
		ids = append(ids, chave)
		precos = append(precos, v.PrecoCentavos)
	}
	linhas, err := gerado.New(bd).ConfirmarPrecoVisto(ctx, gerado.ConfirmarPrecoVistoParams{
		CompradorID: dono, Ids: ids, Precos: precos,
	})
	if err != nil {
		return err
	}
	if linhas != int64(len(ordenados)) {
		return fmt.Errorf("confirmar o preço visto: %d linhas gravadas de %d reportadas: %w", linhas, len(ordenados), ErrCarrinhoMudou)
	}
	return nil
}

// Esvaziar tira do Carrinho do Comprador **só** os Itens que a criação do
// Pedido leu (AD-3): é a única escrita de fora que tira Item do Carrinho, e
// quem a chama é `pedido`, dentro da transação em que o Pedido nasce, depois
// de catalogo.Reservar suceder. Um Item adicionado noutra aba depois da
// leitura fica no Carrinho e não entra no Pedido.
//
// A posse está no WHERE (AD-11). Linha a menos — um dos Itens lidos sumiu
// debaixo da transação — é ErrCarrinhoMudou, e quem chama desfaz tudo: o
// Pedido não pode nascer com um Item que o Carrinho já não tinha. Lista vazia
// é no-op com nil.
//
// tx é parâmetro nomeado, e não valor de contexto (AD-4): é a mesma transação
// do Pedido, e reverter uma reverte as duas.
func Esvaziar(ctx context.Context, tx pgx.Tx, compradorID string, itemIDs []string) error {
	if len(itemIDs) == 0 {
		return nil
	}
	dono, err := uuidDe(compradorID)
	if err != nil {
		return err
	}
	ids := make([]pgtype.UUID, 0, len(itemIDs))
	for _, id := range itemIDs {
		chave, err := uuidDe(id)
		if err != nil {
			return err
		}
		ids = append(ids, chave)
	}
	linhas, err := gerado.New(tx).EsvaziarItens(ctx, gerado.EsvaziarItensParams{Ids: ids, CompradorID: dono})
	if err != nil {
		return fmt.Errorf("esvaziar o Carrinho: %w", err)
	}
	if linhas != int64(len(ids)) {
		return fmt.Errorf("esvaziar o Carrinho: %d Itens apagados de %d lidos: %w", linhas, len(ids), ErrCarrinhoMudou)
	}
	return nil
}

// uuidDe: o identificador malformado vira pgx.ErrNoRows, e não um 500 — para
// quem chama, "não é um uuid" e "não existe" são a mesma coisa.
func uuidDe(texto string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(texto); err != nil {
		return u, pgx.ErrNoRows
	}
	return u, nil
}
