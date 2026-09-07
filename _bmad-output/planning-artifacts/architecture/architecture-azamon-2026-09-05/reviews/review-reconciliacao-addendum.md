---
title: Revisão de reconciliação — ARCHITECTURE-SPINE × addendum técnico
tipo: review-reconciliacao
alvo: _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md
insumo: _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md
apoio: _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/.memlog.md (94 entradas)
created: 2026-09-05
---

# Revisão de reconciliação — a espinha honra o addendum?

## Veredito

A espinha honra o addendum em **estrutura** e falha em **conteúdo de máquina de estados**. Paradigma, fronteiras de módulo, porta de pagamento, blindagens do §6 e cinco dos seis itens delegados pelo §9 estão decididos, alguns melhor do que o addendum pedia. Mas a tabela de nove transições do §2 — o pedaço que o addendum mais detalhou, e o único lugar onde ele fala em "o bug mais caro deste sistema" — chega à espinha com **um** dos nove efeitos colaterais materializado. As sete invariantes têm AD nomeado, duas com lacuna material que as reduz a menos do que o addendum exige.

**Placar:** 1 crítico · 5 altos · 4 médios · 6 baixos.

| Severidade | IDs |
|---|---|
| CRÍTICO | C-1 |
| ALTO | A-1, A-2, A-3, A-4, A-5 |
| MÉDIO | M-1, M-2, M-3, M-4 |
| BAIXO | B-1, B-2, B-3, B-4, B-5, B-6 |

---

## 1. As sete invariantes do §2

| # | Invariante | AD que a cobre | Situação |
|---|---|---|---|
| 1 | A transição é o único ponto de mutação do Pedido | **AD-3** | Honrada. Literal: "escrito uma vez, na criação, e nunca mais… Não existe `UPDATE pedido SET status`". Ver A-5 para o efeito colateral dessa rigidez. |
| 2 | Transição inválida falha alto | **AD-3 + AD-14** | **Parcial — ver A-1.** O erro nomeado existe (`pedido.ErrTransicaoInvalida`, "erro engolido é proibido"), mas o que a espinha descreve é o *compare-and-swap* da invariante 5, não a validação do conjunto de arestas. |
| 3 | Toda transição escreve no histórico | **AD-3 + AD-15** | Honrada, e ampliada: AD-15 acrescenta a correlação, que o addendum não pedia, e amarra uma fonte a três usos (linha do tempo da FR-30, varredura do AD-6, NFR-9). Falta uma coluna — ver A-5. |
| 4 | Efeito sobre Estoque na mesma transação | **AD-4** | Honrada, literal: "Efeito sobre Estoque e transição de Status acontecem na mesma transação, sempre." A atomicidade está garantida; *qual* efeito, não — ver C-1. |
| 5 | Concorrência Administrador × simulação por estado esperado | **AD-3** | Honrada, com citação explícita ao addendum. Reforçada por `FOR UPDATE SKIP LOCKED` no AD-6. |
| 6 | Nenhum estado alcançado por tempo em memória | **AD-6** | Honrada, e é o melhor AD do documento. Proíbe nominalmente `time.Timer`, `time.After` e agendamento em memória, deriva do histórico, exige idempotência, e faz um mecanismo atender quatro requisitos. Ver A-3 para o furo que sobra. |
| 7 | Confirmação tardia não ressuscita Pedido | **AD-7** | **Parcial — ver A-2.** "Registrada e sinalizada, nunca aplicada" está escrito; o mecanismo do AD-7 não tem estado para representar "recebida e não aplicável". |

**Resumo:** 7 de 7 com AD nomeado; **5 plenamente honradas, 2 com lacuna material (invariantes 2 e 7).** Nenhuma invariante ficou órfã — não há achado crítico neste item.

---

## 2. A tabela de transições do §2

### C-1 · CRÍTICO — a espinha materializa 1 dos 9 efeitos colaterais obrigatórios

O addendum traz nove linhas de transição, cada uma com efeito colateral declarado. A espinha ancora **um**: AD-5 diz que `Consolidar` "roda uma vez, na transição `SEPARANDO → ENVIADO`". Os outros cinco pontos de `Liberar` e o segundo ponto de `Reservar` são nomeados como operações existentes e nunca amarrados a transição nenhuma.

