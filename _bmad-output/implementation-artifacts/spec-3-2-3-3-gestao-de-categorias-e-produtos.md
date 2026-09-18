---
title: '3.2 + 3.3 — Gestão de Categorias e de Produtos'
type: 'feature'
created: '2026-09-18'
status: 'done'
baseline_commit: '88ad406cd3ae530feb9fd10f19de61a4268022f8'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-3-context.md'
  - '{project-root}/_bmad-output/implementation-artifacts/spec-3-1-gestao-de-vendedores.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O Administrador só gerencia Vendedores. Categorias e Produtos existem apenas como semente, e o Roteiro C (abastecer a loja) não tem por onde acontecer.

**Approach:** Repetir o molde da 3.1 para Categorias (criar, renomear e remover) e para Produtos (criar, editar, desativar e reativar), com duas telas administrativas novas. O "Produto ativo" entra na mesma VIEW `catalogo.produto_visivel`. O AD-16 é emendado: a listagem administrativa de Produto é do `catalogo`, e a proibição dele vale para a listagem da loja.

## Boundaries & Constraints

**Always:**
- Toda rota nova entra no mux `admin`.
- Os tetos vêm da Config, com os valores padrão:
  - `AZAMON_CATEGORIA_NOME_MAX` 120;
  - `AZAMON_PRODUTO_NOME_MAX` 200 e `AZAMON_PRODUTO_DESCRICAO_MAX` 4000, que já existem;
  - `AZAMON_PRODUTO_PRECO_MAX_CENTAVOS` 100000000;
  - `AZAMON_PRODUTO_ESTOQUE_MAX` 100000.
- Texto é aparado e contado em runas. A mensagem nomeia o limite com separador de milhar, por exemplo "O nome do Produto tem no máximo 200 caracteres" e "A descrição do Produto tem no máximo 4.000 caracteres".
- O banco decide o duplicado (23505) e a referência (23503).
- Toda escrita de Produto regrava `busca_normalizada` com `catalogo.NormalizarBusca`.
- A imagem é `""` ou `/api/v1/media/<arquivo>` de um arquivo que exista em `media.Arquivos`.
- A listagem administrativa usa o envelope do AD-18, `{itens, pagina, por_pagina, total}`, com `por_pagina` = `AZAMON_PAGINA_TAMANHO` e ordem `nome, id`.

**Ask First:** tocar em `Reservar`, `Consolidar` ou no Estoque de Produto já criado. Expor `categoria_pai_id`.

**Never:**
- Remover Produto: a FR-9 diz desativar.
- Ajustar o Estoque depois da criação, porque a guarda das Reservas é da 3.4.
- `Visiveis` e `Disponivel`, que são da 3.4.
- Upload de imagem, que é fase 2.
- `WHERE ativo` fora da VIEW.
- Ponto flutuante no caminho do preço: o navegador converte "12,90" em 1290 por texto.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Esperado | Erro |
|---|---|---|---|
| Categoria: criar/renomear | `POST` / `PUT /admin/categorias/{id}` com `{nome}` | 201/200 `{id,nome}`, sem `categoria_pai_id` | vazio ou longo → 400 `campo:nome`; duplicado → 409 `CATEGORIA_JA_CADASTRADA` `campo:nome`; inexistente → 404 |
| Categoria: remover | `DELETE /{id}` | sem Produto → 204 | com N Produtos → 409 `CATEGORIA_COM_PRODUTOS`, a mensagem nomeia N e `dados.produtos = N` |
| Produto: listar | `GET /admin/produtos?pagina=2` | envelope; itens com Vendedor e Categoria `{id,nome}`, `descricao`, `estoque_total` e `ativo`, ativos e inativos | `pagina` não inteira ou < 1 → 400 `campo:pagina`; além do total → `itens: []` |
| Produto: criar | `POST` com todos os campos e `estoque_total` | 201; a Página de Produto já responde 200 | preço ≤ 0 ou acima do teto, estoque < 0 ou acima do teto, nome ou descrição fora do limite, imagem fora da lista → 400 com o campo; Vendedor ou Categoria inexistente ou malformado → 400 `campo:vendedor_id` ou `campo:categoria_id` |
| Produto: editar | `PUT /{id}` com tudo menos `estoque_total`, mais `ativo` (obrigatório) | 200 | inexistente → 404; mesmas validações |
| Preço congelado | editar o preço de Produto com Pedido | o Pedido lido depois é idêntico | — |
| Produto desativado | `ativo:false` | Página de Produto 404; compra 404 sem Reserva; reativar → 200 | — |
| Mídias | `GET /admin/midias` | 200, `["/api/v1/media/…svg", …]` em ordem | — |
| Sem Sessão de Administrador | qualquer rota nova | — | 404 (guarda) |

