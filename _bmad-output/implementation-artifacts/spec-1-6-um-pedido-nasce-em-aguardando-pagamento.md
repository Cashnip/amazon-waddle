---
title: 'Estória 1.6 — Um Pedido nasce em AGUARDANDO_PAGAMENTO'
type: 'feature'
created: '2026-09-12'
status: 'done'
baseline_commit: 'a7b93cf6345a7c889c6ecf7c83fffd7eea514b5f'
route: 'dispatch'
review_loop_iteration: 0
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problema:** O sistema tem Sessão e Página de Produto, e nada que registre compromisso. Não existe schema `pedido`, não existe Reserva de Estoque, e nenhuma transação foi aberta em ponto nenhum do repositório — nem um `pgx.Tx`, nem um `FOR UPDATE`. Sem o Pedido nascendo, a cadeia 1.7→1.8 não tem objeto sobre o qual agir.

**Abordagem:** Criar o schema `pedido` e a Reserva de Estoque, e o caso de uso que, numa transação só, congela o Item, calcula o total em centavos inteiros, reserva o Estoque na ordem de travamento que o `AD-5` exige, e faz o Pedido nascer em `AGUARDANDO_PAGAMENTO` com número legível e a primeira linha do histórico de transições. Da Página de Produto direto ao Pedido, sem Carrinho.

## Boundaries & Constraints

**Always:**
- A transição é o único ponto de mutação: não existe `UPDATE pedido SET status` em lugar nenhum, só `Transicionar` em compare-and-swap, e zero linhas afetadas devolve `ErrEstadoJaAvancado`.
- Transição primeiro, efeito sobre Estoque depois, na mesma transação.
- Reserva: `SELECT … FROM catalogo.produto WHERE id = ANY($1) ORDER BY id FOR UPDATE` **primeiro**, soma das reservas ativas **depois** — dois comandos, nesta ordem. O `ORDER BY id` não é estilo.
- Uma transação por caso de uso, com `tx pgx.Tx` como parâmetro nomeado, nunca no contexto. Nenhuma chamada de rede dentro da transação aberta.
- `total_centavos` é coluna `bigint` com `CHECK`, nunca derivação de leitura. Nenhuma divisão no caminho monetário.
- O Item congela nome, `preco_praticado_centavos` e Vendedor do Produto no instante da compra.
- Chave estrangeira cruzando schema é proibida: `comprador_id` e `produto_id` são `uuid` sem `REFERENCES`.
- Sentinela novo mora no módulo; a linha de tradução mora só na fatia `registro` de `internal/plataforma/erro/erro.go`.
- Só Comprador autenticado cria Pedido; sem cookie de Sessão válido a rota responde 401.
- **Decidido:** `catalogo.produto` ganha `estoque_total` nesta migração, com padrão, para que a soma das reservas ativas tenha teto contra o que comparar e a ordem de travamento seja exercitada de verdade. Comprar sem Estoque disponível é recusado. A Épica 3 constrói o predicado de visibilidade sobre a coluna já existente; a semente não muda.
- **Decidido:** o Comprador vê o Pedido nascer na própria Página de Produto, pelo filho `"use client"`, no molde da saudação. Nenhuma rota nova no Next e nenhuma rota de leitura no Go — a tela de acompanhamento é da 1.7.

**Never:**
- Sem Carrinho, sem Endereço, sem Frete, sem escolha de quantidade acima de uma unidade — Épicas 4 e 5.
- Sem Tentativa de Pagamento, sem webhook, sem varredura — 1.7 e 1.8.
- Sem tela de acompanhamento do Pedido nem rota de leitura: a estória fecha quando o Pedido nasce, não quando é consultado.
- Sem enum do Postgres: o Status é `text` com `CHECK`, e o mesmo vale para o estado da Reserva.
- Sem middleware de sessão: só duas rotas autenticadas existirão.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Saída esperada | Erro |
|---|---|---|---|
| Compra válida | `POST /api/v1/pedidos` com cookie válido e id de Produto semeado | 201 com `numero` no formato `AZ-2026-000001`, `status` `AGUARDANDO_PAGAMENTO` e `total_centavos` igual ao preço do Produto | — |
| Sem Sessão | mesmo corpo, sem cookie ou com cookie fora do Redis | 401, nenhuma linha criada em `pedido.pedido` | `ErrSessaoInvalida` |
| Produto inexistente | uuid válido sem linha | 404, transação revertida | `ErrNaoEncontrado` |
| Identificador malformado | `"abc"` no corpo | 404 | `ErrNaoEncontrado`, nunca 500 |
| Corpo inválido ou grande demais | JSON malformado ou acima do limite | 400 | `ErrEntradaInvalida` |
| Histórico da transição | Pedido recém-nascido | uma linha em `pedido.transicao_status` com estado anterior vazio, novo `AGUARDANDO_PAGAMENTO`, autor e instante | — |
| Item congelado | Produto semeado | `item_pedido` guarda nome, `preco_praticado_centavos` e nome do Vendedor copiados, não referenciados | — |
| Reserva nasce ativa | Produto semeado | uma linha `ATIVA` em `catalogo.reserva_estoque` com a quantidade comprada | — |
| Estoque esgotado | reservas ativas já somam o `estoque_total` do Produto | 409, nenhum Pedido criado, transação revertida | `ErrEstoqueInsuficiente`, com o disponível em `dados` |
| Numeração por ano | dois Pedidos seguidos | `AZ-2026-000001` e `AZ-2026-000002`, sem colisão e sem depender do uuid | — |
| `Transicionar` sobre estado já avançado | segunda chamada com o mesmo estado de origem | zero linhas afetadas | `ErrEstadoJaAvancado` |

