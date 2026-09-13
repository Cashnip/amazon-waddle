---
title: 'Estória 1.8 — A varredura leva o Pedido até ENTREGUE'
type: 'feature'
created: '2026-09-12'
status: 'done'
route: 'dispatch'
baseline_commit: '1250a5f55c9a1871fe8da5bea4a315fc9d326d9d'
review_loop_iteration: 0
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problema:** A 1.7 deixou o Pedido em `PAGO` com a Reserva de Estoque ainda `ATIVA`, e nada no repositório o move dali: o tique tem só `aplicar` e `emitir`, `catalogo` não sabe consolidar nem baixar Estoque, e a tela para de consultar assim que o Status sai de `AGUARDANDO_PAGAMENTO`. O ciclo fechado que é a razão de existir da épica nunca chega ao fim.

**Abordagem:** Acrescentar o passo `simular` ao tique existente — ele deriva do histórico "está neste estado desde quando" e avança `PAGO` → `EM_SEPARACAO` → `ENVIADO` → `ENTREGUE` —, dar a `catalogo` a consolidação que baixa o Estoque total em `EM_SEPARACAO → ENVIADO`, e estender a consulta da tela para 10 s enquanto o estado não for terminal, para que o Comprador veja a chegada a `ENTREGUE`.

## Boundaries & Constraints

**Always:**
- Nenhum `time.Timer`, `time.After` ou `time.AfterFunc` em módulo nenhum. O relógio continua sendo o único `time.Ticker` de `cmd/azamon`, que **não decide nada**: passa a chamar, nesta ordem, `pedido.Varrer` (aplicar) → `pedido.SimularEntrega` (simular) → `pagamento.EmitirConfirmacoesDevidas` (emitir).
- Todo avanço **deriva do histórico** de `pedido.transicao_status` — "está neste estado desde quando" —, nunca de estado em memória. Reiniciar o contêiner retoma cada Pedido de onde parou.
- Uma transação **por Pedido**, com `FOR UPDATE SKIP LOCKED`; `ErrEstadoJaAvancado` é desfecho esperado, registrado e seguido em frente. O molde é o `Varrer` da 1.7: candidatos lidos fora da transação, decisão reconferida dentro dela.
- `Transicionar` continua o único ponto de mutação do Status. A transição vem **antes** do efeito sobre o Estoque, na mesma transação.
- **`EM_SEPARACAO → ENVIADO` é a única transição que altera o Estoque total.** Consolidar baixa `catalogo.produto.estoque_total` e encerra a Reserva; nenhuma outra toca em quantidade.
- O intervalo entre etapas vem de `cfg.EntregaIntervalo` e a simulação é desligável por `cfg.EntregaSimulacaoAtiva` — **as duas já existem, já estão no `.env` e não são usadas por código de produção nenhum.** Nenhuma variável nova.
- `catalogo.Consolidar` sobre Pedido sem Reserva `ATIVA` é **no-op que devolve `nil`**, nunca erro (armadilha registrada no `AGENTS.md`).
- A baixa de Estoque percorre os Produtos **em ordem de identificador**, pelo mesmo motivo que o `ORDER BY id FOR UPDATE` do `Reservar`: dois Pedidos com os mesmos Produtos em ordem inversa travariam um no outro.
- Existe **uma função só** para "terminal", `pedido.EstadoTerminal`, e terminal é `ENTREGUE` ou `CANCELADO` e mais nada (`AD-18`). A resposta do Pedido carrega o resultado dela — a tela não redeclara a regra.
- **Decidido:** `expirar` **não** é desta estória. As condições de aceite da épica listam o tique como `aplicar → expirar → simular → emitir`, mas a expiração da Tentativa é inteira da Estória 5.11 — que a transita para `PAGAMENTO_RECUSADO` com `TEMPO_ESGOTADO` e libera a Reserva. Aqui o lugar dela fica reservado no comentário da ordem, entre aplicar e simular.
- **Decidido:** `TravarPedido` passa a devolver também o instante da última transição, em vez de nascer uma segunda consulta travada quase idêntica. `aplicarConfirmacao` acompanha a troca em uma linha.
- **Decidido:** a tela ganha a consulta de 10 s enquanto o estado não for terminal, que o `AD-18` já declara para esta mesma superfície. Sem ela nada na interface chega a exibir `ENTREGUE`, que é o objetivo da épica.
- **Decidido:** o autor das transições da simulação é `simulacao-entrega`; o `autorDaVarredura` da 1.7 é renomeado para `autorDoProvedor`, que é o que ele sempre significou.

