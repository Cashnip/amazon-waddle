---
title: 'Estória 5.4 — Revisão do Pedido'
type: 'feature'
created: '2026-09-19'
status: 'done'
baseline_commit: '47073e696c52f201cf1973df4f6864a3ba62638a'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A Revisão é o esboço da 5.2 com o bloco de valores da 5.3: não mostra os Itens com preço unitário e parcela, não declara a forma de pagamento, não tem Confirmar Pedido, deixa um Carrinho vazio chegar, e não existe a `Idempotency-Key` que faz o duplo clique criar **um** Pedido (FR-22, NFR-12).

**Approach:** Substituir o esboço pela Revisão de verdade — Itens do `GET /api/v1/carrinho` com preço unitário e parcela, Endereço escolhido, o bloco Subtotal/Frete/Total vindo pronto do Go, a declaração de pagamento sem campo nenhum, e o botão Confirmar Pedido em `{colors.primary-strong}`. A chave de idempotência nasce ao **entrar** na Revisão e fica em `sessionStorage`, para sobreviver ao recarregamento que ela existe para proteger; quem a envia, e quem apaga o checkout guardado, é a 5.6.

## Boundaries & Constraints

**Always:**
- Nenhuma regra no Node (AD-10): Itens, subtotal, Frete e total vêm do Go. A tela **não soma o total** (NFR-13) — ele é o `total_centavos` da cotação; a parcela da linha é a multiplicação que `parcelaDe` já faz para o Carrinho.
- A cotação é perguntada **a cada abertura** e nada dela é guardado no navegador — nem Frete, nem CEP, nem total. É isso que faz trocar o Endereço recalcular o Frete (FR-20/FR-22).
- Carrinho vazio **nunca** chega à Revisão; voltar ao Carrinho e retornar não perde o Endereço escolhido.
- A `Idempotency-Key` é **uma por tentativa de checkout**: nasce na entrada da Revisão, é reaproveitada no recarregamento e só é apagada na criação do Pedido. Ela é estado de tela, não de domínio.
- `sessionStorage` sempre em `try/catch`: indisponível equivale a nada guardado, e aí a tela avisa em vez de prometer uma confirmação que não seria protegida.
- O laranja aparece **uma vez no fluxo inteiro**, no Confirmar Pedido, em pill (UX-DR7, `{rounded.full}`). Nenhum verde.
- Nada de cartão: nenhum campo, rótulo, máscara ou *placeholder* (§8) — a declaração é uma frase. Nada de rede externa (NFR-15): a chave sai do `crypto` do navegador.
- Glossário literal — Pedido, Produto, Carrinho, Endereço, Frete, Provedor de Pagamento. Piso AA, pelo teclado, de 360 a 1440 px.

**Ask First:**
- Qualquer mudança em Go, rota HTTP nova ou contrato JSON de `/api/v1/carrinho`, `/api/v1/enderecos` ou `/api/v1/frete`.
- Mudar a chave `azamon:checkout:endereco_id` da 5.2, ou o significado do que o navegador guarda.

**Never:**
- **Criar o Pedido.** `POST /api/v1/pedidos` ainda é o do esqueleto (`{produto_id}`, uma unidade, sem Endereço nem Frete): é a 5.6 que o refaz com chave, digest do corpo, congelamento e `TOTAL_DIVERGENTE`. Aqui o Confirmar Pedido fica **desabilitado**, com a razão em uma linha ao lado.
- Revalidação de Estoque e preço na entrada do checkout, e `carrinho.ConfirmarPrecoVisto` (5.5). A Revisão não repete os `Alert` do Carrinho.
- Apagar a escolha e a chave: `limparCheckout` nasce aqui e é testada, mas **quem a chama é a 5.6**.
- Calcular Frete ou total no navegador; consultar CEP; alterar quantidade, remover Item ou editar Endereço dentro da Revisão.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Revisão completa | 2 Itens, Endereço escolhido | cada linha com preço unitário e parcela; Endereço; Subtotal/Frete/Total do Go; declaração de pagamento; Confirmar Pedido | — |
| Primeira entrada | nada guardado | gera a chave e a guarda | — |
| Recarregar | chave já guardada | **a mesma** chave continua; nenhuma nova é gerada | — |
| Trocar o Endereço | volta ao passo Endereço e retorna | Frete e total mudam; a chave **não** muda | — |
| Carrinho vazio | `GET /api/v1/carrinho` sem Itens | volta a `/carrinho` sem piscar a Revisão | — |
| Escolha inválida | nada guardado, `id` fora da lista, ou 404 na cotação | volta a `/checkout/endereco` | — |
| `sessionStorage` indisponível | `getItem`/`setItem` lançam | a Revisão abre e mostra tudo; Confirmar Pedido indisponível, com aviso | mensagem, sem quebra |
| Subtotal do Carrinho ≠ o da cotação | preço mudou entre as duas leituras | o bloco de valores é o do Go, e a divergência não é decidida aqui | — |
| Sem Sessão | 401 em qualquer leitura | Login com `destino` = `/checkout/revisao` | — |
| Falha de rede | `fetch` lança | `FALHA_DE_REDE` em `Alert` | — |

