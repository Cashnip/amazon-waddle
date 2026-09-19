---
title: '3.10 — Chrome da loja: barra superior, busca global e Faixa de Categorias'
type: 'feature'
created: '2026-09-18'
status: 'done'
baseline_commit: 'c26d23a4be4980fa39875dc131e7b8b4847ff19f'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-3-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A barra superior da loja tem só o wordmark e o Menu da conta. A busca existe apenas no painel de filtros da Vitrine, e não há Faixa de Categorias. Assim, a UJ-1 não começa de qualquer tela.

**Approach:** A `Casca` ganha a busca global (seletor de Categoria acoplado, campo e botão de lupa, tudo num pill) e, abaixo da barra, a Faixa de Categorias em `chrome-muted`. O campo de termo sai do painel de filtros. Os botões de ação passam a usar `rounded-full`, para que o pill tenha exatamente os quatro usos do UX-DR5.

## Boundaries & Constraints

**Always:**
- A busca global e a Faixa ficam em toda tela que usa a `Casca`: a Vitrine/Resultados, a Página de Produto e os estados dela, Login, Cadastro, Recuperação de senha, Perfil, Meus pedidos, Meus endereços e Acompanhamento.
- A busca é um `<form role="search">`, e o `Enter` a submete. O destino é `/?termo=<t>&categoria=<id>`, sem os campos vazios. Termo vazio é aceito e devolve a listagem. `maxLength={100}`.
- A busca é uma consulta nova: descarta a faixa de preço, a ordenação e a página que estavam na URL.
- O campo e o seletor mostram o termo e a Categoria da URL atual, e são recriados a cada navegação.
- A Faixa é uma lista plana de links `/?categoria=<id>`, no `<nav aria-label="Categorias">`. Abaixo de 640 px, ela rola dentro de si (`overflow-x-auto`), sem rolagem na página.
- Abaixo de 640 px, a busca ocupa uma linha inteira abaixo do wordmark. O foco visível sobre o chrome segue o molde do `MenuDaConta` (`focus-visible:ring-chrome-foreground`).
- O painel de filtros perde o campo de termo, mas mantém o termo da URL ao aplicar. O chip do termo continua.
- `rounded-full`: busca global, "Adicionar ao Carrinho", "Confirmar compra" e os chips. Nenhum outro elemento o ganha.

**Ask First:** mudar qualquer rota do Go. Pôr o Carrinho ou o contador na barra.

**Never:** Carrinho na barra (Épica 4). Selo de Status do Pedido (Épica 6). Sugestão enquanto digita. Destacar a Categoria atual na Faixa. Mudar a área administrativa, que já tem `chrome-muted`, sem busca e sem Carrinho, desde a 3.1.

## I/O & Edge-Case Matrix

| Cenário | Entrada | Esperado | Erro |
|---|---|---|---|
| Busca de outra tela | em `/produtos/<id>`, "cafe" + Enter | `/?termo=cafe` | — |
| Com Categoria | "cafe" + Eletrônicos | `/?termo=cafe&categoria=<id>` | — |
| Termo vazio, "Todas" | Enter | `/` | — |
| Nova busca sobre filtros | em `/?preco_min=100&ordenacao=preco_asc`, "x" | `/?termo=x` | — |
| Clique na Faixa | Livros | `/?categoria=<id>` | — |
| Categorias falham | `GET /api/v1/categorias` ≠ 2xx | a busca funciona, com "Todas" apenas, e a Faixa fica vazia | sem erro na tela |

</frozen-after-approval>

## Code Map

- `web/components/casca.tsx` -- vira `"use client"`. Carrega `GET /api/v1/categorias` uma vez (molde de `menu-da-conta.tsx:33`) e monta o `<header>` (wordmark · busca · `MenuDaConta`) e o `<nav>` da Faixa. Três telas cliente (`entrar`, `cadastrar`, `esqueci-a-senha`) e as telas servidor a importam, então ela não pode ser `async`.
- `web/components/busca-global.tsx` (novo, client) -- o formulário. Usa `useSearchParams` e por isso precisa de `<Suspense>` na `Casca`, senão o `next build` quebra nas páginas estáticas. O fallback é o mesmo formulário vazio. Um `key` com a querystring recria os campos. Seletor nativo `<select>`, como no mockup. `SearchIcon` do `lucide-react`, com `aria-label="Buscar"`.
- `web/app/filtros.tsx:112` -- tirar o bloco `termo` do `Formulario`. `aplicar` passa `termo: estado.termo`.
- `web/app/produtos/[id]/caixa-de-compra.tsx:79` e `web/app/produtos/[id]/comprar.tsx:59` -- `rounded-full` nos dois botões de ação.
- `web/app/page.tsx:124` -- os chips já são `rounded-full`. O restante continua como está.
- `web/components/ui/badge.tsx` e `web/components/ui/radio-group.tsx` usam pill, mas não são importados por nenhuma tela.
- `mockups/pagina-de-produto.html:32-45` -- as classes `.search` (pill branco, 44 px, `select` em `muted`, botão em `primary`) e `.cats` (13 px, gap de 22 px) são a referência.

## Tasks & Acceptance

