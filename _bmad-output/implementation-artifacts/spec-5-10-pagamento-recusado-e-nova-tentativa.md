---
title: 'Estória 5.10 — Pagamento recusado e nova Tentativa'
type: 'feature'
created: '2026-09-22'
status: 'done'
baseline_commit: '4832120f723315792c66a32a1f4ef947aa9cb83f'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** o Provedor Simulado decide `RECUSADO` para a faixa `,90`–`,94`, mas a emissão só envia o aprovado e a varredura só aplica o aprovado: o Roteiro B para no passo 1, com o Pedido preso em `AGUARDANDO_PAGAMENTO`. A transição `PAGAMENTO_RECUSADO → AGUARDANDO_PAGAMENTO` existe na máquina desde a 5.1, mas nenhuma rota a alcança, `IniciarTentativa` grava `numero = 1` fixo e o teto de 3 (AD-8) não é imposto em lugar nenhum.

**Approach:** emitir e aplicar a recusa pelo mesmo caminho do aprovado (Tentativa corrente, Pedido em `AGUARDANDO_PAGAMENTO`, `Transicionar` com ator `PROVEDOR` liberando a Reserva); `IniciarTentativa` passa a derivar o número e a recusar com `ErrTetoDeTentativas`; uma rota do Comprador inicia a nova Tentativa a partir do próprio Pedido; e a superfície recusada da tela ganha "Tentar pagar de novo", derivado da tripla do AD-18.

## Boundaries & Constraints

**Always:**
- Aplicar só a recusa da Tentativa corrente com o Pedido em `AGUARDANDO_PAGAMENTO`; todo o resto continua `NAO_APLICAVEL_SINALIZADA` — uma recusa atrasada da Tentativa 1 nunca derruba a Reserva da 2.
- A recusa grava o motivo `RECUSADO_PELO_PROVEDOR` na transição; `TEMPO_ESGOTADO` continua reservado à 5.11.
- Nova Tentativa numa transação só, nesta ordem: Pedido do dono (AD-11) → `Transicionar(PAGAMENTO_RECUSADO → AGUARDANDO_PAGAMENTO, COMPRADOR)` (CAS, histórico, nova Reserva sobre os Itens do próprio Pedido) → `pagamento.IniciarTentativa`. A contagem acontece com a linha do Pedido presa pelo CAS; qualquer recusa desfaz tudo e o Pedido permanece em `PAGAMENTO_RECUSADO`.
- O teto é de `pagamento` (AD-8), vem da Config (AD-13) e sai de lá como `pagamento.ErrTetoDeTentativas`; `pedido` o traduz em `pedido.ErrTentativasEsgotadas`, que é o registrado em `plataforma/erro` (409, `TETO_DE_TENTATIVAS`) — o AD-1 não tem aresta `plataforma → pagamento`. *(Emendado pelo humano em 2026-09-22, na revisão.)*
- "Tentar pagar de novo" aparece só com `status = PAGAMENTO_RECUSADO`, `tentativas_restantes > 0` e todo Item `disponivel`. Sem Tentativa ou sem Estoque, a tela diz por quê, nomeando o Produto.
- O botão usa `aria-disabled` e um envio só por clique (`useRef`), sem laranja: o laranja é só do Confirmar Pedido.
- Glossário literal: Tentativa de Pagamento, Provedor de Pagamento, Reserva de Estoque, Item de Pedido.

**Ask First:**
- Migração nova, consulta SQL nova, ou mudança no corpo de `GET /api/v1/pedidos/{id}`.
- Qualquer mudança na ordem do tique em `cmd/azamon`.

