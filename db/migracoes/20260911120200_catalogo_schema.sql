-- Nenhuma chave estrangeira atravessa schema (AD-2): as três tabelas daqui só
-- referenciam umas às outras. Estoque, Reserva de Estoque e o predicado de
-- visibilidade do AD-19 são da Épica 3 e não aparecem.
-- +goose Up
CREATE SCHEMA catalogo;

CREATE TABLE catalogo.vendedor (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    nome text NOT NULL UNIQUE,
    -- FR-8 desativa, nunca remove: Vendedor com Produto não pode sumir sem
    -- levar junto os Pedidos que o referenciam.
    ativo boolean NOT NULL DEFAULT true
);

CREATE TABLE catalogo.categoria (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    nome text NOT NULL UNIQUE,
    -- Addendum §6: nasce nula e não exposta. Hierarquia depois é preencher a
    -- coluna, não migrar dados de um sistema que já tem Pedidos.
    categoria_pai_id uuid REFERENCES catalogo.categoria (id)
);

CREATE TABLE catalogo.produto (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    nome text NOT NULL,
    descricao text NOT NULL,
    preco_centavos bigint NOT NULL CHECK (preco_centavos > 0),
    -- URL relativa (AD-12): quem serve o byte é o Go, de arquivo embutido.
    imagem_url text NOT NULL,
    vendedor_id uuid NOT NULL REFERENCES catalogo.vendedor (id),
    categoria_id uuid NOT NULL REFERENCES catalogo.categoria (id)
);

-- +goose Down
DROP SCHEMA catalogo CASCADE;
