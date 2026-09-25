# Azamon — do clone ao sistema rodando

Réplica do núcleo de comércio eletrônico da Amazon, trabalho de faculdade. Este arquivo
é o **setup**, em duas partes. As seções 1 a 4 levam qualquer pessoa de um `git clone` ao
sistema no ar e a um Pedido que chega sozinho até `ENTREGUE` — é o caminho que o NFR-1
mede, e não exige BMad, Node nem Go na máquina. Da seção 5 em diante é ferramenta de
quem vai **desenvolver**. O que o projeto *é*, e o que fazer depois, está em
[`HANDOFF.md`](HANDOFF.md).

## 1. Subir o Azamon

Pré-requisitos: **Git** e **[Docker Desktop](https://www.docker.com/products/docker-desktop/)
com Compose** — e mais nada. Go, Node.js, PostgreSQL e Redis **não** se instalam na
máquina: rodam nos contêineres, nas versões exatas da tabela Stack da espinha de
arquitetura.

**No Windows, antes de clonar: pasta de caminho curto.** Os documentos de planejamento
têm caminhos de até 116 caracteres dentro do repositório, e o Git para Windows recusa
arquivo cujo caminho completo passe de 260. Numa pasta com mais de uns 140 caracteres o
clone termina com `Filename too long` e `Clone succeeded, but checkout failed`, com
arquivos faltando. Clone em algo como `C:\dev` — o `-c core.longpaths=true` abaixo cobre
o resto, e não muda nada fora do Windows.

```bash
git clone -c core.longpaths=true https://github.com/Cashnip/amazon-waddle azamon
cd azamon
docker compose up
```

**Modo demonstração é o padrão — e é o único que o compose sobe.** `docker compose up`,
sem argumento, sobe os quatro serviços (`web`, `azamon`, `postgres`, `redis`) com o
`.env` versionado, que é o modo demonstração do AD-13: ele encolhe os prazos para caberem
numa apresentação — a Tentativa de Pagamento expira em 60 s e cada etapa da entrega leva
30 s, onde os padrões de produção são 15 min e 24 h. Esses padrões moram no código, em
`internal/plataforma/config.go`, e não há arquivo nem perfil de produção para escolher.
O Provedor Simulado confirma em 5 s nos dois modos.

**A primeira construção usa a rede; o sistema em pé, não.** O `Dockerfile` roda
`go mod download` e o `web/Dockerfile` roda `npm ci`, os dois contra registradores
externos — o NFR-15 é sobre a demonstração rodando (seção 3), não sobre a primeira
construção. O que não precisa ser baixado à mão é o resto: o `.env`, os 50 SVG dos
Produtos, a fonte da marca e o SQL da semente são versionados.

**Deu certo se**, em outro terminal, depois que `docker compose ps` mostrar o serviço
`azamon` como `healthy` (o `web` não tem healthcheck, e até ele subir a porta 3000 recusa
conexão):

```bash
docker compose ps                          # os quatro no ar; azamon, postgres e redis `healthy`
curl http://localhost:8080/api/v1/saude    # {"status":"ok"} pela porta do Go
curl http://localhost:3000                 # a casca do Next respondendo
```

O arranque aplica as migrações e, uma vez só, o **Catálogo Semeado** — 50 Produtos em 5
Categorias, com as duas contas de demonstração abaixo. Não existe produção, então estas
credenciais não são segredo; elas estão versionadas junto com a lista, em `media/gerar.go`.

| Papel | E-mail | Senha |
|---|---|---|
| Comprador | `comprador@azamon.test` | `azamon-comprador` |
| Administrador | `admin@azamon.test` | `azamon-admin` |

### Registro de subidas

Quanto demora do `git clone` ao primeiro Produto na Vitrine — **medido, nunca estimado**.
O teto do NFR-1 é **15 min**. O número é dominado pela banda da máquina, porque a
primeira construção baixa as imagens base, os módulos Go e os pacotes npm: num link mais
lento ele sobe sem que nada tenha regredido. Da segunda subida em diante, com as imagens
já construídas, a pilha inteira volta em **16 s** (medido na 1.9 e de novo em 2026-09-25).

| Data | Quem | Máquina · Docker · cache | Tempo frio · morno | O que não bateu |
|---|---|---|---|---|
| 2026-09-13 | `Cashnip`, na estória 1.9 | Windows 11 · Docker 29.4.3 · `docker builder prune -af` e imagens base apagadas antes do cronômetro | **3 min 26 s** — 3 s de clone, 203 s de `docker compose up` · 16 s | nada |
| 2026-09-25 | agente Claude Code, na estória 7.1, na máquina do time | Windows 10 Pro · Docker Desktop 29.2.0 · builder BuildKit próprio (`docker-container`, v0.32.2) criado vazio; `postgres:18.6` e `redis:8.10.1` já estavam baixados (pull morno) | **4 min 28 s** — 4 s de clone, 264 s de `up --build`, 4 s até a Vitrine responder · 16 s | O primeiro clone, numa pasta de 156 caracteres, parou em `Filename too long` — daí o aviso de Windows acima. 86 s dos 264 s foram exportar a imagem `web` do builder próprio para o Docker, custo que o builder padrão não tem. |
| *em aberto* | **pessoa de fora — dono a nomear pelo time** | | | A SM-3 pede alguém que não escreveu o código seguindo só este arquivo. As duas linhas acima são de quem escreveu, e não fecham essa metade. |

As duas linhas não se comparam entre si: a de 2026-09-13 apagou todas as imagens base, e a
de 2026-09-25 manteve o pull de `postgres` e `redis` morno e pagou a exportação do builder
próprio. Daqui em diante, meça pela receita abaixo e anote o que dela você mudou.

**A regra: uma linha nova por semana, até a entrega, cada semana por um integrante
diferente, com o nome dele na coluna "Quem".** Uma subida verificada em setembro não prova
nada em dezembro. Para medir com cache frio sem apagar o cache de quem desenvolve, use um
builder descartável, com a pilha de desenvolvimento parada (as portas 3000 e 8080 são as
mesmas). O cronômetro vai do `git clone` até a Vitrine responder com o preço de um Produto:

```bash
docker buildx create --name azamon-medicao --driver docker-container
inicio=$(date +%s)
git clone -c core.longpaths=true https://github.com/Cashnip/amazon-waddle azamon-medicao
cd azamon-medicao
BUILDX_BUILDER=azamon-medicao docker compose -p azamon-medicao up -d --build
until curl -s http://localhost:3000 | grep -qF 'R$'; do sleep 1; done
echo "$(( $(date +%s) - inicio )) s"
docker compose -p azamon-medicao down -v --rmi local
cd .. && rm -rf azamon-medicao && docker buildx rm azamon-medicao
```

No PowerShell, a mesma receita:

```powershell
docker buildx create --name azamon-medicao --driver docker-container
$inicio = Get-Date
git clone -c core.longpaths=true https://github.com/Cashnip/amazon-waddle azamon-medicao
cd azamon-medicao
$env:BUILDX_BUILDER = 'azamon-medicao'; docker compose -p azamon-medicao up -d --build
while (-not ((curl.exe -s http://localhost:3000) -match 'R\$')) { Start-Sleep 1 }
"{0:N0} s" -f ((Get-Date) - $inicio).TotalSeconds
docker compose -p azamon-medicao down -v --rmi local; Remove-Item Env:BUILDX_BUILDER
cd ..; Remove-Item -Recurse -Force azamon-medicao; docker buildx rm azamon-medicao
```

Anote data, quem, máquina, versão do Docker, o tempo e o que não bateu. Se algum comando
destas seções não respondeu como o texto diz, a correção entra neste arquivo no mesmo
commit da linha.

## 2. O passeio do esqueleto

O que a Épica 1 entregou foi **um** Pedido atravessando o sistema inteiro. Desde a 5.6 o
mesmo passeio passa pelo caminho de verdade: Carrinho → Endereço → Revisão, e o Pedido
nasce do Carrinho, com Frete, Endereço e Reserva de Estoque.

1. Abra <http://localhost:3000>. É a Vitrine: os Produtos visíveis do Catálogo Semeado,
   20 por página, em cartões que levam à Página de Produto.
2. **Entre** por "Entrar", no canto da barra, com as credenciais do Comprador acima.
3. **Escolha o Produto pelo desfecho que quer ver.** O Provedor Simulado decide a primeira
   Tentativa de Pagamento pelos centavos do **total** — Subtotal mais Frete (§7.1 do PRD) —,
   com as faixas do `.env` (`AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS`,
   `AZAMON_PROVEDOR_RECUSADO_ATE_CENTAVOS`). Com uma unidade e um Endereço de SP:

   | Centavos do total | Desfecho | Um Produto que cai nela |
   |---|---|---|
   | `,00` a `,89` | aprova: `PAGO`, e daí sozinho até `ENTREGUE` | [Caixa de Som Portátil Maré](http://localhost:3000/produtos/a0ae8ff1-da13-5591-9291-4a4f1ce15383), R$ 189,00 — total R$ 204,00 |
   | `,90` a `,94` | recusa: `PAGAMENTO_RECUSADO`, e a nova Tentativa aprova | [Fone de Ouvido Bluetooth Aurora](http://localhost:3000/produtos/3400cd00-3f5e-5433-9171-fde099a52005), R$ 249,90 — total R$ 264,90 |
   | `,95` a `,99` | nenhuma confirmação: a Tentativa expira em 60 s e o Pedido vai a `PAGAMENTO_RECUSADO` | [Anotações de um Andarilho](http://localhost:3000/produtos/cc853ad1-cdb0-5621-aac5-d92fae625359), R$ 39,95 — total R$ 54,95 |

   O Frete é sempre em reais inteiros (R$ 15,00 no Sudeste, e grátis a partir de R$ 299,00
   de Subtotal), então os centavos do total são os dos Produtos — e **a quantidade os
   muda**: duas unidades do Andarilho somam R$ 79,90, e o total de R$ 94,90 cai na recusa,
   não na expiração. Da segunda Tentativa em diante o Simulado sempre aprova (AD-8), com
   teto de 3 Tentativas por Pedido. A Maré está na página 2 da Vitrine; os links acima
   levam direto à Página de Produto.
4. A Página de Produto mostra a Categoria no breadcrumb e, na Caixa de compra (à direita a
   partir de 1024 px), a disponibilidade, a quantidade e **"Adicionar ao Carrinho"**. Sem
   Sessão, esse botão leva ao Login e volta à mesma página com a quantidade escolhida. Um id
   inexistente ou de Produto desativado mostra "Este Produto não está disponível."
5. No **Carrinho** (ícone da barra), **"Fechar o Pedido"** leva ao passo **Endereço**:
   escolha um Endereço ou cadastre um — um CEP de SP, como `01310-100`, paga R$ 15,00 de
   Frete — e **Continuar**. A **Revisão** mostra os Itens do Carrinho, o Endereço, o
   Subtotal, o Frete e o total, todos vindos do Go.
6. **Confirmar Pedido** (o único botão laranja do fluxo) leva a `/pedidos/<id>`, e o
   contador do Carrinho na barra zera. A tela se consulta sozinha — a cada 3 s enquanto o
   Pedido aguarda pagamento, com o relógio do prazo, e a cada 10 s depois disso, com o
   Detalhe e a linha do tempo. Medido em 2026-09-25, pela API, no clone da tabela acima
   (Carrinho → checkout → Pedido): `AGUARDANDO_PAGAMENTO` → `PAGO` em 7 s (o Provedor
   Simulado confirma por webhook depois de `AZAMON_CONFIRMACAO_ATRASO`), e daí `SEPARANDO`
   → `ENVIADO` → `ENTREGUE` de 30 em 30 s: **`ENTREGUE` 1 min 39 s** depois do Confirmar
   Pedido. Nenhum passo é manual: quem move o tempo é a varredura do serviço Go. Na tela,
   cada mudança aparece até um intervalo de consulta (3 s ou 10 s) depois desses tempos.

Os outros dois desfechos, medidos na mesma sessão e também pela API:

- **Recusa (Aurora).** `PAGAMENTO_RECUSADO` em 7 s. A tela troca o relógio por "O Provedor
  de Pagamento recusou a Tentativa de Pagamento." e oferece **"Tentar pagar de novo"**
  enquanto restam Tentativas; a nova Tentativa chegou a `PAGO` 6 s depois de pedida e
  seguiu até `ENTREGUE` no mesmo ritmo.
- **Expiração (Andarilho).** Nenhuma confirmação chega; o relógio corre, e em 60 s
  (`AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO`) a varredura leva o Pedido a
  `PAGAMENTO_RECUSADO` com o motivo "Tempo de pagamento expirado.". Daí sai o mesmo
  "Tentar pagar de novo", enquanto restam Tentativas.

Em qualquer dos três, "Cancelar Pedido" existe enquanto o Pedido está em
`AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO` ou `SEPARANDO`, e libera a Reserva de
Estoque que houver. Os intervalos são do `.env` (`AZAMON_CONFIRMACAO_ATRASO`,
`AZAMON_ENTREGA_INTERVALO`, `AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO`) e existem para caber
numa apresentação.

### Esqueci a senha

Não existe serviço de e-mail (AD-20): o link de redefinição vai para o **log** do serviço
Go. Em "Entrar", siga **"Esqueci minha senha"**, informe `comprador@azamon.test` e envie — a
tela responde o mesmo "Solicitação registrada" para qualquer e-mail, exista a conta ou
não. Em outro terminal, na pasta do clone:

```bash
docker compose logs --no-log-prefix azamon | grep redefinir-senha
# PowerShell: docker compose logs --no-log-prefix azamon | Select-String redefinir-senha
```

Cada linha é um JSON, com `"msg":"redefinição de senha solicitada"` e o campo
`"caminho":"/redefinir-senha/<token>"`. Abra `http://localhost:3000` seguido desse caminho:
é a tela "Criar senha nova". Depois de **"Redefinir senha"** a tela volta a "Entrar", a
senha velha deixa de valer e toda Sessão aberta da conta é encerrada. O link vale 30 min
(`AZAMON_SENHA_TOKEN_VALIDADE`), serve uma vez só, e pedir outro invalida o anterior — use
sempre a **última** linha do log. `docker compose down -v` devolve a senha
`azamon-comprador`.

### Meus endereços

Autenticado, o Comprador cadastra para onde a entrega vai em
<http://localhost:3000/enderecos> — lista, cadastro, edição e remoção, com o vazio
"Nenhum Endereço cadastrado." O Menu da conta (abaixo) linka para cá.

| Rota | O que faz |
|---|---|
| `GET /api/v1/enderecos` | A lista do dono, em ordem de criação. É o "escolher" da FR-5 — não há Endereço padrão, e quem escolhe sobre a lista é o checkout da Épica 5. |
| `POST /api/v1/enderecos` | Cadastra. **201** com o Endereço criado. |
| `PUT /api/v1/enderecos/{id}` | Reescreve a etiqueta inteira. **200**. |
| `DELETE /api/v1/enderecos/{id}` | Apaga de verdade. **204**, e nenhum Pedido muda — o Pedido congela o Endereço na criação (AD-3). |

Três respostas que um cliente precisa tratar:

- **400 `CAMPO_INVALIDO`**, com `dados.campo`, para campo em falta, CEP fora dos oito
  dígitos, UF que não é uma das 27 siglas, ou campo acima de `AZAMON_ENDERECO_TEXTO_MAX`
  (120 caracteres). Um campo por resposta — a tela põe o foco nele.
- **409 `LIMITE_DE_ENDERECOS`** quando a conta já tem `AZAMON_ENDERECO_POR_COMPRADOR_MAX`
  (**20**) Endereços. Não é um campo que está errado, é a conta que está cheia: a saída é
  remover um, e a mensagem diz isso com o número configurado.
- **404 `NAO_ENCONTRADO`** para Endereço de outro Comprador, e é o **mesmo** envelope, byte
  a byte, de um `uuid` que nunca existiu — a posse entra na consulta, não numa checagem
  depois (AD-11). Responder diferente contaria que aquele Endereço existe.

O CEP é guardado em oito dígitos, sem hífen; a máscara é da tela, e a API aceita as duas
formas na entrada.

### Menu da conta, Perfil e Meus pedidos

A porta única para a conta (UX-DR9): autenticado, o cabeçalho de toda tela pública/
Comprador mostra "Olá, `<nome>`" com um `DropdownMenu` do shadcn (UX-DR1, sem alteração)
para Meus pedidos, Meus endereços, Perfil e Sair; sem Sessão, mostra Entrar (preservando
o destino) e Criar conta.

- <http://localhost:3000/perfil> — o e-mail em leitura, e "Trocar senha" leva ao
  `/esqueci-a-senha` existente (FR-3). Nenhuma FR pede troca de senha com a senha atual.
- <http://localhost:3000/pedidos> — **Meus pedidos**: os Pedidos do dono, mais recente
  primeiro, 20 por página, cada linha com número, data, total e o selo do Status do
  Pedido, e um link para o Detalhe. Enquanto a primeira resposta não chega há `Skeleton`;
  conta sem Pedido mostra "Você ainda não fez nenhum Pedido." com a volta à Vitrine.
- `/pedidos/{id}` — o **Detalhe do Pedido**, a mesma tela do passo 6: o relógio enquanto
  aguarda pagamento, o motivo e "Tentar pagar de novo" quando recusado, e depois disso os
  Itens de Pedido, o Endereço congelado, Subtotal, Frete, total e a linha do tempo, uma
  linha por mudança de Status. "Cancelar Pedido" abre um `Dialog` que nomeia o Pedido; em
  `ENVIADO` e `ENTREGUE` o botão não existe, e uma frase diz por quê.

| Rota | O que faz |
|---|---|
| `GET /api/v1/pedidos` | A lista do dono no envelope de listagem (`itens`, `pagina`, `por_pagina`, `total`), mais recente primeiro. O dono entra na própria consulta (AD-11). |
| `GET /api/v1/pedidos/{id}` | O Detalhe, numa chamada só: Status, `pode_cancelar`, `expira_em`, `tentativas_restantes`, Itens, Endereço e histórico. Pedido de outro Comprador é o mesmo **404** de um que não existe. |
| `POST /api/v1/pedidos/{id}/tentativas` | A nova Tentativa de Pagamento do Pedido recusado, sem corpo. **201** com o Pedido de volta a `AGUARDANDO_PAGAMENTO`; **409** `TETO_DE_TENTATIVAS` depois da terceira, e `ESTOQUE_INSUFICIENTE` se um Item acabou. |
| `POST /api/v1/pedidos/{id}/cancelamento` | O cancelamento pelo Comprador, sem corpo. **200** com o Pedido em `CANCELADO`; **409** `FORA_DA_JANELA_DE_CANCELAMENTO` em `ENVIADO` e `ENTREGUE`. |

O caminho de compra do passo 5 também é da API: `GET /api/v1/carrinho`, `POST`,
`PATCH /{id}` e `DELETE` (um Item ou todos) sob `/api/v1/carrinho/itens`,
`POST /api/v1/checkout/entrada`, que revalida o Carrinho ao entrar no checkout,
`GET /api/v1/frete?endereco_id=…` e `POST /api/v1/pedidos`, com o total visto na Revisão e
o cabeçalho `Idempotency-Key`.

Visitar `/perfil` ou `/pedidos` sem Sessão redireciona a `/entrar?destino=...` e volta
exatamente para lá depois de autenticar — o mesmo `paraLogin()` de `meus-enderecos.tsx`.

### Entrar como Administrador

Os dois papéis do FR-4 são **tabelas separadas**, e cada login consulta só a sua: a conta
de Administrador não entra por `/entrar`, e a de Comprador não entra pela porta abaixo. A
credencial já estava na tabela desde a primeira migração — o que faltava era a porta.

```bash
curl -i -X POST http://localhost:8080/api/v1/admin/sessoes \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@azamon.test","senha":"azamon-admin"}'
```

O `Set-Cookie` da resposta é a Sessão de Administrador, e com ela
`curl -b azamon_sessao=<token> http://localhost:8080/api/v1/admin/sessao` responde 200.

A área administrativa fica em **`/admin`**: `/admin/entrar` é o login, e `/admin` leva à
tela **Vendedores** (`/admin/vendedores`), onde o Administrador cria, renomeia, desativa,
reativa e remove Vendedores. Desativar esconde os Produtos do Vendedor na Página de Produto
e na compra sem tocar em Pedido nenhum; Vendedor com Produtos não é removível, e a recusa
oferece desativar no lugar. Sem Sessão de Administrador, `/admin/...` cai em `/admin/entrar`.

Ao lado ficam **Categorias** (`/admin/categorias`: criar, renomear e remover; Categoria com
Produtos não é removível, e a recusa diz quantos estão vinculados) e **Produtos**
(`/admin/produtos`: a lista paginada, com ativos e inativos, e criar, editar, desativar e
reativar — Produto não se remove). O Estoque total entra na criação e, depois, só pelo
**"Ajustar Estoque"** da linha, que tem rota própria (`PUT /api/v1/admin/produtos/{id}/estoque`)
e recusa com **409** `ESTOQUE_COMPROMETIDO` um total abaixo das unidades reservadas em
Pedidos abertos, dizendo quantas são. A imagem é escolhida entre as de `media/`, ou nenhuma.
Produto criado abre em `/produtos/{id}` na hora; desativado, some da Página de Produto e da
compra, e o Pedido já feito não muda. Abaixo de 640 px, os destinos da área administrativa
ficam num menu lateral.

E há **Pedidos** (`/admin/pedidos`): a Tabela de Pedidos de todos os Compradores, filtrável
por Status do Pedido e ordenável por data, com o estado na URL e consulta a cada 10 s
enquanto houver Pedido não terminal — não há busca por número. A linha e o Detalhe
(`/admin/pedidos/{id}`, com o Comprador e os Itens de Pedido) têm os botões que cabem ao
Administrador — "Iniciar a separação", "Registrar o envio" e "Registrar a entrega" —, cada
um confirmado num `Dialog` que nomeia o Pedido e o Status destino. O Administrador **não
cancela** Pedido: não há botão nem rota para isso. Com
`AZAMON_ENTREGA_SIMULACAO_ATIVA=true` (o padrão do `.env`), a simulação de entrega faz as
mesmas três transições de 30 em 30 s; quem chegar primeiro vence, e o outro recebe um aviso
informativo com o Status atual, não um erro. Pedido cancelado cujo pagamento foi aprovado
mesmo assim ganha o marcador "Pagamento aprovado" na Tabela e um aviso no Detalhe.

| Rota | O que faz |
|---|---|
| `GET /api/v1/admin/pedidos` | A Tabela: `?status=`, `?ordenacao=` e `?pagina=`, no envelope de listagem, com `permitidas` por linha. |
| `GET /api/v1/admin/pedidos/{id}` | O Detalhe administrativo, com o Comprador. |
| `POST /api/v1/admin/pedidos/{id}/transicoes` | Corpo `{"de":"PAGO","para":"SEPARANDO"}`. **200** com o Pedido; **409** `TRANSICAO_INVALIDA` (com `dados.permitidas`) fora da tabela do AD-3, e `ESTADO_JA_AVANCADO` quando a simulação chegou antes. |

A guarda é do Go: tudo sob `/api/v1/admin/` exige a Sessão de Administrador, e quem não a tem recebe o mesmo
**404** de uma rota que não existe (a área não aparece para quem não é Administrador,
UX-DR9). A exceção é a porta acima: `POST /api/v1/admin/sessoes` fica **fora** da guarda,
senão ninguém entraria, e por isso ela responde **401** a quem erra a credencial.

Do outro lado, a Sessão de Administrador leva **401** em toda rota da loja que pede Sessão
de Comprador:

- `GET /api/v1/sessao`;
- `GET` e `POST /api/v1/pedidos`, `GET /api/v1/pedidos/{id}`, e os `POST` de
  `…/pedidos/{id}/tentativas` e `…/pedidos/{id}/cancelamento`;
- as quatro de `/api/v1/enderecos`;
- `GET /api/v1/carrinho` e as quatro de `/api/v1/carrinho/itens`;
- `POST /api/v1/checkout/entrada` e `GET /api/v1/frete`.

As rotas que não pedem Sessão nenhuma continuam servindo normalmente (`/api/v1/saude`,
`GET /api/v1/produtos`, `/api/v1/produtos/{id}`, `/api/v1/categorias`,
`/api/v1/media/{arquivo}`), e `DELETE /api/v1/sessao` encerra a Sessão de qualquer papel
com o mesmo **204** — sair é idempotente, e não é lugar de conferir papel.

## 3. Rede desconectada (NFR-15)

A demonstração tem de rodar com a rede fora. Duas verificações, uma automática e uma de
ensaio:

- **Automática, em todo build:** `web/scripts/verificar-offline.mjs` roda como `prebuild`,
  antes do `npm test`, e derruba `npm run build` — e junto a construção da imagem `web` — se aparecer
  `fonts.googleapis`, `@import url(http`, `//cdn.` ou `next/font/google` — ou um
  `middleware.ts`/`middleware.js`, que o Next 16 ignora calado — em qualquer arquivo de
  `web/`, nomeando arquivo, linha e padrão. É o portão do AD-12; não há CI. Ele varre o
  **código deste repositório**: `node_modules`, `.next` e `package-lock.json` ficam de
  fora, então o portão não diz nada sobre o que uma dependência de terceiro busca.
- **Ensaio completo:** com a pilha no ar, percorra a seção 2 de ponta a ponta com o Wi-Fi
  da máquina desligado. Tudo que o navegador carrega — a casca, a fonte, os SVG dos
  Produtos — sai dos contêineres.

**Por que desligar o Wi-Fi e não subir uma rede sem saída.** Um
`docker-compose.offline.yml` com `internal: true` seria mais curto e repetível, foi
escrito e **não funciona**: o Docker descarta a publicação de 3000 e 8080 sem avisar, e o
navegador do host fica sem como chegar. O arquivo não entrou no repositório. A estória 1.9
mediu por dentro dessa rede, com um contêiner de `curl`, e o esqueleto inteiro andou —
`AGUARDANDO_PAGAMENTO` → `ENTREGUE` em 1 min 51 s, com a fonte e o SVG do Produto
respondendo 200 e `https://example.com` sem nem resolver o nome. O relato está no
`addendum.md` §10, na entrada da 1.9.

## 4. Reiniciar a demonstração

`docker compose down -v` e `docker compose up` devolvem o ambiente ao estado inicial: os
mesmos 50 Produtos, com os mesmos identificadores, nenhum Pedido, nenhum Endereço, e as
senhas das duas contas de volta às da tabela da seção 1. O `-v` é o que apaga o volume do
banco; `docker compose down` sem ele só para os contêineres, e a próxima subida volta com
tudo que estava lá. Os identificadores da semente são derivados do nome, então não mudam
entre subidas — é o que permite ligar um Produto pela URL como a seção 2 faz.

É também o que devolve o Estoque. Cada Produto nasce com `estoque_total` 10; a Reserva de
Estoque segura a unidade desde a criação do Pedido e a consolida em `ENVIADO`, e só o
cancelamento, a recusa ou a expiração a devolvem. Depois de dez unidades do mesmo Produto,
a Página de Produto mostra "Indisponível" e o Carrinho recusa a próxima com **409**
`ESTOQUE_INSUFICIENTE`. `down -v` e suba de novo.

---

*Daqui para baixo é ferramenta de desenvolvimento. Quem só queria ver o sistema rodando já
pode parar.*

## 5. Testar e mexer na semente

As duas suítes rodam **na máquina**, e por isso pedem o que a seção 1 não pedia: Go 1.27
(`go.mod`) e Node.js ≥ 24 (`web/package.json`).

```bash
go test ./...            # suíte do Go; precisa do Docker no ar (testcontainers)
cd web && npm test       # node --test sobre os 9 arquivos de web/scripts/*.test.mjs
```

- `go test ./...` sobe um PostgreSQL de verdade por `testcontainers-go`; sem Docker no ar,
  falha. Não é a suíte completa do repositório — o `web/` tem a sua, acima.
- `npm test` roda `node --test` sobre `web/scripts/*.test.mjs` — nove arquivos (`busca`,
  `carrinho`, `casca`, `checkout`, `destino`, `endereco`, `pedido`, `preco`, `quantidade`),
  137 testes em 2026-09-25, sem Docker. Eles prendem as funções puras de
  `web/lib/`; o de `casca` prende também o `rewrites()` de `/api` e as duas faces da guarda
  offline da seção 3. Nenhum monta componente React: o que só existe dentro de um `.tsx`
  não tem teste no `web/`. O mesmo `npm test` roda no `prebuild`, então teste vermelho
  derruba a construção da imagem `web`.
- Mexeu na lista de Produtos? Ela mora em `media/gerar.go`, e `go run media/gerar.go`
  reescreve os SVG e o SQL da semente — nenhum dos dois se edita à mão. Para ver a semente
  nova, `docker compose down -v` e suba de novo: o marcador em `public.semente` faz o banco
  já semeado ignorar qualquer mudança.
- Para medir a busca em volume, `AZAMON_SEMENTE_GRANDE=true` no `.env` acrescenta 5.000
  Produtos de bancada no arranque, com marcador próprio. Não é o catálogo da demonstração,
  que continua com 50 — a medição do NFR-4 vive em `go test ./db -run TestMedicaoNFR4` e
  não precisa da variável.

> O repositório versiona **o trabalho, não a ferramenta**. Depois de clonar você tem os
> documentos, mas não tem o BMad nem as skills: `_bmad/`, `.claude/` e `.agents/` estão
> no `.gitignore` porque somam mais de 2 MB de código de terceiros, reproduzível pelos
> comandos abaixo.

## 6. BMad Method 6.11.0

Pré-requisitos desta seção, que o caminho da seção 1 não pede:
[`uv`](https://docs.astral.sh/uv/) (`brew install uv`), que roda todo script do BMad, e
Node.js ≥ 24 com `npx` (`brew install node`), que roda os dois instaladores.

```bash
npx bmad-method@6.11.0 install
```

Fixe a versão. O `@latest` hoje é 6.12.0 e os caminhos de configuração divergem do que
os documentos deste repositório assumem.

O instalador pergunta; responda **exatamente** isto, porque `_bmad/config.toml` não é
versionado — as respostas *são* a configuração do time:

| Pergunta | Resposta |
|---|---|
| Nome do projeto | `azamon` |
| Idioma dos documentos | `Portugues` |
| Pasta de saída | `_bmad-output` (padrão) |
| Módulos | `core` + `bmm` |
| IDE / agente | `claude-code` |

Seu nome e seu idioma de conversa vão para `_bmad/config.user.toml`, que é pessoal —
responda o que quiser.

**Deu certo se:**

```bash
uv run _bmad/scripts/memlog.py --help   # imprime o uso, sai 0
ls .claude/skills | grep -c '^bmad-'    # 49
```

## 7. Skills de frontend (opcional)

As 13 skills de `Leonxlnx/taste-skill` estão travadas com hash em
[`skills-lock.json`](skills-lock.json). Restaure todas de uma vez:

```bash
npx skills experimental_install   # lê o skills-lock.json
npx skills list                   # deu certo se aparecem design-taste-frontend & cia.
```

**Elas não são usadas no Azamon** — estão no lock só para reprodutibilidade, e o motivo
da exclusão está no memlog da UX. Pular esta seção não quebra nada.

## 8. Plugins do Claude Code (opcional, pessoal)

Plugins são **de usuário**, não do repositório: instalar não muda nada aqui dentro e
nenhum é exigido pelo projeto. Dentro do Claude Code:

```
/plugin                                    # abre o gerenciador
/plugin marketplace add <owner/repo>       # ou direto, pelo marketplace
/plugin install <plugin>@<marketplace>
```

## 9. Convenções que pegam quem chega

- Todo script do BMad roda por `uv run`, **a partir da raiz**. Sem o prefixo, roda fora
  do ambiente e falha; de outro diretório, os caminhos relativos não resolvem.
- Português em tudo que sai: documento, interface, apresentação. Mensagens de commit em
  inglês, conventional commits.
- Nunca edite `SPEC.md` nem `mapa-de-capacidades.md` à mão — são derivados, e a edição é
  sobrescrita em silêncio no próximo `bmad-spec`.
- O resto das armadilhas está em `AGENTS.md` (carregado por agentes) e em `HANDOFF.md`.

## 10. Leia nesta ordem

1. [`HANDOFF.md`](HANDOFF.md) — estado, decisões travadas, bloqueios, próximo passo.
2. `_bmad-output/specs/spec-azamon/SPEC.md` — o contrato canônico; o `companions:` do
   frontmatter lista o que mais precisa ser lido.
3. O que o seu trabalho exigir, descendo por ali. Não releia tudo.

---

*Fez a primeira instalação limpa e algum comando acima não bateu? Corrija este arquivo —
vale mais que a memória de quem escreveu.*