| Transição do addendum | Efeito obrigatório | Onde está na espinha |
|---|---|---|
| (criação) → `AGUARDANDO_PAGAMENTO` | Cria Reserva, tudo ou nada | AD-5 ("tudo-ou-nada sobre todos os itens (FR-24)") — **coberto** |
| `AGUARDANDO_PAGAMENTO` → `PAGO` | Mantém a Reserva | Implícito por omissão (só `Consolidar` mexe) — aceitável |
| `AGUARDANDO_PAGAMENTO` → `PAGAMENTO_RECUSADO` | **Libera** a Reserva | **ausente** |
| `PAGAMENTO_RECUSADO` → `AGUARDANDO_PAGAMENTO` | **Cria nova Reserva; falha se indisponível** | **ausente** |
| `PAGO` → `SEPARANDO` | Nenhum | ok |
| `SEPARANDO` → `ENVIADO` | **Consolida** | AD-5 — **coberto** |
| `ENVIADO` → `ENTREGUE` | Nenhum | ok |
| `AGUARDANDO_PAGAMENTO` → `PAGAMENTO_RECUSADO` (expiração, FR-34) | **Libera** a Reserva | AD-6 detecta o vencimento; o efeito sobre Estoque **não é declarado** |
| {`AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO`, `SEPARANDO`} → `CANCELADO` | **Libera** a Reserva, **se ativa** | AD-4 cita o bug ("cancelar um Pedido e não devolver ao Estoque") como o que previne, mas não declara a regra |

O AD-4 garante que o efeito e a transição sejam atômicos. Ele não diz **qual** efeito, e a espinha é o documento que a construção vai ler. A assimetria é o defeito: quem escreveu se deu ao trabalho de fixar o ponto de `Consolidar` — o mais sutil dos nove, e o que o memlog 61 fechou como achado crítico do PRD — e deixou os cinco `Liberar` implícitos, que são justamente os que uma implementação apressada esquece.

Há ainda um detalhe que só a leitura conjunta revela: cancelar a partir de `PAGAMENTO_RECUSADO` acontece com a Reserva **já liberada** (o addendum escreve "se ativa"). `catalogo.Liberar` precisa ser tolerante à ausência, ou o cancelamento legítimo do FR-31 vira erro. A espinha não diz.

**Correção:** uma tabela de nove linhas dentro do AD-3 ou do AD-5, transição → efeito sobre a Reserva, e uma frase dizendo que `Liberar` sobre Reserva inexistente é no-op.

### A-1 · ALTO — `Transicionar` implementa a invariante 5, não a invariante 2

A assinatura declarada é `pedido.Transicionar(ctx, tx, pedidoID, esperado, novo, ator)`. A regra que a acompanha é: "Transição com origem diferente de `esperado` devolve erro nomeado". Isso é o compare-and-swap que resolve a corrida da invariante 5 — e a espinha diz isso com todas as letras.

Mas `novo` vem do chamador **sem nenhuma regra declarada sobre quais pares são válidos**. Um chamador que passe `esperado=AGUARDANDO_PAGAMENTO, novo=ENTREGUE` passa no CAS. A espinha não enumera as arestas, não diz onde elas vivem, e não diz que `Transicionar` as valida. A invariante 2 exige que origem inválida falhe alto "seja o chamador o Administrador, o Comprador ou o processo de simulação" — o que a espinha entrega é que origem *desatualizada* falha alto.

Mitigação parcial: o AD-3 faz *bind* a FR-26..FR-34, o que inclui a FR-28 e portanto o diagrama de estados do PRD. Mas *bind* não é regra.

**Correção:** uma frase no AD-3 — o conjunto de arestas válidas é dado, declarado em `internal/pedido`, e `Transicionar` recusa qualquer par fora dele com o mesmo erro sentinela.

### B-5 · BAIXO — cancelar Pedido já `CANCELADO`

A FR-31 exige que cancelar um Pedido já cancelado "não produz efeito nem erro de servidor". O caminho da espinha (AD-3 → `ErrTransicaoInvalida` → AD-14 → envelope de erro) produz um 4xx, não um 5xx, então não há violação literal. Vale uma linha explícita para que ninguém resolva isso com um `if` fora do `Transicionar` e reabra o segundo caminho de escrita que o AD-3 fecha.

