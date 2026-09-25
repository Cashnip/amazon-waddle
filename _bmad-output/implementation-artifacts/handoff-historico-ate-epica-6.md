# HANDOFF.md — histórico até a Épica 6

Esta é a seção "Próximo passo" do `HANDOFF.md` no commit `5f54a54`, copiada sem edição (linhas 60–683 de lá),
mais, no fim, o parágrafo de estado da linha 3 do mesmo commit. Saiu do `HANDOFF.md` na compactação antes da
Épica 7 (`epic-6-retro-item-34`); o que era estado, pendência ou armadilha ficou lá.

Citações antigas no formato `HANDOFF.md:NNN` (nas retros, no `sprint-status.yaml`, nas specs) apontam para esse
commit, e não para o arquivo atual: resolva com `git show 5f54a54:HANDOFF.md`.

---

## Próximo passo

**Antes da Épica 7, nesta ordem, com o contexto limpo entre cada sessão** (decisão humana de 2026-09-24):

1. ~~**`bmad-checkpoint-preview` da Épica 6**~~ — **feito em 2026-09-24**: defeitos E1–E5 e a decisão O3 no bloco "O que o passeio da Épica 6 achou" abaixo. A leitura humana aprovou, e as sete estórias e a Épica 6 estão `done`.
2. ~~**`bmad-build` do D3 com D5**~~ — **feito em 2026-09-24** (spec `spec-d3-d5-preco-por-item-sob-a-trava.md`, `done`): sob a trava, cada Item confere o preço contra o preço visto (`TOTAL_DIVERGENTE`), e reenviar sem passar pela entrada no checkout é recusado (AD-17). `Compensados`, `Limiar` e `Janela` em `api/pedido_test.go` a prendem, e `Janela` prende o lugar (passo 5, não passo 2). Revisão com uma lente, por decisão humana.
3. ~~**`bmad-build` do padrão de `Dialog` e foco**~~ — **feito em 2026-09-24** (spec `spec-dialog-e-foco-d2-d6-d7-e2-e3.md`, `done`): o `DialogContent` compartilhado guarda quem tinha o foco ao abrir e o devolve ao fechar, salvo destino escolhido por quem usa — corrige os oito `Dialog`s de uma vez, e não só os três nomeados. **Não ponha `autoFocus` dentro de `Dialog`:** o React foca antes do `FocusScope` e nada é guardado (a revisão achou isso no Ajustar Estoque, corrigido). "Confirmar o novo preço" leva o foco ao título, no passo Endereço e no Carrinho; `Close` virou `Fechar`; "Cancelar Pedido" é vermelho cheio com branco. **O3 aceito sem mudança** (decisão humana): o `Dialog` do Comprador fica no nível da página e não desmonta, e a corrida já termina com a frase certa. Visto no navegador: D6, D7 e D2 no passo Endereço e o `Esc` do Carrinho; não abertos na tela o cancelamento e o avanço de Status. Fica no `deferred-work.md`: o botão que some ao confirmar (remover Endereço, Categoria, Vendedor) ainda deixa o foco no `body`. Uma lente.
4. ~~**`bmad-build` do D1 e do D4**, mais E1, E4 e E5~~ — **feito em 2026-09-24** (spec `spec-d1-d4-e1-e4-e5.md`, `done`): "Anotações de um Andarilho" passa a R$ 39,95, pelo `media/gerar.go`, e o passo 5 do Roteiro B anda sem reconfigurar nada. **A `VersaoSemente` não mudou:** a semente é `INSERT` sem `ON CONFLICT`, e versão nova sobre banco já semeado derruba o arranque — banco existente só vê o preço com `docker compose down -v`. Meus pedidos virou grade de colunas iguais com o sublinhado só no número; a frase da corrida perdida só vale em `ENVIADO`; o `Alert` da 6.5 saiu sem `role="alert"`; o comentário do D4 nomeia a checagem por Item. Não aberto no navegador. Uma lente.

Não um `bmad-build` único para tudo: a mistura força a revisão cara sobre o trivial ou a barata sobre o D3 (política de revisão do `bmad-build`).

**Depois dos quatro, e ainda antes da 7.1, nesta ordem** (decisão humana de 2026-09-24, depois das retrospectivas headless da Épica 5 e da Épica 6 — `epic-5-retro-2026-09-24.md` e `epic-6-retro-2026-09-24.md`, as duas `accepted-with-open-items`):

5. ~~**`bmad-retrospective -H` da Épica 5 e da Épica 6**~~ — **feito em 2026-09-24**: itens `epic-5-retro-item-25`..`29` e `epic-6-retro-item-30`..`34` no `sprint-status.yaml`. Perguntas abertas para decisão humana nas duas retros (O1, O2, dividir `pedido.go`, `epics.md` vivo ou não, o E5 sem registro de decisão). As transições `epic-3-retro-item-19` e `item-23` → `done` foram propostas e **não** gravadas.
6. ~~**`bmad-checkpoint-preview` só pelo teclado (e leitor de tela) das Épicas 5 e 6**~~ — **encerrado em 2026-09-25 por decisão humana, no meio da UJ-1**: acessibilidade com leitor de tela não é o foco do projeto. Percorrido com teclado de verdade e VoiceOver até o desfecho `,95` e a nova Tentativa chegando a `PAGO`; nenhum bloqueio, então os dois vereditos ficam `accepted-with-open-items`. O2 decidido (não é defeito), D6 e D2 confirmados, K1 e K2 no bloco "O que o passeio só pelo teclado achou" abaixo. **Não vistos, e continuam sem ver na tela depois das correções:** os desfechos `,00` e `,90` pelo teclado, o `Alert` de preço (D7), a UJ-3 (E2, E3, cancelar em `AGUARDANDO_PAGAMENTO` com o Produto de R$ 39,95), E1, E4, E5, Meus pedidos vazio, o painel pelo teclado e `AZAMON_ENTREGA_SIMULACAO_ATIVA=false`.
7. **`bmad-build` dos testes que faltam** — `epic-5-retro-item-26` (teste que falha sem o `ORDER BY id` na trava), `epic-6-retro-item-32` (tirar a dependência de ordem de `TestSessaoEProduto`, `api/sessao_test.go:566-588`) e `epic-6-retro-item-33` (teste que falha quando o `de` da transição administrativa muda).

Depois dos sete, `bmad-build` na **Estória 7.1**.

Depois, `bmad-build` na **Estória 7.1** (O README leva do clone ao sistema rodando), a primeira da Épica 7 — a Épica 5 e a Épica 6 estão inteiras em `main`: a 6.1 (`48f6723`), a 6.2 (`5f618db`), a 6.3 (`b6093e9`), a 6.4 (`081a60e`), a 6.5 (`5b01aec`), a 6.6 (`cd27088`) e a 6.7 (`aadb350`). A Épica 7 **verifica** o que as seis anteriores produziram, e não escreve documentação do zero: README do clone ao sistema rodando em ≤ 15 min (NFR-1), o diagrama dos seis módulos contra o código, o addendum percorrido contra o código, e os três roteiros ensaiados a partir de clone limpo. **Antes dela, ou junto da 7.4, vem a dívida maior do projeto:** o passeio da Épica 5 foi feito em 2026-09-24 e deixou sete defeitos para o `bmad-build` (D1–D7, no bloco "O que o passeio da Épica 5 achou" abaixo), e o da Épica 6 foi feito no mesmo dia — os dois defeitos do painel da 6.4, corrigidos em `7014591`, e E1–E5 abertos (bloco "O que o passeio da Épica 6 achou" abaixo). Os passeios abaixo continuam como roteiro do que não foi visto. `epic-6-retrospective` continua `optional`, sem dono. A 5.11 está em `main` (`8a3dfd0`), a 5.10 em `main` (`22500bf`), a 5.9 em `main` (`577a6aa`), a 5.8 em `main` (`e421de8`), a 5.7
em `main` (`7cd26ac`), a 5.6 em `main` (`9174f88`), a 5.5 em `main` (`f3cc0ff`), a 5.4 em `main` (`6e5984f`), a 5.3 em `main` (`dbcb2b8`) e a 5.2 em
`main` (`400861b`), as dez `done` no `sprint-status.yaml` desde a leitura humana de 2026-09-24, com a Épica 5
fechada e D1–D7 abertos para o `bmad-build`. As sete da Épica 6 estão `done` desde a leitura humana de 2026-09-24, com a Épica 6
fechada e E1–E5 abertos para o `bmad-build`; o passeio só pelo teclado, que ninguém fez, segue pendente nas duas épicas. **A Épica 5 foi aberta no navegador em 2026-09-24 (defeitos D1–D7 abaixo); da Épica 6, o passeio começou e o painel da 6.4 foi aberto na correção `7014591`.** Da 6.7, o passeio é: "Meus pedidos" com um Pedido de cada tom (cinza em progresso, vermelho em recusado e cancelado, verde em entregue), a pílula de 14px sem quebrar a linha a 360 px; o Detalhe de um total de `,00` passando de "Enviado" em cinza a "Entregue" em verde sem recarregar, com a região viva anunciando "Status: Entregue"; e `/admin/pedidos?status=CANCELADO` com o selo vermelho e, depois dele, o marcador "Pagamento aprovado" — que continua em 12px ao lado do selo de 14px, e isso está no `deferred-work.md`. Da 6.6, o passeio é o fim da UJ-2 em demonstração: `docker compose up`, um Produto de `,00`, o Pedido chegando a `PAGO` e depois a "Entregue" sozinho, uma etapa a cada ~30 s, sem clicar em nada e sem recarregar; e, com `AZAMON_ENTREGA_SIMULACAO_ATIVA=false`, o mesmo Pedido parado em `PAGO` até o Administrador avançá-lo pelo painel. Da 6.5, o passeio é: um Produto de `,00`, cancelar o Pedido em `AGUARDANDO_PAGAMENTO` pela tela do Comprador, esperar ~5 s a aprovação chegar, e abrir `/admin/pedidos?status=CANCELADO` — o marcador "Pagamento aprovado" na linha dele e em nenhuma outra — e o Detalhe, com o `Alert` abaixo do Status e sem botão de fechar; a 1440 px e só pelo teclado. A tela aberta **antes** da aprovação chegar não a mostra sozinha: `CANCELADO` é terminal e a consulta em intervalo para, então é preciso recarregar. Da 6.4, o passeio é: entrar em `/admin/pedidos`, filtrar por `PAGO`, inverter a ordenação, recarregar conferindo que o estado se mantém na URL, e avançar um Pedido pelos três botões até `ENTREGUE` — com outra aba na Página de Produto para ver o Estoque total baixar no `ENVIADO`; depois, para a corrida, abrir o `Dialog` de um Pedido com a Tabela **filtrada**, esperar a simulação movê-lo (~30 s; o `Dialog` aberto pausa a consulta de 10 s, então a linha não some) e só então confirmar, para ver o `Alert` informativo "O Pedido AZ-… já está em …" — clicar no botão velho sem o `Dialog` não reproduz mais nada, porque a consulta atualiza a linha antes; e abrir o Detalhe de um Pedido para conferir o Comprador e o Vendedor de cada Item. Da 6.3, o passeio é a UJ-3: um Pedido em `SEPARANDO` → "Cancelar Pedido" → o `Dialog` nomeando o número → confirmar, e ver "Cancelado" na região viva e na linha do tempo, o foco no título, e outra aba com o Estoque de volta na Página de Produto; depois, o mesmo com o relógio correndo (`AGUARDANDO_PAGAMENTO`), e, para a corrida, cancelar numa tela `SEPARANDO` que a simulação já levou a `ENVIADO` e ver a frase "Este Pedido foi enviado enquanto você estava nesta tela…" em vez de erro. Da 6.2, o passeio é levar um total de `,00` de `PAGO` até "Entregue" pela simulação vendo a linha do tempo ganhar cada transição sem recarregar, a frase de entregue aparecer (desde a 6.3, junto do Status no topo, e não mais nos valores), o filete entre os marcadores alinhado, e nada rolando na horizontal a 360 px. Da 6.1, o passeio
é: abrir "Meus pedidos" com o histórico vazio e ver a saída única para a Vitrine, criar mais de 20 Pedidos
para a paginação aparecer, recarregar numa página que não a primeira conferindo que ela se mantém, e digitar
`?pagina=9` à mão para ver o cartão de página vazia **sem** rodapé nem "Anterior" ao lado. Da 5.3 e da 5.4, o passeio é
o mesmo: trocar um Endereço de SP por um da BA na Revisão e ver o Frete e o total mudarem, encher o Carrinho
acima de R$ 299,00 para ver "Grátis", e recarregar a Revisão conferindo que `azamon:checkout:idempotency_key`
não muda. Da 5.6: Carrinho → Endereço → Revisão → Confirmar Pedido até `/pedidos/{id}`, com o contador da
barra zerando; e mudar um preço no admin com a Revisão aberta, para ver o aviso de total e a Revisão relida.
Da 5.8: um total de `,00` vai do relógio de 60 s a `PAGO` em ~5 s, virando Detalhe na mesma página; recarregar no
meio mantém o tempo; e um total de `,95` deixa o relógio chegar a zero com "O prazo para o pagamento terminou sem
a confirmação chegar.".
Da 5.9 não há passeio próprio: o que ela acrescenta é do lado do Go e não tem tela. O que falta ver é a
consequência mais visível da FR-26, que é da 5.8 — o total de `,00` virando `PAGO` sozinho, sem recarregar.
Da 5.10: o Roteiro B passos 1–4 — um Produto que feche em `,90` vai sozinho a `PAGAMENTO_RECUSADO` com "O Provedor
de Pagamento recusou a Tentativa de Pagamento.", outra aba mostra o Estoque de volta, "Tentar pagar de novo" traz o
relógio de volta com prazo novo, e em ~5 s o Pedido vira `PAGO`. E o caso triste: outro Comprador leva a última
unidade antes do clique, e o botão sai da tela com "Não há Estoque disponível de {nome} …".
Da 5.11: o Roteiro B passo 5 — um Produto que feche em `,95` deixa o relógio de 60 s chegar a zero, a tela troca o relógio por "Tempo de pagamento expirado." **sem recarregar**, outra aba mostra o Estoque de volta, e "Tentar pagar de novo" traz relógio novo e, em ~5 s, `PAGO`.
A 5.1 está `done` desde 2026-09-19.
A retrospectiva da Épica 3 está `done` (`epic-3-retro-2026-09-19.md`); `epic-4-retrospective` e `epic-2-retrospective` continuam `optional`, sem dono.

