<!-- bmad:context -->
<!-- Verificado em 2026-09-05 contra 5c53317. Gerenciado por bmad-project-context; edições dentro deste bloco são substituídas no refresh. O que você quiser preservar, deixe fora dos marcadores. -->

## Azamon

Réplica do núcleo de comércio eletrônico da Amazon, trabalho de faculdade para 2 a 4 pessoas em um semestre, avaliado em funcionalidade, arquitetura e apresentação ao vivo. Stack imposta: Go, Postgres, Redis, Docker obrigatório; front-end ainda não decidido entre Next.js e React. Todo o planejamento vive em `_bmad-output/planning-artifacts/`; `HANDOFF.md` na raiz é a porta de entrada.

## Policy

- Alterou `prd.md`, `addendum.md` ou uma espinha de UX? Registre a decisão no `.memlog.md` do mesmo diretório, com `uv run _bmad/scripts/memlog.py append` — o memlog é o registro canônico e vence o documento quando os dois divergirem.

## Onde as coisas estão

- `HANDOFF.md` — setup, decisões travadas, bloqueios com dono, e as armadilhas que não couberam aqui.
- PRD: `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md` — comece pelo Sumário Executivo, que tem mapa de leitura por público; índice de requisitos no §4.
- `addendum.md`, mesmo diretório — decisões técnicas, invariantes, alternativas descartadas, ordem de construção. Insumo direto de `bmad-architecture`.
- UX: `_bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/` — `DESIGN.md` (identidade) e `EXPERIENCE.md` (comportamento) são espinhas; o que está em `mockups/` só ilustra e perde em conflito.

## Rodando e verificando

- Todo script do BMad roda por `uv run` — `uv run _bmad/scripts/memlog.py ...`. Sem o prefixo, roda fora do ambiente do projeto e falha.
- TODO, à espera da decisão Next.js vs React: comandos de build, teste e subida. O alvo do NFR-1 é `docker compose up` levantando Go, Postgres e Redis em um comando, com o Catálogo Semeado.

## Convenções que divergem do padrão

- Português em tudo que sai: documento, interface e apresentação. As mensagens de commit seguem conventional commits em inglês.

## Armadilhas conhecidas

- Nunca troque um dos 20 termos do §3 do PRD por sinônimo, em documento ou em tela — "Pedido" e não "compra", "Produto" e não "item". Duas revisões já caçaram violações disso.
- Nunca puxe fonte, CSS ou script da rede: o NFR-15 exige percorrer o Roteiro A com a rede desconectada, então nada de Google Fonts, `@import` remoto ou CDN — a fonte é auto-hospedada no repositório. Falha em silêncio, na sala. Nascido o front-end, troque esta linha por um check de CI que faça grep por `fonts.googleapis`, `@import url(http` e `//cdn.`.
- Nunca use o verde ou o laranja do `DESIGN.md` para enfeitar. Verde é Estoque disponível e `ENTREGUE`; laranja aparece uma vez por fluxo, no passo irreversível. Decorar com eles apaga o único mecanismo semântico de cor do sistema.
- Não comece a construção por camada horizontal. O addendum §8 tem um passo 0 — esqueleto vertical, um Produto e um Comprador e um Pedido até `ENTREGUE`, com telas feias — antes de alargar qualquer camada.
- Não corte FR-19, FR-24, FR-34 nem o teste de concorrência do NFR-7 por não aparecerem em tela nenhuma: são o que separa o sistema da maquete. A ordem de corte legítima está no §6.3 do PRD.
- Não trate o `addendum.md` como rascunho. Está `final`, e a SM-7 exige que continue vivo: decisão de arquitetura tomada durante a construção volta para ele.

<!-- /bmad:context -->
