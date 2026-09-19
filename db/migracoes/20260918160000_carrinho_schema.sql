-- Schema do módulo carrinho (AD-2). O Carrinho vive aqui, e não no Redis: não
-- expira sozinho e sobrevive a reinício. Um por Comprador (FR-16), e o UNIQUE
-- é quem garante — não a disciplina de quem chama.
--
-- `comprador_id` e `produto_id` são uuid sem REFERENCES: apontam para fora
-- deste schema, e FK cruzando schema é proibida (AD-2). A única FK é interna.
-- +goose Up
CREATE SCHEMA carrinho;

CREATE TABLE carrinho.carrinho (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    comprador_id uuid NOT NULL UNIQUE
);

CREATE TABLE carrinho.item_carrinho (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    carrinho_id uuid NOT NULL REFERENCES carrinho.carrinho (id) ON DELETE CASCADE,
    produto_id uuid NOT NULL,
    quantidade int NOT NULL CHECK (quantidade > 0),
    -- O preço na última alteração (AD-2): é o que deixa a 4.4 dizer "de X
    -- para Y". Não congela nada — o Pedido lê o preço atual.
    preco_visto_centavos bigint NOT NULL CHECK (preco_visto_centavos >= 0),
    -- O Produto repetido soma ao Item existente (FR-17): nunca duas linhas.
    UNIQUE (carrinho_id, produto_id)
);

-- +goose Down
DROP SCHEMA carrinho CASCADE;
