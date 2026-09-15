-- O Endereço do Comprador (FR-5). Mora em `identidade` porque o dono é o
-- Comprador, e a chave estrangeira aponta para a tabela AO LADO, no mesmo
-- schema: o AD-2 proíbe a travessia ENTRE schemas, não a integridade interna.
-- É ela que faz o Comprador apagado levar seus Endereços junto, sem código.
--
-- Nenhuma coluna de Endereço padrão ou preferido: "escolher" (FR-5) é a
-- listagem, e quem escolhe é o checkout da 5.2. O Pedido congela o Endereço na
-- criação (AD-3), e por isso remover aqui é DELETE de verdade — não há
-- desativação a filtrar para sempre.
-- +goose Up
CREATE TABLE identidade.endereco (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    comprador_id uuid NOT NULL REFERENCES identidade.comprador (id) ON DELETE CASCADE,
    destinatario text NOT NULL,
    -- Oito dígitos, sem hífen: a máscara é da tela. Uma forma só no banco é o
    -- que deixa a Faixa de Frete da 5.3 comparar por intervalo de texto sem
    -- normalizar a cada leitura. Como o CHECK do e-mail, a normalização é
    -- restrição de banco e não disciplina de chamador.
    cep text NOT NULL CHECK (cep ~ '^[0-9]{8}$'),
    logradouro text NOT NULL,
    numero text NOT NULL,
    -- O único opcional. Vazio, e não NULL: "sem complemento" e "complemento
    -- desconhecido" seriam a mesma coisa numa etiqueta de entrega, e o NULL só
    -- traria um ponteiro a mais em todo caminho de Go.
    complemento text NOT NULL DEFAULT '',
    bairro text NOT NULL,
    cidade text NOT NULL,
    -- Forma, e não só caixa: o regex completo, como o do CEP três linhas
    -- acima. `uf = upper(uf)` sozinho deixaria entrar '', '1' e 'SAOPAULO'.
    -- QUAIS são as 27 siglas continua sendo regra de quem valida.
    uf text NOT NULL CHECK (uf ~ '^[A-Z]{2}$')
);

-- A listagem do dono é a única leitura que existe — não há busca por CEP nem
-- por cidade, e nenhuma tela desta épica lê Endereço por outro caminho.
CREATE INDEX endereco_comprador_idx ON identidade.endereco (comprador_id);

-- +goose Down
DROP TABLE identidade.endereco;
