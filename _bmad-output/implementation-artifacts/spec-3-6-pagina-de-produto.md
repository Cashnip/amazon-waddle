---
title: '3.6 — Página de Produto'
type: 'feature'
created: '2026-09-18'
status: 'done'
baseline_commit: '1b9cf34370a388ec0babb3af418b7e17b84b4928'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-3-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A Página de Produto ainda é a versão crua da 1.5. Ela não mostra a Categoria nem a disponibilidade, não tem Caixa de compra, e o Produto indisponível cai no 404 genérico do Next. O visitante sem Sessão também não tem como começar a compra.

**Approach:** O `GET /api/v1/produtos/<id>` passa a devolver a Categoria e o Estoque disponível. A página ganha o breadcrumb, a Caixa de compra à direita a partir de 1024 px (selo, quantidade e "Adicionar ao Carrinho"), a tela "Este Produto não está disponível." e o `Skeleton` de carregamento. Sem Sessão, "Adicionar ao Carrinho" leva ao Login guardando o Produto e a quantidade.

## Boundaries & Constraints

**Always:**
- A consulta de detalhe continua sobre a VIEW `catalogo.produto_visivel` (AD-19). A Categoria entra por JOIN com `catalogo.categoria`, que é do mesmo schema. Nenhum `WHERE ativo` novo, e a VIEW não muda.
- A rota continua sendo do `catalogo` (AD-16). O número exibido é sempre `estoque_disponivel`, nunca o total, e a Reserva nunca aparece na tela.
- Selo: "Em estoque" em `text-available`, "Indisponível" neutro. Na Caixa, com disponível entre 1 e 9, mostra "Restam N unidades" (N = disponível). Verde só aí.
- A quantidade vai de 1 a `min(10, disponível)`. O teto de 10 por Item de Carrinho é do §7.1, e na 3.6 é uma constante do `web/`.
- Sem Sessão: `paraLogin("/produtos/<id>?quantidade=N")`. A página lê `?quantidade` e a limita ao intervalo válido. Um valor inválido vira 1.
- Com Sessão: "Adicionar ao Carrinho" fica desabilitado, com a nota "O Carrinho ainda não está disponível.".
- "Confirmar compra" (esqueleto, 1 unidade) continua na Caixa, só quando o disponível é maior que 0.
- Ordem de tabulação: título e preço antes da Caixa de compra.

**Ask First:** mudar o `POST /api/v1/pedidos` ou a VIEW. Criar qualquer superfície de Carrinho.

**Never:** Carrinho, contador na barra, botão no Cartão de Produto (Épica 4). Barra superior e busca global (3.10). Distinguir inexistente, desativado e Vendedor desativado.

## I/O & Edge-Case Matrix

| Cenário | Entrada | Esperado | Erro |
|---|---|---|---|
| Visível | `GET /api/v1/produtos/<id>` | 200 com os campos de antes + `categoria_id`, `categoria`, `estoque_disponivel` | — |
| Com Reserva ativa | Produto com total 5 e Reserva ATIVA de 2 | `estoque_disponivel: 3` | — |
| Invisível | id inexistente, `abc`, Produto desativado, Produto de Vendedor desativado | tela "Este Produto não está disponível." + "Voltar à Vitrine" | 404 `NAO_ENCONTRADO` |
| Esgotado | `estoque_disponivel: 0` | selo "Indisponível", sem quantidade, sem os dois botões de ação | — |
| Volta do Login | `/produtos/<id>?quantidade=3`, disponível 5 | seletor em 3 | `?quantidade=50` → 5; `abc`/`0` → 1 |

</frozen-after-approval>

## Code Map

