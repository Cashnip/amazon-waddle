---
title: Revisão de concorrência e integridade de dados — ARCHITECTURE-SPINE
tipo: review-concorrencia
alvo: _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md
foco: AD-3, AD-4, AD-5, AD-6, AD-7, AD-8, AD-17, AD-18
apoio:
  - _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md §2, §3
  - _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md §7 (NFR-7, NFR-12, NFR-13)
created: 2026-09-05
---

# Revisão de concorrência — onde um erro de espinha vira bug irreproduzível

## Veredito

A espinha escolheu os mecanismos certos — bloqueio pessimista por linha, transição condicionada ao estado esperado, uma transação por caso de uso, restrição única para idempotência — e **não escreveu a metade que os torna corretos**. Em todos os casos o que falta é a mesma coisa: a *ordem*. Em que ordem o bloqueio é adquirido, em que ordem a soma é lida, em que ordem os quatro passos da varredura rodam, em que ordem o efeito e a transição são escritos. Mecanismo certo com ordem indefinida não é um sistema com um bug; é um sistema que às vezes funciona, e a diferença aparece na sala, sob concorrência, com a banca olhando.

O NFR-7 **não é garantido pelo texto atual**: nada obriga quem escreve `reserva_estoque` a segurar o bloqueio da linha de Produto, e nada obriga a soma das reservas a ser lida *depois* do bloqueio. As duas omissões, juntas, produzem venda a mais exatamente no teste que o §12 do PRD manda demonstrar.

**Placar:** 2 críticos · 5 altos · 3 médios · 4 baixos.

| Severidade | IDs |
|---|---|
| CRÍTICO | C-1 (o bloqueio não é o mutex da grandeza derivada) · C-2 (confirmação velha derruba a Tentativa viva) |
| ALTO | A-1 (deadlock sem ordem de bloqueio) · A-2 (`Idempotency-Key` sem resposta definida) · A-3 (Reserva sem ciclo de vida) · A-4 (emissão do webhook sem critério derivado) · A-5 (varredura sem granularidade nem ordem de passos) |
| MÉDIO | M-1 (`Transicionar` não é declarado *compare-and-swap*) · M-2 (`total_centavos` coluna ou derivação?) · M-3 (Frete da Revisão × Frete da criação) |
| BAIXO | B-1 (o único ponto de arredondamento não arredonda) · B-2 (`Disponivel` é conselho, não decisão) · B-3 (o erro errado no cancelamento perdido) · B-4 (`Idempotency-Key` não sobrevive ao recarregamento) |

---

## C-1 · CRÍTICO — a linha de Produto é o mutex de uma grandeza que mora em outra tabela, e a espinha não obriga ninguém a segurá-lo

**Onde.** AD-5: *"Estoque disponível nunca é armazenado: é `estoque_total − Σ reservas ativas`, calculado na consulta […] `Reservar` é tudo-ou-nada sobre todos os itens (FR-24), com `SELECT … FOR UPDATE` por linha de Produto."*

**O problema.** `SELECT … FOR UPDATE` sobre `catalogo.produto` bloqueia a linha de **Produto**. A grandeza que precisa de exclusão mútua — `Σ reservas ativas` — mora em `catalogo.reserva_estoque`, e **um `INSERT` nessa tabela não é bloqueado pelo bloqueio da linha de Produto**. O bloqueio só funciona como mutex se *todo* escritor de `reserva_estoque` o adquirir primeiro, por convenção. A espinha declara o bloqueio uma vez, dentro de `Reservar`, e nunca o declara como obrigação de todos os caminhos que decidem sobre disponível. Convenção não escrita não sobrevive à segunda pessoa que escreve código.

Pior: mesmo dentro de `Reservar`, a espinha não diz **quando** a soma é lida em relação ao bloqueio. As duas ordens possíveis têm desfechos opostos, e a errada é a mais natural de escrever — uma consulta só, bonita, com o `JOIN` e o `FOR UPDATE` juntos.

**Entrelaçamento 1 — soma no mesmo comando do bloqueio (venda a mais).**
`estoque_total = 1`, nenhuma reserva ativa. Isolamento READ COMMITTED (padrão do Postgres — ver A-1).

| # | T1 (checkout A) | T2 (checkout B) |
|---|---|---|
| 1 | `BEGIN` | `BEGIN` |
| 2 | executa a consulta única: `SELECT p.estoque_total − COALESCE(SUM(r.quantidade),0) … FROM produto p LEFT JOIN reserva_estoque r … WHERE p.id = $1 … FOR UPDATE OF p` — **o snapshot do comando é tomado agora**, antes de o bloqueio existir. Lê `disponivel = 1`. Segura a linha. | |
| 3 | | executa a **mesma** consulta. Snapshot do comando tomado agora: `reserva_estoque` está vazia (a de T1 nem existe ainda). Bloqueia esperando a linha de Produto. |
| 4 | `INSERT INTO reserva_estoque (…, quantidade=1)`; `INSERT INTO pedido …`; `COMMIT` | |
| 5 | | destrava. O re-teste do qual (`EvaluatePlanQual`) reavalia a condição contra a **nova versão da linha alvo** — `produto` — mas o subselect que soma `reserva_estoque` continua respondendo pelo snapshot do passo 3. Lê `disponivel = 1` outra vez. |
| 6 | | `INSERT INTO reserva_estoque (…, quantidade=1)`; `INSERT INTO pedido …`; `COMMIT` |

**Dois Pedidos, uma unidade.** O teste do NFR-7 falha, e falha de forma intermitente — passa em máquina lenta, quebra sob paralelismo real. É exatamente o passo 5 do Roteiro C.

**Entrelaçamento 2 — o Administrador fura a guarda da FR-11.**
`estoque_total = 5`, nenhuma reserva. FR-11: *"O Administrador não pode reduzir o Estoque total abaixo das Reservas de Estoque ativas."*

