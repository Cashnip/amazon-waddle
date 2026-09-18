---
title: '3.1 — Gestão de Vendedores'
type: 'feature'
created: '2026-09-18'
status: 'done'
baseline_commit: '59324948f6181c3b90982f705ea9fc60744e484d'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-3-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O Administrador não tem tela nenhuma: a 2.4 deixou só o login e a guarda de `/api/v1/admin/`. E Vendedor desativado não esconde nada, porque a Página de Produto e o `Reservar` leem `catalogo.produto` cru.

**Approach:** Criar o CRUD de Vendedor sob a guarda administrativa e a primeira área administrativa do `web/`: login, Navegação administrativa e a tela Vendedores. O "Produto visível" nasce como uma VIEW única, `catalogo.produto_visivel`, por enquanto só com a regra "Vendedor ativo" — a 3.4 acrescenta "Produto ativo" nela.

## Boundaries & Constraints

**Always:** as rotas novas entram no mux `admin` e herdam a guarda (404 para quem não é Administrador). O teto do nome vem de `AZAMON_VENDEDOR_NOME_MAX` (padrão 120, contado em runas), e as bordas do nome são aparadas antes de validar. Desativar é um `UPDATE` de `ativo`: nenhum Pedido é tocado. Remover conta com a FK `produto.vendedor_id` (SQLSTATE 23503), sem `SELECT count` antes. Nome duplicado é decidido pelo `UNIQUE` (23505). As mensagens de erro vêm do envelope do servidor, com `dados.campo` para o erro em linha.

**Ask First:** qualquer mudança na assinatura de `Reservar` ou no status de erro dele. Paginar a lista de Vendedores.

**Never:** escrever `WHERE ativo` em outro lugar além da VIEW (AD-19). Server Action ou regra no Node (AD-10). `catalogo.Visiveis`, `Disponivel` e o "Produto ativo" (são da 3.4). A Vitrine e a busca (3.5+). Sheet de navegação abaixo de 640 px enquanto houver um destino só.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Esperado | Erro |
|---|---|---|---|
| Listar | `GET /api/v1/admin/vendedores` | 200, `[]` ou `[{id,nome,ativo}]` por nome, ativos e inativos | — |
| Criar | `POST` `{"nome":" Loja X "}` | 201 `{id,nome:"Loja X",ativo:true}` | — |
| Nome vazio/longo | `nome` só de espaços ou acima do teto | — | 400 `CAMPO_INVALIDO`, `campo:nome`, com o limite na mensagem |
| Nome duplicado | nome que já existe (criar ou editar) | — | 409 `VENDEDOR_JA_CADASTRADO`, `campo:nome` |
| Editar/desativar | `PUT /{id}` `{"nome","ativo":false}` | 200 com a linha nova | inexistente ou uuid malformado → 404 |
| Remover sem Produto | `DELETE /{id}` | 204 | inexistente → 404 |
| Remover com Produto | `DELETE /{id}` de Vendedor com Produto | — | 409 `VENDEDOR_COM_PRODUTOS`, "…Desative-o no lugar." |
| Produto de inativo | `GET /api/v1/produtos/{id}` | — | 404 `NAO_ENCONTRADO` |
| Comprar de inativo | `POST /api/v1/pedidos` com esse Produto | — | o mesmo status de hoje para Produto inexistente; nenhuma Reserva é gravada |
| Pedido existente | Pedido feito antes de desativar | `GET /api/v1/pedidos/{id}` idêntico | — |
| Sem Sessão de Administrador | qualquer rota nova | — | 404 (guarda) |

</frozen-after-approval>

## Code Map

