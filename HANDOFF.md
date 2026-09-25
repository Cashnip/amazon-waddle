# Azamon — Handoff

**Atualizado:** 2026-09-25 · **Estado:**
- PRD, UX, arquitetura, spec e épicas finalizados e reconciliados entre si.
- **Épicas 1 a 6 fechadas:** as 48 estórias estão `done` e em `main`. O Roteiro A anda de clone limpo em 3 min 26 s (teto do NFR-1: 15 min) e numa rede sem saída (NFR-15); o Roteiro B anda do passo 1 ao 5; o Roteiro C tem a prova do NFR-7.
- Os passeios da Épica 5 e da Épica 6 acharam D1–D7 e E1–E5, e **todos estão corrigidos em `main`** (`7014591`, `9240730`, `ab4558c`, `33ca441`). O2 decidido (não é defeito), O3 aceito sem mudança.
- As retrospectivas headless das Épicas 5 e 6 estão `accepted-with-open-items`; as ações delas, com dono, estão no `sprint-status.yaml`.
- O passeio só pelo teclado foi encerrado em 2026-09-25 por decisão humana e deixou K1 e K2, os dois de severidade baixa.
- Falta o passo 7 abaixo e a Épica 7 (quatro estórias, `backlog`), que verifica e não constrói.
- O relato estória a estória até a Épica 6 saiu para [`handoff-historico-ate-epica-6.md`](_bmad-output/implementation-artifacts/handoff-historico-ate-epica-6.md).

Réplica da Amazon como trabalho de faculdade. Time de 2 a 4 pessoas, um semestre, avaliado em três eixos ao mesmo tempo: funcionalidade entregue, arquitetura e documentação, e apresentação ao vivo.

## Setup

Está em [`README.md`](README.md), com os comandos exatos e como verificar cada um. Ele é
lido em duas partes: as seções 1 a 4 vão do `git clone` ao sistema rodando e ao passeio do
esqueleto, e só pedem **Docker**; da 5 em diante é ferramenta de quem vai desenvolver — o
repositório versiona **o trabalho**, não a ferramenta, então depois de clonar faltam o BMad
6.11.0 e as skills, ambos reproduzíveis em dois comandos.

## Onde está o quê

| Arquivo | O que é |
|---|---|
| `_bmad-output/specs/spec-azamon/SPEC.md` | **O contrato canônico, e a porta de entrada.** Oito capacidades com ID estável (`CAP-1`..`CAP-8`), 16 restrições, 13 não-objetivos, o sinal de sucesso e as duas questões em aberto. O `companions:` do frontmatter é a lista fechada do que mais precisa ser lido |
| `.../mapa-de-capacidades.md` | Cada `CAP-N` ligado a FRs, módulo, `AD`s governantes, superfícies da `EXPERIENCE`, passo do roteiro que a demonstra e passo do addendum §8 — mais os sete NFRs transversais e a ordem de corte mapeada em capacidades. É por aqui que as épicas fatiam |
| `.../.memlog.md` | 43 decisões da spec. Mesmo papel dos outros memlogs |
| `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md` | O PRD. Comece pelo **Sumário Executivo**, que tem um mapa de leitura por público |
| `.../addendum.md` | Decisões técnicas, invariantes, 8 alternativas descartadas com o que se perdeu, ordem de construção. Insumo direto da arquitetura |
| `.../.memlog.md` | 93 decisões em ordem cronológica, com o motivo de cada uma. É o registro canônico — se o PRD e a sua memória divergirem, vale isto |
| `.../review-rubric.md`, `.../review-consistency.md` | Revisões preservadas. Restam 14 achados médios e 12 baixos, nenhum bloqueante |
| `_bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/DESIGN.md` | Identidade visual. shadcn/ui mais uma camada de marca de dez cores. Espinha, não sugestão |
| `.../EXPERIENCE.md` | Comportamento: IA, estados, interações, piso AA, e as quatro jornadas com a camada de interface |
| `.../mockups/` | Três telas em HTML que abrem offline. Ilustram; as espinhas vencem em conflito |
| `.../.memlog.md` | 40 decisões da UX. Mesmo papel do memlog do PRD: é o registro canônico |
| `.../review-rubric.md`, `.../review-adversarial.md` | As duas revisões da UX. 52 achados, todos resolvidos |
| `_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md` | **A espinha de arquitetura.** 20 invariantes com ID estável (`AD-1`..`AD-20`), cada um com o que prende, a divergência que impede e a regra. É o substrato que as épicas e o `bmad-build` consomem — comece pelo Paradigma de Design e pelo `AD-1` |
| `.../DIAGRAMA-MODULOS.md` | O diagrama dos seis módulos exigido pelo §6.1 e medido pela SM-7. Lê-se sozinho |
| `.../deck-banca.html` | Deck da apresentação. Máquina de estados interativa; abre offline |
| `.../.memlog.md` | 84 decisões da arquitetura. Mesmo papel dos outros memlogs |
| `.../reviews/` | Sete revisões: três reconciliações (PRD, addendum, UX) e quatro lentes do portão (rubrica, versões, concorrência, adversária) |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | O rastreamento: Status de cada estória e os itens das retrospectivas com dono. Ao lado, `deferred-work.md` (o adiado, com a condição de fechamento), as specs de cada estória e as retros |
| `.../handoff-historico-ate-epica-6.md` | O "Próximo passo" deste arquivo até a Épica 6, sem edição: o que cada estória deixou pronto, os passeios e os defeitos já corrigidos. Citações antigas `HANDOFF.md:NNN` resolvem por `git show 5f54a54:HANDOFF.md` |