**Never:**
- "Cancelar Pedido" (6.3), expiração (5.11), superfície do Administrador.
- Remontar o Pedido a partir do Carrinho, ou tocar no Carrinho.
- Fazer a emissão consultar o Status do Pedido (AD-7), ou fazer o navegador decidir o teto.
- Qualquer dado de cartão.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Recusa aplicada | Tentativa 1 de total `,90`, Pedido aguardando | `PAGAMENTO_RECUSADO` com ator `PROVEDOR` e o motivo; Reserva `LIBERADA`; inbox `APLICADA`; a Página de Produto mostra o Estoque de volta | — |
| Recusa superada ou tardia | Tentativa 1 recusada chegando com a 2 aberta | Pedido e Reserva da 2 intactos; `NAO_APLICAVEL_SINALIZADA` | — |
| Nova Tentativa | Pedido recusado, 2 restantes, Estoque disponível | 201; `AGUARDANDO_PAGAMENTO`, nova Reserva `ATIVA`, Tentativa `numero = 2`, novo `expira_em`; a emissão aprova | — |
| Sem Estoque | outro Pedido levou a unidade | 409 `ESTOQUE_INSUFICIENTE` com `produto_id`; Pedido permanece recusado e nada é gravado | a tela relê; o botão some e o Produto aparece nomeado |
| Teto atingido | 3 Tentativas feitas | 409 `TETO_DE_TENTATIVAS`; nada é gravado | a tela relê; o botão some |
| Duplo clique ou Status mudou | Pedido já aguardando, pago ou cancelado | 409 `ESTADO_JA_AVANCADO`, sem Tentativa extra | a tela relê, sem aviso de erro |
| Pedido alheio ou inexistente | outro Comprador ou uuid ruim | 404 `NAO_ENCONTRADO` | — |

</frozen-after-approval>

## Code Map

- `internal/pagamento/pagamento.go:115` -- `IniciarTentativa`: hoje grava `primeiraTentativa` fixo. Passa a receber `teto int`, derivar `numero = ContarTentativas + 1` (consulta existente, `:167`) e recusar acima do teto com o novo `ErrTetoDeTentativas`. `EmitirConfirmacoesDevidas:233` deixa de filtrar só `Aprovado` e passa a pular só `SemConfirmacao`. Atualize o comentário `:219` ("só emite o caminho aprovado").
- `internal/pedido/pedido.go:580` -- `aplicarConfirmacao`: o `if` que aplica ganha o ramo `Recusado` → `Transicionar(..., StatusAguardandoPagamento, StatusPagamentoRecusado, AtorProvedor, MotivoRecusadoPeloProvedor)`, que já libera a Reserva (`maquina.go:115,232`). `ErrEstadoJaAvancado` segue o mesmo tratamento. `aprovadaSobreCancelado` não muda.
- `internal/pedido/pedido.go:121,311` -- `Criar` passa o teto a `IniciarTentativa`, então ganha o parâmetro. Chamador único: `api/pedido.go:195`, com `s.cfg.PagamentoTentativasMax`.
- `internal/pedido/pedido.go` -- nova `NovaTentativa(ctx, tx, pedidoID, compradorID string, teto int) (Pedido, error)`, com a ordem do bloco congelado. Leitura com dono: `q.BuscarPedidoDoComprador` (já usada em `Detalhar:413`, traz `total_centavos`). O efeito de Reserva já é do `Transicionar` (`maquina.go:239`, `reservarDeNovo`), e quem chama não o repete.
- `internal/pedido/maquina.go:43` -- `MotivoRecusadoPeloProvedor` ao lado de `MotivoTempoEsgotado`. Não é reservado.
- `api/pedido.go` + `api/rotas.go:59` -- `POST /api/v1/pedidos/{id}/tentativas` dentro de `emTransacao` (`api/transacao.go:36`). O 201 devolve `saidaDoPedido`. Os erros saem pelo molde de `criarPedido`: `ErrNoRows` → 404, `EstoqueInsuficiente` → `dados{produto_id, disponivel}`.
- `internal/plataforma/erro/erro.go:46` -- linha `{pedido.ErrTentativasEsgotadas, 409, "TETO_DE_TENTATIVAS"}`. **Não** `pagamento.ErrTetoDeTentativas`: o AD-1 não tem aresta `plataforma → pagamento`, e `internal/fronteira_test.go` a recusa; `pedido.NovaTentativa` traduz a causa de `pagamento` embrulhando as duas (AD-8, "`pedido` a traduz").
- `web/lib/pedido.ts` -- `motivoDaRecusa` nomeia `RECUSADO_PELO_PROVEDOR`; novas puras: a ação da tripla (`podeTentarDeNovo`), o impedimento em texto e o desfecho da resposta (`releitura` / `semSessao` / `erro`), no molde de `desfechoDaConfirmacao` (`web/lib/checkout.ts:415`).
- `web/app/pedidos/[id]/acompanhamento.tsx` -- o botão na superfície `recusado`; a releitura após o envio precisa alcançar o `consultar` do `useEffect`, via ref. O molde de envio único é `revisao-do-pedido.tsx:101,385`.
- `api/webhook_test.go:60-76` -- **quebra**: o `enviar` do caminho feliz posta toda Tentativa vencida, e com a recusa emitida os Pedidos de R$ 249,90 dos subtestes anteriores seriam recusados e saíriam de `AGUARDANDO_PAGAMENTO`, de onde `umPedidoEm` (`:497`) os tira depois. O transporte entrega só a Tentativa deste Pedido e anota o resto.
- `api/maquina_test.go:213` -- a nova Tentativa sem Estoque já está provada no nível do `Transicionar`; o teste novo prova pela rota.

