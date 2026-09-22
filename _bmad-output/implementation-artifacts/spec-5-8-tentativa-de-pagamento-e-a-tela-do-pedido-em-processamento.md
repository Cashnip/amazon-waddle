---
title: 'Estória 5.8 — Tentativa de Pagamento e a tela do Pedido em processamento'
type: 'feature'
created: '2026-09-22'
status: 'done'
baseline_commit: '17e4bccb60e713fd44d7d80cb49285b754a7e680'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** a tela `/pedidos/{id}` é o acompanhamento do esqueleto: Status, total e "última atualização", sem relógio de expiração, sem as três saídas e sem ação persistente. O `GET /api/v1/pedidos/{id}` não carrega a tripla do AD-18 (`status`, `tentativas_restantes`, `disponivel` por Item), nem `expira_em`, subtotal, Frete, Endereço ou histórico. E o Provedor Simulado decide pelos centavos em **toda** Tentativa, quando o §7.1 manda aprovar da segunda em diante.

**Approach:** `pedido` monta o Detalhe do Pedido a partir de `pagamento` (tentativas restantes) e `catalogo` (disponível), com `expira_em` derivado do histórico mais o prazo da configuração. A tela vira três superfícies na mesma página — processando (relógio), recusado (motivo no lugar do relógio) e Detalhe (Itens, valores, Endereço) —, com "Ver o Pedido em Meus pedidos" sempre presente. `Simulado.Decidir` passa a receber o número da Tentativa.

## Boundaries & Constraints

**Always:**
- Resposta: os campos atuais mais `subtotal_centavos`, `frete_centavos`, `expira_em` (RFC 3339, `null` fora de `AGUARDANDO_PAGAMENTO`), `tentativas_restantes` (piso 0), `endereco` (`null` no Pedido do esqueleto), `itens[]` `{produto_id, nome, quantidade, preco_praticado_centavos, disponivel}` e `historico[]` `{de, para, ator, motivo, em}`, com `de`/`motivo` `null` quando vazios. **Nenhuma chave nomeia a Reserva.**
- `expira_em` = instante da última transição **para** `AGUARDANDO_PAGAMENTO` + `PagamentoTentativaExpiracao`. Sai do histórico, nunca de relógio em memória (AD-6).
- `tentativas_restantes` = `PagamentoTentativasMax` − Tentativas do Pedido, calculado em `pagamento` (dono do teto, AD-8).
- `disponivel` = `catalogo.Disponivel` do Produto ≥ quantidade do Item.
- O navegador só **exibe** o tempo restante a partir de `expira_em`; consulta a cada 3 s em `AGUARDANDO_PAGAMENTO`, 10 s fora dele, e para em `terminal`. A decisão de superfície, intervalo e motivo mora em funções puras de `web/lib/pedido.ts`.
- A superfície recusada mostra o motivo ("Tempo de pagamento expirado" para `TEMPO_ESGOTADO`; "O pagamento foi recusado." para os demais) e as tentativas restantes.
- `Decidir(total, numero)`: `numero > 1` aprova sempre.
- Glossário literal; verde e laranja fora desta tela.

**Ask First:**
- Mudar a resposta do `POST /api/v1/pedidos` ou do `GET /api/v1/pedidos` (lista), ou criar migração.

**Never:**
- Botão "Tentar pagar de novo" (5.10) ou "Cancelar Pedido" (6.3), nem desabilitados: ação que não existe não aparece.
- Emitir confirmação de recusa, expirar Tentativa ou tocar a varredura (5.9–5.11); linha do tempo na tela (6.2).
- Duração em segundos na resposta; relógio que reinicia ao recarregar.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Recém-criado | Pedido do checkout, prazo P | `expira_em` = nascimento + P; `tentativas_restantes` = teto − 1; subtotal + Frete = total; Endereço e Itens congelados; histórico com 1 linha, `de: null` | — |
| Recusado por expiração | transição com `TEMPO_ESGOTADO` | `expira_em: null`; último `motivo` = `TEMPO_ESGOTADO`; `disponivel: true` (Estoque voltou) | — |
| Produto desativado depois | Item de Produto invisível | `disponivel: false` | — |
| Pedido do esqueleto | sem Endereço | `endereco: null` | — |
| Alheio / inexistente / malformado | — | 404 `NAO_ENCONTRADO`, o mesmo nos três | — |
| Relógio zerado | `AGUARDANDO_PAGAMENTO` após `expira_em` | tela diz que o prazo terminou e segue consultando | — |
| Segunda Tentativa | total `,90` com `numero = 2` | `Decidir` → `APROVADO` | — |

