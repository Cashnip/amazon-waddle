---
title: 'Estórias 4.2 e 4.3 — Adicionar, alterar e esvaziar o Carrinho'
type: 'feature'
created: '2026-09-18'
status: 'done'
baseline_commit: '5f5ff69f9ffd42440bd7b7dd10235d70bd259cc9'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-4-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** a 4.1 deixou o Carrinho sem três coisas: recusar a quantidade acima do Estoque informando o disponível (4.2), e, na 4.3, alterar a quantidade, esvaziar e ver o Carrinho — não há `GET`, tela nem contador na barra.

**Approach:**
- `Adicionar` passa a recusar a soma acima do Estoque disponível com `ESTOQUE_INSUFICIENTE` e o número em `dados`. `PATCH` altera a quantidade (zero remove), `GET /api/v1/carrinho` devolve Itens, unidades e subtotal pelos preços atuais, e `DELETE /api/v1/carrinho/itens` esvazia.
- `/carrinho` mostra as linhas, a edição otimista da quantidade e o `Dialog` de esvaziar. A barra superior ganha o ícone do Carrinho com o contador.

## Boundaries & Constraints

**Always:**
- Posse dentro da consulta (AD-11); `carrinho` não importa `identidade`. Item alheio, inexistente e uuid malformado dão o mesmo 404, byte a byte.
- Visibilidade e preço só por `catalogo` (AD-19). `Disponivel` é conselho e não reserva nada (AD-5): quem decide é `Reservar`, na Épica 5.
- O teto vem de `cfg.CarrinhoUnidadesMax` e vale sem olhar o Estoque. Estoque recusa a **soma resultante do Item**, não só o acréscimo.
- `POST` e `PATCH` gravam `preco_visto_centavos` com o preço atual. `GET` é leitura pura e nunca grava (AD-17).
- Dinheiro em centavos `int64`; `R$` só no navegador. Só a quantidade tem atualização otimista (UX-DR18).

**Ask First:** coluna, rota ou sentinela além dos listados; qualquer emenda ao texto do AD-3.

**Never:**
- A revalidação, o `Alert` de preço mudado e o bloqueio por indisponível (4.4); o texto e a composição da 4.5 (Frete, distância até a isenção).
- `carrinho.Esvaziar` e `ConfirmarPrecoVisto` (Épica 5), `DELETE /api/v1/carrinho` e chamar `Reservar`.
- Carrinho anônimo ou no Redis, regra de negócio no Node, consulta em intervalo, mexer no `Comprar` do esqueleto.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Acima do Estoque | Estoque disponível 3, Carrinho sem o Produto, q=4 | nada grava, nem o Carrinho | 409 `ESTOQUE_INSUFICIENTE`, mensagem "Restam 3 unidades de {nome}.", `dados` {produto_id, disponivel: 3, solicitado: 4} |
| Soma acima do Estoque | Item com 3, disponível 4, q=2 | quantidade segue 3 | o mesmo 409, `disponivel: 4`, `solicitado: 5` |
| Sem Estoque | Produto visível, disponível 0 | nada grava | 409, "{nome} está indisponível." |
| Carrinho não reserva | dois Compradores adicionam 3 de um Produto com 3 | 201 nos dois; disponível continua 3 | — |
| Alterar | Item com 3, `PATCH` q=7 | 200, quantidade 7, `preco_visto_centavos` = preço atual | — |
| Zero remove | `PATCH` q=0 | 204, a linha some | — |
| Alterar recusado | q=-1, 11, 1.5, ausente; ou acima do disponível | nada grava | 400 no campo `quantidade`; o acima do Estoque, o 409 |
| Alterar alheio | Item de outro Comprador, inexistente, malformado, ou de Produto que saiu da visibilidade | a linha não muda | o mesmo 404 |
| Ler | sem Carrinho; com dois Itens; Produto com preço mudado ou desativado | `itens: []`, 0, 0; subtotal = Σ preço atual × quantidade; o desativado sai `visivel: false`, fora do subtotal e dentro de `unidades` | sem Sessão, 401 |
| Esvaziar | `DELETE /carrinho/itens` | 204, todos os Itens do dono somem; o de outro Comprador fica; Carrinho vazio também dá 204 | sem Sessão, 401 |
| Tela | edição recusada; esvaziar; Carrinho vazio; visitante | a quantidade volta e a mensagem do envelope aparece em linha; `Dialog` fecha com `Esc`; "Seu Carrinho está vazio." com uma ação, a Vitrine; `/carrinho` leva ao Login e volta | — |

