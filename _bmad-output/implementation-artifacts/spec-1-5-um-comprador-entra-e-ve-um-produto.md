---
title: 'Estória 1.5 — Um Comprador entra e vê um Produto'
type: 'feature'
created: '2026-09-11'
status: 'done'
baseline_commit: 'c7d9dda36376e64d01c07c9676bbcf5d857b76f2'
route: 'dispatch'
review_loop_iteration: 0
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problema:** O binário não tem nenhuma rota que leia o banco nem que emita Sessão. `api.Rotas()` não recebe dependência alguma, não existe pool pgx nem cliente Redis, e a prova de que o `Set-Cookie` atravessa o `rewrites()` ainda é a sonda descartável da 1.2. Sem isso o front não fala com o Go de verdade e a cadeia 1.6→1.7→1.8 não tem por onde começar.

**Abordagem:** Fiar banco e Redis até o handler, autenticar o Comprador semeado com Argon2id, emitir o cookie `azamon_sessao` opaco com a chave no Redis, servir o detalhe de Produto com o nome do Vendedor, e trocar a sonda da 1.2 pela Sessão de verdade. Telas cruas: a estória fecha quando o dado atravessa.

## Boundaries & Constraints

**Always:**
- Cookie `azamon_sessao`: `HttpOnly`, `SameSite=Lax`, `Path=/`, valor opaco de 256 bits, nada de conteúdo; chave no Redis com TTL de `cfg.SessaoExpiracao`.
- Argon2id `m=19456, t=2, p=1`, sal de 16 bytes, formato PHC — os mesmos parâmetros de `media/gerar.go`, senão a semente não autentica.
- O login é chamado **do navegador** por caminho relativo `/api/...`, único jeito de o cookie atravessar o `rewrites()`.
- Sentinela novo mora em `internal/identidade`; a linha de tradução mora só na fatia `registro` de `internal/plataforma/erro/erro.go` (AD-14).
- E-mail normalizado para minúsculas antes de consultar — o `CHECK (email = lower(email))` não casa de outro jeito.
- Domínio não conhece `net/http` nem `internal/plataforma/erro`; o DTO é montado em `api/`.
- **Decidido:** o roteiro chega à Página de Produto por um link temporário na página de conferência da 1.2, com o identificador determinístico de um Produto semeado e comentário marcando a remoção pela Épica 3 — mesmo padrão descartável da sonda de cookie. O login não redireciona para Produto nenhum.

**Never:**
- Nenhuma migração nova: `identidade.comprador.senha_hash` já existe e a Sessão vive no Redis.
- Não registrar `GET /api/v1/produtos` (listagem) — é de `busca`, na Épica 3.
- Sem cadastro, sem encerramento de Sessão, sem bloqueio por tentativas, sem recuperação de senha, sem login de Administrador — Épica 2.
- Sem segunda regra de `rewrites()` (`web/scripts/casca.test.mjs` trava em uma), sem `middleware.ts`, sem token de cor novo.
- Sem exigir Redis no arranque: conexão preguiçosa, para `/api/v1/saude` e `cmd/azamon/main_test.go` seguirem sem infraestrutura.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Saída esperada | Erro |
|---|---|---|---|
| Login válido | `POST /api/v1/sessoes` com `comprador@azamon.test` / `azamon-comprador` | 200, `Set-Cookie: azamon_sessao=…` e `{"nome":"Joana Ribeiro"}` | — |
| Senha errada | mesmo e-mail, senha qualquer | 401 sem cookie | `ErrCredencialInvalida`, mensagem única |
| E-mail inexistente | `ninguem@azamon.test` | 401 sem cookie, no mesmo tempo de resposta | mesmo sentinela — não revela se a conta existe |
| E-mail em maiúsculas | `COMPRADOR@Azamon.test` | 200, autentica | — |
| Corpo inválido ou grande demais | JSON malformado ou acima do limite | 400 | envelope padrão |
| Sessão presente | `GET /api/v1/sessao` com cookie válido | 200, `{"nome":…}` | — |
| Sessão ausente ou expirada | sem cookie, ou valor fora do Redis | 401 | `ErrSessaoInvalida` |
| Produto existente | `GET /api/v1/produtos/{id}` de Produto semeado | 200 com nome, `preco_centavos` inteiro, `imagem_url` relativa e nome do Vendedor | — |
| Produto inexistente | uuid válido sem linha | 404 | `ErrNaoEncontrado` |
| Identificador malformado | `/api/v1/produtos/abc` | 404 | `ErrNaoEncontrado`, nunca 500 |

</frozen-after-approval>

