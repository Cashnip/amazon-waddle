---
title: 'Percurso do addendum contra o código (Estória 7.3)'
created: '2026-09-25'
baseline_commit: '05a3682d589d9d696d85a0aef5a1f7da1546b3e3'
spec: 'spec-7-3-o-addendum-percorrido-contra-o-codigo.md'
---

# Percurso do addendum contra o código

A prova da SM-7. Cada decisão de `addendum.md` §1–§10, na redação de `05a3682`, conferida contra o código de `main` em 2026-09-25, com um veredito e a evidência que o sustenta. Seis leitores em paralelo, um por faixa, só leitura; toda linha que não saiu `confirmada` foi reconferida à mão, e mais uma `confirmada` por bloco, por amostra. `Lnnn` é a linha do addendum **antes** das notas da 7.3, e toda linha de código citada nas tabelas de vereditos é a de `05a3682`; só a tabela `corrigida-no-código`, ao fim, usa a numeração de depois das correções. As notas no addendum são sub-itens `  - *(7.3: …)*` logo abaixo do item; as duas exceções são as linhas da tabela do §9, que não comportam sub-item e ganharam um parágrafo logo depois dela, e a nota de uma troca usa "substituída pela N.M", enquanto a de uma ressalva usa "precisão" ou "correção".

| Veredito | Quantas | O que quer dizer |
|---|---|---|
| `confirmada` | 225 | o código faz o que a decisão diz |
| `substituída-registrada` | 10 | o código mudou, e uma entrada posterior do §10 já dizia por quê — a 7.3 pôs a nota no item antigo |
| `substituída-agora` | 9 | o código mudou e nada dizia por quê — nota no item e o porquê na entrada da 7.3 |
| `histórica` | 16 | fato de processo ou medida, ancorado em commit ou spec |
| `contradita` | **0** | — |
| **Total** | **260** | |

Além dos vereditos, comentários de código mentiam em onze lugares sobre um comportamento que estava certo, e foram corrigidos (`corrigida-no-código`, ao fim). E a espinha tinha três contradições conhecidas (AD-4, AD-6, AD-8), emendadas no lugar, e duas novas achadas no percurso (AD-18 e a Semente Estrutural), que ficaram em `deferred-work.md` porque a spec desta estória só emenda o 4, o 6 e o 8.

## §1–§9

