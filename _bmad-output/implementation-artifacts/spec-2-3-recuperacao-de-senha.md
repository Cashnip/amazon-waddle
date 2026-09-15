---
title: 'Estória 2.3 — Recuperação de senha'
type: 'feature'
created: '2026-09-14'
status: 'done'
route: 'dispatch'
baseline_commit: 'dd00d7f4e73083ae8b7b0580fb5367f59f620969'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Um Comprador que esqueceu a senha não tem como voltar. A FR-3 está sem implementação: não existe token de redefinição, não existe tela, e nada encerra as Sessões de quem trocou a senha.

**Approach:** Uma solicitação por e-mail emite um token opaco de uso único no Redis com TTL de 30 minutos e o escreve no log estruturado (AD-20 — não existe serviço de e-mail). Apresentar o token válido redefine a senha pelas regras da 2.1 e apaga todas as Sessões daquele Comprador. Duas telas novas fecham a UX-DR20(a).

## Boundaries & Constraints

**Always:**
- A resposta da solicitação é **idêntica** exista ou não a conta: mesmo status, mesmo corpo, mesmo caminho visível de fora.
- Um Comprador nunca tem dois tokens válidos: a solicitação nova apaga o token anterior.
- Redefinir encerra **todas** as Sessões ativas daquele Comprador — as chaves somem do Redis, não só o cookie.
- A nova senha passa pelas mesmas regras da 2.1 (`SenhaMin`/`SenhaMax` da `Config`), pela **mesma função**, com erro em linha por campo (UX-DR20(c)).
- Token e prazo vivem no Redis com TTL nativo, como Sessão e contador de bloqueio.
- Limiares só da `Config` — `SenhaTokenValidade` já existe e já está no `.env`.

**Never:**
- Não abrir Sessão depois de redefinir — a condição de aceite manda encerrar todas; o Comprador volta pelo Login.
- Não logar e-mail, nome nem senha (AD-15). Só o caminho com o token.
- Nenhuma migração, nenhum Mailpit ou fila (NFR-15), nada do menu da conta (2.6) nem de Endereço (2.5).

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Saída esperada | Tratamento de erro |
|---|---|---|---|
| Solicitação, conta existe | `POST /api/v1/redefinicoes-de-senha` `{"email":"x@y.br"}` | 202, corpo vazio; chave `redefinicao:<token>` com TTL de 30 min; linha de log com o caminho `/redefinir-senha/<token>` | N/A |
| Solicitação, conta não existe | e-mail desconhecido | **O mesmo 202, corpo vazio**; nenhuma chave criada, nenhuma linha com token | N/A |
| Segunda solicitação | conta com token válido | 202; o token anterior deixa de resolver; só um `redefinicao:*` daquele Comprador | N/A |
| E-mail acima de `EmailMax`, ou corpo inválido | `{"email":"<300 caracteres>"}` | **O mesmo 202** — recusa antes de virar chave, sem contar nada a quem chama | N/A |
| Redefinição válida | `PUT /api/v1/redefinicoes-de-senha/{token}` `{"senha":"nova-senha-1"}` | 204; a senha nova autentica e a antiga não; todas as chaves `sessao:*` daquele Comprador somem; o token não resolve de novo | N/A |
| Token expirado, já usado ou inexistente | token qualquer | 404 `TOKEN_INVALIDO` | Mensagem fechada; a tela oferece solicitar outro |
| Senha nova fora dos limites | `{"senha":"curta"}` | 400 `CAMPO_INVALIDO` com `dados.campo="senha"` | O token **continua válido** — recusa de forma não gasta o token |
| Sessão de outro Comprador | dois Compradores com Sessão, um redefine | só as chaves do que redefiniu somem | N/A |

</frozen-after-approval>

## Code Map

