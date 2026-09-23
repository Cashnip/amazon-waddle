---
title: 'Estória 5.11 — Expiração da Tentativa de Pagamento'
type: 'feature'
created: '2026-09-22'
status: 'done'
route: 'dispatch'
review_loop_iteration: 1
baseline_commit: '9162a0acf63baaa9ddc7e42ed5fee90bdd011530'
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** a Tentativa de Pagamento da faixa `,95`–`,99` nunca recebe confirmação, e nada a encerra: o Pedido fica em `AGUARDANDO_PAGAMENTO` para sempre, com a Reserva de Estoque presa e a unidade fora da prateleira. A FR-34 🔒 é a única coisa que falta — a linha da expiração já está na tabela do AD-3 (ator `VARREDURA`, motivo `TEMPO_ESGOTADO`, liberando a Reserva), a tela já escreve "Tempo de pagamento expirado." e já mostra "Tentar pagar de novo" depois dela, mas o passo `expirar` do tique não existe: o `cmd/azamon` tem só um comentário no lugar dele.

**Approach:** acrescentar o passo `expirar` à varredura, entre `aplicar` e `simular` (AD-6), no mesmo molde do `SimularEntrega` — candidatos lidos fora da transação, uma transação por Pedido, releitura travada e `Transicionar` fazendo o efeito. O instante de vencimento é o mesmo que a tela mostra: a última transição *para* `AGUARDANDO_PAGAMENTO` mais `AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO`. Nenhuma linha do `web/` muda.

## Boundaries & Constraints

**Always:**
- A decisão deriva do histórico (`pedido.transicao_status`), nunca de temporizador em memória: reiniciar o contêiner não congela nem adianta Pedido nenhum.
- O instante é o mesmo de `pedido.ExpiraEm` — para um Pedido em `AGUARDANDO_PAGAMENTO`, `max(ocorrido_em)` é a entrada nesse Status. Se os dois discordarem, a tela e a varredura discordam de quando venceu.
- `aplicar` vem antes de `expirar` no tique: uma aprovação que chegou no prazo não pode ser descartada por um relógio que rodou primeiro.
- **A ordem do tique não basta, e a garantia é a inbox lida sob a trava** *(emendado pelo humano em 2026-09-22, na revisão do passe 1)*. `Expirar` confere, com a linha do Pedido já presa por `TravarPedido` e antes de `Transicionar`, se existe confirmação `PENDENTE` daquele Pedido; havendo, sai calado e deixa o tique seguinte aplicá-la. A ordem sozinha deixa três furos — `Varrer` que falha e não aplica nada, a linha que o `SKIP LOCKED` pulou, e a confirmação que chega entre os dois passos —, e em todos eles o Comprador pagou no prazo e perderia a Reserva. A porta é de `pagamento`, dono da inbox (AD-7), chamada por `pedido`: a aresta contrária continua não existindo.
- **A aprovação que chega depois de uma expiração legítima é avisada** *(emendado pelo humano em 2026-09-22)*: o `WarnContext` de `aplicarConfirmacao` passa a cobrir o Pedido em `PAGAMENTO_RECUSADO` por `TEMPO_ESGOTADO`, e não só o `CANCELADO`. O registro continua sendo a própria linha da inbox; o que muda é deixar de ser perda silenciosa (addendum §2, invariante 7). Quem exibe ao Administrador continua sendo a 6.5.
- **A janela de emissão tem de ser não vazia.** `AZAMON_CONFIRMACAO_ATRASO` maior ou igual a `AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO` faria `criada_em <= ate AND criada_em > desde` nunca casar: nenhuma confirmação emitida, todo Pedido expirando, sem erro e sem log. `CarregarConfig` recusa o par, como já recusa os outros pares comparáveis.
- O efeito é do `Transicionar` (a linha do AD-3 já libera a Reserva); quem chama não invoca `catalogo.Liberar` por conta própria.
- `expirar` não é desligável: o interruptor `AZAMON_ENTREGA_SIMULACAO_ATIVA` é da simulação de entrega e não cobre a FR-34.
- `ErrEstadoJaAvancado` é desfecho esperado, e não erro — a confirmação que chegou no mesmo tique já moveu o Pedido.
- **A janela de emissão passa a ter os dois lados** *(decidido pelo humano em 2026-09-22)*: `TentativasSemConfirmacao` ganha o piso `criada_em > agora - prazo` e `EmitirConfirmacoesDevidas` recebe o prazo da Config, por parâmetro, como já recebe o atraso. Passado o prazo a Tentativa está morta — a varredura já a encerrou —, e emitir sobre ela não teria efeito. Sem migração e sem coluna: a emissão continua **derivada, não marcada** (AD-7). Fecha a entrada do ledger que a 5.10 deixou para esta estória.
- Glossário literal: Tentativa de Pagamento, Reserva de Estoque, Pedido.

