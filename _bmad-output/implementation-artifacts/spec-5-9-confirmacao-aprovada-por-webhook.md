---
title: 'Estória 5.9 — Confirmação aprovada por webhook'
type: 'feature'
created: '2026-09-22'
status: 'done'
baseline_commit: '8d671545389b4e910b86fdf6495194c2a2f2eeb2'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** o caminho aprovado do AD-7 já está de pé desde a 1.7 (inbox com chave única, `pagamento` sem conhecer `pedido`, varredura aplicando, `PAGO` com a Reserva mantida) e a tela que vira sozinha, desde a 5.8. Falta a borda que a FR-26 nomeia e que a invariante 7 do addendum §2 proíbe perder: **aprovação que chega para um Pedido já `CANCELADO`**. Hoje ela cai no balde genérico `NAO_APLICAVEL_SINALIZADA`, sem caminho nomeado, sem uma linha de log e sem um único teste — e ninguém verificou que o sinal que a 6.5 promete ler é alcançável sem arqueologia.

**Approach:** nomear o caso dentro de `aplicarConfirmacao` — mesmo estado terminal, mais um aviso no log —, e provar a corrida inteira pelo caminho de verdade: webhook real → varredura → Status intacto, Reserva ainda `LIBERADA`, confirmação `NAO_APLICAVEL_SINALIZADA`, e o predicado que a 6.5 vai ler devolvendo este Pedido. Sem coluna nova: o sinal é o estado que a própria AC da 6.5 nomeia.

## Boundaries & Constraints

**Always:**
- `pagamento` não conhece `pedido` (AD-7). A emissão derivada emite para a Tentativa de um Pedido cancelado como para qualquer outra; quem decide não aplicar é a varredura, com a trava do Pedido na mão.
- Toda confirmação chega a estado terminal. Aprovada sobre Pedido cancelado: `NAO_APLICAVEL_SINALIZADA`, e **esse** é o registro que a 6.5 lê.
- O sinal da 6.5 é **derivado na leitura**, sem coluna nem estado novo: Pedido em `CANCELADO` **e** confirmação `APROVADO` em `NAO_APLICAVEL_SINALIZADA` numa Tentativa dele.
- A aprovação sobre Pedido cancelado não muda Status, não escreve transição e não reserva nada: a Reserva liberada pelo cancelamento **continua** `LIBERADA`.
- Aviso no log (`slog.WarnContext`) só neste caso, com Pedido e confirmação. Tentativa superada e Status já avançado continuam como estão.
- Glossário literal (Pedido, Tentativa de Pagamento, Reserva de Estoque) no que for texto.

**Ask First:**
- Migração nova, estado novo na inbox, ou qualquer mudança no contrato de `GET /api/v1/pedidos/{id}`.

**Never:**
- Superfície do Administrador: `Alert`, marcador na Tabela e o campo `pagamento_aprovado_sobre_cancelado` na resposta administrativa são a 6.5. Aqui o estado é registrado; lá ele é mostrado.
- Aplicar `RECUSADO` (5.10), expirar Tentativa (5.11), mexer na ordem do tique de `cmd/azamon`.
- Fazer a emissão consultar o Status do Pedido para "não emitir sobre cancelado" — seria a aresta proibida do AD-7, e apagaria justamente a informação que a invariante 7 manda guardar.
- Estorno, reembolso ou qualquer dado de cartão.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Aprovada, Tentativa corrente | Pedido em `AGUARDANDO_PAGAMENTO` | `PAGO`, Reserva `ATIVA`, inbox `APLICADA`, histórico com ator `PROVEDOR` | — (regressão da 1.7) |
| Aprovada sobre Pedido cancelado | Pedido em `CANCELADO`, Reserva `LIBERADA` | Status intacto, nenhuma transição nova, Reserva segue `LIBERADA`, inbox `NAO_APLICAVEL_SINALIZADA`, aviso no log | desfecho esperado, nunca erro da varredura |
| A mesma confirmação duas vezes | webhook repetido sobre Pedido cancelado | 200 nas duas, uma linha na inbox, um efeito | `23505` absorvido em `pagamento`, nunca no cliente |
| Varredura repetida | inbox já terminal | nada acontece; nenhuma transição nova | — |
| Pedido travado por outro tique | `TravarPedido` sem linha | confirmação fica `PENDENTE` | o tique seguinte a encontra |

</frozen-after-approval>

## Code Map

