---
title: Sprint Change Proposal — Catálogo Semeado importado com imagens reais
date: 2026-10-08
author: Sung (com bmad-correct-course)
status: aprovado (2026-10-08)
mode: batch
scope: moderada
---

# Sprint Change Proposal — Catálogo Semeado importado com imagens reais

## 1. Resumo do problema

**Gatilho.** Pós-MVP: as Épicas 1 a 7 estão `done`. Não houve estória que falhasse. A mudança vem da apresentação: os placeholders SVG dos 50 Produtos (nome sobre fundo na cor da Categoria, desenhados por `media/gerar.go`) deixam a Vitrine com cara de protótipo na frente da banca.

**Categoria.** Abordagem descartada que exige outra solução. Antes, a tentativa foi manter os 50 Produtos e trocar só a imagem por fotos CC do Openverse (Flickr/Wikimedia). O resultado ficou amador e inconsistente, e o caminho foi descartado. A alternativa é importar um catálogo pronto, que já tenha fotos no estilo de loja.

**Fonte.** DummyJSON (`dummyjson.com/products?limit=0`), baixado e medido em 2026-10-08:

| Medida | Valor |
|---|---|
| Produtos / Categorias | 194 / 24 |
| Preços terminados em `.99` | **186 de 194** |
| Títulos repetidos | 1 (`Rolex Cellini Moonphase`) |
| `stock` | 0 a 100 |
| `brand` vazio | 92 |
| `thumbnail` | 300×300 WebP, mediana 8 KB |
| `images[0]` | 1000×1000 WebP, mediana 60 KB, máximo 138 KB (amostra de 20) |
| Licença | código MIT (`github.com/Ovi/DummyJSON`). A origem das fotos não é declarada |

## 2. Análise de impacto

### 2.1 Checklist

| Item | Status | Achado |
|---|---|---|
| 1.1 Estória gatilho | [N/A] | Nenhuma: mudança pós-MVP, decidida pela apresentação |
| 1.2 Problema | [x] | Abordagem descartada (Openverse) → importar catálogo pronto |
| 1.3 Evidência | [x] | Medições da tabela acima. Revisão visual do Openverse descartada pelo humano |
| 2.1 Épica atual | [x] | A Épica 7 terminou e não é reaberta |
| 2.2 Mudança de épica | [!] | **Épica 8 nova**, pós-MVP, com duas estórias |
| 2.3 Épicas futuras | [N/A] | Nenhuma planejada |
| 2.4 Épica obsoleta | [N/A] | Nenhuma |
| 2.5 Ordem | [x] | A 8.2 depende da 8.1 |
| 3.1 PRD | [x] | Nada a mudar. O FR-11 já admite "entre 50 e 200 Produtos em pelo menos 5 Categorias". O §3 ("Produtos fictícios") e a linha 671 ficam respeitados com a retirada das marcas |
| 3.2 Arquitetura | [!] | Emenda de uma frase no AD-10 ("o arquivo é SVG"). O AD-12 não muda de regra |
| 3.3 UX | [x] | Nada a mudar. `imagem-do-produto.tsx` já tem o fallback. Verificar na tela a Faixa de Categorias com 8 itens e a foto 1:1 dentro do quadro 4/3 |
| 3.4 Outros artefatos | [!] | Testes, semente-grande, medição do NFR-4, README, roteiro da apresentação, addendum, HANDOFF, sprint-status |
| 4.1 Ajuste direto | Viável | Esforço médio, risco baixo |
| 4.2 Rollback | Inviável | Nada a reverter |
| 4.3 Revisão do MVP | Inviável | O MVP não muda |

### 2.2 O que a análise encontrou além do esperado

1. **Os centavos do preço controlam o Provedor Simulado.** `Simulado.Decidir` (`internal/pagamento/pagamento.go:77`) lê `total_centavos % 100`: de `,00` a `,89` aprova, de `,90` a `,94` recusa e de `,95` a `,99` expira. O Frete é em reais inteiros, então os centavos do total são os centavos do Produto. Com conversão mecânica, 186 de 194 Produtos expirariam. **O preço é escolhido, não só convertido.**
2. **Os testes fixam três papéis por uuid, e não por nome:**
   - `produtoSemeado` (`api/sessao_test.go:46`): hoje o Fone, R$ 249,90. Recusa (`,90`) e custa abaixo de R$ 250, o limiar de isenção de Frete dos testes.
   - `produtoParaEsgotar` (`api/pedido_test.go:32`): hoje a Caixa de Som, R$ 189,00. Aprova (`,00`).
   - `produtoAprovado` (`api/webhook_test.go:27`): hoje o Teclado, R$ 329,00. Aprova (`,00`).

   Cada papel ganha um Produto novo com as mesmas propriedades.
