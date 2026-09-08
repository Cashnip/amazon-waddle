---
title: 'Estória 1.1 — Andaime do serviço: os quatro contêineres em um comando'
type: 'feature'
created: '2026-09-08'
status: 'done'
baseline_commit: '135c249fc705d070abf632fb65a0eef712103757'
review_loop_iteration: 0
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md'
  - '{project-root}/_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O repositório é greenfield — só documentos, nenhuma linha de código. Nada roda, e todas as outras 51 estórias dependem de um andaime que ainda não existe.

**Approach:** Criar a árvore de origem completa do AD-1, o `docker-compose.yml` dos quatro serviços em modo demonstração, e as quatro peças transversais de `internal/plataforma`: configuração única (AD-13), correlação e log (AD-15), tradução de erro (AD-14) e migrações goose embutidas (aplicadas no arranque, falhando alto). Um endpoint de saúde prova que o conjunto sobe. Nenhuma tabela de domínio.

## Boundaries & Constraints

**Always:**
- Versões literais da tabela Stack: Go `1.27.1` · PostgreSQL `18.6` · Redis `8.10.1` · Node `24.20.0` · goose `v3.28.0` · pgx `v5.10.0`.
- Roteamento por `net/http.ServeMux` da stdlib. Rotas em `/api/v1/<recurso-plural-em-português>`.
- Todo limiar em variável `AZAMON_`, lida **uma vez** no arranque para uma struct. Literal de limiar fora dela é defeito.
- Log por `log/slog` em JSON, com `correlacao` e `modulo` em toda linha. Nenhum dado pessoal.
- Erro sai no envelope `{"erro":{"codigo","mensagem","dados","correlacao"}}`, `codigo` em `SCREAMING_SNAKE`, `mensagem` em português. Erro engolido é defeito.
- Módulo de domínio nunca conhece HTTP; `api/` é só tradução.
- Migrações por goose a partir de `embed.FS`, aplicadas no arranque. Falha de migração derruba o processo.
- Identificadores de domínio nos 20 termos do §3 do PRD. Commits em conventional commits, em inglês.

**Ask First:**
- Adicionar qualquer dependência Go ou npm fora da tabela Stack.
- Renomear uma variável `AZAMON_` depois desta spec aprovada — outras estórias já dependerão dela.
- Alterar o grafo de arestas do AD-1.

**Never:**
- Criar tabela de domínio ou migração de schema (é a 1.3).
- `rewrites()`, shadcn, tokens de marca ou fonte (é a 1.2).
- Framework HTTP de terceiros, `time.Timer`/`time.After`, `float` em dinheiro, auto-migração, chave estrangeira cruzando schema.
- Regra de negócio em `api/` ou no Node. Qualquer recurso puxado da rede em tempo de execução.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| Saúde | `GET /api/v1/saude` | `200` com `{"status":"ok"}` | N/A |
| Correlação recebida | Requisição com `X-Correlation-Id: abc` | Mesmo valor no cabeçalho da resposta, no contexto e em toda linha de log | N/A |
| Correlação ausente | Requisição sem o cabeçalho | UUID gerado, devolvido no cabeçalho da resposta e logado | N/A |
| Erro sentinela | Handler devolve um erro sentinela registrado | Envelope de erro com `codigo`, `mensagem`, `dados` e `correlacao`, no status mapeado | Mapeamento num arquivo só |
| Erro desconhecido | Handler devolve erro não registrado | `500` com `codigo: "INTERNO"` e mensagem genérica | Detalhe só no log, nunca no corpo |
| Migração quebrada | Arquivo `.sql` inválido em `db/migracoes/` | Processo **não** sobe; sai com código diferente de zero e log de erro | Nunca servir com schema errado |
| Rota duplicada | Dois handlers no mesmo padrão | Pânico no arranque (comportamento do `ServeMux`) | Falha de subida, não de comportamento |

</frozen-after-approval>

## Code Map

Greenfield: tudo abaixo é criação. Fontes de leitura (não editar) entre parênteses.

