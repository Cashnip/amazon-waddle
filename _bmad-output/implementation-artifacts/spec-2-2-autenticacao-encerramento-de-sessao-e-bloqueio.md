---
title: 'Autenticação, encerramento de Sessão e bloqueio por tentativas'
type: 'feature'
created: '2026-09-14'
status: 'done'
route: 'dispatch'
baseline_commit: '4452994135df566724b18729bfcfca5d1f0953cf'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Desde a 1.5 a Sessão só nasce. Não há como encerrá-la no servidor, nada conta tentativa falha — `AuthTentativasMax` e `AuthBloqueioDuracao` estão na `Config` e no `.env` sem nenhum leitor —, e a tela que leva 401 mostra o erro e para: o Comprador não é levado ao Login nem volta ao ponto em que parou.

**Approach:** Contador `(e-mail, origem)` no Redis com TTL nativo, consultado **antes** do Argon2id e zerado pelo login válido; `DELETE /api/v1/sessao` apagando a chave e o cookie; e o front que vê 401 vai a `/entrar?destino=…` e é devolvido ao destino depois de entrar.

## Boundaries & Constraints

**Always:**
- Credencial inválida continua saindo como hoje: um 401 `CREDENCIAL_INVALIDA` idêntico para conta inexistente e para senha errada — é a única exceção declarada a nomear o motivo (UX-DR16).
- O contador conta **exista ou não a conta**, e a resposta do bloqueio é a mesma nos dois casos: se o bloqueio só valesse para conta existente, a própria mensagem confirmaria o e-mail.
- Chave do contador por e-mail **normalizado** (`lower`+`trim`, a mesma forma do `Autenticar`) e origem, com TTL nativo — `AuthBloqueioDuracao`. Nenhum limiar literal: os dois vêm da `Config` (AD-13/NFR-16), e **nenhuma chave nova entra no `.env`**.
- Requisição bloqueada responde 429 `MUITAS_TENTATIVAS` nomeando os minutos configurados, **sem gastar Argon2id** — o bloqueio é também o limite de custo da rota.
- Login válido esquece as falhas daquele par; senão o Comprador que erra quatro vezes e acerta fica com quatro falhas penduradas.
- Encerrar a Sessão apaga a **chave no Redis** (o cookie sozinho não invalida nada) e expira o cookie. Encerrar sem Sessão, com cookie desconhecido ou malformado responde o mesmo 204 — quem sai não precisa saber se estava dentro.
- Destino de retorno é só caminho relativo: tem de começar por `/` e não por `//`. Qualquer outra coisa vira `/`.
- **Decidido:** o TTL da Sessão **desliza** — toda leitura da Sessão renova o prazo (`GetEx`), que é o que o §7.1 quer dizer com "expiração por inatividade". Uma aba de acompanhamento aberta renova a Sessão enquanto viver, e isso é aceito.
- **Decidido:** "origem" é o **`RemoteAddr`** da requisição, sem ler `X-Forwarded-For`: o cabeçalho chega do navegador e quem quisesse escapar do bloqueio só teria de variá-lo. Em compose isso faz a origem ser constante, e o par vale na prática como só-e-mail — o bloqueio protege a conta e não distingue clientes.

