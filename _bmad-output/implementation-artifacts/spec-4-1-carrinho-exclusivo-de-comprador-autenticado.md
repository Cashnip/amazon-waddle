---
title: 'Estória 4.1 — Carrinho exclusivo de Comprador autenticado'
type: 'feature'
created: '2026-09-18'
status: 'done'
baseline_commit: '5624af102eed928f54732685c17db5da92a9b6d3'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-4-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** o schema `carrinho` não existe. "Adicionar ao Carrinho" leva o visitante ao Login com `?quantidade=N`, mas a volta não cria nada, e o botão fica desabilitado para quem já tem Sessão. A 3.6 deixou essa volta aberta para a 4.x.

**Approach:**
- O schema `carrinho` ganha `carrinho` e `item_carrinho`, e `internal/carrinho` ganha `Adicionar` e `RemoverItem`, com a posse dentro da consulta.
- `api/` ganha `POST /api/v1/carrinho/itens` e `DELETE /api/v1/carrinho/itens/{id}`.
- A Caixa de compra adiciona direto quando há Sessão. Sem Sessão, ela manda ao Login com um marcador `adicionar`, e a volta cria o Item.
- `negacaoPorDono` ganha o Item de Carrinho como recurso.

## Boundaries & Constraints

**Always:**
- Um Carrinho por Comprador (`UNIQUE comprador_id`), no PostgreSQL. Não expira e não tem FK cruzando schema (AD-2).
- `carrinho` não importa `identidade`: o `comprador_id` vem da Sessão, por `api/` (AD-11).
- Visibilidade e preço vêm só de `catalogo.BuscarProduto`, que lê a VIEW do AD-19.
- `preco_visto_centavos` é gravado a cada adição, com o preço atual.
- A quantidade vai de 1 até `cfg.CarrinhoUnidadesMax`, que já existe. Isso vale para a entrada e para a soma.
- Item alheio, Item inexistente e uuid malformado saem no mesmo 404, byte a byte.
- O Produto repetido soma ao Item existente (`UNIQUE (carrinho_id, produto_id)`).

**Ask First:** qualquer coluna, rota ou sentinela além dos listados aqui.

**Never:**
- Carrinho anônimo, Carrinho no Redis, fusão no login.
- `GET` do Carrinho, a tela do Carrinho, o contador na barra superior (4.3, 4.5).
- A recusa por Estoque informando o disponível (4.2).
- Alterar a quantidade e esvaziar (4.3).
- `Esvaziar` e `ConfirmarPrecoVisto` (Épica 5).
- Chamar `Reservar`.
- Regra de negócio no Node.
- Mexer no `Comprar` do esqueleto.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Primeira adição | Comprador sem Carrinho, Produto visível, q=3 | 201 `{id, produto_id, quantidade: 3}`; Carrinho e Item criados | — |
| Repetida | mesmo Produto, q=2 | 201, mesmo `id`, `quantidade: 5`, uma linha só | — |
| Soma passa do teto | Item com 9, q=2 | nada grava | 400 `EscreverCampo("quantidade", …)` nomeando o teto |
| Quantidade fora | q=0, q=11, q ausente ou não inteira | nada grava | 400 no campo `quantidade` |
| Produto invisível | inexistente, desativado, Vendedor desativado ou uuid ruim | nada grava | 404 `NAO_ENCONTRADO` |
| Sem Sessão | sem cookie | nada grava | 401 `SESSAO_INVALIDA` |
| Remover o próprio | `DELETE` do Item do dono | 204, a linha some | — |
| Remover alheio | Item de outro Comprador, ou `uuidNuncaUsado` | a linha continua lá | o mesmo 404 do inexistente |
| Volta do Login | `/produtos/{id}?quantidade=3&adicionar=1` com Sessão | cria o Item uma vez, `router.replace` para `/produtos/{id}`, "Adicionado ao Carrinho." em linha | recusa: a mensagem do envelope em linha, `role="alert"` |

</frozen-after-approval>

## Code Map

