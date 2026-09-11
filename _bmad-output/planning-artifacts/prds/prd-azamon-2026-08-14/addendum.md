---
title: Azamon — Addendum Técnico do PRD
status: final
created: 2026-08-14
updated: 2026-09-11
---

# Addendum — Azamon

Este documento guarda o que o `prd.md` deliberadamente não carrega: escolhas de tecnologia, desenho de mecanismo, alternativas descartadas e o caminho de migração. É insumo direto para `bmad-architecture` — o PRD diz *o que o sistema faz*, este addendum diz *o que já foi decidido sobre como*.

**A arquitetura já rodou** (2026-09-06). A espinha está em `_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md`, com 20 invariantes de ID estável (`AD-1`..`AD-20`). Onde este addendum e a espinha falarem do mesmo assunto, **a espinha decide o mecanismo e este documento guarda o porquê**. O §9 abaixo registra onde cada delegação foi parar.

## 1. Stack imposta

| Camada | Escolha | Origem |
|---|---|---|
| Back-end | **Go** | Imposta pelo usuário |
| Front-end | **Next.js 16** sobre **React 19** | Decidido em `bmad-architecture` (2026-09-06); fecha a Questão em Aberto 1 do PRD |
| Banco de dados | **PostgreSQL** | Imposta |
| Cache / apoio | **Redis** | Imposta |
| Execução | **Docker obrigatório** para subir os serviços | Imposta |

**Observação sobre o Redis, levantada na discussão:** ele precisa de um uso nomeado, senão é dependência decorativa. O Carrinho **não** é um bom candidato — a decisão de carrinho só para autenticado (FR-16) e persistente entre Sessões (§4.4 do PRD) o coloca naturalmente no PostgreSQL, onde ele sobrevive a reinício e não expira sozinho. Três usos foram propostos aqui e levados à arquitetura. **Um sobreviveu, e é dito em voz alta que os outros dois não** — que era exatamente o combinado:

| Uso proposto | Veredito da arquitetura |
|---|---|
| Sessão, token de redefinição de senha (FR-3) e contador de bloqueio de login (FR-2) | **Sobreviveu, e é o emprego único do Redis.** Os três são dado com prazo de validade, que é o que o `TTL` nativo faz melhor que o PostgreSQL. Custo aceito e declarado: reiniciar o Redis encerra todas as Sessões (`AD-20`) |
| Cache de listagens da vitrine e de busca (NFR-4) | **Recusado.** Com 5.000 Produtos a folga vem do tamanho da tabela, e cache sem medição é uma segunda cópia da verdade para invalidar. Se o NFR-4 falhar em medição, o caminho é índice antes de cache |
| Fila leve para a Tentativa de Pagamento (FR-25) e a simulação de entrega (FR-33) | **Recusado.** A invariante 6 abaixo já obriga uma varredura do histórico de transições no PostgreSQL — que sobrevive a reinício, o que a fila não faz. Existindo a varredura obrigatória, a fila vira um segundo caminho para o mesmo efeito, com um teste a mais e uma corrida a mais (`AD-6`) |

O Redis fica no `docker-compose` com um emprego nomeado, não como enfeite.

**Aderência da stack ao destino declarado:** Go com monólito modular é particularmente favorável ao plano de microsserviços do §11 do PRD — pacote por domínio, interface pública explícita por módulo, e a extração vira mover pacote e substituir chamada de função por chamada de rede. A escolha de stack, feita antes deste PRD, já barateia a fase 2 técnica.

## 2. Máquina de estados do Pedido — desenho

A FR-28 do PRD define as transições. Aqui está o que a arquitetura precisa preservar:

**Estados e quem os provoca**

| Transição | Provocada por | Efeito colateral obrigatório |
|---|---|---|
| (criação) → `AGUARDANDO_PAGAMENTO` | Checkout do Comprador (FR-23) | Cria Reserva de Estoque, tudo ou nada (FR-24) |
| `AGUARDANDO_PAGAMENTO` → `PAGO` | Confirmação assíncrona do Provedor de Pagamento (FR-26) | Mantém a Reserva de Estoque |
| `AGUARDANDO_PAGAMENTO` → `PAGAMENTO_RECUSADO` | Confirmação assíncrona do Provedor de Pagamento (FR-27) | Libera a Reserva de Estoque |
| `PAGAMENTO_RECUSADO` → `AGUARDANDO_PAGAMENTO` | Nova Tentativa de Pagamento pelo Comprador (FR-27) | Cria nova Reserva de Estoque; falha se indisponível |
| `PAGO` → `SEPARANDO` | Administrador (FR-32) **ou** simulação de entrega (FR-33) | Nenhum sobre Estoque |
| `SEPARANDO` → `ENVIADO` | Administrador (FR-32) **ou** simulação de entrega (FR-33) | **Consolida a Reserva de Estoque:** o Estoque total baixa e a Reserva encerra (FR-28) |
| `ENVIADO` → `ENTREGUE` | Administrador (FR-32) **ou** simulação de entrega (FR-33) | Nenhum sobre Estoque |
| `AGUARDANDO_PAGAMENTO` → `PAGAMENTO_RECUSADO` | Expiração da Tentativa de Pagamento (FR-34) | Libera a Reserva de Estoque |
| Qualquer de {`AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO`, `SEPARANDO`} → `CANCELADO` | Comprador (FR-31) | Libera a Reserva de Estoque, se ativa |