**Never:**
- Estado novo, coluna nova ou Status novo: o caminho é o da recusa, e o que os distingue é o motivo.
- Emitir confirmação `RECUSADO` pela expiração — o Provedor não recusou, o tempo esgotou.
- Tocar `web/`, a tela do Pedido, `Detalhar`, `ExpiraEm` ou a resposta do AD-18.
- Cancelamento (6.3) e superfície do Administrador (6.4, 6.5).
- **Fechar o `Corrente` fora da trava do Pedido** *(decidido pelo humano em 2026-09-22: continua adiado)*. Segue inalcançável com o Provedor Simulado — uma confirmação pendente da Tentativa 1 implica o Pedido ainda aguardando —, e o piso da janela de emissão acima estreita ainda mais o intervalo. A entrada do ledger é **atualizada com esta leitura, não fechada**.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Tentativa vencida | Pedido em `AGUARDANDO_PAGAMENTO` desde antes do prazo | `PAGAMENTO_RECUSADO`, ator `VARREDURA`, motivo `TEMPO_ESGOTADO`; Reserva `LIBERADA` e o Estoque de volta | — |
| Prazo por vencer | mesma situação, dentro do prazo | nada muda: sem transição, sem linha no histórico | — |
| Nova Tentativa recomeça o prazo | Pedido expirado, Comprador tenta de novo | volta a `AGUARDANDO_PAGAMENTO` e só expira um prazo inteiro depois da nova entrada | — |
| Aprovação no mesmo tique | confirmação `APROVADO` pendente e prazo vencido | `aplicar` roda antes: o Pedido vai a `PAGO` e a expiração não o encontra mais | — |
| Aprovação pendente que `aplicar` não aplicou | `Varrer` falhou, ou o `SKIP LOCKED` pulou a linha, ou a confirmação chegou entre os dois passos | `Expirar` vê a linha `PENDENTE` sob a trava e **não expira**; o tique seguinte aplica e o Pedido vai a `PAGO` | nada é gravado; sem erro |
| Aprovação tardia sobre Pedido expirado | confirmação `APROVADO` chega depois de `TEMPO_ESGOTADO` | Status não muda, a confirmação vira `NAO_APLICAVEL_SINALIZADA` **e sai aviso** — não é perda silenciosa | — |
| Janela de emissão invertida | `AZAMON_CONFIRMACAO_ATRASO` >= `AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO` | o processo **não sobe**: `CarregarConfig` recusa o par | erro de configuração, como os outros pares |
| Corrida perdida | outro caminho moveu o Pedido entre a seleção e a trava | nada é gravado; o tique segue sem erro | `ErrEstadoJaAvancado` registrado como informação |
| Pedido travado | outro tique segura a linha | `FOR UPDATE SKIP LOCKED` pula; o tique seguinte o encontra | — |
| Fora de `AGUARDANDO_PAGAMENTO` | `PAGO`, `SEPARANDO`, `CANCELADO`… | nunca é candidato, por mais tempo que passe | — |
| Emissão depois do prazo | Tentativa sem confirmação, criada além do prazo | sai da lista de `TentativasSemConfirmacao`: o tique não a relê mais | — |

</frozen-after-approval>

## Code Map

