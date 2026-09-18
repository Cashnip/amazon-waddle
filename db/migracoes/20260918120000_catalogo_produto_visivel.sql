-- O "Produto visível" do AD-19, definido uma única vez. Todo leitor de
-- Produto voltado à loja (Página de Produto, Reservar, e depois a Vitrine e a
-- busca) lê daqui, e nenhum outro lugar escreve `WHERE ativo`. Por enquanto a
-- regra é só "Vendedor ativo"; a 3.4 acrescenta "Produto ativo" nesta mesma
-- VIEW.
--
-- As colunas são nomeadas, e não `p.*`: o Postgres congela a lista no
-- CREATE, e uma coluna nova em `produto` não apareceria aqui de qualquer jeito.
-- +goose Up
CREATE VIEW catalogo.produto_visivel AS
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url,
       p.vendedor_id, p.categoria_id, p.busca_normalizada, p.estoque_total,
       v.nome AS vendedor_nome
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
WHERE v.ativo;

-- +goose Down
DROP VIEW catalogo.produto_visivel;