</frozen-after-approval>

## Code Map

- `db/migracoes/` — convenção `YYYYMMDDHHMMSS_<modulo>_<assunto>.sql`, só os pragmas `-- +goose Up` e `-- +goose Down`, comentário em português no topo explicando o porquê. **`StatementBegin` nunca foi usado** e não será preciso se o Status for `text` com `CHECK`. Dois arquivos nesta estória, porque o `schema:` do sqlc recorta por substring do nome: `..._pedido_esqueleto.sql` e `..._catalogo_reserva_estoque.sql`.
- `sqlc.yaml` — bloco novo idêntico em forma aos dois existentes: `schema: "db/migracoes/*_pedido_*.sql"`, `queries: "internal/pedido/db/consultas.sql"`, `out: "internal/pedido/db/gerado"`, `package: gerado`, `sql_package: pgx/v5`, `emit_exact_table_names: true`.
- `internal/pedido/pedido.go` — hoje só doc de pacote. Recebe `Criar`, `Transicionar`, `ErrEstadoJaAvancado` e os tipos do Pedido. Função pública recebe `tx pgx.Tx` (que já satisfaz a `DBTX` do sqlc) como parâmetro nomeado.
- `internal/catalogo/catalogo.go` — recebe `Reservar` e `ErrEstoqueInsuficiente`. `BuscarProduto` já existe e devolve `pgx.ErrNoRows` para uuid malformado.
- `internal/catalogo/db/consultas.sql` — hoje uma consulta só. Recebe o `SELECT … ORDER BY id FOR UPDATE` e a soma das reservas ativas, **como duas consultas nomeadas separadas**: é a ordem entre elas que o `AD-5` protege, e uma consulta só não teria como errá-la.
- `internal/plataforma/erro/erro.go` — a fatia `registro` é ordenada de propósito. Acrescentar as linhas dos sentinelas novos e nada mais.
- `internal/fronteira_test.go` — `pedido → catalogo` e `api → pedido` **já estão na tabela**. A única linha a mudar é `"plataforma": {"identidade"}`, que ganha `"pedido"` e `"catalogo"`, pelos sentinelas dos dois módulos, porque `erro.go` passa a importar a porta desses módulos.
- `db/schema_test.go` — **quebra em cinco lugares** e é onde mora a prova do schema: `schemasDeModulo` ganha `"pedido"`; a lista literal do subteste das cinco tabelas; o `len(tem) != 5` do subteste de chave primária; a heurística de plural, que acusa `transicao_status` e a coluna `status`; e a contagem de chaves estrangeiras cruzando schema, que tem de continuar zero. Helpers a reusar: `subirPostgres`, `conectar`, `textos`, `inteiro`.
- `api/rotas.go` — `servidor{cfg, pool, rdb}`, `escreverJSON` é o único serializador de sucesso. Rota nova entra antes do catch-all `"/"`.
- `api/sessao.go` — `lerSessao` resolve o cookie inline em três passos. Extrair `func (s *servidor) compradorDaRequisicao(r *http.Request) (identidade.Comprador, error)` e chamar dos dois handlers. **Não fazer middleware.** Reusar `corpoMaximo`, o padrão do `http.MaxBytesReader` e o `semCache`, que a resposta do Pedido também exige.
- `api/pedido.go` (novo) — abre a transação com `s.pool.Begin`, `defer tx.Rollback`, `tx.Commit`. É o primeiro `pgx.Tx` do repositório; não existe nem é preciso um ajudante em `plataforma`.
- `api/sessao_test.go` — `ambiente(t)` sobe um par de contêineres por pacote e `TestSessaoEProduto` é o único `Test*`. `cookieValido` já é capturado no primeiro subteste. `postar` está cravado em `/api/v1/sessoes`: generalizar o caminho ou escrever um `postarEm` local. `decodificar` já desembrulha o envelope de erro.
- `api/pedido_test.go` (novo) — funções registradas como `t.Run` dentro de `TestSessaoEProduto`, no mesmo padrão de `api/produto_test.go`.
- `web/app/produtos/[id]/comprar.tsx` (novo) — filho `"use client"`, `fetch` relativo com `credentials: "same-origin"`, no molde de `saudacao.tsx`. O botão usa `primary-strong`, que o `DESIGN.md` reserva para o passo irreversível, uma vez por fluxo. A mensagem de erro exibida é sempre `corpo?.erro?.mensagem`.
- `db/semente/001_catalogo_semeado.sql` — **gerado por `media/gerar.go`**. A coluna de Estoque nasce com `DEFAULT`, então nem a semente nem o gerador nem `db/semente-grande/` precisam mudar, e `VersaoSemente` fica onde está.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/*_pedido_esqueleto.sql` -- Schema `pedido` com `pedido`, `item_pedido` e `transicao_status`, chaves `uuid DEFAULT uuidv7()`, `total_centavos bigint` com `CHECK`, Status em `text` com `CHECK`, e o contador do número por ano.
- [x] `db/migracoes/*_catalogo_reserva_estoque.sql` -- `catalogo.reserva_estoque` já com estado `ATIVA | LIBERADA | CONSOLIDADA` e o índice único parcial do `AD-5`, mais `estoque_total` em `catalogo.produto`, com padrão, para a Reserva ter teto.
- [x] `sqlc.yaml` + `internal/pedido/db/consultas.sql` + `sqlc generate` -- Bloco do módulo `pedido` e as consultas do caso de uso.
- [x] `internal/catalogo/db/consultas.sql` + `sqlc generate` -- Travamento por `ORDER BY id FOR UPDATE` e soma das reservas ativas, em duas consultas nomeadas.
- [x] `internal/catalogo/catalogo.go` -- `Reservar` sobre `tx`, na ordem que o `AD-5` exige, criando a Reserva `ATIVA`.
- [x] `internal/pedido/pedido.go` -- `Criar` e `Transicionar` em compare-and-swap, com `ErrEstadoJaAvancado`, o número por ano e a primeira linha do histórico.
- [x] `internal/plataforma/erro/erro.go` + `internal/fronteira_test.go` -- Traduzir os sentinelas novos e declarar as arestas que a tradução obriga.
- [x] `api/sessao.go` -- Extrair a resolução do Comprador da requisição e usá-la nos dois handlers.
- [x] `api/pedido.go` + `api/rotas.go` -- `POST /api/v1/pedidos`, transação aberta e fechada no handler, DTO e limite de corpo.
- [x] `db/schema_test.go` -- Acompanhar as quatro tabelas novas nos cinco pontos que quebram, e provar o índice único parcial e o `CHECK` do total.
- [x] `api/pedido_test.go` -- Cobrir as linhas da matriz de I/O contra Postgres e Redis reais, incluindo a numeração de dois Pedidos seguidos e o compare-and-swap.
- [x] `internal/pedido/pedido_test.go` -- Provar que `Transicionar` sobre estado já avançado devolve `ErrEstadoJaAvancado`.
- [x] `web/app/produtos/[id]/comprar.tsx` + `page.tsx` -- Botão de confirmar compra e o número do Pedido exibido ali mesmo, sem rota nova.
- [x] `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` + `.memlog.md` -- Registrar a forma do Status em `text`, o mecanismo do número por ano e onde a transação é aberta.

**Acceptance Criteria:**
- Dado o código inteiro, quando procuro por `UPDATE pedido SET status`, então não há nenhuma ocorrência.
- Dado um Comprador autenticado na Página de Produto, quando ele confirma a compra, então nasce um Pedido em `AGUARDANDO_PAGAMENTO` com número legível, e o Comprador o vê no navegador.
- Dado o caso de uso, quando inspeciono a transação, então o travamento dos Produtos acontece antes da soma das reservas ativas, e nenhuma chamada de rede ocorre com a transação aberta.
- Dado `go test ./...` e `npm test` em `web/`, quando rodam, então passam, com fronteira, paridade do `.env` e schema intactos.
- Dado `sqlc generate` rodado duas vezes, quando comparo, então `git status` fica limpo na segunda.

## Implementation Notes

## Spec Change Log

## Review Triage Log

**Passagem 1** — caça-cega (15), casos de borda (13), lacunas de verificação (5 + 2 secundários).

| Verdito | Achado | Evidência | Rota |
|---|---|---|---|
| medium | O contador do número trava a linha do ano antes de a Reserva travar o Produto, e com isso a ordem do `AD-5` fica inalcançável | `ProximoNumeroDoAno` faz `ON CONFLICT DO UPDATE` na linha do ano e segura a trava até o commit. Duas criações do mesmo ano nunca estão dentro de `Reservar` ao mesmo tempo, então a corrida que a ordem trava-depois-soma existe para impedir não acontece por construção. Nada está errado hoje; o que fica sem valor é a prova de concorrência do NFR-7, que é da Épica 5. | defer |
| medium | Nenhum teste falha se as duas consultas do `AD-5` forem trocadas de ordem | O único teste que alcança o ramo insere a Reserva concorrente já commitada e faz uma requisição sequencial, então a soma dá o mesmo nas duas ordens. O teste de concorrência proposto também passaria hoje, pela trava do contador acima — mesma raiz. | defer |
| medium | `Transicionar` grava a linha do histórico e nenhum teste observa isso | O teste de unidade usa um `tx` falso cujo `Exec` devolve a mesma etiqueta para todo comando, e o teste contra Postgres reverte sem consultar `transicao_status`. Apagar o `RegistrarTransicao` de `Transicionar` deixa a suíte verde, e o NFR-9 vale só para a linha do nascimento. | patch |
| medium | `tx.Commit` usa o contexto da requisição, que morre se o cliente desistir | O `Rollback` já usa `WithoutCancel` pelo motivo certo; o `Commit` ficou de fora. Cliente que abandona no meio do commit pode deixar o Pedido gravado e a resposta sair em 500. | patch |
| medium | A confirmação some se o corpo do 201 não decodificar | `resposta.json().catch(() => null)` seguido de `setPedido(corpo)`: com corpo nulo não aparece confirmação nem erro, e o Pedido foi criado. O Comprador compra de novo. | patch |
| medium | Nada prova que uma compra recusada não queima um número | É a justificativa escrita do contador por ano. Os dois subtestes de recusa só comparam contagem de linhas, e nenhum Pedido nasce depois deles. | patch |
| low | Nenhuma asserção sobre `Cache-Control: no-store` na rota do Pedido | O comentário do `semCache` foi alargado nesta mudança justamente para cobrir o Pedido, e a linha que o comentário justifica não é testada. | patch |
| low | Nenhuma asserção de `Content-Type` em resposta de sucesso | `escreverJSON` passou a chamar `WriteHeader`, e a ordem virou carga. Trocar as duas linhas serve `text/plain` com a suíte verde. | patch |
| low | A isenção de plural no teste de schema é sufixo, não nome | `!strings.HasSuffix(nome, "status")` isenta qualquer identificador futuro terminado assim. A exceção do contador logo abaixo já é lista de nomes exatos. | patch |
| low | Identificador malformado em `Transicionar` vira `ErrEstadoJaAvancado` | Erro de quem chama sai como 409 "o Pedido já avançou", e a varredura da 1.8 trata isso como desfecho normal — o defeito seria absorvido em silêncio. | patch |
| low | Comentário em `pedidoComEntradaRuim` alega cobertura que o subteste não dá | Diz que o contador é incrementado dentro da transação revertida, mas os dois 404 voltam antes do contador e os dois 400 voltam antes do `Begin`. | patch |
| low | A região de status da confirmação entra no DOM junto com o conteúdo | `role="status"` substitui o botão em vez de já existir, e leitor de tela costuma perder o anúncio. | patch |
| low | `Reservar` recebe um Produto só, então o `ORDER BY id` não ordena nada | Verdadeiro, e o dano é da Épica 4: com Carrinho, a única forma de usar esta API é laço por Produto, que é exatamente o entrelaçamento que o `ORDER BY` existe para impedir. | defer |
| maybe-false | O `ORDER BY id` pode não garantir a ordem de travamento, porque o Postgres trava durante a varredura | Merece confirmação contra o plano de execução real, e só importa quando houver mais de um Produto por Reserva — Épica 4. | defer |
| low | O botão de comprar aparece para visitante não autenticado | Clicar devolve a mensagem do envelope de 401, que é legível. Levar ao login e voltar é a FR-17, da Épica 4. | rejeitado |
| low | `transicao_status` não tem `CHECK` nas colunas de estado | Os dois únicos escritores passam constantes. A restrição acrescenta guarda por um caminho que ninguém percorre. | rejeitado |
| low | `Reservar` com quantidade menor ou igual a zero violaria o `CHECK` | O único chamador passa a constante `unidade`. Guarda por estado que não foi demonstrado alcançável. | rejeitado |
| low | `Transicionar` com Status fora dos sete violaria o `CHECK` | Mesma razão: os chamadores passam constantes do próprio pacote. | rejeitado |
| low | O sequencial acima de 999.999 alargaria o número | Inalcançável na escala da demonstração, e a guarda acrescenta ramo. | rejeitado |
| low | Sem chave de idempotência, repetir o POST cria um segundo Pedido | Real, e é o assunto da 1.7, que traz a Tentativa de Pagamento e a restrição única sobre a chave de idempotência. | rejeitado |
| low | Falta restrição no banco garantindo que a soma das reservas não passe do teto | Um gatilho no Postgres por uma regra que o caso de uso já segura sob trava. | rejeitado |
| low | O teste pode falhar se cruzar a virada do ano em UTC | Uma janela de milissegundos por ano. | rejeitado |
| low | `pedidoSemEstoqueDa409` deixa uma Reserva órfã e usa um uuid fixo sem asserção | Só de higiene: nada depende do Produto esgotado depois, e o banco morre no fim do teste. | rejeitado |
| low | Nenhum teste cobre `comprar.tsx` | Montar bancada de teste de componente é bem mais que correção direta, e o fluxo foi exercitado contra a pilha no ar pela porta 3000. | rejeitado |
| low | `txComLinhas` embute um `pgx.Tx` nulo e entraria em pânico num caminho de `Query` | O compare-and-swap não tem `RETURNING`, e o dia em que tiver, o pânico aparece na primeira execução. | rejeitado |
| low | A recompilação do contexto da épica perdeu qualificadores | Saíram "a partir de repositório limpo", o porquê do `unoptimized` e a enumeração das nove FRs. O arquivo é regerado sempre que envelhece, então conserto manual se perde no próximo derive. | rejeitado |
| rejeitado | A condição de aceite procura um literal que o SQL nunca casa | A correção seria editar a spec desta construção. A propriedade de verdade foi conferida à mão: existe exatamente um comando que altera o Status no repositório, ele carrega `AND status = @de`, e só `Transicionar` o chama. | rejeitado |
| false | Rastreamento e spec divergem, e as seções de registro estão vazias | `in-progress` durante a revisão é o estado correto do fluxo, e o passo de apresentação promove. As seções de registro são preenchidas por esta passagem. | rejeitado |

## Design Notes

**Por que o Status é `text` com `CHECK` e não enum do Postgres:** não existe `CREATE TYPE` no repositório, e `ALTER TYPE … ADD VALUE` não roda dentro de transação — o que torna cada estória que acrescenta estado uma migração de forma diferente das outras. Os sete Status já são conhecidos, então o `CHECK` nasce completo e a Épica 5 só passa a usá-los.

**Por que o número vem de um contador por ano e não de uma `SEQUENCE`:** uma sequência por ano exigiria `CREATE SEQUENCE` dinâmico dentro do caso de uso. Uma tabela `pedido.contador_numero(ano, ultimo)` com `INSERT … ON CONFLICT DO UPDATE … RETURNING` resolve na mesma transação, sem DDL em tempo de execução e sem buracos na numeração quando a transação é revertida. O custo é serializar a criação de Pedidos do mesmo ano numa linha, o que na escala da demonstração não se mede.

**Por que a transação é aberta no handler:** `api/` é quem já segura o pool, e o `AD-4` pede uma transação por caso de uso com `tx` explícito. Um ajudante em `plataforma` seria uma camada a mais para embrulhar três chamadas do pgx.

## Verification

**Commands:**
- `docker run --rm -v "${PWD}:/src" -w /src sqlc/sqlc:1.31.1 generate` -- expected: `internal/pedido/db/gerado/` criado e `internal/catalogo/db/gerado/` atualizado sem erro.
- `docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v "${PWD}:/src" -v azamon-gomod:/go/pkg/mod -w /src --network host -e TESTCONTAINERS_RYUK_DISABLED=true golang:1.27.1 go test -count=1 ./...` -- expected: tudo verde, incluindo fronteira, paridade do `.env` e o schema.
- `cd web && npm test && npm run build` -- expected: verificador offline e teste da casca passam, e a construção conclui.

**Manual checks (if no CLI):**
- `docker compose up`, entrar, abrir a Página de Produto, confirmar a compra e conferir o número do Pedido na tela; depois, no `psql`, que `pedido.transicao_status` tem uma linha e `catalogo.reserva_estoque` tem uma Reserva `ATIVA`.
