# Azamon — Handoff

**Atualizado:** 2026-09-04 · **Estado:** PRD e UX finalizados, nada construído ainda.

Réplica da Amazon como trabalho de faculdade. Time de 2 a 4 pessoas, um semestre, avaliado em três eixos ao mesmo tempo: funcionalidade entregue, arquitetura e documentação, e apresentação ao vivo.

## Setup

O repositório versiona **o trabalho**, não a ferramenta. Depois de clonar você tem os documentos, mas não tem o BMad nem as skills — os dois são instalados e somam mais de 2 MB de código de terceiros.

**Pré-requisito:** [`uv`](https://docs.astral.sh/uv/). Todo script do BMad roda por `uv run`; sem ele nenhuma skill funciona.

| O quê | De onde | Como saber que deu certo |
|---|---|---|
| **BMad Method 6.11.0** | Instalador oficial — [docs.bmad-method.org](https://docs.bmad-method.org). Use **a mesma versão**, senão os caminhos de configuração divergem | `_bmad/scripts/memlog.py` existe e `.claude/skills/` tem os `bmad-*` |
| **Skills de frontend** (opcional) | `skills-lock.json` na raiz: 14 skills de `Leonxlnx/taste-skill`, com hash | `.claude/skills/design-taste-frontend/` existe |

Ao instalar o BMad ele pergunta nome do projeto e idioma. Responda **azamon** e **Português** — é o que está em `_bmad/config.toml` hoje, e o que faz os artefatos saírem no lugar certo.

As skills de frontend **não são usadas neste projeto** e estão no `skills-lock.json` só para reprodutibilidade — o porquê está no memlog da UX. Pular a segunda linha da tabela não quebra nada.

> Quem fizer a primeira instalação limpa: anote aqui o comando exato que funcionou. Vale mais que este parágrafo.

## Onde está o quê

| Arquivo | O que é |
|---|---|
| `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md` | O PRD. Comece pelo **Sumário Executivo**, que tem um mapa de leitura por público |
| `.../addendum.md` | Decisões técnicas, invariantes, 8 alternativas descartadas com o que se perdeu, ordem de construção. Insumo direto da arquitetura |
| `.../.memlog.md` | 74 decisões em ordem cronológica, com o motivo de cada uma. É o registro canônico — se o PRD e a sua memória divergirem, vale isto |
| `.../review-rubric.md`, `.../review-consistency.md` | Revisões preservadas. Restam 14 achados médios e 12 baixos, nenhum bloqueante |
| `_bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/DESIGN.md` | Identidade visual. shadcn/ui mais uma camada de marca de dez cores. Espinha, não sugestão |
| `.../EXPERIENCE.md` | Comportamento: IA, estados, interações, piso AA, e as quatro jornadas com a camada de interface |
| `.../mockups/` | Três telas em HTML que abrem offline. Ilustram; as espinhas vencem em conflito |
| `.../.memlog.md` | 40 decisões da UX. Mesmo papel do memlog do PRD: é o registro canônico |
| `.../review-rubric.md`, `.../review-adversarial.md` | As duas revisões da UX. 52 achados, todos resolvidos |

Não relea tudo: o PRD tem índice de requisitos no §4 e mapa de leitura no topo.

## Decisões travadas

Estão no Sumário Executivo do PRD. As quatro estruturais, em uma linha: marketplace no modelo de dados mas sem portal de vendedor · pagamento simulado porém assíncrono, com expiração e idempotência · máquina de estados do Pedido como peça central · busca é texto mais filtros, não motor de busca.

Stack imposta: **Go** no back-end, **Postgres + Redis**, **Docker obrigatório**. Front-end em aberto.

## Bloqueios

| # | O quê | Dono | Destrava o quê |
|---|---|---|---|
| 1 | **Next.js ou React** | o time | `bmad-architecture`. Não muda uma linha do PRD |
| 2 | **Enunciado ou rubrica do professor** | Sung | Quatro suposições do §16 (portal do vendedor, verificação de e-mail, devolução, pedido dividido por vendedor). Ao obter: rodar `bmad-prd` em modo atualização, que reconcilia contra o memlog |

Nenhum dos dois impede começar a arquitetura das partes que não dependem do front.

## Próximo passo

`bmad-architecture`, alimentado por `prd.md` + `addendum.md` + as duas espinhas de UX. Depois `bmad-create-epics-and-stories`, depois `bmad-build`.

## O que uma sessão nova erraria

- **Disciplina de glossário.** O §3 do PRD define 20 termos usados literalmente no documento inteiro. Trocar por sinônimo é defeito, não estilo — nunca "compra" no lugar de **Pedido**, nunca "item" no lugar de **Produto**. Duas revisões já caçaram violações disso.
- **Começar horizontal.** O addendum §8 tem um **passo 0**: esqueleto vertical (um Produto, um Comprador, um Pedido até `ENTREGUE`, telas feias) antes de alargar qualquer camada. A ordem por dependência que vem depois é correta e perigosa se seguida sozinha — produz um sistema que só funciona no fim do semestre.
- **Cortar o que é invisível.** FR-19, FR-24, FR-34 e o teste de concorrência do NFR-7 não aparecem em tela nenhuma e são o que separa o sistema da maquete. A ordem de corte legítima está no §6.3.
- **Renumerar suposições.** O §16 foi reordenado por risco; referências a "suposição N" já quebraram uma vez por causa disso.
- **Achar que o addendum é rascunho.** Ele está `final`, mas a SM-7 exige que continue vivo: toda decisão de arquitetura tomada durante a construção volta para ele.
- **Construir o checkout em quatro passos.** O §9 do PRD enumera Endereço → Frete → Revisão → Pagamento, mas a UX o colapsou em **dois** (Endereço → Revisão): o Frete é inteiramente derivado do CEP e o passo Pagamento não coleta dado nenhum, porque o §8 proíbe dado de cartão até no Provedor Simulado. **Os dois documentos divergem de propósito e a UX é a mais recente.** Reconciliar com `bmad-correct-course` ou `bmad-prd` em modo atualização. Não bloqueia a arquitetura.
- **Buscar fonte na rede.** O NFR-15 exige percorrer o Roteiro A com a rede desconectada. Isso proíbe Google Fonts, `@import` remoto e CDN — a fonte é auto-hospedada no repositório. É o erro mais fácil de cometer e quebra a demonstração em silêncio, na sala.
- **Usar o verde ou o laranja para enfeitar.** No `DESIGN.md` os dois têm sentido fechado: verde é Estoque disponível e `ENTREGUE`; laranja aparece **uma vez por fluxo**, no passo irreversível. Usar qualquer um decorativamente apaga o único mecanismo semântico de cor do sistema.

## Para colar numa sessão nova

```
Projeto Azamon: réplica da Amazon, trabalho de faculdade, Go + Postgres + Redis + Docker.
O PRD está finalizado em _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md,
com addendum técnico e memlog no mesmo diretório. Leia HANDOFF.md primeiro.
Próximo passo: [arquitetura / épicas / implementar X].
```

*Este arquivo não é carregado automaticamente por agentes. Se quiser que seja, `bmad-project-context` gera o bloco AGENTS.md do repositório.*