- `internal/identidade/sessao.go` — `CriarSessao` gera o token opaco de 256 bits inline (extrair `novoToken()`); `prefixoSessao` é o que `EncerrarSessoesDoComprador` varre.
- `internal/identidade/bloqueio.go` — o precedente: arquivo novo no módulo, prefixo próprio de Redis, TTL nativo. A redefinição segue a mesma forma.
- `internal/identidade/identidade.go` — `normalizarEmail`, `Comprador{ID,Nome}`, `uuidTexto`; `Cadastrar` é o modelo de função que recebe `gerado.DBTX`.
- `internal/identidade/db/consultas.sql` + `db/gerado/` — `BuscarCompradorPorEmail` já devolve `id`; falta o `UPDATE` da senha. **`sqlc` não está no PATH.**
- `internal/plataforma/erro/erro.go` — `registro` é fatia ordenada casada por `errors.Is`; `EscreverCampo` é o erro em linha por campo.
- `api/comprador.go` — `validarComprador` tem os dois ramos de senha a extrair.
- `api/sessao.go` — `corpoMaximo`, `semCache`, `s.cfg`; o teto do e-mail antes de virar chave de Redis é o precedente da linha do `EmailMax`.
- `api/rotas.go` — `Rotas` monta o `ServeMux`; padrão `MÉTODO /caminho`, plural em português.
- `api/sessao_test.go` — `ambiente(t)` monta a `Config` **à mão** (falta `SenhaTokenValidade`: zero-value = TTL zero); `chavesCom`, `decodificar`. Caso novo entra como subteste de `TestSessaoEProduto`, nunca como `Test…` próprio.
- `web/app/cadastrar/page.tsx` — o padrão de erro em linha por campo (`dados.campo`, `aria-describedby`, foco) a copiar na tela de redefinir; `web/app/entrar/page.tsx` recebe o link e é o padrão de casca das duas telas.
- **Não tocar:** `db/migracoes/`, `.env`, `internal/plataforma/config.go`, `web/lib/destino.ts`.

## Tasks & Acceptance

**Execution:**
- [x] `internal/identidade/db/consultas.sql` — `AtualizarSenhaDoComprador :exec` (`UPDATE … SET senha_hash = $2 WHERE id = $1`) — o único jeito de gravar a senha nova. Rodar `sqlc generate`; sem o binário, `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate` — o gerado é commitado e não se edita à mão.
- [x] `internal/identidade/sessao.go` — extrair `novoToken()` de `CriarSessao` e acrescentar `EncerrarSessoesDoComprador(ctx, rdb, compradorID)`, que varre `prefixoSessao*` por `SCAN`, lê os valores e apaga os do dono — sem índice reverso não há como saber os tokens, e índice reverso custaria comando extra em toda leitura de Sessão.
- [x] `internal/identidade/redefinicao.go` — `ErrTokenInvalido`, `SolicitarRedefinicao` (acha o Comprador, apaga o token anterior pelo ponteiro `redefinicao-de:<id>`, grava `redefinicao:<token>` e o ponteiro com o mesmo prazo, devolve token vazio quando a conta não existe) e `RedefinirSenha` (resolve o token, gera o Argon2id, grava, apaga token e ponteiro, encerra as Sessões) — o ciclo do token é regra de domínio, não tradução.
- [x] `internal/plataforma/erro/erro.go` — registrar `identidade.ErrTokenInvalido` → 404 `TOKEN_INVALIDO` — a tela precisa do código próprio, e o recurso de fato não existe.
- [x] `api/comprador.go` — extrair `validarSenha(senha) (campo, mensagem)` dos dois ramos de `validarComprador`, que passa a chamá-la — "as mesmas regras da 2.1" só é verdade se for a mesma função.
- [x] `api/redefinicao.go` — os dois handlers: o `POST` recusa em silêncio (202) acima de `EmailMax` e escreve o caminho `/redefinir-senha/<token>` no `slog` quando há token; o `PUT` valida a senha **antes** de resgatar o token e devolve 204 — a camada que conhece `Config`, HTTP e log.
- [x] `api/rotas.go` — `POST /api/v1/redefinicoes-de-senha` e `PUT /api/v1/redefinicoes-de-senha/{token}`.
- [x] `api/sessao_test.go` — acrescentar `SenhaTokenValidade` ao `ambiente(t)` e chamar os subtestes novos no fim de `TestSessaoEProduto`.
- [x] `api/redefinicao_test.go` — cobrir a matriz inteira, inclusive as duas linhas que só o Redis mostra (token anterior invalidado; Sessões do dono somem e as do outro ficam) — "não vaza conta" é indistinguível de fora, e só o estado do Redis o prova.
- [x] `web/app/esqueci-a-senha/page.tsx` — formulário de e-mail e o estado "solicitação enviada" com texto neutro, que aparece sempre — UX-DR20(a).
- [x] `web/app/redefinir-senha/[token]/page.tsx` — campo de senha nova, erro em linha por campo, estado "token inválido ou expirado" com link para solicitar outro, e ida para `/entrar` no sucesso — UX-DR20(a).
- [x] `web/app/entrar/page.tsx` — link "Esqueci minha senha" — sem o menu da 2.6, é a única porta para o fluxo.

