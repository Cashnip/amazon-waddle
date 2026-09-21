---
title: 'Estória 5.5 — Entrada no checkout: revalidação e confirmação do preço visto'
type: 'feature'
created: '2026-09-20'
status: 'done'
baseline_commit: '9de32e51f087fa5cec7bd83420adbe57953a49d4'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** a segunda metade da FR-19 não existe. Quem digita `/checkout/endereco` ou `/checkout/revisao` passa por cima do `podeAvancar` do botão do Carrinho e chega ao passo irreversível com Produto indisponível ou com preço que não viu — um Carrinho só de Produtos invisíveis nem sequer é vazio, e hoje chega à Revisão. E `carrinho.ConfirmarPrecoVisto` não existe, então a ciência da mudança de preço nunca é persistida: o mesmo aviso volta em toda abertura, para sempre.

**Approach:** uma rota de entrada no checkout, de `pedido`, que numa transação só revalida o Carrinho, **reporta** o que mudou e **só então** chama `carrinho.ConfirmarPrecoVisto` com exatamente os preços que acabou de reportar. O passo Endereço abre por ela e ganha os dois `Alert` do Carrinho — o que bloqueia e o que exige confirmação —, com o Continuar travado até resolverem. A Revisão não repete a escrita: desvia, por leitura pura, ao Carrinho quando há bloqueio e ao passo Endereço quando um preço mudou depois da entrada.

## Boundaries & Constraints

**Always:**
- A ordem do AD-17 é a razão de ser da estória: ler → montar a resposta → `ConfirmarPrecoVisto` → `Commit` → escrever o JSON. A resposta só sai depois do `Commit`, e a confirmação grava **os preços reportados**, nunca uma releitura — senão a mudança de preço some antes de aparecer.
- `ConfirmarPrecoVisto` é a única função que persiste a **ciência** de uma mudança, e **só `pedido` a chama**. Confirma só as linhas visíveis reportadas com `preco_mudou`; a invisível não tem preço a confirmar. Lista vazia é no-op com `nil`, nunca erro.
- Quem decide bloqueio e mudança continua sendo o Go (AD-10), pela regra que já existe em `carrinho` — a 5.5 a reaproveita, não a reescreve. Produto desativado e de Vendedor desativado saem iguais (FR-12).
- Uma transação por caso de uso, aberta e fechada em `api/` (AD-4). O dono vem da Sessão e entra no `WHERE` (AD-11, NFR-6).
- Confirmar o preço **não** desbloqueia: as duas listas são independentes, como na 4.4.
- Glossário literal (UX-DR16): Produto, Carrinho, Pedido, Frete, Endereço. Nem verde nem laranja nesta tela — o laranja é o Confirmar Pedido da Revisão (UX-DR7). Piso AA, pelo teclado, de 360 a 1440 px.

**Ask First:**
- Mudar o contrato JSON de `/api/v1/carrinho`, `/api/v1/frete` ou `/api/v1/enderecos`, ou a forma da linha do Carrinho.
- Acrescentar coluna, migração ou sentinela de erro além dos listados.
- Qualquer escrita nova no Carrinho a partir de uma leitura.

