-- `busca` não tem tabelas: lê o Catálogo só pela VIEW produto_visivel (AD-2,
-- AD-16), que já aplica o predicado de visibilidade (AD-19). Nenhum `WHERE
-- ativo` aqui. Uma consulta só para termo, filtros e ordenação: cada filtro é
-- opcional por `IS NULL OR`, e a lista e a contagem repetem o mesmo WHERE.
-- O termo chega normalizado e com `\`, `%` e `_` já escapados.

-- name: ListarVisiveis :many
SELECT id, nome, preco_centavos, imagem_url, estoque_disponivel
FROM catalogo.produto_visivel
WHERE (sqlc.narg(termo)::text IS NULL OR busca_normalizada LIKE '%' || sqlc.narg(termo)::text || '%')
  AND (sqlc.narg(categoria_id)::uuid IS NULL OR categoria_id = sqlc.narg(categoria_id)::uuid)
  AND (sqlc.narg(preco_min)::bigint IS NULL OR preco_centavos >= sqlc.narg(preco_min)::bigint)
  AND (sqlc.narg(preco_max)::bigint IS NULL OR preco_centavos <= sqlc.narg(preco_max)::bigint)
-- Uma chave por ordenação; as que não valem viram NULL e empatam. O
-- desempate final é sempre `id` (AD-18).
ORDER BY
  CASE WHEN @ordenacao::text = 'recentes' THEN criado_em END DESC,
  CASE WHEN @ordenacao::text = 'preco_asc' THEN preco_centavos END ASC,
  CASE WHEN @ordenacao::text = 'preco_desc' THEN preco_centavos END DESC,
  id
LIMIT @limite OFFSET @deslocamento;

-- name: ContarVisiveis :one
SELECT count(*)
FROM catalogo.produto_visivel
WHERE (sqlc.narg(termo)::text IS NULL OR busca_normalizada LIKE '%' || sqlc.narg(termo)::text || '%')
  AND (sqlc.narg(categoria_id)::uuid IS NULL OR categoria_id = sqlc.narg(categoria_id)::uuid)
  AND (sqlc.narg(preco_min)::bigint IS NULL OR preco_centavos >= sqlc.narg(preco_min)::bigint)
  AND (sqlc.narg(preco_max)::bigint IS NULL OR preco_centavos <= sqlc.narg(preco_max)::bigint);