| ref | decisão | veredito | evidência |
|---|---|---|---|
| §1#1 (L18) | Back-end em Go | confirmada | `go.mod:3` |
| §1#2 (L19) | Next.js 16 sobre React 19 | confirmada | `web/package.json:15,18` |
| §1#3 (L20) | PostgreSQL | confirmada | `docker-compose.yml:32` |
| §1#4 (L21) | Redis | confirmada | `docker-compose.yml:47` |
| §1#5 (L22) | Docker obrigatório | confirmada | `docker-compose.yml:3-52`, `Dockerfile`, `web/Dockerfile` |
| §1#6 (L24) | Carrinho no PostgreSQL | confirmada | `db/migracoes/20260918160000_carrinho_schema.sql:8-12` |
| §1#7 (L28) | Redis só para Sessão, token e bloqueio | confirmada | `internal/identidade/sessao.go:34`; `redefinicao.go:64-67`; `bloqueio.go:53-56` |
| §1#8 (L29) | Cache de vitrine recusado | confirmada | `internal/plataforma/redis.go:13` é o único cliente |
| §1#9 (L30) | Fila recusada; varredura no PostgreSQL | confirmada | `cmd/azamon/main.go:155-188` |
| §1#10 (L32) | Redis com um emprego nomeado | confirmada | `docker-compose.yml:46-52` |
| §1#11 (L34) | Monólito modular, pacote por domínio | confirmada | `internal/fronteira_test.go:59-73` |
| §2#1 (L44) | Criação com Reserva tudo ou nada | confirmada | `internal/pedido/maquina.go:126`; `internal/catalogo/catalogo.go:286-303` |
| §2#2 (L45) | → PAGO mantém a Reserva | confirmada | `internal/pedido/maquina.go:127` |
| §2#3 (L46) | → RECUSADO libera a Reserva | confirmada | `internal/pedido/maquina.go:128` |
| §2#4 (L47) | Nova Tentativa cria Reserva, falha se indisponível | confirmada | `internal/pedido/maquina.go:130,282-297` |
| §2#5 (L48) | PAGO → SEPARANDO por Administrador ou simulação | confirmada | `internal/pedido/maquina.go:131` |
| §2#6 (L49) | SEPARANDO → ENVIADO consolida | confirmada | `internal/pedido/maquina.go:132,249-250` |
| §2#7 (L50) | ENVIADO → ENTREGUE | confirmada | `internal/pedido/maquina.go:133` |
| §2#8 (L51) | Expiração → RECUSADO libera | confirmada | `internal/pedido/maquina.go:129`; `pedido.go:1078-1086` |
| §2#9 (L52) | Cancelamento de quatro origens libera | confirmada | `internal/pedido/maquina.go:114-116,134,245-248` |
| §2#10 (L56) | Só o Status muda | confirmada | `internal/pedido/db/consultas.sql:68`; `db/migracoes/20260919120000_pedido_maquina_de_estados.sql:71-79` |
| §2#11 (L57) | Transição inválida falha alto | confirmada | `internal/pedido/maquina.go:68,208-214` |
| §2#12 (L58) | Toda transição escreve histórico | confirmada | `internal/pedido/maquina.go:234-242` |
| §2#13 (L59) | Efeito na mesma transação | confirmada | `internal/pedido/maquina.go:244-253` |
| §2#14 (L60) | Transição condicionada ao estado esperado | confirmada | `internal/pedido/maquina.go:224-229` |
| §2#15 (L61) | Tempo deriva do histórico | confirmada | `internal/pedido/pedido.go:1106-1118,1160` |
| §2#16 (L62) | Confirmação tardia sinalizada, sem ressuscitar | confirmada | `internal/pedido/pedido.go:965,985-994` |
| §3#1 (L68) | Disponível derivado; Reserva de vida curta | confirmada | `db/migracoes/20260918140000_catalogo_estoque_disponivel.sql:16-19` |
| §3#2 (L69) | Trava por Produto na transação | confirmada | `internal/catalogo/db/consultas.sql:23-29` |
| §3#3 (L70) | Criação atômica sobre todos os itens | confirmada | `internal/catalogo/catalogo.go:286-303` |
| §3#4 (L71) | Teste do NFR-7 | confirmada | `api/concorrencia_test.go:121-122` |
| §3#5 (L73) | Trava pessimista basta | confirmada | `internal/catalogo/db/consultas.sql:29` |
| §4#1 (L79) | "Uma interface" com duas operações | substituída-agora | `internal/pagamento/pagamento.go:64,135,309`: sem `interface` Go; AD-8 emendado |
| §4#2 (L80) | Centavos decidem só na 1ª Tentativa | confirmada | `internal/pagamento/pagamento.go:77-89` |
| §4#3 (L81) | Confirmação fora da requisição | confirmada | `cmd/azamon/main.go:184,195-224` |
| §4#4 (L82) | Confirmação idempotente por chave | confirmada | `db/migracoes/20260912130000_pagamento_tentativa_e_confirmacao.sql:38` |
| §4#5 (L83) | Tela sem recarregar, por polling | confirmada | `web/app/pedidos/[id]/acompanhamento.tsx:255-269` |
| §4#6 (L87) | Prazo e varredura | confirmada | `internal/pedido/pedido.go:1078-1086` |
| §4#7 (L88) | Tardia: registrar e sinalizar | confirmada | `internal/pedido/pedido.go:965,1017-1019` |
| §4#8 (L90) | Stripe troca só implementação e config | confirmada | `pagamento.Simulado` só em `cmd/azamon/main.go:159` |
| §5#1 (L94) | Frete é dado, com isenção e região padrão | confirmada | `db/migracoes/20260919160000_pedido_faixa_frete.sql:14-51`; `internal/pedido/frete.go:87-94` |
| §6#1 (L100) | Item de Pedido registra o Vendedor | confirmada | `db/migracoes/20260912120000_pedido_esqueleto.sql:40` |
| §6#2 (L101) | Categoria com pai opcional | confirmada | `db/migracoes/20260911120200_catalogo_schema.sql:20` |
| §7#1 (L111) | Marketplace completo descartado | confirmada | `internal/identidade/identidade.go:1` (só Comprador e Administrador têm conta); o Vendedor é dado de Catálogo, `db/migracoes/20260911120200_catalogo_schema.sql:7-12` |
| §7#2 (L112) | Portal do Vendedor fora | confirmada | `internal/identidade/identidade.go:1` |
| §7#3 (L113) | Loja sem Vendedor descartada | confirmada | `db/migracoes/20260911120200_catalogo_schema.sql:7-12` |
| §7#4 (L114) | Gateway real descartado | confirmada | `internal/pagamento/pagamento.go:62-67` |
| §7#5 (L115) | Pagamento síncrono descartado | confirmada | `cmd/azamon/main.go:184` |
| §7#6 (L116) | Carrinho anônimo descartado | confirmada | `db/migracoes/20260918160000_carrinho_schema.sql:10-12` |
| §7#7 (L117) | Elasticsearch descartado | confirmada | `docker-compose.yml:3-52` (quatro serviços); `db/migracoes/20260911120100_catalogo_pg_trgm.sql` |
| §7#8 (L118) | Microsserviços descartados | confirmada | `cmd/azamon/main.go:121` |
| §7#9 (L119) | Pedido não dividido por Vendedor | confirmada | `db/migracoes/20260912120000_pedido_esqueleto.sql:12-29` |
| §8#1 (L123) | Passo 0, esqueleto vertical | histórica | `db/migracoes/20260912120000_pedido_esqueleto.sql` |
| §8#2 (L127) | Ordem de alargamento | histórica | `4452994` (2.1), `bd3b49c` (3.1), `5f5ff69` (4.1), `a454444` (5.1): o Carrinho veio antes da máquina completa |
| §8#3 (L135) | Fronteiras para paralelizar | histórica | conselho de processo, sem artefato; addendum §8 |
| §9#1 (L143) | Next é casca, uma origem | confirmada | `web/next.config.ts:13-14` |
| §9#2 (L144) | Schema por módulo, sem FK cruzada | confirmada | `db/schema_test.go:237-247` |
| §9#3 (L145) | Um arquivo público por módulo | substituída-agora | `internal/identidade/endereco.go:64` chamado por `internal/pedido/pedido.go:167`; `f7994c2` (2.5) |
| §9#4 (L146) | Intervalo em três superfícies e mais nenhuma | substituída-agora | `web/app/admin/pedidos/[id]/detalhe-admin.tsx:84-90`; `081a60e` (6.4). O porquê é inferido pela 7.3: a spec da 6.4 não o registra |
| §9#5 (L147) | Testes híbridos | confirmada | `internal/pedido/maquina_test.go:22-30`; `api/concorrencia_test.go` |
| §9#6 (L148) | goose por timestamp, embed, no arranque | confirmada | `db/embutido.go:13-14`; `cmd/azamon/main.go:71,78` |
| §9#7 (L152) | Catálogo dono do Estoque e da Reserva | confirmada | `db/migracoes/20260912120100_catalogo_reserva_estoque.sql:14-18` |
| §9#8 (L153) | Trava antes da soma | confirmada | `internal/catalogo/catalogo.go:266-277` |
| §9#9 (L154) | Webhook e inbox; pagamento não conhece pedido | confirmada | `internal/pagamento/pagamento.go:3-7,166-185` |
| §9#10 (L155) | Emitir mora em pagamento | confirmada | `internal/pagamento/pagamento.go:309-338` |
| §9#11 (L156) | Token no log | confirmada | `api/redefinicao.go:60` |

