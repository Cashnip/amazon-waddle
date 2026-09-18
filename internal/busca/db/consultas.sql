-- `busca` não tem tabelas: lê o Catálogo só pela VIEW produto_visivel (AD-2,
-- AD-16), que já aplica o predicado de visibilidade (AD-19). Nenhum `WHERE
-- ativo` aqui. O desempate final é sempre `id` (AD-18); a ordenação da 3.9
-- antepõe a sua chave.

-- name: ListarVisiveis :many
SELECT id, nome, preco_centavos, imagem_url, estoque_disponivel
FROM catalogo.produto_visivel
ORDER BY id
LIMIT @limite OFFSET @deslocamento;

-- name: ContarVisiveis :one
SELECT count(*) FROM catalogo.produto_visivel;