**Never:**
- Não mexer em recuperação de senha (2.3), autorização por dono (2.4), Endereços (2.5) nem menu da conta (2.6) — o "Sair" desta estória mora na `Saudacao`, que é a casca que já conhece a Sessão.
- Nenhuma migração e nenhuma coluna nova: contador e Sessão vivem no Redis.
- Nenhuma regra no Next (AD-10): o front só lê o status e navega.
- Não mexer no `POST /api/v1/compradores` — limite por origem no cadastro está no `deferred-work.md` e não é desta estória.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| Sexta tentativa | 5 falhas do mesmo par (e-mail, origem) | 429 `MUITAS_TENTATIVAS`, "Muitas tentativas. Tente novamente em 15 minutos.", sem cookie | Nenhum Argon2id é gasto; o corpo não diz se a conta existe |
| Conta inexistente bloqueia igual | 5 falhas com e-mail que não existe | Mesmo 429, corpo idêntico ao do e-mail existente (só a correlação difere) | N/A |
| Origem diferente não herda | 5 falhas de um `RemoteAddr`, 1ª de outro | A segunda origem entra em 401 `CREDENCIAL_INVALIDA`, não em 429 | N/A |
| Sessão lida renova o prazo | `GET /api/v1/sessao` com cookie válido | O TTL da chave volta a ser o `SessaoExpiracao` cheio | Sessão inexistente continua em 401, sem criar chave |
| Login válido zera | 4 falhas e depois a senha certa | 200 com cookie; o par não tem mais contador no Redis | N/A |
| Encerrar Sessão | `DELETE /api/v1/sessao` com cookie válido | 204; a chave da Sessão some do Redis e o cookie volta expirado | N/A |
| Encerrar sem Sessão | `DELETE` sem cookie, ou com valor desconhecido/curto | 204, o mesmo corpo vazio | Nunca 401: sair é idempotente |
| Cookie encerrado não volta | `GET /api/v1/sessao` com o cookie recém-encerrado | 401 `SESSAO_INVALIDA` | N/A |
| Tela com 401 | `Comprar`/`Acompanhamento` levam 401 | Navega para `/entrar?destino=<caminho atual>` e, ao entrar, volta ao destino | Erro que não é 401 continua em linha, como hoje |
| Destino forjado | `/entrar?destino=http://mal` ou `//mal` | Depois do login vai para `/` | N/A |

</frozen-after-approval>

## Code Map

- `internal/identidade/sessao.go` -- `CriarSessao`/`LerSessao`, `prefixoSessao`, `NomeCookieSessao`, `tamanhoToken`. `LerSessao` já valida a forma do token antes de tocar o Redis; encerrar entra aqui, ao lado.
- `internal/identidade/identidade.go` -- porta do módulo (AD-1); `Autenticar` normaliza com `strings.ToLower(strings.TrimSpace(…))` — a chave do contador usa a **mesma** forma.
- `internal/plataforma/erro/erro.go` -- `registro` (sentinela→status/código) e `EscreverCampo`, que é o precedente para mensagem com limiar da `Config` fora do `registro`. `envelopar` é privado e é o único ponto que serializa.
- `api/sessao.go` -- `criarSessao`, `abrirSessao`, `compradorDaRequisicao`, `semCache`, `corpoMaximo`. É aqui que a `Config` chega (`s.cfg`).
- `api/rotas.go` -- `Rotas` monta o `ServeMux`; padrão `MÉTODO /caminho`, e dois handlers no mesmo padrão derrubam o arranque.
- `api/sessao_test.go` -- `ambiente(t)` sobe Postgres+Redis uma vez e monta a `Config` **à mão**: sem `AuthTentativasMax`/`AuthBloqueioDuracao` o zero-value bloqueia toda tentativa. `postar` fixa `/api/v1/sessoes`; caso novo entra como subteste de `TestSessaoEProduto`, nunca como `Test…` próprio.
- `web/app/entrar/page.tsx` -- formulário do login; lê a mensagem do envelope (AD-14). Recebe o `destino`.
- `web/app/produtos/[id]/saudacao.tsx` -- a única casca que já lê `GET /api/v1/sessao`; é onde o "Sair" cabe sem o menu da 2.6.
- `web/app/produtos/[id]/comprar.tsx` e `web/app/pedidos/[id]/acompanhamento.tsx` -- os dois lugares que hoje recebem 401 e param na mensagem.
- **Não tocar:** `db/migracoes/`, `internal/identidade/db/`, `.env`, `internal/plataforma/config.go` (os dois limiares já existem).

## Tasks & Acceptance