**Never:**
- Sem migração. O `CHECK` da 1.6 já declara os sete Status e a 1.7 fechou o último schema da épica — criar tabela, coluna ou índice aqui é defeito de sequenciamento.
- Sem liberação de Reserva, sem cancelamento, sem painel do Administrador, sem avanço manual — Épicas 3 e 6. Só a consolidação entra.
- Sem as nove transições nomeadas, sem recusa que nomeia o permitido, sem expiração — Épica 5.
- Sem `catalogo` conhecendo `pedido`; a aresta é `pedido → catalogo` e já existe na tabela de fronteira.
- Sem tocar em `internal/plataforma/config.go` nem no `.env`: `env_test.go` compara as duas listas nos dois sentidos e já está verde.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Saída / Comportamento esperado | Tratamento de erro |
|---|---|---|---|
| Avanço normal | Pedido em `PAGO`, última transição há mais que o intervalo | vai a `EM_SEPARACAO`, com linha nova em `transicao_status` e autor `simulacao-entrega`; Estoque total intacto | — |
| Consolidação | Pedido em `EM_SEPARACAO`, intervalo vencido | vai a `ENVIADO`, a Reserva passa a `CONSOLIDADA` e `estoque_total` baixa exatamente a quantidade reservada, na mesma transação | — |
| Fim da linha | Pedido em `ENTREGUE` | a varredura não o seleciona; nenhuma transição, nenhum efeito | — |
| Cedo demais | Pedido em `PAGO` há menos que o intervalo | não é selecionado; nada acontece | — |
| Não elegível | Pedido em `AGUARDANDO_PAGAMENTO` ou `CANCELADO` | nunca é movido pela simulação, independentemente do tempo decorrido | — |
| Desligada | `AZAMON_ENTREGA_SIMULACAO_ATIVA=false` | o passo não roda; aplicar e emitir seguem normais | — |
| Corrida | dois tiques sobrepostos sobre o mesmo Pedido | o segundo não trava a linha ou recebe `ErrEstadoJaAvancado` | desfecho esperado: registrado em nível informativo, a varredura segue |
| Sem Reserva ativa | consolidar Pedido cuja Reserva já é `CONSOLIDADA` | no-op; a transição acontece e o Estoque não baixa duas vezes | devolve `nil`, nunca erro |
| Reinício | contêiner derrubado entre `PAGO` e `ENVIADO` | ao voltar, o Pedido retoma do Status gravado e chega a `ENTREGUE` sem intervenção | — |
| Leitura da tela | `GET /api/v1/pedidos/{id}` com cookie do dono | a resposta carrega `terminal` além de `status` e `atualizado_em` | inalterado: 404 para quem não é dono |

</frozen-after-approval>

## Code Map

