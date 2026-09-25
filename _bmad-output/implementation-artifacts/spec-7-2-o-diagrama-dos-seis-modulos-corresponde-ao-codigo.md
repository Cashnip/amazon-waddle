---
title: '7.2 — O diagrama dos seis módulos corresponde ao código'
type: 'chore'
created: '2026-09-25'
status: 'done'
review_loop_iteration: 0
baseline_commit: '5b3650531292506701a89e9892440fac1bc15e4d'
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-7-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `DIAGRAMA-MODULOS.md` e o grafo do AD-1 na espinha (o mesmo desenho, em dois arquivos) param em 2026-09-06. Estão desatualizados em relação ao código: faltam as 9 arestas de `plataforma`, `db`, `media` e `cmd/azamon` que a tabela de `internal/fronteira_test.go` permite; o relógio aparece com um passo e um nome velho (`EmitirDevidas`); `pedido → carrinho` diz "só `Esvaziar`, único que escreve" (falso desde a 5.5); falta `pedido.contador_numero`; e `ProvedorDePagamento` não existe no código. Mesmo assim o teste passa, porque ele confere a tabela que carrega, e não o desenho. É a SM-C4, e os itens `epic-1-retro-item-12` e `epic-3-retro-item-18`.

**Approach:** Redesenhar o grafo a partir do código, com o mesmo bloco mermaid nos dois arquivos, e reescrever a tabela de travessias e a de schemas. Depois, fazer do "`fronteira_test.go` passando é a prova" algo literal: uma guarda nova no mesmo arquivo exige os dois blocos idênticos e o conjunto de arestas igual à tabela `arestas`.

## Boundaries & Constraints

**Always:** Cada rótulo de seta nomeia símbolos exportados que o código de fato chama, conferidos por `grep`. Os 20 termos do §3 do PRD, literais. Emendar o AD-1 no lugar, sem renumerar, com `uv run _bmad/scripts/memlog.py append --workspace <dir da arquitetura>`. Uma entrada no `addendum.md` §10 para a 7.2.

**Ask First:** Mudar a tabela `arestas`, ou qualquer código de produção, para fazer o código caber no desenho. O desenho segue o código. Se uma aresta real parecer defeito, ela vira linha em `deferred-work.md` e o desenho a mostra.

**Never:** Mexer em código fora de `internal/fronteira_test.go`. Editar `SPEC.md` ou `mapa-de-capacidades.md`. Corrigir `deck-banca.html` (é insumo da 7.4 e vira `deferred-work.md`). Decidir se a porta do AD-8 precisa de `interface` Go, que é assunto da 7.3.

## I/O & Edge-Case Matrix

| Cenário | Estado | Esperado |
|---|---|---|
| Em dia | os dois blocos iguais, e as arestas = `arestas` | `TestDiagramaEhATabela` passa |
| Aresta nova no código | `arestas` ganha `busca → identidade`, sem desenho | falha nomeando a aresta que falta no desenho |
| Desenho com seta a mais | mermaid com `carrinho --> identidade` | falha nomeando a aresta fora da tabela |
| Segundo desenho diverge | só a espinha editada | falha: blocos diferentes |
| Nó desconhecido | id fora das áreas da tabela e de `web`/`porta`/`simulado` | falha nomeando o nó |

</frozen-after-approval>

## Code Map

- `internal/fronteira_test.go:20` -- `arestas`, 23 arestas; `area()` já reduz caminho à área. A guarda roda de `internal/`, então os arquivos ficam em `../_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/`.
- `.../DIAGRAMA-MODULOS.md` -- grafo `:14`, travessias `:37`, schemas `:57`, verificação `:67`. Frontmatter `updated`.
- `.../ARCHITECTURE-SPINE.md:44-72` -- AD-1: prosa, mermaid `:50`, "Este grafo… **é** o diagrama". É o primeiro dos três blocos mermaid.
- Travessias reais (grep em `internal/pedido/*.go`, sem teste): `catalogo` Reservar · Liberar · Consolidar · Disponivel · BuscarProduto; `carrinho` Itens · ConfirmarPrecoVisto · Esvaziar; `identidade` BuscarEndereco · BuscarComprador (`pedido.go:847`, Detalhe do Administrador); `pagamento` IniciarTentativa · ConfirmacoesNaoAplicadas · Marcar · TemConfirmacaoPendente · TentativasRestantes · PedidosComAprovacaoSinalizada. `carrinho → catalogo`: BuscarProduto · Resumos. `busca → catalogo`: VIEW `catalogo.produto_visivel` (`internal/busca/db/consultas.sql:9`) + `catalogo.Normalizar` (`busca.go:60`). `api → pagamento`: RegistrarConfirmacao (webhook). `api → media`: `media.Arquivos`.
- `cmd/azamon/main.go:145-188` -- o relógio, com os passos em ordem: `pedido.Varrer` → `Expirar` → `SimularEntrega` (se `AZAMON_ENTREGA_SIMULACAO_ATIVA`) → `pagamento.EmitirConfirmacoesDevidas`. Tique de 1 s (`.env:36`).
- `internal/pagamento/pagamento.go:64,135,309` -- a porta do AD-8 é `IniciarTentativa` mais o webhook; o adapter é `pagamento.Simulado` + `EmitirConfirmacoesDevidas`. Não existe `interface` Go.
- `internal/plataforma/erro` → identidade, catalogo, carrinho, pedido (sentinelas, AD-14; addendum §10 entradas 1.5, 5.6). A raiz `internal/plataforma` é isenta (`fronteira_test.go:41`). Nenhum domínio importa `plataforma`.
- `db/schema_test.go:47,237` -- "catorze tabelas" (inclui `pedido.contador_numero`) e "nenhuma chave estrangeira cruza schema": o par do AD-2.