3. **Dois uuids sobrevivem se o nome ficar:** a Categoria `Eletrônicos` (`api/vendedor_test.go:20`, `api/produto_test.go:65`) e o Vendedor `Atlântico Importados` (`api/vendedor_test.go:35`). Os dois nomes ficam.
4. **A semente-grande deixa de dar 5.000.** Hoje são 50 × 10 linhas × 10 séries. Com 194, a mesma conta dá 19.400. `db/medicao_nfr4_test.go:72` exige `count(*) = 5050`.
5. **A `VersaoSemente` não pode mudar.** A semente é `INSERT` sem `ON CONFLICT`, e os Vendedores mantêm o uuid. Uma versão nova sobre um banco já semeado bate na chave primária e derruba o arranque (addendum §10, entrada da D1). A versão fica, e quem já tem o banco roda `docker compose down -v`.
6. **O grep do AD-12 barra `//cdn.`.** Um script de importação versionado, com `https://cdn.dummyjson.com`, cairia nele. O script roda uma vez, no scratchpad, e não entra no repositório.

### 2.3 O que não muda

PRD; UX (`DESIGN.md` e `EXPERIENCE.md`); `web/components/imagem-do-produto.tsx`, exceto o comentário; schema e migrações; `internal/catalogo/normalizar_test.go` ("O Silêncio das Marés" é exemplo de texto e não depende do catálogo); contas de demonstração; `VersaoSemente`.

Também ficam como estão, por serem registro histórico: specs 1.3, 1.9, 7.1 e 7.4; `ensaio-dos-roteiros.md`; as entradas do addendum da 1.3, da 1.4 e da D1.

## 3. Abordagem recomendada

**Ajuste direto: uma Épica 8 pós-MVP com duas estórias.** A 8.1 troca o catálogo; a 8.2 ensaia de novo os três roteiros a partir de clone limpo. Esforço médio, quase todo em curadoria e tradução de 194 itens. Risco baixo: o schema não muda, e os testes de papel acusam qualquer preço errado. Não afeta prazo do MVP, que já foi entregue.

### 3.1 Decisões tomadas

| # | Decisão | Escolha | Porquê |
|---|---|---|---|
| 1 | Escopo | **Os 194 Produtos** | Escolha do humano. O FR-11 admite até 200 |
| 1a | Categorias | **8 Categorias em PT-BR**, agrupando as 24 do DummyJSON (tabela 3.2) | 24 Categorias de 3 a 5 itens enchem a Faixa de Categorias. Mantém os 4 nomes atuais que existem lá; Livros sai |
| 2 | Idioma | **PT-BR**: nome, descrição e Categoria traduzidos | AGENTS: "português em tudo que sai". O §3 do PRD vale na tela |
| 3 | Preço | `reais = max(1, round(USD × 5,50))`, com centavos `,00`. As exceções são escolhidas à mão (3.3) | A conversão mecânica faria quase tudo expirar (2.2.1) |
| 4 | Marcas | **Tiradas do nome**: nome descritivo ("Fone Over-ear Prata") | PRD §3: "Produtos fictícios"; o schema não tem marca |
| 5 | Licença | Crédito em `media/CREDITOS.md`, com o aviso MIT e a ressalva de que a origem das fotos não é declarada | Uso acadêmico, não comercial. Sem afirmar que as fotos são livres. Não é parecer jurídico |
| 6 | Vendedores | **Os 5 continuam**, e só "Sertão Livraria" vira **"Sertão Empório"** | Não há livros. Um empório vende Mercado e Beleza |
| 7 | Estoque | **Regra atual**: `DEFAULT 10` da coluna | O `stock` do DummyJSON tem zeros e deixaria Produtos indisponíveis ao acaso |
| 8 | Imagem | **Só `images[0]`, 1000 px, WebP como vem.** Os 50 SVG saem | ~12 MB no binário e no repositório. O thumbnail de 300 px borra na Página de Produto |
| 8a | Onde mora a lista | **Continua em `media/gerar.go`**, sem JSON novo | A tradução é feita à mão de qualquer jeito. JSON só acrescentaria um parser |
| 9 | Artefatos | Seção 4 | — |
| 10 | Semente-grande | **13 linhas × 2 séries**: 194 × 26 = 5.044 derivados, total de 5.238 | A conta mais próxima dos 5.000 do NFR-4 que ainda distribui igual por todas as Categorias |

