---
title: 'Estória 6.3 — Cancelamento pelo Comprador'
type: 'feature'
created: '2026-09-23'
status: 'done'
baseline_commit: '25cc83c748b9f195721a0dee5374b63fe9fc74bd'
review_loop_iteration: 0
context: ['{project-root}/_bmad-output/implementation-artifacts/epic-6-context.md']
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A linha `{janela} → CANCELADO` existe na tabela do AD-3 desde a 5.1, mas nenhuma rota a alcança: o Comprador não cancela, a UJ-3 não anda, e a corrida da FR-26 (aprovação sobre cancelado) só acontece em teste. O duplo clique de cancelar sairia `FORA_DA_JANELA_DE_CANCELAMENTO` de um cancelamento que deu certo (`deferred-work.md`, 5.1).

**Approach:** `POST /api/v1/pedidos/{id}/cancelamento`, sem corpo: `pedido.Cancelar` lê o Pedido do dono **travado** (`FOR UPDATE`), trata `CANCELADO` como sucesso sem efeito e, fora isso, chama `Transicionar(atual → CANCELADO, COMPRADOR)`, que libera a Reserva. Na tela, "Cancelar Pedido" nos quatro Status da janela, com `Dialog` nomeando o Pedido, e a frase da corrida perdida no lugar do erro.

## Boundaries & Constraints

**Always:**
- Toda mudança pelo `Transicionar`; `Liberar` é o efeito dele, incondicional — `Cancelar` não o chama nem checa Reserva (AD-3, AD-5).
- Trava na ordem do AD-4: linha do Pedido primeiro (a leitura travada), Produtos depois (dentro do `Liberar`). Transação aberta na borda por `emTransacao`; JSON só depois do `Commit`.
- Pedido alheio, inexistente e malformado: o mesmo 404 (AD-11).
- Botão ausente fora da janela (a frase da 6.2 continua); durante o envio, presente, `aria-disabled` e com progresso (UX-DR11). Decisão e texto em funções puras de `web/lib/pedido.ts`, testadas.
- Nenhum texto nomeia a Reserva de Estoque; sem laranja nem verde (UX-DR7).

**Ask First:**
- Migração, coluna nova, ou mudar a tabela de transições / as três recusas de `maquina.go`.

**Never:**
- Rota administrativa de cancelamento; o Administrador não cancela (FR-32).
- `Toast` para a corrida perdida; mensagem crua de erro no lugar da frase.
- Selo `Badge` por Status (6.7); ler o sinal de pagamento aprovado sobre cancelado (6.5).
- Prometer reembolso ou devolução em qualquer texto.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Cancela na janela | `AGUARDANDO_PAGAMENTO`, `PAGO` ou `SEPARANDO` | 200 com o Pedido `CANCELADO`; linha `→ CANCELADO` ator `COMPRADOR`; Estoque disponível volta | N/A |
| Recusado, sem Reserva | `PAGAMENTO_RECUSADO` | 200 `CANCELADO`; disponível não muda | `Liberar` sem Reserva ativa é `nil` |
| Já cancelado | `CANCELADO` (inclusive o 2º clique) | 200 `CANCELADO`, nenhuma linha nova no histórico | N/A |
| Fora da janela | `ENVIADO` ou `ENTREGUE` | 409 `FORA_DA_JANELA_DE_CANCELAMENTO`, `dados.status` = Status atual; nada gravado | N/A |
| Corrida perdida na tela | tela em `SEPARANDO`, Pedido já `ENVIADO` | a tela relê: Status e linha do tempo atualizados, e "Este Pedido foi enviado enquanto você estava nesta tela e não pode mais ser cancelado." no lugar da frase da 6.2 | nunca `Toast` nem erro |
| Alheio / inexistente / malformado | outro dono ou uuid inválido | 404 idêntico ao do inexistente | N/A |
| Sem Sessão | sem cookie | 401; a tela vai ao Login | N/A |
| Aprovação depois do cancelamento | `,00` cancelado em `AGUARDANDO_PAGAMENTO`, depois emitir + varrer | segue `CANCELADO`, Reserva liberada, confirmação `NAO_APLICAVEL_SINALIZADA` | N/A |