**O que o passeio da Épica 5 achou (2026-09-24, `bmad-checkpoint-preview` sobre `a454444^..4cbea8b`), e o `bmad-build` corrige em sessão própria:**

Aberto no navegador, com a conta semeada: Carrinho → Endereço (com cadastro em `Dialog`) → Revisão → Pedido, nos três
desfechos — `,00` aprovado em ~6 s virando Detalhe sem recarregar; `,90` recusado com motivo, Estoque de volta e
"Tentar pagar de novo" chegando a `PAGO`; `,95` expirando em 60 s exatos, Estoque de volta e a nova Tentativa com
relógio novo que sobrevive ao recarregamento. Também vistos: Frete por região (SP R$ 15, BA R$ 30, "Grátis" acima de
R$ 299), a `Idempotency-Key` estável no recarregamento, o aviso "de X para Y" na entrada, o `TOTAL_DIVERGENTE` com a
Revisão aberta (volta ao passo Endereço com o aviso, nada gravado), a última unidade levada por outro (o botão sai com
"Não há Estoque disponível de …" sem recarregar), e nada rolando na horizontal a 360 px. `go test ./...` verde: a 5.1 e
a 5.7 passam. Quatro subagentes aprofundaram a expiração adiada pela inbox, o contador por ano, a `NovaTentativa` e a
reconferência do total — só a última tem defeito.

| ID | Sev. | Onde | O quê |
|---|---|---|---|
| D1 | média | `db/semente/001_catalogo_semeado.sql` | Todo preço semeado termina em `,00` ou `,90` e o Frete é real inteiro: nenhum total fecha em `,95`–`,99`. O passo 5 do Roteiro B (FR-34) só se demonstra mudando preço, contra "nada é reconfigurado entre os roteiros" (§7.1). Semear um Produto de `,95` |
| D3 | média | `internal/pedido/pedido.go:282` | Sob a trava confere só subtotal e Frete, nunca preço por Item: dois preços que se compensam passam. **Reproduzido na tela:** Revisão com R$ 81 + R$ 79, Pedido AZ-2026-000035 gravado com R$ 86 + R$ 74, mesmo total. Recusar quando `produto.PrecoCentavos != item.PrecoVistoCentavos` (`PrecoMudou` já existe em `internal/carrinho/carrinho.go:248`) |
| D5 | média | `api/pedido_test.go:1091` | O único teste de `TOTAL_DIVERGENTE` muda o preço antes do POST, então quem recusa é o passo 2 (`pedido.go:188`); a checagem sob a trava (`:282`) pode ser apagada com a suíte verde. Falta também cruzar o limiar de isenção com o mesmo total. Vai junto do D3 — é a prova dele |
| D6 | média | `web/app/checkout/endereco/escolha-de-endereco.tsx:238` | O `Dialog` de novo Endereço abre por estado a partir de um `Button` comum, sem `DialogTrigger`: no `Esc` o foco cai no `body` em vez de voltar a "Cadastrar Endereço" (WCAG 2.4.3, piso AA) |
| D7 | baixa | passo Endereço, `Alert` de preço | "Confirmar o novo preço" desmonta o próprio botão e o foco cai no `body`: quem está no teclado recomeça do topo |
| D2 | baixa | `web/components/ui/dialog.tsx:79` | O fechar de todo `Dialog` se anuncia `Close`, em inglês |
| D4 | baixa | `web/app/checkout/revisao/revisao-do-pedido.tsx:317` | O comentário promete que subtotal da cotação diferente do Carrinho é recusado na criação; não é (a tela manda o total da cotação, que bate). A correção do D3 fecha a janela |

Ordem decidida em "Próximo passo": passeio da Épica 6 primeiro, depois D3 com D5, o padrão de `Dialog` e foco (D2, D6, D7), e D1 com D4. Sem defeito, mas para decidir ou emendar:
- **O1:** "Confirmar o novo preço" é trava só de tela — recarregar ou digitar `/checkout/revisao` a dispensa, porque a
  entrada já gravou a ciência ao reportar. A spec 5.5 prevê isso, e a FR-19 pede "exibido e sinalizado"; com o D3, a
  garantia hoje é só o total. Decisão humana;
- **O2:** no grupo de Endereços, a seta move o foco sem marcar (o Espaço marca). Pode ser da injeção de tecla da
  ferramenta; conferir com teclado de verdade antes de tratar como defeito;
- **AD-4:** a ordem global "pedido → produtos" (`ARCHITECTURE-SPINE.md:117`) não cita o contador do ano, que o `Criar`
  pega antes das duas — emendar, com memlog. E o comentário de `db/migracoes/20260921120000_pedido_criacao.sql:14` diz
  que o gêmeo espera no índice, quando espera no contador;
- **confirmado sem ação:** confirmação `PENDENTE` não prende Reserva para sempre (o adiamento da expiração é de tiques);
  não há ciclo de travas; `travaDeReservaSerializa` mata "sem `FOR UPDATE`" e "somar antes da trava" (tirar o
  `ORDER BY id` não é pego — Produto único); `NovaTentativa` desfaz tudo no teto e o duplo clique sai 409 calado.

Não feito: o passeio só pelo teclado a 1440 px e com leitor de tela. A 360 px foi por `iframe` (o Chrome não estreita
a janela abaixo de ~500 px). Estado deixado no banco: Pedidos AZ-2026-000032 a 000036 da conta semeada (o 000036 em
`PAGAMENTO_RECUSADO`), um Endereço fictício em Salvador/BA nela, e os preços e Estoques tocados já restaurados.

**O que a correção da 6.4 (`7014591`) deixou, e quem mexer no painel precisa saber:**
- **o desfecho de toda transição, sucesso inclusive, é um relato renderizado pela superfície**, fora da linha
  (`DesfechoNaTela`); só quem clicou sabe o destino, e a linha que sai do filtro não diz para onde foi. Na Tabela, a linha
  apaga o próprio Status de `anteriores` antes de reler, para `anuncioDaTabela` não repetir o anúncio;
- **o `de` enviado é capturado no clique**, não no confirmar: com o `Dialog` aberto a consulta pode mudar o Status, e o
  compare-and-swap do Go só nomeia a corrida se receber o Status que o Administrador viu;
- **o foco pós-envio espera o commit da releitura** (um efeito sobre um contador), e vai ao primeiro botão que restou no
  bloco. Logo depois do `await aoMudar()` o DOM ainda tem o botão velho — focar ali perde o foco para o `body`;
- **aceito, e na spec:** no Detalhe o leitor de tela ouve o avanço duas vezes (o `Alert` e a região "Status:"); sob filtro,
  ou em Status terminal, não sobra botão no bloco e o foco fica no documento;
- **aberto no navegador, a 1440 px e com mouse**: a Tabela com e sem filtro e o Detalhe administrativo. A corrida não
  foi reproduzida pela tela (só pelo teste de `desfechoDaTransicao`), e o passeio só pelo teclado não foi feito;
- **armadilha de bancada:** `docker compose up -d --build web` recria também o `azamon` pelo `depends_on`, com o `.env`
  — um override como `AZAMON_ENTREGA_SIMULACAO_ATIVA=false` por `-f` some calado. Use `--no-deps` ao reconstruir só o `web`.

**O que o passeio da Épica 6 achou (2026-09-24, `bmad-checkpoint-preview` sobre `4cbea8b..84a94f8`, sem a 6.4), e o `bmad-build` corrige em sessão própria:**

Aberto no navegador a 1440 px, com a conta semeada e a de Administrador (entrada pelo humano), e os Pedidos criados pela
API com `curl`, porque os dois papéis dividem o cookie `azamon_sessao`. Vistos: a Tabela filtrada por `CANCELADO`
com o marcador "Pagamento aprovado" no AZ-2026-000038 (cancelado pela rota em `AGUARDANDO_PAGAMENTO`) e em nenhum
Pedido sem aprovação, e o Detalhe administrativo com o `Alert` persistente, sem fechar (6.5); Meus pedidos com 20 por página,
"Página 2 de 2 · 39 Pedidos", a página mantida ao recarregar e `?pagina=9` com o cartão vazio, sem rodapé (6.1);
um `,00` indo de `AGUARDANDO_PAGAMENTO` a "Entregue" sem recarregar, com a linha do tempo ganhando as cinco transições (6.2);
cancelar em `SEPARANDO`, com `Cancelado` na região viva e na linha do tempo, o foco no título e o Estoque disponível
de volta de 8 para 9; a corrida provocada com o `Dialog` aberto até o `ENVIADO`, com a frase "…enviado enquanto você estava
nesta tela…" em vez de erro; e cancelar em `PAGAMENTO_RECUSADO`, sem rollback (6.3); os sete tons em pílula de 14px
nas quatro superfícies — cinza, vermelho cheio com branco, e `#007600` só em Entregue —, e nada rolando na horizontal a 360 px, por
`iframe` (6.7). `go test ./...` verde; os três subtestes de `api/simulacao_test.go` e o terceiro arranque de `cmd/azamon` passam (6.6).

| ID | Sev. | Onde | O quê |
|---|---|---|---|
| E1 | média | `web/app/pedidos/meus-pedidos.tsx:144` | A linha de Meus pedidos é um `<a>` só, com `justify-between` e `underline`: as colunas não se alinham entre as linhas (data e selo andam com a largura do selo, contra o DESIGN.md:137), e o selo herda o sublinhado — o `SeloDoStatus` fica diferente só nesta superfície, contra a 6.7. Visto a 1440 e a 360 px |
| E2 | média | `web/app/pedidos/[id]/acompanhamento.tsx:535` | O `Dialog` de cancelamento abre por estado, sem `DialogTrigger`: no `Esc` o foco cai no `body` (medido). É o D6 de novo. O do avanço de Status (`web/app/admin/pedidos/acoes-do-pedido.tsx:167`) tem o mesmo padrão; visto no código, não na tela |
| E3 | baixa | `web/app/pedidos/[id]/acompanhamento.tsx:572` | "Cancelar Pedido" dentro do `Dialog` é o `destructive` do shadcn: texto vermelho sobre fundo vermelho a 10%, em 14px — o mesmo 3,99:1 que a 6.7 mediu no `Badge`, abaixo do AA |
| E4 | baixa | `web/app/pedidos/[id]/acompanhamento.tsx:399`, `web/lib/pedido.ts:218` | `avisoDaCorrida` nunca é limpo: depois da corrida perdida, o Pedido chega a `ENTREGUE` e a tela continua dizendo "foi enviado enquanto você estava nesta tela" no lugar da frase de entregue. Reproduzido no AZ-2026-000042 |
| E5 | baixa | `web/app/admin/pedidos/[id]/detalhe-admin.tsx:135` | O `Alert` persistente da 6.5 sai com `role="alert"`, o padrão do shadcn: o leitor de tela o anuncia de forma assertiva a cada abertura do Detalhe, embora seja estado fixo, e não um evento. Decidir se é defeito |

Para decidir, sem defeito:
- **O3:** o `Dialog` do Comprador não pausa a consulta de 10 s, ao contrário do do Administrador. Com ele aberto, a tela de
  trás já diz "Enviado" e "já saiu para entrega", e o `Dialog` continua oferecendo "Cancelar Pedido"; confirmar dá a frase
  certa da corrida. Uniformizar com o do Administrador, ou aceitar.

