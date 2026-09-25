---
title: 'D1, D4, E1, E4, E5 — Produto de ,95 semeado, linha de Meus pedidos alinhada, frase da corrida que expira, Alert fixo sem anúncio'
type: 'bugfix'
created: '2026-09-24'
status: 'done'
baseline_commit: 'ab4558c587079bd8356ced4eafc8189e577b96fd'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Cinco sobras dos passeios das Épicas 5 e 6. D1: nenhum Produto semeado fecha total em `,95`–`,99`, então o passo 5 do Roteiro B (FR-34) só se demonstra mudando preço, contra "nada é reconfigurado entre os roteiros". D4: o comentário do bloco de valores da Revisão descreve a recusa pelo total, que é a guarda de antes do D3. E1: a linha de Meus pedidos é um `<a>` com `justify-between` e `underline` — colunas desalinhadas entre linhas e o selo sublinhado só nesta superfície. E4: a frase da corrida perdida do cancelamento nunca é limpa, e o Pedido já `ENTREGUE` continua dizendo "foi enviado enquanto você estava nesta tela". E5: o `Alert` persistente da 6.5 sai com `role="alert"` e é anunciado de forma assertiva a cada abertura, embora seja estado fixo.

**Approach:** D1 — um Produto da lista passa de `,90` a `,95`, pelo gerador. D4 — reescrever o comentário. E1 — a linha vira grade de colunas iguais, com o sublinhado só no número. E4 — a frase da corrida só vale enquanto o Status é `ENVIADO`. E5 — o `Alert` fixo perde o `role`.

## Boundaries & Constraints

**Always:** a semente continua saindo de `go run media/gerar.go` (o SQL não se edita à mão). O cartão inteiro continua sendo o alvo do toque em Meus pedidos. Glossário do PRD §3 literal; português.

**Ask First:** mudar `VersaoSemente`; trocar mais de um preço semeado; mexer no README (é da 7.1).

**Never:** tocar Go de produção; mudar o componente `Alert` do shadcn para todos (só o uso da 6.5); dependência nova; cor nova no selo.

## I/O & Edge-Case Matrix

| Cenário | Estado | Esperado |
|---|---|---|
| Corrida perdida, Pedido em `ENVIADO` | `pode_cancelar=false`, aviso da corrida | frase da corrida |
| Mesmo Pedido depois, em `ENTREGUE` | aviso ainda guardado | `porQueNaoCancela("ENTREGUE")` |
| Corrida perdida já em `ENTREGUE` | resposta 409 com Status `ENTREGUE` | frase de entregue |
| Botão na tela | `pode_cancelar=true` | `null`, com ou sem aviso |

</frozen-after-approval>

## Code Map

- `media/gerar.go:86` -- "Anotações de um Andarilho", 3990. Nenhum teste lê este Produto nem este preço; o SVG não carrega preço, então a regeneração só muda uma linha do SQL.
- `db/semente/001_catalogo_semeado.sql` -- saída do gerador. Estoque vem do `DEFAULT 10` da migração.
- `db/embutido.go:31` -- `VersaoSemente`. **Não mudar:** a semente é `INSERT` sem `ON CONFLICT`, então versão nova sobre banco já semeado bate na PK e derruba o arranque. Banco existente vê o preço novo com `docker compose down -v` (README:239).
- `web/app/checkout/revisao/revisao-do-pedido.tsx:318-321` -- comentário do D4. A recusa real é por Item, sob a trava: `internal/pedido/pedido.go:274` (`produto.PrecoCentavos != item.PrecoVistoCentavos` → `TOTAL_DIVERGENTE`).
- `web/app/pedidos/meus-pedidos.tsx:138-152` -- a linha do E1. `SeloDoStatus` só recebe `status`; o `Badge` é `inline-flex` e esticaria em célula de grade.
- `web/lib/pedido.ts:212-219` -- `fraseSemCancelamento`, o ponto do E4. `acompanhamento.tsx:248,399,412` guarda o aviso e chama a função; não precisa mudar.
- `web/scripts/pedido.test.mjs:436-447` -- teste existente de `fraseSemCancelamento`; `npm test` roda `node --test`.
- `web/app/admin/pedidos/[id]/detalhe-admin.tsx:135` -- o `Alert` da 6.5. `components/ui/alert.tsx:29` põe `role="alert"` antes do `{...props}`, então a prop do uso sobrescreve.

