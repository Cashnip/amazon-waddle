---
title: 'D2, D6, D7, E2, E3 — o foco volta a quem abriu o Dialog, e o destrutivo passa do AA'
type: 'bugfix'
created: '2026-09-24'
status: 'done'
baseline_commit: '9240730ace826d961aaf38b0ee3a121b54d36309'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Todo `Dialog` do `web/` abre por estado, sem `DialogTrigger`, e o Radix só devolve o foco ao `Trigger`: no `Esc` ou no "Voltar" o foco cai no `body` (D6 no novo Endereço, E2 no cancelamento e no avanço de Status — e as mesmas cinco telas de remover/ajustar/esvaziar têm o defeito sem nome). "Confirmar o novo preço" desmonta o próprio botão e o foco também cai no `body` (D7 no passo Endereço; o Carrinho tem a mesma cópia). O fechar de todo `Dialog` e `Sheet` se anuncia `Close` (D2). "Cancelar Pedido" no `Dialog` do Comprador é o `destructive` a 10%, 3,99:1 (E3).

**Approach:** Corrigir o foco uma vez, no `DialogContent` compartilhado: guardar o elemento focado ao abrir e devolvê-lo ao fechar, sempre que quem usa não tiver decidido outro destino. "Confirmar o novo preço" leva o foco ao título da tela, como a 6.3 já faz. Traduzir o `Close`. O botão de E3 vira vermelho cheio com branco, o mesmo par da 6.7.

## Boundaries & Constraints

**Always:** os `onCloseAutoFocus` que já existem (título do Pedido na 6.3, `focarBloco` na 6.4) continuam vencendo — o `preventDefault` deles desliga a devolução. Elemento que já saiu do DOM não recebe foco: cai no comportamento do Radix. Português; glossário do PRD §3 literal.

**Ask First:** trocar a variante `destructive` do `button.tsx` para todo o sistema; pausar a consulta do `Dialog` do Comprador (O3 foi **aceito** em 2026-09-24: sem mudança).

**Never:** acrescentar `DialogTrigger` tela por tela (a correção é uma, no compartilhado); dependência nova; laranja ou verde no botão de E3; tocar Go.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Esc sem destino próprio | foco em "Cadastrar Endereço", abre, `Esc` | foco volta a "Cadastrar Endereço" | N/A |
| Destino próprio | cancelamento bem-sucedido (6.3) | foco no título do Pedido, não no botão | N/A |
| Quem abriu sumiu | botão de avanço desmontado pela consulta com o `Dialog` aberto, "Voltar" | nada é focado à força; comportamento do Radix | N/A |
| Confirmar preço | foco em "Confirmar o novo preço", Enter | `Alert` sai, foco no `h1` da tela | N/A |

</frozen-after-approval>

## Code Map

- `web/components/ui/dialog.tsx:52` -- `DialogContent`: ponto único. Radix (`@radix-ui/react-dialog` `index.mjs:154`) compõe `props.onCloseAutoFocus` antes do próprio handler e pula o dele se `defaultPrevented`; o `FocusScope` dispara `onOpenAutoFocus` com `document.activeElement` ainda no elemento de origem (`react-focus-scope` `:79`). `Close` em `:79` e `:118`.
- `web/components/ui/sheet.tsx:80` -- o outro `Close`. Os dois `Sheet` usam `SheetTrigger`; só texto.
- `web/app/checkout/endereco/escolha-de-endereco.tsx:249` -- `h1` sem ref; `:293` botão de confirmar preço; `:335` Dialog de D6.
- `web/app/carrinho/meu-carrinho.tsx:204` -- `h1`; `:260` a mesma cópia do botão de preço.
- `web/app/pedidos/[id]/acompanhamento.tsx:541` -- `onCloseAutoFocus` da 6.3 (reuso: `h1 ref tabIndex={-1} outline-none` em `:418`); `:572` botão de E3.
- `web/app/admin/pedidos/acoes-do-pedido.tsx:174` -- `onCloseAutoFocus` da 6.4 (só age depois do envio; o resto cai na devolução).
- `web/lib/pedido.ts:357` -- `falha: "bg-destructive text-white"`, o par medido a 4,76:1.

