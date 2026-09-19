// Package catalogo é dono de Vendedor, Categoria, Produto, Estoque e Reserva de Estoque.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1). Mora aqui a interface de Estoque do AD-5/AD-19 que Carrinho,
// checkout e cancelamento consomem, com as assinaturas já definitivas:
// Disponivel e Visiveis em lote, lidos da VIEW produto_visivel — o único lugar
// do predicado de visibilidade —, e Reservar, Liberar e Consolidar, as três
// idempotentes. Só Consolidar baixa o Estoque total no caminho da compra; o
// outro que o escreve é o ajuste do Administrador (produto.go), sob a mesma
// ordem de trava. Também mora aqui o detalhe de Produto da Página de Produto.
package catalogo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/catalogo/db/gerado"
)

// ErrEstoqueInsuficiente recusa a compra que não cabe no que sobrou. Quem
// precisa do número disponível usa errors.As com EstoqueInsuficiente.
var ErrEstoqueInsuficiente = errors.New("Estoque insuficiente para este Produto.")

// EstoqueInsuficiente carrega o Produto que faltou e o disponível dele até
// `api/`, que publica o número em `dados` do envelope — a mensagem continua
// sendo a do sentinela (AD-14). ProdutoID tem a grafia que o chamador passou.
type EstoqueInsuficiente struct {
	ProdutoID  string
	Disponivel int64
}

func (e EstoqueInsuficiente) Error() string { return ErrEstoqueInsuficiente.Error() }
func (e EstoqueInsuficiente) Unwrap() error { return ErrEstoqueInsuficiente }

// Produto é o detalhe que a Página de Produto mostra. O preço é int64 de
// centavos de ponta a ponta (AD-3) — quem formata `R$` é o navegador — e a
// imagem é URL relativa, servida pelo Go de arquivo embutido (AD-12).
// `busca_normalizada` fica de fora de propósito: é dado de índice.
type Produto struct {
	ID            string
	Nome          string
	Descricao     string
	PrecoCentavos int64
	ImagemURL     string
	VendedorNome  string
	CategoriaID   string
	CategoriaNome string
	// EstoqueDisponivel é o total menos as Reservas ativas (AD-5), nunca o
	// total: a Reserva não aparece para o Comprador.
	EstoqueDisponivel int32
}

// BuscarProduto devolve o Produto com o nome do Vendedor, a Categoria e o
// Estoque disponível. Identificador
// malformado e Produto inexistente são a mesma coisa para quem chama:
// pgx.ErrNoRows, que api/ traduz em 404 — nunca em 500.
// bd é a DBTX do sqlc pelo mesmo motivo de identidade.Autenticar: `api/` passa
// o pool sem importar o pacote gerado deste módulo (AD-1).
func BuscarProduto(ctx context.Context, bd gerado.DBTX, id string) (Produto, error) {
	var chave pgtype.UUID
	if err := chave.Scan(id); err != nil {
		return Produto{}, pgx.ErrNoRows
	}
	linha, err := gerado.New(bd).BuscarProdutoComVendedor(ctx, chave)
	if err != nil {
		return Produto{}, err
	}
	return Produto{
		// O identificador sai da linha, e não do texto da rota: o Postgres
		// aceita o uuid em maiúsculas e o DTO tem de devolver a forma
		// canônica, que é a única que o banco guarda.
		ID:            linha.ID.String(),
		Nome:          linha.Nome,
		Descricao:     linha.Descricao,
		PrecoCentavos: linha.PrecoCentavos,
		ImagemURL:     linha.ImagemUrl,
		VendedorNome:  linha.VendedorNome,
		CategoriaID:   linha.CategoriaID.String(),
		CategoriaNome: linha.CategoriaNome,

		EstoqueDisponivel: linha.EstoqueDisponivel,
	}, nil
}

// ItemReserva é uma linha do que o Pedido quer segurar: um Produto e quantas
// unidades dele.
type ItemReserva struct {
	ProdutoID  string
	Quantidade int32
}

// Disponivel devolve o Estoque disponível de cada id recebido, com a mesma
// grafia como chave. Produto invisível, inexistente ou de id malformado vale 0,
// sem omitir a chave e sem erro — erro é só o do banco. Orienta a tela, mas
// não decide nada: quem decide é Reservar, sob bloqueio (AD-5).
func Disponivel(ctx context.Context, bd gerado.DBTX, ids []string) (map[string]int, error) {
	linhas, err := disponivelDosVisiveis(ctx, bd, ids)
	if err != nil {
		return nil, err
	}
	saida := make(map[string]int, len(ids))
	for _, id := range ids {
		saida[id] = 0
		if l, ok := linhas[id]; ok {
			saida[id] = int(l)
		}
	}
	return saida, nil
}

// Visiveis diz, para cada id recebido e com a mesma grafia, se o Produto é
// visível (AD-19): Produto ativo de Vendedor ativo. O predicado mora só na VIEW
// produto_visivel; aqui é a mesma consulta de Disponivel.
func Visiveis(ctx context.Context, bd gerado.DBTX, ids []string) (map[string]bool, error) {
	linhas, err := disponivelDosVisiveis(ctx, bd, ids)
	if err != nil {
		return nil, err
	}
	saida := make(map[string]bool, len(ids))
	for _, id := range ids {
		_, saida[id] = linhas[id]
	}
	return saida, nil
}

