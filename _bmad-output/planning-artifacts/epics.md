---
stepsCompleted: [1, 2, 3, 4]
inputDocuments:
  - _bmad-output/specs/spec-azamon/SPEC.md
  - _bmad-output/specs/spec-azamon/mapa-de-capacidades.md
  - _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md
  - _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md
  - _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md
  - _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/DIAGRAMA-MODULOS.md
  - _bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/DESIGN.md
  - _bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/EXPERIENCE.md
  - _bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/mockups/pagina-de-produto.html
  - _bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/mockups/resultados-busca.html
  - _bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/mockups/detalhe-do-pedido.html
  - _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/review-consistency.md
  - _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/review-rubric.md
  - _bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/review-adversarial.md
  - _bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/review-rubric.md
  - _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/reviews/review-rubric.md
  - _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/reviews/review-adversarial.md
  - _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/reviews/review-concorrencia.md
  - _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/reviews/review-versoes.md
  - _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/reviews/review-reconciliacao-prd.md
  - _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/reviews/review-reconciliacao-ux.md
  - _bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/reviews/review-reconciliacao-addendum.md
  - AGENTS.md
  - HANDOFF.md
---

# Azamon — Épicas e Estórias

## Visão geral

Este documento quebra em épicas e estórias implementáveis o que a `SPEC.md` fixa como contrato e o que
o PRD, a espinha de arquitetura e as duas espinhas de UX detalham. **Nada aqui substitui esses
documentos:** o inventário abaixo é índice por ID, não cópia. As consequências testáveis de cada FR —
que são a fonte das Condições de Aceite — vivem no §4 do PRD e continuam sendo leitura obrigatória.

**Precedência em conflito** (herdada da `SPEC.md`): o PRD decide **requisito**; a espinha de
arquitetura decide **mecanismo**; `DESIGN.md` e `EXPERIENCE.md` decidem **design e comportamento**; o
addendum guarda o **porquê**. Mock e material importado perdem para as espinhas.

## Inventário de Requisitos

### Requisitos Funcionais

*34 FRs, numeração global e ID estável, agrupados como no §4 do PRD. Cada linha é o enunciado; as
consequências testáveis estão no PRD e são o que as ACs verificam.*

**§4.1 Conta e Identidade — `internal/identidade`**

- **FR-1 — Cadastro de Comprador.** Visitante cria conta com e-mail e senha. E-mail normalizado antes da verificação de unicidade, senha ≥ 8 caracteres, nunca persistida em claro; após o cadastro já está autenticado.
- **FR-2 — Autenticação e Sessão.** Inicia e encerra Sessão. Mensagem genérica idêntica para e-mail inexistente e senha errada; expiração por inatividade (7 dias); encerrar invalida no servidor; bloqueio após 5 tentativas por 15 min, sem revelar se a conta existe.
- **FR-3 — Recuperação de senha.** Token de uso único válido por 30 min; resposta idêntica exista ou não a conta; redefinir encerra as Sessões ativas; nova solicitação invalida os tokens anteriores. *(1º na ordem de corte.)*
- **FR-4 — Papéis e autorização.** Distingue Comprador de Administrador e restringe cada recurso ao dono ou ao papel, inclusive contra chamada direta à API. Verificação no servidor.
- **FR-5 — Endereços.** Cadastrar, editar, remover e escolher. CEP validado; remover não altera Pedido existente; Comprador sem Endereço é levado ao cadastro e volta ao ponto em que parou.

**§4.2 Catálogo — `internal/catalogo`**

- **FR-6 — Vitrine.** Lista Produtos com nome, imagem, preço e disponibilidade. Estoque zero aparece indisponível e não é adicionável; Catálogo Semeado vazio mostra estado vazio explicativo.
- **FR-7 — Página de Produto.** Nome, descrição, imagem, preço, Categoria, **nome do Vendedor** e disponibilidade. Id inexistente devolve "não encontrado", não erro de servidor; visitante vai ao Login e volta.
- **FR-8 — Gestão de Vendedores.** Criar, editar e desativar. Vendedor desativado oculta seus Produtos sem apagar Pedidos; Vendedor com Produtos não é removível — a operação oferecida é desativar. *(3º na ordem de corte.)*
- **FR-9 — Gestão de Produtos.** Criar, editar e desativar, sempre com Vendedor e Categoria. Preço ≤ 0 e Estoque negativo recusados; Produto criado aparece na Vitrine e na busca sem reiniciar; editar preço não altera Item de Pedido existente.
- **FR-10 — Gestão de Categorias.** Criar e renomear. Categoria com Produtos não é removível; nome duplicado é recusado. *(2º na ordem de corte.)*
- **FR-11 — Estoque e disponibilidade.** Disponível = total − Reservas ativas. Disponível zero recusa Carrinho e checkout; ajuste do Administrador tem efeito imediato; reduzir o total abaixo das Reservas ativas é recusado informando **quantas unidades estão comprometidas**; o disponível nunca é negativo.

**§4.3 Busca e Navegação — `internal/busca`**

- **FR-12 — Busca por texto.** Termo livre sobre nome e descrição, sem caixa e sem acento ("cafe" encontra "café"). Sem resultado devolve estado vazio; termo vazio devolve a listagem completa paginada; Produto e Vendedor desativados não aparecem; `%` e `_` são texto literal; teto de 100 caracteres.
- **FR-13 — Filtros.** Categoria e faixa de preço, combináveis entre si e com o termo. Mínimo > máximo é recusado com mensagem; filtros visíveis e removíveis individualmente; estado na URL.
- **FR-14 — Ordenação.** Preço crescente, preço decrescente e mais recentes. Preservada ao paginar e ao filtrar; desempate estável — a paginação nunca repete nem pula um Produto. *(5º na ordem de corte.)*
- **FR-15 — Paginação.** 20 itens por página, teto de 60 aceito do cliente. Informa página atual e total de resultados; página além do total é vazio tratado, não erro.

**§4.4 Carrinho — `internal/carrinho`**

- **FR-16 — Carrinho exclusivo de Comprador autenticado.** Visitante que tenta adicionar vai ao Login e retorna à Página de Produto de origem. Sem Carrinho anônimo e sem fusão no login.
- **FR-17 — Adicionar Produto.** Produto já presente soma à quantidade do Item de Carrinho existente. Quantidade acima do disponível é recusada informando o disponível; Produto indisponível ou desativado não entra; 1 ≤ quantidade ≤ 10.
- **FR-18 — Alterar e remover.** Quantidade zero remove o Item. Total recalculado a cada alteração, refletindo os preços atuais do Catálogo; esvaziar o Carrinho em uma ação, com confirmação.
- **FR-19 — Revalidação do Carrinho.** Na abertura do Carrinho **e** na entrada do checkout: Produto que ficou indisponível **bloqueia o avanço**; preço mudado é exibido ao lado do anterior e exige confirmação; desativado é sinalizado igual; se o subtotal mudar, o Frete é recalculado — inclusive quando cruza o limiar de isenção. **Nunca cortar.**

**§4.5 Checkout — `internal/pedido` + `internal/pagamento`**

- **FR-20 — Seleção de Endereço.** Escolhe entre os seus ou cadastra um novo; sem Endereço o passo abre direto no formulário; trocar recalcula o Frete antes da confirmação.
- **FR-21 — Cálculo do Frete.** Faixa de CEP → região → valor fixo, com isenção acima de R$ 299,00. Mesmo CEP e mesmo valor produzem sempre o mesmo Frete; CEP fora de faixa cai na região padrão; Frete discriminado do subtotal.
- **FR-22 — Revisão do Pedido.** Cada item com preço unitário e subtotal, mais Endereço, Frete e total. Volta ao Carrinho sem perder o Endereço escolhido; Carrinho vazio não chega à Revisão; declara a forma de pagamento **sem coletar nenhum dado dela**.
- **FR-23 — Criação do Pedido.** Nasce em `AGUARDANDO_PAGAMENTO`. Congela preço praticado, Endereço, Frete e o Vendedor de cada Item; Carrinho esvaziado somente após a criação bem-sucedida; confirmar duas vezes cria **um** Pedido.
- **FR-24 — Reserva de Estoque.** Gerada para cada Item, **tudo ou nada**: se qualquer item faltar, nenhum Pedido é criado. O disponível cai imediatamente para os demais; duas criações simultâneas na última unidade → exatamente uma sucede. **Nunca cortar.**
- **FR-25 — Tentativa de Pagamento.** A resposta do checkout não espera o resultado — o Comprador vai para a tela do Pedido em processamento. Cada submissão gera uma Tentativa com identificador próprio; o Provedor Simulado decide pelos **centavos do total** (§7.1), só na primeira Tentativa; nenhum dado de cartão em ponto nenhum.
- **FR-26 — Confirmação aprovada.** Vai a `PAGO`, a Reserva permanece, a tela reflete sem recarregar. Mesma confirmação duas vezes produz um único efeito. Aprovação que chega para Pedido já `CANCELADO` **não** muda o Status: é registrada na Tentativa e sinalizada ao Administrador.
- **FR-27 — Pagamento recusado.** Vai a `PAGAMENTO_RECUSADO`, a Reserva é liberada e o Estoque volta. Motivo visível no detalhe; nova Tentativa a partir do Pedido cria nova Reserva; sem Estoque a nova tentativa é recusada e o Pedido permanece cancelável; teto de 3 Tentativas por Pedido.
- **FR-34 — Expiração da Tentativa de Pagamento.** Sem confirmação em 15 min (60 s em demonstração), a Tentativa expira e o Pedido transita para `PAGAMENTO_RECUSADO` com motivo "tempo esgotado". A Reserva é liberada na mesma transação; reutiliza o caminho da recusa, sem estado novo; deriva do histórico de transições, nunca de temporizador em memória. **Nunca cortar.**

**§4.6 Pedidos e Pós-venda — `internal/pedido`**

- **FR-28 — Máquina de estados do Pedido.** Exatamente um Status, só as transições declaradas. Transição não declarada é recusada com erro explícito, seja qual for o ator; toda transição registra data, hora, estado anterior, novo e quem provocou; histórico consultável e imutável; `SEPARANDO → ENVIADO` **consolida** a Reserva (o total baixa, a Reserva encerra); nenhuma outra transição altera o total.
- **FR-29 — Histórico de Pedidos.** Lista com número, data, total e Status, mais recentes primeiro; só os Pedidos do Comprador autenticado; estado vazio com caminho para a Vitrine.
- **FR-30 — Detalhe do Pedido.** Itens com **preço praticado**, Endereço congelado, Frete, total e linha do tempo das transições. Botão de cancelar aparece se e somente se o estado permite; quando não permite, o detalhe informa o motivo. Pedido de outro Comprador é negado.
- **FR-31 — Cancelamento pelo Comprador.** Permitido em `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO` e `SEPARANDO`. Exige confirmação explícita; libera a Reserva e as unidades voltam — verificável na Vitrine no mesmo instante. Cancelar em `ENVIADO`/`ENTREGUE` é recusado inclusive por API; cancelar um já `CANCELADO` não produz efeito nem erro.
- **FR-32 — Painel de Pedidos do Administrador.** Lista filtrável por Status e ordenável por data. Executa `PAGO → SEPARANDO`, `SEPARANDO → ENVIADO` e `ENVIADO → ENTREGUE` — e mais nenhuma. Transição inválida é recusada informando quais são permitidas a partir do estado atual; abre o detalhe de qualquer Pedido, com o Comprador e os Vendedores dos Itens.
- **FR-33 — Simulação de entrega.** Avança sozinho de `PAGO` até `ENTREGUE` pelas mesmas transições. Intervalo configurável (24 h padrão, 30 s em demonstração), desligável por configuração; nunca move `CANCELADO` nem `AGUARDANDO_PAGAMENTO`; o avanço deriva do histórico, e reiniciar o sistema retoma de onde parou. *(4º na ordem de corte.)*

### Requisitos Não Funcionais

*16 NFRs transversais. Cada um tem limiar ou critério verificável — sete deles cobram de toda
capacidade e não pertencem a nenhuma épica sozinha.*

- **NFR-1 — Ambiente em um comando.** Aplicação, banco, cache e Catálogo Semeado sobem com um comando em contêineres, a partir do repositório limpo. Integrante novo chega ao sistema funcionando em ≤ 15 min só com o README (SM-3). Semente determinística e versionada — subir duas vezes produz os mesmos dados — e **plausível**.
- **NFR-2 — Fronteiras de módulo explícitas.** Seis módulos de domínio — identidade, catálogo, busca, carrinho, pedido, pagamento — e um só alcança o outro pela interface pública. Verificação automatizável por regra de dependência.
- **NFR-3 — Pagamento atrás de fronteira trocável.** Substituir a implementação altera apenas ela e a configuração: zero alteração no checkout, no Pedido ou na máquina de estados.
- **NFR-4 — Desempenho da busca.** Com 5.000 Produtos semeados, busca com filtros em ≤ 500 ms no p95.
- **NFR-5 — Armazenamento de senha.** Função de derivação lenta com sal por usuário; nenhuma senha legível nem hash de uma passagem.
- **NFR-6 — Autorização no servidor.** Teste automatizado chama a API com identificador de recurso de outro Comprador e espera negação — para Pedido, Carrinho e Endereço.
- **NFR-7 — Consistência de Estoque sob concorrência.** N requisições paralelas na última unidade produzem `pedidos_criados == estoque_inicial`, nunca mais. **O teste é o critério de aceite e nunca é cortado.**
- **NFR-8 — Cobertura do fluxo crítico.** Todo FR do fluxo busca → Carrinho → checkout → Pedido tem ao menos um teste que **falha se a consequência declarada quebrar**. A suíte roda em contêiner.
- **NFR-9 — Rastreabilidade.** Identificador de correlação em toda linha de log e toda transição de Status registrada: dado um número de Pedido, a sequência de eventos é reconstruível.
- **NFR-10 — Responsividade.** 360 px a 1440 px sem rolagem horizontal e sem elemento inacessível.
- **NFR-11 — Acessibilidade.** Fluxo de compra operável por teclado, todo campo com rótulo associado, contraste AA. Verificação: percorrer a UJ-1 inteira sem mouse.
- **NFR-12 — Idempotência.** Criar Pedido e confirmar pagamento são idempotentes por chave: a mesma operação duas vezes produz um único efeito.
- **NFR-13 — Precisão monetária.** Inteiros de centavos, nunca ponto flutuante; arredondamento num único ponto declarado; o total exibido é sempre a soma exata das parcelas exibidas.
- **NFR-14 — Limites de entrada.** Todo campo e parâmetro do cliente tem limite máximo declarado e verificado **no servidor**.
- **NFR-15 — Operação offline.** O fluxo completo funciona sem internet: nenhuma imagem, fonte, folha de estilo, script ou dependência buscada em rede externa em tempo de execução. Verificação: desconectar a rede e percorrer o Roteiro A inteiro.
- **NFR-16 — Parâmetros declarados.** Os 15 limiares do §7.1 têm valor padrão e vivem em configuração, nunca como constante espalhada.

### Requisitos Adicionais

*Vindos da espinha de arquitetura. Os 20 `AD` têm ID estável e são o substrato que as estórias citam;
uma estória que contradiga um AD é defeito, não variação.*

**🚨 Andaime inicial (impacta a Estória 1.1).** A arquitetura **não adota um template inicial de
terceiros** — é greenfield com a árvore de origem já declarada. O que a Estória 1.1 precisa produzir:
`docker-compose.yml` com os quatro serviços (`web`, `azamon`, `postgres`, `redis`), módulo Go com a
árvore `cmd/ · internal/<módulo>/ · api/ · db/migracoes · db/semente · media/ · web/`, **goose** com
versionamento por *timestamp* embutido por `embed.FS` e aplicado no arranque do binário (nunca
auto-migração), **sqlc**, projeto Next.js 16 com `next.config.js` reescrevendo `/api/*` para o Go, e
`npx shadcn init`. `docker compose up` sem argumento sobe em modo demonstração.

