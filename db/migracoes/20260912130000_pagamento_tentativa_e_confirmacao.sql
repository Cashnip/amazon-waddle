-- Schema do módulo pagamento (AD-2): a Tentativa de Pagamento e a inbox de
-- confirmação. Resultado e estado são `text` com CHECK, e não enum do
-- Postgres, pelo mesmo motivo do Status do Pedido — `ALTER TYPE … ADD VALUE`
-- não roda dentro de transação.
--
-- `pedido_id` é uuid sem REFERENCES: chave estrangeira cruzando schema é
-- proibida (AD-2), e `pagamento` não conhece `pedido` (AD-7).
-- +goose Up
CREATE SCHEMA pagamento;

CREATE TABLE pagamento.tentativa_pagamento (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    pedido_id uuid NOT NULL,
    -- O total é congelado aqui: a porta recebe o valor e nunca consulta
    -- `pedido` para saber quanto cobrar.
    total_centavos bigint NOT NULL CHECK (total_centavos > 0),
    -- O identificador que o Provedor conhece. É por ele que a confirmação
    -- volta, e é ele que o Simulado deriva do Pedido e do número da Tentativa.
    id_externo text NOT NULL UNIQUE,
    numero integer NOT NULL CHECK (numero > 0),
    criada_em timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX tentativa_pagamento_pedido_id_idx ON pagamento.tentativa_pagamento (pedido_id);

-- A emissão derivada filtra e ordena por criada_em, e roda a cada tique da
-- varredura. Sem este índice é varredura sequencial uma vez por segundo, e ela
-- cresce com o histórico de Tentativas.
CREATE INDEX tentativa_pagamento_criada_em_idx ON pagamento.tentativa_pagamento (criada_em);

-- A inbox. A restrição única sobre a chave de idempotência é o que transforma
-- a confirmação repetida — reenvio do Provedor ou reemissão depois de um
-- reinício — em no-op: o 23505 é sinal para o código decidir, nunca resposta
-- ao cliente.
CREATE TABLE pagamento.confirmacao_recebida (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tentativa_id uuid NOT NULL REFERENCES pagamento.tentativa_pagamento (id),
    chave_idempotencia text NOT NULL UNIQUE,
    resultado text NOT NULL CHECK (resultado IN ('APROVADO', 'RECUSADO')),
    -- Toda confirmação tem estado terminal: PENDENTE vira APLICADA ou
    -- NAO_APLICAVEL_SINALIZADA. A varredura decide um dos dois em toda
    -- passagem em que consegue travar o Pedido; enquanto não consegue, a
    -- linha fica PENDENTE e o tique seguinte tenta de novo.
    estado text NOT NULL CHECK (estado IN ('PENDENTE', 'APLICADA', 'NAO_APLICAVEL_SINALIZADA')),
    recebida_em timestamptz NOT NULL DEFAULT now()
);

-- Parcial: quem varre só procura o que ainda não foi aplicado, e o índice
-- encolhe junto com a fila em vez de crescer com o histórico.
CREATE INDEX confirmacao_recebida_pendente_idx
    ON pagamento.confirmacao_recebida (recebida_em) WHERE estado = 'PENDENTE';

CREATE INDEX confirmacao_recebida_tentativa_id_idx
    ON pagamento.confirmacao_recebida (tentativa_id);

-- +goose Down
DROP SCHEMA pagamento CASCADE;
