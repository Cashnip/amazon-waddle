---
title: 'Estória 5.1 — A máquina de estados do Pedido, completa e com testes'
type: 'feature'
created: '2026-09-19'
status: 'done'
baseline_commit: '48c8c27ec0054ebc08465145585248e251855657'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `pedido.Transicionar` é um CAS cru: aceita qualquer par de Status, não conhece a tabela do AD-3, não aplica efeito sobre o Estoque (o `Consolidar` mora no chamador), não grava `motivo` e só conhece uma das três recusas. E o código grafa `EM_SEPARACAO` onde o glossário, o AD-3 e a SPEC dizem `SEPARANDO` — decidido em 2026-09-19: **o código passa a `SEPARANDO`**.

**Approach:** Fazer da tabela de nove transições do AD-3 um dado em `internal/pedido`, consultado por `Transicionar(ctx, tx, pedidoID, esperado, novo, ator, motivo)` antes do CAS, e aplicar o efeito da terceira coluna dentro dele, depois do CAS e na mesma transação. Uma migração renomeia o Status, acrescenta `motivo`, troca `autor` por `ator` tipado e torna conteúdo e histórico imutáveis no próprio banco.

## Boundaries & Constraints

**Always:**
- A tabela é exaustiva: `(esperado, novo, ator)` fora dela é `ErrTransicaoInvalida`, **sem tocar o banco**, carregando `Permitidas` (destinos legais a partir de `esperado` para aquele ator).
- Ordem fixa: validar → CAS → histórico → efeito. Zero linhas é `ErrEstadoJaAvancado`. Efeitos: `→ PAGAMENTO_RECUSADO` e `→ CANCELADO` chamam `catalogo.Liberar` incondicionalmente; `PAGAMENTO_RECUSADO → AGUARDANDO_PAGAMENTO` chama `catalogo.Reservar` com os Itens do próprio Pedido e propaga a recusa; `SEPARANDO → ENVIADO` chama `catalogo.Consolidar`.
- Cancelamento reclassifica: `novo = CANCELADO` com `esperado` fora da janela, ou CAS perdido cujo Status relido está fora da janela (`ENVIADO`, `ENTREGUE`, `CANCELADO`), sai como `ErrForaDaJanelaDeCancelamento`.
- Atores: `COMPRADOR` (criação, nova Tentativa, cancelamento), `PROVEDOR` (aprovação, recusa), `VARREDURA` (expiração, só com `motivo = TEMPO_ESGOTADO`), `ADMINISTRADOR` e `SIMULACAO` (as três etapas da entrega).
- `EstadoTerminal(s Status) bool` continua a única noção de terminal: `ENTREGUE` ou `CANCELADO`.

**Ask First:**
- Criar rota HTTP ou tela nova, ou mudar o contrato JSON de `GET /api/v1/pedidos/{id}`.

**Never:**
- Cancelamento pelo Comprador, painel do Administrador, expiração na varredura, nova Tentativa ponta a ponta, `carrinho.Esvaziar`: são 5.6, 5.10, 5.11, 6.3 e 6.4. Aqui só a máquina que eles vão chamar.
- `correlacao` no histórico: `pedido` não importa `plataforma` (AD-1). Vai para o `deferred-work.md`.
- `UPDATE pedido.pedido SET status` fora de `AvancarStatus`, ou `pedido` tocando tabela de Estoque.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Avanço declarado | `PAGO → SEPARANDO`, `ADMINISTRADOR` | Status muda, linha nova com `ator` e `motivo` NULL | — |
| Fora da tabela | `CANCELADO → SEPARANDO`, `ADMINISTRADOR` | nada escrito | `TransicaoInvalida{Permitidas: []}` |
| Ator errado | `PAGO → SEPARANDO`, `COMPRADOR` | nada escrito | `ErrTransicaoInvalida` |
| Expiração sem motivo | `AGUARDANDO → RECUSADO`, `VARREDURA`, motivo vazio | nada escrito | `ErrTransicaoInvalida` |
| Corrida | CAS com `esperado` já superado | nada escrito | `ErrEstadoJaAvancado` |
| Cancelar enviado | `ENVIADO → CANCELADO` | nada escrito | `ErrForaDaJanelaDeCancelamento` |
| Cancelar que perdeu a corrida | `esperado = SEPARANDO`, Pedido já em `ENVIADO` | nada escrito | `ErrForaDaJanelaDeCancelamento`, nunca corrida |
| Nova Tentativa sem Estoque | `RECUSADO → AGUARDANDO`, disponível < quantidade | quem chamou reverte; Status fica `PAGAMENTO_RECUSADO` | `catalogo.ErrEstoqueInsuficiente` |
| Recusa e cancelamento | Pedido com Reserva ativa | Reserva `LIBERADA`, `estoque_total` intacto | — |
| Cancelar de recusado | sem Reserva ativa | `CANCELADO`, sem erro | `Liberar` devolve nil |

