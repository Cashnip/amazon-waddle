---
title: Revisão de reconciliação — ARCHITECTURE-SPINE.md × prd.md
tipo: review-reconciliacao
insumo: _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md
alvo: _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md
apoio:
  - _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md
  - _bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/EXPERIENCE.md
  - _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/.memlog.md
created: 2026-09-05
---

# Revisão de reconciliação — a espinha contra o PRD

## Veredito

A espinha está estruturalmente sólida: os 16 ADs cobrem o núcleo do sistema — máquina de estados, Estoque, transação, idempotência, fronteira de pagamento, autorização — e nenhum deles contradiz o §5 (Não-Objetivos) ou o §8 (Restrições). **Mas ela deixou cair sete coisas com consequência arquitetural real, duas delas em caminho de demonstração**: a Regra de Frete não tem módulo dono, e ninguém decidiu o que dispara a confirmação do Provedor Simulado. Somam-se a isso um NFR inteiro sem cobertura (NFR-5, armazenamento de senha), uma suposição do §4.1 que exige um quinto contêiner ou uma decisão de entrega (FR-3), um interruptor exigido pela FR-33 que a lista fechada do AD-13 exclui, e um predicado de visibilidade de Produto sem dono único.

Nenhum achado exige reabrir um AD existente. Cinco dos sete se resolvem com uma linha em AD já existente; dois pedem um AD novo (Frete, emissão da confirmação simulada).

## Método

Varredura de FR-1 a FR-34 e NFR-1 a NFR-16, um por um, com o teste: *duas pessoas construindo isto independentemente poderiam escolher de forma incompatível, e a espinha não decide?* Requisito que é puro comportamento de tela (NFR-10, NFR-11, FR-6, FR-18) foi verificado e descartado como não-achado — vive na `EXPERIENCE.md`. Também percorridos o §5, o §6.3 e o §8 atrás de restrição silenciosa, e conferidas todas as referências cruzadas da espinha (§, FR, NFR, SM, invariantes do addendum).

---

## CRÍTICO

### C-1 — A Regra de Frete não tem módulo dono, nem schema, nem mecanismo de configuração

**Requisito:** FR-21 (cálculo do Frete), FR-19 (recalcular quando a revalidação muda o subtotal, *inclusive cruzando o limiar de isenção*), FR-20 (trocar Endereço recalcula), FR-22 (Frete discriminado na revisão), FR-23 (Frete congelado no Pedido). §3 define **Regra de Frete** como um dos 20 termos do glossário. Addendum §5: "vive como dado configurável, não como `if` espalhado no checkout".

**O que falta:** a palavra "Frete" aparece duas vezes na espinha — em AD-3 (conteúdo imutável do Pedido) e em AD-9 ("a Regra de Frete é a **única** função que arredonda"). Em nenhum lugar se diz:

1. **Qual dos seis módulos é dono dela.** Não é `pedido` por eliminação: a FR-19 exige o recálculo no Carrinho, e o AD-1 declara suas setas **exaustivas** — `carrinho → pedido` não existe, e `carrinho → identidade` existe mas não dá acesso a nenhuma tabela de faixa de CEP.
2. **Onde a tabela de faixas de CEP vive.** O ERD tem 13 entidades e nenhuma delas é a faixa de CEP. O AD-13 fecha a configuração nos 15 parâmetros do §7.1, dos quais só a *isenção* (R$ 299,00) é de Frete — a tabela faixa → região → valor não é nenhum dos 15.
3. **Quem chama.** O §9 diz que o checkout tem dois passos e que o Frete é bloco da Revisão; a Revisão é servida por quem?

**Consequência:** duas pessoas escolhem, com igual justificativa: (a) tabela `pedido.faixa_frete` com o cálculo em `internal/pedido` — e a FR-19 fica sem caminho legal; (b) um sétimo pacote `frete/` — proibido pelo AD-1 ("não se negocia um sétimo"); (c) constantes em Go no checkout — proibido pelo addendum §5 e pelo espírito do NFR-16; (d) cálculo duplicado em `carrinho` e em `pedido` — que é o modo de falha do NFR-13 ("o total exibido é sempre exatamente a soma dos componentes exibidos"), porque duas implementações arredondam diferente e a AD-9 diz que existe **uma** função que arredonda.

