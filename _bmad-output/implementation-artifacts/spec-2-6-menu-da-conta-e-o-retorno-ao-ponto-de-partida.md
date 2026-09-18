---
title: 'Estória 2.6 — Menu da conta e o retorno ao ponto de partida'
type: 'feature'
created: '2026-09-17'
status: 'done'
route: 'dispatch'
baseline_commit: 'f7994c2bf52245b1084d9c367f29f830105ad407'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Não existe porta única para conta, Meus pedidos, Meus endereços e
Perfil — `/enderecos` só se alcança digitando a URL, e cada tela pública
duplica o mesmo cabeçalho cru. `identidade.Conta` não carrega e-mail, então
Perfil não tem o que exibir.

**Approach:** Extrair a casca (`<header>`) num componente só, e trocar a
`Saudacao` atual por `MenuDaConta` — o `DropdownMenu` do shadcn (UX-DR9) nas
oito telas públicas/Comprador. `Conta` e `saidaSessao` ganham `email`. Perfil e
uma listagem mínima de Pedidos (o esboço que a Estória 6.1 substitui) nascem
como novas rotas.

## Boundaries & Constraints

**Always:** o Menu está presente nas oito telas (as seis existentes + Perfil +
Pedidos) — UX-DR9; com Sessão mostra Meus pedidos, Meus endereços, Perfil,
Sair; sem Sessão mostra Entrar (por `paraLogin()`, preservando o destino) e
Criar conta; `DropdownMenu` do shadcn sem alteração (UX-DR1); ordem de
tabulação igual à de leitura, foco visível a 3:1 (UX-DR14); `GET
/api/v1/pedidos` filtra por `comprador_id` na própria consulta (AD-11), sem
`SELECT` e checagem depois; ordenado por `id DESC` — a chave é UUIDv7, ordenada
no tempo por construção, então o mais recente já sai no topo sem coluna nova;
Perfil mostra e-mail em leitura e "Trocar senha" leva ao `/esqueci-a-senha`
existente (FR-3) — nenhuma FR pede troca com senha atual; toda tela nova sem
Sessão redireciona a `/entrar` com `paraLogin()`, igual a `meus-enderecos.tsx`.

**Never:** construir a Estória 6.1 inteira — filtro por Status, ordenação,
consulta a 10 s, `Skeleton` (UX-DR20b) ficam para ela; formulário de trocar
senha com senha atual; busca global e Faixa de Categorias (3.10); tocar
`api/admin.go` ou o chrome administrativo; migração nova — nenhuma coluna
falta para nada disto.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Saída esperada | Erro |
|---|---|---|---|
| Header autenticado | Sessão de A | `DropdownMenu` com Meus pedidos, Meus endereços, Perfil, Sair | — |
| Header visitante | sem Sessão | Entrar (com destino) e Criar conta, sem dropdown | — |
| Listar Pedidos, dois donos | A tem 2 Pedidos, B tem 1 | `GET /api/v1/pedidos` de A devolve só os 2, mais recente primeiro | — |
| Listar sem nenhum | Sessão nova, sem Pedido | 200 com `[]` | — |
| Sem Sessão | sem cookie, `GET /api/v1/pedidos` | — | 401 `SESSAO_INVALIDA` |
| Perfil | Sessão de A | e-mail de A em leitura, link Trocar senha, botão Sair | — |
| Visitante em `/perfil` ou `/pedidos` | sem Sessão | redireciona a `/entrar?destino=...` e volta ao autenticar | — |

</frozen-after-approval>

## Code Map

- `internal/identidade/identidade.go:54-58` — `Conta` ganha `Email string
  \`json:"email"\``. `Autenticar:78` usa `linha.Email` (já vem da `SELECT` em
  `BuscarCompradorPorEmail`); `AutenticarAdministrador:101` idem com
  `linha.Email`; `Cadastrar:129` usa `normalizarEmail(email)`. Nenhuma consulta
  nova.
- `api/sessao.go:28-30` — `saidaSessao` ganha `Email string
  \`json:"email"\``. Os dois `escreverJSON(..., saidaSessao{Nome: conta.Nome})`
  (linhas ~112 e ~203) passam a incluir `Email: conta.Email`.
- `api/comprador.go:61` — mesmo ajuste no `saidaSessao` do cadastro.
- `internal/pedido/db/consultas.sql` — nova `-- name: ListarPedidosDoComprador
  :many`, molde de `BuscarPedidoDoComprador:41-47`: `SELECT id, numero,
  status, total_centavos FROM pedido.pedido WHERE comprador_id =
  @comprador_id ORDER BY id DESC`. Rodar `sqlc generate` (não está no PATH:
  `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`).