## §10 — Épica 1 (1.1–1.9)

| ref | decisão | veredito | evidência |
|---|---|---|---|
| 1.1#1 (L164) | `plataforma` importável por todos | confirmada | `internal/fronteira_test.go:45,59,72` |
| 1.1#2 (L165) | Convenção `AZAMON_*` | confirmada | `.env:10-12,23-25`; as "23" variáveis hoje são 36, acréscimos registrados (L205, L249) |
| 1.1#3 (L166) | Credenciais no `.env` versionado | confirmada | `.env:7-8`; as obrigatórias sem padrão são três com `AZAMON_WEBHOOK_SEGREDO` (L249) |
| 1.1#4 (L167) | Isolamento não declarado no compose | confirmada | `docker-compose.yml:12-30` |
| 1.1#5 (L168) | Mensagem do envelope é a do sentinela | confirmada | `internal/plataforma/erro/erro.go:111-115` |
| 1.1#6 (L169) | `//go:embed` em `db/embutido.go` | confirmada | `db/embutido.go`; `internal/plataforma/migracao.go:24` |
| 1.1#7 (L170) | Healthcheck `/azamon -saude` | confirmada | `docker-compose.yml:24-25`; `cmd/azamon/main.go:44-45` |
| 1.2#1 (L174) | Verificação 1 por sonda | histórica | sonda criada em `5bb40a4`, apagada em `a7b93cf` |
| 1.2#2 (L175) | `shadcn init` limpo | histórica | `5bb40a4` |
| 1.2#3 (L176) | `Toast` é `sonner` | confirmada | `web/components/ui/sonner.tsx:8,12` |
| 1.2#4 (L177) | `next.config.ts` no estágio de execução | confirmada | `web/Dockerfile:13` |
| 1.2#5 (L178) | Sem `middleware.ts` | confirmada | nenhum `web/middleware.ts` nem `web/proxy.ts`; `web/scripts/verificar-offline.mjs:18` |
| 1.2#6 (L179) | Portão do AD-12 é `prebuild` | confirmada | `web/package.json:7`; `web/scripts/verificar-offline.mjs:17,20` |
| 1.2#7 (L180) | Azamon Sans, WOFF2 de 48 KB | confirmada | `web/public/fonts/azamon-sans.woff2` (48.256 B) |
| 1.2#8 (L181) | Papéis tipográficos como `@utility` | confirmada | `web/app/globals.css:166-181` |
| 1.2#9 (L182) | `AZAMON_API_URL` com padrão | confirmada | `web/next.config.ts:10` |
| 1.3#1 (L186) | Verificação 2: sqlc aceita `uuidv7()` | histórica | `db/migracoes/20260911120000_identidade_schema_e_contas.sql:8` |
| 1.3#2 (L187) | Módulo nunca enxerga schema alheio no sqlc | substituída-agora | `sqlc.yaml:28-31`; `423fe9d` (3.5) |
| 1.3#3 (L188) | `emit_exact_table_names` | confirmada | `sqlc.yaml:16,27` |
| 1.3#4 (L189) | Marcador `ON CONFLICT` decide a semente | confirmada | `internal/plataforma/semente.go:45-52` |
| 1.3#5 (L190) | `Semear` vazio é erro | confirmada | `internal/plataforma/semente.go:26-29` |
| 1.3#6 (L191) | uuid v5 na semente | confirmada | `media/gerar.go:218-221` |
| 1.3#7 (L192) | Catálogo gerado por `media/gerar.go` | confirmada | `media/gerar.go:1,145,193,211` |
| 1.3#8 (L193) | SVG placeholder locais | confirmada | `media/gerar.go:36,57-61` |
| 1.3#9 (L194) | Rota de imagem sob `/api/v1/media` | confirmada | `api/rotas.go:33`; `api/media.go:16-25` |
| 1.3#10 (L195) | Tabelas sem coluna de data | confirmada | `db/migracoes/20260911120200_catalogo_schema.sql:7-32` |
| 1.3#11 (L196) | E-mail em minúscula por `CHECK` | confirmada | `db/migracoes/20260911120000_identidade_schema_e_contas.sql:13,21` |
| 1.3#12 (L197) | Credenciais de demonstração no README | confirmada | `README.md:60-61` |
| 1.4#1 (L201) | Verificação 4: p95 0,34 ms | histórica | `db/medicao_nfr4_test.go:17-29,104-114` — é a sonda; a consulta embarcada foi medida na retro da Épica 3 (ver ressalvas) |
| 1.4#2 (L202) | Planejador não usa o GIN | histórica | `db/medicao_nfr4_test.go:42-51` |
| 1.4#3 (L203) | Índices de FK entraram | confirmada | `db/migracoes/20260911140000_catalogo_busca_e_indices.sql:17-18` |
| 1.4#4 (L204) | Normalização na escrita pelo Go | confirmada | `internal/catalogo/normalizar.go:21-26` |
| 1.4#5 (L205) | Semente grande separada | confirmada | `db/embutido.go`; `internal/plataforma/config.go:135` |
| 1.4#6 (L206) | 50 × 10 × 10 no SQL | confirmada | `db/semente-grande/001_catalogo_grande.sql:15-34` |
| 1.4#7 (L207) | Sonda mora no teste | confirmada | `db/medicao_nfr4_test.go:33-40` (comentário corrigido) |
| 1.5#1 (L211) | Sonda removida; cookie da Sessão | confirmada | `api/sessao.go:141-148` |
| 1.5#2 (L212) | `go-redis/v9` | confirmada | `go.mod:8,11` |
| 1.5#3 (L213) | Conexão Redis preguiçosa | confirmada | `internal/plataforma/redis.go:9-18` |
| 1.5#4 (L214) | Mesma resposta e tempo no login | confirmada | `internal/identidade/identidade.go:38,72-75` |
| 1.5#5 (L215) | `ErrEntradaInvalida` | confirmada | `internal/plataforma/erro/erro.go:29,49` |
| 1.5#6 (L216) | Aresta `plataforma/erro → identidade` | confirmada | `internal/fronteira_test.go:31`; alargada em L231, L328 |
| 1.5#7 (L217) | Portas recebem `DBTX` | confirmada | `internal/identidade/identidade.go:71` |
| 1.5#8 (L218) | Sessão guarda `{id, nome}` | substituída-agora | `internal/identidade/identidade.go:58-62` (`id`, `nome`, `email`, `papel`); `3c4d625` (2.4), `a1bd448` (2.6) |
| 1.5#9 (L219) | Links temporários de conferência | histórica | removidos em `423fe9d` (3.5), como previsto |
| 1.5#10 (L220) | Página de Produto com filho cliente | confirmada | `web/app/produtos/[id]/page.tsx:16,43` |
| 1.6#1 (L224) | Status em `text` com `CHECK` | confirmada | `db/migracoes/20260912120000_pedido_esqueleto.sql:17-25`; "sem migração nova" caiu na 5.1 (L281) |
| 1.6#2 (L225) | Contador por ano | confirmada | `internal/pedido/db/consultas.sql:8-11`; `internal/pedido/pedido.go:200-207` |
| 1.6#3 (L226) | Transação no handler | substituída-registrada | `api/transacao.go:36-57`; L329 |
| 1.6#4 (L227) | `pedido.Criar` orquestra | confirmada | `internal/pedido/pedido.go:138-337` |
| 1.6#5 (L228) | AD-5 em duas consultas | confirmada | `internal/catalogo/catalogo.go:266-277` |
| 1.6#6 (L229) | `estoque_total DEFAULT 10` | confirmada | `db/migracoes/20260912120100_catalogo_reserva_estoque.sql:12` |
| 1.6#7 (L230) | Disponível em `dados` | confirmada | `internal/catalogo/catalogo.go:28-39` |
| 1.6#8 (L231) | `plataforma/erro` importa catalogo e pedido | confirmada | `internal/plataforma/erro/erro.go:16,18` |
| 1.6#9 (L232) | Não há middleware de sessão | substituída-agora | `api/admin.go:51-66` (`somenteAdministrador`); `api/rotas.go:136-137`; `3c4d625` (2.4) |
| 1.6#10 (L233) | `escreverJSON` recebe o status | confirmada | `api/rotas.go:191-200` |
| 1.6#11 (L234) | Pedido nasce na Página de Produto | substituída-registrada | `comprar.tsx` não existe; L331 |
| 1.7#1 (L238) | Faixa sobre `total % 100` | confirmada | `internal/pagamento/pagamento.go:77-89` |
| 1.7#2 (L239) | Só emite a aprovada | substituída-registrada | `internal/pagamento/pagamento.go:301-303,327-331`; L358 |
| 1.7#3 (L240) | Emissão derivada | confirmada | `internal/pagamento/db/consultas.sql:54-62` |
| 1.7#4 (L241) | `Enviar` como parâmetro | confirmada | `cmd/azamon/main.go:195-210` |
| 1.7#5 (L242) | `pagamento` não conhece `pedido` | confirmada | `internal/fronteira_test.go:36-37` |
| 1.7#6 (L243) | Nenhum sentinela novo | confirmada | `internal/fronteira_test.go:31`; "`plataforma/erro` não mudou" cai na L249 da mesma estória |
| 1.7#7 (L244) | Toda confirmação chega a terminal | confirmada | `internal/pedido/pedido.go:985-1008` |
| 1.7#8 (L245) | Uma transação por Pedido, `SKIP LOCKED` | confirmada | `internal/pedido/db/consultas.sql:243-248` |
| 1.7#9 (L246) | Único relógio | confirmada | `cmd/azamon/main.go:145-189` |
| 1.7#10 (L247) | Tela do Pedido sem busca no Next | confirmada | `web/app/pedidos/[id]/acompanhamento.tsx:275-277`; "3 s e para" trocado em L264-265 |
| 1.7#11 (L248) | `comprar.tsx` navega | substituída-registrada | arquivo ausente; L331 |
| 1.7#12 (L249) | Webhook por segredo | confirmada | `api/webhook.go:18,40-43` |
| 1.7#13 (L250) | Cliente com prazo | confirmada | `cmd/azamon/main.go:34-39,197` |
| 1.7#14 (L251) | `RFC3339Nano` | confirmada | `api/pedido.go:121-124` |
| 1.7#15 (L252) | PAGO mantém a Reserva | confirmada | `internal/pedido/maquina.go:127` |
| 1.8#1 (L256) | `expirar` fora do tique | substituída-registrada | `cmd/azamon/main.go:173-184`; L375 |
| 1.8#2 (L257) | Consolidar é a única passagem que muda o total | substituída-agora | `internal/catalogo/produto.go:166-182` (`AjustarEstoque`, 3.4, `f027448`); entre as transições do Pedido, `internal/pedido/maquina.go:132` segue a única |
| 1.8#3 (L258) | Consolidar sem Reserva ativa é no-op | confirmada | `internal/catalogo/catalogo.go:331-346` |
| 1.8#4 (L259) | Baixa ordenada no Go | confirmada | `internal/catalogo/catalogo.go:351-353` |
| 1.8#5 (L260) | `TravarPedido` devolve o instante | confirmada | `internal/pedido/db/consultas.sql:243-248` |
| 1.8#6 (L261) | Avanço deriva do histórico | confirmada | `internal/pedido/db/consultas.sql:283-288` |
| 1.8#7 (L262) | Uma lista de avanços | confirmada | `internal/pedido/pedido.go:36-40` |
| 1.8#8 (L263) | Candidatos fora, decisão dentro | confirmada | `internal/pedido/pedido.go:1104-1196` |
| 1.8#9 (L264) | `EstadoTerminal` único | confirmada | `internal/pedido/maquina.go:184-190` |
| 1.8#10 (L265) | Dois ritmos, 3 s e 10 s | confirmada | `web/lib/pedido.ts:52-53` |
| 1.8#11 (L266) | Autor `simulacao-entrega` | substituída-registrada | `internal/pedido/maquina.go:44-48`; L285 |
| 1.8#12 (L267) | Nenhuma migração nem variável | confirmada | `internal/plataforma/config.go:116-117` |
| 1.9#1 (L271) | NFR-1 em 3 min 26 s | histórica | `cc8f814`; remedido em 4 min 28 s na 7.1 (`9c63713`) |
| 1.9#2 (L272) | Cinco verificações fechadas | histórica | `cc8f814` |
| 1.9#3 (L273) | Compose offline descartado | confirmada | arquivo ausente; `README.md:347-351` |
| 1.9#4 (L274) | Ensaio offline por rede interna | histórica | `cc8f814` |
| 1.9#5 (L275) | Passeio não chega a PAGO | substituída-registrada | `README.md:133-134`; L342, L358 |
| 1.9#6 (L276) | Portão do AD-12 exercitado | histórica | `cc8f814`; `web/scripts/verificar-offline.mjs:17` |
| 1.9#7 (L277) | Nenhuma linha de código na 1.9 | histórica | `git show --stat cc8f814 16405cc` |

