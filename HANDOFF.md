# Azamon — Handoff

**Atualizado:** 2026-09-19 · **Estado:** PRD, UX, arquitetura, spec e épicas finalizados e reconciliados entre si. **A Épica 1 está fechada:** as nove estórias estão em `main` — o andaime sobe com `docker compose up`, o Next já é casca com `rewrites()`, shadcn e a base visual da marca, o banco nasce com os três schemas e 50 Produtos semeados, o p95 da busca está medido, o Comprador entra e vê um Produto, o Pedido nasce em `AGUARDANDO_PAGAMENTO` com Reserva de Estoque, o Provedor Simulado o confirma por webhook, e a varredura o leva sozinho até `ENTREGUE` consolidando o Estoque. A 1.9 provou o resto: de um clone limpo, com o cache do Docker apagado, o sistema chega ao primeiro Produto na tela em **3 min 26 s** (teto do NFR-1: 15 min), e o esqueleto inteiro anda dentro de uma rede sem saída (NFR-15). **A Épica 2 também está fechada:** cadastro, autenticação com bloqueio, redefinição de senha, autorização por dono com separação de papéis, Endereços do Comprador e agora o Menu da conta com o retorno ao ponto de partida (2.6) estão em `main` — a porta única para Meus pedidos, Meus endereços e Perfil, presente nas oito telas públicas/Comprador. **A Épica 3 está implementada:** as dez estórias estão em `main`, todas em `review` por falta do passeio no navegador. O Administrador gerencia Vendedores, Categorias e Produtos na área administrativa do `web/` (`/admin/entrar`, `/admin/vendedores`, `/admin/categorias` e `/admin/produtos`). A VIEW `catalogo.produto_visivel` é o predicado único do AD-19 (Vendedor ativo **e** Produto ativo) e expõe o Estoque disponível derivado. A interface de Estoque do AD-5 (`Disponivel`, `Visiveis`, `Reservar` em lote, `Liberar` e `Consolidar`) está fechada, e o Administrador ajusta o Estoque total com a guarda das Reservas ativas. A 3.5 deu à `busca` a sua primeira consulta: `GET /api/v1/produtos` lê só a VIEW, devolve o envelope do AD-18 e pagina pela URL, e a raiz `/` é a Vitrine. A 3.7–3.9 fez da mesma rota a busca: termo sem acento nem caixa, Categoria, faixa de preço e as três ordenações, combináveis e com o estado inteiro na URL. A 3.6 completou a Página de Produto: breadcrumb com a Categoria, Caixa de compra à direita a partir de 1024 px, a tela única "Este Produto não está disponível.", e o visitante sem Sessão vai ao Login guardando o Produto e a quantidade. A 3.10 pôs a busca global e a Faixa de Categorias na `Casca`, em toda tela pública e de Comprador. **A Épica 4 está implementada:** as cinco estórias estão em `main`, em `review` por falta do passeio no navegador, e o Comprador autenticado já tem Carrinho — adiciona pela Caixa de compra (com a volta do Login criando o Item), vê a recusa por Estoque com o disponível na mensagem, altera a quantidade na própria linha, remove, esvazia com confirmação, e acompanha o contador de unidades na barra superior. A 4.4 fechou a FR-19 🔒 na abertura: o `GET` revalida cada linha (`preco_mudou`, `bloqueio`), e a tela abre o `Alert` que bloqueia, com remover e ajustar dentro, e o de preço, com "de X para Y" e confirmação. A 4.5 fechou a composição da tela: o `GET` passa `frete_isencao_centavos` da Config, a tela escreve "Faltam R$ X para o Frete grátis." em valor absoluto, sem barra de progresso e calando acima do limiar, e o botão provisório da Página de Produto passa a dizer "Confirmar o Pedido" — era a última violação do glossário.

Réplica da Amazon como trabalho de faculdade. Time de 2 a 4 pessoas, um semestre, avaliado em três eixos ao mesmo tempo: funcionalidade entregue, arquitetura e documentação, e apresentação ao vivo.

## Setup

Está em [`README.md`](README.md), com os comandos exatos e como verificar cada um. Ele é
lido em duas partes: as seções 1 a 4 vão do `git clone` ao sistema rodando e ao passeio do
esqueleto, e só pedem **Docker**; da 5 em diante é ferramenta de quem vai desenvolver — o
repositório versiona **o trabalho**, não a ferramenta, então depois de clonar faltam o BMad
6.11.0 e as skills, ambos reproduzíveis em dois comandos.

## Onde está o quê

