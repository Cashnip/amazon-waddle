---
title: 'Estória 6.5 — Pagamento aprovado sobre Pedido cancelado, visível ao Administrador'
type: 'feature'
created: '2026-09-23'
status: 'done'
baseline_commit: '8af60899283ac31a053d90efef4a119340ffe12a'
review_loop_iteration: 0
context: ['{project-root}/_bmad-output/implementation-artifacts/epic-6-context.md']
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Desde a 5.9 a aprovação que chega sobre Pedido `CANCELADO` fica gravada na inbox (`APROVADO` em `NAO_APLICAVEL_SINALIZADA`), e desde a 6.3 a corrida acontece pela rota — mas ninguém lê esse registro: o único rastro visível é um `WarnContext` no log. É a perda silenciosa que a FR-26 e a invariante 7 do addendum §2 proíbem.

**Approach:** Derivar `pagamento_aprovado_sobre_cancelado` na leitura (AD-18): uma porta de `pagamento` diz quais Pedidos têm aprovação sinalizada, e `pedido` a junta com `Status == CANCELADO` em Go. As duas respostas administrativas da 6.4 carregam o campo; o Detalhe mostra um `Alert` persistente e a linha da Tabela ganha um marcador. Sem migração, sem coluna, sem Status novo.

## Boundaries & Constraints

**Always:**
- O predicado é exatamente: Pedido em `CANCELADO` **e** existe confirmação `resultado = 'APROVADO'` com `estado = 'NAO_APLICAVEL_SINALIZADA'` numa Tentativa dele. Tentativa superada conta; Pedido cancelado depois de `PAGO` com uma segunda aprovação sinalizada conta (cobrança duplicada).
- A metade de `pagamento` é lida por porta (aresta `pedido → pagamento` já existe); a junção com o Status é em Go. Nenhum SQL cruza schema (AD-1, AD-2).
- A Tabela lê a inbox numa consulta só para a página inteira, e só para os Pedidos `CANCELADO` dela — nunca uma por linha.
- O campo existe sempre nas duas respostas administrativas, `false` quando não se aplica — nunca ausente.
- O marcador e o `Alert` são texto, não cor: nem verde nem laranja (DESIGN). O `Alert` não tem botão de fechar.

**Ask First:**
- Migração, coluna nova, ou estender o sinal a Pedido que não está `CANCELADO`.

**Never:**
- Mudar o Status do Pedido, o histórico ou a Reserva; tocar `aplicarConfirmacao` ou o `WarnContext`.
- Levar o campo à resposta do Comprador ou a qualquer tela dele; texto que prometa estorno ou reembolso.
- Cobrir aqui a Tentativa superada aprovada com Pedido ativo, ou a aprovação sobre Pedido expirado (`PAGAMENTO_RECUSADO`): inalcançáveis com o Provedor Simulado, ficam no `deferred-work.md` com a decisão registrada.
- Selo `Badge` por Status (6.7) ou filtro novo na Tabela.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Corrida clássica | cancelado pela rota em `AGUARDANDO_PAGAMENTO`, depois a aprovação emitida e varrida | Detalhe e linha da Tabela com `true`; Status `CANCELADO`, histórico com 2 linhas | N/A |
| Cancelado sem aprovação sinalizada | `PAGO` (aprovação `APLICADA`) cancelado pelo Comprador | `false` | N/A |
| Recusa sinalizada | total `,90`, cancelado antes de a recusa ser aplicada | `false` — `RECUSADO` não é dinheiro aprovado | N/A |
| Sinalizada sem cancelamento | `PAGO` com segunda Tentativa aprovada e sinalizada | `false` | N/A |
| Cobrança duplicada | o Pedido anterior, cancelado pela rota | `true` | N/A |
| Não cancelado | qualquer outro Status | `false`, e a Tabela não consulta a inbox para ele | N/A |

</frozen-after-approval>

## Code Map

