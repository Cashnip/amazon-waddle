---
title: 'Estória 1.4 — A medição do NFR-4 com 5.000 Produtos'
type: 'feature'
created: '2026-09-11'
status: 'review'
baseline_commit: '87d97ee82314deff7b0e2b62ebe4c86b6d2f6ff9'
review_loop_iteration: 0
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A verificação 4 do passo 0 (NFR-4 / SM-4) continua aberta: é necessário comprovar que a busca com filtros sobre 5.000 Produtos semeados responde em p95 ≤ 500 ms usando índice GIN `pg_trgm` no PostgreSQL 18, confirmando a decisão "índice antes de serviço" (sem Elasticsearch e sem cache). Além disso, `catalogo.produto` não possui a coluna de busca normalizada na escrita e nasceu com `vendedor_id` e `categoria_id` sem índices (adiados na Estória 1.3).

**Approach:** Criar migração goose que adiciona `busca_normalizada` em `catalogo.produto` com índice GIN `pg_trgm`, além de índices B-tree em `vendedor_id` e `categoria_id`. Implementar a normalização de texto no Go (sem acento e minúscula), atualizar o gerador do Catálogo Semeado (`media/gerar.go`), adicionar suporte configurável para o conjunto de 5.000 Produtos ativado por `AZAMON_SEMENTE_GRANDE=true` (preservando o catálogo de demonstração de 50 itens por padrão conforme SM-C3), executar medição de `EXPLAIN ANALYZE` sobre consulta com termo, filtro de categoria e faixa de preço, e registrar o desfecho da verificação 4 no `addendum.md` §10 e no `.memlog.md`.

## Boundaries & Constraints

**Always:**
- Um schema Postgres por módulo (AD-2); chave estrangeira cruzando schema proibida.
- Normalização (minúscula, sem acento, remoção de diacríticos) executada na escrita pelo Go (AD-16), nunca por função de banco não imutável.
- Índice GIN `pg_trgm` criado sobre a coluna normalizada `catalogo.produto(busca_normalizada gin_trgm_ops)`.
- Índices B-tree criados sobre as chaves estrangeiras locais `catalogo.produto(vendedor_id)` e `catalogo.produto(categoria_id)` resolvendo o trabalho adiado na 1.3.
- O conjunto de 5.000 Produtos é ativado sob configuração (`AZAMON_SEMENTE_GRANDE=true`), nunca substituindo o catálogo padrão de demonstração de 50 Produtos (SM-C3).
- Variável `AZAMON_SEMENTE_GRANDE` declarada em `internal/plataforma/config.go` e presente no `.env` versionado com padrão `false` (validada por `env_test.go`).
- Medição do NFR-4 executada via `EXPLAIN ANALYZE` sobre a consulta com termo de busca, filtro de categoria e faixa de preço, aferindo p95 ≤ 500 ms (NFR-4, SM-4).
- O desfecho da verificação 4 (número medido e confirmação de "índice antes de serviço") é registrado no `addendum.md` §10 e apensado no `.memlog.md` via script canônico.

**Ask First:**
- Nenhuma decisão aberta requer aprovação intermediária além do checkpoint de aprovação da especificação.