- `internal/pedido/pedido.go:534` -- `aplicarConfirmacao`: a transação de um Pedido só. O `estado := NaoAplicavelSinalizada` da linha 561 é o balde a nomear; `slog` já está importado e o `InfoContext` do `ErrEstadoJaAvancado` (l. 570) é o molde do aviso. `Varrer:520` não muda.
- `internal/pagamento/pagamento.go:82,163,196` -- `Pendente` (traz `Corrente`), `Marcar`, `ConfirmacoesNaoAplicadas`. A porta basta; nada novo aqui.
- `internal/pedido/maquina.go:102,122` -- `janelaDeCancelamento` e a linha `{…} → CANCELADO` com `efeitoLiberar`: é por `Transicionar(..., AtorComprador, "")` que o teste cancela, porque a tela de cancelamento é da 6.3.
- `api/webhook_test.go:32,164,187` -- os três subtestes da 1.7; `confirmacaoForaDeAguardandoPagamento` é o molde exato do novo. Helpers: `confirmar:311`, `criarTentativa:321`, `statusDe:340`, `confirmacoesDe:349`, `textoDe`, e `pedidoPeloCheckout` (`api/pedido_test.go:747`).
- `api/sessao_test.go:370-382` -- onde os subtestes da 1.7 são registrados, em ordem que importa. O novo entra depois de "confirmação de Tentativa superada" e antes de "webhook com corpo ruim".
- `api/pedido_detalhe_test.go:233` -- molde de emissão derivada dentro de teste, com `t.Cleanup` apagando Tentativas inventadas.
- `web/lib/pedido.ts:59,70` -- `superficieDoPedido` (`CANCELADO` → `detalhe`) e `intervaloDaConsulta` (terminal → para). Nada a mudar: é só a confirmação de que a tela não promete nada sobre o cancelado.

## Tasks & Acceptance

**Execution:**
- [x] `internal/pedido/pedido.go` -- em `aplicarConfirmacao`, separar "aprovada sobre Pedido cancelado" do resto do não-aplicável: mesmo estado terminal, mais `slog.WarnContext` com Pedido e confirmação, e um comentário fixando o predicado que a 6.5 deriva.
- [x] `api/webhook_test.go` -- novo subteste `confirmacaoAprovadaSobrePedidoCancelado`: Pedido novo pelo checkout (`produtoAprovado`), cancelado por `Transicionar` com `AtorComprador`, confirmação entregue pelo webhook de verdade, `Varrer`, e as asserções da matriz — Status, ausência de transição nova, Reserva `LIBERADA`, inbox sinalizada, repetição sem segundo efeito — mais o predicado da 6.5 devolvendo o Pedido.
- [x] `api/webhook_test.go` -- subteste `confirmacaoDePedidoTravadoFicaPendente`, a última linha da matriz: com a linha do Pedido travada por outra transação, `Varrer` não falha e a confirmação fica `PENDENTE`; solta a trava, o tique seguinte a aplica.
- [x] `api/sessao_test.go` -- registrar os dois subtestes na posição acima, com o comentário de por que a ordem importa.
- [x] `_bmad-output/implementation-artifacts/deferred-work.md`, `sprint-status.yaml`, `addendum.md` §10 + memlog -- registrar a decisão do sinal derivado (sem coluna) e o que a 6.5 herda; `5-9-…` para `review`.

**Acceptance Criteria:**
- Given um Pedido cancelado cuja Tentativa ainda não tinha confirmação, when o Provedor Simulado aprova e a varredura processa, then o dinheiro aprovado fica registrado na Tentativa e o Pedido não ressuscita.
- Given esse mesmo Pedido, when a leitura da 6.5 for construída, then o predicado da Design Notes o encontra sem ler log.
- Given a linha do Pedido travada por outra transação, when o tique passa, then ele não falha, a confirmação continua `PENDENTE` e o tique seguinte a aplica — nada se perde por chegar no instante errado.
- Given o caminho feliz da 1.7, when a suíte inteira roda, then ele continua verde: esta estória não muda o que já aplica.

## Spec Change Log

- **Revisão (Blind Hunter, lente única) — 2026-09-22.** Dezoito achados; nenhum `intent_gap` nem `bad_spec` — nenhum toca o bloco congelado e nenhum muda o desenho, então sem volta ao plano. Amendas fora do congelado: a Verification ganhou a mutação que prende o `if` novo (a anterior atacava linha pré-existente), o `grep` passou a casar os dois subtestes, e entrou o passeio manual da FR-26; as Design Notes passaram a decidir o Pedido cancelado **depois** de `PAGO`; e entrou a AC do Pedido travado. Estado conhecido-ruim evitado: um `if` de produção que podia ser apagado inteiro com a suíte verde, e um predicado herdado pela 6.5 com um caso não decidido. KEEP: o sinal continua derivado, sem coluna nem estado novo; o log continua fora do caminho de leitura; e `Corrente` continua fora do `if` do cancelado, de propósito.

## Design Notes

O predicado que a 6.5 vai derivar (nada aqui o expõe ainda; `pedido` o lerá por uma porta de `pagamento`, como já faz com `TentativasRestantes`):

