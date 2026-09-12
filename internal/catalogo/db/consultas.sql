-- A consulta da Página de Produto (FR-9). O JOIN é dentro do mesmo schema —
-- chave estrangeira cruzando schema é proibida (AD-2), e esta não cruza.
-- `busca_normalizada` fica de fora: é dado de índice, e a busca de verdade,
-- com paginação, é da Épica 3.
-- name: BuscarProdutoComVendedor :one
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url, v.nome AS vendedor_nome
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
WHERE p.id = $1;
