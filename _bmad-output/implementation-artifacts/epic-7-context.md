# Epic 7 Context: Entrega — documentação viva e ensaio

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

No dia da entrega, a banca precisa encontrar três entregáveis de documentação que **não mentem**: um README que leva alguém de fora do clone ao sistema rodando, o diagrama dos seis módulos batendo com a decomposição real do código, e o addendum sem nenhuma decisão contradita pelo que foi construído. Além disso, os três roteiros de demonstração (Roteiro A, Roteiro B e Roteiro C) precisam estar ensaiados a partir de clone limpo, por ao menos duas pessoas diferentes. Esta épica é o fecho, não o começo: ela **verifica** o que as Épicas 1 a 6 produziram, e não escreve documentação do zero. O addendum já é mantido vivo desde a Épica 1, e cada decisão voltou para ele na estória que a tomou. O que falta aqui é prova, não comportamento novo.

## Stories

- Story 7.1: O README leva do clone ao sistema rodando
- Story 7.2: O diagrama dos seis módulos corresponde ao código
- Story 7.3: O addendum percorrido contra o código
- Story 7.4: Os três roteiros ensaiados a partir de clone limpo

## Requirements & Constraints

- **Nenhum FR novo.** O entregável são os artefatos de documentação e o ensaio. Nenhuma funcionalidade nova entra enquanto os três roteiros não passarem (contra-métrica SM-C1): mais telas com o ciclo quebrado valem menos que o ciclo fechado.
- **Ambiente em um comando (NFR-1, SM-3).** Alguém que não escreveu o código, seguindo **apenas** o README, chega ao sistema funcionando com Catálogo Semeado em ≤ 15 min. O Catálogo Semeado é determinístico e plausível. A verificação é **recorrente**: a cada semana, um integrante diferente sobe o ambiente do zero. O README precisa documentar `docker compose up`, o modo demonstração como padrão, `docker compose down -v` como botão de reinício e onde ler no log o token de redefinição de senha.
- **Documentação que não mente (SM-7, SM-C4).** Volume de páginas não conta. Documentação que descreve um sistema diferente do construído é pior que documentação nenhuma. O que conta é o README funcionar e o diagrama bater com o código.
- **Addendum contra o código.** Percorra cada decisão e verifique que nenhuma é contradita. Ele precisa conter o desfecho das cinco verificações do passo 0 e toda decisão de arquitetura tomada na construção, incluindo o número medido do NFR-4. Uma contradição se resolve mudando o código **ou** registrando a mudança de decisão **com o porquê**. Nunca se apaga a decisão antiga.
- **Os três roteiros (SM-1).** Executam de ponta a ponta em ambiente limpo, **sem intervenção manual no banco e sem erro visível**, com alvo de 3 de 3. Nenhum passo entra na apresentação sem ter passado em ensaio. O passo 5 do Roteiro C (dois checkouts simultâneos na última unidade) só é demonstrado se o teste de concorrência do NFR-7 estiver passando.
- **Operação offline (NFR-15).** Percorra o Roteiro A inteiro com a rede desconectada, imagens dos Produtos incluídas. Nada de fonte, CSS, script ou imagem buscados em rede externa.
- **Perguntas prováveis da banca.** Cada pergunta tem um responsável nomeado antes da apresentação, e ninguém responde por parte que não construiu sem antes ler a fonte. As perguntas são: pagamento assíncrono sendo simulado, marketplace ou loja, a última unidade disputada, por que não microsserviços, pagamento que nunca confirma e Carrinho de visitante.
- **Nunca cortar**, mesmo sendo invisíveis em tela: FR-19, FR-24, FR-34 e o teste do NFR-7.

## Technical Decisions