</frozen-after-approval>

## Code Map

- `internal/pedido/maquina.go:199` -- `Transicionar`: CAS, histórico e `Liberar`; com `ENVIADO`/`ENTREGUE` devolve `ErrForaDaJanelaDeCancelamento` sem tocar o banco (:202). Não muda.
- `internal/pedido/pedido.go:350` -- `NovaTentativa`: molde de `Cancelar` (ids → `ErrNoRows`, dono, `Transicionar`).
- `internal/pedido/db/consultas.sql:100,163` -- `BuscarPedidoDoComprador` (dono no `WHERE`) e `TravarPedido` (`SKIP LOCKED`, da varredura): a consulta nova junta o dono com `FOR UPDATE` **que espera**.
- `api/pedido.go:256` -- `novaTentativa`: molde do handler (`semCache`, `compradorDaRequisicao`, `emTransacao`, 404).
- `api/rotas.go:62` -- registrar a rota ao lado de `/tentativas`.
- `internal/plataforma/erro/erro.go:69` -- `FORA_DA_JANELA_DE_CANCELAMENTO` já é 409; `dados` passa direto.
- `api/nova_tentativa_test.go:326` -- molde do teste que segura a linha do Pedido e espera `pg_stat_activity`; `emitirPara`/`varrer` (:55).
- `api/autorizacao_test.go:63` -- tabela de `negacaoPorDono`.
- `api/sessao_test.go:551` -- onde as matrizes de servidor são registradas.
- `web/app/pedidos/[id]/acompanhamento.tsx:299,333` -- `tentarDeNovo` (ref de envio, `consultarAgora`, foco no título) e o Card do topo com a região `role="status"`; a frase da 6.2 mora em :199.
- `web/lib/pedido.ts:143,194` -- `porQueNaoCancela` e `desfechoDaNovaTentativa`: moldes das funções novas.
- `web/app/admin/produtos/produtos.tsx:549` -- `Dialog` de confirmação que não fecha durante o envio.

## Tasks & Acceptance

**Execution:**
- [x] `internal/pedido/db/consultas.sql` + `gerado/` -- `TravarPedidoDoComprador :one` (`id, numero, status, total_centavos`, dono no `WHERE`, `FOR UPDATE`); `sqlc generate` na imagem `sqlc/sqlc:1.31.1`, restaurando os `gerado/` de outros módulos.
- [x] `internal/pedido/pedido.go` -- `Cancelar(ctx, tx, pedidoID, compradorID) (Pedido, error)`: travar; `CANCELADO` → devolve sem efeito; senão `Transicionar`. Na recusa, o `Pedido` devolvido traz o Status lido.
- [x] `api/pedido.go` + `api/rotas.go` -- `cancelarPedido`: 200 com `saidaDoPedido`; `ErrNoRows` → 404; fora da janela → 409 com `dados: {"status": …}`.
- [x] `api/cancelamento_test.go` + `api/sessao_test.go` + `api/autorizacao_test.go` -- matriz inteira contra o banco, mais: duplo clique preso na trava (dois 200, **uma** linha `CANCELADO`); `SEPARANDO → ENVIADO` comitado enquanto o cancelamento espera (409 com `status: ENVIADO`, Reserva consolidada, não liberada); cancelamento por POST na tabela do AD-11.
- [x] `web/lib/pedido.ts` + `web/scripts/pedido.test.mjs` -- `podeCancelar`, `rotaDoCancelamento`, `desfechoDoCancelamento` (200 e `ESTADO_JA_AVANCADO` → relê calado; `FORA_DA_JANELA_DE_CANCELAMENTO` → relê com a frase da corrida; 401 → Login; resto → erro com a mensagem do envelope), com teste.
- [x] `web/app/pedidos/[id]/acompanhamento.tsx` -- bloco de cancelamento no Card do topo, nas três superfícies: botão, `Dialog` e a frase de quando não há (movida de Valores).
- [x] `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` §10 + `.memlog.md` -- a leitura travada e o `CANCELADO` como sucesso; `deferred-work.md` -- `RESOLVIDO —` das entradas do duplo clique (5.1) e da FR-26 inalcançável (5.9).

