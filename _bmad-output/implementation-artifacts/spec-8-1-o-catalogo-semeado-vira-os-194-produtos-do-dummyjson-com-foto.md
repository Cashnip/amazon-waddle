---
title: '8.1 — O Catálogo Semeado vira os 194 Produtos do DummyJSON, com foto'
type: 'feature'
created: '2026-10-08'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
baseline_commit: 'acf3de7178894a6f42a761039f524be5d2edc0fb'
context:
  - '{project-root}/_bmad-output/implementation-artifacts/epic-8-context.md'
  - '{project-root}/_bmad-output/planning-artifacts/sprint-change-proposal-2026-10-08.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Os 50 placeholders SVG (nome sobre fundo colorido) deixam a Vitrine com cara de protótipo na frente da banca.

**Approach:** O Catálogo Semeado passa a ser os 194 Produtos do DummyJSON, traduzidos para PT-BR e sem marca, com a foto `images[0]` em WebP embutida no binário. O download é feito uma vez, no scratchpad. A lista continua escrita à mão em `media/gerar.go`, que gera o SQL da semente. Seguem a proposta `sprint-change-proposal-2026-10-08.md` (§3.2, §3.3 e §4.4) e as condições de aceite da Estória 8.1 em `epics.md`.

## Boundaries & Constraints

**Always:**
- **Composição:** 194 Produtos, 8 Categorias e 5 Vendedores, conforme a tabela §3.2 da proposta. O Vendedor de cada Produto é o da Categoria dele. "Sertão Livraria" vira "Sertão Empório". `Eletrônicos` e `Atlântico Importados` mantêm o nome e, portanto, o uuid.
- **Preço:** `reais = max(1, round(USD × 5,50))`, sempre em `,00`. Há duas exceções, e nelas só os centavos mudam (os reais seguem a regra): uma em `,90` e uma em `,95`.
- **Três papéis com Produto fixo:**

  | Papel | DummyJSON | Preço | Categoria |
  |---|---|---|---|
  | `produtoSemeado` (recusa) | id 104, carregador | R$ 110,90 | Eletrônicos / Atlântico, que `api/produto_test.go` exige |
  | `produtoParaEsgotar` (Roteiro A) | id 61, mixer de mão | R$ 192,00 | — |
  | `produtoAprovado` | id 52, wok de aço carbono | R$ 165,00 | — |
  | expira (Roteiro B, passo 5) | id 147, bola de futebol | R$ 99,95 | — |

- **Nome e descrição:** em PT-BR plausível, sem marca nem modelo comercial (sem "iPhone", "Rolex", "Apple"). Os nomes são únicos.
- **Busca sem acento:** pelo menos dois nomes têm acento e são achados por um termo sem acento.
- **Estoque:** o `DEFAULT 10` da coluna.
- **`VersaoSemente` fica igual.** O README avisa que um banco já semeado precisa de `docker compose down -v`.
- **Registro histórico:** as entradas antigas do addendum §10 ficam. A spec, o roteiro e o ensaio antigos não são tocados.

**Never:**
- Nenhum script de download, URL `dummyjson.com` ou `//cdn.` em arquivo versionado. O download e a cópia das imagens rodam no scratchpad.
- Não mudar o schema, as migrações, o PRD, a UX nem a `VersaoSemente`.
- Não usar `next/image` nem redimensionar ou converter as fotos.
- Não editar `roteiro-da-apresentacao.md`, `ensaio-dos-roteiros.md` nem `HANDOFF.md`, que são da 8.2.

## I/O & Edge-Case Matrix

| Cenário | Entrada / Estado | Saída esperada | Erro |
|---|---|---|---|
| Geração | lista válida e 194 WebP em `media/` | SQL com 194, 8 e 5 linhas | — |
| Nome repetido | dois Produtos com o mesmo nome | o gerador sai com código ≠ 0, nomeando os dois | nada é escrito |
| Apelido repetido | nomes diferentes com o mesmo `apelido()` | sai com código ≠ 0 | nada é escrito |
| Imagem ausente | `media/{apelido}.webp` não existe | sai com código ≠ 0, nomeando o arquivo | nada é escrito |
| Pagamento | Pedido do Produto de `,90` / `,95` / `,00` | recusa / expira / aprova | — |

</frozen-after-approval>

## Code Map

- `media/gerar.go`:
  - Hoje: lista, `svg()`, `quebrar()`, `xml()`, `cor` da Categoria.
  - Mantém: `id()` (uuid v5 do nome), `apelido()`, `literal()` e as contas.
  - A struct `produto` ganha `origem int`.
- `media/embutido.go` -- `//go:embed *.svg`.
- `api/media.go`:
  - linha 29: `image/svg+xml`;
  - linha 38: `Glob("*.svg")`.
