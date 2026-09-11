# Trabalho adiado

Achados reais que a revisão levantou e que não pertencem à estória que os expôs.
Append-only: não edite nem remova entradas existentes.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: Não existe CI, e o AD-12 exige um passo que faça grep por `fonts.googleapis`, `@import url(http`, `//cdn.` e `next/font/google` e falhe o build.
  evidence: Três invariantes de arquitetura (fronteira do AD-1, domínio sem HTTP, config que falha alto) passaram a existir só como testes que alguém precisa lembrar de rodar. O NFR-15 depende do grep, e nada o executa.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: O `.env` versionado repete 19 das 23 variáveis com o mesmo valor do padrão de produção do `config.go`, virando segunda fonte de verdade.
  evidence: O cabeçalho do próprio arquivo diz "aqui só o que a demonstração encolhe", e só cinco variáveis divergem de fato. É o mesmo defeito que a decisão 4 do addendum §10 usa para justificar não declarar `default_transaction_isolation` no compose.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: `AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS=89` é ambígua entre "aprova até R$ 0,89" e "aprova até o resto 89 dos centavos do total".
  evidence: O §7.1 do PRD define a regra pelos dois últimos dígitos do total, mas nem o `.env`, nem o `config.go`, nem o addendum dizem qual leitura o nome carrega. A estória 1.7 vai implementar a que adivinhar.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: O `web/` não tem lint, teste, `next.config.js` nem `output: "standalone"`, e a imagem copia `node_modules` inteiro.
  evidence: O README chama `go test ./...` de suíte completa, o que é falso para o repositório. A estória 1.2 traz fonte auto-hospedada, tokens e shadcn para dentro dessa casca — é o momento mais barato de pôr o guarda-corpo.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: O goose arrasta `modernc.org/sqlite`, `libc`, `memory` e `bigfft` para o `go.sum`, e configura dialeto, logger e FS por variáveis globais de pacote.
  evidence: É uma árvore grande para aplicar `.sql` em ordem contra um único Postgres, e as globais anulam a testabilidade que a parametrização do `fs.FS` conquistou. O goose expõe um `Provider` com `WithFS`; as build tags de dialeto cortam a árvore.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: `erro.Escrever` loga `err.Error()` de todo erro não registrado, e esse texto pode conter e-mail, Endereço ou nome.
  evidence: O AD-15 proíbe dado pessoal em log. Logar só o tipo do erro preserva a regra mas perde o diagnóstico — a tensão é real e recorre em toda estória que criar sentinela, então merece uma decisão registrada e não uma escolha por handler.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: `api/` não tem middleware de recuperação de pânico nem `http.MaxBytesReader`, apesar de o doc do pacote citar "os limites do NFR-14".
  evidence: Um pânico em qualquer handler futuro derruba a conexão, imprime stack trace não estruturado em stderr (furando o AD-15) e escapa do envelope do AD-14. Os limites de corpo do NFR-14 só ganham consumidor quando existir o primeiro DTO.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-2-casca-do-navegador.md`
  summary: Nenhum teste exercita o caminho `:3000 → rewrites() → :8080` de ponta a ponta, então a linha do `Dockerfile` que copia o `next.config.ts` para o estágio de execução pode ser removida sem que nada fique vermelho.
  evidence: A camada de revisão de lacunas reverteu essa linha, construiu a imagem e rodou o contêiner — `/` devolveu 200 e `/api` voltou ao 404 do Next, com `npm test` e a suíte Go verdes. Fechar exige um contêiner de fumaça, e o repositório não tem CI nem arreio de teste de integração onde pendurá-lo.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-2-casca-do-navegador.md`
  summary: O `Badge` do shadcn usa texto de 12px, e a regra de acessibilidade do `DESIGN.md` manda nada abaixo de 14px — as duas regras vêm da mesma espinha de UX, que também proíbe editar o componente.
  evidence: Confirmado em `web/components/ui/badge.tsx` (`text-xs`). O `DESIGN.md` diz ao mesmo tempo "usados do shadcn sem alteração: … Badge" e "todo texto de conteúdo … em no mínimo 14px", e o selo de Status do Pedido é um `Badge` que carrega texto de conteúdo. A tensão é da espinha, não desta estória; quem a resolve é o dono da UX, decidindo se rótulo de selo conta como texto de conteúdo.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-2-casca-do-navegador.md`
  summary: A imagem de execução do `web` carrega 505 MB de `node_modules` porque o estágio de execução copia tudo sem podar.
  evidence: Medido pela camada de revisão de lacunas dentro da imagem construída. Reclassificar as ferramentas para `devDependencies` já foi feito, mas `npm prune --omit=dev` ainda não é seguro: o `next start` lê `next.config.ts` em tempo de execução e precisa do TypeScript instalado. Resolver exige ou converter a configuração para `.js` na imagem, ou adotar `output: "standalone"`.