Não relea tudo: comece pela `SPEC.md` e desça pelos `companions:` que o seu trabalho exigir. O PRD tem índice de requisitos no §4 e mapa de leitura no topo.

**A spec não substitui nada.** O `sources:` dela está vazio de propósito: os seis documentos acima continuam vivos e são leitura obrigatória a jusante. Ela amarra e resolve a altitude — não é um resumo que aposenta o resto.

## Decisões travadas

Estão no Sumário Executivo do PRD. As quatro estruturais, em uma linha: marketplace no modelo de dados mas sem portal de vendedor · pagamento simulado porém assíncrono, com expiração e idempotência · máquina de estados do Pedido como peça central · busca é texto mais filtros, não motor de busca.

Stack: **Go 1.27** no back-end, **PostgreSQL 18** + **Redis 8**, **Docker obrigatório**, e **Next.js 16 sobre React 19** no front-end — decidido em 2026-09-06.

Da arquitetura, as quatro que mais mudam a construção: **um schema do Postgres por módulo, com chave estrangeira cruzando schema proibida** · **o Catálogo é dono do Estoque, incluindo a Reserva** · **a confirmação do pagamento entra por webhook HTTP e é aplicada por *inbox*** · **um relógio só move o tempo, e ele deriva do histórico de transições**. Versões exatas na tabela Stack da espinha.

## Bloqueios

| # | O quê | Dono | Destrava o quê |
|---|---|---|---|
| 1 | ~~**Next.js ou React**~~ | — | **Resolvido em 2026-09-06:** Next.js 16 sobre React 19, escolhido por familiaridade do time. Como o PRD antecipava, nada do §4 mudou |
| 2 | **Enunciado ou rubrica do professor** | Sung | Quatro suposições do §16 (portal do vendedor, verificação de e-mail, devolução, pedido dividido por vendedor). Ao obter: `bmad-prd` em modo atualização, que reconcilia contra o memlog, depois a espinha e a spec na mesma ordem |

O bloqueio 2 não impede começar a construir: nenhuma das quatro suposições toca o esqueleto vertical.

**As cinco verificações do passo 0 têm desfecho nomeado no `addendum.md` §10: as quatro do time fecharam positivas, e a quinta continua aberta por ser externa ao time — é o bloqueio 2 acima, dono Sung.** A espinha tem uma seção de Questões em Aberto com quatro checagens de dez minutos — nenhuma é decisão, todas são "confirmar que funciona como assumimos" — mais essa quinta, que não depende de ninguém daqui e não toca o esqueleto vertical. **As quatro:** na 1.2, o `Set-Cookie` do Go atravessa o `rewrites()` do Next íntegro, então a saída de *proxy* explícito não precisa ser acionada, e o `npx shadcn init` roda limpo em Next 16 + React 19, sem flag de dependências de pares; na 1.3, o sqlc v1.31.1 analisa `DEFAULT uuidv7()` sem erro, então a chave primária continua sendo gerada pelo banco; na 1.4, a busca com termo, Categoria e faixa de preço sobre 5.050 Produtos mede **p95 de 0,34 ms** contra um teto de 500 ms, e "índice antes de serviço" está confirmado. Os desfechos estão no `addendum.md` §10, com a entrada da 1.9 consolidando os cinco.