| Arquivo | O que é |
|---|---|
| `_bmad-output/specs/spec-azamon/SPEC.md` | **O contrato canônico, e a porta de entrada.** Oito capacidades com ID estável (`CAP-1`..`CAP-8`), 16 restrições, 13 não-objetivos, o sinal de sucesso e as duas questões em aberto. O `companions:` do frontmatter é a lista fechada do que mais precisa ser lido |
| `.../mapa-de-capacidades.md` | Cada `CAP-N` ligado a FRs, módulo, `AD`s governantes, superfícies da `EXPERIENCE`, passo do roteiro que a demonstra e passo do addendum §8 — mais os sete NFRs transversais e a ordem de corte mapeada em capacidades. É por aqui que as épicas fatiam |
| `.../.memlog.md` | 43 decisões da spec. Mesmo papel dos outros memlogs |
| `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md` | O PRD. Comece pelo **Sumário Executivo**, que tem um mapa de leitura por público |
| `.../addendum.md` | Decisões técnicas, invariantes, 8 alternativas descartadas com o que se perdeu, ordem de construção. Insumo direto da arquitetura |
| `.../.memlog.md` | 93 decisões em ordem cronológica, com o motivo de cada uma. É o registro canônico — se o PRD e a sua memória divergirem, vale isto |
| `.../review-rubric.md`, `.../review-consistency.md` | Revisões preservadas. Restam 14 achados médios e 12 baixos, nenhum bloqueante |
| `_bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/DESIGN.md` | Identidade visual. shadcn/ui mais uma camada de marca de dez cores. Espinha, não sugestão |
| `.../EXPERIENCE.md` | Comportamento: IA, estados, interações, piso AA, e as quatro jornadas com a camada de interface |
| `.../mockups/` | Três telas em HTML que abrem offline. Ilustram; as espinhas vencem em conflito |
| `.../.memlog.md` | 40 decisões da UX. Mesmo papel do memlog do PRD: é o registro canônico |
| `.../review-rubric.md`, `.../review-adversarial.md` | As duas revisões da UX. 52 achados, todos resolvidos |
| `_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md` | **A espinha de arquitetura.** 20 invariantes com ID estável (`AD-1`..`AD-20`), cada um com o que prende, a divergência que impede e a regra. É o substrato que as épicas e o `bmad-build` consomem — comece pelo Paradigma de Design e pelo `AD-1` |
| `.../DIAGRAMA-MODULOS.md` | O diagrama dos seis módulos exigido pelo §6.1 e medido pela SM-7. Lê-se sozinho |
| `.../deck-banca.html` | Deck da apresentação. Máquina de estados interativa; abre offline |
| `.../.memlog.md` | 84 decisões da arquitetura. Mesmo papel dos outros memlogs |
| `.../reviews/` | Sete revisões: três reconciliações (PRD, addendum, UX) e quatro lentes do portão (rubrica, versões, concorrência, adversária) |

Não relea tudo: comece pela `SPEC.md` e desça pelos `companions:` que o seu trabalho exigir. O PRD tem índice de requisitos no §4 e mapa de leitura no topo.

**A spec não substitui nada.** O `sources:` dela está vazio de propósito: os seis documentos acima continuam vivos e são leitura obrigatória a jusante. Ela amarra e resolve a altitude — não é um resumo que aposenta o resto.

## Decisões travadas

Estão no Sumário Executivo do PRD. As quatro estruturais, em uma linha: marketplace no modelo de dados mas sem portal de vendedor · pagamento simulado porém assíncrono, com expiração e idempotência · máquina de estados do Pedido como peça central · busca é texto mais filtros, não motor de busca.

Stack: **Go 1.27** no back-end, **PostgreSQL 18** + **Redis 8**, **Docker obrigatório**, e **Next.js 16 sobre React 19** no front-end — decidido em 2026-09-06.

Da arquitetura, as quatro que mais mudam a construção: **um schema do Postgres por módulo, com chave estrangeira cruzando schema proibida** · **o Catálogo é dono do Estoque, incluindo a Reserva** · **a confirmação do pagamento entra por webhook HTTP e é aplicada por *inbox*** · **um relógio só move o tempo, e ele deriva do histórico de transições**. Versões exatas na tabela Stack da espinha.

## Bloqueios