### 3.2 Categorias e Vendedores

| Categoria (PT-BR) | Categorias do DummyJSON | Produtos | Vendedor |
|---|---|---|---|
| Eletrônicos *(uuid mantido)* | smartphones, laptops, tablets, mobile-accessories | 38 | Atlântico Importados |
| Casa e Cozinha | kitchen-accessories, furniture, home-decoration | 40 | Casa Boa Utilidades |
| Moda | mens-shirts, mens-shoes, tops, womens-dresses, womens-shoes, womens-bags, sunglasses | 35 | Vale do Sol Distribuidora |
| Mercado | groceries | 27 | Sertão Empório |
| Esporte e Lazer | sports-accessories | 17 | Pampa Esportes |
| Relógios e Joias | mens-watches, womens-watches, womens-jewellery | 14 | Vale do Sol Distribuidora |
| Beleza e Perfumaria | beauty, fragrances, skin-care | 13 | Sertão Empório |
| Veículos | vehicle, motorcycle | 10 | Atlântico Importados |
| **Total** | | **194** | |

### 3.3 Os Produtos com papel

Todo Produto custa `,00`, exceto os marcados abaixo. Os quatro primeiros custam **abaixo de R$ 250**.

| Papel | Centavos | Usado por |
|---|---|---|
| Aprova, e esgota sem estragar os outros subtestes | `,00` | `produtoParaEsgotar`; Roteiro A (README, roteiro da apresentação) |
| Aprova | `,00` | `produtoAprovado` |
| Recusa | `,90` | `produtoSemeado`; Roteiro B, passos 1 a 4 |
| Expira | `,95` | Roteiro B, passo 5 |
| Busca sem acento | — | O ensaio da FR-12: pelo menos dois nomes com acento que um termo sem acento encontra, como em "mare" → "Maré" |

## 4. Propostas de mudança

### 4.1 Épicas: `_bmad-output/planning-artifacts/epics.md`

**Na Lista de Épicas, depois de "Épica 7" (antes de "Requisitos transversais"). ACRESCENTAR:**

```markdown
### Épica 8: Catálogo com fotos — pós-MVP

A Vitrine deixa a cara de protótipo: o Catálogo Semeado passa a ser os 194 Produtos do DummyJSON,
traduzidos, sem marca, com foto de loja embutida no binário. Os três roteiros são ensaiados de novo
sobre ele.

**FRs cobertos:** nenhum novo — FR-6, FR-11, FR-12 e NFR-15 continuam valendo sobre o catálogo novo
**Métricas:** SM-1, SM-4, SM-C3 · **AD:** AD-10, AD-12, AD-16 · **Passo:** pós-MVP

> Origem: `sprint-change-proposal-2026-10-08.md`. A SM-C1 continua sendo portão: a Épica 8 só fecha
> com os três roteiros passando sobre o catálogo novo.
```

**No fim do arquivo, depois da Estória 7.4. ACRESCENTAR:**

