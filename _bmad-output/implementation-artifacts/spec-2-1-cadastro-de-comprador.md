---
title: 'Cadastro de Comprador'
type: 'feature'
created: '2026-09-13'
status: 'ready-for-dev'
route: 'dispatch'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Só existe Comprador semeado. `identidade` sabe autenticar e abrir Sessão, mas não sabe criar conta: não há `INSERT` em `identidade.comprador`, não há geração de hash Argon2id (só verificação), e nenhuma tela cadastra. Sem isso a Épica 2 não começa e nenhum visitante vira Comprador.

**Approach:** Acrescentar `identidade.Cadastrar` — normaliza o e-mail, gera o PHC Argon2id e insere, deixando o `UNIQUE` do banco decidir a duplicidade —, expor `POST /api/v1/compradores` que valida os limites do NFR-14 na decodificação do DTO e já abre a Sessão, e a tela `/cadastrar` com erro em linha por campo.

## Boundaries & Constraints

**Always:**
- E-mail normalizado (minúsculas, bordas aparadas) **antes** da verificação de unicidade — a coluna tem `CHECK (email = lower(email))` e `UNIQUE`.
- A duplicidade é decidida pela violação do `UNIQUE` (SQLSTATE `23505`), nunca por um `SELECT` antes do `INSERT`: dois cadastros simultâneos do mesmo e-mail passariam pelo teste-e-depois-grava.
- Argon2id do AD-9: `m=19456`, `t=2`, `p=1`, sal aleatório de 16 bytes, saída de 32 bytes, formato PHC — os mesmos parâmetros de `media/gerar.go`, senão a semente e as contas novas divergem.
- Todo limite de campo vem de `plataforma.Config` (AD-13/NFR-16) e é verificado no servidor, na decodificação do DTO em `api/`. Chave nova em `config.go` **tem** de entrar no `.env` — `TestEnvTemExatamenteAsChaves…` cai se faltar ou sobrar.
- Erro de campo sai em linha, nomeando o motivo, com `aria-describedby` e foco no campo ao submeter (UX-DR14, UX-DR20(c)). "Já existe uma conta com este e-mail." leva link para o Login.
- Ao final do cadastro o visitante já está autenticado: mesmo cookie opaco, `HttpOnly`, `SameSite=Lax`, TTL de `AZAMON_SESSAO_EXPIRACAO`.

**Never:**
- Não mexer em bloqueio por tentativas (2.2), recuperação de senha (2.3), autorização por dono (2.4), Endereços (2.5) nem menu da conta (2.6).
- Nenhuma migração: `identidade.comprador` já existe com as quatro colunas.
- Nenhuma regra no Next (AD-10): a validação no navegador é conveniência, a defesa é o Go.
- Não editar `internal/identidade/db/gerado/` à mão.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| Cadastro válido | `POST /api/v1/compradores` `{nome,email,senha}` com e-mail livre | 201 com `{"nome":…}`, cookie `azamon_sessao` de 64 hex e chave no Redis com o TTL da config; linha em `identidade.comprador` com e-mail minúsculo e `senha_hash` em PHC | N/A |
| E-mail não normalizado | `" Ana@Exemplo.br "` sobre base vazia | Grava `ana@exemplo.br`; o mesmo e-mail em minúsculas passa a colidir | N/A |
| E-mail já cadastrado | e-mail da semente | 409 `EMAIL_JA_CADASTRADO`, mensagem "Já existe uma conta com este e-mail.", `dados.campo = "email"` | Nada gravado; a corrida entre dois cadastros simultâneos cai aqui pelo `23505` |
| Senha curta | senha com 7 caracteres | 400 `CAMPO_INVALIDO`, `dados.campo = "senha"`, mensagem nomeando o mínimo | Recusado antes de qualquer escrita e antes do Argon2id |
| E-mail sem forma de e-mail | `"ana"` | 400 `CAMPO_INVALIDO`, `dados.campo = "email"` | Idem |
| Campo acima do teto | nome/e-mail/senha acima do limite da config | 400 `CAMPO_INVALIDO` com o limite nomeado no texto | Corpo acima de 4 KiB continua caindo em `ENTRADA_INVALIDA` |
| Corpo malformado | JSON quebrado | 400 `ENTRADA_INVALIDA` | Envelope do AD-14, como no login |

