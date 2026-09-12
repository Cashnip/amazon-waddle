---
title: 'Estória 1.7 — O Provedor Simulado confirma por webhook e o Pedido vai a PAGO'
type: 'feature'
created: '2026-09-12'
status: 'done'
route: 'dispatch'
baseline_commit: '6699518a6d1e5def9857efc91f1cbd9c887d8c8b'
review_loop_iteration: 0
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problema:** O Pedido nasce em `AGUARDANDO_PAGAMENTO` e fica lá para sempre. Não existe schema `pagamento`, não existe Tentativa de Pagamento, não existe inbox, e o binário não tem relógio nenhum — nada no repositório avança um Pedido depois que a requisição do Comprador termina. Sem isso a 1.8 não tem de onde partir e o caminho do gateway real nunca é exercitado.

**Abordagem:** Criar o schema `pagamento` com a Tentativa e a inbox de confirmação (esta com restrição única sobre a chave de idempotência), a porta de duas operações com o Provedor Simulado decidindo pelos centavos do total, o webhook que grava a confirmação, e a varredura de 1 s em `cmd/azamon` que aplica o que chegou e emite o que venceu. O Comprador confirma, vai para a tela do Pedido em processamento, e vê o Status virar `PAGO` sem recarregar.

## Boundaries & Constraints

**Always:**
- Nenhum `time.Timer`, `time.After` ou `time.AfterFunc` em módulo nenhum — inclusive no Provedor Simulado. O único relógio é o tique da varredura em `cmd/azamon`, que **não decide nada**: só chama, nesta ordem, `pedido.Varrer` (aplica) e `pagamento.EmitirConfirmacoesDevidas` (emite).
- `pagamento` **não conhece `pedido`**: a varredura de `pedido` lê a inbox por uma função de `pagamento` e aplica. A aresta contrária não existe e o teste de fronteira a prova.
- A porta tem **duas operações e mais nada**: iniciar a Tentativa e receber a confirmação. O total vai como valor — o adapter não consulta `pedido`.
- A confirmação entra por `POST /api/v1/webhooks/pagamento` e é gravada com **restrição única sobre a chave de idempotência**, determinística por Tentativa. A mesma confirmação recebida duas vezes produz um único efeito; o `23505` é sinal para o código decidir, nunca resposta ao cliente.
- Toda confirmação tem estado terminal: `PENDENTE` → `APLICADA` ou → `NAO_APLICAVEL_SINALIZADA`. Só é aplicada se pertence à Tentativa corrente **e** o Pedido está em `AGUARDANDO_PAGAMENTO`.
- Uma transação por Pedido na varredura, nunca uma por tique, com `FOR UPDATE SKIP LOCKED`. `ErrEstadoJaAvancado` é desfecho esperado — registrado e seguido em frente, nunca erro.
- Emissão é **derivada, não marcada**: Tentativa cujo atraso venceu e que não tem linha na inbox. Nenhuma chamada de rede dentro de transação aberta.
- `Transicionar` continua o único ponto de mutação do Status. `PAGO` mantém a Reserva de Estoque `ATIVA` — consolidar é da 1.8.
- Todo instante que a tela exibe é absoluto, RFC 3339, vindo do servidor. A consulta é de 3 s enquanto `AGUARDANDO_PAGAMENTO`.
- Nenhum dado de cartão é pedido, gerado ou guardado em ponto nenhum, nem no Simulado.
- **Decidido:** a faixa do §7.1 é lida sobre `total_centavos % 100` — é o que "pelos centavos do total" quer dizer, e é o que resolve a ambiguidade que a 1.1 deixou em `deferred-work.md` sobre `AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS`.
- **Decidido:** `pagamento` não pode importar `net/http` (módulo de domínio não conhece HTTP). `EmitirConfirmacoesDevidas` recebe a função de envio como parâmetro; quem a constrói com `http.Post` é `cmd/azamon`.
- **Decidido:** a Tentativa nasce dentro da mesma transação de `pedido.Criar`, estreando a aresta `pedido → pagamento` que a tabela do `AD-1` já previa.
- **Decidido:** o Simulado decide as três faixas do §7.1, mas a 1.7 **só emite a confirmação aprovada**. Um total com centavos em `,90`–`,94` (13 dos 50 Produtos semeados) fica em `AGUARDANDO_PAGAMENTO` até a Épica 5 trazer recusa e expiração — é o que "aqui só o caminho aprovado atravessa" quer dizer.
- **Decidido:** a tela do Pedido em processamento é rota nova no Next, `/pedidos/[id]`, e `comprar.tsx` passa a levar o Comprador até ela depois do 201.

