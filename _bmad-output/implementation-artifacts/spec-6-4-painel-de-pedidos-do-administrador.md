---
title: 'Estória 6.4 — Painel de Pedidos do Administrador'
type: 'feature'
created: '2026-09-23'
status: 'in-review'
baseline_commit: '4b12ac36188e5d659c506a7adc1abe6c638335d7'
route: 'dispatch'
review_loop_iteration: 0
context: ['{project-root}/_bmad-output/implementation-artifacts/epic-6-context.md']
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** As três transições do Administrador (`PAGO → SEPARANDO`, `SEPARANDO → ENVIADO`, `ENVIADO → ENTREGUE`) estão na tabela do AD-3 desde a 5.1, e `TransicaoInvalida.Permitidas` já viaja em `dados.permitidas` — mas nenhuma rota as alcança. Hoje o Administrador só opera a loja tocando no banco, e não tem nenhuma tela que veja Pedido de outro dono.

**Approach:** Área administrativa de Pedidos: `GET /api/v1/admin/pedidos` (filtro por Status, ordenação por data, envelope do AD-18), `GET /api/v1/admin/pedidos/{id}` (Comprador, Vendedores congelados dos Itens, valores, Endereço e histórico) e `POST /api/v1/admin/pedidos/{id}/transicoes`, que recebe `de` e `para` e chama `Transicionar(…, ADMINISTRADOR)`. As respostas carregam `permitidas` pronto, tirado de `Permitidas(status, ADMINISTRADOR)` — a tela nunca redeclara a tabela (o molde de `pode_cancelar`, 6.3).

## Boundaries & Constraints

**Always:**
- Toda mudança pelo `Transicionar`; o efeito sobre o Estoque é dele (`ENVIADO` consolida). Nenhum `UPDATE` de Status novo.
- `de` vem do que a tela leu: é ele que faz o CAS distinguir "já avançou" de "transição inválida". Lido no servidor, o Pedido que andou sairia como transição inválida — causa falsa.
- Toda rota sob `/api/v1/admin/` herda a guarda do prefixo; nada de `if` por handler.
- Botões montados de `permitidas` (a da resposta, e a `dados.permitidas` do erro). Filtro e ordenação na URL, `Skeleton` no carregamento, tabela larga rolando no próprio contêiner.
- O bloco de ação (botões e os dois `Alert`s) aparece nas **duas** superfícies — linha da lista e detalhe —, e mora num componente só: dois tratamentos do mesmo desfecho divergiriam na primeira mudança.
- Consulta a cada 10 s enquanto a página listar Pedido não terminal (`terminal` do Go, de `EstadoTerminal`).

**Ask First:**
- Migração, coluna nova, ou mudar a tabela de transições / as três recusas de `maquina.go`.

**Never:**
- Rota administrativa de cancelamento, e nenhum quarto botão: o Administrador não cancela (FR-32).
- `Toast` genérico para o Pedido que já avançou — é `Alert` informativo, não destrutivo.
- Selo `Badge` por Status (6.7); ler o sinal de pagamento aprovado sobre cancelado (6.5).
- Reusar `saidaPedidoDetalhe` do Comprador: o Administrador não vê `expira_em`, `tentativas_restantes` nem `pode_cancelar`.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Lista | sem filtro | todos os Pedidos, de todos os Compradores, mais recentes primeiro; envelope com o `total` real | N/A |
| Filtro e ordem | `?status=PAGO&ordenacao=antigos` | só os `PAGO`, do mais antigo ao mais novo; a contagem repete o mesmo `WHERE` | Status ou ordenação fora da lista: 400 `CAMPO_INVALIDO` |
| Detalhe | id qualquer | Comprador (nome e e-mail), Itens com o Vendedor congelado, valores, Endereço, histórico e `permitidas` | uuid malformado ou inexistente: 404 |
| Transição válida | `{de: PAGO, para: SEPARANDO}` | 200 com o Pedido em `SEPARANDO` e linha do histórico com ator `ADMINISTRADOR` | N/A |
| Consolidação | `{de: SEPARANDO, para: ENVIADO}` | 200; o `estoque_total` baixa e a Reserva encerra | N/A |
| Transição fora da tabela | `{de: PAGO, para: ENTREGUE}` | 409 `TRANSICAO_INVALIDA` com `dados.permitidas`; nada gravado | `Alert` destrutivo nomeia a tentada e as permitidas |
| Pedido já avançou | tela em `PAGO`, a simulação já o levou a `SEPARANDO` | 409 `ESTADO_JA_AVANCADO` com o Status atual em `dados.status`; nada gravado | `Alert` informativo: "Este Pedido já está em {estado}. A linha foi atualizada." |
| Cancelar pela API | `{para: CANCELADO}` | 409 `TRANSICAO_INVALIDA` — o ator `ADMINISTRADOR` não tem essa linha | N/A |
| Sem Sessão de Administrador | sem cookie, ou Sessão de Comprador | o 404 do prefixo administrativo, como nas outras rotas de `/admin/` | N/A |