- `internal/catalogo/db/consultas.sql` — nascem `ConsolidarReservasDoPedido` (`:many`, `UPDATE … SET estado='CONSOLIDADA' WHERE pedido_id=$1 AND estado='ATIVA' RETURNING produto_id, quantidade`) e `BaixarEstoqueTotal` (`:exec`). As quatro existentes não mudam. `sqlc generate` depois.
- `internal/catalogo/catalogo.go` — nasce `Consolidar(ctx, tx pgx.Tx, pedidoID string) error`, vizinha de `Reservar` (L79). Zero linhas devolvidas é `nil`. Ordenar as linhas devolvidas por `produto_id` antes de baixar. O doc de pacote (L4-6) diz que consolidação é da Épica 3 — atualizar para dizer que a consolidação chegou aqui e só a liberação e o predicado de visibilidade seguem lá.
- `internal/pedido/db/consultas.sql` — `TravarPedido` (L52) ganha o instante da última transição no `RETURNING`/`SELECT`, no mesmo molde do `max(t.ocorrido_em)` que `BuscarPedidoDoComprador` (L41) já usa. Nasce `PedidosParaAvancar` (`:many`): identificadores cujo Status está na lista elegível e cuja última transição é anterior ao corte. `sqlc generate` depois.
- `internal/pedido/pedido.go` — nascem `SimularEntrega(ctx, pool *pgxpool.Pool, intervalo time.Duration) error` (molde do `Varrer`, L~180), o passo privado por Pedido (molde do `aplicarConfirmacao`), `proximoDaSimulacao(status) (string, bool)` com as três transições, `EstadoTerminal(status string) bool`, e as constantes `StatusEmSeparacao`, `StatusEnviado`, `StatusEntregue`, `StatusCancelado`, `autorDaSimulacao`. `autorDaVarredura` (L41) vira `autorDoProvedor`. `aplicarConfirmacao` acompanha o novo retorno de `TravarPedido` em uma linha. `Transicionar` (L242) e `Criar` não mudam.
- `cmd/azamon/main.go` — dentro do `case <-tique.C:` (L167-174), entre `pedido.Varrer` e `pagamento.EmitirConfirmacoesDevidas`, o passo novo guardado por `cfg.EntregaSimulacaoAtiva`. Sem `ticker` novo. O doc de `varrer` (L145-150) anuncia a lista e precisa passar a dizer `aplicar → simular → emitir`, com `expirar` marcado para a 5.11.
- `api/pedido.go` — o DTO de leitura (L27-38) ganha `Terminal bool \`json:"terminal"\``, preenchido por `pedido.EstadoTerminal`. `lerPedido` não muda de forma.
- `web/app/pedidos/[id]/acompanhamento.tsx` — `intervalo` (L9) vira dois valores: 3000 enquanto `AGUARDANDO_PAGAMENTO`, 10000 depois. O `clearInterval` da L79 passa a disparar por `corpo.terminal`, não por sair de `aguardando`. O `rotulo` (L21-24) tem só dois Status e precisa dos cinco restantes — os sete do `CHECK` inteiros, ainda que recusado e cancelado só sejam alcançáveis nas Épicas 5 e 6. O `aria-live` da mudança de Status já existe e vale para as novas viradas.
- `internal/pedido/pedido_test.go` — já existe; é onde `proximoDaSimulacao` e `EstadoTerminal` são provados em memória, sem contêiner.
- `cmd/azamon/main_test.go` — `varreduraLevaOPedidoAPago` (L113) é o molde exato: insere Pedido e Tentativa direto no banco e consulta até o Status virar. O teste novo encurta `AZAMON_ENTREGA_INTERVALO` no `TestExecutarDeixaOCatalogoSemeado` (L42) e consulta até `ENTREGUE`, conferindo Reserva `CONSOLIDADA` e `estoque_total` menor.
- `db/schema_test.go`, `internal/fronteira_test.go`, `internal/plataforma/env_test.go` — **nenhum muda.** Sem tabela nova, `pedido → catalogo` já está na tabela de arestas (L32), e nenhuma variável nova.
- `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` + `.memlog.md` — a decisão sobre `expirar` pertencer à 5.11 e a consolidação ter chegado antes da Épica 3 voltam para o §10 nesta estória.

## Tasks & Acceptance

**Execution:**
- [x] `internal/catalogo/db/consultas.sql` + `sqlc generate` -- As duas consultas da consolidação.
- [x] `internal/catalogo/catalogo.go` -- `Consolidar`, com no-op sobre Pedido sem Reserva ativa e baixa em ordem de identificador; doc de pacote atualizado.
- [x] `internal/pedido/db/consultas.sql` + `sqlc generate` -- `PedidosParaAvancar` e o instante no `TravarPedido`.
- [x] `internal/pedido/pedido.go` -- `SimularEntrega`, as três transições, `EstadoTerminal`, as constantes de Status e o renome do autor.
- [x] `cmd/azamon/main.go` -- O passo `simular` no tique, guardado pelo interruptor, e o doc da ordem.
- [x] `api/pedido.go` -- `terminal` no DTO de leitura do Pedido.
- [x] `web/app/pedidos/[id]/acompanhamento.tsx` -- Os dois intervalos, a parada por `terminal` e os rótulos dos cinco Status que faltam.
- [x] `internal/pedido/pedido_test.go` -- As três transições, o que a simulação recusa a mover, e `EstadoTerminal`, em memória.
- [x] `cmd/azamon/main_test.go` -- O Pedido chega a `ENTREGUE` sozinho contra Postgres real, com Reserva `CONSOLIDADA` e Estoque baixado.
- [x] `.../addendum.md` + `.memlog.md` -- As decisões da estória.