- **O diagrama do AD-1 é o diagrama exigido.** Não existe um segundo desenho a manter. `DIAGRAMA-MODULOS.md` (ao lado da espinha de arquitetura) é a versão que se lê sozinha. Ele traz os seis módulos canônicos (identidade, catalogo, busca, carrinho, pedido, pagamento) mais `web`, `api`, o relógio e a porta do Provedor de Pagamento, com as setas exaustivas e rotuladas pelo que atravessa. A prova de correspondência é `internal/fronteira_test.go` passando, que lê o grafo de importação e falha em aresta fora da tabela. Ela tem um par no banco: uma consulta que falha se existir FK cruzando schema. Nenhum sétimo módulo se negocia.
- **Exceção já registrada:** `internal/plataforma` (config, log, correlação, transação, tradução de erro) é importável por todos e não entra na tabela do AD-1. O diagrama e o teste precisam concordar nisso.
- **Modo demonstração é o padrão (AD-13).** `docker compose up` sem argumento sobe os quatro serviços (`web`, `azamon`, `postgres`, `redis`), aplica as migrações e o Catálogo Semeado no arranque. O `.env` versionado traz os valores de demonstração, com credenciais que não são segredo. A semente roda uma vez por marcador de versão. Banco existente só vê semente nova depois de `down -v`.
- **O token de redefinição de senha vai para o log estruturado (AD-20)**, lido em `docker compose logs`. Não existe serviço de e-mail.
- **Nada de rede externa (AD-12).** A fonte é auto-hospedada. As imagens do Catálogo Semeado saem de `media/`, servidas pelo Go com URL relativa. Um portão no build faz grep por `fonts.googleapis`, `@import url(http`, `//cdn.` e `next/font/google`.
- **Provedor Simulado, determinístico pelos centavos do total**, só na primeira Tentativa de Pagamento: `,00`–`,89` aprova, `,90`–`,94` recusa, `,95`–`,99` nunca confirma e expira. Em demonstração, a expiração é de 60 s e cada etapa de entrega leva 30 s. Não existe superfície de operador: o apresentador escolhe Produto e quantidade para provocar cada desfecho, e o roteiro de ensaio precisa registrar essa escolha.
- **Registro de decisão.** Mudou o addendum ou a espinha? Registre no `.memlog.md` do mesmo diretório com `uv run _bmad/scripts/memlog.py append`. Os `AD-n` têm ID estável: emende no lugar, nunca renumere. `SPEC.md` e `mapa-de-capacidades.md` não se editam à mão.
- **Armadilhas de ensaio já conhecidas:**
  - `docker compose up` não reconstrói as imagens. Antes de ensaiar, rode `up --build` e confira `docker compose images`.
  - Um compose "offline" com rede `internal: true` descarta em silêncio a publicação das portas 3000 e 8080. O ensaio offline é desligar a rede da máquina à mão.
  - Reconstruir só o `web` sem `--no-deps` recria o `azamon` e descarta overrides.

## UX & Interaction Patterns

- A Vitrine com Catálogo Semeado é a primeira tela que a banca vê. Imagem ausente vira bloco neutro com o nome do Produto, nunca o ícone de imagem quebrada.
- As telas se atualizam sozinhas durante a apresentação: Pedido em processamento a cada 3 s, Detalhe do Pedido e Tabela de Pedidos do Administrador a cada 10 s enquanto houver Pedido não terminal. É isso que deixa a simulação de entrega do Roteiro A avançar até `ENTREGUE` enquanto o Roteiro C roda.
- A Tabela de Pedidos não tem busca por número. O ensaio do Roteiro C acha a linha por filtro de Status do Pedido e ordenação por data.
- Erro do servidor aparece como `Toast` com o identificador de correlação, que é o que torna rastreável uma falha ao vivo.

## Cross-Story Dependencies

- A Épica 7 depende das Épicas 1 a 6 fechadas (estão). A 7.1 remediu o Roteiro A de clone limpo em 4 min 28 s com cache frio, ainda cronometrado pelo time. **A metade "pessoa de fora" da SM-3 continua sem prova**: o Registro de subidas do README tem a linha sem dono.
- A 7.2 redesenhou o grafo do AD-1 a partir do código (23 arestas, um bloco mermaid `%% AD-1` idêntico na espinha e no `DIAGRAMA-MODULOS.md`), preso por `TestDiagramaEhATabela` e `TestTabelaEhOCodigo` em `internal/fronteira_test.go`. Deixou para a 7.3 duas entradas em `deferred-work.md`: a porta do AD-8 sem `interface` Go, e a tabela do AD-6 com `Expirar` e `SimularEntrega` dentro de `pedido.Varrer`.
- Decidido em 2026-09-25, antes da 7.3: `internal/pedido/pedido.go` **fica como está** (B2 da retro da Épica 6, aceito), e a emenda do AD-4 com o contador do ano (`epic-5-retro-item-27`) **entra na 7.3**.
- A 7.2 e a 7.3 se apoiam no teste de fronteira e no histórico de decisões do addendum. As duas decisões abertas que tocavam a 7.3 (o AD-4 e a divisão de `pedido.go`) foram tomadas; ver acima.
- A 7.4 consome o resultado da 7.1: o README é o ponto de partida do clone limpo. Ela usa como insumo os roteiros de passeio das Épicas 5 e 6 e os desfechos que ninguém viu em tela depois das correções. O passo 5 do Roteiro C está condicionado ao teste do NFR-7 (estória 5.7).
- Uma pendência externa segue aberta: a rubrica do professor (bloqueio 2, dono Sung) é a 5ª verificação do passo 0. Ela não bloqueia a épica, mas pode mover suposições do PRD se chegar antes da entrega.
