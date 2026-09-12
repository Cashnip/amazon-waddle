-- Reserva de Estoque (AD-5) e o teto contra o qual a soma das reservas ativas
-- é comparada. `estoque_total` nasce com padrão para que a semente, o gerador
-- e o conjunto de medição não precisem mudar — a Épica 3 constrói o predicado
-- de visibilidade sobre a coluna já existente.
--
-- O estado é `text` com CHECK pelo mesmo motivo do Status do Pedido: nada de
-- enum do Postgres. `pedido_id` é uuid sem REFERENCES — a Reserva é de
-- `catalogo` e o Pedido é de `pedido`, e chave estrangeira cruzando schema é
-- proibida (AD-2).
-- +goose Up
ALTER TABLE catalogo.produto
    ADD COLUMN estoque_total integer NOT NULL DEFAULT 10 CHECK (estoque_total >= 0);

CREATE TABLE catalogo.reserva_estoque (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    produto_id uuid NOT NULL REFERENCES catalogo.produto (id),
    pedido_id uuid NOT NULL,
    quantidade integer NOT NULL CHECK (quantidade > 0),
    estado text NOT NULL CHECK (estado IN ('ATIVA', 'LIBERADA', 'CONSOLIDADA'))
);

-- Índice único parcial (AD-5): um Pedido tem no máximo uma Reserva ATIVA por
-- Produto. Parcial porque LIBERADA e CONSOLIDADA podem repetir — o histórico
-- de um Pedido que liberou e reservou de novo não é conflito.
CREATE UNIQUE INDEX reserva_estoque_ativa_idx
    ON catalogo.reserva_estoque (pedido_id, produto_id) WHERE estado = 'ATIVA';

-- A soma das reservas ativas é por Produto, e é ela que roda em toda compra.
CREATE INDEX reserva_estoque_produto_id_idx ON catalogo.reserva_estoque (produto_id);

-- +goose Down
DROP TABLE catalogo.reserva_estoque;
ALTER TABLE catalogo.produto DROP COLUMN estoque_total;
