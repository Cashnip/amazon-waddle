---
title: 'Estória 6.6 — Simulação de entrega'
type: 'feature'
created: '2026-09-24'
status: 'done'
baseline_commit: '7e6c1a8deba7bb8a6b439e867fb2b2fc953c7e8c'
review_loop_iteration: 0
context: ['{project-root}/_bmad-output/implementation-artifacts/epic-6-context.md']
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A simulação de entrega roda desde a 1.8 e a FR-33 está quase toda no código, mas só o caminho feliz e o interruptor têm prova. Nunca mover `CANCELADO` nem `AGUARDANDO_PAGAMENTO`, pular o Pedido travado, isolar o Pedido que falha e retomar depois de reiniciar são afirmados em comentário e não morrem com mutação nenhuma. De carona, o teste do binário pisca porque lê o Status corrente com a simulação a 200 ms (`deferred-work.md`, entrada da 6.3).

**Approach:** Prender cada condição da FR-33 e do AD-6 por teste. Uma matriz nova em `api/` chama `pedido.SimularEntrega` direto. O binário ganha um arranque a mais, que retoma os Pedidos deixados pelo binário desligado. A espera instável passa a ler o histórico. Não se espera nenhuma linha de produção nova.

## Boundaries & Constraints

**Always:**
- O avanço é só pelas três linhas `AtorSimulacao` da tabela do AD-3, via `Transicionar`. Nenhum `UPDATE` de Status no teste; a montagem de estado por `INSERT` direto segue o molde de `cmd/azamon/main_test.go`.
- Toda asserção de "não mexeu" confere Status, contagem de linhas do histórico e estado da Reserva, e não só o Status.
- O subteste novo usa conta, Vendedor, Categoria e Produtos próprios e é registrado por último em `TestSessaoEProduto`: `SimularEntrega` é global, e Pedido envelhecido que sobrasse viraria candidato dos subtestes seguintes.
- O Pedido que falha nasce antes do saudável: `PedidosParaAvancar` ordena por `id` (uuidv7), então só assim a falha acontece antes do avanço do outro.

**Ask First:**
- Qualquer mudança em código de produção (`internal/`, `cmd/azamon/main.go`, `api/` fora de `_test.go`), migração ou variável de ambiente nova. Se um teste novo achar defeito real, pare e pergunte.