| AD | Invariante em uma linha |
|---|---|
| **AD-1** | Seis módulos; um importa outro só por `internal/<módulo>/<módulo>.go`. O grafo de setas é **exaustivo** e verificado por `internal/fronteira_test.go` com `go list -deps -json`. |
| **AD-2** | Um schema do Postgres por módulo; **chave estrangeira cruzando schema é proibida**, verificado por consulta a `information_schema`. `busca` não tem tabelas — lê `catalogo` por VIEW. |
| **AD-3** | A transição é o único ponto de mutação do Pedido. Não existe `UPDATE pedido SET status`: só `pedido.Transicionar(...)`, que é um **compare-and-swap** (zero linhas = `ErrEstadoJaAvancado`). Transição primeiro, efeito sobre Estoque depois. Três recusas distintas, nunca uma só. O Carrinho é esvaziado pela criação do Pedido e por mais ninguém. |
| **AD-4** | Uma transação por caso de uso, passada **como parâmetro nomeado** (`tx pgx.Tx`), nunca no `context`. `READ COMMITTED` declarado. Ordem global de bloqueio: linha de `pedido`, depois `catalogo.produto` **ordenadas por `id`**. `40P01`/`40001` repetem uma vez em `api/`. Nenhuma chamada de rede dentro de transação aberta. |
| **AD-5** | O Catálogo é dono do Estoque e da Reserva; disponível é **derivado**, nunca armazenado. `SELECT … ORDER BY id FOR UPDATE` **primeiro**, soma das reservas depois — dois comandos, nesta ordem. Reserva tem estado (`ATIVA → LIBERADA \| CONSOLIDADA`) com índice único parcial. Toda mutação é idempotente: `Liberar` sem reserva ativa devolve `nil`. `Disponivel` é conselho, nunca decisão. |
| **AD-6** | A varredura é o único motor do tempo — nenhum `time.Timer` em módulo nenhum. Um tique de 1 s chama, **nesta ordem**: aplicar → expirar → simular → emitir. Uma transação por Pedido, `FOR UPDATE SKIP LOCKED`; `ErrEstadoJaAvancado` é desfecho esperado. Emitir mora em `pagamento`, não em `pedido`. |
| **AD-7** | Confirmação entra por `POST /api/v1/webhooks/pagamento`, gravada com **restrição única** sobre a chave de idempotência. `pagamento` **não conhece `pedido`**: a varredura lê e aplica. Só aplica se pertence à Tentativa corrente **e** o Pedido está em `AGUARDANDO_PAGAMENTO`; todo outro caso é terminal e sinalizado. `POST /pedidos` guarda chave **e** digest: igual devolve o Pedido original, diferente é `409`. |
| **AD-8** | Porta com duas operações e mais nada. O Simulado decide pelos centavos, só na primeira Tentativa; sem superfície de operador e sem alternância em tempo de execução. **O teto de 3 Tentativas é de `pagamento`** (`ErrTetoDeTentativas`). |
| **AD-9** | `int64` de centavos ponta a ponta, sufixo `_centavos`. `total_centavos` é **coluna com `CHECK`**, nunca derivação. Nenhuma divisão no caminho monetário. Formatação `R$` só no navegador. |
| **AD-10** | Next.js é casca: `rewrites()` de `/api/*` para o Go, uma origem só, sem CORS. Proibidos Server Actions, Route Handlers com regra e acesso a banco pelo Node. Duas armadilhas silenciosas do Next 16: `middleware.ts` virou `proxy.ts`; `next/image` recusa IP privado → `unoptimized`. |
| **AD-11** | `api/` resolve a Sessão; a posse é verificada **dentro do módulo dono, na mesma consulta** (`WHERE id = $1 AND comprador_id = $2`). Recurso de outro devolve o mesmo erro de inexistente. Administrador por prefixo de rota. |
| **AD-12** | Nada de rede externa: fonte WOFF2 auto-hospedada, imagens em `media/` servidas pelo Go com URL relativa. Passo de CI faz `grep` por `fonts.googleapis`, `@import url(http`, `//cdn.` **e `next/font/google`**, e falha o build. |
| **AD-13** | Todo limiar em variável de ambiente `AZAMON_`, lida uma vez para uma struct no arranque. Os 15 do §7.1 são o mínimo; somam-se intervalo da varredura, atraso da confirmação, interruptor da simulação e URL do webhook. `api/` aplica os limites do NFR-14 na decodificação do DTO. |
| **AD-14** | Erro sentinela por módulo, sem conhecer HTTP; tradução num arquivo só. Envelope `{"erro":{"codigo","mensagem","dados","correlacao"}}` — carrega **dado**, porque a UX precisa dele. Erro engolido é proibido. |
| **AD-15** | `X-Correlation-Id` no contexto, em toda linha de log e **no cabeçalho da resposta**. Pedido tem `numero` (`AZ-<ano>-<6 dígitos>`) separado do `uuid`. `transicao_status` é uma fonte com três usos. Nenhum dado pessoal em log. |
| **AD-16** | `GET /api/v1/produtos` é servido **exclusivamente por `busca`**, com ou sem `termo`; `catalogo` serve `/produtos/<id>` e o CRUD administrativo e **não registra listagem** — dois donos fazem o binário entrar em pânico no arranque. VIEW expõe `estoque_disponivel`, nunca o total. Normalização na escrita, índice GIN `pg_trgm`. |
| **AD-17** | Regra de Frete é a tabela `pedido.faixa_frete` + função pura; `carrinho` **não calcula Frete**. `carrinho.ConfirmarPrecoVisto` só é chamada por `pedido`, na entrada do checkout, depois de reportar a diferença. A criação recalcula sob o mesmo bloqueio e recusa com `TOTAL_DIVERGENTE`. |
| **AD-18** | Nenhuma tela faz mais de uma chamada para derivar uma ação: a resposta do Pedido carrega a tripla. Instante sempre absoluto, RFC 3339, do servidor. Consulta em intervalo em **três superfícies** (3 s / 10 s / 10 s); `pedido.EstadoTerminal` é a única função de terminalidade. Envelope único de listagem, desempate sempre terminando em `id`. |
| **AD-19** | "Produto visível" (Produto ativo **e** Vendedor ativo) definido uma vez, em `catalogo`. Vale também dentro das escritas: `Disponivel` devolve `0` para invisível, `Reservar` verifica visibilidade na mesma consulta `FOR UPDATE`. |
| **AD-20** | Argon2id (`m=19456`, `t=2`, `p=1`, sal de 16 bytes). Sessão, token de redefinição e contador de bloqueio no Redis com TTL. **Não existe serviço de e-mail:** o token da FR-3 sai no log estruturado. |

**Stack fixada** (verificada na web em 2026-09-06): Go 1.27.1 · PostgreSQL 18.6 · Redis 8.10.1 ·
Node 24.20.0 · **Next.js 16.3.4** (a linha 16.3.x é o patch das duas RCE de 25/08/2026; a 16.2.x não
recebe backport) · React 19.2.8 · pgx v5.10.0 · sqlc v1.31.1 · goose v3.28.0 · testcontainers-go
v0.44.0 · `net/http.ServeMux` · shadcn/ui por CLI.

**Convenções que viram AC:** identificadores de domínio em português, literalmente os 20 termos do §3
· `uuid` com `DEFAULT uuidv7()` · `timestamptz` UTC · rotas `/api/v1/<recurso-plural>` e área
administrativa sob `/api/v1/admin/` · estado de listagem na URL · `Idempotency-Key` gerada pelo
navegador ao entrar na Revisão e guardada em `sessionStorage` · cookie `azamon_sessao` `HttpOnly`
`SameSite=Lax` opaco de 256 bits · `CREATE EXTENSION pg_trgm` como primeira migração de `catalogo` ·
semente aplicada no arranque, depois das migrações, só se o marcador de versão não existir · commits
em inglês, tudo que sai para o usuário em português.

**Cinco verificações do passo 0** (Questões em Aberto da espinha — checagens de minutos, não decisões):
`Set-Cookie` do Go atravessando o `rewrites()` do Next · sqlc analisando `DEFAULT uuidv7()` ·
`npx shadcn init` limpo em Next 16 + React 19 · p95 do NFR-4 com 5.000 Produtos · e o enunciado do
professor (fora do controle do time, dono: Sung).

### Requisitos de UX

*Extraídos do par `DESIGN.md` + `EXPERIENCE.md`. São contrato, não sugestão.*

- **UX-DR1 — Base shadcn/ui sobre Tailwind.** Instalar por CLI, componentes copiados para `web/components/ui/`. Usados **sem alteração**: `Card`, `Dialog`, `Sheet`, `Select`, `Table`, `Badge`, `Toast`, `Skeleton`, `Pagination`, `Input`, `Label`, `Separator`, `DropdownMenu`, `RadioGroup`, `Alert`. Customizá-los é violação de disciplina.
- **UX-DR2 — Dez tokens de cor da camada de marca**, e só eles, sobrescrevendo o shadcn: `chrome #131921`, `chrome-foreground #FFFFFF`, `chrome-muted #232F3E`, `primary #FFD814`, `primary-foreground #0F1111`, `primary-strong #FFA41C`, `primary-strong-foreground #0F1111`, `link #007185`, `available #007600`, `available-foreground #FFFFFF`. Todo token não listado herda do shadcn, inclusive `destructive`. Sem modo escuro.
- **UX-DR3 — Três papéis tipográficos de marca:** `wordmark` 22px/700 com tracking −0.02em; `preco` 28px/500; `preco-centavos` 14px/500 sobrescritos e alinhados ao topo. Todo valor monetário e toda quantidade usam `font-variant-numeric: tabular-nums`. Texto de conteúdo nunca abaixo de 14px, com 4,5:1 verificado — o cinza pequeno da referência não é copiado.
- **UX-DR4 — Fonte auto-hospedada.** `Azamon Sans` como WOFF2 em `web/public/fonts/`, apelidada no CSS, com a pilha de sistema declarada como alternativa. Proibidos Google Fonts, `@import` remoto, CDN e `next/font/google`; o passo de CI do AD-12 falha o build (NFR-15).
- **UX-DR5 — Raios.** `sm` 4px, `md` 6px, `lg` 8px herdados. `rounded.full` (9999px) em **exatamente quatro usos**: busca global, botões de ação, selos de Status do Pedido e chips de filtro ativo.
- **UX-DR6 — Ritmo de espaço.** `section` 48px (≥1024px), `section-sm` 32px (360–1023px), `grid-gutter` 24px entre cartões, `page-margin` 16px / `page-margin-lg` 32px, conteúdo limitado a 1440px. A grade de Produtos vai de 1 a 4 colunas. A busca é o único elemento que não respeita o ritmo folgado.
- **UX-DR7 — Semântica fechada de cor.** Verde `{colors.available}` só para **Estoque disponível** e **`ENTREGUE`** — nunca sucesso de operação, confirmação de formulário ou `Toast`. Laranja `{colors.primary-strong}` **uma vez por fluxo**, no botão Confirmar Pedido. Uso decorativo de qualquer um apaga o único mecanismo semântico de cor do sistema.
- **UX-DR8 — Dezesseis composições da camada de marca**, cada uma com comportamento declarado: Barra superior · Faixa de Categorias · Busca global · Botão Adicionar ao Carrinho · Botão Confirmar Pedido · Caixa de compra · Selo de disponibilidade · Cartão de Produto · Barra de filtros · Linha de Item de Carrinho · Menu da conta · Navegação administrativa · Breadcrumb · Passos do checkout · Linha do tempo do Pedido · Selo de Status do Pedido. Mais duas sem delta visual, que são o componente do shadcn direto: Tabela de Pedidos (`Table`) e Paginação (`Pagination`).
- **UX-DR9 — Chrome por área.** Barra superior, busca global, Faixa de Categorias e Menu da conta presentes em **toda** tela pública e de Comprador. A área administrativa é separada da loja: chrome próprio em `chrome-muted`, sem busca e sem Carrinho, e não aparece para quem não é Administrador — nem como link desabilitado.
- **UX-DR10 — Forma de exibição dos sete Status**, fechada e um-para-um: `AGUARDANDO_PAGAMENTO` → "Aguardando pagamento" · `PAGAMENTO_RECUSADO` → "Pagamento recusado" · `PAGO` → "Pago" · `SEPARANDO` → "Separando" · `ENVIADO` → "Enviado" · `ENTREGUE` → "Entregue" · `CANCELADO` → "Cancelado". Mesmo componente, texto e cor nas três superfícies; nunca o identificador cru, nunca ícone.
- **UX-DR11 — Ação derivada da tripla** `(Status, tentativas restantes, disponibilidade dos Itens)` — só o Status não basta. **O Administrador não cancela Pedido:** as três transições da FR-32 são exaustivas. Botão **ausente, não desabilitado**, quando a causa é o estado, com uma frase que explica; durante operação em curso é o contrário — fica na tela, desabilitado e com progresso.
- **UX-DR12 — Os 24 estados de `EXPERIENCE.md.State Patterns` são requisito, e cada um é uma AC.** Os oito de maior consequência, porque nascem de FR e não de tela: Vitrine sem Catálogo Semeado · credencial inválida com a **mesma** mensagem · conta bloqueada contando o par (e-mail, origem) exista ou não a conta · revalidação com preço mudado (confirma) × unidade indisponível (**bloqueia**) · Pedido em processamento com as três saídas declaradas e ação persistente para não prender o Comprador · cancelamento recusado porque o estado mudou (nunca `Toast` genérico) · Pedido já avançado por outro ator (`Alert` **informativo**, não destrutivo) × transição inválida (`Alert` destrutivo nomeando as permitidas) · pagamento aprovado sobre Pedido cancelado, com `Alert` persistente e marcador na tabela.
- **UX-DR13 — Consulta em intervalo em três superfícies e nenhuma outra:** Pedido em processamento a 3 s enquanto `AGUARDANDO_PAGAMENTO`; Detalhe do Pedido a 10 s enquanto não terminal; Tabela de Pedidos do Administrador a 10 s enquanto listar Pedido não terminal. Sem WebSocket. Todo instante é absoluto e vem do servidor — o relógio de expiração nunca é contado no navegador.
- **UX-DR14 — Piso WCAG 2.2 AA completo.** `label` associado programaticamente em todo campo (nunca `placeholder` como rótulo); foco visível a 3:1; ordem de tabulação igual à de leitura; `aria-live="polite"` na mudança de Status e na contagem de resultados; erro associado por `aria-describedby` e com foco ao submeter; alt com o nome do Produto. Verificação mínima: percorrer a UJ-1 inteira sem tocar no mouse.
- **UX-DR15 — Quatro faixas responsivas declaradas em pixels:** 360–639 (1 coluna, busca em linha própria, checkout um passo por tela) · 640–1023 (2 colunas, filtros em `Sheet`) · 1024–1439 (3 colunas, filtros fixos à esquerda, caixa de compra à direita) · ≥1440 (4 colunas, centralizado). Sem rolagem horizontal; tabela larga rola dentro do contêiner. Loja **mobile-first**, painel administrativo **desktop-first**.
- **UX-DR16 — Voz e vocabulário.** Nomear o motivo sempre que o sistema souber, **com a exceção de segurança na autenticação**. Sem urgência fabricada, sem celebração ao concluir Pedido. Os 20 termos do glossário usados literalmente na tela — "Pedido", não "compra"; "Produto", não "item".
- **UX-DR17 — Dinheiro e Estoque na tela.** O total exibido é a soma exata das parcelas exibidas; preço praticado, nunca preço de hoje; a interface não sugere que o Carrinho congela preço ou reserva Estoque; a composição monetária do Carrinho é **subtotal e nada mais**; a distância até o Frete grátis em valor absoluto, sem barra de progresso; **o número exibido é sempre o Estoque disponível**; a Reserva de Estoque nunca é nomeada para o Comprador — é vocabulário legítimo só para o Administrador, na guarda da FR-11.
- **UX-DR18 — Primitivas de interação.** Todo estado de listagem vive na URL; `Enter` submete a busca de qualquer tela; atualização otimista **apenas** na quantidade de Item de Carrinho; `Dialog` no máximo um nível e só onde a ação é irreversível; `Esc` fecha o mais acima. Proibidos rolagem infinita, afordância só em `hover` abaixo de `md` e `Dialog` empilhado.
- **UX-DR19 — Três mockups HTML** como referência de composição — página de Produto, resultados de busca e detalhe do Pedido. Ilustram; **as duas espinhas vencem em conflito**.
- **UX-DR20 — Lacunas conhecidas da UX, a fechar durante a construção.** Achados de revisão que não voltaram às espinhas e que uma estória vai encontrar: (a) a FR-3 não tem estado declarado — faltam "solicitação enviada" com resposta neutra e "token inválido ou expirado"; (b) `Skeleton` só está declarado para Vitrine, Resultados e Página de Produto — Meus pedidos, Detalhe do Pedido e as listas administrativas ficaram sem; (c) o padrão de erro em linha por campo está declarado só para formulários do Administrador — Cadastro (FR-1) e Endereço (FR-5) precisam do mesmo.

### Mapa de Cobertura de FR

*Os 34 FRs, um por linha, cada um com exatamente **uma épica dona**. Dona é onde o FR é **fechado**:
atravessá-lo parcialmente antes — a Épica 1 faz isso com nove deles — não transfere a posse. Épicas 1 e
7 não aparecem nesta coluna porque fecham NFR e entregável, não FR, e isso é desenho, não lacuna.*