**Acceptance Criteria:**
- Dado `go test ./...`, quando a suíte roda, então `TestFronteiraDeModulo` e os dois testes do `.env` continuam verdes — nenhuma aresta nova, nenhuma variável nova.
- Dado `docker compose logs`, quando uma solicitação é feita para conta existente, então há uma linha JSON com `correlacao` e o caminho da redefinição, e **nenhuma** com e-mail ou senha.
- Dada a faixa de 360 a 1440 px, quando as duas telas novas são percorridas só pelo teclado, então o foco é visível, o erro é anunciado e não há rolagem horizontal.

## Implementation Notes

- A tela de redefinição ficou em dois arquivos, como a do Pedido: o Server
  Component lê o `{token}` do caminho e o filho `redefinir.tsx` (`"use client"`)
  fala com o Go por caminho relativo. Validar o token no servidor antes do PUT
  seria uma ida a mais e uma janela entre a pergunta e a resposta.
- `api/redefinicao_test.go` captura o `slog` por requisição (`slog.SetDefault`
  com um `plataforma.NovoLogger` sobre um buffer) para ler o token pelo mesmo
  canal do Comprador da demonstração — e, de carona, provar a condição de aceite
  do AD-15: a linha tem `correlacao` e o caminho, e nunca o e-mail.
- A igualdade das respostas da solicitação é comparada com os cabeçalhos menos
  o `X-Correlation-Id`, que muda a cada requisição — o mesmo tratamento que os
  subtestes da 2.2 dão ao campo `correlacao` do envelope.
- O "token **expirado**" da matriz não tinha teste próprio: `tokenInvalidoDa404`
  cobria o nunca-emitido e o malformado, e o já-usado vinha do caminho feliz.
  Agora o subteste emite um token de verdade e vence a chave à mão antes de
  apresentá-lo — o mesmo truque que a 2.2 usa para provar a renovação do prazo
  da Sessão, porque com o prazo cheio não há como esperar os 30 minutos.

## Spec Change Log

## Review Triage Log

### Passagem 1 — `blind-hunter`, `edge-case-hunter`, `verification-gap`

