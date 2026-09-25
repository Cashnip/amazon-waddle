---
title: '7.1 — O README leva do clone ao sistema rodando'
type: 'chore'
created: '2026-09-25'
status: 'done'
review_loop_iteration: 0
baseline_commit: '17916f7750f8b1f0803a6b1f3cfdbd53058ca29a'
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-7-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O `README.md` foi medido na 1.9 (3 min 26 s, 2026-09-13) e depois só ganhou parágrafos: descreve um sistema anterior às Épicas 5 e 6 ("recusa, nova Tentativa e expiração ainda no backlog", "Meus pedidos é o esboço que a 6.1 substitui", "Estoque total informado só na criação", nenhum painel de Pedidos do Administrador) e não diz onde ler o token da FR-3 no log nem nomeia o modo demonstração. Documentação que descreve outro sistema é pior que nenhuma (SM-C4), e a medida de 15 min nunca foi refeita com o que as Épicas 2–6 acrescentaram.

**Approach:** Percorrer as seções 1 a 5 do README contra o código em `main`, corrigir cada afirmação que não bate, acrescentar o que a CA exige (modo demonstração como padrão, `down -v` como reinício, o token da FR-3 no log), e medir de novo a subida do zero com cache frio, registrando o resultado num **Registro de subidas** que torna a verificação recorrente: a cada semana, um integrante diferente acrescenta uma linha.

## Boundaries & Constraints

**Always:** Números no README são medidos, nunca estimados — com data, máquina e versão do Docker. Os 20 termos do §3 do PRD literais. Cada afirmação sobre comportamento confere com o código ou com uma execução real desta sessão. A medição acontece num clone novo fora do repositório de trabalho, com builder BuildKit próprio (cache frio sem apagar o cache de quem desenvolve).

**Ask First:** Qualquer mudança fora do `README.md` e da spec que a correção de uma afirmação parecer exigir (código, `.env`, compose).

**Never:** Remover `postgres:18.6`, `redis:8.10.1` ou os contêineres de testcontainers já no ar (decisão humana de 2026-09-25: o pull dessas duas fica morno, e o Registro diz isso). Tocar código, `.env`, `docker-compose.yml` ou Dockerfiles nesta estória — divergência vira linha em `deferred-work.md`. Reescrever as seções 6 a 10 (ferramenta de desenvolvimento; a contagem de 49 skills não é verificável sem reinstalar o BMad — vira `deferred-work.md`). Declarar a metade "pessoa de fora" da SM-3 como cumprida: o agente não é essa pessoa.

## I/O & Edge-Case Matrix

| Cenário | Estado | Esperado |
|---|---|---|
| Subida do zero | clone novo, builder sem cache | primeiro Produto em `localhost:3000` em ≤ 15 min; tempo registrado |
| Desfecho `,00` | Caixa de Som Maré R$ 189,00 + Frete SP R$ 15,00 | `PAGO` e depois `ENTREGUE` sem intervenção; tempos registrados |
| Desfecho `,90` | Fone Aurora R$ 249,90 | `PAGAMENTO_RECUSADO`; nova Tentativa aprova |
| Desfecho `,95` | Andarilho R$ 39,95 | sem confirmação; expira em 60 s (`AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO`) |
| FR-3 | `POST /api/v1/redefinicoes-de-senha` com o e-mail do Comprador | a linha `redefinição de senha solicitada` em `docker compose logs azamon` traz `caminho` `/redefinir-senha/<token>`, que abre a tela e troca a senha |

</frozen-after-approval>

## Code Map

