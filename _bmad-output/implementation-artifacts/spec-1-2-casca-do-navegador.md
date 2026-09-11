---
title: 'Estória 1.2 — Casca do navegador: rewrites, shadcn e a base visual da marca'
type: 'feature'
created: '2026-09-09'
status: 'done'
route: 'dispatch'
baseline_commit: '089e31ec8cbd204d41f659b927d709d4ebf57396'
review_loop_iteration: 0
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md'
  - '{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/DESIGN.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O `web/` de hoje é um `create-next-app` cru: duas páginas sem estilo, sem `next.config.js`, sem Tailwind, sem shadcn e sem fonte. Nenhuma tela pode ser construída porque não existe cor, tipografia nem espaçamento definidos, e as duas verificações mais caras do passo 0 — o `Set-Cookie` atravessando o `rewrites()` e o `shadcn init` limpo em Next 16 — continuam abertas.

**Approach:** Transformar o `web/` em casca: `rewrites()` de `/api/*` para o serviço `azamon` numa origem só, `npx shadcn init` com os 15 componentes da UX-DR1 intocados, os dez tokens de marca e os três papéis tipográficos do `DESIGN.md` sobre Tailwind v4, e a `Azamon Sans` como WOFF2 auto-hospedado com a pilha de sistema como alternativa. Um passo automático de verificação offline falha o build ao encontrar rede externa. O desfecho das verificações 1 e 3 volta ao `addendum.md` §10.

## Boundaries & Constraints

**Always:**
- Um caminho de dado só: navegador → `/api` → Go (AD-10). Origem única, sem CORS.
- Os 15 componentes ficam em `web/components/ui/` e são usados **sem alteração** (UX-DR1).
- **Exatamente dez** tokens de cor sobrescrevem o shadcn, nos hex do `DESIGN.md`. Todo o resto herda, `destructive` inclusive. Sem modo escuro.
- Raios 4/6/8px e `full`; ritmo 48/32/24/16/32px; conteúdo limitado a 1440px; faixas responsivas em pixels (360/640/1024/1440).
- Nenhuma requisição de rede em tempo de execução (AD-12, NFR-15). Versões fixadas: Next 16.3.4, React 19.2.8, Node 24.20.0.

**Decisões tomadas (humano, 2026-09-09):**
- **O `web/` é TypeScript.** Os dois arquivos `.js` da 1.1 passam a `.tsx`, com `tsconfig.json`. O registro do shadcn serve `.tsx` nativamente, e as 16 composições das épicas 3 a 6 ganham checagem de tipo no contrato com a API. `tsx: true` no `components.json`.
- **A verificação 1 é provada por sonda temporária.** `GET /api/v1/sonda-cookie` no `api/` emite um cookie descartável e existe só para o `curl` atravessar o `rewrites()`. A 1.5 a remove ao entregar a Sessão de verdade; a spec da 1.5 herda essa remoção como tarefa.

**Never:**
- Server Action, Route Handler com regra, acesso a banco ou Redis pelo Node, `fetch` de RSC para o Go.
- Arquivo chamado `middleware.ts` — o Next 16 o ignora sem erro de build.
- `next/font/google`, `fonts.googleapis`, `@import url(http`, `//cdn.`.
- As 16 composições da UX-DR8. Aqui entra só a base que elas herdam.
- Editar qualquer arquivo gerado em `web/components/ui/`.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Rewrite da API | `GET /api/v1/saude` no Next (`:3000`) | Encaminha ao `azamon:8080`, devolve `200` e o `X-Correlation-Id` do Go | N/A |
| Verificação 1 | `curl -i localhost:3000/api/v1/sonda-cookie` | O `Set-Cookie` da sonda chega íntegro ao cliente | Se não atravessar: *proxy* explícito em vez de *rewrite*, decidido e registrado agora |
| Rota do Next | `GET /` | Página do Next, servida pelo Node, sem tocar o Go | N/A |
| Guarda offline | Fonte remota ou CDN referenciados no `web/` | O build falha nomeando arquivo, linha e padrão | Saída diferente de zero |
| Guarda offline | Só fonte local e CSS próprio | O build segue | N/A |