| Achado | Veredito | Evidência | Rota |
|---|---|---|---|
| `criarRedefinicao` devolve 500 quando `SolicitarRedefinicao` falha, e essa chamada só falha depois de a conta ter sido achada | medium | Confirmado em `api/redefinicao.go:43`: `pgx.ErrNoRows` devolve `"", nil` **antes** do primeiro comando Redis. Com o Redis degradado, conta existente = 500 e desconhecida = 202 — o oráculo que a rota inteira existe para fechar | patch |
| `EncerrarSessoesDoComprador` aborta a varredura no primeiro erro de `Get`/`Del`, com Sessões do dono ainda vivas e a senha já gravada | medium | Confirmado em `internal/identidade/sessao.go`: o `return` dentro do laço deixa o cursor pela metade | patch |
| A cópia de `/esqueci-a-senha` promete caixa de entrada que não existe (AD-20), contra o comentário do próprio arquivo | medium | Confirmado: "já está a caminho" e "Não chegou?" descrevem e-mail; o token sai no log | patch |
| "Ele vale por 30 minutos." fixa `AZAMON_SENHA_TOKEN_VALIDADE` na tela | medium | Confirmado; a própria seção **Always** deste spec diz "limiares só da `Config`" | patch |
| O aviso permanente de `redefinir.tsx` usa `<Alert>`, e `components/ui/alert.tsx` fixa `role="alert"` | medium | Confirmado em `web/components/ui/alert.tsx:29`: um aviso estático é anunciado como interrupção no carregamento, antes do rótulo do campo | patch |
| O comentário do teto de `EmailMax` diz "vira parte de chave de Redis, como no login" — falso nesta rota | low | Confirmado: as chaves são `redefinicao:<token>` e `redefinicao-de:<id>`; nenhuma chaveia por e-mail. O guarda continua certo, o motivo escrito é que não | patch |
| A linha de log do token não traz marcador `ponytail:`, ao contrário do `SCAN` dois arquivos adiante | low | Confirmado; o AD-20 na espinha tem o seu | patch |
| `api/rotas.go`: "a consuma" | low | Confirmado, erro de digitação | patch |
| Comentários de `redefinicao_test.go` dizem "as quatro entradas" e "As três" sobre uma fatia de seis casos | low | Confirmado; neste repositório o comentário é a especificação | patch |
| Ninguém afirma que o ponteiro `redefinicao-de:<id>` some com o uso, nem `Cache-Control: no-store` nas duas rotas | low | Confirmado: `chavesCom` casa pelo **nome** da chave, e o nome do ponteiro não contém o token; `cabecalhosComparaveis` só prova que as duas respostas concordam entre si | patch |
| `normalizarEmail` na busca da redefinição está sem teste: todo pedido do arquivo manda o e-mail exatamente como gravado | medium | Pré-verificado pela camada de lacunas, com demonstração: remover `normalizarEmail` de `redefinicao.go` deixa a suíte verde, e quem digita `Elisa@Exemplo.br` nunca recebe token — em silêncio, porque a rota é muda por construção | patch |
| O laço de `SCAN` só roda com três chaves de Sessão, então sempre devolve cursor 0 na primeira volta | medium | Pré-verificado, com demonstração: trocar o laço por um `Scan` único deixa a suíte verde. É o mecanismo da condição de aceite principal da estória | patch |
| Canal de tempo: conta existente paga SELECT + 1 GET + até 1 DEL + 2 SET; desconhecida paga só o SELECT | medium (não corrigido) | Real. Mas o sinal aqui é ~3 idas ao Redis em soquete local (~0,3 ms) contra a variação do próprio SELECT e do HTTP — ordens de grandeza abaixo do Argon2id de ~50 ms que justificou o `hashDeDescarte` no login. Igualar exigiria trabalho Redis fictício no caminho da conta inexistente: complexidade especulativa | defer |
| Nada limita `POST /api/v1/redefinicoes-de-senha`, e cada pedido invalida o link que o Comprador legítimo está segurando | medium (não corrigido) | Real, mas é consequência direta do invariante que a FR-3 **exige** ("nova solicitação invalida os tokens anteriores"): toda implementação correta numa rota não autenticada tem essa propriedade. Não é defeito desta implementação, é uma aresta do requisito | defer |
| Se a varredura falhar depois do `UPDATE`, a tela diz "não foi possível redefinir" para uma redefinição que aconteceu pela metade | medium (não corrigido) | Real. Mas 204 com Sessões vivas quebra a condição de aceite, e 500 mente ao contrário — não há opção claramente melhor, e ambas só ocorrem com o Redis fora do ar. A resiliência da varredura foi corrigida (linha acima); o dilema 204-vs-500 fica registrado | defer |
| O guarda de `EmailMax` não é observável: removê-lo deixa a suíte verde | low (não corrigido) | A própria camada dispôs `defer`: o dano é limitado pelo `corpoMaximo` de 4 KiB, e a única afirmação honesta seria sobre um caminho interno, sem gancho no arnês | defer |
| Redefinir a senha não limpa o contador de bloqueio da 2.2 | medium (não corrigido) | Real e verificado: `RedefinirSenha` toca Sessões, nunca `prefixoFalhas`. Quem erra cinco logins, é bloqueado e então redefine continua em 429 com a senha nova até o prazo passar. **Não é desvio deste spec nem da intenção capturada:** a FR-3 do PRD lista quatro consequências testáveis e nenhuma cita o bloqueio, e a Épica trata os dois como regras separadas. Há mais de uma leitura defensável (limpar para todas as origens / só para a origem do pedido / não limpar, honrando a regra plana do bloqueio), e o contador se cura sozinho em 15 min. Precisa da decisão do humano, não de um conserto inferido | defer |
| Token não é de uso único de forma atômica: `Get` agora, `Del` depois do Argon2id | low | Real, mas os dois PUTs concorrentes são o mesmo portador do mesmo link (duplo clique / duas abas); o desfecho é o mesmo de um pedido só, com a última escrita vencendo. O `GetDel` que fecharia a janela inverteria a ordem gravar-antes-de-apagar, que é deliberada e documentada — não é correção direta | rejeitado |
| Escrita do token e do ponteiro não é atômica; duas solicitações interleaved deixariam dois tokens válidos | low | Real, mas o dano é limitado pelo TTL de 30 min e os dois tokens pertencem ao mesmo Comprador legítimo. O conserto é script Lua ou pipeline transacional — complexidade acrescentada | rejeitado |
| `AtualizarSenhaDoComprador` é `:exec` e esconde um `UPDATE` de zero linhas | low | O estado é inalcançável: não existe caminho de exclusão de Comprador em lugar nenhum do sistema. O conserto (`:execrows` + regeneração + ramo novo) é mais que correção direta | rejeitado |
| Sessão com JSON ilegível sobrevive à redefinição, porque o `continue` cobre os dois ramos | low | Inalcançável pelo mesmo motivo: só `CriarSessao` escreve sob `sessao:`, e sempre com `json.Marshal` de um `Comprador` | rejeitado |
| As regiões `role="status"` entram no DOM junto com o conteúdo, e podem não ser anunciadas | low | Real como preocupação, mas é exatamente o padrão que a 2.1 e a 2.2 já estabeleceram em `cadastrar`/`entrar` e que foi aceito nas duas. Divergir aqui criaria dois padrões de anúncio na mesma Épica | rejeitado |
| Nada confirma ao Comprador que a redefinição deu certo — `/entrar` é uma tela de login comum | low | O destino foi decidido no spec aprovado, e um cartão de confirmação é superfície nova, não correção direta | rejeitado |
| `sprint-status.yaml` ainda diz `in-progress` | false | O estado está correto: a estória **está** em curso. A sincronização para `review`/`done` é do passo 5, e ainda não rodou | rejeitado |
| A passagem de teclado de 360 a 1440 px não foi percorrida | medium (não corrigido) | Real: não há navegador conectado nesta máquina. Mesmo desfecho da 2.1 e da 2.2 | defer |