**Execution:**
- [x] `internal/identidade/bloqueio.go` -- `RegistrarFalhaDeLogin` (INCR + expiração na primeira falha), `LoginBloqueado` e `EsquecerFalhas`, sobre a chave do par normalizado -- o contador é dado com prazo de validade e mora no Redis por TTL nativo, como a Sessão.
- [x] `internal/identidade/sessao.go` -- `EncerrarSessao(ctx, rdb, token)` apagando a chave; token de forma errada não é erro -- invalidar no servidor é o que a condição de aceite pede.
- [x] `internal/identidade/sessao.go` -- `LerSessao` passa a renovar o prazo na mesma ida ao Redis (`GetEx`), recebendo o TTL de quem chama -- expiração por inatividade, e num comando só: ler e depois expirar seriam duas idas e uma janela entre elas.
- [x] `internal/plataforma/erro/erro.go` -- `EscreverBloqueio(ctx, w, minutos)` → 429 `MUITAS_TENTATIVAS` -- a mensagem nomeia o limiar da `Config`, e por isso fica fora do `registro`, como o `EscreverCampo`.
- [x] `api/sessao.go` -- consultar o bloqueio antes de autenticar, registrar a falha no `ErrCredencialInvalida`, esquecer no sucesso, e `encerrarSessao` apagando chave e cookie -- é a camada que conhece a `Config` e o HTTP.
- [x] `api/rotas.go` -- `DELETE /api/v1/sessao` -- o singular já é a Sessão corrente; encerrar é apagar esse recurso.
- [x] `web/app/entrar/page.tsx` -- ler `destino` da query, validar que é caminho relativo e navegar para ele no sucesso; sem `destino`, vai para `/` -- fecha o "volta para onde estava".
- [x] `web/app/produtos/[id]/saudacao.tsx` -- botão "Sair" que chama o `DELETE` e recarrega a casca -- sem o menu da 2.6, é a única porta para encerrar a Sessão.
- [x] `web/app/produtos/[id]/comprar.tsx` + `web/app/pedidos/[id]/acompanhamento.tsx` -- no 401, navegar para `/entrar?destino=<caminho atual>` -- "Sessão expirada em qualquer tela leva ao Login preservando o destino".
- [x] `api/sessao_test.go` -- acrescentar os dois limiares ao `ambiente(t)` e cobrir a matriz como subtestes de `TestSessaoEProduto`, em ordem própria (o bloqueio suja o par por 15 min) -- o zero-value da `Config` bloquearia toda a suíte.
- [x] `web/lib/destino.ts` + `web/scripts/destino.test.mjs` -- o guarda do destino saiu para módulo `.ts` próprio e ganhou teste -- a linha "destino forjado" da matriz não tinha cobertura: o `node --test` lê `.ts` direto (como o `next.config.ts` em casca.test.mjs), mas não o `.tsx` da página.

**Acceptance Criteria:**
- Dada uma Sessão encerrada, quando qualquer rota autenticada é chamada com o mesmo cookie, então a resposta é 401 — a chave não existe mais no Redis, e não só o cookie foi apagado.
- Dado `go test ./...`, quando a suíte roda, então `TestFronteiraDeModulo` e os dois testes do `.env` continuam verdes — nenhuma aresta nova entre módulos, nenhuma variável nova.
- Dada a faixa de 360 a 1440 px, quando `/entrar` e a casca com "Sair" são percorridas só pelo teclado, então o foco é visível, o botão é alcançável por `Tab` e não há rolagem horizontal.

## Implementation Notes

- O contador vive em `falhas:<e-mail normalizado>|<origem>` com TTL nativo, como
  a Sessão: `INCR` seguido de `EXPIRE NX`. O `NX` — e não um `EXPIRE` no ramo
  `n == 1` — porque os dois comandos não são atômicos: um `EXPIRE` que falhe, ou
  um processo que morra entre os dois, deixaria o contador sem prazo nenhum e o
  par bloqueado para sempre. O `NX` cura a chave sem prazo na falha seguinte e
  não toca na que já tem, então a janela continua não-deslizante.