## Próximo passo

Contexto limpo entre cada sessão. Os passos 1 a 6 da ordem decidida em 2026-09-24 estão feitos; o que eles deixaram está no [histórico](_bmad-output/implementation-artifacts/handoff-historico-ate-epica-6.md).

1. **`bmad-build` dos testes que faltam** (passo 7): `epic-5-retro-item-26` (teste que falha sem o `ORDER BY id` na trava — `travaDeReservaSerializa` usa um Produto só e não o pega), `epic-6-retro-item-32` (tirar a dependência de ordem de `TestSessaoEProduto`, `api/sessao_test.go:566-588`) e `epic-6-retro-item-33` (teste que falha quando o `de` da transição administrativa deixa de ser o Status visto no clique).
2. **`bmad-build` na Estória 7.1** (o README leva do clone ao sistema rodando), a primeira da Épica 7. A Épica 7 **verifica** o que as seis anteriores produziram, e não escreve documentação do zero: README do clone ao sistema rodando em ≤ 15 min (NFR-1), o diagrama dos seis módulos contra o código (7.2), o addendum percorrido contra o código (7.3), e os três roteiros ensaiados a partir de clone limpo (7.4).

Não um `bmad-build` único para tudo: a mistura força a revisão cara sobre o trivial ou a barata sobre o que tem raio de alcance (política de revisão do `bmad-build`).

### Defeitos abertos

Do passeio só pelo teclado de 2026-09-25 (teclado de verdade e VoiceOver no Chrome, 1482 px):

| ID | Sev. | Onde | O quê |
|---|---|---|---|
| K1 | baixa | `web/components/casca.tsx:40` | Sem "Pular para o conteúdo": do topo ao primeiro Produto são 24 `Tab`s (logo, Categoria, busca, Carrinho, conta e a faixa de Categorias). Há `<main>`, então o WCAG 2.4.1 passa pelo rotor do leitor de tela; é atrito só para quem usa o teclado sem leitor |
| K2 | baixa | `web/app/checkout/endereco/escolha-de-endereco.tsx:216` | Depois de salvar um Endereço pelo `Dialog`, o novo fica marcado mas a parada de `Tab` do grupo fica no antigo (`tabindex=0` no desmarcado, `-1` no marcado): entrando com `Shift+Tab`, o VO diz "Paulista, não selecionado" em vez do Endereço que vai receber o Pedido. É o foco itinerante do Radix, que guarda o último item focado e não acompanha o `value` mudado por código. A seta corrige |

O resto do que ficou adiado, com a condição de fechamento de cada um, está em `_bmad-output/implementation-artifacts/deferred-work.md` — entre eles, o botão que some ao confirmar (remover Endereço, Categoria, Vendedor) deixa o foco no `body`.

### Decisões abertas

- **O1** (`epic-5-retro-item-28`): "Confirmar o novo preço" é trava só de tela — recarregar ou digitar `/checkout/revisao` a dispensa, porque a entrada no checkout já gravou a ciência ao reportar. A FR-19 pede "exibido e sinalizado", e ele é exibido; a garantia do servidor hoje é que nenhum Item nasce com preço diferente do que a entrada registrou. Aceitar como leitura da FR-19, ou exigir a ciência no servidor. Dono: Sung.
- **AD-4** (`epic-5-retro-item-27`): a ordem global "pedido → produtos" da espinha não cita o contador do ano (`ProximoNumeroDoAno`), que o `Criar` trava antes dos dois. Emendar no lugar, com memlog da arquitetura; e o comentário de `db/migracoes/20260921120000_pedido_criacao.sql:14` diz que o gêmeo espera no índice, quando espera no contador.
- **Os 14px do selo** (`epic-6-retro-item-31`): a decisão humana de 2026-09-24 vive só na spec da 6.7 — o `DESIGN.md` diz ao mesmo tempo "`Badge` sem alteração" e "no mínimo 14px", e o marcador "Pagamento aprovado" da 6.5 segue em 12px. Pede `uv run _bmad/scripts/memlog.py append` no diretório da UX e a palavra do dono da UX.
- **Dividir `internal/pedido/pedido.go`** (1.196 linhas, três blocos que não se tocam; B2 da retro da Épica 6): agora, antes de a 7.3 percorrer o addendum contra o código, ou aceitar como está.
- **`epics.md` é documento vivo depois da construção?** O `epic-3-retro-item-16` virou contradição: a CA da 3.5 diz verde "**só** para Estoque disponível", e a 6.7 embarcou o verde em `ENTREGUE`, como o `DESIGN.md` manda.
- **O E5 foi tratado como defeito** (`33ca441` tirou o `role="alert"` do `Alert` da 6.5) sem registro de decisão humana; a retro da Épica 6 não o reabre.
- **Transições propostas e não gravadas:** `epic-3-retro-item-19` e `item-23` → `done`.

