// Package catalogo é dono de Vendedor, Categoria, Produto, Estoque e Reserva de Estoque.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1). Estoque, Reserva e o predicado de visibilidade do AD-19 são
// da Épica 3; o que mora aqui é o detalhe de Produto da Estória 1.5.
package catalogo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/catalogo/db/gerado"
)

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