**Acceptance Criteria:**
- Dado o código inteiro, quando procuro `time.Timer`, `time.After` ou `time.AfterFunc` sob `internal/`, então não há nenhuma ocorrência.
- Dado um Pedido que chega a `PAGO` com a simulação ligada e intervalo curto, quando espero, então ele chega a `ENTREGUE` sozinho, com uma linha em `transicao_status` por avanço, e o Comprador vê a virada na tela sem recarregar.
- Dado um Produto com `estoque_total` conhecido e um Pedido que atravessa `EM_SEPARACAO → ENVIADO`, quando comparo antes e depois, então o total baixou exatamente a quantidade reservada, uma única vez, e nenhuma outra transição o alterou.
- Dado o contêiner derrubado com o Pedido em `EM_SEPARACAO` e subido de novo, quando espero, então ele chega a `ENTREGUE` sem intervenção.
- Dado `go test ./...` e `npm test` em `web/`, quando rodam, então passam, com fronteira, paridade do `.env` e schema intactos e sem mudança nenhuma nesses três testes.
- Dado `sqlc generate` rodado duas vezes, quando comparo, então `git status` fica limpo na segunda.

## Implementation Notes

**Auditoria da matriz de I/O forçou dois testes que a lista de tarefas não previa.** Quatro das dez linhas não tinham teste que rodasse: *cedo demais*, *desligada*, *sem Reserva ativa* e *leitura da tela*. As três primeiras e a última ganharam cobertura assim:

- `api/pedido_test.go` — `simulacaoDeEntrega`, registrado como último subteste de `TestSessaoEProduto`. Usa o ambiente que já existe: o Pedido nasce pelo caminho real da compra, então a Reserva é de verdade. Cobre *cedo demais* (intervalo de uma hora sobre transição recém-feita), as três etapas uma a uma, a baixa única do Estoque, *sem Reserva ativa* (segunda chamada a `Consolidar` devolve `nil` e o Estoque não baixa de novo) e *leitura da tela* (`terminal` falso em `PAGO`, verdadeiro em `ENTREGUE`). Envelhecer o histórico com `ocorrido_em - interval '1 hour'` substitui a espera — e prova de passagem que a decisão vem da tabela, que é a linha *reinício*.
- `cmd/azamon/main_test.go` — `interruptorDesligaASimulacao`. A configuração é lida uma vez no arranque, então o interruptor só é observável subindo o binário de novo; sobe no mesmo Postgres, que é a parte cara. O `defer parar(); <-fim` do primeiro arranque virou um `pararEsperando` idempotente, para que o segundo binário não concorra com o primeiro.

**Mutação conferida:** trocar `if cfg.EntregaSimulacaoAtiva` por `if true` faz `TestExecutarDeixaOCatalogoSemeado` falhar com `status = ENTREGUE com a simulação desligada; quero PAGO parado`. O teste não é vazio.

**Linhas cobertas por construção, sem teste próprio:** *corrida* (a metade do compare-and-swap está em `TestTransicionarEhCompareAndSwap`; o `SKIP LOCKED` não tem como ser exercitado sem dois processos) e *reinício* (nenhum estado em memória existe para sobreviver — `SimularEntrega` é uma função sem memória, e `simulacaoDeEntrega` a chama do zero a cada etapa, que é o mesmo que reiniciar entre elas).

**Um estrago meu no caminho:** editei `cmd/azamon/main.go` pelo PowerShell e o `Set-Content` releu o arquivo como ANSI, gravando UTF-8 duplo — todos os acentos viraram mojibake, mais um BOM. Restaurado do git e reaplicado por Python com codificação explícita. **Nunca editar arquivo acentuado deste repositório pelo PowerShell.**