## Design Notes

O token nasce com um ponteiro reverso, e não com um índice:

```
redefinicao:<token>   ->  <comprador_id>   TTL = SenhaTokenValidade
redefinicao-de:<id>   ->  <token>          TTL = SenhaTokenValidade
```

A solicitação nova lê o ponteiro, apaga o token que ele aponta e regrava os dois. É o que faz "nunca dois tokens válidos" ser verdade sem varrer nada: o ponteiro é uma chave só por Comprador, e expira junto.

As Sessões não têm ponteiro. Quem as lê é `LerSessao`, num `GetEx` só, e um índice reverso custaria um comando a mais em **toda** requisição autenticada para servir uma operação que acontece uma vez por esquecimento de senha. `EncerrarSessoesDoComprador` varre por `SCAN` e lê os valores.

## Verification

**Commands:**
- `go build ./...` e `go test ./...` — verde, com os subtestes novos dentro de `TestSessaoEProduto`.
- `cd web && npm run build` — o `prebuild` roda `verificar-offline.mjs` e `node --test`.

**Manual checks:**
- `docker compose up` → `/esqueci-a-senha` com o e-mail semeado → `docker compose logs azamon | grep redefinir-senha` → abrir o caminho → trocar a senha → a Sessão antiga cai em `/` e a senha nova entra em `/entrar`.
- As duas telas a 360 px e a 1440 px, percorridas só pelo teclado.