### O que as correções deixaram sem ver na tela

Pelo teclado ou pelo mouse, ninguém registrou ter visto depois das correções: os desfechos `,00` e `,90` pelo teclado, o `Alert` de preço (D7), a UJ-3 (E2, E3, e cancelar em `AGUARDANDO_PAGAMENTO` com o Produto de R$ 39,95), E1, E4, E5, Meus pedidos vazio, o painel do Administrador pelo teclado, e `AZAMON_ENTREGA_SIMULACAO_ATIVA=false`. Das Épicas 3 e 4 há passeios nas quatro larguras que nunca foram feitos, e cada estória das Épicas 5 e 6 tem o roteiro do seu passeio — os dois estão no histórico, e são insumo da 7.4.

## Quanto custa uma estória, e o que fazer com isso

A 1.1 consumiu ~795.000 tokens de subagente. **As três lentes de revisão adversarial do `bmad-build`, mais a
rodada de patches que elas geram, foram 56% disso** — e escalam com o tamanho do diff, não com a dificuldade da
estória. Sobram 51 estórias; nesse ritmo o orçamento não fecha.

A 1.1 foi o pior caso — greenfield, a maior estória do épico, e 14 lacunas de documentação que precisaram virar
decisão nomeada (estão no addendum §10). Mas a parte cara é justamente a que se repete. Três medidas:

- **Três lentes só onde há raio de alcance real:** reserva de Estoque sob concorrência (5-6, 5-7), máquina de
  estados do Pedido (5-1), webhook e idempotência (5-9). Para CRUD de mesma forma, uma lente basta. *(A 5-9
  acabou levando uma só, e foi a medida certa: a investigação mostrou que o webhook e a idempotência já eram da
  1.7, e o diff da estória era um ramo de log mais testes. O gatilho é o raio do **diff**, não o do tema. A 5-10 levou as três, e o raio justificou: a lente de
  verificação achou o teste de corrida que não se sobrepunha, e a de bordas, a releitura sobrescrita na tela.)*
- **Agrupar estórias de mesma forma numa spec só.** 3-1, 3-2 e 3-3 são o mesmo CRUD três vezes.
- **Limpar o contexto entre estórias.** O `epic-N-context.md` e a spec em disco bastam para retomar.

A spec rodou em 2026-09-06 e adotou os seis documentos como companheiros, sem absorver nenhum; as épicas
rodaram em 2026-09-07 e citam `CAP`, `FR`, `NFR`, `AD` e `UX-DR` por ID, sem copiar enunciado.

E antes de alargar qualquer camada: o **passo 0** do addendum §8. Um Produto, um Comprador, um Pedido que nasce, é pago pelo Provedor Simulado e chega a `ENTREGUE`, com telas feias. É onde as cinco verificações acima acontecem.

## O que uma sessão nova erraria