</frozen-after-approval>

## Code Map

Molde é a 3.1, lida inteira pelo `context:`. O que muda ou é novo:

- `db/migracoes/20260918120000_catalogo_produto_visivel.sql` -- a VIEW atual. A migração nova usa `CREATE OR REPLACE VIEW` com as **mesmas colunas**, e só o `WHERE` ganha `AND p.ativo`.
- `internal/catalogo/normalizar.go:30` -- `NormalizarBusca(nome, descricao)`, a mesma que a semente usa.
- `internal/catalogo/vendedor.go` -- molde de `traduzirVendedor`, `uuidDe` e `vendedorDe`. `uuidDe` já existe no pacote: não duplicar.
- `api/vendedor.go` -- molde dos handlers. No `PUT`, `ativo` é `*bool` obrigatório.
- `api/media.go` + `media/` -- `media.Arquivos` (`embed.FS`, `*.svg`) é a fonte da lista de mídias e da validação da imagem.
- `internal/plataforma/erro/erro.go` -- `EscreverLimiteDeEnderecos` é o molde do 409 com número vindo de fora do sentinela. O registro ganha `ErrCategoriaJaCadastrada`.
- `internal/plataforma/config.go` + `.env` -- o teste de Config compara as duas coisas: variável nova vai nas duas.
- `api/vendedor_test.go`, `api/sessao_test.go:145` -- molde do subteste e da `Config` de teste.
- `web/components/casca-admin.tsx:15` -- `DESTINOS`. O comentário já prevê o `Sheet` (`components/ui/sheet.tsx`) quando houver mais de um destino.
- `web/app/admin/vendedores/vendedores.tsx` -- molde de tela: formulário, erro em linha, `Dialog` e `Alert` de recusa. `components/ui/pagination.tsx`, `select.tsx` e `table.tsx` já estão instalados.
- `web/app/produtos/[id]/page.tsx:61` -- `<Image unoptimized>` da Página de Produto, que passa a usar o componente de imagem com o bloco neutro.
- Espinha: `_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md`, AD-16. O memlog se escreve com `uv run _bmad/scripts/memlog.py append --path <dir>/.memlog.md --type decision --text "…"`.
- sqlc: `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/20260918130000_catalogo_produto_ativo.sql` -- `produto.ativo boolean NOT NULL DEFAULT true`, e a VIEW recriada com `AND p.ativo`. O Down restaura a VIEW da 3.1 e depois derruba a coluna.
- [x] `internal/catalogo/db/consultas.sql` + gerado -- consultas de Categoria (listar, criar, atualizar, remover e contar Produtos da Categoria). De Produto: `ListarProdutosAdmin` (com JOIN em Vendedor e Categoria, `LIMIT/OFFSET`) e `ContarProdutos`, mais `CriarProduto` e `AtualizarProduto :one`.
- [x] `internal/catalogo/categoria.go` + `internal/catalogo/produto.go` -- os tipos e as funções. O 23503 de Produto vira erro de campo pelo nome da constraint (`produto_vendedor_id_fkey` e `produto_categoria_id_fkey`), e o 23503 de Categoria vira `CategoriaComProdutos{N}`, que desembrulha em `ErrCategoriaComProdutos`.
- [x] `internal/plataforma/config.go` + `.env` + `erro/erro.go` -- os três tetos novos, `ErrCategoriaJaCadastrada` (409) e `EscreverCategoriaComProdutos(ctx, w, n)`.
- [x] `api/categoria.go`, `api/produto_admin.go`, `api/media.go`, `api/rotas.go` -- as rotas da matriz. A lista de mídias sai de `fs.Glob(media.Arquivos, "*.svg")`.
- [x] `api/categoria_test.go`, `api/produto_admin_test.go`, `api/sessao_test.go` -- dois subtestes que percorrem a matriz, com os tetos na `Config` de teste. Nunca alterar Produto, Vendedor ou Categoria semeados: criar os próprios.
- [x] `ARCHITECTURE-SPINE.md` (AD-16) + `.memlog.md` -- emendar no lugar, sem renumerar:
  - a Rule diz que a proibição de listagem e de consulta de Produto vale para a **loja**;
  - a tabela ganha a linha `GET /api/v1/admin/produtos` (listagem administrativa, inclui inativos) | `catalogo`;
  - uma entrada `decision` no memlog, com o motivo: a VIEW de `busca` só enxerga Produto visível.
