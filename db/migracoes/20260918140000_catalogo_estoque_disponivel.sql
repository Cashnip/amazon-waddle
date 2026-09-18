-- O Estoque disponível do AD-5 (3.4): derivado aqui, nunca armazenado. A VIEW
-- deixa de expor `estoque_total` (AD-16) e passa a expor `estoque_disponivel`
-- = total − Σ Reservas ATIVA, sem nunca ficar negativo. O predicado de
-- visibilidade (AD-19) continua o mesmo e continua só aqui. DROP + CREATE
-- porque a lista de colunas muda, e o `CREATE OR REPLACE VIEW` não aceita tirar
-- coluna.
-- +goose Up
DROP VIEW catalogo.produto_visivel;

CREATE VIEW catalogo.produto_visivel AS
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url,
       p.vendedor_id, p.categoria_id, p.busca_normalizada,
       -- Subconsulta correlacionada, e não JOIN com agregado: ela usa o
       -- índice `reserva_estoque_produto_id_idx` linha a linha, e a consulta
       -- por uma fatia de ids não soma as reservas do Catálogo inteiro.
       greatest(p.estoque_total - coalesce((
           SELECT sum(r.quantidade) FROM catalogo.reserva_estoque r
           WHERE r.produto_id = p.id AND r.estado = 'ATIVA'
       ), 0), 0)::integer AS estoque_disponivel,
       v.nome AS vendedor_nome
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
WHERE v.ativo AND p.ativo;

-- +goose Down
DROP VIEW catalogo.produto_visivel;

CREATE VIEW catalogo.produto_visivel AS
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url,
       p.vendedor_id, p.categoria_id, p.busca_normalizada, p.estoque_total,
       v.nome AS vendedor_nome
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
WHERE v.ativo AND p.ativo;