- **Disciplina de glossário.** O §3 do PRD define 20 termos usados literalmente no documento inteiro. Trocar por sinônimo é defeito, não estilo — nunca "compra" no lugar de **Pedido**, nunca "item" no lugar de **Produto**. Duas revisões já caçaram violações disso.
- **Começar horizontal.** O addendum §8 tem um **passo 0**: esqueleto vertical (um Produto, um Comprador, um Pedido até `ENTREGUE`, telas feias) antes de alargar qualquer camada. A ordem por dependência que vem depois é correta e perigosa se seguida sozinha — produz um sistema que só funciona no fim do semestre.
- **Cortar o que é invisível.** FR-19, FR-24, FR-34 e o teste de concorrência do NFR-7 não aparecem em tela nenhuma e são o que separa o sistema da maquete. A ordem de corte legítima está no §6.3.
- **Renumerar suposições.** O §16 foi reordenado por risco; referências a "suposição N" já quebraram uma vez por causa disso.
- **Achar que o addendum é rascunho.** Ele está `final`, mas a SM-7 exige que continue vivo: toda decisão de arquitetura tomada durante a construção volta para ele.
- **Dar ao Administrador um botão para forçar a recusa do pagamento.** Não existe superfície de operador para isso. O Provedor Simulado decide pelos **centavos do total do Pedido** (§7.1 do PRD, addendum §4): o apresentador provoca o desfecho que quer escolhendo Produto e quantidade, sem reconfigurar nada entre os roteiros. A faixa vale para a **primeira** Tentativa de Pagamento; da segunda em diante aprova, senão a nova tentativa da FR-27 não teria o que exercitar.
- **Construir o checkout em quatro passos.** Resolvido em 2026-09-05: o §9 diz **Endereço → Revisão**, e Frete e pagamento são blocos da Revisão — nenhum dos dois tem dado a pedir. Se você leu quatro passos em algum lugar, era uma cópia velha.
- **Buscar fonte na rede.** O NFR-15 exige percorrer o Roteiro A com a rede desconectada. Isso proíbe Google Fonts, `@import` remoto e CDN — a fonte é auto-hospedada no repositório. É o erro mais fácil de cometer e quebra a demonstração em silêncio, na sala.
- **Somar as reservas antes de segurar a trava.** É o defeito que vende a mesma última unidade duas vezes, e o texto da arquitetura só passou a impedi-lo depois que uma lente de concorrência mostrou o entrelaçamento. São **dois comandos, nesta ordem**: `SELECT … FROM catalogo.produto WHERE id = ANY($1) ORDER BY id FOR UPDATE`, e **só então** a soma das reservas ativas. O `ORDER BY id` não é estilo — sem ele, dois Pedidos com os mesmos dois Produtos em ordem inversa travam um no outro (`AD-5`, `AD-4`).
- **Pôr a emissão da confirmação simulada dentro de `pedido`.** Emitir é comportamento do Provedor e mora em `pagamento` (`AD-6`). Dentro do Pedido, põe conhecimento de gateway no domínio e quebra o NFR-3 e a SM-5 na primeira troca — que é justamente o que a banca vai perguntar.
- **Tratar "sem reserva ativa" como erro.** `catalogo.Liberar` sem reserva ativa devolve `nil`, `Consolidar` sobre reserva já consolidada é no-op, e `Reservar` com lista vazia é no-op. `pedido` chama `Liberar` **incondicionalmente** em toda transição para `CANCELADO` e em toda recusa. Sem isso, cancelar um Pedido em `PAGAMENTO_RECUSADO` falha sempre, com rollback.
- **Registrar `GET /api/v1/produtos` em dois lugares.** A rota é de `busca`, com ou sem `termo` — a Vitrine é uma listagem, não uma rota à parte. `catalogo` serve `GET /api/v1/produtos/<id>` e o CRUD administrativo, e não registra handler de listagem. Duas equipes registrando o mesmo padrão no `ServeMux` fazem o binário **entrar em pânico no arranque**: falha de subida, não de comportamento (`AD-16`).
- **Fixar o Next.js na linha 16.2.** O patch das duas RCE críticas de 25/08/2026 pousou em **16.3.3**, e a 16.2.x não recebe backport. A espinha fixa **16.3.4**. Vale para toda a tabela Stack: as versões foram verificadas na web, não lembradas.
- **Usar o verde ou o laranja para enfeitar.** No `DESIGN.md` os dois têm sentido fechado: verde é Estoque disponível e `ENTREGUE`; laranja aparece **uma vez por fluxo**, no passo irreversível. Usar qualquer um decorativamente apaga o único mecanismo semântico de cor do sistema.
- **Rodar `bmad-sprint-planning` direto no `epics.md`.** O parser do `sprint_plan.py` só reconhece `## Epic N:` e `### Story N.M:` em inglês — os nossos cabeçalhos são `## Épica N:` e `### Estória N.M:`, e ele devolve zero épicas **sem um único aviso**. Gere uma cópia temporária antes de chamar o script, e passe a cópia no `--epic-file`:
  ```
  sed -E 's/^(#{1,3}) Épica /\1 Epic /; s/^(#{2,4}) Estória /\1 Story /' \\
    _bmad-output/planning-artifacts/epics.md > /tmp/epics-en.md
  ```
  Só a palavra estrutural muda: as chaves do `sprint-status.yaml` continuam saindo do título em português.