- A origem é o **host** do `RemoteAddr`, sem a porta (`net.SplitHostPort`, caindo
  para o valor cru quando a divisão falha). Com a porta efêmera junto, cada
  conexão TCP viraria uma chave nova e o bloqueio nunca dispararia fora do teste,
  que é o único lugar onde o `RemoteAddr` é fixo.
- O e-mail é recusado acima de `EmailMax` antes da consulta do bloqueio, com o
  401 de sempre: ele vira parte de chave de Redis com prazo de 15 minutos, e sem
  teto um chamador não autenticado cunharia chaves de 4 KiB — o tamanho do corpo
  — a cada requisição.
- `normalizarEmail` saiu para função em `internal/identidade/identidade.go` e é
  usada pelos três lugares (`Autenticar`, `Cadastrar`, `chaveFalhas`). Duas
  cópias da expressão divergiriam, e o contador passaria a contar por caixa: cinco
  tentativas de graça a cada variação de maiúsculas.
- `LerSessao` recebeu o TTL por parâmetro e troca `Get` por `GetEx` — ler e depois
  expirar seriam duas idas ao Redis, com uma janela entre elas. O domínio não
  conhece a `Config`: quem passa o prazo é `api/`.
- `cookieSessao(valor, maxAge)` passou a montar o cookie para emitir e para
  expirar. O navegador só apaga o cookie quando Nome e Path batem com os da
  emissão, e duas cópias divergiriam na primeira mudança.
- `EscreverBloqueio` fica fora do `registro` pelo mesmo motivo do `EscreverCampo`:
  a mensagem nomeia o limiar, e limiar vem da `Config`. Recebe a **duração**, e
  não minutos já contados: arredonda para cima (um `=30s`, que a `Config` aceita,
  mandaria tentar de novo "em 0 minutos" se truncasse) e trata o singular.
- Falha ao gravar/apagar o contador não muda a resposta: o 401 já está decidido, e
  virá-lo em 500 por causa do contador diria ao atacante que a rota tropeçou. Vai
  para o `slog`. O `DELETE`, ao contrário, responde 500 se o Redis recusar o
  apagamento: prometer que a Sessão morreu sem ter apagado seria pior.
- A tela `/entrar` lê o `destino` de `window.location.search` na hora de navegar, e
  não por `useSearchParams` — o hook obrigaria a envolver a página num `<Suspense>`
  por causa da renderização estática do Next.
- `web/lib/destino.ts` é módulo `.ts` puro (sem JSX) para o `node --test` do
  `prebuild` poder importá-lo: o Node 24 lê tipo direto. Guarda `destinoSeguro` e
  `paraLogin`, que é o que as duas telas do 401 chamam — o link inclui `pathname`
  **e** `search`, porque voltar sem a busca não é voltar ao ponto em que o
  Comprador parou. São três recusas, e as duas últimas burlam a primeira sozinhas:
  `//mal` e `/\mal` (o navegador normaliza a barra invertida); caractere de
  controle, porque `/%09/mal` vira `//mal` depois de o navegador o descartar; e o
  próprio `/entrar`, que faria a página empurrar para si mesma e prender o botão
  em "Entrando…".
- O alerta "Sessão aberta. Olá, {nome}" de `/entrar` saiu: o sucesso agora navega,
  como no cadastro da 2.1.
- Em `acompanhamento.tsx` o `clearInterval` do 401 virou navegação, e a condição
  que restou é só o 404 — o 401 deixou de chegar ali.
- Os subtestes da 2.2 são os últimos de `TestSessaoEProduto`, nesta ordem: o
  bloqueio suja o par por 15 minutos e não há como desfazê-lo sem apagar chave do
  Redis por fora da API. `encerrar` abre a sua própria Sessão para não matar o
  `cookieValido` de quem veio acima, e `loginValidoEsqueceAsFalhas` cadastra a
  conta que usa — o `emailSemente` já acumulou falha nos subtestes da 1.5.