| FR | Épica dona | O que a épica fecha |
|---|---|---|
| **FR-1** Cadastro de Comprador | Épica 2 | Conta criada com e-mail normalizado, senha em Argon2id, já autenticado |
| **FR-2** Autenticação e Sessão | Épica 2 | Sessão no Redis, mensagem genérica única, bloqueio por (e-mail, origem) |
| **FR-3** Recuperação de senha | Épica 2 | Token de uso único no log estruturado; resposta neutra. *1º na ordem de corte* |
| **FR-4** Papéis e autorização | Épica 2 | Comprador × Administrador; posse verificada na mesma consulta (AD-11) |
| **FR-5** Endereços | Épica 2 | CRUD e escolha; remover não altera Pedido existente |
| **FR-6** Vitrine | Épica 3 | Listagem servida por `busca` sobre a VIEW; estado vazio explicativo |
| **FR-7** Página de Produto | Épica 3 | Detalhe servido por `catalogo`, com nome do Vendedor e disponibilidade |
| **FR-8** Gestão de Vendedores | Épica 3 | Criar, editar, desativar; desativar oculta Produtos sem apagar Pedidos. *3º no corte* |
| **FR-9** Gestão de Produtos | Épica 3 | CRUD com Vendedor e Categoria; aparece na Vitrine sem reiniciar |
| **FR-10** Gestão de Categorias | Épica 3 | Criar e renomear; coluna de categoria-pai nasce nula (addendum §6). *2º no corte* |
| **FR-11** Estoque e disponibilidade | Épica 3 | Disponível derivado; guarda que informa quantas unidades estão comprometidas |
| **FR-12** Busca por texto | Épica 3 | `pg_trgm` sem caixa e sem acento; `%` e `_` literais; teto de 100 caracteres |
| **FR-13** Filtros | Épica 3 | Categoria e faixa de preço combináveis, removíveis, com estado na URL |
| **FR-14** Ordenação | Épica 3 | Três ordens com desempate estável terminando em `id`. *5º na ordem de corte* |
| **FR-15** Paginação | Épica 3 | 20 por página, teto 60; envelope único de listagem do AD-18 |
| **FR-16** Carrinho de autenticado | Épica 4 | Visitante vai ao Login e volta à Página de Produto de origem |
| **FR-17** Adicionar Produto | Épica 4 | Soma no Item existente; recusa acima do disponível informando o disponível |
| **FR-18** Alterar e remover | Épica 4 | Zero remove; total recalculado com os preços atuais do Catálogo |
| **FR-19** Revalidação do Carrinho | Épica 4 | Preço mudado confirma, indisponível **bloqueia**; Frete recalculado. 🔒 **Nunca cortar** |
| **FR-20** Seleção de Endereço | Épica 5 | Escolher ou cadastrar; trocar recalcula o Frete antes da confirmação |
| **FR-21** Cálculo do Frete | Épica 5 | Tabela `pedido.faixa_frete` + função pura; isenção acima de R$ 299,00 |
| **FR-22** Revisão do Pedido | Épica 5 | Itens, Endereço, Frete e total; forma de pagamento declarada sem coletar dado |
| **FR-23** Criação do Pedido | Épica 5 | Nasce em `AGUARDANDO_PAGAMENTO` congelando preço, Endereço, Frete e Vendedor |
| **FR-24** Reserva de Estoque | Épica 5 | Tudo ou nada sob `ORDER BY id FOR UPDATE`. 🔒 **Nunca cortar** |
| **FR-25** Tentativa de Pagamento | Épica 5 | Resposta não espera o resultado; Simulado decide pelos centavos, só na 1ª |
| **FR-26** Confirmação aprovada | Épica 5 | Webhook por *inbox* idempotente; aprovação sobre `CANCELADO` é registrada, não aplicada |
| **FR-27** Pagamento recusado | Épica 5 | Reserva liberada, nova Tentativa a partir do Pedido, teto de 3 |
| **FR-28** Máquina de estados | Épica 5 | *Estória 5.1, com testes, antes das telas.* CAS, nove transições, histórico imutável |
| **FR-34** Expiração da Tentativa | Épica 5 | Derivada do histórico pela varredura, nunca de temporizador. 🔒 **Nunca cortar** |
| **FR-29** Histórico de Pedidos | Épica 6 | Lista do Comprador autenticado, mais recentes primeiro, com estado vazio |
| **FR-30** Detalhe do Pedido | Épica 6 | Preço praticado, Endereço congelado e linha do tempo das transições |
| **FR-31** Cancelamento pelo Comprador | Épica 6 | Quatro estados de origem; libera a Reserva e as unidades voltam na hora |
| **FR-32** Painel do Administrador | Épica 6 | Exatamente três transições; recusa nomeia as permitidas. **Não cancela Pedido** |
| **FR-33** Simulação de entrega | Épica 6 | Avanço derivado do histórico; reiniciar retoma de onde parou. *4º na ordem de corte* |

**34 de 34 FRs mapeados.** Nenhum FR sem dona, nenhuma dona compartilhada.

## Lista de Épicas

**Sete épicas.** A sequência é a do §8 do addendum, que abre com o aviso que governa este documento
inteiro: *"a ordem a seguir é correta por dependência e perigosa por risco: seguida ao pé da letra, ela
produz um sistema que só funciona no fim do semestre, e o fim do semestre é onde o tempo acaba."* Por
isso o passo 0 — o esqueleto vertical — é épica, e é a primeira.

Cada épica é autossuficiente: pode apoiar-se nas anteriores, nunca depender de uma futura para
funcionar. As dependências reais são estas e mais nenhuma: 3, 4, 5 e 6 exigem a 2 (saber quem está
falando); 4, 5 e 6 exigem a 3 (existir Produto); 5 exige a 4; 6 exige a 5.

### Épica 1: Ciclo fechado em um comando

`docker compose up` sobe o Azamon inteiro e **um** Pedido atravessa o sistema do clique ao `ENTREGUE` —
um Produto semeado, um Comprador, telas cruas, sem busca, sem filtro e sem painel. Existe para forçar a
integração entre front e back na semana 2 ou 3, quando ainda há tempo de descobrir que o decimal virou
ponto flutuante e que o cookie de Sessão não atravessa a porta. É a rede de segurança da **SM-6**.

**FRs cobertos:** nenhum fechado. Atravessa o mínimo viável de FR-1, FR-2, FR-7, FR-17, FR-23, FR-25,
FR-26, FR-28 e FR-33 — cada um permanece propriedade da sua épica.
**NFRs:** NFR-1, NFR-15 · **ADs:** AD-1, AD-2, AD-10, AD-12, AD-13, AD-15 · **UX:** UX-DR1 a UX-DR7
**CAP:** CAP-7 · **Passo do addendum §8:** 0 · **Roteiro:** A1

> 🚨 **A Estória 1.1 é o andaime.** Greenfield, sem template de terceiros: `docker-compose.yml` com
> `web`, `azamon`, `postgres` e `redis`; árvore `cmd/ · internal/<módulo>/ · api/ · db/migracoes ·
> db/semente · media/ · web/`; **goose** por `embed.FS` aplicado no arranque do binário, nunca
> auto-migração; **sqlc**; Next.js 16 com `rewrites()` de `/api/*`; `npx shadcn init`. `docker compose
> up` sem argumento sobe em modo demonstração.

As **cinco verificações do passo 0** entram aqui como estórias curtas. A primeira — o `Set-Cookie` do Go
atravessando o `rewrites()` do Next — é a que mais custa se for descoberta tarde.

### Épica 2: Conta, Identidade e Endereços

Um visitante cria conta, abre e encerra Sessão, redefine senha e gerencia seus Endereços — e o sistema
passa a saber quem está falando e a negar recurso de outro dono, inclusive contra chamada direta à API.

**FRs cobertos:** FR-1, FR-2, FR-3, FR-4, FR-5
**NFRs:** NFR-5, NFR-6 *(o mecanismo do AD-11 nasce aqui; cada épica posterior acrescenta seu recurso ao teste)*
**ADs:** AD-11, AD-13, AD-20 · **UX:** UX-DR20(a) — os dois estados faltantes da FR-3 — e UX-DR20(c) — erro em linha por campo no Cadastro e no Endereço
**CAP:** CAP-1 · **Passo:** 1 · **Roteiro:** A2 · **Ordem de corte:** FR-3 é o 1º a cair

### Épica 3: Catálogo, Estoque e Busca

O Administrador abastece a loja com Vendedores, Categorias, Produtos e Estoque; qualquer visitante vê a
Vitrine, encontra Produtos por termo, filtro e ordem, e abre a Página de Produto com o nome do Vendedor
e a disponibilidade real. Produto criado aparece na Vitrine e na busca sem reiniciar a aplicação.

**FRs cobertos:** FR-6, FR-7, FR-8, FR-9, FR-10, FR-11, FR-12, FR-13, FR-14, FR-15
**NFRs:** NFR-4 · **ADs:** AD-2, AD-5, AD-9, AD-16, AD-18, AD-19 · **UX:** UX-DR8 (Cartão de Produto, Selo de disponibilidade, Barra de filtros, Faixa de Categorias), UX-DR9 (chrome administrativo em `chrome-muted`), UX-DR17 (o número exibido é sempre o Estoque disponível)
**CAPs:** CAP-2 **+** CAP-3 · **Passos:** 2 e 6 · **Roteiros:** A3, A4, B3, C1
**Ordem de corte:** FR-10 (2º), FR-8 (3º) e FR-14 (5º) — três dos cinco cortes moram aqui

> **Por que as duas capacidades viraram uma épica.** O **AD-16** dá um dono só ao
> `GET /api/v1/produtos` — servido **exclusivamente por `busca`**, com ou sem termo, e *"dois donos
> fazem o binário entrar em pânico no arranque"*. A Vitrine da FR-6 e os Resultados de busca da FR-12
> são o mesmo endpoint, e termo vazio devolve a listagem paginada. Duas épicas fariam a segunda reabrir
> um módulo que a primeira construiu pela metade.
>
> A paralelização que o addendum §8 valoriza — *"o corte mais independente do sistema"* — sobrevive no
> nível de estória: termo, filtros e ordenação são a raia paralela a partir do momento em que a VIEW
> existe.

### Épica 4: Carrinho

Um Comprador autenticado monta um Carrinho que sobrevive à Sessão e ajusta suas quantidades — e é
avisado antes de qualquer surpresa, porque o Carrinho é intenção, não compromisso: não congela preço e
não reserva Estoque.

**FRs cobertos:** FR-16, FR-17, FR-18, FR-19
**ADs:** AD-5, AD-9, AD-11, AD-17, AD-19 · **UX:** UX-DR8 (Linha de Item de Carrinho), UX-DR17 (composição monetária é subtotal e nada mais; a Reserva de Estoque nunca é nomeada ao Comprador), UX-DR18 (atualização otimista **apenas** na quantidade)
**CAP:** CAP-4 · **Passo:** 4 · **Roteiro:** A5
**🔒 Nunca cortar:** FR-19

### Épica 5: Checkout e pagamento assíncrono

O Comprador escolhe o Endereço, vê o Frete e o total, confirma — e o Pedido nasce em
`AGUARDANDO_PAGAMENTO` com Reserva de Estoque atômica sobre todos os itens. O resultado do pagamento
chega **depois**, na tela do Pedido em processamento; recusa, nova tentativa e expiração são caminhos de
primeira classe, não erros.

**FRs cobertos:** FR-28 *(primeiro, com testes)*, FR-20, FR-21, FR-22, FR-23, FR-24, FR-25, FR-26, FR-27, FR-34
**NFRs:** NFR-3, NFR-7, NFR-12 · **ADs:** AD-3, AD-4, AD-5, AD-6, AD-7, AD-8, AD-9, AD-17, AD-18 · **UX:** UX-DR8 (Passos do checkout, Botão Confirmar Pedido — a única aparição do laranja), UX-DR12 (Pedido em processamento com as três saídas; aprovado sobre cancelado), UX-DR13 (consulta a 3 s)
**CAP:** CAP-5 · **Passos:** 3 e 5 · **Roteiros:** A6, A7, B1 a B5, C5
**🔒 Nunca cortar:** FR-24, FR-34 e o teste de concorrência do NFR-7

> **Por que a FR-28 mora aqui.** O mapa de capacidades declara a CAP-6 *"a única capacidade partida em
> dois passos, e a divisão é deliberada"*: a máquina de estados é o passo 3, o que ela sustenta é o
> passo 7. O §14 do PRD pede construí-la *"com testes, antes das telas que dependem dela"* — como
> **estória 5.1** ela precede toda tela que a consome. A Épica 1 já terá atravessado o caminho mínimo;
> aqui ele é alargado para as nove transições declaradas, as três recusas distintas e a consolidação da
> Reserva em `SEPARANDO → ENVIADO`.

### Épica 6: Pedidos — acompanhamento, cancelamento e operação

O Comprador vê o histórico, abre o detalhe com o **preço praticado** e a linha do tempo, e cancela antes
do envio; o Administrador opera a lista pelas três transições que lhe cabem; e a simulação de entrega
avança sozinha até `ENTREGUE`. Quatro ângulos da mesma máquina — colhe o que a estória 5.1 plantou.

**FRs cobertos:** FR-29, FR-30, FR-31, FR-32, FR-33
**ADs:** AD-3, AD-6, AD-15, AD-18 · **UX:** UX-DR10 (os sete Status, um-para-um), UX-DR11 (ação derivada da tripla; **o Administrador não cancela Pedido**), UX-DR13 (consulta a 10 s em duas superfícies), UX-DR20(b) (`Skeleton` faltante em Meus pedidos, Detalhe e listas administrativas)
**CAP:** CAP-6 *(segunda metade)* · **Passo:** 7 · **Roteiros:** A8, A9, C2 a C4
**Ordem de corte:** FR-33 é o 4º a cair

### Épica 7: Entrega — documentação viva e ensaio

A banca encontra, no dia da entrega, um README que leva do clone ao sistema rodando, o diagrama dos seis
módulos correspondendo à decomposição real do código, e o addendum sem nenhuma decisão contradita pelo
que foi construído. E os três roteiros do §12 ensaiados a partir de clone limpo, por ao menos duas
pessoas diferentes.

**FRs cobertos:** nenhum — o entregável é o §6.1 do PRD
**Métricas:** SM-1, SM-3, SM-7 · **AD:** AD-1 · **CAP:** CAP-8 · **Passo:** contínuo

> ⚠️ **É o fecho, não o começo do trabalho.** O mapa de capacidades avisa que *"uma épica que trate [os
> transversais] como trabalho de alguém no fim do semestre é uma épica que os perde"*. O addendum é
> mantido vivo desde a Épica 1: toda decisão de arquitetura tomada durante a construção volta para ele
> **na mesma estória que a tomou**. Esta épica verifica que nenhuma mente — não as escreve do zero.
>
> A contra-métrica **SM-C1** é um portão, não uma estória: nenhuma funcionalidade nova entra enquanto os
> três roteiros não passarem.

### Requisitos transversais — cobram de toda épica

Sete NFRs e cinco UX-DRs não pertencem a nenhuma épica sozinha e viram AC em **todas** onde se aplicam.
Não existe épica que os recolha no fim.

| Transversal | O que cobra de toda épica | Onde vive a regra |
|---|---|---|
| **NFR-2** | Um módulo só é alcançado pela sua interface pública | AD-1, `internal/fronteira_test.go` |
| **NFR-8** | Todo FR do fluxo crítico com teste que falha se a consequência declarada quebrar | SM-2 |
| **NFR-9** | Correlação em toda linha de log; toda transição de Status registrada | AD-15 |
| **NFR-13** | Centavos inteiros; o total exibido é a soma exata das parcelas exibidas | AD-9, UX-DR17 |
| **NFR-14** | Limite declarado e verificado **no servidor** para todo campo e parâmetro | AD-13, aplicado em `api/` |
| **NFR-16** | Todo limiar em variável `AZAMON_`, nunca constante espalhada | AD-13, §7.1 do PRD |
| **NFR-6** | Recurso de outro dono devolve o mesmo erro de inexistente | AD-11 — mecanismo na Épica 2, recurso novo em cada épica |
| **NFR-10 · UX-DR15** | 360 a 1440 px sem rolagem horizontal, nas quatro faixas declaradas | `EXPERIENCE.md` |
| **NFR-11 · UX-DR14** | WCAG 2.2 AA completo; percorrer a UJ-1 inteira sem tocar no mouse | `EXPERIENCE.md` |
| **UX-DR12** | Os 24 estados do `EXPERIENCE.md` — **cada um é uma AC**, na épica que o origina | `EXPERIENCE.md` |
| **UX-DR16** | Os 20 termos do glossário literais na tela; motivo nomeado, exceto na autenticação | §3 do PRD |
| **UX-DR18** | Estado de listagem na URL; `Dialog` de um nível só; sem rolagem infinita | `EXPERIENCE.md` |

---

## Épica 1: Ciclo fechado em um comando

`docker compose up` sobe o Azamon inteiro e **um** Pedido atravessa o sistema do clique ao `ENTREGUE` —
um Produto semeado, um Comprador, telas cruas, sem busca, sem filtro e sem painel. Existe para forçar a
integração entre front e back na semana 2 ou 3, quando ainda há tempo de descobrir que o decimal virou
ponto flutuante e que o cookie de Sessão não atravessa a porta. É a rede de segurança da **SM-6**, e as
cinco verificações do passo 0 fecham dentro dela.

*Nenhum FR é fechado aqui. Nove são atravessados no mínimo viável e continuam propriedade das suas
épicas — cada estória diz explicitamente o que deixa para depois.*

> **Posse de migração.** Só as oito estórias marcadas com **Migração desta estória** criam tabela, coluna
> ou índice: 1.3, 1.4, 1.6, 1.7, 2.5, 4.1, 5.3 e 5.6. **Nenhuma outra estória do documento cria schema** —
> as demais leem, escrevem ou alargam comportamento sobre o que já existe. Criar tabela fora dessas oito é
> defeito de sequenciamento, não otimização.

