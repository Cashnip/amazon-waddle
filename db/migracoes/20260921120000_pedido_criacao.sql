-- A criação do Pedido a partir do Carrinho (5.6). Tudo que o Pedido congela
-- na criação vira coluna de pedido.pedido, escrita pelo INSERT e protegida
-- pelo gatilho da 5.1 (`to_jsonb(NEW) - 'status'`): coluna nova nasce
-- protegida sem ninguém lembrar dele.
--
-- 1. Subtotal e Frete ao lado do total, com o CHECK do AD-9: o total é a soma
--    das duas parcelas, e nunca derivação de leitura. As linhas antigas — só
--    Pedido do esqueleto, de um Produto e sem Frete — recebem subtotal = total
--    e Frete zero, com o gatilho desligado só no preenchimento.
-- 2. As oito colunas do Endereço, copiadas de identidade.endereco: o Pedido não
--    depende de o Endereço continuar existindo (FR-23). Sem REFERENCES — a
--    chave estrangeira cruzando schema é proibida (AD-2).
-- 3. A chave de idempotência e o digest do corpo (AD-7, NFR-12). O índice
--    único é por Comprador: a chave é dele, e duas pessoas nunca colidem. O
--    INSERT do Pedido é a reivindicação da chave — o gêmeo concorrente espera
--    neste índice.
-- 4. Item de Pedido só nasce antes da primeira linha do histórico: depois
--    dela o Pedido existe, e o conteúdo dele é escrito uma vez (fecha o INSERT
--    tardio que a 5.1 deixou aberto).
--
-- Endereço, chave e digest são anuláveis porque o Pedido do esqueleto não os
-- tem; o CHECK `pedido_criacao_completa` exige que venham todos ou nenhum.
-- +goose Up
ALTER TABLE pedido.pedido
    ADD COLUMN subtotal_centavos bigint,
    ADD COLUMN frete_centavos bigint,
    ADD COLUMN endereco_destinatario text,
    ADD COLUMN endereco_cep text,
    ADD COLUMN endereco_logradouro text,
    ADD COLUMN endereco_numero text,
    ADD COLUMN endereco_complemento text,
    ADD COLUMN endereco_bairro text,
    ADD COLUMN endereco_cidade text,
    ADD COLUMN endereco_uf text,
    ADD COLUMN chave_idempotencia uuid,
    ADD COLUMN digest_corpo text;

-- O preenchimento reescreve linha existente, e o gatilho da 5.1 a recusaria:
-- desligado só aqui, e religado logo depois, dentro da mesma migração.
ALTER TABLE pedido.pedido DISABLE TRIGGER pedido_conteudo_imutavel;
UPDATE pedido.pedido SET subtotal_centavos = total_centavos, frete_centavos = 0;
ALTER TABLE pedido.pedido ENABLE TRIGGER pedido_conteudo_imutavel;

ALTER TABLE pedido.pedido
    ALTER COLUMN subtotal_centavos SET NOT NULL,
    ALTER COLUMN frete_centavos SET NOT NULL,
    ADD CONSTRAINT pedido_subtotal_positivo CHECK (subtotal_centavos > 0),
    ADD CONSTRAINT pedido_frete_nao_negativo CHECK (frete_centavos >= 0),
    -- AD-9: o total fecha ao centavo com as parcelas, no próprio banco.
    ADD CONSTRAINT pedido_total_e_a_soma CHECK (total_centavos = subtotal_centavos + frete_centavos),
    -- Tudo ou nada: um Pedido criado pelo checkout tem Endereço, chave e digest;
    -- o do esqueleto não tem nenhum. Meio-termo seria defeito de quem gravou.
    ADD CONSTRAINT pedido_criacao_completa CHECK (num_nulls(
        endereco_destinatario, endereco_cep, endereco_logradouro, endereco_numero,
        endereco_complemento, endereco_bairro, endereco_cidade, endereco_uf,
        chave_idempotencia, digest_corpo) IN (0, 10));

-- A chave é do Comprador. NULL não colide com NULL, e é isso que deixa os
-- Pedidos do esqueleto conviverem com o índice.
CREATE UNIQUE INDEX pedido_comprador_chave_idx ON pedido.pedido (comprador_id, chave_idempotencia);

-- +goose StatementBegin
CREATE FUNCTION pedido.recusar_item_tardio() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM pedido.transicao_status WHERE pedido_id = NEW.pedido_id) THEN
        RAISE EXCEPTION 'O conteúdo do Pedido é escrito uma vez, na criação.' USING ERRCODE = 'restrict_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER item_pedido_so_na_criacao
    BEFORE INSERT ON pedido.item_pedido
    FOR EACH ROW EXECUTE FUNCTION pedido.recusar_item_tardio();

-- +goose Down
DROP TRIGGER item_pedido_so_na_criacao ON pedido.item_pedido;
DROP FUNCTION pedido.recusar_item_tardio();
DROP INDEX pedido.pedido_comprador_chave_idx;
ALTER TABLE pedido.pedido
    DROP CONSTRAINT pedido_criacao_completa,
    DROP CONSTRAINT pedido_total_e_a_soma,
    DROP CONSTRAINT pedido_frete_nao_negativo,
    DROP CONSTRAINT pedido_subtotal_positivo,
    DROP COLUMN digest_corpo,
    DROP COLUMN chave_idempotencia,
    DROP COLUMN endereco_uf,
    DROP COLUMN endereco_cidade,
    DROP COLUMN endereco_bairro,
    DROP COLUMN endereco_complemento,
    DROP COLUMN endereco_numero,
    DROP COLUMN endereco_logradouro,
    DROP COLUMN endereco_cep,
    DROP COLUMN endereco_destinatario,
    DROP COLUMN frete_centavos,
    DROP COLUMN subtotal_centavos;