## Code Map

- `internal/identidade/senha.go` (novo) — `Verificar(hashPHC, senha string) error`: decodifica o PHC, confere parâmetros, `argon2.IDKey` e `subtle.ConstantTimeCompare`. Base64 raw-std, sem preenchimento.
- `internal/identidade/identidade.go` — hoje só doc de pacote. Recebe `ErrCredencialInvalida`, `ErrSessaoInvalida` e `Autenticar(ctx, q *gerado.Queries, email, senha string) (Comprador, error)`.
- `internal/identidade/sessao.go` (novo) — token de 256 bits por `crypto/rand`; criar e ler a Sessão no Redis com TTL. Copie o formato hexadecimal de `novoIdentificador` em `internal/plataforma/correlacao.go`, sem importar `google/uuid`.
- `internal/identidade/db/consultas.sql` + `db/gerado/` — `BuscarCompradorPorEmail(ctx, email) (IdentidadeComprador, error)` **já existe e já está gerado**. Não mexer no gerado.
- `internal/plataforma/redis.go` (novo) — `AbrirRedis(url string) (*redis.Client, error)`: só `redis.ParseURL` e `redis.NewClient`, sem ping.
- `internal/plataforma/erro/erro.go` — acrescentar duas linhas na fatia `registro` (401 para os dois sentinelas). A ordem importa: o primeiro `errors.Is` que casar vence.
- `internal/catalogo/db/consultas.sql` — consulta nova com `JOIN catalogo.vendedor`, porque `BuscarProdutoPorID` não traz o nome do Vendedor. Depois `sqlc generate` na raiz.
- `internal/catalogo/catalogo.go` — hoje vazio. Recebe o tipo de Produto com nome do Vendedor e a função que o busca. `catalogo` e `identidade` não se importam.
- `api/rotas.go` — `Rotas` passa a receber `cfg plataforma.Config`, o pool e o cliente Redis. Apagar as linhas 18-19 da sonda.
- `api/sessao.go`, `api/produto.go` (novos) — primeiros DTO e primeiro `http.MaxBytesReader` do projeto. `imagem_url` e `preco_centavos` saem crus; `busca_normalizada` nunca vai no DTO.
- `api/sonda_cookie.go`, `api/sonda_cookie_test.go` — **apagar**. Os atributos de cookie a herdar estão no teste.
- `api/saude_test.go`, `api/media_test.go`, `api/rotas_test.go` — quebram com a assinatura nova de `Rotas`; ajustar as chamadas.
- `cmd/azamon/main.go` — abrir pool `pgxpool` e cliente Redis **depois** de `Semear` e antes de servir, fechando os dois no desligamento. Não tornar o arranque dependente de Redis acessível.
- `web/app/entrar/page.tsx` (novo) — `"use client"`, `fetch("/api/v1/sessoes", {method:"POST", credentials:"same-origin"})`. Componha com `input`, `label`, `button`, `card` e `alert`, já instalados e intocáveis.
- `web/app/produtos/[id]/page.tsx` (novo) — Server Component busca o Produto por `process.env.AZAMON_API_URL || "http://azamon:8080"`, porque o caminho relativo não existe no servidor; um filho `"use client"` chama `/api/v1/sessao` e mostra o nome do Comprador — é essa ida e volta que fecha a verificação 1.
- `web/app/page.tsx` — página de conferência da 1.2; ganha o link temporário para um Produto semeado. Os identificadores da semente são v5 derivados do nome em `media/gerar.go`, então são estáveis e podem ser escritos no front.
- `go.mod` — o cliente Redis é a única dependência nova; `golang.org/x/crypto` já está lá como indireta e `go mod tidy` a promove.

## Tasks & Acceptance