- `go.mod` — module `github.com/Cashnip/amazon-waddle` (remoto do `origin`), `go 1.27.1`.
- `cmd/azamon/main.go` — binário único: carrega config, aplica migrações, monta o `ServeMux`, sobe o servidor. A varredura entra na 1.8.
- `internal/plataforma/config.go` — struct de configuração e o único leitor de `AZAMON_*`.
- `internal/plataforma/log.go` — `slog` JSON, `modulo` por logger filho.
- `internal/plataforma/correlacao.go` — middleware `X-Correlation-Id`: resolve ou gera, contexto, cabeçalho de resposta, logger.
- `internal/plataforma/erro/erro.go` — sentinela→(status, código) num arquivo só; escreve o envelope.
- `internal/plataforma/migracao.go` + `db/embutido.go` — o `//go:embed` mora em `db/` porque a diretiva não atravessa `..`; `Migrar` recebe o `fs.FS` e o erro derruba o arranque.
- `internal/{identidade,catalogo,busca,carrinho,pedido,pagamento}/<módulo>.go` — a interface pública de cada módulo, vazia por ora: é a única porta que outro módulo importa (AD-1).
- `internal/fronteira_test.go` — `package internal`; lê `go list -deps -json` e falha em aresta fora da tabela do AD-1.
- `api/rotas.go`, `api/saude.go`, `api/rotas_test.go` — registro do `ServeMux`, o handler de saúde e os testes das linhas 1 e 7 da matriz. Só tradução.
- `db/migracoes/.gitkeep`, `db/semente/.gitkeep`, `media/.gitkeep` — diretórios da árvore, ainda vazios.
- `Dockerfile`, `web/Dockerfile`, `.dockerignore` — multi-stage a partir de `golang:1.27.1` e `node:24.20.0`.
- `docker-compose.yml` — `web`, `azamon`, `postgres`, `redis`; volume nomeado no Postgres; `depends_on` com healthcheck.
- `.env` — os 15 parâmetros do §7.1 nos valores de demonstração, mais os 4 operacionais e os 3 de infraestrutura.
- `web/` — app Next.js 16.3.4 mínimo, só para o contêiner subir. A casca é da 1.2.
- (ler) `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md` §7.1 — os 15 parâmetros e seus valores.
- (ler) `.../ARCHITECTURE-SPINE.md` — AD-1 (grafo de arestas exaustivo), AD-13, AD-14, AD-15, AD-16, tabela Stack, árvore de origem.

## Tasks & Acceptance

**Execution:**
- [x] `go.mod` — inicializar o módulo e fixar pgx `v5.10.0` e goose `v3.28.0` — nenhuma outra dependência.
- [x] `internal/plataforma/config.go` — struct + leitura única de `AZAMON_*` com padrões de produção no código — AD-13 exige que o `.env` só carregue a demonstração.
- [x] `.env` — os 23 parâmetros abaixo, exatamente nestes nomes — outras estórias vão depender deles.
- [x] `internal/plataforma/log.go` e `correlacao.go` — `slog` JSON e o middleware — AD-15; o `Toast` da UX exibe a correlação, então ela precisa voltar no cabeçalho.
- [x] `internal/plataforma/erro/erro.go` — registro sentinela→status e escrita do envelope — AD-14, num arquivo só.
- [x] `internal/plataforma/migracao.go` — `embed.FS` + `goose.Up`, com falha fatal — nunca servir com schema errado.
- [x] `internal/<módulo>/<módulo>.go` (seis arquivos) — a porta pública vazia de cada módulo — materializa a fronteira antes que alguém a atravesse por acidente.
- [x] `internal/fronteira_test.go` — teste exaustivo do grafo do AD-1 — a regra só existe se tiver mecanismo.
- [x] `api/rotas.go`, `api/saude.go` — `ServeMux` e `GET /api/v1/saude` — a prova de que o conjunto sobe.
- [x] `cmd/azamon/main.go` — ordem de arranque: config → migrações → rotas → servir.
- [x] `Dockerfile`, `web/Dockerfile`, `.dockerignore`, `docker-compose.yml` — os quatro serviços nas versões fixadas.
- [x] `web/` — `create-next-app` mínimo em 16.3.4/React 19.2.8, sem rede em tempo de execução.
- [x] `internal/plataforma/*_test.go` — cobrir a matriz de I/O: correlação resolvida e gerada, envelope de erro conhecido e desconhecido, migração quebrada derrubando o arranque.
- [x] `.../addendum.md` + seu `.memlog.md` — registrar as quatro decisões das Design Notes — SM-7 exige que o addendum continue vivo.