- [x] `web/components/imagem-do-produto.tsx` -- `<img>` na proporção 4/3. `src` vazio ou `onError` vira um bloco neutro com o nome do Produto centralizado, nunca o ícone de imagem quebrada. Usado em `web/app/produtos/[id]/page.tsx` e na lista administrativa.
- [x] `web/components/casca-admin.tsx` -- destinos Vendedores, Categorias e Produtos. Abaixo de 640 px, eles viram um `Sheet` aberto por um botão com nome acessível.
- [x] `web/app/admin/categorias/{page,categorias}.tsx` -- no molde de Vendedores, sem "situação". A recusa de remover mostra a mensagem do servidor com a contagem. Vazio: "Nenhuma Categoria cadastrada."
- [x] `web/app/admin/produtos/{page,produtos}.tsx` -- uma `Table` com miniatura, nome, preço em `R$`, Estoque total, Vendedor, Categoria, situação e as ações Editar e Desativar/Reativar, mais `Pagination` com página e total. O formulário tem:
  - nome e descrição, esta num `textarea`;
  - preço em reais;
  - Estoque total, só na criação;
  - `Select` de Vendedor, de Categoria e de imagem, esta com a opção "Sem imagem".

  Erro em linha com foco. Vazio: "Nenhum Produto cadastrado."
- [x] `README.md` -- Categorias e Produtos na área administrativa.

**Acceptance Criteria:**
- Dado o Administrador em `/admin/produtos`, quando ele cria um Produto com imagem, Vendedor e Categoria, então o Produto está na lista e abre em `/produtos/{id}` sem reiniciar nada.
- Dado um Produto semeado cuja `imagem_url` aponta para um arquivo inexistente, quando a Página de Produto abre, então aparece o bloco neutro com o nome, e não o ícone de imagem quebrada.
- Dado uma tela de 360 px, quando a área administrativa abre, então os três destinos estão num `Sheet` e não há rolagem horizontal da página. A tabela rola dentro do próprio contêiner.

## Design Notes

A spec da 3.1 dizia que o "Produto ativo" era da 3.4. Ele veio para cá porque sem ele "desativar Produto" (FR-9) não teria efeito. A 3.4 continua com `Visiveis`, `Disponivel` e a guarda do ajuste de Estoque.

Na Categoria, a contagem só é feita **depois** do 23503: o caminho comum (remover uma Categoria vazia) continua sendo uma ida só ao banco.

## Verification

**Commands:**
- `go build ./... && go vet ./... && go test -count=1 ./...` -- verde, com os dois subtestes novos.
- `cd web && npm run build` -- verde, com a checagem offline e o `node --test`.

**Manual checks:**
- `docker compose up` → `/admin/entrar` → criar uma Categoria e um Produto nela → abrir `/produtos/{id}` → desativar → 404 → reativar → tentar remover a Categoria (recusa com "1 Produto") → abrir a área administrativa a 360 px.

## Review Triage Log