- `db/migracoes/20260918160000_carrinho_schema.sql` -- NOVA. Molde: `20260915120000_identidade_endereco.sql`. `goose Up/Down`, `uuidv7()`.
- `sqlc.yaml` -- acrescentar o bloco `carrinho` (`*_carrinho_*.sql` → `internal/carrinho/db/gerado`), no molde do bloco `pedido`.
- `internal/carrinho/carrinho.go` -- hoje só o doc do pacote. Vira a porta pública. Molde: `internal/identidade/endereco.go`, com `uuidDe`/`uuidTexto`, o `ErrNoRows` como inexistente e o `:execrows` com 0 virando `ErrNoRows`.
- `internal/catalogo/catalogo.go:65` `BuscarProduto` -- lê a VIEW e devolve `pgx.ErrNoRows` para invisível e malformado. Recebe o mesmo pool (as duas `DBTX` têm o mesmo conjunto de métodos). A aresta `carrinho → catalogo` já está em `internal/fronteira_test.go`.
- `api/endereco.go` -- molde dos handlers: `semCache`, `compradorDaRequisicao`, `decodificarCorpo`, `ErrNoRows → erro.ErrNaoEncontrado`, `EscreverCampo`.
- `api/rotas.go:65` -- registrar as duas rotas no mux raiz, depois dos Endereços.
- `api/autorizacao_test.go:40` `negacaoPorDono` -- acrescentar a linha "Item de Carrinho por DELETE" à tabela. O Item do dono é criado por POST antes. No fim, confirmar que ele sobreviveu, com uma segunda adição devolvendo o mesmo `id` somado.
- `api/sessao_test.go:174` `TestSessaoEProduto` -- onde os subtestes entram (`ambiente(t)` compartilhado). O Produto semeado vem dos helpers de `produto_test.go` e `pedido_test.go`.
- `api/endereco_test.go:380-414` -- `idDe`, `postarEndereco`, molde dos helpers novos.
- `web/app/produtos/[id]/caixa-de-compra.tsx` -- `adicionar()` hoje só manda ao Login. A mensagem "ainda não está disponível" sai.
- `web/app/produtos/[id]/page.tsx:88` -- passar `adicionarAoEntrar` a partir do `searchParams.adicionar`.
- `web/lib/destino.ts` `paraLogin` -- reusar como está.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/20260918160000_carrinho_schema.sql` -- `CREATE SCHEMA carrinho`, com as duas tabelas abaixo -- a posse do AD-2, com FK só interna:
  - `carrinho.carrinho(id, comprador_id uuid NOT NULL UNIQUE)`;
  - `carrinho.item_carrinho(id, carrinho_id FK ON DELETE CASCADE, produto_id uuid NOT NULL, quantidade int CHECK > 0, preco_visto_centavos bigint CHECK >= 0, UNIQUE(carrinho_id, produto_id))`.
- [x] `sqlc.yaml` + `internal/carrinho/db/consultas.sql` + `sqlc generate` -- três consultas -- a soma e o teto numa ida ao banco:
  - `GarantirCarrinho` (upsert por `comprador_id`, `RETURNING id`);
  - `AdicionarItem` (`ON CONFLICT … DO UPDATE SET quantidade = quantidade + EXCLUDED.quantidade, preco_visto_centavos = EXCLUDED… WHERE soma <= @teto`, `RETURNING`);
  - `RemoverItem` (`DELETE … USING carrinho.carrinho WHERE item.id = @id AND comprador_id = @comprador_id`, `:execrows`).
- [x] `internal/carrinho/carrinho.go` -- `type Item{ID, ProdutoID string; Quantidade int32}`, `ErrTetoPorItem`, e duas funções -- a porta do AD-1:
  - `Adicionar(ctx, bd, compradorID, produtoID string, q, teto int) (Item, error)`: `catalogo.BuscarProduto` → garantir o Carrinho → `AdicionarItem`. Zero linhas do upsert viram `ErrTetoPorItem`;
  - `RemoverItem(ctx, bd, itemID, compradorID string) error`.
- [x] `api/carrinho.go` -- NOVO. `adicionarAoCarrinho` e `removerItemDoCarrinho`, com a validação `1 ≤ q ≤ cfg.CarrinhoUnidadesMax` antes do domínio. `ErrTetoPorItem` vira `EscreverCampo("quantidade", …)` com o teto, e `ErrNoRows` vira `ErrNaoEncontrado` -- tradução só em `api/`.
- [x] `api/rotas.go` -- registrar as duas rotas.
- [x] `api/carrinho_test.go` -- NOVO, com subteste em `TestSessaoEProduto` que cobre as linhas de API da matriz -- NFR-8.
- [x] `api/autorizacao_test.go` -- o Item de Carrinho em `negacaoPorDono` -- NFR-6.
- [x] `web/app/produtos/[id]/caixa-de-compra.tsx` + `page.tsx` -- a Caixa de compra adiciona e trata a volta do Login -- fecha a volta que a 3.6 abriu:
  - com Sessão, `POST` e "Adicionado ao Carrinho." em linha;
  - sem Sessão, ou com 401 no `POST`, `paraLogin('/produtos/{id}?quantidade=N&adicionar=1')`;
  - com `adicionarAoEntrar` e Sessão, `POST` uma vez só (guarda por `useRef`, contra o efeito duplo do StrictMode), depois `router.replace` para `/produtos/{id}`;
  - o botão fica desabilitado enquanto envia.

**Acceptance Criteria:**
- Dado um visitante sem Sessão na Página de Produto, quando escolhe 3 e clica "Adicionar ao Carrinho" e entra, então volta a `/produtos/{id}` sem `adicionar` na URL, vê "Adicionado ao Carrinho.", e `carrinho.item_carrinho` tem uma linha com quantidade 3.
- Dada a volta já feita, quando a página é recarregada, então nenhuma unidade é somada.
- Dado o stack reiniciado (`docker compose restart`), quando o banco é consultado, então o Item continua lá.
- Dado o código, quando inspecionado, então não há estado de Carrinho no Redis nem no navegador, e `internal/carrinho` não importa `identidade`.

## Design Notes

**Por que o NFR-6 ataca o Item, e não o Carrinho.** O Carrinho é resolvido pela Sessão e não tem identificador na URL. O identificador que um Comprador pode forjar é o do Item. O `DELETE` veio da 4.3 só para isso, por decisão humana de 2026-09-18.

**Onde termina a 4.1.** A soma, o teto e a visibilidade entram aqui porque a migração obriga a decidir o que fazer com Produto repetido, e porque o teto é validação na fronteira de confiança (NFR-14). A 4.2 fica com a recusa por Estoque que informa o disponível e com o restante dos testes da FR-17.

## Verification

**Commands:**
- `sqlc generate && go vet ./... && go test ./...` -- expected: tudo verde, inclusive `TestFronteiraDeModulo`.
- `cd web && npm run build` -- expected: sem erro.

**Manual checks:**
- O passeio da primeira condição de aceite a 360 e 1440 px. Depois, `docker compose exec postgres psql -c 'select * from carrinho.item_carrinho'`.

## Review Triage Log

Uma lente só (Edge Case Hunter), pela política de revisão do projeto para CRUD com posse. Cinco achados:
- **patch:** a volta do Login com o Produto esgotado criava o Item em silêncio. Agora não cria nada e só limpa o marcador.
- **reject:**
  - navegar para fora com o POST em voo: a guarda proposta quebraria o `replace` sob o StrictMode;
  - duplo clique: o evento discreto aplica o `enviando` antes do segundo clique;
  - o marcador fica na URL quando não há Sessão: inofensivo;
  - teto acima de `MaxInt32`: configuração absurda.

## Suggested Review Order

**Posse e soma no banco**

- A soma, o teto e o preço visto numa ida só; zero linhas é o teto.
  [`consultas.sql:12`](../../internal/carrinho/db/consultas.sql#L12)

- O dono no `WHERE` por `USING`: alheio e inexistente afetam as mesmas zero linhas.
  [`consultas.sql:23`](../../internal/carrinho/db/consultas.sql#L23)

- Um Carrinho por Comprador pelo `UNIQUE`; nenhuma FK cruza schema.
  [`20260918160000_carrinho_schema.sql:15`](../../db/migracoes/20260918160000_carrinho_schema.sql#L15)

**Porta do módulo**

- A visibilidade vem de `catalogo.BuscarProduto`; `carrinho` não conhece `identidade`.
  [`carrinho.go:41`](../../internal/carrinho/carrinho.go#L41)

- `:execrows` com 0 vira `ErrNoRows`, no molde de `RemoverEndereco`.
  [`carrinho.go:77`](../../internal/carrinho/carrinho.go#L77)

**Fronteira HTTP**

- A quantidade chega crua, para o 400 nomear o campo; o teto vem da Config.
  [`carrinho.go:47`](../../api/carrinho.go#L47)

- As duas rotas no mux raiz; o Carrinho não tem id na URL, o Item tem.
  [`rotas.go:72`](../../api/rotas.go#L72)

**A volta do Login**

- Cria uma vez (`useRef`), pula o esgotado e limpa o marcador com `replace`.
  [`caixa-de-compra.tsx:87`](../../web/app/produtos/[id]/caixa-de-compra.tsx#L87)

- Com Sessão, adiciona direto; com 401, vai ao Login com o marcador.
  [`caixa-de-compra.tsx:63`](../../web/app/produtos/[id]/caixa-de-compra.tsx#L63)

- O Server Component lê o `adicionar` da URL.
  [`page.tsx:95`](../../web/app/produtos/[id]/page.tsx#L95)

**Testes**

- O NFR-6 ganha o Item de Carrinho, provado por igualdade byte a byte.
  [`autorizacao_test.go:67`](../../api/autorizacao_test.go#L67)

- As linhas de API da matriz: soma, teto, entradas ruins, invisível, remoção.
  [`carrinho_test.go:18`](../../api/carrinho_test.go#L18)

- O `&adicionar=1` sobrevive à ida e volta pelo `destino`.
  [`destino.test.mjs:72`](../../web/scripts/destino.test.mjs#L72)