</frozen-after-approval>

## Code Map

- `web/app/checkout/revisao/esboco-da-revisao.tsx` -- **substituído**. Reaproveitar: leitura da escolha e volta ao passo Endereço (33–56), o 404 da cotação (65–70) e o bloco de valores (115–141). Apagar o arquivo.
- `web/app/checkout/revisao/page.tsx:8` -- Server Component que não busca nada (AD-10): trocar o filho e o comentário do esboço.
- `web/lib/checkout.ts:11` -- `CHAVE_DO_ENDERECO`, `lerEscolha`/`guardarEscolha`, `enderecoEscolhido:59`, `rotaDoFrete:76`, `freteGratis:82`, tipo `Cotacao:67`. O tipo `Armazenamento:13` é `Pick<Storage, "getItem" | "setItem">` -- acrescentar `removeItem`.
- `web/lib/carrinho.ts:12` -- `LinhaDoCarrinho`, `Carrinho:29`, `parcelaDe:53` (a parcela da linha, que cala a invisível), `unidadesTexto:123`. Só leitura.
- `web/app/checkout/endereco/escolha-de-endereco.tsx:86` -- molde da guarda de Carrinho vazio e do 401 em leitura paralela.
- `web/app/carrinho/meu-carrinho.tsx:357` -- molde da linha: `ImagemDoProduto` compacta, "R$ X cada" (`:362`) e a parcela à direita (`:399`).
- `web/app/checkout/etapas.tsx:8` -- `EtapasDoCheckout atual="Revisão"`, por `aria-current`.
- `web/components/formulario-de-endereco.tsx` -- `EnderecoPorExtenso`, já usado pelo esboço.
- `web/app/globals.css:109` -- `--primary-strong` / `--primary-strong-foreground`; `bg-primary-strong text-primary-strong-foreground rounded-full` é o botão do `DESIGN.md:180`.
- `web/lib/preco.ts` -- `formatarPreco`. `web/lib/destino.ts` -- `paraLogin()`. `web/lib/pedir.ts` -- `pedir`, `FALHA_DE_REDE`.
- `web/scripts/checkout.test.mjs:22` -- molde do `node --test`: armazenamento em memória e o que lança.
- `api/pedido.go:17` -- `entradaPedido{ProdutoID}`: a prova de que o POST ainda é o do esqueleto. Só leitura; nenhum `.go` muda nesta estória.

## Tasks & Acceptance

**Execution:**
- [x] `web/lib/checkout.ts` -- `Armazenamento` ganha `removeItem`; `CHAVE_DE_IDEMPOTENCIA = "azamon:checkout:idempotency_key"`; `uuidNovo` (UUIDv4 de `crypto.randomUUID`, caindo para `crypto.getRandomValues` -- contexto não seguro, como um IP da LAN na apresentação, não tem `randomUUID`); `chaveDeIdempotencia` (devolve a guardada, senão gera e guarda, senão `null`); `limparCheckout` (apaga as duas chaves, para a 5.6 chamar).
- [x] `web/app/checkout/revisao/revisao-do-pedido.tsx` (novo) -- a Revisão: Itens com preço unitário e parcela, Endereço, valores do Go, declaração de pagamento e Confirmar Pedido. Apagar `esboco-da-revisao.tsx`.
- [x] `web/app/checkout/revisao/page.tsx` -- apontar para o componente novo; o comentário deixa de prometer a 5.4.
- [x] `web/scripts/checkout.test.mjs` -- as linhas da matriz que são regra: gera na primeira entrada, reaproveita no recarregamento, armazenamento que lança devolve `null`, `limparCheckout` apaga as duas chaves, e a forma do UUID.
- [x] `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` §10 + `uv run _bmad/scripts/memlog.py append` -- a decisão: a chave nasce na entrada da Revisão, sobrevive ao recarregamento, só é apagada na criação do Pedido, e tem queda de geração por contexto não seguro.
- [x] `_bmad-output/implementation-artifacts/deferred-work.md` -- Confirmar Pedido desabilitado até a 5.6 (que envia a chave e chama `limparCheckout`); a divergência entre as duas leituras, que a 5.6 fecha com `TOTAL_DIVERGENTE`; e o teste de tela da troca de Endereço, que segue sem bancada no `web/`.

