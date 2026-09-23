---
title: 'Estória 6.1 — Meus pedidos'
type: 'feature'
created: '2026-09-23'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
baseline_commit: '4cbea8baadb86d95b3cbcc983bfb4726666ab261'
context: ['{project-root}/_bmad-output/implementation-artifacts/epic-6-context.md']
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `GET /api/v1/pedidos` é o esboço deixado pela 2.6: array cru, sem envelope, sem teto e **sem data** — dívida já registrada no ledger. A tela `meus-pedidos.tsx` mostra número, rótulo e total sem estado de carregamento e sem estado vazio, e reimplementa à mão o `Preco` e o mapa de rótulos. O Comprador não encontra o Pedido de ontem, e o payload cresce com o histórico sem tamanho declarado.

**Approach:** A lista passa a ser uma listagem de verdade: a mesma consulta que carrega o dado verifica a posse, ordena do mais recente para o mais antigo com desempate terminando em `id`, pagina em 20 por página e responde no envelope único do AD-18 com `total`. A data do Pedido vem do nascimento já gravado no histórico — **nenhuma coluna nova, nenhuma migração**. A tela ganha `Skeleton`, estado vazio com caminho único para a Vitrine, paginação pela URL, e passa a usar os componentes compartilhados em vez das cópias locais.

## Boundaries & Constraints

**Always:**
- Posse no `WHERE` da própria consulta (`comprador_id = @comprador_id`), nunca filtro na aplicação depois.
- A contagem repete o `WHERE` da listagem, palavra por palavra; página além do total curto-circuita **antes** de ir ao banco, no mesmo formato de `busca.Listar`.
- Envelope `listagem[T]` e `paginacaoDe` de `api/rotas.go` usados como estão — `AZAMON_PAGINA_TAMANHO` já vale 20 e o teto já vale 60.
- Instante absoluto em RFC 3339 na resposta; formatação pt-BR só na tela, dentro de `<time dateTime>`.
- `Skeleton` no carregamento, nunca spinner centralizado; o `null` inicial **não** vira o texto de vazio.
- `pedido.Listar` continua recebendo `gerado.DBTX`, para seguir transacionável.

**Never:**
- Migração, coluna `criado_em`, índice novo ou variável de configuração nova.
- Tocar `pedido.Detalhar`, `internal/pedido/maquina.go`, `web/app/pedidos/[id]/*`, `web/components/ui/*`, `internal/fronteira_test.go`.
- Cancelamento, filtro por Status, busca por número de Pedido, Detalhe pleno, consulta em intervalo nesta tela — são 6.2 a 6.4.
- Verde ou laranja nesta tela: os dois têm sentido fechado (`ENTREGUE`/Estoque disponível e o passo irreversível).

**Decisão — a forma de exibição do Status.** A 6.1 extrai **só o rótulo** dos sete Status para `web/lib/pedido.ts`, com teste em `web/scripts/pedido.test.mjs`, e exibe Status como **texto**. Sem `Badge`, sem variante de cor, sem a classe do verde de `ENTREGUE` — a 6.7 continua dona de fechar o selo nas três superfícies de uma vez. `acompanhamento.tsx` não é tocado nesta estória (está em review): a cópia que sobra lá é a 6.7 que recolhe.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Primeira página | Comprador com 25 Pedidos, sem query | 20 itens, mais recente primeiro; `pagina` 1, `por_pagina` 20, `total` 25; cada item com `numero`, `criado_em`, `total_centavos`, `status` | N/A |
| Página além do total | `?pagina=9`, total 25 | `itens: []` (nunca `null`), `total` 25, sem consulta de linhas | N/A |
| Comprador sem Pedido | nenhum Pedido | `itens: []`, `total` 0; tela mostra "Você ainda não fez nenhum Pedido." com ação única para a Vitrine | N/A |
| Histórico alheio | outro Comprador tem Pedidos | nenhum deles aparece, nem por manipulação de query | N/A |
| Sem Sessão | sem cookie de Sessão | tela vai a `paraLogin()` | 401 `SESSAO_INVALIDA` |
| Paginação fora de faixa | `?por_pagina=999` / `?pagina=abc` | 999 é rebaixado a 60; `abc` é recusado | 400 de campo |

</frozen-after-approval>

## Code Map

