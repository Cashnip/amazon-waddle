---
title: '3.7–3.9 — Busca por texto, filtros de Categoria e faixa de preço, e ordenação'
type: 'feature'
created: '2026-09-18'
status: 'done'
baseline_commit: '9d4dfb23811ff706adefc1033e4c04b657d2921f'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-3-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A Vitrine só pagina por `id`. O visitante não consegue procurar por termo, restringir por Categoria ou preço, nem ordenar, que são as FR-12, FR-13 e FR-14. Os Resultados de busca não existem.

**Approach:** `GET /api/v1/produtos` passa a aceitar `termo`, `categoria`, `preco_min`, `preco_max` e `ordenacao`, todos combináveis, numa consulta só de `busca` sobre a VIEW. A raiz `/` vira Vitrine e Resultados ao mesmo tempo. O estado inteiro fica na URL, com painel de filtros, chips removíveis e seletor de ordenação. "Mais recentes" pede uma coluna `criado_em` no Produto, exposta pela VIEW.

## Boundaries & Constraints

**Always:**
- `busca` lê só a VIEW. O termo passa por `catalogo.Normalizar`, a mesma função da escrita, e casa por `LIKE` sobre `busca_normalizada`. Antes disso, `\`, `%` e `_` são escapados.
- O teto do termo vem de `cfg.BuscaTermoMax` (100) e conta runas, não bytes. Termo vazio ou só com espaços é "sem termo".
- Dinheiro em centavos na API e na URL. A tela converte reais digitados para centavos, e formata centavos em R$ com `web/lib/preco.ts`.
- A ordem é `recentes` (padrão), `preco_asc` ou `preco_desc`, e o desempate final é sempre `id` ascendente.
- Mudar filtro, termo ou ordenação volta para a página 1. Paginar preserva todo o resto.
- As Categorias da loja saem de uma rota nova, `GET /api/v1/categorias`, de `catalogo`, no mux raiz, sem Sessão, e reusam `catalogo.ListarCategorias`.

**Ask First:** um índice novo; mudar a VIEW além de acrescentar `criado_em`; mexer na `Casca`.

**Never:** cache; relevância, correção ou sugestão enquanto digita; `tsvector` ou `unaccent`; barra superior ou Faixa de Categorias (3.10); "Vendido por" e "Adicionar ao Carrinho" no cartão.

## I/O & Edge-Case Matrix

| Cenário | Entrada | Esperado | Erro |
|---|---|---|---|
| Acento | `?termo=CAFE` | encontra "Café" pelo nome ou pela descrição | — |
| Curinga literal | `?termo=50%` / `?termo=a_b` | só Produtos com esse texto literal | — |
| Termo longo | 101 runas | — | 400 `campo:termo` |
| Combinação | termo + `categoria` + `preco_min` + `preco_max` | a interseção; `total` coerente | — |
| Categoria | uuid inexistente / malformado | 200 vazio / — | — / 400 `campo:categoria` |
| Preço | negativo, não inteiro | — | 400 `campo:preco_min` ou `campo:preco_max` |
| Faixa invertida | `preco_min > preco_max` | — | 400 `campo:preco_min`, "O preço mínimo não pode ser maior que o máximo." |
| Ordenação | `?ordenacao=xyz` | — | 400 `campo:ordenacao` |
| Paginação estável | cada ordenação, `por_pagina=7`, todas as páginas | cada id visível aparece exatamente uma vez, na ordem declarada | — |
| Sem resultado | termo sem casamento | tela: "Nenhum Produto encontrado para '{termo}'.", chips e "Limpar filtros" | — |
| Faixa invertida na tela | mín. > máx. no painel | mensagem em linha no filtro; nada é navegado e a listagem anterior fica | — |

</frozen-after-approval>

## Code Map

- `db/migracoes/20260918140000_catalogo_estoque_disponivel.sql` -- molde da migração nova: `DROP VIEW` + `CREATE VIEW`, porque a lista de colunas muda. A nova acrescenta `criado_em timestamptz NOT NULL DEFAULT now()` a `catalogo.produto`, e a VIEW recriada expõe a coluna. A semente (`db/semente/001_catalogo_semeado.sql`) insere numa transação só, então todos os Produtos dela recebem o mesmo instante e o desempate por `id` decide entre eles.
- `internal/busca/db/consultas.sql` -- `ListarVisiveis`/`ContarVisiveis` ganham filtros opcionais por `sqlc.narg` (`@x::tipo IS NULL OR …`). O `ORDER BY` é por `CASE`, uma chave por ordenação, e termina em `id`. `db/medicao_nfr4_test.go:31` tem a forma que a medição aprovou: `LIKE '%'||$1||'%'` + `categoria_id` + `BETWEEN`.
- `internal/busca/busca.go` -- `Listar` recebe um `Filtro{Termo, CategoriaID, PrecoMin, PrecoMax *…; Ordenacao}`. `Produto` não muda. `busca → catalogo` é aresta permitida (`internal/fronteira_test.go:29`), para chamar `catalogo.Normalizar` (`internal/catalogo/normalizar.go:21`).
- `api/busca.go` -- `listarProdutos` lê e valida os parâmetros novos com `erro.EscreverCampo`, no mesmo molde de `paginacaoDe` (`api/rotas.go:177`).
- `api/categoria.go:24` -- o handler administrativo `listarCategorias` e `saidaCategoria` servem de molde. A rota pública, no mux raiz de `api/rotas.go:50`, reusa o tipo sem copiar.
- `internal/plataforma/config.go:104` -- `BuscaTermoMax` já existe.
- `api/busca_test.go` -- o subteste `vitrine` e os helpers `produtoCom`/`categoriaCom`. O subteste novo entra em `api/sessao_test.go:482`, depois da Vitrine.
- `web/app/page.tsx` -- a Vitrine. `Paginacao` monta `?pagina=N` sozinho e precisa preservar os outros parâmetros.
- `web/components/ui/{sheet,select,badge,input,label,button}.tsx` -- prontos. `web/lib/preco.ts` formata os centavos.
- `mockups/resultados-busca.html` -- a composição: contagem e "Ordenar por" no topo, chips abaixo, filtros à esquerda.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/<ts>_catalogo_produto_criado_em.sql` -- coluna e VIEW. O `Down` volta à VIEW da 3.4.
- [x] `internal/busca/db/consultas.sql` + `sqlc generate` -- filtros opcionais, ordenação por `CASE` e a contagem com os mesmos filtros.
- [x] `internal/busca/busca.go` -- `Filtro`, normalização e escape do termo, e a passagem para a consulta.
- [x] `api/busca.go`, `api/categoria.go`, `api/rotas.go` -- ler e validar os parâmetros, e registrar `GET /api/v1/categorias`.
- [x] `api/busca_test.go` + `api/sessao_test.go` -- um subteste que percorre a matriz da API, com Produtos próprios de nome acentuado, `%` e `_` na descrição, e a paginação estável nas três ordenações.
- [x] `web/app/page.tsx`, `web/app/filtros.tsx` (client) -- a página lê os seis parâmetros da URL e repassa. Acima da grade vêm a contagem com `aria-live="polite"` e o `Select` "Ordenar por". Abaixo dele, os chips em `rounded-full`, cada um um link sem aquele parâmetro, e "Limpar filtros". O painel tem termo, Categoria e mín./máx. em reais: `Sheet` abaixo de 1024 px e fixo à esquerda a partir daí. O painel valida mín. > máx. antes de navegar. Uma URL que chega com a faixa invertida mostra a mensagem em linha e lista sem o filtro de preço. `CatalogoVazio` só aparece sem nenhum filtro.