- A renovação do prazo só é observável depois de encurtar o TTL à mão: com o
  prazo cheio, uma leitura que não renovasse passaria batida. O mesmo truque
  prova a janela não-deslizante do contador — as falhas acontecem em
  milissegundos, e o TTL em segundos não distinguiria as duas leituras.
- Consertos da revisão (passagem 1): host sem porta na origem; teto do e-mail
  antes de virar chave de Redis; `EXPIRE NX` no lugar do ramo `n == 1`; minutos
  arredondados para cima dentro do `EscreverBloqueio`; recusa de caractere de
  controle e do próprio `/entrar` no `destinoSeguro`; `paraLogin` compartilhada
  pelos dois sítios do 401, com a busca junto; e os testes que faltavam — TTL do
  contador (nascimento e não-deslizamento), chave pelo e-mail normalizado
  (tentativa com outra caixa e bordas), e `EscreverBloqueio` em unidade com os
  dois ramos da mensagem.
- **Uma linha da matriz fica sem teste automatizado:** "Tela com 401 → navega
  para `/entrar?destino=…`". O guarda do destino foi extraído e testado, mas o
  ramo que dispara a navegação vive dentro de componente React, e `web/` não tem
  jsdom nem biblioteca de render — montar uma seria dependência nova, fora do que
  esta estória pede. Verificado por leitura em `comprar.tsx` e
  `acompanhamento.tsx`, e consta da verificação manual. Anotado em
  `deferred-work.md`, como o contrato de `dados.campo` da 2.1.
- **AC da acessibilidade fechado pela metade**, como na 2.1. Na fonte: o "Sair" é
  um `<button>` de verdade (`Button variant="link"`), alcançável por `Tab`, e o
  `buttonVariants` traz `focus-visible:ring-3`; `/entrar` não mudou de contêiner.
  A parte visual — foco visível a 3:1 e ausência de rolagem horizontal de 360 a
  1440 px — não foi percorrida em navegador: a extensão do Chrome não está
  conectada nesta máquina. Fica como verificação manual antes de aceitar.

## Spec Change Log

## Review Triage Log

Passagem 1 — três camadas (blind-hunter, edge-case-hunter, verification-gap), 25 achados em 15 grupos.

