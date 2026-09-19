---
title: 'Estória 5.3 — Cálculo do Frete'
type: 'feature'
created: '2026-09-19'
status: 'done'
baseline_commit: 'c707ff64f4555e193f603917190a4faa7f86bd61'
review_loop_iteration: 0
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-5-context.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O Frete é termo do glossário sem regra, sem tabela e sem rota: a Revisão não tem o que mostrar, e "trocar o Endereço recalcula o Frete" (FR-20/FR-21) não é demonstrável.

**Approach:** Criar `pedido.faixa_frete` já com as faixas na migração, uma função pura em `internal/pedido` que converte CEP + subtotal + limiar em Frete, e a leitura `GET /api/v1/frete?endereco_id=…` que devolve região, subtotal, Frete e total. O esboço da Revisão (5.2) ganha o bloco com as três parcelas, e o Go é quem soma.

## Boundaries & Constraints

**Always:**
- Regra é dado (AD-17): faixas em `pedido.faixa_frete`; limiar de `cfg.FreteIsencaoCentavos` (AD-13). Nenhum `if` de região ou valor no Go nem no TS.
- Faixas (Correios, 8 dígitos, texto): Sudeste `01000000–39999999` R$ 15,00 · Nordeste `40000000–65999999` R$ 30,00 · Norte `66000000–69999999`, `76800000–76999999`, `77000000–77999999` R$ 35,00 · Centro-Oeste `70000000–76799999`, `78000000–79999999` R$ 25,00 · Sul `80000000–99999999` R$ 20,00 · região padrão (uma linha, sem faixa) R$ 40,00.
- Todo valor em reais inteiros (`valor_centavos % 100 = 0`, por `CHECK`): o Provedor Simulado decide pelos centavos do total.
- Isento quando `subtotal ≥ limiar` — o mesmo limite em que o Carrinho (4.5) cala o "Faltam R$ X".
- `int64` de centavos; nenhuma divisão; o Frete não é rateado por Item (AD-9). O navegador não soma: mostra o `total_centavos` do Go.
- `carrinho` não calcula Frete; `pedido` lê `carrinho.Itens` (subtotal) e o Endereço do dono via `identidade`, pelas portas (AD-1).
- Glossário literal; sem verde nem laranja na tela.

**Ask First:**
- Mudar valores ou faixas acima; qualquer coluna nova em `pedido.pedido`; tocar `/api/v1/carrinho` ou `/api/v1/enderecos`.

**Never:**
- Congelar Frete no Pedido e `TOTAL_DIVERGENTE` (5.6); revalidação e `ConfirmarPrecoVisto` (5.5); a Revisão de verdade, `Idempotency-Key` e recusa do Carrinho vazio (5.4).
- Frete no Carrinho; consulta de CEP pela rede (NFR-15); semente em `db/semente/`.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Na faixa | CEP `01310100`, subtotal 18900 | Sudeste, Frete 1500, total 20400 | — |
| Fora de faixa | CEP `00000001` | região padrão, 4000 — não erro | — |
| Limite exato | subtotal = limiar (29900) | Frete 0 | — |
| Abaixo por 1 | subtotal 29899 | valor da região | — |
| Fronteira de faixa | `39999999` / `40000000` | Sudeste / Nordeste | — |
| Carrinho vazio | subtotal 0 | Frete da região, total = Frete | — |
| Endereço alheio, inexistente ou malformado | `endereco_id` | — | 404, mesmo corpo nos três |
| Sem `endereco_id` | query vazia | — | 400 `ENTRADA_INVALIDA` |
| Sem Sessão | — | — | 401 → Login com destino |
| Sem linha padrão | tabela sem `padrao` | — | erro (500), nunca Frete 0 |

</frozen-after-approval>

## Code Map

