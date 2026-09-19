---
title: 'Estória 4.4 — Revalidação do Carrinho na abertura'
type: 'feature'
created: '2026-09-19'
status: 'done'
baseline_commit: '252a88fbf191827d2cc528da8f3b0f988f09ebd4'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-4-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** o Carrinho abre sem avisar que um preço mudou desde a última alteração, e o Produto que ficou sem Estoque suficiente, desativado ou de Vendedor desativado aparece como linha comum ou como "Produto indisponível." sem `Alert`. Sem isso, o Comprador chegaria ao botão irreversível para receber a recusa da FR-24 (FR-19 🔒).

**Approach:** o `GET /api/v1/carrinho` passa a devolver o preço visto, o Estoque disponível e duas conclusões que **o Go decide** (`preco_mudou`, `bloqueio`). A tela abre dois `Alert`: o que **bloqueia**, com remover e ajustar dentro dele, e o de preço, com "de X para Y" e confirmação. Nada é gravado.

## Boundaries & Constraints

**Always:**
- `carrinho.Itens` continua leitura pura: nunca grava `preco_visto_centavos` (AD-17). A confirmação de preço é estado da tela, some ao recarregar, e o `Alert` volta até o checkout persistir a ciência (5.5).
- Quem decide `bloqueio` e `preco_mudou` é o Go (AD-10), com preço e Estoque só pela porta de `catalogo` (AD-19). `bloqueio`: `indisponivel` (invisível, ou disponível 0) e `acima_do_estoque` (0 < disponível < quantidade). Produto desativado e de Vendedor desativado saem iguais, sem distinção (FR-12).
- Confirmar o preço **não** desbloqueia. O bloqueio só cai por remover ou ajustar, com a quantidade que o Go informou.
- Texto com o glossário (UX-DR16): "Produto", "Pedido", nunca "Reserva", nunca "preço garantido". Valores em `tabular-nums`.

**Ask First:** rota, coluna ou sentinela além das listadas; qualquer coisa que grave no Carrinho a partir do `GET`; um botão de avanço na tela (não existe checkout antes da Épica 5).

**Never:** `ConfirmarPrecoVisto`, o recálculo do Frete e a entrada do checkout (5.5); a composição e o texto de promessa da tela (4.5); consulta em intervalo; regra de preço ou Estoque no Node.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Preço mudou | Item com visto 5000, preço atual 6000 | linha com `preco_mudou: true`, `preco_visto_centavos: 5000`; `Alert` "de R$ 50,00 para R$ 60,00", com "Confirmar os novos preços" | — |
| Preço igual | visto = atual (inclusive depois de mudar e voltar) | `preco_mudou: false`, nenhum `Alert` de preço | — |
| Confirmar preço | clique no botão | o `Alert` de preço some; o de bloqueio, se houver, fica; nada grava | recarregar traz o `Alert` de volta |
| Acima do Estoque | Item com 4, disponível 2 | `bloqueio: acima_do_estoque`; `Alert` "Restam 2 unidades de {nome}; o Carrinho pede 4." com **Ajustar para 2** e **Remover** | ajustar é o `PATCH` de 2; o 409 aparece na linha |
| Sem Estoque | Produto visível, disponível 0 | `bloqueio: indisponivel`; só **Remover** | — |
| Desativado | Produto ou Vendedor desativado | `visivel: false`, `bloqueio: indisponivel`, sem nome nem preço, fora do subtotal; só **Remover** | `PATCH` segue 404 |
| Limpo | nada mudou | `bloqueio: ""`, `preco_mudou: false`; nenhum `Alert` | — |
| Ler não grava | qualquer uma acima | `preco_visto_centavos` no banco intacto | — |
| Duas mudanças | preço mudou **e** acima do Estoque na mesma linha | os dois `Alert`; ajustar grava o preço visto (é uma alteração, 4.3) | — |

</frozen-after-approval>

## Code Map

