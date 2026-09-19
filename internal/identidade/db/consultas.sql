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

-- As quatro do Endereço (2.5). As três que tocam uma linha específica trazem
-- `AND comprador_id = @comprador_id` DENTRO da cláusula, e não numa checagem
-- depois (AD-11): Endereço de outro dono e Endereço inexistente saem os dois
-- como "nenhuma linha", que é o mesmo 404 — não vaza existência.
--
-- A ordem é por `id`, e o id é uuidv7: monotônico no tempo, então ordenar por
-- ele é ordenar por criação sem carregar uma coluna de instante que nenhuma
-- tela exibe.
-- name: ListarEnderecos :many
SELECT id, destinatario, cep, logradouro, numero, complemento, bairro, cidade, uf
FROM identidade.endereco
WHERE comprador_id = $1
ORDER BY id;

-- Um Endereço só, do dono: é o que o Frete da 5.3 lê para tirar o CEP, e o
-- que a criação do Pedido (5.6) congela. O dono no WHERE, como nas outras.
-- name: BuscarEndereco :one
SELECT id, destinatario, cep, logradouro, numero, complemento, bairro, cidade, uf
FROM identidade.endereco
WHERE id = @id AND comprador_id = @comprador_id;

-- O teto por Comprador entra no PRÓPRIO INSERT, e não num SELECT count antes:
-- assim é uma ida ao banco só, e zero linhas devolvidas significa "o teto foi
-- alcançado" — o mesmo idioma do compare-and-swap do Pedido.
-- O CEP chega com os oito dígitos e a UF em maiúsculas: quem normaliza é
-- identidade.CriarEndereco, e as colunas têm os CHECKs.
-- name: CriarEndereco :one
INSERT INTO identidade.endereco
    (comprador_id, destinatario, cep, logradouro, numero, complemento, bairro, cidade, uf)
SELECT @comprador_id, @destinatario, @cep, @logradouro, @numero, @complemento, @bairro, @cidade, @uf
WHERE (SELECT count(*) FROM identidade.endereco WHERE comprador_id = @comprador_id) < @maximo::bigint
RETURNING id, destinatario, cep, logradouro, numero, complemento, bairro, cidade, uf;

-- name: AtualizarEndereco :one
UPDATE identidade.endereco
SET destinatario = @destinatario,
    cep = @cep,
    logradouro = @logradouro,
    numero = @numero,
    complemento = @complemento,
    bairro = @bairro,
    cidade = @cidade,
    uf = @uf
WHERE id = @id AND comprador_id = @comprador_id
RETURNING id, destinatario, cep, logradouro, numero, complemento, bairro, cidade, uf;

-- :execrows, e não :exec: zero linhas é a resposta de "não é seu ou não
-- existe", e o :exec a engoliria em silêncio devolvendo 204 para o DELETE de
-- Endereço alheio.
-- Nenhum Pedido é tocado — o Pedido congela o Endereço na criação (AD-3).
-- name: RemoverEndereco :execrows
DELETE FROM identidade.endereco
WHERE id = @id AND comprador_id = @comprador_id;