</frozen-after-approval>

## Code Map

- `api/pedido.go:51,130` -- `saidaPedidoDetalhe` e `lerPedido`: ganham os campos novos; `s.cfg` já tem `PagamentoTentativasMax` e `PagamentoTentativaExpiracao` (`internal/plataforma/config.go:113`).
- `internal/pedido/pedido.go:344` -- `Buscar` (dono no `WHERE`) é a base; o Detalhe entra ao lado. `maquina.go:288,303` -- `Transicao` e `Historico`, reusados.
- `internal/pedido/db/consultas.sql:97` -- `BuscarPedidoDoComprador` ganha subtotal, Frete e `endereco_*`; falta `ItensDoPedido` (nome, preço praticado, quantidade). `ItensParaReserva:78` é o molde.
- `internal/pagamento/pagamento.go:57,199` + `db/consultas.sql:46` -- `Decidir`, `EmitirConfirmacoesDevidas` e `TentativasSemConfirmacao` (passa a trazer `numero`); falta a contagem por Pedido.
- `internal/catalogo/catalogo.go:102` -- `Disponivel`: invisível vale 0, sem erro.
- `api/pedido_test.go:326` -- `leituraDoPedido` (404 de alheio/inexistente/malformado, já provado); a matriz nova entra em `api/pedido_detalhe_test.go`; helpers `pedidoPeloCheckout:747`, `pegarPedido:478`, `ultimaTransicao:403`. `api/maquina_test.go` tem como transitar num teste. `api/sessao_test.go:145` -- config de teste, sem os dois limiares.
- `web/app/pedidos/[id]/acompanhamento.tsx` -- a tela a refazer; o `role="status"` que vive desde o primeiro render e o tratamento de 401/404 ficam.
- `web/lib/preco.ts:20` -- `formatarPreco`. `web/scripts/checkout.test.mjs` -- molde do `node --test` sobre `lib/`.

## Tasks & Acceptance

**Execution:**
- [x] `internal/pagamento/db/consultas.sql` + `pagamento.go` + `pagamento_test.go` + `sqlc generate` -- `Decidir(total, numero)`, `numero` em `TentativasSemConfirmacao`, `TentativasRestantes(ctx, bd, pedidoID, teto)`; restaurar os `gerado/` alheios.
- [x] `internal/pedido/db/consultas.sql` + `pedido.go` -- `Detalhar(ctx, bd, pedidoID, compradorID, prazo, teto) (Detalhe, error)` e a pura `ExpiraEm(status, historico, prazo)`; `internal/pedido/pedido_test.go` testa a pura.
- [x] `api/pedido.go` + `api/sessao_test.go` + `api/pedido_detalhe_test.go` (novo) -- `lerPedido` usa `Detalhar`; a matriz de servidor inteira, incluindo a varredura das chaves por "reserva".
- [x] `web/lib/pedido.ts` (novo) + `web/scripts/pedido.test.mjs` -- `superficieDoPedido`, `intervaloDaConsulta`, `tempoRestante`, `motivoDaRecusa`.
- [x] `web/app/pedidos/[id]/acompanhamento.tsx` -- as três superfícies, o relógio de tique local sobre `expira_em`, e o link persistente, também no erro e no carregamento.
- [x] `deferred-work.md`, `sprint-status.yaml`, `addendum.md` §10 + memlog -- fechar a entrada "saidaPedido não expõe…"; registrar a decisão da resposta e de `disponivel`; 5-8 para `review`.

**Acceptance Criteria:**
- Given um Pedido recém-criado, when a tela abre e é recarregada, then o tempo restante continua do mesmo ponto, porque vem de `expira_em`.
- Given `AGUARDANDO_PAGAMENTO` que vira `PAGO`, when a consulta de 3 s o traz, then o relógio some e o Detalhe aparece na mesma página, sem navegação, e a região de Status anuncia.
- Given a tela em qualquer superfície, carregando ou com erro, when o Comprador aciona "Ver o Pedido em Meus pedidos", then ele chega a `/pedidos`.

## Design Notes

`disponivel` responde "o Estoque disponível **de agora** cobre este Item?". Enquanto a Reserva do próprio Pedido está ativa, ela conta contra ele — só `PAGAMENTO_RECUSADO` (Reserva já liberada) o consome para derivar ação, e é por isso que descontar a própria Reserva seria nomeá-la para ninguém.

