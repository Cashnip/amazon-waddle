-- Schema do módulo pedido (AD-2). O Status é `text` com CHECK e não enum do
-- Postgres: não existe CREATE TYPE no repositório, e `ALTER TYPE … ADD VALUE`
-- não roda dentro de transação — o que faria de cada estória que acrescenta
-- estado uma migração de forma diferente de todas as outras. Os sete Status já
-- são conhecidos, então o CHECK nasce completo e a Épica 5 só passa a usá-los.
--
-- `comprador_id` e `produto_id` são uuid sem REFERENCES: chave estrangeira
-- cruzando schema é proibida (AD-2), e as duas apontam para fora daqui.
-- +goose Up
CREATE SCHEMA pedido;

CREATE TABLE pedido.pedido (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    -- Legível e separado do uuid (AD-6): AZ-<ano>-<6 dígitos>.
    numero text NOT NULL UNIQUE,
    comprador_id uuid NOT NULL,
    status text NOT NULL CHECK (status IN (
        'AGUARDANDO_PAGAMENTO',
        'PAGAMENTO_RECUSADO',
        'PAGO',
        'EM_SEPARACAO',
        'ENVIADO',
        'ENTREGUE',
        'CANCELADO'
    )),
    -- Coluna com CHECK, nunca derivação de leitura (AD-3, NFR-13). Nenhuma
    -- divisão no caminho monetário: o valor é int64 de centavos ponta a ponta.
    total_centavos bigint NOT NULL CHECK (total_centavos > 0)
);

CREATE INDEX pedido_comprador_id_idx ON pedido.pedido (comprador_id);

CREATE TABLE pedido.item_pedido (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    pedido_id uuid NOT NULL REFERENCES pedido.pedido (id),
    produto_id uuid NOT NULL,
    -- Congelados no instante da compra: o Produto pode mudar de nome, de preço
    -- ou de Vendedor depois, e o Pedido não muda junto.
    nome text NOT NULL,
    vendedor_nome text NOT NULL,
    preco_praticado_centavos bigint NOT NULL CHECK (preco_praticado_centavos > 0),
    quantidade integer NOT NULL CHECK (quantidade > 0)
);

CREATE INDEX item_pedido_pedido_id_idx ON pedido.item_pedido (pedido_id);

-- NFR-9: toda transição registrada com estado anterior, novo, autor e
-- instante. No nascimento o anterior é vazio, porque não havia estado antes.
CREATE TABLE pedido.transicao_status (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    pedido_id uuid NOT NULL REFERENCES pedido.pedido (id),
    status_anterior text NOT NULL,
    status_novo text NOT NULL,
    autor text NOT NULL,
    ocorrido_em timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX transicao_status_pedido_id_idx ON pedido.transicao_status (pedido_id);

-- Contador por ano em vez de SEQUENCE: uma sequência por ano exigiria
-- CREATE SEQUENCE dinâmico dentro do caso de uso. O
-- `INSERT … ON CONFLICT DO UPDATE … RETURNING` resolve na mesma transação, sem
-- DDL em tempo de execução e sem buraco na numeração quando a transação é
-- revertida. O custo é serializar a criação de Pedidos do mesmo ano numa linha.
CREATE TABLE pedido.contador_numero (
    ano integer PRIMARY KEY,
    ultimo bigint NOT NULL DEFAULT 0
);

-- +goose Down
DROP SCHEMA pedido CASCADE;