**Variáveis do `.env` (modo demonstração):**

```
AZAMON_HTTP_ADDR=:8080
AZAMON_POSTGRES_DSN=postgres://azamon:azamon@postgres:5432/azamon?sslmode=disable
AZAMON_REDIS_URL=redis://redis:6379/0
AZAMON_AUTH_TENTATIVAS_MAX=5
AZAMON_AUTH_BLOQUEIO_DURACAO=15m
AZAMON_SESSAO_EXPIRACAO=168h
AZAMON_SENHA_TOKEN_VALIDADE=30m
AZAMON_BUSCA_TERMO_MAX=100
AZAMON_PAGINA_TAMANHO=20
AZAMON_PAGINA_TAMANHO_MAX=60
AZAMON_CARRINHO_UNIDADES_MAX=10
AZAMON_FRETE_ISENCAO_CENTAVOS=29900
AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS=89
AZAMON_PROVEDOR_RECUSADO_ATE_CENTAVOS=94
AZAMON_PAGAMENTO_TENTATIVAS_MAX=3
AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO=60s
AZAMON_ENTREGA_INTERVALO=30s
AZAMON_ENTREGA_SIMULACAO_ATIVA=true
AZAMON_PRODUTO_NOME_MAX=200
AZAMON_PRODUTO_DESCRICAO_MAX=4000
AZAMON_VARREDURA_INTERVALO=1s
AZAMON_CONFIRMACAO_ATRASO=5s
AZAMON_WEBHOOK_BASE_URL=http://azamon:8080
```

**Acceptance Criteria:**
- Dado um clone limpo, quando executo `docker compose up` sem argumento, então sobem exatamente `web`, `azamon`, `postgres` e `redis` nas versões da tabela Stack, sem nenhum outro serviço.
- Dado que o repositório é inspecionado, quando listo a árvore, então existem `cmd/azamon/`, os sete diretórios sob `internal/`, `api/`, `db/migracoes/`, `db/semente/`, `media/`, `web/`, `docker-compose.yml` e `.env`, e cada módulo de domínio tem seu `<módulo>.go`.
- Dado o código pronto, quando rodo `grep -rn` por literal de limiar fora de `config.go`, então não há nenhum — todo valor vem da struct.
- Dado que nenhuma migração existe ainda, quando o binário arranca, então o goose roda contra um `embed.FS` vazio sem erro e o servidor sobe.
- Dado o binário rodando, quando executo `go test ./...`, então o teste de fronteira passa e nenhum módulo de domínio importa `net/http`.

## Spec Change Log

## Design Notes

Quatro decisões que as fontes não fixavam e que voltam para o `addendum.md`:

1. **`internal/plataforma` é aresta permitida para todos.** A tabela do AD-1 não a lista, mas a espinha define plataforma como a camada de config/log/correlação/transação/erro — o teste de fronteira trata `plataforma` como universalmente importável, e todo o resto continua exaustivo.
2. **Nomes das variáveis `AZAMON_*`.** Nenhuma fonte nomeava uma sequer. Convenção: `AZAMON_<ÁREA>_<PARÂMETRO>`, duração como `time.Duration` (`15m`, `60s`), dinheiro com sufixo `_CENTAVOS`, booleano `true`/`false`.
3. **DSN e credenciais ficam no `.env` versionado.** Não existe produção (só demonstração e teste), então as credenciais de demonstração não são segredo. Segredo real, se um dia existir, não entra no repositório.
4. **`default_transaction_isolation` não é declarado no compose.** O padrão do PostgreSQL já é `READ COMMITTED`, que é o que o AD-4 exige; declarar cria uma segunda fonte de verdade que o NFR-7 não vigia.