| # | Achado | Veredito | Evidência | Rota |
|---|---|---|---|---|
| 1 | `origem := r.RemoteAddr` carrega a porta efêmera, e cada conexão TCP vira chave nova | high | Confirmado: o `net/http` escreve `RemoteAddr` como `IP:porta`, então o bloqueio nunca dispararia fora do teste — e o `httptest` fixa o valor, que é por que a suíte não viu. Derruba a condição de aceite central da estória | patch |
| 2 | `INCR` + `EXPIRE` não são atômicos: `EXPIRE` que falha deixa contador sem prazo | medium | Real: só um login válido apagaria a chave, e o próprio bloqueio o impede — o par fica preso para sempre. O `ExpireNX` incondicional resolve em uma chamada e ainda cura a chave sem prazo (o `LoginBloqueado` que não confere TTL ausente cai no mesmo grupo) | patch |
| 3 | O guarda do destino aceita tabulação, CR e LF: `/%09/mal.com` vira `//mal.com` | medium | Verificado no regex: `[^/\]` casa a tabulação, e o navegador descarta o caractere antes de resolver a URL. É o mesmo redirecionador aberto que a barra invertida existia para fechar | patch |
| 4 | O e-mail do login vira chave de Redis sem teto além dos 4 KiB do corpo | medium | Real: o cadastro cobra o `EmailMax` e o login não, então um chamador não autenticado cunha chaves de ~4 KiB com TTL de 15 min a cada requisição | patch |
| 5 | `int(d.Minutes())` trunca: `AZAMON_AUTH_BLOQUEIO_DURACAO=30s` produz "em 0 minutos" | low | Real e aceito pela `Config` (que só confere positividade); o conserto é arredondar para cima, correção direta sem ramo novo | patch |
| 6 | A mensagem nomeia a janela inteira, e não o que resta do prazo | false | É o texto que a UX especifica ("Tente novamente em 15 minutos.") e o que a estória pede — nomear o limiar da `Config`. Servir o restante do TTL seria outra mensagem, não um conserto | rejeitado |
| 7 | O 429 não traz `Retry-After` | low | Real, mas a API não tem cliente de máquina: quem lê o 429 é a tela, pelo envelope. Acrescentar cabeçalho é superfície nova, não correção direta | rejeitado |
| 8 | `/cadastrar` descarta o `destino`, e o link "Criar conta" perde a busca | low | Real, mas a condição de aceite fala do Login preservando o destino; a porta do cadastro nunca teve destino nenhum. Vai ao ledger | defer |
| 9 | Os dois sítios do 401 mandam só `pathname`, descartando a busca, e repetem o literal | low | Real: o `destinoSeguro` é testado para preservar `/produtos/1?x=2`, e o lado que monta joga isso fora. O `paraLogin` ao lado do guarda fecha a ida e volta e a põe sob teste | patch |
| 10 | Os subtestes novos dependem de ordem e compartilham o e-mail da semente | low | Real, mas é o padrão já documentado do arquivo desde a 1.5 (`ambiente(t)` é do `TestSessaoEProduto` inteiro), e o comentário da ordem está escrito. Isolar exigiria reescrever a suíte | rejeitado |
| 11 | Nada prova que o contador chaveia pelo e-mail normalizado | medium | Pré-verificado pela camada de lacunas: trocar `normalizarEmail(email)` por `email` cru em `chaveFalhas` mantém a suíte verde, e em produção cada variação de caixa daria cinco tentativas de graça | patch |
| 12 | Nada confere o TTL do contador, nem que a janela não desliza | medium | Pré-verificado: apagar o `EXPIRE` (bloqueio eterno) ou soltá-lo do ramo (janela deslizante) não derruba nada — os subtestes só contam chaves | patch |
| 13 | O ramo singular do `EscreverBloqueio` nunca executa | low | Pré-verificado: `erro_test.go` não chama `EscreverBloqueio`, e a única cobertura exige que a mensagem contenha "15". Inverter a condição não derruba nada | patch |
| 14 | `?destino=/entrar` faz a página empurrar para si mesma e prender o botão | low | Real: o `setEnviando(false)` saiu do caminho de sucesso, e um push para a rota corrente não desmonta o componente. Recusar o próprio `/entrar` no guarda é correção direta e já testável | patch |
| 15 | "Sair" recarrega mesmo quando o `DELETE` falha; e `GetEx` com TTL zero tornaria a Sessão imortal | low | Os dois reais e os dois inalcançáveis no uso corrente: o 500 do "Sair" só existe com o Redis fora (quando a casca inteira já está quebrada) e se anuncia sozinho — a saudação volta depois do recarregamento; e `CarregarConfig` recusa duração não-positiva, então o TTL zero só chega por `Config` montada à mão. Os dois consertos acrescentam ramo e estado | rejeitado |

## Verification

**Commands:**
- `go build ./...` -- expected: compila.
- `go test ./...` -- expected: tudo verde, incluindo fronteira, `.env` e os subtestes novos de `api/`.
- `npm --prefix web run build` -- expected: o `prebuild` (guarda offline + `node --test`) passa e o `next build` compila `/entrar` com o `destino`.

**Manual checks (if no CLI):**
- `docker compose up`; errar a senha cinco vezes e ver o 429 com a mensagem de 15 minutos; entrar, clicar em "Sair" e confirmar que a Vitrine volta a oferecer "Entrar"; abrir `/pedidos/<id>` sem Sessão e confirmar a ida a `/entrar?destino=/pedidos/<id>` e o retorno depois do login.
