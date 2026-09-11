-- AD-16: Coluna normalizada na escrita e índice GIN pg_trgm para busca textual (NFR-4).
-- Índices nas chaves estrangeiras locais vendedor_id e categoria_id (adiados na 1.3).
-- +goose Up
ALTER TABLE catalogo.produto ADD COLUMN busca_normalizada text NOT NULL DEFAULT '';

UPDATE catalogo.produto SET busca_normalizada = lower(nome || ' ' || descricao) WHERE busca_normalizada = '';

CREATE INDEX produto_busca_normalizada_idx ON catalogo.produto USING gin (busca_normalizada gin_trgm_ops);
CREATE INDEX produto_vendedor_id_idx ON catalogo.produto (vendedor_id);
CREATE INDEX produto_categoria_id_idx ON catalogo.produto (categoria_id);

-- +goose Down
DROP INDEX IF EXISTS catalogo.produto_categoria_id_idx;
DROP INDEX IF EXISTS catalogo.produto_vendedor_id_idx;
DROP INDEX IF EXISTS catalogo.produto_busca_normalizada_idx;
ALTER TABLE catalogo.produto DROP COLUMN IF EXISTS busca_normalizada;