---

## 3. §3 — Reserva de Estoque e concorrência

O ciclo de vida bate **nos dois extremos e no meio declarado**, e fica solto nos pontos de morte:

| Exigência do §3 | Na espinha |
|---|---|
| Disponível é derivado, não contador mutável | AD-5, literal: "Estoque disponível nunca é armazenado — é `estoque_total − Σ reservas ativas`, calculado na consulta" ✓ |
| Nasce com o Pedido | AD-5 ✓ |
| Termina no envio, consolidando com baixa do total | AD-5 ✓ — e no ponto exato que o memlog 61 fixou |
| Termina no cancelamento, na recusa e na expiração | **Ver C-1** ✗ |
| Nunca ativa indefinidamente | Consequência do acima; sem os pontos de morte declarados, não é garantida ✗ |
| Decisão dentro de transação, com bloqueio de linha por Produto | AD-5 (`SELECT … FOR UPDATE` por linha de Produto) + AD-4 ✓ |
| Atômica sobre todos os itens | AD-5 ✓ |
| Teste de concorrência do NFR-7 como critério de aceite | **M-3** ✗ |

Dono correto: AD-5 põe a Reserva em `catalogo`, `pedido` nunca toca tabela de Estoque, e a travessia `reserva_estoque.pedido_id` é `catalogo → pedido` sem FK, congelando nada. Isso é fiel ao §3 e ao AD-2 ao mesmo tempo. O comentário `ponytail:` do addendum §3 foi preservado literalmente no AD-5, com o mesmo caminho de escape (reserva otimista com repetição, não fila) — sinal de reconciliação bem feita.

### M-3 · MÉDIO — o teste de concorrência do NFR-7 não aparece como entregável

O addendum §3 encerra dizendo que "o teste de concorrência do NFR-7 é o critério de aceite" do desenho, e o §6.3 do PRD o lista entre os quatro itens que nunca se corta. A espinha cita NFR-7 nos *binds* do AD-5 e monta o ambiente que o viabiliza (testcontainers-go, Postgres efêmero), mas em nenhum lugar o nomeia. É o único critério de aceite que o addendum atribuiu explicitamente à arquitetura.

**Correção:** uma linha nos Ambientes ou no AD-5 — "N requisições paralelas sobre a última unidade, `pedidos_criados == estoque_inicial`, roda em `go test ./...`".

### B-2 (parte) · BAIXO — a guarda da FR-11

O §3 menciona que o disponível travado faria "a guarda da FR-11 travar o Administrador" — a regra de que o Estoque total não desce abaixo das reservas ativas. AD-5 faz *bind* a FR-11 e não repete a guarda. Herdada do PRD, mas é a mesma família de omissão do C-1.

---

## 4. §4 — Fronteira do Provedor de Pagamento

Os **dois casos que o mock precisa resolver** estão ambos endereçados, um plenamente e outro pela metade:

| Caso | Cobertura |
|---|---|
| **Confirmação que nunca chega** (FR-34) | **Coberto.** AD-6 lista "Tentativa de Pagamento vencida" entre as três decisões da varredura; AD-13 põe o prazo em configuração (15 min / 60 s em demonstração). A faixa `,95`–`,99` do §7.1 chega intacta ao AD-8. |
| **Confirmação depois do cancelamento** (FR-26) | **Coberto no texto, furado no mecanismo — A-2.** AD-7 diz "registrada e sinalizada, nunca aplicada"; o modelo não tem onde guardar esse estado. |

O resto do §4 está honrado com fidelidade notável: porta com duas operações e mais nada (AD-8); total como valor, adapter sem consultar `pedido` (AD-8); decisão pelos centavos conforme §7.1, primeira Tentativa apenas (AD-8); confirmação **fora da requisição do checkout**, pelo mesmo caminho HTTP que o Stripe usaria (AD-7 + o laço tracejado do diagrama de contêineres, deliberado e explicado); idempotência por restrição única em vez de código defensivo (AD-7). O "**nenhuma superfície de operador, nenhuma variável para forçar recusa**" do AD-8 fecha a armadilha nova que o memlog 92 registrou no HANDOFF.

