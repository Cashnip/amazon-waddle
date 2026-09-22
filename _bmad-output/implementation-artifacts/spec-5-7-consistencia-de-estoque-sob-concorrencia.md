---
title: 'Estória 5.7 — Consistência de Estoque sob concorrência'
type: 'feature'
created: '2026-09-22'
status: 'done'
route: 'oneshot'
baseline_commit: 'f7d1cf8970ac38b066b2f168e99b36eaefa5a21e'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** a NFR-7 é a pergunta mais provável da banca — "o que impede dois Compradores de levarem a última unidade?" — e hoje a resposta é o desenho da 5.6 (`INSERT` do Pedido, depois `catalogo.Reservar` com `ORDER BY id FOR UPDATE`, soma das Reservas ativas sob a trava), não um teste. O único teste concorrente que existe é o duplo clique da 5.6: **um** Comprador, **uma** chave de idempotência, que prova idempotência e não disputa de Estoque. Sem o teste da 5.7, o passo 5 do Roteiro C não pode ser demonstrado ao vivo.

**Approach:** um teste que põe N Compradores distintos — conta, Endereço e Carrinho próprios, cada um com o mesmo Produto — a confirmar o Pedido ao mesmo tempo, com N > estoque inicial, e confere que `pedidos_criados == estoque_inicial`, **nunca mais**; que todo perdedor recebeu 409 `ESTOQUE_INSUFICIENTE`; e que o Estoque no banco fecha com isso (soma das Reservas `ATIVA` = estoque inicial, `estoque_disponivel` = 0 na VIEW). Roda sobre o Postgres real que `ambiente()` já sobe, porque a transação **é** o objeto do teste.

## Boundaries & Constraints

**Always:**
- N > estoque inicial (a rubrica da PRD: `pedidos_criados == min(N, estoque_inicial)` só vira o enunciado da NFR-7 com N maior).
- As N requisições partem juntas, de uma barreira, cada uma com Sessão, Endereço e chave de idempotência próprios.
- O caso da **última unidade** (estoque inicial 1) é obrigatório; um segundo estoque inicial > 1 prova que o invariante não é um acidente do 1.
- O teste falha se sobrar Pedido: a comparação é de igualdade, nunca `<=`.

**Never:**
- Alterar o caminho de produção da criação sem dizer. Se o teste reprovar, a falha é reportada como achado da 5.6 antes de qualquer correção.
- Marcar o teste com build tag, `testing.Short()` ou qualquer chave que o tire do `go test ./...` padrão — este é o teste que "nunca é cortado".
- Tocar nas telas ou no `web/`: a 5.7 não tem superfície.

</frozen-after-approval>

## Implementation Notes

**Arquivos.** `api/concorrencia_test.go` (novo) e uma chamada em `api/sessao_test.go:523`. Nenhuma linha de produção mudou.

**Onde o teste mora.** Dentro de `TestSessaoEProduto`, e não como `Test` de topo: `ambiente()` sobe um par de contêineres por função de topo, e uma função nova custaria mais dois por execução sem provar nada a mais. As três rodadas somam ~1,6 s.

**A decisão que a estória tinha de tomar.** O `HANDOFF.md` e a última entrada do `deferred-work.md` já avisavam: o `ProximoNumeroDoAno` (`internal/pedido/db/consultas.sql:9`) é um `ON CONFLICT (ano) DO UPDATE` sobre a **única linha do ano**, e toda criação o atravessa antes de `catalogo.Reservar` — N criações paralelas correm em fila, e o teste de ponta a ponta passaria sem nunca disputar o `ORDER BY id FOR UPDATE`. A escolha posta era mover a numeração para depois da Reserva ou disputar `catalogo.Reservar` diretamente.

**Confirmado por mutação, antes de escolher.** Removido o `FOR UPDATE OF p` de `TravarProdutosParaReserva` (mutação temporária no `gerado/`, revertida), os oito checkouts na última unidade continuaram verdes: o aviso estava certo, e um teste que não morre quando a trava do desenho some não é critério de aceite de nada.

**Escolhido: disputar `Reservar` direto.** Mover a numeração é mudança de produção em `pedido.Criar` e na ordem que a 5.6 fixou — o `INSERT` do Pedido é a reivindicação da chave de idempotência, e adiá-lo trocaria o desfecho do duplo clique de "o gêmeo espera no índice único" para "o gêmeo perde a última unidade". Fora do escopo de uma estória cujo entregável é teste, e registrado no `deferred-work.md` como teto de vazão. Em lugar disso entrou o terceiro subteste, `travaDeReservaSerializa`: duas transações abertas à mão chamam `catalogo.Reservar` direto, sem contador no caminho; a segunda tem de **bloquear** enquanto a primeira está aberta e sair em `EstoqueInsuficiente` com disponível 0 depois do commit. Repetida a mesma mutação, ele falha em 0,03 s ("a linha do Produto não está travada"). É a prova de que a suíte tem dente.

**O que a revisão corrigiu.** A asserção de Estoque zerado lia `estoque_disponivel` de `catalogo.produto_visivel`, que envolve a conta num `greatest(..., 0)`: ela devolveria zero tanto para uma unidade vendida quanto para oito, ou seja, passaria exatamente no caso de venda a mais que existe para pegar. Passou a ler a diferença crua de `catalogo.produto` menos as Reservas `ATIVA`. Entrou também a contagem de linhas novas em `pedido.pedido`: o `INSERT` da chave roda antes da Reserva, então um Pedido órfão — sem Item e sem Reserva — escaparia das contagens que olham pelo Produto.