- `cmd/azamon/main.go:148,169` -- o comentário "`expirar` entra aqui, na 5.11" marca o ponto exato, entre `pedido.Varrer` e `pedido.SimularEntrega`. Chamada nova com `cfg.PagamentoTentativaExpiracao`; **sem** guarda de `EntregaSimulacaoAtiva`. Atualize também o cabeçalho do `varrer` (`:141`), que hoje anuncia a ordem em três passos.
- `internal/pedido/pedido.go:678-757` -- `SimularEntrega` + `avancarEntrega` são o molde e a peça a reusar: mesmo `ate`, mesmo `PedidosParaAvancar`, mesmo `TravarPedido`, mesma reconferência `!travado.Desde.Valid || travado.Desde.Time.After(ate)`, mesmo tratamento de `ErrEstadoJaAvancado`. Prefira parametrizar a casca (origens, próximo Status, ator, motivo) a duplicar as ~45 linhas; `SimularEntrega` mantém assinatura e comportamento, que o teste da 1.8 já guarda.
- `internal/pedido/db/consultas.sql:153` -- `PedidosParaAvancar` serve como está: `status = ANY(@status)` mais `max(ocorrido_em) <= @ate`. Só o comentário acima dela precisa deixar de falar só da simulação. **Sem `sqlc generate`** se nenhuma consulta mudar.
- `internal/pedido/maquina.go:120` -- a linha da expiração já existe e é a quarta da tabela; `motivoReservado` garante que só ela aceita `TEMPO_ESGOTADO`. Não mexer.
- `internal/pedido/pedido.go:446` -- `ExpiraEm` é a fonte do instante que a tela mostra. Não mexer: a varredura chega ao mesmo instante pelo `desde` do `TravarPedido`, e é isso que o teste tem de prender.
- `internal/pagamento/pagamento.go:242,248` -- `EmitirConfirmacoesDevidas` e o comentário que diz que a faixa sem confirmação "fica na lista até lá". O prazo entra como parâmetro novo e o piso, na consulta `internal/pagamento/db/consultas.sql:48` (`criada_em > @desde`) — aqui **há** `sqlc generate`. Os chamadores de teste (`api/nova_tentativa_test.go:70`, `api/webhook_test.go`) passam a ter de informar um prazo largo.
- `api/maquina_test.go:300` -- `envelhecerHistorico` é o que substitui esperar: recua `ocorrido_em` do Pedido, desligando o gatilho de imutabilidade só naquela transação.
- `api/pedido_test.go:616-621` -- **leia esta advertência antes de escrever o teste**: a varredura pega *todo* Pedido elegível do banco, e a suíte inteira compartilha o contêiner. Use prazo largo (meia hora) mais `envelhecerHistorico` só do Pedido do subteste, ou os Pedidos em `AGUARDANDO_PAGAMENTO` de outros subtestes expiram junto e derrubam as contas de Estoque deles.
- `api/nova_tentativa_test.go:31-110` -- o molde do subteste novo: conta, Vendedor, Categoria e Produtos próprios; `emitirPara` entregando só as Tentativas dos seus Pedidos. Para cair na faixa que nunca confirma, o total (preço mais o Frete em reais inteiros) precisa fechar em `,95`–`,99`.
- `api/sessao_test.go:550` -- registrar o subteste novo depois de `recusaENovaTentativa`; `api/pedido_detalhe_test.go:23` tem `expiracaoDaTentativaDeTeste` e `tentativasMaxDeTeste`, e `api/sessao_test.go:175` já põe o prazo na Config de teste.
- `web/lib/pedido.ts:98,121` -- `TEMPO_ESGOTADO` já tem frase e `podeTentarDeNovo` já não olha o motivo. **Nada a fazer no `web/`** — confira, não altere.

## Tasks & Acceptance

**Execution:**
- [x] `internal/pedido/pedido.go` -- `Expirar(ctx, pool, prazo)` reusando a casca de `SimularEntrega`/`avancarEntrega` parametrizada; comentário dizendo que o instante é o de `ExpiraEm`.
- [x] `internal/pedido/db/consultas.sql` -- só o comentário de `PedidosParaAvancar`, que passa a servir os dois passos do tempo.
- [x] `cmd/azamon/main.go` -- `pedido.Expirar` entre `Varrer` e `SimularEntrega`, sem interruptor; cabeçalho do `varrer` com a ordem de quatro passos.
- [x] `internal/pagamento/db/consultas.sql`, `pagamento.go` + `pagamento_test.go` -- o piso `criada_em > @desde`, o prazo por parâmetro em `EmitirConfirmacoesDevidas`, `sqlc generate`, e o teste de que a Tentativa passada do prazo sai da lista.
- [x] `api/expiracao_test.go` (novo) -- as linhas da matriz pela pilha real: o vencido expira com ator e motivo e devolve o Estoque; o não vencido não; a nova Tentativa recomeça o prazo; a aprovação no mesmo tique vence a expiração; o Pedido fora de `AGUARDANDO_PAGAMENTO` nunca é candidato.
- [x] `api/sessao_test.go` -- registrar o subteste depois de `recusaENovaTentativa`.
- [x] `deferred-work.md` -- fechar as entradas das linhas 386–387 e 398–399 (a superfície recusada agora é alcançável pelas duas faixas) e 414–415 (o piso da emissão); **atualizar, não fechar**, a de 406–407 (`Corrente` fora da trava). `sprint-status.yaml` (`5-11-…` para `review`), `HANDOFF.md`, `addendum.md` §10 mais o memlog do PRD.

**Acceptance Criteria:**
- Given um Pedido cujo total cai na faixa `,95`–`,99` e o prazo de 60 s da demonstração, when o tique corre, then ele sai sozinho de `AGUARDANDO_PAGAMENTO` e o Estoque volta — o passo 5 do Roteiro B, e a única forma de a SM-1 exercitar a expiração.
- Given a suíte inteira, when ela roda, then os caminhos aprovado (1.7, 5.9), recusado (5.10) e a simulação de entrega (1.8) continuam verdes — a expiração não pode arrastar Pedido de outro subteste.
- Given o contêiner derrubado no meio do prazo, when ele volta, then o Pedido expira no instante que já estava marcado, porque o prazo sai do histórico.

## Implementation Notes

