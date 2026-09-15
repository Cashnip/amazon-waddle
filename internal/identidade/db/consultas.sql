-- Uma consulta por módulo nesta estória: o que a 1.3 precisa provar é que o
-- sqlc analisa `DEFAULT uuidv7()` (verificação 2 do passo 0). As consultas de
-- verdade chegam com a regra, nas Épicas 2 e 3.

-- name: BuscarCompradorPorEmail :one
SELECT id, nome, email, senha_hash
FROM identidade.comprador
WHERE email = $1;

-- O e-mail chega já normalizado: quem grava é identidade.Cadastrar, e a coluna
-- tem CHECK (email = lower(email)). A duplicidade sai da violação do UNIQUE —
-- um SELECT antes do INSERT deixaria dois cadastros simultâneos passarem.
-- name: CriarComprador :one
INSERT INTO identidade.comprador (nome, email, senha_hash)
VALUES ($1, $2, $3)
RETURNING id, nome;

-- Quem chama já resolveu o token de redefinição no Redis, e é ele que provou
-- de quem é a conta — por isso a cláusula é pelo id, e não pelo e-mail.
-- name: AtualizarSenhaDoComprador :exec
UPDATE identidade.comprador
SET senha_hash = $2
WHERE id = $1;