**Invariantes que a implementação precisa garantir**

1. **A transição é o único ponto de mutação do Pedido.** Conteúdo do Pedido (Itens de Pedido, preços praticados, Endereço, Frete) é imutável após a criação. Isso é o que faz a FR-30 (preço praticado no detalhe) funcionar sem esforço.
2. **Transição inválida falha alto.** Não há caminho silencioso: origem inválida devolve erro nomeado, seja o chamador o Administrador, o Comprador ou o processo de simulação.
3. **Toda transição escreve no histórico** (estado anterior, novo, quem, quando) — é o que sustenta NFR-9 e a linha do tempo da FR-30.
4. **Efeito sobre Estoque acontece na mesma transação da transição.** Cancelar e devolver ao Estoque não podem divergir; um sem o outro é o bug mais caro deste sistema.
5. **Concorrência entre Administrador e simulação de entrega** (nota do PM em §4.6 do PRD) resolve-se por transição condicionada ao estado esperado — quem chega segundo perde e recebe a recusa da invariante 2.
6. **Nenhum estado é alcançado por tempo em memória.** Tanto a expiração da Tentativa de Pagamento (FR-34) quanto o avanço da simulação de entrega (FR-33) derivam do histórico de transições — "está neste estado desde quando" — e não de temporizador de processo. Reiniciar o sistema não pode congelar Pedido nenhum. Este é o requisito que mais provavelmente seria violado por uma implementação ingênua, e o que mais provavelmente quebraria a apresentação.
7. **Confirmação tardia não ressuscita Pedido.** Uma aprovação que chega depois do cancelamento é registrada na Tentativa de Pagamento e sinalizada ao Administrador, mas não altera o Status do Pedido (FR-26). A transição inválida da invariante 2 não pode virar perda silenciosa da informação de que o pagamento foi aprovado.

## 3. Reserva de Estoque e concorrência

NFR-7 exige que duas criações simultâneas de Pedido para a última unidade produzam exatamente um Pedido. O desenho a validar na arquitetura:

- Estoque disponível é derivado (`total − reservas ativas`), não um contador mutável espalhado. A Reserva tem vida curta e limitada: nasce com o Pedido e termina no envio (consolidando, com baixa do total), no cancelamento, na recusa ou na expiração. Nunca fica ativa indefinidamente — se ficasse, o disponível cairia para sempre e a guarda da FR-11 travaria o Administrador.
- A criação do Pedido decide sobre o Estoque **dentro de uma transação**, com bloqueio de linha por Produto ou verificação condicional que falhe sob conflito.
- A criação é atômica sobre todos os itens: se qualquer um faltar, nada é criado (FR-24).
- O teste de concorrência do NFR-7 é o critério de aceite — dispara N requisições paralelas e verifica `pedidos_criados == estoque_inicial`.

`ponytail:` bloqueio pessimista por linha de Produto é o suficiente para a escala deste projeto; se o volume da demonstração revelar contenção, o caminho é reserva otimista com repetição, não fila de pedidos.

## 4. Fronteira do Provedor de Pagamento

O que o NFR-3 e a SM-5 cobram na prática:

- Uma interface com as operações mínimas: **iniciar Tentativa de Pagamento** e **receber confirmação**.
- O Provedor Simulado implementa a interface e decide o resultado **pelos centavos do total do Pedido**, com as faixas declaradas no §7.1 do PRD — determinístico e acionável na demonstração sem reconfigurar nada e sem superfície de operador (FR-25), não aleatório. A faixa vale para a primeira Tentativa de Pagamento; da segunda em diante o resultado é `APROVADO`, senão a nova tentativa da FR-27 não existiria.
- A confirmação chega **fora da requisição do checkout**, mesmo no simulado. Esse é o ponto que a discussão marcou como inegociável: o gateway real confirma por fora, então o MVP também confirma por fora, ou a virada vira reescrita.
- A confirmação é idempotente por chave (NFR-12) — a mesma confirmação recebida duas vezes avança o Pedido uma vez só. É o comportamento exato que um webhook real exige.
- A tela do Pedido reflete o novo Status do Pedido sem recarregamento manual (FR-26). O mecanismo — polling curto, SSE ou WebSocket — é decisão de arquitetura; polling é o suficiente e o mais barato.