**Never:**
- Criar o Pedido, congelar Frete, `TOTAL_DIVERGENTE`, enviar a `Idempotency-Key` e chamar `limparCheckout` — tudo 5.6. Esta estória não toca `POST /api/v1/pedidos`.
- Segunda escrita de `preco_visto_centavos` na Revisão: ela só desvia, por leitura.
- Calcular Frete, somar total ou decidir bloqueio no Node; consultar em intervalo; rede externa (NFR-15).
- Distinguir Produto desativado de Vendedor desativado na tela, ou nomear "Reserva" ao Comprador.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Entrada limpa | nada mudou | linhas com `preco_mudou: false` e `bloqueio: ""`; Continuar habilitado | — |
| Preço mudou | visto 5000, atual 6000 | a resposta traz `preco_visto_centavos: 5000` e `preco_mudou: true`; **depois**, no banco, o visto é 6000 | — |
| Entrar de novo | logo após a anterior | `preco_mudou: false`, nenhum `Alert`; e `GET /api/v1/carrinho` concorda | — |
| Acima do Estoque | Item com 4, disponível 2 | `bloqueio: acima_do_estoque`; Continuar travado, com o caminho de volta ao Carrinho | — |
| Só Produtos invisíveis | Carrinho não vazio, todas as linhas invisíveis | `bloqueio: indisponivel` em todas; o passo trava e a Revisão desvia ao Carrinho | — |
| Invisível com preço antigo | linha invisível | o visto dela **não** é tocado | — |
| Preço mudou numa linha bloqueada | as duas coisas na mesma linha | os dois avisos; o preço é confirmado, o bloqueio fica | — |
| Cruzar o limiar | subtotal cai de R$ 320,00 para R$ 280,00 | a cotação seguinte deixa de ser Grátis e cobra a faixa; o total acompanha | — |
| Carrinho vazio | sem Itens | o passo volta a `/carrinho` | — |
| Outro dono | Carrinho de outro Comprador | intacto: nenhum visto dele muda | — |
| Sem Sessão | sem cookie | 401, e a tela vai ao Login com `destino` | — |
| Falha de rede | `fetch` lança | `FALHA_DE_REDE` em `Alert` | — |

</frozen-after-approval>

## Code Map

- `internal/carrinho/db/consultas.sql:52` -- `ListarItens` é a leitura pura que já traz `preco_visto_centavos`; não muda. Entra `ConfirmarPrecoVisto` como `:execrows`, com `unnest(@ids::uuid[], @precos::bigint[])` e o dono no `WHERE` (molde: `AlterarItem:44`). Depois, `sqlc generate`.
- `internal/carrinho/carrinho.go:83` -- `bloqueioDe` e `Itens:210` são a regra da FR-19, já prontas: **reusar, não duplicar**. Entram `PrecoVisto{ItemID, PrecoCentavos}` e `ConfirmarPrecoVisto`, ao lado de `uuidDe:280`. `Adicionar:112` e `AlterarQuantidade:165` continuam gravando o visto da alteração — ver Design Notes.
- `internal/pedido/frete.go:100` -- `CotarFrete` é o molde exato do arquivo novo: recebe `gerado.DBTX` de `pedido` e o passa a `carrinho.Itens`, que aceita por tipagem estrutural. O novo `internal/pedido/checkout.go` faz o mesmo.
- `api/pedido.go:45` -- `criarPedido` é o molde da transação (AD-4): `Begin`, `defer tx.Rollback(context.WithoutCancel(...))`, chamada, `Commit` com `WithoutCancel`, e **só então** `escreverJSON`.
- `api/carrinho.go:38` -- `saidaLinhaCarrinho` e `saidaCarrinho:56`; `lerCarrinho:213` monta as duas. A rota nova devolve **o mesmo envelope**: extrair o montador e reaproveitá-lo dos dois lados.
- `api/rotas.go:78` -- onde as rotas do Carrinho moram; a nova entra junto de `GET /api/v1/frete:71`. `semCache` em `api/sessao.go:166`; o cookie é `SameSite=Lax` (`sessao.go:148`), que é o que dispensa corpo nesta rota.
- `api/carrinho_test.go:403` -- `carrinhoRevalidado` é a bancada pronta: `precoVistoNoBanco:404`, `novoProduto`, `mudarPreco`, `ajustarEstoque`, `linhas` e `quer:468`. O subteste novo entra em `api/sessao_test.go:502`, ao lado dele.
- `api/frete_test.go:21` -- `freteDoCarrinho`, `querCotacao:140` e `pegarFrete:161`: o molde do caso do limiar. O limiar é `s.cfg.FreteIsencaoCentavos`.
- `web/lib/carrinho.ts:108` -- `revalidacaoDe`, `comPrecosConfirmados:118` e `textoDoBloqueio:131` já fazem tudo que a tela precisa, sobre o tipo `Carrinho:29`. Nada de lógica nova aqui.
- `web/app/carrinho/meu-carrinho.tsx:204` -- os dois `Alert` a copiar (bloqueio destrutivo e preço com confirmação). No passo Endereço, porém, o bloqueio **não** traz Ajustar/Remover: traz "Voltar ao Carrinho".
- `web/app/checkout/endereco/escolha-de-endereco.tsx:71` -- o `useEffect` de abertura, hoje `Promise.all([GET carrinho, GET enderecos])`: o primeiro vira a rota nova. `continuar():101` é onde o `podeAvancar` passa a valer.
- `web/app/checkout/revisao/revisao-do-pedido.tsx:105` -- a guarda de Carrinho vazio, onde entra o desvio. A Revisão já lê `GET /api/v1/carrinho`, que traz `bloqueio` e `preco_mudou`: nenhuma leitura nova.
- `web/lib/checkout.ts:76` -- `rotaDoFrete` e vizinhos; entra `desvioDaRevisao`, pura. `web/scripts/checkout.test.mjs:22` é o molde do `node --test`.
- `_bmad-output/implementation-artifacts/deferred-work.md:266` e `:297` -- as duas entradas que esta estória fecha; marcar RESOLVIDO na linha, sem apagar.

