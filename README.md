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

```bash
git clone https://github.com/Cashnip/amazon-waddle azamon && cd azamon
docker compose up
```

**Quanto demora — medido, nunca estimado.** Em 2026-09-13, na estória 1.9, com o cache de
construção e as imagens base **apagados** antes do cronômetro (Docker 29.4.3, Windows 11):
**3 min 26 s** do `git clone` ao primeiro Produto na tela. Três segundos são o clone; os
outros 203 s são o `docker compose up` **baixando** as quatro imagens base, os módulos Go
e os pacotes npm e construindo as duas imagens do projeto — ou seja, o número é dominado
pela banda da máquina, e num link mais lento ele sobe sem que nada tenha regredido. O teto
do NFR-1 é 15 minutos, com folga para isso. Da segunda vez em diante, com as imagens já
construídas, a subida inteira leva **16 s**.

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

## 2. O passeio do esqueleto

O que a Épica 1 entrega é **um** Pedido atravessando o sistema inteiro, com telas cruas:
não há busca, Carrinho nem checkout — eles são das Épicas 3 a 5.

1. Abra <http://localhost:3000>. É a Vitrine: os Produtos visíveis do Catálogo Semeado,
   20 por página, em cartões que levam à Página de Produto.
2. **Entre** por "Entrar", no canto da barra, com as credenciais do Comprador acima.
3. **Escolha o Produto com cuidado.** O Provedor Simulado decide pelos centavos do total
   (§7.1 do PRD): até `,89` ele aprova, de `,90` a `,94` recusa, e de `,95` a `,99` não
   manda confirmação nenhuma. Só a aprovação é emitida hoje — recusa e expiração são da
   Épica 5 —, então **qualquer** total acima de `,89` deixa o Pedido parado em
   `AGUARDANDO_PAGAMENTO`, para sempre e sem erro na tela. O **Fone de Ouvido Bluetooth
   Aurora, R$ 249,90**, por exemplo, cai justamente na faixa parada. Para ver o ciclo
   inteiro, abra um Produto de centavos `,00` — este está na página 2 da Vitrine:

   <http://localhost:3000/produtos/a0ae8ff1-da13-5591-9291-4a4f1ce15383> — Caixa de Som
   Portátil Maré, R$ 189,00.
4. **Confirmar compra** leva a `/pedidos/<id>`, que se atualiza sozinha. Medido no mesmo
   ensaio: `AGUARDANDO_PAGAMENTO` → `PAGO` em 7 s (o Provedor Simulado confirma por
   webhook), e daí `EM_SEPARACAO` → `ENVIADO` → `ENTREGUE` de 30 em 30 s, **1 min 41 s**
   do clique ao fim. Nenhum passo é manual: quem move o tempo é a varredura do serviço Go.

Os intervalos são do `.env` (`AZAMON_CONFIRMACAO_ATRASO`, `AZAMON_ENTREGA_INTERVALO`) e
existem para caber numa apresentação.

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
- <http://localhost:3000/pedidos> — a listagem mínima dos Pedidos do dono, mais recente
  primeiro, cada linha um link para `/pedidos/{id}`. É o esboço que a Estória 6.1
  substitui: sem filtro por Status, sem paginação e sem `Skeleton` (UX-DR20b).

| Rota | O que faz |
|---|---|
| `GET /api/v1/pedidos` | A lista do dono, mais recente primeiro. O dono entra na própria consulta (AD-11), e a ordem é `id DESC` — a chave é `uuidv7()`, ordenada no tempo por construção, então não precisa de coluna nova para isso. |

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
reativar — Produto não se remove). O Estoque total é informado só na criação, e a imagem é
escolhida entre as de `media/`, ou nenhuma. Produto criado abre em `/produtos/{id}` na hora;
desativado, some da Página de Produto e da compra, e o Pedido já feito não muda. Abaixo de
640 px, os destinos da área administrativa ficam num menu lateral.

A guarda é do Go: tudo sob `/api/v1/admin/` exige a Sessão de Administrador, e quem não a tem recebe o mesmo
**404** de uma rota que não existe (a área não aparece para quem não é Administrador,
UX-DR9). A exceção é a porta acima: `POST /api/v1/admin/sessoes` fica **fora** da guarda,
senão ninguém entraria, e por isso ela responde **401** a quem erra a credencial.

Do outro lado, a Sessão de Administrador leva **401** nas rotas da loja que pedem Sessão
de Comprador — `GET /api/v1/sessao`, `POST` e `GET /api/v1/pedidos`, e as quatro de
`/api/v1/enderecos`. As rotas que não
pedem Sessão nenhuma continuam servindo normalmente (`/api/v1/saude`,
`/api/v1/produtos/{id}`, `/api/v1/media/{arquivo}`), e `DELETE /api/v1/sessao` encerra a
Sessão de qualquer papel com o mesmo **204** — sair é idempotente, e não é lugar de
conferir papel.

## 3. Rede desconectada (NFR-15)

A demonstração tem de rodar com a rede fora. Duas verificações, uma automática e uma de
ensaio:

- **Automática, em todo build:** `web/scripts/verificar-offline.mjs` roda como `prebuild`
  e derruba `npm run build` — e junto a construção da imagem `web` — se aparecer
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

`docker compose down -v` devolve o ambiente ao estado inicial: os mesmos 50 Produtos, com
os mesmos identificadores, e nenhum Pedido. Os identificadores da semente são derivados do
nome, então não mudam entre subidas — é o que permite ligar um Produto pela URL como a
seção 2 faz.

É também o que devolve o Estoque: cada Produto nasce com `estoque_total` 10, e a Reserva
consolida em `ENVIADO`, então o décimo primeiro passeio sobre o mesmo Produto sai em 409
`ESTOQUE_INSUFICIENTE`. `down -v` e suba de novo.

---

*Daqui para baixo é ferramenta de desenvolvimento. Quem só queria ver o sistema rodando já
pode parar.*

## 5. Testar e mexer na semente

As duas suítes rodam **na máquina**, e por isso pedem o que a seção 1 não pedia: Go 1.27
(`go.mod`) e Node.js ≥ 24 (`web/package.json`).

```bash
go test ./...            # suíte do Go; precisa do Docker no ar (testcontainers)
cd web && npm test       # a guarda offline e o teste da casca
```

- `go test ./...` sobe um PostgreSQL de verdade por `testcontainers-go`; sem Docker no ar,
  falha. Não é a suíte completa do repositório — o `web/` tem a sua, acima.
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
