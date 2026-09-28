---
title: '7.4 — Os três roteiros ensaiados a partir de clone limpo'
type: 'chore'
created: '2026-09-28'
status: 'in-progress'
route: 'dispatch'
review_loop_iteration: 0
baseline_commit: 'ddcc36139e47b63539454dd8a4d216e73b74a8aa'
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-7-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A SM-1 ("os três roteiros do §12 executam de ponta a ponta em ambiente limpo, sem intervenção manual no banco e sem erro visível — alvo 3 de 3") nunca foi verificada como percurso: os passeios das Épicas 5 e 6 exercitaram estórias isoladas, a 7.1 mediu a subida e percorreu os desfechos **pela API, sem clicar em nenhuma tela**, e o HANDOFF lista o que ninguém registrou ter visto depois das correções de D1–D7 e E1–E5 (a UJ-3 inteira, o `Alert` de preço, Meus pedidos vazio, o painel do Administrador pelo teclado). Não existe roteiro de ensaio: os três roteiros vivem no §12 do PRD como prosa de planejamento, sem passo numerado, sem o dado de demonstração que provoca cada desfecho, sem lugar para registrar quem ensaiou. E o `deck-banca.html`, que vai projetado, desenha a porta como `ProvedorDePagamento` — nome que o código não tem — e a fronteira do banco sem `pedido.contador_numero`.

**Approach:** Executar os três roteiros de ponta a ponta a partir de clone limpo, **no navegador**, e deixar um `ensaio-dos-roteiros.md`: um passo numerado por passo do §12, com o dado exato que provoca o desfecho, o que tem de aparecer na tela, a evidência do que esta execução viu, e um **Registro de ensaios** que torna a regra das duas pessoas verificável em vez de presumida. Defeito achado é registrado, não corrigido aqui. O deck é emendado nos dois pontos em que mente.

**Decidido em 2026-09-28:** o agente executa os três roteiros como **ensaio técnico** — linha 1 do Registro, marcada "agente" —, e o Registro abre **duas linhas nomeadas em branco** para o time: a metade humana da regra de ensaio fica declaradamente pendente, como a linha da pessoa de fora no Registro de subidas da 7.1. A tabela das perguntas prováveis nasce com a **coluna Responsável em branco** e a fonte de cada resposta preenchida; os nomes são do time. As cinco divergências da espinha adiadas pela 7.3 (AD-18, Semente Estrutural, AD-10, AD-19, tabela de rotas do AD-16) **ficam fora desta estória** e seguem em `deferred-work.md` com prazo antes da entrega.

## Boundaries & Constraints

**Always:** Clone limpo num diretório fora do repositório de trabalho, `docker compose up --build` (a armadilha do HANDOFF: `up` sem `--build` sobe imagem velha; conferir a hora em `docker compose images`), e nada de `psql` — intervenção no banco reprova o roteiro. Cada passo do ensaio cita o FR do §12 e o dado de demonstração escolhido (Produto, quantidade, CEP), porque o desfecho do pagamento vem dos centavos do total e não de configuração. Evidência é o que a tela mostrou, com o status e o horário; passo não executado fica marcado como não ensaiado, nunca como aprovado. Os 20 termos do §3 do PRD, literais. O deck emendado ganha `uv run _bmad/scripts/memlog.py append --workspace <dir da arquitetura> --by dev --type change`.

**Ask First:** Qualquer defeito cuja correção mude código de produção — HALT com a lista antes de tocar em arquivo fora dos entregáveis desta spec.

**Never:** Corrigir defeito de comportamento aqui (é defeito da épica de origem: vira entrada em `deferred-work.md` e linha no HANDOFF). Funcionalidade nova (SM-C1). Tocar no grafo do AD-1, na tabela `arestas`, no `ARCHITECTURE-SPINE.md` ou no `addendum.md`. Reescrever o §12 do PRD. Declarar cumprida a metade humana da regra de ensaio: o agente não é uma das duas pessoas. Emendar a espinha ou o `DIAGRAMA-MODULOS.md` pelas cinco divergências adiadas — elas são de outra estória.