### Estória 1.1: Andaime do serviço — os quatro contêineres em um comando

**Como** integrante do time,
**quero** que `docker compose up` a partir de um clone limpo levante a aplicação, o banco e o cache,
**para que** ninguém precise instalar nada além do Docker para ter o Azamon rodando.

**Condições de Aceite:**

**Dado** um clone limpo do repositório,
**Quando** executo `docker compose up` sem nenhum argumento,
**Então** sobem exatamente quatro serviços — `web`, `azamon`, `postgres` e `redis` — nas versões fixadas na tabela Stack (Go 1.27.1 · PostgreSQL 18.6 · Redis 8.10.1 · Node 24.20.0),
**E** o serviço `azamon` responde `200` em `GET /api/v1/saude`,
**E** o modo de execução é **demonstração**, sem argumento adicional (AD-13).

**Dado** que a árvore é greenfield, sem template de terceiros,
**Quando** o repositório é inspecionado,
**Então** existem `cmd/azamon/`, `internal/{identidade,catalogo,busca,carrinho,pedido,pagamento,plataforma}/`, `api/`, `db/migracoes/`, `db/semente/`, `media/`, `web/`, `docker-compose.yml` e `.env`,
**E** cada módulo de domínio contém ao menos `<módulo>.go` — a interface pública, o único arquivo que outro módulo importa (AD-1),
**E** `api/` contém apenas tradução: rotas, *middleware* e DTO, nenhuma regra.

**Dado** que todo limiar vem de configuração (AD-13, NFR-16),
**Quando** o binário arranca,
**Então** as variáveis com prefixo `AZAMON_` são lidas **uma única vez** para uma struct de configuração,
**E** o `.env` versionado traz os 15 parâmetros do §7.1 nos valores de demonstração — incluindo expiração da Tentativa em **60 s** e intervalo da simulação em **30 s** —, mais os quatro operacionais da arquitetura: intervalo da varredura, atraso da confirmação simulada, interruptor da simulação de entrega e URL base do webhook,
**E** nenhum limiar aparece como constante literal fora dessa struct.

**Dado** que toda requisição precisa ser rastreável (AD-15, NFR-9),
**Quando** uma requisição entra pela `api/`,
**Então** um `X-Correlation-Id` é resolvido ou gerado, colocado no contexto, **devolvido no cabeçalho da resposta** e presente em toda linha de log,
**E** o log sai em JSON por `log/slog` com os campos `correlacao` e `modulo`,
**E** nenhuma linha de log contém dado pessoal.

**Dado** que erro nomeado no módulo é traduzido num único ponto (AD-14),
**Quando** um erro sentinela sobe de um módulo até a `api/`,
**Então** a resposta usa o envelope `{"erro":{"codigo","mensagem","dados","correlacao"}}`, com `dados` carregando o dado que a UX precisa,
**E** nenhum módulo de domínio conhece HTTP,
**E** erro engolido é defeito — não existe caminho silencioso.

**Dado** que migrações são embutidas e aplicadas no arranque, nunca por auto-migração,
**Quando** o binário arranca,
**Então** o goose v3.28.0 aplica as migrações de `db/migracoes/` a partir de um `embed.FS`,
**E** o arranque **falha alto** se uma migração falhar, em vez de servir com o schema errado.

> **Deixa para depois:** nenhuma tabela de domínio é criada nesta estória — a primeira migração vem na 1.3, quando existe algo para guardar.

### Estória 1.2: Casca do navegador — rewrites, shadcn e a base visual da marca

**Como** integrante do time,
**quero** que o Next.js seja uma casca que reescreve `/api/*` para o Go e já carregue a base visual da marca,
**para que** exista uma origem só, sem CORS, e nenhuma tela precise inventar cor, fonte ou espaçamento depois.

**Condições de Aceite:**

**Dado** o projeto Next.js 16.3.4 sobre React 19.2.8,
**Quando** o navegador pede `/api/v1/qualquer-coisa`,
**Então** o `rewrites()` do `next.config.js` encaminha para o serviço `azamon`, numa origem só e sem CORS (AD-10),
**E** não existe Server Action, Route Handler com regra nem acesso a banco pelo Node,
**E** as duas armadilhas silenciosas do Next 16 estão tratadas: `middleware.ts` renomeado para `proxy.ts`, e `next/image` com `unoptimized` porque recusa IP privado.

**Dado** a **verificação 1 do passo 0** — a que mais custa se for descoberta tarde,
**Quando** um `curl -i` bate no Next apontando para um endpoint do Go que emite `Set-Cookie`,
**Então** o cabeçalho `Set-Cookie` chega íntegro ao cliente através do `rewrites()`,
**E** o desfecho é registrado no `addendum.md`,
**E** se não atravessar, a saída declarada é *proxy* explícito em vez de *rewrite* — decidida aqui, não em novembro.

**Dado** a **verificação 3 do passo 0**,
**Quando** executo `npx shadcn init` neste projeto,
**Então** a instalação conclui limpa em Next 16 + React 19,
**E** os 15 componentes da UX-DR1 — `Card`, `Dialog`, `Sheet`, `Select`, `Table`, `Badge`, `Toast`, `Skeleton`, `Pagination`, `Input`, `Label`, `Separator`, `DropdownMenu`, `RadioGroup`, `Alert` — ficam em `web/components/ui/` e são usados **sem alteração**.

**Dado** a camada de marca da UX-DR2,
**Quando** os tokens são definidos,
**Então** existem **exatamente dez** sobrescrevendo o shadcn: `chrome #131921`, `chrome-foreground #FFFFFF`, `chrome-muted #232F3E`, `primary #FFD814`, `primary-foreground #0F1111`, `primary-strong #FFA41C`, `primary-strong-foreground #0F1111`, `link #007185`, `available #007600`, `available-foreground #FFFFFF`,
**E** todo token não listado herda do shadcn, `destructive` inclusive,
**E** não existe modo escuro.

**Dado** a UX-DR3, a UX-DR5 e a UX-DR6,
**Quando** a folha de estilo é inspecionada,
**Então** existem três papéis tipográficos de marca — `wordmark` 22px/700 com *tracking* −0,02em; `preco` 28px/500; `preco-centavos` 14px/500 sobrescrito e alinhado ao topo,
**E** todo valor monetário e toda quantidade usam `font-variant-numeric: tabular-nums`,
**E** nenhum texto de conteúdo fica abaixo de 14px, com 4,5:1 verificado,
**E** os raios são `sm` 4px, `md` 6px, `lg` 8px, com `rounded.full` reservado a exatamente quatro usos,
**E** o ritmo é `section` 48px, `section-sm` 32px, `grid-gutter` 24px, `page-margin` 16px / 32px, conteúdo limitado a 1440px.

**Dado** a UX-DR4 e o AD-12,
**Quando** a fonte é carregada,
**Então** `Azamon Sans` é um WOFF2 auto-hospedado em `web/public/fonts/`, apelidado no CSS, com a pilha de sistema declarada como alternativa,
**E** o passo de CI faz `grep` por `fonts.googleapis`, `@import url(http`, `//cdn.` **e `next/font/google`** e **falha o build** ao encontrar qualquer um.

> **Deixa para depois:** as 16 composições da UX-DR8 nascem nas épicas que as usam. Aqui só entra a base que elas herdam.

### Estória 1.3: Primeira migração e o Catálogo Semeado determinístico

**Como** integrante do time,
**quero** que o arranque crie o schema mínimo e aplique um Catálogo Semeado determinístico,
**para que** subir o ambiente duas vezes produza exatamente os mesmos dados e ninguém precise tocar no banco à mão.

**Migração desta estória:** `identidade.comprador`, `identidade.administrador`, `catalogo.vendedor`, `catalogo.categoria`, `catalogo.produto` — mais `CREATE EXTENSION pg_trgm` como a primeira migração de `catalogo`.

**Condições de Aceite:**

**Dado** que a fronteira do módulo é um schema do Postgres (AD-2),
**Quando** as migrações desta estória são aplicadas,
**Então** existem os schemas `identidade` e `catalogo`, e apenas as tabelas de que o esqueleto precisa: `identidade.comprador`, `identidade.administrador`, `catalogo.vendedor`, `catalogo.categoria` e `catalogo.produto`,
**E** a **primeira migração de `catalogo` é `CREATE EXTENSION pg_trgm`** (AD-16),
**E** `catalogo.categoria` nasce com `categoria_pai_id` nulo e não exposto — a blindagem barata do addendum §6,
**E** uma consulta a `information_schema` confirma que **nenhuma chave estrangeira cruza schema**.

**Dado** as Convenções de Consistência,
**Quando** o schema é inspecionado,
**Então** as chaves primárias são `uuid` com `DEFAULT uuidv7()`, as datas são `timestamptz` em UTC, os nomes são `snake_case` singular, e todo valor monetário é `bigint` com sufixo `_centavos`,
**E** os identificadores de domínio usam literalmente os termos do §3 do PRD.

**Dado** a **verificação 2 do passo 0**,
**Quando** executo `sqlc generate` com a v1.31.1 sobre essas migrações,
**Então** o sqlc analisa `DEFAULT uuidv7()` do PostgreSQL 18 sem erro e a saída vai para `internal/<módulo>/db/gerado/`,
**E** se falhar, a saída declarada é gerar o `uuid` no lado Go — nada mais muda,
**E** o desfecho é registrado no `addendum.md`.

**Dado** que a semente é determinística e versionada,
**Quando** o binário arranca,
**Então** o Catálogo Semeado de `db/semente/` é aplicado **depois** das migrações e **só se o marcador de versão não existir**,
**E** subir duas vezes produz exatamente os mesmos Vendedores, Categorias e Produtos,
**E** a semente **não cria Pedido** — nada nela consome `numero`,
**E** ela inclui uma conta de Administrador e uma de Comprador, com nomes, descrições e preços **plausíveis** (NFR-1),
**E** as imagens vêm de `media/`, servidas pelo Go, com URL relativa no banco (AD-12),
**E** `docker compose down -v` devolve o ambiente ao estado inicial.

> **Deixa para depois:** Estoque, Reserva de Estoque e o predicado de visibilidade do AD-19 são da Épica 3. Aqui `catalogo.produto` guarda o mínimo que a página crua da 1.5 exibe.

### Estória 1.4: A medição do NFR-4 com 5.000 Produtos

**Como** time,
**quero** medir a busca com 5.000 Produtos semeados agora,
**para que** a decisão "índice antes de serviço" seja tomada no passo 0 e não em novembro.

**Migração desta estória:** coluna normalizada em `catalogo.produto` (sem acento, minúscula, preenchida na escrita) e o índice GIN `pg_trgm` sobre ela.

**Condições de Aceite:**

**Dado** a **verificação 4 do passo 0**,
**Quando** o conjunto de 5.000 Produtos da semente é aplicado — ligado por configuração, sem virar o catálogo da demonstração (SM-C3),
**Então** um `EXPLAIN ANALYZE` sobre a consulta com termo, filtro de Categoria e faixa de preço responde em **≤ 500 ms no p95** (NFR-4, SM-4),
**E** existe índice GIN `pg_trgm` sobre a coluna normalizada, com a normalização feita **na escrita** (AD-16).

**Dado** que a medição falhe o limiar,
**Quando** a resposta é decidida,
**Então** o caminho é **índice antes de serviço** — nunca cache, nunca Elasticsearch (Adiado da espinha),
**E** a decisão e o número medido voltam para o `addendum.md` nesta mesma estória.

> **Deixa para depois:** a consulta de busca completa é da Épica 3. Aqui a consulta existe como sonda de medição, não como funcionalidade.

### Estória 1.5: Um Comprador entra e vê um Produto

**Como** Comprador semeado,
**quero** entrar com e-mail e senha e abrir a página de um Produto,
**para que** o cookie de Sessão prove que atravessa a casca do Next e o front fale com o Go de verdade.

**Condições de Aceite:**

**Dado** o Comprador vindo do Catálogo Semeado,
**Quando** ele autentica,
**Então** a resposta emite o cookie `azamon_sessao` — `HttpOnly`, `SameSite=Lax`, opaco de 256 bits — cujo valor é chave no Redis com TTL de 7 dias (AD-20),
**E** nada de conteúdo viaja no cookie,
**E** o cookie chega ao navegador **através do `rewrites()`**, fechando na prática a verificação 1 da estória 1.2.

**Dado** o armazenamento de credencial (NFR-5, AD-20),
**Quando** a senha é guardada e conferida,
**Então** é Argon2id com `m=19456`, `t=2`, `p=1` e sal de 16 bytes por usuário,
**E** nenhuma senha é persistida em claro, nem como hash de uma passagem.

**Dado** um Produto do Catálogo Semeado,
**Quando** o Comprador abre `GET /api/v1/produtos/<id>`,
**Então** a resposta traz nome, preço em centavos inteiros, imagem por URL relativa e o **nome do Vendedor**,
**E** a rota de detalhe é servida por `catalogo` (AD-16),
**E** a tela pode ser crua: a estória fecha quando o dado atravessa, não quando fica bonita.

> **Deixa para depois:** cadastro, encerramento de Sessão, bloqueio por tentativas, recuperação de senha e Endereços são da Épica 2. A Vitrine e a listagem são da Épica 3.

### Estória 1.6: Um Pedido nasce em `AGUARDANDO_PAGAMENTO`

**Como** Comprador autenticado,
**quero** confirmar a compra de um Produto e ver um Pedido nascer com número,
**para que** o sistema tenha pela primeira vez um compromisso registrado, e não apenas uma tela.

**Migração desta estória:** `pedido.pedido`, `pedido.item_pedido`, `pedido.transicao_status` e `catalogo.reserva_estoque` — esta última já **com o estado `ATIVA → LIBERADA | CONSOLIDADA` e o índice único parcial** do AD-5, porque separar o enum do índice em duas migrações é pior que criá-los juntos.

**Condições de Aceite:**

**Dado** que a transição é o único ponto de mutação do Pedido (AD-3),
**Quando** o código é inspecionado,
**Então** **não existe `UPDATE pedido SET status`** em lugar nenhum — só `pedido.Transicionar(...)`,
**E** `Transicionar` é um *compare-and-swap*: zero linhas afetadas devolve `ErrEstadoJaAvancado`,
**E** a transição acontece **primeiro**, o efeito sobre Estoque depois, na mesma transação.

**Dado** que toda transição é registrada (AD-15, NFR-9),
**Quando** o Pedido nasce,
**Então** `pedido.transicao_status` grava estado anterior, estado novo, quem provocou e quando,
**E** o Pedido recebe um `numero` no formato `AZ-<ano>-<6 dígitos>`, de sequência do Postgres por ano, **separado do `uuid`**.

**Dado** que o Pedido é imutável depois de criado,
**Quando** o Item de Pedido é gravado,
**Então** ele congela nome, `preco_praticado_centavos` e **Vendedor** do Produto,
**E** `pedido.total_centavos` é **coluna com `CHECK`**, `int64`, nunca derivação em tempo de leitura,
**E** não existe divisão em nenhum ponto do caminho monetário (AD-9).

**Dado** que o Catálogo é dono do Estoque (AD-5),
**Quando** a Reserva de Estoque é criada,
**Então** a consulta faz `SELECT … ORDER BY id FOR UPDATE` **primeiro** e soma as reservas ativas **depois** — dois comandos, nesta ordem,
**E** a Reserva nasce `ATIVA`, com índice único parcial,
**E** o caso de uso corre em **uma** transação, com `tx pgx.Tx` passado como **parâmetro nomeado**, nunca no `context` (AD-4),
**E** nenhuma chamada de rede acontece dentro da transação aberta.

> **Deixa para depois:** o esqueleto vai da Página de Produto direto ao Pedido — **sem passar pelo Carrinho**. A Épica 5 roteia o checkout pelo Carrinho, pelo Endereço e pelo Frete. A atomicidade tudo-ou-nada da FR-24 sobre vários Itens e as recusas nomeadas também são da Épica 5.

### Estória 1.7: O Provedor Simulado confirma por webhook e o Pedido vai a `PAGO`

**Como** Comprador,
**quero** que o resultado do pagamento chegue depois da minha confirmação, por conta própria,
**para que** o caminho do gateway real esteja construído desde o primeiro dia, em vez de ser reescrita depois.

**Migração desta estória:** `pagamento.tentativa_pagamento` e `pagamento.confirmacao_recebida`, esta com a **restrição única sobre a chave de idempotência**.

**Condições de Aceite:**

**Dado** que o pagamento confirma fora da requisição do checkout,
**Quando** o Comprador confirma,
**Então** a resposta **não espera** o resultado — ele vai para a tela do Pedido em processamento,
**E** nenhum dado de cartão é pedido, gerado ou guardado em ponto nenhum.

**Dado** a porta de pagamento (AD-8, NFR-3),
**Quando** a interface é inspecionada,
**Então** ela tem **duas operações e mais nada** — iniciar Tentativa e receber confirmação,
**E** o Provedor Simulado decide pelos **centavos do total**, conforme o §7.1, **só na primeira Tentativa**,
**E** não existe superfície de operador nem alternância em tempo de execução.