## Tasks & Acceptance

**Execution:**
- [x] `internal/pagamento/pagamento.go` + `pagamento_test.go` -- teto, número derivado, `ErrTetoDeTentativas`, emissão da recusa.
- [x] `internal/pedido/maquina.go`, `pedido.go` -- motivo, ramo da recusa em `aplicarConfirmacao`, `Criar` com o teto, `NovaTentativa`.
- [x] `internal/plataforma/erro/erro.go` + `erro_test.go` -- registro e caso de tabela do novo código.
- [x] `api/pedido.go`, `api/rotas.go` -- rota nova; `criarPedido` passa o teto.
- [x] `api/webhook_test.go`, `api/sessao_test.go` -- ajustar o transporte do caminho feliz; registrar `recusaENovaTentativa` depois de `detalheDoPedido`. A recusa aplicada e a superada foram para `api/nova_tentativa_test.go`, com Produtos próprios: comprar o `produtoSemeado` deixaria Reserva ativa no disponível que os subtestes seguintes conferem.
- [x] `api/nova_tentativa_test.go` (novo) -- as linhas da matriz pela rota: 201 com `numero = 2` e Reserva `ATIVA`, emissão da 2 aprovando até `PAGO`; sem Estoque, teto, duplo clique e 404, cada um conferindo que nenhuma Tentativa, Reserva ou transição foi gravada.
- [x] `web/lib/pedido.ts`, `web/scripts/pedido.test.mjs`, `web/app/pedidos/[id]/acompanhamento.tsx` -- funções puras testadas e o botão.
- [x] `deferred-work.md` (fechar as entradas das linhas 246–248 e 385–387), `sprint-status.yaml` (`5-10-…` para `review`), `addendum.md` §10 + memlog do PRD, e o AD-8 emendado no lugar (assinatura com `teto`) + memlog da arquitetura.

**Acceptance Criteria:**
- Given um Pedido de total `,90`, when o tique emite e aplica, then o Pedido chega a `PAGAMENTO_RECUSADO` sozinho e a tela troca o relógio por "O Provedor de Pagamento recusou a Tentativa de Pagamento." e pelas Tentativas restantes, sem recarregar.
- Given esse Pedido, when o Comprador clica "Tentar pagar de novo", then a tela volta ao relógio com prazo novo e, alguns segundos depois, vira `PAGO` — o passo 4 do Roteiro B.
- Given o caminho aprovado e o cancelado da 1.7 e da 5.9, when a suíte roda, then os dois continuam verdes.

## Spec Change Log