## I/O & Edge-Case Matrix

| Cenário | Dado da demonstração | Esperado na tela |
|---|---|---|
| Roteiro A (passos 1–9) | Caixa de Som Maré R$ 189,00, 1 un., CEP `01310-100` → total R$ 204,00 | busca com acento errado, filtro por faixa, ordenação, Vendedor na Página de Produto, Carrinho, dois passos do checkout com Frete recalculado ao trocar de região, `PAGO` sem recarregar, Meus pedidos com preço praticado e linha do tempo, `ENTREGUE` durante o resto |
| Roteiro B passos 1–4 | Fone Aurora R$ 249,90 → total R$ 264,90 | `PAGAMENTO_RECUSADO` com motivo; Estoque **volta** na Página de Produto em outra aba; "Tentar pagar de novo" chega a `PAGO` |
| Roteiro B passo 5 | Anotações de um Andarilho R$ 39,95 → total R$ 54,95 | sai sozinho de `AGUARDANDO_PAGAMENTO` em 60 s, motivo "Tempo de pagamento expirado.", Estoque volta |
| Roteiro C passos 1–4 | `admin@azamon.test` em `/admin/entrar`; Vendedor e Produto novos; o Pedido `PAGO` do Roteiro A | Produto novo na Vitrine; `PAGO`→`SEPARANDO` pelo painel; cancelamento pelo Comprador com Estoque voltando; recusa do cancelamento em `ENVIADO` |
| Roteiro C passo 5 | só se o teste da NFR-7 passar | dois checkouts na última unidade, um Pedido só — sem o teste verde, o passo fica **fora da apresentação** |
| Erro visível em qualquer passo | notificação com identificador de correlação | roteiro **reprova**; defeito registrado, não corrigido |
| Reinício entre roteiros | `docker compose down -v && up --build` | nada reconfigurado entre roteiros; o Registro diz se houve reinício e por quê |

</frozen-after-approval>

## Code Map

- `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md:720–765` -- §12: os três roteiros com o FR de cada passo, as regras de ensaio e as seis perguntas prováveis. É a fonte do ensaio; não editar.
- `README.md:117–177` -- §2 tem a tabela dos três desfechos por centavos com um Produto e um uuid de cada, os tempos medidos (`PAGO` em 7 s, `ENTREGUE` 1 min 39 s), o Frete de SP e o efeito da quantidade. Reusar esses dados no ensaio em vez de remedir. `:30–62` credenciais e "deu certo se"; `:261+` a porta do Administrador (`/admin/entrar`, não `/entrar`); `:356` o reinício.
- `README.md:63` -- **Registro de subidas** da 7.1: o formato de registro recorrente a imitar (data · quem · máquina/Docker · tempo · o que não bateu), inclusive a linha sem dono da pessoa de fora.
- `api/concorrencia_test.go:19,164` -- `consistenciaSobConcorrencia` é o teste da NFR-7, chamado de `api/sessao_test.go:540` (`TestSessaoEProduto`); o subteste "a última unidade: oito checkouts, um Pedido" é o passo 5 do Roteiro C. Verde é a condição de o passo entrar na apresentação.
- `_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/deck-banca.html:526` -- `<code>ProvedorDePagamento</code>`, nome inexistente; pela emenda do AD-8 na 7.3 a porta é contrato (`pagamento.IniciarTentativa` mais a rota do webhook), não `interface` Go. `:367` a linha do schema `pedido` sem `contador_numero`. `:612–631` a seção "Onde mora cada resposta", com sete perguntas (as seis do §12 mais o Redis) — a prosa já promete responsável nomeado.
- `_bmad-output/implementation-artifacts/deferred-work.md:529` -- a entrada do deck, "insumo da 7.4"; fecha com o prefixo `RESOLVIDO —` no `summary`, como `:55`.
- `HANDOFF.md:115` -- o que ninguém registrou ter visto depois das correções: é a lista que o ensaio tem de atravessar. `:163,175` as duas armadilhas (imagem velha; semente sobre banco existente).
- `.env` -- faixas do Provedor (aprova até `,89`, recusa até `,94`), 3 Tentativas, expiração 60 s, entrega 30 s, confirmação 5 s; `AZAMON_ENTREGA_SIMULACAO_ATIVA`.
- `_bmad-output/implementation-artifacts/percurso-do-addendum.md` -- o formato de registro por veredito da 7.3, a imitar na contagem por passo.

