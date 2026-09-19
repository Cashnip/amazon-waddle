---
title: 'Estória 5.2 — Seleção de Endereço no checkout'
type: 'feature'
created: '2026-09-19'
status: 'done'
baseline_commit: '876ff56266ffb138a768a5a23e8c4c883be8362b'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O Carrinho não tem saída: não há checkout, e o Comprador não tem onde escolher para qual Endereço vai o Pedido — nem cadastrar o primeiro sem sair do fluxo (FR-20).

**Approach:** Criar o primeiro dos dois passos do checkout (Endereço → Revisão) como rota própria, `/checkout/endereco`, com a lista dos Endereços do Comprador para escolher e a criação em `Dialog`; o Comprador sem Endereço cai direto no formulário. O navegador guarda só o `id` escolhido, em `sessionStorage`. `/checkout/revisao` nasce como esboço que só mostra o Endereço escolhido — a 5.4 o substitui. O Carrinho ganha o botão de entrada.

## Boundaries & Constraints

**Always:**
- Nenhuma regra no Node (AD-10): lista e criação usam as rotas existentes de `/api/v1/enderecos`; a mensagem de erro exibida é a do envelope.
- O navegador guarda **só** `endereco_id` — nunca Frete, CEP ou total. É isso que garante que trocar o Endereço recalcula o Frete: a Revisão sempre perguntará ao Go.
- Duas rotas em toda largura: "um passo por tela" de 360 a 639 px vale por construção (UX-DR15). Sem rolagem horizontal.
- O formulário de Endereço é um só componente, usado por "Meus endereços" e pelo checkout. `Dialog` de um nível; `Esc` fecha.
- Glossário literal: Endereço, Pedido, Carrinho, Produto. Nenhum verde nem laranja (o laranja é da 5.4).
- `sessionStorage` sempre dentro de `try/catch`: indisponível equivale a nada guardado.

**Ask First:**
- Qualquer mudança em Go, rota HTTP nova ou contrato JSON de `/api/v1/enderecos` ou `/api/v1/carrinho`.

**Never:**
- Frete (5.3), a Revisão de verdade e a `Idempotency-Key` (5.4), revalidação no servidor e `ConfirmarPrecoVisto` (5.5), criação do Pedido (5.6).
- Editar ou remover Endereço dentro do checkout — isso é "Meus endereços".
- Endereço "padrão" persistido no banco.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Com Endereços | 2 Endereços, nada guardado | lista com o primeiro marcado; "Continuar" leva à Revisão | — |
| Escolha guardada | `id` guardado presente na lista | ele vem marcado, inclusive ao voltar do Carrinho | — |
| Guardado sumiu | `id` guardado removido em "Meus endereços" | primeiro da lista marcado | — |
| Sem Endereço | lista vazia | formulário aberto direto, sem lista vazia; ao salvar, o novo fica marcado | 400 no campo, com foco |
| Novo no Dialog | Comprador com Endereços cadastra | Dialog fecha, lista recarrega, o novo fica marcado | 409 do teto no alerta do Dialog |
| Carrinho vazio | `GET /api/v1/carrinho` sem itens | volta a `/carrinho` | — |
| Sem Sessão | 401 em qualquer leitura | Login com `destino` = passo atual | — |
| Revisão sem escolha | nada guardado, ou `id` fora da lista | volta a `/checkout/endereco` | — |

</frozen-after-approval>

## Code Map

- `web/app/enderecos/meus-enderecos.tsx` -- formulário, `CAMPOS`, `VAZIO`, `comMascara`, refs, erro por `dados.campo` e foco (UX-DR16/20c). Fonte da extração.
- `web/app/carrinho/meu-carrinho.tsx:290` -- rodapé com subtotal; `revalidacaoDe(...).podeAvancar` já calculado acima (linha ~200).
- `web/lib/carrinho.ts` -- tipo `Carrinho`; `revalidacaoDe`/`podeAvancar`.
- `web/lib/destino.ts` -- `paraLogin()` usa o caminho atual: o 401 volta ao passo certo.
- `web/lib/pedir.ts` -- `pedir`, `FALHA_DE_REDE`.
- `web/app/carrinho/page.tsx` -- molde da página: `Casca` + filho `"use client"`, sem busca no Server Component.
- `web/components/ui/radio-group.tsx`, `dialog.tsx` -- já instalados.
- `web/scripts/carrinho.test.mjs` -- molde do `node --test` importando `.ts` de `lib/`.
- `api/rotas.go:66-69` -- as quatro rotas de Endereço. Só leitura.

## Tasks & Acceptance

**Execution:**
- [x] `web/lib/endereco.ts` (novo) -- tipos `Endereco`/`Campo`, `CAMPOS`, `VAZIO`, `comMascara` saem de `meus-enderecos.tsx`.
- [x] `web/lib/checkout.ts` (novo) -- chave do `sessionStorage`, `lerEscolha`/`guardarEscolha` com `try/catch`, e a regra pura `enderecoMarcado(lista, guardado)` (guardado se presente, senão o primeiro, senão `null`).
- [x] `web/components/formulario-de-endereco.tsx` (novo) -- o formulário com `POST`/`PUT`, erro em linha, 401 → Login e `aoSalvar(endereco)`.
- [x] `web/app/enderecos/meus-enderecos.tsx` -- usar o componente; comportamento inalterado.
- [x] `web/app/checkout/endereco/page.tsx` + `escolha-de-endereco.tsx` (novos) -- indicador "Endereço → Revisão", guarda do Carrinho vazio, `RadioGroup` com os Endereços, `Dialog` "Novo Endereço", formulário direto se vazio, "Continuar" guarda e vai à Revisão.
- [x] `web/app/checkout/revisao/page.tsx` + `esboco-da-revisao.tsx` (novos) -- valida a escolha contra a lista; mostra o Endereço, "Trocar o Endereço" e "Voltar ao Carrinho". Comentário: substituída pela 5.4.
- [x] `web/app/carrinho/meu-carrinho.tsx` -- botão "Fechar o Pedido" no rodapé, desabilitado sem `podeAvancar`, levando a `/checkout/endereco`.
- [x] `web/scripts/checkout.test.mjs` (novo) -- as linhas da matriz que são regra: marcado, guardado sumido, lista vazia, `sessionStorage` que lança.
- [x] `_bmad-output/implementation-artifacts/deferred-work.md` -- a consequência "trocar o Endereço recalcula o Frete" fica provada na 5.3/5.4.