**Dado** a idempotência estrutural (AD-7, NFR-12),
**Quando** a confirmação chega em `POST /api/v1/webhooks/pagamento`,
**Então** ela é gravada com **restrição única** sobre a chave de idempotência,
**E** a mesma confirmação recebida duas vezes produz **um único efeito**,
**E** `pagamento` **não conhece `pedido`**: a varredura lê o *inbox* e aplica,
**E** só é aplicada se pertence à Tentativa corrente **e** o Pedido está em `AGUARDANDO_PAGAMENTO`.

**Dado** que emitir a confirmação simulada é comportamento do Provedor (AD-6),
**Quando** o código é inspecionado,
**Então** a emissão mora em `internal/pagamento`, **não** em `internal/pedido`,
**E** é disparada pela varredura, **depois** do commit.

**Dado** um total cujos centavos caem na faixa `,00`–`,89`,
**Quando** a confirmação é aplicada,
**Então** o Pedido transita para `PAGO` e a Reserva de Estoque **permanece**,
**E** a tela reflete o novo Status **sem recarregar**, por consulta em intervalo de 3 s enquanto `AGUARDANDO_PAGAMENTO` (AD-18, UX-DR13),
**E** todo instante exibido é absoluto, em RFC 3339, vindo do servidor.

> **Deixa para depois:** recusa, nova Tentativa, teto de 3, expiração por tempo esgotado e aprovação sobre Pedido cancelado são da Épica 5. Aqui só o caminho aprovado atravessa.

### Estória 1.8: A varredura leva o Pedido até `ENTREGUE`

**Como** integrante do time,
**quero** que o Pedido avance sozinho de `PAGO` até `ENTREGUE` derivando do histórico,
**para que** reiniciar o contêiner não congele Pedido nenhum — o defeito que mais provavelmente quebraria a apresentação.

**Condições de Aceite:**

**Dado** que a varredura é o único motor do tempo (AD-6),
**Quando** o código é inspecionado,
**Então** **não existe `time.Timer` em módulo nenhum**,
**E** um tique de 1 s chama, **nesta ordem**: aplicar → expirar → simular → emitir,
**E** cada Pedido é processado em sua própria transação, com `FOR UPDATE SKIP LOCKED`,
**E** `ErrEstadoJaAvancado` é desfecho **esperado**, não erro registrado como falha.

**Dado** que nenhum estado é alcançado por tempo em memória,
**Quando** a varredura decide avançar um Pedido,
**Então** a decisão deriva de "está neste estado desde quando", lido de `pedido.transicao_status`,
**E** o intervalo entre etapas vem de configuração — 30 s em demonstração, 24 h padrão — e a simulação é desligável por interruptor (§7.1, AD-13),
**E** a simulação nunca move um Pedido `CANCELADO` nem um `AGUARDANDO_PAGAMENTO`.

**Dado** um Pedido em `SEPARANDO`,
**Quando** a varredura o move para `ENVIADO`,
**Então** a Reserva de Estoque é **consolidada** — o Estoque total baixa e a Reserva encerra,
**E** **nenhuma outra transição altera o total**.

**Dado** um Pedido em `PAGO` e o contêiner reiniciado no meio do avanço,
**Quando** o sistema volta,
**Então** o Pedido **retoma de onde parou** e chega a `ENTREGUE` sem intervenção,
**E** o histórico permanece consultável e imutável.

> **Deixa para depois:** a máquina de estados completa — as nove transições declaradas, as três recusas distintas e a recusa nomeando as transições permitidas — é a estória 5.1. O painel do Administrador e o cancelamento são da Épica 6.

### Estória 1.9: Clone limpo, rede desconectada, README de 15 minutos

**Como** integrante que nunca rodou o projeto,
**quero** chegar ao sistema funcionando em 15 minutos seguindo apenas o README,
**para que** a SM-3 seja verdade agora, quando ainda dá para consertar, e não na véspera.

**Condições de Aceite:**

**Dado** um clone limpo e alguém que não escreveu o código,
**Quando** essa pessoa segue **apenas** o README,
**Então** ela chega ao sistema funcionando em **≤ 15 minutos** (NFR-1, SM-3).

**Dado** a rede desconectada (NFR-15),
**Quando** o esqueleto é percorrido inteiro — entrar, ver o Produto, criar o Pedido, vê-lo em `PAGO` e chegar a `ENTREGUE`,
**Então** nenhuma imagem, fonte, folha de estilo, script ou dependência é buscada em rede externa em tempo de execução,
**E** o passo de CI do AD-12 falha o build ao encontrar `fonts.googleapis`, `@import url(http`, `//cdn.` ou `next/font/google`.

**Dado** as fronteiras de módulo (NFR-2, AD-1),
**Quando** `internal/fronteira_test.go` roda `go list -deps -json`,
**Então** o teste falha se qualquer módulo alcançar outro por caminho diferente de `internal/<módulo>/<módulo>.go`,
**E** o grafo de dependência verificado é **exaustivo**, não uma amostra.

**Dado** que o addendum é mantido vivo desde esta épica (SM-7),
**Quando** a épica fecha,
**Então** o `addendum.md` registra o desfecho de cada uma das cinco verificações do passo 0,
**E** toda decisão de arquitetura tomada durante a construção está registrada nele, não apenas no código.

> **Verificação 5 não é estória de desenvolvimento.** O enunciado ou rubrica do professor está fora do controle do time — dono: **Sung**, bloqueio 2 do `HANDOFF.md`. Trava quatro suposições do §16 do PRD, mas **nenhuma toca o esqueleto vertical**, então não bloqueia esta épica.

---

## Épica 2: Conta, Identidade e Endereços

Um visitante cria conta, abre e encerra Sessão, redefine senha e gerencia seus Endereços — e o sistema
passa a saber quem está falando e a negar recurso de outro dono, inclusive contra chamada direta à API.
É o passo 1 do addendum §8: tudo depois daqui depende de saber quem está do outro lado.

**FRs:** FR-1 a FR-5 · **NFRs:** NFR-5, NFR-6 · **ADs:** AD-11, AD-13, AD-20

### Estória 2.1: Cadastro de Comprador

**Como** visitante,
**quero** criar uma conta com e-mail e senha,
**para que** eu possa montar um Carrinho e fazer Pedidos.

**Condições de Aceite:**

**Dado** um e-mail que ninguém usou,
**Quando** o visitante se cadastra,
**Então** o e-mail é **normalizado antes** da verificação de unicidade,
**E** a senha tem no mínimo 8 caracteres e é guardada em Argon2id (`m=19456`, `t=2`, `p=1`, sal de 16 bytes), nunca em claro,
**E** ao final do cadastro ele já está **autenticado**, sem passar pelo Login.

**Dado** um e-mail que já existe,
**Quando** o cadastro é submetido,
**Então** o erro aparece **em linha no campo** — "Já existe uma conta com este e-mail." — com link para o Login,
**E** nunca como resumo genérico no topo (UX-DR20(c)),
**E** o erro é associado por `aria-describedby` e o foco vai para o campo ao submeter (UX-DR14).

**Dado** os limites do NFR-14,
**Quando** qualquer campo é enviado,
**Então** o limite máximo é declarado e verificado **no servidor**, na decodificação do DTO em `api/`, lendo da struct de configuração — nunca só no navegador.

### Estória 2.2: Autenticação, encerramento de Sessão e bloqueio por tentativas

**Como** Comprador,
**quero** entrar e sair da minha conta com segurança,
**para que** ninguém descubra pelas mensagens do sistema se um e-mail tem conta.

**Condições de Aceite:**

**Dado** uma credencial inválida,
**Quando** o Login é submetido,
**Então** a mensagem é **exatamente a mesma** para e-mail inexistente e para senha errada: "E-mail ou senha incorretos." (FR-2),
**E** essa é a **única exceção declarada** à regra de nomear o motivo (UX-DR16).

**Dado** cinco tentativas falhas,
**Quando** a sexta chega,
**Então** o par **(e-mail, origem)** fica bloqueado por 15 minutos — **exista a conta ou não** —, com contador no Redis por TTL,
**E** a mensagem é "Muitas tentativas. Tente novamente em 15 minutos.", sem revelar se a conta existe,
**E** os dois limiares vêm da configuração (§7.1, AD-13).

**Dado** uma Sessão aberta,
**Quando** o Comprador encerra a Sessão,
**Então** ela é invalidada **no servidor** — a chave no Redis deixa de existir —, não apenas o cookie apagado,
**E** a Sessão expira sozinha por inatividade em 7 dias.

**Dado** uma Sessão expirada em qualquer tela,
**Quando** o Comprador age,
**Então** ele é levado ao Login **preservando o destino**, e volta para onde estava depois de entrar.

### Estória 2.3: Recuperação de senha

**Como** Comprador que esqueceu a senha,
**quero** redefini-la por um token de uso único,
**para que** eu recupere o acesso sem que ninguém descubra, pela resposta, se o e-mail tem conta.

**Condições de Aceite:**

**Dado** uma solicitação de redefinição,
**Quando** ela é submetida,
**Então** a resposta é **idêntica** exista ou não a conta — estado "solicitação enviada", com texto neutro (UX-DR20(a)),
**E** havendo conta, um token de uso único válido por 30 minutos é emitido **para o log estruturado** (`docker compose logs`), porque não existe serviço de e-mail (AD-20),
**E** uma nova solicitação **invalida os tokens anteriores**.

**Dado** um token expirado, já usado ou inexistente,
**Quando** ele é apresentado,
**Então** a tela mostra o estado "token inválido ou expirado" com caminho para solicitar outro (UX-DR20(a)).

**Dado** um token válido,
**Quando** a senha é redefinida,
**Então** **todas as Sessões ativas do Comprador são encerradas**,
**E** a nova senha passa pelas mesmas regras da estória 2.1.

> *FR-3 é o **1º item da ordem de corte** do §6.3. Não aparece em nenhum roteiro do §12 — as contas da demonstração vêm do Catálogo Semeado.*

### Estória 2.4: Autorização por dono do recurso e separação de papéis

**Como** Comprador,
**quero** que meus Pedidos, Carrinho e Endereços sejam inacessíveis a qualquer outra pessoa,
**para que** esconder o link na interface não seja a única proteção.

**Condições de Aceite:**

**Dado** o mecanismo do AD-11,
**Quando** um recurso de Comprador é carregado,
**Então** `api/` resolve a Sessão e injeta a identidade, e **a verificação de posse acontece dentro do módulo dono, na mesma consulta que carrega o dado** (`WHERE id = $1 AND comprador_id = $2`) — FR-4,
**E** nunca como `if` depois da leitura.

**Dado** o identificador de um recurso de **outro** Comprador,
**Quando** a API é chamada diretamente, sem passar por tela,
**Então** a resposta é **o mesmo erro de inexistente** — não existe tela de "acesso negado" com dica,
**E** um teste automatizado faz exatamente isso para **Endereço** e espera negação (NFR-6); Carrinho e Pedido entram no mesmo teste nas Épicas 4 e 6.

**Dado** os dois papéis,
**Quando** a área administrativa é acessada,
**Então** `Administrador` é verificado em `api/` **por prefixo de rota** (`/api/v1/admin/`),
**E** `Comprador` e `Administrador` são tabelas separadas em `identidade` — sem coluna de papel, nenhuma linha de Comprador vira Administrador por engano,
**E** a área administrativa **não aparece** para quem não é Administrador, nem como link desabilitado (UX-DR9).

### Estória 2.5: Endereços do Comprador

**Como** Comprador,
**quero** cadastrar, editar, remover e escolher meus Endereços,
**para que** o checkout não me obrigue a redigitar tudo a cada Pedido.

**Migração desta estória:** `identidade.endereco`.

**Condições de Aceite:**

**Dado** um Comprador autenticado,
**Quando** ele gerencia Endereços,
**Então** pode cadastrar, editar, remover e escolher, com o CEP validado,
**E** remover um Endereço **não altera nenhum Pedido existente** — o Pedido congelou o Endereço na criação (AD-3),
**E** erro de campo aparece **em linha**, com o limite nomeado (UX-DR20(c)).

**Dado** um Comprador sem nenhum Endereço,
**Quando** ele abre "Meus endereços",
**Então** vê "Nenhum Endereço cadastrado." com ação única: cadastrar,
**E** quando o fluxo o exigir, ele é levado ao cadastro e **volta ao ponto em que parou**.

### Estória 2.6: Menu da conta e o retorno ao ponto de partida

**Como** Comprador,
**quero** alcançar minha conta, meus pedidos e meus endereços de qualquer tela,
**para que** eu nunca precise voltar à raiz para chegar ao que é meu.

**Condições de Aceite:**

**Dado** a UX-DR8 e a UX-DR9,
**Quando** qualquer tela de Comprador é renderizada,
**Então** o Menu da conta está presente, dando acesso a Meus pedidos, Meus endereços, Perfil e encerrar Sessão,
**E** o Perfil traz o mínimo declarado: e-mail em leitura, trocar senha pela FR-3 e sair — nenhuma FR pede mais,
**E** a ordem de tabulação é igual à de leitura e o foco é visível a 3:1 (UX-DR14),
**E** o fluxo de compra é **inteiramente operável por teclado** — a verificação mínima é percorrer a UJ-1 sem tocar no mouse (NFR-11).

**Dado** um visitante que tenta uma ação de Comprador,
**Quando** ele é levado ao Login,
**Então** a origem é guardada e ele **volta exatamente para onde estava** ao autenticar.

> **Deixa para depois:** a busca global e a Faixa de Categorias da barra superior são da estória 3.10, que completa a UX-DR9. Até lá, as telas desta épica herdam a base visual da estória 1.2.

---

## Épica 3: Catálogo, Estoque e Busca

O Administrador abastece a loja com Vendedores, Categorias, Produtos e Estoque; qualquer visitante vê a
Vitrine, encontra Produtos por termo, filtro e ordem, e abre a Página de Produto com o nome do Vendedor
e a disponibilidade real. Produto criado aparece na Vitrine e na busca **sem reiniciar** a aplicação.

**FRs:** FR-6 a FR-15 · **NFRs:** NFR-4 · **ADs:** AD-2, AD-5, AD-9, AD-16, AD-18, AD-19
**UX:** UX-DR12 — onze dos 24 estados nascem nesta épica, cada um como AC · UX-DR19 — `mockups/resultados-busca.html` e `mockups/pagina-de-produto.html` são referência de composição; **as duas espinhas vencem em conflito**

### Estória 3.1: Gestão de Vendedores

**Como** Administrador,
**quero** criar, editar e desativar Vendedores,
**para que** o Catálogo tenha a espinha de marketplace que a página de Produto exibe.

**Condições de Aceite:**

**Dado** a área administrativa,
**Quando** um Vendedor é desativado,
**Então** os Produtos dele deixam de aparecer na Vitrine, na busca e na Página de Produto,
**E** **nenhum Pedido existente é apagado ou alterado**.

**Dado** um Vendedor que possui Produtos,
**Quando** o Administrador tenta removê-lo,
**Então** a operação é recusada e o `Alert` **oferece desativar no lugar**, com a ação alternativa dentro do próprio `Alert`,
**E** a lista vazia mostra "Nenhum Vendedor cadastrado." com ação única: criar.

> *FR-8 é o **3º item da ordem de corte**. As Categorias e Vendedores vêm semeados; o Roteiro C perde um passo e continua fazendo sentido.*

### Estória 3.2: Gestão de Categorias

**Como** Administrador,
**quero** criar e renomear Categorias,
**para que** os Produtos sejam agrupáveis e filtráveis.

**Condições de Aceite:**

**Dado** uma Categoria com Produtos vinculados,
**Quando** o Administrador tenta removê-la,
**Então** a recusa **nomeia quantos Produtos estão vinculados**,
**E** nome duplicado é recusado **em linha no campo**.

**Dado** o modelo de dados,
**Quando** a Categoria é criada,
**Então** `categoria_pai_id` existe e permanece **nulo e não exposto** — a blindagem barata da Suposição 4 (addendum §6): hierarquia depois é preencher o campo, não migrar dados.

> *FR-10 é o **2º item da ordem de corte**.*

### Estória 3.3: Gestão de Produtos

**Como** Administrador,
**quero** criar, editar e desativar Produtos com Vendedor e Categoria,
**para que** eu abasteça a loja e o resultado apareça na hora.

**Condições de Aceite:**

**Dado** um Produto novo,
**Quando** ele é criado com Vendedor e Categoria,
**Então** ele **aparece na Vitrine e na busca sem reiniciar a aplicação** (Roteiro C, passo 1),
**E** preço ≤ 0 e Estoque negativo são recusados (FR-9),
**E** editar o preço **não altera nenhum Item de Pedido existente** — o Pedido congelou o preço praticado.

**Dado** os limites do §7.1,
**Quando** um campo excede,
**Então** o erro aparece em linha **nomeando o limite** — "O nome do Produto tem no máximo 200 caracteres", descrição em 4.000 —, nunca como resumo genérico no topo,
**E** os limites são verificados no servidor, lidos da struct de configuração (NFR-14, AD-13).