**Os dois casos que o mock precisa resolver porque o gateway real vai exigir**

- **Confirmação que nunca chega** (FR-34). Um pagamento assíncrono sem prazo de expiração deixa o Pedido preso e o Estoque comprometido indefinidamente. Com o Provedor Simulado, o sintoma é um Pedido travado; com o Stripe, é um Pedido travado *e* uma cobrança em estado indefinido. Resolver agora custa um prazo e uma varredura periódica; resolver depois custa descobrir isso em produção.
- **Confirmação que chega depois do cancelamento** (FR-26). É a corrida clássica de checkout. No mock, registrar e sinalizar basta. Na fase 2, este é exatamente o ponto onde entra o estorno automático — e o registro na Tentativa de Pagamento é o que torna esse estorno possível sem arqueologia de log.

**Na virada para o Stripe (fase 2):** troca-se a implementação e a configuração. O checkout, o Pedido, a Reserva de Estoque e a máquina de estados não são tocados. Se isso não for verdade quando chegar a hora, o NFR-3 foi violado em algum ponto do caminho.

## 5. Regra de Frete

Proposta a confirmar (Suposição 6 do PRD): tabela de faixas de CEP → região → valor fixo, com isenção acima de um limiar de subtotal. Vive como dado configurável, não como `if` espalhado no checkout. CEP fora de faixa cai na região padrão (FR-21).

## 6. Blindagens baratas contra suposições frágeis

A auditoria de suposições (§16 do PRD) encontrou duas de alto impacto cuja reversão fica barata **se** o modelo de dados nascer preparado. É o mesmo movimento do Vendedor sem portal: a estrutura comporta, a interface não expõe.

- **Pedido não dividido por Vendedor** (Suposição 2). O Item de Pedido já registra o Vendedor (FR-23). Se o enunciado do trabalho exigir separação por Vendedor, o caminho é agrupar na consulta e na exibição — nenhuma migração, nenhum remodelamento do Pedido. **Custo hoje: zero, já está no requisito.**
- **Categoria plana** (Suposição 4). A Categoria nasce com referência opcional a uma categoria-pai, não preenchida no MVP e não exposta na interface. Hierarquia depois é preencher o campo e ajustar a navegação. **Custo hoje: uma coluna nula.** Sem ela, é migração de dados num sistema que já tem Produtos e Pedidos.

Registrado aqui, e não no PRD, porque são decisões de modelo — o PRD mantém a capacidade como plana e a interface como plana.

## 7. Alternativas consideradas e descartadas

*Registrado porque a defesa do trabalho vale mais com o que foi rejeitado do que com o que foi escolhido.*

| Alternativa | Por que foi descartada | O que se perdeu |
|---|---|---|
| **Marketplace completo** (auto-cadastro, moderação de anúncio, repasse financeiro, painel de vendas) | Dobra o escopo — é o negócio inteiro, não uma funcionalidade. Repasse sem dinheiro real é ficção, e moderação sem volume é tela vazia | Fidelidade à Amazon na parte que mais a define. Mitigado pelo meio-termo: o Vendedor existe no modelo de dados e aparece na página de Produto |
| **Portal do Vendedor mínimo** (Vendedor entra e gerencia apenas os Produtos dele) | **Reexaminado — a rejeição original estava mal fundamentada.** Não é metade do projeto: as telas são as da FR-8 e FR-9, que já existem, e a diferença é um predicado de autorização que a FR-4 já obriga a testar. Fica fora do MVP por sequenciamento, não por custo | Nada estrutural. É o **candidato número um a ser puxado para dentro** se o cronograma segurar — e o único item de fase 2 que muda a natureza do sistema, não só a vitrine |
| **Loja única sem Vendedor** | O Catálogo viraria um CRUD e o sistema deixaria de ter qualquer traço de marketplace | Nada de esforço; muito de identidade do produto |
| **Gateway real em sandbox** (Stripe test mode) desde o MVP | **Conflita com o NFR-15:** a demonstração precisa rodar sem internet, e integração real exige rede. *(A justificativa antiga — "túnel público é risco de véspera" — não se sustenta mais: o Stripe CLI encaminha webhook para localhost sem ngrok. Substituída pela razão estrutural.)* | A demonstração de integração real. Compensado pelo NFR-3 e pela SM-5, que provam que a porta está aberta — e pelo fato de que FR-34, FR-26 e NFR-12 já implementam a parte difícil da integração |
| **Pagamento síncrono simulado** (`if` retornando aprovado na própria requisição) | Seria menos código hoje e reescrita do checkout amanhã | Nada que valha o preço |
| **Carrinho anônimo com fusão no login** | O custo não é a fusão — a FR-19 já revalida preço e disponibilidade, então fundir é somar quantidades respeitando o Estoque. O custo é o **estado anônimo com ciclo de vida próprio**: sessão de visitante, expiração, limpeza de carrinho órfão, tudo num sistema sem visitantes reais | Aderência ao comportamento canônico de comércio eletrônico. É a rejeição mais visível para quem abrir o sistema sem conta |
| **Elasticsearch no MVP** | Um serviço a mais no `docker-compose`, indexação a manter sincronizada, e o ganho é invisível com 200 Produtos | Relevância e tolerância a erro de digitação. Recuperável na fase 2 atrás da interface de busca |
| **Microsserviços desde o início** | Consome o semestre em orquestração e entrega serviços que não conversam | Demonstração de sistema distribuído — **e essa perda é real**, porque em trabalho de faculdade a fase 2 costuma nunca acontecer. Resposta: o marco opcional de extração de um serviço, no §11 do PRD |
| **Divisão do Pedido por Vendedor** | Só faz sentido com repasse financeiro, que não existe aqui | Fidelidade ao comportamento real de marketplace |