**Never:**
- Não usar Elasticsearch nem introduzir cache Redis para a busca (a decisão da espinha e do addendum é índice antes de serviço).
- Não transformar o catálogo padrão de demonstração (`AZAMON_SEMENTE_GRANDE=false`) em 5.000 Produtos (SM-C3).
- Não criar tabelas fora do escopo desta estória (Pedido, Reserva e Tentativa pertencem a 1.6 e 1.7).
- Não implementar busca completa com paginação e endpoints públicos de UI (pertencem à Épica 3 — aqui a consulta é sonda de medição e benchmark).

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Consulta com termo, categoria e preço com 5.000 Produtos | Categoria válida, faixa min/max, termo ≥ 3 chars ("fone", "livro") | `EXPLAIN ANALYZE` executa com plano indexado, p95 ≤ 500 ms | N/A |
| Consulta com termo curto (< 3 chars) | Categoria válida, faixa min/max, termo com 1-2 caracteres ("ar", "ca") | Executa em ≤ 500 ms mesmo com varredura (tabela cabe em memória), p95 ≤ 500 ms comprovado | N/A |
| Arranque em modo demonstração padrão (`AZAMON_SEMENTE_GRANDE=false`) | Banco novo ou recriado | Semeia exatamente 50 Produtos, 5 Categorias, 5 Vendedores e 2 Contas | Falha de semente derruba o arranque |
| Arranque com semente grande ativa (`AZAMON_SEMENTE_GRANDE=true`) | Banco novo com flag ligada | Semeia os 5.000 Produtos de forma determinística e idempotente | Transação única, rollback em caso de falha |
| Normalização com caracteres especiais e acentos | Strings com acentuação PT-BR ("Café Torrado & Moído", "Açúcar") | `busca_normalizada` em minúsculas e sem acento ("cafe torrado & moido", "acucar") | N/A |
| `sqlc generate` sobre as migrações atualizadas | `db/migracoes/*_catalogo_*.sql` | Analisa a nova coluna e índices e gera `internal/catalogo/db/gerado/` sem erro | N/A |

</frozen-after-approval>

## Code Map

- `db/migracoes/20260911140000_catalogo_busca_e_indices.sql` -- Nova migração goose adicionando `busca_normalizada text NOT NULL DEFAULT ''`, índice GIN `pg_trgm` sobre `busca_normalizada gin_trgm_ops`, e índices B-tree sobre `vendedor_id` e `categoria_id`.
- `internal/catalogo/normalizar.go` -- Função de normalização de texto em Go (converte para minúsculas e remove acentos/diacríticos) para uso na escrita de Produtos.
- `internal/catalogo/normalizar_test.go` -- Testes unitários para a função de normalização cobrindo acentos do português, pontuação e maiúsculas.
- `media/gerar.go` -- Atualização do gerador determinístico do Catálogo Semeado para preencher a coluna `busca_normalizada` nos 50 produtos da demonstração.
- `db/semente/001_catalogo_semeado.sql` -- Arquivo regenerado com a coluna `busca_normalizada` preenchida nos INSERTs.
- `internal/plataforma/config.go` -- Adição da variável `AZAMON_SEMENTE_GRANDE` (booleano) na struct `Config` e leitor de configuração.
- `.env` -- Declaração de `AZAMON_SEMENTE_GRANDE=false` mantendo a sincronia exigida por `internal/plataforma/env_test.go`.
- `db/semente-grande/001_catalogo_grande.sql` -- Conjunto de medição: 5.000 Produtos por `CROSS JOIN` de 10 linhas × 10 séries sobre os 50 do Catálogo Semeado, em semente e marcador separados. Substitui o `db/semente_grande.go` previsto: `db` não pode importar `internal/catalogo` (AD-1), e 5.000 `INSERT` literais custariam ~1,6 MB de SQL versionado pelo mesmo efeito.
- `cmd/azamon/main.go` -- Segunda chamada a `plataforma.Semear`, com o `fs.FS` e o marcador do conjunto de medição, quando `cfg.SementeGrande` estiver ligada.
- `db/medicao_nfr4_test.go` -- Teste com `testcontainers-go` que aplica migrações, semeia os dois conjuntos, roda `ANALYZE` e mede 100 execuções de duas sondas sob `EXPLAIN (ANALYZE, FORMAT JSON)` — termo + Categoria + faixa de preço, e só termo —, achata a árvore do plano e valida p95 ≤ 500 ms (NFR-4).
- `db/schema_test.go` -- Atualização dos testes de schema para verificar a existência da coluna `busca_normalizada`, do índice GIN `pg_trgm` e dos índices em FKs.
- `internal/catalogo/db/consultas.sql` -- `BuscarProdutoPorID` passou a trazer `busca_normalizada`. A sonda de medição NÃO virou consulta sqlc: ela mora no teste, e a busca de verdade é da Épica 3 — gerar agora código que ninguém chama seria código morto.
- `internal/catalogo/db/gerado/` -- Arquivos regenerados pelo `sqlc generate`.
- `_bmad-output/implementation-artifacts/deferred-work.md` -- Atualização do ledger marcando o item de índices de vendedor_id e categoria_id como resolvido nesta estória.
- `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` §10 -- Registro do resultado da verificação 4 (p95 medido, confirmação do índice GIN vs Elasticsearch).