- `internal/pedido/pedido.go:185-200` — `Buscar` é o molde; nova `func
  Listar(ctx, bd gerado.DBTX, compradorID string) ([]Pedido, error)` usando
  `ListarPedidosDoComprador`.
- `api/pedido.go:107-142` — `lerPedido` é o molde do handler-com-dono; nova
  `listarPedidos` devolve `[]saidaPedido` (struct de `:25-30`, já pronta).
- `api/rotas.go:53` — `mux.HandleFunc("GET /api/v1/pedidos",
  s.listarPedidos)`, antes do catch-all `"/"`.
- `web/components/casca.tsx` (novo) — `Casca({children})`: o `<div
  className="min-h-svh"><header className="bg-chrome ...">` que hoje está
  repetido em `cadastrar/page.tsx:136`, `entrar/page.tsx:63`,
  `esqueci-a-senha/page.tsx:52`, `produtos/[id]/page.tsx:57`,
  `pedidos/[id]/page.tsx:21`, `enderecos/page.tsx:14`. `<main>` continua em
  cada página — as larguras divergem (`max-w-md`, `max-w-conteudo`,
  `max-w-2xl`).
- `web/components/menu-da-conta.tsx` (novo, `"use client"`) — substitui
  `web/app/produtos/[id]/saudacao.tsx` (apagar depois de migrar os três
  importadores). Mesmo fetch de `GET /api/v1/sessao`
  (`saudacao.tsx:16-23`... agora lendo `{nome, email}`) e o mesmo `sair()`
  (`saudacao.tsx:29-34`). Autenticado: `DropdownMenu`/`DropdownMenuTrigger`/
  `DropdownMenuContent`/`DropdownMenuItem`/`DropdownMenuSeparator`
  (`web/components/ui/dropdown-menu.tsx`) com Meus pedidos (`/pedidos`), Meus
  endereços (`/enderecos`), Perfil (`/perfil`), separador, Sair. Sem Sessão:
  `<a href={paraLogin()}>Entrar</a>` e `<a href="/cadastrar">Criar conta</a>`.
- `web/app/cadastrar/page.tsx`, `entrar/page.tsx`, `esqueci-a-senha/page.tsx`,
  `produtos/[id]/page.tsx`, `pedidos/[id]/page.tsx`, `enderecos/page.tsx` —
  trocar o `<div className="min-h-svh"><header>...` por `<Casca>`, e
  `<Saudacao />` por `<MenuDaConta />` onde já existia.
- `web/app/perfil/page.tsx` + `web/app/perfil/perfil.tsx` (novos) — molde é
  `enderecos/page.tsx` + `meus-enderecos.tsx`: Server Component casca +
  Client Component que busca `/api/v1/sessao`, mostra e-mail, link "Trocar
  senha" para `/esqueci-a-senha`, botão Sair (`sair()` de
  `menu-da-conta.tsx`). 401 → `paraLogin()` (`web/lib/destino.ts:31-35`).
- `web/app/pedidos/page.tsx` + `web/app/pedidos/meus-pedidos.tsx` (novos) —
  mesmo molde; busca `GET /api/v1/pedidos`, lista `numero` + rótulo do
  Status (rótulos fechados em `pedidos/[id]/acompanhamento.tsx:29-37` —
  copiar o mapa, é o esboço que a 6.1 substitui), cada linha um link para
  `/pedidos/{id}`. 401 → `paraLogin()`.
- `api/sessao_test.go:275-294` — o subteste "a Sessão do cookie é lida de
  volta" ganha a asserção de `email`. Novo `t.Run("Meus pedidos: lista só os
  do dono", ...)` perto da linha 367, no molde de
  `enderecosDoComprador` (`api/endereco_test.go:42-59`): Nara cadastra, lista
  vazia é `200 []`, cria dois Pedidos, outro Comprador cria um terceiro, a
  lista de Nara só traz os dela, mais recente primeiro.
- `README.md` — as duas rotas novas (`GET /api/v1/pedidos`, `/perfil`,
  `/pedidos`) onde as superfícies da loja são listadas, molde da 2.5.
- **Não tocar:** `api/admin.go`, `db/migracoes/`, `internal/catalogo/`,
  `internal/pagamento/`.