- `internal/pedido/db/consultas.sql:124` -- `ListarPedidosDoComprador` a alterar: acrescentar o instante de nascimento, `LIMIT`/`OFFSET`; e uma `ContarPedidosDoComprador` nova com o mesmo `WHERE`.
- `internal/pedido/pedido.go:551` -- `Listar` a alterar: recebe página e tamanho, devolve `([]Pedido, int64, error)`; `Pedido` (:52) ganha o instante. Modelo a copiar: `internal/busca/busca.go:55` (conta primeiro, curto-circuita, fatia vazia não-nil).
- `internal/pedido/pedido.go:302` -- o nascimento gravado no histórico com `status_anterior` vazio, na mesma transação do `INSERT`: é a fonte da data.
- `api/pedido.go:36` -- `saidaPedido` ganha o instante; `api/pedido.go:324` -- o handler passa a usar `paginacaoDe` e `listagem[T]`.
- `api/rotas.go:191,202` -- `listagem[T]` e `paginacaoDe`: reusar, não tocar. Handler de referência: `api/busca.go:25`.
- `api/pedido_test.go:475` -- afirma array cru; a mudança de contrato é intencional e o teste acompanha.
- `web/app/pedidos/meus-pedidos.tsx` -- a reescrever: envelope, data, `Skeleton`, estado vazio, paginação; apagar o `Preco` local (:33) e usar `@/components/preco`.
- `web/app/pedidos/page.tsx` -- `<Suspense>` em volta se a tela ler `useSearchParams`, senão o build do Docker quebra.
- `web/app/admin/produtos/produtos.tsx:105,386` -- paginação cliente por `?pagina=`: é o padrão a copiar.
- `web/lib/pedido.ts` + `web/scripts/pedido.test.mjs` -- só acrescentar; é onde função pura de Pedido mora e é testada.
- `web/lib/pedir.ts`, `web/lib/destino.ts`, `web/lib/preco.ts` -- `pedir`, `paraLogin`, `formatarPreco`: reusar.

## Tasks & Acceptance

**Execution:**
- [x] `internal/pedido/db/consultas.sql` -- alterar `ListarPedidosDoComprador` (instante de nascimento por subconsulta com cast, `ORDER BY` do mais recente terminando em `id`, `LIMIT`/`OFFSET`) e acrescentar `ContarPedidosDoComprador` -- a posse e o teto passam a viver no SQL.
- [x] `internal/pedido/db/gerado/` -- regerar por `sqlc generate` na imagem `sqlc/sqlc:1.31.1` e restaurar com `git checkout` os módulos não tocados -- o gerador reescreve todos os `gerado/` em LF.
- [x] `internal/pedido/pedido.go` -- `Listar` com página, tamanho e total; `Pedido` com o instante -- contrato do envelope sem vazar `gerado` para fora da porta.
- [x] `api/pedido.go` -- handler por `paginacaoDe` + `listagem[T]`; `saidaPedido` com o instante em RFC 3339 -- envelope único do AD-18.
- [x] `api/pedido_test.go` -- cobrir a matriz de E/S: primeira página, página além do total, Comprador sem Pedido, histórico alheio invisível, `por_pagina` rebaixado -- é a verificação da estória.
- [x] `web/lib/pedido.ts` + `web/scripts/pedido.test.mjs` -- acrescentar `rotuloDoStatus` (os sete Status, com o identificador cru como fallback) e o teste da função pura -- passa a existir um ponto único de rótulo; só acrescentar, nunca reescrever o que a 5.8 deixou.
- [x] `web/app/pedidos/meus-pedidos.tsx` -- reescrever: número, data, total e Status; `Skeleton`; estado vazio; paginação por `?pagina=`; componentes compartilhados -- fecha as três dívidas do ledger apontadas a esta estória.
- [x] `web/app/pedidos/page.tsx` -- `<Suspense>` se necessário -- sem isso o build quebra, não o comportamento.

**Acceptance Criteria:**
- Dado um Comprador autenticado com Pedidos de datas diferentes, quando abre "Meus pedidos", então vê número, data, total e Status, do mais recente para o mais antigo, e só os seus.
- Dado um Comprador com mais de 20 Pedidos, quando avança a paginação, então a página seguinte continua a ordem sem repetir nem perder Pedido, e o número da página sobrevive a recarregar a tela.
- Dado que a primeira resposta ainda não chegou, quando a tela está montada, então há `Skeleton` — nunca spinner, nunca o texto de vazio.
- Dado um Comprador sem nenhum Pedido, quando a tela abre, então aparece "Você ainda não fez nenhum Pedido." com uma única ação: voltar à Vitrine.

## Implementation Notes