**Acceptance Criteria:**
- Given um Carrinho com dois Produtos de quantidades diferentes, when a Revisão abre, then cada linha mostra o preço unitário e a parcela, e a soma das parcelas é o Subtotal que o Go devolveu.
- Given a Revisão aberta, when o Comprador recarrega a página, then a `Idempotency-Key` guardada continua a mesma.
- Given a Revisão aberta, when o Comprador volta ao Carrinho e retorna, then o Endereço e a chave continuam os mesmos, e o Frete é perguntado de novo.
- Given a tela inteira, when se procura laranja, then ele existe só no Confirmar Pedido — e nenhum campo de cartão existe em lugar nenhum.

## Spec Change Log

## Design Notes

A chave nasce na **entrada** da Revisão, e não no clique: gerada no clique, o duplo clique produziria duas chaves e dois Pedidos, e a idempotência do servidor estaria correta e inútil. Guardá-la em `sessionStorage` é o que a faz sobreviver ao recarregamento — o caso que ela existe para proteger. É estado de tela: o Node continua sem estado de domínio (AD-10).

O bloco de valores é o do Go inteiro, e a parcela da linha é só apresentação. Quando as duas leituras discordam, porque um preço mudou entre elas, a Revisão não arbitra: quem recusa é a criação do Pedido, sob a trava, com `TOTAL_DIVERGENTE` (5.6).

Quem implementa não faz `git commit`, `git add` nem `push`.

## Verification

**Commands:**
- `docker build ./web` -- expected: verde (roda `verificar-offline` e `npm test`, incluindo `checkout.test.mjs`).
- `git diff --stat -- '*.go' 'db/*'` -- expected: vazio.

**Manual checks (if no CLI):**
- A 360 e a 1440 px, só pelo teclado: Carrinho com dois Produtos → Revisão; conferir preço unitário, parcela e o total; trocar o Endereço de SP para a BA e ver Frete e total mudarem; recarregar e conferir que `azamon:checkout:idempotency_key` não mudou; esvaziar o Carrinho noutra aba e reabrir a Revisão, que volta a `/carrinho`.

## Review Triage Log

Duas lentes (blind, edge-case) sobre o diff, e não três: o HANDOFF reserva as três para raio de alcance real (5.1, 5.6, 5.7, 5.9), e a 5.4 não toca Go, SQL nem concorrência. Decidido com o humano. Nenhum `intent_gap` nem `bad_spec`.

Patches aplicados: promoção de glossário no título do cartão ("Itens do Pedido" → **"Itens do Carrinho"** — na Revisão nenhum Pedido existe, e a FR-22 diz Item de Carrinho); `comChave` virou load-bearing, com `CRIACAO_DISPONIVEL` como a única linha que a 5.6 vira e a guarda de armazenamento travando o botão por conta própria; a descrição acessível passou a nomear as razões que valem, em vez de contradizer o `Alert` da tela; `chaveDeIdempotencia` valida a forma do que estava guardado antes de reaproveitar, porque na 5.6 a string vira cabeçalho HTTP; e três correções nos testes (o `try` por chave de `limparCheckout` agora é provado nos dois sentidos, um comentário que prometia três navegações foi estreitado, e `assert.match` sobre `string | null` ganhou a asserção de não-nulo).