### A-2 · ALTO — a invariante 7 não tem estado terminal

AD-7 monta o inbox assim: `pagamento` grava em `pagamento.confirmacao_recebida`, e a varredura de `pedido` lê `pagamento.ConfirmacoesNaoAplicadas` e aplica a transição. Para uma confirmação `APROVADO` sobre Pedido `CANCELADO`, a transição é inválida e o AD-3 devolve erro nomeado. E então?

- Se a confirmação continua "não aplicada", a varredura a relê **a cada segundo, para sempre** — e o AD-6 exige idempotência, que aqui vira laço quente.
- Se alguém a marca como "aplicada" para calar o laço, perde-se exatamente a informação que a invariante 7 proíbe perder: "a transição inválida da invariante 2 não pode virar perda silenciosa da informação de que o pagamento foi aprovado".

Falta um terceiro estado — recebida, não aplicável, sinalizada — e falta dizer como o painel do Administrador (FR-32) o lê, já que ele vive em `pedido` e o dado vive em `pagamento`, com AD-2 proibindo consulta cruzando schema. O grafo do AD-1 permite a composição em `api/`, mas AD-1 também diz que `api/` não tem regra.

**Correção:** nomear o estado no AD-7 (`confirmacao_recebida.situacao ∈ {pendente, aplicada, descartada_pedido_cancelado}`) e dizer quem compõe a sinalização do painel.

### A-3 · ALTO — ninguém emite a confirmação do Provedor Simulado

O addendum é inegociável sobre a confirmação chegar fora da requisição do checkout, "mesmo no simulado". A espinha honra o *caminho* (webhook) e não decide o *gatilho*. AD-6 enumera o que a varredura lê e decide — "confirmação a aplicar, Tentativa de Pagamento vencida, etapa de entrega devida" — e **nenhuma das três é "confirmação simulada devida a emitir"**.

Sobra um buraco na forma exata da invariante 6: sem um gatilho declarado, a implementação natural é `go func(){ time.Sleep(…); post() }()` ou `time.AfterFunc` dentro do `IniciarTentativa` — agendamento em memória, que o AD-6 proíbe e que reiniciar o contêiner apaga, deixando o Pedido preso até a expiração do FR-34 salvar o dia por acidente. É o cenário que o addendum classificou como "o requisito que mais provavelmente seria violado por uma implementação ingênua".

Agrava: `IniciarTentativa(ctx, tx, …)` recebe a transação do checkout. O agendamento precisa sobreviver ao commit, e um agendamento em memória não tem como.

**Correção:** uma quarta linha no AD-6 — a varredura também emite as confirmações simuladas devidas, derivando de `pagamento.tentativa_pagamento` a que ainda não confirmou e já passou do atraso simulado.

### B-1 · BAIXO — a assinatura não carrega o número da Tentativa

O §7.1 escopa a regra por centavos à **primeira** Tentativa de Pagamento (memlog 90, sem o qual o Roteiro B passo 4 é impossível). A assinatura do AD-8 é `IniciarTentativa(ctx, tx, pedidoID, totalCentavos)` — o adapter não recebe qual tentativa é, então precisa contar por conta própria. Funciona (as Tentativas vivem no próprio schema `pagamento`, sem violar "o adapter não consulta `pedido`"), mas põe no adapter — o objeto que amanhã vira Stripe — uma consulta que o módulo poderia passar de graça. Um `numeroDaTentativa int` na assinatura resolve.

---

## 5. §5 (Frete) e §6 (blindagens baratas)

### As duas blindagens do §6 — **ambas preservadas, e explicitamente**

- **Item de Pedido carrega o Vendedor** (Suposição 2). Tabela de referências cruzadas da espinha: `item_pedido.produto_id`, `pedido → catalogo`, congela "Nome, preço praticado **e Vendedor**, na criação". Fiel ao addendum e ao FR-23. ✓
- **Categoria nasce com pai opcional** (Suposição 4). ERD: `CATEGORIA ||--o| CATEGORIA : pai_opcional`. ✓