- `internal/pagamento/db/consultas.sql:69` -- `TemConfirmacaoPendente`: molde da consulta nova (mesma junção inbox ↔ Tentativa por `tentativa_id`). Índices em `tentativa_pagamento(pedido_id)` e `confirmacao_recebida(tentativa_id)` já existem.
- `internal/pagamento/pagamento.go:191,230` -- `TentativasRestantes`/`TemConfirmacaoPendente`: molde da porta (uuid por `Scan`, erro embrulhado).
- `internal/pedido/pedido.go:681` -- `ListarParaAdministrador`: devolve `[]Pedido`; passa a devolver a linha com o sinal.
- `internal/pedido/pedido.go:729,748` -- `DetalheAdmin` e `DetalharParaAdministrador`: roda na transação `REPEATABLE READ` de `api/`, então a porta vê o mesmo instante.
- `internal/pedido/pedido.go:823` -- `TransicionarPeloAdministrador`: devolve `Pedido`; não muda.
- `api/pedido_admin.go:34,88,96,167,268` -- `saidaPedidoAdmin`, `saidaAdminDoPedido`, `saidaAdminDoDetalhe`, `listarPedidosAdmin` e o 200 da transição.
- `internal/fronteira_test.go:32` -- `pedido → pagamento` já permitida; `pagamento` segue sem arestas.
- `api/pedido_admin_test.go:47` -- `painelDePedidosDoAdministrador`: molde de conta, Produto, `emitirPara`, `pago` e `listar`; compara com o total global, então o teste novo registra-se **depois** dele.
- `api/cancelamento_test.go:491` -- a corrida pela rota que produz o sinal; `api/webhook_test.go:498,508,548` -- `confirmar`, `criarTentativa`, `confirmacoesDe`.
- `api/sessao_test.go:563` -- onde as matrizes de servidor são registradas.
- `web/lib/pedido.ts:327,345` -- `PedidoNaTabela` e `DetalheAdmin`.
- `web/app/admin/pedidos/pedidos.tsx:246` -- a célula de Status da linha.
- `web/app/admin/pedidos/[id]/detalhe-admin.tsx:126` -- a região do Status no Card do topo.
- `web/components/ui/alert.tsx:75`, `badge.tsx` -- `AlertTitle`/`AlertDescription` e `Badge` `outline`, sem alteração.

## Tasks & Acceptance

**Execution:**
- [x] `internal/pagamento/db/consultas.sql` + `pagamento.go` -- consulta `PedidosComAprovacaoSinalizada` (`DISTINCT t.pedido_id` com `t.pedido_id = ANY(@pedido_ids::uuid[])`, `APROVADO`, `NAO_APLICAVEL_SINALIZADA`) e a porta `PedidosComAprovacaoSinalizada(ctx, bd, pedidoIDs []string) (map[string]bool, error)`, lista vazia sem ir ao banco. `sqlc generate` na imagem `sqlc/sqlc:1.31.1`, restaurando os `gerado/` dos outros módulos.
- [x] `internal/pedido/pedido.go` -- `PedidoAdmin{Pedido; PagamentoAprovadoSobreCancelado bool}`; `ListarParaAdministrador` devolve `[]PedidoAdmin`, chamando a porta só com os ids `CANCELADO` da página; `DetalheAdmin` embute `PedidoAdmin` e `DetalharParaAdministrador` chama a porta só se `CANCELADO`.
- [x] `api/pedido_admin.go` -- `pagamento_aprovado_sobre_cancelado` em `saidaPedidoAdmin`, sem `omitempty`; o 200 da transição sai `false` (o Administrador não tem linha para `CANCELADO`), com comentário.
- [x] `api/aprovado_sobre_cancelado_test.go` + `api/sessao_test.go` -- a matriz inteira pela rota, com conta e Produtos próprios, nas duas respostas (Detalhe e linha da Tabela filtrada por `CANCELADO`), conferindo também Status e histórico intactos e a chave ausente na resposta do Comprador.
- [x] `web/lib/pedido.ts` -- o campo nos dois tipos e os textos do marcador e do `Alert` como constantes.
- [x] `web/app/admin/pedidos/pedidos.tsx` -- `Badge` `outline` "Pagamento aprovado" ao lado do Status, na linha com o sinal.
- [x] `web/app/admin/pedidos/[id]/detalhe-admin.tsx` -- `Alert` persistente no Card do topo, abaixo do Status.
- [x] `addendum.md` §10 + `.memlog.md` (por `uv run _bmad/scripts/memlog.py append`) -- a derivação na leitura e o escopo `CANCELADO`; `deferred-work.md` -- fechar a entrada da 5.9/6.3 sobre o sinal sem leitor e registrar a decisão da entrada da Tentativa superada.