- **A casca parametrizada.** `SimularEntrega` e `Expirar` são duas linhas cada sobre `avancarPeloTempo` + `avancarUm`, com um `passoDoTempo{avancos, ator, motivo}` como parâmetro. `avancos` é um mapa `Status→Status` — `simulacao`, que já existia, e `expiracao`, uma linha só — e é dele que saem tanto as origens da consulta quanto o destino da releitura travada, pelo mesmo motivo que já valia na 1.8: duas listas seriam duas oportunidades de divergirem. Um `proximo` sempre-verdadeiro para a expiração teria transformado o Pedido que virou `PAGO` entre a seleção e a trava num `ErrTransicaoInvalida`; o mapa devolve `false` e a função sai calada. `SimularEntrega` manteve assinatura e comportamento, e o `maps` do import saiu junto com o `maps.Keys`.
- **O teste do piso da emissão ficou em `api/`, e não em `internal/pagamento/pagamento_test.go`.** As Tasks pediam `pagamento_test.go`, mas esse arquivo é só de unidade, sem banco nenhum — montar um `testcontainers` novo ali para uma consulta seria bancada inteira por uma linha de SQL. O subteste "a Tentativa passada do prazo sai da lista de emissão" foi para `api/expiracao_test.go`, que é onde `EmitirConfirmacoesDevidas` já era exercitada contra Postgres de verdade (precedente: `api/pedido_detalhe_test.go`). Ele emite duas vezes sobre a mesma Tentativa `,90` — dentro da janela emite `RECUSADO`, e com `criada_em` recuada duas horas não emite nada —, então o piso tem prova dos dois lados.
- **Passe 2: a garantia deixou de ser a ordem do tique e passou a ser a inbox lida sob a trava.** `Expirar` pergunta a `pagamento.TemConfirmacaoPendente` (porta nova, AD-7) com a linha do Pedido já presa por `TravarPedido` e antes de `Transicionar`; havendo confirmação por aplicar, sai calado. É o único jeito de cobrir os três furos que a ordem sozinha deixa — `Varrer` que falhou, a linha que o SKIP LOCKED pulou, e a confirmação que chegou entre os passos. O `conferirInbox` é só da expiração: a simulação move Pedido já pago. Preso por "a aprovação pendente que aplicar não aplicou segura a expiração"; neutralizada a conferência, o subteste morre em `PAGAMENTO_RECUSADO`.
- **Passe 2: o aviso da aprovação tardia.** `aplicarConfirmacao` agora também avisa quando o Pedido travado está em `PAGAMENTO_RECUSADO` e a última transição é `TEMPO_ESGOTADO`, lido de `HistoricoDoPedido` — consulta que já existia, em vez de uma nova só para a última linha. Continua saindo depois do commit, pelo motivo que já estava escrito.
- **Passe 2: `CarregarConfig` recusa o par que esvazia a janela de emissão**, na forma dos outros pares comparáveis, com teste próprio em `config_test.go`.
- **Passe 2: três buracos no `cmd/azamon/main_test.go`.** O prazo do tique não estava preso ao campo da Config (qualquer duração abaixo de uma hora passava) — agora o binário sobe com `AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO=6s` e um Pedido recém-nascido tem de sobreviver a metade do prazo antes de expirar. Nada provava que `expirar` fica **fora** do `if cfg.EntregaSimulacaoAtiva` — `interruptorDesligaASimulacao`, que já sobe o binário com o interruptor desligado, passou a exigir que um Pedido vencido expire mesmo assim, liberando a Reserva. E a assertiva do Estoque era só sobre o estado da linha da Reserva; agora confere o `estoque_disponivel` da `produto_visivel` antes e depois. De quebra, o Pedido com aprovação pendente nascia em duas escritas com o binário já ticando — um tique naquela janela o expirava; histórico, Tentativa e confirmação vão numa transação só, com a linha do histórico por último.
- **A ordem do tique ganhou teste no binário inteiro (`cmd/azamon/main_test.go`, `oTiqueExpiraNaOrdemDoAD6`).** A mutação da Verification — pôr `Expirar` antes de `Varrer` — não é alcançável por `api/`, que chama os dois passos à mão e escolhe a ordem que quiser: sem esse teste, inverter as duas linhas de `varrer()` deixava `go test ./...` verde. (A ordem continua normativa e continua presa aqui, mas a garantia é a conferência da inbox acima — ver Passe 2.) Dois Pedidos com o histórico já vencido entram no mesmo tique — um sem confirmação nenhuma, que tem de expirar liberando a Reserva, e um com uma aprovação `PENDENTE` na inbox, que tem de terminar `PAGO`. Verificado por mutação: invertidos, o segundo termina em `PAGAMENTO_RECUSADO` e o teste morre.
- **A reconferência de `avancarUm` — as duas metades, `!travado.Desde.Valid` e `travado.Desde.Time.After(ate.Time)` — não tem teste que morra quando ela sai**, ao contrário do que a Verification previa. `PedidosParaAvancar` já filtra por `max(ocorrido_em) <= @ate`, então o Pedido dentro do prazo nunca chega à releitura travada: a guarda protege só a corrida entre a seleção e a trava, inalcançável num teste de uma goroutine. Confirmado por mutação — apagada a comparação, `go test ./...` inteiro fica verde, e isso já valia para a simulação de entrega da 1.8, de onde a linha veio. A linha ficou (está correta e é barata) e o buraco foi para o `deferred-work.md`.
- **Achado fora de escopo, corrigido aqui: a suíte de `api/` piscava.** O subteste da emissão da faixa `,95` (5.8) falhava em cerca de uma execução em quatro — reproduzido **no código de antes desta estória**, por `git stash`. Causa: os chamadores de teste passavam `atraso = 0`, então o corte `ate` saía do relógio do host e `criada_em`, do relógio do contêiner do Postgres; com o contêiner um instante à frente, a Tentativa recém-criada caía fora da janela. Trocado por `atrasoDeTeste = -time.Minute`, com o porquê escrito na constante. Quatro execuções seguidas de `go test -count=1 ./...` ficaram verdes depois.

