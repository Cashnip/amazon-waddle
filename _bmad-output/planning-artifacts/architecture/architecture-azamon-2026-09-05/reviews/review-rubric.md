---
title: 'Revisão de rubrica — Espinha de Arquitetura do Azamon'
alvo: '_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md'
tipo: review-rubric
altitude: initiative
data: '2026-09-05'
insumos:
  - 'prds/prd-azamon-2026-08-14/prd.md'
  - 'prds/prd-azamon-2026-08-14/addendum.md'
  - 'ux-designs/ux-azamon-2026-09-02/EXPERIENCE.md'
  - 'ux-designs/ux-azamon-2026-09-02/DESIGN.md'
  - 'architecture/architecture-azamon-2026-09-05/.memlog.md'
---

# Revisão de rubrica — Espinha de Arquitetura do Azamon

**Veredito geral.** Espinha forte e genuinamente terse. Os 20 ADs quase todos passam no teste de pertencimento — se duas das seis funcionalidades do §4 fossem construídas em paralelo, cada AD nomeia uma escolha incompatível que elas fariam. O documento já sobreviveu a três reconciliações (addendum, PRD, UX) e as correções pegaram. O que resta é **um crítico** (a varredura tem quatro deveres e um dono só, e um deles quebra o NFR-3), **três altos** (contrato de listagem paginada inexistente, teto de Tentativas de Pagamento sem dono, semente e volume sem decisão) e um punhado de aparas. Nada disso é reescrita; é uma passada de correção.

| # | Dimensão | Veredito |
|---|---|---|
| 1 | Fixa os pontos de divergência reais | **adequate** |
| 2 | Rule executável que previne o Prevents | **strong** |
| 3 | Nada em Adiado deixa unidades divergirem | **strong** |
| 4 | Tecnologia nomeada verificada e atual | **adequate** |
| 5 | Cobre as capacidades do §4 do PRD | **adequate** |
| 6 | Toda dimensão decidida, adiada ou aberta | **adequate** |
| 7 | Ausência de excesso | **strong** |
| 8 | Diagramas mermaid válidos e carregando forma | **adequate** |

---

## 1. Fixa os pontos de divergência reais do nível abaixo — **adequate**

O nível abaixo são as seis funcionalidades do §4, com 2 a 4 pessoas em paralelo, e o addendum §8 diz que os cortes mais limpos para dividir o time são "busca" e "identidade + catálogo". A espinha acerta os pontos onde essas frentes colidem: a direção da dependência (AD-1), a fronteira de schema (AD-2), a máquina de estados com efeitos exaustivos (AD-3), a transação como parâmetro (AD-4), o dono do Estoque (AD-5), o motor do tempo (AD-6), a idempotência estrutural (AD-7) e o contrato de resposta (AD-18). Essa lista cobre praticamente toda superfície onde `carrinho`, `pedido`, `catalogo` e `pagamento` se tocam.

Três divergências reais ficaram de fora.

### CRÍTICO — C-1: a varredura tem quatro deveres, um dono declarado, e o dever 1 quebra o NFR-3

O AD-6 dá quatro tarefas ao *goroutine*: (1) **emitir** a confirmação simulada por POST, (2) **aplicar** as confirmações não aplicadas, (3) **expirar** a Tentativa vencida, (4) **avançar** a simulação de entrega. Três leituras da espinha se contradizem sobre onde isso mora:

- O AD-1 desenha `varredura` como **nó próprio**, com **uma seta só**, `varredura --> pedido` — e declara que "uma seta que não está aqui é defeito".
- A árvore de origem coloca a varredura **dentro de `internal/pedido/`** (`pedido/ # Pedido, Item de Pedido, máquina de estados, Frete, varredura`).
- O AD-7 exige que a varredura leia `pagamento.ConfirmacoesNaoAplicadas`, e o AD-6 exige que ela faça o POST que é comportamento do **Provedor Simulado**.

Se a varredura é `pedido`, ela não tem seta para `pagamento` no grafo exaustivo — mesmo defeito de forma que o reconciliador do PRD já pegou no Frete. Pior: o dever 1 é o que, em produção, o **servidor do Stripe** faria. Emitir a confirmação a partir de `internal/pedido/` significa que trocar o Provedor exige **apagar código do Pedido** — exatamente o que o NFR-3 proíbe ("zero alteração no checkout, no Pedido ou na máquina de estados") e o que a SM-5 verifica por inspeção antes da entrega. A espinha inteira do AD-8 existe para proteger essa promessa, e o AD-6 a fura sem dizer.

*Correção:* separar os deveres por dono. O dever 1 pertence a `pagamento` (é o adapter simulando o mundo externo, e some junto com ele na virada para o Stripe); os deveres 2, 3 e 4 pertencem a `pedido`. Uma varredura, dois donos de tarefa, ou duas varreduras. Qualquer que seja a escolha, o AD-1 precisa da seta correspondente e a árvore precisa refletir a divisão.

### ALTO — A-2: o teto de Tentativas de Pagamento não tem dono