- **Revisão (três lentes: Blind Hunter, Edge Case Hunter, Verification Gap) — 2026-09-22.** Um achado tocava o bloco congelado: a regra "sai como `pagamento.ErrTetoDeTentativas`, registrado em `plataforma/erro`" contrariava o AD-1, que não tem aresta `plataforma → pagamento` (o teste de fronteira a recusou). O humano escolheu emendar a frase, e o código seguiu o AD-8 ao pé da letra: `pedido` traduz o teto em `pedido.ErrTentativasEsgotadas`. Nenhum `bad_spec`; sem volta ao plano. Patches: o teste do duplo clique passou a forçar a sobreposição segurando a linha do Pedido até as duas requisições estarem presas em trava — sem isso a mutação "Tentativa antes do CAS" sobrevivia (verificado); a releitura da tela ganhou guarda de sequência (a consulta do intervalo que saiu antes do clique sobrescrevia a releitura e trazia o botão de volta); sem Estoque e sem Tentativa passaram a avisar além de reler; chaves de lista pelo Produto; `Object.hasOwn` em `motivoDaRecusa`; foco no título depois da nova Tentativa; o motivo fixado pelo literal no teste de Go; `nadaGravado` por fotografia completa (Status, Tentativas, histórico, Reservas, inbox); o 404 alheio sobre Pedido recusado; subteste do Produto desativado (a entrada da 5.1 que a estória fecha não tinha teste); e "compra" trocado por "Pedido" no addendum. Três adiados no `deferred-work.md`: `Corrente` fora da trava (alcançável com a 5.11), aprovação superada com Pedido ativo sem sinal, e `TentativasSemConfirmacao` crescendo para sempre. KEEP: a ordem CAS → `IniciarTentativa`; a recusa aplicada só com `Corrente` e o Pedido aguardando; a tela relendo o Detalhe em vez de guardar estado da tentativa; os Produtos próprios no teste.

## Design Notes

**Por que a contagem vem depois do CAS.** Com `IniciarTentativa` antes de `Transicionar`, dois cliques simultâneos contam o mesmo número e colidem no `UNIQUE` de `id_externo`: o segundo espera o primeiro e sai em 500. Depois do CAS, o segundo espera na linha do Pedido e cai em `ESTADO_JA_AVANCADO`, que é o desfecho que a tela já sabe tratar. O custo é que, com Estoque e teto esgotados ao mesmo tempo, a resposta é `ESTOQUE_INSUFICIENTE`, e a tela relida esconde o botão do mesmo jeito.

**Produto desativado depois de o Pedido nascer** (entrada da 5.1): sai como `disponivel: false` e `ESTOQUE_INSUFICIENTE`, igual ao esgotado. A FR-12 manda os dois saírem iguais, e a tela nomeia o Produto pelo `nome` congelado no Item de Pedido. Com isso a entrada fecha.

Textos da tela: sem Estoque, "Não há Estoque disponível de {nome} para uma nova Tentativa de Pagamento." (um por Item indisponível); sem Tentativa, o `textoDasTentativas(0)` que já existe.

Quem implementa não faz `git commit`, `git add` nem `push`.

## Verification

**Commands:**
- `go vet ./... && go test -count=1 ./...` na imagem `azamon-dev` (HANDOFF) -- expected: verde. Sem `sqlc generate`: nenhuma consulta nova.
- `docker build ./web` -- expected: verde (o `prebuild` roda `node --test` e o portão offline).
- Mutação: tirar `c.Corrente &&` do `if` que aplica em `aplicarConfirmacao` -- expected: "a recusa de uma Tentativa superada não derruba a Reserva da nova" falha. Reverter.
- Mutação: o ramo `Recusado` levar a `StatusPago` -- expected: "a recusa do Provedor é aplicada e o Estoque volta" falha. Reverter.
- Mutação: em `NovaTentativa`, `IniciarTentativa` antes de `Transicionar` -- expected: "o duplo clique abre uma Tentativa só" falha (o perdedor sai 500). Reverter.
- Mutação: em `IniciarTentativa`, voltar ao `numero` fixo em 1 -- expected: o 201 da nova Tentativa e o teste do teto falham. Reverter.