</frozen-after-approval>

## Code Map

- `web/package.json` — só `next`/`react`/`react-dom` hoje; recebe Tailwind 4, shadcn e o gancho `prebuild`.
- `web/app/layout.js`, `web/app/page.js` — viram `.tsx`; o comentário do `layout.js` já aponta esta estória. `layout` passa a importar o CSS global e a fonte.
- `web/Dockerfile` — `npm ci` no estágio de construção; se a resolução de peer deps falhar em React 19, a flag entra aqui (ressalva M1 de `reviews/review-versoes.md`).
- `api/rotas.go`, `api/saude.go` — alvo do rewrite e possível casa da sonda de cookie.
- `docker-compose.yml` — o destino do rewrite é o nome de serviço `azamon`, não `localhost`.
- (ler) `DESIGN.md` — o frontmatter é a fonte dos dez tokens, dos três papéis e das escalas; `EXPERIENCE.md` §Responsive traz as quatro faixas em pixels.
- (ler) `addendum.md` §10 — imitar o formato da entrada da Estória 1.1.
- (não editar) `internal/`, `cmd/`, `db/` — nenhum módulo de domínio é tocado aqui.

## Tasks & Acceptance

**Execution:**
- [x] `web/tsconfig.json` + `app/layout.tsx`, `app/page.tsx` — migrar os dois arquivos da 1.1 para TypeScript antes do `shadcn init` — o CLI detecta a linguagem no arranque e grava `tsx` no `components.json`.
- [x] `web/next.config.ts` — `rewrites()` de `/api/:caminho*` para o `azamon` e `images.unoptimized` — AD-10; o `unoptimized` evita a recusa silenciosa de IP privado antes que exista imagem.
- [x] `web/` — rodar `npx shadcn init` e adicionar os 15 componentes da UX-DR1 — é a verificação 3; instala Tailwind 4 e escreve `components.json`.
- [x] `web/app/globals.css` — os dez tokens, os três papéis tipográficos, os raios, o ritmo e `tabular-nums` — camada de marca do `DESIGN.md`, em `@theme` do Tailwind 4; nada além dos dez.
- [x] `web/public/fonts/` + `@font-face` — `Azamon Sans` em WOFF2 com a pilha de sistema declarada como alternativa — AD-12.
- [x] `web/scripts/verificar-offline.mjs` + `prebuild` no `package.json` — grep dos quatro padrões, saída diferente de zero — AD-12 exige que **falhe o build**, e o build é o único portão que já existe.
- [x] `api/sonda_cookie.go` + registro em `api/rotas.go` — `GET /api/v1/sonda-cookie` emitindo um cookie descartável, com comentário nomeando a 1.5 como quem o remove — é o alvo do `curl` da verificação 1.
- [x] `web/app/page.tsx` — uma página de conferência que exibe os dez tokens, os três papéis e um componente do shadcn — torna a base inspecionável sem inventar composição da UX-DR8.
- [x] `.../addendum.md` §10 + `.memlog.md` — desfecho das verificações 1 e 3 e as decisões desta estória — SM-7 exige o addendum vivo, na mesma estória que decidiu.
- [x] `web/scripts/casca.test.mjs` e `api/sonda_cookie_test.go` — cobrir as linhas da matriz de I/O — o rewrite e a guarda offline pelo `node --test`, o `Set-Cookie` da sonda pelo `go test`.

