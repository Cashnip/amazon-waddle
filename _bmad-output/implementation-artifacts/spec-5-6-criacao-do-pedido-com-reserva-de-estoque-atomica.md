---
title: 'Estória 5.6 — Criação do Pedido com Reserva de Estoque atômica'
type: 'feature'
created: '2026-09-21'
status: 'done'
baseline_commit: '7b0f84c96a23af9355ee8fc62c0addd10f2565ad'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** o `POST /api/v1/pedidos` ainda é o do esqueleto — um Produto, uma unidade, sem Endereço, Frete, Carrinho nem idempotência —, e o Confirmar Pedido da Revisão está desligado. A FR-23, a FR-24 e a NFR-12 não existem de ponta a ponta.

**Approach:** refazer a criação a partir do Carrinho, numa transação: reivindicar a `Idempotency-Key` inserindo o Pedido, reservar todos os Itens sob a trava do AD-5, **recalcular o total sob essa trava** e recusar com `TOTAL_DIVERGENTE`, congelar Itens (preço praticado, Vendedor), Endereço, Frete e total, esvaziar os Itens lidos por `carrinho.Esvaziar` e abrir a Tentativa. A Revisão liga o botão, envia a chave e chama `limparCheckout`; o botão do esqueleto sai da Página de Produto.

## Boundaries & Constraints

**Always:**
- Corpo `{endereco_id, total_centavos}` (o total que a Revisão exibiu) e cabeçalho `Idempotency-Key` com forma de uuid; sem ele, ou fora da forma, 400.
- Ordem dentro da transação: (1) chave já usada por este Comprador → mesmo digest devolve o Pedido original (200), digest diferente é 409; (2) Endereço do dono, Carrinho, Frete e total, com `TOTAL_DIVERGENTE` se já difere do corpo; (3) `INSERT` do Pedido com a chave, que é a reivindicação — o gêmeo concorrente espera no índice único e cai no passo (1); (4) `catalogo.Reservar`; (5) **só então** releitura dos preços e do Frete, e `TOTAL_DIVERGENTE` se mudou; (6) Itens, histórico, `carrinho.Esvaziar`, `pagamento.IniciarTentativa`.
- Chave e digest são do Comprador: índice único `(comprador_id, chave_idempotencia)`. O `23505` nunca chega ao navegador (`ON CONFLICT DO NOTHING` e releitura).
- `CHECK (total_centavos = subtotal_centavos + frete_centavos)`. Congelamento em colunas do `INSERT`, protegidas pelo gatilho da 5.1.
- `carrinho.Esvaziar(ctx, tx, compradorID, itemIDs)` apaga **só** os Itens lidos; linha a menos é `carrinho.ErrCarrinhoMudou` e desfaz tudo.
- `api/` repete a transação uma vez, e só uma, em `40P01`/`40001`.
- Glossário literal; o laranja continua sendo só o Confirmar Pedido da Revisão.

**Ask First:**
- Mudar o contrato de `/api/v1/carrinho`, `/api/v1/frete` ou `/api/v1/checkout/entrada`.
- Sentinela ou coluna além de `ErrTotalDivergente`, `ErrCarrinhoVazio`, `ErrChaveReutilizada`, `carrinho.ErrCarrinhoMudou` e das colunas desta spec.

**Never:**
- Criar Pedido fora do Carrinho, ou manter o caminho de um Produto.
- Somar total ou decidir divergência no Node (NFR-13, AD-10).
- Mexer no teste de concorrência do NFR-7 (5.7) ou nas telas pós-criação (5.8+).

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Criação | Carrinho de 2 Produtos, Endereço de SP, total da Revisão | 201; Pedido `AGUARDANDO_PAGAMENTO` com subtotal, Frete R$ 15,00, Endereço e Itens congelados; Reservas ATIVAS; Carrinho vazio; uma Tentativa | — |
| Reenvio | mesma chave, mesmo corpo | 200, o mesmo Pedido; nada novo no banco | — |
| Chave reaproveitada | mesma chave, outro corpo | 409 `CHAVE_REUTILIZADA`, com o Pedido original em `dados` | tela chama `limparCheckout` e mostra o Pedido |
| Duplo clique concorrente | duas requisições iguais em paralelo | um Pedido; as duas respostas com o mesmo `id` | — |
| Preço mudou | Administrador altera o preço depois da Revisão | 409 `TOTAL_DIVERGENTE`; nada gravado | tela relê a Revisão e avisa |
| Falta Estoque | um dos Itens sem disponível | 409 `ESTOQUE_INSUFICIENTE`; nenhum Pedido, nenhuma Reserva, Carrinho intacto | link ao Carrinho |
| Carrinho vazio | sem Itens | 409 `CARRINHO_VAZIO` | link ao Carrinho |
| Endereço alheio | `endereco_id` de outro Comprador | 404, nada gravado | volta ao passo Endereço |
| Item adicionado noutra aba | Item novo entre a leitura e o fim | fica no Carrinho; não entra no Pedido | — |
| Sem Sessão | sem cookie | 401 | Login com `destino` |