</frozen-after-approval>

## Code Map

- `internal/identidade/identidade.go` -- porta do módulo; `Autenticar` já normaliza com `strings.ToLower(strings.TrimSpace(…))` e `uuidTexto` já converte o `pgtype.UUID`. `Cadastrar` entra aqui, com a mesma assinatura `gerado.DBTX` (não `*gerado.Queries`).
- `internal/identidade/senha.go` -- só tem `Verificar`, que lê os parâmetros do próprio PHC. `Gerar` entra aqui com as constantes do AD-9 e `base64.RawStdEncoding` (sem preenchimento), a mesma forma que `media/gerar.go` escreve.
- `internal/identidade/db/consultas.sql` -- hoje tem só `BuscarCompradorPorEmail`. Acrescentar `CriarComprador :one` com `RETURNING id, nome`.
- `internal/plataforma/erro/erro.go` -- `registro` é fatia ordenada sentinela→(status, código); `Escrever` já repassa `dados` para sentinela conhecido. É o único arquivo de tradução (AD-14).
- `internal/plataforma/config.go` -- `leitor.inteiro` já recusa não-positivo; o bloco final faz as checagens de faixa cruzada.
- `.env` -- espelho obrigatório das chaves lidas; `internal/plataforma/env_test.go` compara os dois conjuntos.
- `api/sessao.go` -- `criarSessao` tem o trecho de `CriarSessao` + `http.SetCookie` a extrair; `corpoMaximo` (4 KiB) e `semCache` já existem e se reaproveitam.
- `api/rotas.go` -- `Rotas` monta o `ServeMux`; `escreverJSON(w, status, v)` aceita 201.
- `api/sessao_test.go` -- `ambiente(t)` sobe Postgres 18.6 + Redis 8.10.1 uma vez para todo o pacote e `TestSessaoEProduto` chama os subtestes; `postar`/`decodificar` são os auxiliares. Caso novo entra como subteste dessa função, **não** como `Test…` próprio.
- `web/app/entrar/page.tsx` -- padrão da tela de identidade: `"use client"`, `fetch` relativo a `/api/...` com `credentials: "same-origin"`, mensagem vinda do envelope.
- `internal/fronteira_test.go` -- tabela do AD-1; `api → identidade` e `plataforma → identidade` já existem, nenhuma aresta nova.
- **Não tocar:** `db/migracoes/`, `db/semente/`, `internal/identidade/db/gerado/` (só via `sqlc`).

## Tasks & Acceptance