## 8. Ordem de construção sugerida

**Passo 0 — esqueleto vertical, antes de tudo o que vem abaixo.** A ordem a seguir é correta por dependência e perigosa por risco: seguida ao pé da letra, ela produz um sistema que só funciona no fim do semestre, e o fim do semestre é onde o tempo acaba. Antes de alargar qualquer camada, atravessar o sistema inteiro uma vez: **um** Produto semeado, **um** Comprador, **um** Pedido que nasce, é pago pelo Provedor Simulado e chega a `ENTREGUE` — com telas feias, sem busca, sem filtro, sem painel. Isso força a integração entre front e back na semana 2 ou 3, quando ainda há tempo de descobrir que o decimal virou ponto flutuante e que o cookie de sessão não atravessa a porta.

Fechado o esqueleto, a ordem abaixo governa o alargamento:

1. **Identidade e papéis** (FR-1 a FR-5) — tudo depende de saber quem está falando.
2. **Catálogo e Estoque** (FR-6 a FR-11) — sem Produto não há nada para buscar nem comprar.
3. **Máquina de estados do Pedido** (FR-28) — com testes, **antes** das telas que a consomem. É o pré-requisito de quatro funcionalidades e de dois roteiros de demonstração.
4. **Carrinho** (FR-16 a FR-19).
5. **Checkout e pagamento assíncrono** (FR-20 a FR-27) — junta Estoque, Pedido e Provedor de Pagamento.
6. **Busca e filtros** (FR-12 a FR-15) — paralelizável a partir do passo 2, é o corte mais independente do sistema.
7. **Pós-venda e operação** (FR-29 a FR-33) — colhe o que o passo 3 plantou.

Os passos 6 e 1–2 são as fronteiras mais limpas para dividir o time em paralelo.

## 9. O que este addendum delegou à arquitetura

Os seis itens abaixo foram deixados em aberto de propósito e resolvidos por `bmad-architecture` em 2026-09-06. O mecanismo mora na espinha; a linha aqui existe para que ninguém procure no lugar errado.

| Delegação | Onde foi parar |
|---|---|
| Escolha entre Next.js e React | **Next.js 16 sobre React 19**, como casca de apresentação sem regra de negócio, reescrevendo `/api/*` para o Go — uma origem só, sem CORS (`AD-10`) |
| Desenho de esquema de banco | **Um schema do PostgreSQL por módulo, chave estrangeira cruzando schema proibida** (`AD-2`). Entidades e travessias na Semente Estrutural da espinha; colunas vivem nas migrações |
| Estrutura de pastas do repositório | Árvore na Semente Estrutural: `internal/<módulo>/` com um arquivo público por módulo, `api/` só de tradução, `internal/plataforma/` para o transversal |
| Mecanismo de atualização da tela do Pedido | **Consulta em intervalo**, em três superfícies e mais nenhuma, com todo instante absoluto e vindo do servidor (`AD-18`) — como a `EXPERIENCE.md` já pedia. Sem WebSocket |
| Biblioteca de testes | Híbrido: máquina de estados em memória; `testcontainers-go` com PostgreSQL real onde a transação **é** o objeto do teste (NFR-7, NFR-12, FR-24) |
| Estratégia de migração de banco | **goose** com versionamento por *timestamp*, arquivos SQL embutidos por `embed.FS` e aplicados no arranque do binário, antes da semente. Nunca auto-migração |

**O que a arquitetura decidiu e este addendum não previa** — registrado aqui porque a SM-7 exige que este documento continue vivo:

- **O Catálogo é dono do Estoque, incluindo a Reserva de Estoque** (`AD-5`). A alternativa, o Pedido guardar a Reserva, cria ciclo: calcular o disponível exigiria o Catálogo perguntar ao Pedido, que já pergunta o preço ao Catálogo.
- **A ordem é metade da corretude do §3 acima.** Bloquear a linha de Produto com `ORDER BY id FOR UPDATE` **antes** de somar as reservas ativas, em dois comandos. Ler a soma antes de segurar a trava vende a mesma última unidade duas vezes, e o `ORDER BY` fecha o impasse entre dois Pedidos com os mesmos dois Produtos em ordem inversa.
- **A confirmação do pagamento entra por webhook e é aplicada por *inbox*** (`AD-7`): `pagamento` grava com restrição única e **não conhece `pedido`**; a varredura de `pedido` lê e aplica. Sem ciclo, com idempotência estrutural em vez de código defensivo, e sobrevivendo a reinício.
- **Emitir a confirmação simulada mora em `pagamento`, não em `pedido`** (`AD-6`). É comportamento do Provedor: dentro do Pedido, poria conhecimento de gateway no domínio e quebraria o NFR-3 na primeira troca.
- **A FR-3 não tem canal de entrega.** Não existe serviço de e-mail (§5 do PRD, e o NFR-15 proíbe rede), então o token vai para o log estruturado (`AD-20`). A FR-3 é a primeira da ordem de corte do §6.3; se sobreviver, o caminho é um contêiner Mailpit, que roda offline.

## 10. O que a construção decidiu

A SM-7 exige que este documento continue vivo: decisão de arquitetura tomada durante a construção volta para cá, na mesma estória que a tomou.

**Estória 1.1 — andaime do serviço** (2026-09-08):

- **`internal/plataforma` é aresta permitida para todos.** A tabela do `AD-1` não a lista porque ela não é um dos seis módulos do NFR-2: é a camada de config, log, correlação, transação e tradução de erro que a própria espinha define. O teste de fronteira (`internal/fronteira_test.go`) trata `plataforma` como universalmente importável e mantém todo o resto exaustivo — importar `internal/pedido/db/gerado` de fora de `pedido`, por exemplo, continua sendo falha de teste.
- **Convenção de nome das variáveis `AZAMON_*`.** Nenhuma fonte nomeava uma sequer, e a partir daqui outras estórias dependem delas: `AZAMON_<ÁREA>_<PARÂMETRO>`, duração no formato do Go (`15m`, `60s`, `168h`), valor monetário com sufixo `_CENTAVOS`, booleano `true`/`false`. As 23 estão no `.env` versionado. A regra do Provedor Simulado (`§7.1`) virou duas variáveis, `AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS` e `AZAMON_PROVEDOR_RECUSADO_ATE_CENTAVOS`, porque são duas faixas e a terceira é o complemento.
- **DSN e credenciais ficam no `.env` versionado.** Não existe produção (só demonstração e teste), então as credenciais de demonstração não são segredo e versioná-las é o que faz o `docker compose up` de clone limpo funcionar (NFR-1). Segredo real, se um dia existir, não entra no repositório — por isso `AZAMON_POSTGRES_DSN` e `AZAMON_REDIS_URL` são as duas únicas variáveis **sem padrão no código**: faltando, o processo não sobe.
- **`default_transaction_isolation` não é declarado no `docker-compose.yml`.** O padrão do PostgreSQL já é `READ COMMITTED`, que é o que o `AD-4` exige; declará-lo no compose criaria uma segunda fonte de verdade que nenhum teste vigia. A corretude do `AD-5` depende de bloqueio explícito, não do nível de isolamento.
- **A mensagem do envelope de erro é a do sentinela, nunca a do erro embrulhado.** Descoberto ao escrever o tradutor do `AD-14`: `err.Error()` de um `fmt.Errorf("reservar: %w", …)` levaria o contexto de desenvolvedor para dentro de um texto que a *Voice and Tone* governa. O tradutor usa o `Error()` do sentinela que casou, e o específico ("restam 2 unidades") viaja em `dados`.
- **O `//go:embed` das migrações mora em `db/embutido.go`, não em `internal/plataforma/migracao.go`.** A diretiva não atravessa `..` — só enxerga a árvore do próprio pacote. `plataforma.Migrar` recebe o `fs.FS` como parâmetro, o que de quebra torna o caso "migração quebrada derruba o arranque" testável sem banco.
- **O healthcheck do serviço `azamon` é o próprio binário, por `/azamon -saude`.** A imagem é `scratch` — sem shell e sem `curl` —, e a alternativa seria engordar a imagem só para o compose conseguir perguntar se o serviço está de pé. O subcomando faz um GET em `127.0.0.1:<porta>/api/v1/saude` e sai com 0 ou 1, o que dá ao `web` um `depends_on: service_healthy` de verdade e faz o endpoint de saúde ser usado por alguém, em vez de só existir.

**Estória 1.2 — casca do navegador** (2026-09-09):