</frozen-after-approval>

## Code Map

- `api/pedido.go:47` -- `criarPedido` do esqueleto (entrada `{produto_id}`), a refazer; o molde de transação (`WithoutCancel`, JSON só depois do `Commit`) fica.
- `internal/pedido/pedido.go:69` -- `Criar` do esqueleto; a constante `unidade:47` sai. `CotarFrete` e `CalcularFrete` (`frete.go:121,160`) são o cálculo a reusar nos dois pontos do total.
- `internal/pedido/db/consultas.sql:21` -- `CriarPedido`/`CriarItemPedido` ganham as colunas; entra `PedidoPorChave` (dono + chave → id, numero, status, total, digest).
- `db/migracoes/20260919120000_pedido_maquina_de_estados.sql:93` -- o gatilho que compara `to_jsonb(NEW) - 'status'`: colunas novas nascem protegidas, e o preenchimento das linhas antigas o desliga dentro da migração nova.
- `internal/catalogo/catalogo.go:224` -- `Reservar`: trava `ORDER BY id`, soma, insere; Produto invisível sai `EstoqueInsuficiente{Disponivel:0}`. `BuscarProduto:65` dá preço e `VendedorNome` sob a trava.
- `internal/carrinho/carrinho.go:217` -- `Itens` (leitura), `Limpar:254` (não usar), `ConfirmarPrecoVisto:319` (a escrita parcial passa a sair como `ErrCarrinhoMudou`). Consulta molde: `RemoverItem` em `db/consultas.sql:67`.
- `internal/identidade/endereco.go:64` -- `BuscarEndereco` com dono no `WHERE`.
- `internal/pagamento/pagamento.go:108` -- `IniciarTentativa(ctx, tx, pedidoID, total)`, sem mudança.
- `internal/plataforma/erro/erro.go:46` -- `registro`: as linhas novas entram aqui.
- `api/checkout.go` -- `entrarNoCheckout`, segunda usuária da repetição em `40P01`.
- `api/pedido_test.go:415` -- `postarPedido` e ~20 chamadores em `endereco_test`, `estoque_test`, `maquina_test`, `produto_admin_test`, `vendedor_test`, `webhook_test`, `sessao_test`. Helpers de bancada: `postarItem` (`carrinho_test.go:365`), `postarEndereco` (`endereco_test.go:397`), `pegarFrete` (`frete_test.go:161`).
- `web/app/checkout/revisao/revisao-do-pedido.tsx:56` -- `CRIACAO_DISPONIVEL`, a única linha a virar; o botão ganha `onClick`.
- `web/lib/checkout.ts:290,310` -- `chaveDeIdempotencia`, `limparCheckout`; testes em `web/scripts/checkout.test.mjs`.
- `web/app/produtos/[id]/comprar.tsx` + `page.tsx:98` -- o botão do esqueleto, a remover. `README.md:79` descreve esse passeio.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/<timestamp>_pedido_criacao.sql` (`goose create`) -- em `pedido.pedido`: `subtotal_centavos`, `frete_centavos` (linhas antigas: subtotal = total, Frete 0, gatilho desligado só no preenchimento), o `CHECK` do AD-9, as oito colunas `endereco_*` e `chave_idempotencia uuid`/`digest_corpo text` anuláveis (só Pedido do esqueleto as tem nulas) com índice único `(comprador_id, chave_idempotencia)`; gatilho que recusa `INSERT` em `item_pedido` de Pedido que já tem histórico -- fecha o `INSERT` tardio da 5.1.
- [x] `internal/pedido/db/consultas.sql` + `internal/carrinho/db/consultas.sql` + `sqlc generate` -- `CriarPedido` com `ON CONFLICT DO NOTHING`, `PedidoPorChave`, `EsvaziarItens :execrows` com dono no `WHERE` -- restaurar os `gerado/` alheios.
- [x] `internal/carrinho/carrinho.go` -- `Esvaziar` e `ErrCarrinhoMudou`, também na escrita parcial de `ConfirmarPrecoVisto`.
- [x] `internal/pedido/pedido.go` -- `Criar(ctx, tx, compradorID, NovoPedido{EnderecoID, TotalCentavos, Chave}, isencao) (Pedido, bool, error)`, na ordem de Always; o `bool` diz reenvio; digest calculado aqui.
- [x] `internal/plataforma/erro/erro.go` -- `TOTAL_DIVERGENTE`, `CARRINHO_VAZIO`, `CARRINHO_MUDOU`, `CHAVE_REUTILIZADA`, todos 409.
- [x] `api/transacao.go` (novo) + `api/pedido.go` + `api/checkout.go` -- `emTransacao` com a repetição única; `criarPedido` novo (reenvio desfaz a transação e responde 200); `entrarNoCheckout` passa a usá-la.
- [x] `api/pedido_test.go` + chamadores -- helper `pedidoPeloCheckout(produto, quantidade)` (limpa o Carrinho, adiciona, garante Endereço de SP, cota, confirma com chave nova); a matriz de servidor inteira, com o duplo clique em duas goroutines; totais com Frete. Testes de Produto desativado passam a pôr no Carrinho antes e esperar `ESTOQUE_INSUFICIENTE`.
- [x] `api/transacao_test.go` -- `40P01` uma vez repete; duas vezes desiste; outro código não repete.
- [x] `web/lib/checkout.ts` + `web/scripts/checkout.test.mjs` -- `requisicaoDeConfirmar(chave, enderecoID, total)` e `desfechoDaConfirmacao(resposta)`, puras e testadas.
- [x] `web/app/checkout/revisao/revisao-do-pedido.tsx` -- `CRIACAO_DISPONIVEL = true`; envio único por `useRef` e botão travado em voo; sucesso: `limparCheckout`, `avisarCarrinhoAlterado`, `/pedidos/{id}`.
- [x] `web/app/produtos/[id]/comprar.tsx`, `page.tsx`, `README.md` §2 -- remover o botão; o passeio passa por Carrinho → Endereço → Revisão.
- [x] `addendum.md` §10 + memlog, `deferred-work.md`, `sprint-status.yaml` -- decisões abaixo; RESOLVIDO nas entradas fechadas; 5-6 para `review`.

**Acceptance Criteria:**
- Given dois Produtos no Carrinho, um sem Estoque, when o Comprador confirma, then nenhum Pedido nem Reserva nasce, e o disponível do outro não muda.
- Given um Pedido criado, when o preço do Produto e o Endereço mudam ou somem, then Itens, Frete, Endereço e total do Pedido continuam os mesmos.
- Given a mesma chave depois de um `TOTAL_DIVERGENTE`, when o Comprador confirma o total novo, then o Pedido nasce — recusa não consome a chave.

## Design Notes

**A chave só se prende quando o Pedido existe.** Ela é coluna do próprio Pedido, então recusa por Estoque, por total ou por Carrinho desfaz a transação e a deixa livre — é isso que resolve a dívida da 5.4 (voltar ao Carrinho, mudar a quantidade e confirmar sob a mesma chave não é 409). O 409 só sobra para a chave que **já criou** um Pedido; aí a tela mostra esse Pedido e apaga a chave.

**Por que o Pedido é inserido antes da trava dos Produtos.** O `INSERT` é a reivindicação da chave: o gêmeo do duplo clique espera no índice único antes de tocar Estoque. Invertido, o gêmeo chegaria à última unidade depois do primeiro e voltaria `ESTOQUE_INSUFICIENTE` em vez do Pedido original. O total vai no `INSERT` calculado antes da trava, e o passo (5) confirma que nada mudou — o gatilho impede reescrever.

`vendedor_nome` congelado desde a 1.6 já cumpre "o Pedido registra o Vendedor de cada Item" (FR-23): nenhuma coluna de Vendedor nova.

Quem implementa não faz `git commit`, `git add` nem `push`.

## Verification

**Commands:**
- `sqlc generate && go vet ./... && go test -count=1 ./...` -- expected: verde, na imagem `azamon-dev` (HANDOFF).
- `cd web && npm test && docker build ./web` -- expected: verde.
- `grep -rn "carrinho.Esvaziar\|produto_id" api/pedido.go internal/pedido/pedido.go web/app/produtos` -- expected: `Esvaziar` com um chamador (`pedido.Criar`); nenhum `produto_id` como corpo da criação.

**Manual checks (if no CLI):**
- A 360 e 1440 px, só pelo teclado: Carrinho → Endereço → Revisão → Confirmar Pedido leva a `/pedidos/{id}` com o total da Revisão; o Carrinho da barra zera.
- Mudar um preço no admin com a Revisão aberta e confirmar: aviso de total, Revisão relida com o valor novo.

## Suggested Review Order

**A criação numa transação, na ordem que é a estória**

- Porta de entrada: os seis passos, da chave ao esvaziar, numa função só.
  [`pedido.go:121`](../../internal/pedido/pedido.go#L121)

- Passo 1: a chave já usada devolve o Pedido original, ou recusa com 409.
  [`pedido.go:320`](../../internal/pedido/pedido.go#L320)

- Toda recusa antes do INSERT relê a chave: o gêmeo do duplo clique ganha o original.
  [`pedido.go:142`](../../internal/pedido/pedido.go#L142)

- O INSERT do Pedido é a reivindicação da chave; o gêmeo espera no índice único.
  [`pedido.go:189`](../../internal/pedido/pedido.go#L189)

- Reserva tudo ou nada, sob a trava `ORDER BY id` do AD-5.
  [`pedido.go:234`](../../internal/pedido/pedido.go#L234)

- Passo 5: preços relidos com os Produtos travados; subtotal e Frete conferidos separados.
  [`pedido.go:238`](../../internal/pedido/pedido.go#L238)

- O digest cobre só Endereço e total — o Carrinho some no reenvio legítimo.
  [`pedido.go:92`](../../internal/pedido/pedido.go#L92)

**Borda HTTP: cabeçalho, 200 do reenvio e a repetição única**

- Chave obrigatória com forma de uuid; reenvio desfaz a transação e responde 200.
  [`pedido.go:66`](../../api/pedido.go#L66)

- 40P01/40001 repetem uma vez, inclusive no Commit; nada mais repete.
  [`transacao.go:36`](../../api/transacao.go#L36)

- A entrada no checkout da 5.5 passa a usar a mesma repetição.
  [`checkout.go:53`](../../api/checkout.go#L53)

**Banco: congelamento, AD-9 e imutabilidade**

- Colunas novas; linhas do esqueleto preenchidas com o gatilho desligado só ali.
  [`20260921120000_pedido_criacao.sql:35`](../../db/migracoes/20260921120000_pedido_criacao.sql#L35)

- Total é soma das parcelas por CHECK, nunca derivação de leitura.
  [`20260921120000_pedido_criacao.sql:50`](../../db/migracoes/20260921120000_pedido_criacao.sql#L50)

- Chave única por Comprador: a mesma chave noutra conta é livre.
  [`20260921120000_pedido_criacao.sql:60`](../../db/migracoes/20260921120000_pedido_criacao.sql#L60)

- Item de Pedido depois do histórico é recusado — fecha o INSERT tardio da 5.1.
  [`20260921120000_pedido_criacao.sql:74`](../../db/migracoes/20260921120000_pedido_criacao.sql#L74)

- `ON CONFLICT DO NOTHING`: o 23505 nunca chega ao navegador.
  [`consultas.sql:27`](../../internal/pedido/db/consultas.sql#L27)

**Carrinho: só os Itens lidos saem**

- `Esvaziar` apaga exatamente os Itens lidos; linha a menos desfaz tudo.
  [`carrinho.go:372`](../../internal/carrinho/carrinho.go#L372)

- Sentinela comum às duas escritas de fora que perdem a corrida.
  [`carrinho.go:48`](../../internal/carrinho/carrinho.go#L48)

- As quatro recusas novas, todas 409, no ponto único do AD-14.
  [`erro.go:73`](../../internal/plataforma/erro/erro.go#L73)

**Tela: decidir em função pura, executar no componente**

- O que cada resposta da criação significa para a tela.
  [`checkout.ts:415`](../../web/lib/checkout.ts#L415)

- Os efeitos na ordem certa: limpar, avisar, navegar — testáveis sem DOM.
  [`checkout.ts:471`](../../web/lib/checkout.ts#L471)

- Nova tentativa de checkout gira a chave: um 201 perdido não prende a seguinte.
  [`checkout.ts:352`](../../web/lib/checkout.ts#L352)

- O Carrinho chama a rotação ao "Fechar o Pedido".
  [`meu-carrinho.tsx:59`](../../web/app/carrinho/meu-carrinho.tsx#L59)

- Envio único, prazo de 30 s e `aria-disabled` no passo irreversível.
  [`revisao-do-pedido.tsx:193`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L193)

**Testes**

- A matriz de servidor inteira, contra Postgres real, com o duplo clique em goroutines.
  [`pedido_test.go:763`](../../api/pedido_test.go#L763)

- O helper que todo teste antigo passou a usar: Carrinho → Endereço → cotação → confirmação.
  [`pedido_test.go:747`](../../api/pedido_test.go#L747)

- A repetição única, com pool falso: Begin, fn e Commit.
  [`transacao_test.go:51`](../../api/transacao_test.go#L51)

- As recusas da criação no tradutor de erro, inclusive embrulhadas.
  [`erro_test.go:261`](../../internal/plataforma/erro/erro_test.go#L261)

- Os efeitos da confirmação, em ordem, sem navegador.
  [`checkout.test.mjs:589`](../../web/scripts/checkout.test.mjs#L589)
