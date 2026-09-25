---
title: 'D3+D5 — preço por Item conferido sob a trava da criação do Pedido'
type: 'bugfix'
created: '2026-09-24'
status: 'done'
baseline_commit: '4e7f5a84de919ea9874c79855396aef12d907a1f'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `pedido.Criar` confere, sob a trava do AD-5, só o subtotal e o Frete (`internal/pedido/pedido.go:282`), nunca o preço de cada Item. Dois preços que mudam e se compensam — ou um que cai e cruza o limiar de isenção fechando o mesmo total — passam: o Pedido AZ-2026-000035 nasceu com R$ 86 + R$ 74 depois de a Revisão mostrar R$ 81 + R$ 79 (D3). E o único teste de `TOTAL_DIVERGENTE` é recusado no passo 2, então a conferência sob a trava pode ser apagada com a suíte verde (D5).

**Approach:** Sob a trava, recusar com `ErrTotalDivergente` todo Item cujo `produto.PrecoCentavos` difere de `item.PrecoVistoCentavos` — o preço de que o Comprador tem ciência (AD-17) e que a Revisão exibiu. Provar com dois cenários que só essa checagem recusa.

## Boundaries & Constraints

**Always:** a checagem fica no passo 5, com os Produtos já travados, e mantém a conferência de subtotal e Frete que já existe. A recusa é `409 TOTAL_DIVERGENTE`, nada gravado, chave de idempotência livre. Português nas mensagens; glossário do PRD §3 literal.

**Ask First:** código de erro novo, ou qualquer mudança no `web/`.

**Never:** checar o preço por Item no passo 2 (a prova do D5 depende de só o passo 5 recusar); tocar `SPEC.md`; mexer no comentário de `revisao-do-pedido.tsx:317` (D4, sessão própria).

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Preços que se compensam | Revisão com R$ 50 + R$ 30; Administrador muda para R$ 55 + R$ 25; POST com o total da Revisão | 409 `TOTAL_DIVERGENTE` | nada gravado, Carrinho intacto |
| Limiar cruzado, mesmo total | Produto de R$ 250,00 (isenção 25000, Frete grátis); cai para R$ 235,00 (+ R$ 15 de Frete Sudeste = mesmo total) | 409 `TOTAL_DIVERGENTE` | nada gravado |
| Ciência pela entrada | depois da recusa, `POST /api/v1/checkout/entrada`, nova cotação, mesma chave | 201, Item congelado com o preço novo | — |
| Preço inalterado | fluxo normal | 201, como hoje | — |

</frozen-after-approval>

## Code Map

- `internal/pedido/pedido.go:260-284` -- passo 5 de `Criar`: laço que relê `catalogo.BuscarProduto` sob a trava; a checagem entra nele, antes do `append`. `item` é `carrinho.ItemDoCarrinho`, que já traz `PrecoVistoCentavos` (só preenchido se visível; invisível já é recusado por `Reservar`/`BuscarProduto`).
- `internal/pedido/pedido.go:67-71` -- doc e mensagem de `ErrTotalDivergente`: hoje diz "o total mudou"; passa a cobrir o preço de um Item.
- `internal/pedido/pedido.go:115-128` -- comentário "a ordem é o contrato", passo 5: acrescentar o preço por Item.
- `internal/carrinho/carrinho.go:245-248` -- origem de `PrecoVistoCentavos` e `PrecoMudou` (leitura pura, não mexer).
- `api/pedido_test.go:1090-1110` -- teste atual de `TOTAL_DIVERGENTE`: reenvia com o preço novo **sem** entrada no checkout e passa a quebrar (medido: `pedido_test.go:1109`, 409). Precisa de `postarEntrada` antes de `totalDaRevisao`.
- `api/checkout_test.go:254` -- `postarEntrada`, reaproveitar. `api/pedido_test.go:860` `totalDaRevisao`; `:944` `publicar`; `:969` `fotografia`. `freteSudeste = 1500`, `freteIsencaoDeTeste = 25000` (`api/carrinho_test.go:20`).
- `web/lib/checkout.ts:435` e `:106-110` -- sem mudança: `TOTAL_DIVERGENTE` → releitura; `desvioDaRevisao` vê `preco_mudou` e manda ao passo Endereço, que reporta "de X para Y".

## Tasks & Acceptance

**Execution:**
- [x] `internal/pedido/pedido.go` -- no laço do passo 5, `if produto.PrecoCentavos != item.PrecoVistoCentavos { return …, ErrTotalDivergente }`; atualizar o doc de `ErrTotalDivergente`, a mensagem ("Um preço ou o total do Pedido mudou desde a Revisão. Confira os valores e confirme de novo.") e o passo 5 do comentário de contrato -- fecha o D3.
- [x] `api/pedido_test.go` -- no teste existente, `postarEntrada` antes de reler o total e reenviar; acrescentar os dois cenários da matriz, cada um conferindo 409, `fotografia` inalterada e o Carrinho intacto, e depois entrada + mesma chave = 201 com `preco_praticado_centavos` novo -- fecha o D5.

**Acceptance Criteria:**
- Given a checagem por Item apagada de `pedido.go`, when `go test ./api/...`, then os dois cenários novos falham.
- Given a conferência de subtotal/Frete da `:282` intacta e a por Item presente, when `go test ./...`, then a suíte inteira passa.

## Design Notes

Comparar com `PrecoVistoCentavos`, e não com o preço lido no passo 2: o visto é o que a Revisão exibiu (a Revisão desvia sempre que `preco_mudou`), e o do passo 2 já é o preço novo nos dois cenários — é por isso que as somas batem e o furo passa. A consequência é que reenviar sem passar pela entrada no checkout é recusado, e isso é o AD-17: a ciência só se grava na entrada.

## Verification

**Commands:**
- `go test ./...` -- expected: tudo `ok`.
- apagar a checagem por Item e rodar `go test ./api/...` -- expected: os dois cenários novos falham; restaurar.

## Suggested Review Order

**A checagem sob a trava**

- Entrada: o preço visto contra o relido com os Produtos travados, antes do congelamento.
  [`pedido.go:274`](../../internal/pedido/pedido.go#L274)

- O contrato de ordem do `Criar` ganha a conferência por Item no passo 5.
  [`pedido.go:130`](../../internal/pedido/pedido.go#L130)

- Mesmo código `TOTAL_DIVERGENTE`; a mensagem passa a cobrir o preço de um Item.
  [`pedido.go:73`](../../internal/pedido/pedido.go#L73)

**A prova (D5)**

- `Janela`: preços trocados entre o passo 2 e a trava — prende o lugar da checagem.
  [`pedido_test.go:1174`](../../api/pedido_test.go#L1174)

- `Compensados` e `Limiar`: mesmo total, outra divisão; entrada e mesma chave dão 201.
  [`pedido_test.go:1120`](../../api/pedido_test.go#L1120)

- O teste antigo passa pela entrada no checkout antes de reenviar (AD-17).
  [`pedido_test.go:1105`](../../api/pedido_test.go#L1105)