**Acceptance Criteria:**
- Dado o Catálogo Semeado, quando o visitante busca "cafe" pelo painel, então vê os Produtos com "café" e a URL reproduz o resultado ao recarregar e ao voltar no navegador.
- Dada uma ordenação e um filtro ativos, quando o visitante pagina, então os dois continuam na URL e na tela.
- Dado 360 a 1440 px, quando os filtros são abertos, então não há rolagem horizontal, e o painel é `Sheet` abaixo de 1024 px.

## Design Notes

**Uma consulta, não três.** As três estórias mexem no mesmo `WHERE` e no mesmo `ORDER BY`. Os filtros opcionais por `IS NULL OR` mantêm uma consulta sqlc só. Com `ORDER BY CASE`, o plano não usa índice para ordenar, e a 5.050 Produtos isso não aparece no p95 (a 1.4 mediu 0,34 ms com o `ORDER BY` fixo).

**O termo fica no painel até a 3.10.** Decisão do humano: o campo é provisório, e a 3.10 o leva para a busca global da barra superior.

## Verification

**Commands:**
- `go build ./... && go vet ./... && go test -count=1 ./...` -- verde, incluindo o subteste novo, o `fronteira_test.go` e a medição do NFR-4.
- `cd web && npm run build` -- verde.

**Manual checks:**
- `docker compose up --build` → buscar "cafe" → filtrar por Categoria e faixa → trocar a ordenação → página 2 → voltar no navegador, a 360, 640, 1024 e 1440 px.

## Review Triage Log

Uma lente só (Edge Case Hunter), pela política do HANDOFF: sem concorrência nem máquina de estados. Foram 5 achados:

- **patch** — `api/busca.go`: um termo com `%00` passava pelo `TrimSpace` e pela normalização, e o Postgres o recusava com 500. Agora é 400 `campo:termo`, e o subteste cobre o caso.
- **patch** — `web/app/page.tsx`: com uuid de Categoria em maiúsculas, o filtro valia, mas o chip dizia "desconhecida" e nenhum rádio vinha marcado. O uuid agora é passado a minúsculo.
- **reject** — o teto de 100 caracteres fixo na tela: diverge só se `AZAMON_BUSCA_TERMO_MAX` mudar o padrão, que é o valor da FR-12.
- **reject** — preço com mais de 15 dígitos de centavos (acima de R$ 10 trilhões) some do filtro sem mensagem.
- **reject** — `Filtro.Ordenacao` desconhecida vinda de fora de `api/`: não existe outro chamador, e `api/` valida.

Fora da spec, feito pela implementação:
- a tela descarta da URL os valores que o Go recusaria (termo longo, preço que não é centavo, ordenação desconhecida, Categoria que não é uuid), em vez de mostrar página de erro;
- sem termo, a lista vazia diz "Nenhum Produto encontrado com estes filtros.";
- a configuração de teste ganhou `BuscaTermoMax`, e o subteste da Vitrine da 3.5 passou a esperar a ordem padrão `criado_em DESC, id`.

Não verificado:
- as linhas "Sem resultado" e "Faixa invertida na tela" da matriz: o `web/` não tem bancada de teste de componente;
- o passeio manual a 360, 640, 1024 e 1440 px, e os três Acceptance Criteria de tela;
- abaixo de 1024 px o formulário existe duas vezes no DOM (o `aside` oculto e o `Sheet`). Com a faixa invertida vinda da URL há dois `role="alert"`, um em cada cópia, e o oculto não é anunciado;
- o `loading.tsx` ainda mostra só a grade, sem o painel.

## Suggested Review Order

**A consulta única de `busca`**

- Entrada: termo, Categoria e faixa opcionais por `IS NULL OR`, e a mesma cláusula na contagem.
  [`consultas.sql:10`](../../internal/busca/db/consultas.sql#L10)

- Uma chave `CASE` por ordenação, e `id` fecha todo desempate.
  [`consultas.sql:16`](../../internal/busca/db/consultas.sql#L16)

- O termo passa pela mesma normalização da escrita, e só depois pelo escape do `LIKE`.
  [`busca.go:56`](../../internal/busca/busca.go#L56)

- `\`, `%` e `_` viram texto literal.
  [`busca.go:52`](../../internal/busca/busca.go#L52)

**"Mais recentes"**

- `criado_em` com `now()`: a semente empata, e o `id` desempata.
  [`20260918150000_catalogo_produto_criado_em.sql:6`](../../db/migracoes/20260918150000_catalogo_produto_criado_em.sql#L6)

**A borda HTTP**

- Toda validação dos parâmetros novos vive em um lugar, com erro em linha por campo.
  [`busca.go:55`](../../api/busca.go#L55)

- NUL recusado antes do banco (patch da revisão).
  [`busca.go:62`](../../api/busca.go#L62)

- Faixa invertida recusada no servidor com a mensagem da estória.
  [`busca.go:96`](../../api/busca.go#L96)

- Categorias da loja: mesmo handler da área administrativa, no mux raiz, sem Sessão.
  [`rotas.go:55`](../../api/rotas.go#L55)

**A tela**

- A URL crua vira estado aceito pelo Go; valor recusado cai calado.
  [`page.tsx:29`](../../web/app/page.tsx#L29)

- Faixa invertida vinda da URL: lista sem preço, com a mensagem no painel.
  [`page.tsx:66`](../../web/app/page.tsx#L66)

- Chips como links sem o próprio parâmetro.
  [`page.tsx:83`](../../web/app/page.tsx#L83)

- A chave remonta o painel a cada URL, então voltar no navegador não deixa valor velho.
  [`page.tsx:104`](../../web/app/page.tsx#L104)

- Contagem anunciada por `aria-live`.
  [`page.tsx:111`](../../web/app/page.tsx#L111)

- Validação antes de navegar: a listagem anterior fica na tela.
  [`filtros.tsx:81`](../../web/app/filtros.tsx#L81)

- Painel fixo a partir de 1024 px, `Sheet` abaixo.
  [`filtros.tsx:35`](../../web/app/filtros.tsx#L35)

- Paginar preserva termo, filtros e ordenação.
  [`page.tsx:170`](../../web/app/page.tsx#L170)

**Periféricos**

- A matriz da API num subteste, com o `strpos` da VIEW como oráculo.
  [`busca_test.go:123`](../../api/busca_test.go#L123)

- Paginação estável nas três ordenações, por 7.
  [`busca_test.go:224`](../../api/busca_test.go#L224)