O §7.1 fixa "Tentativas de Pagamento por Pedido = 3" e a FR-27 diz "atingido o limite, a única ação restante para o Comprador é cancelar". A espinha expõe `tentativas_restantes` na resposta (AD-18) e nomeia `pagamento` como dono da tabela de Tentativas (tabela de travessias), mas **nenhum AD diz quem recusa a quarta**. A tabela do AD-3 é declarada "exaustiva e obrigatória" e a linha `PAGAMENTO_RECUSADO → AGUARDANDO_PAGAMENTO` carrega só o efeito de Estoque, sem a guarda do teto; a porta do AD-8 tem duas operações e nenhuma menciona limite.

Duas pessoas resolvem isso de dois jeitos — a guarda em `pedido` (que conhece a transição) ou em `pagamento` (que conta as Tentativas) — e o modo de falha silencioso é ninguém a escrever, porque cada um assume que o outro escreveu. É um limiar do §7.1 com consequência testável da FR-27.

*Correção:* uma coluna na linha de transição do AD-3, ou uma condição nomeada na porta do AD-8. Uma frase resolve.

### ALTO — A-1: não existe contrato de listagem paginada

A FR-15 pagina quatro superfícies — Resultados de busca, listagens de Categoria, Meus pedidos e as listas administrativas — e a `EXPERIENCE.md` exige delas o mesmo comportamento ("informa página atual e total de resultados", "página além do total é estado vazio tratado"). Essas superfícies são construídas por **três módulos diferentes** (`busca`, `pedido`, `catalogo` via admin). A espinha contrata o **envelope de erro** (AD-14) e o **envelope de um Pedido** (AD-18), e nada sobre listas: nem o formato (`{itens, pagina, total}` × `{data, meta}`), nem quem conta o total, nem se `pagina` é 0- ou 1-indexada.

Pior, a FR-14 exige "critério de desempate estável — a paginação nunca repete nem pula um Produto", e a `EXPERIENCE.md` repete a promessa. A espinha menciona o desempate **uma vez, de passagem**, dentro do AD-16, como justificativa para não usar `tsvector` — nunca como regra. Qual é a chave de desempate? A `uuidv7()` das Convenções seria a escolha óbvia (ordenada no tempo, única), mas óbvio-para-mim não é decidido, e um `ORDER BY preco` sem desempate quebra a FR-14 sem nenhum teste vermelho.

*Correção:* uma linha nas Convenções com o envelope de lista, e uma linha fixando o desempate (`ORDER BY <chave>, id` com a `uuidv7` como último critério em toda ordenação).

### ALTO — A-3: quem carrega a semente, e o Postgres persiste entre subidas?

O NFR-1 é métrica primária (SM-3, medida semanalmente por pessoas diferentes) e diz: "subir o ambiente duas vezes produz exatamente os mesmos dados". As Convenções colocam o Catálogo Semeado em `db/semente/`, versionado, com semente fixa de aleatoriedade — e param aí. Fica sem decisão:

- **Quem aplica a semente?** É uma migração goose (então roda uma vez e nunca mais)? Um passo do arranque do binário? Um serviço do compose? O dono é `plataforma` (que aplica migrações no arranque) ou `catalogo` (que possui os dados)?
- **A semente é idempotente?** Se o volume do Postgres persiste, a segunda subida não pode duplicar 5.000 Produtos. Se não persiste, a promessa de determinismo é trivial mas todo Pedido criado em ensaio evapora.
- **O `docker-compose.yml` declara volume nomeado?** O diagrama de topologia não mostra nenhum, e a seção Ambientes não diz.

Some-se a isso uma afirmação que não fecha: as Convenções prometem que subir duas vezes produz "os mesmos dados **e os mesmos `numero` de Pedido**". O glossário do PRD define Catálogo Semeado como "Vendedores, Categorias e Produtos fictícios" — **não há Pedido semeado**, e o `numero` nasce no checkout, em tempo de execução. A promessa é ou falsa ou revela um Pedido semeado que nenhum outro lugar declara.

*Correção:* uma linha na seção Ambientes dizendo quem aplica a semente, se há volume nomeado, e o que acontece na segunda subida. E cortar ou consertar a promessa dos `numero`.

### MÉDIO — M-7: o formato de migração colide sob trabalho paralelo

As Convenções fixam `db/migracoes/NNNN_<módulo>_<descrição>.sql`. Com 2 a 4 pessoas em ramos paralelos — que é a premissa desta altitude — dois desenvolvedores criam `0007_` na mesma semana e o conflito só aparece no merge, ou pior, não aparece e a ordem de aplicação vira aleatória entre máquinas. O goose suporta versionamento por *timestamp* exatamente para esse caso. A escolha do prefixo sequencial é defensável se o time for pequeno e disciplinado, mas é uma decisão que a espinha tomou sem nomear o custo.

### BAIXO — B-3: seta sem uso declarado

`carrinho --> identidade` está no grafo exaustivo do AD-1, mas a tabela de travessias diz que `carrinho.comprador_id → identidade` congela **"Nada"**, e o AD-11 põe a verificação de posse dentro do módulo dono, na própria consulta (`WHERE ... AND comprador_id = $2`) — que não exige chamar `identidade`. Uma seta a mais num grafo declarado exaustivo é permissão para acoplar. Ou justifique-a numa palavra, ou apague-a.