- `README.md` -- único arquivo de entrega. §1 (subida, credenciais), §2 (passeio; o passo 3 dos centavos e o passo 6 dos tempos estão velhos), subseções Meus endereços / Menu da conta / Administrador (velhas nas partes de Meus pedidos, Estoque e rotas 401), §3 (offline: confere), §4 (reinício: confere), §5 (`npm test` diz "a guarda offline e o teste da casca"; hoje são 9 arquivos em `web/scripts/*.test.mjs`).
- `api/redefinicao.go:60` -- o log do token: `slog.InfoContext(..., "redefinição de senha solicitada", "caminho", "/redefinir-senha/"+token)`, em JSON (`internal/plataforma/log.go:36`). Validade `AZAMON_SENHA_TOKEN_VALIDADE=30m`; nova solicitação invalida a anterior (`internal/identidade/redefinicao.go:39`). Tela: `web/app/esqueci-a-senha/`, `web/app/redefinir-senha/[token]/`.
- `.env` -- cabeçalho diz "Modo demonstração — é o que `docker compose up` sem argumento sobe (AD-13)"; padrões de produção em `internal/plataforma/config.go`. Faixas do Provedor: aprova até `,89`, recusa até `,94`; 3 Tentativas; expiração 60 s; entrega 30 s; confirmação 5 s.
- `db/migracoes/20260919160000_pedido_faixa_frete.sql:42` -- Sudeste R$ 15,00 (CEP `01310-100` confere).
- `db/migracoes/20260912120100_catalogo_reserva_estoque.sql:12` -- `estoque_total` padrão 10; `internal/pedido/maquina.go:132` consolida em `ENVIADO` (§4 confere).
- `db/semente/001_catalogo_semeado.sql` -- Maré `a0ae8ff1-da13-5591-9291-4a4f1ce15383` confere.
- `api/*.go` rotas -- `/api/v1/admin/pedidos`, `/admin/pedidos/{id}/transicoes`, `/admin/produtos/{id}/estoque`, `/carrinho/*`, `/checkout/entrada`, `/pedidos/{id}/cancelamento`, `/pedidos/{id}/tentativas` ausentes do README.
- `web/app/admin/pedidos/` -- Tabela de Pedidos e Detalhe com ações; `AZAMON_ENTREGA_SIMULACAO_ATIVA` (`config.go:117`).
- Rótulos citados no passo a passo existem: "Fechar o Pedido" (`carrinho/meu-carrinho.tsx:307`), "Confirmar Pedido" (`checkout/revisao/revisao-do-pedido.tsx:389`), "Adicionar ao Carrinho".

## Tasks & Acceptance

**Execution:**
- [x] (scratchpad) -- clonar `main` num diretório novo; `docker buildx create --name azamon-medicao --driver docker-container`; cronometrar `BUILDX_BUILDER=azamon-medicao docker compose up -d --build` até a Vitrine responder com um nome de Produto; percorrer por API os três desfechos e a FR-3 da matriz, anotando os tempos; `down -v`, remover builder e imagens da medição -- a medida que o README vai citar.
- [x] `README.md` §1 -- nomear o modo demonstração como padrão (`.env` versionado, AD-13; produção só em `config.go`); trocar o parágrafo "Quanto demora" pelo **Registro de subidas** (data · quem · máquina/Docker · tempo · o que não bateu), com a linha da 1.9 e a desta sessão, e a regra semanal com rodízio de integrante -- CA de recorrência.
- [x] `README.md` §2 -- reescrever o passo 3 com as três faixas vigentes e um Produto de cada (e que a quantidade muda os centavos); atualizar os tempos do passo 6 com os medidos; nova subseção "Esqueci a senha" com o comando `docker compose logs azamon | grep redefinir-senha` e a URL a abrir -- FR-3 e SM-C4.
- [x] `README.md` subseções de conta e Administrador -- Meus pedidos e Detalhe do Pedido como estão hoje (cancelamento, nova Tentativa), painel `/admin/pedidos` (transição e simulação de entrega), ajuste de Estoque pela rota própria, rotas 401 completas -- SM-C4.
- [x] `README.md` §5 -- descrever o `npm test` real.
- [x] `_bmad-output/implementation-artifacts/deferred-work.md` -- uma entrada para a contagem de skills da §6, e uma por divergência de código achada.

**Acceptance Criteria:**
- Dado o README novo, quando cada comando e URL das seções 1 a 4 é executado nesta sessão sobre o clone medido, então todos respondem como o texto diz.
- Dado o Registro de subidas, quando lido, então tem a medida desta sessão com cache frio, ≤ 15 min, e declara quem mediu — e a linha "pessoa de fora" fica em aberto, com dono a nomear pelo time.
- Dado `grep -n "backlog\|esboço\|só na criação" README.md`, então nada descreve como futuro o que já está em `main`.