</frozen-after-approval>

## Code Map

- `internal/pedido/maquina.go:130,155` -- `transicoes` (as três de `AtorAdministrador`) e `Permitidas`. Não muda.
- `internal/pedido/maquina.go:196` -- `Transicionar`: valida, CAS, histórico e efeito. É a única escrita de Status.
- `internal/pedido/pedido.go:527,615` -- `Detalhar` e `Listar` do Comprador: moldes das funções novas, e não reuso — o dono está no `WHERE`, e o Detalhe carrega a tripla que o Administrador não vê.
- `internal/pedido/db/consultas.sql:100,113,142,152` -- `BuscarPedidoDoComprador`, `ItensDoPedido` (acrescentar `vendedor_nome`, já congelado na tabela), `ListarPedidosDoComprador` e `ContarPedidosDoComprador`: moldes dos pares administrativos.
- `internal/busca/db/consultas.sql:16` -- a ordenação por `CASE WHEN @ordenacao = … THEN … END`, com desempate final em `id`: o padrão a copiar.
- `internal/busca/busca.go:22,71` -- as constantes de ordenação e o vazio valendo o padrão.
- `internal/identidade/identidade.go:58,71` -- `Conta` e o molde de porta; `db/consultas.sql:5` -- `BuscarCompradorPorEmail`, molde da busca por id.
- `api/produto_admin.go:52` -- `listarProdutosAdmin`: molde do handler administrativo, com `paginacaoDe` e envelope.
- `api/pedido.go:48,300` -- `saidaDoPedido` (reusável) e `cancelarPedido` (molde de `emTransacao` e da recusa que leva `dados`).
- `api/busca.go:99` -- validação de `ordenacao` por lista fechada, com `EscreverCampo`.
- `api/rotas.go:110,203` -- o mux `admin` (a guarda mora no prefixo) e `paginacaoDe`.
- `internal/plataforma/erro/erro.go:66,121` -- as três recusas já registradas, e `dados.permitidas` preenchido num ponto só.
- `internal/fronteira_test.go:32` -- `pedido → identidade` já é aresta permitida do AD-1.
- `api/sessao_test.go:551` e `api/autorizacao_test.go:63` -- onde as matrizes de servidor são registradas e a tabela de negação.
- `web/components/casca-admin.tsx:25` -- `DESTINOS`: acrescentar "Pedidos".
- `web/app/admin/produtos/produtos.tsx` -- tabela administrativa com `Table`, `Select`, `Skeleton`, paginação na URL e `Alert`.
- `web/app/pedidos/meus-pedidos.tsx:37` -- `Data` (RFC 3339 → pt-BR) e o molde de listagem.
- `web/lib/pedido.ts:283,307` -- `rotuloDoStatus`, ponto único de rótulo, e `linhaDoTempo` para o histórico do detalhe.

## Tasks & Acceptance

