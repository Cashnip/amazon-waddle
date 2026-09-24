---
title: 'Estória 6.7 — A forma de exibição dos sete Status'
type: 'feature'
created: '2026-09-24'
status: 'done'
baseline_commit: '881fc11820c12ae98076613c800ea7ca819c0fe7'
review_loop_iteration: 0
context: ['{project-root}/_bmad-output/implementation-artifacts/epic-6-context.md']
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O texto dos sete Status já é único (`rotuloDoStatus`), mas a forma não: as quatro telas escrevem o rótulo solto, sem selo nem cor, e a UX-DR10 e a UX-DR7 pedem o mesmo componente, texto e cor em Meus pedidos, no Detalhe do Pedido e na Tabela do Administrador, com o verde só em `ENTREGUE`.

**Approach:** Um componente só, `SeloDoStatus`: um `Badge` do shadcn em `rounded-full` cujo tom sai de uma função pura e testada em `web/lib/pedido.ts`. As quatro telas trocam o rótulo solto por ele. É estória só de `web/`: o Go não muda.

## Boundaries & Constraints

**Always:**
- O tom é fechado: `secondary` em `AGUARDANDO_PAGAMENTO`, `PAGO`, `SEPARANDO` e `ENVIADO`; `destructive` em `PAGAMENTO_RECUSADO` e `CANCELADO`; `{colors.available}` sobre `{colors.available-foreground}` só em `ENTREGUE`. Status desconhecido cai em `secondary`, com o texto cru que `rotuloDoStatus` já devolve, e **nunca** em verde.
- O texto do selo sai de `rotuloDoStatus`, e nenhum mapa novo de rótulo é criado. Sem ícone.
- Selo em 14px (`text-sm`, altura livre), por decisão humana de 2026-09-24: o rótulo do selo conta como texto de conteúdo (`DESIGN.md`, "Sem cinza pequeno"). O ajuste vai no `className` do uso, e `components/ui/badge.tsx` não é editado.
- O `destructive` do selo é preenchido (`bg-destructive text-white`, 4,76:1). O `Badge` `destructive` do shadcn tinge a 10% e dá 3,99:1, abaixo do piso AA.
- As regiões `role="status"` do Detalhe do Comprador e do Detalhe administrativo continuam existindo desde o primeiro render. Só o conteúdo delas troca, e o selo entra dentro.
- Na célula de Status da Tabela, o marcador `outline` "Pagamento aprovado" da 6.5 e o separador `sr-only` ficam onde estão, depois do selo.

**Ask First:**
- Qualquer mudança fora de `web/`, e qualquer edição em `web/components/ui/*`.

**Never:**
- Selo na linha do tempo, nas opções do filtro por Status, nas frases (`anuncioDaTabela`, `desfechoDaTransicao`, "Nenhum Pedido em …") ou no marcador da 6.5. Esses lugares continuam só com o texto de `rotuloDoStatus`.
- `bg-available` ou `text-available` em qualquer outro lugar novo do `web/`. Status ou transição copiados no Node.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Progresso | `AGUARDANDO_PAGAMENTO`, `PAGO`, `SEPARANDO`, `ENVIADO` | tom `progresso` (selo `secondary`) com o rótulo | N/A |
| Falha | `PAGAMENTO_RECUSADO`, `CANCELADO` | tom `falha` (selo `destructive` preenchido) | N/A |
| Fim bem-sucedido | `ENTREGUE` | tom `entregue` (verde), o único | N/A |
| Status desconhecido | `DEVOLVIDO`, `constructor` | tom `progresso`, texto cru | sem verde e sem exceção |

</frozen-after-approval>

## Code Map

