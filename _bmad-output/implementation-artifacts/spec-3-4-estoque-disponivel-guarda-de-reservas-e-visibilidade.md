---
title: '3.4 — Estoque disponível, guarda de Reservas e o predicado de visibilidade'
type: 'feature'
created: '2026-09-18'
status: 'done'
baseline_commit: 'eba2c0ac76d7271769410b6a6f7db6a594e556a6'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-3-context.md'
  - '{project-root}/_bmad-output/implementation-artifacts/spec-3-2-3-3-gestao-de-categorias-e-produtos.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O Estoque total só é informado na criação do Produto, e o `PUT` o ignora. `Disponivel`, `Visiveis` e `Liberar` não existem. O `Reservar` recebe um Produto só e, quando o Produto está invisível, devolve `pgx.ErrNoRows`, e não `ErrEstoqueInsuficiente`. A VIEW expõe `estoque_total`, que o AD-16 proíbe. Carrinho, checkout e cancelamento (Épicas 4 a 6) precisam dessa interface com as assinaturas já definitivas.

**Approach:**
- Completar a interface de Estoque do AD-5/AD-19 em `catalogo`.
- Trocar `estoque_total` por `estoque_disponivel` derivado na VIEW.
- Dar ao Administrador o ajuste do Estoque total numa rota própria, com a guarda das Reservas ativas sob a ordem obrigatória do AD-5. A rota tem uma ação própria na tela.

## Boundaries & Constraints

**Always:**
- A ordem de dois comandos do AD-5: primeiro `SELECT … WHERE id = ANY($1) ORDER BY id FOR UPDATE OF p`, e só depois a soma das reservas ativas. Vale no `Reservar` e no ajuste.
- O predicado de visibilidade continua existindo **só** na VIEW `catalogo.produto_visivel`. `Disponivel` e `Visiveis` leem dela. Nenhum `WHERE ativo` novo.
- Mutações idempotentes, com isso declarado no comentário de cada função:
  - `Reservar` com lista vazia é no-op;
  - `Liberar` sem reserva ativa devolve `nil`;
  - `Consolidar` repetido não faz nada (já é assim).
- `Liberar` muda só o estado da Reserva (`ATIVA → LIBERADA`) e nunca escreve `estoque_total`.
- O ajuste vale para Produto ativo e inativo: é tela do Administrador.
- O teto do ajuste é o mesmo da criação, `AZAMON_PRODUTO_ESTOQUE_MAX`.

**Ask First:** chamar `Liberar` de dentro de `pedido`, porque a expiração é da 5.x e o cancelamento é da 6.3. Mudar o comportamento HTTP de `POST /api/v1/pedidos`.

**Never:**
- Armazenar o disponível.
- Cache.
- Aceitar `estoque_total` no `PUT` de Produto. O `definirAtivo` da tela reenvia a linha lida, e isso regravaria um total velho por cima de uma consolidação.
- Teste de concorrência do NFR-7, que é da 5-7.
- Mostrar o disponível na loja, que é da 3.5 e da 3.6.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Esperado | Erro |
|---|---|---|---|
| Ajuste aceito | `PUT /api/v1/admin/produtos/{id}/estoque` `{estoque_total: 5}`, 4 comprometidas | 200 com a linha administrativa; a VIEW mostra disponível 1 | — |
| Ajuste abaixo das Reservas | `{estoque_total: 2}`, 4 comprometidas | nada muda | 409 `ESTOQUE_COMPROMETIDO`, "Há 4 unidades comprometidas em Pedidos abertos. O Estoque total não pode ficar abaixo disso.", `dados: {campo: "estoque_total", comprometidas: 4}`. Com N = 1, a frase fica no singular: "Há 1 unidade comprometida…" |
| Ajuste fora do teto | `-1`, acima do teto, ausente | — | 400 `campo:estoque_total`, a mesma mensagem da criação |
| Ajuste de inexistente | id inexistente ou malformado | — | 404 |
| `Disponivel` | ids visível, invisível, inexistente e malformado | uma chave por id recebido, com a mesma grafia; invisível, inexistente e malformado valem `0`; sem erro | só erro de banco |
| `Visiveis` | os mesmos ids | `true` só para o visível; toda chave presente | só erro de banco |
| `Reservar` | lista vazia | no-op, `nil` | — |
| `Reservar` | um item invisível ou inexistente | nenhuma Reserva gravada | `EstoqueInsuficiente{ProdutoID, Disponivel: 0}` |
| `Liberar` | com Reserva ativa; depois, repetido | `LIBERADA`, `estoque_total` intacto; a repetição devolve `nil` | — |

</frozen-after-approval>

## Code Map