## Tasks & Acceptance

**Execution:**
- [x] `internal/carrinho/db/consultas.sql` + `sqlc generate` -- a consulta `ConfirmarPrecoVisto`, com o dono no `WHERE` -- restaurar com `git checkout` os `gerado/` de outros módulos que o LF reescrever.
- [x] `internal/carrinho/carrinho.go` -- `PrecoVisto` e `ConfirmarPrecoVisto(ctx, bd, compradorID, vistos)`: ordena por `ItemID` antes de gravar (ordem determinística, AD-4), lista vazia é no-op -- é a única porta da ciência (AD-17).
- [x] `internal/pedido/checkout.go` (novo) -- `EntrarNoCheckout(ctx, tx, compradorID) (carrinho.Conteudo, error)`: `carrinho.Itens`, monta o relatório, e **depois** confirma as linhas visíveis com `PrecoMudou` -- a ordem é a invariante.
- [x] `api/checkout.go` (novo) + `api/rotas.go` -- `POST /api/v1/checkout/entrada`, sem corpo, com a transação do molde de `criarPedido` e o envelope de `saidaCarrinho` extraído de `lerCarrinho` -- resposta só depois do `Commit`.
- [x] `api/checkout_test.go` (novo) + `api/sessao_test.go` -- as linhas de servidor da matriz, num subteste novo: reporta e grava na mesma chamada, a segunda entrada silencia, o invisível não é tocado, a linha bloqueada tem o preço confirmado, o Carrinho alheio fica intacto, e o cruzamento do limiar muda a cotação -- NFR-8.
- [x] `web/lib/checkout.ts` + `web/scripts/checkout.test.mjs` -- `desvioDaRevisao(carrinho)`: `/carrinho` com bloqueio, `/checkout/endereco` com preço mudado, senão `null` -- teste de `lib`.
- [x] `web/app/checkout/endereco/escolha-de-endereco.tsx` -- abrir pela rota nova, os dois `Alert` por `revalidacaoDe`, e o Continuar travado enquanto houver bloqueio ou preço por confirmar.
- [x] `web/app/checkout/revisao/revisao-do-pedido.tsx` -- aplicar `desvioDaRevisao` junto da guarda de Carrinho vazio, antes de qualquer pintura.
- [x] `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` §10 + `uv run _bmad/scripts/memlog.py append` -- a decisão: a entrada é rota própria de `pedido`, devolve o envelope do Carrinho, e a ciência é dos preços reportados.
- [x] `_bmad-output/implementation-artifacts/deferred-work.md` + `sprint-status.yaml` -- marcar RESOLVIDO nas duas entradas que esta estória fecha; 5-5 para `review`.