**Forma mínima de resolver:** um AD que dê a Regra de Frete a um dono nomeado, com a tabela de faixas como dado semeado e versionado (o mesmo veículo do Catálogo Semeado, o que também atende o "sempre o mesmo Frete para o mesmo CEP" da FR-21), e uma seta nova declarada no AD-1 para quem precisa consultá-la. Se o dono for `pedido`, a seta `carrinho → pedido` precisa ser aberta e justificada; se for `identidade` (dono do CEP), o cálculo mora ao lado do Endereço e nenhuma seta nova é preciso.

### C-2 — Ninguém emite a confirmação do Provedor Simulado

**Requisito:** FR-25 ("a resposta da requisição de checkout não espera o resultado do pagamento"), §7.1 parâmetro 10 (faixa `,95`–`,99` → **a confirmação nunca chega**), addendum §4 ("a confirmação chega fora da requisição do checkout, **mesmo no simulado**"), Roteiro B passo 5.

**O que falta:** a espinha descreve com precisão o lado *receptor* — AD-7 (webhook, inbox, restrição única), AD-6 (a varredura lê `ConfirmacoesNaoAplicadas` e aplica a transição), AD-8 (a porta tem `IniciarTentativa` e "a confirmação que chega por webhook"). O lado *emissor* não existe em lugar nenhum: **o que faz o `POST /api/v1/webhooks/pagamento` acontecer, quando, e com qual atraso?** O laço tracejado do diagrama de contêineres mostra que ele acontece, não o que o dispara.

**Consequência:** o caminho mais curto — e o que alguém escreve primeiro — é o adapter fazer o POST no fim de `IniciarTentativa`. Mas o AD-4 exige que `IniciarTentativa` receba a `tx` do caso de uso de checkout, que ainda não foi commitada: o handler do webhook abre outra conexão, não enxerga a Tentativa de Pagamento nem o Pedido, e grava uma confirmação órfã que a varredura pode aplicar antes ou depois da criação existir. O AD-2 tirou a FK que pegaria isso. As outras escolhas plausíveis são `time.AfterFunc` (viola o espírito do AD-6, "nenhum estado é alcançado por agendamento em memória" — morre no reinício do contêiner, exatamente o que a invariante 6 do addendum existe para impedir) e uma segunda varredura (um mecanismo a mais para o que o AD-6 diz ser um só).

**Além disso, o atraso da confirmação simulada não é nenhum dos 15 parâmetros do §7.1** — e ele precisa ser curto o bastante para a virada `AGUARDANDO_PAGAMENTO → PAGO` parecer imediata na banca (a tela consulta a cada 3 s, por `EXPERIENCE.md`) e diferente de zero para o assincronismo não virar teatro.

**Forma mínima:** declarar que a emissão é uma responsabilidade da própria varredura do AD-6 (que já roda a 1 s, já lê a tabela de Tentativas e já é idempotente) — "a varredura decide também qual Tentativa de Pagamento pendente já merece confirmação, e o Provedor Simulado a emite por HTTP" — mais um 16º parâmetro de atraso. Isso mantém "um mecanismo, quatro requisitos" virando "um mecanismo, cinco requisitos", e nada é emitido dentro de transação aberta.

---

## ALTO

### A-1 — NFR-5 (armazenamento de senha) não aparece na espinha

**Requisito:** NFR-5 — "senhas guardadas com função de derivação lenta e sal por usuário; verificação: inspeção do registro no banco, nenhuma senha legível nem hash simples de uma passagem". Reforçado pelo §8 (Segurança) e pela FR-1 ("nunca persistida em texto claro nem em hash reversível").