## Tasks & Acceptance

**Execution:**
- [x] `internal/identidade/identidade.go` -- `Conta.Email` + as três
  construções -- Perfil precisa do e-mail sem consulta nova.
- [x] `api/sessao.go`, `api/comprador.go` -- `saidaSessao.Email` -- é o
  mesmo envelope que `MenuDaConta` e Perfil consomem.
- [x] `internal/pedido/db/consultas.sql` + `sqlc generate` -- `ListarPedidosDoComprador` -- base da listagem.
- [x] `internal/pedido/pedido.go` -- `Listar` -- porta pública do módulo (AD-1).
- [x] `api/pedido.go` + `api/rotas.go` -- `listarPedidos` em `GET /api/v1/pedidos` -- dono verificado na consulta (AD-11).
- [x] `web/components/casca.tsx` -- extrai o cabeçalho -- fecha a duplicação em seis arquivos (deferred-work, spec-2-5).
- [x] `web/components/menu-da-conta.tsx` -- `DropdownMenu` com os quatro destinos e o estado sem Sessão -- é a própria estória (UX-DR8, UX-DR9).
- [x] Seis páginas existentes -- trocar cabeçalho cru por `<Casca>` + `<MenuDaConta />` -- unifica a casca.
- [x] `web/app/produtos/[id]/saudacao.tsx` -- apagar -- substituído por `MenuDaConta`.
- [x] `web/app/perfil/` (2 arquivos novos) -- e-mail em leitura, trocar senha, sair -- AC da estória.
- [x] `web/app/pedidos/` (2 arquivos novos) -- listagem mínima por dono -- esboço que a 6.1 substitui.
- [x] `api/sessao_test.go` -- asserção de `email` + subteste da listagem por dono -- prova o AD-11 na rota nova.
- [x] `README.md` -- rotas novas -- mesma disciplina da 2.5.

**Acceptance Criteria:**
- Dado qualquer uma das oito telas, quando ela renderiza, então o Menu da
  conta está presente e no mesmo lugar (UX-DR9).
- Dado um visitante sem Sessão que abre `/perfil` ou `/pedidos`, quando a
  tela detecta o 401, então ele vai a `/entrar?destino=...` e volta exatamente
  para lá ao autenticar.
- Dado `go test ./...`, quando a suíte roda, então `TestFronteiraDeModulo`,
  `TestDominioNaoConheceHTTP` e os testes do `.env` continuam verdes.
- Dado `cd web && npm run build`, quando o `prebuild` roda, então
  `verificar-offline.mjs` e `node --test` passam.
- Dado a UJ-1 inteira, quando percorrida só pelo teclado (Tab/Shift+Tab/Enter/
  Esc, sem mouse), então nenhuma etapa fica inalcançável (NFR-11) — inclusive
  abrir e navegar o `DropdownMenu` novo.

## Spec Change Log

## Design Notes

**Por que `email` entra em `Conta` e não numa consulta à parte.** As duas
linhas de login já trazem `email` da tabela (`BuscarCompradorPorEmail` e
`BuscarAdministradorPorEmail` fazem `SELECT id, nome, email, senha_hash`); só
não estava mapeado. Uma rota `GET /api/v1/perfil` separada pagaria uma ida a
mais ao Postgres pelo mesmo dado que a Sessão já carrega.

**Por que a listagem de Pedidos é `id DESC` e não por data.**
`pedido.pedido` não tem `criado_em` — só `transicao_status.ocorrido_em`, uma
tabela à parte. A chave primária é `uuidv7()`, ordenada no tempo por
construção (mesma convenção do resto do schema); somar um `JOIN` para uma
tela que a 6.1 substitui é trabalho que não sobra.

## Verification

**Commands:**
- `go build ./...` && `go test ./...` -- verde, com o novo subteste de
  `api/sessao_test.go`.
- `cd web && npm run build` -- `verificar-offline.mjs` e `node --test` verdes.

**Manual checks:**
- `docker compose up` → entrar → o Menu aparece em `/`, `/produtos/{id}`,
  `/pedidos/{id}`, `/enderecos`, `/perfil`, `/pedidos` → abrir cada destino.
- Sair pelo Menu → header volta a Entrar/Criar conta em toda tela.
- Visitar `/perfil` sem Sessão → cai em `/entrar?destino=/perfil` → entrar →
  volta a `/perfil`.