- `api/media_test.go`:
  - linha 30: tipo;
  - linha 33: `<svg`;
  - linha 43: `nao-existe.svg` (pode ficar).
- **Testes que fixam o catálogo velho:**
  - `api/sessao_test.go:46` -- `produtoSemeado`.
  - `api/pedido_test.go` -- linha 32 (`produtoParaEsgotar`); 81–82, 113, 123 e 342 (24990 e "Fone de Ouvido Bluetooth Aurora|Atlântico Importados|24990|1").
  - `api/produto_test.go` -- linhas 29 e 34 (nome e 24990); 46 e 65 ficam válidas.
  - `api/webhook_test.go` -- linhas 22–27, 55, 65 e 95 (uuid, 32900 e os comentários de R$ 249,90).
  - `api/vendedor_test.go:52` -- "Sertão Livraria".
  - `api/produto_admin_test.go:55` -- `air-fryer-4-litros.svg`.
  - `db/schema_test.go` -- linha 368 (`!= 50`) e linha 390 (5 Categorias).
  - `db/medicao_nfr4_test.go` -- linhas 72–73 (5050) e 85–91 (casos por Categoria, com "Livros").
- **Comentários "50 Produtos":**
  - `cmd/azamon/main.go:91`;
  - `db/embutido.go:38`;
  - `internal/pagamento/pagamento_test.go:19` (o valor 24990 fica, e só o comentário muda).
- `db/semente-grande/001_catalogo_grande.sql` -- hoje 10 linhas × 10 séries. A soma de preço `l.ordem*1000 + s.ordem*100` preserva os centavos.
- `web/components/imagem-do-produto.tsx` -- só o comentário ("SVG embutido").
- **Documentação:**
  - `README.md` -- linhas 41, 137–139, 348, 356 e 400.
  - `ARCHITECTURE-SPINE.md:211` -- AD-10.
  - `addendum.md` -- §10 na linha 161.
  - Cada um tem o seu `.memlog.md` irmão.
- **Fonte:** o JSON já baixado está em `C:/Users/Ste/AppData/Local/Temp/claude/C--Users-Ste-Git-azamon/1182929c-55d8-48a0-b5f0-a2caa839601d/scratchpad/dj/products.json`. As imagens ainda não foram baixadas.

## Tasks & Acceptance

**Execution:**
- [x] **scratchpad (não versionado)** -- Copiar `products.json` e baixar o `images[0]` de cada Produto como `{id}.webp`. O nome de destino, `{apelido}.webp`, sai do gerador.
- [x] `media/gerar.go` -- Escrever a lista de 194 itens (`origem`, nome, descrição, centavos, Categoria, Vendedor) e as 8 Categorias. Tirar `svg`, `quebrar`, `xml` e `cor`. Validar nomes e apelidos únicos e o `.webp` presente antes de escrever qualquer coisa. URL `.webp`.
- [x] `media/*.svg` → `media/*.webp` -- Remover os 50 SVG e copiar as 194 fotos.
- [x] `media/embutido.go`, `api/media.go`, `api/media_test.go` -- Trocar para WebP. O teste confere `RIFF` nos bytes 0–3 e `WEBP` nos bytes 8–11.
- [x] `media/CREDITOS.md` -- Criar o arquivo: crédito ao DummyJSON (Ovi, MIT, com o aviso), dizer que a origem das fotos não é declarada e que o uso é acadêmico. Não afirmar que as fotos são livres.
- [x] `db/semente/001_catalogo_semeado.sql` -- Regerar com `go run media/gerar.go`.
- [x] `db/semente-grande/001_catalogo_grande.sql` -- Passar a 13 linhas ASCII × 2 séries e atualizar o comentário da conta (194 × 26 = 5.044).
- [x] **Testes do Code Map** -- Pôr os novos uuids, nomes e preços dos papéis, "Sertão Empório", 194 e 8, e o total de 5238. Nos casos do NFR-4, um termo ASCII por Categoria nova que case com pelo menos um nome dela. Manter o caso `"ca"`/Moda.
- [x] **Comentários** -- Atualizar `cmd/azamon/main.go`, `db/embutido.go`, `internal/pagamento/pagamento_test.go` e `web/components/imagem-do-produto.tsx`.
- [x] `README.md` -- Trocar os SVG pelas fotos e refazer a tabela das três faixas com os uuids e os totais novos (preço + R$ 15 de Frete Sudeste). Avisar do `docker compose down -v`.
- [x] `ARCHITECTURE-SPINE.md` + `.memlog.md` -- Emendar o AD-10 no lugar, com o texto da proposta §4.2.
- [x] `addendum.md` §10 + `.memlog.md` -- Escrever a entrada da 8.1 com as quatro decisões da proposta §4.3, sem apagar as entradas antigas.