**Acceptance Criteria:**
- Dado o compose no ar, quando peço `/api/v1/saude` na porta 3000, então a resposta vem do Go pela origem única, sem cabeçalho de CORS em ponto nenhum.
- Dado o `web/` pronto, quando listo `web/components/ui/`, então estão lá os 15 componentes nomeados na UX-DR1, e `git diff` mostra que nenhum foi editado depois de gerado.
- Dado o CSS global, quando conto as sobrescritas de token do shadcn, então são exatamente dez, e não existe bloco de modo escuro.
- Dado um `grep -rn` por `middleware.ts`, `next/font/google`, `fonts.googleapis`, `@import url(http` e `//cdn.` em `web/`, então não há nenhuma ocorrência fora do próprio verificador.
- Dado `npm run build` no `web/`, quando um `@import url(https://…)` é inserido de propósito no CSS, então o build falha nomeando o arquivo e o padrão, e volta a passar quando removido.
- Dado texto de conteúdo em tela, quando meço tamanho e contraste, então nada abaixo de 14px e nada abaixo de 4,5:1.

## Implementation Notes

**As duas verificações de risco fecharam positivas.** O `Set-Cookie` atravessa o `rewrites()` íntegro (`azamon_sonda=atravessou; Path=/; Max-Age=60; HttpOnly; SameSite=Lax`, pela porta 3000, junto do `X-Correlation-Id` do Go), e o `shadcn init` rodou limpo em Next 16.3.4 + React 19.2.8 **sem flag de peer deps** — a ressalva M1 de `review-versoes.md` não se materializou com o npm 11 do Node 24.20.0. Os desfechos estão no `addendum.md` §10.

**Três desvios do plano, todos registrados no addendum:**

- O CLI `shadcn` 4.21 **não instala mais o Tailwind** — exige o Tailwind 4 já presente e aborta em "No Tailwind CSS configuration found". A ordem virou: `npm i tailwindcss @tailwindcss/postcss`, `postcss.config.mjs`, `@import "tailwindcss"`, e só então `shadcn init -b radix -p nova`.
- **`Toast` saiu do registro radix; o componente é `sonner`.** `components/ui/sonner.tsx` é o Toast do Azamon, e a lista de quinze da UX-DR1 fica intacta em papel. `button.tsx` entrou como décimo sexto arquivo porque `pagination` depende dele. Nenhum arquivo gerado foi editado.
- O `init` injetou `next/font/google` (Geist) no `layout.tsx` e um bloco `.dark` no `globals.css`. Os dois saíram: fonte remota viola o AD-12, e o `DESIGN.md` não tem modo escuro. O `@custom-variant dark (&:is(.dark *))` **ficou** — sem nenhum `.dark` no documento, ele torna todo `dark:` dos componentes gerados letra morta, o que é mais seguro que removê-lo e deixar o Tailwind cair no `prefers-color-scheme` do sistema.

**Uma correção fora da lista de tarefas:** o `web/Dockerfile` não copiava o `next.config.ts` para o estágio de execução. O `rewrites()` é resolvido pelo `next start` em tempo de execução, não assado no `.next`, então a imagem subia, servia `/` e devolvia 404 do Next em `/api`.

**`Azamon Sans` é o Inter variável latino renomeado** (OFL, 48 KB, eixo de peso 100–900 num arquivo só), o que fecha o `[ASSUMPTION]` do `DESIGN.md` sobre o arquivo concreto. Os três papéis tipográficos são `@utility` do Tailwind 4, não tokens `--text-*`, porque só assim `tabular-nums` nasce embutido nos dois papéis monetários.

## Spec Change Log

## Review Triage Log