O histórico entra na resposta agora porque é onde o AD-18 põe o `motivo`; a 6.2 o desenha como linha do tempo, sem rota nova.

Relógio: `setInterval` de 1 s só re-renderiza `tempoRestante(expira_em, Date.now())`. Desvio entre o relógio do navegador e o do servidor fica fora: mesma máquina na demonstração.

Quem implementa não faz `git commit`, `git add` nem `push`.

## Verification

**Commands:**
- `sqlc generate && go vet ./... && go test -count=1 ./...` -- expected: verde, na imagem `azamon-dev` (HANDOFF); só `internal/pedido` e `internal/pagamento` em `gerado/`.
- `cd web && npm test && docker build ./web` -- expected: verde.
- `grep -rniE 'json:"[a-z_]*reserv' api/pedido.go; grep -rni reserv web/app/pedidos web/lib/pedido.ts` -- expected: vazio (comentário de Go não conta). A guarda de verdade é `chavesDe` em `api/pedido_detalhe_test.go`.

**Manual checks (if no CLI):**
- A 360 e 1440 px, só pelo teclado: checkout de total `,00` → relógio de 60 s → `PAGO` em ~5 s, virando Detalhe; recarregar no meio mantém o tempo; total `,95` deixa o relógio chegar a zero com a frase de prazo encerrado.

## Review Triage Log

Camada única, como combinado para esta estória: Blind Hunter, sem contexto, sobre o diff inteiro desde `17e4bcc`. Vinte achados; nenhum de intenção nem de spec, então sem volta ao plano.

- **patch** O `deferred-work.md` é *append-only* e a entrada da 5.6 tinha sido reescrita no lugar — revertido; o fechamento virou entrada nova.
- **patch** O `disponivel: true` da expiração não tinha como falhar com Estoque 3, e nada provava a regra deliberada (a Reserva do próprio Pedido conta contra ele). O Produto do teste passou a Estoque 1: recém-criado lê `false`, recusado lê `true`.
- **patch** `Decidir(total, numero)` só tinha teste de unidade. Entrou o caso de emissão: `,95` na Tentativa 1 não emite, na 2 emite `APROVADO`.
- **patch** Tentativas inventadas pelo teste ficavam no banco, e a emissão derivada as pegaria num teste posterior. Apagadas em `t.Cleanup`.
- **patch** A frase do relógio zerado prometia "atualizado em instantes", o que nada cumpre antes da 5.11. Agora diz só que o prazo terminou sem a confirmação chegar. A superfície recusada inalcançável no sistema rodando foi para o `deferred-work.md`.
- **patch** O motivo e o prazo encerrado não eram anunciados. O motivo entrou na região `role="status"`, e a frase do prazo ganhou uma região própria, presente desde a montagem.
- **patch** `<time dateTime>` carregava o instante sobre um texto de duração. Passou a `PT{n}S`.
- **patch** Cópia local de `Preco` no lugar de `components/preco.tsx` — trocada pelo compartilhado.
- **patch** A superfície recusada escondia o que o Pedido contém, e a EXPERIENCE põe "Pagamento recusado" e "Tentativa expirada" no Detalhe. O Detalhe passou a aparecer também embaixo dela.
- **patch** A "Última atualização" tinha sumido com `atualizado_em` ainda na resposta. Voltou.
- **patch** O teste "recarregar não recomeça" comparava a mesma chamada duas vezes, uma tautologia. Removido.
- **patch** Comentário velho de `primeiraTentativa`; a Verification com um grep que casava comentário de Go; o Code Map apontando `api/pedido_test.go`. Os três corrigidos.
- **patch** `entradaEndereco` servia de saída, e uma mudança na entrada de "Meus endereços" mudaria esta resposta. Ganhou tipo próprio, `saidaEnderecoCongelado`.
- **defer** A superfície recusada só é alcançável pelo teste até a 5.10/5.11 — entrada nova no `deferred-work.md`.
- **reject** Status em três lugares divergindo: era o meio do workflow; a sincronização a `review` é deste passo.
- **reject** `expira_em` pelo histórico e a emissão por `tentativa_pagamento.criada_em`: as duas linhas nascem na mesma transação, e o `now()` do Postgres é o instante da transação — é o mesmo instante.
- **reject** Prazo não congelado na Tentativa: a spec e o AD-13 põem o prazo na Config, e trocar de modo com Pedido aberto não é caso da demonstração.
- **reject** A tripla não é lida no front: quem a lê é a ação da 5.10, e botão para ação inexistente é `Never` nesta spec.
- **reject** Falta `UNIQUE (pedido_id, produto_id)` em `item_pedido`: o Carrinho já tem `UNIQUE (carrinho_id, produto_id)`, e o Pedido nasce dele. Virou comentário em `Detalhar`.