**Manual checks (if no CLI):**
- Roteiro B passos 1–4 no navegador a 360 e 1440 px, só pelo teclado: Produto de `,90` → recusado sozinho → outra aba mostra o Estoque de volta → "Tentar pagar de novo" → relógio → `PAGO`.

## Suggested Review Order

**A recusa aplicada (FR-27)**

- Entrada: o mesmo `if` aplica aprovado e recusado; só a Tentativa corrente com o Pedido aguardando.
  [`pedido.go:648`](../../internal/pedido/pedido.go#L648)

- A recusa grava `RECUSADO_PELO_PROVEDOR`; liberar a Reserva é do próprio `Transicionar`.
  [`pedido.go:642`](../../internal/pedido/pedido.go#L642)

- A emissão passa a mandar o recusado; só a faixa que nunca confirma cala.
  [`pagamento.go:263`](../../internal/pagamento/pagamento.go#L263)

**A nova Tentativa a partir do Pedido**

- A ordem é o contrato: dono, CAS com nova Reserva, e só então a Tentativa.
  [`pedido.go:346`](../../internal/pedido/pedido.go#L346)

- Tentativa depois do CAS: o duplo clique sai `ESTADO_JA_AVANCADO`, e não 500.
  [`pedido.go:367`](../../internal/pedido/pedido.go#L367)

- Número derivado e teto conferido por `pagamento`, dono da Tentativa (AD-8).
  [`pagamento.go:135`](../../internal/pagamento/pagamento.go#L135)

- `pedido` traduz o teto; é a tradução que a borda HTTP conhece (AD-1).
  [`pedido.go:372`](../../internal/pedido/pedido.go#L372)

- A rota sem corpo; `SameSite=Lax` a protege como a entrada no checkout.
  [`api/pedido.go:247`](../../api/pedido.go#L247)

- O código novo no registro, pela tradução de `pedido`.
  [`erro.go:81`](../../internal/plataforma/erro/erro.go#L81)

**A tela: botão, impedimento e releitura**

- A ação sai da tripla do AD-18; só o Status apareceria para sempre.
  [`pedido.ts:121`](../../web/lib/pedido.ts#L121)

- Sucesso e recusas do Pedido relêem; Estoque e teto também avisam.
  [`pedido.ts:171`](../../web/lib/pedido.ts#L171)

- Envio único por `useRef`, releitura forçada e foco no título.
  [`acompanhamento.tsx:267`](../../web/app/pedidos/[id]/acompanhamento.tsx#L267)

- Só a consulta mais recente escreve: a antiga traria o botão de volta.
  [`acompanhamento.tsx:212`](../../web/app/pedidos/[id]/acompanhamento.tsx#L212)

**As provas**

- O duplo clique forçado a se sobrepor, segurando a linha do Pedido.
  [`nova_tentativa_test.go:309`](../../api/nova_tentativa_test.go#L309)

- A recusa superada não derruba a Reserva da nova — o que o `Corrente` guarda.
  [`nova_tentativa_test.go:408`](../../api/nova_tentativa_test.go#L408)

- Da emissão ao Estoque de volta na Página de Produto (Roteiro B, passo 3).
  [`nova_tentativa_test.go:181`](../../api/nova_tentativa_test.go#L181)

- O teto recusa sem gravar nada, pela fotografia completa.
  [`nova_tentativa_test.go:386`](../../api/nova_tentativa_test.go#L386)

- O Produto desativado sai igual ao esgotado (FR-12).
  [`nova_tentativa_test.go:448`](../../api/nova_tentativa_test.go#L448)

**Periferia**

- O caminho feliz da 1.7 entrega só a própria Tentativa, e anota o resto.
  [`webhook_test.go:71`](../../api/webhook_test.go#L71)

- O AD-8 emendado no lugar: a assinatura com o teto.
  [`ARCHITECTURE-SPINE.md:170`](../planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md#L170)