Uma lente só (Edge Case Hunter), pela política de revisão do projeto para CRUD. Foram 7 achados:

- **patch** — `config.go`: se `AZAMON_PRODUTO_ESTOQUE_MAX` passasse do teto de `int32`, a conversão para `int32` daria a volta e gravaria um Estoque negativo. Agora a Config recusa esse valor no arranque.
- **patch** — `produtos.tsx`: um Estoque com dígitos demais passava na regex e caía num 400 genérico. Agora exige `Number.isSafeInteger`, e o erro sai no campo.
- **reject** (5):
  - o 23503 vindo da FK de `categoria_pai_id`, que é sempre nulo e não exposto;
  - a corrida entre o `DELETE` e a contagem de Produtos, e as duas de edição simultânea, porque só um Administrador está semeado;
  - o duplicado só pela caixa das letras, porque o `UNIQUE` decide, como na 3.1.

Fora da spec, feito pela implementação:
- `decodificarCorpoAte`: o teto de 4 KiB do corpo não comportava uma descrição de 4.000 caracteres, então as rotas de Produto usam `6 × (nome + descrição) + 4 KiB`;
- `BuscarProdutoAdmin`, que relê a linha com os nomes do Vendedor e da Categoria depois de gravar;
- `erro.Milhar`, com teste próprio;
- `web/lib/preco.ts` e `web/lib/pedir.ts`, este extraído de `vendedores.tsx`;
- o Estoque é obrigatório na criação, e a descrição pode ficar vazia.

Não verificado:
- a troca da imagem quebrada pelo bloco neutro no navegador;
- o `Sheet` a 360 px.

O Postgres do `docker compose` guarda "Categoria Manual" e "Produto Manual", este com imagem quebrada de propósito. `docker compose down -v` apaga os dois.

## Suggested Review Order

**Visibilidade e a espinha**

- "Produto ativo" entra na mesma VIEW, com as mesmas colunas, via `CREATE OR REPLACE`.
  [`20260918130000_catalogo_produto_ativo.sql:8`](../../db/migracoes/20260918130000_catalogo_produto_ativo.sql#L8)

- A emenda no AD-16: a proibição de listar vale para a loja, e a listagem administrativa é do `catalogo`.
  [`ARCHITECTURE-SPINE.md:227`](../planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md#L227)

**Produtos**

- Validação do corpo: tetos da Config, imagem da lista embutida, Estoque só na criação.
  [`produto_admin.go:128`](../../api/produto_admin.go#L128)

- O 23503 vira erro de campo pelo nome da constraint.
  [`produto.go:155`](../../internal/catalogo/produto.go#L155)

- A listagem paginada com o envelope do AD-18.
  [`produto.go:56`](../../internal/catalogo/produto.go#L56)

- O teto de corpo por rota, para caber uma descrição de 4.000 caracteres.
  [`rotas.go:137`](../../api/rotas.go#L137)

**Categorias**

- Conta os Produtos só depois do 23503; o caminho comum é uma ida ao banco.
  [`categoria.go:72`](../../internal/catalogo/categoria.go#L72)

- A mensagem da recusa com a contagem e o separador de milhar.
  [`erro.go:140`](../../internal/plataforma/erro/erro.go#L140)

**web/**

- Imagem quebrada vira bloco neutro, inclusive quando falha antes da hidratação.
  [`imagem-do-produto.tsx:31`](../../web/components/imagem-do-produto.tsx#L31)

- Três destinos, e o `Sheet` abaixo de 640 px.
  [`casca-admin.tsx:8`](../../web/components/casca-admin.tsx#L8)

- O preço vira centavos por texto, nunca por ponto flutuante.
  [`preco.ts:6`](../../web/lib/preco.ts#L6)

**Periféricos**

- O preço congelado no Pedido depois da edição.
  [`produto_admin_test.go:147`](../../api/produto_admin_test.go#L147)

- O patch da revisão: o teto do Estoque cabe em `integer`.
  [`config.go:151`](../../internal/plataforma/config.go#L151)