```sql
SELECT EXISTS (
  SELECT 1 FROM pagamento.confirmacao_recebida c
  JOIN pagamento.tentativa_pagamento t ON t.id = c.tentativa_id
  WHERE t.pedido_id = $1 AND c.resultado = 'APROVADO'
    AND c.estado = 'NAO_APLICAVEL_SINALIZADA')
```
— combinado, em Go, com `Status == CANCELADO`. Confirmação de Tentativa superada que caia aqui não é falso positivo: o Provedor aprovou aquela cobrança, e o Pedido está cancelado do mesmo jeito.

Isso inclui, de propósito, o Pedido que passou por `PAGO` e só depois foi cancelado — `janelaDeCancelamento` cobre `PAGO` e `SEPARANDO`. Ali a sinalizada é uma **segunda** aprovação além da que já virou `PAGO`, e o estorno é mais urgente, não menos; restringir o predicado a "nunca esteve em `PAGO`" esconderia justamente a cobrança duplicada. O aviso no log segue a mesma regra.

Quem protege o Status do Pedido é o CAS de `Transicionar`, e não a condição de `aplicarConfirmacao`: sem ela, o `UPDATE … WHERE status = 'AGUARDANDO_PAGAMENTO'` casaria zero linhas e sairia `ErrEstadoJaAvancado`. A condição existe para **nomear** o caso e não gastar uma transição perdida — por isso a mutação que tem dente é sobre o estado gravado na inbox, e não sobre ela.

Coluna de motivo na inbox foi rejeitada: a AC da 6.5 nomeia `NAO_APLICAVEL_SINALIZADA` como o sinal, e o Status do Pedido já distingue o caso na leitura. Migração aqui seria estado a manter para responder o que uma junção responde.

Quem implementa não faz `git commit`, `git add` nem `push`.

## Verification

**Commands:**
- `go vet ./... && go test -count=1 ./...` -- expected: verde, na imagem `azamon-dev` (HANDOFF). Sem `sqlc generate`: nenhuma consulta nova.
- `go test -count=1 -run TestSessaoEProduto ./api/ -v 2>&1 | grep -E 'cancelado|travado'` -- expected: os dois subtestes novos aparecem e passam.
- Mutação 1: trocar o `estado := pagamento.NaoAplicavelSinalizada` inicial por `pagamento.Aplicada` -- expected: os três subtestes de sinalização falham. Reverter.
- Mutação 2: em `aprovadaSobreCancelado`, trocar `StatusCancelado` por `StatusPago` -- expected: o subteste do Pedido cancelado falha na leitura do log. Reverter. É a mutação que prende o único código de produção novo: sem a captura do aviso, o `if` inteiro podia ser apagado com a suíte verde.

**Manual checks (if no CLI):**
- `grep -n "pagamento_aprovado_sobre_cancelado" api/ web/ -r` -- expected: vazio. A superfície é da 6.5.
- A consequência mais visível da FR-26: checkout de total `,00` e, na tela do Pedido, `AGUARDANDO_PAGAMENTO` virando `PAGO` sozinho em ~5 s — o relógio some, o Detalhe aparece no lugar, sem recarregar e sem navegação. O mecanismo é da 5.8; aqui é a estória que o fecha confirmando que o viu funcionar.

## Suggested Review Order

**A borda que a estória fecha: aprovado sobre cancelado (FR-26)**

- Entrada: o caso ganha nome antes do `if` que aplica, e `Corrente` fica fora de propósito.
  [`pedido.go:570`](../../internal/pedido/pedido.go#L570)

- O aviso sai depois do commit: commit que falha se repetiria a cada tique.
  [`pedido.go:603`](../../internal/pedido/pedido.go#L603)

- Quem protege o Status é o CAS, e não a condição — o comentário no lugar certo.
  [`pedido.go:582`](../../internal/pedido/pedido.go#L582)

**A prova ponta a ponta**

- Cancelar vai por `Transicionar`: histórico e Reserva iguais aos de produção.
  [`webhook_test.go:216`](../../api/webhook_test.go#L216)

- O aviso capturado do log — sem isto, o `if` novo podia ser apagado inteiro.
  [`webhook_test.go:269`](../../api/webhook_test.go#L269)

- O predicado que a 6.5 herda, com a junção em Go e não em SQL (AD-1).
  [`webhook_test.go:302`](../../api/webhook_test.go#L302)

**A fila: nada se perde por chegar no instante errado**

- Pedido travado por outra transação: `PENDENTE` fica, e o tique seguinte aplica.
  [`webhook_test.go:322`](../../api/webhook_test.go#L322)

- `defer` no Rollback: `t.Fatalf` de dentro de helper deixaria a trava de pé.
  [`webhook_test.go:341`](../../api/webhook_test.go#L341)

**Periferia**

- Os dois helpers que tiraram três cópias de SQL do teste feliz.
  [`webhook_test.go:515`](../../api/webhook_test.go#L515)

- A razão verdadeira da ordem dos subtestes: `Varrer` é global.
  [`sessao_test.go:381`](../../api/sessao_test.go#L381)
