---
title: 'Estória 2.4 — Autorização por dono do recurso e separação de papéis'
type: 'feature'
created: '2026-09-15'
status: 'done'
route: 'dispatch'
baseline_commit: '6c6320c1518315548875163c39d04d23da5a4def'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A posse já é verificada na consulta do Pedido desde a 1.8, mas isso é
precedente, não mecanismo: nada obriga o próximo recurso a repetir a forma, não existe
prova automatizada de negação, e os dois papéis do FR-4 não são distinguíveis em lugar
nenhum — `identidade.administrador` está semeado desde a 1.3 e não tem por onde entrar.

**Approach:** A Sessão passa a carregar o papel; cada autenticação consulta a sua própria
tabela; a guarda de Administrador mora no prefixo `/api/v1/admin/` dentro de `Rotas`, e
não em cada handler; e nasce um teste de negação por dono que as Épicas 4, 5 e 6
estendem em vez de reescrever.

## Boundaries & Constraints

**Always:** posse verificada dentro do módulo dono, na mesma consulta que carrega o dado
(`WHERE id = $1 AND comprador_id = $2`) — AD-11. Recurso de outro dono devolve o **mesmo**
erro de inexistente. Papel verificado em `api/`, por prefixo de rota. `Comprador` e
`Administrador` continuam tabelas separadas, sem coluna de papel: cada login consulta
**só** a sua tabela, e quem decide o papel é a rota, não a ordem de consulta. Limiares na
`Config`.

**Never:** coluna de papel; migração nova; `if` de posse depois da leitura; tela ou rota
administrativa de negócio (Épica 3); `identidade.endereco` (2.5); menu da conta (2.6);
promoção de Comprador a Administrador.

## Decisões

1. **A negação por dono é provada com Pedido**, o único recurso de Comprador que existe
   hoje. A 2.5 estende o mesmo teste para Endereço, como a própria estória já prevê para
   Carrinho e Pedido nas Épicas 4 e 6. O NFR-6 fica provado para Endereço uma estória
   depois.
2. **O mecanismo de papéis nasce completo:** login de Administrador, Sessão carregando o
   papel, guarda no prefixo e uma rota administrativa mínima para a guarda ter o que
   proteger. Sem tela — a área administrativa de negócio continua sendo da Épica 3.
3. **A brecha de CSRF fecha aqui:** toda rota que decodifica JSON exige
   `Content-Type: application/json`, o que derruba o formulário cross-site com
   `enctype=text/plain` que hoje fixa Sessão. Nenhum chamador legítimo muda — o `web/` e o
   Provedor Simulado já mandam o cabeçalho.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Saída esperada | Erro |
|---|---|---|---|
| Pedido próprio | Sessão de Comprador A, `GET /api/v1/pedidos/{id de A}` | 200 com o Pedido | — |
| Pedido alheio | Sessão de A, `GET /api/v1/pedidos/{id de B}` | resposta **idêntica** à do uuid inexistente | 404 `NAO_ENCONTRADO` |
| Sem Sessão | sem cookie, rota de Comprador | — | 401 `SESSAO_INVALIDA` |
| Papel errado na loja | Sessão de Administrador em `GET /api/v1/pedidos/{id}` | — | 401 `SESSAO_INVALIDA` |
| Papel errado no admin | Sessão de Comprador (ou nenhuma) em `/api/v1/admin/*` | — | 404 `NAO_ENCONTRADO` |
| Login de Administrador | `POST /api/v1/admin/sessoes` com `admin@azamon.test` | 200, cookie com papel de Administrador | — |
| Administrador na loja | `POST /api/v1/sessoes` com `admin@azamon.test` | — | 401 `CREDENCIAL_INVALIDA` |
| Comprador no admin | `POST /api/v1/admin/sessoes` com `comprador@azamon.test` | — | 401 `CREDENCIAL_INVALIDA` |
| Corpo cross-site | `POST /api/v1/sessoes` com `Content-Type: text/plain` | — | 400 `ENTRADA_INVALIDA` |

</frozen-after-approval>

## Code Map

