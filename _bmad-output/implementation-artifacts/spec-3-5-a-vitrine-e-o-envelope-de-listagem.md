---
title: '3.5 — A Vitrine e o envelope de listagem'
type: 'feature'
created: '2026-09-18'
status: 'done'
baseline_commit: '77a048654072321c6c31debc971768e41a8e73c1'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-3-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `GET /api/v1/produtos` não existe e o módulo `busca` está vazio. A raiz `/` ainda é a página de conferência da 1.2, cujo único link de Produto cai na faixa de centavos que o Provedor Simulado deixa parada. O visitante não tem como ver o que a loja vende.

**Approach:** `busca` ganha a sua primeira consulta: a listagem paginada sobre a VIEW `catalogo.produto_visivel`, servida em `GET /api/v1/produtos` com o envelope do AD-18. A raiz `/` passa a ser a Vitrine: a grade de cartões, o `Skeleton` no carregamento, o vazio tratado e a paginação pela URL.

## Boundaries & Constraints

**Always:**
- `busca` lê só a VIEW, nunca tabela (AD-2/AD-16). Nenhum `WHERE ativo` novo.
- `busca` não importa `catalogo` nem é importado por ele; o `fronteira_test.go` continua verde sem mudar a tabela.
- Desempate final sempre em `id` ascendente. Nesta estória a ordem é **só** `id`: "mais recentes", o padrão da FR-14, é da 3.9.
- `por_pagina` padrão e teto vêm de `AZAMON_PAGINA_TAMANHO` e `AZAMON_PAGINA_TAMANHO_MAX`. Acima do teto é **rebaixado** ao teto, e não recusado.
- O cartão mostra `estoque_disponivel` como selo ("Em estoque" em `text-available`, "Indisponível" neutro, sem fundo e sem ícone). O verde é usado só aí.
- A Reserva nunca aparece na tela.

**Ask First:** aceitar `termo`, filtros ou `ordenacao` na rota (são da 3.7, 3.8 e 3.9). Mexer na VIEW.

**Never:**
- Cache de listagem.
- Botão "Adicionar ao Carrinho" no cartão, porque o Carrinho é da Épica 4. Em Estoque zero, "não adicionável" se cumpre pelo selo "Indisponível".
- A barra superior com busca e a Faixa de Categorias (3.10).
- Rolagem infinita, consulta em intervalo, spinner.

## I/O & Edge-Case Matrix

| Cenário | Entrada | Esperado | Erro |
|---|---|---|---|
| Padrão | `GET /api/v1/produtos` | 200 `{itens, pagina:1, por_pagina:20, total}`; cada item traz `id, nome, preco_centavos, imagem_url, estoque_disponivel` | — |
| Teto | `?por_pagina=100` | `por_pagina: 60`, até 60 itens | — |
| Além do total | `?pagina=999` | 200, `itens: []`, `total` real | — |
| Inválido | `pagina` ou `por_pagina` = `0`, `-1`, `abc` | — | 400 `campo:pagina` / `campo:por_pagina` |
| Invisível | Produto inativo, ou de Vendedor inativo | fora de `itens` e de `total` | — |
| Paginação completa | percorrer todas as páginas com `por_pagina=7` | cada id visível aparece exatamente uma vez | — |
| Catálogo vazio | VIEW sem linhas | tela: "O Catálogo ainda não tem Produtos."; com Sessão de Administrador, link para `/admin/produtos` | — |

</frozen-after-approval>

## Code Map