Adiados: a chave sobrevivendo a uma alteração do Carrinho entre duas passagens pela Revisão (a 5.6 decide, senão o Comprador legítimo leva 409); e o Carrinho só de Produtos invisíveis, que não é vazio e chega à Revisão (a 5.5 fecha, com a revalidação na entrada).

Rejeitados: a chave gerada sem quem a envie e `limparCheckout` sem chamador (é a decisão da spec, e a 5.6 é a chamadora); o laranja gasto num botão desabilitado (decisão humana no CHECKPOINT 1); `role="alert"` no `Alert` (o componente do shadcn já o traz); guardas de forma sobre o JSON das nossas próprias rotas; `crypto` global sem guarda (só alcançável fora do `try`, e o `web/` não tem SSR desta tela); `<h2>` dentro de `CardTitle` e âncora em vez do router (padrões anteriores a esta estória); e a ausência de teste de componente, já registrada no `deferred-work.md`.

## Suggested Review Order

**A chave de idempotência: a regra, em `lib/`**

- Porta de entrada: a chave é uma por tentativa de checkout, e o recarregamento a reaproveita.
  [`checkout.ts:139`](../../web/lib/checkout.ts#L139)

- A forma é conferida antes de reaproveitar: na 5.6 a string vira cabeçalho HTTP.
  [`checkout.ts:124`](../../web/lib/checkout.ts#L124)

- UUIDv4 sem rede e sem biblioteca, com queda para contexto não seguro — o IP da LAN na apresentação.
  [`checkout.ts:108`](../../web/lib/checkout.ts#L108)

- `randomUUID` é opcional de propósito: é o que a queda existe para cobrir.
  [`checkout.ts:100`](../../web/lib/checkout.ts#L100)

- O fim da tentativa, com uma chave por `try`. Nasce aqui; quem a chama é a 5.6.
  [`checkout.ts:159`](../../web/lib/checkout.ts#L159)

**A tela: o que a 5.6 encosta, e o que ela não pode derrubar**

- A única linha que a 5.6 vira para ligar a confirmação.
  [`revisao-do-pedido.tsx:51`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L51)

- As razões acumulam: ligar a criação não alcança a guarda de armazenamento.
  [`revisao-do-pedido.tsx:158`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L158)

- A chave nasce na entrada da Revisão, antes de qualquer leitura — nunca no clique.
  [`revisao-do-pedido.tsx:79`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L79)

**Os três desvios que impedem uma Revisão sem sentido**

- Carrinho vazio nunca chega à Revisão.
  [`revisao-do-pedido.tsx:105`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L105)

- Sem escolha válida, o passo volta ao Endereço — não há "o primeiro" de consolação.
  [`revisao-do-pedido.tsx:117`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L117)

- Endereço removido entre a lista e a cotação: mesmo desvio, e não um beco sem saída.
  [`revisao-do-pedido.tsx:131`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L131)

**O que a FR-22 exige na tela**

- Itens de Carrinho, e não de Pedido: na Revisão nada foi criado ainda.
  [`revisao-do-pedido.tsx:185`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L185)

- Preço unitário à esquerda, parcela à direita; a parcela é só apresentação.
  [`revisao-do-pedido.tsx:321`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L321)

- O total é o do Go: a tela não soma (NFR-13).
  [`revisao-do-pedido.tsx:238`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L238)

- A forma de pagamento declarada numa frase, sem um campo sequer (§8).
  [`revisao-do-pedido.tsx:255`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L255)

**Testes**

- Entrar de novo não troca a chave: é o que faz o recarregamento ser seguro.
  [`checkout.test.mjs:160`](../../web/scripts/checkout.test.mjs#L160)

- Lixo guardado é descartado e regerado, em onze formas de lixo.
  [`checkout.test.mjs:254`](../../web/scripts/checkout.test.mjs#L254)

- A chave que lança não leva a outra junto, nos dois sentidos.
  [`checkout.test.mjs:217`](../../web/scripts/checkout.test.mjs#L217)

- Sem armazenamento não há chave, e a tela avisa em vez de prometer.
  [`checkout.test.mjs:172`](../../web/scripts/checkout.test.mjs#L172)

- A queda do contexto não seguro monta o UUIDv4 à mão.
  [`checkout.test.mjs:236`](../../web/scripts/checkout.test.mjs#L236)
