// Package catalogo é dono de Vendedor, Categoria, Produto, Estoque e Reserva de Estoque.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1). O predicado de visibilidade do AD-19, a liberação e a
// consolidação da Reserva são da Épica 3; o que mora aqui é o detalhe de
// Produto da Estória 1.5 e a Reserva que a 1.6 faz nascer.
package catalogo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/catalogo/db/gerado"
)

// ErrEstoqueInsuficiente recusa a compra que não cabe no que sobrou. Quem
// precisa do número disponível usa errors.As com EstoqueInsuficiente.
var ErrEstoqueInsuficiente = errors.New("Estoque insuficiente para este Produto.")

// EstoqueInsuficiente carrega o disponível até `api/`, que o publica em
// `dados` do envelope — a mensagem continua sendo a do sentinela (AD-14).
type EstoqueInsuficiente struct{ Disponivel int64 }

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
}

// BuscarProduto devolve o Produto com o nome do Vendedor. Identificador
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
	}, nil
}

// Reservar segura a unidade comprada antes que outro Pedido a leve. A ordem
// dos dois comandos é metade da corretude (AD-5): a trava dos Produtos vem
// primeiro, e só com ela segura a soma das reservas ativas é confiável —
// invertida, dois Pedidos simultâneos leem o mesmo "sobra uma" e vendem a
// mesma última unidade duas vezes.
//
// tx é parâmetro nomeado, e não valor de contexto (AD-4): é a mesma transação
// em que o Pedido nasceu, e reverter uma reverte os dois.
func Reservar(ctx context.Context, tx pgx.Tx, produtoID, pedidoID string, quantidade int32) error {
	var produto, pedido pgtype.UUID
	if err := produto.Scan(produtoID); err != nil {
		return pgx.ErrNoRows
	}
	if err := pedido.Scan(pedidoID); err != nil {
		return fmt.Errorf("identificador de Pedido inválido: %w", err)
	}
	q := gerado.New(tx)

	// Primeiro a trava, sobre a fatia inteira e em ordem de identificador.
	travados, err := q.TravarProdutosParaReserva(ctx, []pgtype.UUID{produto})
	if err != nil {
		return fmt.Errorf("travar o Produto: %w", err)
	}
	if len(travados) == 0 {
		return pgx.ErrNoRows
	}

	// Só então a soma.
	reservado, err := q.SomarReservasAtivas(ctx, produto)
	if err != nil {
		return fmt.Errorf("somar as reservas ativas: %w", err)
	}

	disponivel := int64(travados[0].EstoqueTotal) - reservado
	if int64(quantidade) > disponivel {
		return EstoqueInsuficiente{Disponivel: max(disponivel, 0)}
	}
	if err := q.CriarReservaAtiva(ctx, gerado.CriarReservaAtivaParams{
		ProdutoID:  produto,
		PedidoID:   pedido,
		Quantidade: quantidade,
	}); err != nil {
		return fmt.Errorf("criar a Reserva de Estoque: %w", err)
	}
	return nil
}
