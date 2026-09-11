-- Uma consulta por módulo nesta estória: o que a 1.3 precisa provar é que o
-- sqlc analisa `DEFAULT uuidv7()` (verificação 2 do passo 0). As consultas de
-- verdade chegam com a regra, nas Épicas 2 e 3.

-- name: BuscarCompradorPorEmail :one
SELECT id, nome, email, senha_hash
FROM identidade.comprador
WHERE email = $1;