**Never:**
- Gancho de teste dentro de `avancarPeloTempo` para alcançar a releitura travada de `Desde` (fica no `deferred-work.md`).
- `LIMIT` nos candidatos, corte de tempo vindo do banco, mudança em `web/`.
- Tocar a expiração (`Expirar`) ou o sinal da 6.5.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Nunca move | `CANCELADO`, `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `ENTREGUE`, histórico de 1 h, intervalo de 30 min | Status, histórico e Reserva intactos | `SimularEntrega` devolve `nil` |
| Pedido travado | `PAGO` vencido, linha presa por outra transação aberta (o Administrador no meio da transição) | pulado nesta chamada; outro `PAGO` vencido avança na mesma chamada; solta a trava, a chamada seguinte o avança | `nil` com a trava presa |
| Pedido que falha | `SEPARANDO` vencido com Reserva `ATIVA` de quantidade maior que o `estoque_total` do Produto (o `CHECK (estoque_total >= 0)` derruba o `Consolidar`) | fica em `SEPARANDO`, Reserva `ATIVA`, `estoque_total` intacto; outro `SEPARANDO` vencido, criado depois, vai a `ENVIADO` e consolida | erro que nomeia o id do Pedido que falhou, e só ele |
| Retomada no arranque | binário desligado deixou um `PAGO` e um `SEPARANDO` (com Reserva `ATIVA`) vencidos | novo arranque com a simulação ligada leva os dois a `ENTREGUE`: 3 e 2 linhas `SIMULACAO`, uma consolidação cada | N/A |

</frozen-after-approval>

## Code Map

- `internal/pedido/pedido.go:36,1042,1096,1122` -- `simulacao` (o mapa único), `SimularEntrega`, `avancarPeloTempo` (erros juntados por `errors.Join`, cada um com o id) e `avancarUm` (`TravarPedido` com `SKIP LOCKED` devolve `ErrNoRows` → `nil`). Somente leitura.
- `internal/pedido/db/consultas.sql:243,283` -- `TravarPedido` e `PedidosParaAvancar` (`ORDER BY p.id`).
- `db/migracoes/20260912120100_catalogo_reserva_estoque.sql:12` -- `CHECK (estoque_total >= 0)`, a falha natural do cenário "Pedido que falha"; `reserva_estoque` não tem gatilho que barre o `INSERT` direto.
- `api/pedido_test.go:678` -- `simulacaoDeEntrega`: caminho feliz, "cedo demais" e a baixa presa a `ENVIADO` (`conferirEstoque`); já cobre "nenhuma outra transição altera o total". Não muda.
- `api/maquina_test.go:300` -- `envelhecerHistorico`; `api/pedido_test.go` -- `textoDe`.
- `api/aprovado_sobre_cancelado_test.go:37-53` -- molde de conta de Administrador, Vendedor, Categoria e `novoProduto` pela rota.
- `api/sessao_test.go:596` -- último `t.Run` (6.5); o novo entra depois dele.
- `cmd/azamon/main_test.go:138,154` -- `interruptorDesligaASimulacao`: sobe o binário com a simulação desligada e deixa `AZ-INTERRUPTOR-0001` em `PAGO` vencido; o arranque novo vem depois dele parar.
- `cmd/azamon/main_test.go:505,541` -- `esperarSair` e a asserção instável do `noPrazo`.
- `internal/plataforma/config_test.go:23`, `env_test.go:52` -- 24 h padrão e 30 s do `.env` já presos. Não mudam.

## Tasks & Acceptance

**Execution:**
- [x] `api/simulacao_test.go` -- `simulacaoDeEntregaFR33`: as três primeiras linhas da matriz, com `pedido.SimularEntrega(ctx, pool, 30*time.Minute)` e histórico de 1 h. Na trava, a transação aberta faz `SELECT … FOR UPDATE` na linha do Pedido.
- [x] `api/sessao_test.go` -- registrar o subteste por último, com o comentário do porquê.
- [x] `cmd/azamon/main_test.go` -- `reinicioRetomaAEntrega`: antes de o binário desligado parar, `SEPARANDO` vencido com Reserva `ATIVA`; depois, arranque com a simulação ligada (`AZAMON_ENTREGA_INTERVALO=200ms`, endereço próprio) e espera por `ENTREGUE` nos dois, conferindo contagem de linhas `SIMULACAO`, Reserva `CONSOLIDADA` e `estoque_total` baixado uma vez.
- [x] `cmd/azamon/main_test.go` -- `noPrazo` julgado pelo histórico: a primeira saída de `AGUARDANDO_PAGAMENTO` é para `PAGO` e nenhuma linha vai a `PAGAMENTO_RECUSADO`.
- [x] `addendum.md` §10 + `.memlog.md` (por `uv run _bmad/scripts/memlog.py append`) -- o que a 6.6 provou e que nenhuma linha de produção mudou; `deferred-work.md` -- entrada "RESOLVIDO" para o teste instável da 6.3.

**Acceptance Criteria:**
- Dado cada mutação abaixo, aplicada sozinha, quando `go test ./api/ ./cmd/azamon/` roda, então um teste de `api/` ou `cmd/azamon/` morre (e não só o `pedido_test.go`, que prende o mapa): `CANCELADO` ou `AGUARDANDO_PAGAMENTO` no mapa `simulacao`; `SKIP LOCKED` trocado por espera com `return` na primeira falha de `avancarPeloTempo`; `return err` no lugar do `append` de falhas.
- Dado o binário com a simulação a 200 ms, quando o `noPrazo` é aplicado, então a asserção passa esteja ele em `PAGO` ou já adiante.

## Spec Change Log

## Design Notes

**Por que `ErrEstadoJaAvancado` não tem cenário próprio:** com a linha presa por `TravarPedido`, o CAS de `Transicionar` na mesma transação não perde. O ramo é defensivo; o "alguém chegou primeiro" do AD-6 acontece de verdade no `SKIP LOCKED` (linha "Pedido travado") e na reconferência de `Status` sob a trava.

**Por que a trava vira espera na mutação:** trocar `SKIP LOCKED` por `FOR UPDATE` que espera faria a chamada pendurar até o `context` do teste. O subteste dá prazo curto ao `ctx` da chamada e trata o estouro como falha.

Quem implementa não faz `git commit`, `git add` nem `push`.

## Verification

**Commands:**
- `go vet ./... && go test -count=1 ./...` na imagem `azamon-dev` (HANDOFF) -- verde, duas execuções seguidas.
- As três mutações do Critério de Aceite, uma de cada vez, revertidas -- o teste morre em cada uma.

**Manual checks (if no CLI):**
- `docker compose up`, Produto de `,00`, Pedido até `PAGO`, e a tela do Pedido chegando a "Entregue" sozinha em ~90 s, sem clicar.

## Review Triage Log

Passe 1 — uma lente (`verification-gap`), pela regra de custo do HANDOFF: o diff é só de teste e documento, sem linha de produção. A lente confirmou que cada mutação do Critério de Aceite derruba uma asserção, que o Pedido que falha para sempre não vaza (contêiner por função de teste, subteste registrado por último) e que o `noPrazo` reescrito não ficou mais fraco.

| # | Achado | Veredito | Rota |
|---|--------|----------|------|
| 1 | O addendum §10 e o comentário de `reinicioRetomaAEntrega` diziam que o terceiro arranque prova que o avanço deriva do histórico, e não de relógio em memória; com o intervalo de 200 ms, um relógio zerado no arranque também passaria | low (documento canônico creditando a prova ao teste errado) | patch — os dois textos passam a dizer o que o teste prova (retomar sem perder nem refazer etapa, uma consolidação) e apontam a prova do instante para `api/` |

## Suggested Review Order

**A matriz da FR-33 em `api/`: o que só comentário afirmava**

- Ponto de entrada: `SimularEntrega` chamada direto, estado por `INSERT`, histórico de 1 h contra 30 min.
  [`simulacao_test.go:42`](../../api/simulacao_test.go#L42)

- "Não mexeu" é Status, histórico, linhas `SIMULACAO` e Reserva numa linha só.
  [`simulacao_test.go:83`](../../api/simulacao_test.go#L83)

- Prazo curto por chamada: a trava que espera vira falha, e não pendura a suíte.
  [`simulacao_test.go:150`](../../api/simulacao_test.go#L150)

- Os quatro Status que a simulação nunca move, vencidos há uma hora.
  [`simulacao_test.go:161`](../../api/simulacao_test.go#L161)

- `SKIP LOCKED`: o travado é pulado sem erro e o do lado avança na mesma chamada.
  [`simulacao_test.go:196`](../../api/simulacao_test.go#L196)

- Falha natural pelo `CHECK (estoque_total >= 0)`, sem gancho; o que falha nasce primeiro.
  [`simulacao_test.go:248`](../../api/simulacao_test.go#L248)

- A ordem por uuidv7 é conferida, e não suposta.
  [`simulacao_test.go:261`](../../api/simulacao_test.go#L261)

**O binário: reiniciar retoma de onde parou**

- O terceiro arranque leva o `PAGO` e o `SEPARANDO` deixados até `ENTREGUE`, uma consolidação cada.
  [`main_test.go:224`](../../cmd/azamon/main_test.go#L224)

- O `SEPARANDO` nasce enquanto o binário desligado ainda tica.
  [`main_test.go:345`](../../cmd/azamon/main_test.go#L345)

- Montagem por `INSERT` com o histórico por último, no mesmo commit.
  [`main_test.go:157`](../../cmd/azamon/main_test.go#L157)

**O teste que piscava**

- `noPrazo` julgado pelo histórico: primeira saída para `PAGO`, nenhuma recusa.
  [`main_test.go:702`](../../cmd/azamon/main_test.go#L702)

**Periféricos**

- Registrado por último: `SimularEntrega` é global e o Pedido que falha continua falhando.
  [`sessao_test.go:587`](../../api/sessao_test.go#L587)

- O que a 6.6 provou, e o que o arranque novo não prova.
  [`addendum.md:408`](../planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md#L408)

- O teste instável da 6.3, fechado.
  [`deferred-work.md:497`](deferred-work.md#L497)