**Never:**
- Sem recusa, sem nova Tentativa, sem teto de 3, sem expiração por tempo esgotado, sem aprovação sobre Pedido cancelado — tudo da Épica 5. Aqui só o caminho aprovado atravessa.
- Sem os passos `expirar` e `simular` da varredura, e sem avanço para `EM_SEPARACAO` em diante — 1.8.
- Sem superfície de operador, sem alternância em tempo de execução e sem reconfiguração entre roteiros.
- Sem enum do Postgres: estado e resultado são `text` com `CHECK`.
- Sem chave estrangeira cruzando schema: `pedido_id` em `pagamento` é `uuid` sem `REFERENCES`.
- Sem WebSocket e sem duração contada no navegador.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Saída esperada | Erro |
|---|---|---|---|
| Compra válida | `POST /api/v1/pedidos` com cookie válido | 201 como hoje, **e** uma Tentativa em `pagamento.tentativa_pagamento` na mesma transação | — |
| Confirmação aprovada | Tentativa de um total com centavos em `,00`–`,89`, atraso vencido | a varredura emite o POST, grava a confirmação, aplica, e o Pedido vai a `PAGO` com linha nova em `transicao_status` | — |
| Confirmação repetida | mesmo corpo, mesma chave, duas vezes | 200 nas duas; uma linha só na inbox; um único efeito | `23505` absorvido, nunca vazado |
| Confirmação sobre Pedido fora de `AGUARDANDO_PAGAMENTO` | Pedido já `PAGO` | a confirmação vira `NAO_APLICAVEL_SINALIZADA` e o Pedido fica intacto | — |
| Webhook com corpo inválido ou grande demais | JSON malformado | 400 | `ErrEntradaInvalida` |
| Webhook com chave desconhecida | chave sem Tentativa | 404, nada gravado | `ErrNaoEncontrado` |
| Leitura do Pedido | `GET /api/v1/pedidos/{id}` com cookie do dono | 200 com `numero`, `status`, `total_centavos` e o instante absoluto da última transição | — |
| Leitura por quem não é dono | cookie de outro Comprador, ou uuid malformado | 404 — não vaza existência | `ErrNaoEncontrado` |
| Leitura sem Sessão | sem cookie | 401 | `ErrSessaoInvalida` |
| Reinício no meio | contêiner reiniciado entre o commit e o POST | a emissão derivada reemite, e a chave única impede efeito duplo | — |
| Reserva depois do `PAGO` | Pedido aprovado | a Reserva continua `ATIVA`; nenhum total de Estoque muda | — |

</frozen-after-approval>

## Code Map

