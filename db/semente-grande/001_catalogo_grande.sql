-- Conjunto de medição do NFR-4 — 5.000 Produtos.
--
-- NÃO é o catálogo da demonstração (SM-C3): entra só com
-- AZAMON_SEMENTE_GRANDE=true, sempre depois do Catálogo Semeado, e existe
-- para a sonda de latência da Estória 1.4.
--
-- Os 5.000 saem de 50 × 10 × 10: cada Produto semeado ganha uma linha e uma
-- série. O SELECT enxerga o instantâneo anterior ao INSERT, então só os 50
-- do Catálogo Semeado entram na multiplicação — e o marcador em
-- public.semente garante que isto roda uma vez só.
--
-- `lower()` é toda a normalização aqui porque linha e série são ASCII de
-- propósito: sem diacrítico, o resultado é o mesmo que catalogo.Normalizar
-- daria, e `lower()` é IMMUTABLE (AD-16).
WITH linha (nome, ordem) AS (VALUES
  ('Prata', 1), ('Bronze', 2), ('Grafite', 3), ('Marfim', 4), ('Cobalto', 5),
  ('Carmim', 6), ('Turmalina', 7), ('Jade', 8), ('Quartzo', 9), ('Safira', 10)
), serie (nome, ordem) AS (VALUES
  ('2016', 1), ('2017', 2), ('2018', 3), ('2019', 4), ('2020', 5),
  ('2021', 6), ('2022', 7), ('2023', 8), ('2024', 9), ('2025', 10)
)
INSERT INTO catalogo.produto
  (nome, descricao, preco_centavos, imagem_url, vendedor_id, categoria_id, busca_normalizada)
SELECT
  p.nome || ' ' || l.nome || ' ' || s.nome,
  p.descricao,
  p.preco_centavos + l.ordem * 1000 + s.ordem * 100,
  p.imagem_url,
  p.vendedor_id,
  p.categoria_id,
  p.busca_normalizada || ' ' || lower(l.nome) || ' ' || lower(s.nome)
FROM catalogo.produto p
CROSS JOIN linha l
CROSS JOIN serie s;
