# Azamon — do clone ao sistema rodando

Réplica do núcleo de comércio eletrônico da Amazon, trabalho de faculdade. Este arquivo
é o **setup**, em duas partes. As seções 1 a 4 levam qualquer pessoa de um `git clone` ao
sistema no ar e a um Pedido que chega sozinho até `ENTREGUE` — é o caminho que o NFR-1
mede, e não exige BMad, Node nem Go na máquina. Da seção 5 em diante é ferramenta de
quem vai **desenvolver**. O que o projeto *é*, e o que fazer depois, está em
[`HANDOFF.md`](HANDOFF.md).

## 1. Subir o Azamon

Pré-requisito **único**: [Docker Desktop](https://www.docker.com/products/docker-desktop/)
com Compose. Go, Node.js, PostgreSQL e Redis **não** se instalam na máquina — rodam nos
contêineres, nas versões exatas da tabela Stack da espinha de arquitetura.

```bash
git clone https://github.com/Cashnip/amazon-waddle azamon && cd azamon
docker compose up
```

**Quanto demora — medido, nunca estimado.** Em 2026-09-13, na estória 1.9, com o cache de
construção e as imagens base **apagados** antes do cronômetro (Docker 29.4.3, Windows 11):
**3 min 26 s** do `git clone` ao primeiro Produto na tela. Três segundos são o clone; o
resto é o `docker compose up` baixando as quatro imagens base e construindo as duas do
projeto. O teto do NFR-1 é 15 minutos. Da segunda vez em diante, com as imagens já
construídas, a subida inteira leva **16 s**.

Nada mais precisa ser baixado à mão: o `.env`, os 50 SVG dos Produtos, a fonte da
marca e o SQL da semente são versionados — é o que faz o clone limpo subir sem buscar
nada além das imagens base.

**Deu certo se**, em outro terminal:

```bash
curl http://localhost:8080/api/v1/saude     # {"status":"ok"} pela porta do Go
curl -s http://localhost:3000 | head -1     # a casca do Next respondendo
docker compose ps                           # os quatro no ar; azamon, postgres e redis `healthy`
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

1. Abra <http://localhost:3000>. É a página de conferência da casca, com dois links
   temporários no rodapé do último cartão: `/entrar` e um Produto semeado.
2. **Entre** por `/entrar`, com as credenciais do Comprador acima.
3. **Escolha o Produto com cuidado.** O Provedor Simulado decide pelos centavos do total
   (§7.1 do PRD): até `,89` ele aprova, e de `,90` a `,94` recusa — mas *recusar* é da
   Épica 5, então hoje um Pedido dessa faixa fica parado em `AGUARDANDO_PAGAMENTO`, para
   sempre e sem erro na tela. O Produto ligado na página de conferência é o **Fone de
   Ouvido Bluetooth Aurora, R$ 249,90** — ele cai justamente na faixa parada. Para ver o
   ciclo inteiro, troque o identificador na URL pelo de um Produto de centavos `,00`:

   <http://localhost:3000/produtos/a0ae8ff1-da13-5591-9291-4a4f1ce15383> — Caixa de Som
   Portátil Maré, R$ 189,00.
4. **Confirmar compra** leva a `/pedidos/<id>`, que se atualiza sozinha. Medido no mesmo
   ensaio: `AGUARDANDO_PAGAMENTO` → `PAGO` em 7 s (o Provedor Simulado confirma por
   webhook), e daí `EM_SEPARACAO` → `ENVIADO` → `ENTREGUE` de 30 em 30 s, **1 min 41 s**
   do clique ao fim. Nenhum passo é manual: quem move o tempo é a varredura do serviço Go.

Os intervalos são do `.env` (`AZAMON_CONFIRMACAO_ATRASO`, `AZAMON_ENTREGA_INTERVALO`) e
existem para caber numa apresentação.

## 3. Rede desconectada (NFR-15)

A demonstração tem de rodar com a rede fora. Duas verificações, uma automática e uma de
ensaio:

- **Automática, em todo build:** `web/scripts/verificar-offline.mjs` roda como `prebuild`
  e derruba `npm run build` — e junto a construção da imagem `web` — se aparecer
  `fonts.googleapis`, `@import url(http`, `//cdn.` ou `next/font/google` em qualquer
  arquivo de `web/`, nomeando arquivo, linha e padrão. É o portão do AD-12; não há CI.
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

## 4. Reiniciar, semear, armadilhas

`docker compose down -v` devolve o ambiente ao estado inicial: os mesmos 50 Produtos, com
os mesmos identificadores, e nenhum Pedido. Os identificadores da semente são derivados do
nome, então não mudam entre subidas — é o que permite ligar um Produto pela URL como a
seção 2 faz.

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

---

*Daqui para baixo é ferramenta de desenvolvimento. Quem só queria ver o sistema rodando já
pode parar.*

> O repositório versiona **o trabalho, não a ferramenta**. Depois de clonar você tem os
> documentos, mas não tem o BMad nem as skills: `_bmad/`, `.claude/` e `.agents/` estão
> no `.gitignore` porque somam mais de 2 MB de código de terceiros, reproduzível pelos
> comandos abaixo.

## 5. BMad Method 6.11.0

Pré-requisitos desta seção, que o caminho da seção 1 não pede:
[`uv`](https://docs.astral.sh/uv/) (`brew install uv`), que roda todo script do BMad, e
Node.js ≥ 20 com `npx` (`brew install node`), que roda os dois instaladores.

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

## 6. Skills de frontend (opcional)

As 13 skills de `Leonxlnx/taste-skill` estão travadas com hash em
[`skills-lock.json`](skills-lock.json). Restaure todas de uma vez:

```bash
npx skills experimental_install   # lê o skills-lock.json
npx skills list                   # deu certo se aparecem design-taste-frontend & cia.
```

**Elas não são usadas no Azamon** — estão no lock só para reprodutibilidade, e o motivo
da exclusão está no memlog da UX. Pular esta seção não quebra nada.

## 7. Plugins do Claude Code (opcional, pessoal)

Plugins são **de usuário**, não do repositório: instalar não muda nada aqui dentro e
nenhum é exigido pelo projeto. Dentro do Claude Code:

```
/plugin                                    # abre o gerenciador
/plugin marketplace add <owner/repo>       # ou direto, pelo marketplace
/plugin install <plugin>@<marketplace>
```

## 8. Convenções que pegam quem chega

- Todo script do BMad roda por `uv run`, **a partir da raiz**. Sem o prefixo, roda fora
  do ambiente e falha; de outro diretório, os caminhos relativos não resolvem.
- Português em tudo que sai: documento, interface, apresentação. Mensagens de commit em
  inglês, conventional commits.
- Nunca edite `SPEC.md` nem `mapa-de-capacidades.md` à mão — são derivados, e a edição é
  sobrescrita em silêncio no próximo `bmad-spec`.
- O resto das armadilhas está em `AGENTS.md` (carregado por agentes) e em `HANDOFF.md`.

## 9. Leia nesta ordem

1. [`HANDOFF.md`](HANDOFF.md) — estado, decisões travadas, bloqueios, próximo passo.
2. `_bmad-output/specs/spec-azamon/SPEC.md` — o contrato canônico; o `companions:` do
   frontmatter lista o que mais precisa ser lido.
3. O que o seu trabalho exigir, descendo por ali. Não releia tudo.

---

*Fez a primeira instalação limpa e algum comando acima não bateu? Corrija este arquivo —
vale mais que a memória de quem escreveu.*
