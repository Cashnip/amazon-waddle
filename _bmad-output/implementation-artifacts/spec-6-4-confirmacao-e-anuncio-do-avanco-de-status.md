---
title: 'Confirmação e anúncio do avanço de Status no painel do Administrador'
type: 'bugfix'
created: '2026-09-24'
status: 'done'
baseline_commit: '22e3d7d3a558d1c4f1a73d87a625599653c5779a'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** No painel de Pedidos do Administrador (6.4), avançar o Status aplica a transição no clique, sem o `Dialog` que a EXPERIENCE exige (`EXPERIENCE.md:131`, `:193`, `:272`). E, com a Tabela filtrada por Status, o avanço bem-sucedido não é anunciado: a releitura tira a linha, e a região viva da Tabela só anuncia Pedido que continua na lista. De carona: o Alert de `ESTADO_JA_AVANCADO` diz "Este Pedido", ambíguo quando a linha já saiu.

**Approach:** `AcoesDoPedido` passa a abrir um `Dialog` controlado que nomeia o Pedido e o Status destino; o POST só sai no confirmar. O sucesso vira um relato informativo ("Pedido {numero}: {Status}."), renderizado pela superfície fora da linha — o mesmo caminho que o Alert de `ESTADO_JA_AVANCADO` já usa e que sobrevive à saída da linha. Esse Alert passa a nomear o Pedido pelo número.

## Boundaries & Constraints

**Always:** Só `web/`. Os botões continuam saindo de `permitidas` do Go; o `de` enviado continua sendo o Status que a tela leu. Durante o envio o Dialog não fecha (Esc, clique fora, "Voltar") e os botões ficam com `aria-disabled`, como no Dialog do cancelamento (6.3). Um só nível de Dialog. Textos com os termos do §3 do PRD ("Pedido", "Status do Pedido").

**Ask First:** Qualquer mudança no Go ou no contrato de `POST .../transicoes`.

**Never:** Regra de Status no Node (nenhuma lista de transições, nenhum `if` por Status). Toast. Laranja no botão de confirmar — o laranja é do checkout; o confirmar usa a variante `default`.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Clique no botão | Pedido 2026-000042 em `PAGO`, clique em "Iniciar a separação" | Abre Dialog "Mudar o Pedido 2026-000042 para Separando?"; nenhum POST | N/A |
| Voltar | Dialog aberto, "Voltar" ou Esc | Fecha; nenhum POST | N/A |
| Sucesso sob filtro | `?status=PAGO`, confirma | Dialog fecha, POST, Alert informativo "Pedido 2026-000042: Separando." fica na tela depois de a linha sair | N/A |
| Sucesso sem filtro | Sem filtro, confirma | Mesmo Alert; a região viva da Tabela não repete o mesmo anúncio | N/A |
| Corrida | 409 `ESTADO_JA_AVANCADO`, `dados.status=ENVIADO` | Alert "O Pedido 2026-000042 já está em Enviado. A linha foi atualizada." | Releitura |
| Erro | 500 / rede | Dialog fecha; Alert destrutivo como hoje | Botões voltam a aceitar clique |

</frozen-after-approval>

## Code Map

- `web/app/admin/pedidos/acoes-do-pedido.tsx` -- `AcoesDoPedido` (usado pela linha da Tabela e pelo Detalhe) e `DesfechoNaTela` (regiões vivas fora da linha). Hoje `acionar(destino)` faz o POST direto no `onClick`.
- `web/lib/pedido.ts:441` `rotuloDaAcao`, `:485` `desfechoDaTransicao(r, tentada)` -- sucesso devolve `alerta: null`; `:508` texto "Este Pedido já está em …". `:172` `textosDoCancelamento` é o modelo para os textos do Dialog.
- `web/app/admin/pedidos/pedidos.tsx` -- `anteriores` (ref) + `anuncioDaTabela` geram a região viva do intervalo; `AcoesDoPedido` recebe `aoMudar={carregar}`.
- `web/app/admin/pedidos/[id]/detalhe-admin.tsx:143` -- a mesma `AcoesDoPedido`; o objeto `pedido` tem `numero`.
- `web/app/pedidos/[id]/acompanhamento.tsx:530-580` -- padrão do Dialog de cancelamento (controlado, `showCloseButton={false}`, guarda em ref, `aria-disabled`).
- `web/scripts/pedido.test.mjs:485-545` -- testes de `desfechoDaTransicao`; `:449` teste dos textos do cancelamento.

## Tasks & Acceptance