## Tasks & Acceptance

**Execution:**
- [x] `web/components/ui/dialog.tsx` -- em `DialogContent`, `useRef` do elemento focado capturado em `onOpenAutoFocus` (depois de chamar o do usuário); em `onCloseAutoFocus`, chamar o do usuário, e se não `defaultPrevented` e o elemento ainda `isConnected`, `preventDefault()` e focá-lo. `Close` → `Fechar` nos dois lugares -- D6, E2, D2.
- [x] `web/components/ui/sheet.tsx` -- `Close` → `Fechar` -- D2.
- [x] `web/app/checkout/endereco/escolha-de-endereco.tsx` -- `h1` com ref, `tabIndex={-1}`, `outline-none`; o botão de confirmar preço foca o título depois do `setConfirmados` -- D7.
- [x] `web/app/carrinho/meu-carrinho.tsx` -- o mesmo -- D7 na cópia.
- [x] `web/app/pedidos/[id]/acompanhamento.tsx` -- no botão de confirmar, `className` com `bg-destructive text-white` e hover escurecido por `color-mix` com preto, nunca clareado -- E3.

**Acceptance Criteria:**
- Given qualquer um dos oito `Dialog`s aberto pelo teclado, when `Esc`, then o foco está no botão que o abriu.
- Given o leitor de tela no botão de fechar de um `Dialog` ou `Sheet`, when lido, then diz "Fechar".
- Given o `Dialog` de cancelamento, when medido, then "Cancelar Pedido" tem contraste ≥ 4,5:1 em repouso e em hover.

## Design Notes

A captura fica em `onOpenAutoFocus`, e não no render: só ali se sabe que é abertura, e o `FocusScope` ainda não moveu o foco. No Safari o clique não foca botão, o elemento capturado é o `body`, e aí não se força nada — o Radix segue como hoje.

## Verification

**Commands:**
- `cd web && npm run build` -- expected: verificação offline, `npm test` e build verdes.

**Manual checks (if no CLI):**
- No navegador, pelo teclado: passo Endereço (`Esc` no novo Endereço; confirmar preço), Carrinho (esvaziar, `Esc`), Detalhe do Pedido ("Manter o Pedido"), `/admin/pedidos` ("Voltar"). `document.activeElement` conferido em cada um.

## Suggested Review Order

**Foco devolvido a quem abriu (D6, E2)**

- Ponto único: captura na abertura, devolve no fechamento se ninguém escolheu outro destino.
  [`dialog.tsx:84`](../../web/components/ui/dialog.tsx#L84)

- A captura: em `onOpenAutoFocus`, antes do `FocusScope` mover o foco; `body` não conta.
  [`dialog.tsx:78`](../../web/components/ui/dialog.tsx#L78)

- Achado da revisão: `autoFocus` pula a captura; o Radix já foca o primeiro campo.
  [`produtos.tsx:562`](../../web/app/admin/produtos/produtos.tsx#L562)

- A regra para quem vier depois.
  [`dialog.tsx:55`](../../web/components/ui/dialog.tsx#L55)

**Confirmar preço sem perder o foco (D7)**

- O botão some com o `Alert`; o foco vai ao título, como na 6.3.
  [`escolha-de-endereco.tsx:300`](../../web/app/checkout/endereco/escolha-de-endereco.tsx#L300)

- A mesma cópia no Carrinho.
  [`meu-carrinho.tsx:267`](../../web/app/carrinho/meu-carrinho.tsx#L267)

**Contraste e rótulo (E3, D2)**

- Vermelho cheio com branco, hover escurece, e parado quando `aria-disabled`.
  [`acompanhamento.tsx:577`](../../web/app/pedidos/[id]/acompanhamento.tsx#L577)

- `Close` vira `Fechar` no `Dialog` e no `Sheet`.
  [`dialog.tsx:104`](../../web/components/ui/dialog.tsx#L104)