## Tasks & Acceptance

**Execution:**
- [x] `media/gerar.go` -- Andarilho 3990 → 3995; rodar `go run media/gerar.go` na raiz -- D1: R$ 39,95 com qualquer Frete inteiro fecha em `,95`, abaixo da isenção.
- [x] `web/app/checkout/revisao/revisao-do-pedido.tsx` -- o comentário diz que a Revisão não arbitra e que quem recusa é a criação, comparando sob a trava o preço de cada Item com o preço visto -- D4.
- [x] `web/app/pedidos/meus-pedidos.tsx` -- o `<a>` vira `grid` de 2 colunas, 4 a partir de `sm`, com `justify-items-start`, `tabular-nums` e sem `underline`/`text-link`; só o número leva `text-link underline`; o preço fica à direita (`justify-self-end`) -- E1.
- [x] `web/lib/pedido.ts` -- `fraseSemCancelamento` só devolve o aviso da corrida com `status === "ENVIADO"`; fora disso, a frase do Status. Atualizar o comentário -- E4.
- [x] `web/scripts/pedido.test.mjs` -- acrescentar ao teste existente: aviso da corrida com `ENTREGUE` devolve a frase de entregue -- prende o E4.
- [x] `web/app/admin/pedidos/[id]/detalhe-admin.tsx` -- `<Alert role={undefined}>` com comentário de uma linha: é estado fixo, não evento -- E5.

**Acceptance Criteria:**
- Given um clone limpo, when o Comprador compra uma unidade de "Anotações de um Andarilho", then o total fecha em `,95` e o Pedido expira sozinho no prazo (Roteiro B passo 5), sem reconfigurar nada.
- Given Meus pedidos com selos de larguras diferentes, when vista a 1440 e a 360 px, then número, data, selo e total ficam nas mesmas colunas em toda linha, e só o número é sublinhado.
- Given o Detalhe administrativo de um Pedido com aprovação sobre cancelado, when aberto, then o `Alert` aparece sem `role="alert"`.

## Design Notes

E4 pela função pura e não por `useEffect` que limpa o estado: a frase da corrida só é verdadeira enquanto o Pedido está em `ENVIADO`, e a regra fica onde o `node --test` a prende.

## Verification

**Commands:**
- `cd web && npm test` -- verde, com o caso novo.
- `cd web && npx tsc --noEmit` -- sem erro. (O `web/` não tem configuração de ESLint.)
- `git diff --stat db/semente` -- uma linha trocada, só o preço do Andarilho.

## Suggested Review Order

**A frase da corrida perdida (E4)**

- Regra na função pura: a frase só vale em `ENVIADO`, e o `node --test` a prende.
  [`pedido.ts:215`](../../web/lib/pedido.ts#L215)

- O caso novo: aviso guardado e Pedido já `ENTREGUE`.
  [`pedido.test.mjs:447`](../../web/scripts/pedido.test.mjs#L447)

**A linha de Meus pedidos (E1)**

- Grade de colunas iguais, com o sublinhado só no número e o selo intacto.
  [`meus-pedidos.tsx:146`](../../web/app/pedidos/meus-pedidos.tsx#L146)

**O Produto de ,95 (D1)**

- Um preço trocado na lista; o SQL saiu do gerador, e a `VersaoSemente` ficou.
  [`gerar.go:86`](../../media/gerar.go#L86)

**Periféricos**

- O `Alert` fixo da 6.5 perde o `role="alert"` só neste uso (E5).
  [`detalhe-admin.tsx:137`](../../web/app/admin/pedidos/[id]/detalhe-admin.tsx#L137)

- O comentário passa a nomear a checagem por Item sob a trava (D4).
  [`revisao-do-pedido.tsx:318`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L318)