---

## 2. Cada AD tem Rule executável que previne o Prevents declarado — **strong**

Percorridos os 20. A qualidade média é alta e o padrão é reconhecível: quase toda Rule é uma **proibição verificável por leitura ou por `grep`**, não uma aspiração. Amostra do que sustenta o veredito:

- **AD-4** — "Transação em `context.Context` é proibida" com a assinatura literal `Reservar(ctx context.Context, tx pgx.Tx, …)`. Previne exatamente o que Prevents afirma, e a violação é visível na assinatura.
- **AD-5** — "Estoque disponível **nunca é armazenado**". Previne a segunda cópia da verdade por construção: não existe coluna para divergir.
- **AD-7** — a idempotência é **restrição única no banco**, não código defensivo. O Prevents ("que a fronteira seja promessa em vez de fato") é impedido pelo mecanismo, não pela intenção.
- **AD-11** — "a verificação de posse acontece **na mesma consulta que carrega o dado**, nunca como `if` depois da leitura". É a diferença entre uma regra e um lembrete.
- **AD-12** — "um passo de CI faz `grep` pelos três padrões e falha o build". É o único AD com mecanismo de execução explícito, e é o modelo que os outros deviam seguir.
- **AD-19** — "Nenhum outro lugar escreve `WHERE ativo = true`". Uma linha, `grep`-ável, fecha o Prevents inteiro.
- **AD-9, AD-18, AD-20** — limiares concretos (`int64`, RFC 3339 absoluto, `m=19456 t=2 p=1`), sem "deve ser consistente" em lugar nenhum.

Nenhuma Rule é vaga. Duas ficam abaixo do padrão da própria espinha.

### MÉDIO — M-1: o AD-1 não tem mecanismo, e é o AD que mais precisa de um

O AD-1 é o invariante mais importante do documento e sua Rule é puramente declarativa: "as setas abaixo são exaustivas; uma seta que não está aqui é defeito". Quem detecta o defeito? O AD-12, muito menos crítico, ganhou um passo de CI. O NFR-2 oferece a verificação de bandeja — "checável por inspeção **ou por regra de dependência automatizada**" — e a espinha escolheu inspeção sem dizer que escolheu. Com 2 a 4 pessoas e um semestre, inspeção é a verificação que não acontece.

*Correção:* o mesmo passo de CI do AD-12 pode rodar um `grep` de importação proibida (`internal/<x>` importado de fora do arquivo público), ou `go-arch-lint`/`depguard`. Custa uma linha no mesmo passo que já existe.

### BAIXO — B-2: `numero` do Pedido é adjetivo, não regra

O AD-15 diz "**`numero`**, curto, legível e estável, separado do `uuid`" e mostra `AZ-2026-000148` num exemplo. "Curto e legível" não é regra executável, e o exemplo insinua um gerador (sequência anual com prefixo) que nunca é declarado. O dono é único (`pedido`), então não há divergência entre unidades — mas quatro superfícies exibem esse valor, o NFR-9 parte dele, e a promessa de determinismo da semente o cita. Uma linha com o gerador fecha.

---

## 3. Nada em "Adiado" poderia deixar duas unidades divergirem — **strong**

Dez itens, **todos com condição de revisita declarada** — inclusive um com a condição correta de "nunca volta para cá" (esquema completo de colunas, que passa a viver nas migrações). Passei cada um pelo teste: nenhum item adiado é uma escolha que duas unidades fariam de forma incompatível, porque em cada caso a espinha já fixou a **costura** e adiou só a implementação atrás dela:

| Adiado | Costura já fixada | Divergência possível? |
|---|---|---|
| Elasticsearch | AD-16 (só `busca` monta consulta) | Não |
| Cache no Redis | AD-20 + Convenções (Redis só tem três usos, todos com TTL) | Não |
| Stripe | AD-7 + AD-8 (porta de duas operações, webhook, inbox) | Não |
| S3 | AD-12 (URL relativa, servida pelo Go) | Não |
| Extrair um serviço | AD-1 + AD-2 (schema por módulo) | Não |
| Portal do Vendedor | AD-11 (posse por dono do recurso) | Não |
| Mailpit | AD-20 (token no log) | Não |
| Esquema de colunas | ERD + tabela de travessias fixam o que cada lado congela | Não |
| Observabilidade | AD-15 (log estruturado com correlação) | Não |
| CI além do `grep` | AD-12 tem o passo mínimo | Não — mas ver M-1 |

A qualidade das justificativas também é alta: cada uma nomeia a **medição** que dispararia a revisita ("o NFR-4 falhar em medição — e aí índice antes de serviço"), não uma data. Isso é o padrão certo.

O achado desta dimensão não está no que foi adiado, e sim no que **não chegou nem a Adiado** — o contrato de listagem (A-1), o teto de tentativas (A-2) e a semente (A-3), que a espinha não decide nem adia. Silêncio não é adiamento.

---

## 4. Tecnologia nomeada é verificada e atual — **adequate**