| Veredito | Achado | Evidência |
|---|---|---|
| high | `format("woff2-variations")` faz o navegador descartar o único `src` e a fonte cai calada na pilha de sistema | Token removido do CSS Fonts 4; UA que não o reconhece pula a entrada. É a falha silenciosa que o AD-12 existe para evitar. → patch |
| medium | Cartões nascem com raio de 14px: `--radius-xl` deriva de `--radius: 0.625rem`, e a marca só fixa `sm/md/lg` | Confirmado: `card.tsx` usa `rounded-xl`. O `DESIGN.md` fixa `cartao-produto: {rounded.lg}` = 8px, e a fronteira congelada diz "raios 4/6/8px e full". → patch |
| medium | `sonner.tsx` resolve `prefers-color-scheme` sozinho, e o addendum afirma o contrário | `useTheme()` sem provedor devolve `theme` indefinido, o padrão `"system"` assume e o Sonner decide por conta. Nenhuma tela o monta ainda, então o dano é futuro. → patch (corrigir o addendum e declarar `color-scheme`) |
| medium | `??` deixa passar `AZAMON_API_URL` vazia, virando auto-rewrite, e barra final duplicada | `??` só cai no padrão com nulo ou indefinido; string vazia passa e o destino vira `/api/:caminho*`. → patch |
| medium | A guarda offline cobre um padrão de quatro, casa maiúsculas errado e não vigia `middleware.ts` de volta | A camada de lacunas apagou dois padrões e os quatro testes continuaram verdes. → patch |
| medium | O teste do rewrite quebra quando `AZAMON_API_URL` está exportada, e o ramo de sobrescrita nunca é exercitado | Reproduzido: com a variável definida, um dos quatro testes falha. A asserção mede o padrão do `??`, não a leitura do ambiente. → patch |
| medium | `npm test` não roda em portão nenhum, então os testes da guarda são peso morto | `prebuild` chama só o verificador; não há CI. Teste que não roda conta como ausente. → patch |
| low | Sem `engines`, o `npm test` quebra em Node 22 com erro de sintaxe ilegível | `casca.test.mjs` importa `next.config.ts` e depende do *type stripping* nativo do Node 24. → patch |
| low | `MaxAge` é o atributo que torna o cookie da sonda descartável e nada o afere | Leitura de `api/sonda_cookie_test.go`: afere status, nome, `HttpOnly`, `SameSite` e `Path`. → patch |
| low | `page.tsx` não tem nenhum heading, regressão frente ao `<h1>` que substituiu | `CardTitle` do shadcn renderiza `div`. → patch |
| low | `AZAMON_API_URL` só existe no addendum, sem menção no `.env` | Confirmado: ausente do `.env` e do `docker-compose.yml`. → patch |
| low | Ferramenta de construção em `dependencies` leva o CLI inteiro para a imagem (505 MB medidos) | Verdadeiro, mas podar exige resolver que o `next start` lê `next.config.ts` e precisa do TypeScript em execução. Reclassificar o manifesto é a parte trivial. → patch parcial, resto adiado |
| medium (adiado) | Ninguém exercita `:3000 → rewrites() → :8080` de ponta a ponta | A camada de lacunas reverteu a linha do `Dockerfile` e toda a suíte seguiu verde. Fechar exige contêiner de fumaça, e não há CI. → defer |
| medium (adiado) | `Badge` do shadcn usa 12px, contra o critério de nada abaixo de 14px | Confirmado no `badge.tsx`. Tensão pré-existente do `DESIGN.md`, que manda usar o componente sem alteração **e** manter 14px — não foi esta estória que a criou. → defer |
| false | Pacotes novos com `^` violam "versões fixadas" | A fronteira congelada nomeia três pacotes — Next, React e Node — e os três estão exatos. O `package-lock.json` fixa o resto para o `npm ci`. |
| false | `cn@0.2.6` é substituição obscura de `clsx` + `tailwind-merge` | Publicado por `shadcn` a partir de `github.com/shadcn-ui/cn`, MIT, zero dependências. Mesma autoria do resto. |
| false | `web/lib/utils.ts` é código morto | Sustenta o alias `utils` declarado no `components.json`; um `shadcn add` futuro que emita `@/lib/utils` resolve por causa dele. |
| false | `AZAMON_API_URL` foge da convenção de nomes da 1.1 | A convenção é `AZAMON_<ÁREA>_<PARÂMETRO>`, e `AZAMON_REDIS_URL` já no `.env` tem a mesma forma. Área `API`, parâmetro `URL`. |
| false | `sprint-status.yaml` devia estar em `review` | É a sequência do próprio fluxo: a sincronização para `review` acontece na etapa de apresentação. |
| low (rejeitado) | Guarda devia pegar qualquer `https://`, não só os quatro literais | O AD-12 nomeia exatamente quatro padrões, e um regex amplo acusaria a URL do esquema no `components.json` e a licença OFL, derrubando o build sem defeito. |
| low (rejeitado) | Mudar `AZAMON_HTTP_ADDR` deixa o rewrite apontando para porta morta | Ninguém muda a porta no uso corrente, e o conserto exige fiar ambiente no compose — mais que correção direta. |
| low (rejeitado) | Raiz inválida no argumento da guarda solta rastro de pilha | Só alcançável por quem passa caminho errado à mão; o conserto acrescenta ramo. |
| low (rejeitado) | Diretório apontado por link simbólico é pulado calado | Improvável em `web/`, e o conserto acrescenta uma chamada de `stat` por entrada. |
| low (rejeitado) | A sonda de cookie fica registrada em toda construção | Não existe implantação — o addendum da 1.1 fixou demonstração e teste como os únicos ambientes —, o cookie é inerte e a 1.5 remove a rota. |
| low (rejeitado) | O array `TOKENS` de `page.tsx` redigita os dez hexes | Página de conferência temporária cujo propósito é exibir o hex como rótulo; ler estilo computado acrescentaria JavaScript de cliente a uma página estática. |