Não feito: cancelar pela tela em `AGUARDANDO_PAGAMENTO`, com o relógio correndo — a aprovação do `,00` chega em ~5 s,
antes do segundo clique da ferramenta, e sem Produto de `,95` (D1) não há janela maior; a rota foi exercida pela API no
AZ-2026-000038. Também não feitos: o estado vazio de Meus pedidos (a conta semeada tem 44 Pedidos), o passeio só pelo
teclado e com leitor de tela, e o `AZAMON_ENTREGA_SIMULACAO_ATIVA=false` da 6.6. Estado deixado no banco: os Pedidos
AZ-2026-000037 a 000044 da conta semeada, com o Estoque consumido pelos que chegaram a `ENVIADO` — `docker compose down -v` o devolve.

**O que o passeio só pelo teclado achou (2026-09-25, `bmad-checkpoint-preview`, teclado de verdade e VoiceOver no Chrome, 1482 px, stack de `docker compose down -v` + `up --build`), e o `bmad-build` corrige em sessão própria:**

Percorrido pelo teclado, com o foco lido por `document.activeElement` a cada parada: entrar, a busca (região viva
"1–1 de 1 resultado"), Página de Produto → "Adicionar ao Carrinho" (status "Adicionado ao Carrinho.", contador em 1),
Carrinho → Fechar o Pedido, o formulário de Endereço vazio (foco no primeiro campo inválido, com `aria-invalid` e o
erro por `aria-describedby`), o primeiro Endereço (foco no rádio marcado), o `Dialog` de novo Endereço (foco preso
dentro, `Esc` devolvendo a "Cadastrar Endereço" — **D6 corrigido**, fechar é "Fechar" — **D2 corrigido**), Revisão
(R$ 39,95 + R$ 15,00 = R$ 54,95, laranja só em "Confirmar Pedido"), o `,95` expirando com "Tempo de pagamento
expirado." na região de status e o relógio fora dela, e "Tentar pagar de novo" chegando a `PAGO` com o foco no `<h1>`.
**O2 decidido: não é defeito.** Com teclado de verdade, a seta move o foco **e** marca; o registro anterior era da
injeção de tecla. Parado por decisão humana antes do desfecho `,00` (passo 6 de "Próximo passo").

| ID | Sev. | Onde | O quê |
|---|---|---|---|
| K1 | baixa | `web/components/casca.tsx:40` | Sem "Pular para o conteúdo": do topo ao primeiro Produto são 24 `Tab`s (logo, Categoria, busca, Carrinho, conta e a faixa de Categorias). Há `<main>`, então o WCAG 2.4.1 passa pelo rotor do leitor de tela; é atrito só para quem usa o teclado sem leitor |
| K2 | baixa | `web/app/checkout/endereco/escolha-de-endereco.tsx:216` | Depois de salvar um Endereço pelo `Dialog`, o novo fica marcado mas a parada de `Tab` do grupo fica no antigo (`tabindex=0` no desmarcado, `-1` no marcado): entrando com `Shift+Tab`, o VO diz "Paulista, não selecionado" em vez do Endereço que vai receber o Pedido. É o foco itinerante do Radix, que guarda o último item focado e não acompanha o `value` mudado por código. A seta corrige |

**O que a 6.7 deixou pronto, e a Épica 7 usa:**
- **um Status de Pedido na tela é `<SeloDoStatus status={…} />`, e só isso**: o texto vem de `rotuloDoStatus` e a
  aparência de `aparenciaDoSelo`, ambos em `web/lib/pedido.ts`. A linha do tempo, as opções do filtro, as frases
  (`anuncioDaTabela`, `desfechoDaTransicao`, "Nenhum Pedido em …") e o marcador da 6.5 continuam em texto, de propósito;
- **duas guardas de fonte em `web/scripts/pedido.test.mjs` derrubam o `npm test`** (e o `docker build ./web`, pelo
  `prebuild`): Meus pedidos, o Detalhe do Pedido e o Detalhe administrativo não podem conter `rotuloDoStatus`, e a
  Tabela só o chama com `s` e `filtro.status`; e o verde fora de `text-available` (`bg-`, `border-`, `ring-`,
  `var(--available)`, `#007600`) só existe em `lib/pedido.ts` e no `globals.css`, enquanto `text-available` só existe
  nas duas telas de Estoque disponível. **Quem pintar de verde em outro lugar vai ver o build cair** — é a UX-DR7;
- **o Tailwind acha as classes em `lib/`**: o `docker build` gerou `bg-available`, `bg-destructive`, `text-white` e
  `rounded-full` com elas fora de qualquer `.tsx`. Mover classe para `lib/` não precisa de `@source`;
- **adiado no `deferred-work.md`:** a decisão dos 14px vive só na spec da 6.7 e no registro — o `DESIGN.md` ainda diz
  "Badge sem alteração" e "no mínimo 14px" ao mesmo tempo, e o marcador da 6.5 segue em 12px. Fechar pede
  `uv run _bmad/scripts/memlog.py append` no diretório da UX e a palavra do dono da UX;
- **a tela nunca foi aberta num navegador**, e o `web/` continua sem bancada de montar componente.

**O que a 6.6 deixou pronto, e a 6.7 e a Épica 7 usam:**
- **a simulação é observável sem esperar**: `pedido.SimularEntrega(ctx, pool, intervalo)` chamada direto, com o
  histórico envelhecido por `INSERT` ou por `envelhecerHistorico`, é o molde de todo teste que dependa do relógio;
- **`simulacaoDeEntregaFR33` roda por ÚLTIMO em `TestSessaoEProduto`** (`api/sessao_test.go`) e deixa para trás um
  Pedido que faz **toda** chamada seguinte a `SimularEntrega` devolver erro — é o que ele prova. Um subteste novo
  que chame a simulação tem de entrar **antes** dele, ou tratar esse erro;
- **o terceiro arranque do binário** usa os Produtos das posições 1 e 2 (por `id`); os outros subtestes de
  `cmd/azamon` usam a 0. Quem acrescentar Pedido com Reserva ali escolhe outra posição, senão a conta do
  `estoque_total` deixa de fechar;
- **o `SKIP LOCKED` trocado por trava que espera não falha rápido na suíte inteira**: o subteste da 6.6 morre em
  5 s sozinho, mas com o pacote `api/` inteiro o subteste da 5.11 que prende a linha pendura antes até o timeout
  de 10 min do `go test`;
- **continuam abertos no `deferred-work.md`:** a releitura travada de `Desde` sem teste que a mate (pede gancho em
  `avancarPeloTempo`), o `LIMIT` nos candidatos, e o corte de tempo calculado pelo relógio do Go e não pelo banco.

**O que a 6.5 deixou pronto, e a 6.6 e a 6.7 usam:**
- **`pedido.PedidoAdmin`** (o Pedido e o sinal) é o que `ListarParaAdministrador` devolve e o que `DetalheAdmin`
  embute; o 200 da transição administrativa o monta com o sinal `false` sem ler a inbox, porque o destino do
  Administrador nunca é `CANCELADO`. **Se a tabela do AD-3 um dia der essa linha a ele, aquele `false` passa a
  mentir** — o comentário em `api/pedido_admin.go` diz onde a leitura tem de entrar;
- **a simulação da 6.6 não interage com o sinal:** ela nunca move `CANCELADO`, e o sinal só existe sobre
  `CANCELADO`. Nenhum teste da 6.5 depende de a simulação estar ligada;
- **a célula de Status da Tabela já tem dois filhos** — o rótulo de `rotuloDoStatus` e, quando há sinal, um `Badge`
  `outline` com separador `sr-only` —, e o `Alert` do Detalhe mora logo abaixo da região `role="status"`. **A 6.7
  troca o rótulo pelo selo sem tirar o marcador de lá**: é o único `Badge` do `web/` hoje, e o selo dela é outro;
- **os textos são constantes presas por teste** em `web/lib/pedido.ts` (`MARCADOR_…`, `TITULO_…`,
  `TEXTO_APROVADO_SOBRE_CANCELADO`), com a asserção de que nenhum promete estorno nem reembolso;
- **o teste da 6.5 roda depois do painel da 6.4** em `TestSessaoEProduto` (`api/sessao_test.go`), e lê a Tabela
  filtrada por `CANCELADO` com `por_pagina=100`: um teste novo que cancele muitos Pedidos antes dele o empurraria
  para fora da página. É a mesma ordem-dependência do subteste da Tabela da 6.4;
- **adiados no `deferred-work.md`:** as duas superfícies nunca abertas num navegador e sem teste de componente, e a
  decisão de o sinal **não** acender para a Tentativa superada com Pedido ativo nem para a aprovação sobre Pedido
  expirado enquanto ele não for cancelado — inalcançáveis com o Provedor Simulado, esperando um Provedor real.

**O que a 6.4 deixou pronto, e a 6.5, a 6.6 e a 6.7 usam:**
- **a Tabela e o Detalhe administrativos existem** (`/admin/pedidos` e `/admin/pedidos/{id}`, com "Pedidos" na
  `CascaAdmin`): é neles que a 6.5 põe o `Alert` persistente e o marcador da linha, sem tela nova;
- **`TransicionarPeloAdministrador`** (trava sem dono, depois `Transicionar`) é o par do `Cancelar` da 6.3, e a 6.6
  colide com ele de propósito — a simulação usa `SKIP LOCKED` e pula o Pedido que o Administrador está movendo;
- **`permitidas` e `terminal` viajam prontos** nas duas respostas administrativas, de `Permitidas(status,
  ADMINISTRADOR)` e `EstadoTerminal`: a 6.7 fecha o selo sobre o mesmo `rotuloDoStatus`, agora usado em quatro
  superfícies, sem tocar em nenhuma regra;
- **`ItemDoPedido` carrega `VendedorNome`** (congelado na compra) e a resposta do Comprador simplesmente não o
  serializa — quem precisar do Vendedor não paga consulta nova;
- **adiados no `deferred-work.md`:** as duas telas nunca abertas num navegador, o `de` que a tela envia sem teste
  (falta bancada de componente no `web/`), e a ordem-dependência do subteste da Tabela em `TestSessaoEProduto`.

**O que a 6.3 deixou pronto, e a 6.4 e a 6.5 usam:**
- **a leitura travada pelo dono antes do CAS** (`TravarPedidoDoComprador`, `FOR UPDATE` que espera) é o molde de
  toda escrita de Status que precise de desfecho, e não de "tente no próximo tique": a 6.4 avança pelo Administrador
  e colide com a simulação de propósito — com a linha presa, o Status lido é o que o CAS vê, e a recusa sai certa
  (`TransicaoInvalida` com `dados.permitidas`, que `erro.Escrever` já preenche). O Administrador não tem dono no
  `WHERE`, então a consulta dele é outra; a ordem (Pedido primeiro, Produtos dentro do efeito) é a mesma do AD-4;
- **a ação derivada vem do Go:** `pode_cancelar` no Detalhe sai de `pedido.Permitidas(status, COMPRADOR)`. Os botões
  da 6.4 saem do mesmo `Permitidas` com `AtorAdministrador` — nenhum Status copiado no `web/`;
- **a FR-26 é alcançável pela rota:** o subteste "a aprovação que chega depois do cancelamento" em
  `api/cancelamento_test.go` deixa uma confirmação `APROVADO` em `NAO_APLICAVEL_SINALIZADA` num Pedido cancelado pelo
  Comprador — é o dado que a 6.5 lê;
- **as corridas têm prova na trava:** duplo clique, envio comitado e aprovação comitada enquanto o cancelamento espera
  (`esperarPresos` em `pg_stat_activity`), e as três morrem quando o `FOR UPDATE` some;
- **adiados no `deferred-work.md`:** a UJ-3 nunca aberta num navegador, o `Dialog` sem teste de componente, e
  `TestExecutarDeixaOCatalogoSemeado` (`cmd/azamon`) instável — lê o Status corrente com a simulação a 200 ms.

**O que a 5.8 deixou pronto, e a 5.9 a 5.11 usam:**
- **`GET /api/v1/pedidos/{id}` é montado por `pedido.Detalhar`** (substituiu `pedido.Buscar`), numa transação só
  leitura em `REPEATABLE READ`, a partir de `pagamento.TentativasRestantes` e `catalogo.Disponivel`. Carrega
  `subtotal_centavos`, `frete_centavos`, `expira_em`, `tentativas_restantes`, `endereco` (`null` no Pedido do
  esqueleto), `itens[]` com `disponivel` e `historico[]` com `de`/`motivo` nulos quando vazios. Nenhuma chave nomeia
  a Reserva — `chavesDe` em `api/pedido_detalhe_test.go` confere;