A tabela de Stack traz carimbo — "*Verificado na web em 2026-09-05*" — e o `.memlog.md` guarda duas entradas `(version)` com as verificações, incluindo detalhes que só uma verificação real produz ("Next.js 16 exige Node 20.9+", "PostgreSQL 19 em beta 3", "Node 26.x é current"). Esse é o padrão que a rubrica pede, e ele foi cumprido.

**Limite desta revisão, declarado:** meu corte de conhecimento é anterior às versões carimbadas, então não posso reconferir os números de release por conta própria — só a coerência interna e as afirmações técnicas que não dependem de data. Todas as que pude checar se sustentam:

- **`uuidv7()` nativo no PostgreSQL 18, sem extensão** — correto. É a função nova do 18, e o argumento (ordenada no tempo, PK sem geração no lado Go) está certo.
- **`net/http.ServeMux` com método + curinga de caminho desde o Go 1.22** — correto, e a conclusão ("nenhuma necessidade deste projeto justifica chi ou gin") é honesta: não há restrição por regex nem introspecção de rota no §4.
- **Argon2id com `m=19456` (19 MiB), `t=2`, `p=1`, sal de 16 bytes** — é uma das configurações documentadas do OWASP para Argon2id, e `golang.org/x/crypto/argon2` expõe `argon2.IDKey` com exatamente esses parâmetros.
- **GIN com `pg_trgm` sobre coluna normalizada preenchida na escrita** — correto, e o motivo registrado no memlog é o tipo de detalhe que separa verificação de chute: `unaccent` **não é `IMMUTABLE`**, então não pode entrar em coluna gerada nem em índice de expressão. Normalizar na escrita, pelo Go, é a saída certa.
- **sqlc lê anotações goose nativamente** — correto, e é o que torna a dupla coerente.
- **Node 24.x como LTS ativa em setembro de 2026** — coerente com a cadência do Node (par entra em LTS em outubro do ano de lançamento). O memlog registra 26.x como *current*, o que fecha.

### BAIXO — B-10: um número para reconferir antes de o README fixar

**PostgreSQL 18.6 em 2026-08-13.** O 18.0 saiu em setembro de 2025 e a cadência normal de *minor* do Postgres é quadrimestral (quatro por ano), o que colocaria o 18.4 em agosto de 2026, não o 18.6. Releases fora de cadência por CVE acontecem e explicariam a diferença — mas como o README e o `docker-compose.yml` vão fixar essa tag e o NFR-1 é medido semanalmente, vale reconferir antes de gravar.

### BAIXO — B-6: duas dependências de arranque que a Stack não nomeia

O AD-16 depende de `pg_trgm`, que exige `CREATE EXTENSION` em migração — não aparece em lugar nenhum. E o Roteiro C começa com "entrar como Administrador", cuja conta a suposição da FR-4 diz nascer "semeada por migração/seed": a ERD tem a tabela `ADMINISTRADOR` com `senha_hash`, mas nada diz que a semente carrega um hash Argon2id fixo (necessário, já que a semente é versionada e determinística). Nenhum dos dois é invariante entre unidades; ambos são o tipo de detalhe que faz o `docker compose up` do NFR-1 falhar na primeira vez.

---

## 5. Cobre as capacidades do PRD que dirigiu a espinha — **adequate**

O Mapa Funcionalidade → Arquitetura tem **seis linhas para as seis seções do §4**, na mesma ordem e com os mesmos intervalos de FR do índice — fecha um a um. A sétima linha (§6.1, entregáveis de documentação) é um acréscimo bom: é escopo declarado do MVP, medido pela SM-7, e costuma ser esquecido por espinhas focadas em domínio.

Varri os 34 FRs contra os ADs. Todos têm casa, exceto os buracos já levantados. Cobertura dos NFRs: 14 dos 16 têm AD nomeado; ver B-1 abaixo.

Achados desta dimensão, além dos altos já registrados:

### MÉDIO — M-4: `ErrTransicaoInvalida` precisa carregar as transições permitidas

A FR-32 tem consequência testável explícita: "Transição inválida é recusada **informando quais transições são permitidas a partir do estado atual**". A `EXPERIENCE.md` (caso de borda do Roteiro C) fecha o mesmo pedido: "`Alert` destrutivo nomeando a transição tentada e dizendo qual seria permitida a partir daquele estado". O AD-14 criou o campo `dados` no envelope justamente para casos assim, e o AD-3 nomeia o erro — mas **nenhum dos dois diz que `ErrTransicaoInvalida` carrega a lista de transições permitidas**. Sem isso, a alternativa que o desenvolvedor do painel vai escolher é duplicar a tabela do AD-3 no navegador, que é a máquina de estados morando num sétimo lugar.

### MÉDIO — M-5: a idempotência de `POST /pedidos` precisa devolver o Pedido, não o erro

O AD-7 fecha com "Criar Pedido segue a mesma forma: cabeçalho `Idempotency-Key` com restrição única". A **forma** é a mesma; o **desfecho correto**, não. No webhook, o `INSERT` que falha pode ser descartado — a confirmação já foi registrada. Em `POST /pedidos`, a FR-23 exige que o duplo clique "crie **um** Pedido, não dois", e o Comprador que reenviou precisa ir para a tela do Pedido que existe, não receber erro de chave duplicada. É a diferença entre um 409 e um 200 apontando para o recurso existente, e ela decide o clímax da UJ-1.