**Acceptance Criteria:**
- Given o repositório, when `go run media/gerar.go` roda, then o SQL não muda (`git diff` vazio) e não há acesso à rede.
- Given o binário, when `GET /api/v1/media/{apelido}.webp`, then a resposta é 200, `image/webp`, e `GET /api/v1/admin/midias` lista 194 URLs `.webp`.
- Given os arquivos versionados, when o verificador do AD-12 (`web/scripts/verificar-offline.mjs`, no `prebuild`) roda, then passa, nenhum código nem arquivo de `media/` traz URL de CDN (citações da regra em documentos ficam) e `media/` não tem nenhum `.svg`.
- Given a Vitrine, a Página de Produto, a Faixa de Categorias e o seletor de imagem do Administrador, de 360 a 1440 px, then nenhuma foto quebra, não há rolagem horizontal e não entra verde nem laranja como enfeite.

## Implementation Notes

**Papéis (uuid lido do SQL gerado):**

| Papel | Produto | uuid | Preço |
|---|---|---|---|
| `produtoSemeado` (recusa) | Carregador de Celular com Cabo (origem 104) | `71ea1e58-91d9-5022-ba93-6a2518f42eed` | R$ 110,90 |
| `produtoParaEsgotar` | Mixer de Mão (origem 61) | `7446cd4e-bfa7-5bbe-8929-c987cc39a1d5` | R$ 192,00 |
| `produtoAprovado` | Wok de Aço Carbono (origem 52) | `9803e7d5-899c-5b1e-8db7-bbcf57fa90b1` | R$ 165,00 |
| expira (Roteiro B, passo 5) | Bola de Futebol (origem 147) | `0faf8e31-3b0c-5a47-8c00-34d462d42aa6` | R$ 99,95 |

`Eletrônicos` (`cc9fbd39-…`) e `Atlântico Importados` (`33cb4ea4-…`) mantiveram o uuid. Sertão Empório: `1ec181e5-e9df-5f43-a034-4b1dcb3d3be9`.

**Preço do `produtoSemeado` (decidido pelo humano, 2026-10-08).** A tabela de papéis dizia R$ 109,90, mas 19,99 × 5,50 = 109,945 arredonda para R$ 110. Vale a regra: R$ 110,90 (11090), com os testes de `api/`, o README, o addendum e o memlog do PRD acompanhando; o uuid não muda, porque o nome é o mesmo.

**Busca sem acento:** "Maçã Vermelha" (`maca`), "Limão Siciliano" (`limao`), "Água Mineral" (`agua`), "Ração para Gatos" (`racao`), entre outros.

**Cópia das fotos:** `gerar.exe` compilado no scratchpad, rodado em laço; cada falha "imagem ausente para a origem N: … media\{apelido}.webp" virou a cópia de `{N}.webp` para `media/{apelido}.webp`, até a geração passar. 194 arquivos, 12 MB.

**Matriz de erro (falha introduzida e revertida; o SQL conferido byte a byte depois de cada uma):**

```text
# {30, "Kiwi"} → {30, "Amora"}
gerar: nome "Amora" repetido nas origens 30 e 33
exit status 1   (SQL intocado)

# {30, "Kiwi"} → {30, "Amora!"}
gerar: apelido "amora" repetido nas origens 30 e 33
exit status 1   (SQL intocado)

# media/kiwi.webp movido para fora
gerar: imagem ausente para a origem 30: GetFileAttributesEx media\kiwi.webp: The system cannot find the file specified.
exit status 1   (SQL intocado)

# revertido
194 Produtos, 8 Categorias e 5 Vendedores em db/semente/001_catalogo_semeado.sql   (idêntico ao anterior)
```

**NFR-4:** 5.238 Produtos; termo + Categoria + faixa de preço, 160 execuções: mediana 0,17 ms, p95 0,26 ms, máximo 0,58 ms. Só termo: p95 1,21 ms.

**`git grep -n "//cdn\."`:** nenhum arquivo de código ou de `media/` casa. O padrão continua aparecendo em documentos que o citam como regra (AGENTS.md, README, espinha, `verificar-offline.mjs`), e a URL do CDN do DummyJSON já aparecia em `epic-8-context.md`, na proposta e na seção congelada desta spec, que esta estória não edita.

## Spec Change Log

## Review Triage Log

Passe 1 (Blind Hunter, Edge Case Hunter, Verification Gap):