**Execution:**
- [x] `internal/identidade/db/consultas.sql` + `identidade.go` -- `BuscarCompradorPorID` e a porta `BuscarComprador(ctx, bd, id) (Conta, error)`; uuid malformado vira `pgx.ErrNoRows`.
- [x] `internal/pedido/db/consultas.sql` -- `ListarPedidosAdmin`/`ContarPedidosAdmin` (filtro opcional por Status, com o mesmo `WHERE` nas duas; ordenação `recentes`/`antigos`; desempate em `id`), `BuscarPedidoParaAdministrador` (sem dono, com `comprador_id`) e `vendedor_nome` em `ItensDoPedido`. `sqlc generate` na imagem `sqlc/sqlc:1.31.1`, restaurando os `gerado/` dos outros módulos.
- [x] `internal/pedido/pedido.go` -- `ListarParaAdministrador` (conta antes e curto-circuita a página além do total) e `DetalharParaAdministrador` (Comprador por `identidade`, Itens com Vendedor, Endereço e histórico), com tipo de saída próprio, sem a tripla.
- [x] `api/pedido_admin.go` + `api/rotas.go` -- as três rotas no mux `admin`; `permitidas` de `pedido.Permitidas(status, AtorAdministrador)` na lista e no detalhe; `ESTADO_JA_AVANCADO` relê o Status atual e o leva em `dados.status`, como o cancelamento.
- [x] `api/pedido_admin_test.go` + `api/sessao_test.go` + `api/autorizacao_test.go` -- a matriz inteira contra o banco, mais: as três transições andando com ator `ADMINISTRADOR`, `CANCELADO` recusado como transição inválida, a corrida contra a simulação, e as três rotas na tabela de negação administrativa.
- [x] `web/lib/pedido.ts` + `web/scripts/pedido.test.mjs` -- o rótulo da ação por destino, `desfechoDaTransicao` (200 → relê; `TRANSICAO_INVALIDA` → `Alert` destrutivo com a tentada e as permitidas; `ESTADO_JA_AVANCADO` → `Alert` informativo com o Status; resto → erro) e a leitura de filtro e ordenação da URL, com teste.
- [x] `web/app/admin/pedidos/page.tsx` + `pedidos.tsx` -- a lista: filtro por Status, ordenação por data, `Skeleton`, paginação e consulta de 10 s enquanto houver Pedido não terminal.
- [x] `web/app/admin/pedidos/[id]/page.tsx` + o componente do detalhe -- Comprador, Vendedores dos Itens, valores, Endereço, linha do tempo e o mesmo bloco de ação da lista.
- [x] `web/components/casca-admin.tsx` -- "Pedidos" na Navegação administrativa.
- [x] `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` §10 + `.memlog.md` -- o `de` vindo da tela como chave do CAS; `deferred-work.md` -- o que a estória deixar aberto.

**Acceptance Criteria:**
- Dado um Pedido em `PAGO`, quando o Administrador aciona a ação, então o Status muda na tela sem recarregar e o histórico ganha a linha com ator `ADMINISTRADOR`.
- Dado um Pedido que a simulação levou adiante enquanto a tela estava aberta, quando o Administrador aciona a ação que viu, então a linha é atualizada e o `Alert` informativo diz em que Status ele está — nunca "transição inválida".
- Dado 1440 px e só o teclado, quando a tabela não cabe, então ela rola dentro do próprio contêiner e a página não rola na horizontal.

## Implementation Notes

**Uma função a mais em `pedido.go` do que a lista de Tasks previa.** As Tasks punham a sequência da transição em `api/pedido_admin.go`; ela ficou em `pedido.TransicionarPeloAdministrador`, no molde exato de `Cancelar`: `TravarPedidoParaAdministrador` (consulta nova, `FOR UPDATE` sem dono) e só então `Transicionar(…, ADMINISTRADOR)`. A trava resolve de uma vez as duas coisas que a rota precisava e o CAS sozinho não dá — o 404 do Pedido inexistente ou malformado, que o CAS vazio não distingue da corrida, e o Status que a recusa leva em `dados.status`. O handler continua sendo só tradução.

**`de` vazio ou fora dos sete não ganhou validação própria.** Chega à máquina e sai 409 `TRANSICAO_INVALIDA` com `permitidas` vazio, sem gravar nada: é o mesmo desfecho de qualquer transição inexistente, e um 400 a mais aqui seria uma segunda declaração de quais Status existem.