```markdown
## Épica 8: Catálogo com fotos — pós-MVP

A banca vê uma Vitrine com foto de loja, e não placeholder. O catálogo é importado uma vez, traduzido
e versionado; em tempo de execução nada vem da rede (AD-12).

**FRs:** nenhum novo · **Métricas:** SM-1, SM-4, SM-C3 · **AD:** AD-10, AD-12, AD-16

### Estória 8.1: O Catálogo Semeado vira os 194 Produtos do DummyJSON, com foto

**Como** Comprador na demonstração,
**quero** ver Produtos com foto de loja,
**para que** a Vitrine não pareça um protótipo.

**Condições de Aceite:**

**Dado** o DummyJSON baixado uma vez para fora do repositório,
**Quando** `go run media/gerar.go` roda,
**Então** `db/semente/001_catalogo_semeado.sql` tem **194 Produtos, 8 Categorias e 5 Vendedores**,
na tabela da §3.2 da proposta,
**E** a lista mora em `media/gerar.go`, com nome, descrição e Categoria em PT-BR, sem marca no nome, e
o id do DummyJSON de cada Produto num campo `origem`,
**E** o gerador falha se dois Produtos tiverem o mesmo nome ou o mesmo apelido de arquivo, ou se a
imagem de um Produto não existir em `media/`,
**E** o gerador não desenha imagem nem acessa a rede.

**Dado** a regra de preço,
**Quando** os preços são escritos,
**Então** `reais = max(1, round(USD × 5,50))` com centavos `,00`, exceto um Produto em `,90` e um em
`,95`, os dois abaixo de R$ 250,
**E** os três uuids de papel dos testes (`produtoSemeado`, `produtoParaEsgotar`, `produtoAprovado`)
apontam para Produtos com os centavos e a faixa de preço do papel (§3.3 da proposta).

**Dado** as imagens,
**Quando** o binário é construído,
**Então** `media/` tem uma WebP por Produto (`images[0]`, 1000 px, como veio) e nenhum SVG,
**E** `media/embutido.go` embute `*.webp`, e `GET /api/v1/media/{arquivo}` responde `image/webp`,
**E** `media/CREDITOS.md` credita o DummyJSON com o aviso MIT e diz que a origem das fotos não é
declarada,
**E** nenhum arquivo versionado contém `//cdn.` (AD-12).

**Dado** a medição do NFR-4,
**Quando** `AZAMON_SEMENTE_GRANDE=true`,
**Então** `db/semente-grande` deriva 194 × 13 linhas × 2 séries = **5.044** Produtos, total de 5.238, e
`db/medicao_nfr4_test.go` confere esse total e o p95 ≤ 500 ms.

**Dado** a suíte,
**Quando** `go test ./...` roda,
**Então** passa, inclusive `api/media_test.go` (RIFF/WEBP no lugar de `<svg`) e
`api/vendedor_test.go` (Sertão Empório).

**Dado** a tela,
**Quando** a Vitrine, a Página de Produto, a Faixa de Categorias com 8 itens e o seletor de imagem do
Administrador são abertos de 360 a 1440 px,
**Então** nenhuma foto quebra, a foto 1:1 cabe no quadro 4/3 sem rolagem horizontal (NFR-10), e
nenhum verde nem laranja entra como enfeite (UX-DR7).

**Dado** a documentação,
**Quando** a estória fecha,
**Então** o README troca os 50 SVG pelas fotos, refaz a tabela das três faixas com os Produtos novos
(uuid e total) e diz que um banco já semeado precisa de `docker compose down -v`,
**E** o addendum §10 ganha a entrada da 8.1 com memlog, sem apagar as entradas da 1.3, da 1.4 e da D1,
**E** a frase do AD-10 sobre SVG é emendada no lugar, com memlog da arquitetura.

### Estória 8.2: Os três roteiros ensaiados de novo sobre o catálogo novo

**Como** time que vai apresentar,
**quero** os três roteiros ensaiados de novo a partir de clone limpo com o catálogo novo,
**para que** nada seja demonstrado ao vivo pela primeira vez (SM-1).

**Condições de Aceite:**

**Dado** a 8.1 em `main`,
**Quando** os três roteiros do §12 são executados a partir de clone limpo, com o Roteiro A de rede
desconectada (NFR-15),
**Então** os três passam de ponta a ponta, sem intervenção manual no banco e sem erro visível,
**E** o Roteiro B mostra a recusa com o Produto de `,90` e a expiração com o de `,95`,
**E** a busca sem acento mostra um Produto com acento no nome.

**Dado** o registro,
**Quando** o ensaio fecha,
**Então** `roteiro-da-apresentacao.md` traz os Produtos, preços e totais novos,
**E** `ensaio-dos-roteiros.md` ganha uma seção datada no fim, e o ensaio de 2026-09-28 fica intacto,
**E** `HANDOFF.md` aponta o catálogo novo.