**Acceptance Criteria:**
- Given um preço alterado pelo Administrador depois da última alteração do Item, when o Comprador entra no checkout, then a resposta mostra o preço anterior e o atual **e** o banco passa a guardar o atual — nessa ordem, numa transação só; e uma segunda entrada não avisa mais nada.
- Given um Carrinho com qualquer linha bloqueada, when o Comprador entra no checkout ou digita a URL da Revisão, then não há caminho até o Confirmar Pedido: o passo Endereço trava e a Revisão devolve ao Carrinho.
- Given o código, when inspecionado, then `ConfirmarPrecoVisto` tem um chamador só — `internal/pedido` — e nenhuma leitura de Carrinho grava.

## Spec Change Log

- 2026-09-20 (revisão) -- a alegação de ordem de travas do Execution item de `carrinho.go` está **retratada**. A tarefa pedia "ordena por `ItemID` antes de gravar (ordem determinística, AD-4)", e o código a implementou dizendo que isso fazia duas entradas simultâneas tomarem as travas na mesma ordem: não faz. Num `UPDATE ... FROM (unnest ...)` quem ordena o bloqueio é o plano do executor; a garantia do AD-5 vem de `SELECT ... ORDER BY id FOR UPDATE`, que esta confirmação não usa e não precisa — ela não decide Estoque. A ordenação **fica**, pelo que a tarefa de fato pede: determinismo do comando. O resíduo de deadlock foi para o `deferred-work.md`.
- 2026-09-20 (revisão) -- `ConfirmarPrecoVisto` passou a conferir as linhas afetadas do `:execrows` contra o número de vistos, e a recusar a escrita parcial. Sem sentinela novo, que era `Ask First`: a recusa sai no 500 genérico, e nomeá-la ficou registrado como trabalho adiado.
- 2026-09-20 (revisão) -- a decisão de abertura das duas telas saiu dos `useEffect` para funções puras em `web/lib/checkout.ts` (`aberturaDoPassoEndereco`, `aberturaDaRevisao`, `desvioDaAberturaDaRevisao`, `podeContinuar`, `razoesDoContinuar`, `ENTRADA_NO_CHECKOUT`), fixadas em `web/scripts/checkout.test.mjs`. Duas correções de comportamento vieram junto: o relatório da entrada é aplicado antes da guarda da lista de Endereços, e a entrada dispara uma vez só por tentativa (`useRef`, contra a montagem dupla do StrictMode). O Continuar passou a `aria-disabled`, para que a razão seja alcançável pelo teclado.
- 2026-09-20 -- a consulta `ConfirmarPrecoVisto` não usa `unnest(@ids::uuid[], @precos::bigint[])`, como o Code Map previa: o analisador do sqlc v1.31.1 recusa a forma de dois argumentos ("function unnest(unknown, unknown) does not exist"), que no Postgres é construção de `ROWS FROM` e não função comum. O par sai de dois `unnest ... WITH ORDINALITY` juntados pela posição — mesma semântica, e o gerador aceita. Nada mais da consulta muda: `:execrows`, o dono no `WHERE`, o molde de `AlterarItem`.

## Design Notes

**A ordem é a estória inteira.** Confirmar antes de reportar, ou reportar antes de o `Commit` passar, são os dois jeitos de a mudança de preço sumir sem ninguém ver — exatamente o que o AD-17 proíbe. Por isso a resposta é montada com o que se leu, a confirmação grava **esses** valores (e não uma releitura, que já pode ser outro preço), e o JSON só sai depois do `Commit`: se o `Commit` falhar, nada foi reportado e o aviso volta na próxima entrada. O sentido oposto — ciência gravada e resposta perdida — não existe.