- `internal/catalogo/db/consultas.sql:4` -- `BuscarProdutoComVendedor`, que ganha `JOIN catalogo.categoria c ON c.id = pv.categoria_id` e as colunas `categoria_id`, `categoria_nome` e `estoque_disponivel`. Depois, gerar com sqlc (`go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`).
- `internal/catalogo/catalogo.go:44` -- `Produto` e `BuscarProduto` ganham `CategoriaID`, `CategoriaNome` e `EstoqueDisponivel`. `internal/pedido/pedido.go:116` também chama `BuscarProduto`, mas campo a mais não o afeta.
- `api/produto.go:17` -- `saidaProduto` e `detalheDoProduto`. O comentário da linha 26 ("chega na Épica 3") está velho.
- `api/produto_test.go:13` -- `produtoSemeadoSai`, que ganha a Categoria semeada e o `estoque_disponivel` comparado com a VIEW. A Reserva ativa pode seguir o molde de `api/estoque_test.go`. Os casos de invisível já estão cobertos em `produto_admin_test.go:178` e `vendedor_test.go:122`.
- `web/app/produtos/[id]/page.tsx` -- reescrever: 404 → `notFound()`, outro erro → `throw`; lê `searchParams.quantidade`; breadcrumb com `<nav aria-label>`, Vitrine › Categoria (`/?categoria=<id>`); grade `lg:` de três colunas (imagem | informação | Caixa), empilhada abaixo disso.
- `web/app/produtos/[id]/comprar.tsx` -- o "Confirmar compra" do esqueleto. Não muda.
- `web/app/produtos/[id]/caixa-de-compra.tsx` (novo, client) -- selo, `Select` de quantidade (`components/ui/select.tsx`) e "Adicionar ao Carrinho". Consulta a Sessão por `GET /api/v1/sessao`, no molde de `components/menu-da-conta.tsx:32`. Recebe o `Comprar` como filho.
- `web/app/produtos/[id]/not-found.tsx` (novo) e `loading.tsx` (novo, `Skeleton` com a forma de imagem, título e Caixa).
- `web/lib/destino.ts:39` -- `paraLogin(caminho)` já aceita o caminho.
- `web/components/{casca,preco,imagem-do-produto}.tsx`, `ui/{card,button,skeleton,select}.tsx` -- prontos.

## Tasks & Acceptance

**Execution:**
- [x] `internal/catalogo/db/consultas.sql` + gerado, `internal/catalogo/catalogo.go` -- Categoria e disponível no detalhe.
- [x] `api/produto.go` -- DTO com `categoria_id`, `categoria` e `estoque_disponivel`.
- [x] `api/produto_test.go` -- Categoria e disponível, este também com Reserva ativa.
- [x] `web/app/produtos/[id]/{page,caixa-de-compra,not-found,loading}.tsx` -- a página, conforme a matriz.
- [x] `README.md` -- revisar o passo do passeio que cita a Página de Produto.

**Acceptance Criteria:**
- Dado o Catálogo Semeado, quando o visitante abre um Produto em 1024 px ou mais, então vê o breadcrumb com a Categoria, nome, imagem, descrição, "Vendido por", e a Caixa de compra à direita. Abaixo de 1024 px, a Caixa fica empilhada, sem rolagem horizontal a 360 px.
- Dado um visitante sem Sessão, quando escolhe 3 e clica em "Adicionar ao Carrinho", então vai a `/entrar?destino=…quantidade%3D3`, e ao entrar volta à mesma Página de Produto com 3 no seletor.

## Design Notes

**Por que o JOIN e não mexer na VIEW.** A Categoria é dado do `catalogo`, e a consulta é do `catalogo`. Pôr o nome da Categoria na VIEW mudaria o contrato que `busca` lê por causa de uma tela só.

**`not-found.tsx` do segmento.** O `notFound()` do App Router já responde 404 e renderiza o arquivo mais próximo, então a tela não precisa de estado à mão. Erro que não é 404 não pode virar "não disponível", e cai no erro genérico (a falta de `error.tsx` está no `deferred-work.md`).

## Verification

**Commands:**
- `go build ./... && go vet ./... && go test -count=1 ./...` -- verde.
- `cd web && npm run build` -- verde.

**Manual checks:**
- `docker compose up --build` → um Produto a 360 e a 1440 px → um id inválido → o fluxo sem Sessão com quantidade 3 até o Login e a volta.