| # | Lente | Achado | Veredito | Evidência / rota |
|---|---|---|---|---|
| 1 | VG, BH | Nenhum teste prende a regra de centavos (só ,90 e ,95) nem o Produto que expira | medium | `db/schema_test.go` não confere centavos; mudar a Bola para 9900 deixa tudo verde e o Roteiro B sem expiração → patch |
| 2 | VG | Apelido repetido não tem guarda fora do gerador `//go:build ignore` | low | Nada compara `imagem_url` entre linhas; uma linha na tabela de contagens resolve → patch |
| 3 | ECH, BH, VG | Addendum e contexto dizem que os Vendedores mantêm o uuid; o do Sertão mudou com o nome | low | `9eb5d89b-…` → `1ec181e5-…` no SQL; correção direta do texto → patch |
| 4 | BH | README atribui as medições de 2026-09-25 aos Produtos novos (Carregador, Bola) | medium | O parágrafo diz "medido em 2026-09-25 … no clone da tabela acima", e esses Produtos não existiam → patch (texto) |
| 5 | BH | Comentário de `pagamento_test.go` cita o Produto de R$ 110,90 sobre o valor 24990 | low | Correção direta do comentário → patch |
| 6 | BH | `medicao_nfr4_test.go` ainda diz "5.000 linhas" e afirma um termo por Categoria, e Moda só tem o caso de dois caracteres | low | Correção direta do comentário → patch |
| 7 | ECH, BH | `conferir()` não valida o Vendedor, `origem` repetida nem WebP órfã | low | O Vendedor inexistente cai na FK, que `db/schema_test.go` pega ao semear; hoje não há órfã nem repetida; a correção acrescenta guardas → rejeitado (low, fora do uso cotidiano) |
| 8 | ECH, BH | Volume anterior à 8.1 fica com imagens 404 e edição de Produto recusada | low | A intenção aceita isso explicitamente (`VersaoSemente` igual + `down -v` no README) → rejeitado |
| 9 | ECH | As `.webp` estão não rastreadas, e um commit sem elas quebra o `go:embed` | false | O commit do passo 5 inclui `media/*.webp`; não é defeito do código |
| 10 | ECH, VG | Os termos do NFR-4 não têm asserção de casar com algum Produto | low | Hoje os 8 casam (ECH e VG conferiram); a correção acrescenta asserção → rejeitado |
| 11 | BH | `<img>` sem `loading="lazy"`; "não há o que otimizar" | low | Demonstração local, mesma origem; o comentário fala de `next/image`, não do peso → rejeitado |
| 12 | BH | `/api/v1/media` sem `Cache-Control`/ETag | low | localhost, sem custo visível na demonstração; a correção acrescenta cabeçalhos → rejeitado |
| 13 | BH | A busca sem acento não tem teste | low | Comprovada pelos nomes (Maçã, Limão, Água, Ração) e ensaiada na 8.2; a correção acrescenta teste → rejeitado |
| 14 | BH | Só uma WebP tem o cabeçalho conferido no teste | low | As 194 foram conferidas no build; o gerador não escreve imagem → rejeitado |
| 15 | BH | A spec estreita a condição de aceite do `//cdn.` do epics.md | false | O estreitamento foi decidido pelo humano em 2026-10-08; o conserto seria editar a spec → rejeitado |
| 16 | BH | Higiene da spec (caminho local, comando repetido, sem manifesto) | low | O conserto edita a spec → rejeitado |
| 17 | BH | `sprint-status` em `in-progress` com a spec em `in-review` | false | A sincronização para `review` é do passo 5 |

## Design Notes

Os uuids vêm de `id("produto", nome)`. Trocar o nome troca o uuid, então os testes recebem os uuids depois da geração, lidos do SQL.

A cópia `{origem}.webp → media/{apelido}.webp` fica fora do repositório. Um caminho simples: rodar o gerador uma vez, ler do erro de imagem ausente (ou do SQL) a lista de apelidos e copiar. O gerador não ganha flag nem modo novo para isso.

`gerar.go` é `//go:build ignore`, então as três linhas de erro da Matriz não ganham `go test`: cada uma é verificada rodando o gerador com a falha introduzida de propósito e revertida em seguida. A saída de cada execução vai para as Implementation Notes.

## Verification

**Commands:**
- `go run media/gerar.go && git diff --exit-code db/semente` -- é idempotente.
- `go vet ./... && go test ./...` -- tudo verde, inclusive o `db` com o NFR-4 (5.238 e p95 ≤ 500 ms) e o `api`.
- `cd web && npm run build` (o `prebuild` roda o verificador do AD-12) ; `ls media/*.svg` -- o primeiro passa, o segundo sai vazio.
- `cd web && npm run build` -- passa.

**Manual checks:**
- `docker compose down -v && docker compose up --build`: abrir a Vitrine, uma Página de Produto e a Faixa de Categorias em 360 e 1440 px. As fotos aparecem e não há rolagem horizontal.