- **Auditoria da matriz, depois da implementação (sessão que orquestra).** Duas linhas não tinham teste próprio. A do **Pedido travado** ganhou o subteste "o Pedido travado é pulado, e o tique seguinte o encontra": uma transação segura a linha com `FOR UPDATE`, `Expirar` devolve `nil` sem gravar nada, e solta a trava o tique seguinte o expira liberando a Reserva — é a prova de que o `SKIP LOCKED` não perde Pedido nem serializa a varredura. Fica o aviso, escrito no próprio comentário: trocar o `SKIP LOCKED` por espera faz este subteste **pendurar** até o prazo do `go test`, em vez de falhar rápido, porque a trava é da mesma goroutine. A linha da **corrida perdida** não ganhou teste próprio, e a razão está abaixo.
- **A linha "Corrida perdida" da matriz nomeia um mecanismo que a expiração não alcança.** A coluna *Error Handling* diz `ErrEstadoJaAvancado` registrado, herdado do comentário do `avancarEntrega` da 1.8. No caminho da expiração esse ramo é inatingível: a releitura travada lê o Status já mudado, `passo.avancos[atual]` devolve `false` e a função sai calada **antes** de chegar ao compare-and-swap. O comportamento observável que a linha promete — "nada é gravado; o tique segue sem erro" — está provado, pelos mesmos assertos que provam a linha "Fora de `AGUARDANDO_PAGAMENTO`" (o `PAGO` que não expira, o `PAGAMENTO_RECUSADO` que não é candidato na segunda expiração). O bloco congelado é do humano: a frase fica como está, e a imprecisão vai no relato.
- **`proximoDaSimulacao` virou código morto e saiu.** Com `avancarUm` lendo `passo.avancos` direto, a função só era chamada pelo próprio teste — cobertura que parecia real e não guardava mais nada de produção. Removida; `TestProximoDaSimulacao` virou `TestOsMapasDoTempo`, que compara os **dois** mapas com `maps.Equal` (é deles que `avancarUm` lê), e `TestSimulacaoCabeNaTabela` passou a conferir também a linha da expiração contra a tabela do AD-3, com ator `VARREDURA` e motivo `TEMPO_ESGOTADO`.

**Verificação executada (2026-09-22):**

- `go vet ./...` limpo e `go test -count=1 ./...` verde, **quatro execuções seguidas** (o package `api` piscava antes — ver Implementation Notes).
- `docker build ./web` verde; nenhuma linha do `web/` mudou, conferido em `web/lib/pedido.ts` (`frasesDoMotivo[TEMPO_ESGOTADO]` e `podeTentarDeNovo`).
- Mutação `Expirar` antes de `Varrer` no tique: **falha** — `main_test.go:123`, "o Pedido pago no prazo está em PAGAMENTO_RECUSADO". Revertida.
- Mutação `AtorVarredura` → `AtorProvedor`: **falha** — `ErrTransicaoInvalida` derruba o subteste do vencido. Revertida.
- Mutação do piso `criada_em > @desde` (neutralizado por `OR true`): **falha** — "passada do prazo a Tentativa emitiu RECUSADO". Revertida.
- Mutação da reconferência `travado.Desde.After(ate)`: **NÃO falha** — suíte inteira verde sem ela. Ver Implementation Notes; registrado no `deferred-work.md`.
- Passeio manual do Roteiro B passo 5: **não feito** — nenhuma tela da Épica 5 foi aberta num navegador ainda.
- Rerrodado pela sessão que orquestra, depois da auditoria da matriz e da limpeza do código morto: `go vet ./...` limpo e `go test -count=1 ./...` verde, e os seis subtestes de `a expiração da Tentativa de Pagamento` confirmados em execução por `go test -v -run`, nenhum pulado.

