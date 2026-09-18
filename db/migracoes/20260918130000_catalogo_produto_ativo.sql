-- O "Produto ativo" do AD-19 (3.3): desativar Produto (FR-9) é um UPDATE desta
-- coluna, e quem esconde é a mesma VIEW `produto_visivel` — nenhum outro lugar
-- escreve `WHERE ativo`. As colunas da VIEW são as mesmas da 3.1: o
-- `CREATE OR REPLACE VIEW` só aceita trocar a definição mantendo a lista.
-- +goose Up
ALTER TABLE catalogo.produto ADD COLUMN ativo boolean NOT NULL DEFAULT true;

CREATE OR REPLACE VIEW catalogo.produto_visivel AS
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url,
       p.vendedor_id, p.categoria_id, p.busca_normalizada, p.estoque_total,
       v.nome AS vendedor_nome
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
WHERE v.ativo AND p.ativo;

-- +goose Down
CREATE OR REPLACE VIEW catalogo.produto_visivel AS
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url,
       p.vendedor_id, p.categoria_id, p.busca_normalizada, p.estoque_total,
       v.nome AS vendedor_nome
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
WHERE v.ativo;

ALTER TABLE catalogo.produto DROP COLUMN ativo;