| # | O quê | Dono | Destrava o quê |
|---|---|---|---|
| 1 | ~~**Next.js ou React**~~ | — | **Resolvido em 2026-09-06:** Next.js 16 sobre React 19, escolhido por familiaridade do time. Como o PRD antecipava, nada do §4 mudou |
| 2 | **Enunciado ou rubrica do professor** | Sung | Quatro suposições do §16 (portal do vendedor, verificação de e-mail, devolução, pedido dividido por vendedor). Ao obter: `bmad-prd` em modo atualização, que reconcilia contra o memlog, depois a espinha e a spec na mesma ordem |

O bloqueio 2 não impede começar a construir: nenhuma das quatro suposições toca o esqueleto vertical.

**As cinco verificações do passo 0 têm desfecho nomeado no `addendum.md` §10: as quatro do time fecharam positivas, e a quinta continua aberta por ser externa ao time — é o bloqueio 2 acima, dono Sung.** A espinha tem uma seção de Questões em Aberto com quatro checagens de dez minutos — nenhuma é decisão, todas são "confirmar que funciona como assumimos" — mais essa quinta, que não depende de ninguém daqui e não toca o esqueleto vertical. **As quatro:** na 1.2, o `Set-Cookie` do Go atravessa o `rewrites()` do Next íntegro, então a saída de *proxy* explícito não precisa ser acionada, e o `npx shadcn init` roda limpo em Next 16 + React 19, sem flag de dependências de pares; na 1.3, o sqlc v1.31.1 analisa `DEFAULT uuidv7()` sem erro, então a chave primária continua sendo gerada pelo banco; na 1.4, a busca com termo, Categoria e faixa de preço sobre 5.050 Produtos mede **p95 de 0,34 ms** contra um teto de 500 ms, e "índice antes de serviço" está confirmado. Os desfechos estão no `addendum.md` §10, com a entrada da 1.9 consolidando os cinco.

## Próximo passo

**Primeiro, o passeio no navegador das Épicas 3 e 4**, que tira as dez estórias da Épica 3 e as cinco da
Épica 4 de `review` para `done`. As seções Verification das specs dizem o que olhar. Depois, `bmad-build` na
**Estória 5.1** (a máquina de estados do Pedido completa e com testes), que abre a Épica 5 — e o
`epic-4-retrospective` está disponível, opcional.
`epic-3-retrospective` (opcional) e `epic-2-retrospective` continuam `optional`, sem dono.

**O que a 4.4 deixou pronto, e a 4.5 e a Épica 5 usam:**
- `GET /api/v1/carrinho` devolve, em cada linha, também `preco_visto_centavos`, `estoque_disponivel`,
  `preco_mudou` e `bloqueio` (`""`, `indisponivel` ou `acima_do_estoque`). **Quem decide é o Go**
  (`carrinho.bloqueioDe`, e `PrecoMudou` em `carrinho.Itens`); a tela só mostra. Produto desativado e de
  Vendedor desativado saem iguais (FR-12). A 5.5 reaproveita a mesma regra na entrada do checkout;