| # | T1 (checkout, 4 unidades) | T2 (Administrador ajusta para 2) |
|---|---|---|
| 1 | `BEGIN`; bloqueia a linha de Produto; reserva 4 | |
| 2 | | `BEGIN`; `UPDATE produto SET estoque_total = 2 WHERE id=$1 AND 2 >= (SELECT COALESCE(SUM(quantidade),0) FROM reserva_estoque WHERE produto_id=$1 AND ativa)` — bloqueia na linha |
| 3 | `COMMIT` | |
| 4 | | destrava; o subselect não enxerga a reserva de T1; a condição passa; `COMMIT` |

Resultado: `estoque_total = 2` com 4 unidades reservadas. **O disponível fica negativo** — a consequência que a FR-11 declara impossível ("o disponível nunca é negativo") — e a Vitrine, o Carrinho e o `Disponivel` do AD-5 passam a devolver `-2` ou a saturar em zero conforme quem escreveu o `COALESCE`. O Produto some da loja e ninguém sabe por quê.

**Aperto de Rule (AD-5).** Três frases, e o bloqueio vira mutex de fato:

> A linha de `catalogo.produto` é o **mutex do disponível daquele Produto**. Toda escrita que dependa do disponível — `Reservar` e o ajuste de `estoque_total` da FR-11 — adquire esse bloqueio **antes** de somar as reservas, e a soma é um **comando separado, posterior ao bloqueio**, nunca o mesmo `SELECT` que o adquire:
>
> 1. `SELECT id FROM catalogo.produto WHERE id = ANY($1) ORDER BY id FOR UPDATE`
> 2. só então: `estoque_total − Σ reservas ativas`, e a decisão
>
> Em READ COMMITTED cada comando toma snapshot novo, então o passo 2 enxerga tudo que foi confirmado enquanto o passo 1 esperava. Juntar os dois num comando só reintroduz a venda a mais (o snapshot antecede o bloqueio) e **é a forma que o Postgres nem aceita quando há `GROUP BY`** — se a consulta compilar, ela está errada.

---

## C-2 · CRÍTICO — uma confirmação de Tentativa velha derruba a Reserva da Tentativa viva

**Onde.** AD-7: *"a varredura de `pedido` lê por `pagamento.ConfirmacoesNaoAplicadas` e aplica. Cada confirmação tem estado terminal […] `PENDENTE` → `APLICADA` · ou → `NAO_APLICAVEL_SINALIZADA`, quando o Pedido já estava `CANCELADO`."*

**O problema.** A decisão de aplicar olha **só o Status do Pedido**, e só distingue um caso não aplicável: `CANCELADO`. Mas um Pedido tem várias Tentativas de Pagamento (FR-27, padrão 3), e uma confirmação pertence a **uma** delas. A espinha nunca amarra a confirmação à Tentativa corrente. Como o caminho de recusa devolve o Pedido a `AGUARDANDO_PAGAMENTO` na nova tentativa, o Status do Pedido volta a ser exatamente o que a confirmação velha espera encontrar.

**Entrelaçamento.** Total na faixa `,95`–`,99` (§7.1: "confirmação nunca chega"). Expiração em 60 s na demonstração.

| # | Sistema | Comprador |
|---|---|---|
| 1 | Tentativa #1 criada. Pedido `AGUARDANDO_PAGAMENTO`, Reserva #1 ativa. | |
| 2 | 60 s depois, a varredura expira: `→ PAGAMENTO_RECUSADO` (`TEMPO_ESGOTADO`), `Liberar` da Reserva #1. | vê "tempo esgotado" |
| 3 | | clica "tentar de novo" (FR-27) |
| 4 | Tentativa #2 criada; `PAGAMENTO_RECUSADO → AGUARDANDO_PAGAMENTO`; **Reserva #2 ativa**. | espera |
| 5 | A confirmação atrasada da **Tentativa #1** chega ao webhook (rede lenta, contêiner que voltou, ou simplesmente a emissão do passo 1 do AD-6 que ficou represada). `INSERT` na inbox, `PENDENTE`. | |
| 6 | Varredura, passo 2: lê a confirmação, olha o Pedido — está em `AGUARDANDO_PAGAMENTO`, que não é `CANCELADO`, então **é aplicável**. Aplica `RECUSADO`: `→ PAGAMENTO_RECUSADO` + **`Liberar` da Reserva #2**. | vê o Pedido recusado sem ter esperado nada |