**Execution:**
- [x] `internal/identidade/senha.go` + `senha_test.go` -- Verificação Argon2id PHC, com teste sobre um hash real da semente e sobre PHC malformado.
- [x] `internal/plataforma/redis.go` -- Abrir o cliente Redis a partir da URL, sem ping.
- [x] `internal/identidade/sessao.go` -- Criar e ler a Sessão no Redis, com token opaco de 256 bits e TTL de `cfg.SessaoExpiracao`.
- [x] `internal/identidade/identidade.go` -- Sentinelas e `Autenticar`, normalizando o e-mail e devolvendo o mesmo erro para conta inexistente e senha errada.
- [x] `internal/plataforma/erro/erro.go` -- Traduzir os dois sentinelas para 401.
- [x] `internal/catalogo/db/consultas.sql` + `sqlc generate` -- Consulta de Produto com `JOIN` em `catalogo.vendedor`.
- [x] `internal/catalogo/catalogo.go` -- Busca de Produto por identificador devolvendo o nome do Vendedor.
- [x] `cmd/azamon/main.go` -- Pool `pgxpool` e cliente Redis fiados até `Rotas`, fechados no desligamento.
- [x] `api/rotas.go` -- Assinatura nova, três rotas novas, sonda removida.
- [x] `api/sessao.go` + `api/produto.go` -- Handlers, DTO e limite de corpo.
- [x] `api/sonda_cookie.go` + `api/sonda_cookie_test.go` -- Apagar os dois arquivos.
- [x] `api/sessao_test.go` + `api/produto_test.go` -- Cobrir as linhas da matriz de I/O contra Postgres e Redis reais por `testcontainers-go`, no padrão de `db/schema_test.go`.
- [x] `api/saude_test.go`, `api/media_test.go`, `api/rotas_test.go` -- Acompanhar a assinatura nova de `Rotas`.
- [x] `web/app/entrar/page.tsx` -- Formulário de login cru, com erro visível em caso de 401.
- [x] `web/app/produtos/[id]/page.tsx` -- Página de Produto crua com nome, preço, imagem, nome do Vendedor e nome do Comprador autenticado.
- [x] `web/app/page.tsx` -- Link temporário para um Produto semeado, com comentário marcando a remoção pela Épica 3.
- [x] `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` + `.memlog.md` -- Registrar a escolha do cliente Redis e o fechamento da verificação 1 pela Sessão de verdade.

**Acceptance Criteria:**
- Dado o sistema semeado, quando o Comprador entra pelo navegador em `/entrar` com as credenciais da semente, então o cookie `azamon_sessao` chega ao navegador através do `rewrites()` e o valor é chave existente no Redis com TTL de 7 dias.
- Dado esse cookie, quando a Página de Produto é aberta, então ela mostra o nome do Comprador autenticado, fechando a verificação 1 da épica.
- Dado o repositório, quando procuro por `sonda_cookie`, então não há mais nenhuma ocorrência em `api/`.
- Dado `go test ./...` e `npm test` em `web/`, quando rodam, então passam, com o teste de fronteira e a paridade do `.env` intactos.
- Dado `sqlc generate` rodado duas vezes, quando comparo, então `git status` fica limpo na segunda.

## Implementation Notes

Três desvios do Code Map, todos forçados pelo teste de fronteira do `AD-1` ou pela matriz de I/O, e nenhum deles muda o que a estória entrega:

1. **`Autenticar` e `BuscarProduto` recebem `gerado.DBTX`, não `*gerado.Queries`.** Com `*gerado.Queries` no parâmetro, `api/` teria de importar `internal/identidade/db/gerado` para montá-lo — e `internal/fronteira_test.go` falha nisso: a única porta de um módulo de domínio é `internal/<módulo>`. Tipado na interface do sqlc, `api/` passa o `*pgxpool.Pool` e não importa nada de dentro do módulo. Pelo mesmo motivo, `api/sessao_test.go` aplica migração e semente por `os.DirFS("..")` em vez de importar o pacote `db`.
2. **Um sentinela de 400 a mais no registro: `erro.ErrEntradaInvalida`.** A matriz pede 400 no envelope padrão para corpo malformado ou acima do limite, e o registro só tinha `ErrNaoEncontrado`. Mora no próprio `plataforma/erro` porque é falha de tradução, não de domínio — mesmo lugar e mesmo motivo do `ErrNaoEncontrado`. Os dois sentinelas de `identidade` entraram como previsto, os dois em 401.
3. **`internal/fronteira_test.go` ganhou a aresta `plataforma → identidade`.** É a consequência mecânica do `AD-14`: o arquivo único de tradução precisa alcançar a porta de cada módulo cujo sentinela ele traduz. A tabela continua exaustiva e a seta contrária continua proibida.

Detalhes que valem lembrar: a Sessão guarda `{id, nome}` em JSON sob `sessao:<token>` — o `id` não é usado nesta estória, e existe porque a 1.6 precisa saber de quem é o Pedido; o DTO de Produto leva `descricao` além do que a matriz lista, porque a tela crua a mostra; e `api/saude_test.go` não existe — o teste de `/api/v1/saude` sempre morou em `api/rotas_test.go`, que é quem foi ajustado para a assinatura nova.

## Spec Change Log

## Review Triage Log

**Passagem 1** — três camadas: caça-cega (15 achados), casos de borda (9), lacunas de verificação (2 + 3 secundários).