**`ItemDoPedido` ganhou `VendedorNome` em vez de nascer um segundo tipo.** A coluna já estava na tabela e a linha já era lida; o Detalhe do Comprador simplesmente não a serializa. Pelo mesmo raciocínio, a saída administrativa não carrega `Disponivel`: ler `catalogo.Disponivel` ali seria uma consulta por um campo que a resposta não tem.

**A 6.7 continua dona do selo.** O Status sai como texto por `rotuloDoStatus` nas duas telas novas, sem `Badge` e sem cor, como nas outras duas superfícies.

**Verificado por mutação, como a spec pedia:** trocado o `esperado` do corpo pelo Status lido sob a trava em `TransicionarPeloAdministrador`, o subteste "o Pedido que a simulação já levou adiante sai como já avançado" falha com `codigo = TRANSICAO_INVALIDA, quero ESTADO_JA_AVANCADO`. Revertido em seguida.

**De carona, uma correção:** as duas telas novas numeram as consultas (`ultima`), como `acompanhamento.tsx` desde a 5.8 — sem isso, a consulta de 10 s que saiu antes do clique voltaria depois da releitura e traria o Status velho de volta logo depois de o `Alert` dizer que a linha foi atualizada.

## Spec Change Log

## Review Triage Log

Passe 1 — três lentes (`blind-hunter`, `edge-case-hunter`, `verification-gap`): o diff abre rota administrativa nova, mexe na máquina de estados pela borda e cria duas telas.

