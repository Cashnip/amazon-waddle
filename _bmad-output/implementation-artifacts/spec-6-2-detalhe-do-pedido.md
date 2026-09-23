---
title: 'Estória 6.2 — Detalhe do Pedido'
type: 'feature'
created: '2026-09-23'
status: 'done'
baseline_commit: '7b9f869781f99c324e19aacfc0119a403bc15c44'
review_loop_iteration: 0
context: ['{project-root}/_bmad-output/implementation-artifacts/epic-6-context.md']
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A tela `/pedidos/{id}` já recebe do Go, numa chamada só, tudo o que a FR-30 pede — Itens com preço praticado, Endereço, Frete, total e o `historico` inteiro —, e já consulta a cada 10 s até `terminal`. Mas o histórico nunca aparece: o Comprador não vê por onde o Pedido passou, e o clímax da UJ-2 (a linha do tempo ganhando `ENVIADO` e `ENTREGUE` na frente dele) não existe. Quando o Pedido já saiu para entrega, nada diz que não dá mais para cancelar.

**Approach:** Só `web/`. A linha do tempo entra no Detalhe, uma linha por transição registrada, em ordem cronológica, com o estado novo, o anterior, o motivo quando houver e data e hora. Em `ENVIADO` e `ENTREGUE` uma frase explica por que não há cancelamento. O mapa de rótulos local de `acompanhamento.tsx` dá lugar a `rotuloDoStatus`.

## Boundaries & Constraints

**Always:**
- A linha do tempo é o `historico` da resposta, na ordem em que vem (`ORDER BY ocorrido_em, id`); nenhuma etapa futura, nenhum passo esperado.
- Rótulo por `rotuloDoStatus`; frase do motivo pelo mesmo mapa que `motivoDaRecusa` já usa — um ponto só por texto.
- Instante em `<time dateTime>` com o valor RFC 3339 do servidor, formatado em pt-BR pelo `Instante` já existente.
- Forma do DESIGN.md: lista vertical em `rounded-md`, marcador de 8 px na cor `border`, filete de 1 px ligando as linhas, a última sem filete adiante. Sem verde, sem laranja, sem ícone.
- Funções novas são puras em `web/lib/pedido.ts`, com teste em `web/scripts/pedido.test.mjs`.

**Ask First:**
- Qualquer mudança no Go, no SQL ou no contrato de `GET /api/v1/pedidos/{id}`.

**Never:**
- Botão "Cancelar Pedido", `Dialog`, rota de cancelamento — são da 6.3; aqui a tela não promete o que não tem.
- Selo `Badge` com cor por Status — é da 6.7.
- Data "Feito em" no cabeçalho: o nascimento já é a primeira linha da linha do tempo, e trazê-lo exigiria tocar `Detalhar`.
- Mostrar o ator ao Comprador; nomear a Reserva de Estoque.
- Tocar `web/components/ui/*`, `internal/`, `api/`.

**Decisão — o que da UX-DR11 cabe na 6.2.** Sem rota de cancelamento, o botão não existe; a 6.2 entrega só o lado "ausente com frase": em `ENVIADO` e `ENTREGUE` a frase aparece. Nos quatro estados canceláveis a tela não diz nada — a 6.3 põe o botão no lugar, e o "desabilitado com progresso" é dela.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Caminho feliz | `ENTREGUE`, 5 transições | 5 linhas, de "Aguardando pagamento" a "Entregue", cada uma com data e hora; frase "Este Pedido já foi entregue e não pode mais ser cancelado." | N/A |
| Nascimento | linha com `de: null` | só o estado novo, sem "antes" | N/A |
| Recusa e nova Tentativa | `…→ PAGAMENTO_RECUSADO (TEMPO_ESGOTADO) → AGUARDANDO_PAGAMENTO` | a linha recusada traz "Tempo de pagamento expirado."; a nova Tentativa é mais uma linha | N/A |
| Motivo desconhecido | `motivo: "OUTRO"` ou `"constructor"` | a linha sai sem frase de motivo | N/A |
| Status desconhecido | `para: "NOVO"` | a linha mostra `NOVO` cru | N/A |
| Enviado | `ENVIADO` | "Este Pedido já saiu para entrega e não pode mais ser cancelado." | N/A |
| Cancelável ou cancelado | `PAGO`, `SEPARANDO`, `CANCELADO`… | nenhuma frase de cancelamento | N/A |
| Mudança durante a consulta | 10 s trazem `ENVIADO` | a linha nova aparece; a região `role="status"` anuncia o Status | N/A |

</frozen-after-approval>

## Code Map

- `web/app/pedidos/[id]/acompanhamento.tsx:43` -- mapa `rotulo` local: apagar, usar `rotuloDoStatus` (uso em :315).
- `web/app/pedidos/[id]/acompanhamento.tsx:56` -- `Instante`: reusar nas linhas da linha do tempo.
- `web/app/pedidos/[id]/acompanhamento.tsx:105` -- `Detalhe`: recebe a linha do tempo (Card "Histórico", depois dos Itens, como no mockup) e a frase de cancelamento (junto de "Valores", onde a 6.3 porá o botão). O comentário de :102 anuncia exatamente isto.
- `web/app/pedidos/[id]/acompanhamento.tsx:312` -- região `role="status"` que já anuncia o Status: é o `aria-live` da UX-DR14; não duplicar.
- `web/lib/pedido.ts:97,105` -- `frasesDoMotivo` e `motivoDaRecusa`: extrair a busca de uma frase (com `hasOwn`) para função exportada, e `motivoDaRecusa` passa a usá-la sem mudar o comportamento.
- `web/lib/pedido.ts:198` -- `rotuloDoStatus`: reusar.
- `web/scripts/pedido.test.mjs` -- testes de `motivoDaRecusa` (:71–95) e `rotuloDoStatus` (:174) ficam intactos; acrescentar.
- `api/pedido.go:106` -- `saidaTransicao` (`de` null no nascimento, `motivo` null sem motivo): leitura, não tocar.
- `api/autorizacao_test.go:66` -- "Pedido por GET" já prova o NFR-6 por igualdade com o inexistente: a 6.2 não acrescenta teste Go.
- `_bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/mockups/detalhe-do-pedido.html:128` -- a linha do tempo ilustrada (ilustra; DESIGN.md:190 e EXPERIENCE.md:130 vencem).