| Verdito | Achado | Evidência | Rota |
|---|---|---|---|
| medium | O gasto de Argon2id no caminho "conta inexistente" não é observado por asserção nenhuma | `Verificar` embrulha `ErrCredencialInvalida` em todo caminho de PHC malformado, então `TestHashDeDescarteEhValido` fica verde mesmo se a constante for corrompida e a função sair antes do `argon2.IDKey`. A defesa contra enumeração de contas cairia sem teste vermelho. | patch |
| medium | O TTL da Sessão é conferido por limite superior, não contra a configuração | `ttl <= 0 \|\| ttl > expiracaoDeTeste` aceita um TTL de um segundo. Gravar `time.Hour` no lugar de `ttl` passa despercebido, e o cookie continuaria prometendo sete dias. | patch |
| low | `BuscarProdutoPorID` ficou sem nenhum chamador | Busca em todo o repositório: só a definição, o SQL e uma menção em comentário de teste. `BuscarProdutoComVendedor` tomou o único uso que existia. | patch |
| low | O DTO devolve o identificador na forma em que o pedido chegou, não a da linha | `catalogo.BuscarProduto` monta `Produto{ID: id}` a partir do texto da rota; `linha.ID` já vem da consulta e é descartado. Pedir em maiúsculas devolve um `id` que o banco nunca guardou. | patch |
| low | O identificador entra na URL do servidor sem escape | `fetch(\`${api}/api/v1/produtos/${id}\`)` com um segmento contendo `%2F` normaliza para outra rota. `/produtos/..%2Fmedia%2Ffoo.svg` alcança a rota de mídia, devolve SVG, e o `resposta.json()` estoura numa página de erro em vez de 404. | patch |
| low | O comentário do pool descreve uma garantia que o código não dá | `pgxpool.New` não disca: analisa a DSN e devolve. O comentário diz que abrir conexão cedo "adianta a falha para o lugar errado", e nenhuma conexão é aberta ali. | patch |
| low | `GET /api/v1/sessao` devolve conteúdo por usuário sem diretiva de cache | Resposta 200 sem `Cache-Control`, heuristicamente armazenável. Uma linha resolve. | patch |
| low | A asserção de chave única no Redis depende da ordem dos subtestes | `len(chaves) != 1` sobre `Keys("*")`; o subteste seguinte grava uma segunda Sessão. Verde hoje só porque os subtestes correm na ordem de declaração. | patch |
| low | A aresta nova da tabela de fronteiras é por área, não por pacote | `"plataforma": {"identidade"}` libera qualquer pacote sob `internal/plataforma`, embora o comentário justifique só o `erro`. Apertar exige mudar a granularidade da tabela — complexidade maior que o defeito. | rejeitado |
| low | `catalogo` não tem sentinela próprio e devolve `pgx.ErrNoRows` | `api/` já importa `pgx/v5/pgxpool` e é a camada dona do pool, então comparar com `pgx.ErrNoRows` não cria acoplamento novo, e o 404 sai pelo `erro.ErrNaoEncontrado` da própria arquitetura. Um sentinela novo acrescenta superfície pública sem remover divergência existente. | rejeitado |
| low | `Verificar` aceita custo Argon2 sem limite superior | `m=4294967295` derrubaria o processo, mas o valor vem de `identidade.comprador.senha_hash`, cujo único escritor hoje é o gerador da semente. Quem escreve no banco já tem tudo. | rejeitado |
| low | Hash corrompido sai como senha errada, sem linha de log | `Escrever` só registra o que não está no registro, e `Verificar` embrulha o sentinela. Exige linha corrompida; o hash da semente é coberto por `TestVerificarAceitaASenhaDaSemente`. | rejeitado |
| low | `Rotas` com pool e cliente nulos ainda registra as três rotas dependentes | `main.go` sempre passa os dois; os testes que usam nulos só tocam saúde, mídia e rota inexistente. Condicionar registro de rota a estado trocaria o pânico por um 404 silencioso. | rejeitado |
| low | `ttl <= 0` vira Sessão sem expiração no Redis | Exige `AZAMON_SESSAO_EXPIRACAO=0s` ou `Config{}` zerada num chamador novo. A guarda acrescenta ramo por uma configuração que ninguém escreve. | rejeitado |
| low | `strings.ToLower` do Go pode divergir do `lower()` do Postgres em e-mail não-ASCII | Não existe conta com e-mail não-ASCII: as duas são semeadas em ASCII e o cadastro é da Épica 2, que é quem precisa decidir isso. | rejeitado |
| low | Falha do serviço Go na Página de Produto vira 404, e `fetch` rejeitado vira página de erro | FR-7 exige "não encontrado" para identificador inexistente, e isso está cumprido. Distinguir 500 de 404 acrescenta ramo por um desfecho que não muda o roteiro. | rejeitado |
| low | A formatação de uuid fica duplicada entre `identidade` e `catalogo` | Três linhas repetidas. Unificar exige exportar um ajudante novo em `plataforma` — superfície pública por cosmética. | rejeitado |
| low | Nenhum teste de front para `/entrar` e `/produtos/[id]` | `casca.test.mjs` cobre rewrite e guarda offline. Montar bancada de teste de componente é bem mais que correção direta, e o roteiro foi percorrido no navegador. | rejeitado |
| low | Duplicação do literal `http://azamon:8080` e da normalização entre `next.config.ts` e a Página de Produto | Os dois padrões são iguais hoje; unificar exige exportar valor por `env_file` no serviço `web`, mudança de compose fora do que a estória pede. | rejeitado |
| low | O comando de verificação da estória 1.2 aponta para a sonda removida | `spec-1-2` está `done` e é registro histórico; a própria estória 1.2 documentou que a 1.5 removeria a rota. | rejeitado |
| false | Cookie de Sessão sem `Secure` | A demonstração roda em HTTP puro por decisão, e o `AD-9`/`AD-20` fixa exatamente `HttpOnly`, `SameSite=Lax` e valor opaco. `Secure` quebraria o roteiro. | rejeitado |
| false | Corpo com lixo à frente, campos desconhecidos ou `Content-Type` ausente é aceito | Nada disso produz desfecho ruim: o excedente é ignorado, credencial vazia sai em 401 pelo caminho normal, e a consulta é parametrizada pelo sqlc. | rejeitado |
| false | `uuidTexto` ignora `pgtype.UUID.Valid` | `identidade.comprador.id` é chave primária `NOT NULL`; a linha nunca volta com uuid inválido. | rejeitado |
| false | Linha "Sessão ausente ou expirada" da matriz não é exercitada | A própria linha define a entrada como "sem cookie, ou valor fora do Redis", e o subteste cobre os dois — chave expirada é chave ausente. | rejeitado |
| false | `AZAMON_API_URL` não chega ao contêiner `web` | A chave está comentada no `.env` de propósito, porque é do Next e não do binário; a paridade do `.env` conta só linhas ativas. Os dois lados caem no mesmo padrão. | rejeitado |
| false | Rastreamento e `epic-1-context.md` divergem do que o diff entrega | `sprint-status` em `in-progress` durante a revisão é o estado correto do fluxo, e o passo de apresentação o promove. As cinco verificações da épica já estavam fechadas pela 1.2 e pela 1.4. | rejeitado |
| rejeitado | Tarefa cita `api/saude_test.go`, que não existe | A correção seria editar a spec desta construção. Já está registrado nas Notas de Implementação. | rejeitado |

