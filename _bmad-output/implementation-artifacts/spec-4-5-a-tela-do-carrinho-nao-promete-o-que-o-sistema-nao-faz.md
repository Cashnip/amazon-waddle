---
title: 'Estória 4.5 — A tela do Carrinho não promete o que o sistema não faz'
type: 'feature'
created: '2026-09-19'
status: 'done'
route: 'dispatch'
baseline_commit: '23cbd30ea0fa006f739a3126c4791f05d3a526c8'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-4-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** a tela do Carrinho da 4.3/4.4 já mostra só o subtotal e já fala sempre em Estoque disponível, mas falta a única promessa que a `EXPERIENCE.md` manda fazer — a distância até o Frete grátis, em valor absoluto — e o limiar `AZAMON_FRETE_ISENCAO_CENTAVOS` está carregado na Config sem nenhum consumidor. Sem ela, a UX-DR17 fica meio cumprida por omissão, não por decisão.

**Approach:** o `GET /api/v1/carrinho` passa o limiar da configuração junto com o subtotal, e a tela escreve "Faltam R$ 42,00 para o Frete grátis." ao lado do subtotal, sem barra de progresso e sem nomear Frete em lugar nenhum da composição monetária. Junto, a varredura de vocabulário da UX-DR16 nas telas de Comprador corrige o que sobrou.

## Boundaries & Constraints

**Always:**
- A composição monetária continua **subtotal e nada mais** — a distância até a isenção é uma frase, não uma parcela, e não entra em nenhuma soma.
- A distância é aritmética de centavos inteiros sobre o subtotal exibido, no mesmo lugar em que o subtotal otimista é refeito (`web/lib/carrinho.ts`), para que a frase nunca discorde do número na tela durante a edição otimista. O **limiar** é da Config e vem do Go: nenhum número mora no Node.
- Glossário literal (UX-DR16): "Produto", nunca "item"; "Pedido", nunca "compra". A Reserva de Estoque não é nomeada, não há "preço garantido", "reservado para você" nem urgência fabricada.
- Valores em `tabular-nums`, formatados só no navegador.

**Decidido:**
- **Acima do limiar a tela cala.** Nenhuma frase quando `falta` é zero: a composição é subtotal e nada mais, e o Carrinho não afirma nada sobre o Frete antes do passo Endereço.
- **`comprar.tsx` é corrigido nesta estória.** "Confirmar compra" e as duas mensagens de erro passam a dizer "Pedido" — a condição de aceite fala de qualquer texto, e a tela é de Comprador mesmo sendo provisória.

**Never:** valor de Frete, total estimado, barra de progresso, contagem regressiva; `carrinho` (o módulo Go) calculando Frete; qualquer escrita nova a partir do `GET`; mexer na área administrativa, onde "Reserva" é vocabulário legítimo (FR-11).

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Abaixo do limiar | subtotal 25.700, limiar 29.900 | "Faltam R$ 42,00 para o Frete grátis." ao lado do subtotal | — |
| No limiar | subtotal 29.900 | nenhuma frase (`falta` = 0) | — |
| Acima do limiar | subtotal 35.000 | nenhuma frase | — |
| Falta de um centavo | subtotal 29.899 | "Faltam R$ 0,01 para o Frete grátis." | — |
| Durante a edição otimista | quantidade sobe e o subtotal otimista cruza o limiar | a frase some no mesmo instante em que o subtotal muda; se o servidor recusar, volta com a reversão | — |
| Linha indisponível | Produto invisível no Carrinho | não entra no subtotal, logo não conta para a distância | — |
| Carrinho vazio | sem Itens | tela vazia da 4.3, sem subtotal e sem a frase | — |
| API | `GET /api/v1/carrinho` | `frete_isencao_centavos` na raiz, igual à Config | — |

</frozen-after-approval>

## Code Map