## Tasks & Acceptance

**Execution:**
- [x] `internal/catalogo/normalizar.go` + `internal/catalogo/normalizar_test.go` -- Implementar `Normalizar(s string) string` e `NormalizarBusca(nome, descricao string) string` tratando minúsculas e acentos portugueses com testes unitários.
- [x] `db/migracoes/20260911140000_catalogo_busca_e_indices.sql` -- Criar migração goose com `ALTER TABLE catalogo.produto ADD COLUMN busca_normalizada text NOT NULL DEFAULT ''`, índice GIN `pg_trgm` sobre `busca_normalizada gin_trgm_ops` e índices B-tree sobre `vendedor_id` e `categoria_id`.
- [x] `media/gerar.go` -- Atualizar gerador para incluir `busca_normalizada` nos INSERTs de produtos e regenerar `db/semente/001_catalogo_semeado.sql`.
- [x] `internal/plataforma/config.go` + `.env` -- Adicionar `AZAMON_SEMENTE_GRANDE` como parâmetro booleano (padrão `false`), mantendo paridade em `env_test.go`.
- [x] `db/semente-grande/001_catalogo_grande.sql` -- Conjunto de medição determinístico e idempotente pelo marcador próprio (o conteúdo é determinístico; o `id` fica com o `DEFAULT uuidv7()`, que é o certo para dado de bancada).
- [x] `cmd/azamon/main.go` -- Integrar chamada de semeadura dos 5.000 produtos condicionada a `cfg.SementeGrande`.
- [x] `internal/catalogo/db/consultas.sql` + `sqlc generate` -- `busca_normalizada` na consulta e `internal/catalogo/db/gerado/` regenerado pelo sqlc v1.31.1, que analisou a migração nova sem erro.
- [x] `db/schema_test.go` -- Atualizar testes de schema para verificar os novos índices e coluna normalizada.
- [x] `db/medicao_nfr4_test.go` -- Criar teste automatizado sobre PostgreSQL real (testcontainers) com 5.000 produtos executando `EXPLAIN ANALYZE` sobre consulta com termo, filtro de categoria e faixa de preço, confirmando p95 ≤ 500 ms.
- [x] `_bmad-output/implementation-artifacts/deferred-work.md` -- Registrar a inclusão dos índices de `vendedor_id` e `categoria_id`.
- [x] `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` §10 + `.memlog.md` -- Registrar o desfecho da verificação 4 (latência p95 obtida e decisão "índice antes de serviço").

**Acceptance Criteria:**
- Dado o banco migrado, quando consulto os índices de `catalogo.produto`, então existem o índice GIN `pg_trgm` sobre `busca_normalizada` e os índices B-tree sobre `vendedor_id` e `categoria_id`.
- Dado um produto inserido na semente, quando inspeciono `busca_normalizada`, então o conteúdo reflete o nome e a descrição em minúsculas e sem acento.
- Dado o banco semeado com 5.000 Produtos (`AZAMON_SEMENTE_GRANDE=true`), quando executo `EXPLAIN ANALYZE` sobre a consulta com termo de texto, categoria e faixa de preço em múltiplas repetições, então o tempo no percentil 95 é ≤ 500 ms (NFR-4, SM-4).
- Dado o arranque padrão de demonstração (`AZAMON_SEMENTE_GRANDE=false`), quando o sistema sobe, então o catálogo permanece com exatamente 50 Produtos plausíveis (SM-C3).
- Dado `sqlc generate` v1.31.1 executado sobre as migrações, quando ele processa a nova migração, então conclui com sucesso e atualiza `internal/catalogo/db/gerado/`.