## Spec Change Log

- **Passe 1 de revisão (Blind Hunter, Edge Case Hunter, Verification Gap) — 2026-09-22.** Achado disparador (as três lentes, por caminhos diferentes): o bloco congelado nomeava a **ordem do tique** como o mecanismo que protege a aprovação recebida no prazo, e a ordem sozinha não protege — `Varrer` devolve cedo quando a leitura da inbox falha e `varrer()` segue para `Expirar` assim mesmo; `aplicarConfirmacao` devolve `nil` quando o `SKIP LOCKED` pula a linha; e a confirmação que chega entre os dois passos nunca é vista. Estado ruim evitado: Comprador que pagou dentro do prazo com o Pedido em `PAGAMENTO_RECUSADO`/`TEMPO_ESGOTADO`, Reserva liberada, e a aprovação marcada `NAO_APLICAVEL_SINALIZADA` **sem aviso** — a invariante 7 do addendum §2 quebrada. Roteado como `intent_gap`: a intenção era clara, mas havia duas leituras defensáveis de até onde ir. **O humano emendou o bloco congelado** com três decisões: `Expirar` confere a inbox sob a trava do Pedido por uma porta de `pagamento` (a ordem do tique continua, mas deixa de ser a garantia); o `WarnContext` passa a cobrir o expirado além do cancelado; e `CarregarConfig` recusa `atraso >= prazo`, que esvaziaria a janela de emissão em silêncio. O humano também escolheu **emendar e aplicar por cima**, em vez do revert e re-derivação que o workflow prescreve: a árvore estava verde e o resto da implementação, boa.
  **KEEP — o que funcionou e tem de sobreviver:** a casca única `avancarPeloTempo`/`avancarUm` parametrizada por `passoDoTempo`, com os mapas `simulacao` e `expiracao` como lista única de origens e destinos (é o mapa devolvendo `false` que faz o Pedido virado `PAGO` sair calado em vez de dar `ErrTransicaoInvalida`); `PedidosParaAvancar` reusada sem mudança; o instante da expiração saindo do mesmo `max(ocorrido_em)` que `ExpiraEm`; `expirar` sem interruptor e entre `aplicar` e `simular`; o piso da janela de emissão derivado, sem migração nem coluna (AD-7); `oTiqueExpiraNaOrdemDoAD6` prendendo a ordem no binário inteiro; o subteste do `SKIP LOCKED` com o aviso de que a mutação pendura em vez de falhar; `atrasoDeTeste = -time.Minute` com o porquê da deriva de relógio escrito na constante; e nenhuma linha do `web/`.

## Review Triage Log

**Passe 1 — 2026-09-22 (Blind Hunter · Edge Case Hunter · Verification Gap).**

