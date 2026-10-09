-- Conjunto de medição do NFR-4 — 5.044 Produtos.
--
-- NÃO é o catálogo da demonstração (SM-C3): entra só com
-- AZAMON_SEMENTE_GRANDE=true, sempre depois do Catálogo Semeado, e existe
-- para a sonda de latência da Estória 1.4.
--
-- Os 5.044 saem de 194 × 13 × 2 (desde a 8.1): cada Produto semeado ganha
-- uma linha e uma série, e o total com o Catálogo Semeado é 5.238. É a conta
-- mais perto dos 5.000 do NFR-4 que ainda distribui igual por todas as
-- Categorias. O SELECT enxerga o instantâneo anterior ao INSERT, então só os
-- 194 do Catálogo Semeado entram na multiplicação — e o marcador em
-- public.semente garante que isto roda uma vez só. A soma no preço mexe só
-- nos reais, e os centavos do Produto de origem ficam.
--
-- `lower()` é toda a normalização aqui porque linha e série são ASCII de
-- propósito: sem diacrítico, o resultado é o mesmo que catalogo.Normalizar
-- daria, e `lower()` é IMMUTABLE (AD-16).
WITH linha (nome, ordem) AS (VALUES
  ('Prata', 1), ('Bronze', 2), ('Grafite', 3), ('Marfim', 4), ('Cobalto', 5),
  ('Carmim', 6), ('Turmalina', 7), ('Jade', 8), ('Quartzo', 9), ('Safira', 10),
  ('Onix', 11), ('Coral', 12), ('Ambar', 13)
), serie (nome, ordem) AS (VALUES
  ('2024', 1), ('2025', 2)
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
