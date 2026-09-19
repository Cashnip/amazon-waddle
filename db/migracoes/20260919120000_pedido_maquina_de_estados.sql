-- A máquina de estados do Pedido completa (5.1). Quatro mudanças no schema
-- `pedido`, todas sobre o mesmo par de tabelas, e por isso numa migração só:
--
-- 1. `EM_SEPARACAO` vira `SEPARANDO`. O glossário do PRD, o AD-3, a
--    EXPERIENCE e a SPEC escrevem `SEPARANDO`; o esqueleto da Épica 1 é que
--    divergia (retro da Épica 1, item A1, decidido em 2026-09-19 na direção
--    do glossário).
-- 2. `motivo` no histórico — é onde `TEMPO_ESGOTADO` (FR-34) e o motivo da
--    recusa (FR-27) moram.
-- 3. `autor` vira `ator`, com os cinco valores que a tabela de transições
--    conhece. O Comprador que criou o Pedido já está em `pedido.comprador_id`,
--    então trocar o uuid por `COMPRADOR` não perde nada.
-- 4. Conteúdo e histórico imutáveis no próprio banco: do Pedido só o Status
--    muda; Item de Pedido e transição não mudam nem somem. É a prova de
--    "escrito uma vez" (AD-3) e de "imutável" (FR-28) que o Go sozinho não dá.
--
-- Uma migração futura que precise reescrever linha antiga destas tabelas —
-- o preenchimento de uma coluna nova, por exemplo — desliga o gatilho dentro
-- dela mesma (`ALTER TABLE … DISABLE TRIGGER`, e `ENABLE` no fim): esta
-- reescreve o Status antes de criá-los pelo mesmo motivo.
-- +goose Up
ALTER TABLE pedido.pedido DROP CONSTRAINT pedido_status_check;
UPDATE pedido.pedido SET status = 'SEPARANDO' WHERE status = 'EM_SEPARACAO';
ALTER TABLE pedido.pedido ADD CONSTRAINT pedido_status_check CHECK (status IN (
    'AGUARDANDO_PAGAMENTO',
    'PAGAMENTO_RECUSADO',
    'PAGO',
    'SEPARANDO',
    'ENVIADO',
    'ENTREGUE',
    'CANCELADO'
));

UPDATE pedido.transicao_status SET status_anterior = 'SEPARANDO' WHERE status_anterior = 'EM_SEPARACAO';
UPDATE pedido.transicao_status SET status_novo = 'SEPARANDO' WHERE status_novo = 'EM_SEPARACAO';
-- O anterior vazio é o nascimento: não havia estado antes.
ALTER TABLE pedido.transicao_status ADD CONSTRAINT transicao_status_status_anterior_check
    CHECK (status_anterior IN ('', 'AGUARDANDO_PAGAMENTO', 'PAGAMENTO_RECUSADO', 'PAGO',
                               'SEPARANDO', 'ENVIADO', 'ENTREGUE', 'CANCELADO'));
ALTER TABLE pedido.transicao_status ADD CONSTRAINT transicao_status_status_novo_check
    CHECK (status_novo IN ('AGUARDANDO_PAGAMENTO', 'PAGAMENTO_RECUSADO', 'PAGO',
                           'SEPARANDO', 'ENVIADO', 'ENTREGUE', 'CANCELADO'));

-- NULL quando a transição não tem motivo a registrar: nem toda tem.
ALTER TABLE pedido.transicao_status ADD COLUMN motivo text;

-- Só três formas de autor existiram: os dois nomes fixos e o uuid do
-- Comprador no nascimento. Qualquer outro valor vira NULL, que a coluna
-- NOT NULL desde a Épica 1 recusa, e a migração inteira reverte — melhor a migração falhar alto do que o histórico imutável ganhar
-- um COMPRADOR que ninguém sabe se foi.
ALTER TABLE pedido.transicao_status RENAME COLUMN autor TO ator;
UPDATE pedido.transicao_status SET ator = CASE
    WHEN ator = 'provedor-pagamento' THEN 'PROVEDOR'
    WHEN ator = 'simulacao-entrega' THEN 'SIMULACAO'
    WHEN ator ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN 'COMPRADOR'
END;
ALTER TABLE pedido.transicao_status ADD CONSTRAINT transicao_status_ator_check
    CHECK (ator IN ('COMPRADOR', 'PROVEDOR', 'VARREDURA', 'ADMINISTRADOR', 'SIMULACAO'));

-- O índice que a leitura do histórico em ordem usa, e o que o `ponytail:` de
-- PedidosParaAvancar já nomeava. Ele cobre o de `pedido_id` sozinho, que sai.
CREATE INDEX transicao_status_pedido_id_ocorrido_em_idx
    ON pedido.transicao_status (pedido_id, ocorrido_em);
DROP INDEX pedido.transicao_status_pedido_id_idx;