Este é o item mais bem reconciliado da revisão: as duas blindagens sobreviveram nominalmente à travessia PRD → addendum → espinha, cada uma no artefato certo (uma na tabela de travessias, outra no ERD), sem virar prosa.

**B-4 · BAIXO** — o addendum acrescenta "não preenchida no MVP e não exposta na interface". A espinha não repete a metade "não exposta". É restrição de produto, herdada do PRD, mas o custo de uma cláusula é zero e o risco de alguém desenhar a navegação hierárquica não é.

### M-1 · MÉDIO — a Regra de Frete não tem casa

O §5 do addendum faz uma única prescrição arquitetural: a tabela de faixas de CEP → região → valor fixo **"vive como dado configurável, não como `if` espalhado no checkout"**, com CEP fora de faixa caindo na região padrão. É proposta "a confirmar" — ou seja, endereçada à arquitetura.

Na espinha, a Regra de Frete aparece exatamente uma vez, no AD-9: "a **única** função que arredonda, e arredonda uma vez". Isso resolve o NFR-13 e não resolve o §5. Não há entidade de faixa ou região no ERD, nem menção na árvore de origem, nem AD. O limiar de isenção (R$ 299) está coberto pelo AD-13, por ser um dos 15 parâmetros do §7.1 — mas a tabela de faixas não é um escalar e não cabe em variável de ambiente. O mapa Funcionalidade → Arquitetura aloca §4.5 inteiro em `internal/pedido`, o que dá o *onde* por implicação; falta o *em que forma*, que é a única coisa que o addendum pediu.

**Correção:** uma linha no ERD (ou no Adiado, dizendo que a tabela nasce semeada em `db/semente/` junto ao Catálogo) e uma frase dizendo que a região padrão é a saída do CEP não mapeado.

### B-2 · BAIXO — dois configuráveis fora dos "15 parâmetros"

AD-13 ancora toda a configuração nos 15 parâmetros do §7.1 — contagem correta, confirmada contra o PRD. Mas dois itens configuráveis exigidos por requisito ficam fora dessa lista e, portanto, fora do AD-13: o **interruptor da simulação de entrega** (FR-33: "a simulação pode ser desligada por configuração, para que o Administrador conduza tudo na apresentação") e a **tabela de faixas de Frete** (M-1). Vale uma cláusula dizendo que a lista dos 15 é o piso, não o teto.

---

## 6. §8 — a espinha torna o passo 0 possível?

**Sim. Não obriga construção horizontal.** Verificação item a item:

- AD-1 fixa as setas entre módulos; não obriga construir os seis. O esqueleto vertical atravessa cinco e pula `busca`, que o próprio §8 chama de "o corte mais independente do sistema".
- AD-2 exige um schema por módulo — custo de cinco linhas de DDL.
- **AD-10 favorece ativamente o passo 0**: o rewrite de `/api/*` com origem única é a resposta antecipada a um dos dois riscos que o §8 nomeia ("o cookie de sessão não atravessa a porta"). AD-9 é a resposta ao outro ("o decimal virou ponto flutuante"). Os dois medos do passo 0 têm AD.
- O Adiado da espinha ajuda: "Esquema completo de colunas — o ERD fixa entidades e travessias; coluna é detalhe que o código passa a possuir".
- Atrito real, pequeno: a árvore de origem exige `db/consultas.sql` + `db/gerado/` por módulo (sqlc) e a stack fixa goose + testcontainers desde o arranque. É ferramental que se paga uma vez, antes de atravessar.

### B-3 · BAIXO — o passo 0 não é mencionado na espinha

Nada na espinha aponta para o §8 do addendum nem para o esqueleto vertical, e a espinha é o documento que a construção vai ler como substrato. Mitigado: `AGENTS.md` linha 33 e `HANDOFF.md` linha 62 já carregam o aviso, e o §14 do PRD tem "construção horizontal" como risco do pre-mortem. A recomendação é de uma linha, não de uma seção.

---

## 7. §9 — os seis itens delegados à arquitetura