- **Editar a `SPEC.md` à mão.** Ela é **derivada** do `.memlog.md` da spec a cada execução, e `bmad-spec` é a única escritora — uma edição manual é sobrescrita no próximo derive, em silêncio. Mudou algo? Rode `bmad-spec` de novo apontando para a mesma pasta: os `CAP` são preservados por ID. O mesmo vale para o `mapa-de-capacidades.md`.
- **Delegar o passo 2 do `bmad-build` (investigação) a um subagente sem escopo redundante.** Um fork/subagent herda a conversa inteira, inclusive as instruções dos passos seguintes do próprio workflow — e pode segui-las em vez da instrução específica do turno ("só investigue, não escreva código"). Foi o que aconteceu na 2.6: o subagente de investigação implementou a estória inteira e commitou em `main` sem passar pelo CHECKPOINT 1 nem por nenhum outro HALT. O commit acabou correto depois de revisão e verificação independentes (`go build`/`go vet`/`go test ./...` e o build Docker do `web/`, rodados de novo fora do relato do subagente), mas por sorte de execução, não por garantia do processo. Peça investigação numa sessão nova sem o workflow completo no contexto, ou revise o commit resultante linha por linha antes de confiar nele — nunca trate a delegação como segura por padrão.
- **Passear sobre imagem velha.** `docker compose up` não reconstrói: se `azamon` e `web` já têm imagem, ela sobe como está, com o código de quando foi construída. Em 2026-09-25 o `down -v && up` subiu imagens de 16 h e 10 h antes, com o Andarilho a R$ 39,90 e nenhuma correção de D1/E1–E5 no ar. Antes de passeio ou apresentação, `docker compose up --build`, e confira a hora em `docker compose images`.
- **Semear dado de regra em `db/semente/`.** A semente roda **uma vez por marcador de versão**: um banco que já a tem nunca vê arquivo novo, e trocar `VersaoSemente` reexecuta os `INSERT` do Catálogo, que colidem em uuid literal. Dado que é regra — a Faixa de Frete é o caso — nasce na própria migração, e aí existe em todo banco, testcontainers inclusive. `db/semente/` é do Catálogo Semeado da demonstração, e o arquivo é **gerado** por `media/gerar.go`.
- **Pôr centavos quebrados num valor que entra no total do Pedido.** O Provedor Simulado decide pelos **centavos do total**, então um Frete de R$ 19,90 transformaria todo Produto de `,00` numa recusa, e o apresentador perderia o controle do desfecho. Os valores da `faixa_frete` são reais inteiros, com `CHECK (valor_centavos % 100 = 0)`; quem acrescentar faixa mantém isso.
- **Achar que Go, sqlc e Next rodam direto nesta máquina Windows.** Não rodam: `go` não está no `PATH` e o `web/`
  não tem o binário `next` instalado. O que funcionou na 4.2/4.3: `go vet` e `go test` na imagem
  `azamon-dev:latest`, com `-v azamon-gocache:/root/.cache`, `-v //var/run/docker.sock:/var/run/docker.sock` e
  `-e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal -e TESTCONTAINERS_RYUK_DISABLED=true` (o testcontainers
  precisa do socket); `sqlc generate` na imagem `sqlc/sqlc:1.31.1`; e o build do `web/` por `docker build ./web`.
  Dois efeitos colaterais: o sqlc reescreve **todos** os `gerado/` com LF por cima do CRLF do checkout, então
  restaure com `git checkout` os módulos que você não mexeu (`git diff --ignore-cr-at-eol --stat` mostra o que é
  real); e `gofmt -l` lista todo arquivo CRLF, mudado ou não — confira com `tr -d '\r' < arquivo | gofmt -l`.
