-- Uma consulta por módulo nesta estória: o que a 1.3 precisa provar é que o
-- sqlc analisa `DEFAULT uuidv7()` (verificação 2 do passo 0). As consultas de
-- verdade chegam com a regra, nas Épicas 2 e 3.

-- name: BuscarCompradorPorEmail :one
SELECT id, nome, email, senha_hash
FROM identidade.comprador
WHERE email = $1;

-- O molde é o de Comprador, e a tabela é outra de propósito: são duas consultas
-- porque são dois papéis, e o papel de quem entra é decidido pela rota que
-- chamou — nunca pela ordem em que as tabelas seriam tentadas.
-- name: BuscarAdministradorPorEmail :one
SELECT id, nome, email, senha_hash
FROM identidade.administrador
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
--
-- O RETURNING devolve o e-mail porque o contador de tentativas do bloqueio
-- chaveia por ele, e quem redefiniu a senha tem de sair do bloqueio: sem isto
-- a consulta seria :exec, e quem chama pagaria um SELECT a mais só para saber
-- de quem era o id que ele acabou de escrever. De quebra, o :one distingue o
-- UPDATE que não achou linha nenhuma, que o :exec engoliria em silêncio.
-- name: AtualizarSenhaDoComprador :one
UPDATE identidade.comprador
SET senha_hash = $2
WHERE id = $1
RETURNING email;