## §10 — Épica 5 (5.1–5.11)

| ref | decisão | veredito | evidência |
|---|---|---|---|
| 5.1#1 (L281) | `SEPARANDO`, e o código mudou | confirmada | `db/migracoes/20260919120000_pedido_maquina_de_estados.sql:23,34-35` |
| 5.1#2 (L282) | Tabela do AD-3 é dado | confirmada | `internal/pedido/maquina.go:125-135` |
| 5.1#3 (L283) | Efeito dentro de `Transicionar` | confirmada | `internal/pedido/maquina.go:244-253` |
| 5.1#4 (L284) | Cancelamento reclassifica pela releitura | confirmada | `internal/pedido/maquina.go:262-276` |
| 5.1#5 (L285) | `autor` virou `ator` | confirmada | `db/migracoes/20260919120000_pedido_maquina_de_estados.sql:45,51-58` |
| 5.1#6 (L286) | Imutável por gatilho | confirmada | `db/migracoes/20260919120000_pedido_maquina_de_estados.sql:71-104` |
| 5.1#7 (L287) | `correlacao` adiada | confirmada | `deferred-work.md:227` |
| 5.3#1 (L291) | Faixas na migração | confirmada | `db/migracoes/20260919160000_pedido_faixa_frete.sql:5-8` |
| 5.3#2 (L292) | Reais inteiros | confirmada | `db/migracoes/20260919160000_pedido_faixa_frete.sql:25` |
| 5.3#3 (L293) | Região padrão é linha | confirmada | `internal/pedido/frete.go:20,87-89` |
| 5.3#4 (L294) | `subtotal ≥ limiar` | confirmada | `internal/pedido/frete.go:94` |
| 5.3#5 (L295) | Cotação é leitura de `pedido` | confirmada | `api/rotas.go:79`; `internal/pedido/frete.go:106-128` |
| 5.4#1 (L300) | Chave nasce na entrada da Revisão | confirmada | `web/app/checkout/revisao/revisao-do-pedido.tsx:113` |
| 5.4#2 (L301) | Chave em `sessionStorage`, sobrevive ao Carrinho | substituída-registrada | `web/lib/checkout.ts:352-354`; L332 |
| 5.4#3 (L302) | Queda sem `randomUUID` | confirmada | `web/lib/checkout.ts:261-271` |
| 5.4#4 (L303) | Sem armazenamento, Revisão abre sem Confirmar | confirmada | `web/lib/checkout.ts:296-304`; bloqueio total volta ao Endereço antes (`revisao-do-pedido.tsx:115-118`) |
| 5.4#5 (L304) | Divergência não se decide na Revisão | confirmada | `revisao-do-pedido.tsx:318-345` |
| 5.4#6 (L305) | Correções da revisão (a)(b)(c) | confirmada | `web/lib/checkout.ts:298`; `revisao-do-pedido.tsx:243-246,291` |
| 5.5#1 (L309) | Entrada no checkout é rota que escreve | confirmada | `internal/pedido/checkout.go:37-54` |
| 5.5#2 (L310) | `ConfirmarPrecoVisto` porta única | confirmada | `internal/carrinho/carrinho.go:327-357`; "a borda não repete" caiu em L329 |
| 5.5#3 (L311) | Resposta é o envelope do Carrinho | confirmada | `api/carrinho.go:175,183` |
| 5.5#4 (L312) | Revisão desvia sem escrever | confirmada | `web/lib/checkout.ts:106-123` |
| 5.5#5 (L313) | Confirmar o preço não desbloqueia | confirmada | `internal/pedido/checkout.go:46` |
| 5.5#6 (L314) | `unnest` de dois argumentos | confirmada | `internal/carrinho/db/consultas.sql:89-101` |
| 5.5#7 (L315) | Escrita parcial sai em 500 | substituída-registrada | `internal/carrinho/carrinho.go:353-354` (409 `CARRINHO_MUDOU`); L328 |
| 5.5#8 (L316) | Abertura por função pura | confirmada | `web/lib/checkout.ts:172-235` |
| 5.5#9 (L317) | `aria-disabled` | confirmada | `web/app/checkout/endereco/escolha-de-endereco.tsx:161,320` |
| 5.5#10 (L318) | Nenhuma migração | confirmada | `git ls-files db/migracoes`: nenhum arquivo entre `20260919160000` e `20260921120000` |
| 5.5#11 (L319) | Recálculo do Frete provado | confirmada | `api/checkout_test.go:227-247` |
| 5.6#1 (L323) | Chave é coluna do Pedido | confirmada | `db/migracoes/20260921120000_pedido_criacao.sql:37-38,62` |
| 5.6#2 (L324) | `INSERT` reivindica a chave antes da trava | confirmada | `internal/pedido/pedido.go:197-232,251` |
| 5.6#3 (L325) | Total conferido duas vezes | confirmada | `internal/pedido/pedido.go:281-294`; por Item desde L422 |
| 5.6#4 (L326) | Congelamento em colunas | confirmada | `db/migracoes/20260921120000_pedido_criacao.sql:42-58` |
| 5.6#5 (L327) | `item_pedido` recusa INSERT tardio | confirmada | `db/migracoes/20260921120000_pedido_criacao.sql:65-78` |
| 5.6#6 (L328) | `Esvaziar` só os lidos | confirmada | `internal/carrinho/carrinho.go:372-394` |
| 5.6#7 (L329) | `emTransacao` repete uma vez | confirmada | `api/transacao.go:36-67` |
| 5.6#8 (L330) | Carrinho só de invisíveis | confirmada | `internal/pedido/pedido.go:181-182` |
| 5.6#9 (L331) | Botão do esqueleto saiu | confirmada | `revisao-do-pedido.tsx:384` é o único `bg-primary-strong` |
| 5.6#10 (L332) | Chave vale uma tentativa | confirmada | `web/lib/checkout.ts:352-354,478-490` |
| 5.6#11 (L333) | Desfecho puro | confirmada | `web/lib/checkout.ts:359,471-497` |
| 5.8#1 (L337) | Resposta com a tripla | confirmada | `api/pedido.go:72-84`; `internal/pedido/pedido.go:542,583-591` |
| 5.8#2 (L338) | `expira_em` do histórico | confirmada | `internal/pedido/pedido.go:522-532` |
| 5.8#3 (L339) | Teto de Tentativas é de `pagamento` | confirmada | `internal/pagamento/pagamento.go:191-201` |
| 5.8#4 (L340) | `disponivel` de agora | confirmada | `internal/pedido/pedido.go:601-617` |
| 5.8#5 (L341) | Leitura em REPEATABLE READ | confirmada | `api/pedido.go:363-371` |
| 5.8#6 (L342) | Centavos só na 1ª Tentativa | confirmada | `internal/pagamento/pagamento.go:77-90`; "só o aprovado" caiu em L358 |
| 5.8#7 (L343) | Três superfícies, nenhuma promessa | substituída-registrada | `acompanhamento.tsx:468-480,495-506`; L365, L385 |
| 5.8#8 (L344) | Desvio de relógio fica de fora | confirmada | `web/lib/pedido.ts:83-90` |
| 5.9#1 (L348) | Estória chegou quase pronta | histórica | `577a6aa` |
| 5.9#2 (L349) | Sinal é a linha da inbox | confirmada | `internal/pagamento/db/consultas.sql:88-94` |
| 5.9#3 (L350) | "Registrada na Tentativa" | confirmada | `db/migracoes/20260912130000_pagamento_tentativa_e_confirmacao.sql:35-37` |
| 5.9#4 (L351) | Cancelado depois de PAGO casa | confirmada | `internal/pedido/maquina.go:114-116` |
| 5.9#5 (L352) | `WarnContext` depois do commit | confirmada | `internal/pedido/pedido.go:1011-1020` |
| 5.9#6 (L353) | CAS protege o Status | confirmada | `internal/pedido/maquina.go:224-229` |
| 5.9#7 (L354) | Emissão cega ao Pedido | confirmada | `internal/pagamento/pagamento.go:309-337` |
| 5.10#1 (L358) | Recusa pelo mesmo caminho | confirmada | `internal/pedido/pedido.go:985-1007` |
| 5.10#2 (L359) | `RECUSADO_PELO_PROVEDOR` | confirmada | `internal/pedido/maquina.go:58` |
| 5.10#3 (L360) | Número derivado, teto por valor | confirmada | `internal/pagamento/pagamento.go:55-60,135-158` |
| 5.10#4 (L361) | `ErrTentativasEsgotadas` | confirmada | `internal/pedido/pedido.go:92,385-386` |
| 5.10#5 (L362) | CAS antes de `IniciarTentativa` | confirmada | `internal/pedido/pedido.go:360-390` |
| 5.10#6 (L363) | POST sem corpo; razão da tripla relida | confirmada | `api/pedido.go:264-297`; as recusas também avisam (ver ressalvas) |
| 5.10#7 (L364) | Desativado sai como esgotado | confirmada | `internal/catalogo/catalogo.go:217-219` |
| 5.10#8 (L365) | "Tentar pagar de novo" | confirmada | `acompanhamento.tsx:468-480`; "Cancelar fora até a 6.3" caiu em L385 |
| 5.10#9 (L366) | Subtestes da 1.7 | confirmada | `api/webhook_test.go:63-80` |
| 5.11#1 (L370) | Casca comum `avancarPeloTempo` | confirmada | `internal/pedido/pedido.go:1106-1196` |
| 5.11#2 (L371) | Instante por `ExpiraEm` | confirmada | `internal/pedido/pedido.go:1160` |
| 5.11#3 (L372) | Expirar confere a inbox | confirmada | `internal/pedido/pedido.go:1170-1180` |
| 5.11#4 (L373) | Aprovação depois de expiração avisa | confirmada | `internal/pedido/pedido.go:972-978,1021-1024` |
| 5.11#5 (L374) | Atraso ≥ prazo recusado | confirmada | `internal/plataforma/config.go:153-155` |
| 5.11#6 (L375) | `expirar` sem interruptor | confirmada | `cmd/azamon/main.go:173-183` |
| 5.11#7 (L376) | Piso `criada_em > @desde` | confirmada | `internal/pagamento/db/consultas.sql:54-62` |
| 5.11#8 (L377) | Expiração não emite | confirmada | `internal/pedido/pedido.go:1078-1086` |