**O que falta:** as palavras *senha*, *hash*, *argon*, *bcrypt* não ocorrem uma vez na espinha. A tabela de Stack desce ao nível de `pgx v5.10.0` e `goose v3.28.0` e não nomeia a função de derivação; nenhuma convenção fixa parâmetros de custo. É o único NFR sem nenhum AD, convenção ou seção que o toque — nem sequer citado em `Binds`.

**Consequência:** a escolha entre `argon2id`, `bcrypt` e `scrypt` e seus parâmetros é feita por quem escrever `identidade` primeiro, sem registro — e o NFR-5 é verificado *por inspeção do banco* na entrega. Pior: a conta de Administrador semeada (suposição 11 do §16, FR-4) carrega um hash que precisa ter sido produzido pela mesma função com os mesmos parâmetros; se a semente for gerada antes da decisão, ela precisa ser refeita. Também é o tipo de item que a banca pergunta e que não tem resposta escrita em lugar nenhum.

### A-2 — FR-3: o token de redefinição não tem por onde ser entregue

**Requisito:** FR-3 e sua suposição: "o token é entregue por uma caixa de correio de desenvolvimento local, sem serviço externo de e-mail — o ambiente roda offline na demonstração". A FR-3 continua dentro do escopo (§6.1); o §6.3 apenas a coloca como **primeira** a ser cortada, o que não é o mesmo que estar cortada.

**O que falta:** a espinha declara a topologia como fechada — "os quatro serviços acima" — e o NFR-15 (AD-12) proíbe rede externa em tempo de execução. Nada diz como o Comprador vê o link do token. A seção "Adiado" também não registra a FR-3, então ela não está deferida com condição declarada.

**Consequência:** três construções incompatíveis, todas defensáveis: um quinto contêiner (Mailpit/MailHog — muda a topologia declarada e o "um comando" do NFR-1), o link impresso no log estruturado do AD-15 (que colide com o §8 Privacidade, ver M-6, e com "erro engolido é proibido"), ou uma tela de desenvolvimento em `/dev/emails` (que é regra fora dos seis módulos, roçando o AD-10). O Redis já guarda o token (memlog), o que só resolve o armazenamento — não a entrega.

**Forma mínima:** ou uma linha na topologia (quinto contêiner, aceito como custo), ou uma linha em "Adiado" com condição — "a FR-3 é a primeira do §6.3 a cair; enquanto existir, o token é devolvido na resposta em modo demonstração e o mecanismo real fica na fase 2".

### A-3 — FR-33 exige um interruptor que a lista fechada do AD-13 não comporta

**Requisito:** FR-33 — "a simulação **pode ser desligada por configuração**, para que o Administrador conduza tudo manualmente na apresentação".

**O que falta:** o AD-13 fixa "os **15** parâmetros do §7.1 são variáveis de ambiente" e o §7.1 tem apenas o *intervalo* entre etapas, não o liga/desliga. A espinha herda uma lista fechada de 15 e a FR-33 pede um 16º item que não é um limiar — é um booleano. O AD-6 ("a varredura é o único motor do tempo") também não menciona poder ser desligada parcialmente.

**Consequência:** ou alguém acha que a FR-33 já está coberta pelo intervalo (e a resposta na sala vira "coloca 24 h"), ou nasce uma constante/flag fora do AD-13, que é exatamente o defeito que o AD-13 nomeia. E o desligamento não é enfeite: o Roteiro C move um Pedido manualmente pelo painel enquanto a simulação do Roteiro A ainda roda — a colisão FR-32 × FR-33 está agendada dentro da apresentação (a própria `EXPERIENCE.md` a antecipa).

### A-4 — O predicado "Produto visível" não tem dono único

**Requisito:** FR-8 ("Vendedor desativado tem seus Produtos ocultados da vitrine **e da busca**"), FR-9 (Produto desativado), FR-12 ("Produtos de Vendedor desativado e Produtos desativados não aparecem nos resultados"), FR-6, FR-17 e FR-19 (não adicionável / sinalizado no Carrinho).