| # | Delegado pelo addendum | Decidido? | Onde |
|---|---|---|---|
| 1 | Escolha entre Next.js e React | **Sim** | Stack (Next.js 16.2.7) + AD-10 ("é casca; nenhuma regra mora no Node"). Fecha o bloqueio do memlog 72. |
| 2 | Desenho de esquema de banco | **Sim**, no nível certo | AD-2 (schema por módulo, sem FK cruzando), ERD, tabela de travessias, convenções (uuidv7, `timestamptz`, `bigint _centavos`). Colunas no Adiado, com "nunca volta para cá". Lacuna: a entidade de Frete (M-1). |
| 3 | Estrutura de pastas do repositório | **Sim** | Árvore de origem completa, mais a forma interna repetida de cada módulo. |
| 4 | **Mecanismo de atualização da tela do Pedido** | **NÃO — A-4** | Ausente do documento inteiro. |
| 5 | Biblioteca de testes | **Sim** | `go test ./...` da stdlib + testcontainers-go v0.44.0 + "testes puros em memória" nos Ambientes. A escolha da stdlib é a decisão, e é a certa. Ver B-6. |
| 6 | Estratégia de migração de banco | **Sim** | Convenção de Migrações: goose, `db/migracoes/NNNN_<módulo>_<descrição>.sql`, `embed.FS`, aplicadas no arranque, "Nunca auto-migração". |

**5 de 6 decididos. Um ficou sem decisão: o item 4.**

### A-4 · ALTO — o mecanismo de atualização da tela do Pedido não foi decidido

O addendum delega explicitamente e até dá a resposta barata: "o mecanismo — polling curto, SSE ou WebSocket — é decisão de arquitetura; polling é o suficiente e o mais barato". A espinha não contém `polling`, `SSE`, `WebSocket` nem qualquer descrição de como a tela sai de `AGUARDANDO_PAGAMENTO` para `PAGO` sem recarregamento.

Não é lacuna cosmética. A FR-26 exige atualização sem recarga manual; o Roteiro A passo 7 do §12 é literalmente "mostrar o Pedido em processamento virar `PAGO` sem recarregar a página", e a SM-1 é o roteiro passar. Além disso, o AD-10 fecha portas em volta do assunto (proíbe Server Actions, Route Handlers com regra, e `fetch` do servidor Next para o Go dentro de RSC) sem dizer qual porta fica aberta — a leitura literal do AD-10 deixa o navegador com uma única saída, `/api`, o que já indica polling do cliente, mas por eliminação, não por decisão.

**Correção:** uma convenção de uma linha — polling do navegador em `GET /api/v1/pedidos/{id}` a cada N segundos enquanto o Status for não terminal, com N no AD-13.

### B-6 · BAIXO — nada sobre teste do lado `web/`

O item 5 do §9 fica coberto para o Go e vazio para o Next.js. Pode ser deliberado (o §12 cobre a casca por ensaio de roteiro, não por teste), mas a espinha não diz isso, e o Adiado — que é onde ela normalmente diz — não traz a linha.

---

## 8. Contradições

Não há contradição frontal: nenhum AD decide o contrário do que o addendum decidiu. Duas divergências de segunda ordem:

### A-5 · ALTO — o motivo da recusa não tem onde morar

O AD-3 fecha a mutação do Pedido com força ("só o Status do Pedido muda") e o AD-15 enumera as colunas de `pedido.transicao_status`: estado anterior, novo, ator, instante e correlação. **Nenhuma delas é o motivo.** Mas:

- FR-27: "O motivo da recusa fica visível no detalhe do Pedido."
- FR-34: "o Pedido transita para `PAGAMENTO_RECUSADO` **com motivo 'tempo esgotado'**" — e é isso que distingue a recusa do provedor da expiração, já que ambas chegam ao mesmo estado pelo mesmo caminho.
- §12, Roteiro B passo 2: "Mostrar o Pedido em `PAGAMENTO_RECUSADO` com o motivo."

Sem coluna de motivo, ou o detalhe do Pedido passa a compor com `pagamento` a cada leitura — contra a promessa do próprio ERD de que "`item_pedido` e `pedido` congelarem o que exibem é o que faz o Detalhe do Pedido (FR-30) se ler sem tocar em nenhum outro schema" — ou o motivo se perde. Correção de uma coluna: `transicao_status.motivo`, nula por padrão.

### M-2 · MÉDIO — "bloqueio" no Redis é o enfeite que o addendum mandou evitar