## §10 — Épica 6, correções e 7.2

| ref | decisão | veredito | evidência |
|---|---|---|---|
| 6.3#1 (L381) | Cancelar trava com espera | confirmada | `internal/pedido/pedido.go:427-443` |
| 6.3#2 (L382) | CANCELADO lido é sucesso | confirmada | `internal/pedido/pedido.go:440-442` |
| 6.3#3 (L383) | Cancelar não chama Liberar | confirmada | `internal/pedido/maquina.go:245-248` |
| 6.3#4 (L384) | POST sem corpo; 409 com Status | confirmada | `api/pedido.go:312-343` |
| 6.3#5 (L385) | Botão no Card do topo | confirmada | `acompanhamento.tsx:495-506`; frase da corrida só em ENVIADO (ver ressalvas) |
| 6.3#6 (L386) | `pode_cancelar` do Go | confirmada | `api/pedido.go:139` |
| 6.4#1 (L390) | `de` da tela é a chave do CAS | confirmada | `api/pedido_admin.go:243-256` |
| 6.4#2 (L391) | Trava sem dono | confirmada | `internal/pedido/pedido.go:886-905` |
| 6.4#3 (L392) | Administrador não cancela | confirmada | `internal/pedido/maquina.go:125-135` |
| 6.4#4 (L393) | `permitidas` e `terminal` | confirmada | `api/pedido_admin.go:86-101` |
| 6.4#5 (L394) | Detalhe por consulta própria | confirmada | `internal/pedido/pedido.go:847` |
| 6.4#6 (L395) | `vendedor_nome` na consulta que existia | confirmada | `internal/pedido/db/consultas.sql:118-122`; a coluna é da 1.6, não da 5.6 |
| 6.4#7 (L396) | Ordena por `min(ocorrido_em)` | confirmada | `internal/pedido/db/consultas.sql:165-183` |
| 6.4#8 (L397) | Ação e Alerts num componente, sem Dialog | substituída-agora | `web/app/admin/pedidos/acoes-do-pedido.tsx:38-44`; `7014591` |
| 6.4#9 (L398) | Estado da Tabela na URL | confirmada | `web/lib/pedido.ts:547-567` |
| 6.5#1 (L402) | Sinal derivado na leitura | confirmada | `internal/pedido/pedido.go:703-715` |
| 6.5#2 (L403) | Só CANCELADO | confirmada | `internal/pedido/pedido.go:704-708` |
| 6.5#3 (L404) | Em lote; REPEATABLE READ | confirmada | `internal/pedido/pedido.go:727,766` |
| 6.5#4 (L405) | Campo só no administrativo | confirmada | `api/pedido_admin.go:40-45` |
| 6.5#5 (L406) | Alert fixo e Badge | confirmada | `web/app/admin/pedidos/[id]/detalhe-admin.tsx:131-140` |
| 6.6#1 (L410) | 6.6 só prende a FR-33 | confirmada | `cd27088`; `api/simulacao_test.go:44` |
| 6.6#2 (L415) | Subteste roda por último | substituída-agora | `api/simulacao_test.go:39-41`; `a610f86` |
| 6.6#3 (L416) | Terceiro arranque retoma | confirmada | `cmd/azamon/main_test.go:157-196` |
| 6.6#4 (L417) | `ErrEstadoJaAvancado` defensivo | confirmada | `internal/pedido/pedido.go:1141-1147` |
| 6.6#5 (L418) | `noPrazo` lê o histórico | confirmada | `cmd/azamon/main_test.go:684-704` |
| D3D5#1 (L422) | Preço por Item sob a trava | confirmada | `internal/pedido/pedido.go:272-276` |
| D3D5#2 (L423) | Só no passo 5 | confirmada | `api/pedido_test.go:1164-1174` |
| 7.2#1 (L427) | Desenho segue a tabela | confirmada | `internal/fronteira_test.go:24-40` |
| 7.2#2 (L428) | `TestDiagramaEhATabela` | confirmada | `internal/fronteira_test.go:148-156` |
| 7.2#3 (L429) | Rótulos fora da guarda | confirmada | `internal/fronteira_test.go:136,146-147` |
| 7.2#4 (L430) | Grafo nos dois arquivos | confirmada | `ARCHITECTURE-SPINE.md:51`; `DIAGRAMA-MODULOS.md:15` |
| 7.2#5 (L431) | Quatro achados ao `deferred-work` | histórica | dois deles (AD-8, AD-6) fechados por esta estória |

