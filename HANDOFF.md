# Azamon — Handoff

**Atualizado:** 2026-09-11 · **Estado:** PRD, UX, arquitetura, spec e épicas finalizados e reconciliados entre si. As Estórias 1.1 e 1.2 estão feitas e mescladas (PR #5 e #6) — o andaime sobe com `docker compose up` e o Next já é casca, com `rewrites()`, shadcn e a base visual da marca. A 1.3 é o próximo passo.

Réplica da Amazon como trabalho de faculdade. Time de 2 a 4 pessoas, um semestre, avaliado em três eixos ao mesmo tempo: funcionalidade entregue, arquitetura e documentação, e apresentação ao vivo.

## Setup

Está em [`README.md`](README.md), com os comandos exatos e como verificar cada um: o
repositório versiona **o trabalho**, não a ferramenta — depois de clonar faltam o BMad
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

**Cinco verificações antes de alargar qualquer camada.** A espinha tem uma seção de Questões em Aberto com quatro checagens de dez minutos — nenhuma é decisão, todas são "confirmar que funciona como assumimos". Faça-as no passo 0. **Duas já fecharam na Estória 1.2, as duas positivas:** o `Set-Cookie` do Go atravessa o `rewrites()` do Next íntegro, então a saída de *proxy* explícito não precisa ser acionada; e o `npx shadcn init` roda limpo em Next 16 + React 19, sem flag de dependências de pares. Os desfechos estão no `addendum.md` §10. Restam a análise de `DEFAULT uuidv7()` pelo sqlc e o p95 da busca com 5.000 Produtos.

## Próximo passo

`bmad-build` na **Estória 1.3** — a primeira migração e o Catálogo Semeado determinístico. As duas anteriores
estão feitas: a **1.1** (PR #5) trouxe a árvore do `AD-1`, os quatro contêineres e as quatro peças transversais
de `internal/plataforma`; a **1.2** (PR #6) trouxe o `rewrites()` de `/api` para o Go, os 15 componentes do
shadcn, os dez tokens de marca e a fonte auto-hospedada. As sete épicas e as 52 estórias estão em
`_bmad-output/planning-artifacts/epics.md`, e o rastreamento de sprint em
`_bmad-output/implementation-artifacts/sprint-status.yaml`.

### Quanto custa uma estória, e o que fazer com isso

A 1.1 consumiu ~795.000 tokens de subagente. **As três lentes de revisão adversarial do `bmad-build`, mais a
rodada de patches que elas geram, foram 56% disso** — e escalam com o tamanho do diff, não com a dificuldade da
estória. Sobram 51 estórias; nesse ritmo o orçamento não fecha.

A 1.1 foi o pior caso — greenfield, a maior estória do épico, e 14 lacunas de documentação que precisaram virar
decisão nomeada (estão no addendum §10). Mas a parte cara é justamente a que se repete. Três medidas:

- **Três lentes só onde há raio de alcance real:** reserva de Estoque sob concorrência (5-6, 5-7), máquina de
  estados do Pedido (5-1), webhook e idempotência (5-9). Para CRUD de mesma forma, uma lente basta.
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

## Para colar numa sessão nova

```
Projeto Azamon: réplica da Amazon, trabalho de faculdade.
Go 1.27 + Postgres 18 + Redis 8 + Docker, front em Next.js 16 sobre React 19.
PRD, UX, arquitetura e spec estão finalizados e reconciliados.
Leia HANDOFF.md primeiro. O contrato é _bmad-output/specs/spec-azamon/SPEC.md,
com 8 CAPs de ID estável e o companions: que lista o resto — inclusive a
ARCHITECTURE-SPINE.md, com 20 ADs de ID estável.
Próximo passo: [épicas / implementar X].
```

*Este arquivo não é carregado automaticamente por agentes — o `AGENTS.md` da raiz é. Ele carrega as armadilhas de maior consequência e aponta para cá; as de escopo estreito, como não renumerar as suposições do §16, vivem só aqui. Depois de mudança significativa, refresque com `bmad-project-context`.*