- `db/migracoes/*_pedido_*.sql` -- o sqlc do `pedido` recorta por este prefixo (`sqlc.yaml:52`); goose por timestamp.
- `internal/pedido/pedido.go` -- porta do módulo; já importa `catalogo`/`pagamento`. `pedido → carrinho` e `pedido → identidade` já estão em `internal/fronteira_test.go:30`.
- `internal/pedido/db/consultas.sql` -- acrescentar `ListarFaixasDeFrete` (`ORDER BY cep_inicio`, padrão à parte).
- `internal/carrinho/carrinho.go:214` -- `Itens(...)` → `Conteudo.SubtotalCentavos` (só Itens visíveis, preço de agora). Só leitura.
- `internal/identidade/endereco.go:115` -- molde de dono no WHERE; `uuidDe` malformado → `pgx.ErrNoRows`. Não há leitura de um Endereço só: criar `BuscarEndereco`.
- `internal/plataforma/config.go:109` -- `FreteIsencaoCentavos` já existe (29900).
- `api/endereco.go`, `api/carrinho.go:159` -- molde de handler: `semCache`, `compradorDaRequisicao`, `erro.Escrever`, 404 do `ErrNoRows`.
- `api/rotas.go:66` -- registrar a rota no mux raiz, junto aos Endereços.
- `api/sessao_test.go:88` -- `ambiente(t)`: Postgres + Redis por testcontainers, migrações aplicadas.
- `web/app/checkout/revisao/esboco-da-revisao.tsx` -- já lê o `id` guardado e o Endereço; acrescentar o bloco. `web/lib/preco.ts:formatarPreco`.
- `web/scripts/checkout.test.mjs` -- molde de `node --test` sobre `lib/`.

## Tasks & Acceptance

**Execution:**
- [x] `db/migracoes/20260919160000_pedido_faixa_frete.sql` -- tabela (`cep_inicio`, `cep_fim` 8 dígitos ou nulos, `regiao`, `valor_centavos bigint`, `padrao`) com `CHECK`s de forma/ordem/reais inteiros, índice único parcial em `padrao`, e o `INSERT` das nove linhas -- regra presente em todo banco, testcontainers inclusive.
- [x] `internal/identidade/db/consultas.sql` + `endereco.go` -- `BuscarEndereco(ctx, bd, enderecoID, compradorID)` com dono no WHERE; depois `sqlc generate` (restaurar `gerado/` não mexidos).
- [x] `internal/pedido/frete.go` (novo) + `consultas.sql` -- `FaixaDeFrete`, a pura `CalcularFrete(faixas, cep, subtotal, isencao) (Frete, error)` e `CotarFrete(ctx, bd, compradorID, enderecoID, isencao) (Cotacao, error)` -- a regra num lugar só.
- [x] `internal/pedido/frete_test.go` (novo) -- linhas da matriz que são regra, em memória, e determinismo (duas chamadas, mesmo resultado).
- [x] `api/frete.go` (novo) + `api/rotas.go` -- `GET /api/v1/frete` → `{regiao, subtotal_centavos, frete_centavos, total_centavos}`.
- [x] `api/frete_test.go` (novo) -- 401, 400, 404 alheio, Sudeste vs Nordeste para o mesmo Carrinho, e a padrão contra o banco real.
- [x] `web/lib/checkout.ts` -- tipo `Cotacao`; `web/app/checkout/revisao/esboco-da-revisao.tsx` -- bloco Subtotal / Frete (região; "Grátis" se 0) / Total, lido a cada abertura.
- [x] `_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/addendum.md` §10 + `uv run _bmad/scripts/memlog.py append` -- as três decisões (faixas na migração, reais inteiros, `≥`).
- [x] `_bmad-output/implementation-artifacts/deferred-work.md` -- a entrada "trocar o Endereço recalcula o Frete": metade provada aqui (rota), a tela fica com a 5.4.

**Acceptance Criteria:**
- Given dois Endereços, um em SP e um em BA, when o Comprador troca de um para o outro e volta à Revisão, then o Frete muda de R$ 15,00 para R$ 30,00 e o total acompanha, ao centavo.
- Given subtotal ≥ R$ 299,00, when a Revisão abre, then o Frete sai R$ 0,00 e a tela diz "Grátis".
- Given `go test ./internal/`, when roda, then `TestFronteiraDeModulo` continua verde.

## Design Notes

A região padrão é linha, não configuração: o AD-13 fecha a config nos 15 parâmetros do §7.1. Sem linha padrão é erro de montagem, e Frete 0 silencioso daria Frete grátis a todo CEP fora de faixa.

As faixas vão na migração, e não em `db/semente/`: a semente roda uma vez por marcador de versão, então bancos existentes nunca a veriam, e trocar `VersaoSemente` reexecuta os `INSERT` do Catálogo, que colidem.

Quem implementa não faz `git commit`, `git add` nem `push`.

## Verification

**Commands:**
- `go vet ./... && go test ./...` na imagem `azamon-dev:latest` (receita do HANDOFF) -- expected: verde.
- `docker build ./web` -- expected: verde (`verificar-offline` + `npm test`).
- `git diff --ignore-cr-at-eol --stat` -- expected: `gerado/` só de `identidade` e `pedido`.