**Dado** uma imagem ausente ou com caminho errado,
**Quando** o Produto é exibido em qualquer superfície,
**Então** aparece um bloco neutro na proporção declarada com o nome do Produto centralizado,
**E** **nunca o ícone de imagem quebrada do navegador** — um caminho errado na semente não pode abrir um buraco na primeira tela da apresentação.

### Estória 3.4: Estoque disponível, guarda de Reservas e o predicado de visibilidade

**Como** Administrador,
**quero** ajustar o Estoque total sem nunca deixar o disponível negativo,
**para que** o sistema não prometa unidade que já está comprometida em Pedido aberto.

**Condições de Aceite:**

**Dado** o AD-5,
**Quando** a disponibilidade é calculada,
**Então** ela é **derivada** — `estoque_total − Σ reservas ativas` —, calculada na consulta e **nunca armazenada**,
**E** `catalogo.Disponivel(ctx, produtoIDs) map[uuid]int` é **em lote**, porque toda tela precisa dele para vários Produtos,
**E** o ajuste do Administrador tem efeito imediato e o disponível nunca fica negativo.

**Dado** um Produto com 4 unidades comprometidas em Pedidos abertos,
**Quando** o Administrador tenta baixar o Estoque total para 2,
**Então** a operação é recusada **informando a contagem em tempo real**: "Há 4 unidades comprometidas em Pedidos abertos. O Estoque total não pode ficar abaixo disso." (FR-11),
**E** a consulta segue a mesma ordem obrigatória do AD-5: `SELECT … ORDER BY id FOR UPDATE` **primeiro**, soma das reservas **depois**.

**Dado** o AD-19,
**Quando** "Produto visível" é avaliado,
**Então** o predicado — Produto ativo **e** Vendedor ativo — é definido **uma única vez** em `catalogo` e exposto por `catalogo.Visiveis`,
**E** **nenhum outro lugar escreve `WHERE ativo = true`**,
**E** `Disponivel` devolve `0` para Produto inexistente ou invisível — nunca omite a chave, nunca devolve erro,
**E** `Reservar` verifica visibilidade **na mesma consulta `FOR UPDATE`** em que verifica Estoque, falhando com o mesmo `catalogo.ErrEstoqueInsuficiente`.

**Dado** a natureza da Reserva de Estoque,
**Quando** o schema é inspecionado,
**Então** ela tem estado `ATIVA → LIBERADA | CONSOLIDADA` com **índice único parcial sobre `(pedido_id, produto_id) WHERE estado = 'ATIVA'`**,
**E** `Liberar` sem reserva ativa devolve `nil`, `Consolidar` sobre reserva já consolidada é *no-op*, `Reservar` com lista vazia é *no-op* — toda mutação é idempotente e declara isso na assinatura,
**E** **`Liberar` nunca escreve `estoque_total`; só `Consolidar` escreve.**

### Estória 3.5: A Vitrine e o envelope de listagem

**Como** visitante,
**quero** ver a grade de Produtos com preço e disponibilidade logo na primeira tela,
**para que** eu saiba imediatamente o que a loja tem.

**Condições de Aceite:**

**Dado** o AD-16,
**Quando** as rotas são registradas,
**Então** `GET /api/v1/produtos` é servido **exclusivamente por `busca`**, com ou sem `termo` — "listar sem termo" é "buscar com termo vazio", nunca uma rota separada,
**E** `catalogo` **não registra handler de listagem nem de paginação**; expõe `ObterVisivel(id)` e `Visiveis`/`Disponivel` por chamada Go,
**E** dois donos do mesmo padrão no `ServeMux` fazem o binário **entrar em pânico no arranque**.

**Dado** a VIEW que `busca` lê,
**Quando** ela é definida,
**Então** `busca` lê `catalogo` **por VIEW, nunca por tabela**, sobre o mesmo predicado de visibilidade do AD-19,
**E** a VIEW expõe **`estoque_disponivel` derivado, nunca `estoque_total`** — senão a Vitrine promete unidade que o Carrinho recusa,
**E** `busca` **não tem tabelas próprias** (AD-2).

**Dado** a paginação (FR-15) e o AD-18,
**Quando** qualquer listagem responde,
**Então** ela usa o envelope único `{ "itens": [...], "pagina": 1, "por_pagina": 20, "total": 137 }`,
**E** são 20 itens por página, com teto de 60 aceito do cliente e aplicado no servidor,
**E** página além do total é **vazio tratado, não erro**,
**E** o desempate da ordenação **sempre termina em `id` ascendente** — sem isso a paginação repete e some com linhas.

**Dado** um Catálogo Semeado vazio,
**Quando** a Vitrine carrega,
**Então** aparece "O Catálogo ainda não tem Produtos." — sem ação para o Público, e com caminho para cadastrar se houver Sessão de Administrador,
**E** o carregamento usa `Skeleton` **em grade, com a mesma forma dos cartões**, nunca spinner centralizado,
**E** o Estoque zero aparece **indisponível e não adicionável**.

**Dado** a UX-DR7 e a UX-DR15,
**Quando** a grade é renderizada,
**Então** o verde `{colors.available}` é usado **só** para Estoque disponível,
**E** a grade vai de 1 a 4 colunas nas quatro faixas declaradas (360–639 · 640–1023 · 1024–1439 · ≥1440), **sem rolagem horizontal e sem elemento inacessível** (NFR-10).

### Estória 3.6: Página de Produto

**Como** visitante,
**quero** abrir um Produto e ver quem o vende e se está disponível,
**para que** eu decida a compra com a informação que define o marketplace.

**Condições de Aceite:**

**Dado** um Produto visível,
**Quando** a página abre,
**Então** ela mostra nome, descrição, imagem, preço, Categoria, **nome do Vendedor** e disponibilidade (FR-7),
**E** a rota é `GET /api/v1/produtos/<id>`, servida por `catalogo` (AD-16),
**E** a Caixa de compra fica à direita a partir de 1024px e o preço usa os papéis `preco` e `preco-centavos` da UX-DR3.

**Dado** um id inexistente, um Produto desativado ou um Produto de Vendedor desativado,
**Quando** a página é pedida,
**Então** a resposta é **"não encontrado", nunca erro de servidor**,
**E** a tela mostra "Este Produto não está disponível." com ação única: voltar à Vitrine,
**E** **a interface não distingue os três casos**.

**Dado** um visitante sem Sessão,
**Quando** ele tenta adicionar ao Carrinho,
**Então** é levado ao Login **guardando o Produto de origem e a quantidade**, e volta à Página de Produto ao autenticar.

### Estória 3.7: Busca por texto

**Como** visitante com pressa,
**quero** digitar "cafe" e encontrar "café",
**para que** um acento esquecido no ônibus não me deixe sem resultado.

**Condições de Aceite:**

**Dado** um termo livre,
**Quando** a busca corre sobre nome e descrição,
**Então** ela ignora caixa e acento — "cafe" encontra "café" (FR-12),
**E** a normalização acontece **na escrita**, em coluna dedicada preenchida pelo Go, com índice GIN `pg_trgm` (AD-16),
**E** não há relevância por pontuação, correção ortográfica nem sugestão enquanto digita (§5 do PRD).

**Dado** um termo com caracteres especiais,
**Quando** ele é submetido,
**Então** `%` e `_` são tratados como **texto literal** — "50%" procura o texto, não devolve o catálogo inteiro,
**E** o termo tem teto de **100 caracteres**, verificado no servidor (§7.1, NFR-14).

**Dado** um termo sem resultado,
**Quando** a busca responde,
**Então** aparece "Nenhum Produto encontrado para '{termo}'." com os filtros ativos como chips e um botão "Limpar filtros",
**E** **nunca uma página em branco**,
**E** Produto e Vendedor desativados **não aparecem**,
**E** a contagem de resultados é anunciada por `aria-live="polite"` (UX-DR14).

### Estória 3.8: Filtros de Categoria e faixa de preço

**Como** visitante,
**quero** restringir por Categoria e por faixa de preço e ver o que está filtrando,
**para que** eu possa compartilhar ou recarregar a URL e ver exatamente a mesma listagem.

**Condições de Aceite:**

**Dado** filtros e termo,
**Quando** são aplicados,
**Então** Categoria e faixa de preço são **combináveis entre si e com o termo** (FR-13),
**E** os filtros ativos ficam **visíveis como chips e removíveis individualmente**,
**E** todo o estado vive na URL (`?termo=&categoria=&pagina=`): recarregar ou compartilhar reproduz a mesma listagem, e voltar no navegador funciona.

**Dado** um preço mínimo maior que o máximo,
**Quando** o filtro é aplicado,
**Então** a mensagem aparece **em linha no filtro** — "O preço mínimo não pode ser maior que o máximo." —,
**E** **a listagem anterior permanece**; nunca uma lista incoerente.

**Dado** a faixa 640–1023px,
**Quando** os filtros são abertos,
**Então** eles aparecem em `Sheet`; de 1024px em diante ficam fixos à esquerda (UX-DR15).

### Estória 3.9: Ordenação

**Como** visitante,
**quero** ordenar por preço crescente, preço decrescente e mais recentes,
**para que** eu chegue ao mais barato sem percorrer a lista inteira.

**Condições de Aceite:**

**Dado** as três ordenações declaradas,
**Quando** uma é escolhida,
**Então** ela é **preservada ao paginar e ao filtrar**,
**E** o desempate é estável e **termina sempre em `id` ascendente** — a paginação **nunca repete nem pula** um Produto,
**E** a ordenação vive na URL como o resto do estado de listagem.

> *FR-14 é o **5º e último item da ordem de corte**. Os filtros sozinhos sustentam o Roteiro A.*

### Estória 3.10: Chrome da loja — barra superior, busca global e Faixa de Categorias

**Como** visitante ou Comprador,
**quero** a busca ao alcance em qualquer tela,
**para que** o caminho da UJ-1 comece de onde eu estiver.

**Condições de Aceite:**

**Dado** a UX-DR9,
**Quando** qualquer tela **pública ou de Comprador** é renderizada,
**Então** a Barra superior, a busca global, a Faixa de Categorias e o Menu da conta estão presentes,
**E** a busca global é o **único elemento que não respeita o ritmo folgado** da UX-DR6,
**E** `Enter` submete a busca de qualquer tela (UX-DR18).

**Dado** a área administrativa,
**Quando** ela é renderizada,
**Então** tem chrome próprio em `chrome-muted`, **sem busca e sem Carrinho**,
**E** a loja é **mobile-first** e o painel administrativo **desktop-first** (UX-DR15).

**Dado** a UX-DR5,
**Quando** os raios são aplicados,
**Então** `rounded.full` aparece em **exatamente quatro usos**: busca global, botões de ação, selos de Status do Pedido e chips de filtro ativo.

---

## Épica 4: Carrinho

Um Comprador autenticado monta um Carrinho que sobrevive à Sessão e ajusta suas quantidades — e é
avisado **antes** de qualquer surpresa, porque o Carrinho é intenção, não compromisso: não congela preço
e não reserva Estoque.

**FRs:** FR-16 a FR-19 · **ADs:** AD-5, AD-9, AD-11, AD-17, AD-19 · **🔒 Nunca cortar:** FR-19
**UX:** UX-DR12 · UX-DR17 · UX-DR18

### Estória 4.1: Carrinho exclusivo de Comprador autenticado

**Como** visitante que encontrou o que queria,
**quero** ser levado ao Login e voltar com o Produto já no Carrinho,
**para que** autenticar não me custe refazer o caminho.

**Migração desta estória:** `carrinho.carrinho` e `carrinho.item_carrinho`, com `preco_visto_centavos` no Item.

**Condições de Aceite:**

**Dado** um visitante sem Sessão,
**Quando** ele tenta adicionar um Produto,
**Então** é levado ao Login **guardando o Produto de origem e a quantidade**,
**E** ao autenticar volta à **Página de Produto de origem** com o Item de Carrinho **já criado**.

**Dado** a decisão declarada do §6.2 do PRD,
**Quando** o sistema é inspecionado,
**Então** **não existe Carrinho anônimo e não existe fusão no login** — nenhum estado de visitante com ciclo de vida próprio,
**E** o Carrinho vive no PostgreSQL, não no Redis: sobrevive a reinício e não expira sozinho.

**Dado** o identificador do Carrinho de outro Comprador,
**Quando** a API é chamada diretamente,
**Então** a resposta é o mesmo erro de inexistente, verificado na mesma consulta que carrega o dado (AD-11),
**E** o teste automatizado do NFR-6 ganha o **Carrinho** como segundo recurso, ao lado do Endereço da estória 2.4.

### Estória 4.2: Adicionar Produto ao Carrinho

**Como** Comprador,
**quero** adicionar um Produto e somar à quantidade se ele já estiver lá,
**para que** o Carrinho nunca tenha duas linhas do mesmo Produto.

**Condições de Aceite:**

**Dado** um Produto já presente no Carrinho,
**Quando** ele é adicionado de novo,
**Então** a quantidade **soma ao Item de Carrinho existente**, sem criar linha nova (FR-17).

**Dado** uma quantidade acima do disponível,
**Quando** a adição é tentada,
**Então** ela é recusada **informando o disponível**,
**E** Produto indisponível ou invisível pelo predicado do AD-19 **não entra**,
**E** a quantidade respeita 1 ≤ q ≤ 10, com o teto vindo da configuração (§7.1, NFR-14).

**Dado** que `Disponivel` é conselho, nunca decisão (AD-5),
**Quando** o número exibido diverge do aceito,
**Então** a divergência é **projetada**, não defeito: quem decide é o `Reservar` sob bloqueio, e a recusa é o caminho declarado.

### Estória 4.3: Alterar quantidade, remover Item e esvaziar o Carrinho

**Como** Comprador,
**quero** ajustar as quantidades e limpar o Carrinho numa ação,
**para que** eu chegue ao checkout com exatamente o que decidi levar.

**Condições de Aceite:**

**Dado** um Item de Carrinho,
**Quando** a quantidade vai a zero,
**Então** o Item é **removido**,
**E** o total é recalculado a cada alteração, **refletindo os preços atuais do Catálogo** (FR-18),
**E** a atualização otimista é permitida **apenas** na quantidade de Item de Carrinho (UX-DR18).

**Dado** um Carrinho com Itens,
**Quando** o Comprador escolhe esvaziar,
**Então** a ação acontece **numa só operação, com confirmação** em `Dialog` de primeiro nível,
**E** `Esc` fecha o `Dialog` (UX-DR18).

**Dado** um Carrinho sem Itens,
**Quando** a tela abre,
**Então** aparece "Seu Carrinho está vazio." com ação única: voltar à Vitrine.

### Estória 4.4: Revalidação do Carrinho na abertura 🔒

**Como** Comprador,
**quero** saber que um preço mudou ou que uma unidade acabou **antes** de avançar,
**para que** eu nunca seja cobrado por um preço que não vi.

**Condições de Aceite:**

**Dado** um Item cujo preço mudou desde a última alteração,
**Quando** o Carrinho é aberto,
**Então** um `Alert` lista **o preço anterior e o atual por Item**, exigindo confirmação antes de seguir,
**E** `item_carrinho.preco_visto_centavos` guarda o preço no momento da última alteração, que é o que permite dizer "de X para Y".

**Dado** um Item que ficou indisponível ou cujo Vendedor foi desativado,
**Quando** o Carrinho é aberto,
**Então** o `Alert` **bloqueia o avanço** até o Item ser removido ou a quantidade ajustada, com **as duas ações dentro do próprio `Alert`**,
**E** confirmar **não** é bloquear: deixar passar levaria o Comprador ao botão irreversível para receber a recusa atômica da FR-24 no único ponto do fluxo que não admite erro,
**E** o desativado é sinalizado igual ao indisponível.

**Dado** o AD-17,
**Quando** a revalidação exibe a mudança,
**Então** `carrinho.Itens` é **leitura pura e nunca escreve `preco_visto_centavos`**,
**E** a tela mostra o preço atual ao lado do visto **sem gravar nada** — quem persiste a ciência da mudança é sempre o checkout, uma vez, quando o Comprador de fato avança.

> **Deixa para depois:** a outra metade da FR-19 — a revalidação na **entrada do checkout**, a chamada a `carrinho.ConfirmarPrecoVisto` e o recálculo do Frete ao cruzar o limiar de isenção — é a estória 5.5. `carrinho` **não calcula Frete** (AD-17).

### Estória 4.5: A tela do Carrinho não promete o que o sistema não faz

**Como** Comprador,
**quero** que a tela não sugira que o Carrinho guarda preço ou segura unidade,
**para que** a revalidação da estória 4.4 não pareça uma quebra de promessa.

**Condições de Aceite:**

**Dado** a UX-DR17,
**Quando** o Carrinho é renderizado,
**Então** a composição monetária é **subtotal e nada mais** — sem Frete, sem total estimado,
**E** a distância até o Frete grátis aparece em **valor absoluto, sem barra de progresso**, usando só o limiar da configuração,
**E** **a Reserva de Estoque nunca é nomeada para o Comprador** — é vocabulário legítimo só para o Administrador, na guarda da FR-11,
**E** o número exibido é **sempre o Estoque disponível**, nunca o total,
**E** nada na interface sugere que o Carrinho congela preço ou reserva Estoque.