</frozen-after-approval>

## Code Map

- `internal/pedido/pedido.go` -- `Transicionar` (fim do arquivo), `Varrer`/`aplicarConfirmacao` (autor `provedor-pagamento`), `avancarEntrega` (chama `Consolidar` à mão — sai), constantes `Status*`, `EstadoTerminal`, `Criar` (histórico do nascimento com autor = compradorID).
- `internal/pedido/db/consultas.sql` -- `RegistrarTransicao`, `AvancarStatus`; faltam leitura do Status atual, Itens do Pedido para a nova Reserva e o histórico.
- `internal/pedido/pedido_test.go` -- molde do falso `pgx.Tx` (`txComLinhas`) para teste em memória.
- `db/migracoes/20260912120000_pedido_esqueleto.sql` -- CHECK com `EM_SEPARACAO`, coluna `autor`. Só lida: a mudança vai em migração nova.
- `internal/catalogo/catalogo.go:224,315,337` -- `Reservar`, `Liberar`, `Consolidar`, `ItemReserva`; o comentário do `Consolidar` cita `EM_SEPARACAO`.
- `internal/plataforma/erro/erro.go` -- `registro`; o `Escrever` recebe `dados`.
- `api/pedido.go:140` -- `pedido.EstadoTerminal(p.Status)`, passa a receber `Status`.
- `api/pedido_test.go:228,243,511,536,582` e `api/webhook_test.go:137` -- chamam `Transicionar` com a assinatura velha, leem `autor`, envelhecem o histórico por `UPDATE`, esperam `EM_SEPARACAO`.
- `api/sessao_test.go:177` -- `TestSessaoEProduto`: o contêiner compartilhado onde os subtestes de banco entram.
- `cmd/azamon/main_test.go:153,196,291` -- insere histórico com `autor`, cita `EM_SEPARACAO`.
- `web/app/pedidos/meus-pedidos.tsx:23`, `web/app/pedidos/[id]/acompanhamento.tsx:33`, `README.md:81` -- rótulo e texto de `EM_SEPARACAO`. O rótulo da EXPERIENCE é "Separando".

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/<timestamp>_pedido_maquina_de_estados.sql` -- (goose create) reescrever `EM_SEPARACAO` em `pedido` e `transicao_status`, trocar o CHECK; `motivo text NULL`; `autor` vira `ator` com CHECK nos cinco valores (backfill: `provedor-pagamento` → `PROVEDOR`, `simulacao-entrega` → `SIMULACAO`, o resto → `COMPRADOR`); triggers que recusam UPDATE de qualquer coluna de `pedido.pedido` que não seja `status`, DELETE de `pedido.pedido`, e UPDATE/DELETE de `item_pedido` e `transicao_status`. Down desfaz.
- [x] `internal/pedido/db/consultas.sql` + `sqlc generate` -- `RegistrarTransicao` com `ator` e `motivo`; `StatusDoPedido`, `ItensParaReserva`, `HistoricoDoPedido`. Restaurar os `gerado/` de outros módulos.
- [x] `internal/pedido/maquina.go` (novo) -- `Status`, `Ator`, `MotivoTempoEsgotado`, a tabela de nove linhas, `Permitidas`, `TransicaoInvalida`, os três sentinelas, `Transicionar` e `Historico`. O `Transicionar` sai do `pedido.go`.
- [x] `internal/pedido/pedido.go` -- `SEPARANDO`, `Status` tipado, `Varrer` com `AtorProvedor`, `avancarEntrega` com `AtorSimulacao` e sem `Consolidar` próprio, nascimento com `ator = COMPRADOR`.
- [x] `internal/pedido/maquina_test.go` (novo) -- em memória: exatamente nove linhas, varredura de 7×7×5 combinações recusadas sem Exec, `Permitidas`, a reclassificação e o CAS perdido (falso `Tx` com `QueryRow`).
- [x] `internal/plataforma/erro/erro.go` + teste -- `TRANSICAO_INVALIDA` e `FORA_DA_JANELA_DE_CANCELAMENTO` (409); `Escrever` preenche `dados.permitidas` por `errors.As` quando `dados` vier nil.
- [x] `api/maquina_test.go` (novo) + linha em `api/sessao_test.go` -- contra Postgres: efeito de cada transição sobre a Reserva e o total, nova Tentativa sem Estoque, histórico com `ator`/`motivo`, `Historico` em ordem, os triggers recusando UPDATE/DELETE, a reclassificação com Status relido.
- [x] `api/pedido_test.go`, `api/webhook_test.go`, `cmd/azamon/main_test.go`, `internal/catalogo/catalogo.go`, `api/pedido.go`, os dois `.tsx`, `README.md` -- nova assinatura, `ator`, `SEPARANDO`/"Separando"; envelhecer o histórico dentro de uma transação com `SET LOCAL session_replication_role = replica`.
- [x] `_bmad-output/implementation-artifacts/deferred-work.md` e `sprint-status.yaml` -- `correlacao` adiada; fechar o `epic-1-retro-item-1` como resolvido na direção inversa.
- [x] `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` + `.memlog.md` -- entrada da 5.1 no §10 e as três menções antigas a `EM_SEPARACAO` corrigidas -- a SM-7 manda a decisão tomada na construção voltar ao addendum (AGENTS.md); o `epic-1-retro-item-8` (`motivo`/`correlacao`) também fechou.

**Acceptance Criteria:**
- Given a tabela do AD-3, when a suíte roda, then ela tem exatamente nove linhas e toda outra combinação de Status e ator é recusada com erro explícito.
- Given um Pedido criado, when se tenta `UPDATE` de `total_centavos`, de um Item ou de uma linha do histórico, ou `DELETE` de qualquer um deles, then o banco recusa.
- Given `docker compose up` sobre volume antigo, when a migração roda, then todo `EM_SEPARACAO` gravado vira `SEPARANDO` e a simulação segue até `ENTREGUE`.

## Design Notes

A tabela é dado, e as três perguntas saem dela:

```go
type transicao struct{ de, para Status; atores []Ator; motivo string /* "" = qualquer um não reservado */ }
var transicoes = []transicao{ /* as nove, na ordem do AD-3; o cancelamento é uma linha com de ∈ janela */ }
```

`TEMPO_ESGOTADO` é motivo reservado: só a linha da expiração o aceita, senão a recusa do Provedor se passaria por ela. A criação é a linha `"" → AGUARDANDO_PAGAMENTO`: conta nas nove e não passa pelo CAS (`Transicionar` recusa `esperado` vazio). Para contar nove, o cancelamento é **uma** linha cujo `de` é um conjunto. Os triggers são a prova de "escrito uma vez" e de "imutável" que o código sozinho não dá. O custo é que o teste que envelhece o histórico precisa de `session_replication_role`, e o usuário do contêiner de teste é superusuário.

## Verification

**Commands:**
- `go vet ./... && go test ./...` na imagem `azamon-dev:latest` (receita do HANDOFF, com o socket do Docker) -- expected: verde, incluindo `TestSessaoEProduto` e `db/schema_test.go`.
- `docker run --rm -v "$PWD":/src -w /src sqlc/sqlc:1.31.1 generate` -- expected: só `internal/pedido/db/gerado` difere (`git diff --ignore-cr-at-eol --stat`).
- `docker build ./web` -- expected: build verde.
- `grep -rn EM_SEPARACAO --include=*.go --include=*.ts --include=*.tsx --include=*.md . | grep -v _bmad-output` -- expected: vazio, exceto a migração antiga.

## Suggested Review Order

**A tabela do AD-3 como dado**

- Porta de entrada: as nove linhas, com ator, motivo exigido e efeito de cada uma.
  [`maquina.go:112`](../../internal/pedido/maquina.go#L112)

- Validar → CAS → histórico → efeito; a recusa pela tabela nunca toca o banco.
  [`maquina.go:194`](../../internal/pedido/maquina.go#L194)

- O efeito da terceira coluna mora aqui, só para quem ganhou o CAS.
  [`maquina.go:231`](../../internal/pedido/maquina.go#L231)

- TEMPO_ESGOTADO é reservado à expiração; recusa do Provedor não se passa por ela.
  [`maquina.go:142`](../../internal/pedido/maquina.go#L142)

- CAS perdido no cancelamento relê o Status e reclassifica fora da janela.
  [`maquina.go:249`](../../internal/pedido/maquina.go#L249)

- Nova Tentativa reserva os Itens do próprio Pedido; Pedido sem Item é erro.
  [`maquina.go:269`](../../internal/pedido/maquina.go#L269)

**Quem chama a máquina**

- A simulação deixou de consolidar por conta própria; o efeito é do Transicionar.
  [`pedido.go:308`](../../internal/pedido/pedido.go#L308)

- A confirmação aplicada passa com o Provedor como ator.
  [`pedido.go:251`](../../internal/pedido/pedido.go#L251)

**Banco: renomeação, ator e imutabilidade**

- Backfill do ator: uuid vira COMPRADOR, valor desconhecido derruba a migração.
  [`20260919120000_pedido_maquina_de_estados.sql:51`](../../db/migracoes/20260919120000_pedido_maquina_de_estados.sql#L51)

- Só o Status muda no Pedido: comparação da linha inteira menos `status`.
  [`20260919120000_pedido_maquina_de_estados.sql:71`](../../db/migracoes/20260919120000_pedido_maquina_de_estados.sql#L71)

- TRUNCATE tem gatilho próprio; o de linha não dispara nele.
  [`20260919120000_pedido_maquina_de_estados.sql:107`](../../db/migracoes/20260919120000_pedido_maquina_de_estados.sql#L107)

- O Down declara o que perde e devolve os nomes que o código antigo lia.
  [`20260919120000_pedido_maquina_de_estados.sql:121`](../../db/migracoes/20260919120000_pedido_maquina_de_estados.sql#L121)

**Tradução para HTTP**

- Três códigos 409 distintos; `dados.permitidas` preenchido num ponto só, nunca `null`.
  [`erro.go:110`](../../internal/plataforma/erro/erro.go#L110)

**Testes e periferia**

- A tabela reescrita em texto e a varredura de todas as combinações.
  [`maquina_test.go:106`](../../internal/pedido/maquina_test.go#L106)

- CAS perdido não aplica efeito; o falso entraria em pânico se aplicasse.
  [`maquina_test.go:237`](../../internal/pedido/maquina_test.go#L237)

- Contra Postgres: efeitos por passo, recusas, histórico e gatilhos 23001.
  [`maquina_test.go:28`](../../api/maquina_test.go#L28)

- Envelhecer o histórico imutável exige o papel de réplica, só no teste.
  [`maquina_test.go:300`](../../api/maquina_test.go#L300)

- Volume antigo com EM_SEPARACAO sobe e desce pela migração.
  [`schema_test.go:257`](../../db/schema_test.go#L257)

- O rótulo de tela segue o glossário.
  [`acompanhamento.tsx:33`](../../web/app/pedidos/%5Bid%5D/acompanhamento.tsx#L33)