## Suggested Review Order

**A leitura do Pedido: uma chamada, tudo que a tela deriva (AD-18)**

- Porta de entrada: Detalhe montado de `pedido`, `pagamento` e `catalogo`, com o dono no WHERE.
  [`pedido.go:404`](../../internal/pedido/pedido.go#L404)

- O prazo sai do histórico: última entrada em AGUARDANDO_PAGAMENTO mais a Config.
  [`pedido.go:384`](../../internal/pedido/pedido.go#L384)

- `disponivel` contra o Estoque de agora; a Reserva própria conta de propósito.
  [`pedido.go:463`](../../internal/pedido/pedido.go#L463)

- Cinco consultas no mesmo instante: transação só leitura em REPEATABLE READ.
  [`pedido.go:249`](../../api/pedido.go#L249)

- A forma JSON: nulos explícitos, listas nunca `null`, nenhuma chave de Reserva.
  [`pedido.go:118`](../../api/pedido.go#L118)

- Endereço congelado com tipo próprio, desacoplado da entrada de "Meus endereços".
  [`pedido.go:74`](../../api/pedido.go#L74)

**Pagamento: teto e primeira Tentativa (AD-8)**

- A faixa dos centavos vale só na primeira Tentativa; da segunda, aprova.
  [`pagamento.go:61`](../../internal/pagamento/pagamento.go#L61)

- O teto e a conta são de `pagamento`, com piso zero.
  [`pagamento.go:162`](../../internal/pagamento/pagamento.go#L162)

- O número da Tentativa atravessa a emissão derivada.
  [`pagamento.go:233`](../../internal/pagamento/pagamento.go#L233)

**Tela: três superfícies, nenhuma promessa falsa**

- Ler a resposta é função pura: superfície, ritmo, relógio, motivo.
  [`pedido.ts:59`](../../web/lib/pedido.ts#L59)

- Relógio que só subtrai o agora de `expira_em`; zerado, diz e não promete.
  [`acompanhamento.tsx:64`](../../web/app/pedidos/%5Bid%5D/acompanhamento.tsx#L64)

- Status e motivo na região viva que existe desde o primeiro render.
  [`acompanhamento.tsx:246`](../../web/app/pedidos/%5Bid%5D/acompanhamento.tsx#L246)

- O link persistente, também carregando e com erro.
  [`acompanhamento.tsx:293`](../../web/app/pedidos/%5Bid%5D/acompanhamento.tsx#L293)

- O Detalhe no lugar, fora do relógio: Itens, valores, Endereço.
  [`acompanhamento.tsx:301`](../../web/app/pedidos/%5Bid%5D/acompanhamento.tsx#L301)

**Testes e periferia**

- A matriz de servidor contra Postgres, com Estoque 1 para o `disponivel` ter dente.
  [`pedido_detalhe_test.go:35`](../../api/pedido_detalhe_test.go#L35)

- A emissão real respeita o número da Tentativa.
  [`pedido_detalhe_test.go:233`](../../api/pedido_detalhe_test.go#L233)

- O teste da 3.1 continua exigindo o congelado idêntico; só o derivado muda.
  [`vendedor_test.go:137`](../../api/vendedor_test.go#L137)

- `ExpiraEm` pela última entrada, não pela primeira.
  [`pedido_test.go:62`](../../internal/pedido/pedido_test.go#L62)

- Da segunda Tentativa em diante, aprova em toda faixa.
  [`pagamento_test.go:33`](../../internal/pagamento/pagamento_test.go#L33)

- As leituras puras da tela.
  [`pedido.test.mjs:17`](../../web/scripts/pedido.test.mjs#L17)

- As consultas novas: Endereço congelado, Itens, contagem e `numero`.
  [`consultas.sql:100`](../../internal/pedido/db/consultas.sql#L100)

- O Endereço por extenso passa a aceitar a cópia sem id.
  [`formulario-de-endereco.tsx:225`](../../web/components/formulario-de-endereco.tsx#L225)
