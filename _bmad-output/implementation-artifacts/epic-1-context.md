# Epic 1 Context: Ciclo fechado em um comando

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Provar o sistema inteiro de ponta a ponta antes de qualquer funcionalidade ficar completa: `docker compose up` a partir de um clone limpo levanta os quatro contêineres e **um** Pedido atravessa do clique ao `ENTREGUE` — um Produto semeado, um Comprador, telas cruas, sem busca, sem filtro, sem Carrinho e sem painel administrativo. A épica existe para forçar a integração front↔back cedo, quando ainda há tempo de descobrir que o valor monetário virou ponto flutuante, que o cookie de Sessão não atravessa a casca do Next, ou que a busca não cabe no orçamento de tempo. **Nenhum FR é fechado aqui:** nove são atravessados no mínimo viável (cadastro/sessão, página de Produto, adicionar, criação de Pedido, tentativa e confirmação de pagamento, máquina de estados, simulação de entrega) e continuam propriedade das épicas 2 a 6. Cada estória declara explicitamente o que deixa para depois — respeitar esse recorte é parte do trabalho.

## Stories

- Estória 1.1: Andaime do serviço — os quatro contêineres em um comando
- Estória 1.2: Casca do navegador — rewrites, shadcn e a base visual da marca
- Estória 1.3: Primeira migração e o Catálogo Semeado determinístico
- Estória 1.4: A medição de desempenho da busca com 5.000 Produtos
- Estória 1.5: Um Comprador entra e vê um Produto
- Estória 1.6: Um Pedido nasce em `AGUARDANDO_PAGAMENTO`
- Estória 1.7: O Provedor Simulado confirma por webhook e o Pedido vai a `PAGO`
- Estória 1.8: A varredura leva o Pedido até `ENTREGUE`
- Estória 1.9: Clone limpo, rede desconectada, README de 15 minutos

## Requirements & Constraints

- **Ambiente em um comando.** `docker compose up` sem argumento sobe `web`, `azamon`, `postgres` e `redis` em **modo demonstração**, a partir de repositório limpo. Alguém que nunca rodou o projeto chega ao sistema funcionando em ≤ 15 min seguindo só o README.
- **Semente determinística e plausível.** Aplicada no arranque, depois das migrações, só se o marcador de versão não existir; subir duas vezes produz exatamente os mesmos dados. Nomes, preços e imagens críveis — a semente é a primeira coisa que a banca vê. Não cria Pedido. `docker compose down -v` volta ao estado inicial.
- **Operação offline.** Nada buscado em rede externa em tempo de execução: fonte auto-hospedada, imagens locais servidas pelo Go por URL relativa. Um passo de CI faz `grep` por `fonts.googleapis`, `@import url(http`, `//cdn.` e `next/font/google` e falha o build.
- **Precisão monetária.** Centavos em `int64` ponta a ponta, sem divisão no caminho monetário; o total é coluna com `CHECK`, nunca derivação de leitura; formatação `R$` só no navegador.
- **Rastreabilidade.** Identificador de correlação em toda requisição e toda linha de log; toda transição de Status registrada com estado anterior, novo, autor e instante. Nenhum dado pessoal em log.
- **Configuração, não constantes.** Todos os limiares vivem em variáveis `AZAMON_`, lidas uma vez no arranque para uma struct. Em demonstração: expiração da Tentativa 60 s, intervalo da simulação 30 s. Somam-se quatro operacionais: intervalo da varredura, atraso da confirmação simulada, interruptor da simulação e URL base do webhook.
- **Cinco verificações de risco fecham nesta épica** e o desfecho de cada uma volta ao addendum: (1) `Set-Cookie` do Go atravessando o `rewrites()` do Next — a mais cara se descoberta tarde, com saída declarada de *proxy* explícito; (2) sqlc analisando `DEFAULT uuidv7()`, com saída de gerar o `uuid` no lado Go; (3) `npx shadcn init` limpo em Next 16 + React 19; (4) p95 ≤ 500 ms na busca com 5.000 Produtos, com resposta declarada "índice antes de serviço" — nunca cache, nunca Elasticsearch; (5) o enunciado do professor, fora do controle do time, não bloqueia a épica.