- `web/lib/pedido.ts:295-322` -- `rotulosDoStatus`, `rotuloDoStatus` (com `Object.hasOwn`) e `STATUS`. O comentário "o selo é da 6.7" é atualizado. `tomDoStatus` entra ao lado.
- `web/components/ui/badge.tsx` -- somente leitura. A base traz `rounded-4xl`, que o `globals.css:143` rebaixa a 8px, então o selo precisa de `rounded-full`. `h-5` e `text-xs` são sobrepostos pelo `className` via `cn` (pacote `cn`, que funde como o `tailwind-merge`).
- `web/app/globals.css:112-124` -- `--available`, `--color-available` e `--color-available-foreground`. As classes `bg-available` e `text-available-foreground` existem.
- `web/app/pedidos/meus-pedidos.tsx:149` -- `<span>{rotuloDoStatus(pedido.status)}</span>`, dentro do `<a>` sublinhado da linha. O sublinhado não desce para o `inline-flex` do selo.
- `web/app/pedidos/[id]/acompanhamento.tsx:426` -- `Status: <span className="font-medium">{rotuloDoStatus(...)}</span>` na região viva.
- `web/app/admin/pedidos/pedidos.tsx:253` -- o rótulo na célula, ao lado do `Badge` `outline` da 6.5. `:164` (o filtro) e `:215` (a frase do vazio) não mudam.
- `web/app/admin/pedidos/[id]/detalhe-admin.tsx:129` -- mesmo padrão do Detalhe do Comprador.
- `web/components/cartao-de-produto.tsx:37`, `web/app/produtos/[id]/caixa-de-compra.tsx:115` -- os dois únicos `text-available` de hoje, os dois de Estoque disponível. Não mudam.
- `web/scripts/pedido.test.mjs:203-213,448` -- os testes de rótulo. O de tom entra ao lado. O import é pelo `import()` do `.ts`, e `npm test` roda `node --test scripts/*.test.mjs`.

## Tasks & Acceptance

**Execution:**
- [x] `web/lib/pedido.ts` -- exportar `tomDoStatus(status): "progresso" | "falha" | "entregue"`, com mapa próprio de dois Status e `Object.hasOwn`. Tudo que não é falha nem `ENTREGUE` é progresso. -- a regra de cor numa função pura, testável pelo `node --test`.
- [x] `web/components/selo-do-status.tsx` -- `SeloDoStatus({ status })`: um `Badge` com `rounded-full text-sm h-auto` e as classes de cada tom num `Record`, o texto vindo de `rotuloDoStatus`. -- o componente único da UX-DR10, e o único dono de `bg-available`.
- [x] `web/app/pedidos/meus-pedidos.tsx`, `web/app/pedidos/[id]/acompanhamento.tsx`, `web/app/admin/pedidos/pedidos.tsx`, `web/app/admin/pedidos/[id]/detalhe-admin.tsx` -- trocar o rótulo solto do Status do Pedido por `<SeloDoStatus status={…} />` e tirar o import de `rotuloDoStatus` que ficar sem uso. -- as três superfícies (quatro telas) com o mesmo selo.
- [x] `web/scripts/pedido.test.mjs` -- testar a matriz de I/O inteira sobre `STATUS` e sobre os desconhecidos, e que `entregue` sai para exatamente um Status. Acrescentar uma guarda de fonte: as quatro telas contêm `<SeloDoStatus` e não contêm `rotuloDoStatus(pedido.status)` nem `rotuloDoStatus(p.status)`, e `bg-available` só aparece em `components/selo-do-status.tsx` dentro de `app/`, `components/` e `lib/`. -- sem bancada de componente, é a guarda que impede uma quinta forma de voltar.
- [x] `_bmad-output/implementation-artifacts/deferred-work.md` -- marcar como resolvida a entrada da 1.2 sobre os 12px do `Badge`, com a decisão de 2026-09-24. -- fecha o registro que esperava o dono da UX.

**Acceptance Criteria:**
- Given qualquer das quatro telas com um Pedido carregado, when o Status é exibido, then aparece como pílula (`rounded-full`), em 14px, com o rótulo de `rotuloDoStatus`, sem ícone e nunca com o identificador cru de um dos sete.
- Given um Pedido que a consulta de 10 s leva de `ENVIADO` a `ENTREGUE`, when a tela relê, then o selo passa de `secondary` a verde dentro da mesma região viva, que anuncia "Status: Entregue".
- Given a Tabela com um Pedido `CANCELADO` sinalizado, when a linha é exibida, then a célula mostra o selo `destructive` e, depois dele, o marcador `outline` "Pagamento aprovado".
- Given `npm test`, when roda, then os testes de tom e a guarda de fonte passam, e a guarda cai se uma das quatro telas voltar ao rótulo solto.