**O que falta:** o AD-16 diz que `busca` lê `catalogo` por VIEW, mas não diz que **a VIEW é a definição do que é visível**. A vitrine (FR-6) e a página de Produto (FR-7) são servidas por `catalogo`, não por `busca`, e portanto não passam pela VIEW; as guardas do Carrinho (FR-17/FR-19) também não. O predicado é `produto.ativo AND vendedor.ativo`, precisa valer em pelo menos quatro superfícies, e a espinha não o ancora em nenhum lugar.

**Consequência:** o modo de falha clássico — a busca filtra corretamente e a vitrine mostra o Produto de um Vendedor desativado, ou vice-versa. É invisível em teste unitário e aparece na demonstração. Uma linha no AD-16 ou no AD-2 ("a VIEW `catalogo.produto_visivel` é a única definição de Produto vendável, e a vitrine, o Carrinho e a busca leem dela") fecha o buraco sem AD novo.

---

## MÉDIO

### M-1 — NFR-14 (limites de entrada) não tem camada designada, e recusar × limitar não é decidido

**Requisito:** NFR-14 e §8 ("entradas do navegador são validadas no servidor; a validação no cliente é conveniência, nunca a defesa"). FR-12 (termo acima de 100 caracteres é **recusado**), FR-15 (tamanho de página acima do teto é **limitado ao teto, não obedecido**), FR-17 (teto de 10 unidades, verificado *independentemente* do Estoque), FR-9 (nome 200, descrição 4.000).

**O que falta:** a tabela de camadas diz que `api/` faz "rota, autenticação, DTO, erro→HTTP. **Nenhuma regra**" e que o domínio tem "**toda** a regra". Limite de entrada cai exatamente na costura: é regra ou é DTO? E as duas semânticas exigidas são opostas — a FR-12 recusa, a FR-15 silenciosamente limita. O AD-13 põe os tetos em configuração, mas configuração não é quem os aplica.

**Consequência:** cada handler resolve à sua maneira e o NFR-14 vira "verificado onde alguém lembrou" — o mesmo defeito que o AD-11 existe para prevenir na autorização, sem o AD correspondente na validação.

### M-2 — Contradição: AD-13 × AD-8 sobre o parâmetro 10 do §7.1

**Onde:** o AD-13 afirma "os 15 parâmetros do §7.1 são variáveis de ambiente". O parâmetro 10 do §7.1 **é** a regra de decisão do Provedor Simulado. O AD-8 afirma, na mesma espinha: "Nenhuma superfície de operador, **nenhuma variável para forçar recusa**".

**Consequência:** os dois ADs não podem ser obedecidos ao mesmo tempo. E o AD-8 é mais estrito do que os insumos: a FR-25 proíbe **superfície de operador** e reconfiguração no meio da demonstração, não variável de configuração; a `EXPERIENCE.md` diz explicitamente que "a faixa exata de centavos por desfecho é decisão de arquitetura e **vive na configuração do NFR-16**". A correção é de redação, no AD-8: proibir a superfície de operador e o gatilho manual, não a configuração.

### M-3 — O contrato da semente cobre um terço do que o PRD cobra dela

**Requisito:** NFR-1 (Catálogo Semeado determinístico, versionado e **plausível**), §16 nº 5 e NFR-4/SM-4 (conjunto ampliado de até **5.000 Produtos** só para medir a busca), FR-4/§16 nº 11 (o **Administrador nasce semeado**), §6.3 ("as contas da demonstração vêm do Catálogo Semeado").

**O que falta:** a árvore de origem tem um `db/semente/  # Catálogo Semeado determinístico e versionado` — um artefato só, escopado a catálogo. Faltam: a conta de Administrador (sem a qual o Roteiro C não começa e que depende de A-1), e o segundo conjunto de 5.000 Produtos, que é a única forma de o NFR-4 e a condição de revisita do Elasticsearch ("o NFR-4 falhar em medição") serem verificáveis. Também não se diz que o conjunto grande **não** sobe em `docker compose up` — o SM-C3 é explícito que 5.000 Produtos servem para medir, não para demonstrar.