- `api/sessao.go:147` — `compradorDaRequisicao` resolve o cookie; hoje devolve
  `identidade.Comprador` sem olhar papel. `abrirSessao:104` e `cookieSessao:118` são os
  únicos pontos que emitem cookie. O comentário de `:143` ("por isso isto é uma função, e
  não um middleware") é o que esta estória revisa.
- `api/rotas.go:27` — `Rotas` monta o `ServeMux` e devolve `plataforma.Correlacao(mux)`.
  Padrão mais específico ganha do prefixo (Go 1.22+): é o que deixa o login de
  Administrador fora da própria guarda. `naoEncontrado:65` já é o 404 do envelope.
- `internal/identidade/identidade.go:41` — `Comprador{ID,Nome}` é o que a Sessão
  serializa; `Autenticar:52` e `Cadastrar:75` o devolvem. `hashDeDescarte:36` é o gasto de
  Argon2id que impede enumeração por tempo — a autenticação nova precisa do mesmo.
- `internal/identidade/sessao.go` — `NomeCookieSessao`, `prefixoSessao`, `CriarSessao`
  grava `json.Marshal(Comprador)`, `LerSessao` desserializa e renova o TTL.
- `internal/identidade/db/consultas.sql` — `BuscarCompradorPorEmail` é o molde da consulta
  do Administrador. **`sqlc` não está no PATH:**
  `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`; o gerado é commitado.
- `internal/pedido/db/consultas.sql:41` — `BuscarPedidoDoComprador` já é a forma do AD-11
  (`WHERE p.id = @pedido_id AND p.comprador_id = @comprador_id`). É o alvo do teste de
  negação, não o alvo de mudança.
- `internal/plataforma/erro/erro.go` — `registro` já tem `ErrNaoEncontrado`→404 e
  `ErrSessaoInvalida`/`ErrCredencialInvalida`→401. Nenhum erro novo é preciso.
- `api/sessao_test.go:82` — `ambiente(t)` sobe Postgres+Redis e monta a `Config` à mão;
  `emailSemente`/`senhaSemente` e `origemDeTeste` moram no topo. Caso novo entra como
  subteste, nunca como `Test…` próprio.
- **Não tocar:** `db/migracoes/`, `.env`, `internal/plataforma/config.go`, `web/`.

## Tasks & Acceptance

**Execution:**
- [x] `internal/identidade/db/consultas.sql` — `BuscarAdministradorPorEmail :one`, molde
  do de Comprador; rodar `sqlc generate`.
- [x] `internal/identidade/identidade.go` — `Conta{ID,Nome,Papel}` substitui `Comprador`,
  com `PapelComprador`/`PapelAdministrador`; `Autenticar` e `Cadastrar` carimbam o papel de
  Comprador, e `AutenticarAdministrador` consulta **só** `identidade.administrador`, com o
  mesmo `hashDeDescarte`. Uma função por tabela é o que faz a separação ser estrutural em
  vez de convenção.
- [x] `internal/identidade/sessao.go` — a Sessão serializa a `Conta` inteira; `LerSessao`
  devolve `Conta`. Sem o papel dentro da Sessão a guarda de prefixo iria ao Postgres a
  cada requisição.
- [x] `api/sessao.go` — `compradorDaRequisicao` recusa com `ErrSessaoInvalida` a Sessão
  cujo papel não é Comprador; `administradorDaRequisicao` é o espelho; `abrirSessao` passa
  a receber `Conta`.
- [x] `api/admin.go` — `criarSessaoAdministrador`, `lerSessaoAdministrador` e
  `somenteAdministrador(http.Handler)`, a guarda que devolve o 404 de `naoEncontrado`.
- [x] `api/rotas.go` — `mux.Handle("/api/v1/admin/", s.somenteAdministrador(admin))` com um
  mux próprio, e `POST /api/v1/admin/sessoes` no mux raiz, fora da guarda. Rota nova da
  Épica 3 entra no mux de dentro e herda a guarda sem ninguém lembrar dela.
- [x] `api/rotas.go` — um ponto só que decodifica JSON de corpo e recusa com
  `ErrEntradaInvalida` o que não vem como `application/json`. A recusa mora junto do
  `MaxBytesReader` que já existe em cada handler, e não num middleware: `DELETE` e
  `GET` não têm corpo e não podem passar a exigir cabeçalho.
- [x] `api/autorizacao_test.go` — a matriz inteira, inclusive a igualdade literal entre a
  resposta do Pedido alheio e a do uuid inexistente (menos o `X-Correlation-Id`); o segundo
  Comprador sai de `POST /api/v1/compradores`, sem tocar na semente.
- [x] `api/sessao_test.go` — constantes do Administrador semeado ao lado das do Comprador.
- [x] `README.md` — como entrar como Administrador; a credencial já está na tabela, o que
  faltava era a porta.

**Acceptance Criteria:**
- Dado `go test ./...`, quando a suíte roda, então `TestFronteiraDeModulo` e
  `TestDominioNaoConheceHTTP` continuam verdes — nenhuma aresta nova, e nenhum `net/http`
  dentro de `identidade`.
- Dado o `.env`, quando o teste do ambiente roda, então nenhuma variável nova é exigida —
  esta estória não tem limiar próprio.
- Dada uma rota nova registrada no mux administrativo sem nenhuma checagem no handler,
  quando um Comprador a chama, então a resposta é 404 — a guarda é do prefixo, não do
  handler.

## Implementation Notes

- **Os dois logins passam pelo mesmo `entrar`** (`api/sessao.go`), com a autenticação
  chegando como fecho (`autenticador`). O que muda entre loja e admin é só a tabela
  consultada; duas cópias do fluxo divergiriam no bloqueio por tentativas, e o login de
  Administrador ficaria com um Argon2id sem teto de custo — a conta mais valiosa atrás da
  porta mais fraca. `api/` não nomeia a `gerado.DBTX` (AD-1): o fecho já carrega o pool.
- **`decodificarCorpo` (`api/rotas.go`) substituiu os cinco `json.NewDecoder(MaxBytesReader(...))`**
  espalhados pelos handlers. O `Content-Type` é analisado com `mime.ParseMediaType`, e não
  comparado cru: `application/json; charset=utf-8` é o mesmo tipo e um chamador legítimo
  que mande o parâmetro não pode ser recusado. `criarRedefinicao` continua respondendo o
  mesmo 202 silencioso — a função devolve erro, e quem chama decide o que escrever.
- **`lerSessaoAdministrador` lê a Sessão duas vezes** — uma na guarda e outra no handler —,
  duas `GetEx` da mesma chave. Passar a `Conta` pelo contexto pouparia uma ida e custaria
  um tipo de chave e um cast em todo handler administrativo: o preço certo quando a Épica 3
  trouxer mais que um.
- **`EncerrarSessoesDoComprador` passou a exigir `Papel == PapelComprador`** além do
  identificador na varredura. Colisão de uuid entre as duas tabelas não acontece, mas a
  varredura que apaga Sessão por engano é a que ninguém vê falhar.
- **Os seis ajudantes de requisição dos testes de `api/` passaram a mandar o cabeçalho** —
  é a mesma requisição que o `web/` e o Provedor Simulado já faziam.
- **`api/produto_test.go:pegar` virou delegação** para o `pegarCom` novo, que aceita
  cookie: era o mesmo GET com um parâmetro a mais.

## Spec Change Log

## Review Triage Log

### Passo 1 (3 camadas: blind-hunter, edge-case-hunter, verification-gap)

| Veredito | Achado | Evidência |
|---|---|---|
| high | `GET /api/v1/admin` (sem barra) devolve 307 com `Location` e corpo HTML, sem passar pela guarda | Reproduzido num `ServeMux` com os mesmos padrões: `status=307 location="/api/v1/admin/" guarda=false`. Descobre a subárvore para chamador anônimo (contra a UX-DR9 e a linha da matriz) e dá à API um segundo contrato de erro (contra o AD-14). |
| high | O corpo do 307 é `text/html`, não o envelope do AD-14 | Mesma causa-raiz da linha acima; o corpo medido foi `<a href="/api/v1/admin/">Temporary Redirect</a>`. |
| medium | A matriz de teste não cobre `/api/v1/admin` sem barra nem a paridade de cabeçalhos | Mesma causa-raiz: sem a linha na tabela, o 307 passa verde. |
| medium | `somenteAdministrador` vira qualquer erro em 404, inclusive Redis fora do ar, sem log | `contaDaRequisicao` propaga `fmt.Errorf("ler a Sessão: %w", err)`; a guarda descarta e escreve 404. `erro.Escrever`, que era quem logava, deixa de ser chamado — a queda do Redis vira 404 mudo. |
| medium | O `Content-Type` só é fixado por teste em `/api/v1/sessoes` | Pré-verificado pela camada de lacunas. `POST /api/v1/compradores` e `POST /api/v1/admin/sessoes` também emitem cookie por `abrirSessao`, e nenhum teste manda cabeçalho não-JSON para eles: reverter `criarComprador` ao decodificador antigo mantém a suíte verde. |
| medium | A Sessão sem `papel` (formato anterior) não tem teste nenhum | Pré-verificado. Todo valor de Sessão da suíte nasce de login real; afrouxar para `conta.Papel != "" && conta.Papel != papel` passa verde e entrega o prefixo administrativo a todo navegador anterior ao deploy. |
| medium | `README.md` afirma que a Sessão de Administrador leva 401 "em qualquer rota da loja" | Falso: `encerrarSessao` não olha papel e responde 204, e `/api/v1/saude`, `/api/v1/produtos/{id}` e `/api/v1/media/{arquivo}` servem normalmente. Só as duas rotas com checagem de papel dão 401. |
| medium | `README.md` afirma que tudo sob `/api/v1/admin/` exige Sessão de Administrador | O próprio `POST /api/v1/admin/sessoes` fica fora da guarda por necessidade e responde 401, não 404 — a área continua confirmável. A frase promete mais do que a guarda entrega. |
| low | `lerSessaoAdministrador` responde 401 se a Sessão sumir entre a guarda e a releitura | Janela real, de microssegundos, e só para quem acabou de provar posse da Sessão; mas o 401 revela que a rota existe. Correção direta: usar `naoEncontrado`. |
| low | A guarda `Papel == PapelComprador` na varredura de `EncerrarSessoesDoComprador` não tem teste | `semearSessoes` copia valores já existentes, então toda chave semeada carrega papel; nada afirma que a Sessão de Administrador sobrevive à redefinição de senha de um Comprador. |
| low | O 202 silencioso de `criarRedefinicao` com `Content-Type` errado não tem teste | `api/redefinicao.go` dobra o erro de `decodificarCorpo` no mesmo 202 de propósito, para não enumerar contas; um 400 ali abriria a enumeração e nada falharia. |
| low | `negacaoPorDono` e `guardaDoPrefixoAdministrativo` abrem com `t.Skip` | Os subtestes que produzem a pré-condição dão `t.Fatalf`, então a corrida ficaria vermelha de qualquer jeito — mas a prova da 2.4 some do relatório em vez de ser atribuída. |
| low | O comentário da leitura dupla em `api/admin.go` não tem o prefixo `ponytail:` | O repo usa esse prefixo para dívida deliberada (`internal/pedido/db/consultas.sql`), e é por ele que o ledger a encontra. |
| low (rejeitado) | O contador de bloqueio é compartilhado entre os dois papéis | Real, mas a direção é a conservadora: compartilhar impede que trocar de rota dê orçamento novo a quem força bruta o mesmo e-mail. O lado ruim — login de Administrador bem-sucedido limpar o contador do Comprador homônimo — exige já saber a senha do Administrador e vir da mesma origem. Corrigir custaria mudar três assinaturas em `bloqueio.go` e o comportamento da 2.2. |
| low (rejeitado) | A herança do bloqueio pelo login de Administrador não é testada | `entrar` é uma função só; um fork seria refatoração visível, e o caminho já tem teste pelo login da loja. Afirmar o 429 custaria cinco falhas e sujaria o contador do par usado depois. |
| false | O webhook pode ser recusado por `Content-Type` ausente ou malformado | `cmd/azamon/main.go:203` sempre manda o cabeçalho, e `varreduraLevaOPedidoAPago` roda o binário de verdade sobre HTTP de verdade e falha se a confirmação não chegar a `APLICADA`. |
| false | O README publica a senha do Administrador sem nota de rotação | O próprio README já diz, três linhas acima da tabela: "Não existe produção, então estas credenciais não são segredo; elas estão versionadas junto com a lista". |
| false | Não existe encerramento de Sessão para o Administrador | `encerrarSessao` é agnóstico de papel e apaga a chave pelo valor do cookie: `DELETE /api/v1/sessao` encerra a Sessão de Administrador. A assimetria é de rota, não de função — e é a mesma frase do README que a linha de cima corrige. |
| low (já corrigido) | Rastreamento divergente: `sprint-status.yaml` e a spec em `in-progress` com tudo `[x]` | Divergência recorrente deste repo. A spec já está em `in-review`; o `sprint-status.yaml` é sincronizado no fecho. |

## Design Notes

**Por que 404, e não 403, na área administrativa.** A UX-DR9 e a `EXPERIENCE.md:22` pedem
que a área "não apareça para quem não é Administrador — nem como link desabilitado, nem
como item cinza no menu". Um 403 conta ao Comprador que a rota existe; o 404 é a mesma
resposta da rota inexistente, e é o que "não aparece" quer dizer do lado do servidor. É a
mesma leitura que o AD-11 já faz para recurso de outro dono.

**Por que o web não muda.** A casca da loja só lê `GET /api/v1/sessao`, que a partir daqui
só responde a Sessão de Comprador. Sem menu (2.6) e sem tela administrativa (Épica 3), não
há superfície onde um link de Administrador pudesse vazar: a separação por papel de Sessão
já entrega a UX-DR9 sem uma linha de TSX.

**O mesmo e-mail nas duas tabelas** (duas entradas do `deferred-work.md` esperavam esta
estória). Fica permitido, e a precedência deixa de ser ambígua porque não existe ordem de
consulta: `POST /api/v1/sessoes` só olha `comprador`, `POST /api/v1/admin/sessoes` só olha
`administrador`, e a redefinição de senha continua sendo de Comprador. Proibir exigiria
restrição compartilhada dentro do schema — migração nova, que esta estória não faz.

**Por que o `Content-Type` fecha a fixação de Sessão.** Um formulário HTML cross-site só
consegue emitir `text/plain`, `application/x-www-form-urlencoded` ou `multipart/form-data`
— `application/json` exigiria `fetch`, e aí o preflight do CORS entra na frente. Conferir o
cabeçalho é, na prática, a guarda; `Sec-Fetch-Site` faria o mesmo com suporte de navegador
mais novo e sem ajudar em nada a mais. Os testes de `api/` que hoje postam sem cabeçalho
passam a mandá-lo — é a mesma requisição que o `web/` já faz.

**Sessões antigas no Redis.** O valor gravado ganha o campo `papel`; Sessão criada antes da
subida desserializa com papel vazio e é recusada como inválida. Na demonstração o Redis
sobe junto, e forçar um novo login é o comportamento correto para uma Sessão que não sabe
dizer o que é.

## Verification

**Commands:**
- `go build ./...` e `go test ./...` — verde, com `api/autorizacao_test.go` novo.
- `cd web && npm run build` — o `prebuild` roda `verificar-offline.mjs` e `node --test`.

**Manual checks:**
- `docker compose up` → entrar em `/entrar` como Comprador →
  `curl -b azamon_sessao=<token> localhost:8080/api/v1/admin/sessao` devolve o mesmo
  envelope 404 de uma rota que não existe.
- `curl -X POST localhost:8080/api/v1/admin/sessoes -H 'Content-Type: application/json' -d '{"email":"admin@azamon.test","senha":"azamon-admin"}'`
  devolve 200 e o cookie; o mesmo cookie em `/api/v1/pedidos/{id}` devolve 401.