**Acceptance Criteria:**
- Dado um Pedido com o sinal, quando o Administrador abre o Detalhe, então vê o `Alert` com título "Pagamento aprovado sobre Pedido cancelado" e o texto "O Provedor de Pagamento aprovou uma Tentativa de Pagamento deste Pedido cancelado. O Status do Pedido não muda, e a aprovação fica registrada na Tentativa de Pagamento.", sem ação de fechar.
- Dado a Tabela filtrada por `CANCELADO`, quando há Pedidos com e sem o sinal, então só as linhas com o sinal mostram "Pagamento aprovado", e o leitor de tela lê o marcador junto do Status.

## Spec Change Log

- **Revisão, passe 1 (edge-case-hunter) — o texto do `Alert` afirmava uma ordem que o predicado não garante.** O Critério de Aceite fixava "…aprovou uma Tentativa de Pagamento deste Pedido depois do cancelamento.", mas o próprio Always manda acender na cobrança duplicada (segunda aprovação sinalizada em `PAGO`, depois cancelado), e o sinal acende também na aprovação sobre Pedido expirado que o Comprador depois cancela: nos dois a aprovação veio **antes**. Emenda, por decisão do usuário (opção 1, sem reverter o código): o texto passa a "O Provedor de Pagamento aprovou uma Tentativa de Pagamento deste Pedido cancelado. O Status do Pedido não muda, e a aprovação fica registrada na Tentativa de Pagamento." — no AC, em `web/lib/pedido.ts`, em `pedido.test.mjs` e no addendum. Estado evitado: o Administrador ler que a aprovação veio depois do cancelamento quando veio antes. KEEP: o predicado (`CANCELADO` e `APROVADO` em `NAO_APLICAVEL_SINALIZADA`, sem filtro por instante), a porta em lote, a matriz e as duas mutações.

## Design Notes

**Por que o sinal não reacende a consulta em intervalo:** `CANCELADO` é terminal e a tela para de consultar. A aprovação que chega depois aparece na próxima abertura ou recarga — "persistente" é o registro não sumir, e não chegar ao vivo. Fazer a Tabela consultar enquanto listar `CANCELADO` a faria consultar para sempre.

**Por que um mapa em lote, e não `bool` por Pedido:** a Tabela tem até 20 linhas; uma porta por linha seria N+1 contra a inbox. O Detalhe usa a mesma porta com um id.

Quem implementa não faz `git commit`, `git add` nem `push`.

## Verification

**Commands:**
- `go vet ./... && go test -count=1 ./...` na imagem `azamon-dev` (HANDOFF) -- verde.
- `docker build ./web` -- verde (`prebuild` roda `verificar-offline.mjs` e `npm test`).
- Mutações, uma de cada vez, revertidas: tirar `resultado = 'APROVADO'` (a recusa sinalizada vira `true`); tirar a condição `CANCELADO` (o `PAGO` sinalizado vira `true`) -- o teste morre em cada uma.

**Manual checks (if no CLI):**
- A 1440 px, só pelo teclado: Produto de `,00`, cancelar em `AGUARDANDO_PAGAMENTO`, esperar ~5 s, abrir `/admin/pedidos?status=CANCELADO` e o Detalhe.

## Review Triage Log

Passe 1 — uma lente (`edge-case-hunter`), pela regra de custo do HANDOFF: o diff é uma porta de leitura, um campo derivado, um `Alert` e um marcador.