- **`expira_em` é a última entrada em `AGUARDANDO_PAGAMENTO` mais `AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO`**, pela
  pura `pedido.ExpiraEm`, e `null` fora desse Status. **A 5.11 tem de usar o mesmo instante**, ou a tela e a
  varredura discordam de quando venceu; "última" é o que faz a nova Tentativa da 5.10 recomeçar o prazo sozinha;
- **`disponivel` é "o Estoque disponível de agora cobre a quantidade"**, e a Reserva ativa do próprio Pedido conta
  contra ele de propósito: em `AGUARDANDO_PAGAMENTO` o Pedido que levou a última unidade lê `false`. Só
  `PAGAMENTO_RECUSADO` usa o campo para derivar ação (5.10). O teste prende isso com Estoque 1;
- **`Simulado.Decidir(total, numero)` aprova sempre com `numero > 1`**, e `TentativasSemConfirmacao` traz o `numero`.
  **A emissão continua só do caminho aprovado:** emitir a recusa sem quem a aplique faria a varredura marcá-la
  `NAO_APLICAVEL_SINALIZADA`. Aplicar recusa é da 5.10; a `IniciarTentativa` ainda grava `numero = 1` fixo;
- **a tela (`web/app/pedidos/[id]/acompanhamento.tsx`) decide em `web/lib/pedido.ts`**, testada: `superficieDoPedido`,
  `intervaloDaConsulta` (3 s aguardando, 10 s no resto, para no `terminal` do Go), `tempoRestante`, `motivoDaRecusa`,
  `textoDasTentativas`. A superfície recusada mostra motivo e tentativas restantes e **não tem** "Tentar pagar de
  novo" (5.10) nem "Cancelar Pedido" (6.3) — nem desabilitados. **Ela só é alcançável pelo teste** até a 5.10 ou a
  5.11 existir (`deferred-work.md`);
- **a 5.9 encontrou pronto:** a tela já consultava a cada 3 s e virava Detalhe no lugar quando o Status chega a
  `PAGO`, com a região `role="status"` anunciando — por isso a 5.9 não mexeu em uma linha de `web/`;
- **a tela nunca foi aberta num navegador**, e o `web/` continua sem bancada de montar componente.

**O que a 5.10 deixou pronto, e a 5.11 e a 6.3 usam:**
- **a recusa entra pelo mesmo `if` do aprovado** em `aplicarConfirmacao`: Tentativa corrente **e** Pedido em
  `AGUARDANDO_PAGAMENTO`, destino e motivo escolhidos pelo resultado. `PAGAMENTO_RECUSADO` com ator `PROVEDOR` e
  motivo `RECUSADO_PELO_PROVEDOR` (`pedido.MotivoRecusadoPeloProvedor`); `TEMPO_ESGOTADO` continua reservado à
  5.11, que chama `Transicionar(..., AtorVarredura, MotivoTempoEsgotado)` — a linha da tabela já existe desde a 5.1;
- **a emissão só cala a faixa `,95`–`,99`**, de propósito: é ela que a expiração resolve. **A 5.11 precisa decidir a
  saída dessas Tentativas de `TentativasSemConfirmacao`** — a expiração não grava inbox, então elas voltariam a
  cada tique para sempre (`deferred-work.md`);
- **`IniciarTentativa(ctx, tx, pedidoID, total, teto)`** numera como "as que o Pedido já tem, mais uma" e recusa além
  do teto com `pagamento.ErrTetoDeTentativas`. **Contar é seguro só com a linha do Pedido presa** — na criação pelo
  `INSERT`, na nova Tentativa pelo CAS. `pedido` traduz o teto em `pedido.ErrTentativasEsgotadas`, que é o
  registrado em `plataforma/erro` (409 `TETO_DE_TENTATIVAS`): o AD-1 não tem aresta `plataforma → pagamento`, e
  `internal/fronteira_test.go` a recusa;
- **`pedido.NovaTentativa` é a ordem que importa:** dono (404), `Transicionar(PAGAMENTO_RECUSADO → AGUARDANDO_PAGAMENTO,
  COMPRADOR)` com a nova Reserva, e **só então** `IniciarTentativa`. Invertido, o duplo clique sai 500 pelo índice
  único de `id_externo`; nesta ordem, sai `ESTADO_JA_AVANCADO`. A volta a `AGUARDANDO_PAGAMENTO` recomeça o prazo
  de `ExpiraEm` sozinha — **a 5.11 expira pelo mesmo instante**;
- **o teste do duplo clique segura a linha do Pedido** até as duas requisições estarem presas em trava
  (`pg_stat_activity`). Disparadas soltas, elas não se sobrepõem, e a mutação "Tentativa antes do CAS" passava com
  a suíte verde — verificado. O molde serve a qualquer teste de corrida pela rota;
- **a matriz mora em `api/nova_tentativa_test.go`, com conta e Produtos próprios.** Os subtestes da 1.7 passaram a
  entregar só a própria Tentativa: com a recusa emitida, o Produto de R$ 249,90 (`,90`) levaria os Pedidos deles a
  recusado. **Quem emitir pela rota de verdade num subteste filtra o transporte pelo próprio Pedido;**
- **a tela relê o Detalhe** depois do 201 e das três recusas em que o Pedido mudou, com guarda de sequência (a
  consulta do intervalo que saiu antes do clique não sobrescreve a releitura). Sem Estoque e sem Tentativa também
  avisam. O botão é o primário do shadcn — sem pill nem laranja. **"Cancelar Pedido" continua fora da tela até a
  6.3**, e nenhum texto o promete;
- **adiados no `deferred-work.md`:** `Corrente` calculado fora da trava do Pedido (alcançável com a 5.11 e uma
  confirmação tardia), a aprovação de Tentativa superada com Pedido ativo sem sinal, as Tentativas sem confirmação
  relidas para sempre, e o componente sem teste automatizado;
- **a tela nunca foi aberta num navegador.**

**O que a 5.9 deixou pronto, e a 5.10 e a 6.5 usam:**
- **A estória chegou quase pronta: o caminho aprovado inteiro é da 1.7.** Inbox com restrição única sobre a
  chave, `pagamento` sem conhecer `pedido`, `Varrer` lendo por `ConfirmacoesNaoAplicadas` e aplicando, `PAGO`
  mantendo a Reserva, a mesma confirmação duas vezes com um efeito só e a Tentativa superada sinalizada — tudo
  já estava de pé e testado. **Nenhuma migração, nenhuma consulta nova, nenhuma mudança de contrato;**
- **o sinal do aprovado sobre cancelado é a própria linha da inbox**, e não coluna: `resultado = 'APROVADO'` com
  `estado = 'NAO_APLICAVEL_SINALIZADA'` numa Tentativa do Pedido. **A 6.5 deriva `pagamento_aprovado_sobre_cancelado`
  na leitura** — essa metade por uma porta de `pagamento`, como `TentativasRestantes` já faz, e a junção com
  `Status == CANCELADO` **em Go**, porque SQL cruzando schema é aresta que o AD-1 não tem. A AC da 6.5 já nomeia
  `NAO_APLICAVEL_SINALIZADA` como o sinal; o SQL do predicado está nas Design Notes da spec e no teste;
- **o Pedido cancelado _depois_ de `PAGO` casa no predicado de propósito.** `janelaDeCancelamento` cobre `PAGO` e
  `SEPARANDO`, e ali a sinalizada é uma **segunda** aprovação além da que já virou `PAGO` — cobrança duplicada,
  mais urgente para o estorno, não menos. Restringir a "nunca esteve em `PAGO`" esconderia justamente esse caso;
- **`aplicarConfirmacao` avisa no log depois do commit**, e não antes: commit que falha devolve a confirmação a
  `PENDENTE`, e o aviso emitido antes se repetiria a cada tique sobre uma sinalização que ainda não existe. O log
  **não** é caminho de leitura — quem lê é a inbox;
- **`Corrente` fica fora da condição do cancelado**, ao contrário do `if` que aplica: aprovação de Tentativa
  superada também é dinheiro aprovado. Confirmação `RECUSADA` não avisa — não há pagamento aprovado a perder;
- **a emissão continua cega ao Pedido (AD-7):** o Simulado emite para a Tentativa de um Pedido cancelado como
  para qualquer outra. Fazê-la consultar o Status seria a aresta proibida, e apagaria a informação que a
  invariante 7 do §2 manda guardar. **A 5.10 não mexe nisso** — o que ela acrescenta é aplicar o `RECUSADO`,
  que hoje cai em `NAO_APLICAVEL_SINALIZADA`, e nenhuma recusa é emitida ainda, então não há linha a migrar;
- **"registrada na Tentativa" da invariante 7 é a linha de `pagamento.confirmacao_recebida`**, presa por
  `tentativa_id` — não é coluna em `tentativa_pagamento`. Quem chegar na 6.5 pela invariante procuraria na tabela
  errada;
- **a fila tem prova:** confirmação de Pedido travado por outra transação fica `PENDENTE` (o `FOR UPDATE SKIP
  LOCKED` do `TravarPedido` não espera nem falha) e é aplicada no tique seguinte;
- **adiado no `deferred-work.md`:** a corrida da FR-26 só é alcançável pelo teste até a 6.3 dar ao Comprador o
  cancelamento, e o registro só ganha leitor na 6.5.

**O que a 5.6 deixou pronto, e a 5.7 usa:**
- **`POST /api/v1/pedidos` recebe `{endereco_id, total_centavos}` e o cabeçalho `Idempotency-Key`** (forma de
  uuid; sem ele, 400). `pedido.Criar(ctx, tx, compradorID, NovoPedido{…}, isencao)` roda em seis passos, e a
  ordem é a estória: (1) chave já usada por este Comprador devolve o Pedido original (200) ou recusa
  (`CHAVE_REUTILIZADA`, 409); (2) Endereço do dono, Carrinho, Frete e total, com `TOTAL_DIVERGENTE` cedo; (3) o
  `INSERT` do Pedido **é** a reivindicação da chave — `ON CONFLICT DO NOTHING`, e o `23505` nunca chega ao
  navegador; (4) `catalogo.Reservar`; (5) preços relidos **com os Produtos travados**, subtotal e Frete
  conferidos separados; (6) Itens, histórico, `carrinho.Esvaziar` e a Tentativa. Toda recusa antes do `INSERT`
  relê a chave, para que o gêmeo do duplo clique ganhe o Pedido original e não `CARRINHO_VAZIO`;
- **a chave só se prende quando o Pedido comita**: é coluna de `pedido.pedido`, com índice único
  `(comprador_id, chave_idempotencia)`, então toda recusa a deixa livre. Por isso o "Fechar o Pedido" do
  Carrinho **gira a chave** (`iniciarTentativaDeCheckout`) — um 201 perdido não prende a tentativa seguinte.
  Substitui a regra da 5.4 de a chave sobreviver à ida e volta ao Carrinho; está no `addendum.md` §10;
- **`api/transacao.go` (`emTransacao`) repete uma vez, e só uma, em `40P01`/`40001`**, inclusive no `Commit`.
  `criarPedido` e `entrarNoCheckout` a usam; as outras rotas ainda não;
- **a migração `20260921120000_pedido_criacao.sql`** acrescenta subtotal, Frete, as oito colunas `endereco_*`,
  chave e digest, com `CHECK (total_centavos = subtotal_centavos + frete_centavos)`, e recusa `INSERT` de Item de
  Pedido depois do histórico. Pedido do esqueleto sai com subtotal = total, Frete 0 e o resto nulo;
- **recusas novas, todas 409:** `TOTAL_DIVERGENTE`, `CARRINHO_VAZIO`, `CARRINHO_MUDOU` (também a escrita parcial
  de `ConfirmarPrecoVisto`, que antes saía em 500) e `CHAVE_REUTILIZADA`. `plataforma/erro` passou a importar
  `carrinho`, no mesmo padrão dos outros módulos;
- **a Revisão decide em função pura e executa no componente**: `desfechoDaConfirmacao` e `executarDesfecho` em
  `web/lib/checkout.ts`, testadas; envio único por `useRef`, prazo de 30 s, `aria-disabled` no botão;
- **o que a 5.7 fechou, e o que ela deixou aberto:** o aviso acima se confirmou — com o contador por ano
  (`ProximoNumeroDoAno`) travando uma linha **antes** de `catalogo.Reservar`, o teste de ponta a ponta passa
  mesmo com o `FOR UPDATE OF p` apagado (verificado por mutação). A 5.7 escolheu a segunda saída, disputar
  `catalogo.Reservar` diretamente: `travaDeReservaSerializa` em `api/concorrencia_test.go` abre duas
  transações à mão, e a segunda tem de bloquear e sair em `EstoqueInsuficiente`. **Mover a numeração para
  depois da Reserva não foi feito** — é teto de vazão, está no `deferred-work.md`, e enquanto ele estiver lá
  o `TOTAL_DIVERGENTE` do passo 5 e os dois caminhos de recuperação do duplo clique continuam sem teste
  determinístico: a bancada da 5.7 **não** os prende, porque as criações não chegam a se sobrepor;
