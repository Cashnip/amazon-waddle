-- Uma consulta por módulo nesta estória: o que a 1.3 precisa provar é que o
-- sqlc analisa `DEFAULT uuidv7()` (verificação 2 do passo 0). As consultas de
-- verdade chegam com a regra, nas Épicas 2 e 3.

-- name: BuscarProdutoPorID :one
SELECT id, nome, descricao, preco_centavos, imagem_url, vendedor_id, categoria_id, busca_normalizada
FROM catalogo.produto
WHERE id = $1;