## Spec Change Log

## Review Triage Log

| Veredito | Achado | Evidência | Rota |
|---|---|---|---|
| medium | Nada prende a etapa em que o Estoque baixa; só que baixou uma vez | Demonstrado pela camada de verificação: mover `Consolidar` para qualquer outra transição mantém tudo verde, porque ele é idempotente e as duas asserções só olham o fim da cadeia. A Reserva pode ser consumida uma etapa cedo enquanto código e addendum afirmam que ela ainda está segura | patch |
| medium | `simulacaoDeEntrega` é global e fica à mercê do relógio de parede | `SimularEntrega` avança todo candidato do banco, mas `envelhecer()` só envelhece o Pedido do teste. Com intervalo de um minuto, qualquer Pedido alheio em `PAGO` com Reserva `ATIVA` sobre o mesmo Produto entra junto assim que a suíte passar de um minuto, e a asserção `antes-1` quebra sem defeito nenhum | patch |
| medium | A espera da 1.7 virou corrida por causa desta estória | Verificado em `main_test.go:232`: o laço quebra só em `status == "PAGO"`, e agora a simulação a 200 ms sobre um tique de 1 s tira o Pedido de `PAGO` em ~1 tique. Máquina carregada perde a janela e o teste falha com "ficou em ENTREGUE" sem nada ter regredido | patch |
| low | O `epic-1-context.md` deste mesmo diff ainda diz que a 1.8 traz `expirar` | Verificado: duas passagens contradizem o addendum escrito no mesmo commit e o `main.go`, que já lê `aplicar → simular → emitir`. É o primeiro arquivo que a próxima estória abre | patch |
| low | `avancarEntrega` ignora `travado.Desde.Valid` | Verificado: `Desde` é anulável e um NULL vira o tempo zero, que nunca é `After(ate)` — a guarda passa. Inalcançável hoje, porque `Criar` sempre grava a primeira transição, mas as duas consultas só concordam por acidente: a de candidatos exclui NULL, a releitura não | patch |
| low | O comentário de ordenação dos subtestes nomeia a dependência errada | Verificado: `pedidoSemEstoqueDa409` usa `produtoParaEsgotar`, outro Produto. O acoplamento real é com quem compra `produtoSemeado` | patch |
| low | A fixação em `PAGO` é `UPDATE pedido SET status` — o comando que o AD-6 diz não existir | Verificado no teste novo: Status e histórico entram por duas escritas independentes, que podem divergir do que `Transicionar` de fato produz | patch |
| low | `PedidosParaAvancar` varre sem `LIMIT` e sem índice, a cada tique | Verificado: `pedido.pedido` só tem `pedido_comprador_id_idx`, e a subconsulta correlata roda por linha. Irrelevante na escala da demonstração (dezenas de Pedidos) e a épica proíbe migração nova a partir daqui — o que cabe é registrar o teto | patch |
| low | `interruptorDesligaASimulacao` recebe `dsn` e não usa | Verificado: a DSN já está no ambiente por `t.Setenv`. Parâmetro morto | patch |
| false | `sprint-status.yaml` ficou em `in-progress` | Não é deriva: o passo 5 do próprio fluxo é que sincroniza, e a revisão ainda estava correndo quando o diff foi tirado | rejeitado |
| false | `BaixarEstoqueTotal` descarta a contagem de linhas e pode não achar o Produto | Refutado na migração: `reserva_estoque.produto_id` é `NOT NULL REFERENCES catalogo.produto (id)`. A linha que a consolidação devolve tem Produto por construção — não existe o caso de zero linhas | rejeitado |
| false | `PAGAMENTO_RECUSADO` consulta para sempre | Refutado pelo `AD-18`: terminal é `ENTREGUE` ou `CANCELADO` "e mais nada", e a `EXPERIENCE.md` diz explicitamente que parar no recusado "deixaria um relógio morto na tela". A Épica 5 traz nova Tentativa, que a tela precisa ver chegar | rejeitado |
| false | `reprogramar` rearma a consulta depois do `clearInterval` do 401/404 | Refutado em `acompanhamento.tsx:93`: o ramo do erro faz `clearInterval(timer)` e **`return`** — não cai no `reprogramar`. E uma resposta em voo que resolvesse depois acharia `atual` igual e sairia cedo | rejeitado |
| low | `AZAMON_ENTREGA_INTERVALO` zero ou negativo faz tudo avançar por tique | Real, mas só por configuração errada de uma variável de demonstração, e a correção mora em `config.go`, que o **Never** desta spec fecha explicitamente | rejeitado |
| low | `Consolidar` devolve `nil` para Pedido que nunca teve Reserva | Inalcançável: `Criar` cria a Reserva na mesma transação do Pedido. A correção proposta acrescenta consulta e ramo para um estado que não existe | rejeitado |
| low | `time.Sleep(2s)` e porta fixa em `interruptorDesligaASimulacao` | Portas fixas são a convenção do arquivo (8098, 8099). Os dois segundos são o preço de provar uma ausência, e encurtar troca custo por complexidade | rejeitado |
| low | A condição de aceite do `time.Timer` casa com um comentário | A correção seria editar a spec desta construção, o que a triagem não permite. Verificado que o único acerto é o comentário de `migracao.go:69`, que diz que `time.After` está proibido | rejeitado |
| maybe-false | A releitura travada de `Desde` nunca é exercitada | Pré-verificado pela camada de verificação: só dois processos sobre o mesmo Postgres alcançam o ramo, e `internal/pedido` não mantém contêiner de propósito. Na demonstração roda um contêiner só | defer |
| maybe-false | A tela não tem verificação nenhuma do novo critério de parada | Pré-verificado: `web/` só tem `node --test` sobre `scripts/`, sem jsdom nem playwright. Fechar significa trazer um arranjo de teste de componente que o repositório nunca teve | defer |

