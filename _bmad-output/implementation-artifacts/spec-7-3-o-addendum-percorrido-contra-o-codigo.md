---
title: '7.3 — O addendum percorrido contra o código'
type: 'chore'
created: '2026-09-25'
status: 'done'
review_loop_iteration: 1
baseline_commit: '05a3682d589d9d696d85a0aef5a1f7da1546b3e3'
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-7-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A SM-7 é presumida: o `addendum.md` acumulou ~200 decisões em §1–§10 ao longo de seis épicas, e ninguém as conferiu contra o código de `main`. Já se sabe de três contradições na espinha que o addendum cita — o AD-4 sem o contador do ano na ordem de trava, a tabela do AD-6 com `Expirar` e `SimularEntrega` "dentro" de `pedido.Varrer`, e o AD-8 sem dizer se a porta é tipo ou contrato — e decisões de estórias antigas substituídas por estórias novas sem marca no lugar.

**Approach:** Percorrer cada decisão do addendum contra o código e deixar um veredito com evidência por decisão num registro do percurso. Cada contradição se resolve sem apagar a decisão antiga: nota no lugar mais entrada da 7.3 no §10 com o porquê, ou correção de comentário no código. As três contradições da espinha são emendadas no lugar. **Segundo sentido (renegociado pelo humano em 2026-09-25, na revisão):** as estórias sem bloco no §10 (Épicas 2–4, 5.2, 5.7, 6.1, 6.2, 6.7, 7.1 e as correções de passeio) têm spec e commit varridos atrás de decisão de arquitetura que não chegou ao addendum, e cada uma entra no §10, no lugar cronológico, num bloco marcado "registrado na 7.3". Decidido em 2026-09-25: `internal/pedido/pedido.go` fica como está (B2), e o AD-4 entra aqui.

## Boundaries & Constraints

**Always:** Nunca apagar nem reescrever uma decisão antiga: a nota entra como sub-item novo logo abaixo do item, em itálico, no formato `  - *(7.3: substituída pela N.M — …)*`, sem tocar a linha antiga. Emendas no AD-4, AD-6 e AD-8 são no lugar, sem renumerar, cada uma com `uv run _bmad/scripts/memlog.py append --workspace <dir da arquitetura> --by dev --type change`. O addendum alterado ganha memlog do PRD pelo mesmo script. Os 20 termos do §3 do PRD, literais. Toda evidência é `arquivo:linha` conferido, ou commit/spec quando a decisão é de processo.

**Ask First:** Qualquer contradição cuja resolução exija mudar comportamento de código de produção (não comentário): HALT com a lista, antes de registrar como mudança de decisão.

**Never:** Dividir `pedido.go`. Editar `SPEC.md`, `mapa-de-capacidades.md` ou `deck-banca.html` (é da 7.4). Mexer no grafo do AD-1 ou na tabela `arestas`. Emendar AD fora do 4, 6 e 8.

## I/O & Edge-Case Matrix

| Cenário | Estado | Veredito e resolução |
|---|---|---|
| Código faz o que a decisão diz | ex.: §10 1.6 "contador por ano" ↔ `ProximoNumeroDoAno` | `confirmada`, com `arquivo:linha` |
| Estória posterior já registrou a troca | ex.: §10 1.7 "relógio chama Varrer e emitir" ↔ 5.11 acrescentou `Expirar` | `substituída-registrada`; nota no item antigo apontando a entrada nova |
| Troca sem registro | código diverge e nenhum §10 diz por quê | `substituída-agora`; nota no item + linha na entrada 7.3 com o porquê (da spec ou do commit que trocou) |
| Comentário mente | ex.: migração da 5.6 diz que o gêmeo espera no índice | `corrigida-no-código` |
| Fato de processo ou medida | ex.: "3 min 26 s", "verificação 1 fechada" | `histórica`, com commit ou spec |
| Comportamento contradiz decisão vigente | defeito | HALT (Ask First) |
| Decisão da construção fora do addendum | ex.: a guarda `somenteAdministrador` da 2.4 | bloco novo no §10 com o porquê da spec, marcado "registrado na 7.3" |

</frozen-after-approval>

## Code Map