// Resumo é o que o Carrinho mostra de um Produto: nome, imagem, preço atual e
// Estoque disponível. É recorte de propósito — quem quer o detalhe da Página de
// Produto usa BuscarProduto.
type Resumo struct {
	ID                string
	Nome              string
	ImagemURL         string
	PrecoCentavos     int64
	EstoqueDisponivel int32
}

// Resumos devolve o Resumo dos Produtos visíveis (AD-19) numa ida só, indexado
// pelo id canônico — o que o Postgres guarda, e o que quem chama já tem quando
// os ids vêm de uma coluna uuid. Produto invisível ou inexistente não volta, e
// a chave ausente é como quem chama o reconhece.
func Resumos(ctx context.Context, bd gerado.DBTX, ids []string) (map[string]Resumo, error) {
	chaves := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		if chave, err := uuidDe(id); err == nil {
			chaves = append(chaves, chave)
		}
	}
	saida := make(map[string]Resumo, len(chaves))
	if len(chaves) == 0 {
		return saida, nil
	}
	linhas, err := gerado.New(bd).ResumosDosVisiveis(ctx, chaves)
	if err != nil {
		return nil, fmt.Errorf("ler os Produtos do Carrinho: %w", err)
	}
	for _, l := range linhas {
		id := l.ID.String()
		saida[id] = Resumo{
			ID:                id,
			Nome:              l.Nome,
			ImagemURL:         l.ImagemUrl,
			PrecoCentavos:     l.PrecoCentavos,
			EstoqueDisponivel: l.EstoqueDisponivel,
		}
	}
	return saida, nil
}

// disponivelDosVisiveis devolve o disponível só dos ids visíveis, indexado pela
// grafia recebida — o Postgres aceita o uuid em maiúsculas, e a linha volta na
// forma canônica.
func disponivelDosVisiveis(ctx context.Context, bd gerado.DBTX, ids []string) (map[string]int32, error) {
	grafias := map[[16]byte][]string{}
	chaves := []pgtype.UUID{}
	for _, id := range ids {
		chave, err := uuidDe(id)
		if err != nil {
			continue
		}
		if _, visto := grafias[chave.Bytes]; !visto {
			chaves = append(chaves, chave)
		}
		grafias[chave.Bytes] = append(grafias[chave.Bytes], id)
	}
	saida := map[string]int32{}
	if len(chaves) == 0 {
		return saida, nil
	}
	linhas, err := gerado.New(bd).DisponivelDosVisiveis(ctx, chaves)
	if err != nil {
		return nil, fmt.Errorf("ler o Estoque disponível: %w", err)
	}
	for _, l := range linhas {
		for _, id := range grafias[l.ID.Bytes] {
			saida[id] = l.EstoqueDisponivel
		}
	}
	return saida, nil
}