**Escolhas menores.** N = 8 sobre Estoque 1 e N = 12 sobre Estoque 3 (a segunda rodada existe porque uma trava que só distinguisse "existe ou não existe" passaria na primeira e venderia demais na segunda). A largada é um canal fechado depois de todas as goroutines estarem paradas nele; todo o preparo — conta, Carrinho, Endereço, cotação, chave — é sequencial, e depois da largada a única chamada de `t` nas goroutines é o `t.Helper()` de dentro de `postarPedido`, que o `testing` declara seguro em paralelo.

O `MaxConns` do `pgxpool` ficou no padrão do `ambiente()` (`max(4, NumCPU)`): ele limita quantas das doze requisições estão dentro do Postgres ao mesmo tempo, e portanto quanta corrida a rodada gera — mas não o que a igualdade afere, e mexer nele mudaria a bancada de todos os outros testes de `api/`.

## Review Triage Log

Camada única do workflow: Blind Hunter, sem contexto, sobre o worktree. Treze achados.

- **high** `estoque_disponivel = 0` não detecta venda a mais — confirmado: `catalogo.produto_visivel` usa `greatest(p.estoque_total - sum(ATIVA), 0)` (`db/migracoes/20260918150000_catalogo_produto_criado_em.sql:13`), então a asserção passava no exato caso que existia para pegar. Corrigido: diferença crua das tabelas.
- **medium** Nada contava linhas em `pedido.pedido` — confirmado: o `INSERT` da chave precede `catalogo.Reservar`, e um Pedido órfão passaria pelas contagens por Produto. Corrigido com `contarPedidos` antes e depois da largada.
- **medium** A narrativa tratava como descoberta o que `HANDOFF.md:97` e a última entrada do `deferred-work.md` já registravam, e não dizia por que mover a numeração foi recusado — confirmado. Reescrito como decisão.
- **medium** `-race` ausente da verificação — confirmado e rodado: `go test -race -count=1 ./api/` verde em 18,6 s. Registrado na seção Verification, que também estava faltando (o HANDOFF acompanha o passeio dessa seção por spec).
- **medium** Metade do débito do contador segue aberta e não re-arquivada — confirmado. Entrada nova no `deferred-work.md`, como teto de vazão.
- **medium** A bancada não prende o `TOTAL_DIVERGENTE` do passo 5 nem os dois caminhos do duplo clique, que duas entradas do ledger esperavam dela — confirmado: pela mesma fila do contador, as criações não se sobrepõem dentro de `pedido.Criar`. Re-arquivado no `deferred-work.md` em vez de silenciado.
- **low** `disponivel[0]` sem guarda de tamanho — confirmado (`textoDe` devolve `nil` em zero linhas). Some junto com a asserção que foi reescrita, na forma guardada de `carrinho_test.go:224`.
- **low** O comentário dizia que `postarPedido` "não toca em `t`" — confirmado falso: ele chama `t.Helper()` (`api/pedido_test.go:679`). É seguro assim mesmo, mas a razão escrita era errada. Corrigido.
- **low** A justificativa da espera de 500 ms negava haver evento observável — confirmado falso: `pg_locks` com `granted = false` mostra a segunda transação parada. Comentário corrigido, com a sondagem nomeada como saída se algum dia flacar.
- **low, rejeitado** Guarda `n <= estoqueInicial` e contagem de recusas seriam redundantes. A primeira documenta a condição da rubrica da PRD no lugar onde ela é violável; a segunda afere pelo lado oposto. Três linhas, e apagá-las é churn.
- **false** "A spec não tem I/O Matrix, Code Map, Tasks, Design Notes, Suggested Review Order." A rota `oneshot` do próprio workflow manda apagá-las. Só a Verification tinha consequência (o HANDOFF a acompanha), e foi acrescentada.
- **false** "Tracking e HANDOFF ficaram para trás." Estavam em `in-progress` porque o passo de finalização ainda não tinha rodado quando a revisão leu o worktree; `sprint-status.yaml` foi a `review` e o HANDOFF recebeu o commit `docs(5.7)` de sempre.
- **low, absorvido** O `MaxConns` padrão limita quanta corrida a rodada gera. Verdade, e não muda a igualdade aferida; a ressalva foi para junto da escolha de N.

## Verification

**Commands:**
- `go vet ./... && go test -count=1 ./...` -- expected: verde, na imagem `azamon-dev:latest` (receita do HANDOFF).
- `go test -race -count=1 ./api/` -- expected: verde. É o pacote onde N goroutines escrevem na mesma fatia e atravessam a pilha de handlers em paralelo pela primeira vez.
- Mutação de prova, com `git checkout` depois: apagar `FOR UPDATE OF p` de `travarProdutosParaReserva` em `internal/catalogo/db/gerado/consultas.sql.go` -- expected: `a trava do Produto serializa duas Reservas da última unidade` **falha**. Se passar, a suíte deixou de provar o AD-5.

**Manual checks (if no CLI):**
- Nenhum: a estória não tem superfície. O `web/` não foi tocado.