**Dado** a UX-DR16 e o glossário do §3,
**Quando** qualquer texto é escrito,
**Então** os 20 termos são usados literalmente — "Produto", nunca "item"; "Pedido", nunca "compra",
**E** não há urgência fabricada.

**Dado** a UX-DR3 e o NFR-13,
**Quando** valores aparecem,
**Então** usam `font-variant-numeric: tabular-nums`, são inteiros de centavos formatados só no navegador com `Intl.NumberFormat('pt-BR')`,
**E** **o total exibido é a soma exata das parcelas exibidas**.

---

## Épica 5: Checkout e pagamento assíncrono

O Comprador escolhe o Endereço, vê o Frete e o total, confirma — e o Pedido nasce em
`AGUARDANDO_PAGAMENTO` com Reserva de Estoque atômica sobre todos os itens. O resultado do pagamento
chega **depois**; recusa, nova tentativa e expiração são caminhos de primeira classe, não erros.
É o que separa maquete de sistema.

**FRs:** FR-28 *(primeiro, com testes)*, FR-20 a FR-27, FR-34 · **NFRs:** NFR-3, NFR-7, NFR-12
**ADs:** AD-3 a AD-9, AD-17, AD-18 · **🔒 Nunca cortar:** FR-24, FR-34 e o teste do NFR-7
**UX:** UX-DR12 — seis dos oito estados de maior consequência vivem aqui, porque nascem de FR e não de tela

### Estória 5.1: A máquina de estados do Pedido, completa e com testes

**Como** integrante do time,
**quero** a máquina de estados fechada e testada antes de qualquer tela que a consuma,
**para que** o risco nº 1 do §14 do PRD — "a máquina de estados atrasa tudo" — seja pago agora.

**Condições de Aceite:**

**Dado** a tabela de transições do AD-3,
**Quando** a máquina é implementada,
**Então** ela contém **exatamente as nove transições declaradas** e nenhuma outra — a tabela é exaustiva e obrigatória,
**E** nenhuma acontece sem o efeito da terceira coluna, **na mesma transação** (AD-4),
**E** o conteúdo do Pedido — Itens, preço praticado, Endereço, Frete, total — é escrito **uma vez, na criação, e nunca mais**.

**Dado** que `Transicionar` é um *compare-and-swap*,
**Quando** duas chamadas concorrem pelo mesmo Pedido,
**Então** é literalmente `UPDATE pedido SET status = $novo WHERE id = $1 AND status = $esperado`,
**E** **zero linhas afetadas é `ErrEstadoJaAvancado`** — é assim que a corrida entre Administrador e simulação se resolve sem bloqueio explícito,
**E** a ordem dentro da transação é fixa: **transição primeiro, efeito sobre o Estoque depois**, para que o efeito só rode para quem ganhou o CAS e ninguém libere uma Reserva duas vezes.

**Dado** as **três recusas distintas** do AD-3,
**Quando** uma transição falha,
**Então** existem três erros nomeados, nunca um só: `ErrEstadoJaAvancado` (outro ator avançou), `ErrTransicaoInvalida` (fora da tabela, carregando **`dados.permitidas`** com as transições legais) e `ErrForaDaJanelaDeCancelamento`,
**E** o cancelamento **reclassifica**: um `ErrEstadoJaAvancado` cuja causa foi sair da janela é devolvido como `ErrForaDaJanelaDeCancelamento`, nunca como corrida,
**E** fundi-las apagaria o clímax da UJ-3 e da UJ-4.

**Dado** o histórico,
**Quando** qualquer transição ocorre,
**Então** `pedido.transicao_status` grava estado anterior, novo, ator, **`motivo`** e instante — é onde `TEMPO_ESGOTADO` (FR-34) e o motivo da recusa (FR-27) moram,
**E** o histórico é **consultável e imutável**,
**E** existe **uma função só**, `pedido.EstadoTerminal(s Status) bool`, onde terminal significa `ENTREGUE` ou `CANCELADO` **e mais nada** (AD-18).

**Dado** a estratégia de testes,
**Quando** a suíte roda,
**Então** a máquina é testada **em memória**, e `testcontainers-go` com PostgreSQL real é usado **onde a transação é o objeto do teste**,
**E** toda transição não declarada é recusada com erro explícito, **seja qual for o ator**,
**E** todo FR do fluxo crítico busca → Carrinho → checkout → Pedido ganha, a partir daqui, ao menos um teste que **falha se a consequência declarada quebrar** (NFR-8, SM-2).

### Estória 5.2: Seleção de Endereço no checkout

**Como** Comprador,
**quero** escolher entre meus Endereços ou cadastrar um novo no próprio checkout,
**para que** o primeiro Pedido não me obrigue a sair do fluxo.

**Condições de Aceite:**

**Dado** um Comprador com Endereços,
**Quando** o passo Endereço abre,
**Então** ele escolhe entre os seus ou cadastra um novo em `Dialog` de primeiro nível.

**Dado** um Comprador **sem nenhum** Endereço,
**Quando** o passo abre,
**Então** ele abre **direto no formulário de novo Endereço**, sem lista vazia intermediária — é o primeiro Pedido de todo Comprador.

**Dado** um Endereço trocado,
**Quando** o Comprador avança,
**Então** o Frete é **recalculado antes da confirmação**,
**E** de 360 a 639px o checkout mostra **um passo por tela** (UX-DR15).

### Estória 5.3: Cálculo do Frete

**Como** Comprador,
**quero** ver o Frete calculado e discriminado do subtotal,
**para que** eu saiba exatamente o que estou pagando por quê.

**Migração desta estória:** `pedido.faixa_frete` (faixa de CEP → região → `valor_centavos`), semeada junto com o Catálogo Semeado.

**Condições de Aceite:**

**Dado** o AD-17,
**Quando** o Frete é calculado,
**Então** a regra é a tabela **`pedido.faixa_frete`** (faixa de CEP → região → valor em centavos) mais uma **função pura** em `internal/pedido`,
**E** o limiar de isenção — R$ 299,00 — vem da configuração, nunca de `if` espalhado,
**E** **`carrinho` não calcula Frete**.

**Dado** o mesmo CEP e o mesmo valor,
**Quando** o cálculo roda de novo,
**Então** produz **sempre o mesmo Frete**,
**E** CEP fora de qualquer faixa cai na **região padrão** (FR-21),
**E** o Frete aparece **discriminado do subtotal**,
**E** não existe divisão no caminho monetário — o Frete **não é rateado por item** (AD-9).

### Estória 5.4: Revisão do Pedido

**Como** Comprador,
**quero** conferir tudo numa tela antes de confirmar,
**para que** o passo irreversível não tenha surpresa.

**Condições de Aceite:**

**Dado** a Revisão,
**Quando** ela é renderizada,
**Então** mostra cada item com **preço unitário e subtotal**, mais Endereço, Frete e total (FR-22),
**E** declara a forma de pagamento **sem coletar nenhum dado dela** — nada de cartão é pedido, gerado ou guardado,
**E** o botão Confirmar Pedido é a **única aparição do laranja `{colors.primary-strong}` no fluxo inteiro** (UX-DR7).

**Dado** o Comprador voltando ao Carrinho,
**Quando** ele retorna à Revisão,
**Então** o **Endereço escolhido não é perdido**,
**E** um Carrinho vazio **nunca chega à Revisão**.

**Dado** as Convenções de Consistência,
**Quando** o Comprador entra na Revisão,
**Então** o navegador **gera a `Idempotency-Key` neste momento** e a guarda em `sessionStorage`, para sobreviver ao recarregamento que ela existe para proteger,
**E** ela é enviada em `POST /api/v1/pedidos`.

### Estória 5.5: Entrada no checkout — revalidação e confirmação do preço visto 🔒

**Como** Comprador,
**quero** que a última checagem contra o Catálogo aconteça ao entrar no checkout,
**para que** nada mude entre a minha decisão e o meu compromisso sem eu ver.

**Condições de Aceite:**

**Dado** a segunda metade da FR-19,
**Quando** o Comprador entra no checkout,
**Então** a revalidação corre de novo: preço mudado **exige confirmação**, unidade indisponível **bloqueia o avanço**,
**E** se o subtotal mudou, **o Frete é recalculado — inclusive quando cruza o limiar de isenção**.

**Dado** o AD-17,
**Quando** a diferença já foi reportada,
**Então** **`pedido`** chama `carrinho.ConfirmarPrecoVisto` na transação de entrada no checkout, **depois** de reportar,
**E** essa é a **única função** que atualiza `preco_visto_centavos`, e **só `pedido` a chama** — os dois módulos escrevendo seria a mudança de preço que some antes de aparecer.

### Estória 5.6: Criação do Pedido com Reserva de Estoque atômica 🔒

**Como** Comprador,
**quero** que confirmar crie exatamente um Pedido, com todas as unidades garantidas ou nenhuma,
**para que** eu nunca receba metade do que pedi.

**Migração desta estória:** a chave de idempotência do Pedido — **chave e digest do corpo** — com restrição única (AD-7).

**Condições de Aceite:**

**Dado** a FR-24,
**Quando** o Pedido é criado,
**Então** a Reserva é gerada para cada Item **tudo ou nada**: se qualquer item faltar, **nenhum Pedido é criado**,
**E** o disponível cai **imediatamente** para os demais Compradores,
**E** a ordem de aquisição de bloqueio é global e única: primeiro a linha de `pedido`, depois as linhas de `catalogo.produto` **ordenadas por `id`** — sem isso, dois Pedidos com os mesmos dois Produtos em ordem inversa produzem *deadlock* (AD-4),
**E** `40P01` e `40001` são repetidos **uma vez, e só uma**, em `api/`.

**Dado** o AD-3,
**Quando** a transação de criação corre,
**Então** o Pedido nasce em `AGUARDANDO_PAGAMENTO` congelando preço praticado, Endereço, Frete e **o Vendedor de cada Item** (FR-23),
**E** `pedido.Criar` chama `carrinho.Esvaziar` **na mesma transação, depois de `catalogo.Reservar` suceder** — é o único ponto em que outro módulo escreve no Carrinho, e **não existe `DELETE /api/v1/carrinho` para esse fim**,
**E** cancelar um Pedido **não devolve nada ao Carrinho**.

**Dado** o AD-17,
**Quando** a criação recalcula o total,
**Então** ela recalcula **sob o mesmo bloqueio em que reserva** e recusa com **`TOTAL_DIVERGENTE`** se o total mudou desde a Revisão,
**E** `total_centavos` é coluna com `CHECK (total_centavos = subtotal_centavos + frete_centavos)`, nunca derivação em leitura (AD-9).

**Dado** a idempotência (NFR-12, AD-7),
**Quando** `POST /api/v1/pedidos` chega,
**Então** a chave **e o digest do corpo** são guardados: mesma chave com mesmo corpo devolve **o Pedido original**, não erro; mesma chave com corpo diferente é **`409`**,
**E** o `23505` do banco é sinal para o código decidir, **nunca** resposta para o navegador,
**E** confirmar duas vezes cria **um** Pedido.

### Estória 5.7: Consistência de Estoque sob concorrência 🔒

**Como** integrante do time,
**quero** o teste que dispara N criações paralelas na última unidade,
**para que** a resposta à pergunta mais provável da banca seja um teste que passa, não uma opinião.

**Condições de Aceite:**

**Dado** o NFR-7,
**Quando** N requisições paralelas disputam a última unidade,
**Então** `pedidos_criados == estoque_inicial`, **nunca mais**,
**E** **o teste é o critério de aceite e nunca é cortado**,
**E** ele roda com `testcontainers-go` sobre PostgreSQL real, porque a transação **é** o objeto do teste.

**Dado** o passo 5 do Roteiro C,
**Quando** a apresentação é planejada,
**Então** ele **só é demonstrado ao vivo se este teste estiver passando** — não se demonstra o que nunca passou automatizado.

### Estória 5.8: Tentativa de Pagamento e a tela do Pedido em processamento

**Como** Comprador,
**quero** ver meu Pedido em processamento com o tempo restante,
**para que** eu saiba que algo está acontecendo e não fique preso na tela.

**Condições de Aceite:**

**Dado** a FR-25,
**Quando** o Comprador confirma,
**Então** **a resposta do checkout não espera o resultado** — ele vai para a tela do Pedido em processamento,
**E** cada submissão gera uma Tentativa com **identificador próprio**,
**E** o Provedor Simulado decide pelos **centavos do total** conforme o §7.1, **só na primeira Tentativa**; da segunda em diante aprova,
**E** não existe superfície de operador nem alternância em tempo de execução — o apresentador provoca o desfecho **escolhendo Produto e quantidade** (AD-8).

**Dado** a tela do Pedido em processamento — apontada pelo §14 do PRD como o elemento de interface mais arriscado do documento,
**Quando** ela é renderizada,
**Então** é **estado próprio, não modal e não spinner**, mostrando número do Pedido, Status e tempo restante até a expiração,
**E** existe **ação persistente em todos os estados**: "Ver o Pedido em Meus pedidos" — o Comprador **nunca fica preso**,
**E** as **três saídas são declaradas**: `PAGO` → a tela vira o Detalhe no lugar, sem navegação nova; `PAGAMENTO_RECUSADO` por recusa **ou** por expiração → a tela substitui o relógio pelo motivo e pela ação de tentar de novo, **na mesma superfície**.

**Dado** o AD-18,
**Quando** o tempo restante é exibido,
**Então** ele vem de `expira_em`, **instante absoluto em RFC 3339 vindo do servidor** — nunca uma duração em segundos, porque duração recomeça quando a página recarrega,
**E** a consulta é a cada **3 s enquanto `AGUARDANDO_PAGAMENTO`**,
**E** a resposta carrega a **tripla completa** — `status`, `tentativas_restantes` e `disponivel` por Item —, para que nenhuma tela faça mais de uma chamada para derivar uma ação,
**E** **nenhum campo nomeia a Reserva de Estoque para o Comprador**.

### Estória 5.9: Confirmação aprovada por webhook

**Como** Comprador,
**quero** ver o Pedido virar `PAGO` sozinho, sem recarregar,
**para que** a espera termine na mesma tela em que começou.

**Condições de Aceite:**

**Dado** o AD-7,
**Quando** a confirmação chega em `POST /api/v1/webhooks/pagamento`,
**Então** `pagamento` grava em `pagamento.confirmacao_recebida` com **restrição única sobre a chave de idempotência** — receber duas vezes é um `INSERT` que falha, **não código defensivo**,
**E** `pagamento` **não chama `pedido`**: a varredura lê por `pagamento.ConfirmacoesNaoAplicadas` e aplica,
**E** cada confirmação tem estado terminal — `PENDENTE → APLICADA` ou `→ NAO_APLICAVEL_SINALIZADA` —, senão a varredura nunca para,
**E** a emissão é **derivada, não marcada**: "Tentativa cujo atraso venceu **e** que não tem linha na inbox", com chave determinística por Tentativa.

**Dado** uma confirmação aprovada,
**Quando** ela é aplicada,
**Então** o Pedido vai a `PAGO`, **a Reserva permanece**, e a tela reflete sem recarregar,
**E** a mesma confirmação duas vezes produz **um único efeito** (NFR-12),
**E** ela **só é aplicada se pertence à Tentativa corrente _e_ o Pedido está em `AGUARDANDO_PAGAMENTO`**.

**Dado** uma aprovação que chega para um Pedido já `CANCELADO`,
**Quando** a varredura a processa,
**Então** o Status **não muda** (FR-26): a confirmação vira `NAO_APLICAVEL_SINALIZADA`, registrada na Tentativa e **sinalizada ao Administrador**,
**E** a informação **não pode virar perda silenciosa** — é a invariante 7 do addendum §2,
**E** o mesmo vale para a confirmação de uma Tentativa anterior chegando depois da nova: sem isso, uma recusa atrasada da Tentativa 1 derrubaria a Reserva que a Tentativa 2 acabou de criar.

> **Deixa para depois:** as duas superfícies onde o Administrador **vê** esse sinal — `Alert` persistente no Detalhe e marcador na Tabela de Pedidos — são a estória 6.5. Aqui o estado é registrado; lá ele é mostrado.

### Estória 5.10: Pagamento recusado e nova Tentativa

**Como** Comprador cujo pagamento foi recusado,
**quero** tentar de novo a partir do próprio Pedido,
**para que** eu não precise remontar nada.

**Condições de Aceite:**

**Dado** uma recusa,
**Quando** ela é aplicada,
**Então** o Pedido vai a `PAGAMENTO_RECUSADO`, **a Reserva é liberada e o Estoque volta** — verificável na Página de Produto em outra aba no mesmo instante (Roteiro B, passo 3),
**E** o motivo fica visível no Detalhe, com as tentativas restantes de 3,
**E** os Itens ficam no próprio Pedido — **nada é remontado**.

**Dado** uma nova Tentativa,
**Quando** ela é iniciada a partir do Pedido,
**Então** o Pedido volta a `AGUARDANDO_PAGAMENTO` criando **nova Reserva**, e **falha se indisponível** (AD-3),
**E** sem Estoque, a mensagem é explícita, o Pedido **permanece em `PAGAMENTO_RECUSADO`**, a ação de tentar de novo **sai da tela** e resta cancelar.