- `internal/plataforma/config.go:39,109` -- `FreteIsencaoCentavos` (padrão 29900) já existe e não tem consumidor. Não mudar.
- `api/carrinho.go:53` -- `saidaCarrinho` ganha `FreteIsencaoCentavos int64` (`json:"frete_isencao_centavos"`); `lerCarrinho` (linha ~150) preenche de `s.cfg.FreteIsencaoCentavos`. Nenhuma outra rota muda.
- `api/sessao_test.go:145` -- a `plataforma.Config` dos testes; falta a chave nova (o zero valor faria o campo sair 0). Const nova ao lado de `carrinhoUnidadesMaxDeTeste`.
- `api/carrinho_test.go:132` -- `carrinhoNaTela` é onde a leitura do Carrinho já é conferida; a asserção do campo novo entra ali, sem subteste novo.
- `web/lib/carrinho.ts:31` -- `Carrinho` ganha `frete_isencao_centavos`. **`comQuantidade` (linha ~63) reconstrói o objeto e perderia o campo** — tem de repassá-lo. Entra `faltaParaFreteGratis(subtotal, isencao)`, vizinha de `subtotalDe`.
- `web/app/carrinho/meu-carrinho.tsx:~270` -- a linha do subtotal (`Subtotal: …`) ganha a frase abaixo. `formatarPreco` já vem importado de `@/lib/preco`.
- `web/scripts/carrinho.test.mjs:64` -- `carrinhoDe(...)` monta o Carrinho dos testes e precisa do campo novo; as asserções da distância entram depois das da revalidação.
- **Auditados e já conformes, não mexer:** `web/components/carrinho-na-barra.tsx` (unidades, `tabular-nums`, sem urgência), `web/app/produtos/[id]/caixa-de-compra.tsx` (Estoque disponível, sem Reserva), `web/lib/carrinho.ts:textoDoBloqueio`, `web/components/preco.tsx` e `web/app/globals.css:164` (`tabular-nums` nos dois papéis monetários).

## Tasks & Acceptance

**Execution:**
- [x] `api/carrinho.go` -- `frete_isencao_centavos` na raiz de `saidaCarrinho`, vindo de `s.cfg` -- o limiar é da configuração (AD-13) e nenhum número mora no Node (AD-10).
- [x] `api/sessao_test.go` + `api/carrinho_test.go` -- const de teste, chave na `Config` e a asserção do campo na leitura do Carrinho -- NFR-8: falha se o limiar deixar de sair.
- [x] `web/lib/carrinho.ts` -- campo no tipo, repasse em `comQuantidade` e `faltaParaFreteGratis` -- a frase e o subtotal têm de vir da mesma aritmética.
- [x] `web/scripts/carrinho.test.mjs` -- as linhas da matriz que são de `lib`: abaixo, no, acima do limiar, um centavo, e o subtotal otimista -- teste de `lib`, sem navegador.
- [x] `web/app/carrinho/meu-carrinho.tsx` -- a frase abaixo do subtotal, sem barra de progresso -- UX-DR17.
- [x] `web/app/produtos/[id]/comprar.tsx` -- "Confirmar o Pedido" e "Não foi possível confirmar o Pedido." nos três textos -- glossário literal (UX-DR16).
- [x] `_bmad-output/implementation-artifacts/sprint-status.yaml` -- 4-5 para `review` (falta o passeio no navegador).

**Acceptance Criteria:**
- Dado o Carrinho renderizado, quando a tela é inspecionada, então o único valor monetário composto é o subtotal, não há elemento de progresso, e a distância até o Frete grátis é texto em valor absoluto.
- Dado o código das telas públicas e de Comprador, quando varrido, então nenhuma palavra da família "Reserva", nenhum "preço garantido"/"reservado para você" e nenhuma urgência aparecem em texto visível, e "item"/"compra" não aparecem como substantivo de interface.
- Dado `AZAMON_FRETE_ISENCAO_CENTAVOS` trocado, quando o Carrinho abre, então a frase acompanha o valor novo sem recompilar o `web/`.

## Implementation Notes

**Feito como planejado**, com dois desvios pequenos:

- **`freteIsencaoDeTeste` é 25000, não o padrão de produção 29900.** Com o valor igual ao padrão, o teste passaria mesmo se alguém cravasse o número no handler; diferente, ele prova que a leitura repassa a Config — que é a terceira condição de aceite.
- **`web/app/produtos/[id]/caixa-de-compra.tsx`** entrou com uma linha só: o comentário citava o botão pelo nome antigo ("Confirmar compra"). Nenhum código mudou ali.