**`Adicionar` e `AlterarQuantidade` continuam gravando o visto, e isso não contradiz o AD-17:** elas gravam o preço de uma alteração que o Comprador acabou de fazer, que já é ciência. O que só `ConfirmarPrecoVisto` faz é persistir a ciência de uma mudança que o Comprador **não** provocou.

**A Revisão desvia, não escreve.** Ela já lê `GET /api/v1/carrinho`, que traz `bloqueio` e `preco_mudou`; um preço que mudou depois da entrada a manda de volta ao passo Endereço, que é a rota que reporta e confirma. Duas escritas seriam dois módulos gravando o mesmo campo.

**O envelope é o mesmo do Carrinho de propósito:** a tela reaproveita `revalidacaoDe` e `textoDoBloqueio` sem uma linha de regra nova, e os dois `Alert` ficam idênticos aos que o Comprador acabou de ver. Sem corpo na requisição: o cookie é `SameSite=Lax`, então um `<form>` cross-site não o envia, e o precedente é `DELETE /api/v1/carrinho/itens`.

**O Frete já é recalculado** — `CotarFrete` lê `carrinho.Itens` a cada abertura da Revisão, sempre pelos preços de agora. O que falta é a **prova** de que ele cruza o limiar de isenção nos dois sentidos; por isso o caso entra no teste, e não um campo novo na resposta.

Quem implementa não faz `git commit`, `git add` nem `push`.

## Verification

**Commands:**
- `sqlc generate && go vet ./... && go test -count=1 ./...` -- expected: verde, inclusive `TestFronteiraDeModulo` e `TestSessaoEProduto` (imagem `azamon-dev`, ver HANDOFF).
- `cd web && npm test && docker build ./web` -- expected: verde (roda `verificar-offline`).
- `grep -rn "ConfirmarPrecoVisto" --include=*.go . | sed 's/:.*//' | sort | uniq -c` -- expected: quatro arquivos, e **uma** chamada só — `internal/carrinho/carrinho.go` (a função), `internal/carrinho/db/gerado/consultas.sql.go` (o gerado pelo sqlc), `internal/pedido/checkout.go` (**o único chamador**, 1 ocorrência) e `api/checkout_test.go` (menção em comentário). O que a terceira AC exige é o `1` em `internal/pedido` e nenhum outro chamador; o gerado e o comentário casam com o grep e não são chamadas.

**Manual checks (if no CLI):**
- A 360 e a 1440 px, só pelo teclado: mudar o preço no admin, abrir o Carrinho, confirmar, "Fechar o Pedido" — o aviso reaparece no passo Endereço, com o Continuar travado até confirmar; recarregar e ver que ele não volta.
- Desativar um Produto do Carrinho e digitar `/checkout/revisao`: volta ao Carrinho sem piscar a Revisão.
- Encher o Carrinho acima de R$ 299,00, ver "Grátis" na Revisão, baixar um preço no admin, reentrar no checkout e ver o Frete cobrado de novo, com o total acompanhando.

## Suggested Review Order

**A ordem que é a estória: reportar, e só então confirmar**