## Design Notes

O tom mora em `lib/`, e as classes moram no componente. Assim a regra "verde só em `ENTREGUE`" é testada como dado, e a guarda de fonte prende o único arquivo que pode pintar de verde.

**Emenda da revisão (patch, 2026-09-24):** as classes saíram do componente para `aparenciaDoSelo(status)` em `web/lib/pedido.ts`, que devolve `{ variant, className }` e é testada sobre os sete Status e os desconhecidos. Com as classes no componente, trocar `falha` por `entregue`, tirar o `text-sm` ou voltar ao `destructive` tingido passava com a suíte verde. O `SeloDoStatus` só repassa o resultado. O dono único do verde de `ENTREGUE` passou a ser `lib/pedido.ts`, e a guarda de fonte foi alargada para `border-`, `ring-`, `var(--available)` e `#007600`.

```tsx
const classes: Record<Tom, string> = {
  progresso: "",
  falha: "bg-destructive text-white",
  entregue: "bg-available text-available-foreground",
};
<Badge variant={tom === "progresso" ? "secondary" : "default"}
       className={cn("h-auto rounded-full text-sm", classes[tom])}>
```

## Verification

**Commands:**
- `cd web && npm test` -- expected: todos os testes passam, os novos inclusos.
- `docker build ./web` -- expected: verde (`prebuild` roda `verificar-offline.mjs` e `npm test`, e o `next build` confere os tipos; o `tsc` local não resolve o módulo).

**Manual checks (if no CLI):**
- A 360 e a 1440 px, e só pelo teclado: Meus pedidos com um Pedido de cada tom, o Detalhe chegando a "Entregue" em verde, e `/admin/pedidos?status=CANCELADO` com o selo vermelho e o marcador ao lado. Nenhuma tela rola na horizontal.

## Suggested Review Order

**A regra de cor como dado**

- Ponto de entrada: tom e classes numa função pura, testada, dona única do verde.
  [`pedido.ts:361`](../../web/lib/pedido.ts#L361)

- Verde por comparação isolada; o mapa só conhece as duas falhas.
  [`pedido.ts:331`](../../web/lib/pedido.ts#L331)

- O componente só repassa `variant` e `className` ao `Badge` intocado.
  [`selo-do-status.tsx:11`](../../web/components/selo-do-status.tsx#L11)

**As quatro telas**

- Selo dentro da região viva que já existia; o anúncio continua "Status: …".
  [`acompanhamento.tsx:426`](../../web/app/pedidos/%5Bid%5D/acompanhamento.tsx#L426)

- Mesmo padrão no Detalhe administrativo.
  [`detalhe-admin.tsx:129`](../../web/app/admin/pedidos/%5Bid%5D/detalhe-admin.tsx#L129)

- Na célula, o selo vem antes do marcador "Pagamento aprovado" da 6.5.
  [`pedidos.tsx:254`](../../web/app/admin/pedidos/pedidos.tsx#L254)

- Dentro do `<a>` da linha; o sublinhado não desce ao `inline-flex`.
  [`meus-pedidos.tsx:149`](../../web/app/pedidos/meus-pedidos.tsx#L149)

**Testes e guardas**

- Aparência sobre os sete e os desconhecidos: pílula, 14px, falha preenchida.
  [`pedido.test.mjs:241`](../../web/scripts/pedido.test.mjs#L241)

- Guarda de fonte: nenhuma tela volta ao rótulo solto.
  [`pedido.test.mjs:287`](../../web/scripts/pedido.test.mjs#L287)

- Guarda do verde: `ENTREGUE` só em `lib/pedido.ts`, Estoque só nas suas telas.
  [`pedido.test.mjs:339`](../../web/scripts/pedido.test.mjs#L339)

**Registro**

- A entrada da 1.2 fechada por apêndice, e a tensão da espinha adiada.
  [`deferred-work.md:502`](deferred-work.md#L502)