**Execution:**
- [x] `web/lib/pedido.ts` -- `desfechoDaTransicao` recebe `tentada: { numero, de, para }`; 200 devolve `alerta: { variante: "informativo", texto: "Pedido {numero}: {rotuloDoStatus(para)}." }`; `ESTADO_JA_AVANCADO` diz "O Pedido {numero} já está em …". Nova `textosDaTransicao(numero, destino)` → `{ titulo, descricao, voltar, confirmar: rotuloDaAcao(destino), enviando }`. -- texto testável fora do componente.
- [x] `web/app/admin/pedidos/acoes-do-pedido.tsx` -- `pedido` passa a incluir `numero`; `onClick` só guarda o destino e abre o Dialog; o POST sai do confirmar; fechar antes de relatar e reler. Foco: ao fechar depois de um envio, `onCloseAutoFocus` não devolve ao botão clicado (a releitura o tira); depois da releitura, se o bloco ainda estiver montado, foca o primeiro botão dele. Sob filtro a linha sai e o foco fica no documento — quem informa é o Alert em `role="status"`. -- defeito 1.
- [x] `web/app/admin/pedidos/pedidos.tsx` -- `aoMudar` da linha apaga `anteriores.current[p.id]` antes de `carregar()`, para a região do intervalo não repetir o anúncio que o relato já fez. -- defeito 2 sem anúncio duplo.
- [x] `web/scripts/pedido.test.mjs` -- atualizar os casos de `desfechoDaTransicao` (sucesso e corrida com número) e testar `textosDaTransicao`.

**Acceptance Criteria:**
- Given a linha ou o Detalhe de um Pedido com transição permitida, when o Administrador clica no botão, then nenhuma requisição sai até ele confirmar no Dialog que nomeia o Status destino.
- Given o Dialog em envio, when o Administrador aperta Esc ou "Voltar", then o Dialog continua aberto.
- Given `grep` em `web/`, then nenhuma lista de Status destino nova além de `rotulosDaAcao` existente.

## Design Notes

O sucesso vira relato em vez de mexer em `anuncioDaTabela` porque só quem clicou sabe o destino: a linha que sai do filtro não diz para onde foi, e ela também sai quando é a simulação que a move. No Detalhe, o Alert de sucesso soma-se ao anúncio da região "Status:"; aceito — é o mesmo relato nas duas superfícies, e o Detalhe não tem filtro.

## Verification

**Commands:**
- `cd web && npm test` -- expected: todos passam.
- `cd web && npx tsc --noEmit` -- expected: sem erro.

**Manual checks (if no CLI):**
- `docker compose up`, `/admin/pedidos?status=PAGO`: clicar, ver o Dialog, confirmar, ver o Alert "Pedido …: Separando." com a linha fora da lista.

## Suggested Review Order

**Confirmação antes do POST**

- O clique só guarda `{ de, para }` e abre o Dialog; o `de` é o Status visto no clique.
  [`acoes-do-pedido.tsx:152`](../../web/app/admin/pedidos/acoes-do-pedido.tsx#L152)

- O POST sai daqui, com a guarda de envio único e o Dialog fechando antes do relato.
  [`acoes-do-pedido.tsx:91`](../../web/app/admin/pedidos/acoes-do-pedido.tsx#L91)

- Textos do Dialog: nomeia Pedido e Status destino, confirmar repete o rótulo da ação.
  [`pedido.ts:455`](../../web/lib/pedido.ts#L455)

- A Tabela pausa a consulta de 10 s com o Dialog aberto, para a linha não sumir.
  [`pedidos.tsx:131`](../../web/app/admin/pedidos/pedidos.tsx#L131)

**Anúncio que sobrevive à saída da linha**

- O 200 vira Alert informativo nomeando o Pedido, renderizado fora da linha.
  [`pedido.ts:508`](../../web/lib/pedido.ts#L508)

- Sem o Status anterior, a região viva da Tabela não repete o anúncio.
  [`pedidos.tsx:275`](../../web/app/admin/pedidos/pedidos.tsx#L275)

**Foco**

- O foco espera o commit da releitura e vai ao primeiro botão que restou.
  [`acoes-do-pedido.tsx:124`](../../web/app/admin/pedidos/acoes-do-pedido.tsx#L124)

- No fechamento pós-envio, não devolve o foco ao botão que vai sumir.
  [`acoes-do-pedido.tsx:174`](../../web/app/admin/pedidos/acoes-do-pedido.tsx#L174)

**Testes**

- Sucesso e corrida agora com número; textos do Dialog testados.
  [`pedido.test.mjs:496`](../../web/scripts/pedido.test.mjs#L496)
  [`pedido.test.mjs:635`](../../web/scripts/pedido.test.mjs#L635)