| # | Achado | Veredito | Evidência |
|---|--------|----------|-----------|
| 1 | `Expirar` roda mesmo quando `Varrer` falhou, e a ordem do tique não protege a aprovação que ficou `PENDENTE` (3 lentes) | **high** | `Varrer` devolve cedo quando `ConfirmacoesNaoAplicadas` falha (`internal/pedido/pedido.go:583`) — nada foi aplicado — e `varrer()` só loga e segue para `Expirar` (`cmd/azamon/main.go:173`). O Pedido vai a `PAGAMENTO_RECUSADO`/`TEMPO_ESGOTADO` com a Reserva liberada, e no tique seguinte `aplicarConfirmacao` marca a aprovação `NAO_APLICAVEL_SINALIZADA` **sem aviso** (o `WarnContext` exige `CANCELADO`). Mesmo desfecho por outros dois caminhos: `aplicarConfirmacao` devolve `nil` quando o `SKIP LOCKED` pula a linha, e uma confirmação que chega entre `Varrer` e `Expirar` no mesmo tique nunca é vista. Contraria a invariante 7 do addendum §2 |
| 2 | `atraso >= prazo` esvazia a janela de emissão em silêncio | **medium** | `ate = agora-atraso` e `desde = agora-prazo` (`internal/pagamento/pagamento.go:257`); invertidos, `criada_em <= ate AND criada_em > desde` nunca casa. São duas variáveis independentes e `CarregarConfig` valida pares comparáveis (`config.go:141,144,147`) mas não esta. Nenhum erro, nenhum log: todo Pedido passaria a expirar |
| 3 | Os cortes saem do relógio do host e são comparados com instantes escritos pelo Postgres | **medium** | `avancarPeloTempo` e `EmitirConfirmacoesDevidas` usam `time.Now()`; `ocorrido_em` e `criada_em` têm `DEFAULT now()` do banco. A estória diagnosticou essa classe (uma piscada em quatro no baseline) e corrigiu **só os chamadores de teste**. Com o prazo de 60 s da demonstração, segundos de deriva são ~10% do prazo. Pré-existente na 1.8 e na emissão; a 5.11 o põe no caminho da FR-34 |
| 4 | Nada prende o prazo do tique ao mesmo campo da Config que a tela conta | **medium** | `api/expiracao_test.go` passa `prazoLargo` (30 min) enquanto a tela responde com `expiracaoDaTentativaDeTeste` (15 min), e `oTiqueExpiraNaOrdemDoAD6` envelhece uma hora — qualquer duração entre ~0 e 1 h passa. Trocar `main.go:176` por `cfg.PagamentoTentativaExpiracao*2` deixa a suíte verde, e a tela passaria a discordar da varredura |
| 5 | Nada prova que `expirar` está fora do interruptor `EntregaSimulacaoAtiva` | **medium** | `oTiqueExpiraNaOrdemDoAD6` roda com o interruptor ligado, e `interruptorDesligaASimulacao` cria só um Pedido `PAGO`. Mover a chamada para dentro do `if` deixa `go test ./...` verde, e a FR-34 🔒 vira dependente de configuração |
| 6 | O próprio `oTiqueExpiraNaOrdemDoAD6` pisca | **medium** | `novo("AZ-EXPIRACAO-00002")` grava a transição recuada uma hora **antes** de inserir a Tentativa e a confirmação, com o binário já rodando e o tique em 1 s: um tique no meio expira `noPrazo` antes de a inbox existir, e a asserção `PAGO` morre. Achado novo, no teste escrito para prender a ordem |
| 7 | `oTiqueExpiraNaOrdemDoAD6` não confere que o Estoque voltou | **medium** | Fabrica a Reserva com `INSERT … 'ATIVA'` sobre `ORDER BY id LIMIT 1` e afere só `estado = 'LIBERADA'`; passaria com a prateleira vazia. `api/expiracao_test.go` já tem a forma certa (`disponivelDe`) |
| 8 | Aprovação tardia sobre Pedido já expirado some sem aviso | **medium** | `aplicarConfirmacao` só emite `WarnContext` quando o Pedido travado está `CANCELADO`; expirado cai em `NAO_APLICAVEL_SINALIZADA` calado. Era inalcançável antes da 5.11 e agora é o desfecho normal da faixa `,95`–`,99` paga em atraso. Mesma superfície de decisão do achado 1 |
| 9 | `Expirar` sem teto de lote no caminho da FR-34 | **low** | Depois de parada longa, o primeiro tique expira todo Pedido vencido em sequência, atrasando `aplicar` e `emitir` no mesmo ticker. A nota `ponytail:` de `PedidosParaAvancar` cobre a varredura sequencial, não o lote por tique |
| 10 | `numero` fixo (`AZ-EXPIRACAO-00001/2`) quebra `-count=2` | **low** | `pedido.numero` é `NOT NULL UNIQUE`. Precedente idêntico já existia em `AZ-INTERRUPTOR-0001`; não é regressão desta estória |
| 11 | Loop "parado" redundante em `TestOsMapasDoTempo` | **low** | Os dois `maps.Equal` já fixam os mapas inteiros; o loop seguinte não pode falhar sozinho. Introduzido nesta sessão, na limpeza do código morto |
| 12 | A metade `!travado.Desde.Valid` da reconferência é tão inalcançável quanto a `.After` | **low** | `PedidosParaAvancar` filtra por `max(ocorrido_em) <= @ate`, que é NULL para Pedido sem histórico — nunca selecionado. A entrada do `deferred-work.md` nomeia só a comparação `.After` |
| 13 | `fmt.Errorf("calcular o corte de %s", passo.ator)` nomeia o ator, não o relógio | **low** | A linha substituída dizia "do intervalo de entrega"; a nova rende "calcular o corte de VARREDURA" |
| 14 | `HANDOFF.md` diz que a 5.11 "está em `main`" sem hash, e o `last_updated` do sprint diverge do memlog | **low** | Correto no instante do diff (nada foi commitado ainda); o hash entra no commit |
| 15 | O piso da emissão (`criada_em`) e o vencimento (histórico) poderiam divergir | **false** | Refutado: `criada_em` e `ocorrido_em` têm ambos `DEFAULT now()` e são escritos na **mesma transação** (`Criar` e `NovaTentativa`), e `now()` no Postgres é o instante de início da transação — idênticos por construção, não por coincidência |
| 16 | A linha estagnada da `## Verification` contradiz a `Verificação executada` do mesmo arquivo | **low, rejeitado** | Real, mas a correção é editar a spec desta construção — rejeitado pela regra de triagem |
| 17 | A linha "Corrida perdida" da matriz promete `ErrEstadoJaAvancado`, inalcançável na expiração | **low, rejeitado** | Confirmado (a releitura travada devolve `false` antes do compare-and-swap) e já registrado nas Implementation Notes, mas a correção é editar o bloco congelado — que é do humano |