### MÉDIO — M-6: o sinal de "pagamento aprovado sobre Pedido cancelado" não tem caminho de leitura

O AD-7 fixa o estado terminal `NAO_APLICAVEL_SINALIZADA` e diz "sinalizada ao Administrador". A `EXPERIENCE.md` especifica onde: `Alert` persistente no Detalhe do Administrador **e marcador na linha da Tabela de Pedidos**. O dado nasce em `pagamento`; as duas superfícies são servidas por `pedido` + `api`. A seta `pedido --> pagamento` existe, então o caminho é legal — mas o AD-18 contrata **só a resposta do Comprador**, e nenhuma resposta administrativa é contratada em lugar nenhum. É a mesma classe de omissão que a reconciliação de UX pegou antes do AD-18 existir, sobrevivendo no lado administrativo.

### MÉDIO — M-2: "terminal" não é definido, e as duas fontes discordam

O AD-18 fecha as superfícies de consulta em intervalo com "o mesmo a cada 10 s **enquanto o estado não for terminal**". O PRD (§4.6) diz "`ENVIADO`, `ENTREGUE` e `CANCELADO` são terminais **para o Comprador**"; a `EXPERIENCE.md` diz "enquanto o estado não for terminal (`ENTREGUE` ou `CANCELADO`)". Com a definição do PRD, a tela para de consultar em `ENVIADO` e o passo 9 do Roteiro A — "deixar a simulação avançar até `ENTREGUE` durante o resto da apresentação" — nunca acontece na tela. O termo tem duas definições nas fontes da própria espinha e a espinha usa a palavra sem escolher.

### BAIXO — B-7: cancelar um Pedido já `CANCELADO`

A FR-31 exige que cancelar duas vezes "não produza efeito **nem erro de servidor**". A tabela do AD-3 é exaustiva e não tem `CANCELADO → CANCELADO`, então o caso cai em `ErrTransicaoInvalida`, que o AD-3 classifica como "destrutivo — é **defeito de quem chamou**". A `EXPERIENCE.md` não oferece o botão nesse estado, então o caso só é alcançável por chamada direta à API e o risco prático é baixo — mas a FR-31 lista a chamada direta explicitamente em outra consequência, e "defeito de quem chamou" é a classificação errada para um duplo clique.

### BAIXO — B-1: NFR-10 e NFR-11 sem uma palavra

O *frontmatter* promete `binds: PRD §7 NFR-1..NFR-16`. Catorze têm AD ou convenção. Responsividade (NFR-10) e acessibilidade básica (NFR-11) não aparecem — nem decididos, nem adiados. Na prática estão **governados pela espinha de UX**, e bem: a `DESIGN.md` fixa 360 px a 1440 px sem rolagem horizontal, contrastes AA verificados com números, e a `EXPERIENCE.md` fixa `aria-live` e operação por teclado. A delegação é a decisão certa; o que falta é declará-la. Uma linha ("NFR-10 e NFR-11 são governados pela `DESIGN.md`") fecha, e evita que a promessa do *frontmatter* seja lida como cobertura.

---

## 6. Toda dimensão desta altitude está decidida, adiada ou é questão em aberto — **adequate**

Esta é a dimensão em que rascunhos focados em domínio costumam falhar, e é onde esta espinha se destaca: **o envelope operacional existe e é substancial**. Contêineres e topologia com diagrama, tabela de Ambientes, `docker compose up` como comando único, modo demonstração como padrão (AD-13), migrações embutidas por `embed.FS` e aplicadas no arranque, segredo fora do repositório, log estruturado (AD-15), passo de CI (AD-12). E — o que é raro e correto — a **recusa declarada** de um terceiro ambiente, com a justificativa ancorada no §5 e no §8 do PRD: "não existe produção; um terceiro ambiente seria arquitetura de mentira". Isso é uma decisão, não um silêncio.

Percorrendo as dimensões de uma altitude de iniciativa: paradigma ✓, decomposição ✓, persistência e fronteira de dados ✓, consistência transacional ✓, máquina de estados ✓, tempo e assincronia ✓, integração externa ✓, contrato de API ✓, autenticação e autorização ✓, configuração ✓, observabilidade ✓ (log decidido, métrica e traço adiados com condição), testes ✓, implantação e ambientes ✓, custo ✓.

Faltas:

- **A-3 (ALTO, acima)** — a semente e a persistência de volume são o buraco do envelope operacional. É o item que a demonstração toca primeiro.
- **MÉDIO — M-8: a afirmação de desempenho do Adiado não conta o que o AD-16 acrescentou.** O item "Elasticsearch" justifica a espera com "5.000 Produtos com GIN `pg_trgm` ficam **muito abaixo** dos 500 ms do NFR-4". Essa estimativa nasceu antes de o AD-16 passar a exigir que a VIEW da busca exponha `estoque_disponivel` **derivado** — ou seja, uma agregação sobre `catalogo.reserva_estoque` em toda listagem, somada ao GIN. Provavelmente continua folgado (as reservas são poucas e de vida curta), mas a afirmação "muito abaixo" agora é sobre uma consulta diferente da que foi estimada, e a SM-4 mede exatamente isso. Vale reduzir a afirmação a "a medir na SM-4" ou nomear o índice parcial sobre reservas ativas.
- **BAIXO — B-9: a espinha não tem seção de Questões em Aberto.** Tudo está decidido ou adiado. Isso é geralmente bom sinal, e a Questão 1 do PRD (Next.js × React) foi de fato **resolvida** com razão registrada no memlog. Mas a Questão 3 do PRD — "existe rubrica escrita do professor?" — segue declarada bloqueadora, com quatro suposições penduradas nela, e a espinha a menciona uma única vez, de lado, na condição de revisita do Portal do Vendedor. Uma seção de duas linhas dizendo o que muda na arquitetura se a rubrica chegar seria mais honesta que a ausência.
- **BAIXO — B-8: a lista de `grep` do AD-12 tem um furo conhecido.** Os três padrões (`Google Fonts`, `@import url(http`, `//cdn.`) não pegam `next/font/google`, que é justamente a forma idiomática de puxar fonte no Next.js e a que um desenvolvedor escreveria por reflexo. Ela busca em tempo de *build*, não de execução, então não viola a letra do NFR-15 — mas é o padrão que a próxima pessoa vai escrever, e um quarto termo no `grep` custa nada.

---

## 7. Ausência de excesso — **strong**

Vinte ADs para uma altitude de iniciativa com seis módulos é denso, não inflado. Submeti cada um ao teste de pertencimento — "duas unidades um nível abaixo escolheriam de forma incompatível?" — e **dezoito passam com folga**. Mais importante: os ADs que *parecem* repetir o PRD, não repetem:

- **AD-12** poderia ser o NFR-15 recitado. Não é: ele acrescenta a decisão que decide a fase 2 — as imagens são servidas **pelo Go**, com URL relativa, "é assim que a costura do S3 fica do lado certo da fronteira". Se morassem em `web/public/`, o §11 ficaria caro. Isso é informação arquitetural que o PRD não tem.
- **AD-11** poderia ser o NFR-6. Não é: acrescenta *onde* (dentro do módulo dono, na mesma consulta) e *como o papel é checado* (prefixo de rota, em `api/`).
- **AD-3** copia a tabela do addendum §2 — e deve copiar. A reconciliação já tinha marcado como crítica a ausência dos nove efeitos colaterais; a espinha é o substrato de construção, e o substrato precisa carregar a tabela inteira. O que ela acrescenta por cima (a assinatura de `Transicionar`, a proibição de `UPDATE pedido SET status`, os três erros distintos) é decisão nova.
- **AD-9** é o mais próximo de recitar o NFR-13, mas ganha o lugar ao **nomear qual é o único ponto que arredonda** — a Regra de Frete do AD-17. O NFR-13 exige "um único ponto declarado"; a espinha é quem declara.

Nenhum AD é cerimônia. Quatro aparas menores:

### BAIXO — B-4: duas linhas das Convenções não são arquitetura

- **"Commits — Conventional commits em inglês."** Zero consequência arquitetural; é convenção de time, e a metade que importa (*"tudo que sai para o usuário, em português"*) já é a disciplina do glossário, coberta na primeira linha da tabela.
- **"Componentes de tela — shadcn/ui em `web/components/ui/`, tokens vindos do `DESIGN.md`. Verde só para Estoque disponível e `ENTREGUE`; laranja uma vez por fluxo, no passo irreversível."** A primeira frase é invariante (onde os componentes moram, de onde vêm os tokens). As duas regras de cor são da `DESIGN.md`, transcritas — e transcrição cria dois lugares para mudar, que é exatamente o que a SM-7 pune ("documentação que descreve um sistema diferente do construído é pior que documentação nenhuma"). Trocar por um ponteiro.

### BAIXO — B-5: os parâmetros do OWASP no AD-20 são detalhe de um módulo só

Pelo teste de pertencimento estrito, `m=19456 t=2 p=1` não é invariante: exatamente **uma** unidade (`identidade`) implementa hash de senha, então não há como duas divergirem. O que é invariante no AD-20 é a outra metade — Sessão, token e contador de bloqueio vivem no Redis com TTL (decide o papel do Redis no sistema inteiro) e **não existe serviço de e-mail**, então o token da FR-3 sai no log (decide o envelope de demonstração e fecha um FR que não tinha canal). Manter os parâmetros é defensável: é piso de segurança, e piso de segurança não se simplifica. Registro como apara consciente, não como corte recomendado.

### BAIXO — B-11: o exemplo JSON do AD-18 podia ser metade

O invariante é a **tripla** — `(Status, tentativas restantes, disponibilidade dos Itens)` — mais "todo instante é absoluto, em RFC 3339". O exemplo com nove linhas, `historico`, `subtotal/frete/total` e nomes de Produto é forma completa dos dados, que a rubrica classifica como semente que o código passa a possuir. Não faz mal — deixa o invariante concreto — mas é o único ponto do documento onde a espinha desce até o formato.