- `internal/busca/busca.go` -- vazio; recebe `Listar(ctx, bd gerado.DBTX, pagina, porPagina int) ([]Produto, int64, error)`. O molde de paginação é `catalogo.ListarProdutosAdmin` (`internal/catalogo/produto.go:57`): conta primeiro, e página além do total devolve vazio antes de multiplicar.
- `sqlc.yaml` -- ganha o bloco `busca`: `schema: db/migracoes/*_catalogo_*.sql` (é onde a VIEW nasce), `queries: internal/busca/db/consultas.sql`, `out: internal/busca/db/gerado`, com as mesmas opções dos outros blocos.
- `db/migracoes/20260918140000_catalogo_estoque_disponivel.sql` -- a VIEW atual, só leitura. Colunas: `id, nome, descricao, preco_centavos, imagem_url, vendedor_id, categoria_id, busca_normalizada, estoque_disponivel, vendedor_nome`.
- `api/produto_admin.go:50` -- `listagem[T]` e a leitura de `pagina`. O tipo vai para `api/rotas.go`, junto de `escreverJSON`, e não é copiado.
- `api/rotas.go:48` -- o comentário "a listagem é de `busca`, na Épica 3" vira a rota `GET /api/v1/produtos`, no mux raiz.
- `internal/plataforma/config.go:105` -- `PaginaTamanho`/`PaginaTamanhoMax` já existem.
- `api/rotas_test.go:60` -- o pânico por padrão duplicado já é testado; nada novo.
- `api/sessao_test.go:475` -- onde entra o subteste novo, no fim, depois do `gestaoDeProdutos`. O molde é `api/estoque_test.go`.
- `web/app/page.tsx` -- a página de conferência; vira a Vitrine.
- `web/app/produtos/[id]/page.tsx` -- molde de Server Component com `fetch(${api}…, {cache:"no-store"})` e `Casca`. O `Preco` local (linha 28) é extraído para `web/components/preco.tsx` e reusado pelos dois.
- `web/components/imagem-do-produto.tsx`, `web/components/ui/{skeleton,pagination,card}.tsx` -- prontos.
- `README.md:61` -- o passo 1 do passeio descreve a página de conferência e o link temporário.
- sqlc: `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`.

## Tasks & Acceptance

**Execution:**
- [x] `sqlc.yaml`, `internal/busca/db/consultas.sql` + gerado -- `ListarVisiveis` (`ORDER BY id LIMIT OFFSET`) e `ContarVisiveis`, ambas sobre `catalogo.produto_visivel`.
- [x] `internal/busca/busca.go` -- `Produto` e `Listar`; atualizar o comentário do pacote.
- [x] `api/rotas.go`, `api/produto_admin.go`, `api/busca.go` -- mover `listagem[T]`. Criar um `paginacaoDe(r, cfg)` que leia `pagina` e `por_pagina` e sirva também à listagem administrativa. Criar o handler `listarProdutos` com `semCache`.
- [x] `api/busca_test.go` + `api/sessao_test.go` -- um subteste que percorre a matriz. O `total` é comparado com um `count(*)` da VIEW, e o invisível é um Produto próprio desativado.
- [x] `web/components/preco.tsx`, `web/app/produtos/[id]/page.tsx` -- extrair o `Preco`.
- [x] `web/app/page.tsx`, `web/app/loading.tsx`, `web/components/cartao-de-produto.tsx`, `web/app/catalogo-vazio.tsx` -- montar a Vitrine:
  - Server Component que lê `?pagina` e repassa;
  - grade `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 min-[1440px]:grid-cols-4`. O `globals.css` não redefine breakpoints, e o `2xl` padrão é 1536;
  - cartão inteiro como link para `/produtos/{id}`;
  - `Pagination` com links `?pagina=N` quando `total > por_pagina`;
  - `loading.tsx` com 8 `Skeleton` na forma do cartão;
  - vazio como client component, que só consulta `GET /api/v1/admin/sessao` para decidir o link.
- [x] `README.md` -- o passo 1 do passeio passa a descrever a Vitrine; a Caixa de Som Maré continua sendo o Produto indicado.

**Acceptance Criteria:**
- Dado o `docker compose up` com o Catálogo Semeado, quando o visitante abre `/`, então vê 20 cartões, com o link para a página 2 e sem rolagem horizontal de 360 a 1440 px.
- Dado um Produto criado ou desativado em `/admin/produtos`, quando a Vitrine é recarregada, então ele aparece ou some sem reiniciar nada.
- Dado um Produto com Estoque disponível 0, quando o cartão é renderizado, então mostra "Indisponível", sem verde.

## Design Notes

**Por que só `id` por ora.** A 3.9 fixa as três ordenações e o padrão "mais recentes". Inventar agora um padrão provisório seria decidir a 3.9 aqui. Com `id` a paginação já é estável, e a 3.9 só antepõe a chave de ordenação.

**Server Component + `loading.tsx`.** O `Skeleton` em grade sai do Suspense nativo do App Router, sem estado de carregamento à mão. A URL (`?pagina=`) é o estado, então recarregar e voltar reproduzem a página.

## Verification

**Commands:**
- `go build ./... && go vet ./... && go test -count=1 ./...` -- verde, com o subteste novo e o `fronteira_test.go`.
- `cd web && npm run build` -- verde.

