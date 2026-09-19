-- O Carrinho nasce na primeira adição. O DO UPDATE que não muda nada existe
-- só para o RETURNING devolver o id também quando a linha já existia.
-- name: GarantirCarrinho :one
INSERT INTO carrinho.carrinho (comprador_id)
VALUES (@comprador_id)
ON CONFLICT (comprador_id) DO UPDATE SET comprador_id = EXCLUDED.comprador_id
RETURNING id;

-- A soma e o teto numa ida só: o Produto repetido soma ao Item existente, e a
-- soma acima do teto não grava — zero linhas no RETURNING, que carrinho
-- traduz em ErrTetoPorItem. O preço visto é sempre o da adição mais recente.
-- name: AdicionarItem :one
INSERT INTO carrinho.item_carrinho AS item (carrinho_id, produto_id, quantidade, preco_visto_centavos)
VALUES (@carrinho_id, @produto_id, @quantidade, @preco_visto_centavos)
ON CONFLICT (carrinho_id, produto_id) DO UPDATE
SET quantidade = item.quantidade + EXCLUDED.quantidade,
    preco_visto_centavos = EXCLUDED.preco_visto_centavos
WHERE item.quantidade + EXCLUDED.quantidade <= @teto::int
RETURNING item.id, item.produto_id, item.quantidade;

-- A posse entra no WHERE (AD-11): Item alheio e inexistente afetam as mesmas
-- zero linhas.
-- name: RemoverItem :execrows
DELETE FROM carrinho.item_carrinho AS item
USING carrinho.carrinho AS c
WHERE item.id = @id AND item.carrinho_id = c.id AND c.comprador_id = @comprador_id;