- `internal/catalogo/catalogo.go:79` -- `Reservar(ctx, tx, produtoID, pedidoID, quantidade)`. Muda para `Reservar(ctx, tx, pedidoID string, itens []ItemReserva)`, com `ItemReserva{ProdutoID string; Quantidade int32}`. A trava sobre todos os ids em ordem; a soma em lote depois; toda a verificação antes de qualquer `INSERT`. `EstoqueInsuficiente` ganha `ProdutoID`. Id de Produto malformado também vira `EstoqueInsuficiente`.
- `internal/pedido/pedido.go:169` -- o único chamador de `Reservar`: vira `[]catalogo.ItemReserva{{produto.ID, unidade}}`. O 404 de Produto invisível continua vindo do `BuscarProduto` da linha 116, então o HTTP não muda.
- `internal/catalogo/db/consultas.sql:22` -- `TravarProdutosParaReserva` fica como está. `SomarReservasAtivas` vira em lote: `produto_id = ANY(@ids) … GROUP BY produto_id`. Novas: `DisponivelDosVisiveis` (`SELECT id, estoque_disponivel FROM catalogo.produto_visivel WHERE id = ANY`), `LiberarReservasDoPedido :exec` e `TravarProdutoParaAjuste` (`FOR UPDATE`, **sem** a VIEW) e `AjustarEstoqueTotal`.
- `db/migracoes/20260918130000_catalogo_produto_ativo.sql` -- a VIEW atual. A migração nova precisa de `DROP VIEW` + `CREATE VIEW`, porque a lista de colunas muda. `estoque_total` sai e entra `estoque_disponivel = greatest(estoque_total − coalesce(Σ ATIVA, 0), 0)`. Nada no Go lê `estoque_total` da VIEW.
- `internal/catalogo/produto.go:108` -- molde de função administrativa, `produtoAdminDe`, `uuidDe`. `AjustarEstoque(ctx, tx pgx.Tx, id string, total int32) (ProdutoAdmin, error)` mora aqui, com `EstoqueComprometido{N}`, que desembrulha em `ErrEstoqueComprometido`.
- `api/pedido.go:45` -- molde de `Begin`/`defer Rollback(WithoutCancel)`/`Commit(WithoutCancel)`, que o novo handler copia.
- `api/produto_admin.go:152` -- a mensagem do teto do Estoque é reaproveitada no ajuste. `api/rotas.go:91` -- a rota nova entra no mux `admin`, e com corpo pequeno usa `decodificarCorpo`.
- `internal/plataforma/erro/erro.go:140` -- `EscreverCategoriaComProdutos` é o molde de `EscreverEstoqueComprometido(ctx, w, n)`. O singular e o plural seguem o mesmo padrão, e o número usa `Milhar`.
- `api/produto_admin_test.go`, `api/pedido_test.go:169` -- molde do subteste, e de como criar Reserva ATIVA direto por SQL. O subteste novo entra em `api/sessao_test.go`, depois da gestão de Categorias. Ele usa Produto próprio e nunca toca Produto semeado.
- `web/app/admin/produtos/produtos.tsx` -- `abrir`/`salvar`/`mostrarErro`/`pedir` e a regex do Estoque (linha 188).
- sqlc: `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/20260918140000_catalogo_estoque_disponivel.sql` -- recriar a VIEW com `estoque_disponivel`. O Down restaura a VIEW da 3.3.
- [x] `internal/catalogo/db/consultas.sql` + gerado -- as consultas do Code Map.
- [x] `internal/catalogo/catalogo.go` -- `Disponivel` e `Visiveis(ctx, bd, ids []string) (map[string]int / map[string]bool, error)`, as duas sobre a mesma consulta. Mais o `Reservar` em lote e o `Liberar(ctx, tx, pedidoID) error`. Atualizar o comentário do pacote.
- [x] `internal/catalogo/produto.go` -- `AjustarEstoque` e `EstoqueComprometido`.
- [x] `internal/pedido/pedido.go` -- a nova chamada do `Reservar`.
- [x] `internal/plataforma/erro/erro.go` -- `EscreverEstoqueComprometido`.
- [x] `api/produto_admin.go` + `api/rotas.go` -- `ajustarEstoque`, numa transação. A tabela de rotas do AD-16 já cobre a rota com `/api/v1/admin/produtos…` (CRUD, Estoque).
- [x] `api/estoque_test.go` + `api/sessao_test.go` -- um subteste que percorre a matriz:
  - Produto próprio, com Reserva ATIVA por SQL e ajustes aceito e recusado;
  - `Disponivel`, `Visiveis`, `Reservar` e `Liberar` chamados direto, dentro de uma transação revertida quando mutam;
  - a VIEW sem a coluna `estoque_total`.
- [x] `web/app/admin/produtos/produtos.tsx` -- a ação "Ajustar Estoque" na linha abre um `Dialog` com um campo só, preenchido com o total atual. O 409 e o 400 aparecem em linha no campo, com foco. O sucesso recarrega a lista.

**Acceptance Criteria:**
- Dado um Produto com 4 unidades em Reservas ativas, quando o Administrador ajusta para 2 na tela, então a mensagem com "4 unidades comprometidas" aparece sob o campo e o Estoque total da lista não muda.
- Dado o ajuste aceito, quando a Página de Produto e a compra acontecem logo em seguida, então valem o novo total sem reiniciar nada.
- Dado o código de `internal/` e as consultas sqlc, quando se procura um filtro por `ativo` em `WHERE`/`AND`, então ele só aparece na definição da VIEW, nas migrações.