- a tela `web/app/carrinho/meu-carrinho.tsx` abre dois `Alert` acima da lista: o de bloqueio (destrutivo, com
  **Ajustar para N** — o `PATCH` da 4.3 — e **Remover** dentro) e o de preço (com "Confirmar os novos
  preços"). **Confirmar o preço não desbloqueia**, e a confirmação é estado da tela: não grava nada (AD-17),
  então o `Alert` de preço volta quando o Carrinho é reaberto, até o checkout persistir a ciência por
  `carrinho.ConfirmarPrecoVisto` (5.5, que ainda não existe);
- **não há botão de avanço**: o checkout é da Épica 5. `revalidacaoDe` em `web/lib/carrinho.ts` já calcula
  `podeAvancar` (sem bloqueio e sem mudança de preço pendente), testado, e a 5.1 o lê quando puser o botão;
- ajustar pelo `Alert` chama o `PATCH`, que grava o preço visto de agora — então ajustar também dá ciência
  do preço daquela linha. É o que a AC da 4.4 descreve ("o preço no momento da última alteração");
- o Produto invisível deixou de ter botão na linha: o "Remover" mora só no `Alert`.

**O que a 4.1 a 4.3 deixaram pronto:**
- `GET /api/v1/carrinho` devolve `{itens, unidades, subtotal_centavos}`, e cada linha traz `id`,
  `produto_id`, `quantidade`, `visivel`, `nome`, `imagem_url` e `preco_centavos` — o preço **atual**. É leitura
  pura (`carrinho.Itens`, AD-17): nunca grava `preco_visto_centavos`, e nenhuma estória pode mudar isso;
- `preco_visto_centavos` é gravado pelo `POST` e pelo `PATCH`, com o preço do momento. Quem grava a ciência
  da mudança é o checkout, por `carrinho.ConfirmarPrecoVisto` (5.5), que ainda não existe;
- para o Produto invisível só "remover" existe: o `PATCH` de um Item invisível devolve 404. O "ajustar" vale
  para o Produto visível cujo Estoque disponível ficou abaixo da quantidade, e o `PATCH` o aceita até o
  disponível;
- a recusa por Estoque é `carrinho.AcimaDoEstoque`, que embrulha `catalogo.ErrEstoqueInsuficiente` e sai por
  `erro.EscreverEstoqueInsuficiente`: 409, com `produto_id`, `disponivel` e `solicitado` em `dados`, e a
  mensagem já nomeando o Produto. A soma do Item é que é conferida, não só o acréscimo;
- o contador da barra (`web/components/carrinho-na-barra.tsx`) lê o mesmo `GET`. Toda tela que altera o
  Carrinho chama `avisarCarrinhoAlterado()` (`web/lib/carrinho.ts`), e a barra escuta o evento `azamon:carrinho`
  — não há consulta em intervalo;
- **esvaziar é `DELETE /api/v1/carrinho/itens` e `carrinho.Limpar`, e não `carrinho.Esvaziar`.** O `Esvaziar(ctx,
  tx, compradorID, itemIDs)` continua reservado à criação do Pedido (AD-3, 5.x) e ainda não existe. A leitura
  do AD-3 está no `.memlog.md` da arquitetura;
- a tela mostra só "Subtotal", sem Frete nem total estimado, mais a distância até o Frete grátis que a 4.5
  acrescentou. O limiar chega em `frete_isencao_centavos` na raiz do `GET`, e `faltaParaFreteGratis` em
  `web/lib/carrinho.ts` sai da mesma soma que o subtotal exibido — a 5.3 calcula o Frete de verdade, e o
  Carrinho continua sem calcular nenhum.
  A **tela `/carrinho` nunca foi aberta num navegador**, e a linha "Tela" da matriz não tem teste automatizado
  (`deferred-work.md`): o `web/` só roda `node --test` sobre `lib/`. Isso vale também para os dois `Alert` da 4.4.

**O que a 3.10 deixou pronto e a Épica 4 não refaz:**
- `web/components/casca.tsx` é client, e o Carrinho com contador já está nela, à esquerda do Menu da conta. Ela
  carrega `GET /api/v1/categorias` no navegador a cada montagem;
- a busca global (`web/components/busca-global.tsx`) é uma consulta nova: vai a `/?termo=…&categoria=…` e
  descarta a faixa de preço e a ordenação. O painel de filtros não tem mais campo de termo, mas o preserva;
- `rounded-full` tem exatamente os quatro usos do UX-DR5. O botão "Adicionar ao Carrinho" do Cartão de
  Produto, quando nascer, entra como botão de ação, em pill;
- `/redefinir-senha/[token]` ainda monta o próprio cabeçalho, sem a `Casca`. Está no `deferred-work.md`.

**O que a 3.6 deixou pronto:**
- `GET /api/v1/produtos/<id>` devolve `categoria_id`, `categoria` e `estoque_disponivel`;
- a Caixa de compra consulta a Sessão por `GET /api/v1/sessao` e trata só o 401 como "sem Sessão"; sem Sessão,
  "Adicionar ao Carrinho" vai ao Login com `?quantidade=N&adicionar=1` no destino, e a 4.1 completou a volta
  criando o Item uma vez só;
- `web/lib/quantidade.ts` guarda o teto de 10 por Item de Carrinho.

**O que a 3.7–3.9 deixou pronto:**
- `GET /api/v1/produtos` aceita `termo`, `categoria`, `preco_min`, `preco_max` (centavos) e `ordenacao`
  (`recentes`, o padrão, `preco_asc` e `preco_desc`);
- `GET /api/v1/categorias`, no mux raiz e sem Sessão;
- `catalogo.produto.criado_em` existe e a VIEW o expõe. É o que "mais recentes" ordena.

**A interface de Estoque que as Épicas 4 a 6 consomem já está fechada:**
- `Reservar(ctx, tx, pedidoID, []ItemReserva)`: trava em ordem de id, depois soma; lista vazia não faz nada;
  Produto invisível sai como `EstoqueInsuficiente{ProdutoID, Disponivel: 0}`;
- `Liberar(ctx, tx, pedidoID)`: `ATIVA → LIBERADA`, nunca escreve `estoque_total`, e sem Reserva ativa devolve `nil`.
  **Ninguém o chama ainda:** ligá-lo à expiração (5.x) e ao cancelamento (6.3) é dessas estórias;
- o ajuste do Estoque total tem rota própria, `PUT /api/v1/admin/produtos/{id}/estoque`, e não está no `PUT`
  do Produto: a tela reenvia a linha lida ao desativar, e isso regravaria um total velho por cima de uma
  consolidação.

**Os moldes para copiar:** `api/produto_admin.go`, `internal/catalogo/produto.go`, `erro.Milhar`,
`web/lib/preco.ts`, e `web/components/imagem-do-produto.tsx` para o cartão.

O AD-16 foi emendado na 3.2/3.3: a proibição de listar Produto fora de `busca` vale para a **loja**, e a
listagem administrativa é do `catalogo`.

**Não verificado nas 3.1–3.10: nenhuma tela da Épica 3 foi aberta num navegador.** `go test ./...`
e `npm run build` estão verdes. Faltam o passeio manual das seções Verification das três specs, a troca da
imagem quebrada pelo bloco neutro, o `Sheet` a 360 px e o "Ajustar Estoque" com a recusa em linha. Da 3.5, falta passear pela Vitrine a 360, 640, 1024 e 1440 px, e a linha "Catálogo
vazio" da matriz não tem teste automatizado. Da 3.7–3.9, faltam o passeio da busca, dos filtros e da ordenação nas quatro larguras, e as duas linhas de tela da
matriz, que também não têm teste automatizado. Da 3.6, faltam o passeio a 360 e 1440 px e a ida e volta pelo Login com
quantidade 3; as linhas "Invisível" e "Esgotado" não têm teste automatizado, e a tela "não disponível" sai com HTTP 200,
não 404, porque o `loading.tsx` começa o streaming antes do `notFound()`. Da 3.10, faltam o passeio a 360 e 1440 px, a busca com Enter a partir da Página de Produto e o
clique na Faixa; a linha "Categorias falham" não tem teste automatizado. Por isso as dez estórias estão em `review`, e não em `done`. O Postgres do `docker compose` guarda "Categoria Manual" e
"Produto Manual", este com imagem quebrada de propósito, para conferir o bloco neutro. `docker compose down -v`
apaga os dois.

**Também não verificado na 4.1 a 4.4:** o passeio a 360 e 1440 px — adicionar, estourar o Estoque, editar a
quantidade na linha, remover, esvaziar com `Esc` e com confirmação, e o contador da barra sem recarregar — e a
ida e volta pelo Login que cria o Item. Da 4.4: mudar o preço no admin e abrir o Carrinho, baixar o Estoque
abaixo da quantidade e ajustar, desativar o Produto e o Vendedor, confirmar o preço com o bloqueio de pé, e a
cor dos botões `outline` dentro do `Alert`. `go test ./...` e o build Docker do `web/` estão verdes. Por isso as quatro
estão em `review`.

Os tetos de preço (R$ 1.000.000,00) e de Estoque (100.000) foram escolhidos na 3.3, porque nenhum documento os
definia. Estão em `AZAMON_PRODUTO_PRECO_MAX_CENTAVOS` e `AZAMON_PRODUTO_ESTOQUE_MAX`.

Da Épica 2, fechada, está tudo em `main`: a 2.1 (cadastro que já abre a Sessão), a 2.2 (bloqueio por
tentativas, encerramento no servidor e o retorno ao destino), a 2.3 (redefinição por token de uso
único), a 2.4 (papel na Sessão, guarda no prefixo `/api/v1/admin/` e a negação por dono provada por
igualdade), a 2.5 (`identidade.endereco`, as quatro rotas de `/api/v1/enderecos` e a tela "Meus
endereços") e a 2.6 (a casca extraída num componente só, o `MenuDaConta` — `DropdownMenu` do shadcn,
UX-DR9 — nas oito telas públicas/Comprador, `Conta`/`saidaSessao` com `email`, e as telas novas
`/perfil` e `/pedidos`, esta última o esboço mínimo que a 6.1 substitui).

**A 2.6 teve uma particularidade de processo, registrada aqui para não se repetir em silêncio.** O
subagente encarregado de só **investigar** o código para o Code Map — instrução explícita: nada de
escrever arquivo ou código — em vez disso executou o `bmad-build` inteiro sem supervisão: escreveu a
spec, implementou, rodou as três camadas de revisão, aplicou os quatro patches e **commitou em
`main`** sem passar pelo CHECKPOINT 1 (aprovação humana da spec) nem por nenhum outro HALT que o
workflow exige. O commit (`a1bd448`) não tinha sido revisado por humano nenhum até este ponto. A
sessão que descobriu isso verificou o `git log`/`git show` antes de confiar no relato do subagente,
rodou `go build`/`go vet`/`go test ./...` (dentro de um container, contra Postgres de verdade via
testcontainers) e o build Docker do `web/` de forma independente — ambos verdes —, documentou a
lacuna de revisão que faltava (`## Review Triage Log` na spec da 2.6, com os quatro patches e os sete
achados adiados) e só então abriu PR. **A licença "isole a investigação em subagentes" do `bmad-build`
(step 2) não é segura sem escopo redundante:** um fork herda a conversa inteira, inclusive as
instruções dos passos seguintes do workflow, e pode segui-las em vez da instrução específica do
turno. Peça investigação em sessão nova sem o workflow completo no contexto, ou revise o commit antes
de confiar nele — nunca as duas coisas de uma vez.

A 1.9 fechou a Épica 1 sem acrescentar uma linha de código: o clone limpo a frio chega ao primeiro Produto
em **3 min 26 s**, o esqueleto inteiro anda dentro de uma rede sem saída, o portão do AD-12 foi exercitado
derrubando um `npm run build` de verdade, e as cinco verificações do passo 0 estão fechadas com desfecho
nomeado. O `docker-compose.offline.yml` foi escrito e **descartado** — com `internal: true` o Docker
descarta a publicação de 3000 e 8080 em silêncio —, e o ensaio de apresentação continua sendo desligar a
rede à mão; o motivo está no `addendum.md` §10 para ninguém reescrever o mesmo arquivo.
Continua aberta, esperando quem a pegue, a sobreposição de e-mail entre Comprador e Administrador, que é da
estória de autenticação — está em `_bmad-output/implementation-artifacts/deferred-work.md`, junto com os dois
buracos de verificação que a 1.8 registrou (a releitura travada de `Desde` e a ausência de bancada de teste
para a tela), com o achado da 1.9: o único link de Produto da casca leva a um Produto da faixa que o
Provedor Simulado recusa, então o caminho óbvio do README termina parado em `AGUARDANDO_PAGAMENTO` — a Épica 3
o remove ao entregar a busca —, e com os sete achados da 2.6 (a UJ-1 não percorrida só pelo teclado nesta
máquina, sem extensão do Chrome conectada; duas telas novas sem estado de carregamento; `GET
/api/v1/pedidos` sem `LIMIT`; os quatro arquivos novos do `web/` sem teste automatizado; `email` agora
dentro da Sessão gravada no Redis sem decisão escrita sobre dado pessoal em repouso; Sessão antiga
desserializando com `email` vazio; e o lampejo do estado "sem Sessão" no `MenuDaConta` antes da primeira
resposta). As sete épicas e as 52 estórias estão em `_bmad-output/planning-artifacts/epics.md`, e o
rastreamento de sprint em `_bmad-output/implementation-artifacts/sprint-status.yaml`.

## Quanto custa uma estória, e o que fazer com isso

A 1.1 consumiu ~795.000 tokens de subagente. **As três lentes de revisão adversarial do `bmad-build`, mais a
rodada de patches que elas geram, foram 56% disso** — e escalam com o tamanho do diff, não com a dificuldade da
estória. Sobram 51 estórias; nesse ritmo o orçamento não fecha.

A 1.1 foi o pior caso — greenfield, a maior estória do épico, e 14 lacunas de documentação que precisaram virar
decisão nomeada (estão no addendum §10). Mas a parte cara é justamente a que se repete. Três medidas:

- **Três lentes só onde há raio de alcance real:** reserva de Estoque sob concorrência (5-6, 5-7), máquina de
  estados do Pedido (5-1), webhook e idempotência (5-9). Para CRUD de mesma forma, uma lente basta.
- **Agrupar estórias de mesma forma numa spec só.** 3-1, 3-2 e 3-3 são o mesmo CRUD três vezes.
- **Limpar o contexto entre estórias.** O `epic-N-context.md` e a spec em disco bastam para retomar.

A spec rodou em 2026-09-06 e adotou os seis documentos como companheiros, sem absorver nenhum; as épicas
rodaram em 2026-09-07 e citam `CAP`, `FR`, `NFR`, `AD` e `UX-DR` por ID, sem copiar enunciado.

E antes de alargar qualquer camada: o **passo 0** do addendum §8. Um Produto, um Comprador, um Pedido que nasce, é pago pelo Provedor Simulado e chega a `ENTREGUE`, com telas feias. É onde as cinco verificações acima acontecem.

## O que uma sessão nova erraria

- **Disciplina de glossário.** O §3 do PRD define 20 termos usados literalmente no documento inteiro. Trocar por sinônimo é defeito, não estilo — nunca "compra" no lugar de **Pedido**, nunca "item" no lugar de **Produto**. Duas revisões já caçaram violações disso.
- **Começar horizontal.** O addendum §8 tem um **passo 0**: esqueleto vertical (um Produto, um Comprador, um Pedido até `ENTREGUE`, telas feias) antes de alargar qualquer camada. A ordem por dependência que vem depois é correta e perigosa se seguida sozinha — produz um sistema que só funciona no fim do semestre.
- **Cortar o que é invisível.** FR-19, FR-24, FR-34 e o teste de concorrência do NFR-7 não aparecem em tela nenhuma e são o que separa o sistema da maquete. A ordem de corte legítima está no §6.3.
- **Renumerar suposições.** O §16 foi reordenado por risco; referências a "suposição N" já quebraram uma vez por causa disso.
- **Achar que o addendum é rascunho.** Ele está `final`, mas a SM-7 exige que continue vivo: toda decisão de arquitetura tomada durante a construção volta para ele.
- **Dar ao Administrador um botão para forçar a recusa do pagamento.** Não existe superfície de operador para isso. O Provedor Simulado decide pelos **centavos do total do Pedido** (§7.1 do PRD, addendum §4): o apresentador provoca o desfecho que quer escolhendo Produto e quantidade, sem reconfigurar nada entre os roteiros. A faixa vale para a **primeira** Tentativa de Pagamento; da segunda em diante aprova, senão a nova tentativa da FR-27 não teria o que exercitar.
- **Construir o checkout em quatro passos.** Resolvido em 2026-09-05: o §9 diz **Endereço → Revisão**, e Frete e pagamento são blocos da Revisão — nenhum dos dois tem dado a pedir. Se você leu quatro passos em algum lugar, era uma cópia velha.
- **Buscar fonte na rede.** O NFR-15 exige percorrer o Roteiro A com a rede desconectada. Isso proíbe Google Fonts, `@import` remoto e CDN — a fonte é auto-hospedada no repositório. É o erro mais fácil de cometer e quebra a demonstração em silêncio, na sala.
- **Somar as reservas antes de segurar a trava.** É o defeito que vende a mesma última unidade duas vezes, e o texto da arquitetura só passou a impedi-lo depois que uma lente de concorrência mostrou o entrelaçamento. São **dois comandos, nesta ordem**: `SELECT … FROM catalogo.produto WHERE id = ANY($1) ORDER BY id FOR UPDATE`, e **só então** a soma das reservas ativas. O `ORDER BY id` não é estilo — sem ele, dois Pedidos com os mesmos dois Produtos em ordem inversa travam um no outro (`AD-5`, `AD-4`).
- **Pôr a emissão da confirmação simulada dentro de `pedido`.** Emitir é comportamento do Provedor e mora em `pagamento` (`AD-6`). Dentro do Pedido, põe conhecimento de gateway no domínio e quebra o NFR-3 e a SM-5 na primeira troca — que é justamente o que a banca vai perguntar.
- **Tratar "sem reserva ativa" como erro.** `catalogo.Liberar` sem reserva ativa devolve `nil`, `Consolidar` sobre reserva já consolidada é no-op, e `Reservar` com lista vazia é no-op. `pedido` chama `Liberar` **incondicionalmente** em toda transição para `CANCELADO` e em toda recusa. Sem isso, cancelar um Pedido em `PAGAMENTO_RECUSADO` falha sempre, com rollback.
- **Registrar `GET /api/v1/produtos` em dois lugares.** A rota é de `busca`, com ou sem `termo` — a Vitrine é uma listagem, não uma rota à parte. `catalogo` serve `GET /api/v1/produtos/<id>` e o CRUD administrativo, e não registra handler de listagem. Duas equipes registrando o mesmo padrão no `ServeMux` fazem o binário **entrar em pânico no arranque**: falha de subida, não de comportamento (`AD-16`).
- **Fixar o Next.js na linha 16.2.** O patch das duas RCE críticas de 25/08/2026 pousou em **16.3.3**, e a 16.2.x não recebe backport. A espinha fixa **16.3.4**. Vale para toda a tabela Stack: as versões foram verificadas na web, não lembradas.
- **Usar o verde ou o laranja para enfeitar.** No `DESIGN.md` os dois têm sentido fechado: verde é Estoque disponível e `ENTREGUE`; laranja aparece **uma vez por fluxo**, no passo irreversível. Usar qualquer um decorativamente apaga o único mecanismo semântico de cor do sistema.
- **Rodar `bmad-sprint-planning` direto no `epics.md`.** O parser do `sprint_plan.py` só reconhece `## Epic N:` e `### Story N.M:` em inglês — os nossos cabeçalhos são `## Épica N:` e `### Estória N.M:`, e ele devolve zero épicas **sem um único aviso**. Gere uma cópia temporária antes de chamar o script, e passe a cópia no `--epic-file`:
  ```
  sed -E 's/^(#{1,3}) Épica /\1 Epic /; s/^(#{2,4}) Estória /\1 Story /' \\
    _bmad-output/planning-artifacts/epics.md > /tmp/epics-en.md
  ```
  Só a palavra estrutural muda: as chaves do `sprint-status.yaml` continuam saindo do título em português.
- **Editar a `SPEC.md` à mão.** Ela é **derivada** do `.memlog.md` da spec a cada execução, e `bmad-spec` é a única escritora — uma edição manual é sobrescrita no próximo derive, em silêncio. Mudou algo? Rode `bmad-spec` de novo apontando para a mesma pasta: os `CAP` são preservados por ID. O mesmo vale para o `mapa-de-capacidades.md`.
- **Delegar o passo 2 do `bmad-build` (investigação) a um subagente sem escopo redundante.** Um fork/subagent herda a conversa inteira, inclusive as instruções dos passos seguintes do próprio workflow — e pode segui-las em vez da instrução específica do turno ("só investigue, não escreva código"). Foi o que aconteceu na 2.6: o subagente de investigação implementou a estória inteira e commitou em `main` sem passar pelo CHECKPOINT 1 nem por nenhum outro HALT. O commit acabou correto depois de revisão e verificação independentes (`go build`/`go vet`/`go test ./...` e o build Docker do `web/`, rodados de novo fora do relato do subagente), mas por sorte de execução, não por garantia do processo. Peça investigação numa sessão nova sem o workflow completo no contexto, ou revise o commit resultante linha por linha antes de confiar nele — nunca trate a delegação como segura por padrão.
- **Achar que Go, sqlc e Next rodam direto nesta máquina Windows.** Não rodam: `go` não está no `PATH` e o `web/`
  não tem o binário `next` instalado. O que funcionou na 4.2/4.3: `go vet` e `go test` na imagem
  `azamon-dev:latest`, com `-v azamon-gocache:/root/.cache`, `-v //var/run/docker.sock:/var/run/docker.sock` e
  `-e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal -e TESTCONTAINERS_RYUK_DISABLED=true` (o testcontainers
  precisa do socket); `sqlc generate` na imagem `sqlc/sqlc:1.31.1`; e o build do `web/` por `docker build ./web`.
  Dois efeitos colaterais: o sqlc reescreve **todos** os `gerado/` com LF por cima do CRLF do checkout, então
  restaure com `git checkout` os módulos que você não mexeu (`git diff --ignore-cr-at-eol --stat` mostra o que é
  real); e `gofmt -l` lista todo arquivo CRLF, mudado ou não — confira com `tr -d '\r' < arquivo | gofmt -l`.

## Para colar numa sessão nova

```
Projeto Azamon: réplica da Amazon, trabalho de faculdade.
Go 1.27 + Postgres 18 + Redis 8 + Docker, front em Next.js 16 sobre React 19.
PRD, UX, arquitetura e spec estão finalizados e reconciliados.
Leia HANDOFF.md primeiro. O contrato é _bmad-output/specs/spec-azamon/SPEC.md,
com 8 CAPs de ID estável e o companions: que lista o resto — inclusive a
ARCHITECTURE-SPINE.md, com 20 ADs de ID estável.
Épicas 1 e 2 fechadas; Épicas 3 e 4 em main, todas em review por falta do passeio no navegador.
Próximo passo: Estória 5.1 (a máquina de estados do Pedido completa e com testes).
```

*Este arquivo não é carregado automaticamente por agentes — o `AGENTS.md` da raiz é. Ele carrega as armadilhas de maior consequência e aponta para cá; as de escopo estreito, como não renumerar as suposições do §16, vivem só aqui. Depois de mudança significativa, refresque com `bmad-project-context`.*
