---
title: 'Estória 1.3 — Primeira migração e o Catálogo Semeado determinístico'
type: 'feature'
created: '2026-09-11'
status: 'done'
route: 'dispatch'
baseline_commit: '5c97c073128a2573425fe3815698e170a83d83b3'
review_loop_iteration: 0
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O banco sobe vazio: `db/migracoes/` e `db/semente/` só têm `.gitkeep`, e `plataforma.Migrar` já trata "nenhuma migração" como caso normal. Sem schema e sem dados não existe Comprador nem Produto para as estórias 1.4 a 1.6 atravessarem, e a verificação 2 do passo 0 — sqlc analisando `DEFAULT uuidv7()` — continua aberta.

**Approach:** Escrever as migrações goose que criam `identidade` e `catalogo` com as cinco tabelas do esqueleto, tendo `CREATE EXTENSION pg_trgm` como a primeira de `catalogo`, e um Catálogo Semeado versionado em `db/semente/` que o binário aplica depois das migrações e só uma vez, marcado numa tabela de versão. As imagens viajam embutidas no binário e são servidas pelo Go sob `/api/v1`, com URL relativa no banco. O `sqlc generate` roda sobre essas migrações e o desfecho volta ao `addendum.md` §10.

## Boundaries & Constraints

**Always:**
- Um schema Postgres por módulo (AD-2). **Nenhuma chave estrangeira cruza schema** — referência cruzada carrega `uuid` sem FK.
- Chaves `uuid DEFAULT uuidv7()`, datas `timestamptz` UTC, `snake_case` singular, monetário `bigint` com sufixo `_centavos`, identificadores literalmente nos 20 termos do §3 do PRD.
- Arquivo de migração: `db/migracoes/<timestamp>_<módulo>_<descrição>.sql`, formato goose, versionado por timestamp. Nunca `NNNN` sequencial.
- A semente é aplicada **depois** das migrações, numa transação só, e **só se o marcador de versão não existir**. Subir duas vezes produz exatamente os mesmos dados; `docker compose down -v` volta ao estado inicial.
- A semente não cria Pedido, e nada nela consome `numero`.
- Nenhuma requisição de rede em tempo de execução (AD-12): as imagens são arquivos do repositório, embutidos no binário, com URL **relativa** no banco.
- `catalogo.categoria` nasce com `categoria_pai_id` nulo, com FK interna ao próprio schema, e não exposto (addendum §6).
- Senha semeada em Argon2id `m=19456`, `t=2`, `p=1`, sal de 16 bytes, no formato PHC.

**Decisões tomadas (humano, 2026-09-11):**
- **As imagens são placeholders gerados no próprio repositório.** Um SVG determinístico por Produto, com a cor da Categoria e o nome do Produto, escrito por um script versionado. Nada é baixado, nada depende de arquivo externo chegar, e a troca por fotos de verdade depois é substituir arquivo em `media/` sem tocar no banco.
- **O catálogo de demonstração tem 50 Produtos em 5 Categorias** — o piso da suposição do §4.2 do PRD. Nomes, descrições e preços em português plausível, escritos à mão; os 5.000 da 1.4 continuam à parte, ligados por configuração.
- **A spec fica inteira.** O tamanho vem do Code Map e das tarefas, não de objetivos independentes; a rota de mídia e o sqlc continuam nesta estória, como o `epics.md` escreveu.

**Never:**
- `Estoque`, `Reserva de Estoque` e o predicado de visibilidade do AD-19 — são da Épica 3.
- A coluna normalizada e o índice GIN — são migração da 1.4. Aqui entra só a extensão.
- Tabela de Pedido, item, transição ou tentativa de pagamento — são 1.6 e 1.7.
- Auto-migração, migração aplicada fora do arranque, ou semente que mascare falha de migração.
- Coluna de papel que transforme Comprador em Administrador: são duas tabelas separadas.
- Editar qualquer arquivo sob `internal/<módulo>/db/gerado/`.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Primeiro arranque | Banco vazio | Migrações aplicadas, semente aplicada, marcador gravado | Falha derruba o arranque |
| Segundo arranque | Marcador presente | Semente é pulada; contagem e identificadores idênticos | N/A |
| Semente falha no meio | SQL inválido no meio do arquivo | Nada é gravado e o marcador não existe; o arranque cai | Transação única, rollback |
| Dois binários ao mesmo tempo | Ambos veem banco vazio | Só um aplica a semente; o outro segue sem erro | `ON CONFLICT DO NOTHING` no marcador decide o vencedor |
| Imagem do Produto | `GET /api/v1/media/<arquivo>` | O byte do arquivo embutido, com `Content-Type` correto | Arquivo inexistente devolve o envelope do AD-14 |
| FK cruzando schema | Consulta a `information_schema` | Zero linhas | N/A |