**Manual checks (if no CLI):**
- A 360 e 1440 px: Revisão com Endereço de SP, trocar para BA, voltar: Frete e total mudam; Carrinho acima de R$ 299,00 mostra "Grátis".

## Review Triage Log

Três lentes (blind, edge-case, verification-gap) sobre o diff. Nenhum `intent_gap` nem `bad_spec`. Patches aplicados: guarda de forma do CEP em `CalcularFrete` (`ErrCEPInvalido`); `EXCLUDE` contra faixa sobreposta; 404 da cotação volta ao passo Endereço e corpo vazio vira erro; teste com subtotal entre o limiar do ambiente e o padrão (prova a ligação com a configuração); um CEP por linha da migração contra o banco real; nome de teste enganoso e comentário de `pedido.go`. Rejeitados: limiar ≤ 0 (a config já recusa no arranque), formato do envelope de erro (correto), Carrinho vazio (definido na matriz), snapshot entre leituras (leitura pura; a 5.6 recalcula sob a trava), CRLF (conhecido no HANDOFF).

## Suggested Review Order

**A Regra de Frete: dado mais função pura**

- Porta de entrada: a regra inteira, pura — forma do CEP, faixa, padrão obrigatória, isenção `≥`.
  [`frete.go:67`](../../internal/pedido/frete.go#L67)

- Sem padrão é erro mesmo com CEP na faixa: nunca Frete zero silencioso.
  [`frete.go:87`](../../internal/pedido/frete.go#L87)

- Isenção no mesmo ponto em que o Carrinho da 4.5 cala.
  [`frete.go:94`](../../internal/pedido/frete.go#L94)

**O dado: a migração**

- Reais inteiros por `CHECK`: o Provedor decide pelos centavos do total.
  [`20260919160000_pedido_faixa_frete.sql:25`](../../db/migracoes/20260919160000_pedido_faixa_frete.sql#L25)

- Faixas não se sobrepõem, garantido pelo banco; a padrão fica de fora.
  [`20260919160000_pedido_faixa_frete.sql:34`](../../db/migracoes/20260919160000_pedido_faixa_frete.sql#L34)

- As nove linhas nascem na migração, não na semente.
  [`20260919160000_pedido_faixa_frete.sql:42`](../../db/migracoes/20260919160000_pedido_faixa_frete.sql#L42)

**A cotação atravessando os módulos**

- Endereço do dono, subtotal do Carrinho e faixas; o total é somado aqui.
  [`frete.go:106`](../../internal/pedido/frete.go#L106)

- Leitura nova de um Endereço, com o dono no `WHERE`.
  [`endereco.go:64`](../../internal/identidade/endereco.go#L64)

- A rota: 400 sem `endereco_id`, 404 único para alheio/inexistente/malformado.
  [`frete.go:26`](../../api/frete.go#L26)

- Registrada no mux raiz, junto aos Endereços.
  [`rotas.go:72`](../../api/rotas.go#L72)

**A tela: esboço da Revisão**

- Pergunta ao Go a cada abertura; nada da cotação é guardado.
  [`esboco-da-revisao.tsx:59`](../../web/app/checkout/revisao/esboco-da-revisao.tsx#L59)

- Endereço sumido entre a lista e a cotação volta ao passo Endereço.
  [`esboco-da-revisao.tsx:67`](../../web/app/checkout/revisao/esboco-da-revisao.tsx#L67)

- "Grátis" quando o Go diz zero; a tela não soma.
  [`esboco-da-revisao.tsx:132`](../../web/app/checkout/revisao/esboco-da-revisao.tsx#L132)

**Testes**

- Matriz contra o banco real: troca de Endereço, padrão, 404 idêntico.
  [`frete_test.go:21`](../../api/frete_test.go#L21)

- Um CEP por linha da migração: a cópia do teste puro encontra o dado real.
  [`frete_test.go:64`](../../api/frete_test.go#L64)

- Subtotal entre R$ 250 e R$ 299: só isenta se a rota usa a config.
  [`frete_test.go:92`](../../api/frete_test.go#L92)

- Sobreposição recusada pelo banco (23P01).
  [`frete_test.go:132`](../../api/frete_test.go#L132)

- As fronteiras da regra pura, em memória.
  [`frete_test.go:28`](../../internal/pedido/frete_test.go#L28)

- Catorze tabelas: `faixa_frete` entra na lista do esqueleto.
  [`schema_test.go:57`](../../db/schema_test.go#L57)