- **`criado_em` é `omitempty`.** `saidaPedido` é compartilhada com a criação, a nova Tentativa e o Detalhe (embutida em `saidaPedidoDetalhe`), e nenhuma dessas leituras preenche `CriadoEm` — `Detalhar` está na lista de não tocar. Sem `omitempty`, as três passariam a responder `"criado_em":"0001-01-01T00:00:00Z"`, que é data errada, não ausência. Se a 6.2 quiser o instante no Detalhe, é `Detalhar` que passa a lê-lo.
- **A Sessão é conferida antes da paginação** em `listarPedidos`: quem não tem Sessão leva 401, e não um 400 de campo que contaria que a rota existe e como ela pagina. O teste prende isso com `?pagina=abc` sem cookie.
- **Dois Pedidos no teste, não os 25 da matriz.** O Estoque do `produtoSemeado` é compartilhado com os outros subtestes de `api/` e o terceiro Pedido derrubava o subteste de posse com `ESTOQUE_INSUFICIENTE`. A paginação é provada com `?por_pagina=1` em duas páginas; que o padrão é 20 e o teto 60 fica provado pelas constantes `paginaTamanhoDeTeste`/`paginaTamanhoMaxDeTeste` que o próprio subteste afirma. Os números da matriz eram ilustração do comportamento, não fixture.
- **Estado extra não pedido:** com `total > 0` e a página vazia (`?pagina=9`), a tela diz "Esta página não tem Pedidos." com volta ao começo, em vez do estado vazio da conta — dizer "Você ainda não fez nenhum Pedido" a quem tem Pedidos seria falso.
- **Comentário `ponytail:` novo** em `consultas.sql`, nomeando o teto da subconsulta por linha na ordenação e o caminho de upgrade que a própria spec rejeita.
- `envelopeDeMeusPedidos` em `api/pedido_test.go` é a terceira cópia da mesma forma de envelope no pacote de teste (`envelopeDaVitrine`, e a anônima de `produto_admin_test.go`). Unificar tocaria dois arquivos fora do escopo desta estória; fica registrado, não feito.

## Spec Change Log

## Review Triage Log

Passe 1 — três camadas (`blind-hunter`, `edge-case-hunter`, `verification-gap`).

| # | Achado | Veredito | Evidência | Rota |
|---|--------|----------|-----------|------|
| 1 | `criado_em` só é afirmado como "existe e é RFC 3339"; trocar `min` por `max` na consulta passa verde | high | Confirmado: os dois Pedidos do subteste têm exatamente uma transição, logo `min == max`, e o desempate por `p.id DESC` reproduz a mesma ordem. A data do nascimento é o entregável central da estória e nenhuma asserção a prende | patch |
| 2 | Rodapé e navegação não limitam a página: `?pagina=9` com 1 página mostra "Página 9 de 1" e oferece "Anterior" para outra página vazia | medium | Confirmado em `meus-pedidos.tsx`: `paginas` vem do envelope, `lista.pagina` vem do servidor sem limite, e o bloco de navegação só testa `pagina > 1`. É a linha `?pagina=9` da matriz, a única que a tela trata errado | patch |
| 3 | `api/pedido_test.go` indexa `env.Itens[0]` depois de um `t.Errorf` não fatal | medium | Confirmado: primeira página vazia vira `index out of range`, que aborta o binário inteiro de `TestSessaoEProduto` e leva junto o sinal dos outros subtestes | patch |
| 4 | `criado_em` nulo/ausente: `ORDER BY ... DESC` é `NULLS FIRST`, `CriadoEm.Valid` falso vira zero silencioso, e o tipo TS declara o campo obrigatório enquanto o Go usa `omitempty` | low | Confirmado como forma, não como caminho: `Criar` sempre grava a transição de nascimento na mesma transação, então é inalcançável pelo aplicativo — mas vários testes inserem `pedido.pedido` direto. O Pedido degenerado iria ao topo da página 1 e sem data | patch |
| 5 | A tela monta `fetch` à mão e repete o literal de falha de rede, em vez de `pedir`/`FALHA_DE_REDE` | low | Confirmado: `web/lib/pedir.ts` exporta os dois, o Code Map os lista como reuso e `admin/produtos` — o padrão mandado copiar — usa `pedir` | patch |
| 6 | `vazia.Code` nunca é conferido e o helper força uma segunda requisição idêntica | low | Confirmado: uma resposta 500 sem cabeçalho seria reportada como "Cache-Control vazio", escondendo o 500 | patch |
| 7 | O comentário da consulta afirma que o desempate "não depende do relógio", mas a chave primária da ordenação é `min(ocorrido_em)`, que é relógio | low | Confirmado: `id` só desempata empates exatos; a ordenação principal é temporal. O comentário descreve o desenho anterior (`id DESC` puro) | patch |
| 8 | O laço de paginação do teste não tem teto: servidor que ignore `OFFSET` roda até o timeout | low | Confirmado: `for pagina := 1; ; pagina++` só sai com página vazia | patch |
| 9 | A decisão do `omitempty` não tem asserção: removê-lo faria criação, nova Tentativa e Detalhe responderem com o ano 1 | low | Confirmado: `saidaPedido` é compartilhada pelas três e nenhuma afirma a ausência do campo | patch |
| 10 | `acompanhamento.tsx` mantém o mapa de rótulos local, então o "ponto único" ainda não é único | low | Real, e explicitamente excluído pelo bloco congelado: a decisão humana foi a opção (a), com a cópia restante recolhida pela 6.7. Reduziu de duas cópias à mão para uma cópia + ponto compartilhado | defer |
| 11 | `Listar` com `porPagina` 0 ou `pagina` < 1: divisão por zero ou `OFFSET` negativo | false | `paginacaoDe` (`api/rotas.go:208`) recusa não-inteiro e `n < 1` com 400 de campo antes de `Listar` ser chamada; nenhum caminho produz o gatilho | — |
| 12 | Corrida entre `ContarPedidosDoComprador` e `ListarPedidosDoComprador` desloca páginas | low | Real e inerente à paginação por `OFFSET` — idêntico a `busca.Listar`, o padrão que a spec mandou copiar. O Comprador teria de criar Pedido enquanto pagina a própria lista; o pior caso é uma linha repetida entre páginas. Rejeitado: improvável no uso comum e a correção (transação `REPEATABLE READ`) acrescenta complexidade e diverge do repo | rejeitado |
| 13 | Resposta 200 com corpo não-JSON deixa `Skeleton` eterno ou estoura em `lista.itens` | low | Real na forma, inalcançável com o próprio servidor, que sempre devolve o envelope. Rejeitado: a correção acrescenta ramo de guarda para estado não demonstrado | rejeitado |
| 14 | `?pagina=` gigantesco escapa do limite inferior e vira 400 com alerta genérico | low | Real e exótico (exige URL digitada à mão com número acima de `MAX_SAFE_INTEGER`). Rejeitado pelo mesmo critério da linha 13 | rejeitado |
| 15 | Falha ao carregar a página 2 deixa os itens da página 1 na tela | false | A paginação navega por `<a href>` — documento inteiro recarrega e o componente remonta com `lista` em `null`. Não há troca de página sem remontagem | — |
| 16 | O `Preco` compartilhado é `<p>` e a margem de parágrafo desalinha as colunas | false | `web/app/globals.css:1` importa `tailwindcss`, cujo preflight zera `margin` de todos os elementos; o `<p>` é item de flex sem margem | — |
| 17 | O rastreamento não acompanha o diff (`sprint-status.yaml` em `in-progress`, `HANDOFF.md` termina na 5.11) | false | `in-progress` é o estado correto enquanto a estória corre; o avanço para `review` e a atualização do `HANDOFF` são o encerramento, que ainda não aconteceu | — |