---

## 8. Os diagramas mermaid são válidos e carregam a forma — **adequate**

**Validade de sintaxe: os três passam.** Verifiquei nó a nó:

- **AD-1 (`graph TD`)** — ids simples, um hexágono `porta{{"…"}}`, rótulos com `<br/>` e `·` entre aspas. Válido.
- **Topologia (`graph LR`)** — `subgraph compose["docker compose up"]` com nó externo ligando para dentro, cilindros `pg[("…")]`, rótulos de aresta entre `|"…"|` contendo `/` e `*` (protegidos pelas aspas), e um auto-laço tracejado `app -.->|"…"| app`. Tudo válido; o auto-laço renderiza.
- **ERD (`erDiagram`)** — cardinalidades corretas, auto-relação `CATEGORIA ||--o| CATEGORIA : pai_opcional` (suportada), blocos de atributo com tipos livres. Válido.

**Forma carregada.** Dois dos três ganham o lugar com folga. O grafo do AD-1 é um DAG de onze nós — prosa nenhuma carrega isso, e a declaração de exaustividade só é verificável porque o desenho existe. A topologia carrega o que a prosa mais erraria: o **auto-laço tracejado**, mostrando que a confirmação sai do processo e volta pelo mesmo caminho que o Stripe usaria. A frase abaixo dele ("o laço tracejado é deliberado") só funciona porque o laço está desenhado.

Três achados.

### MÉDIO — M-3: o ERD desenha travessias de schema com a notação que o AD-2 proíbe

O AD-2 é um dos invariantes centrais: um schema por módulo, **chave estrangeira cruzando schema é proibida**. O ERD desenha nove relacionamentos com a mesma notação `||--o{`, que em ER é exatamente a notação de FK. Sete são intra-schema (legítimos: `PRODUTO`→`RESERVA_ESTOQUE`, `PEDIDO`→`ITEM_PEDIDO`, …). **Dois atravessam schema e não podem ter FK**:

- `COMPRADOR ||--|| CARRINHO : tem` — `identidade` → `carrinho`
- `PEDIDO ||--o{ TENTATIVA_PAGAMENTO : submete` — `pedido` → `pagamento`

Os dois constam corretamente da tabela de travessias logo abaixo, o que salva o documento da contradição — mas o leitor que abre só o ERD (a banca, no dia) vê nove FKs onde existem sete. Some-se: o ERD **não diz de que schema é cada entidade**, e o schema é a fronteira que o AD-2 torna objeto de banco. Um sufixo no nome, ou dois estilos de linha, ou uma legenda de uma linha resolvem.

Anexo ao mesmo achado: `ADMINISTRADOR` e `FAIXA_FRETE` são as **únicas** entidades com bloco de atributos, e as únicas sem relacionamento nenhum. A assimetria não é escolha de desenho — é o contorno do mermaid, que descarta entidade sem aresta e sem atributo. Funciona, mas lido de fora parece arbitrário.

### MÉDIO — M-9: o AD-1 é declarado o entregável do §6.1 e não cumpre o que o §6.1 pede

A última linha do Mapa afirma: "o diagrama do AD-1 **é** o diagrama exigido". O §6.1 do PRD pede "diagrama dos seis módulos do NFR-2 **com suas fronteiras e o que atravessa cada uma**". O grafo do AD-1 tem onze nós e **zero rótulos de aresta** — mostra que `pedido` depende de `catalogo`, não mostra que o que atravessa é `Disponivel/Reservar/Liberar/Consolidar`. O conteúdo existe espalhado (AD-5, AD-7, AD-16, tabela de travessias), mas o entregável medido pela SM-7 é o diagrama, e a SM-7 verifica que ele "corresponde à decomposição real do código".

Junto a isso, o grafo **mistura três naturezas de aresta sem distingui-las**: importação Go (a maioria), dependência de VIEW no banco (`busca --> catalogo`, que pelo AD-16 é leitura de VIEW e não import) e salto HTTP (o webhook, que nem aparece neste grafo). A distinção não é estética: no marco de extração do §11, um import vira chamada de rede, uma VIEW compartilhada vira problema de sincronização de dados, e um salto HTTP já está pronto. O diagrama que a banca vai ler apaga justamente essa diferença.

*Correção:* rotular as arestas com a operação que atravessa, e dar estilo diferente à aresta de VIEW. É o mesmo diagrama, com mais informação por pixel.

### BAIXO — B-12: a topologia não mostra duas coisas que o próprio documento decide

O `media/` servido pelo Go (AD-12) e a existência ou não de volume nomeado no Postgres (A-3) são decisões da espinha que não aparecem no único diagrama que descreve o `docker compose`. O primeiro é a costura do S3; o segundo é o que decide o determinismo do NFR-1.

---

## Achados consolidados