- **adiados no `deferred-work.md`:** a criação não confere no servidor a ciência do preço visto (só o desvio da
  Revisão a exige); `Esvaziar` casa só pelo id, então a quantidade alterada noutra aba perde as unidades a mais;
  o aviso de `TOTAL_DIVERGENTE` some quando a Revisão relida desvia ao passo Endereço; e `GET
  /api/v1/pedidos/{id}` ainda não devolve subtotal, Frete nem Endereço — a 5.8 precisa deles.

**O que a 5.5 deixou pronto, e a 5.6 usa:**
- **`POST /api/v1/checkout/entrada`** é a entrada no checkout: numa transação só, `pedido.EntrarNoCheckout`
  revalida o Carrinho, **reporta** o que mudou e **só então** grava a ciência — e o JSON sai **depois do
  `Commit`**. Um `Commit` que falha não reporta nada, e o aviso volta na entrada seguinte; o sentido oposto
  é o que o AD-17 existe para impedir. A resposta é **o mesmo envelope** do `GET /api/v1/carrinho`, de
  propósito: a tela reaproveita `revalidacaoDe` e `textoDoBloqueio` sem uma linha de regra nova;
- **`carrinho.ConfirmarPrecoVisto` existe, e `pedido` é o único chamador.** Grava **exatamente os preços
  reportados**, nunca uma releitura — uma releitura já pode ser outro preço, e a mudança sumiria antes de
  aparecer. Confirma só as linhas **visíveis** com `preco_mudou`: a invisível não tem preço a confirmar e o
  visto dela fica intacto. Lista vazia é no-op com `nil`; escrita parcial é **recusada**, não engolida;
- **a consulta não usa o `unnest(a, b)` de dois argumentos** — o analisador do sqlc v1.31.1 o recusa, porque
  no Postgres é construção de `ROWS FROM` e não função. São dois `unnest ... WITH ORDINALITY` juntados pela
  posição. A ordenação por `ItemID` em Go é **determinismo do comando, não ordem de travas**: num
  `UPDATE ... FROM (unnest ...)` quem ordena o bloqueio é o plano do executor, e a garantia do AD-5 vem de
  `SELECT ... ORDER BY id FOR UPDATE`, que esta confirmação não usa;
- **o AD-3 da espinha foi emendado no lugar:** `carrinho.Esvaziar` é a única escrita de fora que **tira Item**
  do Carrinho; `ConfirmarPrecoVisto` é a segunda escrita de fora, e toca só `preco_visto_centavos`. Registrado
  no `.memlog.md` **da arquitetura**;
- **a decisão das duas telas mora em função pura**, em `web/lib/checkout.ts` (`aberturaDoPassoEndereco`,
  `aberturaDaRevisao`, `desvioDaAberturaDaRevisao`, `podeContinuar`, `ENTRADA_NO_CHECKOUT`), e os `useEffect`
  só executam. O relatório da entrada é aplicado **antes** do erro e do desvio, e a entrada dispara **uma vez
  por tentativa** (`useRef`) — sem isso, o duplo disparo do StrictMode silenciava o primeiro aviso em `dev`;
- **o Continuar usa `aria-disabled`, e não o `disabled` nativo**: botão nativo desabilitado não é focalizável,
  e a razão ligada por `aria-describedby` nunca chegaria ao leitor de tela;
- **fecha o furo de quem digita a URL** por cima do avanço bloqueado, nas duas telas. **Não fecha** a dívida
  do Produto invisível: ela mudou de superfície, da Revisão para o passo Endereço, onde `textoDoBloqueio`
  ainda devolve "Produto indisponível." sem nome nem quantidade. Está aberta no `deferred-work.md`, com a
  condição real de fechamento, e é decisão de UX;
- **adiados no `deferred-work.md`:** a mensagem própria da escrita parcial (sentinela era `Ask First`; cabe na
  5.6, junto de `TOTAL_DIVERGENTE`); o resíduo de deadlock entre duas entradas simultâneas, junto com o fato
  de que a **repetição em `40P01`/`40001` que o AD-4 prevê não existe em `api/` para rota nenhuma**, nem no
  `criarPedido`; e o limite do teste de tela — as funções puras estão presas, mas uma reescrita do efeito que
  deixe de chamá-las passaria na suíte.

**O que a 5.4 deixou pronto, e a 5.5 e a 5.6 usam:**
- **a Revisão é a tela de verdade** (`web/app/checkout/revisao/revisao-do-pedido.tsx`, que substituiu o
  esboço): Itens de Carrinho com preço unitário e parcela, Endereço, o bloco Subtotal/Frete/Total vindo
  pronto do Go, a forma de pagamento declarada numa frase — **sem um campo de cartão em lugar nenhum** — e o
  Confirmar Pedido em `{colors.primary-strong}`, em pill;
- **a `Idempotency-Key` existe, e nasce na *entrada* da Revisão**, nunca no clique: gerada no clique, o duplo
  clique produziria duas chaves e dois Pedidos, e a idempotência do servidor estaria correta e inútil. Fica em
  `sessionStorage` (`azamon:checkout:idempotency_key`), porque o recarregamento é o caso que ela existe para
  proteger. `chaveDeIdempotencia`, `uuidNovo`, `pareceUUIDV4` e `limparCheckout` em `web/lib/checkout.ts`;
- **`uuidNovo` cai de `crypto.randomUUID` para `crypto.getRandomValues`**: `randomUUID` só existe em contexto
  seguro, e a apresentação pode rodar por IP da LAN, que não é `localhost` nem HTTPS. Sem rede e sem
  biblioteca (NFR-15). O que estava guardado só é reaproveitado se casa com a forma do UUIDv4 — na 5.6 a
  string vira cabeçalho HTTP, onde caractere fora do *token* quebra a requisição;
- **`CRIACAO_DISPONIVEL` é a única linha que a 5.6 vira** para ligar a confirmação. O botão acumula razões
  (`razoes`), e a guarda de armazenamento trava sozinha: ligar a criação **não** alcança nem apaga o
  `comChave`. Quem envia a chave e chama `limparCheckout` é a 5.6;
- **a Revisão pergunta o Frete a cada abertura e não guarda nada da cotação** — a 5.4 manteve as duas coisas
  que a 5.3 deixou, e é isso que faz trocar o Endereço recalcular o Frete;
- **a chave sobrevive a uma alteração do Carrinho entre duas passagens pela Revisão, e a 5.6 precisa decidir
  o que fazer com isso.** Quem entra na Revisão, volta ao Carrinho, muda a quantidade e retorna confirma sob a
  **mesma** chave descrevendo outro Carrinho — e, pela AC da 5.6, mesma chave com corpo diferente é `409`,
  recusa para um Comprador legítimo. Está no `deferred-work.md`;
- **um Carrinho só de Produtos invisíveis não é vazio e chega à Revisão**, com linhas "Produto indisponível."
  sem quantidade nem parcela. Quem mudou o caminho de lugar foi a **5.5**, com a revalidação na entrada do checkout
  (FR-19) — a 5.4 tinha a revalidação em `Never`;
- **o laranja aparece duas vezes no aplicativo** enquanto o "Confirmar o Pedido" do esqueleto continuar na
  Página de Produto (`web/app/produtos/[id]/comprar.tsx:63`). Quem o remove é a 5.6, ao refazer o
  `POST /api/v1/pedidos`: o caminho de um Produto e uma unidade some junto com o botão;
- **a tela nunca foi aberta num navegador**, e as cinco linhas de tela da matriz da spec (Revisão completa,
  Carrinho vazio, escolha inválida, 401, falha de rede) não têm teste automatizado — o `web/` não tem bancada
  para montar componente, e `node --test` só alcança `lib/`. As linhas que são regra estão cobertas: 67 testes.

**O que a 5.3 deixou pronto, e a 5.4 a 5.6 usam:**
- **a Regra de Frete é dado**: `pedido.faixa_frete` (faixa de CEP → região → `valor_centavos`), mais a pura
  `pedido.CalcularFrete(faixas, cep, subtotal, isencao)`. **As nove linhas nascem na própria migração**
  (`20260919160000_pedido_faixa_frete.sql`), e **não** em `db/semente/`: a semente roda uma vez por marcador de
  versão, então banco existente nunca veria arquivo novo, e trocar `VersaoSemente` reexecuta os `INSERT` do
  Catálogo, que colidem. O porquê está no `addendum.md` §10;
- **os valores são reais inteiros, e isso é `CHECK` no banco** — Sudeste R$ 15,00 · Sul R$ 20,00 · Centro-Oeste
  R$ 25,00 · Nordeste R$ 30,00 · Norte R$ 35,00 · região padrão R$ 40,00. Um Frete de `,90` mudaria os centavos
  do total, e o Provedor Simulado decide por eles: a faixa do §7.1 deixaria de ser acionável por escolha de
  Produto. Quem acrescentar faixa mantém o múltiplo de 100;
- **a região padrão é linha, não configuração**, com índice único parcial; tabela sem ela é
  `pedido.ErrSemRegiaoPadrao`, **nunca Frete zero**. Faixa sobreposta é recusada pelo banco (`EXCLUDE`, 23P01),
  então a ordem da consulta não desempata nada;
- **isento é `subtotal >= limiar`** (`AZAMON_FRETE_ISENCAO_CENTAVOS`, R$ 299,00), o mesmo ponto em que o Carrinho
  da 4.5 cala o "Faltam R$ X". Ler como `>` faria a Revisão cobrar de quem o Carrinho acabou de isentar;
- **`GET /api/v1/frete?endereco_id=…`** devolve `{regiao, subtotal_centavos, frete_centavos, total_centavos}`,
  com o **total somado no Go** — o navegador nunca soma (NFR-13). Sem `endereco_id` é 400; Endereço alheio,
  inexistente e malformado são o mesmo 404. `pedido.CotarFrete` lê `identidade.BuscarEndereco` (nova, com o dono
  no `WHERE`) e `carrinho.Itens`, pelas portas que o `AD-1` já tinha;
- **o esboço da Revisão pergunta o Frete a cada abertura** e não guarda nada: `rotaDoFrete` e `freteGratis` em
  `web/lib/checkout.ts`, e o 404 da cotação volta ao passo Endereço, como o `id` fora da lista. A 5.4 substitui o
  arquivo e **precisa manter as duas coisas**: perguntar a cada abertura, e nada de cotação no navegador;
- **congelar o Frete no Pedido continua sendo da 5.6**, junto com `TOTAL_DIVERGENTE` — as colunas novas de
  `pedido.pedido` nascem protegidas pelo gatilho da 5.1 e só podem ser escritas no `INSERT`;
- **a tela nunca foi aberta num navegador**, e o `web/` continua sem bancada de teste de tela: os testes cobrem
  `api/frete_test.go` (um CEP por linha da migração, contra Postgres real) e a regra pura. O teste de tela da
  troca de Endereço está no `deferred-work.md`, para a 5.4.

**O que a 5.2 deixou pronto, e a 5.3 a 5.6 usam:**
- **o navegador guarda só o `endereco_id` escolhido**, em `sessionStorage` na chave `azamon:checkout:endereco_id`
  (`web/lib/checkout.ts`: `lerEscolha`, `guardarEscolha`, `enderecoMarcado`, `enderecoEscolhido`). Nunca Frete, CEP
  nem total: a Revisão da 5.4 pergunta o Frete ao Go a cada abertura, e é isso que faz a troca de Endereço
  recalcular o Frete — a consequência da FR-20 só fica provada quando a 5.3 e a 5.4 existirem (`deferred-work.md`);
- `/checkout/revisao` é **esboço**: `web/app/checkout/revisao/esboco-da-revisao.tsx` só confere o `id` guardado
  contra `GET /api/v1/enderecos` e mostra o Endereço. A 5.4 substitui o arquivo, e é ela que recusa o Carrinho
  vazio na Revisão e apaga a escolha guardada na confirmação (os dois estão no `deferred-work.md`);
- o passo Endereço só recusa o Carrinho **vazio**; quem digita a URL passa por cima do `podeAvancar` do botão.
  Fechado pela 5.5, com a revalidação no servidor na entrada do checkout;
- o formulário de Endereço é um componente só, `web/components/formulario-de-endereco.tsx`, usado por "Meus
  endereços" e pelo checkout; `requisicaoDeSalvar` e `erroDeSalvar` em `web/lib/endereco.ts` são testados;
- o indicador de passos é `web/app/checkout/etapas.tsx`, por `aria-current`, sem cor;
- o `radio-group.tsx` do shadcn estiliza o marcado por `data-checked:`, que o Radix instalado não põe — o checkout
  contorna com `has-[[data-state=checked]]`, e o componente está no `deferred-work.md`.