**Acceptance Criteria:**
- Given o Carrinho com Produto e sem bloqueio, when o Comprador clica "Fechar o Pedido", then chega ao passo Endereço.
- Given o Endereço escolhido e a Revisão aberta, when o Comprador volta ao Carrinho e retorna ao checkout, then o mesmo Endereço continua marcado (FR-22).
- Given "Meus endereços", when cadastra, edita e remove, then o comportamento é o de antes da extração.

## Design Notes

"Fechar o Pedido" é o rótulo do botão do Carrinho: não é o passo irreversível, por isso sem laranja. A escolha fica em `sessionStorage` e não na URL porque tem de atravessar a ida ao Carrinho e a volta (FR-22), e o botão do Carrinho não tem como saber o `id`.

Quem implementa não faz `git commit`, `git add` nem `push`: o diff fica no working tree para a revisão.

## Verification

**Commands:**
- `docker build ./web` -- expected: verde (roda `verificar-offline` e `npm test`, incluindo `checkout.test.mjs`).
- `git diff --stat -- '*.go'` -- expected: vazio.

**Manual checks (if no CLI):**
- A 360 e a 1440 px: Comprador novo cai no formulário; Comprador com dois Endereços troca, cadastra no Dialog (`Esc` fecha), vai à Revisão, volta ao Carrinho e retorna com a escolha mantida — tudo pelo teclado.

## Suggested Review Order

**A escolha: só o `id`, e a regra pura que a marca**

- Porta de entrada: prioridade recém-cadastrado → guardado → primeiro, testável sem navegador.
  [`checkout.ts:45`](../../web/lib/checkout.ts#L45)

- A Revisão só aceita o guardado presente na lista; sem "primeiro" de consolação.
  [`checkout.ts:59`](../../web/lib/checkout.ts#L59)

- `sessionStorage` sempre em `try/catch`; indisponível equivale a nada guardado.
  [`checkout.ts:20`](../../web/lib/checkout.ts#L20)

**O passo Endereço**

- Carrinho e Endereços lidos juntos; Carrinho vazio volta ao Carrinho.
  [`escolha-de-endereco.tsx:86`](../../web/app/checkout/endereco/escolha-de-endereco.tsx#L86)

- Sem Endereço, o formulário abre direto, sem lista vazia (FR-20).
  [`escolha-de-endereco.tsx:129`](../../web/app/checkout/endereco/escolha-de-endereco.tsx#L129)

- `RadioGroup` em cartões; o selecionado ganha borda, sem verde nem laranja.
  [`escolha-de-endereco.tsx:148`](../../web/app/checkout/endereco/escolha-de-endereco.tsx#L148)

- "Continuar" guarda o `id` e para com aviso se o navegador não guardar.
  [`escolha-de-endereco.tsx:104`](../../web/app/checkout/endereco/escolha-de-endereco.tsx#L104)

- `Dialog` de um nível para o Endereço novo, que volta marcado.
  [`escolha-de-endereco.tsx:205`](../../web/app/checkout/endereco/escolha-de-endereco.tsx#L205)

**O formulário compartilhado**

- Um formulário só para "Meus endereços" e o checkout; estado nasce do Endereço recebido.
  [`formulario-de-endereco.tsx:54`](../../web/components/formulario-de-endereco.tsx#L54)

- `aoSalvar` só com `id` no corpo: é por ele que o checkout marca o novo.
  [`formulario-de-endereco.tsx:119`](../../web/components/formulario-de-endereco.tsx#L119)

- Rota/método e erro por campo viraram funções puras, testadas.
  [`endereco.ts:38`](../../web/lib/endereco.ts#L38)

- Regressão fechada: Endereço removido com o formulário aberto fecha o formulário.
  [`meus-enderecos.tsx:75`](../../web/app/enderecos/meus-enderecos.tsx#L75)

**Entrada e esboço da Revisão**

- "Fechar o Pedido" no Carrinho, desabilitado sem `podeAvancar`, sem laranja.
  [`meu-carrinho.tsx:293`](../../web/app/carrinho/meu-carrinho.tsx#L293)

- Esboço que a 5.4 substitui: confere o `id` guardado contra a lista do Go.
  [`esboco-da-revisao.tsx:1`](../../web/app/checkout/revisao/esboco-da-revisao.tsx#L1)

- Indicador "Endereço → Revisão" marcado por `aria-current`, não por cor.
  [`etapas.tsx:16`](../../web/app/checkout/etapas.tsx#L16)

**Testes**

- As linhas da matriz que são regra, e a prioridade do recém-cadastrado.
  [`checkout.test.mjs:36`](../../web/scripts/checkout.test.mjs#L36)

- POST/PUT, campo desconhecido e a máscara do CEP.
  [`endereco.test.mjs:16`](../../web/scripts/endereco.test.mjs#L16)