- `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` -- §1–§9 `:14–156`; §10 `:158–431`, 24 blocos por estória (1.1–1.9, 5.1, 5.3–5.6, 5.8–5.11, 6.3–6.6, D3/D5, 7.2); nenhum para as Épicas 2–4, 5.2, 5.7, 6.1, 6.2, 6.7, 7.1 nem para as correções de passeio. 126 KB: ler por faixa. Frontmatter `updated: 2026-09-11` está velho.
- `.../architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md:138` -- AD-4: "primeiro a linha de `pedido`, depois `catalogo.produto` ordenadas por `id`". O código de `pedido.Criar` (`internal/pedido/pedido.go:198–206`) trava antes `pedido.contador_numero` do ano (`ProximoNumeroDoAno`, `internal/pedido/db/consultas.sql:8`, `INSERT … ON CONFLICT DO UPDATE`), depois insere o Pedido, depois `catalogo.Reservar` (`:251`).
- `ARCHITECTURE-SPINE.md:162` -- AD-6: tabela com passos 2 e 3 como "idem" de `pedido.Varrer`, "dois passos" na prosa. Código: `cmd/azamon/main.go:173–186`, `pedido.Varrer` → `pedido.Expirar` → `pedido.SimularEntrega` (só se `cfg.EntregaSimulacaoAtiva`) → `pagamento.EmitirConfirmacoesDevidas`.
- `ARCHITECTURE-SPINE.md:193` -- AD-8. `internal/pagamento/pagamento.go:64` `Simulado`, `:135` `IniciarTentativa`, `:309` `EmitirConfirmacoesDevidas`; nenhum `interface`. Addendum §4 `:79` diz "uma interface… o Provedor Simulado implementa a interface".
- `db/migracoes/20260921120000_pedido_criacao.sql:13–15` -- "o gêmeo concorrente espera neste índice"; a espera real é no contador (comentário de `pedido.go:198`).
- `.../architecture-azamon-2026-09-05/.memlog.md`, `prd-azamon-2026-08-14/.memlog.md` -- só pelo script.
- `deferred-work.md:524` e `:532` -- as duas entradas da 7.3 (porta do AD-8, AD-6); fecham com o prefixo `RESOLVIDO —` no `summary`, como `:55`.
- `sprint-status.yaml:107` -- `7-3-…`; `epic-5-retro-item-27` (AD-4) fecha aqui.

## Tasks & Acceptance

**Execution:**
- [x] Percurso -- seis subagentes novos (não fork), só leitura, em paralelo, um por faixa: §1–§9 · 1.1–1.5 · 1.6–1.9 · 5.1–5.6 · 5.8–5.11 · 6.3 até 7.2. Cada um devolve linhas `| bloco#item | decisão (≤12 palavras) | veredito | evidência |` e não edita nada. Toda linha que não seja `confirmada` é reconferida por mim, mais uma `confirmada` por bloco, por amostra.
- [x] `_bmad-output/implementation-artifacts/percurso-do-addendum.md` -- o registro: uma tabela por bloco, contagem por veredito no topo. É a prova da SM-7.
- [x] `addendum.md` -- notas no lugar; entrada `**Estória 7.3 — o addendum percorrido contra o código** (2026-09-25)` no §10 com o método, a contagem, cada `substituída-agora` com o porquê, as três emendas da espinha e a decisão de manter `pedido.go`; `updated: 2026-09-25`.
- [x] `ARCHITECTURE-SPINE.md` -- AD-4: ordem global "contador do ano (só na criação) → linha de `pedido` → `catalogo.produto` por `id`"; AD-6: tabela com as três funções de `pedido` e a condição da simulação; AD-8: a porta é contrato (assinatura exportada mais a rota do webhook), não `interface` Go, com o porquê.
- [x] Comentários que o percurso achou mentindo, só comentário -- `db/migracoes/20260921120000_pedido_criacao.sql` (onde o gêmeo espera), `web/app/checkout/revisao/revisao-do-pedido.tsx`, `api/rotas.go`, `internal/catalogo/catalogo.go` e `internal/catalogo/db/consultas.sql` (com o `gerado/` conferido por `sqlc generate`), `db/medicao_nfr4_test.go` e o cabeçalho dos seis módulos.
- [x] Segundo sentido -- quatro subagentes novos, só leitura, varrem as specs sem bloco no §10 (Épica 2 · Épica 3 · Épica 4 com 5.2 e 5.7 · 6.1, 6.2, 6.7, correções e 7.1) e devolvem as decisões de arquitetura que faltam, com o porquê da spec e `arquivo:linha`. Cada estória com decisão ganha bloco no §10, no lugar cronológico, marcado "registrado na 7.3"; o percurso e a entrada da 7.3 contam os dois sentidos.
- [x] Os dois `.memlog.md` -- pelo script.
- [x] `deferred-work.md`, `sprint-status.yaml` -- três entradas `RESOLVIDO —` (AD-8, AD-6, comentários de `api/rotas.go`) e duas novas (AD-18, Semente Estrutural); `epic-5-retro-item-27` → `done` com referência a esta spec.

**Acceptance Criteria:**
- Dado `percurso-do-addendum.md`, então todo item de §1–§10 tem veredito e evidência, e nenhum fica `contradita`.
- Dado `grep -n "idem" ARCHITECTURE-SPINE.md` e o AD-4, então a espinha não diz nada que `cmd/azamon/main.go` e `pedido.Criar` desmintam.
- Dado o addendum depois, então `git diff` só acrescenta linhas nele (nenhuma removida, salvo o `updated`).
- Dado o código, quando `go test ./...` roda, então passa como antes.

## Spec Change Log

