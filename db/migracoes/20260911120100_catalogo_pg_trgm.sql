-- AD-16: esta é a PRIMEIRA migração de catalogo. A imagem do Postgres traz a
-- extensão no disco, não habilitada no banco; a coluna normalizada e o índice
-- GIN que a consomem são migração da estória 1.4.
-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- +goose Down
DROP EXTENSION IF EXISTS pg_trgm;