**Dado** o checkpoint humano,
**Quando** a estória é revisada,
**Então** ela só fecha com o ensaio feito por uma pessoa.
```

**Justificativa.** A Épica 7 terminou, e a mudança é pós-MVP com origem própria. Reabrir a 7.4 reescreveria um registro histórico.

### 4.2 Arquitetura: `ARCHITECTURE-SPINE.md`, AD-10 (linha 211). Emenda no lugar, na 8.1

**ANTES:**
> … porque o arquivo é SVG embutido no Go e servido na mesma origem, e não há o que otimizar.

**DEPOIS:**
> … porque o arquivo é uma imagem embutida no Go (WebP, desde a 8.1) e servida na mesma origem, e não há o que otimizar.

Com memlog da arquitetura (`uv run _bmad/scripts/memlog.py append`). O AD-12 fica igual: a regra já diz "arquivos de `media/`, servidos pelo Go".

### 4.3 PRD e addendum

- `prd.md`: **nenhuma mudança**.
- `addendum.md` §10: **entrada nova da 8.1**, escrita na própria estória, com memlog do PRD. Ela registra quatro decisões: a regra de preço e o motivo dela (o Simulado lê os centavos), a retirada das marcas, a conta 13 × 2 da semente-grande e a `VersaoSemente` mantida.

### 4.4 Código (8.1)

| Arquivo | Mudança |
|---|---|
| `media/gerar.go` | Lista nova (194 itens) com campo `origem` no Produto; `cor` sai da Categoria; `svg()` e `quebrar()` saem; `.webp` no lugar de `.svg`; checagens de nome e apelido duplicados e de imagem ausente |
| `media/*.svg` | Removidos; entram 194 `*.webp` |
| `media/embutido.go` | `//go:embed *.webp` e comentário |
| `media/CREDITOS.md` | Novo |
| `api/media.go` | `image/webp`, `Glob("*.webp")` |
| `api/media_test.go` | Corpo começa com `RIFF` e tem `WEBP` no byte 8; tipo `image/webp` |
| `api/produto_admin_test.go` | Nome de arquivo existente no lugar de `air-fryer-4-litros.svg` |
| `api/sessao_test.go`, `api/pedido_test.go`, `api/webhook_test.go`, `api/produto_test.go` | uuids e asserções dos três papéis |
| `api/vendedor_test.go` | "Sertão Empório" |
| `db/semente/001_catalogo_semeado.sql` | Regerado |
| `db/semente-grande/001_catalogo_grande.sql` | 13 linhas ASCII × 2 séries, comentário da conta |
| `db/medicao_nfr4_test.go` | Total de 5.238 |
| `web/components/imagem-do-produto.tsx` | Só o comentário ("SVG embutido" → "imagem embutida") |
| `README.md` | Linhas 41, 137–139, 348, 356 e 400 |

### 4.5 Rastreamento

`sprint-status.yaml`, depois de `epic-7-retrospective`:

```yaml
  epic-8: backlog
  8-1-o-catálogo-semeado-vira-os-194-produtos-do-dummyjson-com-foto: backlog
  8-2-os-três-roteiros-ensaiados-de-novo-sobre-o-catálogo-novo: backlog
  epic-8-retrospective: optional
```

## 5. Handoff

**Escopo: moderado.** É uma épica nova no backlog, sem replanejamento de produto nem de arquitetura.

| Quem | O quê |
|---|---|
| Developer (`bmad-build`) | 8.1 inteira: download no scratchpad, curadoria, tradução, código, testes e documentos |
| Sung (humano) | Revisa na 8.1 os nomes traduzidos e a curadoria. Faz o ensaio da 8.2, que é checkpoint humano |

**Critérios de sucesso:**
- `go test ./...` verde.
- `docker compose down -v && docker compose up`: 194 Produtos com foto e nenhum placeholder visível.
- Grep do AD-12 limpo.
- Os três roteiros passam sobre o catálogo novo, com o Roteiro A offline.

**Ordem:** a 8.1, depois a 8.2. Pela memória do projeto, cada estória faz commit e push direto na `main`.