| # | Achado | Veredito | Rota |
|---|--------|----------|------|
| 1 | O texto do `Alert` diz "depois do cancelamento", mas o sinal acende também quando a aprovação sinalizada veio antes (cobrança duplicada; Pedido expirado depois cancelado) | low (inalcançável com o Provedor Simulado), texto fixado no AC | emenda de spec por decisão do usuário — ver Spec Change Log |
| 2 | A entrada do `deferred-work.md` dizia que o sinal "não cobre" a Tentativa superada e o Pedido expirado; ele acende nos dois depois que o Comprador cancela | low | patch — a entrada diz quando acende e por quê |
| 3 | Subteste anterior que falha deixa id vazio; as chaves vazias colidem no mapa da Tabela e a contagem acusa erro em cascata | low | patch — `t.Skip` com os ids vazios |
| 4 | Asserção de tipo sem `ok` nas linhas da Tabela derrubaria `TestSessaoEProduto` inteiro numa regressão | low | patch — `t.Fatalf` com a linha |

## Suggested Review Order

**A metade de `pagamento`: a porta em lote**

- A consulta não conhece Status: só `APROVADO` em `NAO_APLICAVEL_SINALIZADA`, para a página inteira.
  [`consultas.sql:88`](../../internal/pagamento/db/consultas.sql#L88)

- Lista vazia não vai ao banco; o ausente do mapa lê `false`.
  [`pagamento.go:253`](../../internal/pagamento/pagamento.go#L253)

**A junção em Go, em `pedido`**

- Entrada: pergunta à inbox só pelos `CANCELADO` — a outra metade do predicado.
  [`pedido.go:693`](../../internal/pedido/pedido.go#L693)

- O tipo das leituras administrativas: o Pedido e o sinal, sem tocar o `Pedido` do Comprador.
  [`pedido.go:683`](../../internal/pedido/pedido.go#L683)

- A Tabela: uma ida à inbox por página, nunca por linha.
  [`pedido.go:756`](../../internal/pedido/pedido.go#L756)

- O Detalhe: a mesma junção, na transação `REPEATABLE READ` da rota.
  [`pedido.go:817`](../../internal/pedido/pedido.go#L817)

**O contrato HTTP**

- Sem `omitempty`: sempre presente nas duas respostas administrativas, nunca na do Comprador.
  [`pedido_admin.go:44`](../../api/pedido_admin.go#L44)

- O 200 da transição leva `false` sem ler: o Administrador nunca chega a `CANCELADO`.
  [`pedido_admin.go:281`](../../api/pedido_admin.go#L281)

**As duas superfícies**

- `Alert` neutro, sem fechar, abaixo do Status que ele não muda.
  [`detalhe-admin.tsx:135`](../../web/app/admin/pedidos/[id]/detalhe-admin.tsx#L135)

- Marcador `outline` na mesma célula do Status, lido junto pelo leitor de tela.
  [`pedidos.tsx:254`](../../web/app/admin/pedidos/pedidos.tsx#L254)

- Os textos, sem timing e sem promessa de estorno.
  [`pedido.ts:348`](../../web/lib/pedido.ts#L348)

**Testes**

- A matriz pela rota, com conta e Produtos próprios.
  [`aprovado_sobre_cancelado_test.go:37`](../../api/aprovado_sobre_cancelado_test.go#L37)

- Mata a mutação sem `resultado = 'APROVADO'`.
  [`aprovado_sobre_cancelado_test.go:224`](../../api/aprovado_sobre_cancelado_test.go#L224)

- Cobrança duplicada: o mesmo Pedido apaga em `PAGO` e acende cancelado — mata a mutação sem `CANCELADO`.
  [`aprovado_sobre_cancelado_test.go:256`](../../api/aprovado_sobre_cancelado_test.go#L256)

- Registrada depois do painel da 6.4, que confere o total global.
  [`sessao_test.go:578`](../../api/sessao_test.go#L578)

- Os três textos presos, e nenhum promete estorno.
  [`pedido.test.mjs:487`](../../web/scripts/pedido.test.mjs#L487)