O addendum §1 é explícito: o Redis "precisa de um uso nomeado, senão é dependência decorativa", e propõe três usos, pedindo que a arquitetura diga em voz alta o que não sobreviver. A espinha:

- **Sessão e token** — adotado (convenção de Sessão, AD-13). ✓
- **Cache de listagens** — recusado **em voz alta**, no Adiado ("segunda cópia da verdade sem medição que a justifique"). ✓ Exatamente o que o addendum pediu.
- **Fila leve para pagamento e entrega** — substituído pela varredura do AD-6, decisão melhor, **mas não dita em voz alta** (M-4).

E aparece um quarto uso que ninguém pediu: o diagrama de contêineres rotula o Redis "sessão · token · **bloqueio**". Nenhum AD menciona bloqueio no Redis. Pior, os dois ADs que falam de bloqueio o põem no Postgres — AD-5 (`SELECT … FOR UPDATE`) e AD-6 (`FOR UPDATE SKIP LOCKED`). Um bloqueio distribuído no Redis ao lado desses seria uma segunda fonte de exclusão mútua sobre os mesmos dados, que é uma classe de bug, não uma capacidade.

**Correção:** apagar "bloqueio" do rótulo, ou nomear em um AD qual bloqueio é (candidato legítimo: o bloqueio de conta por tentativas de login da FR-2, onde a expiração nativa do Redis é o mecanismo certo — e aí o rótulo devia dizer isso).

### M-4 · MÉDIO — a fila do Redis morreu sem obituário

O addendum §1 propôs a fila leve como forma de fazer a Tentativa de Pagamento (FR-25) e a simulação de entrega (FR-33) rodarem fora da requisição sem broker adicional. O AD-6 resolve os dois com uma varredura no Postgres — decisão superior, porque o histórico de transições já é a fonte da verdade e a fila seria uma segunda. Mas o addendum pediu nominalmente que um uso descartado do Redis fosse declarado, e este não foi. Uma linha no Adiado, no mesmo tom da linha do cache, fecha.

---

## O que a espinha faz melhor do que o addendum pedia

Registrado porque uma revisão que só lista defeito calibra mal a próxima rodada.

- **AD-6 unifica quatro requisitos num mecanismo** (FR-25, FR-26, FR-27, FR-33, FR-34) e fecha a invariante 6 nominalmente, proibindo as três formas de temporizador em memória em vez de recomendar a alternativa.
- **AD-7 troca código defensivo por restrição única** — a idempotência do NFR-12 vira propriedade do banco, não disciplina do programador. E o webhook para si mesmo mantém a fronteira do §4 honesta, com o laço tracejado desenhado e explicado.
- **AD-8 fecha a armadilha do HANDOFF**: "nenhuma superfície de operador, nenhuma variável para forçar recusa" é a resposta direta à entrada 92 do memlog.
- **AD-15 amarra uma fonte a três usos** — linha do tempo da FR-30, varredura do AD-6, rastreabilidade do NFR-9 — em vez de três tabelas.
- **O comentário `ponytail:` do §3 atravessou íntegro**, com o mesmo teto e o mesmo caminho de upgrade.
- **O Adiado tem condição de revisita em todas as nove linhas**, não só justificativa.

---

## Ações, na ordem em que valem a pena

1. **C-1** — tabela de nove linhas transição → efeito sobre a Reserva, no AD-5. É o achado que sozinho justifica a revisão.
2. **A-3** — quarta linha no AD-6: a varredura emite as confirmações simuladas devidas.
3. **A-2** — estado terminal para a confirmação não aplicável, no AD-7, e quem compõe a sinalização do painel.
4. **A-4** — uma convenção de polling, com o intervalo no AD-13.
5. **A-5** — coluna `motivo` em `transicao_status`, no AD-15.
6. **A-1** — o conjunto de arestas válidas é dado, e `Transicionar` o valida.
7. **M-1, M-2, M-3, M-4** — Frete como dado semeado; apagar ou nomear o "bloqueio" do Redis; o teste do NFR-7 como entregável; a fila do Redis no Adiado.
8. Os seis baixos, se sobrar tempo. B-3 e B-4 custam uma linha cada.