- `db/migracoes/20260912130000_pagamento_tentativa_e_confirmacao.sql` (novo) — convenção `YYYYMMDDHHMMSS_<modulo>_<assunto>.sql`, só os pragmas `-- +goose Up`/`-- +goose Down`, comentário em português no topo. `StatementBegin` nunca foi usado e não é preciso. Um arquivo só: o `schema:` do sqlc recorta por substring `*_pagamento_*`.
- `sqlc.yaml` — bloco novo idêntico em forma aos três existentes (`schema`, `queries`, `out`, `package: gerado`, `sql_package: pgx/v5`, `emit_exact_table_names: true`).
- `internal/pagamento/pagamento.go` — hoje só doc de pacote. Recebe a porta (`IniciarTentativa`, `RegistrarConfirmacao`, `ConfirmacoesNaoAplicadas`, marcação terminal e `EmitirConfirmacoesDevidas`), o Simulado e os sentinelas. Função pública recebe a `DBTX` do sqlc ou `tx pgx.Tx` como parâmetro nomeado — nunca `*gerado.Queries`.
- `internal/pedido/pedido.go` — `Criar` ganha a chamada a `pagamento.IniciarTentativa` dentro da transação; nasce `Varrer`, que abre uma transação por Pedido. `Transicionar` e `StatusInicial` já existem e não mudam; `PAGO` já está no `CHECK` da migração da 1.6.
- `internal/plataforma/erro/erro.go` — a fatia `registro` é ordenada de propósito. Acrescentar as linhas dos sentinelas novos e nada mais.
- `internal/fronteira_test.go` — `pedido → pagamento` e `cmd/azamon → pedido, pagamento` **já estão na tabela**. Só muda `"plataforma"`, e só se um sentinela de `pagamento` passar a ser traduzido.
- `db/schema_test.go` — quebra nos mesmos cinco pontos da 1.6: `schemasDeModulo` ganha `"pagamento"`; a lista literal de tabelas (hoje dez); o `len(tem) != 10` da chave primária; a heurística de plural; e a contagem de chaves estrangeiras cruzando schema, que continua zero. Helpers a reusar: `subirPostgres`, `conectar`, `textos`, `inteiro`.
- `api/rotas.go` — as rotas novas entram antes do catch-all `"/"`. `escreverJSON(w, status, valor)` é o único serializador de sucesso; `semCache` e `corpoMaximo` valem também aqui.
- `api/sessao.go` — `compradorDaRequisicao` já existe; a leitura do Pedido é a terceira rota autenticada. Continua sem middleware.
- `api/pedido.go` — o handler de criação não muda de forma; ganha vizinho `lerPedido`. O padrão de `pgx.Tx` com `defer tx.Rollback(context.WithoutCancel(...))` e `Commit` com `WithoutCancel` é o da 1.6 e vale para a varredura.
- `api/webhook.go` (novo) — o webhook não é autenticado por Sessão, e sim por um segredo compartilhado no cabeçalho, comparado em tempo constante (ver Implementation Notes). `http.MaxBytesReader` com `corpoMaximo`.
- `cmd/azamon/main.go` — o doc do pacote já anuncia a varredura. O goroutine entra depois do `pool`/`rdb` e antes do `ListenAndServe`, com `ticker` de `cfg.VarreduraIntervalo` e saída por `ctx.Done()`, no mesmo padrão do goroutine de `Shutdown`. É daqui que sai a função de envio com `http.Post` que `pagamento` recebe.
- `internal/plataforma/config.go` — `VarreduraIntervalo`, `ConfirmacaoAtraso`, `WebhookBaseURL`, `ProvedorAprovadoAteCentavos` e `ProvedorRecusadoAteCentavos` **já existem e já estão no `.env`**. Nenhuma variável nova; `env_test.go` continua verde sem mudança.
- `api/sessao_test.go` — `ambiente(t)` sobe um par de contêineres por pacote e `TestSessaoEProduto` é o único `Test*`; subtestes novos entram como `t.Run` ali. `cookieValido` já é capturado no primeiro subteste.
- `api/pedido_test.go` — `postarPedido`, `contarPedidos` e `textoDe` são reusáveis; `pegar` está em `api/produto_test.go`.
- `web/app/pedidos/[id]/page.tsx` (novo) e o filho `"use client"` — o molde é `web/app/produtos/[id]/page.tsx` com `saudacao.tsx`/`comprar.tsx`: Server Component busca por `AZAMON_API_URL`, filho `"use client"` consulta por caminho relativo com `credentials: "same-origin"`. O cabeçalho `chrome` com o `wordmark` se repete.
- `web/app/produtos/[id]/comprar.tsx` — o bloco de confirmação dá lugar à navegação para `/pedidos/<id>`; o `id` já vem no 201 da 1.6, que o incluiu justamente para isto.
- `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` + `.memlog.md` — decisão tomada durante a construção volta para o addendum na mesma estória.
- `_bmad-output/implementation-artifacts/deferred-work.md` — a entrada da ambiguidade de `AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS` é resolvida por esta estória e deve ser marcada como tal, no molde da entrada `RESOLVIDO` dos índices da 1.3.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/*_pagamento_tentativa_e_confirmacao.sql` -- Schema `pagamento` com `tentativa_pagamento` (Pedido, total congelado, id externo, número da tentativa, instante) e `confirmacao_recebida` (chave de idempotência `UNIQUE`, resultado, estado, instante), resultado e estado em `text` com `CHECK`.
- [x] `sqlc.yaml` + `internal/pagamento/db/consultas.sql` + `sqlc generate` -- Bloco do módulo e as consultas: criar Tentativa, gravar confirmação, listar não aplicadas, marcar terminal, e listar Tentativas cujo atraso venceu e que não têm linha na inbox.
- [x] `internal/pagamento/pagamento.go` -- A porta de duas operações, o Simulado decidindo por `total_centavos % 100` sobre as faixas da configuração, e `EmitirConfirmacoesDevidas` com a função de envio injetada.
- [x] `internal/pedido/pedido.go` -- `Criar` abre a Tentativa na mesma transação; `Varrer` lê a inbox, aplica sob `FOR UPDATE SKIP LOCKED`, uma transação por Pedido, e marca cada confirmação terminal.
- [x] `internal/pedido/db/consultas.sql` + `sqlc generate` -- Leitura do Pedido pelo dono com o instante da última transição, e a leitura travada da varredura.
- [x] `internal/plataforma/erro/erro.go` + `internal/fronteira_test.go` -- Traduzir os sentinelas novos e declarar a aresta que a tradução obrigar. **`erro.go` só:** o único sentinela novo é `ErrNaoAutorizado`, do próprio `plataforma/erro`, e nenhuma aresta muda — ver Implementation Notes.
- [x] `api/webhook.go` + `api/pedido.go` + `api/rotas.go` -- `POST /api/v1/webhooks/pagamento` e `GET /api/v1/pedidos/{id}`, com DTO, limite de corpo e `semCache`.
- [x] `cmd/azamon/main.go` -- O tique da varredura, com saída limpa por `ctx.Done()` e a função de envio do webhook.
- [x] `db/schema_test.go` -- Acompanhar as duas tabelas novas nos cinco pontos que quebram, e provar a restrição única sobre a chave de idempotência.
- [x] `api/webhook_test.go` + `api/pedido_test.go` -- Cobrir as linhas da matriz contra Postgres e Redis reais, incluindo a confirmação repetida e a leitura por quem não é dono.
- [x] `internal/pagamento/pagamento_test.go` -- Provar a decisão do Simulado nas fronteiras das faixas e que a chave de idempotência é determinística por Tentativa.
- [x] `web/app/pedidos/[id]/` + `web/app/produtos/[id]/comprar.tsx` -- A tela do Pedido em processamento com consulta de 3 s enquanto `AGUARDANDO_PAGAMENTO`, e o Comprador levado até ela depois do 201.
- [x] `.../addendum.md` + `.memlog.md` + `deferred-work.md` -- Registrar as decisões da estória e marcar resolvida a ambiguidade da faixa do Provedor.

**Acceptance Criteria:**
- Dado o código inteiro, quando procuro por `time.Timer`, `time.After` ou `time.AfterFunc` sob `internal/`, então não há nenhuma ocorrência.
- Dado um Comprador que confirma a compra de um Produto cujo total cai na faixa aprovada, quando espera na tela do Pedido, então o Status vira `PAGO` sem recarregar, e a Reserva continua `ATIVA`.
- Dada a interface de `pagamento`, quando inspeciono o pacote, então ele não importa `pedido` nem `net/http`, e o teste de fronteira prova as duas coisas.
- Dado `go test ./...` e `npm test` em `web/`, quando rodam, então passam, com fronteira, paridade do `.env` e schema intactos.
- Dado `sqlc generate` rodado duas vezes, quando comparo, então `git status` fica limpo na segunda.

## Implementation Notes

**O webhook é autenticado por segredo compartilhado, e a decisão veio da revisão.** A spec dizia que "quem o valida é a chave de idempotência contra a Tentativa", e isso era falso: `IDExterno` é `sim-<pedido_id>-1`, o `pedido_id` volta no 201 e vai na URL de acompanhamento, então o próprio Comprador derivava a chave. O furo foi reproduzido na pilha em execução — um Produto de R$ 39,90, cujos centavos caem na faixa que o Provedor nunca aprova, foi a `PAGO` com um POST que o navegador alcança. A correção é `AZAMON_WEBHOOK_SEGREDO` no cabeçalho, comparado com `subtle.ConstantTimeCompare` antes de qualquer gravação, e enviado por `cmd/azamon` junto com a confirmação. É o caminho que um gateway real usaria, que é justamente o que o `AD-7` diz que esta rota imita. A revisão roteou o achado como `bad_spec`; a decisão humana foi consertar no lugar, mantendo o código já verificado, em vez de reverter e re-derivar.

**Nenhum sentinela de `pagamento`, e por isso `fronteira_test.go` não mudou.** As quatro linhas de erro da matriz já tinham sentinela: `ErrEntradaInvalida`, `ErrNaoEncontrado` e `ErrSessaoInvalida`. Chave de idempotência sem Tentativa sai de `pagamento.RegistrarConfirmacao` como `pgx.ErrNoRows`, no mesmo padrão que `pedido.Criar` já usava para Produto inexistente, e `api/` a traduz em 404. O Code Map previa esta possibilidade ("só se um sentinela de `pagamento` passar a ser traduzido"); a aresta `plataforma → pagamento` não existe. `erro.go` ganhou um sentinela **do próprio `plataforma/erro`** — `ErrNaoAutorizado`, 401 — para o webhook sem segredo: falha de tradução, como `ErrEntradaInvalida`, e por isso sem aresta nenhuma.

**A tela do Pedido não busca no Server Component.** O molde de `produtos/[id]` não serve inteiro: a leitura do Pedido é autenticada, e o cookie de Sessão só existe no navegador. Buscar no servidor do Next exigiria reencaminhar o cabeçalho `Cookie` à mão — regra no Node, proibida pelo `AD-10`. `page.tsx` ficou só com a casca e o cabeçalho `chrome`; o filho `"use client"` `acompanhamento.tsx` faz a primeira leitura e a consulta de 3 s, as duas por caminho relativo com `credentials: "same-origin"`.

**O corpo do webhook é `pagamento.Confirmacao`, e `api/` não define DTO próprio.** Quem emite e quem recebe têm de concordar sobre o corpo; duas definições divergiriam na primeira mudança. As etiquetas `json` no tipo não dão HTTP ao módulo — quem serializa é a função de envio, em `cmd/azamon`.

**`resultado` fora de `APROVADO`/`RECUSADO` é barrado em `api/webhook.go`.** Sem a validação na fronteira de confiança, o valor desconhecido só falharia no `CHECK` da migração e um 400 sairia como 500.

**O cast `::timestamptz` em `BuscarPedidoDoComprador` é carga, não decoração.** Sem ele o sqlc não infere o tipo da subconsulta `max(ocorrido_em)` e emite `interface{}`, que falharia no `Scan` em tempo de execução.

**O identificador externo da Tentativa é derivado, não sorteado:** `sim-<pedido>-<numero>`. É dele que sai a chave de idempotência, e um sorteio faria da reemissão um efeito novo em vez de um no-op. O `numero` é sempre 1 nesta estória — nova Tentativa é da Épica 5, e a coluna já existe para que essa estória só precise incrementá-la.

**Ordem dos subtestes em `api/sessao_test.go`.** Os quatro subtestes novos entram depois dos da 1.6, e `leituraDoPedido` é o último: ele cria um Pedido de outro Comprador com número `AZ-0000-000001`, que antecede os do ano corrente na ordenação que os outros subtestes usam para escolher um Pedido.

**Auditoria da matriz (passo 3).** A linha "Compra válida" promete a Tentativa *na mesma transação*, e o teste só provava que ela existe. `pedidoSemEstoqueDa409` ganhou a contagem de Tentativas depois do 409: uma Tentativa órfã seria emitida pela varredura para um Pedido que nunca existiu.

**`git status` depois de `sqlc generate`.** `internal/catalogo/db/gerado/` e `internal/pedido/db/gerado/db.go` aparecem como modificados em worktree Windows porque o sqlc escreve LF e a cópia de trabalho estava em CRLF; `git diff` nesses arquivos é vazio — o conteúdo gerado é idêntico entre as duas execuções.

**A rota do webhook é autenticada por segredo compartilhado (revisão).** A chave de idempotência é derivada do identificador do Pedido, que o Comprador recebe no 201 e vê na URL de `/pedidos/<id>` — sozinha, ela deixava qualquer Comprador levar o próprio Pedido a `PAGO` sem pagar. `AZAMON_WEBHOOK_SEGREDO` é obrigatória e sem padrão, viaja em `X-Azamon-Segredo` e é comparada com `subtle.ConstantTimeCompare` antes de o corpo ser lido; ausente e errado saem no mesmo 401, sem gravar nada. É a única variável nova da estória — o Code Map dizia "nenhuma variável nova", e essa linha deixou de valer.

**`enviarConfirmacao` usa `&http.Client{Timeout: tempoDoWebhook}`.** O `http.DefaultClient` não tem prazo, e a emissão é serial dentro do goroutine da varredura: um POST pendurado congelaria aplicar e emitir. Constante de pacote, no precedente de `tempoDeCabecalho`.

**`atualizado_em` sai em `RFC3339Nano`.** Duas transições do mesmo Pedido cabem no mesmo segundo; com resolução de segundo, nem a tela nem o teste distinguem o nascimento do avanço.

**`cmd/azamon/main_test.go` exercita a varredura do binário.** O teste que já sobe `executar` contra Postgres real aponta `AZAMON_WEBHOOK_BASE_URL` para o próprio endereço, encurta `AZAMON_CONFIRMACAO_ATRASO` e espera o `status` virar `PAGO` — sem isto, apagar o `go varrer(...)` deixava a suíte inteira verde.

## Spec Change Log

## Review Triage Log

**Passagem 1** — caça-cega (12), casos de borda (12), lacunas de verificação (3 + 2 secundários).

| Verdito | Achado | Evidência | Rota |
|---|---|---|---|
| high | O webhook é forjável: o Comprador leva o próprio Pedido a `PAGO` sem pagar | `IDExterno` é `sim-<pedido_id>-1`, e o `pedido_id` volta no 201 e vai na URL `/pedidos/<id>`. A rota não tem segredo, assinatura nem restrição de origem — só a busca da Tentativa. Percorri o caminho: Sessão válida mais um `fetch("/api/v1/webhooks/pagamento")` do próprio navegador, atravessando o `rewrites()`, e a varredura aplica. O comentário que chama a chave de "quem valida" é falso: ela é derivável por quem criou o Pedido. | patch |
| high | O comentário de `api/rotas.go` e `api/webhook.go` afirma que a chave de idempotência valida a rota | Mesma raiz do achado acima: a chave não autentica ninguém quando o chamador a deriva. | patch |
| medium | `RECUSADO` é aceito pelo webhook e nada o consome; a linha na inbox suprime a emissão derivada | Verificado: `aplicarConfirmacao` marca terminal (`NAO_APLICAVEL_SINALIZADA`), então não fica parada — mas `TentativasSemConfirmacao` passa a excluir a Tentativa, e o Pedido nunca mais recebe confirmação. Só é alcançável por quem posta no webhook, ou seja, pela mesma brecha. | patch |
| medium | `enviarConfirmacao` usa `http.DefaultClient`, que não tem `Timeout` | Verificado em `cmd/azamon/main.go`: o envio é serial dentro do único goroutine da varredura, e o `ctx` só morre no encerramento. Um POST pendurado congela aplicar **e** emitir para o sistema inteiro, sem erro registrado. O precedente de `tempoDeCabecalho` diz que higiene de servidor é constante de pacote, não variável `AZAMON_`. | patch |
| medium | A varredura do binário e o transporte do webhook não são executados por teste nenhum | Pré-verificado pela lente de verificação, e confirmado: apagar `go varrer(...)` de `main.go`, ou renomear a rota em `rotas.go` mexendo só no literal de `webhook_test.go`, deixa `go test ./...` inteiro verde. `main_test.go` sobe o processo mas só afere porta e semente. É o comportamento-título da estória sem rede de proteção. | patch |
| medium | Qualquer resposta não-2xx para a consulta, inclusive um 500 transitório, para o polling para sempre | Verificado em `acompanhamento.tsx`: o ramo `!resposta.ok` sempre chama `clearInterval`. Um reinício do Go atrás do `rewrites()` congela a tela justamente na janela em que ela existe para esperar. O ramo `catch` já segue tentando — os dois discordam. | patch |
| medium | `atualizado_em` é afirmado só por analisar como RFC 3339, nunca por ser a última transição | Pré-verificado: trocar `max` por `min` na consulta, ou quebrar a correlação e devolver ano 0001, passa pela única asserção que existe. Nenhum teste lê a rota depois de o Status mudar. | patch |
| low | Nenhum índice sustenta a consulta de emissão | Verificado: `tentativa_pagamento` tem índice de `pedido_id` e o único de `id_externo`, e `TentativasSemConfirmacao` filtra e ordena por `criada_em`. Varredura sequencial a cada tique. Na escala da demonstração não se mede, mas a linha do índice é correção direta, e "índice antes de serviço" é a lição da 1.4. | patch |
| low | O comentário da migração promete que "a varredura nunca a deixa parada", e há um caminho em que ela fica | Verificado: `TravarPedido` devolve `ErrNoRows` tanto para "travado por outro tique" quanto para "o Pedido não existe", e o segundo caso deixaria a confirmação `PENDENTE` para sempre. O estado é inalcançável pelo código (só `pedido.Criar` cria Tentativa, na transação do Pedido), então o defeito é a promessa escrita, não o comportamento. Corrigir o comentário é a correção direta; a guarda protegeria estado não demonstrado. | patch |
| low | O instante é exibido cru ao Comprador: `2026-09-12T16:04:11Z` numa interface em pt-BR | Verificado em `acompanhamento.tsx`. A condição de aceite pede instante absoluto vindo do servidor, e o `dateTime` do `<time>` já guarda a forma canônica — formatar o filho não faz o navegador contar duração. | patch |
| low | As duas consultas da varredura não têm `LIMIT`, e toda Tentativa da faixa `,90`–`,94` é relida a cada segundo | Verificado, e é consequência direta da decisão de escopo registrada nesta spec: sem expiração, a Tentativa não aprovada nunca ganha linha na inbox. O filtro de faixa está no Go, depois da leitura. A Épica 5 o resolve trazendo expiração; o conserto agora acrescentaria parâmetro de lote por um custo que não se mede em 50 Produtos. | rejeitado |
| low | `TravarPedido` confunde "travado por outro tique" com "o Pedido não existe" | Mesma verificação da linha do comentário acima: o estado não é alcançável pelo código, e a guarda protegeria o que não foi demonstrado. | rejeitado |
| low | `max(ocorrido_em)` é NULL para Pedido sem transição, e a API devolveria `0001-01-01T00:00:00Z` | Verificado: todo Pedido criado por `pedido.Criar` grava a transição de nascimento na mesma transação. Só SQL direto produz o estado, e nesse caso o dono não bate e a rota já responde 404. Guarda por estado não demonstrado. | rejeitado |
| low | A lista de `resultado` válido mora só na borda HTTP, e não em `RegistrarConfirmacao` | Verificado: existe um chamador só, e ele valida. A guarda no módulo protegeria um segundo chamador que não existe. | rejeitado |
| low | O `setInterval` não trava requisição em voo: resposta acima de 3 s pode sobrescrever estado novo | Verificado como real, mas exige resposta de mais de 3 s de uma API em loopback, e o conserto acrescenta guarda. Fica registrado: se acontecer, a tela mostra `AGUARDANDO_PAGAMENTO` depois de já ter visto `PAGO`, e sem novas consultas. | rejeitado |
| low | O goroutine da varredura não é esperado no encerramento, e `pool.Close()` pode rodar com transação aberta | Verificado: a transação aborta do lado do servidor, que é reversão — sem perda de dado, e a confirmação `PENDENTE` é reaplicada no próximo arranque, que é justamente o desenho. O conserto acrescenta canal e espera. | rejeitado |
| low | `aplicarConfirmacao` registra pelo `slog` de pacote em vez do logger injetado | Verificado: `slog.SetDefault(raiz)` faz a linha sair com `modulo=plataforma` em vez de `varredura`. Incômodo de consulta de log; o conserto acrescenta parâmetro a função de domínio. | rejeitado |
| low | `pgx.ErrNoRows` é o contrato de "não encontrado" atravessando fronteira de módulo | Verificado como real, e **pré-existente**: a 1.6 já o fazia em `pedido.Criar` sobre `catalogo.BuscarProduto`. Não foi causado por esta estória. | defer |
| low | A tela de acompanhamento e a navegação pós-compra não têm verificação nenhuma | Pré-verificado: `web/` tem `node --test` sobre `.mjs` e nenhuma bancada que renderize componente. Fechar isto é decidir por uma pilha de teste de DOM, ou extrair as decisões para ajudantes puros — decisão maior que esta estória. | defer |
| false | A porta exporta seis símbolos, e a spec fala em "duas operações" | O `AD-8` fala das duas operações **do Provedor**; o `AD-6` nomeia `pagamento.EmitirConfirmacoesDevidas` como passo 4 da varredura e o `AD-7` manda a inbox ser lida por `pagamento`. Os outros símbolos são exatamente o que esses dois invariantes exigem. | rejeitado |
| false | A região `role="status"` que anunciava "Pedido criado" foi removida sem substituta | `Acompanhamento` monta o `<p role="status">` vazio desde o primeiro render e só depois o preenche — a mudança acontece numa região viva já no DOM, que é o padrão correto. O bloco removido de `comprar.tsx` ficava numa tela da qual o Comprador sai. | rejeitado |

## Design Notes

**Por que a emissão é derivada e não marcada:** marcar "já emiti" exigiria escrever depois do POST, e a queda entre o commit e a escrita deixaria a Tentativa sem confirmação para sempre. Derivar de "venceu o atraso e não tem linha na inbox" faz da reemissão o comportamento normal, e a chave única transforma o reenvio em no-op.

**Por que a função de envio é injetada:** `TestDominioNaoConheceHTTP` proíbe `net/http` em módulo de domínio, e o `AD-6` põe a emissão em `pagamento`. As duas regras convivem se o transporte entrar como parâmetro — `cmd/azamon` já é quem monta o servidor e pode falar HTTP.

## Verification

**Commands:**
- `docker run --rm -v "${PWD}:/src" -w /src sqlc/sqlc:1.31.1 generate` -- expected: `internal/pagamento/db/gerado/` criado e `internal/pedido/db/gerado/` atualizado sem erro.
- `docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v "${PWD}:/src" -v azamon-gomod:/go/pkg/mod -w /src --network host -e TESTCONTAINERS_RYUK_DISABLED=true golang:1.27.1 go test -count=1 ./...` -- expected: tudo verde, incluindo fronteira, paridade do `.env` e o schema.
- `cd web && npm test && npm run build` -- expected: verificador offline e teste da casca passam, e a construção conclui.

**Manual checks (if no CLI):**
- `docker compose up`, entrar, comprar um Produto da faixa aprovada e ver a tela do Pedido virar `PAGO` sozinha; no `psql`, uma linha em `pagamento.confirmacao_recebida` com estado `APLICADA` e duas em `pedido.transicao_status`.
