-- "Mais recentes" (3.9) pede o instante de criação do Produto. A VIEW passa a
-- expor `criado_em`; o resto dela é o da 3.4. DROP + CREATE pelo mesmo motivo
-- da 3.4: a lista de colunas muda. Produtos que já existem ficam com o instante
-- da migração, e o desempate por `id` decide entre eles.
-- +goose Up
ALTER TABLE catalogo.produto ADD COLUMN criado_em timestamptz NOT NULL DEFAULT now();

DROP VIEW catalogo.produto_visivel;

CREATE VIEW catalogo.produto_visivel AS
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url,
       p.vendedor_id, p.categoria_id, p.busca_normalizada,
       greatest(p.estoque_total - coalesce((
           SELECT sum(r.quantidade) FROM catalogo.reserva_estoque r
           WHERE r.produto_id = p.id AND r.estado = 'ATIVA'
       ), 0), 0)::integer AS estoque_disponivel,
       v.nome AS vendedor_nome,
       p.criado_em
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
WHERE v.ativo AND p.ativo;

-- +goose Down
DROP VIEW catalogo.produto_visivel;

CREATE VIEW catalogo.produto_visivel AS
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url,
       p.vendedor_id, p.categoria_id, p.busca_normalizada,
       greatest(p.estoque_total - coalesce((
           SELECT sum(r.quantidade) FROM catalogo.reserva_estoque r
           WHERE r.produto_id = p.id AND r.estado = 'ATIVA'
       ), 0), 0)::integer AS estoque_disponivel,
       v.nome AS vendedor_nome
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
WHERE v.ativo AND p.ativo;

ALTER TABLE catalogo.produto DROP COLUMN criado_em;