- A porta de entrada. Lê, monta o relatório, confirma a partir dele — nunca de uma releitura.
  [`checkout.go:37`](../../internal/pedido/checkout.go#L37)

- Por que a ordem é a função inteira, e o que cada inversão apagaria.
  [`checkout.go:18`](../../internal/pedido/checkout.go#L18)

- A lista de confirmação sai do relatório que vai na resposta, e não de uma segunda leitura.
  [`checkout.go:44`](../../internal/pedido/checkout.go#L44)

- O JSON sai depois do `Commit`: antes, prometeria uma ciência que o banco ainda podia recusar.
  [`checkout.go:68`](../../api/checkout.go#L68)

- Rota que escreve e não lê corpo: `SameSite=Lax` é o que a dispensa.
  [`checkout.go:28`](../../api/checkout.go#L28)

- A rota nova, ao lado da cotação do Frete, que é do mesmo módulo.
  [`rotas.go:77`](../../api/rotas.go#L77)

**A única porta da ciência, em `carrinho`**

- A função que só `pedido` chama. Grava os preços recebidos, nunca uma releitura.
  [`carrinho.go:319`](../../internal/carrinho/carrinho.go#L319)

- Escrita parcial não passa calada: linhas afetadas conferidas contra o que se pediu.
  [`carrinho.go:345`](../../internal/carrinho/carrinho.go#L345)

- O par de vetores anda pela ordinalidade, e a posse entra no `WHERE` (AD-11).
  [`consultas.sql:92`](../../internal/carrinho/db/consultas.sql#L92)

- Por que não é o `unnest` de dois argumentos: o analisador do sqlc o recusa.
  [`consultas.sql:90`](../../internal/carrinho/db/consultas.sql#L90)

- A ordenação é determinismo do comando — a ordem das travas é do plano, não do vetor.
  [`carrinho.go:328`](../../internal/carrinho/carrinho.go#L328)

**O envelope reaproveitado, que faz a tela não ganhar regra nova**

- Extraído de `lerCarrinho` para as duas rotas devolverem exatamente a mesma forma.
  [`carrinho.go:183`](../../api/carrinho.go#L183)

**As telas: decidir em função pura, executar no efeito**

- A abertura do passo Endereço, inteira, testável sem DOM.
  [`checkout.ts:172`](../../web/lib/checkout.ts#L172)

- O relatório é aplicado antes do erro e antes do desvio — nunca é descartado.
  [`escolha-de-endereco.tsx:132`](../../web/app/checkout/endereco/escolha-de-endereco.tsx#L132)

- Uma entrada por tentativa: o duplo disparo do StrictMode silenciaria a primeira.
  [`escolha-de-endereco.tsx:121`](../../web/app/checkout/endereco/escolha-de-endereco.tsx#L121)

- A rota da abertura é dado, para o teste poder prendê-la.
  [`checkout.ts:130`](../../web/lib/checkout.ts#L130)

- A Revisão desvia por leitura pura: bloqueio ao Carrinho, preço mudado ao Endereço.
  [`checkout.ts:120`](../../web/lib/checkout.ts#L120)

- A decisão da Revisão roda antes de qualquer pintura.
  [`revisao-do-pedido.tsx:108`](../../web/app/checkout/revisao/revisao-do-pedido.tsx#L108)

- `aria-disabled`, e não `disabled`: botão nativo desabilitado não é alcançável por leitor de tela.
  [`escolha-de-endereco.tsx:315`](../../web/app/checkout/endereco/escolha-de-endereco.tsx#L315)

- Os dois `Alert` do Carrinho, sobre a mesma `revalidacaoDe` — aqui o bloqueio leva de volta.
  [`escolha-de-endereco.tsx:260`](../../web/app/checkout/endereco/escolha-de-endereco.tsx#L260)

**Testes**

- A linha central: a resposta traz o "de 5000 para 6000" e o banco já guarda 6000.
  [`checkout_test.go:117`](../../api/checkout_test.go#L117)

- A matriz de servidor inteira, contra PostgreSQL real.
  [`checkout_test.go:18`](../../api/checkout_test.go#L18)

- A abertura pede `POST` na rota nova — e explicitamente não o Carrinho.
  [`checkout.test.mjs:373`](../../web/scripts/checkout.test.mjs#L373)

- A lista de Endereços falha depois do `Commit`, e o relatório sobrevive.
  [`checkout.test.mjs:413`](../../web/scripts/checkout.test.mjs#L413)

- A trava do avanço nas quatro combinações.
  [`checkout.test.mjs:437`](../../web/scripts/checkout.test.mjs#L437)