## Review Triage Log

Uma lente só (Edge Case Hunter), pela política do HANDOFF: sem concorrência nem máquina de estados. Foram 4 achados:

- **patch** — `caixa-de-compra.tsx`: um 5xx do `GET /api/v1/sessao` habilitava o botão e mandava ao Login quem já tinha entrado. Agora só o 401 conta como "sem Sessão".
- **reject** — falta de `error.tsx`: já está no `deferred-work.md` desde a 3.5.
- **reject** — `useState` preso à quantidade inicial em navegação no cliente: nenhum `Link` leva a outro Produto sem remontar a página.
- **reject** — teste da Reserva com disponível abaixo de 2: a semente é determinística, com 10.

Fora da spec, feito pela implementação:
- `web/lib/quantidade.ts` + `web/scripts/quantidade.test.mjs`: o teto e a regra do `?quantidade`, testados no `npm test`;
- "Resta 1 unidade" no singular;
- "Adicionar ao Carrinho" desabilitado até a Sessão ser lida.

Não verificado:
- **a tela "não disponível" sai com HTTP 200**, e não 404: com o `loading.tsx`, o streaming começa antes do `notFound()`. A API continua respondendo 404 `NAO_ENCONTRADO`, e a tela está correta. Um 404 real na página exigiria tirar o `loading.tsx`;
- as linhas "Invisível" (tela) e "Esgotado" da matriz não têm teste automatizado, porque o `web/` não tem bancada de componente. A tela do invisível foi conferida por curl;
- o passeio no navegador a 360 e 1440 px e a ida e volta pelo Login com quantidade 3.

## Suggested Review Order

**O detalhe no `catalogo`**

- Entrada: Categoria por JOIN sobre a VIEW, que não muda; o número é o disponível.
  [`consultas.sql:10`](../../internal/catalogo/db/consultas.sql#L10)

- O `Produto` público ganha Categoria e disponível.
  [`catalogo.go:52`](../../internal/catalogo/catalogo.go#L52)

- O DTO da Página de Produto, sem `estoque_total`.
  [`produto.go:24`](../../api/produto.go#L24)

**A página**

- Só o 404 vira "não disponível"; outro erro é lançado.
  [`page.tsx:50`](../../web/app/produtos/[id]/page.tsx#L50)

- Breadcrumb Vitrine › Categoria, que filtra a Vitrine.
  [`page.tsx:57`](../../web/app/produtos/[id]/page.tsx#L57)

- Três colunas a partir de 1024 px; a ordem do DOM é a da tabulação.
  [`page.tsx:73`](../../web/app/produtos/[id]/page.tsx#L73)

- A tela única dos três casos invisíveis.
  [`not-found.tsx:8`](../../web/app/produtos/[id]/not-found.tsx#L8)

**A Caixa de compra**

- Só o 401 é "sem Sessão" (patch da revisão).
  [`caixa-de-compra.tsx:41`](../../web/app/produtos/[id]/caixa-de-compra.tsx#L41)

- Sem Sessão, Login com Produto e quantidade no destino.
  [`caixa-de-compra.tsx:48`](../../web/app/produtos/[id]/caixa-de-compra.tsx#L48)

- Selo: número abaixo de 10, verde só na disponibilidade.
  [`caixa-de-compra.tsx:61`](../../web/app/produtos/[id]/caixa-de-compra.tsx#L61)

- O `?quantidade` limitado a 1..min(10, disponível).
  [`quantidade.ts:13`](../../web/lib/quantidade.ts#L13)

**Periféricos**

- Categoria, disponível contra a VIEW, e a Reserva ativa de 2.
  [`produto_test.go:64`](../../api/produto_test.go#L64)

- Skeleton na forma da imagem, do título e da Caixa.
  [`loading.tsx:5`](../../web/app/produtos/[id]/loading.tsx#L5)

- O passeio do README ganha a Página de Produto.
  [`README.md:74`](../../README.md#L74)
