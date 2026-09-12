-- A consulta da Página de Produto (FR-9). O JOIN é dentro do mesmo schema —
-- chave estrangeira cruzando schema é proibida (AD-2), e esta não cruza.
-- `busca_normalizada` fica de fora: é dado de índice, e a busca de verdade,
-- com paginação, é da Épica 3.
-- name: BuscarProdutoComVendedor :one
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url, v.nome AS vendedor_nome
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
WHERE p.id = $1;

-- As duas consultas abaixo são o AD-5, e são duas de propósito: é a ordem
-- entre elas que protege o Estoque. Uma consulta só não teria como errá-la —
-- e errá-la é vender a mesma última unidade duas vezes.
--
-- Primeiro a trava. O `ORDER BY id` não é estilo: sem ele, dois Pedidos com os
-- mesmos dois Produtos em ordem inversa travam um no outro.
-- name: TravarProdutosParaReserva :many
SELECT id, estoque_total
FROM catalogo.produto
WHERE id = ANY(@ids::uuid[])
ORDER BY id
FOR UPDATE;

-- Só depois a soma. Com a trava segura, nenhuma Reserva nova entra entre a
-- soma e o INSERT que a sucede.
-- name: SomarReservasAtivas :one
SELECT coalesce(sum(quantidade), 0)::bigint AS reservado
FROM catalogo.reserva_estoque
WHERE produto_id = $1 AND estado = 'ATIVA';

-- name: CriarReservaAtiva :exec
INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
VALUES ($1, $2, $3, 'ATIVA');
