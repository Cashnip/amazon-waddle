---
title: Azamon — Addendum Técnico do PRD
status: final
created: 2026-08-14
updated: 2026-08-16
---

# Addendum — Azamon

Este documento guarda o que o `prd.md` deliberadamente não carrega: escolhas de tecnologia, desenho de mecanismo, alternativas descartadas e o caminho de migração. É insumo direto para `bmad-architecture` — o PRD diz *o que o sistema faz*, este addendum diz *o que já foi decidido sobre como*.

## 1. Stack imposta

| Camada | Escolha | Origem |
|---|---|---|
| Back-end | **Go** | Imposta pelo usuário |
| Front-end | **Next.js ou React** — ainda em aberto | Imposta parcialmente (Questão em Aberto 1 do PRD) |
| Banco de dados | **PostgreSQL** | Imposta |
| Cache / apoio | **Redis** | Imposta |
| Execução | **Docker obrigatório** para subir os serviços | Imposta |

**Observação sobre o Redis, levantada na discussão:** ele precisa de um uso nomeado, senão é dependência decorativa. O Carrinho **não** é um bom candidato — a decisão de carrinho só para autenticado (FR-16) e persistente entre Sessões (§4.4 do PRD) o coloca naturalmente no PostgreSQL, onde ele sobrevive a reinício e não expira sozinho. Usos legítimos no MVP, a confirmar na arquitetura:

- Armazenamento de Sessão e do token de redefinição de senha (FR-3), onde a expiração nativa do Redis é o mecanismo certo.
- Cache de listagens da vitrine e de resultados de busca frequentes (apoia NFR-4).
- Fila leve ou canal para o processamento assíncrono da Tentativa de Pagamento (FR-25) e da simulação de entrega (FR-33), evitando um broker adicional no MVP.

Se nenhum desses usos sobreviver ao desenho da arquitetura, vale dizer isso em voz alta em vez de manter o Redis no `docker-compose` como enfeite.

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
- O Provedor Simulado implementa a interface e decide o resultado por configuração — regra determinística acionável na demonstração (FR-25), não aleatória.
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

## 9. O que este addendum não decide

Escolha entre Next.js e React; desenho de esquema de banco; estrutura de pastas do repositório; mecanismo de atualização da tela do Pedido; biblioteca de testes; estratégia de migração de banco. Tudo isso é `bmad-architecture`.