// Reservar segura as unidades do Pedido antes que outro Pedido as leve. A
// ordem dos dois comandos é metade da corretude (AD-5): a trava dos Produtos
// vem primeiro, sobre todos os ids e em ordem de identificador, e só com ela
// segura a soma das reservas ativas é confiável — invertida, dois Pedidos
// simultâneos leem o mesmo "sobra uma" e vendem a mesma última unidade duas
// vezes. Toda a verificação acontece antes de qualquer INSERT: ou tudo é
// reservado, ou nada.
//
// Idempotente no vazio: lista vazia é no-op e devolve nil.
//
// Produto invisível, inexistente ou de id malformado falha como
// EstoqueInsuficiente com disponível 0 — a visibilidade entra na mesma
// consulta FOR UPDATE (AD-19). O mesmo Produto repetido na lista soma as
// quantidades numa Reserva só.
//
// tx é parâmetro nomeado, e não valor de contexto (AD-4): é a mesma transação
// em que o Pedido nasceu, e reverter uma reverte os dois.
func Reservar(ctx context.Context, tx pgx.Tx, pedidoID string, itens []ItemReserva) error {
	if len(itens) == 0 {
		return nil
	}
	var pedido pgtype.UUID
	if err := pedido.Scan(pedidoID); err != nil {
		return fmt.Errorf("identificador de Pedido inválido: %w", err)
	}

	// Agrupa por Produto, na ordem em que chegaram; a grafia guardada é a
	// primeira recebida, que é a que o erro devolve.
	type pedida struct {
		chave      pgtype.UUID
		grafia     string
		quantidade int64
	}
	porProduto := map[[16]byte]*pedida{}
	ordem := []*pedida{}
	for _, item := range itens {
		// Quantidade não positiva é defeito de quem chama, e não falta de
		// Estoque: somada a outra linha do mesmo Produto, abateria a Reserva.
		if item.Quantidade <= 0 {
			return fmt.Errorf("quantidade inválida na Reserva: %d", item.Quantidade)
		}
		chave, err := uuidDe(item.ProdutoID)
		if err != nil {
			return EstoqueInsuficiente{ProdutoID: item.ProdutoID}
		}
		p, ok := porProduto[chave.Bytes]
		if !ok {
			p = &pedida{chave: chave, grafia: item.ProdutoID}
			porProduto[chave.Bytes] = p
			ordem = append(ordem, p)
		}
		p.quantidade += int64(item.Quantidade)
	}
	chaves := make([]pgtype.UUID, len(ordem))
	for i, p := range ordem {
		chaves[i] = p.chave
	}
	q := gerado.New(tx)

	// Primeiro a trava, sobre a fatia inteira e em ordem de identificador.
	travados, err := q.TravarProdutosParaReserva(ctx, chaves)
	if err != nil {
		return fmt.Errorf("travar os Produtos: %w", err)
	}
	total := make(map[[16]byte]int64, len(travados))
	for _, t := range travados {
		total[t.ID.Bytes] = int64(t.EstoqueTotal)
	}

	// Só então a soma, em lote.
	somas, err := q.SomarReservasAtivas(ctx, chaves)
	if err != nil {
		return fmt.Errorf("somar as reservas ativas: %w", err)
	}
	reservado := make(map[[16]byte]int64, len(somas))
	for _, s := range somas {
		reservado[s.ProdutoID.Bytes] = s.Reservado
	}

	for _, p := range ordem {
		estoque, visivel := total[p.chave.Bytes]
		if !visivel {
			return EstoqueInsuficiente{ProdutoID: p.grafia}
		}
		if disponivel := estoque - reservado[p.chave.Bytes]; p.quantidade > disponivel {
			return EstoqueInsuficiente{ProdutoID: p.grafia, Disponivel: max(disponivel, 0)}
		}
	}
	for _, p := range ordem {
		if err := q.CriarReservaAtiva(ctx, gerado.CriarReservaAtivaParams{
			ProdutoID:  p.chave,
			PedidoID:   pedido,
			Quantidade: int32(p.quantidade),
		}); err != nil {
			return fmt.Errorf("criar a Reserva de Estoque: %w", err)
		}
	}
	return nil
}

// Liberar devolve ao disponível as unidades que o Pedido segurava: a Reserva
// passa de ATIVA a LIBERADA, e `estoque_total` nunca é escrito — o disponível
// é derivado, e subir sozinho é o efeito.
//
// Idempotente: Pedido sem Reserva ativa é no-op que devolve nil, então liberar
// duas vezes (a expiração e o cancelamento, por exemplo) é inofensivo.
//
// tx é a mesma transação da transição do Pedido (AD-4).
func Liberar(ctx context.Context, tx pgx.Tx, pedidoID string) error {
	var pedido pgtype.UUID
	if err := pedido.Scan(pedidoID); err != nil {
		return fmt.Errorf("identificador de Pedido inválido: %w", err)
	}
	if err := gerado.New(tx).LiberarReservasDoPedido(ctx, pedido); err != nil {
		return fmt.Errorf("liberar a Reserva de Estoque: %w", err)
	}
	return nil
}

// Consolidar encerra a Reserva do Pedido e baixa o Estoque total na mesma
// quantidade. É o efeito da transição EM_SEPARACAO → ENVIADO, e a única
// passagem do sistema em que `estoque_total` muda — daí em diante o
// cancelamento não é mais possível, e a Reserva não tem mais o que segurar.
//
// Pedido sem Reserva ATIVA é no-op que devolve nil, nunca erro: consolidar duas
// vezes tem de ser inofensivo, ou um tique repetido baixaria o Estoque duas
// vezes e um Pedido já consolidado derrubaria a transação inteira.
//
// tx é a mesma transação em que o Status avançou (AD-4): a transição vem
// antes, e reverter uma reverte as duas.
func Consolidar(ctx context.Context, tx pgx.Tx, pedidoID string) error {
	var pedido pgtype.UUID
	if err := pedido.Scan(pedidoID); err != nil {
		return fmt.Errorf("identificador de Pedido inválido: %w", err)
	}
	q := gerado.New(tx)
	reservas, err := q.ConsolidarReservasDoPedido(ctx, pedido)
	if err != nil {
		return fmt.Errorf("consolidar a Reserva de Estoque: %w", err)
	}
	// Em ordem de identificador, pela mesma disciplina do `ORDER BY id FOR
	// UPDATE` do Reservar: na Épica 1 o Pedido tem um Produto só e a ordem não
	// muda nada, e ela existe para que o Carrinho da Épica 4 não descubra o
	// impasse já pronto.
	slices.SortFunc(reservas, func(a, b gerado.ConsolidarReservasDoPedidoRow) int {
		return bytes.Compare(a.ProdutoID.Bytes[:], b.ProdutoID.Bytes[:])
	})
	for _, r := range reservas {
		if err := q.BaixarEstoqueTotal(ctx, gerado.BaixarEstoqueTotalParams{
			ProdutoID:  r.ProdutoID,
			Quantidade: r.Quantidade,
		}); err != nil {
			return fmt.Errorf("baixar o Estoque total: %w", err)
		}
	}
	return nil
}