**Manual checks:**
- `docker compose up --build` → `/` a 360, 640, 1024 e 1440 px (1, 2, 3 e 4 colunas) → página 2 → voltar no navegador → abrir um cartão.

## Review Triage Log

Uma lente só (Edge Case Hunter), pela política do HANDOFF: sem concorrência nem máquina de estados. Foram 4 achados:

- **patch** — `page.tsx`: em `?pagina` além do total, o "Anterior" apontava para outra página vazia. Agora a paginação só aparece dentro do intervalo, e o caminho de volta é o link à primeira página.
- **defer** — o `web/` não tem `error.tsx`, e a Vitrine cai no erro genérico do Next quando a API falha. É lacuna de todas as telas Server Component e está no `deferred-work.md`.
- **reject** — timeout no `fetch`: a API está no mesmo compose, e a Página de Produto segue o mesmo padrão.
- **reject** — contagem e lista em consultas separadas podem divergir por um instante. É o mesmo molde da listagem administrativa, e sem cache recarregar corrige.

Fora da spec, feito pela implementação e pela verificação:
- a listagem administrativa passa a aceitar `por_pagina`, porque divide o `paginacaoDe`;
- página além do total, com Catálogo não vazio, mostra "Esta página não tem Produtos." e o link à primeira página;
- o subteste ganhou o caso do Produto de Vendedor desativado, que faltava para cobrir a linha "Invisível" da matriz.

Não verificado:
- a linha "Catálogo vazio" da matriz: o `web/` não tem bancada de teste de componente;
- o passeio manual a 360, 640, 1024 e 1440 px.

## Suggested Review Order

**A rota e a posse do AD-16**

- Entrada: `GET /api/v1/produtos` no mux raiz, servida por `busca`, ao lado do detalhe do `catalogo`.
  [`rotas.go:50`](../../api/rotas.go#L50)

- O handler só traduz: paginação, `busca.Listar` e o envelope.
  [`busca.go:20`](../../api/busca.go#L20)

**A consulta de `busca`**

- Só a VIEW, ordenada por `id`; a 3.9 antepõe a sua chave.
  [`consultas.sql:6`](../../internal/busca/db/consultas.sql#L6)

- Conta primeiro; página além do total sai vazia sem transbordar.
  [`busca.go:26`](../../internal/busca/busca.go#L26)

- O bloco sqlc de `busca` enxerga o schema de `catalogo` só para a VIEW.
  [`sqlc.yaml:29`](../../sqlc.yaml#L29)

**O envelope e a paginação**

- Uma leitura de `pagina` e `por_pagina` para as duas listagens; o teto rebaixa.
  [`rotas.go:177`](../../api/rotas.go#L177)

- O envelope do AD-18, movido para ser de todos.
  [`rotas.go:167`](../../api/rotas.go#L167)

- A listagem administrativa passa a usar o mesmo leitor.
  [`produto_admin.go:54`](../../api/produto_admin.go#L54)

**A Vitrine**

- Server Component com `no-store`; a URL é o estado.
  [`page.tsx:21`](../../web/app/page.tsx#L21)

- Vazio, além do total e grade; patch da revisão na paginação.
  [`page.tsx:56`](../../web/app/page.tsx#L56)

- O selo: verde só para "Em estoque", sem botão de Carrinho.
  [`cartao-de-produto.tsx:32`](../../web/components/cartao-de-produto.tsx#L32)

- A quarta coluna em 1440 px, dividida com o `loading.tsx`.
  [`cartao-de-produto.tsx:8`](../../web/components/cartao-de-produto.tsx#L8)

- O caminho para cadastrar só com Sessão de Administrador.
  [`catalogo-vazio.tsx:12`](../../web/app/catalogo-vazio.tsx#L12)

**Periféricos**

- A matriz num subteste, com `total` contra o `count(*)` da VIEW.
  [`busca_test.go:24`](../../api/busca_test.go#L24)

- O Skeleton em grade, na forma do cartão.
  [`loading.tsx:11`](../../web/app/loading.tsx#L11)

- O `Preco` extraído da Página de Produto.
  [`preco.tsx:6`](../../web/components/preco.tsx#L6)

- O passeio do README começa na Vitrine.
  [`README.md:61`](../../README.md#L61)
