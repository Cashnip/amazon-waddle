---
title: 'Estória 1.9 — Clone limpo, rede desconectada, README de 15 minutos'
type: 'chore'
created: '2026-09-13'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
baseline_commit: '5cc7ae1e1bab12575f10bf7bab2a166b95e03986'
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problema:** As oito estórias anteriores fecharam o ciclo na máquina de quem as escreveu. Ninguém provou que um clone limpo chega ao sistema rodando em ≤ 15 min seguindo **apenas** o README (NFR-1, SM-3), nem que o esqueleto inteiro anda com a rede desconectada (NFR-15) — e o README de hoje abre com a instalação do BMad, deixa `docker compose up` para a §5 e ainda diz, por escrito, que o README de 15 minutos "é da Estória 1.9".

**Approach:** Inverter o README (sistema primeiro, ferramenta de desenvolvimento depois), tornar o ensaio offline um comando reproduzível em vez de um cabo na mão, ensaiar de fato a partir de clone limpo com cronômetro, e fechar a épica no `addendum.md` §10 com o desfecho nomeado das cinco verificações do passo 0.

## Boundaries & Constraints

**Always:** Todo número publicado no README é **medido nesta estória**, nunca estimado. **A medição é a frio** (decidido em 2026-09-13): o cache de construção e as imagens base do Docker são apagados desta máquina antes do cronômetro, porque o que a SM-3 mede é o tempo de quem chega do zero, e um número medido sobre cache já quente seria propaganda. O custo — reconstruir o cache depois — é aceito. Docker é o único pré-requisito do caminho de 15 min — Go, Node, PostgreSQL e Redis ficam nos contêineres. Português em tudo que sai; commits em inglês.

**Never:** Nenhum comportamento novo — sem rota, tabela, coluna, índice, migração ou variável `AZAMON_` nova; criar schema a partir da 1.8 é defeito de sequenciamento. Não tocar em `internal/fronteira_test.go`: ele já é exaustivo e a condição de aceite fecha **rodando-o**. Não alargar `verificar-offline.mjs` para fora de `web/` — os quatro padrões já foram varridos no resto do repositório e não existem, e alargar traria os mockups e o `deck-banca.html` de `_bmad-output` para dentro do portão. Não acrescentar CI: o AD-12 já é `prebuild`, e o addendum §10 da 1.2 registra o porquê. Não ensaiar o Roteiro A completo nem os Roteiros B e C — busca, Carrinho e checkout em dois passos são das Épicas 3 a 5, e o ensaio dos três roteiros é a Estória 7.4. O esqueleto desta estória é: entrar → ver Produto → criar Pedido → `PAGO` → `ENTREGUE`.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Comportamento esperado | Tratamento de erro |
|---|---|---|---|
| Clone limpo cronometrado **a frio** | cache do Docker apagado (`docker builder prune -af` + remoção das imagens base e das do projeto), `git clone` em pasta nova, `docker compose up` seguindo só o README | quatro contêineres saudáveis, 50 Produtos semeados; tempo do clone ao primeiro Produto na tela anotado | passou de 15 min: o número vai para o README e o addendum **como medido**, e o excedente vira entrada em `deferred-work.md` |
| Esqueleto com a rede desconectada | ambiente no ar pelo override offline | entrar, ver Produto, criar Pedido, `PAGO`, `ENTREGUE`; aba de rede do navegador sem nenhuma requisição a domínio externo | requisição externa é defeito: nomear arquivo e linha, corrigir na estória |
| Portão do AD-12 exercitado | `fonts.googleapis` introduzido numa linha de `web/app/globals.css` | `npm run build` sai ≠ 0 nomeando arquivo, linha e padrão | a linha é removida logo após a prova; `git status` volta limpo |
| Fronteira de módulo | `go test ./internal/...` num clone limpo | passa, e o grafo verificado é exaustivo, não amostral | — |
| Segunda subida | `docker compose down -v` e subir de novo | mesmos 50 Produtos com os mesmos identificadores, nenhum Pedido | — |
| Override offline quebra as portas | `internal: true` impede publicar 3000/8080 | o arquivo **não entra**; o ensaio volta a ser desconectar a rede à mão | a tentativa e o motivo ficam registrados no addendum |

</frozen-after-approval>

## Code Map

