<!-- bmad:context -->
<!-- Verificado em 2026-09-06 contra 7aa9c83. Gerenciado por bmad-project-context; edições dentro deste bloco são substituídas no refresh. O que você quiser preservar, deixe fora dos marcadores. -->

## Azamon

Réplica do núcleo de comércio eletrônico da Amazon, trabalho de faculdade para 2 a 4 pessoas em um semestre, avaliado em funcionalidade, arquitetura e apresentação ao vivo. Stack: Go 1.27, PostgreSQL 18, Redis 8, Docker obrigatório, e Next.js 16 sobre React 19 no front-end. PRD, UX e arquitetura estão finalizados e reconciliados entre si; nada foi construído ainda. Todo o planejamento vive em `_bmad-output/planning-artifacts/`; `HANDOFF.md` na raiz é a porta de entrada.

## Policy

- Alterou `prd.md`, `addendum.md`, uma espinha de UX ou a espinha de arquitetura? Registre a decisão no `.memlog.md` do mesmo diretório, com `uv run _bmad/scripts/memlog.py append` — o memlog é o registro canônico e vence o documento quando os dois divergirem.
- Os `AD-1`..`AD-20` da espinha de arquitetura têm **ID estável**: emende a regra no lugar, acrescente o próximo `AD-n` para decisão nova, nunca renumere nem reaproveite um ID retirado. As épicas e as histórias citam por ID.

## Onde as coisas estão

- `HANDOFF.md` — setup, decisões travadas, bloqueios com dono, e as armadilhas que não couberam aqui.
- PRD: `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/prd.md` — comece pelo Sumário Executivo, que tem mapa de leitura por público; índice de requisitos no §4.
- `addendum.md`, mesmo diretório — decisões técnicas, invariantes, alternativas descartadas, ordem de construção. Insumo direto de `bmad-architecture`.
- UX: `_bmad-output/planning-artifacts/ux-designs/ux-azamon-2026-09-02/` — `DESIGN.md` (identidade) e `EXPERIENCE.md` (comportamento) são espinhas; o que está em `mockups/` só ilustra e perde em conflito.
- Arquitetura: `_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/ARCHITECTURE-SPINE.md` — 20 invariantes, a decomposição em seis módulos, o contrato de resposta e a tabela Stack com versões verificadas. `DIAGRAMA-MODULOS.md` no mesmo diretório é o entregável do §6.1 e lê-se sozinho. Onde a espinha e o addendum falarem do mesmo assunto, a espinha decide o mecanismo e o addendum guarda o porquê.

## Rodando e verificando

- Todo script do BMad roda por `uv run` — `uv run _bmad/scripts/memlog.py ...`. Sem o prefixo, roda fora do ambiente do projeto e falha.
- `docker compose up` sobe os quatro serviços — `web` (Next.js), `azamon` (Go), `postgres`, `redis` — já em modo demonstração, aplicando migrações e o Catálogo Semeado no arranque do binário. É o alvo do NFR-1 e não tem argumento a lembrar.
- `go test ./...` roda a suíte: máquina de estados em memória, e PostgreSQL real via `testcontainers-go` onde a transação é o objeto do teste.
- Mudou SQL? `sqlc generate`. Migração nova? `goose create` — versionamento por *timestamp*, nunca `NNNN` sequencial, que colide com duas pessoas trabalhando no mesmo dia.

## Convenções que divergem do padrão

- Português em tudo que sai: documento, interface e apresentação. As mensagens de commit seguem conventional commits em inglês.

## Armadilhas conhecidas

- Nunca troque um dos 20 termos do §3 do PRD por sinônimo, em documento ou em tela — "Pedido" e não "compra", "Produto" e não "item". Duas revisões já caçaram violações disso.
- Nunca puxe fonte, CSS ou script da rede: o NFR-15 exige percorrer o Roteiro A com a rede desconectada, então nada de Google Fonts, `@import` remoto ou CDN — a fonte é auto-hospedada em `web/public/fonts/`. Falha em silêncio, na sala. O `AD-12` manda um passo de CI fazer grep por `fonts.googleapis`, `@import url(http`, `//cdn.` e **`next/font/google`**, que é a forma que se escreve sem perceber.
- Nunca some as reservas antes de segurar a trava. São dois comandos, nesta ordem: `SELECT … FROM catalogo.produto WHERE id = ANY($1) ORDER BY id FOR UPDATE`, e **só então** a soma das reservas ativas. Invertido, vende a mesma última unidade duas vezes — que é exatamente o que o teste do NFR-7 dispara. O `ORDER BY id` evita que dois Pedidos com os mesmos dois Produtos em ordem inversa travem um no outro (`AD-5`, `AD-4`).
- Nunca ponha regra de negócio no Node. O Next.js é casca: sem Server Actions, sem Route Handlers com regra, sem acesso a banco. Um caminho de dado só, navegador → `/api` → Go (`AD-10`). Duas armadilhas do Next 16 que falham calado: `middleware.ts` virou `proxy.ts` e o nome antigo é ignorado sem erro de build; e `next/image` recusa upstream de IP privado, então as imagens do Catálogo Semeado usam `unoptimized`.
- Nunca trate "sem reserva ativa" como erro. `catalogo.Liberar` devolve `nil`, `Consolidar` sobre reserva já consolidada é no-op, e `pedido` chama `Liberar` **incondicionalmente** em toda transição para `CANCELADO` e em toda recusa. Tratado como erro, cancelar um Pedido em `PAGAMENTO_RECUSADO` falha sempre, com rollback.
- Nunca use o verde ou o laranja do `DESIGN.md` para enfeitar. Verde é Estoque disponível e `ENTREGUE`; laranja aparece uma vez por fluxo, no passo irreversível. Decorar com eles apaga o único mecanismo semântico de cor do sistema.
- Não comece a construção por camada horizontal. O addendum §8 tem um passo 0 — esqueleto vertical, um Produto e um Comprador e um Pedido até `ENTREGUE`, com telas feias — antes de alargar qualquer camada.
- Não corte FR-19, FR-24, FR-34 nem o teste de concorrência do NFR-7 por não aparecerem em tela nenhuma: são o que separa o sistema da maquete. A ordem de corte legítima está no §6.3 do PRD.
- Não trate o `addendum.md` como rascunho. Está `final`, e a SM-7 exige que continue vivo: decisão de arquitetura tomada durante a construção volta para ele.

<!-- /bmad:context -->