**Acceptance Criteria:**
- Dado um Pedido na janela, quando o Comprador confirma no `Dialog` que nomeia o número do Pedido, então Status e linha do tempo mudam sem recarregar, o foco vai ao título, e outra aba vê o Estoque disponível de volta.
- Dado o envio em curso, quando o Comprador clica de novo, então nada é enviado, e o `Dialog` não fecha por Esc nem por clique fora.
- Dado 360 px e só o teclado, quando o `Dialog` abre, então o foco fica preso nele e nada rola na horizontal.

## Design Notes

**Por que travar antes do CAS:** sem a trava, o 2º clique lê `SEPARANDO`, espera no `UPDATE`, perde, e `corridaPerdida` relê `CANCELADO` → `FORA_DA_JANELA`; e o cancelamento concorrente com `AGUARDANDO → PAGO` sairia `ESTADO_JA_AVANCADO` de um Pedido que ainda cancela. Travado, o Status lido é o que o CAS vê, e as duas corridas viram desfecho certo. A varredura usa `SKIP LOCKED` e só pula esse Pedido naquele tique.

**Por que o Card do topo:** em `AGUARDANDO_PAGAMENTO` o Detalhe não é mostrado, então o botão em Valores (onde a 6.2 o anunciou) faltaria em um dos quatro Status. Botão, frase de "não dá mais" e frase da corrida ficam num lugar só, junto da região que anuncia o Status: `avisoDaCorrida ?? porQueNaoCancela(status)`.

`Dialog`: título "Cancelar o Pedido {numero}?", texto "Depois de cancelado, o Pedido não volta a andar.", botões "Manter o Pedido" (outline) e "Cancelar Pedido" (`destructive`; enviando: "Cancelando o Pedido…"). Erro que não é corrida fica dentro do `Dialog`, em `role="alert"`.

## Verification

**Commands:**
- `go vet ./... && go test -count=1 ./...` na imagem `azamon-dev` (HANDOFF) -- verde.
- `docker build ./web` -- verde (`prebuild` roda `verificar-offline.mjs` e `npm test`).
- Mutação: tirar o `FOR UPDATE` da consulta nova -- o teste do duplo clique falha. Reverter.

**Manual checks (if no CLI):**
- UJ-3 a 360 e 1440 px, só pelo teclado: Pedido em `SEPARANDO` → "Cancelar Pedido" → confirmar → "Cancelado" na região viva e na linha do tempo; outra aba na Página de Produto mostra o Estoque de volta.

## Review Triage Log

Passe 1 — três lentes (`blind-hunter`, `edge-case-hunter`, `verification-gap`): o diff toca trava de linha, máquina de estados e Estoque.

| # | Achado | Veredito | Rota |
|---|--------|----------|------|
| 1 | A janela de cancelamento copiada à mão no TS (`JANELA_DE_CANCELAMENTO`): regra no Node, e diverge do Go em silêncio | medium | patch — `pode_cancelar` no Detalhe, de `Permitidas(status, COMPRADOR)` |
| 2 | POST que nunca volta prende o `Dialog` aberto (Esc, clique fora e "Manter" bloqueados em voo) | medium | patch — `AbortController` com o prazo de 30 s do Confirmar Pedido |
| 3 | O cancelamento preso atrás de `AGUARDANDO → PAGO` não comitado não tinha teste | medium | patch — subteste novo; morre sem o `FOR UPDATE` |
| 4 | `negacaoPorDono` passaria vazio com o Pedido do dono fora da janela | low | patch — pré-condição com `t.Fatalf` |
| 5 | Comentário de `cancelarPedido` dizia que a tela usa `dados.status` | low | patch |
| — | Sem teste de componente nem passeio no navegador; teste instável de `cmd/azamon` | — | já no `deferred-work.md` |
| — | Espera de trava sem `lock_timeout`, correlação no erro do `Dialog`, id do Comprador malformado em 404, `ESTADO_JA_AVANCADO` relido calado, texto sobre dinheiro no `Dialog` | — | rejeitados: molde das rotas existentes ou decisão da spec |