**Execution:**
- [x] `web/lib/busca.ts` + `web/scripts/busca.test.mjs` -- `destinoDaBusca(termo, categoria)` e `destinoDaCategoria(id)`, puras, testadas no `npm test` pelas linhas de URL da matriz (molde: `lib/quantidade.ts`).
- [x] `web/components/busca-global.tsx` -- o formulário, conforme a matriz.
- [x] `web/components/casca.tsx` -- a barra com a busca em `Suspense`, a Faixa e a carga das Categorias.
- [x] `web/app/filtros.tsx` -- sem o campo de termo, preservando-o.
- [x] `web/app/produtos/[id]/{caixa-de-compra,comprar}.tsx` -- `rounded-full` nos botões de ação.

**Acceptance Criteria:**
- Dado qualquer tela pública ou de Comprador, quando ela é renderizada, então a barra superior tem o wordmark, a busca global e o Menu da conta, e a Faixa de Categorias aparece logo abaixo.
- Dado 360 a 1440 px, quando a tela é aberta, então não há rolagem horizontal. A 360 px, a busca ocupa a própria linha e a Faixa rola dentro de si.
- Dado `grep -rn rounded-full web/app web/components` (fora de `components/ui/`), quando executado, então só aparecem a busca global, os dois botões de ação e os chips.

## Design Notes

**A `Casca` carrega as Categorias no navegador.** Ela serve às telas cliente e às telas servidor. Um `fetch` no servidor exigiria passar as Categorias por props nas treze chamadas. A Vitrine continua com a própria carga, para os filtros.

**Nova busca descarta os filtros de preço e a ordenação.** É o comportamento esperado de um campo global: o que se digita é uma consulta nova. Refinar continua sendo papel do painel.

## Verification

**Commands:**
- `cd web && npm run build && npm test` -- verde.

**Manual checks:**
- `docker compose up --build`. De `/produtos/<id>`, buscar "cafe" com Enter. Clicar numa Categoria da Faixa. Conferir a 360 e a 1440 px.

## Review Triage Log

Uma lente só (Edge Case Hunter), pela política do HANDOFF. Foram 7 achados:

- **patch** — `busca-global.tsx`: um termo da URL com mais de 100 caracteres aparecia inteiro no campo. Agora é cortado como no `estadoDe` da Vitrine.
- **reject** — o `key` com a querystring inteira apaga um termo ainda não enviado quando outro parâmetro muda: a spec congelada manda recriar os campos a cada navegação.
- **reject** — as Categorias são recarregadas a cada montagem da `Casca`: é decisão da spec, e a Faixa tem altura fixa, sem salto de layout.
- **reject** — payload malformado de `GET /api/v1/categorias`: a API é nossa.
- **reject** — texto digitado no fallback do `Suspense` durante a hidratação.
- **reject** — aplicar filtros descarta o texto ainda não enviado na barra.
- **reject** — a busca nova descarta a faixa de preço e a ordenação: é regra congelada da spec.

**defer** — `/redefinir-senha/[token]` não usa a `Casca` (anterior à 3.10).

Não verificado:
- o passeio no navegador a 360 e 1440 px, o Enter a partir de `/produtos/<id>` e o clique na Faixa. As URLs estão cobertas pelo `scripts/busca.test.mjs`, e o HTML servido foi conferido por curl;
- a linha "Categorias falham" da matriz não tem teste automatizado, porque o `web/` não tem bancada de componente;
- abaixo de 640 px, a ordem de tabulação (wordmark, busca, Menu) difere da ordem visual (a busca aparece abaixo do Menu).

## Suggested Review Order

**A barra da loja**

- Entrada: a `Casca` vira client e carrega as Categorias uma vez.
  [`casca.tsx:19`](../../web/components/casca.tsx#L19)

- A busca em `Suspense`, exigido pelo `useSearchParams` nas páginas estáticas.
  [`casca.tsx:45`](../../web/components/casca.tsx#L45)

- Abaixo de 640 px, a busca desce para uma linha só dela.
  [`casca.tsx:44`](../../web/components/casca.tsx#L44)

- A Faixa: links planos, rolagem dentro de si, altura fixa.
  [`casca.tsx:52`](../../web/components/casca.tsx#L52)

**A busca global**

- Termo e Categoria da URL, cortados no teto (patch da revisão).
  [`busca-global.tsx:16`](../../web/components/busca-global.tsx#L16)

- Categoria fora da lista carregada vira "Todas".
  [`busca-global.tsx:30`](../../web/components/busca-global.tsx#L30)

- Consulta nova: só termo e Categoria vão para a URL.
  [`busca.ts:3`](../../web/lib/busca.ts#L3)

**O painel e os pills**

- O painel perde o campo, mas preserva o termo da URL.
  [`filtros.tsx:92`](../../web/app/filtros.tsx#L92)

- Os dois botões de ação em `rounded-full` (UX-DR5).
  [`caixa-de-compra.tsx:79`](../../web/app/produtos/[id]/caixa-de-compra.tsx#L79)
  [`comprar.tsx:63`](../../web/app/produtos/[id]/comprar.tsx#L63)

**Periféricos**

- As linhas de URL da matriz.
  [`busca.test.mjs:9`](../../web/scripts/busca.test.mjs#L9)