**Execution:**
- [ ] `internal/plataforma/config.go` + `.env` -- acrescentar `CompradorNomeMax` (`AZAMON_COMPRADOR_NOME_MAX`, 120), `EmailMax` (`AZAMON_EMAIL_MAX`, 254), `SenhaMin` (`AZAMON_SENHA_MIN`, 8) e `SenhaMax` (`AZAMON_SENHA_MAX`, 128), com checagem de faixa `SenhaMin ≤ SenhaMax` -- NFR-16 exige limiar declarado; o par `.env`/config é verificado por teste.
- [ ] `internal/identidade/senha.go` -- `Gerar(senha string) (string, error)` devolvendo o PHC com os parâmetros do AD-9 -- hoje só existe verificação; sem isto não há como gravar senha.
- [ ] `internal/identidade/db/consultas.sql` -- `CriarComprador :one` (`INSERT … RETURNING id, nome`) e regenerar com `sqlc` -- o pacote gerado é a única porta para o banco.
- [ ] `internal/identidade/identidade.go` -- `ErrEmailJaCadastrado` e `Cadastrar`, traduzindo `pgconn.PgError` `23505` na sentinela -- a unicidade é decidida pelo banco, nunca por leitura prévia.
- [ ] `internal/plataforma/erro/erro.go` -- registrar `ErrEmailJaCadastrado` como 409 `EMAIL_JA_CADASTRADO` e acrescentar `EscreverCampo(ctx, w, campo, mensagem)` → 400 `CAMPO_INVALIDO` com `dados.campo` -- UX-DR16 manda nomear o motivo, e `ENTRADA_INVALIDA` é genérico demais para erro em linha.
- [ ] `api/comprador.go` -- DTO, validação contra a config e o handler `criarComprador`, que cadastra, abre a Sessão e responde 201 -- é onde o NFR-14 é cobrado.
- [ ] `api/sessao.go` -- extrair `abrirSessao(w, r, comprador)` de `criarSessao` -- login e cadastro emitem o mesmo cookie; duplicar perderia um atributo no futuro.
- [ ] `api/rotas.go` -- `POST /api/v1/compradores` -- recurso plural em português, como `sessoes`.
- [ ] `web/app/cadastrar/page.tsx` -- formulário Nome/E-mail/Senha com erro em linha por campo, `aria-describedby`, foco ao submeter, link para o Login no erro de duplicidade e ida a `/` no sucesso -- UX-DR14 e UX-DR20(c).
- [ ] `web/app/entrar/page.tsx` -- link "Criar conta" para `/cadastrar` -- sem o menu da conta (2.6), é a única porta para a tela nova.
- [ ] `api/comprador_test.go` + `internal/identidade/senha_test.go` -- cobrir a matriz de I/O como subteste de `TestSessaoEProduto` e o ciclo `Gerar`→`Verificar` -- o hash tem de autenticar com o mesmo código que lê a semente.

**Acceptance Criteria:**
- Dado um cadastro bem-sucedido, quando se consulta `GET /api/v1/sessao` com o cookie devolvido, então o nome volta sem nenhuma chamada a `POST /api/v1/sessoes`.
- Dada a conta recém-criada, quando se lê `identidade.comprador`, então `senha_hash` começa por `$argon2id$v=19$m=19456,t=2,p=1$`, a senha em claro não aparece em nenhuma coluna nem em log, e `POST /api/v1/sessoes` autentica com ela.
- Dado `go test ./...`, quando a suíte roda, então `TestFronteiraDeModulo` e os dois testes do `.env` continuam verdes.
- Dada a faixa de 360 a 1440 px, quando a tela `/cadastrar` é percorrida só pelo teclado, então todo campo tem rótulo associado, o foco é visível e não há rolagem horizontal.

## Implementation Notes

## Spec Change Log

## Review Triage Log

## Design Notes

`Cadastrar` devolve o mesmo `Comprador` de `Autenticar`, e a duplicidade sai do banco:

```go
linha, err := gerado.New(bd).CriarComprador(ctx, gerado.CriarCompradorParams{...})
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) && pgErr.Code == "23505" {
    return Comprador{}, ErrEmailJaCadastrado
}
```

O contrato do erro em linha é `dados.campo`: a tela liga a mensagem do envelope ao campo de mesmo nome e põe o foco nele. Vale para o 409 da duplicidade e para o 400 de cada regra de campo — uma leitura só no cliente.

`sqlc` não está no PATH; a imagem `sqlc/sqlc:1.31.1` já está local (Docker é dependência do projeto de qualquer forma).

## Verification

**Commands:**
- `docker run --rm -v "$PWD:/src" -w /src sqlc/sqlc:1.31.1 generate` -- expected: sem erro, e `git status` sem diferença ao rodar duas vezes.
- `go build ./...` -- expected: compila.
- `go test ./...` -- expected: tudo verde, incluindo fronteira, `.env` e os casos novos de `api/`.
- `npm --prefix web run build` -- expected: o `prebuild` (guarda offline + `node --test`) passa e o `next build` compila `/cadastrar`.

**Manual checks (if no CLI):**
- `docker compose up`, abrir `/cadastrar`, criar conta com e-mail novo: cai na Vitrine já autenticado. Repetir o mesmo e-mail: erro em linha no campo com link para o Login. Percorrer o formulário só com `Tab`.
