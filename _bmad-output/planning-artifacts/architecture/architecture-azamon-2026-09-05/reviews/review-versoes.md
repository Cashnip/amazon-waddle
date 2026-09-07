# Revisão de Verificação de Realidade — Stack e Afirmações Técnicas

**Alvo:** `ARCHITECTURE-SPINE.md`
**Data da verificação:** 2026-09-05 (toda linha abaixo foi checada na web nesta data)
**Escopo:** só fato verificável. Desenho, decomposição em módulos e regra de negócio ficam fora.

---

## Veredito

A espinha está factualmente sólida: **13 das 22 afirmações verificáveis foram confirmadas sem ressalva**, 5 confirmadas com ressalva material, 2 desmentidas e 4 não puderam ser confirmadas. O ponto que muda decisão é um só: **Next.js 16.2.7 está fora da linha que recebe correções de segurança**, e o security release de 25/08/2026 (duas falhas de severidade Crítica) só existe em 16.3.3 e acima.

O cabeçalho "*Verificado na web em 2026-09-05*" sobre a tabela Stack não se sustenta em duas das treze linhas.

---

## Tabela de verificação

### Stack — versões

| # | Item | Afirmado na espinha | Confirmado na web | Fonte | Veredito |
|---|---|---|---|---|---|
| 1 | Go | 1.27.1 | Existe. Lançada 2026-09-01. É a mais recente (1.27.0 em 2026-08-19). Corrige cgo, compilador, runtime, `database/sql`, `encoding/json`, `net/http`, `os` | [go.dev/doc/devel/release](https://go.dev/doc/devel/release) | ✅ **Confirmado** |
| 2 | PostgreSQL | 18.6 | Existe. Lançada 2026-08-13. É a mais recente da linha 18 (a 18.5 nunca foi publicada — regressão). Corrige 28 CVEs e >110 bugs | [postgresql.org/docs/current/release-18-6](https://www.postgresql.org/docs/current/release-18-6.html) · [anúncio](https://www.postgresql.org/about/news/postgresql-186-1711-1615-1519-1424-and-19-beta-3-released-3365/) | ✅ **Confirmado** |
| 3 | Redis | 8.10.1 | Existe. Lançada 2026-08-17. É a mais recente. É release de **segurança** — corrige CVE-2026-62356 (heap OOB no CMSketch), use-after-free em TLS, corrupção de memória no carregamento de RDB, falhas em Vector Sets | [github.com/redis/redis/releases/tag/8.10.1](https://github.com/redis/redis/releases/tag/8.10.1) · [redis.io 8.10 release notes](https://redis.io/docs/latest/operate/oss_and_stack/stack-with-enterprise/release-notes/redisce/redisos-8.10-release-notes/) | ✅ **Confirmado** |
| 4 | Node.js | 24.20.0 (LTS ativa) | Existe. Lançada 2026-08-26. É a última 24.x. v24 "Krypton" está de fato em **Active LTS** (v26 é Current, v22 é Maintenance). Suportada até 2028-04-30 | [nodejs.org/en/about/previous-releases](https://nodejs.org/en/about/previous-releases) · [release v24.20.0](https://github.com/nodejs/node/releases/tag/v24.20.0) | ✅ **Confirmado**, inclusive o rótulo "LTS ativa" |
| 5 | Next.js | 16.2.7 | Existe (2026-06). **Não é a estável atual**: 16.3.4 saiu em 2026-08-31. E **não contém** o security release de agosto/2026, cujas versões corrigidas são 16.3.3 e 15.5.24 | [nextjs.org/blog/august-2026-security-release](https://nextjs.org/blog/august-2026-security-release) · [release v16.3.4](https://github.com/vercel/next.js/releases/tag/v16.3.4) · [support policy](https://nextjs.org/support-policy) | ❌ **Desmentido** — ver CRÍTICO C1 |
| 6 | React | 19.2.7 | Existe (2026-06-01). Não é o último patch: 19.2.8 saiu em 2026-07-21 (só performance de decode em RSC). Sem CVE conhecido contra 19.2.7 — o fix do DoS de RSC de maio/2026 exige ≥ 19.2.6 | [release v19.2.7](https://github.com/facebook/react/releases/tag/v19.2.7) · [release v19.2.8](https://github.com/react/react/releases/tag/v19.2.8) · [Netlify changelog mai/2026](https://www.netlify.com/changelog/2026-05-08-react-nextjs-security-vulnerabilities/) | ⚠️ **Confirmado com ressalva** — ver BAIXO B1 |
| 7 | pgx | v5.10.0 | Existe. Lançada 2026-06-03. É a tag v5 mais recente. Requer Go 1.25+, suporta PostgreSQL 14+ | [github.com/jackc/pgx/tags](https://github.com/jackc/pgx/tags) · [pkg.go.dev](https://pkg.go.dev/github.com/jackc/pgx/v5) | ✅ **Confirmado** |
| 8 | sqlc | v1.31.1 | Existe. Lançada 2026-04-22. É a mais recente | [github.com/sqlc-dev/sqlc/releases](https://github.com/sqlc-dev/sqlc/releases) · [changelog](https://docs.sqlc.dev/en/latest/reference/changelog.html) | ✅ **Confirmado** |
| 9 | goose | v3.28.0 | Existe. Lançada 2026-09-02 (três dias antes da espinha). É a mais recente. **Não há v4 oficial** no `pressly/goose` — os `/v4` que aparecem no pkg.go.dev são forks de terceiros. Exige Go 1.26+ | [github.com/pressly/goose/tags](https://github.com/pressly/goose/tags) · [releases](https://github.com/pressly/goose/releases) | ✅ **Confirmado** |
| 10 | testcontainers-go | v0.44.0 | Existe. Lançada 2026-08-07. É a mais recente. Corrige alertas do Dependabot (grpc, OTel). Política: testa contra as duas últimas versões do Go | [release v0.44.0](https://github.com/testcontainers/testcontainers-go/releases/tag/v0.44.0) · [system requirements](https://golang.testcontainers.org/system_requirements/) | ✅ **Confirmado** |
| 11 | shadcn/ui | CLI, componentes copiados | O modelo de distribuição está correto (CLI + cópia para o repositório; CLI v4 desde mar/2026, faz scaffolding de projeto Next.js). Compatibilidade com Next.js 16 → ver linha 20 | [changelog shadcn/cli v4](https://ui.shadcn.com/docs/changelog/2026-03-cli-v4) | ✅ **Confirmado** (o modelo) |
| 12 | `golang.org/x/crypto/argon2` | Argon2id, `m=19456 t=2 p=1` | Pacote existe e expõe `IDKey(password, salt []byte, time, memory uint32, threads uint8, keyLen uint32) []byte` — que é o Argon2id. Mapeamento: `m`→`memory` (KiB), `t`→`time`, `p`→`threads` | [pkg.go.dev/golang.org/x/crypto/argon2](https://pkg.go.dev/golang.org/x/crypto/argon2) | ✅ **Confirmado** — ver BAIXO B3 |
| 13 | Roteamento HTTP | `net/http.ServeMux` da stdlib | Ver linha 15 | — | ✅ **Confirmado** |

### Afirmações técnicas feitas como fato

| # | Item | Afirmado na espinha | Confirmado na web | Fonte | Veredito |
|---|---|---|---|---|---|
| 14 | `uuidv7()` no PostgreSQL 18 | "nativo do Postgres 18 … **sem extensão**" | Correto. `uuidv7()` é função **core** do PG 18, conforme RFC 9562, com parâmetro `shift` opcional. Garante monotonicidade em intervalos sub-milissegundo (12 bits de fração de ms). Funciona como `DEFAULT` de coluna | [PG 18 release notes](https://www.postgresql.org/docs/current/release-18.html) · [Aiven](https://aiven.io/blog/exploring-postgresql-18-new-uuidv7-support) · [Sawada](https://masahikosawada.github.io/en/2025/09/04/UUIDv7-in-PostgreSQL/) | ✅ **Confirmado** |
| 15 | `net/http.ServeMux` | método + curinga de caminho, tornando chi desnecessário | Correto desde Go 1.22 e inalterado no 1.27. Padrão aceita `"GET /produtos/{id}"`, devolve **405 automático** quando só o método não bate, e `r.PathValue("id")` extrai o valor. Curinga tem que ser segmento inteiro; `{path...}` captura a cauda; padrões conflitantes causam pânico no registro | [go.dev/blog/routing-enhancements](https://go.dev/blog/routing-enhancements) · [go.dev/doc/go1.27](https://go.dev/doc/go1.27) | ✅ **Confirmado** |
| 16 | sqlc lê anotações do goose | "formato goose … fonte do sqlc" | Correto. sqlc entende `-- +goose Up`/`Down` nativamente e monta o schema a partir do histórico de migrações. Ferramentas suportadas: atlas, dbmate, golang-migrate, **goose**, sql-migrate | [docs.sqlc.dev — Modifying the database schema](https://docs.sqlc.dev/en/latest/howto/ddl.html) · [goose blog — Using sqlc and goose](https://pressly.github.io/goose/blog/2024/goose-sqlc/) | ✅ **Confirmado** |
| 17 | Interface de transação do sqlc | `DBTX` permite passar `pgx.Tx` explicitamente (`Reservar(ctx, tx pgx.Tx, …)`) | Correto no efeito. Com o driver pgx/v5 o sqlc gera `DBTX` e `WithTx`, e `queries.WithTx(tx)` recebe a transação aberta por `pool.Begin(ctx)`. **Nuance factual:** `WithTx` recebe o tipo **concreto** (`pgx.Tx`), não a interface `DBTX` — há issue aberta pedindo o contrário. Consequência: a assinatura do AD-4 acopla os módulos ao pgx | [docs.sqlc.dev — Using transactions](https://docs.sqlc.dev/en/latest/howto/transactions.html) · [issue #3603](https://github.com/sqlc-dev/sqlc/issues/3603) | ✅ **Confirmado com nuance** |
| 18 | Argon2id `m=19456, t=2, p=1` | "parâmetros do OWASP … sal de 16 bytes" | Os parâmetros estão **exatamente** na Password Storage Cheat Sheet atual, como a segunda de cinco configurações de defesa equivalente (`m=47104 t=1 p=1` · **`m=19456 t=2 p=1`** · `m=12288 t=3 p=1` · `m=9216 t=4 p=1` · `m=7168 t=5 p=1`). **O sal de 16 bytes não é recomendação do OWASP** — a cheat sheet não fixa comprimento de sal | [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) | ✅ **Confirmado** (parâmetros) / ⚠️ atribuição do sal — ver BAIXO B2 |
| 19 | `pg_trgm` + GIN | adequado para substring sem acento em ~5.000 linhas, alvo 500 ms p95 | Verdadeiro **pela escala**, não pelo índice. `pg_trgm` fornece a classe de operador GIN que atende `LIKE`, `ILIKE`, `~`, `~*`. Duas ressalvas documentadas: (a) `pg_trgm` **é extensão** (`CREATE EXTENSION pg_trgm`) — vem na imagem oficial do Postgres, mas exige o comando numa migração; (b) padrão com **menos de 3 caracteres não extrai trigrama** e degenera para varredura completa do índice | [PG 18 — pg_trgm](https://www.postgresql.org/docs/current/pgtrgm.html) · [thread pgsql-hackers sobre <3 caracteres](https://postgrespro.com/list/thread-id/1821635) · [pganalyze — GIN indexes](https://pganalyze.com/blog/gin-index) | ⚠️ **Confirmado com ressalva** — ver MÉDIO M2 |
| 20 | shadcn/ui × Next.js 16 + React 19 | afirmado como fato na convenção de componentes | **Não confirmado oficialmente.** A página de compatibilidade da própria shadcn/ui é "**Next.js 15** + React 19" (out/2024). O changelog de 2026 (jan a set) não tem nenhuma entrada sobre Next.js 16. Fontes de terceiros relatam uso com Next 16 + React 19 + Tailwind v4. Ressalva conhecida: com **npm** a resolução de peer deps pede flag; pnpm/bun/yarn não | [ui.shadcn.com/docs/react-19](https://ui.shadcn.com/docs/react-19) · [ui.shadcn.com/docs/changelog](https://ui.shadcn.com/docs/changelog) | ❓ **Não confirmado** — ver MÉDIO M1 |
| 21 | `rewrites()` encaminha `/api/*` e elimina CORS | "o navegador conhece uma origem só, então não há CORS nem `SameSite=None`" | Correto. A doc do Next.js 16.3.4 mantém `rewrites()` em `next.config.js`, com destino **externo** e curinga `/:path*`. Do ponto de vista do navegador é a mesma origem, então cookies do domínio do Next são enviados e não há preflight. **Ressalva documentada:** `rewrites` **não define cabeçalhos de proxy** (`X-Forwarded-For`/`X-Forwarded-Host`) e não é um proxy HTTP completo | [nextjs.org — rewrites](https://nextjs.org/docs/app/api-reference/config/next-config-js/rewrites) · [discussão #17325](https://github.com/vercel/next.js/discussions/17325) | ✅ **Confirmado com ressalva** — ver MÉDIO M4 |
| 22 | Next.js 16 exige Node 20.9+; Node 24.20.0 satisfaz e é LTS | afirmado nos dois lados | Correto nos dois lados. O guia oficial de upgrade fixa **Node.js 20.9.0 mínimo** (Node 18 fora), TypeScript 5.1+, Chrome/Edge/Firefox 111+, Safari 16.4+. Node 24.20.0 é a última 24.x e v24 está em Active LTS | [Upgrading to version 16](https://nextjs.org/docs/app/guides/upgrading/version-16) · [nodejs.org previous releases](https://nodejs.org/en/about/previous-releases) | ✅ **Confirmado** |

### Compatibilidade entre as escolhas

| # | Combinação | Verificado | Veredito |
|---|---|---|---|
| 23 | Next.js 16.x × React 19.2.7 | O guia oficial diz que o App Router do Next.js 16 usa o canary de React que traz as features do **19.2**. Fixar 19.2.7 está dentro da faixa | ✅ Compatível |
| 24 | Go 1.27.1 × goose v3.28.0 (mín. 1.26) × pgx v5.10.0 (mín. 1.25) × testcontainers-go v0.44.0 (duas últimas do Go) | Todos os mínimos são satisfeitos por 1.27.1 | ✅ Compatível |
| 25 | pgx v5.10.0 × PostgreSQL 18.6 | pgx declara suporte a PostgreSQL 14+ | ✅ Compatível |
| 26 | sqlc v1.31.1 × `DEFAULT uuidv7()` no DDL do PG 18 | Nenhuma fonte confirma que o parser da v1.31.1 (abr/2026) conheça o catálogo do PG 18. `uuidv7()` é apenas chamada de função em `DEFAULT` — sintaxe antiga, risco baixo — mas é asserção sem confirmação | ❓ **Não confirmado** — ver MÉDIO M3 |
| 27 | Next.js 16 × imagens servidas pelo Go atrás do rewrite (AD-12) | O Next.js 16 introduziu `images.dangerouslyAllowLocalIP`, que **bloqueia por padrão** a otimização de imagem cujo upstream resolva para IP privado/local. Em `docker compose`, o Go é `http://azamon:8080`, rede privada. Erro observado na comunidade: `400` com "url parameter is not allowed" / "upstream image … resolved to private ip" | ⚠️ **Armadilha real** — ver ALTO A2 |
| 28 | Next.js 16 × envelope operacional | Turbopack passou a ser **o padrão** em `next dev` e `next build`; `next lint` foi **removido**; `middleware.ts` virou `proxy.ts` e um `middleware.ts` remanescente é **ignorado sem erro** | ⚠️ Ajuste de scripts e CI — ver BAIXO B4/B5 |
| 29 | PostgreSQL 18.6 × índice GIN | A 18.6 corrigiu builds paralelos de GIN que gravavam `reltuples` inválido (podia chegar a `Infinity`/`NaN` e travar autovacuum). Banco nascendo em 18.6 não é afetado; um GIN criado em 18.0–18.4 pediria `ANALYZE` | ✅ Sem impacto — ver BAIXO B8 |
| 30 | Redis 8.10.1 × uso da espinha | O uso é `SET`/`GET` com TTL (sessão, token, contador). As correções da 8.10.1 tocam Vector Sets, CMSketch e carregamento de RDB — áreas não exercidas | ✅ Sem impacto |

---

## Achados

### 🔴 CRÍTICO

**C1 — Next.js 16.2.7 está fora da linha que recebe correção de segurança.**

O security release de **25/08/2026** corrigiu duas falhas de severidade **Crítica**:

- **RCE não autenticado na Image Optimization API via AVIF** ([GHSA-2xp9-vwfh-vxw4](https://github.com/vercel/next.js/security/advisories/GHSA-2xp9-vwfh-vxw4)) — falha no libheif usado pelo `sharp`, disparável quando o Next.js otimiza uma imagem AVIF controlada pelo atacante.
- **RCE não autenticado em servidores hospedados em Windows** ([CVE-2026-75604](https://www.cve.org/CVERecord?id=CVE-2026-75604)) — aplicações que usam Pages Router e App Router sem Cache Components. **Linux e macOS não são afetados.**

As versões corrigidas são **16.3.3** (Active LTS) e **15.5.24** (Maintenance LTS). Pela [política de suporte do Next.js](https://nextjs.org/support-policy), correções pousam na minor mais recente do major em Active LTS — ou seja, em **16.3.x**. **A linha 16.2.x não recebe backport e nunca receberá esses patches.**

*Exposição real do Azamon, para calibrar:* rodando Linux em contêiner, o CVE-2026-75604 não se aplica; e sendo demonstração offline, o vetor AVIF exige que uma imagem controlada pelo atacante chegue ao otimizador. O risco operacional é baixo — mas a decisão de fixar uma versão que sai da esteira de patches é um fato que a espinha registra como se fosse o contrário ("verificado na web").

**Correção:** fixar **Next.js 16.3.4** (2026-08-31), que além de conter os dois patches reabilita a otimização AVIF com o libheif já corrigido.

### 🟠 ALTO

**A1 — A nota "*Verificado na web em 2026-09-05*" sobre a tabela Stack não se sustenta.**
Duas das treze linhas estavam desatualizadas nessa data — Next.js (16.2.7 vs 16.3.4, com o agravante do C1) e React (19.2.7 vs 19.2.8). Como a nota é o que autoriza um leitor futuro a confiar na tabela sem reverificar, ela precisa ou virar verdade ou sair. As outras onze linhas se sustentaram integralmente.

**A2 — `images.dangerouslyAllowLocalIP` pode apagar as imagens da demonstração.**
O AD-12 serve as imagens do Catálogo Semeado **pelo Go**, com URL relativa, atrás do rewrite. Em `docker compose`, o upstream é um hostname de rede privada. O Next.js 16 passou a **bloquear por padrão** a otimização de imagens cujo upstream resolva para IP privado, devolvendo `400`. Duas saídas, ambas de uma linha: `unoptimized` no `next/image` para essas imagens (coerente com o AD-12, que já quer o Go como dono do byte), ou `images.dangerouslyAllowLocalIP: true`. Sem uma das duas, a Vitrine sobe sem imagem na sala.

### 🟡 MÉDIO

**M1 — Compatibilidade shadcn/ui × Next.js 16 não tem confirmação oficial.**
A página de compatibilidade da própria shadcn/ui continua sendo "Next.js 15 + React 19" e nenhuma das entradas do changelog de 2026 menciona Next.js 16. Isso não é o mesmo que incompatível — os componentes são copiados, não uma dependência versionada, e há relato consistente de uso com Next 16 + React 19 + Tailwind v4. Registre como "não confirmado", não como fato. Ressalva prática: com **npm** a instalação exige flag de peer deps; pnpm/bun/yarn não.

**M2 — `pg_trgm` é extensão, e o índice não é o motivo do desempenho.**
Duas coisas: (a) `CREATE EXTENSION pg_trgm` precisa existir numa migração de `db/migracoes/` — a árvore de origem e o AD-16 não a mencionam, e a extensão vem na imagem oficial do Postgres mas não vem habilitada; (b) a justificativa da tabela "Adiado" ("5.000 Produtos com GIN `pg_trgm` ficam muito abaixo dos 500 ms") é verdadeira **pelo tamanho da tabela**, não pelo índice — e com termo de busca de 1 ou 2 caracteres o `pg_trgm` não extrai trigrama nenhum e degenera para varredura completa. A 5.000 linhas isso segue irrelevante para o NFR-4, mas a frase atribui ao índice um mérito que é da escala.

**M3 — sqlc v1.31.1 analisando `DEFAULT uuidv7()` não foi confirmado.**
O sqlc analisa o DDL com parser próprio derivado de uma versão específica do PostgreSQL, e a v1.31.1 é de abril/2026. Nada na web confirma nem desmente que ela aceite o catálogo do PG 18. Como `uuidv7()` é só uma chamada de função num `DEFAULT` — sintaxe que existe desde sempre — o risco é baixo. É um spike de dez minutos: uma migração, `sqlc generate`, ver se passa.

**M4 — `rewrites()` não define cabeçalhos de proxy.**
Não há `X-Forwarded-For`/`X-Forwarded-Host`. O `X-Correlation-Id` do AD-15 atravessa normalmente (é cabeçalho da requisição do navegador), mas qualquer log de IP de origem no Go verá o IP do contêiner do Next, não o do navegador. Como o AD-15 proíbe dado pessoal em log e não exige IP, isso não quebra nada — só precisa não virar surpresa.

### 🟢 BAIXO

| Id | Achado |
|---|---|
| **B1** | React 19.2.7 não é o último patch (19.2.8, 2026-07-21, só performance de decode em RSC). Sem CVE conhecido contra 19.2.7 — o fix do DoS de RSC de maio/2026 exige ≥ 19.2.6, satisfeito. |
| **B2** | "Sal de 16 bytes" não vem do OWASP: a Password Storage Cheat Sheet não fixa comprimento de sal. O número é razoável (RFC 9106 / prática corrente); só a atribuição está errada. |
| **B3** | A doc do `golang.org/x/crypto/argon2` recomenda a RFC 9106 (`t=1, m=2 GiB, p=4` ou `t=3, m=64 MiB, p=4`) — parâmetros bem mais pesados que o piso do OWASP. Nada impede usar o OWASP (a espinha diz explicitamente "piso do OWASP"), mas quem abrir a doc do pacote verá outro número e vai perguntar. |
| **B4** | Next.js 16 removeu `next lint` e `next build` não roda mais lint; Turbopack é o padrão em dev e build. O passo de CI do AD-12 e os scripts de `web/package.json` precisam refletir isso. |
| **B5** | `middleware.ts` virou `proxy.ts` no Next 16, e um `middleware.ts` remanescente é **ignorado em silêncio, sem erro de build**. A espinha já proíbe lógica ali (AD-10), então o risco é nulo — mas vale uma linha na convenção, porque "ignorado em silêncio" é exatamente o modo de falha que o AD-10 quer evitar. |
| **B6** | Defaults de `next/image` mudaram no 16: `qualities` passou a `[75]`, `imageSizes` perdeu o `16`, `minimumCacheTTL` foi de 60 s para 4 h, `maximumRedirects` virou 3, e src local com query string agora exige `images.localPatterns.search`. |
| **B7** | Go 1.27.1 é de 2026-09-01 — quatro dias antes da data da espinha, e a linha 1.27 inteira tem duas semanas e meia (1.27.0 em 2026-08-19). Nada de errado; só registre que a alternativa conservadora existe e é Go 1.26.8 (mesma data, linha madura, oito patches de rodagem). |
| **B8** | A nota de migração da 18.6 sobre `reltuples` corrompido em índices GIN vale só para bancos que já rodavam 18.0–18.4. Um banco nascendo em 18.6 não é afetado. |

---

## Não confirmado (diferente de errado)

Estas quatro afirmações não foram desmentidas — apenas não encontrei fonte que as sustente. Ficam registradas como asserção pendente de checagem, não como defeito.

| Item | Por que não deu para confirmar | Custo de confirmar |
|---|---|---|
| shadcn/ui × Next.js 16 | A documentação oficial de compatibilidade para em Next.js 15; o changelog de 2026 não toca no assunto | `npx shadcn init` num Next 16 vazio |
| sqlc v1.31.1 × catálogo/parser do PostgreSQL 18 (`uuidv7()`) | Nenhuma issue, release note ou doc trata de PG 18 no sqlc | Uma migração e um `sqlc generate` |
| NFR-4: 500 ms p95 com 5.000 Produtos | Não existe benchmark público de `pg_trgm`/GIN nessa escala — os que existem medem tabelas grandes. A inferência por tamanho é sólida (5.000 linhas cabem em memória e nem seq scan chega perto de 500 ms), mas inferência não é medição, e a própria espinha condiciona o Elasticsearch a "o NFR-4 falhar em medição" | O Catálogo Semeado já é determinístico; é um `EXPLAIN ANALYZE` |
| `Set-Cookie` do Go atravessando `rewrites()` até o navegador (AD-20) | O padrão é descrito como funcional por várias fontes da comunidade — mesma origem, cookie persiste no domínio do Next — mas não achei documentação oficial da Vercel **garantindo** o repasse do `Set-Cookie` por rewrite. O único caso documentado de "cookie preso no servidor" é em Route Handler / Server Action (fluxo que o AD-10 já proíbe), **não** em rewrite | Um `curl -i` contra o Next apontando para o Go |

---

## Fontes

- [Go — Release History](https://go.dev/doc/devel/release) · [Go 1.27 Release Notes](https://go.dev/doc/go1.27) · [Routing Enhancements for Go 1.22](https://go.dev/blog/routing-enhancements)
- [PostgreSQL 18.6 Release Notes](https://www.postgresql.org/docs/current/release-18-6.html) · [Anúncio 18.6](https://www.postgresql.org/about/news/postgresql-186-1711-1615-1519-1424-and-19-beta-3-released-3365/) · [PG 18 Release Notes](https://www.postgresql.org/docs/current/release-18.html) · [F.35 pg_trgm](https://www.postgresql.org/docs/current/pgtrgm.html)
- [Redis 8.10.1](https://github.com/redis/redis/releases/tag/8.10.1) · [Redis 8.10 release notes](https://redis.io/docs/latest/operate/oss_and_stack/stack-with-enterprise/release-notes/redisce/redisos-8.10-release-notes/)
- [Node.js Previous Releases](https://nodejs.org/en/about/previous-releases) · [Node v24.20.0](https://github.com/nodejs/node/releases/tag/v24.20.0)
- [Next.js — August 2026 Security Release](https://nextjs.org/blog/august-2026-security-release) · [Support Policy](https://nextjs.org/support-policy) · [Upgrading to version 16](https://nextjs.org/docs/app/guides/upgrading/version-16) · [rewrites](https://nextjs.org/docs/app/api-reference/config/next-config-js/rewrites) · [v16.3.4](https://github.com/vercel/next.js/releases/tag/v16.3.4)
- [React v19.2.7](https://github.com/facebook/react/releases/tag/v19.2.7) · [React v19.2.8](https://github.com/react/react/releases/tag/v19.2.8) · [Netlify — React/Next security mai/2026](https://www.netlify.com/changelog/2026-05-08-react-nextjs-security-vulnerabilities/)
- [jackc/pgx tags](https://github.com/jackc/pgx/tags) · [pkg.go.dev pgx/v5](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [sqlc releases](https://github.com/sqlc-dev/sqlc/releases) · [sqlc — Using transactions](https://docs.sqlc.dev/en/latest/howto/transactions.html) · [sqlc — Modifying the database schema](https://docs.sqlc.dev/en/latest/howto/ddl.html) · [issue #3603](https://github.com/sqlc-dev/sqlc/issues/3603)
- [pressly/goose tags](https://github.com/pressly/goose/tags) · [goose — Using sqlc and goose](https://pressly.github.io/goose/blog/2024/goose-sqlc/)
- [testcontainers-go v0.44.0](https://github.com/testcontainers/testcontainers-go/releases/tag/v0.44.0) · [System requirements](https://golang.testcontainers.org/system_requirements/)
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) · [pkg.go.dev x/crypto/argon2](https://pkg.go.dev/golang.org/x/crypto/argon2)
- [shadcn/ui — Next.js 15 + React 19](https://ui.shadcn.com/docs/react-19) · [Changelog](https://ui.shadcn.com/docs/changelog)
