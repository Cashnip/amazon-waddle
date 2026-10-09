# Epic 8 Context: Catálogo com fotos — pós-MVP

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Os placeholders SVG dos 50 Produtos (o nome sobre um fundo na cor da Categoria) deixam a Vitrine com cara de protótipo na frente da banca. A Épica 8 é pós-MVP: as Épicas 1 a 7 estão concluídas e o MVP não muda. Ela troca o Catálogo Semeado pelos 194 Produtos do DummyJSON, traduzidos para PT-BR, sem marca e com foto de loja embutida no binário. Depois, ensaia de novo os três roteiros de demonstração sobre esse catálogo. O catálogo é importado uma vez e versionado. Em tempo de execução, nada vem da rede. Uma tentativa anterior, com fotos CC do Openverse, ficou amadora e foi descartada.

## Stories

- Story 8.1: O Catálogo Semeado vira os 194 Produtos do DummyJSON, com foto
- Story 8.2: Os três roteiros ensaiados de novo sobre o catálogo novo

## Requirements & Constraints

- **Nenhum FR novo.** Vitrine, estoque e disponibilidade, busca sem acento e operação offline continuam valendo sobre o catálogo novo. O catálogo de demonstração pode ter de 50 a 200 Produtos, em pelo menos 5 Categorias.
- **Composição:** 194 Produtos, 8 Categorias em PT-BR (agrupando as 24 do DummyJSON) e os mesmos 5 Vendedores. Só "Sertão Livraria" muda de nome, para **"Sertão Empório"**. A Categoria `Eletrônicos` e o Vendedor `Atlântico Importados` mantêm o nome e o uuid, porque os testes dependem deles. A Categoria Livros sai.
- **Tradução e marcas.** Nome, descrição e Categoria em PT-BR. O nome do Produto não leva marca, e sim uma descrição ("Fone Over-ear Prata"), porque todos os dados de demonstração são fictícios. O id do DummyJSON fica no campo `origem`.
- **Estoque:** vale o `DEFAULT 10` da coluna. O `stock` do DummyJSON não é usado, porque tem zeros e deixaria Produtos indisponíveis ao acaso.
- **Preço é escolhido, não só convertido.** A regra é `reais = max(1, round(USD × 5,50))`, com centavos `,00`. As únicas exceções são um Produto em `,90` e um em `,95`, os dois abaixo de R$ 250. O motivo: o Provedor Simulado decide pelos centavos do total (`,00`–`,89` aprova, `,90`–`,94` recusa, `,95`–`,99` expira), e o Frete é em reais inteiros. Como 186 dos 194 preços do DummyJSON terminam em `.99`, a conversão mecânica faria quase tudo expirar.
- **Três papéis nos testes, fixados por uuid.** Cada papel ganha um Produto novo com as mesmas propriedades:
  - `produtoSemeado` recusa (`,90`) e custa abaixo de R$ 250, o limiar de isenção de Frete dos testes.
  - `produtoParaEsgotar` aprova (`,00`) e esgota sem estragar os outros subtestes.
  - `produtoAprovado` aprova (`,00`).
- **Busca sem acento demonstrável.** Pelo menos dois nomes com acento que um termo sem acento encontra.
- **Medição de desempenho da busca.** A semente grande deriva 194 × 13 linhas × 2 séries = **5.044** Produtos, num total de **5.238**. A medição confere esse total e exige p95 ≤ 500 ms. O conjunto grande serve para medir, não para demonstrar.
- **Tela:** de 360 a 1440 px sem rolagem horizontal. Nenhuma foto quebrada.
- **Portão do ensaio (SM-1, SM-C1).** A épica só fecha com os três roteiros passando sobre o catálogo novo, a partir de clone limpo, sem intervenção manual no banco e sem erro visível. O Roteiro A roda com a rede desconectada, imagens incluídas. A 8.2 é checkpoint humano: só fecha com o ensaio feito por uma pessoa.

## Technical Decisions