</frozen-after-approval>

## Code Map

- `internal/carrinho/carrinho.go` -- `Adicionar` (`BuscarProduto` já devolve `EstoqueDisponivel` e `Nome`), `RemoverItem` e `uuidDe` são o molde; a porta ganha `AlterarQuantidade`, `Itens`, `Limpar` e o erro `AcimaDoEstoque`.
- `internal/carrinho/db/consultas.sql` -- `AdicionarItem` é o molde da posse por `USING`; entram `QuantidadeDoProduto` (SUM por dono e Produto, 0 sem linha), `BuscarItem`, `AlterarItem`, `ListarItens` (`ORDER BY item.id`), `LimparItens`.
- `internal/catalogo/catalogo.go` + `db/consultas.sql` -- `Visiveis` e `disponivelDosVisiveis` leem a VIEW `produto_visivel`; entra `Resumos` em lote (nome, preço, imagem, disponível), chaveado pelo id canônico.
- `internal/plataforma/erro/erro.go` -- molde `EscreverEstoqueComprometido`; entra `EscreverEstoqueInsuficiente`, fora do registro porque a mensagem nomeia número e Produto. `erro_test.go` cobre.
- `api/carrinho.go` -- `adicionarAoCarrinho` (a validação da quantidade vira função comum com o mínimo), `removerItemDoCarrinho`; entram `lerCarrinho`, `alterarItemDoCarrinho`, `esvaziarOCarrinho`.
- `api/rotas.go:72` -- três rotas ao lado das duas. Go 1.22+ separa `DELETE …/itens` de `…/itens/{id}`.
- `api/carrinho_test.go` -- `carrinhoDoComprador` e os helpers `postarItem`/`deletarItem`; o Produto de Estoque 3 sai de `corpoDe` com `produtoCom`.
- `api/autorizacao_test.go:67` -- a tabela de `negacaoPorDono` ganha "Item de Carrinho por PATCH", antes do DELETE; a sobrevivência do Item cobre os dois.
- `web/components/casca.tsx` -- a barra; o ícone entra entre a busca e `MenuDaConta`. Molde de Sessão: `menu-da-conta.tsx`.
- `web/app/enderecos/{page,meus-enderecos}.tsx` -- molde da tela: Server Component fino e filho `"use client"` que fala com o Go, 401 → `paraLogin`. `Dialog` de um nível já é usado ali.
- `web/app/produtos/[id]/caixa-de-compra.tsx` -- `enviar()` mostra `json.erro.mensagem`, então a recusa por Estoque já aparece; falta avisar a barra.
- `web/lib/{pedir,preco,destino}.ts`, `web/components/{preco,imagem-do-produto}.tsx` -- reuso. Testes de `lib` rodam por `node --test scripts/*.test.mjs`.
- `_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/.memlog.md` -- onde a leitura do AD-3 fica registrada.

## Tasks & Acceptance

