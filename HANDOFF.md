# Azamon — Handoff

**Atualizado:** 2026-08-16 · **Estado:** PRD finalizado, nada construído ainda.

Réplica da Amazon como trabalho de faculdade. Time de 2 a 4 pessoas, um semestre, avaliado em três eixos ao mesmo tempo: funcionalidade entregue, arquitetura e documentação, e apresentação ao vivo.

## Onde está o quê

| Arquivo | O que é |
|---|---|
| `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md` | O PRD. Comece pelo **Sumário Executivo**, que tem um mapa de leitura por público |
| `.../addendum.md` | Decisões técnicas, invariantes, 8 alternativas descartadas com o que se perdeu, ordem de construção. Insumo direto da arquitetura |
| `.../.memlog.md` | 74 decisões em ordem cronológica, com o motivo de cada uma. É o registro canônico — se o PRD e a sua memória divergirem, vale isto |
| `.../review-rubric.md`, `.../review-consistency.md` | Revisões preservadas. Restam 14 achados médios e 12 baixos, nenhum bloqueante |

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

`bmad-architecture`, alimentado por `prd.md` + `addendum.md`. Depois `bmad-create-epics-and-stories`, depois `bmad-build`.

## O que uma sessão nova erraria

- **Disciplina de glossário.** O §3 do PRD define 20 termos usados literalmente no documento inteiro. Trocar por sinônimo é defeito, não estilo — nunca "compra" no lugar de **Pedido**, nunca "item" no lugar de **Produto**. Duas revisões já caçaram violações disso.
- **Começar horizontal.** O addendum §8 tem um **passo 0**: esqueleto vertical (um Produto, um Comprador, um Pedido até `ENTREGUE`, telas feias) antes de alargar qualquer camada. A ordem por dependência que vem depois é correta e perigosa se seguida sozinha — produz um sistema que só funciona no fim do semestre.
- **Cortar o que é invisível.** FR-19, FR-24, FR-34 e o teste de concorrência do NFR-7 não aparecem em tela nenhuma e são o que separa o sistema da maquete. A ordem de corte legítima está no §6.3.
- **Renumerar suposições.** O §16 foi reordenado por risco; referências a "suposição N" já quebraram uma vez por causa disso.
- **Achar que o addendum é rascunho.** Ele está `final`, mas a SM-7 exige que continue vivo: toda decisão de arquitetura tomada durante a construção volta para ele.

## Para colar numa sessão nova

```
Projeto Azamon: réplica da Amazon, trabalho de faculdade, Go + Postgres + Redis + Docker.
O PRD está finalizado em _bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md,
com addendum técnico e memlog no mesmo diretório. Leia HANDOFF.md primeiro.
Próximo passo: [arquitetura / épicas / implementar X].
```

*Este arquivo não é carregado automaticamente por agentes. Se quiser que seja, `bmad-project-context` gera o bloco AGENTS.md do repositório.*