## Ressalvas e precisões

A decisão principal vale (ou é histórica); uma parte secundária não, e a nota no addendum diz qual.

- **1.4#1** — o p95 de 0,34 ms é da sonda de `db/medicao_nfr4_test.go`, que lê `catalogo.produto` direto. A consulta que a busca roda (`internal/busca/db/consultas.sql:8-20`: VIEW `produto_visivel` com o Estoque disponível, filtros `IS NULL OR`, `ORDER BY CASE`, mais `ContarVisiveis`) foi medida à mão na retrospectiva da Épica 3 (`epic-3-retro-2026-09-19.md:50-72`): pior p95 de 8,47 ms por requisição. O teste ainda mede a sonda; trazê-lo para a consulta real é o `epic-3-retro-item-13`, aberto.
- **5.4#4** — com o `sessionStorage` bloqueado de todo, a Revisão nem abre: a escolha do Endereço já falha antes e volta ao passo Endereço. O aviso "sem chave" é do caso em que só a escrita da chave falha (cota cheia).
- **5.10#6** — as recusas `ESTOQUE_INSUFICIENTE` e `TETO_DE_TENTATIVAS` também mostram a mensagem depois da releitura (`web/lib/pedido.ts:264-268,285-286`), decidido na revisão da 5.10 (`spec-5-10-…md:88`).
- **6.3#5** — a frase da corrida perdida só aparece enquanto o Status for ENVIADO (`web/lib/pedido.ts:220`, `33ca441`).
- **6.4#6** — `vendedor_nome` está em `pedido.item_pedido` desde a 1.6 (`db/migracoes/20260912120000_pedido_esqueleto.sql:40`), não desde a 5.6.
- Registradas no próprio §10, sem nota nova: 1.1#2 e 1.1#3 (L205, L249), 1.6#1 (L281), 1.7#6 e 1.7#10 (L249, L264), 5.5#2 (L329), 5.6#3 (L422), 5.8#6 (L358), 5.10#8 (L385).