## Design Notes

**Normalização na escrita vs unaccent:**
No PostgreSQL, a função `unaccent()` não possui marcação `IMMUTABLE` (depende do dicionário de busca textual carregado no schema), o que impede seu uso direto em colunas geradas (`GENERATED ALWAYS AS ... STORED`) ou índices de expressão sem wrappers complexos. A normalização na escrita executada pelo Go resolve isso de forma limpa, portável e determinística, gravando o texto já desprovido de diacríticos e em caixa baixa na coluna `busca_normalizada`.

**Trigramas e operadores GIN:**
O índice `gin (busca_normalizada gin_trgm_ops)` suporta buscas textuais por subsequência com os operadores `LIKE '%termo%'` e `ILIKE`. Termos com 3 ou mais caracteres utilizam extração de trigramas via Bitmap Index Scan; termos com 1 ou 2 caracteres degradam para varredura do índice/tabela, mas com 5.000 linhas o volume de dados totaliza menos de 2 MB em RAM, respondendo em ordens de grandeza inferiores ao teto de 500 ms (típico < 5 ms).

## Verification

**Commands:**
- `docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v "${PWD}:/src" -w /src azamon-dev go test -v ./db -run TestSchemaESemente` -- expected: testes de schema verdes confirmando novos índices e colunas.
- `docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v "${PWD}:/src" -w /src azamon-dev go test -v ./db -run TestMedicaoNFR4` -- expected: teste do NFR-4 verde comprovando p95 ≤ 500 ms com 5.000 produtos.
- `docker run --rm -v "${PWD}:/src" -w /src azamon-dev go test ./internal/catalogo/...` -- expected: testes de normalização verdes.
- `docker run --rm -v "${PWD}:/src" -w /src azamon-dev go test ./internal/plataforma` -- expected: testes de ambiente e plataforma verdes com `AZAMON_SEMENTE_GRANDE`.
- `docker run --rm -v "${PWD}:/src" -w /src azamon-dev go test ./internal/fronteira_test.go` -- expected: grafo de importações e fronteiras preservado.
- `docker run --rm -v "${PWD}:/src" -w /src sqlc/sqlc:1.31.1 generate` -- expected: código em `internal/catalogo/db/gerado/` atualizado sem erros.

**Manual checks (if no CLI):**
- Conferir no `psql` via `\d catalogo.produto` que a coluna `busca_normalizada` e os três índices (`produto_busca_normalizada_idx`, `produto_vendedor_id_idx`, `produto_categoria_id_idx`) estão presentes e ativos.

## Desfecho

Rodado em 2026-09-11 sobre PostgreSQL 18.6 por `testcontainers-go`, com 5.050 Produtos no banco.

| Sonda | Mediana | p95 | Máximo | Plano |
|---|---|---|---|---|
| termo + Categoria + faixa de preço | 0,24 ms | 0,34 ms | 0,56 ms | `Bitmap Index Scan em produto_categoria_id_idx → Bitmap Heap Scan` |
| só termo | 0,93 ms | 1,33 ms | 1,89 ms | `Seq Scan sobre produto` |

O teto do NFR-4 é 500 ms. A verificação 4 do passo 0 fecha positiva e a decisão **"índice antes de serviço"** está confirmada — nem Elasticsearch, nem cache.

O achado que vale mais que o número: **nesta escala o planejador não escolhe o índice GIN `pg_trgm` em nenhuma das duas sondas.** 5.000 linhas cabem em memória e varrer é barato. O GIN fica como seguro de crescimento, não como a causa do p95 verde — concluir o contrário a partir desta mesma medição é o erro fácil. Está registrado no `addendum.md` §10 e no `.memlog.md` do PRD.