**Dado** o teto de 3 Tentativas (§7.1),
**Quando** ele é atingido,
**Então** a recusa vem de **`pagamento`**, dono da entidade, como **`ErrTetoDeTentativas`**, e `pedido` a traduz na recusa da transição (AD-8),
**E** com as tentativas esgotadas a ação de tentar de novo **sai da tela**; resta cancelar.

### Estória 5.11: Expiração da Tentativa de Pagamento 🔒

**Como** Comprador,
**quero** que um pagamento que nunca confirma não prenda meu Pedido para sempre,
**para que** a unidade volte para a prateleira em vez de ficar comprometida indefinidamente.

**Condições de Aceite:**

**Dado** uma Tentativa sem confirmação dentro do prazo — 15 min padrão, **60 s em demonstração** (§7.1),
**Quando** a varredura passa,
**Então** a Tentativa expira e o Pedido transita para `PAGAMENTO_RECUSADO` com `motivo = TEMPO_ESGOTADO`,
**E** a Reserva é liberada **na mesma transação**,
**E** o caminho da recusa é **reutilizado, sem estado novo**.

**Dado** a invariante 6 do addendum §2,
**Quando** a expiração é decidida,
**Então** ela **deriva do histórico de transições** — "está neste estado desde quando" —, **nunca de temporizador em memória**,
**E** reiniciar o contêiner **não congela Pedido nenhum**.

**Dado** a ordem normativa do AD-6,
**Quando** o tique de 1 s corre,
**Então** **aplicar vem antes de expirar** — senão uma aprovação que chegou no prazo é descartada por um relógio que rodou primeiro,
**E** no Detalhe o tratamento é o mesmo do recusado, com o motivo "Tempo de pagamento expirado".

**Dado** o passo 5 do Roteiro B,
**Quando** um Pedido cujo total cai na faixa `,95`–`,99` é criado com prazo de 60 s,
**Então** ele **sai sozinho** de `AGUARDANDO_PAGAMENTO` e o Estoque volta — é a única forma de a SM-1 exercitar a expiração.

---

## Épica 6: Pedidos — acompanhamento, cancelamento e operação

O Comprador vê o histórico, abre o detalhe com o **preço praticado** e a linha do tempo, e cancela antes
do envio; o Administrador opera a lista pelas três transições que lhe cabem; e a simulação de entrega
avança sozinha até `ENTREGUE`. Quatro ângulos da mesma máquina — colhe o que a estória 5.1 plantou.

**FRs:** FR-29 a FR-33 · **ADs:** AD-3, AD-6, AD-15, AD-18
**UX:** UX-DR12 · UX-DR19 — `mockups/detalhe-do-pedido.html` ilustra a linha do tempo e a ação derivada nos estados `SEPARANDO` e `PAGAMENTO_RECUSADO`

### Estória 6.1: Meus pedidos

**Como** Comprador,
**quero** ver meus Pedidos com o mais recente no topo,
**para que** eu encontre o de ontem sem procurar.

**Condições de Aceite:**

**Dado** um Comprador autenticado,
**Quando** ele abre "Meus pedidos",
**Então** a lista traz **número, data, total e Status**, com os mais recentes primeiro,
**E** mostra **somente os Pedidos do Comprador autenticado**, verificado na mesma consulta que carrega o dado (AD-11),
**E** usa o envelope único de listagem do AD-18, com desempate terminando em `id`.

**Dado** um Comprador sem nenhum Pedido,
**Quando** a tela abre,
**Então** aparece "Você ainda não fez nenhum Pedido." com ação única: voltar à Vitrine.

**Dado** a UX-DR20(b),
**Quando** a lista carrega,
**Então** existe `Skeleton` — a lacuna conhecida da UX, que declarava `Skeleton` só para Vitrine, Resultados e Página de Produto, é fechada aqui.

### Estória 6.2: Detalhe do Pedido

**Como** Comprador,
**quero** ver o que paguei, para onde vai e por onde passou,
**para que** eu saiba onde está meu Pedido sem perguntar a ninguém.

**Condições de Aceite:**

**Dado** um Pedido do Comprador,
**Quando** o Detalhe abre,
**Então** mostra os Itens com **preço praticado — não o preço de hoje** —, o Endereço congelado, o Frete, o total e a **linha do tempo das transições** com data e hora (FR-30),
**E** o Pedido de outro Comprador é negado com o **mesmo erro de inexistente** (NFR-6), fechando o terceiro recurso do teste automatizado iniciado na estória 2.4,
**E** dado um número de Pedido, a sequência de eventos que levou ao estado atual é **reconstruível** (NFR-9).

**Dado** o AD-18,
**Quando** a resposta é montada,
**Então** ela carrega a **tripla completa** — `status`, `tentativas_restantes`, `disponivel` por Item — mais `expira_em` e o `historico`,
**E** **nenhuma tela faz mais de uma chamada para derivar uma ação**,
**E** a consulta é a cada **10 s enquanto o estado não for terminal**, usando `pedido.EstadoTerminal` — a única função de terminalidade.

**Dado** a UX-DR11,
**Quando** a ação disponível é derivada,
**Então** o botão de cancelar aparece **se e somente se** o estado permite,
**E** quando não permite, o botão fica **ausente, não desabilitado**, com uma frase que explica o motivo,
**E** durante uma operação em curso é o contrário: fica na tela, desabilitado e com progresso,
**E** a mudança de Status é anunciada por `aria-live="polite"` (UX-DR14).

### Estória 6.3: Cancelamento pelo Comprador

**Como** Comprador que comprou errado,
**quero** cancelar antes do envio e ver a unidade voltar à prateleira,
**para que** minha desistência não custe uma unidade ao Catálogo.

**Condições de Aceite:**

**Dado** um Pedido em `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO` ou `SEPARANDO`,
**Quando** o Comprador cancela,
**Então** um `Dialog` de primeiro nível pede **confirmação explícita nomeando o Pedido** (FR-31),
**E** ao confirmar, `catalogo.Liberar` é chamado **incondicionalmente** e as unidades voltam — verificável na Vitrine **no mesmo instante**,
**E** o selo muda e a linha do tempo ganha a transição.

**Dado** um Pedido em `ENVIADO` ou `ENTREGUE`,
**Quando** o cancelamento é tentado **inclusive por chamada direta à API**,
**Então** ele é recusado com `ErrForaDaJanelaDeCancelamento`, que **explica por que não dá mais**,
**E** cancelar um Pedido **já `CANCELADO` não produz efeito nem erro** — `Liberar` sem reserva ativa devolve `nil` (AD-5).

**Dado** a janela entre a consulta de 10 s e a simulação de 30 s,
**Quando** o Comprador clica "Cancelar Pedido" numa tela `SEPARANDO` que já virou `ENVIADO`,
**Então** o selo e a linha do tempo são atualizados e a tela explica: "Este Pedido foi enviado enquanto você estava nesta tela e não pode mais ser cancelado.",
**E** **nunca cai no `Toast` genérico de erro** — seria um erro cru no clímax da UJ-3.

### Estória 6.4: Painel de Pedidos do Administrador

**Como** Administrador,
**quero** filtrar os Pedidos e avançá-los pelas transições que me cabem,
**para que** eu opere a loja sem tocar no banco.

**Condições de Aceite:**

**Dado** o painel,
**Quando** ele abre,
**Então** a lista é **filtrável por Status e ordenável por data**, com consulta a cada **10 s enquanto listar Pedido não terminal**,
**E** o Administrador abre o detalhe de qualquer Pedido, com o **Comprador e os Vendedores dos Itens**.

**Dado** a FR-32,
**Quando** as ações são montadas,
**Então** ele executa **exatamente três transições e mais nenhuma**: `PAGO → SEPARANDO`, `SEPARANDO → ENVIADO` e `ENVIADO → ENTREGUE`,
**E** **o Administrador não cancela Pedido** — as três transições são exaustivas (UX-DR11),
**E** os botões são montados a partir de `dados.permitidas`, que `ErrTransicaoInvalida` carrega.

**Dado** uma transição **inválida**,
**Quando** ela é tentada,
**Então** um `Alert` **destrutivo** nomeia a transição tentada e quais são permitidas a partir do estado atual — é defeito de quem chamou, e o addendum exige que falhe alto.

**Dado** um Pedido que **já avançou por outro ator** — o Administrador e a simulação colidindo,
**Quando** o CAS devolve `ErrEstadoJaAvancado`,
**Então** o `Alert` é **informativo, não destrutivo**: "Este Pedido já está em {estado}. A linha foi atualizada.",
**E** dizer "transição inválida" aqui reportaria **causa falsa** — o Pedido não falhou, ele avançou.

**Dado** as listas administrativas,
**Quando** carregam,
**Então** têm `Skeleton` (UX-DR20(b)), e a tabela larga **rola dentro do próprio contêiner**, sem rolagem horizontal na página (UX-DR15).

### Estória 6.5: Pagamento aprovado sobre Pedido cancelado, visível ao Administrador

**Como** Administrador,
**quero** ver quando um pagamento foi aprovado para um Pedido já cancelado,
**para que** a corrida clássica de checkout não vire perda silenciosa de informação.

**Condições de Aceite:**

**Dado** o sinal `NAO_APLICAVEL_SINALIZADA` gravado na estória 5.9,
**Quando** a resposta administrativa do Pedido é montada,
**Então** ela carrega **`pagamento_aprovado_sobre_cancelado`** — um estado sem caminho de leitura é um estado que ninguém vê (AD-18).

**Dado** as duas superfícies do Administrador,
**Quando** o caso ocorre,
**Então** o Detalhe mostra um `Alert` **persistente** e a Tabela de Pedidos ganha um **marcador na linha**,
**E** o Status do Pedido **não muda** (FR-26),
**E** é este registro que torna possível o estorno automático da fase 2 **sem arqueologia de log** (addendum §4).

### Estória 6.6: Simulação de entrega

**Como** apresentador,
**quero** que o Pedido avance sozinho até `ENTREGUE` durante a apresentação,
**para que** a UJ-2 se complete sem eu clicar em nada.

**Condições de Aceite:**

**Dado** a FR-33,
**Quando** a simulação corre,
**Então** ela avança de `PAGO` até `ENTREGUE` **pelas mesmas transições** da tabela do AD-3 — nenhum caminho paralelo,
**E** o intervalo é configurável (24 h padrão, **30 s em demonstração**) e a simulação é **desligável por configuração**,
**E** ela **nunca move um `CANCELADO` nem um `AGUARDANDO_PAGAMENTO`**.

**Dado** o AD-6,
**Quando** o avanço é decidido,
**Então** ele **deriva do histórico**, e **reiniciar o sistema retoma de onde parou**,
**E** cada Pedido corre em sua própria transação com `FOR UPDATE SKIP LOCKED` — um Pedido que falhe **não derruba a varredura dos outros**,
**E** `ErrEstadoJaAvancado` é registrado e seguido em frente, **nunca como falha**.

**Dado** `SEPARANDO → ENVIADO`,
**Quando** a simulação a executa,
**Então** `catalogo.Consolidar` baixa o `estoque_total` e encerra a Reserva,
**E** **nenhuma outra transição altera o total**.

> *FR-33 é o **4º item da ordem de corte**. Sem ela, o Administrador avança o Status manualmente pela FR-32: custa um clique a mais na apresentação e devolve tempo de desenvolvimento.*

### Estória 6.7: A forma de exibição dos sete Status

**Como** qualquer pessoa que olhe uma tela,
**quero** ver o mesmo texto e a mesma cor para o mesmo Status em toda superfície,
**para que** eu não precise aprender dois vocabulários para o mesmo sistema.

**Condições de Aceite:**

**Dado** a UX-DR10,
**Quando** um Status é exibido,
**Então** o mapeamento é **fechado e um-para-um**: `AGUARDANDO_PAGAMENTO` → "Aguardando pagamento" · `PAGAMENTO_RECUSADO` → "Pagamento recusado" · `PAGO` → "Pago" · `SEPARANDO` → "Separando" · `ENVIADO` → "Enviado" · `ENTREGUE` → "Entregue" · `CANCELADO` → "Cancelado",
**E** é o **mesmo componente, texto e cor nas três superfícies** — Meus pedidos, Detalhe do Pedido e Tabela do Administrador,
**E** **nunca o identificador cru e nunca ícone**.

**Dado** a UX-DR7,
**Quando** a cor do selo é escolhida,
**Então** o verde `{colors.available}` aparece **só** em `ENTREGUE` — nunca como sucesso de operação, confirmação de formulário ou `Toast`,
**E** o selo usa `rounded.full`, um dos quatro usos declarados (UX-DR5).

---

## Épica 7: Entrega — documentação viva e ensaio

A banca encontra, no dia da entrega, um README que leva do clone ao sistema rodando, o diagrama dos seis
módulos correspondendo à decomposição real do código, e o addendum sem nenhuma decisão contradita pelo
que foi construído. E os três roteiros do §12 ensaiados a partir de clone limpo.

**FRs:** nenhum — o entregável é o §6.1 do PRD · **Métricas:** SM-1, SM-3, SM-7 · **AD:** AD-1

*Esta épica **verifica** o que as seis anteriores produziram; ela não escreve documentação do zero. O
addendum é mantido vivo desde a estória 1.2, e cada decisão volta para ele na estória que a tomou.*

### Estória 7.1: O README leva do clone ao sistema rodando

**Como** pessoa de fora do time,
**quero** clonar o repositório e chegar ao sistema funcionando seguindo só o README,
**para que** a nota de documentação não dependa de alguém explicar por cima do ombro.

**Condições de Aceite:**

**Dado** um clone limpo e alguém que não escreveu o código,
**Quando** essa pessoa segue **apenas** o README,
**Então** ela chega ao sistema funcionando com Catálogo Semeado em **≤ 15 minutos** (NFR-1, SM-3),
**E** a verificação é **recorrente, não única**: a cada semana, um integrante diferente sobe o ambiente do zero,
**E** o README documenta `docker compose up`, o modo demonstração como padrão, `docker compose down -v` como botão de reinício, e onde ler o token da FR-3 no log.

### Estória 7.2: O diagrama dos seis módulos corresponde ao código

**Como** membro da banca,
**quero** um diagrama que descreva o sistema que foi construído,
**para que** eu possa perguntar sobre o que estou vendo.

**Condições de Aceite:**

**Dado** o §6.1 do PRD,
**Quando** o diagrama é entregue,
**Então** ele mostra os **seis módulos do NFR-2 com suas fronteiras e o que atravessa cada uma**,
**E** **o diagrama do AD-1 é o diagrama exigido** — não há um segundo desenho a manter,
**E** ele corresponde à decomposição real: `internal/fronteira_test.go` passando é a prova, não a boa vontade.

**Dado** a contra-métrica SM-C4,
**Quando** o volume é avaliado,
**Então** páginas escritas não contam — **documentação que descreve um sistema diferente do construído é pior que documentação nenhuma**.

### Estória 7.3: O addendum percorrido contra o código

**Como** time,
**quero** percorrer cada decisão do addendum contra o que foi construído,
**para que** a SM-7 seja verificada, e não presumida.

**Condições de Aceite:**

**Dado** o `addendum.md`,
**Quando** as decisões são percorridas contra o código antes da entrega,
**Então** **nenhuma é contradita pelo que foi construído** (SM-7),
**E** o desfecho das cinco verificações do passo 0 está registrado nele (estória 1.9),
**E** toda decisão de arquitetura tomada durante a construção está lá, incluindo o número medido do NFR-4 (estória 1.4).

**Dado** uma contradição encontrada,
**Quando** ela é resolvida,
**Então** ou o código muda, ou o addendum registra a mudança de decisão **com o porquê** — nunca se apaga a decisão antiga.

### Estória 7.4: Os três roteiros ensaiados a partir de clone limpo

**Como** time que vai apresentar,
**quero** os três roteiros ensaiados do zero por duas pessoas diferentes,
**para que** nada seja demonstrado ao vivo pela primeira vez.

**Condições de Aceite:**

**Dado** a SM-1,
**Quando** os três roteiros do §12 são executados em ambiente limpo,
**Então** todos passam de ponta a ponta, **sem intervenção manual no banco e sem erro visível** — alvo 3 de 3.

**Dado** as regras de ensaio,
**Quando** a apresentação é preparada,
**Então** os roteiros são ensaiados **a partir de clone limpo, por ao menos duas pessoas diferentes**,
**E** **nenhum passo entra na apresentação sem ter passado em ensaio**,
**E** o passo 5 do Roteiro C só é demonstrado **se o teste do NFR-7 estiver passando** (estória 5.7).

**Dado** as perguntas prováveis da banca,
**Quando** os responsáveis são definidos,
**Então** cada pergunta tem **um responsável nomeado antes da apresentação**,
**E** ninguém responde por uma parte que não construiu sem antes ler a fonte.

**Dado** a contra-métrica SM-C1,
**Quando** o escopo é revisto,
**Então** **nenhuma funcionalidade nova entra enquanto os três roteiros não passarem** — mais telas com o ciclo quebrado vale menos que o ciclo fechado.