## Review Triage Log

Duas lentes (Blind Hunter e Verification Gap; a de bordas ficou de fora pelo raio de diff só de documentação, regra do HANDOFF). Verification Gap: nada. Blind Hunter: 20 achados, nenhum intent_gap nem bad_spec.

- **patch** (aplicados): aviso de caminho longo antes do primeiro comando e `-c core.longpaths=true` no clone principal e na receita; receita de medição com `up -d`, condição de parada por `curl`, limpeza da pasta do clone e versão PowerShell; linhas do Registro declaradas não comparáveis, coluna morno com os 16 s, autor da 1.9 nomeado (`Cashnip`); "confirma em 5 s" tirado do que o modo demonstração encolhe (produção também é 5 s); tempos da tela até um intervalo de consulta depois dos da API; "6 s depois do clique" → "depois de pedida"; "Tentar pagar de novo" só enquanto restam Tentativas; `--no-log-prefix` para a linha sair JSON; "pela metade" → "com arquivos faltando".
- **reject**: códigos 409 que faltam em rotas (o README não é o contrato da API), lista de rotas sem Sessão incompleta (é exemplo), `TETO_DE_TENTATIVAS` inalcançável no Simulado (é contrato verdadeiro da rota), reembolso de Pedido cancelado pago (fora do escopo do README), contagens de teste datadas (têm data), página da Vitrine só da Maré (os links levam direto), dono da entrada de caminho longo, data da entrega.

## Design Notes

O registro fica no próprio README, e não num arquivo à parte, porque é o README que a pessoa de fora segue e a data da última subida é o que diz se ele ainda vale. Pull de `postgres`/`redis` morno é aceitável se registrado como tal; o grosso do tempo é o download de módulos Go, pacotes npm e imagens base de construção, que o builder próprio deixa frio.

## Verification

**Commands:**
- `docker compose -p <medição> ps` -- `azamon`, `postgres`, `redis` `healthy`.
- `curl -s localhost:8080/api/v1/saude` -- `{"status":"ok"}`.
- `git diff --stat` -- só `README.md`, `deferred-work.md` e esta spec.

**Manual checks:**
- Ler o README do topo ao fim da §4 como quem nunca viu o projeto: nenhum termo sem explicação, nenhum passo que dependa do HANDOFF.

## Suggested Review Order

**A verificação recorrente (o coração da CA)**

- O Registro: cada subida do zero vira linha datada; a da pessoa de fora fica em aberto.
  [`README.md:63`](../../README.md#L63)

- A regra semanal e a receita de cache frio que não apaga o cache de quem desenvolve.
  [`README.md:81`](../../README.md#L81)

**O que a CA pede nomeado**

- Modo demonstração como padrão, com os padrões de produção ao lado.
  [`README.md:30`](../../README.md#L30)

- O token da FR-3 lido no log, com o comando exato.
  [`README.md:178`](../../README.md#L178)

- `down -v` contra `down`: o que cada um apaga.
  [`README.md:365`](../../README.md#L365)

**O que deixou de mentir (SM-C4)**

- As três faixas do Provedor Simulado com um Produto medido em cada.
  [`README.md:126`](../../README.md#L126)

- Meus pedidos e o Detalhe como estão depois da Épica 6.
  [`README.md:235`](../../README.md#L235)

- O painel de Pedidos do Administrador, que o README não citava.
  [`README.md:293`](../../README.md#L293)

- Ajuste de Estoque pela rota própria, no lugar de "só na criação".
  [`README.md:286`](../../README.md#L286)

- A lista completa de rotas que dão 401 à Sessão de Administrador.
  [`README.md:316`](../../README.md#L316)

**Periféricos**

- Aviso de caminho longo no Windows, achado na própria medição.
  [`README.md:17`](../../README.md#L17)

- O `npm test` real, com contagem datada.
  [`README.md:388`](../../README.md#L388)

- O que ficou adiado: contagem de skills, comentários velhos em `api/rotas.go`, caminho longo.
  [`deferred-work.md:513`](deferred-work.md#L513)