- **A lista mora em `media/gerar.go`, sem JSON novo**, e o gerador escreve `db/semente/001_catalogo_semeado.sql`.
  - O gerador não desenha imagem e não acessa a rede.
  - Ele falha com nome ou apelido de arquivo duplicado, ou com imagem ausente em `media/`. Existe um título repetido no DummyJSON (`Rolex Cellini Moonphase`).
- **O download do DummyJSON roda uma vez, no scratchpad, e nunca entra no repositório.** O grep do AD-12 barra `//cdn.` em qualquer arquivo versionado, e um script com a URL do CDN do DummyJSON cairia nele.
- **Imagens:**
  - Só `images[0]`: 1000 px, WebP como veio (cerca de 60 KB de mediana, uns 12 MB no total). O thumbnail de 300 px borra na Página de Produto.
  - Os 50 SVG saem.
  - O binário embute `*.webp`, e `GET /api/v1/media/{arquivo}` responde `image/webp`, com URL relativa no banco.
- **Licença.** `media/CREDITOS.md` credita o DummyJSON com o aviso MIT e diz que a origem das fotos não é declarada. O arquivo não afirma que as fotos são livres.
- **A `VersaoSemente` não muda.** A semente é `INSERT` sem `ON CONFLICT`, e quatro dos cinco Vendedores mantêm o uuid (o do Sertão mudou junto com o nome). Os quatro bastam: uma versão nova sobre um banco já semeado bate na chave primária e derruba o arranque. Quem já tem o banco roda `docker compose down -v`, e o README precisa dizer isso.
- **Schema e migrações não mudam.** O PRD e os documentos de UX também não.
- **Documentação viva.**
  - O AD-10 recebe a emenda de uma frase no lugar ("imagem embutida no Go (WebP, desde a 8.1)"). O AD-12 não muda.
  - O addendum §10 ganha a entrada da 8.1 com as quatro decisões: a regra de preço com o motivo, a retirada das marcas, a conta 13 × 2 e a `VersaoSemente` mantida.
  - Nada se apaga: as entradas da 1.3, da 1.4 e da D1 ficam.
  - Cada mudança na espinha ou no addendum vai para o `.memlog.md` do mesmo diretório, via `uv run _bmad/scripts/memlog.py append`. Os `AD-n` nunca são renumerados.
- **Registro histórico intocado:** as specs 1.3, 1.9, 7.1 e 7.4, o ensaio de 2026-09-28 em `ensaio-dos-roteiros.md` e as entradas antigas do addendum.

## UX & Interaction Patterns

- A imagem do Produto continua como `<img>` em `web/components/imagem-do-produto.tsx`, sem `next/image`. O fallback já existe, e só o comentário muda. A foto 1:1 precisa caber no quadro 4/3 sem rolagem horizontal.
- Confira na tela a Faixa de Categorias com 8 itens, a Vitrine, a Página de Produto e o seletor de imagem do Administrador.
- **Semântica fechada de cor (UX-DR7).** O verde só marca Estoque disponível e `ENTREGUE`. O laranja aparece uma vez por fluxo, no botão Confirmar Pedido. Nenhum dos dois entra como enfeite.

## Cross-Story Dependencies

- A 8.2 depende da 8.1 já em `main`.
- **A 8.1 atualiza o README:**
  - troca os 50 SVG pelas fotos;
  - refaz a tabela das três faixas com o uuid e o total dos Produtos novos;
  - avisa sobre o `down -v`.
- **A 8.2 registra o ensaio:**
  - atualiza `roteiro-da-apresentacao.md` com Produtos, preços e totais novos;
  - acrescenta uma seção datada no fim de `ensaio-dos-roteiros.md`;
  - aponta o catálogo novo em `HANDOFF.md`.
- O Roteiro B usa o Produto de `,90` para a recusa (passos 1 a 4) e o de `,95` para a expiração (passo 5). O Roteiro A usa o Produto que aprova e esgota.
- **Armadilhas de ensaio herdadas da Épica 7:**
  - `docker compose up` não reconstrói as imagens. Use `up --build` e confira com `docker compose images`.
  - O ensaio offline é desligar a rede da máquina à mão. Uma rede `internal: true` no compose descarta em silêncio a publicação das portas.