- **Revisão 1 (2026-09-25), Blind Hunter — intent_gap renegociado pelo humano.** Achado: o percurso só ia do addendum ao código, e a CA da épica pede também que toda decisão de arquitetura da construção esteja no addendum; 24 estórias não têm bloco no §10. O humano escolheu estender a 7.3 em vez de reverter: o Approach congelado ganhou o segundo sentido e a matriz, a linha "decisão da construção fora do addendum". Estado ruim evitado: declarar a SM-7 provada com metade do caminho. KEEP: o percurso de 260 decisões, as 23 notas, as emendas do AD-4, AD-6 e AD-8, as correções de comentário e os `RESOLVIDO —` ficam; a revisão os corrigiu por patch (o NFR-4 foi medido na retro da Épica 3, 1.8#2 é `substituída-agora`, o "recusa" do gêmeo, a ordem de trava das linhas não disputadas, o AD-8 diante do Stripe, a quebra de linha dos cabeçalhos).

## Design Notes

Os subagentes são novos, e não fork, pela armadilha da 2.6 no HANDOFF: fork herda o workflow e pode implementar em vez de investigar. O percurso vale pela evidência, e não pela prosa: um veredito sem `arquivo:linha` não conta. A porta do AD-8 como contrato: com um adapter só, um `interface` Go seria cerimônia; na troca pelo Stripe, sai `Simulado` + `EmitirConfirmacoesDevidas` e entra quem chama o gateway depois do commit (o AD-4 proíbe rede dentro da transação), e ficam a assinatura de `IniciarTentativa`, a rota do webhook e a inbox; a autenticação do handler e o passo síncrono com o navegador mudam com o gateway.

## Verification

**Commands:**
- `docker run --rm -v "$PWD":/src -w /src -v azamon-gocache:/root/.cache -v //var/run/docker.sock:/var/run/docker.sock -e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal -e TESTCONTAINERS_RYUK_DISABLED=true azamon-dev:latest go test ./...` -- expected: tudo `ok`.
- `git diff --numstat -- '*addendum.md'` -- expected: remoções só no frontmatter.
- `docker run --rm -v "$PWD":/src -w /src sqlc/sqlc:1.31.1 generate` e `git diff --ignore-cr-at-eol --stat -- '*/gerado/*'` -- expected: só a mudança de comentário de `internal/catalogo/db/gerado/consultas.sql.go` (restaurar o fim de linha dos outros).

## Suggested Review Order

**A prova, nos dois sentidos**

- O registro: 260 decisões, 225/10/9/16, nenhuma contradita; comece pela contagem.
  [`percurso-do-addendum.md:11`](percurso-do-addendum.md#L11)

- O segundo sentido: 27 blocos, 109 decisões que não estavam no addendum.
  [`percurso-do-addendum.md:332`](percurso-do-addendum.md#L332)

- A entrada da 7.3 no §10: método, as nove trocas com o porquê, as emendas.
  [`addendum.md:666`](../planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md#L666)

**As notas no lugar, sem apagar nada**

- Uma troca registrada agora: a porta do §4 era "interface" desde a 1.7.
  [`addendum.md:80`](../planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md#L80)

- As duas linhas de tabela que não comportam sub-item ganham parágrafo.
  [`addendum.md:151`](../planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md#L151)

**Os blocos retroativos (amostra)**

- A guarda administrativa e o `Content-Type` contra fixação de Sessão: decisões da 2.4.
  [`addendum.md:320`](../planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md#L320)

- O ajuste do Estoque total com rota própria, a outra passagem que o muda.
  [`addendum.md:365`](../planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md#L365)

- Como o teste do NFR-7 força a sobreposição, e por que a trava tem prova isolada.
  [`addendum.md:493`](../planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md#L493)

**A espinha emendada**

- AD-4: o contador do ano entra primeiro, e as linhas não disputadas têm lugar.
  [`ARCHITECTURE-SPINE.md:144`](../planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md#L144)

- AD-6: as três funções de `pedido`, e o passo 3 só com a simulação ligada.
  [`ARCHITECTURE-SPINE.md:166`](../planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md#L166)

- AD-8: a porta é contrato; o que fica e o que muda na troca pelo Stripe.
  [`ARCHITECTURE-SPINE.md:199`](../planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md#L199)

**Comentários que mentiam**

- Onde o gêmeo do duplo clique espera de verdade, e o que o índice faz.
  [`20260921120000_pedido_criacao.sql:13`](../../db/migracoes/20260921120000_pedido_criacao.sql#L13)

- O cabeçalho dos módulos: a porta é o pacote, não um arquivo.
  [`pedido.go:3`](../../internal/pedido/pedido.go#L3)

- A sonda do NFR-4 não é a consulta da busca, e onde está o número real.
  [`medicao_nfr4_test.go:31`](../../db/medicao_nfr4_test.go#L31)

- Trocar `VersaoSemente` derruba o arranque de banco já semeado.
  [`embutido.go:27`](../../db/embutido.go#L27)

- A rota do Pedido é o Confirmar da Revisão, não a Página de Produto.
  [`rotas.go:56`](../../api/rotas.go#L56)

**Periféricos**

- O que ficou para antes da entrega: AD-18 e a espinha divergindo do código.
  [`deferred-work.md:541`](deferred-work.md#L541)

- Os dois `RESOLVIDO —` da 7.2 fechados aqui.
  [`deferred-work.md:525`](deferred-work.md#L525)

- O item de retro do AD-4 fechado.
  [`sprint-status.yaml:318`](sprint-status.yaml#L318)

