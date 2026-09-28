---
title: '7.4 — Os três roteiros ensaiados a partir de clone limpo'
type: 'chore'
created: '2026-09-28'
status: 'done'
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
- [x] (scratchpad) -- clonar `main` num diretório novo fora do repositório; `docker compose up --build -d`; conferir `docker compose images` e `curl /api/v1/saude`; ao fim, `down -v` e remover as imagens do ensaio -- o clone limpo que a CA exige.
- [x] (scratchpad) -- executar os três roteiros no navegador, passo a passo, anotando por passo o que a tela mostrou, o status e o horário; antes do passo 5 do Roteiro C, rodar o teste da NFR-7 -- a execução é a prova da SM-1.
- [x] `_bmad-output/implementation-artifacts/ensaio-dos-roteiros.md` -- o entregável: contagem de passos aprovados/reprovados/não ensaiados no topo; uma seção por roteiro com passo numerado, FR, dado da demonstração, o que tem de aparecer e a evidência desta execução; a condição do passo 5 do Roteiro C; **Registro de ensaios** no formato do Registro de subidas; a tabela das perguntas prováveis com Responsável e Fonte; e as armadilhas de quem for apresentar.
- [x] `.../architecture-azamon-2026-09-05/deck-banca.html` -- a porta como contrato no lugar de `ProvedorDePagamento`, e `contador_numero` na linha do schema `pedido`; memlog da arquitetura pelo script -- o deck vai projetado e não pode mostrar nome que o código não tem.
- [x] `_bmad-output/implementation-artifacts/deferred-work.md` -- `RESOLVIDO —` na entrada do deck (`:529`), e uma entrada por defeito que o ensaio achar, com o roteiro e o passo.
- [x] `README.md:63` e `HANDOFF.md` -- um ponteiro do Registro de subidas para o Registro de ensaios, e o parágrafo da 7.4 no HANDOFF com o que ficou aberto -- a 7.4 é a última da épica, e a entrega lê o HANDOFF.
- [x] `_bmad-output/implementation-artifacts/sprint-status.yaml` -- `7-4-…` para `review`.

**Acceptance Criteria:**
- Dado `ensaio-dos-roteiros.md`, então todo passo dos três roteiros tem dado de demonstração e veredito, e nenhum veredito é presumido: passo não executado aparece como não ensaiado.
- Dado o ensaio concluído, quando algum roteiro reprova, então existe entrada em `deferred-work.md` com roteiro e passo, e nenhum arquivo de produção foi alterado.
- Dado `grep -c ProvedorDePagamento deck-banca.html`, então dá 0, e `grep -c contador_numero` dá ao menos 1.
- Dado o código, quando `go test ./...` roda, então passa como antes — esta estória não toca código.

## Implementation Notes

## Spec Change Log

## Review Triage Log

Passe 1 (2026-09-28). Camadas: Blind Hunter (BH) e Edge Case Hunter (EC). A Verification Gap foi pulada e anunciada: o diff não tem uma linha de código nem de teste, e a regra de custo do HANDOFF põe o gatilho no raio do diff.