**Roteamento.** O achado 1 é `intent_gap`: o bloco congelado nomeia a **ordem do tique** como o mecanismo que protege a aprovação recebida no prazo, e a ordem sozinha não a protege. A intenção ("uma aprovação que chegou no prazo não pode ser descartada por um relógio") é clara, mas há duas leituras defensáveis de até onde a implementação deve ir, com pegadas bem diferentes — e a regra proíbe inferir intenção quando há mais de uma leitura. Os achados 8 e 2 dividem a mesma superfície de decisão. Pela cascata, os demais ficam suspensos até a intenção ser resolvida: o código será re-derivado.

**Passe 2 — 2026-09-22, depois da emenda do humano.** Os achados 1, 2 e 8 foram fechados no código (inbox sob a trava por `pagamento.TemConfirmacaoPendente`, `CarregarConfig` recusando o par, e o `WarnContext` cobrindo o expirado). Os achados 4, 5, 6, 7, 11, 12 e 13 foram corrigidos como `patch`. Os achados 3, 9 e 10 foram para o `deferred-work.md` com o porquê. Os achados 14, 16 e 17 ficaram como estavam: o hash do `HANDOFF.md` entra no commit, e os dois últimos só se corrigiriam editando esta spec ou o bloco congelado.

**Auditoria da matriz (sessão que orquestra).** As onze linhas têm teste que rodou e passou — conferido por `go test -v -run`, nenhum pulado. As três linhas novas: "aprovação pendente que `aplicar` não aplicou" e "aprovação tardia sobre Pedido expirado" em `api/expiracao_test.go` (a segunda captura o log, então o aviso é asserção e não promessa), e "janela de emissão invertida" em `TestConfigComAtrasoAlcancandoAExpiracaoFalha`. A linha "Corrida perdida" continua provada no comportamento observável e não no mecanismo que a coluna nomeia — registrado nas Implementation Notes e no achado 17.

**Verificação independente (2026-09-22, depois dos patches):** `go vet ./...` limpo e `go test -count=1 ./...` verde na imagem `azamon-dev`, `internal` (o teste de fronteira do AD-1) inclusive — a aresta nova `pedido → pagamento` é legal. `docker build ./web` verde. O par da demonstração no `.env` (atraso 5 s, prazo 60 s) passa na validação nova, então `docker compose up` continua subindo.

## Design Notes

**Por que `PedidosParaAvancar` serve sem mudar.** Para um Pedido em `AGUARDANDO_PAGAMENTO`, `max(ocorrido_em)` **é** a última transição para esse Status: qualquer linha posterior o teria tirado de lá. É por isso que o candidato da varredura e o `expira_em` da tela caem no mesmo instante sem duas contas.

**Por que a expiração não é mais uma linha do mapa `simulacao`.** Relógios diferentes (`AZAMON_ENTREGA_INTERVALO`, 24 h, contra `AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO`, 15 min) e interruptores diferentes: a simulação de entrega desliga por configuração, a FR-34 🔒 nunca.

Quem implementa não faz `git commit`, `git add` nem `push`.

## Verification

**Commands:**
- `go vet ./... && go test -count=1 ./...` na imagem `azamon-dev` (HANDOFF) -- expected: verde.
- `docker build ./web` -- expected: verde (nada do `web/` muda, e é isso que se confirma).
- Mutação: pôr `Expirar` **antes** de `Varrer` no tique -- expected: "a aprovação que chegou no prazo vence a expiração" falha. Reverter.
- Mutação: trocar o ator por `AtorProvedor` ou o motivo por `RECUSADO_PELO_PROVEDOR` -- expected: `TransicaoInvalida` derruba o subteste do vencido (a tabela do AD-3 só aceita `VARREDURA` com `TEMPO_ESGOTADO`). Reverter.
- Mutação: tirar a reconferência `travado.Desde.After(ate)` da transação -- expected: o subteste do prazo por vencer falha. Reverter.
- Mutação: tirar o piso `criada_em > @desde` de `TentativasSemConfirmacao` -- expected: o teste da Tentativa passada do prazo falha. Reverter.

**Manual checks (if no CLI):**
- Roteiro B passo 5 no navegador a 360 e 1440 px, só pelo teclado: Produto que feche em `,95` → o relógio de 60 s chega a zero → a tela troca o relógio por "Tempo de pagamento expirado." sem recarregar → outra aba mostra o Estoque de volta → "Tentar pagar de novo" traz relógio novo e, em ~5 s, `PAGO`.