- Percorrer a UJ-1 (achar um Produto, comprar, ver o Pedido) só pelo teclado,
  abrindo o Menu da conta no caminho — usar a extensão do Chrome se
  conectada; senão registrar em `deferred-work.md`, como as estórias 2.1-2.3.

## Suggested Review Order

**Identidade da Sessão ganha e-mail**

- `Conta` ganha `Email` — o fato que destrava o Perfil sem consulta nova.
  [`identidade.go:58`](../../internal/identidade/identidade.go#L58)

- `Autenticar` reaproveita o `email` que a própria linha já trazia.
  [`identidade.go:83`](../../internal/identidade/identidade.go#L83)

- `AutenticarAdministrador` deliberadamente NÃO ganhou `Email` — patch pós-revisão para não vazar o e-mail do Administrador de forma assimétrica entre as duas rotas de Sessão administrativa.
  [`identidade.go:94`](../../internal/identidade/identidade.go#L94)

- `saidaSessao` ganha `Email` — o mesmo envelope que `MenuDaConta` e o Perfil consomem.
  [`sessao.go:32`](../../api/sessao.go#L32)

**Menu da conta (UX-DR9)**

- `MenuDaConta`: autenticado abre o `DropdownMenu` do shadcn; sem Sessão mostra Entrar/Criar conta.
  [`menu-da-conta.tsx:24`](../../web/components/menu-da-conta.tsx#L24)

- `Casca` extrai o cabeçalho repetido nas seis telas antigas — fecha a lacuna registrada na 2.5.
  [`casca.tsx:10`](../../web/components/casca.tsx#L10)

- `sair()` foi para o próprio módulo no patch pós-revisão, para o Perfil não importar de um componente de UI.
  [`sessao.ts:10`](../../web/lib/sessao.ts#L10)

**Meus pedidos (o esboço que a 6.1 substitui)**

- `listarPedidos` resolve o dono pela Sessão; sem filtro, ordenação ou `Skeleton` por decisão de escopo.
  [`pedido.go:147`](../../api/pedido.go#L147)

- `Listar` — o dono entra no `WHERE` (AD-11), nunca numa checagem depois da leitura.
  [`pedido.go:215`](../../internal/pedido/pedido.go#L215)

- `ListarPedidosDoComprador` usa `id DESC`: a UUIDv7 já ordena no tempo, sem coluna nova.
  [`consultas.sql:54`](../../internal/pedido/db/consultas.sql#L54)

- `GET /api/v1/pedidos` registrado antes do padrão `{id}` — sem colisão de rota (AD-16).
  [`rotas.go:56`](../../api/rotas.go#L56)

- `MeusPedidos`: 401 leva a `paraLogin()`, o mesmo padrão de `meus-enderecos.tsx`.
  [`meus-pedidos.tsx:44`](../../web/app/pedidos/meus-pedidos.tsx#L44)

- `Preco` foi copiado, não importado — o arquivo inteiro é o esboço que a 6.1 substitui.
  [`meus-pedidos.tsx:33`](../../web/app/pedidos/meus-pedidos.tsx#L33)

**Perfil**

- E-mail em leitura, "Trocar senha" leva ao fluxo de FR-3 já existente.
  [`perfil.tsx:17`](../../web/app/perfil/perfil.tsx#L17)

**Adoção da `Casca` nas seis telas antigas**

- `enderecos/page.tsx` é a tela para a qual o Menu da conta foi construído — antes só se alcançava digitando a URL.
  [`enderecos/page.tsx:13`](../../web/app/enderecos/page.tsx#L13)

- `entrar/page.tsx` e `cadastrar/page.tsx` ganham o Menu pela primeira vez — antes não tinham `Saudacao` nenhuma.
  [`entrar/page.tsx:64`](../../web/app/entrar/page.tsx#L64)

**Peripherais (testes, docs, rastreamento)**

- `meusPedidosListaPorDono` prova o recorte por dono na rota nova, no molde de `enderecosDoComprador`.
  [`pedido_test.go:347`](../../api/pedido_test.go#L347)

- A leitura da Sessão passa a afirmar `email` também.
  [`sessao_test.go:275`](../../api/sessao_test.go#L275)

- O e-mail normalizado do cadastro passa a ser afirmado de ponta a ponta — patch pós-revisão.
  [`comprador_test.go:49`](../../api/comprador_test.go#L49)

- `README.md` documenta o Menu, o Perfil e a rota nova de listagem.
  [`README.md:111`](../../README.md#L111)