### M-4 — NFR-8/SM-2: a espinha declara o runtime de teste, não onde o teste vive

**Requisito:** NFR-8 ("todo FR do fluxo busca → Carrinho → checkout → Pedido tem ao menos um teste que falha se a consequência declarada quebrar") e SM-2, que é métrica **primária**.

**O que falta:** a tabela de Ambientes dá o runtime ("Postgres efêmero por testcontainers-go, mais os testes puros em memória"), e o memlog registra o critério de escolha entre os dois — *"testcontainers só onde a transação É o objeto do teste"* — que **não sobreviveu à destilação**. A árvore "dentro de cada módulo, uma forma só" lista cinco itens e nenhum é teste. E o fluxo crítico do NFR-8 atravessa quatro módulos e a camada HTTP (o teste de concorrência do NFR-7 precisa disparar N requisições paralelas), logo não cabe em teste de módulo — a espinha não diz onde ele mora.

### M-5 — `USUARIO` no ERD viola a convenção de glossário da própria espinha

**Onde:** a convenção de identificadores diz "Português, **literalmente os 20 termos do §3 do PRD**. Sinônimo é defeito, em código como em tela". O ERD abre com `USUARIO ||--o{ ENDERECO` e `USUARIO ||--|| CARRINHO`. **"Usuário" não é um dos 20 termos** — o §3 tem Comprador e Administrador, e seu preâmbulo diz que sinônimo é violação de disciplina.

**Consequência:** é pequeno em código e grande em avaliação: a SM-7 verifica que "o diagrama de módulos corresponde à decomposição real" e a disciplina de glossário já foi violada e caçada duas vezes neste projeto (memlog). Resolve-se nomeando a tabela `comprador` com coluna de papel, ou registrando `Usuário` como 21º termo no addendum — o que a espinha não pode fazer sozinha.

### M-6 — §8 Privacidade não tem contraparte estrutural

**Requisito (silencioso — não é FR nem NFR):** §8 — "todos os dados de demonstração são fictícios; nenhum nome, e-mail, CPF ou endereço de pessoa real entra no Catálogo Semeado nem nas contas de teste" e "o sistema coleta apenas o que os FRs exigem: e-mail, senha e Endereço. Sem telemetria, sem rastreamento, sem terceiros".

**O que falta:** a metade "sem terceiros" está coberta de fato pelo AD-12 (nada de rede externa). A outra metade sumiu na estrutura de ADs: o AD-15 obriga log JSON em **toda linha de requisição** e não diz o que nunca pode entrar nele — senha, cookie de Sessão, token de redefinição, e-mail. E nada amarra a semente a dado fictício, que é a mesma linha que o NFR-1 já pede quando exige um catálogo "plausível". Uma frase no AD-15 e uma no contrato da semente (M-3) bastam.

### M-7 — Quem gera a `Idempotency-Key`, e com que estabilidade

**Requisito:** FR-23 — "confirmar duas vezes (duplo clique, reenvio do formulário) cria **um** Pedido, não dois"; NFR-12.

**O que falta:** a convenção exige o cabeçalho `Idempotency-Key` em `POST /pedidos` e o AD-7 dá o mecanismo no banco (restrição única). Nenhum dos dois diz quem **mina** a chave. Chave gerada no momento do clique não impede duplo clique — gera duas chaves e dois Pedidos, e a consequência testável da FR-23 falha com a idempotência inteira implementada corretamente do lado do servidor. A chave precisa nascer com a **tentativa de checkout** (uma por Revisão renderizada), não com o clique. Como o AD-10 declara o Next.js casca sem estado de domínio, vale dizer explicitamente que essa chave é estado de tela, não de domínio.

---

## BAIXO