## Design Notes

**Por que candidatos fora da transação e decisão dentro dela:** é o molde que o `Varrer` da 1.7 já estabeleceu. Selecionar e travar na mesma consulta faria uma transação por tique em vez de uma por Pedido, e um Pedido que falhasse desfaria o avanço de todos os outros. A releitura dentro da transação é o que torna a corrida entre dois tiques inofensiva.

**Por que o instante entra no `TravarPedido` em vez de uma consulta nova:** a decisão de avançar precisa do Status e de "desde quando", travados juntos. Duas consultas travadas quase idênticas seriam duas oportunidades de divergirem; a junção com `max(t.ocorrido_em)` já existe no `BuscarPedidoDoComprador` e é a mesma aqui.

**Por que ordenar por identificador antes de baixar o Estoque:** é a mesma disciplina do `ORDER BY id FOR UPDATE` do `Reservar`, registrada como armadilha no `AGENTS.md`. Na Épica 1 o Pedido tem um Produto só e a ordem não muda nada — ela existe para que a Épica 4 não descubra o impasse com o Carrinho já pronto.

## Verification

**Commands:**
- `docker run --rm -v "${PWD}:/src" -w /src sqlc/sqlc:1.31.1 generate` -- expected: `internal/catalogo/db/gerado/` e `internal/pedido/db/gerado/` atualizados sem erro.
- `docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v "${PWD}:/src" -v azamon-gomod:/go/pkg/mod -w /src --network host -e TESTCONTAINERS_RYUK_DISABLED=true golang:1.27.1 go test -count=1 ./...` -- expected: tudo verde, incluindo fronteira, paridade do `.env` e o schema, nenhum dos três alterado.
- `cd web && npm test && npm run build` -- expected: verificador offline e teste da casca passam, e a construção conclui.

**Manual checks (if no CLI):**
- `docker compose up`, comprar um Produto da faixa aprovada e deixar a tela aberta: o Status percorre `PAGO` → `EM_SEPARACAO` → `ENVIADO` → `ENTREGUE` sozinho, a 30 s por etapa, e a consulta para ao chegar. No `psql`, quatro linhas em `pedido.transicao_status`, a Reserva em `CONSOLIDADA` e o `estoque_total` do Produto uma unidade menor.
