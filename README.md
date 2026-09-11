# Azamon — ambiente de desenvolvimento

Réplica do núcleo de comércio eletrônico da Amazon, trabalho de faculdade. Este arquivo
é só o **setup**: como sair de um `git clone` para uma máquina que roda o projeto e as
skills. O que o projeto *é*, e o que fazer depois, está em [`HANDOFF.md`](HANDOFF.md).

> O repositório versiona **o trabalho, não a ferramenta**. Depois de clonar você tem os
> documentos, mas não tem o BMad nem as skills: `_bmad/`, `.claude/` e `.agents/` estão
> no `.gitignore` porque somam mais de 2 MB de código de terceiros, reproduzível pelos
> comandos abaixo.

## 1. Pré-requisitos

| Ferramenta | Para quê | Como instalar (macOS) |
|---|---|---|
| [`uv`](https://docs.astral.sh/uv/) | roda todo script do BMad | `brew install uv` |
| Node.js ≥ 20 + `npx` | roda os dois instaladores | `brew install node` |
| Docker + Compose | sobe os quatro serviços | [Docker Desktop](https://www.docker.com/products/docker-desktop/) |
| Git | óbvio | já vem no macOS |

Go, PostgreSQL e Redis **não** se instalam na máquina: rodam nos contêineres. As versões
exatas estão na tabela Stack da espinha de arquitetura.

## 2. BMad Method 6.11.0

```bash
git clone <url-do-repo> azamon && cd azamon
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

## 3. Skills de frontend (opcional)

As 13 skills de `Leonxlnx/taste-skill` estão travadas com hash em
[`skills-lock.json`](skills-lock.json). Restaure todas de uma vez:

```bash
npx skills experimental_install   # lê o skills-lock.json
npx skills list                   # deu certo se aparecem design-taste-frontend & cia.
```

**Elas não são usadas no Azamon** — estão no lock só para reprodutibilidade, e o motivo
da exclusão está no memlog da UX. Pular esta seção não quebra nada.

## 4. Plugins do Claude Code (opcional, pessoal)

Plugins são **de usuário**, não do repositório: instalar não muda nada aqui dentro e
nenhum é exigido pelo projeto. Dentro do Claude Code:

```
/plugin                                    # abre o gerenciador
/plugin marketplace add <owner/repo>       # ou direto, pelo marketplace
/plugin install <plugin>@<marketplace>
```

## 5. Rodando o projeto

```bash
docker compose up   # web (Next.js) · azamon (Go) · postgres · redis, em modo demonstração
go test ./...       # suíte completa
```

Depois do `up`: a casca em <http://localhost:3000> e a saúde do serviço em
<http://localhost:8080/api/v1/saude>. `docker compose down -v` devolve o ambiente ao
estado inicial. O README de 15 minutos que o NFR-1 exige é da Estória 1.9.

O arranque aplica as migrações e, uma vez só, o **Catálogo Semeado** — 50 Produtos em 5
Categorias, com as duas contas de demonstração abaixo. Não existe produção, então estas
credenciais não são segredo; elas estão versionadas junto com a lista, em `media/gerar.go`.

| Papel | E-mail | Senha |
|---|---|---|
| Comprador | `comprador@azamon.test` | `azamon-comprador` |
| Administrador | `admin@azamon.test` | `azamon-admin` |

`go test ./...` precisa do Docker no ar: o teste de schema sobe um PostgreSQL de verdade
por `testcontainers-go`. Mexeu na lista de Produtos? Ela mora em `media/gerar.go`, e
`go run media/gerar.go` reescreve os SVG e o SQL da semente — nenhum dos dois se edita
à mão. Para ver a semente nova, `docker compose down -v` e suba de novo: o marcador em
`public.semente` faz o banco já semeado ignorar qualquer mudança.

## 6. Convenções que pegam quem chega

- Todo script do BMad roda por `uv run`, **a partir da raiz**. Sem o prefixo, roda fora
  do ambiente e falha; de outro diretório, os caminhos relativos não resolvem.
- Português em tudo que sai: documento, interface, apresentação. Mensagens de commit em
  inglês, conventional commits.
- Nunca edite `SPEC.md` nem `mapa-de-capacidades.md` à mão — são derivados, e a edição é
  sobrescrita em silêncio no próximo `bmad-spec`.
- O resto das armadilhas está em `AGENTS.md` (carregado por agentes) e em `HANDOFF.md`.

## 7. Leia nesta ordem

1. [`HANDOFF.md`](HANDOFF.md) — estado, decisões travadas, bloqueios, próximo passo.
2. `_bmad-output/specs/spec-azamon/SPEC.md` — o contrato canônico; o `companions:` do
   frontmatter lista o que mais precisa ser lido.
3. O que o seu trabalho exigir, descendo por ali. Não releia tudo.

---

*Fez a primeira instalação limpa e algum comando acima não bateu? Corrija este arquivo —
vale mais que a memória de quem escreveu.*