| # | Achado | Veredito | Rota |
|---|--------|----------|------|
| 1 | O `Alert` do desfecho mora no componente da linha: com a Tabela filtrada por Status, a releitura tira a linha da lista, o componente desmonta e o Administrador não lê desfecho nenhum — o Critério de Aceite 2 falha sempre que há filtro | medium | patch — o desfecho sobe para a superfície |
| 2 | A região `role="status"` é inserida no DOM junto com o texto; região viva que não existia antes costuma não ser anunciada | medium | patch — a região passa a existir sempre |
| 3 | `aria-live="polite"` em cada `TableCell` de Status: 20 regiões vivas numa página, e o anúncio ("Separando") não diz de qual Pedido | medium | patch — uma região só, nomeando o Pedido |
| 4 | O 404 do POST de transição com Sessão de Administrador (id inexistente e malformado) não tem teste: tirar a tradução do `ErrNoRows` deixa a suíte verde e a rota passa a devolver 500 | medium | patch — pré-verificado pela lente |
| 5 | O filtro valida contra `pedido.Statuses`, e só `PAGO` é exercitado: tirar um dos sete da lista deixa a suíte verde e o `Select` da tela passa a levar 400 | medium | patch — pré-verificado pela lente |
| 6 | `de` vazio ou fora dos sete está documentado nas Implementation Notes (409 `TRANSICAO_INVALIDA`, `permitidas` vazio, nada gravado) e não tem teste | medium | patch |
| 7 | O comentário do handler e o addendum dizem que a tela remonta os botões da própria resposta 200; ela relê a superfície nas três saídas | low | patch — correção direta da frase |
| 8 | `ListarPedidosAdmin` avalia a subconsulta correlacionada três vezes por linha, sobre varredura global, sem o `ponytail:` que as outras consultas do arquivo têm | low | patch — comentário |
| 9 | O Detalhe administrativo redeclara a cadência (`pedido !== null && !pedido.terminal`) em vez de usar `intervaloDaTabela`: mudar o ritmo exigiria dois lugares | low | patch — trivial |
| 10 | `epic-6-context.md` foi regenerado e perdeu o que as estórias 6.1–6.3 acumularam (paginação de 20, NFR-10/11/13, AD-3/6/15/18, e o "`dados` precisa ser mapa, porque struct perde `permitidas` em silêncio", que é justamente o que esta estória usa) | medium | corrigido no passe — restaurado de `4b12ac3`; a regeneração foi do passo 1 deste build, não da implementação |
| 11 | O `de` que a tela envia — a base inteira da distinção entre as duas recusas — não tem teste, porque `web/` não tem bancada de montar componente | medium (pré-verificado) | defer — fechar exige a bancada, maior que a estória |
| 12 | `painelDePedidosDoAdministrador` depende de rodar por último em `TestSessaoEProduto`: a primeira asserção compara com o total global | low | defer |
| — | `criado_em` zero na resposta 200 da transição, e o `criado_em?` opcional no TS escondendo isso | false | `saidaDoPedido` só preenche o campo com `!IsZero`, e a tag é `omitempty`: a resposta da transição **omite** a chave, e o opcional no TS é o contrato correto |
| — | `atualizado_em` exibindo "01/01/0001" no Pedido sem histórico | false | `Criar` grava a transição de nascimento na mesma transação; `Historico` devolve `ErrNoRows` com histórico vazio, e o handler traduz em 404 — a tela nunca recebe o instante zero |
| — | Todos os botões da linha viram "Salvando…" durante o envio | low, rejeitado | `Permitidas(status, ADMINISTRADOR)` nunca devolve mais de um destino na tabela do AD-3; a correção acrescentaria estado para uma situação que não se alcança |
| — | `aria-disabled` sem bloqueio real deixa o estado desabilitado cosmético | low, rejeitado | é exatamente o que a UX-DR11 pede no passo em curso — presente, no caminho do foco e com progresso; e o rótulo "Salvando…" é o retorno do clique engolido |
| — | O contêiner de rolagem da tabela não é focável (WCAG 2.1.1) | low, rejeitado | cada linha tem link e botão focáveis, então o conteúdo rolado é alcançável pelo teclado; é o mesmo `Table` das outras três tabelas administrativas |
| — | O 404 de Pedido inexistente leva o Administrador autenticado ao login | low, rejeitado | Pedido não se remove no sistema; só um id digitado à mão chega lá, e distinguir os dois 404 acrescentaria ramo para isso |
| — | `ESTADO_JA_AVANCADO` sem `dados.status` escreveria "já está em ." | low, rejeitado | o Status sai da leitura travada, que só falha antes por `ErrNoRows` — a rota nunca emite essa recusa sem Status |
| — | Falta caminho de recuperação quando a primeira leitura falha (as duas telas) | low, rejeitado | mesmo comportamento de "Meus pedidos" e das outras telas administrativas; recarregar resolve, e um botão de repetir acrescentaria ramo em quatro telas ou em nenhuma |
| — | `sprint-status.yaml` e a spec ainda em `in-progress` com as Tasks marcadas | false | os dois já estavam em `in-review` quando a revisão rodou; o passo 5 os leva a `review` |

## Design Notes

**Por que `de` viaja no corpo:** o CAS precisa do estado esperado, e o estado que o Administrador viu é o da tela. Lido travado no servidor, a corrida sumiria — a transição seria aplicada a partir de um estado que ninguém viu, ou recusada como inválida, que é a causa falsa que a UX proíbe nomear assim.

**Por que não reusar o Detalhe do Comprador:** `saidaPedidoDetalhe` carrega a tripla, `expira_em` e `pode_cancelar`, que são do Comprador; e `BuscarPedidoDoComprador` tem o dono no `WHERE`, que é a garantia do AD-11 e não se afrouxa com um parâmetro nulo. São duas leituras, com públicos diferentes.

## Verification

**Commands:**
- `go vet ./... && go test -count=1 ./...` na imagem `azamon-dev` (HANDOFF) -- verde.
- `docker build ./web` -- verde (`prebuild` roda `verificar-offline.mjs` e `npm test`).
- Mutação: trocar o `de` do corpo pelo Status lido no servidor -- o teste da corrida com a simulação falha. Reverter.

**Manual checks (if no CLI):**
- A 1440 e 360 px, só pelo teclado: filtrar por `PAGO`, inverter a ordenação, recarregar (o estado se mantém na URL) e avançar um Pedido pelos três botões até `ENTREGUE`, vendo o Estoque total baixar no `ENVIADO`.