- `README.md` (130 linhas) — hoje §1–§4 são BMad e skills, §5 é "Rodando o projeto". A §5 já tem credenciais, `down -v`, semente grande e armadilhas: é matéria-prima, não descarte. Apagar a frase "O README de 15 minutos que o NFR-1 exige é da Estória 1.9". A ordem inverte; a seção do BMad passa a dizer que serve só a quem vai desenvolver.
- `docker-compose.yml` — quatro serviços na rede padrão implícita; `azamon` com `env_file: .env`, `web` sem. O override acrescenta **só** a rede.
- `docker-compose.offline.yml` — **nasce**, se a matriz permitir: rede `internal: true` nos quatro serviços, subida por `docker compose -f docker-compose.yml -f docker-compose.offline.yml up`.
- `web/scripts/verificar-offline.mjs` — o portão do AD-12, ligado como `prebuild` em `web/package.json`. **Não muda**; é exercitado pela terceira linha da matriz.
- `internal/fronteira_test.go` — tabela exaustiva de arestas (L19-34) mais `TestDominioNaoConheceHTTP`. **Não muda.**
- `.env`, `db/semente/001_catalogo_semeado.sql`, `media/*.svg` (52 arquivos), `web/public/fonts/azamon-sans.woff2` — todos versionados; é o que faz o clone limpo subir sem buscar nada além das imagens base.
- `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` §10 (L158+) — oito entradas, uma por estória. Nasce a de 1.9, com as cinco verificações consolidadas: 1 e 3 fechadas na 1.2 (L174-175), 2 na 1.3 (L186), 4 na 1.4 (L201), e a 5 declarada **externa ao time** — bloqueio 2 do `HANDOFF.md`, dono Sung, sem toque no esqueleto vertical.
- `.../.memlog.md` — o par de sempre do addendum.
- `HANDOFF.md` — o topo (L3) e o "Próximo passo" (L58-72) ainda apontam para a 1.9; passam a declarar a Épica 1 fechada e a apontar para a Épica 2.

## Tasks & Acceptance

**Execution:**
- [x] Ensaio do clone limpo **a frio** -- `docker builder prune -af` (14,1 GB) e remoção das imagens base e das duas do projeto; clone em pasta nova; `docker compose up` seguindo só o README. **206 s** do clone ao primeiro Produto na tela, contra um teto de 900 s. Segunda subida: 16 s.
- [x] `docker-compose.offline.yml` -- escrito, provado e **não entrou**: com `internal: true` o Docker descarta a publicação de 3000 e 8080 em silêncio (`docker inspect` devolve `{"3000/tcp":[]}`).
- [x] Ensaio offline -- feito por dentro da rede interna, com um contêiner de `curl`: `example.com` não resolve e o esqueleto inteiro anda, `AGUARDANDO_PAGAMENTO` → `ENTREGUE` em 111 s, com SVG do Produto e fonte servidos de dentro.
- [x] Prova do portão do AD-12 -- `@import url(https://fonts.googleapis.com/...)` no topo de `web/app/globals.css` derrubou `npm run build` com saída 1, nomeando arquivo, linha e os dois padrões; linha removida e `git status` limpo.
- [x] `README.md` -- invertido: §1 a §4 do clone ao sistema (só Docker), §5 em diante a ferramenta de desenvolvimento. Número medido, passeio do esqueleto e o porquê de não haver comando de ensaio offline.
- [x] `addendum.md` + `.memlog.md` -- a entrada da 1.9, com as cinco verificações, o tempo medido e as duas coisas que o ensaio derrubou.
- [x] `HANDOFF.md` -- Épica 1 fechada, próximo passo na Estória 2.1.

**Acceptance Criteria:**
- Dado um clone limpo e apenas o README, quando alguém que não escreveu o código o segue, então chega ao sistema funcionando, e o tempo **medido** está publicado no README e no addendum.
- Dado o `addendum.md` §10, quando procuro as cinco verificações do passo 0, então cada uma tem desfecho nomeado, a quinta inclusive, como externa ao time.
- Dado `go test ./...` e `cd web && npm test && npm run build` num clone limpo, quando rodam, então passam sem nenhuma mudança em `internal/fronteira_test.go` nem em `web/scripts/verificar-offline.mjs`.
- Dado o diff inteiro desta estória, quando procuro migração, rota, coluna, índice ou variável `AZAMON_` nova, então não há nenhuma.