- `db/migracoes/20260911120200_catalogo_schema.sql` -- `vendedor(id, nome UNIQUE, ativo)` já existe; `produto.vendedor_id` já tem FK. Nada muda no schema de tabela.
- `internal/catalogo/db/consultas.sql:5` -- `BuscarProdutoComVendedor` passa a ler da VIEW. `:19` `TravarProdutosParaReserva` ganha a visibilidade na mesma consulta `FOR UPDATE`.
- `internal/catalogo/catalogo.go` -- `BuscarProduto` e `Reservar` não mudam de assinatura. Quando `travados` sai vazio, o retorno continua `pgx.ErrNoRows`.
- `internal/identidade/endereco.go` -- molde do CRUD: `uuidDe` (malformado → `ErrNoRows`), `:execrows` → 0 vira `ErrNoRows`. `identidade.go:133` é o molde do 23505 → sentinela.
- `api/endereco.go` -- molde dos handlers: `semCache`, `decodificarCorpo`, `erro.EscreverCampo`, lista vazia como `[]`. `api/comprador.go:50` mostra o 409 com `dados.campo`.
- `api/rotas.go:80` -- o mux `admin`; rota nova entra aqui.
- `api/admin.go` -- o comentário `ponytail:` fica como está: os handlers de Vendedor não precisam da Conta.
- `internal/plataforma/erro/erro.go:44` -- `registro`; duas linhas novas.
- `internal/plataforma/config.go:25,86` -- molde de `CompradorNomeMax`.
- `api/sessao_test.go:145` -- `ambiente(t)` monta a `Config` à mão; subteste novo é registrado perto de `:447`. `api/autorizacao_test.go:101` usa `postarAdmin`, e `emailAdmin`/`senhaAdmin` estão em `sessao_test.go:40`.
- `web/app/entrar/page.tsx`, `web/app/enderecos/meus-enderecos.tsx` -- moldes de formulário, erro em linha com foco, `Dialog` de confirmação e 401 → login.
- `web/components/casca.tsx`, `web/lib/sessao.ts` (`sair()`) -- a casca da loja e a saída. `--chrome-muted` está em `globals.css:106`.
- sqlc: `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`; o código gerado é commitado.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/20260918120000_catalogo_produto_visivel.sql` -- `CREATE VIEW catalogo.produto_visivel` = `produto` + `vendedor.nome AS vendedor_nome`, com `JOIN vendedor ... WHERE v.ativo`. Down derruba a VIEW -- é o predicado do AD-19 num lugar só.
- [x] `internal/catalogo/db/consultas.sql` -- `BuscarProdutoComVendedor` lê de `produto_visivel`. `TravarProdutosParaReserva` fica `FROM produto p WHERE p.id = ANY(..) AND p.id IN (SELECT id FROM catalogo.produto_visivel) ORDER BY p.id FOR UPDATE OF p`: trava só o Produto, nunca o Vendedor. Novas consultas: `ListarVendedores` (por nome, id), `CriarVendedor`, `AtualizarVendedor :one`, `RemoverVendedor :execrows`. Depois, regerar com sqlc.
- [x] `internal/catalogo/vendedor.go` -- tipo `Vendedor{ID,Nome,Ativo}`, `ListarVendedores`, `CriarVendedor`, `AtualizarVendedor`, `RemoverVendedor`. Sentinelas: `ErrVendedorJaCadastrado` (23505) e `ErrVendedorComProdutos` (23503). Inexistente ou malformado → `pgx.ErrNoRows`.
- [x] `internal/plataforma/config.go` -- `VendedorNomeMax` ← `AZAMON_VENDEDOR_NOME_MAX`, padrão 120.
- [x] `internal/plataforma/erro/erro.go` -- as duas sentinelas, ambas 409.
- [x] `api/vendedor.go` + `api/rotas.go` -- `GET/POST /api/v1/admin/vendedores` e `PUT/DELETE /api/v1/admin/vendedores/{id}` no mux `admin`.
- [x] `api/vendedor_test.go` + `api/sessao_test.go` -- um subteste que percorre a matriz inteira, com `VendedorNomeMax` na `Config` de teste. O Vendedor é criado pela API e o Produto dele é inserido pelo `pool`: nunca desativar um Vendedor semeado, porque os outros subtestes dependem dele.
- [x] `web/components/casca-admin.tsx` -- client. `GET /api/v1/admin/sessao`: 404 → `router.replace("/admin/entrar")`, e nada renderiza até vir 200. Header em `bg-chrome-muted` com wordmark, destinos (hoje só "Vendedores"), o nome da Sessão e "Sair" (`sair()`).
- [x] `web/app/admin/entrar/page.tsx` -- login contra `POST /api/v1/admin/sessoes`, no molde de `/entrar`, sem `Casca` da loja. Sucesso → `/admin/vendedores`.
- [x] `web/app/admin/page.tsx` -- `redirect("/admin/vendedores")`.
- [x] `web/app/admin/vendedores/page.tsx` + `vendedores.tsx` -- `Table` com nome, situação (Ativo/Inativo) e ações Editar, Desativar/Reativar e Remover. Formulário de criar/editar com erro em linha e foco. Remover pede confirmação em `Dialog`; no 409, `Alert` com a mensagem e o botão "Desativar" dentro dele. Lista vazia: "Nenhum Vendedor cadastrado." com a ação única "Cadastrar Vendedor".
- [x] `README.md` -- a área administrativa fica em `/admin`.

**Acceptance Criteria:**
- Dado um Vendedor com Produto e um Pedido desse Produto, quando o Vendedor é desativado, então a Página de Produto responde 404, um Pedido novo não reserva, e o Pedido antigo é lido idêntico.
- Dado um Comprador, ou nenhuma Sessão, quando `/admin/vendedores` é aberto, então a tela não aparece e o navegador cai em `/admin/entrar`.
- Dado o Administrador semeado, quando ele entra em `/admin/entrar`, então vê a lista dos cinco Vendedores semeados e consegue criar, renomear, desativar, reativar e remover sem sair da tela.

## Design Notes

A Navegação administrativa mostra só os destinos que existem. Link para Categorias, Produtos e Pedidos antes das estórias deles seria link para tela nenhuma; cada estória acrescenta o seu destino, e o `Sheet` abaixo de 640 px entra quando houver mais de um.

`FOR UPDATE` sobre a VIEW travaria também a linha do Vendedor, e todo Pedido de Produtos diferentes do mesmo Vendedor passaria a esperar um pelo outro. Por isso a consulta trava `OF p` e consulta a visibilidade por subconsulta. A janela que isso deixa (Vendedor desativado no meio de uma Reserva) é aceita; a 3.4 revisita o predicado inteiro.

## Verification

**Commands:**
- `go build ./... && go vet ./... && go test ./...` -- verde, com o subteste novo e `TestFronteiraDeModulo`.
- `cd web && npm run build` -- verde, com `verificar-offline.mjs` e `node --test`.

**Manual checks:**
- `docker compose up` → `/admin` sem Sessão cai em `/admin/entrar` → entrar com `admin@azamon.test` → criar, renomear, desativar e remover Vendedor → tentar remover "Atlântico Importados" mostra o `Alert` com Desativar → desativá-lo e abrir um Produto dele na loja dá "não encontrado" → reativar.

## Review Triage Log

Uma lente só (Edge Case Hunter), pela política de revisão do projeto para CRUD. Foram 7 achados:

- **patch** — `vendedores.tsx`: se o Vendedor em edição saía da lista, o "Salvar" caía no `POST` e criava um Vendedor novo. Agora avisa "Vendedor não encontrado." e recarrega a lista.
- **patch** — `vendedor_test.go`: a ordem da lista era conferida com `slices.IsSorted` (ordem de bytes do Go), e não pela collation do Postgres. Agora compara com a lista semeada literal.
- **reject** (5):
  - a janela entre desativar o Vendedor e uma Reserva em andamento, já aceita em Design Notes;
  - a edição simultânea por dois Administradores, porque só um está semeado;
  - `ativo:false` ignorado no `POST`, porque o Vendedor nasce ativo, pela matriz;
  - duplicado só pela caixa das letras, porque o `UNIQUE` decide, como manda Always;
  - nome feito de caracteres invisíveis.

Fora da spec, feito pela implementação: `db/schema_test.go` passou a contar só `BASE TABLE`, senão a VIEW quebraria a contagem de tabelas. `.env` ganhou `AZAMON_VENDEDOR_NOME_MAX`, que o teste de configuração exige. No `PUT`, `ativo` é obrigatório (400 `campo:ativo`), para um campo esquecido não desativar o Vendedor.

Não verificado: o passeio manual no navegador (`docker compose up`) não foi feito.

## Suggested Review Order

**O predicado de visibilidade**

- A VIEW é o único lugar que diz "Vendedor ativo"; a 3.4 acrescenta "Produto ativo".
  [`20260918120000_catalogo_produto_visivel.sql:10`](../../db/migracoes/20260918120000_catalogo_produto_visivel.sql#L10)

- A trava é `OF p`: trava o Produto, nunca o Vendedor, e a visibilidade entra por subconsulta.
  [`consultas.sql:19`](../../internal/catalogo/db/consultas.sql#L19)

- A Página de Produto lê da VIEW: Vendedor inativo dá o mesmo 404 do inexistente.
  [`consultas.sql:4`](../../internal/catalogo/db/consultas.sql#L4)

**O CRUD de Vendedor**

- O banco decide o duplicado (23505) e a remoção recusada (23503), sem SELECT antes.
  [`vendedor.go:84`](../../internal/catalogo/vendedor.go#L84)

- Apara e valida o nome em runas; no `PUT`, `ativo` é obrigatório.
  [`vendedor.go:100`](../../api/vendedor.go#L100)

- As rotas moram no mux `admin` e herdam a guarda sem checagem própria.
  [`rotas.go:77`](../../api/rotas.go#L77)

**A área administrativa no web/**

- Nada renderiza sem uma Sessão de Administrador; qualquer outra resposta vai para `/admin/entrar`.
  [`casca-admin.tsx:24`](../../web/components/casca-admin.tsx#L24)

- O patch da revisão: editar um Vendedor que sumiu não cria outro.
  [`vendedores.tsx:113`](../../web/app/admin/vendedores/vendedores.tsx#L113)

- A lista vazia com a ação única de criar.
  [`vendedores.tsx:215`](../../web/app/admin/vendedores/vendedores.tsx#L215)

- O login do Administrador, fora da casca da loja.
  [`page.tsx:27`](../../web/app/admin/entrar/page.tsx#L27)

**Periféricos**

- A desativação e o Pedido antigo, que continua lido idêntico.
  [`vendedor_test.go:120`](../../api/vendedor_test.go#L120)

- A contagem de tabelas ignora a VIEW.
  [`schema_test.go:48`](../../db/schema_test.go#L48)

- O teto do nome vem da Config.
  [`config.go:50`](../../internal/plataforma/config.go#L50)