- `internal/carrinho/db/consultas.sql` -- `ListarItens` não seleciona `preco_visto_centavos`; entra na lista, continua sem `UPDATE`. Depois `sqlc generate` (`ListarItensRow`).
- `internal/carrinho/carrinho.go` -- `ItemDoCarrinho` ganha `PrecoVistoCentavos`, `EstoqueDisponivel`, `PrecoMudou` e `Bloqueio`; `Itens` já tem o `Resumo` (que traz `EstoqueDisponivel`) e o comentário "O aviso que bloqueia é da 4.4" deve sair. As constantes de `Bloqueio` e a regra ficam aqui, em função pura.
- `internal/catalogo/catalogo.go:143` -- `Resumos` já devolve `EstoqueDisponivel`; nada a mudar.
- `api/carrinho.go` -- `saidaLinhaCarrinho` e `lerCarrinho` levam os quatro campos novos.
- `api/carrinho_test.go` -- molde: `carrinhoNaTela` (Produtos pelo admin, `produtoCom` PUT muda o preço e desativa, `precoVistoNoBanco`); ligar o subteste novo em `api/sessao_test.go:498`. Reduzir o Estoque: `PUT /api/v1/admin/produtos/{id}/estoque` (`api/estoque_test.go`).
- `web/lib/carrinho.ts` + `web/scripts/carrinho.test.mjs` -- tipo da linha; entra a função que reparte o Carrinho em bloqueios e mudanças de preço ainda não confirmadas, e o texto de unidades (singular e plural).
- `web/app/carrinho/meu-carrinho.tsx` -- `Linha` (`!linha.visivel`) troca o botão por texto; os dois `Alert` entram acima da lista; `alterar` e `remover` são reusados pelas ações do `Alert`. `web/components/ui/alert.tsx` tem `Alert`, `AlertTitle`, `AlertDescription`.

## Tasks & Acceptance

**Execution:**
- [x] `internal/carrinho/db/consultas.sql` + `sqlc generate` -- `preco_visto_centavos` em `ListarItens` -- leitura pura, restaurar com `git checkout` os `gerado/` de outros módulos que o LF reescrever.
- [x] `internal/carrinho/carrinho.go` -- campos, constantes `BloqueioIndisponivel` e `BloqueioAcimaDoEstoque`, a regra e `Itens` preenchendo -- o Go decide (AD-10), e a 5.5 reaproveita.
- [x] `api/carrinho.go` -- `preco_visto_centavos`, `estoque_disponivel`, `preco_mudou`, `bloqueio` na saída.
- [x] `api/carrinho_test.go` + `api/sessao_test.go` -- as linhas de API da matriz, num subteste novo -- NFR-8: falha se o `bloqueio` ou o `preco_mudou` quebrar, e se ler gravar.
- [x] `web/lib/carrinho.ts` + `web/scripts/carrinho.test.mjs` -- a repartição em bloqueios e mudanças, com "confirmar preço não desbloqueia" -- teste de `lib`.
- [x] `web/app/carrinho/meu-carrinho.tsx` -- `Alert` destrutivo de bloqueio (Remover e Ajustar para N dentro dele) e `Alert` de preço com confirmação em estado local; a linha indisponível deixa de ter botão próprio.
- [x] `_bmad-output/implementation-artifacts/sprint-status.yaml` -- 4-4 para `review` (o passeio no navegador falta).

**Acceptance Criteria:**
- Dado um preço alterado pelo Administrador, quando o Carrinho abre, então o `Alert` mostra o preço anterior e o atual de cada Produto, e `preco_visto_centavos` no banco continua o antigo.
- Dado um Produto indisponível, desativado ou de Vendedor desativado, quando o Carrinho abre, então o `Alert` de bloqueio o nomeia igual e só some quando o Produto é removido ou a quantidade ajustada; confirmar o preço não o tira.
- Dado o código, quando inspecionado, então nenhuma função de `carrinho` além de `Adicionar`, `AlterarQuantidade`, `RemoverItem` e `Limpar` escreve, e `ConfirmarPrecoVisto` não existe.

## Spec Change Log

## Design Notes

**Não há botão de avanço nesta estória, e a tela diz isso pela cópia.** O checkout é da Épica 5, então o "bloqueia o avanço" da FR-19 fica como estado que a 5.1/5.5 lê: a tela calcula `podeAvancar` (sem bloqueio e sem mudança de preço pendente) em `lib`, testada, e o título do `Alert` diz "Ajuste o Carrinho antes de fazer o Pedido." em vez de "continuar".

**Confirmar é da tela; ciência é do checkout.** A confirmação guarda, por Item, o preço que estava na tela; se o preço mudar de novo, o `Alert` volta. Não vai ao servidor: se fosse, o `GET` ou uma rota nova escreveria `preco_visto_centavos` e a mudança de preço sumiria antes de o checkout reportá-la (AD-17).

**O ajustar chama o `PATCH` da 4.3**, que já recusa acima do disponível e grava o preço visto de agora; então ajustar também dá ciência do preço daquela linha, e o `Alert` de preço mostra o "de X para Y" ao lado no mesmo instante em que ela é oferecida.