| ID | Grau | Achado | Onde |
|---|---|---|---|
| C-1 | **CRÍTICO** | Os quatro deveres da varredura atravessam `pedido` e `pagamento`, mas o AD-1 dá uma seta só e a árvore a coloca em `internal/pedido/`. O dever de **emitir** a confirmação é comportamento do Provedor: em `pedido`, ele viola o NFR-3 e a SM-5 | AD-1, AD-6, AD-8, Árvore |
| A-1 | **ALTO** | Não existe contrato de listagem paginada (envelope, total, índice de página) nem regra de desempate estável da FR-14, e quatro superfícies de três módulos precisam do mesmo | Convenções, AD-16, AD-18 |
| A-2 | **ALTO** | O teto de 3 Tentativas de Pagamento (§7.1, FR-27) não tem dono: nem a tabela exaustiva do AD-3, nem a porta do AD-8 | AD-3, AD-8 |
| A-3 | **ALTO** | Quem aplica `db/semente/`, quando, e o Postgres persiste entre subidas? O NFR-1 (SM-3) depende disso, e a promessa dos "mesmos `numero` de Pedido" não fecha com o glossário | Convenções, Ambientes |
| M-1 | MÉDIO | O AD-1, o invariante mais importante, é o único sem mecanismo de execução — enquanto o AD-12, menor, tem passo de CI | AD-1 |
| M-2 | MÉDIO | "Terminal" não é definido no AD-18 e as duas fontes discordam (PRD inclui `ENVIADO`; `EXPERIENCE.md` não) — decide se o Roteiro A passo 9 aparece na tela | AD-18 |
| M-3 | MÉDIO | O ERD desenha duas travessias de schema com a mesma notação de FK que o AD-2 proíbe, e não diz de que schema é cada entidade | ERD |
| M-4 | MÉDIO | `ErrTransicaoInvalida` precisa carregar as transições permitidas (FR-32 + `EXPERIENCE.md`); nem o AD-3 nem o AD-14 declaram | AD-3, AD-14 |
| M-5 | MÉDIO | Idempotência de `POST /pedidos`: o `INSERT` que falha precisa devolver o Pedido existente, não erro — desfecho diferente do webhook | AD-7 |
| M-6 | MÉDIO | O sinal `NAO_APLICAVEL_SINALIZADA` precisa chegar a duas superfícies administrativas, e nenhuma resposta administrativa é contratada | AD-7, AD-18 |
| M-7 | MÉDIO | Migrações com prefixo `NNNN` sequencial colidem com 2 a 4 pessoas em paralelo; goose tem versionamento por timestamp para isso | Convenções |
| M-8 | MÉDIO | O Adiado afirma folga "muito abaixo" dos 500 ms sobre uma consulta anterior ao AD-16, que acrescentou agregação de reservas à VIEW | Adiado, AD-16 |
| M-9 | MÉDIO | O grafo do AD-1 é declarado o entregável do §6.1 sem rotular o que atravessa cada fronteira, e mistura import Go, VIEW e HTTP na mesma aresta | AD-1, Mapa |
| B-1 | BAIXO | NFR-10 e NFR-11 sem menção, embora o *frontmatter* prometa NFR-1..NFR-16 (delegação correta à UX, não declarada) | *frontmatter* |
| B-2 | BAIXO | `numero` do Pedido descrito por adjetivos ("curto, legível, estável"), sem gerador declarado | AD-15 |
| B-3 | BAIXO | `carrinho --> identidade` num grafo declarado exaustivo, sem uso identificável | AD-1 |
| B-4 | BAIXO | Convenções "Commits" e as regras de cor de "Componentes de tela" são cerimônia ou transcrição da `DESIGN.md` | Convenções |
| B-5 | BAIXO | Os parâmetros do OWASP no AD-20 são detalhe de um módulo só (mantidos por serem piso de segurança) | AD-20 |
| B-6 | BAIXO | `CREATE EXTENSION pg_trgm` e a conta semeada do Administrador não aparecem em lugar nenhum | Convenções, Stack |
| B-7 | BAIXO | Cancelar Pedido já `CANCELADO` cai em "defeito de quem chamou", contra a FR-31 | AD-3 |
| B-8 | BAIXO | O `grep` do AD-12 não cobre `next/font/google`, a forma que o próximo desenvolvedor vai escrever | AD-12 |
| B-9 | BAIXO | Nenhuma seção de Questões em Aberto, apesar da Questão 3 bloqueadora do PRD | estrutura |
| B-10 | BAIXO | PostgreSQL 18.6 implica cadência de *minor* acima da usual; reconferir antes do README fixar a tag | Stack |
| B-11 | BAIXO | O exemplo JSON do AD-18 desce a formato completo; a tripla bastaria | AD-18 |
| B-12 | BAIXO | A topologia não mostra `media/` servido pelo Go nem o volume do Postgres | Semente Estrutural |

**Total:** 1 crítico · 3 altos · 9 médios · 12 baixos.

## Recomendação

Fechar C-1 e os três altos numa passada de correção — nenhum exige repensar decisão, todos são uma a três frases no AD que já existe. Os médios M-2, M-4, M-5 e M-6 são de custo idêntico e evitam retrabalho em `web/`, então vale levá-los na mesma passada. Feito isso, a espinha está pronta para dirigir a construção: ela é o tipo de documento em que se pode dividir seis frentes de trabalho sem uma reunião de reconciliação por semana, que é o único teste que interessa.