## Technical Decisions

- **Sete pacotes sob `internal/`:** identidade, catalogo, busca, carrinho, pedido, pagamento e plataforma. Um módulo só alcança outro por `internal/<módulo>/<módulo>.go`; dentro de cada um, a forma é fixa (interface pública, serviço, domínio, `db/consultas.sql`, `db/gerado/`). Um teste de fronteira roda `go list -deps -json` e é exaustivo, não amostral. `api/` é só tradução — rotas, middleware, DTO e os limites de entrada; nenhum módulo de domínio conhece HTTP.
- **Um schema Postgres por módulo, chave estrangeira cruzando schema é proibida** (verificável por `information_schema`). Chaves `uuid DEFAULT uuidv7()`, datas `timestamptz` UTC, `snake_case` singular, monetário `bigint` com sufixo `_centavos`, identificadores de domínio literalmente nos termos do glossário.
- **Migrações por goose, embutidas via `embed.FS`, aplicadas no arranque do binário** — nunca auto-migração; falha de migração derruba o arranque em vez de servir com schema errado. A primeira migração de `catalogo` é `CREATE EXTENSION pg_trgm`. Só quatro estórias desta épica criam schema: 1.3 (comprador, administrador, vendedor, categoria, produto), 1.4 (coluna normalizada + índice GIN), 1.6 (pedido, item_pedido, transicao_status, reserva_estoque já com estado e índice único parcial) e 1.7 (tentativa_pagamento, confirmacao_recebida com restrição única sobre a chave de idempotência). Criar tabela fora dessas é defeito de sequenciamento.
- **A transição é o único ponto de mutação do Pedido.** Não existe `UPDATE pedido SET status`: só um `Transicionar` em compare-and-swap, onde zero linhas afetadas é "estado já avançou". Transição primeiro, efeito sobre Estoque depois, na mesma transação. O Pedido tem um `numero` legível `AZ-<ano>-<6 dígitos>` separado do `uuid`.
- **Uma transação por caso de uso, passada como parâmetro nomeado** (`tx pgx.Tx`), nunca no contexto; nenhuma chamada de rede dentro de transação aberta. Reserva de Estoque: `SELECT … ORDER BY id FOR UPDATE` primeiro, soma das reservas ativas depois — dois comandos, nesta ordem.
- **A varredura é o único motor do tempo:** nenhum `time.Timer` em módulo nenhum. Um tique de 1 s em `cmd/azamon` chama, nesta ordem, aplicar → expirar → simular → emitir; uma transação por Pedido com `FOR UPDATE SKIP LOCKED`; "estado já avançado" é desfecho esperado, não falha. Todo avanço deriva do histórico de transições, então reiniciar o contêiner retoma de onde parou.
- **Pagamento é assíncrono desde o primeiro dia.** A resposta do checkout não espera o resultado. A porta tem duas operações e mais nada; o Provedor Simulado decide pelos centavos do total, só na primeira Tentativa, sem superfície de operador. A confirmação entra por webhook, é gravada com restrição única sobre a chave de idempotência, e `pagamento` não conhece `pedido` — a varredura lê o inbox e aplica, só se pertence à Tentativa corrente e o Pedido está aguardando pagamento. A emissão da confirmação simulada mora em `pagamento`, disparada depois do commit. Nenhum dado de cartão em ponto nenhum.
- **Erros sentinela por módulo, traduzidos num arquivo só**, no envelope `{"erro":{"codigo","mensagem","dados","correlacao"}}` — `dados` carrega o dado que a UX precisa. Erro engolido é defeito.
- **Next.js é casca:** `rewrites()` de `/api/*` para o Go, uma origem só, sem CORS. Proibidos Server Actions, Route Handlers com regra e acesso a banco pelo Node. Duas armadilhas do Next 16: `middleware.ts` virou `proxy.ts`; `next/image` recusa IP privado, então `unoptimized`.
- **Sessão no Redis com TTL de 7 dias**, cookie `azamon_sessao` `HttpOnly` `SameSite=Lax`, opaco de 256 bits, sem conteúdo. Senha em Argon2id (`m=19456`, `t=2`, `p=1`, sal de 16 bytes).
- **Rotas** `/api/v1/<recurso-plural>`; o detalhe de Produto é servido por `catalogo`.
- **Stack fixada:** Go 1.27.1 · PostgreSQL 18.6 · Redis 8.10.1 · Node 24.20.0 · Next.js 16.3.4 · React 19.2.8 · pgx v5.10.0 · sqlc v1.31.1 · goose v3.28.0 · testcontainers-go v0.44.0 · `net/http.ServeMux` · shadcn/ui por CLI. Commits em inglês, tudo que sai para o usuário em português.

