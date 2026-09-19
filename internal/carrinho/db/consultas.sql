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

-- Quanto o dono já tem do Produto: a soma que o teto e o Estoque recusam. A
-- soma sobre nenhuma linha é o zero do COALESCE, e por isso a consulta não
-- cria o Carrinho de quem só tentou adicionar.
-- name: QuantidadeDoProduto :one
SELECT COALESCE(SUM(item.quantidade), 0)::int AS quantidade
FROM carrinho.item_carrinho AS item
JOIN carrinho.carrinho AS c ON c.id = item.carrinho_id
WHERE c.comprador_id = @comprador_id AND item.produto_id = @produto_id;

-- O Item do dono, para saber de que Produto ele é. Alheio e inexistente
-- voltam as mesmas zero linhas (AD-11).
-- name: BuscarItem :one
SELECT item.produto_id, item.quantidade
FROM carrinho.item_carrinho AS item
JOIN carrinho.carrinho AS c ON c.id = item.carrinho_id
WHERE item.id = @id AND c.comprador_id = @comprador_id;

-- A quantidade é absoluta, e o preço visto é o da alteração (AD-2). A posse
-- volta ao WHERE: BuscarItem já a conferiu, mas a linha pode ter mudado de mãos
-- — ou sumido — entre as duas idas.
-- name: AlterarItem :one
UPDATE carrinho.item_carrinho AS item
SET quantidade = @quantidade, preco_visto_centavos = @preco_visto_centavos
FROM carrinho.carrinho AS c
WHERE item.id = @id AND item.carrinho_id = c.id AND c.comprador_id = @comprador_id
RETURNING item.id, item.produto_id, item.quantidade;

-- Leitura pura (AD-17): não há UPDATE aqui. O preço visto sai da consulta para
-- a revalidação dizer "de X para Y" (4.4), mas quem o grava não é esta.
-- O id é uuidv7, então a ordem é a da adição.
-- name: ListarItens :many
SELECT item.id, item.produto_id, item.quantidade, item.preco_visto_centavos
FROM carrinho.item_carrinho AS item
JOIN carrinho.carrinho AS c ON c.id = item.carrinho_id
WHERE c.comprador_id = @comprador_id
ORDER BY item.id;

-- O esvaziar do Comprador (FR-18). O Carrinho em si fica: é um por Comprador e
-- não expira. Sem Item nenhum, zero linhas e nenhum erro.
-- name: LimparItens :exec
DELETE FROM carrinho.item_carrinho AS item
USING carrinho.carrinho AS c
WHERE item.carrinho_id = c.id AND c.comprador_id = @comprador_id;

-- A posse entra no WHERE (AD-11): Item alheio e inexistente afetam as mesmas
-- zero linhas.
-- name: RemoverItem :execrows
DELETE FROM carrinho.item_carrinho AS item
USING carrinho.carrinho AS c
WHERE item.id = @id AND item.carrinho_id = c.id AND c.comprador_id = @comprador_id;