## Design Notes

**Por que `POST /api/v1/sessoes` e não `/login`:** a convenção do projeto é `/api/v1/<recurso-plural>`, e a Sessão é o recurso criado. `GET /api/v1/sessao` no singular lê a Sessão corrente — é o que a Épica 2 reaproveita no menu da conta.

**Por que o mesmo erro para conta inexistente e senha errada:** duas mensagens distintas viram enumeração de contas. O caminho sem linha encontrada também precisa rodar `argon2.IDKey` sobre um hash de descarte, senão o tempo de resposta denuncia a diferença.

**Por que Redis preguiçoso:** `cmd/azamon/main_test.go` aponta `AZAMON_REDIS_URL` para uma porta morta nos dois testes. Exigir Redis no arranque quebra os dois e obriga um contêiner a mais pelo mesmo efeito.

## Verification

**Commands:**
- `docker run --rm -v "${PWD}:/src" -w /src sqlc/sqlc:1.31.1 generate` -- expected: `internal/catalogo/db/gerado/` atualizado sem erro.
- `docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v "${PWD}:/src" -w /src azamon-dev go test ./...` -- expected: tudo verde, incluindo fronteira, paridade do `.env` e os testes novos de `api/`.
- `cd web && npm test && npm run build` -- expected: verificador offline e teste da casca passam, e a construção conclui.

**Manual checks (if no CLI):**
- `docker compose up`, entrar em `http://localhost:3000/entrar` com as credenciais da semente, conferir no inspetor do navegador que `azamon_sessao` veio `HttpOnly` e `SameSite=Lax`, e que a Página de Produto mostra o nome do Comprador e o nome do Vendedor.