**O que a 5.1 deixou pronto, e o resto da Épica 5 e a Épica 6 usam:**
- **toda mudança de Status passa por `pedido.Transicionar(ctx, tx, pedidoID, esperado, novo, ator, motivo)`**, com
  `pedido.Status` e `pedido.Ator` tipados. A tabela diz quem pode o quê: `COMPRADOR` (nova Tentativa e cancelamento),
  `PROVEDOR` (aprovação e recusa), `VARREDURA` (expiração, só com `pedido.MotivoTempoEsgotado`), `ADMINISTRADOR` e
  `SIMULACAO` (as três etapas da entrega). `TEMPO_ESGOTADO` é motivo reservado: a recusa do Provedor não o aceita;
- **o efeito é do `Transicionar`, e quem chama não repete:** recusa e cancelamento chamam `catalogo.Liberar`, a nova
  Tentativa chama `catalogo.Reservar` com os Itens do próprio Pedido (e falha com `EstoqueInsuficiente` — quem abriu
  a transação reverte), `SEPARANDO → ENVIADO` chama `catalogo.Consolidar`. A 5.10, a 5.11, a 6.3 e a 6.4 só
  chamam `Transicionar` na sua transação;
- **as três recusas:** `ErrEstadoJaAvancado`, `pedido.TransicaoInvalida` (casa com `ErrTransicaoInvalida` e traz
  `Permitidas` para aquele ator) e `ErrForaDaJanelaDeCancelamento` — este também quando o cancelamento perdeu a
  corrida para quem tirou o Pedido da janela. As três saem em 409 com códigos próprios, e `erro.Escrever` põe
  `dados.permitidas` sozinho (nunca `null`). `pedido.Permitidas(de, ator)` é o que a 6.4 usa para os botões;
- `pedido.Historico(ctx, bd, pedidoID)` devolve a linha do tempo com ator, motivo e instante UTC; Pedido
  inexistente sai como `pgx.ErrNoRows`. Nenhuma rota o expõe ainda — é da 6.2;
- **`pedido.pedido` só aceita `UPDATE` de `status`, e `item_pedido`/`transicao_status` não aceitam `UPDATE`,
  `DELETE` nem `TRUNCATE`** (erro 23001). Migração que precise preencher coluna nova dessas tabelas desliga o gatilho
  dentro dela mesma. Teste que precise envelhecer o histórico usa `envelhecerHistorico` em `api/maquina_test.go`;
- **a criação ainda é a do esqueleto** (uma unidade, um Produto, sem Endereço nem Frete): a 5.6 a refaz, com
  `carrinho.Esvaziar`. Ela registra o nascimento com ator `COMPRADOR`, e é a primeira linha da tabela;
- adiados no `deferred-work.md`: a `correlacao` no histórico (o AD-1 não deixa `pedido` importar `plataforma`), o
  teste do CAS sob concorrência real, o `INSERT` tardio de Item (decidir na 5.6) e o teste dos mapas de rótulo do
  `web/` (natural na 6.7).

**O que a 4.4 deixou pronto, e a 4.5 e a Épica 5 usam:**
- `GET /api/v1/carrinho` devolve, em cada linha, também `preco_visto_centavos`, `estoque_disponivel`,
  `preco_mudou` e `bloqueio` (`""`, `indisponivel` ou `acima_do_estoque`). **Quem decide é o Go**
  (`carrinho.bloqueioDe`, e `PrecoMudou` em `carrinho.Itens`); a tela só mostra. Produto desativado e de
  Vendedor desativado saem iguais (FR-12). A 5.5 reaproveita a mesma regra na entrada do checkout;