- **Verificação 1 fechada, e o resultado é o bom: o `Set-Cookie` do Go atravessa o `rewrites()` do Next íntegro.** Provado por sonda temporária — `GET /api/v1/sonda-cookie` no `api/`, um cookie descartável que existe só para o `curl` ter o que atravessar. Pela porta 3000 o cabeçalho volta completo, `HttpOnly` e `SameSite=Lax` inclusive, e o `X-Correlation-Id` do Go vem junto. A saída declarada — *proxy* explícito no lugar do *rewrite* — **não precisa ser acionada**, e o cookie de Sessão da 1.5 pode contar com o caminho. A estória 1.5 remove `api/sonda_cookie.go` e o registro dele em `api/rotas.go` ao entregar a Sessão de verdade.
- **Verificação 3 fechada: `npx shadcn init` roda limpo em Next 16.3.4 + React 19.2.8.** A ressalva M1 de `reviews/review-versoes.md` — "com npm a resolução de peer deps pede flag" — **não se materializou**: o npm 11 que vem no Node 24.20.0 instalou os 15 componentes sem `--legacy-peer-deps` e sem `--force`. Um desvio real apareceu no lugar: o CLI 4.21 **não instala mais o Tailwind**, ele exige o Tailwind 4 já presente e aborta em "No Tailwind CSS configuration found". A ordem correta é `npm i tailwindcss @tailwindcss/postcss`, `postcss.config.mjs`, `@import "tailwindcss"` e só então `shadcn init`.
- **`Toast` não existe mais no registro do shadcn; o componente é `sonner`.** O CLI recusa `add toast` com "only available for Base UI projects". A lista de quinze da UX-DR1 fica intacta em número e em papel — `components/ui/sonner.tsx` é o Toast do Azamon —, e `button` entrou como décimo sexto arquivo porque `pagination` depende dele. O `sonner.tsx` gerado importa `next-themes`; sem `ThemeProvider` o `theme` cai em `"system"`, e aí **quem resolve o `prefers-color-scheme` é o próprio Sonner, por dentro** — fora da variante `dark` do Tailwind, que o `globals.css` já neutralizou. Ou seja: num sistema operacional em tema escuro, o Toast sairia escuro num aplicativo que declara não ter modo escuro. O arquivo fica **sem edição**, como a UX-DR1 manda; a correção é no ponto de montagem, e a primeira estória que montar o Toaster escreve `<Toaster theme="light" />`.
- **O `next.config.ts` precisa entrar no estágio de execução da imagem `web`.** O `rewrites()` é resolvido pelo `next start` em tempo de execução, não assado no `.next` da construção. O `Dockerfile` da 1.1 copiava só `package.json`, `node_modules`, `.next` e `public` — com isso a imagem subia, servia `/` e devolvia 404 do Next em `/api`, que é a falha mais cara de diagnosticar da estória inteira.
- **A armadilha do `middleware.ts` foi tratada por omissão.** Nada em 1.2 precisa de middleware, então o arquivo simplesmente não existe — e um arquivo que não existe não pode ser ignorado em silêncio. Quando alguma estória precisar, o nome é `proxy.ts`.
- **O verificador do AD-12 é `prebuild` no `package.json`, não um passo de CI.** Não existe CI no repositório e o rodapé "Adiado" da espinha não pede um. `web/scripts/verificar-offline.mjs` é Node puro, faz `grep` pelos quatro padrões em todo o `web/` fora de `node_modules`, `.next` e do próprio verificador, e sai diferente de zero nomeando arquivo, linha e padrão. Como `prebuild`, derruba junto `npm run build` e a construção da imagem `web` — que é exatamente o portão que o AD-12 pede, sem infraestrutura nova, e roda igual no Windows e no Alpine.
- **`Azamon Sans` é o Inter variável, subconjunto latino, renomeado no repositório.** Um WOFF2 de 48 KB em `web/public/fonts/`, licença OFL ao lado, faixa de peso 100–900 num arquivo só — o `[ASSUMPTION]` do `DESIGN.md` sobre o arquivo concreto está resolvido. A pilha de sistema segue declarada como alternativa no mesmo `--font-sans`.
- **Os três papéis tipográficos viraram `@utility` do Tailwind 4, não tokens `--text-*`.** `wordmark`, `preco` e `preco-centavos` precisam carregar peso, tracking e `font-variant-numeric` juntos, e a chave `--text-*` não comporta o último. Como `@utility`, os dois papéis monetários já nascem com `tabular-nums` embutido, o que tira do autor de cada tela a chance de esquecer o NFR-13.
- **Uma variável nova: `AZAMON_API_URL`, padrão `http://azamon:8080`.** Não está no `.env` porque o padrão já é o nome de serviço do compose; existe só para quem roda `next dev` fora do compose apontar o rewrite para `localhost`.

**Estória 1.3 — primeira migração e o Catálogo Semeado** (2026-09-11):