A Reserva liberada é a da Tentativa viva. O Estoque volta para a loja, o Comprador perde a vez, e o gêmeo do entrelaçamento (confirmação `APROVADO` da #1 chegando depois da #2) marca o Pedido como `PAGO` **pela Tentativa errada** — com a Tentativa #2 pendente para sempre, porque nada mais a resolve.

**Segundo defeito no mesmo lugar.** `NAO_APLICAVEL_SINALIZADA` cobre só `CANCELADO`. Uma confirmação que chega para um Pedido em `PAGAMENTO_RECUSADO` (expirado), `PAGO`, `SEPARANDO`, `ENVIADO` ou `ENTREGUE` não tem estado terminal: fica `PENDENTE`, e a varredura a relê **a cada segundo, para sempre**. É o laço quente que o próprio AD-7 diz querer evitar ("senão a varredura nunca para") — a frase está lá, a cobertura não.

**Aperto de Rule (AD-7).** Duas condições, não uma:

> Uma confirmação só é aplicada se **(a)** pertence à Tentativa de Pagamento **corrente** do Pedido — a de maior `criada_em`, e a espinha declara essa como a única Tentativa que a confirmação pode mover — **e (b)** o Pedido está em `AGUARDANDO_PAGAMENTO`. Falhando qualquer uma das duas, a confirmação vai para `NAO_APLICAVEL_SINALIZADA` com o motivo (`TENTATIVA_SUPERADA` ou `ESTADO_NAO_APLICAVEL`) e é sinalizada ao Administrador. **Nenhuma confirmação permanece `PENDENTE` depois de ser lida uma vez** — o estado terminal é obrigatório em todos os caminhos, não só no do Pedido cancelado.

Isso também dá ao AD-3 a linha que falta na tabela: nenhuma transição sai de `PAGAMENTO_RECUSADO` para `PAGO`, e agora está escrito *quem* impede.

---

## A-1 · ALTO — sem ordem de bloqueio há deadlock entre dois Pedidos; sem nível de isolamento declarado, o remédio errado piora

**Onde.** AD-5 fixa o bloqueio e não fixa a ordem. Nenhum AD declara nível de isolamento em lugar nenhum do documento.

**Entrelaçamento — deadlock.** Produtos P e Q. Comprador A tem `[P, Q]` no Carrinho; Comprador B tem `[Q, P]`. A ordem dos Itens de Carrinho vem da ordem de inserção, então ela **é** diferente entre os dois na vida real.

| # | T1 (Pedido de A) | T2 (Pedido de B) |
|---|---|---|
| 1 | `SELECT … WHERE id = P FOR UPDATE` → obtém | |
| 2 | | `SELECT … WHERE id = Q FOR UPDATE` → obtém |
| 3 | `SELECT … WHERE id = Q FOR UPDATE` → **espera T2** | |
| 4 | | `SELECT … WHERE id = P FOR UPDATE` → **espera T1** |

O Postgres detecta em `deadlock_timeout` (1 s por padrão) e aborta uma das duas com `40P01`. O que o Comprador vê depende de quem tratou o erro — e ninguém tratou: o AD-14 manda "ou é tratado, ou sobe", então sobe como 500, no clique irreversível do checkout, um segundo depois de clicar. Não é raro: são dois Produtos populares e dois cliques simultâneos, e é o cenário que a apresentação provoca de propósito no passo 5 do Roteiro C. E ele reaparece entre `Reservar` (checkout) e `Consolidar` (varredura), que também escreve em `produto`.

**Nível de isolamento.** O desenho de C-1 — bloquear, *depois* somar — é correto **em READ COMMITTED e só nele**. Em REPEATABLE READ o snapshot é da transação, não do comando: ao destravar, o passo 2 não enxergaria a reserva recém-confirmada e a transação abortaria com `40001` em vez de ler o valor novo. Isso falha para o lado seguro (não vende a mais) mas **transforma o teste do NFR-7 em erro intermitente sem laço de repetição**, e a espinha não tem onde esse laço moraria. Um `AZAMON_`-qualquer-coisa, um `default_transaction_isolation` no `docker-compose`, ou um pgx configurado por alguém que leu que "isolamento mais forte é mais seguro" — e a demonstração quebra sem que nenhuma regra tenha sido violada, porque nenhuma regra existe.

**Aperto de Rule (AD-4 + AD-5).**

> **Ordem global de bloqueio, exaustiva:** `pedido.pedido` antes de `catalogo.produto`; múltiplos Produtos sempre por `id` crescente, num **único comando** com `ORDER BY id … FOR UPDATE` (o nó `LockRows` fica acima do `Sort`, então a ordem de bloqueio é a ordenada). Nenhum caminho adquire bloqueio fora dessa ordem.
>
> **O isolamento é READ COMMITTED**, explicitamente — é o padrão do Postgres e é do que este desenho depende: o bloqueio-depois-soma do AD-5 e o *compare-and-swap* do `Transicionar` são corretos porque cada comando toma snapshot novo. Nenhum ponto do sistema pede isolamento mais forte; elevá-lo quebra o AD-5. `40P01` e `40001` são erros de infraestrutura, não de domínio: `api/` repete a requisição **uma vez** e, persistindo, devolve o envelope do AD-14 com `codigo: "CONFLITO_TEMPORARIO"`.

---

## A-2 · ALTO — `Idempotency-Key` repetida: a restrição única impede o segundo Pedido e ninguém decidiu o que o Comprador recebe

**Onde.** AD-7: *"receber duas vezes é um `INSERT` que falha, não código defensivo […] Criar Pedido segue a mesma forma: cabeçalho `Idempotency-Key` com restrição única."* Convenções: *"`Idempotency-Key` gerado pelo navegador (UUID por tentativa de checkout, reaproveitado no reenvio) […] Restrição única no banco é o mecanismo."*

**O problema.** "O `INSERT` falha" descreve o mecanismo e para aí. Três coisas ficam indefinidas, e as três aparecem no caminho mais sensível do sistema.

**(a) O que o chamador faz com o `23505`.** No webhook, ignorar e devolver `200` é óbvio. Em `POST /pedidos`, não: o comportamento correto é devolver **o Pedido original**, e o comportamento que o texto atual produz é um erro. O `23505` aborta a transação inteira no Postgres — a partir dele nem dá para ler o Pedido que ganhou sem abrir transação nova. Se o erro subir pelo AD-14, o Comprador que deu duplo clique vê um toast vermelho sobre um Pedido que **foi criado com sucesso**, com o Carrinho já esvaziado, no único passo irreversível do fluxo. É a pior tela possível do sistema, e ela nasce de a espinha ter descrito o mecanismo sem descrever o desfecho.

**(b) Onde a chave é guardada.** A espinha não diz. Se ela vive numa tabela de idempotência isolada, o segundo pedido não consegue nem *achar* o Pedido original para devolvê-lo. A restrição única precisa ser **sobre uma coluna do próprio `pedido`**, senão a chave impede a duplicata e não permite a resposta correta.

**(c) Mesma chave, corpo diferente.** Indefinido — e não é hipótese acadêmica. A convenção diz "UUID por tentativa de checkout, reaproveitado no reenvio": a mesma chave sobrevive o tempo que a Revisão ficar aberta.

**Entrelaçamento (c).** Marina abre a Revisão, a chave `K` nasce. Em outra aba, ajusta o Carrinho e o Endereço. Volta à primeira aba e confirma:

| # | Requisição | Resultado com o texto atual |
|---|---|---|
| 1 | `POST /pedidos` chave `K`, corpo `{2 carregadores, Endereço casa}` | Pedido `AZ-…-148` criado |
| 2 | `POST /pedidos` chave `K`, corpo `{1 carregador, Endereço trabalho}` | `23505` → erro 500, **ou**, com o "devolve o original" ingênuo, `200` com o Pedido de **2 carregadores para o Endereço de casa** |

O segundo desfecho é pior que o primeiro: entrega silenciosa do Pedido errado, sem nada na tela dizendo que o que foi confirmado não é o que estava escrito.

**Aperto de Rule (AD-7 + convenção de idempotência).**

> `pedido.pedido` carrega `idempotency_key` (único) e `requisicao_digest` — hash do corpo normalizado (itens, quantidades, `endereco_id` e `total_centavos` revisado). Em `POST /pedidos`:
>
> | Situação | Resposta |
> |---|---|
> | chave nova | cria; `201` |
> | chave repetida, digest igual | `200` com o **Pedido original**, corpo idêntico ao da primeira resposta |
> | chave repetida, digest diferente | `409`, `codigo: "CHAVE_IDEMPOTENTE_REUTILIZADA"` |
>
> No `23505`, `api/` **desfaz a transação, abre uma nova, lê pela chave e responde** — o erro do banco é o sinal, nunca a resposta. A mesma disciplina vale para a inbox do webhook, onde a resposta é sempre `200`.

O digest tem um segundo emprego: é ele que fecha o M-3.

---

## A-3 · ALTO — a Reserva de Estoque não tem ciclo de vida, então `Liberar` e `Consolidar` duas vezes são improváveis, não impossíveis

**Onde.** AD-5 nomeia `Reservar`, `Liberar` e `Consolidar` e nunca diz o que a Reserva *é*. AD-3 pendura a proteção inteira num adjetivo: *"`catalogo.Liberar`, **se ativa**"*. "Ativa" não é definido em lugar nenhum — nem como coluna, nem como estado, nem como restrição.

**O contraste é o próprio documento.** O AD-7 deu à confirmação um estado terminal explícito e justificou por quê. O AD-3 deu ao Pedido uma tabela exaustiva de transições e um erro para quem chega segundo. A Reserva de Estoque — que o addendum §2 chama de "o bug mais caro deste sistema" — não ganhou nem estado, nem transição, nem restrição.

**Entrelaçamento — `Liberar` duas vezes.** A tabela do AD-3 tem seis caminhos que chamam `Liberar`, e dois deles são alcançáveis em sequência sobre a **mesma** Reserva:

| # | Transição | Efeito |
|---|---|---|
| 1 | `AGUARDANDO_PAGAMENTO → PAGAMENTO_RECUSADO` (expiração) | `Liberar` |
| 2 | `PAGAMENTO_RECUSADO → CANCELADO` (Comprador, FR-31) | `Liberar` "se ativa" |

Cada transição tem seu *compare-and-swap*, então cada uma acontece uma vez — mas as duas acontecem, e as duas chamam `Liberar` sobre a Reserva #1. O desfecho depende inteiramente de como `Liberar` foi escrito, e a espinha não escolheu:

- `DELETE FROM reserva_estoque WHERE pedido_id = $1` → idempotente por acidente. Sobrevive.
- `UPDATE reserva_estoque SET ativa = false WHERE pedido_id = $1` → idempotente por acidente. Sobrevive.
- `UPDATE produto SET estoque_total = estoque_total + $q` → **crédito duplo: unidades inventadas do nada.** A loja passa a vender mais do que tem, para sempre, e nenhum log registra o momento.

A terceira forma é a que alguém escreve quando pensa "liberar é devolver ao estoque" — que é literalmente como o PRD descreve o efeito ("as unidades voltam ao Estoque disponível"). O AD-5 diz que o disponível é derivado e nunca armazenado, o que **implica** que `Liberar` não pode tocar `estoque_total`; implicação não é regra.

**Entrelaçamento — `Consolidar` duas vezes.** `Consolidar` **escreve** `estoque_total` (AD-3: "baixa o total"). Só o *compare-and-swap* de `SEPARANDO → ENVIADO` o protege, e o AD-3 nomeia essa exata transição como o ponto de corrida entre o Administrador (FR-32) e a simulação (FR-33). Enquanto `Consolidar` estiver **dentro da mesma transação e depois do CAS**, o perdedor desfaz tudo. Nada na espinha exige essa ordem — o AD-4 exige a mesma transação, não a mesma sequência. `Consolidar` antes do CAS, com o CAS falhando, ainda desfaz (`ROLLBACK`); `Consolidar` num `defer` ou num segundo `COMMIT` não. Ver M-1.

**Aperto de Rule (AD-5).** Um índice, e "improvável" vira "impossível" — a mesma troca que o AD-7 já fez para a confirmação:

> A Reserva de Estoque tem estado explícito: `ATIVA → LIBERADA | CONSOLIDADA`, terminais. **Índice único parcial `(pedido_id, produto_id) WHERE estado = 'ATIVA'`** — duas Reservas ativas do mesmo Pedido sobre o mesmo Produto são impossíveis no banco, o que fecha o caminho de nova Tentativa da FR-27 sem código defensivo.
>
> `Liberar` e `Consolidar` são `UPDATE … WHERE pedido_id = $1 AND estado = 'ATIVA'`, e **zero linhas afetadas é sucesso, não erro** — a idempotência é a mesma que o `Transicionar` já tem. `Liberar` **nunca escreve `estoque_total`**: só encerra a Reserva, e o disponível volta sozinho porque é derivado (AD-5). `Consolidar` é o **único** ponto do sistema que escreve `estoque_total` fora do ajuste do Administrador (FR-11).
>
> "Σ reservas ativas" passa a ter definição: `WHERE estado = 'ATIVA'`.

---

## A-4 · ALTO — a emissão do webhook não tem critério derivado: ou a confirmação se perde, ou ela é reemitida para sempre

**Onde.** AD-6, passo 1: *"emite a confirmação da Tentativa de Pagamento cujo atraso configurado já venceu, fazendo o POST do AD-7"*. AD-4: *"o webhook do AD-8 é disparado pela varredura, depois do commit"*.

**O problema.** "Depois do commit" resolve a regra de não fazer rede dentro de transação e cria o buraco clássico: entre o `COMMIT` e o `POST` existe uma janela em que o processo pode morrer. O que a espinha faz nessa janela depende de algo que ela não declara — **o que exatamente foi confirmado no commit**.

**Entrelaçamento (a) — se o commit marca "emitida".**

| # | Varredura | Estado |
|---|---|---|
| 1 | seleciona a Tentativa cujo atraso venceu | `PENDENTE` |
| 2 | `UPDATE tentativa SET emitida = true`; `COMMIT` | marcada |
| 3 | `docker compose restart` / panic / Ctrl-C do apresentador | — |
| 4 | volta; a Tentativa está marcada, o `POST` nunca saiu | **a confirmação nunca existirá** |

O Pedido fica em `AGUARDANDO_PAGAMENTO` até a FR-34 o expirar. Não é perda permanente — a expiração é a rede de segurança, e é bom que ela exista — mas **o desfecho muda de `PAGO` para `PAGAMENTO_RECUSADO/TEMPO_ESGOTADO`**. Na demonstração, com expiração em 60 s, um reinício durante o Roteiro A transforma o caminho feliz no caminho triste, sem erro nenhum na tela, e o apresentador não tem como explicar o que aconteceu.

**Entrelaçamento (b) — se o commit não marca nada** (que é o que o texto atual descreve): a condição "atraso venceu" continua verdadeira no tique seguinte, e no seguinte. A varredura **reemite a cada segundo**. Se a chave de idempotência for gerada nova a cada `POST` — e a convenção diz apenas "gerado pelo Provedor no webhook" — cada reemissão vira uma **linha nova** na inbox, todas `PENDENTE`, todas lidas a cada tique. A restrição única não absorve nada, porque as chaves são diferentes. Para a faixa `,95`–`,99` (confirmação que nunca chega) o laço não tem fim natural.

**Aperto de Rule (AD-6 + AD-7).** A saída não é uma coluna de marcação — é derivar a emissão do mesmo jeito que o AD-6 já deriva tudo o mais, o que também a torna reentrante de graça:

> O passo 1 não escreve nada. Ele seleciona as Tentativas em que `criada_em + atraso ≤ now()` **e que ainda não têm linha em `pagamento.confirmacao_recebida`**, e emite o `POST` fora de qualquer transação. A **chave de idempotência é determinística: derivada do `id` da Tentativa de Pagamento** — reemitir produz a mesma chave, que a restrição única do AD-7 absorve.
>
> Consequências, que é o que torna a regra suficiente: morrer antes do `POST` → o tique seguinte reemite; morrer depois do `POST` → a linha na inbox já existe e o tique seguinte não reemite; `POST` duplicado → `23505`, resposta `200`, nada acontece duas vezes. **Falha de emissão nunca desfaz nada** — não há nada a desfazer.

---

## A-5 · ALTO — a varredura não declara granularidade de transação nem ordem entre os quatro passos

**Onde.** AD-6: *"Um único goroutine com ticker de 1 s […] faz exatamente quatro coisas […] Toda leitura usa `FOR UPDATE SKIP LOCKED`. A varredura é idempotente: rodar duas vezes sobre o mesmo estado produz um efeito só."*

Quatro passos numerados, e três omissões que decidem o comportamento sob concorrência.

**(a) Uma transação por tique ou uma por Pedido?** Se por tique: um único `ErrEstadoJaAvancado` — que é o desfecho **normal** quando o Comprador cancela ao mesmo tempo — desfaz o trabalho de todos os outros Pedidos do tique; e os bloqueios `FOR UPDATE SKIP LOCKED` do passo 1 ficam segurados até o fim do passo 4, fazendo o `SKIP LOCKED` pular linhas que a própria varredura segura.

**(b) O `ErrEstadoJaAvancado` dentro da varredura colide com o AD-14.** O AD-3 classifica esse erro como "informativo — recarrega e mostra o estado real", que é semântica de **tela**. Na varredura não há tela: o desfecho correto é seguir para o próximo Pedido em silêncio. Mas o AD-14 diz "erro engolido é proibido: ou é tratado, ou sobe" — e quem seguir o AD-14 ao pé da letra faz o tique inteiro subir, a cada segundo, sempre que um Comprador cancelar algo.

**(c) A ordem entre "aplicar" (passo 2) e "expirar" (passo 3) não é declarada, e ela decide o desfecho.**

**Entrelaçamento.** Total na faixa `,00`–`,89` (aprovado). Atraso da confirmação simulada e prazo de expiração próximos — na demonstração, 60 s de prazo, e um `POST` que chegou no fio.

| # | Tique único da varredura | Consequência |
|---|---|---|
| 1 | passo 3 roda primeiro: a Tentativa venceu o prazo → `→ PAGAMENTO_RECUSADO` (`TEMPO_ESGOTADO`), `Liberar` | Pedido recusado |
| 2 | passo 2 roda depois: a confirmação `APROVADO` está na inbox, `PENDENTE` | |
| 3 | tenta aplicar sobre um Pedido em `PAGAMENTO_RECUSADO` — não há transição `PAGAMENTO_RECUSADO → PAGO` | fica `PENDENTE` para sempre (ver C-2) e o pagamento aprovado **desaparece** |

Com a ordem invertida, o mesmo estado produz um Pedido `PAGO`. O desfecho de um pagamento aprovado depende da ordem em que dois `for` foram escritos num arquivo, e nenhuma linha da espinha diz qual é a certa.

**Aperto de Rule (AD-6).**

> **Uma transação por Pedido, nunca por tique.** Falha de um Pedido não toca os outros, e o bloqueio dura o que dura a decisão daquele Pedido.
>
> **A ordem dos quatro passos é normativa e é esta:** (1) aplicar confirmações recebidas · (2) expirar Tentativas vencidas · (3) avançar a simulação de entrega · (4) emitir confirmações cujo atraso venceu. **Aplicar vem antes de expirar** — uma confirmação que já está na inbox ganha do relógio, sempre. Emitir vem por último porque é o único passo que faz rede e o único que não escreve nada.
>
> **Dentro da varredura, `pedido.ErrEstadoJaAvancado` é desfecho esperado, não erro:** o Pedido é pulado, registra-se `slog.Info` com a correlação, e o tique segue. É a exceção declarada ao "erro engolido é proibido" do AD-14 — declarada aqui, porque um comportamento que a arquitetura *espera* precisa estar escrito onde ele acontece.

---

## M-1 · MÉDIO — `Transicionar` nunca é declarado *compare-and-swap*, e é dele que a exatidão de tudo depende

**Onde.** AD-3 dá a assinatura (`Transicionar(ctx, tx, pedidoID, esperado, novo, ator, motivo)`), a tabela exaustiva de transições, os três erros e a proibição do `UPDATE pedido SET status`. Não diz **como** o `esperado` é verificado.

**O problema.** Duas implementações honram o texto atual, e só uma é correta:

```sql
-- correta: uma instrução, atômica
UPDATE pedido.pedido SET status = $novo WHERE id = $1 AND status = $esperado;
-- 0 linhas afetadas → ErrEstadoJaAvancado
```

```sql
-- que o texto também permite: duas instruções, uma corrida entre elas
SELECT status FROM pedido.pedido WHERE id = $1;   -- confere em Go
UPDATE pedido.pedido SET status = $novo WHERE id = $1;
```

**Entrelaçamento (segunda forma).** Administrador (FR-32) e simulação (FR-33) sobre `SEPARANDO → ENVIADO` — a corrida que o próprio AD-3 nomeia:

| # | Administrador | Varredura |
|---|---|---|
| 1 | `SELECT status` → `SEPARANDO`; passa na conferência | |
| 2 | | `SELECT status` → `SEPARANDO`; passa na conferência |
| 3 | `UPDATE … SET status='ENVIADO'`; `Consolidar` (baixa o total); `COMMIT` | |
| 4 | | `UPDATE … SET status='ENVIADO'`; **`Consolidar` de novo**; `COMMIT` |

`estoque_total` baixa **duas vezes**, duas linhas idênticas entram em `transicao_status`, e a linha do tempo da FR-30 mostra `SEPARANDO → ENVIADO` duplicado. Nenhum erro em lugar nenhum. O bug aparece como "sumiu estoque" semanas depois, sem correlação nenhuma que o explique, e é exatamente o que a `SELECT … FOR UPDATE` do passo 1 evitaria se estivesse escrita — mas o AD-6 só exige `FOR UPDATE SKIP LOCKED` para a leitura *da varredura*, e o caminho do Administrador não tem regra de bloqueio nenhuma.

**Aperto de Rule (AD-3).**

> `Transicionar` é **uma única instrução**: `UPDATE pedido.pedido SET status = $novo WHERE id = $1 AND status = $esperado`, e **zero linhas afetadas é `ErrEstadoJaAvancado`**. Ler o status e conferir em Go antes de escrever é a forma proibida — é o `UPDATE pedido SET status` de novo, com um `SELECT` na frente. A validação da aresta contra a tabela do AD-3 (`ErrTransicaoInvalida`) acontece em memória, **antes** da instrução; o `esperado` é a defesa contra concorrência, não contra chamador errado. O efeito da terceira coluna roda **depois** do `UPDATE` bem-sucedido, na mesma transação — nessa ordem, para que o `ROLLBACK` de quem perdeu não tenha nada que já tenha vazado.

---

## M-2 · MÉDIO — `total_centavos` é coluna ou derivação? A resposta decide o desfecho do pagamento

**Onde.** AD-3 lista "total" entre o conteúdo escrito uma vez na criação. AD-18 devolve `subtotal_centavos`, `frete_centavos`, `total_centavos`. A tabela de travessias diz que `tentativa_pagamento` congela `total_centavos` na criação da Tentativa. AD-8 recebe `totalCentavos` **como valor**. Em nenhum lugar está escrito se `pedido.total_centavos` existe como coluna ou se cada leitura o recompõe somando `item_pedido` mais `frete`.

**Por que importa aqui e não é preciosismo.** Este é o único sistema do MVP em que **os centavos decidem o comportamento**: o §7.1 manda o Provedor Simulado escolher `APROVADO`, `RECUSADO` ou "nunca chega" pelos dois últimos dígitos do total. Se um caminho de código derivar o total e outro ler a coluna — ou se um incluir o Frete e o outro não, que é o erro que a nomenclatura `subtotal`/`total` provoca — o apresentador escolhe Produto e quantidade para cair na faixa de recusa (Roteiro B, passo 1) e o Pedido é **aprovado**, porque quem decidiu foi um total diferente do exibido. Não há erro, não há log, e o roteiro não roda. É o modo de falha mais caro possível: aquele que só aparece na sala.

Com o conteúdo do Pedido imutável (AD-3), as duas formas *convergem* — enquanto ninguém errar a fórmula. A espinha não deve depender de ninguém não errar uma fórmula que ela não escreveu.

**Aperto de Rule (AD-9 + AD-3).**

> `pedido.pedido` guarda `subtotal_centavos`, `frete_centavos` e `total_centavos` como **colunas `bigint`, escritas uma vez na criação**, e `total_centavos = subtotal_centavos + frete_centavos` é verificado por `CHECK`. **Nenhum caminho de leitura recompõe o total**: `IniciarTentativa` (AD-8), a resposta do AD-18 e a decisão do Provedor Simulado leem a mesma coluna. Somar `item_pedido` fora da criação do Pedido é defeito.

---

## M-3 · MÉDIO — o Frete revisado e o Frete criado podem ser dois, e o Comprador paga o que não viu

**Onde.** AD-17: *"O Frete é congelado no Pedido na criação, e o recálculo da FR-19 acontece na entrada do checkout, que é `pedido`."* FR-19: *"o Comprador nunca é cobrado por um preço que não viu"*, e o Frete é recalculado quando a revalidação muda o subtotal, "inclusive quando a mudança faz o subtotal cruzar o limiar de isenção".

**O problema.** "Entrada do checkout" e "criação" são dois instantes, e a Revisão fica aberta entre eles. O Carrinho reflete preços atuais (FR-18) e o Pedido congela preços praticados (FR-23) — então o subtotal que decidiu a isenção na Revisão e o subtotal que a criação congela são leituras diferentes do Catálogo. A espinha não diz o que a criação faz quando eles divergem.

**Entrelaçamento.** Limiar de isenção: R$ 299,00 (§7.1).

| # | Marina | Rafael (Administrador) |
|---|---|---|
| 1 | entra no checkout; subtotal R$ 305,00; Revisão mostra **Frete R$ 0,00**, total R$ 305,00 | |
| 2 | lê, confere, hesita | corrige o preço de um Produto para baixo (FR-9) |
| 3 | confirma | |
| 4 | a criação relê o Catálogo: subtotal R$ 289,00, **abaixo do limiar** → Frete R$ 19,90 → total R$ 308,90 | |

Marina revisou R$ 305,00 e recebeu um Pedido de R$ 308,90 — **mais caro** que o revisado, ainda que cada Produto tenha ficado mais barato. É a consequência testável da FR-19 quebrada por uma corrida, e é a que aparece na fatura. O mesmo entrelaçamento com o preço subindo produz o gêmeo: um Produto congelado por um preço que a tela nunca mostrou.

**Aperto de Rule (AD-17).**

> `POST /pedidos` carrega o `total_centavos` que a Revisão exibiu. A criação recalcula tudo — preços praticados, subtotal, Frete — **dentro da transação, sob os bloqueios do AD-5** — e, se o total recalculado diferir do revisado, **recusa** com o envelope do AD-14 (`codigo: "TOTAL_DIVERGENTE"`, `dados` com o total anterior, o novo e o que mudou), sem criar Pedido nenhum. A tela reexibe a Revisão com o que mudou destacado, que é a mesma forma da FR-19. Esse mesmo campo é o que entra no `requisicao_digest` do A-2 — uma verificação, dois usos.

---

## B-1 · BAIXO — o "único ponto de arredondamento" não arredonda, e nenhuma divisão existe no MVP

**Onde.** AD-9: *"A Regra de Frete (AD-17) é a única função que arredonda, e arredonda uma vez."*

Confrontado com o AD-17, isso não se sustenta: a Regra de Frete é uma busca em tabela (`faixa de CEP → região → valor em centavos`) mais uma comparação com o limiar de isenção. Busca em tabela não arredonda; comparação não arredonda. O ponto de arredondamento declarado pelo NFR-13 **não arredonda nada**.

Isso não é um defeito de concorrência e não produz centavo errado hoje — produz confusão amanhã. Varrendo o MVP inteiro: subtotal de item é `preço × quantidade` (multiplicação inteira), total é soma, Frete é constante, não há cupom, não há imposto, não há rateio. **Não existe divisão nenhuma no caminho monetário.** A única divisão do sistema é `centavos / 100` na formatação do navegador, exata para qualquer valor abaixo de 2^53 — e cujo instrumento nativo é `Intl.NumberFormat`, não `toFixed` sobre aritmética própria.

O risco real é o dia em que alguém adicionar um percentual (desconto, imposto) ou **ratear o Frete por Vendedor** — que a blindagem do addendum §6 torna plausível, já que o Item de Pedido registra o Vendedor e agrupar por ele é o caminho previsto. Aí a divisão nasce sem lugar declarado para morar.

**Aperto de Rule (AD-9).**

> **Nenhuma divisão existe no caminho monetário do MVP:** subtotal é multiplicação inteira, total é soma, Frete é valor de tabela e a isenção é comparação. A frase "a Regra de Frete é a única função que arredonda" descreve um lugar reservado, não um cálculo existente — e ele continua sendo o único lugar onde uma divisão pode nascer. **O Frete nunca é rateado entre Itens de Pedido nem entre Vendedores**: agrupar por Vendedor (addendum §6) agrupa itens, nunca reparte o Frete. `centavos / 100` acontece uma vez, na renderização, por `Intl.NumberFormat('pt-BR', {style:'currency', currency:'BRL'})`.

---

## B-2 · BAIXO — `Disponivel` é conselho, e a espinha não diz isso

**Onde.** AD-5: `catalogo.Disponivel(ctx, produtoIDs []uuid) map[uuid]int` — **sem `tx`**. Pelo AD-4 ("quem inicia o caso de uso abre a transação e a passa adiante explicitamente"), uma função sem `tx` é, por construção, uma leitura fora da transação do caso de uso.

Isso está certo e é o que se quer: a Vitrine, a busca (VIEW do AD-16), a página de Produto e a checagem da FR-17 no Carrinho leem sem bloquear ninguém. A consequência inevitável é que **o número exibido é uma foto de um instante já passado**: uma reserva em curso e ainda não confirmada é invisível, então a Vitrine pode dizer "3 disponíveis" enquanto o próximo `Reservar` aceita 1. O AD-14 já modelou a saída certa para isso — o envelope de `ESTOQUE_INSUFICIENTE` carrega `disponivel` e `solicitado`.

O que falta é dizer em voz alta que essa divergência é **projetada**, não tolerada. Sem isso, a primeira pessoa a "consertar" a inconsistência vai propor cache, contador materializado ou bloqueio na leitura — três formas de quebrar o AD-5 tentando honrá-lo.

**Aperto de Rule (AD-5).**

> `Disponivel` é **conselho, nunca decisão**: informa tela e revalidação, e nenhum caminho de escrita depende dele. A decisão sobre Estoque acontece uma vez só, dentro de `Reservar`, sob o bloqueio da linha de Produto. Divergência entre o número exibido e o número aceito **é esperada** — é o preço de não bloquear a Vitrine, e a mensagem do AD-14 é o lugar onde ela é resolvida para o Comprador.

---

## B-3 · BAIXO — quando a varredura ganha, o Comprador recebe o erro errado

**Onde.** AD-3 distingue `ErrEstadoJaAvancado` ("informativo — recarrega e mostra o estado real") de `ErrForaDaJanelaDeCancelamento` ("explica por que não dá mais"), e a `EXPERIENCE.md` dá comportamentos de tela diferentes aos dois.

**Entrelaçamento.** Pedido em `SEPARANDO`. Marina abre o detalhe (o botão de cancelar aparece, FR-30) e clica.

| # | Marina | Varredura (FR-33) |
|---|---|---|
| 1 | `POST /pedidos/{id}/cancelar`; o caso de uso lê o Pedido para conferir a janela → `SEPARANDO`, cancelável | |
| 2 | | avança `SEPARANDO → ENVIADO`; `Consolidar`; `COMMIT` |
| 3 | `Transicionar(esperado='SEPARANDO', novo='CANCELADO')` → 0 linhas → `ErrEstadoJaAvancado` | |

Marina recebe "recarregue para ver o estado real" quando a resposta certa é "seu Pedido já saiu para entrega, e por isso não dá mais para cancelar" — que é o caso de borda que a UJ-3 nomeia explicitamente. As duas telas terminam com ela recarregando, então o dano é pequeno; mas é a UJ-3 perdendo seu clímax por uma corrida, no roteiro que existe para demonstrá-la.

**Aperto de Rule (AD-3).**

> No caminho de cancelamento, `ErrEstadoJaAvancado` é reclassificado antes de sair do módulo: se o estado real do Pedido está **fora** da janela de cancelamento, o erro devolvido é `ErrForaDaJanelaDeCancelamento`, com o estado real em `dados`. A janela é consultada depois da falha, não antes da tentativa.

---

## B-4 · BAIXO — a `Idempotency-Key` não sobrevive ao recarregamento que ela existe para proteger

**Onde.** Convenções: *"`Idempotency-Key` gerado **pelo navegador** (UUID por tentativa de checkout, reaproveitado no reenvio)"*. AD-10: *"O processo Node não guarda estado de domínio nenhum."*

Uma chave em estado de React morre no `F5`. E o recarregamento é justamente o gesto de quem clicou em confirmar e viu a tela demorar — o mesmo gesto que a idempotência existe para tornar inofensivo. Recarregar a Revisão gera chave nova, e a chave nova cria o segundo Pedido que a FR-23 proíbe ("confirmar duas vezes cria **um** Pedido, não dois").

**Aperto de Rule (convenção de idempotência).**

> A chave nasce com a **Revisão** e vive em `sessionStorage`, indexada pelo Carrinho, até a criação do Pedido responder — sobrevive ao recarregamento, morre com a aba. É estado de tela, não de domínio: nada no Node a conhece (AD-10), e o servidor nunca a gera.

---

## Resumo dos apertos

| # | AD | Aperto em uma linha |
|---|---|---|
| C-1 | AD-5 | A linha de Produto é o mutex do disponível: bloquear com `ORDER BY id FOR UPDATE`, **depois** somar as reservas, em comandos separados — e a FR-11 segue a mesma ordem |
| C-2 | AD-7 | Aplicar confirmação só se pertence à Tentativa corrente **e** o Pedido está em `AGUARDANDO_PAGAMENTO`; todo outro caso é terminal e sinalizado |
| A-1 | AD-4, AD-5 | Ordem global de bloqueio (`pedido` → `produto`, Produtos por `id`); **READ COMMITTED declarado**; `40P01`/`40001` repetem uma vez em `api/` |
| A-2 | AD-7 | Chave e digest do corpo em `pedido`; repetida+igual devolve o original, repetida+diferente é `409`; `23505` é sinal, nunca resposta |
| A-3 | AD-5 | Reserva com estado (`ATIVA → LIBERADA \| CONSOLIDADA`) e índice único parcial; `Liberar` nunca escreve `estoque_total`; zero linhas é sucesso |
| A-4 | AD-6, AD-7 | Emissão derivada ("sem linha na inbox"), sem marcação, com chave determinística por Tentativa |
| A-5 | AD-6 | Uma transação por Pedido; ordem normativa aplicar → expirar → simular → emitir; `ErrEstadoJaAvancado` é desfecho esperado da varredura |
| M-1 | AD-3 | `Transicionar` é um `UPDATE … WHERE id AND status = esperado`; zero linhas é `ErrEstadoJaAvancado`; o efeito vem depois |
| M-2 | AD-9, AD-3 | `total_centavos` é coluna com `CHECK`; nenhum caminho de leitura recompõe o total |
| M-3 | AD-17 | A criação recalcula sob bloqueio e recusa com `TOTAL_DIVERGENTE` se divergir do revisado |
| B-1 | AD-9 | Nenhuma divisão no caminho monetário; Frete nunca é rateado; `Intl.NumberFormat` na renderização |
| B-2 | AD-5 | `Disponivel` é conselho, nunca decisão; a divergência é projetada |
| B-3 | AD-3 | Cancelamento reclassifica `ErrEstadoJaAvancado` para o erro de janela quando é isso que aconteceu |
| B-4 | convenção | Chave em `sessionStorage`, nascida com a Revisão |

**O padrão.** Onze dos catorze achados são a mesma omissão vista de ângulos diferentes: a espinha escolheu o mecanismo e não escreveu a **ordem**. Bloquear antes de somar (C-1), bloquear em ordem fixa (A-1), aplicar antes de expirar (A-5), transicionar antes do efeito (M-1), recalcular antes de congelar (M-3). Nenhum dos apertos acrescenta mecanismo novo — todos ordenam o que já está lá, e três deles (A-2, A-3, A-4) trocam código defensivo por restrição de banco, que é o movimento que o próprio AD-7 já fez e que o resto do documento ainda não copiou.