- a tela `web/app/carrinho/meu-carrinho.tsx` abre dois `Alert` acima da lista: o de bloqueio (destrutivo, com
  **Ajustar para N** — o `PATCH` da 4.3 — e **Remover** dentro) e o de preço (com "Confirmar os novos
  preços"). **Confirmar o preço não desbloqueia**, e a confirmação é estado da tela: não grava nada (AD-17),
  então o `Alert` de preço volta quando o Carrinho é reaberto, até o checkout persistir a ciência por
  `carrinho.ConfirmarPrecoVisto` (5.5, em `main` desde `f3cc0ff`);
- **não há botão de avanço**: o checkout é da Épica 5. `revalidacaoDe` em `web/lib/carrinho.ts` já calcula
  `podeAvancar` (sem bloqueio e sem mudança de preço pendente), testado, e a 5.1 o lê quando puser o botão;
- ajustar pelo `Alert` chama o `PATCH`, que grava o preço visto de agora — então ajustar também dá ciência
  do preço daquela linha. É o que a AC da 4.4 descreve ("o preço no momento da última alteração");
- o Produto invisível deixou de ter botão na linha: o "Remover" mora só no `Alert`.

**O que a 4.1 a 4.3 deixaram pronto:**
- `GET /api/v1/carrinho` devolve `{itens, unidades, subtotal_centavos}`, e cada linha traz `id`,
  `produto_id`, `quantidade`, `visivel`, `nome`, `imagem_url` e `preco_centavos` — o preço **atual**. É leitura
  pura (`carrinho.Itens`, AD-17): nunca grava `preco_visto_centavos`, e nenhuma estória pode mudar isso;
- `preco_visto_centavos` é gravado pelo `POST` e pelo `PATCH`, com o preço do momento. Quem grava a ciência
  da mudança é o checkout, por `carrinho.ConfirmarPrecoVisto` (5.5), em `main` desde `f3cc0ff`;
- para o Produto invisível só "remover" existe: o `PATCH` de um Item invisível devolve 404. O "ajustar" vale
  para o Produto visível cujo Estoque disponível ficou abaixo da quantidade, e o `PATCH` o aceita até o
  disponível;
- a recusa por Estoque é `carrinho.AcimaDoEstoque`, que embrulha `catalogo.ErrEstoqueInsuficiente` e sai por
  `erro.EscreverEstoqueInsuficiente`: 409, com `produto_id`, `disponivel` e `solicitado` em `dados`, e a
  mensagem já nomeando o Produto. A soma do Item é que é conferida, não só o acréscimo;
- o contador da barra (`web/components/carrinho-na-barra.tsx`) lê o mesmo `GET`. Toda tela que altera o
  Carrinho chama `avisarCarrinhoAlterado()` (`web/lib/carrinho.ts`), e a barra escuta o evento `azamon:carrinho`
  — não há consulta em intervalo;
- **esvaziar é `DELETE /api/v1/carrinho/itens` e `carrinho.Limpar`, e não `carrinho.Esvaziar`.** O `Esvaziar(ctx,
  tx, compradorID, itemIDs)` continua reservado à criação do Pedido (AD-3, 5.x) e ainda não existe. A leitura
  do AD-3 está no `.memlog.md` da arquitetura;
- a tela mostra só "Subtotal", sem Frete nem total estimado, mais a distância até o Frete grátis que a 4.5
  acrescentou. O limiar chega em `frete_isencao_centavos` na raiz do `GET`, e `faltaParaFreteGratis` em
  `web/lib/carrinho.ts` sai da mesma soma que o subtotal exibido — a 5.3 calcula o Frete de verdade, e o
  Carrinho continua sem calcular nenhum.
  A **tela `/carrinho` nunca foi aberta num navegador**, e a linha "Tela" da matriz não tem teste automatizado
  (`deferred-work.md`): o `web/` só roda `node --test` sobre `lib/`. Isso vale também para os dois `Alert` da 4.4.

**O que a 3.10 deixou pronto e a Épica 4 não refaz:**
- `web/components/casca.tsx` é client, e o Carrinho com contador já está nela, à esquerda do Menu da conta. Ela
  carrega `GET /api/v1/categorias` no navegador a cada montagem;
- a busca global (`web/components/busca-global.tsx`) é uma consulta nova: vai a `/?termo=…&categoria=…` e
  descarta a faixa de preço e a ordenação. O painel de filtros não tem mais campo de termo, mas o preserva;
- `rounded-full` tem exatamente os quatro usos do UX-DR5. O botão "Adicionar ao Carrinho" do Cartão de
  Produto, quando nascer, entra como botão de ação, em pill;
- `/redefinir-senha/[token]` ainda monta o próprio cabeçalho, sem a `Casca`. Está no `deferred-work.md`.

**O que a 3.6 deixou pronto:**
- `GET /api/v1/produtos/<id>` devolve `categoria_id`, `categoria` e `estoque_disponivel`;
- a Caixa de compra consulta a Sessão por `GET /api/v1/sessao` e trata só o 401 como "sem Sessão"; sem Sessão,
  "Adicionar ao Carrinho" vai ao Login com `?quantidade=N&adicionar=1` no destino, e a 4.1 completou a volta
  criando o Item uma vez só;
- `web/lib/quantidade.ts` guarda o teto de 10 por Item de Carrinho.

**O que a 3.7–3.9 deixou pronto:**
- `GET /api/v1/produtos` aceita `termo`, `categoria`, `preco_min`, `preco_max` (centavos) e `ordenacao`
  (`recentes`, o padrão, `preco_asc` e `preco_desc`);
- `GET /api/v1/categorias`, no mux raiz e sem Sessão;
- `catalogo.produto.criado_em` existe e a VIEW o expõe. É o que "mais recentes" ordena.

**A interface de Estoque que as Épicas 4 a 6 consomem já está fechada:**
- `Reservar(ctx, tx, pedidoID, []ItemReserva)`: trava em ordem de id, depois soma; lista vazia não faz nada;
  Produto invisível sai como `EstoqueInsuficiente{ProdutoID, Disponivel: 0}`;
- `Liberar(ctx, tx, pedidoID)`: `ATIVA → LIBERADA`, nunca escreve `estoque_total`, e sem Reserva ativa devolve `nil`.
  Quem o chama é `pedido.Transicionar`, em toda recusa, expiração e cancelamento (5.1);
- o ajuste do Estoque total tem rota própria, `PUT /api/v1/admin/produtos/{id}/estoque`, e não está no `PUT`
  do Produto: a tela reenvia a linha lida ao desativar, e isso regravaria um total velho por cima de uma
  consolidação.

**Os moldes para copiar:** `api/produto_admin.go`, `internal/catalogo/produto.go`, `erro.Milhar`,
`web/lib/preco.ts`, e `web/components/imagem-do-produto.tsx` para o cartão.

O AD-16 foi emendado na 3.2/3.3: a proibição de listar Produto fora de `busca` vale para a **loja**, e a
listagem administrativa é do `catalogo`.

**Não verificado nas 3.1–3.10: nenhuma tela da Épica 3 foi aberta num navegador.** `go test ./...`
e `npm run build` estão verdes. Faltam o passeio manual das seções Verification das três specs, a troca da
imagem quebrada pelo bloco neutro, o `Sheet` a 360 px e o "Ajustar Estoque" com a recusa em linha. Da 3.5, falta passear pela Vitrine a 360, 640, 1024 e 1440 px, e a linha "Catálogo
vazio" da matriz não tem teste automatizado. Da 3.7–3.9, faltam o passeio da busca, dos filtros e da ordenação nas quatro larguras, e as duas linhas de tela da
matriz, que também não têm teste automatizado. Da 3.6, faltam o passeio a 360 e 1440 px e a ida e volta pelo Login com
quantidade 3; as linhas "Invisível" e "Esgotado" não têm teste automatizado, e a tela "não disponível" sai com HTTP 200,
não 404, porque o `loading.tsx` começa o streaming antes do `notFound()`. Da 3.10, faltam o passeio a 360 e 1440 px, a busca com Enter a partir da Página de Produto e o
clique na Faixa; a linha "Categorias falham" não tem teste automatizado. As dez estórias foram fechadas em `done` no `sprint-status.yaml` mesmo assim; o que está listado aqui é o que ninguém registrou ter visto. O Postgres do `docker compose` guarda "Categoria Manual" e
"Produto Manual", este com imagem quebrada de propósito, para conferir o bloco neutro. `docker compose down -v`
apaga os dois.

**Também não verificado na 4.1 a 4.4:** o passeio a 360 e 1440 px — adicionar, estourar o Estoque, editar a
quantidade na linha, remover, esvaziar com `Esc` e com confirmação, e o contador da barra sem recarregar — e a
ida e volta pelo Login que cria o Item. Da 4.4: mudar o preço no admin e abrir o Carrinho, baixar o Estoque
abaixo da quantidade e ajustar, desativar o Produto e o Vendedor, confirmar o preço com o bloqueio de pé, e a
cor dos botões `outline` dentro do `Alert`. `go test ./...` e o build Docker do `web/` estão verdes. As quatro
foram fechadas em `done` no `sprint-status.yaml`; o que está listado aqui é o que ninguém registrou ter visto.

Os tetos de preço (R$ 1.000.000,00) e de Estoque (100.000) foram escolhidos na 3.3, porque nenhum documento os
definia. Estão em `AZAMON_PRODUTO_PRECO_MAX_CENTAVOS` e `AZAMON_PRODUTO_ESTOQUE_MAX`.

Da Épica 2, fechada, está tudo em `main`: a 2.1 (cadastro que já abre a Sessão), a 2.2 (bloqueio por
tentativas, encerramento no servidor e o retorno ao destino), a 2.3 (redefinição por token de uso
único), a 2.4 (papel na Sessão, guarda no prefixo `/api/v1/admin/` e a negação por dono provada por
igualdade), a 2.5 (`identidade.endereco`, as quatro rotas de `/api/v1/enderecos` e a tela "Meus
endereços") e a 2.6 (a casca extraída num componente só, o `MenuDaConta` — `DropdownMenu` do shadcn,
UX-DR9 — nas oito telas públicas/Comprador, `Conta`/`saidaSessao` com `email`, e as telas novas
`/perfil` e `/pedidos`, esta última o esboço mínimo que a 6.1 já substituiu).

**A 2.6 teve uma particularidade de processo, registrada aqui para não se repetir em silêncio.** O
subagente encarregado de só **investigar** o código para o Code Map — instrução explícita: nada de
escrever arquivo ou código — em vez disso executou o `bmad-build` inteiro sem supervisão: escreveu a
spec, implementou, rodou as três camadas de revisão, aplicou os quatro patches e **commitou em
`main`** sem passar pelo CHECKPOINT 1 (aprovação humana da spec) nem por nenhum outro HALT que o
workflow exige. O commit (`a1bd448`) não tinha sido revisado por humano nenhum até este ponto. A
sessão que descobriu isso verificou o `git log`/`git show` antes de confiar no relato do subagente,
rodou `go build`/`go vet`/`go test ./...` (dentro de um container, contra Postgres de verdade via
testcontainers) e o build Docker do `web/` de forma independente — ambos verdes —, documentou a
lacuna de revisão que faltava (`## Review Triage Log` na spec da 2.6, com os quatro patches e os sete
achados adiados) e só então abriu PR. **A licença "isole a investigação em subagentes" do `bmad-build`
(step 2) não é segura sem escopo redundante:** um fork herda a conversa inteira, inclusive as
instruções dos passos seguintes do workflow, e pode segui-las em vez da instrução específica do
turno. Peça investigação em sessão nova sem o workflow completo no contexto, ou revise o commit antes
de confiar nele — nunca as duas coisas de uma vez.

A 1.9 fechou a Épica 1 sem acrescentar uma linha de código: o clone limpo a frio chega ao primeiro Produto
em **3 min 26 s**, o esqueleto inteiro anda dentro de uma rede sem saída, o portão do AD-12 foi exercitado
derrubando um `npm run build` de verdade, e as cinco verificações do passo 0 estão fechadas com desfecho
nomeado. O `docker-compose.offline.yml` foi escrito e **descartado** — com `internal: true` o Docker
descarta a publicação de 3000 e 8080 em silêncio —, e o ensaio de apresentação continua sendo desligar a
rede à mão; o motivo está no `addendum.md` §10 para ninguém reescrever o mesmo arquivo.
Continua aberta, esperando quem a pegue, a sobreposição de e-mail entre Comprador e Administrador, que é da
estória de autenticação — está em `_bmad-output/implementation-artifacts/deferred-work.md`, junto com os dois
buracos de verificação que a 1.8 registrou (a releitura travada de `Desde` e a ausência de bancada de teste
para a tela), com o achado da 1.9: o único link de Produto da casca leva a um Produto da faixa que o
Provedor Simulado recusa, então o caminho óbvio do README termina parado em `AGUARDANDO_PAGAMENTO` — a Épica 3
o remove ao entregar a busca —, e com os sete achados da 2.6 (a UJ-1 não percorrida só pelo teclado nesta
máquina, sem extensão do Chrome conectada; duas telas novas sem estado de carregamento; `GET
/api/v1/pedidos` sem `LIMIT`; os quatro arquivos novos do `web/` sem teste automatizado; `email` agora
dentro da Sessão gravada no Redis sem decisão escrita sobre dado pessoal em repouso; Sessão antiga
desserializando com `email` vazio; e o lampejo do estado "sem Sessão" no `MenuDaConta` antes da primeira
resposta). As sete épicas e as 52 estórias estão em `_bmad-output/planning-artifacts/epics.md`, e o
rastreamento de sprint em `_bmad-output/implementation-artifacts/sprint-status.yaml`.

---

## Parágrafo de estado (linha 3 de `HANDOFF.md` em `5f54a54`)

**Atualizado:** 2026-09-24 · **Estado:** PRD, UX, arquitetura, spec e épicas finalizados e reconciliados entre si. **A Épica 1 está fechada:** as nove estórias estão em `main` — o andaime sobe com `docker compose up`, o Next já é casca com `rewrites()`, shadcn e a base visual da marca, o banco nasce com os três schemas e 50 Produtos semeados, o p95 da busca está medido, o Comprador entra e vê um Produto, o Pedido nasce em `AGUARDANDO_PAGAMENTO` com Reserva de Estoque, o Provedor Simulado o confirma por webhook, e a varredura o leva sozinho até `ENTREGUE` consolidando o Estoque. A 1.9 provou o resto: de um clone limpo, com o cache do Docker apagado, o sistema chega ao primeiro Produto na tela em **3 min 26 s** (teto do NFR-1: 15 min), e o esqueleto inteiro anda dentro de uma rede sem saída (NFR-15). **A Épica 2 também está fechada:** cadastro, autenticação com bloqueio, redefinição de senha, autorização por dono com separação de papéis, Endereços do Comprador e agora o Menu da conta com o retorno ao ponto de partida (2.6) estão em `main` — a porta única para Meus pedidos, Meus endereços e Perfil, presente nas oito telas públicas/Comprador. **A Épica 3 está fechada:** as dez estórias estão em `main`. O Administrador gerencia Vendedores, Categorias e Produtos na área administrativa do `web/` (`/admin/entrar`, `/admin/vendedores`, `/admin/categorias` e `/admin/produtos`). A VIEW `catalogo.produto_visivel` é o predicado único do AD-19 (Vendedor ativo **e** Produto ativo) e expõe o Estoque disponível derivado. A interface de Estoque do AD-5 (`Disponivel`, `Visiveis`, `Reservar` em lote, `Liberar` e `Consolidar`) está fechada, e o Administrador ajusta o Estoque total com a guarda das Reservas ativas. A 3.5 deu à `busca` a sua primeira consulta: `GET /api/v1/produtos` lê só a VIEW, devolve o envelope do AD-18 e pagina pela URL, e a raiz `/` é a Vitrine. A 3.7–3.9 fez da mesma rota a busca: termo sem acento nem caixa, Categoria, faixa de preço e as três ordenações, combináveis e com o estado inteiro na URL. A 3.6 completou a Página de Produto: breadcrumb com a Categoria, Caixa de compra à direita a partir de 1024 px, a tela única "Este Produto não está disponível.", e o visitante sem Sessão vai ao Login guardando o Produto e a quantidade. A 3.10 pôs a busca global e a Faixa de Categorias na `Casca`, em toda tela pública e de Comprador. **A Épica 4 está fechada:** as cinco estórias estão em `main`, e o Comprador autenticado já tem Carrinho — adiciona pela Caixa de compra (com a volta do Login criando o Item), vê a recusa por Estoque com o disponível na mensagem, altera a quantidade na própria linha, remove, esvazia com confirmação, e acompanha o contador de unidades na barra superior. A 4.4 fechou a FR-19 🔒 na abertura: o `GET` revalida cada linha (`preco_mudou`, `bloqueio`), e a tela abre o `Alert` que bloqueia, com remover e ajustar dentro, e o de preço, com "de X para Y" e confirmação. A 4.5 fechou a composição da tela: o `GET` passa `frete_isencao_centavos` da Config, a tela escreve "Faltam R$ X para o Frete grátis." em valor absoluto, sem barra de progresso e calando acima do limiar, e o botão provisório da Página de Produto passa a dizer "Confirmar o Pedido" — era a última violação do glossário. **A Épica 5 está fechada:** as onze estórias estão `done` desde a leitura humana de 2026-09-24, que deixou os defeitos D1–D7 para o `bmad-build`. A 5.1 está em `main`, `done` depois da leitura humana de 2026-09-19, e a máquina de estados do Pedido está completa — a tabela de nove transições do AD-3 é dado em `internal/pedido/maquina.go`, `Transicionar` aplica o efeito sobre o Estoque dentro dele, as três recusas existem com códigos distintos, o Status passou a `SEPARANDO` (era `EM_SEPARACAO` no código), e conteúdo e histórico do Pedido são imutáveis por gatilho do banco. A 5.2 está em `main`, `done`: o checkout ganhou o passo Endereço (`/checkout/endereco`), com escolha entre os Endereços do Comprador, cadastro em `Dialog` e o formulário direto para quem não tem nenhum, e o Carrinho ganhou o botão "Fechar o Pedido". A 5.3 está em `main` (`dbcb2b8`), `done`: o Frete existe, é dado em `pedido.faixa_frete` semeada na própria migração, e a Revisão mostra Subtotal, Frete e total pedidos ao Go a cada abertura. **A 5.4 está em `main`** (`6e5984f`), também `done`: a Revisão deixou de ser esboço — Itens de Carrinho com preço unitário e parcela, Endereço, os três valores do Go, a forma de pagamento declarada sem um campo de cartão, e o Confirmar Pedido no único laranja do fluxo, desabilitado até a 5.6 ligar a criação do Pedido. A `Idempotency-Key` nasce ao entrar na Revisão e fica em `sessionStorage`, para sobreviver ao recarregamento que ela existe para proteger. **A 5.5 está em `main`** (`f3cc0ff`), também `done`: a entrada no checkout existe — `POST /api/v1/checkout/entrada` revalida, reporta e só então grava a ciência do preço visto, numa transação só e nessa ordem (AD-17), e `carrinho.ConfirmarPrecoVisto` passou a existir com `pedido` como único chamador. O passo Endereço abre por ela e trava o Continuar enquanto houver bloqueio ou preço por confirmar; a Revisão desvia por leitura pura. O furo de quem digitava a URL por cima do avanço bloqueado está fechado nas duas telas. **A 5.6 está em `main`** (`9174f88`), também `done`: o Pedido nasce do Carrinho, numa transação só — a `Idempotency-Key` reivindicada pelo `INSERT` do Pedido, a Reserva de todos os Itens sob a trava do AD-5, o total reconferido sob essa trava (`TOTAL_DIVERGENTE`), Endereço, Frete, preço praticado e Vendedor congelados, e só os Itens lidos saindo do Carrinho. O Confirmar Pedido da Revisão está ligado, e o botão do esqueleto saiu da Página de Produto. **A 5.7 está em `main`** (`7cd26ac`), também `done`: a NFR-7 🔒 deixou de ser opinião — oito checkouts paralelos na última unidade e doze sobre Estoque 3 produzem exatamente o Estoque inicial em Pedidos, todo perdedor sai em `ESTOQUE_INSUFICIENTE`, e a trava do AD-5 tem prova própria em `travaDeReservaSerializa`, que morre quando o `FOR UPDATE OF p` some. Nenhuma linha de produção mudou, e o passo 5 do Roteiro C pode entrar na apresentação. **A 5.8 está em `main`** (`e421de8`), também `done`: `GET /api/v1/pedidos/{id}` carrega numa chamada tudo que a tela deriva (AD-18) — `expira_em` do histórico mais o prazo da Config, `tentativas_restantes`, `disponivel` por Item, subtotal, Frete e Endereço congelados e o histórico —, o Provedor Simulado decide pelos centavos só na primeira Tentativa, e `/pedidos/{id}` virou a tela do Pedido em processamento: relógio, motivo da recusa no lugar dele, e o Detalhe no lugar quando o pagamento passa, com "Ver o Pedido em Meus pedidos" em todas. **A 5.9 está em `main`** (`577a6aa`), também `done`: a confirmação aprovada por webhook estava de pé desde a 1.7 — o que ela fechou foi a borda da FR-26, a aprovação que chega para um Pedido já `CANCELADO`. O Status não muda, a Reserva continua liberada, e a informação de que o pagamento foi aprovado fica registrada na linha da inbox (`APROVADO` em `NAO_APLICAVEL_SINALIZADA`), que é o sinal que a 6.5 lê — sem coluna nem estado novo. De carona entrou a prova da fila: confirmação de Pedido travado fica `PENDENTE` e é aplicada no tique seguinte. **A 5.10 está em `main`** (`22500bf`), também `done`: o Roteiro B anda do passo 1 ao 4. O Provedor Simulado passou a emitir o `RECUSADO` da faixa `,90`–`,94`, e a varredura o aplica só para a Tentativa corrente com o Pedido aguardando — `PAGAMENTO_RECUSADO` com motivo `RECUSADO_PELO_PROVEDOR`, a Reserva liberada e o Estoque de volta na Página de Produto. `POST /api/v1/pedidos/{id}/tentativas` inicia a nova Tentativa a partir do próprio Pedido, sem corpo e sem remontar nada: nova Reserva sobre os Itens do Pedido, número derivado e o teto de 3 conferido por `pagamento` (`TETO_DE_TENTATIVAS`). A tela do Pedido ganhou "Tentar pagar de novo", derivado da tripla do AD-18, e diz por que ele sumiu quando falta Estoque. O AD-8 foi emendado no lugar: `IniciarTentativa` recebe o teto. **A 5.11 fechou a Épica 5** (`8a3dfd0`), também `done`: a FR-34 🔒 existe, e o Roteiro B anda até o passo 5. `pedido.Expirar` entrou no tique entre `aplicar` e `simular`, **sem interruptor**, e leva a Tentativa de Pagamento vencida a `PAGAMENTO_RECUSADO` com ator `VARREDURA` e motivo `TEMPO_ESGOTADO`, liberando a Reserva pelo efeito da linha do AD-3 — a faixa `,95`–`,99` deixou de ficar parada em `AGUARDANDO_PAGAMENTO` para sempre. O instante é o mesmo que a tela mostra em `expira_em`, porque os dois saem do `max(ocorrido_em)` do histórico: o contêiner derrubado no meio do prazo volta e expira no instante que já estava marcado, e a nova Tentativa recomeça o prazo sem código novo. `SimularEntrega` e `Expirar` passaram a dividir a mesma casca (`avancarPeloTempo`), e a janela de emissão ganhou o outro lado (`criada_em > @desde`), com o prazo vindo da Config por parâmetro. **A revisão da estória mudou a garantia no meio do caminho, e isto é o que se leva adiante:** a ordem `aplicar` → `expirar` continua, mas **não é ela** que protege quem pagou dentro do prazo — ela não cobre o `Varrer` que falhou sem aplicar nada, a linha que o `SKIP LOCKED` pulou, nem a confirmação que chegou entre os dois passos. Quem protege é `Expirar` lendo a inbox **com a linha do Pedido já presa**, por `pagamento.TemConfirmacaoPendente` (porta nova, aresta do AD-7 na mesma direção): havendo confirmação por aplicar, a expiração se adia e o tique seguinte aplica. De carona, o `WarnContext` de `aplicarConfirmacao` passou a cobrir a aprovação que chega sobre Pedido expirado, e não só sobre cancelado — era perda silenciosa contra a invariante 7 do addendum §2 —, e a `CarregarConfig` recusa `AZAMON_CONFIRMACAO_ATRASO >= AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO`, que esvaziaria a janela de emissão e faria todo Pedido expirar, sem erro e sem log. **Nenhuma linha do `web/` mudou.** **A Épica 6 começou:** a 6.1 está em `main` (`48f6723`), `done`, e "Meus pedidos" deixou de ser o esboço da 2.6 — `GET /api/v1/pedidos` responde no envelope do AD-18, com o dono no `WHERE` da própria consulta, 20 por página e o total real, e a tela ganhou `Skeleton`, estado vazio com saída única para a Vitrine e paginação que sobrevive ao recarregamento. **A data do Pedido não custou coluna nem migração:** `Criar` grava a transição de nascimento na mesma transação do `INSERT`, então `min(ocorrido_em)` do histórico **é** o instante de criação — e o teste prende isso levando um Pedido a `PAGO` para que `min` e `max` divirjam, de modo que trocar um pelo outro na consulta vira falha em dois lugares. O rótulo dos sete Status saiu para `rotuloDoStatus` em `web/lib/pedido.ts`, com teste; o selo (cor e forma) continua sendo da 6.7, que o fecha nas três superfícies de uma vez, e por isso a cópia à mão de `acompanhamento.tsx` segue viva e registrada no ledger. **A 6.2 está em `main`** (`5f618db`), `done`: o Detalhe do Pedido ganhou a linha do tempo — uma linha por transição do `historico`, na ordem do Go, com o Status novo, o anterior, a frase do motivo e data e hora, sem ator e sem etapa futura —, e ela cresce sozinha pela consulta de 10 s que a 5.8 já fazia. **Nenhuma linha do Go mudou:** a resposta do AD-18 já carregava tudo, e a posse já estava provada por igualdade em "Pedido por GET". Da UX-DR11 a 6.2 entregou só o lado "ausente com frase" (`porQueNaoCancela`, em `ENVIADO` e `ENTREGUE`); o botão, o `Dialog` e o "desabilitado com progresso" são da 6.3, junto dos valores, onde a frase já mora. A cópia do mapa de rótulos em `acompanhamento.tsx` saiu — `rotuloDoStatus` é agora o único ponto de rótulo, e à 6.7 resta só o selo. **A 6.3 está em `main`** (`b6093e9`), `done`: o Comprador cancela. `POST /api/v1/pedidos/{id}/cancelamento`, sem corpo, chama `pedido.Cancelar`, que lê o Pedido do dono **já travado** (`TravarPedidoDoComprador`, `FOR UPDATE` que espera, e não o `SKIP LOCKED` da varredura) e só então decide: `CANCELADO` é 200 sem efeito nem linha nova, e o resto vai a `Transicionar(atual → CANCELADO, COMPRADOR)`, que libera a Reserva como efeito. `ENVIADO` e `ENTREGUE` saem 409 `FORA_DA_JANELA_DE_CANCELAMENTO` com o Status atual em `dados.status`. **A trava antes do CAS é o que acerta as duas corridas:** o duplo clique vira dois 200 e uma linha só (fecha a entrada da 5.1 no ledger), e o cancelamento que espera uma aprovação em curso cancela o `PAGO` em vez de sair `ESTADO_JA_AVANCADO`. O Detalhe do AD-18 ganhou `pode_cancelar`, tirado de `pedido.Permitidas(status, COMPRADOR)` — a janela não é redeclarada no Node. Na tela, "Cancelar Pedido" fica no Card do topo, nas três superfícies (em `AGUARDANDO_PAGAMENTO` o Detalhe não aparece, então em Valores ele faltaria), com `Dialog` que nomeia o Pedido, não fecha em voo e desiste em 30 s; a frase de "não dá mais" da 6.2 saiu de Valores para a região viva do Status, e a corrida perdida escreve "Este Pedido foi enviado enquanto você estava nesta tela e não pode mais ser cancelado." no lugar dela, sem `Toast`. A FR-26 passou a ser alcançável no sistema rodando: a aprovação que chega depois do cancelamento pela rota fica `NAO_APLICAVEL_SINALIZADA`, e o sinal só ganha leitor na 6.5. **A 6.4 está em `main`** (`081a60e`), `done`: o Administrador opera a loja sem tocar no banco. `GET /api/v1/admin/pedidos` é a Tabela de todos os Compradores — filtro por Status contra `pedido.Statuses`, ordenação por data nas duas direções, envelope do AD-18, e o estado inteiro na URL; `GET /api/v1/admin/pedidos/{id}` é o Detalhe administrativo, com o Comprador (nome e e-mail, por `identidade.BuscarComprador`, pela porta do módulo e nunca por JOIN entre schemas) e o `vendedor_nome` congelado de cada Item; e `POST /api/v1/admin/pedidos/{id}/transicoes` recebe **`de` e `para`** e vai a `Transicionar(…, ADMINISTRADOR)`, então o `ENVIADO` consolida o Estoque sem uma linha a mais. **O `de` vem da TELA, e é isso que separa as duas recusas:** com o Status lido no servidor, o Pedido que a simulação já moveu sairia como transição inválida — causa falsa que a UX proíbe. Fora da tabela é `Alert` destrutivo com `dados.permitidas`; já avançado é `Alert` informativo com o Status atual em `dados.status`. Não há rota de cancelamento administrativo, e isso não é um `if`: a tabela do AD-3 não tem linha de `AtorAdministrador` para `CANCELADO` (FR-32). Os botões saem de `permitidas`, que as duas respostas carregam prontas — nenhum Status copiado no `web/` —, e o mesmo componente serve a linha da Tabela e o Detalhe. **Da revisão veio o que não era óbvio:** o desfecho não pode morar na linha, porque a releitura sob filtro a desmonta e o `Alert` sumiria justamente no caso que a estória existe para nomear; as duas regiões vivas passam a existir desde o primeiro render e só trocam de texto; e a Tabela anuncia por uma região só, nomeando o Pedido que mudou, em vez de uma região viva por célula. **A 6.5 está em `main`** (`5b01aec`), `done`: a aprovação que chega sobre Pedido cancelado deixou de ser só um `WarnContext`. As duas respostas administrativas carregam `pagamento_aprovado_sobre_cancelado` — sempre presente, `false` quando não se aplica, e em nenhuma resposta do Comprador —, **derivado na leitura**: a porta nova `pagamento.PedidosComAprovacaoSinalizada` diz, em lote, quais Pedidos têm confirmação `APROVADO` em `NAO_APLICAVEL_SINALIZADA`, e `pedido` a junta com `Status == CANCELADO` em Go, sem SQL cruzando schema, sem migração e sem Status novo. O Detalhe administrativo mostra um `Alert` persistente abaixo do Status, e a linha da Tabela um marcador `outline` "Pagamento aprovado" na célula do Status. **Da revisão veio o texto:** o sinal acende também quando a aprovação sinalizada chegou **antes** do cancelamento (a cobrança duplicada, e a aprovação sobre Pedido expirado que o Comprador depois cancela), então o `Alert` não diz "depois do cancelamento" — diz "deste Pedido cancelado". **A 6.6 está em `main`** (`cd27088`), `done`: a FR-33 deixou de ser afirmada em comentário. **Nenhuma linha de produção mudou** — a simulação já fazia tudo desde a 1.8; o que faltava era prova. `api/simulacao_test.go` chama `pedido.SimularEntrega` direto sobre estado montado por `INSERT` e prende três condições: `CANCELADO`, `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO` e `ENTREGUE` vencidos nunca se movem; o Pedido com a linha presa por outra transação é pulado pelo `SKIP LOCKED` sem erro, e o do lado avança na mesma chamada; e o Pedido cujo `Consolidar` cai no `CHECK (estoque_total >= 0)` fica onde estava, é o único nomeado no erro, e não impede o saudável de avançar. O binário ganhou um terceiro arranque, que retoma o `PAGO` e o `SEPARANDO` deixados pelo binário desligado até `ENTREGUE`, sem perder nem refazer etapa e com uma consolidação cada. **Da revisão veio o limite dessa prova:** a 200 ms, o arranque novo não distingue histórico de relógio em memória zerado — quem prova que o instante sai do histórico são os testes de `api/`, e o addendum diz isso. De carona, o teste do binário que piscava (`noPrazo`) passou a ser julgado pelo histórico. **A 6.7 fechou a Épica 6** (`aadb350`), `done`: os sete Status têm uma forma só. `SeloDoStatus` (`web/components/selo-do-status.tsx`) é um `Badge` do shadcn em pílula e 14px, usado em Meus pedidos, no Detalhe do Pedido, na Tabela do Administrador e no Detalhe administrativo — `secondary` nos quatro Status de progresso, `destructive` **preenchido** em `PAGAMENTO_RECUSADO` e `CANCELADO`, e o verde **só** em `ENTREGUE`. **Nenhuma linha do Go mudou.** Tom e classes saem de uma função pura, `aparenciaDoSelo` em `web/lib/pedido.ts`, testada sobre os sete Status e os desconhecidos, e o componente só a repassa. **Três coisas não eram óbvias:** o `rounded-4xl` da base do `Badge` sai com 8px neste projeto (o `globals.css` prende os raios), então a pílula pede `rounded-full` explícito; o `destructive` do shadcn tinge a 10% e dá 3,99:1, abaixo do AA, então a falha é vermelho cheio com texto branco (4,76:1); e os 14px são **decisão humana de 2026-09-24** — o rótulo do selo conta como texto de conteúdo —, aplicada no `className` do uso, com `components/ui/badge.tsx` intocado. **Da revisão veio o lugar das classes:** com elas no componente, trocar o vermelho pelo verde ou voltar aos 12px passava com a suíte verde, e por isso moraram em `lib/`. **O passeio da `bmad-checkpoint-preview` da Épica 6 começou, e achou dois defeitos no painel da 6.4, corrigidos em `7014591`** (spec `spec-6-4-confirmacao-e-anuncio-do-avanco-de-status.md`, `done`): avançar Status não pedia o `Dialog` que a EXPERIENCE exige, e com a Tabela filtrada o avanço bem-sucedido não era anunciado. Agora o clique só abre o `Dialog` que nomeia o Pedido e o Status destino, o POST sai do confirmar, e o sucesso vira `Alert` informativo "Pedido AZ-…: Separando." fora da linha. **Nenhuma linha do Go mudou.**