## Tasks & Acceptance

**Execution:**
- [ ] (scratchpad) -- clonar `main` num diretório novo fora do repositório; `docker compose up --build -d`; conferir `docker compose images` e `curl /api/v1/saude`; ao fim, `down -v` e remover as imagens do ensaio -- o clone limpo que a CA exige.
- [ ] (scratchpad) -- executar os três roteiros no navegador, passo a passo, anotando por passo o que a tela mostrou, o status e o horário; antes do passo 5 do Roteiro C, rodar o teste da NFR-7 -- a execução é a prova da SM-1.
- [ ] `_bmad-output/implementation-artifacts/ensaio-dos-roteiros.md` -- o entregável: contagem de passos aprovados/reprovados/não ensaiados no topo; uma seção por roteiro com passo numerado, FR, dado da demonstração, o que tem de aparecer e a evidência desta execução; a condição do passo 5 do Roteiro C; **Registro de ensaios** no formato do Registro de subidas; a tabela das perguntas prováveis com Responsável e Fonte; e as armadilhas de quem for apresentar.
- [ ] `.../architecture-azamon-2026-09-05/deck-banca.html` -- a porta como contrato no lugar de `ProvedorDePagamento`, e `contador_numero` na linha do schema `pedido`; memlog da arquitetura pelo script -- o deck vai projetado e não pode mostrar nome que o código não tem.
- [ ] `_bmad-output/implementation-artifacts/deferred-work.md` -- `RESOLVIDO —` na entrada do deck (`:529`), e uma entrada por defeito que o ensaio achar, com o roteiro e o passo.
- [ ] `README.md:63` e `HANDOFF.md` -- um ponteiro do Registro de subidas para o Registro de ensaios, e o parágrafo da 7.4 no HANDOFF com o que ficou aberto -- a 7.4 é a última da épica, e a entrega lê o HANDOFF.
- [ ] `_bmad-output/implementation-artifacts/sprint-status.yaml` -- `7-4-…` para `review`.

**Acceptance Criteria:**
- Dado `ensaio-dos-roteiros.md`, então todo passo dos três roteiros tem dado de demonstração e veredito, e nenhum veredito é presumido: passo não executado aparece como não ensaiado.
- Dado o ensaio concluído, quando algum roteiro reprova, então existe entrada em `deferred-work.md` com roteiro e passo, e nenhum arquivo de produção foi alterado.
- Dado `grep -c ProvedorDePagamento deck-banca.html`, então dá 0, e `grep -c contador_numero` dá ao menos 1.
- Dado o código, quando `go test ./...` roda, então passa como antes — esta estória não toca código.

## Implementation Notes

## Spec Change Log

## Review Triage Log

## Design Notes

O ensaio é documento de apresentação, não de setup: mora em `implementation-artifacts` ao lado do `percurso-do-addendum.md` porque é prova de métrica (SM-1) — o README continua sendo o caminho do clone ao sistema rodando, e aponta para cá. O navegador é o modo obrigatório: a 7.1 já percorreu os três desfechos pela API, e "sem erro visível" é afirmação sobre a tela. Defeito achado não se corrige aqui porque a Épica 7 verifica e não constrói: corrigir dentro do ensaio destruiria a evidência de que o roteiro reprovou.

## Verification

**Commands:**
- `docker run --rm -v "$PWD":/src -w /src -v azamon-gocache:/root/.cache -v //var/run/docker.sock:/var/run/docker.sock -e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal -e TESTCONTAINERS_RYUK_DISABLED=true azamon-dev:latest go test ./...` -- expected: tudo `ok`, inclusive o subteste da última unidade.
- `git diff --stat` -- expected: nenhum arquivo em `api/`, `internal/`, `cmd/`, `db/` ou `web/`.