Envelope de erro, esqueleto do tradutor:

```go
var registro = map[error]struct {
    status int
    codigo string
}{
    // preenchido pelas estórias que criam sentinelas
}
// erro desconhecido → 500 / "INTERNO", detalhe só no log
```

## Verification

**Commands:**
- `go build ./...` — expected: compila sem erro.
- `go vet ./...` — expected: sem achados.
- `go test ./...` — expected: tudo verde, incluindo `internal/fronteira_test.go`.
- `docker compose up -d` seguido de `docker compose ps` — expected: quatro serviços, todos `running`/`healthy`.
- `curl -i localhost:8080/api/v1/saude` — expected: `200`, corpo `{"status":"ok"}`, cabeçalho `X-Correlation-Id` presente.
- `curl -s -H 'X-Correlation-Id: teste-123' -i localhost:8080/api/v1/saude` — expected: o mesmo `teste-123` de volta no cabeçalho.
- `docker compose logs azamon | head -5` — expected: JSON com `correlacao` e `modulo` em cada linha.
- `docker compose down -v && docker compose up` — expected: sobe de novo do zero, sem estado residual.

## Suggested Review Order

**A sequência de arranque**

- Entrada: config → migrações → rotas → servir, separada de `main` para virar testável.
  [`main.go:45`](../../cmd/azamon/main.go#L45)
- Conexão exigida mesmo sem migração: subir com o banco inalcançável só adiaria a descoberta.
  [`migracao.go:24`](../../internal/plataforma/migracao.go#L24)
- A espera obedece ao contexto sem usar `time.After`, que o AD-6 proíbe.
  [`migracao.go:59`](../../internal/plataforma/migracao.go#L59)
- Sonda do healthcheck: a imagem é `scratch`, então quem sonda é o binário.
  [`main.go:87`](../../cmd/azamon/main.go#L87)

**As fronteiras do AD-1 e do AD-14**

- A tabela de arestas, exaustiva; `plataforma` só é destino de `api` e `cmd`.
  [`fronteira_test.go:20`](../../internal/fronteira_test.go#L20)
- Concatena `Imports`, `TestImports` e `XTestImports` — sem isso, `_test.go` fura a regra.
  [`fronteira_test.go:93`](../../internal/fronteira_test.go#L93)
- Registro é fatia ordenada, não mapa: iteração aleatória tornaria a tradução não determinística.
  [`erro.go:29`](../../internal/plataforma/erro/erro.go#L29)
- Mensagem do sentinela, nunca do erro embrulhado; desconhecido vira 500 sem vazar detalhe.
  [`erro.go:51`](../../internal/plataforma/erro/erro.go#L51)

**Configuração e observabilidade**

- Leitura única de `AZAMON_*`, com positividade e invariantes cruzadas acumuladas.
  [`config.go:50`](../../internal/plataforma/config.go#L50)
- `modulo` e `correlacao` injetados no handler, não por `With`: chave nunca duplica.
  [`log.go:16`](../../internal/plataforma/log.go#L16)
- Fronteira de confiança: correlação do cliente é recusada se longa ou fora do charset.
  [`correlacao.go:41`](../../internal/plataforma/correlacao.go#L41)

**Superfície HTTP**

- Catch-all para a rota inexistente: sem ele a API teria dois contratos de erro.
  [`rotas.go:15`](../../api/rotas.go#L15)

**Periféricos**

- O `.env` e o `config.go` amarrados nos dois sentidos — typo derruba a demonstração calado.
  [`env_test.go:18`](../../internal/plataforma/env_test.go#L18)
- Healthcheck do `azamon` pelo próprio binário, e `web` esperando por ele.
  [`docker-compose.yml:24`](../../docker-compose.yml#L24)