| # | Achado | Veredito | Evidência da triagem | Rota |
|---|---|---|---|---|
| EC1 | Roteiro A passo 2 diz "sem Sessão — cadastrar e entrar são dois atos" | **high** | Falso no código: `api/comprador.go:58` chama `abrirSessao` no `POST /api/v1/compradores`, e `web/app/cadastrar/page.tsx:76` comenta "O cadastro já abriu a Sessão: o visitante chega à Vitrine autenticado". O documento de apresentação afirma um passo que não existe | patch |
| BH9 + EC6 + EC7 + EC8 + EC9 + EC10 | O `epic-7-context.md` foi recompilado no mesmo diff e perdeu o que a estória cita e usa | **high** | Um só defeito, seis sintomas. `git show ddcc361:…` tem, e a versão nova não: `:25` "Nunca cortar FR-19, FR-24, FR-34 e o teste do NFR-7"; `:36–39` as armadilhas de ensaio (a rede `internal: true` descartando as portas em silêncio, e `--no-deps`); `:45` "a Tabela de Pedidos não tem busca por número"; `:50` a metade "pessoa de fora" da SM-3; `:55` a rubrica do professor. A recompilação foi minha, no passo 1, e trocou conteúdo operacional por prosa por propósito | patch |
| BH1 + EC3 | `AZ-2026-000004` não aparece em evidência nenhuma, e a aritmética do Estoque precisa dele | **medium** | Real e explicável: a Maré aparece em **8** às 11:37:26, e a semente dá 10 (`20260912120100_catalogo_reserva_estoque.sql:12`); `maquina.go:132` consolida em `ENVIADO`, então cada Pedido entregue baixa o total. Um Pedido (a tentativa perdida do passo 3, `PAGO` 11:35:39 → `ENTREGUE` 11:36:52) consumiu a segunda unidade e ficou sem linha | patch |
| BH4 | Dois passos passaram na segunda tentativa, e a contagem 19/19 não diz | **medium** | Real: o passo 3 do Roteiro C perdeu a janela de 30 s na primeira tentativa, e a tabela de Contagem só tem aprovado/reprovado/não ensaiado. A SM-1 é sobre conduzir ao vivo e de uma vez | patch |
| BH3 | O passo 3 do Roteiro C não tem a leitura do Estoque durante a própria Reserva | **medium** | Real: registra 8 antes (11:37:26) e 8 depois (11:38:06), e empresta o 7 de 11:39:08, que é de outro Pedido. O comportamento está provado alhures (Roteiro B passos 3 e 5); o passo que a matriz encarrega dele, não | patch |
| BH5 | Roteiro C 1–4 passou com contorno que o §12 não contém | **medium** | Real: o roteiro como está escrito perde a Sessão do Comprador no login do painel; passou porque o ensaio usou perfis separados. 19/19 sem essa marca se lê como "o roteiro funciona como escrito" | patch |
| EC4 | A evidência do FR-32 não distingue painel de varredura | **medium** | Real: `Pago` 11:37:54 e `Separando` 11:37:55 no histórico, mas a interação do painel está narrada às 11:37:57 — a transição precede a ação que a teria causado. Ou o horário é de outro relógio, ou está transcrito errado; o entregável não diz qual | patch |
| EC5 | O HANDOFF tira da lista de "sem ver na tela" um cancelamento que não foi feito | **medium** | Real: a lista antiga nomeava "cancelar em `AGUARDANDO_PAGAMENTO` com o Produto de R$ 39,95"; o ensaio cancelou a partir de `SEPARANDO`, e o Pedido de R$ 39,95 expirou por tempo. O item saiu da lista sem ter sido visto | patch |
| BH8 | O `deferred-work.md` diz "append-only" e o diff reescreveu o `evidence` de uma entrada | **low** | Real: o `RESOLVIDO —` no `summary` tem precedente em `:55`, mas a leitura original (`grep` dando 1) foi sobrescrita pela de depois (0) — é justamente o que append-only protege | patch |
| BH2 | O Estoque foi medido pelo seletor de quantidade, que satura em 10 | **low** | Parcialmente real: `web/lib/quantidade.ts:5` tem `TETO_POR_ITEM = 10` e `caixa-de-compra.tsx:116` mostra "Em estoque" sem número a partir de 10, então "10" é "≥ 10". Mas 9 → 10 ainda prova que subiu, e é isso que o passo alega. É redação, não prova | patch |
| BH12 | Negrito aninhado quebra a armadilha 4 | **low** | Real: a ênfase interna em torno de "total" fecha a externa no meio da frase | patch |
| BH10 | O HANDOFF não diz o que fecha a épica | **low** | Parcialmente real: ele nomeia as cinco emendas e o ensaio humano como próximo passo, mas não a retrospectiva da Épica 7, que o `sprint-status.yaml` carrega como `optional` e é a última porta da épica | patch |
| BH13a | "Os quatro serviços no ar" seguido de três nomes | **low** | Real: o `web` não tem healthcheck e a frase deixa o leitor adivinhar | patch |
| BH13b | A armadilha 2 pede dois perfis de navegador; o ensaio usou três | **low** | Real: o segundo Comprador do passo 5 do Roteiro C é o terceiro perfil | patch |
| BH13c | O Registro de ensaios não tem coluna de commit | **low** | Real: `ddcc361` está espremido na célula da máquina, e o documento avisa que verde num dia não é verde no outro | patch |
| BH13d | `go test ./...` alegado "tudo ok" sem o comando que o invocou | **low** | Real: a seção Verification da spec nomeia uma invocação precisa, e o entregável cola só os subtestes de concorrência | patch |
| BH13e | A desmontagem que a tarefa exige não está registrada | **low** | Real no documento, verdadeiro no mundo: conferi que o clone do ensaio não existe mais e que não sobrou contêiner nem imagem `ensaio74`. Falta a linha | patch |
| EC12 | O Registro de ensaios não tem campo para reinício entre roteiros | **low** | Real: a matriz congelada exige que o Registro diga se houve reinício e por quê, e hoje isso só existe na prosa deste ensaio — as duas linhas humanas não têm onde declarar | patch |
| BH6 | O Roteiro A offline (NFR-15) fica sem condição de fechamento | **medium** | Real: é restrição da épica ("o Roteiro A percorre inteiro com a rede desconectada"), não foi refeito, e só existe como prosa no HANDOFF. Todo outro adiamento desta estória virou entrada com condição | defer |
| BH7 | A metade humana do ensaio e os responsáveis não têm dono nem prazo fora da prosa | **medium** | Real: são o maior pedaço de trabalho que a estória deixa aberto, e o `deferred-work.md` é onde o repositório guarda "o resto do que ficou adiado, com a condição de fechamento de cada um" | defer |
| EC14 | A tabela de fronteiras do deck não menciona `public.semente` | **low** | Real e pré-existente: o deck é de 2026-09-06 e a estória só tinha dois pontos nomeados para emendar | defer |
| BH11 | As quatro estórias da Épica 7 estão em `review` e nada as move a `done` | **false** | `review` é o estado terminal do desenvolvimento neste projeto: o cabeçalho do `sprint-status.yaml` diz "Dev moves story to 'review', then runs code-review". Quem move a `done` é a revisão humana, não esta estória | rejeitado |
| EC11 | A matriz congelada diz "o Pedido `PAGO` do Roteiro A" e a execução usou Pedidos novos | **low** | Rejeitado por duas razões: a leitura é única (o passo 9 do Roteiro A leva aquele Pedido a `ENTREGUE` de propósito, e em `ENTREGUE` não há transição nem cancelamento — nenhum outro caminho existe), e a correção pedida é editar a spec desta construção | rejeitado |
| EC2 | A corrida do passo 4b não foi seguida até `ENTREGUE` para ver o E4 | **low** | Rejeitado: o E4 continua listado como sem registro na seção de não cobertos e no HANDOFF. Um passo a mais que não foi dado não é defeito do que foi | rejeitado |
| EC13 | "são insumo da 7.4" saiu do HANDOFF sem os passeios terem sido consumidos | **false** | O ensaio foi construído do §12 do PRD, não dos passeios por estória, e a mesma frase continua dizendo que os roteiros de passeio das Épicas 5 e 6 estão no histórico. O ponteiro saiu porque a 7.4 deixou de estar pendente | rejeitado |


## Design Notes

O ensaio é documento de apresentação, não de setup: mora em `implementation-artifacts` ao lado do `percurso-do-addendum.md` porque é prova de métrica (SM-1) — o README continua sendo o caminho do clone ao sistema rodando, e aponta para cá. O navegador é o modo obrigatório: a 7.1 já percorreu os três desfechos pela API, e "sem erro visível" é afirmação sobre a tela. Defeito achado não se corrige aqui porque a Épica 7 verifica e não constrói: corrigir dentro do ensaio destruiria a evidência de que o roteiro reprovou.

## Verification

**Commands:**
- `docker run --rm -v "$PWD":/src -w /src -v azamon-gocache:/root/.cache -v //var/run/docker.sock:/var/run/docker.sock -e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal -e TESTCONTAINERS_RYUK_DISABLED=true azamon-dev:latest go test ./...` -- expected: tudo `ok`, inclusive o subteste da última unidade.
- `git diff --stat` -- expected: nenhum arquivo em `api/`, `internal/`, `cmd/`, `db/` ou `web/`.