- **B-1 — FR-1, normalização de e-mail.** "`Marina@x.com` e `marina@x.com` são a mesma conta", normalizado *antes* da verificação de unicidade. A espinha decidiu com cuidado a normalização da **busca** (AD-16: na escrita, coluna dedicada, e por quê) e não disse nada da normalização do e-mail — que é a mesma classe de decisão (normalizar na escrita × índice funcional × `citext`) com uma restrição única em jogo.
- **B-2 — Endpoint da consulta em intervalo.** A `EXPERIENCE.md` decide o mecanismo (3 s na tela de processamento, 10 s no detalhe e no painel, sem WebSocket), então não é omissão da espinha. O que fica sem decisão é a forma no servidor: recarregar `GET /api/v1/pedidos/{id}` inteiro a cada 3 s ou um recurso leve de status. Com a varredura a 1 s e vários avaliadores com abas abertas, a escolha aparece.
- **B-3 — `.env` versionado × §8.** O AD-13 versiona o `.env` com os valores de demonstração; o §8 diz que "credenciais e segredos de configuração não vivem no repositório". Não é contradição enquanto o `.env` só carregar limiares — mas a senha do Postgres e o segredo de Sessão precisam de um lugar dito, senão vão parar lá por gravidade.
- **B-4 — §6.3 não é registrado.** A seção "Adiado" da espinha é sobre adiamento **técnico** (Elasticsearch, S3, Stripe). A ordem de corte de **escopo** do §6.3 e, sobretudo, a lista "nunca cortar" (FR-19, FR-24, FR-34 e o teste de concorrência do NFR-7) não aparecem. Na prática a espinha já as protege — as três viraram AD-5, AD-6 e AD-7, que são invariantes e não itens de escopo — mas um leitor sob pressão vai procurar "o que pode esperar" na tabela "Adiado", e ela não diz o que não pode.
- **B-5 — FR-12, caracteres especiais.** "`%` e `_` tratados como texto literal: buscar '50%' procura o texto '50%', não devolve o catálogo inteiro". Com `pg_trgm` sobre `ILIKE`, é a diferença entre um resultado e o catálogo inteiro. Cabe numa linha do AD-16.
- **B-6 — FR-2, expiração por inatividade.** O §7.1 diz "expiração da Sessão por **inatividade**: 7 dias". TTL de Redis é expiração por criação; inatividade exige renovar o TTL a cada requisição autenticada. A convenção de Sessão diz "chave no Redis com TTL" e não diz que ele desliza.
- **B-7 — A coluna normalizada de busca mora em `catalogo`.** O AD-16 põe o índice GIN e a coluna sem acento na tabela de `catalogo` (porque "`busca` não tem tabelas"), mas não diz quem a preenche nem que ela sai junto quando o Elasticsearch entrar. É uma travessia de fronteira consciente que merece estar dita onde o AD-2 lista as travessias.

---

## Varredura de cobertura