## UX & Interaction Patterns

Esta épica constrói **só a base visual** que as demais herdam — as composições de marca nascem nas épicas que as usam; aqui as telas podem ser cruas.

- shadcn/ui sobre Tailwind instalado por CLI, componentes em `web/components/ui/` usados **sem alteração** (Card, Dialog, Sheet, Select, Table, Badge, Toast, Skeleton, Pagination, Input, Label, Separator, DropdownMenu, RadioGroup, Alert).
- **Exatamente dez tokens de cor** sobrescrevem o shadcn (`chrome`, `chrome-foreground`, `chrome-muted`, `primary`, `primary-foreground`, `primary-strong`, `primary-strong-foreground`, `link`, `available`, `available-foreground`); todo o resto herda, `destructive` inclusive. Sem modo escuro. Verde é só disponibilidade e `ENTREGUE`; laranja aparece uma vez por fluxo.
- Três papéis tipográficos de marca (`wordmark`, `preco`, `preco-centavos`), `tabular-nums` em todo valor monetário e quantidade, nenhum texto de conteúdo abaixo de 14px com 4,5:1 verificado. Raios 4/6/8px, com `rounded.full` reservado a quatro usos. Ritmo de espaço 48/32/24/16-32px, conteúdo limitado a 1440px.
- Fonte `Azamon Sans` como WOFF2 auto-hospedado em `web/public/fonts/`, com pilha de sistema como alternativa.
- A tela do Pedido em processamento atualiza por consulta em intervalo de 3 s enquanto aguarda pagamento — sem WebSocket. Todo instante exibido é absoluto, RFC 3339, vindo do servidor; o relógio nunca é contado no navegador.

## Cross-Story Dependencies

- **1.1 → todas.** O andaime (árvore, compose, config, correlação, envelope de erro, goose) precede tudo.
- **1.2 → 1.5.** O `rewrites()` precisa existir antes que o cookie de Sessão possa provar que atravessa; a verificação 1 abre na 1.2 e **fecha na prática na 1.5**.
- **1.3 → 1.4, 1.5, 1.6.** Sem schema e semente não há Comprador nem Produto para atravessar.
- **1.5 → 1.6 → 1.7 → 1.8.** Cadeia linear do fluxo: Sessão, depois Pedido, depois confirmação, depois avanço no tempo. A 1.6 vai da Página de Produto direto ao Pedido, **sem passar pelo Carrinho**.
- **1.9 fecha a épica** e depende de todas as anteriores: README, rede desconectada, teste de fronteira e addendum com o desfecho das cinco verificações.
- **1.4 é a raia paralela** — depende só da 1.3 e não bloqueia ninguém; sua migração de índice é independente do resto.
- **Para fora:** a Épica 2 fecha identidade, a 3 fecha catálogo/busca (incluindo Estoque, Reserva e o predicado de visibilidade, deixados de fora aqui), a 4 o Carrinho, a 5 a máquina de estados completa com as nove transições e as recusas nomeadas, a 6 o pós-venda. O `addendum.md` é mantido vivo a partir desta épica — decisão tomada durante a construção volta para ele **na mesma estória que a tomou**.