-- Do Pedido, só o Status muda. A comparação é da linha inteira menos o
-- Status, e não coluna a coluna: a coluna que a 5.3 e a 5.6 acrescentarem
-- (subtotal, Frete, Endereço) nasce protegida sem ninguém lembrar deste
-- gatilho — e é por isso que o preenchimento dela, se houver, o desliga.
-- +goose StatementBegin
CREATE FUNCTION pedido.recusar_mudanca_de_conteudo() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'O Pedido não é apagado.' USING ERRCODE = 'restrict_violation';
    END IF;
    IF (to_jsonb(NEW) - 'status') IS DISTINCT FROM (to_jsonb(OLD) - 'status') THEN
        RAISE EXCEPTION 'O conteúdo do Pedido é escrito uma vez, na criação.' USING ERRCODE = 'restrict_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION pedido.recusar_mudanca() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% é imutável.', TG_TABLE_NAME USING ERRCODE = 'restrict_violation';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER pedido_conteudo_imutavel
    BEFORE UPDATE OR DELETE ON pedido.pedido
    FOR EACH ROW EXECUTE FUNCTION pedido.recusar_mudanca_de_conteudo();

CREATE TRIGGER item_pedido_imutavel
    BEFORE UPDATE OR DELETE ON pedido.item_pedido
    FOR EACH ROW EXECUTE FUNCTION pedido.recusar_mudanca();

CREATE TRIGGER transicao_status_imutavel
    BEFORE UPDATE OR DELETE ON pedido.transicao_status
    FOR EACH ROW EXECUTE FUNCTION pedido.recusar_mudanca();

-- Gatilho de linha não dispara no TRUNCATE, que apagaria as três de uma vez.
CREATE TRIGGER pedido_sem_truncate
    BEFORE TRUNCATE ON pedido.pedido
    FOR EACH STATEMENT EXECUTE FUNCTION pedido.recusar_mudanca();
CREATE TRIGGER item_pedido_sem_truncate
    BEFORE TRUNCATE ON pedido.item_pedido
    FOR EACH STATEMENT EXECUTE FUNCTION pedido.recusar_mudanca();
CREATE TRIGGER transicao_status_sem_truncate
    BEFORE TRUNCATE ON pedido.transicao_status
    FOR EACH STATEMENT EXECUTE FUNCTION pedido.recusar_mudanca();

-- O Down perde o que a Épica 1 não sabia guardar: o `motivo` some, os atores
-- VARREDURA e ADMINISTRADOR e o uuid do Comprador do nascimento não têm
-- grafia antiga (ficam como estão), e só PROVEDOR e SIMULACAO voltam aos
-- nomes que o código anterior lia.
-- +goose Down
DROP TRIGGER transicao_status_sem_truncate ON pedido.transicao_status;
DROP TRIGGER item_pedido_sem_truncate ON pedido.item_pedido;
DROP TRIGGER pedido_sem_truncate ON pedido.pedido;
DROP TRIGGER transicao_status_imutavel ON pedido.transicao_status;
DROP TRIGGER item_pedido_imutavel ON pedido.item_pedido;
DROP TRIGGER pedido_conteudo_imutavel ON pedido.pedido;
DROP FUNCTION pedido.recusar_mudanca();
DROP FUNCTION pedido.recusar_mudanca_de_conteudo();

CREATE INDEX transicao_status_pedido_id_idx ON pedido.transicao_status (pedido_id);
DROP INDEX pedido.transicao_status_pedido_id_ocorrido_em_idx;

ALTER TABLE pedido.transicao_status DROP CONSTRAINT transicao_status_ator_check;
ALTER TABLE pedido.transicao_status RENAME COLUMN ator TO autor;
UPDATE pedido.transicao_status SET autor = CASE autor
    WHEN 'PROVEDOR' THEN 'provedor-pagamento'
    WHEN 'SIMULACAO' THEN 'simulacao-entrega'
    ELSE autor
END;
ALTER TABLE pedido.transicao_status DROP COLUMN motivo;

ALTER TABLE pedido.transicao_status DROP CONSTRAINT transicao_status_status_novo_check;
ALTER TABLE pedido.transicao_status DROP CONSTRAINT transicao_status_status_anterior_check;
UPDATE pedido.transicao_status SET status_anterior = 'EM_SEPARACAO' WHERE status_anterior = 'SEPARANDO';
UPDATE pedido.transicao_status SET status_novo = 'EM_SEPARACAO' WHERE status_novo = 'SEPARANDO';

ALTER TABLE pedido.pedido DROP CONSTRAINT pedido_status_check;
UPDATE pedido.pedido SET status = 'EM_SEPARACAO' WHERE status = 'SEPARANDO';
ALTER TABLE pedido.pedido ADD CONSTRAINT pedido_status_check CHECK (status IN (
    'AGUARDANDO_PAGAMENTO',
    'PAGAMENTO_RECUSADO',
    'PAGO',
    'EM_SEPARACAO',
    'ENVIADO',
    'ENTREGUE',
    'CANCELADO'
));