## Tasks & Acceptance

**Execution:**
- [x] `web/lib/pedido.ts` -- exportar `fraseDoMotivo(motivo)` (string ou `null`) e fazer `motivoDaRecusa` usá-la; acrescentar `porQueNaoCancela(status)` (frase em `ENVIADO`/`ENTREGUE`, `null` no resto) -- texto num ponto só, testável.
- [x] `web/scripts/pedido.test.mjs` -- cobrir as linhas da matriz: nascimento, motivo conhecido/desconhecido/`constructor`, Status desconhecido pelo rótulo, as frases de cancelamento nos sete Status -- é a verificação da estória.
- [x] `web/app/pedidos/[id]/acompanhamento.tsx` -- `LinhaDoTempo` como `<ol>` dentro do `Detalhe`; frase de cancelamento; trocar `rotulo` por `rotuloDoStatus` -- FR-30, NFR-9, UX-DR11.

**Acceptance Criteria:**
- Dado um Pedido do Comprador fora de `AGUARDANDO_PAGAMENTO`, quando o Detalhe abre, então Itens com preço praticado, Endereço, Frete, total e a linha do tempo aparecem juntos, de uma única resposta.
- Dado o Detalhe aberto num Pedido não terminal, quando a varredura o avança, então a linha do tempo ganha a transição sem recarregar e o Status é anunciado.
- Dado 360 px de largura, quando o Detalhe abre, então não há rolagem horizontal e a linha do tempo se lê inteira pelo teclado e pelo leitor de tela.

## Design Notes

Cada `<li>` leva o marcador (`size-2 rounded-full border`) e o filete (`border-l` em pseudo-elemento ou `span` absoluto) só quando não é o último — o que fecha `ENTREGUE` e `CANCELADO` sem filete adiante sem caso especial. O anterior entra como texto secundário ("antes: Pago"), porque a EXPERIENCE pede anterior e novo e a nova Tentativa é justamente a linha em que o anterior não é o óbvio.

## Verification

**Commands:**
- `cd web && npm test` -- esperado: tudo passa, inclusive os testes novos de `pedido.test.mjs`.
- `docker build ./web` -- esperado: build completo (`prebuild` roda `verificar-offline.mjs` e `npm test`).

**Manual checks (if no CLI):**
- Com `docker compose up`, um total `,00` vai a `PAGO` e segue pela simulação: a linha do tempo cresce a cada 10 s até "Entregue" e a frase de entregue aparece; a 360 px sem rolagem horizontal.

## Review Triage Log

Passe 1 — uma lente (`edge-case-hunter`), pela política de revisão do projeto: estória só de tela, sem concorrência, máquina de estados nem dinheiro.

| # | Achado | Veredito | Evidência | Rota |
|---|--------|----------|-----------|------|
| 1 | Linha em `PAGAMENTO_RECUSADO` com motivo nulo ou desconhecido sai sem frase, enquanto a região de Status mostra a genérica | low | A matriz congelada decide o contrário ("Motivo desconhecido → a linha sai sem frase de motivo"), e a genérica já está na região de Status logo acima | rejeitado |

## Suggested Review Order

**A linha do tempo**

- Entrada: o histórico vira linhas puras — rótulo, anterior, frase do motivo; o ator não sai.
  [`pedido.ts:126`](../../web/lib/pedido.ts#L126)

- Marcador e filete do DESIGN.md; a última linha sem filete fecha os terminais sem caso especial.
  [`acompanhamento.tsx:98`](../../web/app/pedidos/[id]/acompanhamento.tsx#L98)

- Card "Histórico" depois dos Itens, como no mockup.
  [`acompanhamento.tsx:152`](../../web/app/pedidos/[id]/acompanhamento.tsx#L152)

**Texto num ponto só**

- A frase do motivo extraída; `motivoDaRecusa` a reusa sem mudar comportamento.
  [`pedido.ts:105`](../../web/lib/pedido.ts#L105)

- O mapa local de rótulos saiu; a região viva usa `rotuloDoStatus`.
  [`acompanhamento.tsx:347`](../../web/app/pedidos/[id]/acompanhamento.tsx#L347)

**Ausente com frase (UX-DR11, metade da 6.2)**

- Frase só em `ENVIADO`/`ENTREGUE`; nos canceláveis a 6.3 põe o botão.
  [`pedido.ts:143`](../../web/lib/pedido.ts#L143)

- A frase mora junto dos valores, onde o botão da 6.3 vai morar.
  [`acompanhamento.tsx:199`](../../web/app/pedidos/[id]/acompanhamento.tsx#L199)

**Testes**

- Nascimento, recusa com nova Tentativa, motivo e Status desconhecidos.
  [`pedido.test.mjs:190`](../../web/scripts/pedido.test.mjs#L190)

- A frase de cancelamento nos sete Status e em `constructor`.
  [`pedido.test.mjs:216`](../../web/scripts/pedido.test.mjs#L216)
