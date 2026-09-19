-- A Regra de Frete é dado (AD-17): faixa de CEP → região → valor em centavos,
-- mais uma função pura em internal/pedido. O limiar de isenção NÃO mora aqui:
-- é o AZAMON_FRETE_ISENCAO_CENTAVOS da configuração (AD-13).
--
-- As linhas nascem nesta migração, e não em db/semente/: a semente roda uma
-- vez por marcador de versão, então um banco que já a tem nunca veria um
-- arquivo novo, e trocar a versão reexecuta os INSERT do Catálogo Semeado, que
-- colidem. A regra tem de existir em todo banco, testcontainers inclusive.
--
-- A região padrão é uma linha, e não configuração: o CEP fora de toda faixa
-- recebe o valor dela (FR-21), nunca erro. Uma linha só, sem faixa — o índice
-- único parcial garante que não existam duas.
-- +goose Up
CREATE TABLE pedido.faixa_frete (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    -- Oito dígitos, a mesma forma de identidade.endereco.cep: com o mesmo
    -- comprimento, a ordem do texto é a ordem numérica, e o intervalo compara
    -- sem normalizar.
    cep_inicio text CHECK (cep_inicio ~ '^[0-9]{8}$'),
    cep_fim text CHECK (cep_fim ~ '^[0-9]{8}$'),
    regiao text NOT NULL CHECK (regiao <> ''),
    -- Reais inteiros: o Provedor Simulado decide pelos centavos do total
    -- (PRD §7.1), e um Frete de ,90 transformaria todo Produto de ,00 numa
    -- recusa. O Frete não pode mexer nos centavos do total.
    valor_centavos bigint NOT NULL CHECK (valor_centavos >= 0 AND valor_centavos % 100 = 0),
    padrao boolean NOT NULL DEFAULT false,
    CHECK (
        (padrao AND cep_inicio IS NULL AND cep_fim IS NULL)
        OR (NOT padrao AND cep_inicio IS NOT NULL AND cep_fim IS NOT NULL AND cep_inicio <= cep_fim)
    ),
    -- Duas faixas nunca se sobrepõem: um CEP pertence a uma região só, e a
    -- ordem da consulta não precisa desempatar nada. A padrão fica de fora —
    -- sem faixa, o intervalo dela seria infinito e colidiria com todas.
    EXCLUDE USING gist (int8range(cep_inicio::bigint, cep_fim::bigint, '[]') WITH &&) WHERE (NOT padrao)
);

CREATE UNIQUE INDEX faixa_frete_uma_padrao_idx ON pedido.faixa_frete (padrao) WHERE padrao;

-- As faixas seguem o primeiro dígito do CEP dos Correios, sem sobreposição e
-- contíguas de 01000000 a 99999999. O que sobra (00000000–00999999) cai na
-- região padrão.
INSERT INTO pedido.faixa_frete (cep_inicio, cep_fim, regiao, valor_centavos, padrao) VALUES
    ('01000000', '39999999', 'Sudeste', 1500, false),
    ('40000000', '65999999', 'Nordeste', 3000, false),
    ('66000000', '69999999', 'Norte', 3500, false),
    ('70000000', '76799999', 'Centro-Oeste', 2500, false),
    ('76800000', '76999999', 'Norte', 3500, false),
    ('77000000', '77999999', 'Norte', 3500, false),
    ('78000000', '79999999', 'Centro-Oeste', 2500, false),
    ('80000000', '99999999', 'Sul', 2000, false),
    (NULL, NULL, 'Região padrão', 4000, true);

-- +goose Down
DROP TABLE pedido.faixa_frete;