## Tasks & Acceptance

**Execution:**
- [x] `internal/fronteira_test.go` -- `TestDiagramaEhATabela`: extrair de cada arquivo o bloco mermaid cuja primeira linha é `%% AD-1` (exata, ou seguida de espaço; `%% AD-12` não conta), normalizando CRLF; exigir os dois iguais; ler arestas `a --> b`, `a -.-> b` e `a -->|"…"| b`; mapear `relogio` → `cmd/azamon`; ignorar `web`, `porta` e `simulado`; comparar os dois sentidos com `arestas`. `TestGuardaDoDiagrama` aplica a matriz ao desenho e à tabela reais, mais seta ilegível e linha não reconhecida; `TestExtracaoDoAD1` cobre zero blocos, dois blocos, `%% AD-12` e bloco que não fecha. `TestTabelaEhOCodigo`, o sentido contrário: toda aresta de `arestas` tem ao menos um importe real (a raiz isenta `internal/plataforma` conta como `→ plataforma`), com o caso de mutação de uma aresta sem importe.
- [x] `DIAGRAMA-MODULOS.md` -- novo mermaid (primeira linha `%% AD-1 — conferido por internal/fronteira_test.go`, nós declarados antes das arestas, setas pontilhadas para `plataforma`/`db`/`media`); tabela de travessias com os símbolos reais e o AD-3 emendado; `contador_numero` e a nota sobre `public.semente`; verificação nomeando a guarda e os dois subtestes de `db/schema_test.go`; porta descrita como as duas operações do AD-8; `updated: 2026-09-25`.
- [x] `ARCHITECTURE-SPINE.md` -- AD-1: o mesmo bloco, byte a byte, e a prosa com `plataforma/erro` e a guarda.
- [x] `.../architecture-azamon-2026-09-05/.memlog.md` -- `(change by dev)` pelo script.
- [x] `prd-azamon-2026-08-14/.memlog.md` -- `(change by dev)` pelo script, registrando a entrada do addendum §10 (política do AGENTS.md).
- [x] `addendum.md` §10 -- entrada da 7.2.
- [x] `deferred-work.md` -- quatro entradas: porta do AD-8 sem `interface` Go (para a 7.3); tabela do AD-6 pondo `Expirar` e `SimularEntrega` dentro de `Varrer` (para a 7.3); `deck-banca.html` com `ProvedorDePagamento` e sem `contador_numero` (para a 7.4); `db/schema_test.go` sem conferir o schema `carrinho` (sem dono ainda).
- [x] `sprint-status.yaml` -- `epic-1-retro-item-12` e `epic-3-retro-item-18` → `done`, com referência a esta spec.

**Acceptance Criteria:**
- Dado o código em `main`, quando os testes de `internal/` rodam, então `TestFronteiraDeModulo`, `TestDominioNaoConheceHTTP`, `TestDiagramaEhATabela`, `TestGuardaDoDiagrama`, `TestExtracaoDoAD1` e `TestTabelaEhOCodigo` passam.
- Dada cada mutação da matriz, aplicada por `TestGuardaDoDiagrama` ao desenho e à tabela reais, então a guarda devolve o problema previsto, e o caso "em dia" não devolve nenhum.
- Dado `grep -n "EmitirDevidas\|ProvedorDePagamento\|Único ponto" DIAGRAMA-MODULOS.md ARCHITECTURE-SPINE.md`, então não resta nenhuma afirmação que o código desminta.

## Design Notes

A guarda exige os dois blocos idênticos. Os dois arquivos continuam com o grafo porque a espinha é o substrato que os agentes leem, e o `DIAGRAMA-MODULOS` é o que se lê sozinho. Ela confere só as arestas; os rótulos são conferidos por `grep`, porque um parser de símbolos seria cerimônia. `plataforma`, `db` e `media` entram pontilhados: estão na tabela e são exaustivos, mas não são módulos do NFR-2.

## Verification