## Design Notes

**A data sem migração.** `pedido.Criar` grava a primeira linha do histórico com `status_anterior` vazio na mesma transação do `INSERT` (`pedido.go:302`), então `min(ocorrido_em)` de `pedido.transicao_status` **é** o instante de nascimento, não uma aproximação. `BuscarPedidoDoComprador` já usa exatamente essa forma para `atualizado_em` (com `max`), inclusive o cast explícito sem o qual o sqlc devolve `interface{}` e só falha no `Scan`:

```sql
(SELECT min(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id)::timestamptz AS criado_em
```

O índice `(pedido_id, ocorrido_em)` já existe. O teto disto é a subconsulta por linha na ordenação; no tamanho deste sistema não paga índice novo nem coluna denormalizada, e a alternativa — `criado_em` em migração nova — seria irreversível e mentiria sobre os Pedidos já gravados.

**Ordenação.** `id` é uuidv7, logo `ORDER BY id DESC` já é cronológico e já é desempate estável por si. A ordenação declarada termina em `id` de todo modo: o `OFFSET` fica estável mesmo que dois Pedidos de transações concorrentes empatem no instante.

## Verification

**Commands:**
- `docker run --rm -v "$PWD":/src -w /src sqlc/sqlc:1.31.1 generate` -- esperado: `internal/pedido/db/gerado/` regerado sem erro; `git diff --ignore-cr-at-eol --stat` mostra só `pedido`.
- `docker run --rm -v "$PWD":/src -w /src -v azamon-gocache:/root/.cache azamon-dev:latest go vet ./...` -- esperado: silêncio.
- `docker run --rm -v "$PWD":/src -w /src -v azamon-gocache:/root/.cache -v //var/run/docker.sock:/var/run/docker.sock -e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal -e TESTCONTAINERS_RYUK_DISABLED=true azamon-dev:latest go test -count=1 ./...` -- esperado: tudo passa, inclusive a matriz nova e `internal/fronteira_test.go`.
- `docker build ./web` -- esperado: build completo (o `prebuild` roda `verificar-offline.mjs` e `npm test` dentro da imagem).