**A linha do Carrinho vazio da matriz vale por construção, não por teste:** `falta` é calculado depois do retorno antecipado do Carrinho vazio, então não há como a frase aparecer sem subtotal. O que fica sem teste automatizado é o JSX — a frase renderizada, como os dois `Alert` da 4.4 — e isso vai ao `deferred-work.md` junto com o passeio no navegador.

**`git` normaliza o fim de linha**: três arquivos foram gravados com LF por um script Python; `git diff --ignore-cr-at-eol` é idêntico ao `git diff`, então não há ruído de CRLF no diff.

## Spec Change Log

## Review Triage Log

Duas lentes (Edge Case Hunter e Verification Gap) rodadas na própria sessão, sem subagente — a pedido, e no mesmo molde da 4.4. A Blind Hunter não rodou: o raio é pequeno (um campo de configuração repassado, uma subtração pura e três textos), e a política do projeto é uma ou duas lentes fora da concorrência e da máquina de estados. Nenhum achado de `intent_gap`, `bad_spec` ou `patch`.

- **false:** "limiar ausente ou zero renderiza `R$ NaN`". Não acontece: `Math.max(0, undefined - n)` é `NaN`, `NaN > 0` é falso, e a frase simplesmente não sai. Além disso `internal/plataforma/config.go:215` recusa `AZAMON_FRETE_ISENCAO_CENTAVOS` menor ou igual a zero no arranque, então em produção o limiar é sempre positivo.
- **false:** "a claim de que a varredura de vocabulário fechou as telas de Comprador não foi verificada". Foi: a varredura por `compra|item|reserva|garantido` em `web/app` e `web/components` fora de `/admin/` só devolve identificadores, chaves JSON e internos do shadcn. O texto visível da Vitrine já diz "Produto" e "resultados".
- **rejeitado (maybe-false, seria `low`):** o `<div className="text-right">` novo envolve o subtotal, e no `flex-wrap` estreito a caixa pode alinhar diferente do `<p>` de antes. Só o passeio a 360 px decide, e a consequência máxima é cosmética.
- **rejeitado (`low`):** "a frase devia virar `textoDoFreteGratis()` em `lib`, como `textoDoBloqueio`". O paralelo não se sustenta: `textoDoBloqueio` devolve texto puro, e esta frase tem o valor dentro de um `<span className="tabular-nums">` — extraí-la para `lib` perderia o `tabular-nums` que a condição de aceite exige. A composição em JSX segue o `Alert` de preço da 4.4.
- **defer:** a frase renderizada não tem teste automatizado, e a ausência dela no Carrinho vazio vale por construção (o retorno antecipado) e não por teste (`deferred-work.md`). A aritmética que a alimenta tem: `faltaParaFreteGratis` cobre as cinco fronteiras da matriz, e o campo da API tem asserção com valor **diferente** do padrão de produção.

## Design Notes

**Por que o limiar vai para o navegador em vez de a distância.** Mandar `falta_para_frete_gratis_centavos` pronto seria uma linha a menos, mas a edição otimista da 4.3 recalcula o subtotal na tela entre o clique e a resposta: a distância vinda do servidor ficaria velha exatamente enquanto o número ao lado dela muda. Com o limiar, as duas saem da mesma soma, e o NFR-13 ("o total exibido é a soma exata das parcelas exibidas") continua valendo durante o intervalo. O limiar é dado de configuração, não regra — o AD-10 segue intacto.

**A varredura do glossário é auditoria, não reescrita.** As telas de Comprador foram varridas nesta investigação e só `comprar.tsx` destoa; por isso a tarefa de vocabulário é de três textos, e o Code Map registra o que já foi conferido para que a revisão não repita a busca.

## Verification

**Commands:**
- `go vet ./... && go test -count=1 ./...` -- verde, inclusive `TestSessaoEProduto` (na imagem `azamon-dev`, ver HANDOFF).
- `cd web && npm test && docker build ./web` -- verde.

**Manual checks (if no CLI):**
- Passeio a 360 e 1440 px: Carrinho abaixo do limiar mostra a frase; subir a quantidade até cruzar R$ 299,00 faz a frase sumir junto com a mudança do subtotal; esvaziar não deixa frase órfã.
