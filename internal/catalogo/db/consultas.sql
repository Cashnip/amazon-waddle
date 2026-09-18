-- A consulta da Página de Produto (FR-9). Lê da VIEW do AD-19: Produto de
-- Vendedor desativado sai daqui como zero linhas, o mesmo 404 do inexistente.
-- `busca_normalizada` fica de fora: é dado de índice.
-- name: BuscarProdutoComVendedor :one
SELECT id, nome, descricao, preco_centavos, imagem_url, vendedor_nome
FROM catalogo.produto_visivel
WHERE id = $1;

-- As duas consultas abaixo são o AD-5, e são duas de propósito: é a ordem
-- entre elas que protege o Estoque. Uma consulta só não teria como errá-la —
-- e errá-la é vender a mesma última unidade duas vezes.
--
-- Primeiro a trava. O `ORDER BY id` não é estilo: sem ele, dois Pedidos com os
-- mesmos dois Produtos em ordem inversa travam um no outro.
--
-- A visibilidade entra na mesma consulta, por subconsulta à VIEW, e a trava é
-- `OF p`: `FOR UPDATE` sobre a VIEW travaria também a linha do Vendedor, e
-- Pedidos de Produtos diferentes do mesmo Vendedor esperariam um pelo outro.
-- name: TravarProdutosParaReserva :many
SELECT p.id, p.estoque_total
FROM catalogo.produto p
WHERE p.id = ANY(@ids::uuid[])
  AND p.id IN (SELECT id FROM catalogo.produto_visivel)
ORDER BY p.id
FOR UPDATE OF p;

-- Só depois a soma. Com a trava segura, nenhuma Reserva nova entra entre a
-- soma e o INSERT que a sucede.
-- name: SomarReservasAtivas :one
SELECT coalesce(sum(quantidade), 0)::bigint AS reservado
FROM catalogo.reserva_estoque
WHERE produto_id = $1 AND estado = 'ATIVA';

-- name: CriarReservaAtiva :exec
INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
VALUES ($1, $2, $3, 'ATIVA');

-- As duas consultas da consolidação (Estória 1.8), a única passagem em que o
-- Estoque total muda. O `estado = 'ATIVA'` no WHERE é compare-and-swap como o
-- do Status: zero linhas devolvidas é Reserva já consolidada, que é no-op e
-- nunca erro.
-- name: ConsolidarReservasDoPedido :many
UPDATE catalogo.reserva_estoque
SET estado = 'CONSOLIDADA'
WHERE pedido_id = @pedido_id AND estado = 'ATIVA'
RETURNING produto_id, quantidade;

-- A baixa é uma por Produto, e quem as ordena por identificador é o Go: o
-- UPDATE … RETURNING não aceita ORDER BY, e a ordem é a mesma disciplina do
-- `ORDER BY id FOR UPDATE` do Reservar — dois Pedidos com os mesmos Produtos
-- em ordem inversa travariam um no outro. O CHECK (estoque_total >= 0) da
-- migração é quem barra a baixa a mais.
-- name: BaixarEstoqueTotal :exec
UPDATE catalogo.produto
SET estoque_total = estoque_total - @quantidade
WHERE id = @produto_id;

-- A gestão de Vendedores (3.1). A lista traz ativos e inativos — é a tela do
-- Administrador, e não a loja — por nome, com desempate estável por id.
-- name: ListarVendedores :many
SELECT id, nome, ativo FROM catalogo.vendedor ORDER BY nome, id;

-- Nome duplicado é decidido pelo UNIQUE (23505), nunca por SELECT antes.
-- name: CriarVendedor :one
INSERT INTO catalogo.vendedor (nome) VALUES (@nome)
RETURNING id, nome, ativo;

-- Desativar é este UPDATE de `ativo`: nenhum Pedido é tocado, e quem esconde
-- os Produtos é a VIEW produto_visivel.
-- name: AtualizarVendedor :one
UPDATE catalogo.vendedor SET nome = @nome, ativo = @ativo
WHERE id = @id
RETURNING id, nome, ativo;

-- Vendedor com Produto é barrado pela FK `produto.vendedor_id` (23503), sem
-- contagem antes.
-- name: RemoverVendedor :execrows
DELETE FROM catalogo.vendedor WHERE id = @id;