| Requisito | Coberto por | Achado |
|---|---|---|
| FR-1 Cadastro | AD-11, `identidade` | B-1; senha → A-1 |
| FR-2 Autenticação e Sessão | Convenção de Sessão, Redis (`sessão · token · bloqueio`), AD-13 | B-6 |
| FR-3 Recuperação de senha | Redis (token) | **A-2** (entrega) |
| FR-4 Papéis e autorização | AD-11 | Administrador semeado → M-3 |
| FR-5 Endereços | AD-2 (congelamento em `pedido`) | — |
| FR-6 Vitrine · FR-7 Produto | `catalogo`, AD-14 (não encontrado) | A-4 |
| FR-8 Vendedores · FR-9 Produtos · FR-10 Categorias | `catalogo`, AD-12 (imagem), FK intra-schema | A-4, M-1 |
| FR-11 Estoque | AD-5 | — |
| FR-12..FR-15 Busca | AD-16, convenção de estado na URL, uuidv7 (desempate) | B-5, M-1 (teto de página) |
| FR-16..FR-18 Carrinho | AD-1 (`carrinho → catalogo`), AD-9, AD-11 | M-1 (teto de unidades) |
| FR-19 Revalidação | AD-5 | **C-1** (recálculo de Frete) |
| FR-20..FR-22 Endereço, Frete, Revisão | AD-9 (arredondamento único) | **C-1** |
| FR-23 Criação do Pedido | AD-3, AD-4, AD-7, tabela de congelamento | M-7 |
| FR-24 Reserva de Estoque | AD-5 (tudo-ou-nada, `FOR UPDATE`) | — |
| FR-25 Tentativa de Pagamento | AD-8 | **C-2** |
| FR-26 Aprovação · FR-27 Recusa | AD-3, AD-6, AD-7 | — |
| FR-28..FR-32 Máquina de estados, histórico, detalhe, cancelamento, painel | AD-3, AD-4, AD-6, AD-11, AD-15 | — |
| FR-33 Simulação de entrega | AD-6 | **A-3** (desligar) |
| FR-34 Expiração | AD-6, AD-3 | — |
| NFR-1 Um comando | Topologia, Ambientes, `db/semente` | M-3 |
| NFR-2 Fronteiras | AD-1, AD-2 | — |
| NFR-3 Pagamento trocável | AD-7, AD-8 | — |
| NFR-4 Desempenho da busca | AD-16, "Adiado" | M-3 (5.000 Produtos) |
| NFR-5 Senha | — | **A-1** |
| NFR-6 Autorização | AD-11, AD-10 | — |
| NFR-7 Concorrência | AD-5, AD-4 | M-4 (onde vive o teste) |
| NFR-8 Cobertura do fluxo crítico | Ambientes (testcontainers) | **M-4** |
| NFR-9 Rastreabilidade | AD-15 | — |
| NFR-10 Responsividade · NFR-11 Acessibilidade | `DESIGN.md`, `EXPERIENCE.md` | não-achado (comportamento de tela) |
| NFR-12 Idempotência | AD-7, convenção | M-7 |
| NFR-13 Precisão monetária | AD-9 | C-1 (duas implementações arredondariam) |
| NFR-14 Limites de entrada | AD-13 (valores) | **M-1** (aplicação) |
| NFR-15 Offline | AD-12 | A-2 |
| NFR-16 Parâmetros | AD-13 | A-3, M-2, C-1, C-2 |

## Referências verificadas — nenhuma quebrada

- `binds` do front-matter contra o índice do §4: FR-1..FR-5, FR-6..FR-11, FR-12..FR-15, FR-16..FR-19, **FR-20..FR-27 + FR-34**, FR-28..FR-33 — todos batem.
- "os 15 parâmetros do §7.1" — a tabela tem exatamente 15 linhas. ✔
- "os 20 termos do §3" — o glossário tem exatamente 20 verbetes. ✔ (mas ver M-5, `USUARIO`)
- "addendum, invariante 5" (AD-3, corrida Administrador × simulação) — é de fato a invariante 5. ✔
- "§5 do PRD proíbe relevância por pontuação" (AD-16) — §5, "não é um motor de busca". ✔
- "SM-5" em AD-7 e AD-8, "SM-C1" em Adiado, "bloqueio 2 do HANDOFF" (`HANDOFF.md:51`, rubrica do professor) — todos existem e dizem o que a espinha diz que dizem. ✔
- Contra o §5 (Não-Objetivos): nenhuma decisão da espinha contradiz um não-objetivo. O AD-16 honra "sem relevância por pontuação" explicitamente e "Não existe produção" honra "não precisa estar publicado".
- Blindagens do addendum §6 preservadas: `CATEGORIA ||--o| CATEGORIA : pai_opcional` no ERD (suposição 4) e o Vendedor congelado em `item_pedido` (suposição 2). ✔

## O que a espinha faz bem, e não deve ser tocado ao corrigir

O AD-7 (inbox, quebrando o ciclo `pedido ↔ pagamento` com uma restrição única em vez de código defensivo) e o AD-6 (um mecanismo para quatro requisitos, derivado do histórico) são as duas peças que respondem à pergunta central da banca — "e se o pagamento nunca confirmar?" — com estrutura em vez de promessa. O C-2 pede que a emissão entre **dentro** do AD-6, não que ele seja repensado.
