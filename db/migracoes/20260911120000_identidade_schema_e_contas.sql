-- Um schema Postgres por módulo (AD-2). Comprador e Administrador são duas
-- tabelas separadas de propósito: não existe coluna de papel que promova um
-- Comprador, e o Administrador nasce semeado (FR-4).
-- +goose Up
CREATE SCHEMA identidade;

CREATE TABLE identidade.comprador (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    nome text NOT NULL,
    -- A normalização é restrição de banco, não disciplina de chamador: com o
    -- CHECK, "Ana@Exemplo.br" não entra por caminho nenhum e o UNIQUE passa a
    -- valer sobre a forma normalizada de verdade.
    email text NOT NULL UNIQUE CHECK (email = lower(email)),
    -- Argon2id no formato PHC ($argon2id$v=19$m=19456,t=2,p=1$sal$hash).
    senha_hash text NOT NULL
);

CREATE TABLE identidade.administrador (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    nome text NOT NULL,
    email text NOT NULL UNIQUE CHECK (email = lower(email)),
    senha_hash text NOT NULL
);

-- +goose Down
DROP SCHEMA identidade CASCADE;