## Suggested Review Order

**A leitura travada e o cancelamento**

- Entrada: trava o Pedido do dono, `CANCELADO` é sucesso sem efeito, o resto vai a `Transicionar`.
  [`pedido.go:409`](../../internal/pedido/pedido.go#L409)

- `FOR UPDATE` que espera: o Status lido é o que o CAS vê; dono no `WHERE`.
  [`consultas.sql:182`](../../internal/pedido/db/consultas.sql#L182)

- O segundo clique lê `CANCELADO` aqui e sai 200, sem histórico novo.
  [`pedido.go:430`](../../internal/pedido/pedido.go#L430)

**A borda HTTP**

- Handler no molde da nova Tentativa; JSON só depois do `Commit`.
  [`pedido.go:312`](../../api/pedido.go#L312)

- Fora da janela leva o Status atual em `dados`, para quem chama a API direto.
  [`pedido.go:334`](../../api/pedido.go#L334)

- `pode_cancelar` sai da tabela do AD-3: a janela não é redeclarada no Node.
  [`pedido.go:139`](../../api/pedido.go#L139)

- A rota, sem par administrativo (FR-32).
  [`rotas.go:66`](../../api/rotas.go#L66)

**A tela**

- O clique: guarda por ref, prazo de 30 s, fecha o `Dialog` antes de reler.
  [`acompanhamento.tsx:366`](../../web/app/pedidos/[id]/acompanhamento.tsx#L366)

- O `Dialog` não fecha em voo; ao fechar, o foco vai ao título.
  [`acompanhamento.tsx:534`](../../web/app/pedidos/[id]/acompanhamento.tsx#L534)

- Botão no Card do topo, nas três superfícies, pelo `pode_cancelar`.
  [`acompanhamento.tsx:495`](../../web/app/pedidos/[id]/acompanhamento.tsx#L495)

- A frase de "não dá mais" — ou a da corrida — entra na região viva do Status.
  [`acompanhamento.tsx:435`](../../web/app/pedidos/[id]/acompanhamento.tsx#L435)

- Desfecho da resposta: 409 fora da janela relê e explica, nunca `Toast`.
  [`pedido.ts:197`](../../web/lib/pedido.ts#L197)

- Frase da corrida no lugar da de Status, só sem o botão.
  [`pedido.ts:213`](../../web/lib/pedido.ts#L213)

**Testes**

- As duas corridas na trava: duplo clique (uma linha só) e envio comitado (409 `ENVIADO`).
  [`cancelamento_test.go:350`](../../api/cancelamento_test.go#L350)

- A aprovação comitada enquanto espera: o `PAGO` é cancelado, não `ESTADO_JA_AVANCADO`.
  [`cancelamento_test.go:441`](../../api/cancelamento_test.go#L441)

- A matriz na janela, com a Reserva liberada e o disponível de volta.
  [`cancelamento_test.go:217`](../../api/cancelamento_test.go#L217)

- AD-11: o POST alheio é o mesmo 404, e o Pedido do dono não muda.
  [`autorizacao_test.go:75`](../../api/autorizacao_test.go#L75)

- O botão segue a flag do Go; nenhum texto nomeia a Reserva nem promete reembolso.
  [`pedido.test.mjs:232`](../../web/scripts/pedido.test.mjs#L232)