- **Reconstruir só o `web` sem `--no-deps`.** `docker compose up -d --build web` recria também o `azamon` pelo `depends_on`, com o `.env` — um override como `AZAMON_ENTREGA_SIMULACAO_ATIVA=false` passado por `-f` some calado. Use `docker compose up -d --build --no-deps web`.
- **Mudar a semente e esperar ver no banco que já existe.** A semente é `INSERT` sem `ON CONFLICT`, e trocar a `VersaoSemente` sobre banco já semeado derruba o arranque — por isso ela não mudou quando o Andarilho foi a R$ 39,95. Banco existente só vê a semente nova com `docker compose down -v`, que apaga também todo Pedido de passeio.
- **Pôr `autoFocus` dentro de um `Dialog`.** O React foca antes do `FocusScope` do Radix, e o `DialogContent` compartilhado não guarda quem tinha o foco — no fechar, ele cai no `body`. O `DialogContent` devolve o foco ao botão que abriu, salvo destino escolhido por quem o usa; escolha o destino por ele, não por `autoFocus`.
- **Escrever um Status de Pedido na tela sem `<SeloDoStatus status={…} />`.** O texto vem de `rotuloDoStatus` e a aparência de `aparenciaDoSelo`, os dois em `web/lib/pedido.ts`. Duas guardas de fonte em `web/scripts/pedido.test.mjs` derrubam o `npm test` e o `docker build ./web` (pelo `prebuild`): Meus pedidos e os dois Detalhes não podem chamar `rotuloDoStatus`, a Tabela só o chama com `s` e `filtro.status`, e o verde (`bg-`, `border-`, `ring-`, `var(--available)`, `#007600`) só existe em `lib/pedido.ts` e no `globals.css`, com `text-available` só nas duas telas de Estoque disponível. **Pintar de verde em outro lugar derruba o build** — é a UX-DR7.
- **Confiar nos padrões do shadcn para o vermelho e a pílula.** O `destructive` tinge a 10% e dá 3,99:1, abaixo do AA: botão e selo destrutivos são vermelho cheio com texto branco (4,76:1). O `rounded-4xl` do `Badge` sai com 8px aqui (o `globals.css` prende os raios), então pílula pede `rounded-full` explícito. O Tailwind acha as classes em `web/lib/` sem `@source`.
- **Ler o Status no servidor para a transição administrativa.** O `de` vem da tela, capturado **no clique** e não no confirmar do `Dialog`: é o que deixa o compare-and-swap separar "já avançado" de "fora da tabela". E o desfecho mora fora da linha da Tabela, porque a releitura sob filtro a desmonta e o `Alert` sumiria justamente no caso da corrida.
- **Chamar `catalogo.Liberar`, `Reservar` ou `Consolidar` ao lado de uma mudança de Status.** Toda mudança de Status passa por `pedido.Transicionar`, e o efeito sobre o Estoque é dele: quem chama não repete. Quem precisa de desfecho (Comprador, Administrador) trava a linha com `FOR UPDATE` que espera (`TravarPedidoDoComprador`) **antes** do CAS; quem roda em tique (varredura, simulação) usa `SKIP LOCKED` e tenta no próximo.
- **Fazer `UPDATE` em `pedido.item_pedido` ou `pedido.transicao_status`.** Os gatilhos recusam com 23001 (e `pedido.pedido` só aceita `UPDATE` de `status`). Migração que preencha coluna nova dessas tabelas desliga o gatilho dentro dela mesma; teste que precise envelhecer o histórico usa `envelhecerHistorico` em `api/maquina_test.go`.
- **Pôr subteste depois de `simulacaoDeEntregaFR33` em `TestSessaoEProduto`.** Ele roda por último e deixa um Pedido que faz toda chamada seguinte a `SimularEntrega` devolver erro — é o que ele prova. Os subtestes da Tabela da 6.4 e da 6.5 também dependem da ordem (a 6.5 lê a Tabela filtrada com `por_pagina=100`). É o `epic-6-retro-item-32`; até ele fechar, subteste novo entra antes.
- **Escrever teste de corrida com as duas requisições soltas.** Elas não se sobrepõem, e a mutação que o teste existe para matar passa com a suíte verde. Segure a linha até `pg_stat_activity` mostrar as duas presas em trava, como em `api/nova_tentativa_test.go` e `api/cancelamento_test.go`. E a prova da trava do AD-5 é `travaDeReservaSerializa`, que disputa `catalogo.Reservar` direto: o teste de ponta a ponta passa mesmo sem o `FOR UPDATE`, porque `ProximoNumeroDoAno` serializa antes.
- **Gerar a `Idempotency-Key` no clique.** Ela nasce na entrada da Revisão e fica em `sessionStorage`, senão o duplo clique produz duas chaves e dois Pedidos. `uuidNovo` cai para `crypto.getRandomValues` porque `crypto.randomUUID` só existe em contexto seguro, e a apresentação pode rodar por IP da LAN.
- **Esperar repetição em deadlock em toda rota.** `emTransacao` (`api/transacao.go`) repete uma vez em `40P01`/`40001`, mas só `criarPedido` e `entrarNoCheckout` a usam. Rota nova que trave mais de uma linha passa por ela.
- **Gravar a ciência do preço fora do checkout, ou por releitura.** `carrinho.Itens` é leitura pura (AD-17), e `carrinho.ConfirmarPrecoVisto`, chamada só por `pedido`, grava exatamente os preços reportados — uma releitura já pode ser outro preço, e a mudança sumiria antes de aparecer.
- **Usar `unnest(a, b)` de dois argumentos numa consulta.** O sqlc v1.31.1 o recusa (no Postgres é `ROWS FROM`, não função): são dois `unnest ... WITH ORDINALITY` juntados pela posição.
- **Desabilitar botão com `disabled` quando há razão a dizer.** Botão nativo desabilitado não é focalizável e a razão por `aria-describedby` nunca chega ao leitor de tela: use `aria-disabled`. E efeito que dispara requisição corre uma vez só (`useRef`), senão o duplo disparo do StrictMode cala o primeiro aviso em `dev`.
- **Confiar no `data-checked:` do `radio-group.tsx`.** O Radix instalado não põe esse atributo; o checkout estiliza o marcado com `has-[[data-state=checked]]`.
- **Calcular o vencimento da Tentativa de Pagamento em outro lugar.** `expira_em` na tela e `pedido.Expirar` na varredura saem do mesmo `max(ocorrido_em)` do histórico mais o prazo da Config (`pedido.ExpiraEm`); instante próprio faz a tela e a varredura discordarem.
- **Escrever o isento de Frete como `>`.** É `subtotal >= AZAMON_FRETE_ISENCAO_CENTAVOS`, o mesmo ponto em que o Carrinho cala o "Faltam R$ X"; com `>`, a Revisão cobra de quem o Carrinho acabou de isentar.
- **Ajustar o Estoque total pelo `PUT` do Produto.** Tem rota própria (`PUT /api/v1/admin/produtos/{id}/estoque`): a tela reenvia a linha lida ao desativar, e isso regravaria um total velho por cima de uma consolidação.
- **Reescrever um compose de rede desconectada.** O `docker-compose.offline.yml` foi escrito e descartado na 1.9: com `internal: true` o Docker descarta a publicação de 3000 e 8080 em silêncio. O ensaio é desligar a rede à mão (`addendum.md` §10).

## Para colar numa sessão nova

```
Projeto Azamon: réplica da Amazon, trabalho de faculdade.
Go 1.27 + Postgres 18 + Redis 8 + Docker, front em Next.js 16 sobre React 19.
PRD, UX, arquitetura e spec estão finalizados e reconciliados.
Leia HANDOFF.md primeiro. O contrato é _bmad-output/specs/spec-azamon/SPEC.md,
com 8 CAPs de ID estável e o companions: que lista o resto — inclusive a
ARCHITECTURE-SPINE.md, com 20 ADs de ID estável.
Épicas 1 a 6 fechadas: 48 estórias done em main. Os defeitos dos passeios
das Épicas 5 e 6 (D1–D7, E1–E5) estão corrigidos; do passeio só pelo
teclado ficaram K1 e K2, de severidade baixa.
Próximo passo: bmad-build dos testes que faltam (retro items 26, 32, 33),
depois bmad-build da Estória 7.1.
```

*Este arquivo não é carregado automaticamente por agentes — o `AGENTS.md` da raiz é. Ele carrega as armadilhas de maior consequência e aponta para cá; as de escopo estreito, como não renumerar as suposições do §16, vivem só aqui. Depois de mudança significativa, refresque com `bmad-project-context`.*