- **Verificação 2 fechada, e o resultado é o bom: o sqlc v1.31.1 analisa `DEFAULT uuidv7()` sem erro.** `sqlc generate` roda limpo sobre as migrações goose e escreve `internal/identidade/db/gerado/` e `internal/catalogo/db/gerado/`; a saída declarada — gerar o `uuid` no lado Go — **não precisa ser acionada**, e a chave primária continua `uuid DEFAULT uuidv7()` no banco. Com isso as cinco verificações de risco da épica ficam com quatro fechadas; sobra só o p95 ≤ 500 ms com 5.000 Produtos, que é a 1.4.
- **O recorte de schema por módulo no sqlc é um glob sobre o nome do arquivo de migração** (`db/migracoes/*_catalogo_*.sql`). É isto que torna o prefixo de módulo no nome da migração um mecanismo, e não decoração: um módulo nunca enxerga o schema do outro, que é a fronteira do `AD-2` um nível acima. Migração nova de `catalogo` entra no recorte sozinha, sem editar o `sqlc.yaml`.
- **`emit_exact_table_names: true` nos dois módulos.** A inflexão do sqlc trata os nomes como inglês e devolve `CatalogoCategorium` para `catalogo.categoria`. Como a convenção do `AD-2` já é `snake_case` singular, a inflexão só tem o que estragar. O `rename:` do sqlc não resolve — ele age sobre coluna, não sobre o nome do struct da tabela.
- **O marcador da semente é a decisão, não uma consulta prévia.** `INSERT INTO public.semente (versao) … ON CONFLICT DO NOTHING` roda dentro da mesma transação que aplica o SQL: zero linhas afetadas significa que outro binário já semeou, e a transação fecha sem escrever. Ler a tabela e depois decidir seria a corrida clássica, com dois binários lendo vazio no mesmo arranque. Resolve "aplicar uma vez" e "dois binários ao mesmo tempo" com um comando, sem trava separada — e o teste sobre `testcontainers` dispara os dois de fato, em paralelo.
- **`Semear` sem nenhum `.sql` é erro; `Migrar` sem nenhuma migração é caso normal.** A assimetria é deliberada: um embed quebrado que passasse por "catálogo vazio" gravaria o marcador e o arranque seguinte nem tentaria de novo. A migração vazia não tem esse efeito colateral.
- **Os `uuid` da semente são derivados por SHA-1 do nome (o formato do v5), não sorteados.** `DEFAULT uuidv7()` é o certo para dado nascido em execução e o errado para a semente: cada `docker compose down -v` daria identificador novo, e a condição de aceite pede exatamente o contrário. Derivar do nome dá determinismo sem que ninguém precise digitar 62 literais à mão, e o SQL versionado continua sendo só literais.
- **O Catálogo Semeado é gerado, e o gerador é o original.** `media/gerar.go` (com `//go:build ignore`, rodado por `go run media/gerar.go`) carrega a lista escrita à mão — 50 Produtos, 5 Categorias, 5 Vendedores, nomes, descrições e preços em português plausível — e escreve os dois artefatos determinísticos: um SVG por Produto em `media/` e `db/semente/001_catalogo_semeado.sql`. Gerador e saída ficam os dois versionados: a construção é offline e ninguém precisa do Go instalado para subir o projeto. Editar o SQL à mão é perder a edição no próximo `go run`.
- **As imagens dos Produtos são SVG placeholder desenhados no próprio repositório** — cor da Categoria como fundo, nome do Produto no meio, fonte da pilha do sistema. Nenhum byte vem da rede (`AD-12`), nenhum arquivo externo precisa chegar, e trocar placeholder por foto de verdade depois é substituir o arquivo em `media/` sem tocar no banco, porque a URL gravada no Produto não muda. As cores escolhidas evitam de propósito o verde e o laranja do `DESIGN.md`, que carregam significado.
- **A rota da imagem é `GET /api/v1/media/{arquivo}`, servida do `embed.FS`.** Fica sob `/api/v1` porque é o único prefixo que o `rewrites()` do Next atravessa, e é o que permite que a URL gravada no Produto seja **relativa**. Quem barra travessia de diretório é o `fs.ValidPath` de dentro do `ReadFile` do embed, que recusa qualquer caminho com `..` ou com `/` à frente — **não** o `{arquivo}` do `ServeMux`, que casa o caminho escapado e entrega o segmento já decodificado (`%2F` chega ao handler como `/`). Arquivo inexistente sai no envelope do `AD-14`, como qualquer rota. O pacote `media` entrou na tabela de arestas do `AD-1` como destino permitido de `api` e origem sem destino nenhum.
- **As cinco tabelas nascem sem coluna de data.** Nenhuma estória desta épica lê `criado_em`, e a semente determinística não ganha nada com um `now()` dentro. A convenção "toda data é `timestamptz`" continua valendo — só não tem, ainda, a quem se aplicar dentro de `identidade` e `catalogo`. Quem precisar de auditoria de linha acrescenta a coluna na estória que precisar dela.
- **E-mail normalizado é restrição de banco, não disciplina de chamador:** `email text NOT NULL UNIQUE CHECK (email = lower(email))`. Com o `CHECK`, o `UNIQUE` passa a valer sobre a forma normalizada de verdade, e nenhum caminho de escrita futuro consegue furar a regra esquecendo um `lower()`.
- **As credenciais de demonstração estão no `README.md`:** `comprador@azamon.test` / `azamon-comprador` e `admin@azamon.test` / `azamon-admin`. Os hashes são Argon2id no formato PHC com os parâmetros do `AD-9` (`m=19456`, `t=2`, `p=1`, sal de 16 bytes), literais no gerador porque a semente é determinística — o sal fixo é consequência do determinismo, não descuido. Não existe produção, e estas contas não são segredo.