## Design Notes

**Rota própria, e não o `PUT`.** Hoje desativar e reativar reenviam a linha inteira lida da lista. Se o total fosse junto, um Administrador com a tela aberta desfaria em silêncio a baixa de uma consolidação. Com a rota própria, cada escrita do total é intencional.

**Por que o ajuste trava sem a VIEW.** Ele vale para Produto inativo. Travar pela VIEW transformaria "inativo" em 404.

## Verification

**Commands:**
- `go build ./... && go vet ./... && go test -count=1 ./...` -- verde, com o subteste novo.
- `cd web && npm run build` -- verde.

**Manual checks:**
- `docker compose up` → `/admin/produtos` → "Ajustar Estoque" do "Produto Manual" → abaixo das Reservas (criadas por um Pedido em `AGUARDANDO_PAGAMENTO`) → recusa em linha → valor válido → a lista atualiza.

## Review Triage Log

Uma lente só (Edge Case Hunter), por escolha do usuário: a ordem do AD-5 no `Reservar` e no `AjustarEstoque` foi lida à mão, e a concorrência de verdade é testada na 5-7. Foram 3 achados:

- **patch** — `catalogo.go`: `ItemReserva` com `Quantidade <= 0` caía no CHECK do banco como 500, e somada a outra linha do mesmo Produto abatia a Reserva. Agora é erro de quem chama, antes da trava.
- **patch** — `produtos.tsx`: fechar o `Dialog` do ajuste com a requisição em voo perdia a resposta. Agora ele não fecha enquanto `enviando`.
- **reject** — o Produto desativado entre o `BuscarProduto` e o `Reservar` sai como 409 com disponível 0, e não como 404. É o que o AD-19 manda: reservar Produto invisível falha com `ErrEstoqueInsuficiente`.

Fora da spec, feito pela implementação:
- o mesmo Produto repetido na lista do `Reservar` soma as quantidades numa Reserva só, porque o índice único parcial recusaria o segundo `INSERT`;
- `mensagemDoEstoque` e `estoqueDe`, extraídos para servir à criação e ao ajuste.

Não verificado: o passeio manual no navegador (a recusa em linha e o recarregar da lista em `/admin/produtos`).

## Suggested Review Order

**A ordem do AD-5**

- Entrada: a trava em ordem de id sobre a fatia inteira, antes de qualquer soma.
  [`catalogo.go:214`](../../internal/catalogo/catalogo.go#L214)

- Só depois a soma em lote; toda a verificação antes do primeiro INSERT.
  [`catalogo.go:224`](../../internal/catalogo/catalogo.go#L224)

- O ajuste do Administrador segue a mesma ordem, travando sem a VIEW para aceitar inativo.
  [`produto.go:172`](../../internal/catalogo/produto.go#L172)

**A interface de Estoque**

- `Reservar` em lote, com a assinatura definitiva e o no-op da lista vazia.
  [`catalogo.go:171`](../../internal/catalogo/catalogo.go#L171)

- Patch da revisão: quantidade não positiva é defeito de quem chama.
  [`catalogo.go:192`](../../internal/catalogo/catalogo.go#L192)

- `Liberar` só troca o estado; nunca escreve `estoque_total`.
  [`catalogo.go:262`](../../internal/catalogo/catalogo.go#L262)

- `Disponivel` e `Visiveis` sobre a mesma consulta, com toda chave presente.
  [`catalogo.go:92`](../../internal/catalogo/catalogo.go#L92)

- O único chamador adaptado; o HTTP da compra não muda.
  [`pedido.go:169`](../../internal/pedido/pedido.go#L169)

**Disponível derivado e visibilidade**

- A VIEW troca `estoque_total` por `estoque_disponivel`, nunca negativo.
  [`20260918140000_catalogo_estoque_disponivel.sql:16`](../../db/migracoes/20260918140000_catalogo_estoque_disponivel.sql#L16)

- O predicado do AD-19 continua só aqui.
  [`20260918140000_catalogo_estoque_disponivel.sql:23`](../../db/migracoes/20260918140000_catalogo_estoque_disponivel.sql#L23)

**HTTP e tela**

- A rota própria do ajuste, numa transação, no mux `admin`.
  [`produto_admin.go:118`](../../api/produto_admin.go#L118)

- O 409 com singular e plural, e a contagem em `dados`.
  [`erro.go:151`](../../internal/plataforma/erro/erro.go#L151)

- O `Dialog` do ajuste: erro em linha no campo, sucesso recarrega a lista.
  [`produtos.tsx:260`](../../web/app/admin/produtos/produtos.tsx#L260)

**Periféricos**

- O subteste que percorre a matriz inteira com Produtos próprios.
  [`estoque_test.go:20`](../../api/estoque_test.go#L20)

- Registrado depois da gestão de Categorias.
  [`sessao_test.go:475`](../../api/sessao_test.go#L475)