</frozen-after-approval>

## Code Map

- `db/migracoes/` -- vazio (só `.gitkeep`); recebe os arquivos goose desta estória.
- `db/semente/` -- vazio; recebe o SQL da semente.
- `db/embutido.go` -- já embute `all:migracoes` e exporta `Migracoes`/`DirMigracoes`; acrescentar `Semente`, `DirSemente` e a constante de versão. O comentário do topo já aponta esta estória.
- `internal/plataforma/migracao.go` -- `Migrar(ctx, dsn, fs.FS, dir)` é o molde a imitar: goose com `SetBaseFS`, espera de conexão, erro que derruba o arranque. A semente ganha um arquivo irmão.
- `cmd/azamon/main.go` -- `executar()` chama `plataforma.Migrar` logo após a config; a semente entra imediatamente depois, antes de montar o servidor.
- `internal/fronteira_test.go` -- a tabela `arestas` é exaustiva: um pacote novo (`media`) exige linha própria e entrada na lista de destinos de quem o importa. `db` é o precedente exato.
- `api/rotas.go` -- `ServeMux` da biblioteca padrão, rotas `/api/v1/...`; `naoEncontrado` já devolve o envelope do AD-14.
- `internal/plataforma/erro/erro.go` -- `Escrever` e o registro de sentinelas; `ErrNaoEncontrado` já existe e serve à mídia ausente.
- `internal/plataforma/env_test.go` -- compara o `.env` versionado com as variáveis que o código lê. Variável nova entra nos dois ou a suíte quebra.
- `Dockerfile`, `.dockerignore` -- imagem final é `scratch`; `media/` não está ignorado, então o `//go:embed` alcança os arquivos na construção. Nada de arquivo solto no contêiner.
- `web/next.config.ts` -- o `rewrites()` cobre **só** `/api/:caminho*`; por isso a rota de mídia nasce sob `/api/v1`.
- (ler) `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` §10 -- imitar a forma da entrada da Estória 1.2.
- (não editar) `internal/catalogo/catalogo.go`, `internal/identidade/identidade.go` -- as interfaces públicas seguem vazias; a regra chega nas Épicas 2 e 3.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/<ts>_identidade_schema_e_contas.sql` -- criar o schema `identidade` com `comprador` (e-mail único normalizado, `senha_hash`) e `administrador` (mesma forma, tabela separada) -- o Administrador nasce semeado (FR-4) e não existe coluna de papel.
- [x] `db/migracoes/<ts>_catalogo_pg_trgm.sql` -- `CREATE EXTENSION IF NOT EXISTS pg_trgm` -- AD-16 exige que seja a **primeira** migração de `catalogo`; a imagem do Postgres traz a extensão, não habilitada.
- [x] `db/migracoes/<ts>_catalogo_schema.sql` -- schema `catalogo` com `vendedor` (com estado ativo, porque FR-8 desativa e não remove), `categoria` (nome único, `categoria_pai_id` nulo com FK interna) e `produto` (nome, descrição, `preco_centavos`, URL relativa da imagem, `vendedor_id`, `categoria_id`) -- é o mínimo que a página crua da 1.5 exibe.
- [x] `db/migracoes/<ts>_plataforma_marcador_de_semente.sql` -- tabela `public.semente` com a versão como chave primária -- fica no `public`, ao lado do `goose_db_version`, porque a semente não pertence a módulo nenhum.
- [x] `db/semente/*.sql` -- o Catálogo Semeado: Vendedores, Categorias, Produtos, um Administrador e um Comprador, com `uuid` **literais e fixos** -- determinismo é a condição de aceite; `uuidv7()` sortearia identificador novo a cada `down -v`.
- [x] `db/embutido.go` -- embutir `db/semente` e declarar a versão da semente -- `//go:embed` não atravessa `..`, e o embed das migrações já mora aqui.
- [x] `internal/plataforma/semente.go` -- `Semear` numa transação só: insere o marcador com `ON CONFLICT DO NOTHING` e só aplica o SQL se a inserção pegou -- resolve "aplicar uma vez" e dois binários simultâneos com um comando, sem trava separada.
- [x] `cmd/azamon/main.go` -- chamar `Semear` depois de `Migrar` -- a ordem é condição de aceite, e a semente não pode mascarar migração que falhou.
- [x] `media/gerar.go` (ou script equivalente) + os 50 SVG versionados -- gerar um placeholder determinístico por Produto, com a cor da Categoria e o nome do Produto -- gerador e saída ficam os dois no repositório para que a construção seja offline e o arquivo possa ser trocado por foto depois sem tocar no banco.
- [x] `media/embutido.go` -- embutir a mídia no binário -- a imagem final é `scratch`: arquivo não embutido simplesmente não existe no contêiner.
- [x] `api/media.go` + registro em `api/rotas.go` -- servir `GET /api/v1/media/{arquivo}` do sistema de arquivos embutido -- fica sob `/api/v1` porque é o único prefixo que o `rewrites()` do Next atravessa (AD-10).
- [x] `internal/fronteira_test.go` -- acrescentar `media` à tabela de arestas e à lista de destinos de `api` -- a tabela é exaustiva; sem a linha o teste acusa aresta desconhecida.
- [x] `sqlc.yaml` + `internal/catalogo/db/consultas.sql` + `internal/identidade/db/consultas.sql` -- configurar sqlc v1.31.1 sobre as migrações goose, uma consulta por módulo, gerando em `internal/<módulo>/db/gerado/` -- é a verificação 2; o prefixo de módulo no nome do arquivo de migração é o que permite recortar o schema por pacote.
- [x] `db/schema_test.go` -- com `testcontainers-go` sobre PostgreSQL 18.6: migrar, semear duas vezes e cobrir as linhas da matriz de I/O, incluindo a consulta ao `information_schema` -- nenhuma das condições de aceite é observável sem um Postgres de verdade.
- [x] `README.md` -- registrar as credenciais de demonstração do Comprador e do Administrador -- quem clona precisa entrar no sistema em 15 minutos.
- [x] `.../addendum.md` §10 + `.memlog.md` -- desfecho da verificação 2 e as decisões desta estória -- a SM-7 exige o addendum vivo, na mesma estória que decidiu.

**Acceptance Criteria:**
- Dado o banco migrado, quando pergunto ao `information_schema` pelos schemas e tabelas, então existem `identidade` e `catalogo` e **apenas** `comprador`, `administrador`, `vendedor`, `categoria` e `produto`.
- Dado o histórico de migrações de `catalogo`, quando as ordeno por versão, então `CREATE EXTENSION pg_trgm` é a primeira.
- Dado o schema inspecionado, quando percorro colunas e chaves, então toda chave primária é `uuid DEFAULT uuidv7()`, toda data é `timestamptz`, todo valor monetário é `bigint` terminado em `_centavos`, e todo nome é `snake_case` singular.
- Dado `sqlc generate` com a v1.31.1 sobre essas migrações, quando ele analisa `DEFAULT uuidv7()`, então gera sem erro em `internal/<módulo>/db/gerado/` — e, se falhar, o `uuid` passa a ser gerado no lado Go e nada mais muda.
- Dado o compose no ar, quando abro a URL de imagem gravada num Produto, então recebo a imagem pelo Go, sem requisição a rede externa em ponto nenhum.
- Dado `docker compose down -v` seguido de `up`, quando comparo o catálogo com o anterior, então Vendedores, Categorias e Produtos têm os mesmos identificadores, nomes e preços.

## Implementation Notes

**Verificação 2 (sqlc × `DEFAULT uuidv7()`): fechada, positiva.** `sqlc generate` com a v1.31.1 roda limpo sobre as migrações goose e escreve os dois pacotes gerados. A saída declarada — gerar o `uuid` no lado Go — **não foi acionada**, e a chave primária continua `uuid DEFAULT uuidv7()`. Um desvio menor apareceu no lugar: a inflexão do sqlc devolve `CatalogoCategorium` para `catalogo.categoria`, e `rename:` não alcança nome de struct de tabela — o knob certo é `emit_exact_table_names: true`, que é o correto de qualquer forma porque a convenção do AD-2 já é `snake_case` singular.

**O recorte por módulo é glob no nome do arquivo de migração.** `schema: "db/migracoes/*_catalogo_*.sql"` — é o que faz o prefixo de módulo ser mecanismo, e migração nova entra no recorte sem tocar no `sqlc.yaml`.

**O Catálogo Semeado é gerado, e o gerador está versionado junto com a saída.** `media/gerar.go` (`//go:build ignore`) carrega a lista escrita à mão e escreve os 50 SVG e `db/semente/001_catalogo_semeado.sql`. Os `uuid` saem de SHA-1 do nome, no formato v5: determinismo sem 62 literais digitados à mão, e o SQL versionado continua só com literais. Consequência a lembrar: editar o SQL da semente à mão é perder a edição no próximo `go run`.

**As cinco tabelas nascem sem coluna de data.** Nenhuma estória da épica lê `criado_em` e a semente determinística não ganha nada com um `now()` dentro. A convenção `timestamptz` segue valendo — só não tem ainda a quem se aplicar dentro de `identidade` e `catalogo`.

**`Semear` sem nenhum `.sql` é erro, ao contrário de `Migrar` sem migração.** Embed quebrado que passasse por "catálogo vazio" gravaria o marcador, e o arranque seguinte nem tentaria de novo.

**E-mail normalizado virou `CHECK (email = lower(email))`.** Restrição de banco, não disciplina de chamador: o `UNIQUE` passa a valer sobre a forma normalizada de verdade.

**Não foi preciso variável `AZAMON_` nova.** A versão da semente é constante de código (`db.VersaoSemente`), não limiar de operação — nada em `.env` mudou, e `internal/plataforma/env_test.go` segue verde sem edição.

## Spec Change Log

## Review Triage Log

Três camadas rodaram (blind-hunter, edge-case-hunter, verification-gap). Toda constatação tem linha; veredito por dano real, não pela severidade que o revisor atribuiu.

**Correções aplicadas (patch)**

- `medium` — README manda bater `VersaoSemente` quando a semente muda, e isso derruba o arranque: os INSERT da semente carregam chave literal sem `ON CONFLICT`, então versão nova sobre volume já semeado viola chave primária dentro da transação e o processo não sobe. O outro lado da mesma moeda: não bater a constante faz a mudança ser silenciosamente ignorada. O caminho certo já existe e é `docker compose down -v`. (blind-hunter, edge-case-hunter, verification-gap)
- `medium` — Nada liga os 50 `imagem_url` da semente ao conjunto embutido em `media/`. `api/media_test.go` pede `nomes[0]` derivado do próprio embed, e `db/schema_test.go` compara o catálogo consigo mesmo, então URL apontando para arquivo inexistente passa verde. O comentário do teste afirma vigiar exatamente isso, e não vigia. (verification-gap, edge-case-hunter)
- `medium` — Nada verifica que o arranque semeia. `cmd/azamon/main_test.go` falha dentro de `Migrar` e nunca alcança a chamada a `Semear`; `db/schema_test.go` chama `Semear` direto. Apagar as dez linhas de `cmd/azamon/main.go` deixa `go test ./...` inteiramente verde e o compose sobe com loja vazia. (verification-gap)
- `low` — O comentário de `api/media.go` nomeia a defesa errada contra travessia de diretório, e o addendum §10 repete a afirmação. Confirmado em execução: `GET /api/v1/media/%61-ultima-estacao-do-vale.svg` devolve 200, ou seja, o curinga do `ServeMux` entrega o valor já desescapado. Quem recusa `../` é o `fs.ValidPath` do `embed.FS`. O comportamento de hoje é seguro; o raciocínio escrito é que não sustenta um backend futuro fora do embed. (blind-hunter, edge-case-hunter)
- `low` — `mime.TypeByExtension` consulta a tabela do sistema operacional (registro do Windows, `/etc/mime.types`). No contêiner `scratch` não há tabela e a embutida do Go responde, mas na máquina de quem roda `go test` o `.svg` pode estar registrado como `text/xml` e o teste falha por motivo alheio ao código. O embed tem um tipo só. (blind-hunter, edge-case-hunter)
- `low` — A garantia de transação única depende de uma propriedade não escrita: aplicar um arquivo com vários comandos num `ExecContext` só funciona porque o pgx cai no protocolo simples quando não há argumento. É a única propriedade em que o rollback da semente se apoia. (blind-hunter)
- `low` — `db/schema_test.go` afirma `!= 50` com a mensagem "o §4.2 do PRD pede ao menos 50". O exato é intencional, como nas contagens de Categoria e Vendedor; a mensagem é que mente. (verification-gap)
- `low` — O README diz que as credenciais estão em `db/semente/` e, duas frases depois, que `db/semente/` não se edita à mão. A origem das duas é `media/gerar.go`. (blind-hunter)

**Adiadas (defer)**

- `medium` — `catalogo.produto.vendedor_id` e `categoria_id` não têm índice: o Postgres indexa só o lado referenciado. A 1.4 é a estória que mede p95 com 5.000 Produtos e é quem deve decidir os índices, como já decide o GIN. (blind-hunter)
- `low` — `UNIQUE` de e-mail vale por tabela, então o mesmo endereço pode existir como Comprador e como Administrador. Não há leitor hoje; a estória que construir autenticação precisa decidir a precedência de qualquer forma. Adiada em vez de virar lacuna de intenção porque nenhum consumidor atual observa o defeito. (blind-hunter)

**Rejeitadas**

- `false` — "`categoria.cor` interpolada sem escape de XML": as cores são literais hexadecimais escritos no gerador, nunca entrada de terceiro. (edge-case-hunter)
- `false` — "o driver `pgx` vem de import em branco no arquivo irmão, e mexer nele vira pânico em execução": `Migrar` abre conexão pelo mesmo driver e roda antes, em todo arranque, então a dependência não apodrece em silêncio. (blind-hunter)
- `false` — "a convenção `timestamptz` é afirmada e nunca exercitada": as Notas de Implementação e o addendum já declaram que as cinco tabelas nascem sem coluna de data. A regra do teste é guarda latente e dispara na primeira coluna temporal, que é o comportamento correto. (edge-case-hunter)
- `false` — "`Semear` abre um segundo pool em vez de reaproveitar o de `Migrar`": um `sql.DB` para uma transação, fechado no `defer`, não é defeito. (blind-hunter)
- rejeitada por editar a própria especificação — "a frase da Verificação diz que `db/schema_test.go` cobre as seis linhas da matriz, e a linha da mídia está em `api/media_test.go`": correto, mas o conserto é texto desta especificação. (edge-case-hunter)
- `low` — o gerador não apaga SVG órfão de Produto renomeado ou removido. Dano é arquivo morto no binário; o conserto é laço de reconciliação. (blind-hunter, edge-case-hunter)
- `low` — o gerador não valida Vendedor fora da lista nem nome repetido, e valida Categoria já dentro do laço que escreve. As três falhas caem alto — violação de chave estrangeira ou primária no arranque — sobre uma lista escrita à mão. (blind-hunter, edge-case-hunter)
- `low` — `media/gerar.go` não é compilado por `go vet` nem coberto por teste, e `quebrar` não parte palavra maior que a largura; `apelido` aceita dígito não-ASCII e acento fora de `semAcento`. Lista escrita à mão, saída conferida a olho, dano cosmético no cartaz de 800px. (blind-hunter, edge-case-hunter)
- `low` — nada confere que a saída versionada ainda corresponde ao gerador. A divergência aparece no próprio diff de quem mexe (gerador alterado, artefato não), e o conserto é teste que roda `go run` contra diretório temporário. Não há CI neste repositório, por decisão registrada na 1.2. (blind-hunter, verification-gap)
- `low` — os dois hash Argon2id são literais sem verificação no repositório. Conferi os dois nesta revisão com `argon2.IDKey` e os parâmetros do AD-9: batem com as senhas do README. O conserto promoveria `golang.org/x/crypto` de indireta a direta. (blind-hunter)
- `low` — a rota de mídia não manda `Cache-Control` nem trata requisição condicional. São 50 SVG de menos de 1 KB numa demonstração local, e as URL não são endereçadas por conteúdo, então `immutable` seria mentira no dia em que o arquivo virar foto. A 1.4 mede a listagem, não a imagem. (blind-hunter, edge-case-hunter)
- `low` — a rota não tem lista de extensões permitidas nem `nosniff`. O embed é `*.svg` de arquivos do próprio repositório, e não há upload em lugar nenhum. (blind-hunter)
- `low` — `db/schema_test.go` não tem guarda de `testing.Short()`. A especificação e o README declaram Docker como pré-requisito da suíte, por decisão explícita. (blind-hunter)
- `low` — o teste de schema deixa linhas de fixture no banco para os subtestes seguintes. O comentário do arquivo já avisa que dali em diante a ordem importa, e o contêiner morre no fim. (blind-hunter)
- `low` — os dois `CHECK` novos (`email = lower(email)`, `preco_centavos > 0`) não são afirmados por teste. São migrações escritas uma vez, sem caminho de escrita que as altere. (blind-hunter)
- `low` — `public.semente` guarda a versão e não quando foi aplicada. Ninguém lê a tabela hoje. (blind-hunter)
- `low` — `Semear` não espera conexão como `Migrar` faz. Roda imediatamente depois de `Migrar` ter provado o banco de pé; a janela é de microssegundos e a falha é alta. (edge-case-hunter)
- `low` — `catalogo.categoria` aceita ser pai de si mesma e a auto-referência não declara `ON DELETE`. A coluna nasce nula, não é exposta no MVP (addendum §6) e não há escritor; quem a expuser decide as duas coisas. (edge-case-hunter)
- `low` — a versão do Postgres aparece como literal em `db/schema_test.go` e em `docker-compose.yml`, sem ligação. Um comentário não impede a divergência, e ler o compose a partir do teste custa mais do que o risco. (verification-gap)

## Design Notes

**O marcador decide, não uma consulta prévia.** Ler `public.semente` e depois decidir é corrida clássica: dois binários leem vazio e semeiam duas vezes. A inserção do marcador é a decisão, dentro da mesma transação que aplica o SQL:

```sql
BEGIN;
INSERT INTO public.semente (versao) VALUES ($1) ON CONFLICT DO NOTHING;
-- zero linhas afetadas: alguém já semeou, COMMIT e siga.
-- uma linha: aplique db/semente/*.sql aqui e COMMIT.
```

**Os `uuid` da semente são literais.** `DEFAULT uuidv7()` é o certo para dado nascido em execução, e o errado para a semente: cada `down -v` daria identificadores novos, e a condição de aceite pede exatamente o contrário.

## Verification

**Resultado (2026-09-11):**
- `go test ./...` — verde, 34 s. `db/schema_test.go` sobe `postgres:18.6` por `testcontainers-go` e cobre as seis linhas da matriz de I/O; `internal/fronteira_test.go` aceita `media` como aresta nova.
- `sqlc generate` — sem erro; rodado duas vezes, sem diferença na segunda.
- `docker compose down -v && docker compose up -d --build` — quatro serviços de pé, `azamon`/`postgres`/`redis` `healthy`, e o log diz `Catálogo Semeado aplicado`.
- `docker compose restart azamon` — o log diz `Catálogo Semeado pulado; o marcador já existe`, e a contagem fica em 50 Produtos / 5 Categorias / 5 Vendedores.
- FK cruzando schema (`information_schema`, pelo `psql` do contêiner) — zero linhas.
- `curl localhost:3000/api/v1/media/fone-de-ouvido-bluetooth-aurora.svg` — `200 image/svg+xml`, 847 bytes, atravessando o `rewrites()`; arquivo inexistente devolve `404` no envelope do AD-14.
- `down -v` seguido de `up`: o `SELECT id, nome, preco_centavos, imagem_url FROM catalogo.produto ORDER BY id` bate linha a linha com o de antes, nas 50 linhas.

**Commands:**
- `go test ./...` -- expected: tudo verde, incluindo o teste de fronteira com `media` e o teste de schema sobre testcontainers.
- `sqlc generate` -- expected: sem erro, e `git status` limpo depois de rodar duas vezes.
- `docker compose down -v && docker compose up -d --build` -- expected: os quatro serviços `healthy`, com a semente aplicada no log do arranque.
- `docker compose restart azamon` -- expected: o log diz que a semente foi pulada e a contagem de Produtos não muda.
- `docker compose exec postgres psql -U azamon -d azamon -c "<consulta de FK cruzando schema>"` -- expected: zero linhas.
- `curl -i localhost:3000/api/v1/media/<arquivo>` -- expected: `200` e a imagem, atravessando o `rewrites()` do Next.

**Manual checks (if no CLI):**
- Ler nomes, descrições e preços dos Produtos semeados em voz alta: nada de "Produto Teste 3" nem de preço absurdo (NFR-1).