**Estória 1.4 — a medição do NFR-4 com 5.000 Produtos** (2026-09-11):

- **Verificação 4 fechada, e o resultado é o bom — com uma surpresa útil.** Com 5.050 Produtos no banco (os 50 do Catálogo Semeado mais os 5.000 do conjunto de medição), a consulta com termo, Categoria e faixa de preço mede **p95 = 0,34 ms**, mediana 0,24 ms e máximo 0,56 ms em 100 execuções sob `EXPLAIN (ANALYZE, FORMAT JSON)`. O teto do `NFR-4` é 500 ms: a folga é de três ordens de grandeza. A decisão **"índice antes de serviço" está confirmada** — nem Elasticsearch, nem cache Redis para a busca, e o assunto não precisa ser reaberto nesta escala. As cinco verificações de risco da épica ficam todas fechadas.
- **A surpresa: nesta escala o planejador não usa o índice GIN `pg_trgm`.** Com o filtro de Categoria presente, o plano é `Bitmap Index Scan em produto_categoria_id_idx → Bitmap Heap Scan`, e o `LIKE` vira filtro sobre as ~1.010 linhas que voltam. Sem Categoria escolhida — a Vitrine com só o termo digitado —, o plano é `Seq Scan`, e ainda assim p95 = 1,33 ms. O GIN não é o que faz o número ficar verde; 5.000 linhas cabem em memória e varrer é barato. Ele fica como **seguro de crescimento**: o dia em que o Catálogo tiver volume para o planejador preferi-lo, o índice já está lá e a consulta não muda. Registrar isto importa porque o contrário — "o p95 passou porque o GIN está lá" — é a conclusão errada mais fácil de tirar da mesma medição.
- **Os dois índices de chave estrangeira adiados na 1.3 entraram nesta migração**, `produto_vendedor_id_idx` e `produto_categoria_id_idx` — e o segundo é justamente o que o planejador escolheu. O trabalho adiado não era higiene: era o índice que paga.
- **A normalização é na escrita, feita pelo Go** (`catalogo.Normalizar`, minúscula e sem diacrítico), e `media/gerar.go` passou a usar a mesma função em vez da tabela de acentos que mantinha por conta própria. O preenchimento das linhas que já existiam, dentro da migração, usa `translate()` com o mesmo mapa: `unaccent()` não é `IMMUTABLE`, depende do dicionário carregado e a extensão nem está instalada, o que é o motivo de o `AD-16` mandar normalizar fora do banco.
- **O conjunto de 5.000 é semente separada, com marcador próprio**, em `db/semente-grande/` e ligada só por `AZAMON_SEMENTE_GRANDE=true`. O catálogo da demonstração continua com exatamente 50 Produtos (`SM-C3`), e `public.semente` guarda as duas versões de forma independente — `plataforma.Semear` já recebia o `fs.FS` e o marcador como parâmetro, então o mecanismo não mudou.
- **Os 5.000 saem de 50 × 10 × 10, dentro do próprio SQL.** Cada Produto semeado ganha uma linha (`Prata`, `Bronze`, …) e uma série (`2016`…`2025`), por `CROSS JOIN` sobre `catalogo.produto` — o `SELECT` enxerga o instantâneo anterior ao `INSERT`, então só os 50 entram na multiplicação. A alternativa, gerar 5.000 `INSERT` literais pelo `media/gerar.go`, custaria ~1,6 MB de SQL versionado para o mesmo efeito. As palavras de linha e série são ASCII de propósito: `lower()` dá o mesmo resultado que `catalogo.Normalizar` daria, e `lower()` é `IMMUTABLE`.
- **A sonda de medição mora no teste, não no `sqlc`.** `db/medicao_nfr4_test.go` é quem sobe o Postgres, semeia os dois conjuntos, roda `ANALYZE` e mede. A busca de verdade, com paginação e rota, é da Épica 3 — gerar agora uma consulta `sqlc` que ninguém chama seria código morto nascido com a estória.