## Verification

**Commands:**
- `sqlc generate && go vet ./... && go test -count=1 ./...` -- verde, inclusive `TestFronteiraDeModulo` e `TestSessaoEProduto` (na imagem `azamon-dev`, ver HANDOFF).
- `cd web && npm test && docker build ./web` -- verde.

**Manual checks (if no CLI):**
- Passeio a 360 e 1440 px: mudar o preço no admin e abrir o Carrinho; baixar o Estoque abaixo da quantidade e ajustar; desativar o Produto; desativar o Vendedor; confirmar o preço com o bloqueio ainda de pé.

## Review Triage Log

Duas lentes (Edge Case Hunter e Verification Gap), rodadas na própria sessão, sem subagente, e por isso sem o isolamento de contexto que as lentes pedem. A Blind Hunter não rodou: a estória é de raio pequeno (leitura pura, sem escrita nova), e a política do projeto é uma ou duas lentes fora da concorrência e da máquina de estados. Nenhum achado de `intent_gap`, `bad_spec` ou `patch`.
- **defer:** o `Alert` de bloqueio e o de preço não têm teste automatizado, e a tela não foi aberta num navegador (`deferred-work.md`). A regra que os alimenta tem: `bloqueio` e `preco_mudou` na API, `revalidacaoDe` e `textoDoBloqueio` em `lib`.
- **reject:** dois Produtos invisíveis dão o mesmo texto e o mesmo `aria-label` no `Alert` — já era assim na linha da 4.3, e o Produto invisível não tem nome a dar.
- **reject:** `podeAvancar` é calculado e não consumido pela tela. É de propósito (Design Notes): a 5.1 o lê quando o checkout existir, e está testado.
- **reject:** `ListarItens` e `Resumos` são duas idas sem transação, então o Produto pode sair da visibilidade entre elas. É o conselho do AD-5; o `PATCH` recusa em 404 ou 409 e a tela recarrega.
- **verificado, não achado:** a cor dos botões `outline` dentro de `Alert` (herdam a do texto do `Alert`) só se confere no passeio.

## Suggested Review Order

**A regra é do Go (AD-10)**

- `bloqueioDe`: invisível ou sem unidade é indisponível; disponível abaixo da quantidade é acima do Estoque.
  [`carrinho.go:83`](../../internal/carrinho/carrinho.go#L83)

- `Itens` preenche preço visto, disponível, `PrecoMudou` e `Bloqueio` sem gravar nada.
  [`carrinho.go:237`](../../internal/carrinho/carrinho.go#L237)

- A consulta só ganha a coluna: leitura pura continua leitura pura (AD-17).
  [`consultas.sql:52`](../../internal/carrinho/db/consultas.sql#L52)

- Os quatro campos novos saem na linha do `GET`.
  [`carrinho.go:39`](../../api/carrinho.go#L39)

**A tela só mostra**

- Bloqueio primeiro, com Ajustar e Remover dentro do `Alert`; confirmar preço não o tira.
  [`meu-carrinho.tsx:204`](../../web/app/carrinho/meu-carrinho.tsx#L204)

- O `Alert` de preço lista "de X para Y" e a confirmação fica só em estado local.
  [`meu-carrinho.tsx:237`](../../web/app/carrinho/meu-carrinho.tsx#L237)

- A linha indisponível perde o botão, que agora mora no `Alert`.
  [`meu-carrinho.tsx:332`](../../web/app/carrinho/meu-carrinho.tsx#L332)

- `revalidacaoDe` reparte bloqueios e mudanças pendentes; `podeAvancar` é o que a 5.1 vai ler.
  [`carrinho.ts:93`](../../web/lib/carrinho.ts#L93)

- O texto do bloqueio nomeia o Produto, diz o disponível e nunca fala em Reserva.
  [`carrinho.ts:115`](../../web/lib/carrinho.ts#L115)

**Testes**

- A matriz de API: preço, disponível, sem Estoque, desativado, Vendedor desativado e ler não grava.
  [`carrinho_test.go:392`](../../api/carrinho_test.go#L392)

- O subteste entra ao lado dos da 4.1 a 4.3.
  [`sessao_test.go:502`](../../api/sessao_test.go#L502)

- Confirmar preço não desbloqueia, e o aviso volta se o preço muda de novo.
  [`carrinho.test.mjs:69`](../../web/scripts/carrinho.test.mjs#L69)