**Execution:**
- [x] `internal/carrinho/db/consultas.sql` + `sqlc generate` -- as cinco consultas acima -- posse no `WHERE`, sem FK cruzando schema.
- [x] `internal/catalogo/db/consultas.sql` + `catalogo.go` -- `Resumos(ctx, bd, ids)` sobre a VIEW -- preço e visibilidade só pela porta de `catalogo`, sem N+1.
- [x] `internal/carrinho/carrinho.go` -- em `Adicionar`, teto da soma e depois Estoque, antes de tocar no Carrinho; `AlterarQuantidade(ctx, bd, itemID, compradorID, q)`: item do dono, `BuscarProduto`, Estoque, `AlterarItem`; `Itens` e `Limpar`. Invisível ou alheio sai `pgx.ErrNoRows`.
- [x] `internal/plataforma/erro/erro.go` + `erro_test.go` -- `EscreverEstoqueInsuficiente` (409, código `ESTOQUE_INSUFICIENTE`), com singular, plural e zero.
- [x] `api/carrinho.go` + `api/rotas.go` -- `GET /api/v1/carrinho`, `PATCH /api/v1/carrinho/itens/{id}` (0 chama `RemoverItem`), `DELETE /api/v1/carrinho/itens`; o `AcimaDoEstoque` vira o 409 do `POST` e do `PATCH`.
- [x] `api/carrinho_test.go` + `api/autorizacao_test.go` -- as linhas de API da matriz, num subteste de `TestSessaoEProduto`; o PATCH em `negacaoPorDono` -- NFR-8, NFR-6.
- [x] `web/lib/carrinho.ts` + `web/scripts/carrinho.test.mjs` -- soma de unidades e subtotal em centavos inteiros para o estado otimista, e o evento `azamon:carrinho` que a barra escuta.
- [x] `web/components/carrinho-na-barra.tsx` + `casca.tsx` -- ícone com contador de unidades. Sem Sessão, o ícone vai ao Login sem número. Atualiza pelo evento, sem consulta em intervalo.
- [x] `web/app/carrinho/{page,meu-carrinho}.tsx` -- linhas com `Input`, `Button` e `Separator`, sem `Card` aninhado; edição otimista de 1 a teto, revertida com a mensagem do envelope; zero e "Remover" esperam o servidor; subtotal; `Dialog` de esvaziar; o Produto que saiu da visibilidade aparece só como "Produto indisponível." com Remover.
- [x] `web/app/produtos/[id]/caixa-de-compra.tsx` -- dispara o evento depois do 201, inclusive na volta do Login.
- [x] `.memlog.md` da arquitetura, por `uv run _bmad/scripts/memlog.py append` -- o AD-3 proíbe rota de esvaziar **para a criação do Pedido**; o esvaziar do Comprador (FR-18) é outro fim, fica sob `/carrinho/itens` e se chama `Limpar`.

**Acceptance Criteria:**
- Dado um Produto com 3 em Estoque, quando um Comprador tenta pôr 4 no Carrinho, então a tela mostra "Restam 3 unidades de {nome}." e nada foi gravado.
- Dado um preço alterado pelo Administrador, quando o Carrinho é aberto, então cada linha e o subtotal usam o preço novo, e `preco_visto_centavos` continua o antigo.
- Dado o Carrinho com Itens, quando o Comprador confirma "Esvaziar" no `Dialog`, então aparece "Seu Carrinho está vazio." e o contador da barra sai a 0, sem recarregar; `Esc` fecha o `Dialog` sem esvaziar.
- Dada uma edição de quantidade aceita, quando o servidor responde, então a tela recarrega o Carrinho pelo `GET`, e o subtotal exibido é o do servidor; o cálculo local só cobre o intervalo da edição otimista.
- Dado o código, quando inspecionado, então `carrinho` não importa `identidade` e nenhuma rota `DELETE /api/v1/carrinho` existe.

## Spec Change Log

## Design Notes

**`Limpar`, e não `Esvaziar`.** `carrinho.Esvaziar(ctx, tx, compradorID, itemIDs)` é a assinatura que `pedido.Criar` vai chamar na 5.x e continua reservada. O esvaziar do Comprador não recebe `tx` nem `itemIDs`; nomes, rota e arquivo de consulta distintos impedem que um seja usado pelo outro.

**Estoque na soma, não no acréscimo.** Um Item com 3 de um Produto com 4 disponíveis recusa +2 porque o Item passaria a pedir 5. O número que a mensagem dá é o disponível, e a tela não reescreve a mensagem (AD-14). A leitura do Estoque e a gravação são duas idas: a divergência é o comportamento projetado do AD-5, e o teto continua atômico na consulta.

**Produto que saiu da visibilidade.** Entra no `GET` com `visivel: false`, sem nome nem preço, para o Comprador poder removê-lo. É o mínimo que a 4.3 exige para não esconder Item nem quebrar o subtotal; o `Alert` que bloqueia é da 4.4.

## Verification

**Commands:**
- `sqlc generate && go vet ./... && go test -count=1 ./...` -- verde, inclusive `TestFronteiraDeModulo` e `TestSessaoEProduto`.
- `cd web && npm test && npm run build` -- verde.

**Manual checks (if no CLI):**
- Passeio a 360 e 1440 px: adicionar, estourar o Estoque, editar a quantidade, remover, esvaziar com `Esc` e com confirmação, e o contador da barra sem recarregar.

## Review Triage Log

