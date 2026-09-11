-- O marcador do Catálogo Semeado mora no public, ao lado do goose_db_version,
-- porque a semente não pertence a módulo nenhum. A inserção dele É a decisão
-- de semear: ON CONFLICT DO NOTHING resolve dois binários no mesmo arranque.
-- +goose Up
CREATE TABLE public.semente (
    versao text PRIMARY KEY
);

-- +goose Down
DROP TABLE public.semente;