## `corrigida-no-código`

Só comentário; nenhum comportamento mudou.

| Onde | Mentia | Agora |
|---|---|---|
| `db/migracoes/20260921120000_pedido_criacao.sql:13-18` | o gêmeo espera no índice da chave | espera na linha do ano de `pedido.contador_numero` (AD-4) |
| `web/app/checkout/revisao/revisao-do-pedido.tsx:108-111` | a chave sobrevive à ida e volta ao Carrinho | o "Fechar o Pedido" a descarta (L332) |
| `api/rotas.go:56-58,69-70` | "da Página de Produto direto ao Pedido"; "o esboço que a 6.1 substitui" | Confirmar Pedido da Revisão; Meus pedidos da 6.1 |
| `internal/catalogo/catalogo.go:326-329`, `internal/catalogo/db/consultas.sql:78-80` e o `gerado/` | "a única passagem" em que o Estoque total muda | a única do Pedido; a outra é o ajuste do Administrador |
| `db/medicao_nfr4_test.go:30-34` | a sonda "é a forma que a Épica 3 vai implementar" | é a que a 1.4 mediu, e não é a da busca |
| cabeçalho de `identidade.go`, `catalogo.go`, `busca.go`, `carrinho.go`, `pedido.go`, `pagamento.go` | "o ÚNICO [arquivo] que outro módulo importa" | abre a interface; o pacote é a porta (§9#3) |
| `internal/identidade/endereco.go:228-229` | a coluna tem `CHECK (uf = upper(uf))` | `CHECK (uf ~ '^[A-Z]{2}$')`, como a migração |
| `db/embutido.go:27-31` | mudou `db/semente/`, muda `VersaoSemente` | trocá-la sobre banco semeado derruba o arranque; a D1 não a trocou |
| `web/components/preco.tsx:3-6` | o `R$` é escrito aqui "e em nenhum outro lugar" | aqui e em `formatarPreco` |
| `web/lib/quantidade.ts:1-4` | o teto "passa a vir do Go" | continua literal, e o Go decide; adiado |
| `api/produto_admin.go:21`, `api/produto.go:13`, `web/lib/preco.ts:1`, `web/components/preco.tsx:3` | dinheiro em centavos citado como AD-3 | AD-9 |

## Segundo sentido: da construção para o addendum

A CA da 7.3 pede também que toda decisão de arquitetura tomada na construção esteja no addendum. Vinte e sete estórias e correções não tinham bloco no §10. Quatro leitores novos, só leitura, varreram a spec e o commit de cada uma, descartaram o que o addendum ou a espinha já diziam, conferiram o resto no código com `arquivo:linha`, e as decisões entraram no §10 em blocos marcados "registrado na 7.3", no lugar cronológico. Amostras de âncora foram reconferidas à mão (`api/sessao.go:152-161`, `internal/identidade/bloqueio.go:46-48`, `api/concorrencia_test.go:196`, `web/lib/carrinho.ts:41`, `api/carrinho_test.go:20`, `web/components/ui/dialog.tsx:50-55`).

| Faixa | Blocos | Itens |
|---|---|---|
| Épica 2 (2.1–2.6) | 6 | 31 |
| Épica 3 (3.1, 3.2–3.3, 3.4, 3.5, 3.6, 3.7–3.9, 3.10) | 7 | 38 |
| Épica 4 (4.1, 4.2–4.3, 4.4, 4.5), 5.2 e 5.7 | 6 | 21 |
| 6.1, 6.2, 6.7, `7014591`, D1–D4/E1/E4/E5, D2/D6/D7/E2/E3, testes das retros, 7.1 | 8 | 19 |
| **Total** | **27** | **109** |

O que a varredura achou além das decisões:

- **A espinha descreve o código de outro jeito em mais pontos** — AD-10 (`next/image` e o `fetch` em RSC), AD-19 (a VIEW é a fonte, e não `Visiveis`), a assinatura de `Disponivel`, a tabela de rotas do AD-16 sem `GET /api/v1/categorias`, e o desempate `id DESC` de Meus pedidos contra o AD-18. Ficaram em `deferred-work.md`, com o AD-18 e a Semente Estrutural: a spec da 7.3 só emendava o AD-4, o AD-6 e o AD-8.
- **O teto por Item de Carrinho está em dois lugares**, `AZAMON_CARRINHO_UNIDADES_MAX` e `TETO_POR_ITEM` no `web/`. Adiado.
- **O NFR-4 da consulta embarcada** foi medido na retro da Épica 3 (pior p95 de 8,47 ms); o teste ainda mede a sonda (`epic-3-retro-item-13`).
- **Comentários velhos**, corrigidos: estão na tabela `corrigida-no-código` acima.