**Commands:**
- `docker run --rm -v "$PWD":/src -w /src -v azamon-gocache:/root/.cache -v //var/run/docker.sock:/var/run/docker.sock -e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal -e TESTCONTAINERS_RYUK_DISABLED=true azamon-dev:latest go test ./internal/ ./db/` -- expected: `ok` nos dois.
- `git diff --stat` -- só os sete documentos da lista de tarefas (`DIAGRAMA-MODULOS.md`, `ARCHITECTURE-SPINE.md`, os dois `.memlog.md`, `addendum.md`, `deferred-work.md` e `sprint-status.yaml`), `internal/fronteira_test.go` e esta spec.

**Manual checks:**
- O mermaid renderiza (GitHub ou a pré-visualização do editor) sem nó solto nem seta sem rótulo nas arestas de domínio.

## Review Triage Log

Duas lentes (Blind Hunter e Verification Gap; a de bordas ficou de fora pelo raio de diff, que é documentação mais um arquivo de teste, regra do HANDOFF). Nenhum intent_gap nem bad_spec.

- **patch** (aplicados): sentido contrário, tabela ⊆ código (`TestTabelaEhOCodigo`; as duas lentes acharam); marca `%% AD-1` exata, para `%% AD-12` não colidir; extração pura com `TestExtracaoDoAD1` e os ramos de falha do leitor testados; caso "só a espinha editada" tirando seta conferida; comentários sobre o diretório de planejamento, o `.dockerignore` e a sintaxe mermaid aceita; legenda das pontilhadas com o webhook; "as 9 entram pontilhadas" corrigido no addendum e no memlog; rótulos com `SementeGrande`, `NovoLogger`, `AbrirRedis` e `(Simulado, enviar)`; exceção das setas sem rótulo; redação de `plataforma/erro`; termos do §3 por extenso; dono do 4º adiado; Tasks e AC desta spec alinhados ao entregue. As mutações da matriz passaram de conferência à mão a `TestGuardaDoDiagrama`, exigido pela auditoria da matriz.
- **defer**: nenhum novo. O AD-6 contraditório já tem entrada da 7.3.
- **reject**: menções de processo no `DIAGRAMA-MODULOS.md`, renomear o nó `relogio`, seta duplicada ou estilo de linha não conferidos, status desencontrado entre spec e sprint (sincronizado no fecho), corrigir o schema `carrinho` em `db/schema_test.go` (a spec proíbe; fica no adiado).

## Suggested Review Order

**A prova que antes não existia**

- A guarda: os dois blocos iguais, e as setas iguais à tabela nos dois sentidos.
  [`fronteira_test.go:148`](../../internal/fronteira_test.go#L148)

- O sentido contrário: toda aresta da tabela sustentada por um importe real.
  [`fronteira_test.go:323`](../../internal/fronteira_test.go#L323)

- O leitor do mermaid: sintaxe fechada, e o que não entende falha alto.
  [`fronteira_test.go:377`](../../internal/fronteira_test.go#L377)

- A guarda morde: cada linha da matriz aplicada ao desenho e à tabela reais.
  [`fronteira_test.go:162`](../../internal/fronteira_test.go#L162)

**O desenho redesenhado a partir do código**

- O bloco do AD-1: 23 arestas, rótulos com os símbolos que o código chama.
  [`DIAGRAMA-MODULOS.md:15`](../planning-artifacts/architecture/architecture-azamon-2026-09-05/DIAGRAMA-MODULOS.md#L15)

- O que a guarda confere, o que fica fora e o que as pontilhadas querem dizer.
  [`DIAGRAMA-MODULOS.md:60`](../planning-artifacts/architecture/architecture-azamon-2026-09-05/DIAGRAMA-MODULOS.md#L60)

- Travessias: `pedido → carrinho` coerente com o AD-3 emendado na 5.5.
  [`DIAGRAMA-MODULOS.md:82`](../planning-artifacts/architecture/architecture-azamon-2026-09-05/DIAGRAMA-MODULOS.md#L82)

- Como a regra é verificada, com a cobertura real do par no banco.
  [`DIAGRAMA-MODULOS.md:106`](../planning-artifacts/architecture/architecture-azamon-2026-09-05/DIAGRAMA-MODULOS.md#L106)

**A espinha e os registros**

- O mesmo bloco no AD-1, e a prosa que nomeia as duas guardas.
  [`ARCHITECTURE-SPINE.md:51`](../planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md#L51)

- A decisão com o porquê, no addendum vivo (SM-7).
  [`addendum.md:425`](../planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md#L425)

**Periféricos**

- Os quatro adiados: porta sem `interface`, AD-6, deck, schema `carrinho`.
  [`deferred-work.md:524`](deferred-work.md#L524)

- Os dois itens de retro fechados por esta estória.
  [`sprint-status.yaml:201`](sprint-status.yaml#L201)