## Design Notes

**Tailwind 4 muda onde os tokens moram.** O Tailwind 4 não gera `tailwind.config.js`: os tokens vivem no CSS. (Planejamento supunha que o CLI do shadcn instalaria o Tailwind; a implementação achou o contrário — ver Implementation Notes.) A camada de marca fica assim, e nada mais é sobrescrito:

```css
:root {
  --primary: #FFD814;            /* sobrescreve o shadcn */
  --chrome: #131921;             /* token novo, não existe no shadcn */
}
@theme inline {
  --color-primary: var(--primary);
  --color-chrome: var(--chrome);
}
```

**Não existe arquivo de middleware nesta estória.** A armadilha do Next 16 é escrever `middleware.ts` e vê-lo ignorado calado; nada em 1.2 precisa de middleware, então tratá-la é não escrever o arquivo e deixar o registro no addendum. Quando alguma estória precisar, o nome é `proxy.ts`.

**O verificador offline é um script Node, não um passo de CI.** Não existe CI no repositório e o rodapé "Adiado" da espinha não pede um. `prebuild` no `package.json` derruba `npm run build` e a construção da imagem `web` — que é o que o AD-12 pede — sem infraestrutura nova, e roda igual no Windows e no Alpine.

## Verification

**Commands:**
- `cd web && npm test` — expected: quatro testes verdes (rewrite, guarda offline nas duas faces, `web/` como está).
- `go test ./...` — expected: tudo verde, incluindo o `Set-Cookie` da sonda e o teste de fronteira do AD-1.
- `cd web && npm run build` — expected: compila, e o verificador offline passa antes.
- `docker compose up -d --build && docker compose ps` — expected: os quatro serviços `running`/`healthy`.
- `curl -i localhost:3000/api/v1/saude` — expected: `200`, `{"status":"ok"}` e `X-Correlation-Id`, provando o rewrite.
- `curl -i localhost:3000/api/v1/sonda-cookie` — expected: `Set-Cookie` íntegro; é a verificação 1.
- `cd web && npx tsc --noEmit` — expected: sem erro de tipo.
- `git diff --stat web/components/ui/` depois de gerar — expected: vazio.

**Manual checks (if no CLI):**
- Inserir um `@import url(https://…)` no CSS: `npm run build` falha nomeando arquivo e padrão, e volta a passar quando removido.
- Abrir `localhost:3000` com a rede desconectada: fonte, cores e componentes renderizam sem requisição externa na aba de rede.