Uma lente só (Edge Case Hunter), pela política de revisão do projeto para CRUD de mesma forma. Rodou na própria sessão, sem subagente, e por isso sem o isolamento de contexto que a lente pede. Três achados, todos no `web/`:
- **patch:**
  - duas leituras seguidas do contador podem voltar fora de ordem, e o número velho ganhava;
  - a reversão de uma edição recusada restaurava um retrato que já levava a edição otimista de outra linha em voo; agora reverte e recarrega;
  - um zero que o servidor recusou deixava o "0" digitado no campo; a `key` do campo agora inclui o aviso.
- **defer:** a linha "Tela" da matriz não tem teste automatizado, e a tela não foi aberta num navegador (`deferred-work.md`).
- **reject:** o recarregar que falha depois de um esvaziar bem-sucedido deixa a lista velha com o alerta em cima — janela curta e o alerta a explica.

Uma correção de preparo de teste, e não de expectativa: o subteste da 4.1 usava o Produto semeado, cujo disponível cai a cada Pedido dos subtestes anteriores (chegou a 1) e não comporta mais a soma até 9. Ele agora cria o próprio Produto, com Estoque 20.

## Suggested Review Order

**A recusa por Estoque e o teto (4.2)**

- A soma do Item é conferida contra o teto e depois contra o Estoque, antes de tocar no Carrinho.
  [`carrinho.go:87`](../../internal/carrinho/carrinho.go#L87)

- O erro embrulha o sentinela do Pedido e leva o disponível e o solicitado.
  [`carrinho.go:28`](../../internal/carrinho/carrinho.go#L28)

- O 409 nomeia o Produto e o disponível; zero vira "está indisponível".
  [`erro.go:169`](../../internal/plataforma/erro/erro.go#L169)

**Alterar, ler e esvaziar (4.3)**

- Posse duas vezes, em `BuscarItem` e em `AlterarItem`; invisível e alheio saem o mesmo 404.
  [`carrinho.go:141`](../../internal/carrinho/carrinho.go#L141)

- A leitura é pura, em lote pela VIEW, e o subtotal só soma o que é visível.
  [`carrinho.go:181`](../../internal/carrinho/carrinho.go#L181)

- `Limpar` e `Esvaziar` têm nomes e rotas distintos de propósito (AD-3).
  [`carrinho.go:214`](../../internal/carrinho/carrinho.go#L214)

- `Resumos`: preço e visibilidade só pela porta de `catalogo`, numa ida.
  [`catalogo.go:147`](../../internal/catalogo/catalogo.go#L147)

**Fronteira HTTP**

- A validação da quantidade e a tradução das recusas são comuns ao `POST` e ao `PATCH`.
  [`carrinho.go:56`](../../api/carrinho.go#L56)

- Zero no `PATCH` chama `RemoverItem` e responde 204.
  [`carrinho.go:118`](../../api/carrinho.go#L118)

- As cinco rotas; o esvaziar fica sob `/itens`, e não em `DELETE /carrinho`.
  [`rotas.go:75`](../../api/rotas.go#L75)

**A tela e a barra**

- Edição otimista com reversão e recarga; zero e "Remover" esperam o servidor.
  [`meu-carrinho.tsx:103`](../../web/app/carrinho/meu-carrinho.tsx#L103)

- A `key` refaz o campo não controlado depois de reversão ou de zero recusado.
  [`meu-carrinho.tsx:288`](../../web/app/carrinho/meu-carrinho.tsx#L288)

- O contador só aceita a resposta da leitura mais recente e escuta o evento.
  [`carrinho-na-barra.tsx:18`](../../web/components/carrinho-na-barra.tsx#L18)

- O ícone entra entre a busca e o Menu da conta.
  [`casca.tsx:52`](../../web/components/casca.tsx#L52)

- A Caixa de compra avisa a barra depois do 201, e só nele.
  [`caixa-de-compra.tsx:78`](../../web/app/produtos/[id]/caixa-de-compra.tsx#L78)

**Testes**

- As linhas de API da matriz: recusa, alteração, leitura, invisível e esvaziar.
  [`carrinho_test.go:132`](../../api/carrinho_test.go#L132)

- O NFR-6 ganha o PATCH, e o esvaziar de outro Comprador não alcança o Item do dono.
  [`autorizacao_test.go:69`](../../api/autorizacao_test.go#L69)

- Singular, plural, zero e milhar da mensagem do 409.
  [`erro_test.go`](../../internal/plataforma/erro/erro_test.go)

- A aritmética otimista de centavos e a leitura do campo.
  [`carrinho.test.mjs`](../../web/scripts/carrinho.test.mjs)