## Implementation Notes

**O ensaio derrubou duas coisas, e nenhuma delas era código.**

A primeira foi o próprio `docker-compose.offline.yml`. A matriz previa o desfecho e ele aconteceu: numa rede `internal: true` o Docker **não publica porta nenhuma** — `docker inspect` devolve `{"3000/tcp":[]}` e `{"8080/tcp":[]}`, `docker compose ps` mostra `3000/tcp` sem mapeamento e `curl http://localhost:3000` não conecta, tudo sem um aviso sequer no `up`. O arquivo saiu, como a matriz mandava. O ensaio, porém, não foi perdido: com a pilha no ar sob o override e um contêiner de `curl` anexado à mesma rede, o esqueleto inteiro andou de dentro — é a prova mais forte que existia à mão, porque ali não há rota para fora nem para o `curl` nem para os quatro contêineres. Desligar a rede da máquina à mão continua sendo o ensaio de véspera de apresentação, e o README diz por quê.

A segunda foi o caminho que o README descrevia. O único link de Produto da página de conferência é o Fone de R$ 249,90, e `,90` é a faixa que o Provedor Simulado **recusa** — recusa que a 1.7 decidiu não emitir, porque transitar para `PAGAMENTO_RECUSADO` é da Épica 5. Resultado medido: 45 s depois da compra o Pedido segue em `AGUARDANDO_PAGAMENTO`, sem erro na tela. Quem seguisse o README sem saber concluiria que o sistema travou justamente no passo que a épica existe para provar. Corrigir em código era proibido nesta estória e seria errado de qualquer forma (o link é temporário e a Épica 3 o remove), então a correção é o README mandar trocar o identificador na URL por um Produto de centavos `,00` — e o achado ficou em `deferred-work.md`.

**O que não foi ensaiado, de propósito.** Só o esqueleto: entrar → ver Produto → criar Pedido → `PAGO` → `ENTREGUE`. Busca, Carrinho e checkout em dois passos são das Épicas 3 a 5, e o ensaio dos três roteiros é a Estória 7.4.

## Spec Change Log

## Review Triage Log

## Design Notes

**Por que um override de compose em vez de puxar o cabo:** o ensaio precisa ser repetível por quem não estava na sala, e desconectar a rede da máquina que hospeda a sessão não é opção para quem trabalha remoto. Uma rede `internal: true` nega exatamente o que o NFR-15 nega — saída para fora — e mantém os quatro contêineres conversando entre si. Se a publicação de portas não sobreviver, a decisão se inverte sem discussão: o arquivo sai e o addendum registra a tentativa.

**Por que o README inverte em vez de se dividir em dois arquivos:** o `HANDOFF.md` já aponta para o README como "o setup", e um terceiro arquivo faria três lugares onde a mesma instrução envelhece. Uma ordem, dois públicos: quem vai só ver o sistema para na primeira seção; quem vai desenvolver continua descendo.

## Verification

**Commands:**
- `docker compose -f docker-compose.yml -f docker-compose.offline.yml config` -- expected: os quatro serviços na rede interna, com 3000 e 8080 ainda publicadas.
- `docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v "${PWD}:/src" -v azamon-gomod:/go/pkg/mod -w /src --network host -e TESTCONTAINERS_RYUK_DISABLED=true golang:1.27.1 go test -count=1 ./...` -- expected: tudo verde, fronteira e schema intactos.
- `cd web && npm test && npm run build` -- expected: verificador offline e teste da casca passam, e a construção conclui.
- `git diff --stat` -- expected: só `README.md`, `HANDOFF.md`, `addendum.md`, `.memlog.md` e (talvez) `docker-compose.offline.yml`.

**Manual checks (if no CLI):**
- Clone em pasta nova, `docker compose up`, cronômetro: <http://localhost:3000> mostra a casca e um Produto; entrar com `comprador@azamon.test` / `azamon-comprador`, comprar, e o Pedido percorre `AGUARDANDO_PAGAMENTO` → `PAGO` → `EM_SEPARACAO` → `ENVIADO` → `ENTREGUE` sozinho.
- Mesmo passeio pelo override offline, com a aba de rede do navegador aberta: nenhuma requisição a domínio que não seja `localhost`.
