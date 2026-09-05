---
name: Azamon
description: Réplica do núcleo de comércio eletrônico da Amazon, construída como trabalho acadêmico. shadcn/ui sobre Tailwind — este DESIGN.md especifica apenas a camada de marca.
status: final
updated: 2026-09-04
sources:
  - ../../prds/prd-azamon-2026-08-14/prd.md
  - ../../prds/prd-azamon-2026-08-14/addendum.md
colors:
  # Camada de marca sobre os defaults do shadcn. Todo token não listado aqui
  # — background, foreground, muted, muted-foreground, card, card-foreground,
  # popover, border, input, ring, secondary, destructive — herda do shadcn
  # sem alteração. Se a marca não justifica sobrescrever, não sobrescreve.
  chrome: '#131921'
  chrome-foreground: '#FFFFFF'
  chrome-muted: '#232F3E'
  primary: '#FFD814'
  primary-foreground: '#0F1111'
  primary-strong: '#FFA41C'
  primary-strong-foreground: '#0F1111'
  link: '#007185'
  available: '#007600'
  available-foreground: '#FFFFFF'
typography:
  # Corpo, label e muted herdam a rampa do shadcn. A família é declarada por
  # arquivo auto-hospedado, não por pacote de framework — ver a seção
  # Typography. Apenas os três papéis abaixo são da camada de marca.
  wordmark:
    fontFamily: 'Azamon Sans'
    fontSize: 22px
    fontWeight: '700'
    letterSpacing: -0.02em
  preco:
    fontFamily: 'Azamon Sans'
    fontSize: 28px
    fontWeight: '500'
    lineHeight: '1'
  preco-centavos:
    fontFamily: 'Azamon Sans'
    fontSize: 14px
    fontWeight: '500'
    lineHeight: '1'
rounded:
  # sm / md / lg reproduzem os defaults do shadcn — declarados aqui, e não
  # só herdados em prosa, para que toda referência {rounded.*} resolva e o
  # código a jusante espelhe a espinha. O pill é a única marca própria.
  sm: 4px
  md: 6px
  lg: 8px
  full: 9999px
spacing:
  # Escala do Tailwind herdada. Estes tokens nomeados existem só para
  # sustentar o ritmo folgado — são o oposto do aperto da Amazon.
  section: 48px
  section-sm: 32px
  grid-gutter: 24px
  page-margin: 16px
  page-margin-lg: 32px
components:
  barra-superior:
    background: '{colors.chrome}'
    foreground: '{colors.chrome-foreground}'
  faixa-categorias:
    background: '{colors.chrome-muted}'
    foreground: '{colors.chrome-foreground}'
  busca-global:
    background: '#FFFFFF'
    radius: '{rounded.full}'
  botao-adicionar-carrinho:
    background: '{colors.primary}'
    foreground: '{colors.primary-foreground}'
    radius: '{rounded.full}'
  botao-confirmar-pedido:
    background: '{colors.primary-strong}'
    foreground: '{colors.primary-strong-foreground}'
    radius: '{rounded.full}'
  cartao-produto:
    radius: '{rounded.lg}'
    imageAspectRatio: '4 / 3'
  caixa-de-compra:
    radius: '{rounded.lg}'
    padding: '{spacing.section-sm}'
  menu-da-conta:
    background: '{colors.chrome}'
    foreground: '{colors.chrome-foreground}'
    radius: '{rounded.md}'
  nav-administrativa:
    background: '{colors.chrome-muted}'
    foreground: '{colors.chrome-foreground}'
  breadcrumb:
    foreground: '{colors.link}'
  passos-checkout:
    radius: '{rounded.md}'
  selo-status-entregue:
    background: '{colors.available}'
    foreground: '{colors.available-foreground}'
    radius: '{rounded.full}'
  linha-do-tempo-pedido:
    radius: '{rounded.md}'
  selo-disponibilidade:
    foreground: '{colors.available}'
  selo-status-pedido:
    radius: '{rounded.full}'
---

## Brand & Style

O Azamon é uma réplica **assumida** do núcleo de comércio eletrônico da Amazon. A postura estética decorre disso: a tela precisa ser reconhecida em dois segundos por qualquer pessoa da banca, sem que ninguém precise explicar de que produto é a cópia. Reconhecimento é o requisito, não originalidade — e é por isso que a disciplina anti-genérico dos frameworks de "taste" foi avaliada e deixada de fora.

Mas a réplica para em três marcas. A assinatura da Amazon vive na **cromática do chrome**, na **busca dominante** e nos **padrões de preço e compra**. Tudo o mais respira. A Amazon é densa porque cobre livros, pranchas de surf e medicamentos no mesmo layout; o Catálogo Semeado não tem essa largura, então copiar o aperto dela seria teatro — densidade encenando um catálogo que não existe.

O Azamon herda o shadcn/ui por inteiro. Este documento especifica só a camada de marca: quatro famílias de cor, três papéis tipográficos, o pill dos botões de ação e um ritmo de espaço mais generoso. Os componentes que vêm prontos do shadcn usam o visual do shadcn como está — a lista fechada está em Components. Customizar esses componentes é violação de disciplina, não gosto pessoal.

## Colors

- **Chrome `#131921`** é a barra superior e o rodapé. É o quase-preto azulado da Amazon e é a única cor que a pessoa registra antes de ler qualquer texto. Nunca aparece em conteúdo, nunca em card, nunca como fundo de seção.
- **Chrome Muted `#232F3E`** tem dois usos, e só esses dois: a faixa secundária de Categorias logo abaixo da barra superior, e a barra da **navegação administrativa**. Nos dois casos o papel é o mesmo — um segundo nível de chrome. Na área administrativa ele é o único chrome, e a diferença de tom em relação a `{colors.chrome}` é o sinal de que se está fora da loja.
- **Ação `#FFD814`** é o amarelo de **Adicionar ao Carrinho**. Sobre `{colors.primary-foreground}` (`#0F1111`) o contraste passa AA com folga larga. Substitui o `primary` do shadcn.
- **Ação Forte `#FFA41C`** é o laranja e aparece **uma vez por fluxo**: no botão que confirma o Pedido no checkout. É o passo irreversível — o único momento em que o Comprador cria compromisso de Estoque. Usar laranja em qualquer outro lugar apaga esse significado.
- **Link `#007185`** é o azul-petróleo de link de texto. Marca legítima da Amazon, 4,6:1 sobre branco, passa AA. Cobre nome de Produto no resultado de busca, breadcrumb e navegação textual.
- **Disponível `#007600`** carrega **dois sentidos, e só esses dois**: *Estoque disponível* (o selo "Em estoque") e *estado terminal bem-sucedido do Pedido* (o selo `ENTREGUE`). Nunca sucesso genérico de operação, nunca confirmação de formulário, nunca `Toast`. Como texto sobre fundo claro tem 5,1:1; como preenchimento, com `{colors.available-foreground}` (branco) por cima, tem os mesmos 5,1:1 — os dois passam AA. A distinção que importa: verde marca **uma coisa que chegou ao fim bem**, não **uma ação que deu certo**.
- **Todo o resto** herda do shadcn, inclusive `destructive`. O Azamon não tem cor de erro própria.

**Sem modo escuro.** A referência não tem, o MVP roda em demonstração local controlada, e manter dois temas dobra a superfície de verificação de contraste sem ganhar nada na banca. `[ASSUMPTION: modo escuro fora do MVP; nenhum requisito do PRD o menciona.]`

## Typography

Corpo, label, caption e heading herdam a **rampa** do shadcn (tamanhos, pesos e entrelinhas). A **família** é `Azamon Sans`: um único arquivo WOFF2 auto-hospedado em `/fonts`, apelidado no CSS, com a pilha de sistema como alternativa declarada.

Isso é deliberado. O default do shadcn na prática é o Geist, que é a fonte do template do **Next.js** — não do shadcn. Amarrar o token tipográfico ao Geist fecharia de lado a **Questão em Aberto 1 do §15** (Next.js vs React), que está marcada como bloqueante e pertence ao time. Com o arquivo auto-hospedado, a espinha funciona nos dois caminhos e o NFR-15 é atendido de qualquer jeito. `[ASSUMPTION: o arquivo concreto ainda não foi escolhido. Qualquer sans neutra de leitura densa serve; se o time escolher Next.js, o Geist é a escolha óbvia e o apelido continua valendo.]`

Três papéis são da camada de marca:

- **`wordmark`** — a palavra `azamon` na barra superior, 22px/700, tracking apertado. Sem símbolo, sem SVG para alguém desenhar. É o que a Amazon fazia antes do sorriso.
- **`preco`** e **`preco-centavos`** — o preço inteiro em 28px e os centavos em 14px sobrescritos, alinhados ao topo da linha. É o padrão de comércio mais reconhecível da referência e custa uma regra de CSS.

Todo número monetário e toda quantidade usam **`font-variant-numeric: tabular-nums`**. Sem isso as colunas de preço na lista de Pedidos dançam ao paginar, e o NFR-13 exige que o total exibido seja exatamente a soma visível das parcelas — o que só se lê como verdade quando os dígitos alinham.

**A fonte é auto-hospedada no repositório.** O NFR-15 exige que o Roteiro A rode com a rede desconectada: nenhuma requisição a `fonts.googleapis.com`, nenhum `@import` remoto, nenhum CDN. Se a fonte auto-hospedada virar problema de peso, a saída é a pilha de sistema (`system-ui, -apple-system, Segoe UI, Roboto, Arial`) — nunca uma fonte remota.

**Sem cinza pequeno.** A Amazon usa cinza claro em corpo de 12px em vários pontos e isso reprova em AA. É um desvio consciente da réplica: no Azamon, todo texto de conteúdo fica em `foreground` ou `muted-foreground` do shadcn em no mínimo 14px, com 4,5:1 verificado.

## Layout & Spacing

Escala do Tailwind herdada. Os tokens nomeados sustentam o respiro que separa o Azamon da referência:

- **`{spacing.section}` (48px)** entre blocos de conteúdo de 1024px para cima; **`{spacing.section-sm}` (32px)** de 360px a 1023px.
- **`{spacing.grid-gutter}` (24px)** entre cartões de Produto. A Amazon usa perto da metade disso.
- **`{spacing.page-margin}` (16px)** de margem lateral de 360px a 1023px; **`{spacing.page-margin-lg}` (32px)** de 1024px para cima.
- Largura máxima de conteúdo: **1440px**, o teto do NFR-10. A grade de Produtos vai de 1 coluna em 360px a 4 colunas em 1440px.

Os cortes são declarados em **pixels**, nas mesmas quatro faixas da tabela de `EXPERIENCE.md.Responsive & Platform`. Os prefixos `sm` / `md` / `lg` não aparecem em prosa neste documento: `sm` significa "faixa pequena" em linguagem de design e "≥640px" em Tailwind, e as duas leituras produzem código diferente.

A busca ocupa a largura disponível da barra superior — é o único elemento que não respeita o ritmo folgado, porque o peso visual dela **é** a assinatura. O §9 do PRD já a define como navegação principal do Comprador; aqui ela ganha o tamanho que a decisão merece.

Nenhuma tela do fluxo de compra rola horizontalmente entre 360px e 1440px (NFR-10). Tabela larga — a lista de Pedidos do Administrador — rola dentro do próprio contêiner, nunca arrastando a página.

## Elevation & Depth

Herdado do shadcn e nada mais. Sombra sutil em `Card` e em estado de foco; elevação não é dispositivo de hierarquia no Azamon. A caixa de compra da página de Produto se separa por **borda e espaço**, não por sombra pesada — a referência faz igual.

## Shapes

`sm` 4px, `md` 6px, `lg` 8px do shadcn, herdados sem alteração. O `{rounded.full}` é a única declaração própria e cobre exatamente quatro usos: o campo da **busca global**, os **botões de ação** (Adicionar ao Carrinho e Confirmar Pedido, ambos em pill, como na referência), os **selos de Status do Pedido** e os chips de filtro ativo. Botão que não é de ação primária usa o raio do shadcn.

Cantos retos foram rejeitados junto com a densidade utilitária. O Azamon não quer parecer ferramenta industrial.

## Components

Usados do shadcn **sem alteração**: `Card`, `Dialog`, `Sheet`, `Select`, `Table`, `Badge`, `Toast`, `Skeleton`, `Pagination`, `Input`, `Label`, `Separator`, `DropdownMenu`, `RadioGroup`, `Alert`.

Duas composições nomeadas na espinha de experiência não têm delta visual: são o componente do shadcn direto. **Tabela de Pedidos** é `Table`; **Paginação** é `Pagination`.

Sobrescritos e composições da camada de marca:

- **Barra superior** — `{colors.chrome}` de fundo, `{colors.chrome-foreground}` de texto. Contém, da esquerda para a direita: o wordmark, a busca global ocupando o espaço restante, o **Menu da conta** e o Carrinho com contador. Presente em **toda** tela pública e de Comprador. Ausente na área administrativa, que é separada da loja (§9).
- **Faixa de Categorias** — `{colors.chrome-muted}`, logo abaixo da barra superior. Lista plana de Categorias, sem hierarquia, porque o glossário define **Categoria** como agrupamento plano.
- **Busca global** — campo branco em `{rounded.full}` com seletor de Categoria acoplado à esquerda e botão de lupa à direita. Teto de 100 caracteres no termo (§7.1).
- **Botão Adicionar ao Carrinho** — `{colors.primary}` sobre `{colors.primary-foreground}`, `{rounded.full}`. Aparece no cartão de Produto e na caixa de compra.
- **Botão Confirmar Pedido** — `{colors.primary-strong}` sobre `{colors.primary-strong-foreground}`, `{rounded.full}`. Um por fluxo, no último passo do checkout.
- **Caixa de compra** — coluna à direita na página de Produto de 1024px para cima, empilhada abaixo da descrição de 360px a 1023px. Contém preço, selo de disponibilidade, seletor de quantidade (teto de 10 unidades, §7.1) e o botão de ação. Separada por borda de 1px e `{spacing.section-sm}` de respiro interno.
- **Selo de disponibilidade** — texto em `{colors.available}`, sem fundo, sem ícone. "Em estoque" ou "Indisponível".
- **Cartão de Produto** — `Card` do shadcn em `{rounded.lg}`, com imagem em proporção fixa no topo, nome do Produto em `{colors.link}`, preço nos papéis `preco` / `preco-centavos`, selo de disponibilidade e botão de ação. Espaçado por `{spacing.grid-gutter}` na grade.
- **Barra de filtros** — composição de `Badge` e `Select` do shadcn. Chips de filtro ativo em `{rounded.full}`.
- **Linha de Item de Carrinho** — composição de `Input` e `Button` do shadcn, sem delta de marca. Separada por `Separator`, nunca por `Card` aninhado.
- **Menu da conta** — `DropdownMenu` do shadcn, acionado por rótulo de duas linhas na barra superior ("Olá, {nome}" sobre "Minha conta"). Herda `{colors.chrome}` enquanto fechado; o painel aberto usa `popover` do shadcn.
- **Navegação administrativa** — barra superior própria da área administrativa em `{colors.chrome-muted}`, sem busca e sem Carrinho. Contém o wordmark seguido de "Administração", os quatro destinos e a identificação de quem está na Sessão, com a saída. Nunca usa `{colors.chrome}`.
- **Breadcrumb** — trilha de Categoria em `{colors.link}`, acima do título na Página de Produto e nos Resultados. É o único caminho de volta da Página de Produto para uma listagem.
- **Passos do checkout** — **dois** passos nomeados em `{rounded.md}` (Endereço → Revisão). O passo atual usa `foreground` do shadcn; os concluídos, `muted-foreground`; os futuros, `muted-foreground` com 60% de opacidade. Sem ícone de check, sem barra de progresso preenchida.
- **Linha do tempo do Pedido** — lista vertical em `{rounded.md}`, uma linha por transição registrada, com marcador de 8px em `border` do shadcn ligado por filete de 1px. `ENTREGUE` e `CANCELADO` fecham a linha sem filete adiante.
- **Selo de Status do Pedido** — `Badge` do shadcn em `{rounded.full}`, um para cada um dos sete estados. Cor: os quatro estados de progresso normal usam `secondary` do shadcn; `PAGAMENTO_RECUSADO` e `CANCELADO` usam `destructive`; `ENTREGUE` usa `selo-status-entregue`, preenchimento `{colors.available}` sobre `{colors.available-foreground}` a 5,1:1. O selo nunca carrega ícone, e o texto é a **forma de exibição** declarada em `EXPERIENCE.md.A máquina de estados na interface` — não o identificador cru.

→ Vistos aplicados em [`mockups/pagina-de-produto.html`](mockups/pagina-de-produto.html) (barra superior, menu da conta, busca global, breadcrumb, caixa de compra, botão de ação, selo de disponibilidade), [`mockups/resultados-busca.html`](mockups/resultados-busca.html) (cartão de Produto, barra de filtros, paginação) e [`mockups/detalhe-do-pedido.html`](mockups/detalhe-do-pedido.html) (selo de Status do Pedido, linha do tempo). Os mocks ilustram; **as duas espinhas vencem em caso de conflito**.

## Do's and Don'ts

| Faça | Não faça |
|---|---|
| Herdar o shadcn em tudo que não está na camada de marca | Sobrescrever token do shadcn além dos **dez** declarados aqui |
| Usar `{colors.primary-strong}` uma vez por fluxo, no passo irreversível | Espalhar laranja para "dar destaque" |
| Usar `{colors.available}` para Estoque disponível e para `ENTREGUE` | Usar verde para sucesso de operação, confirmação de formulário ou `Toast` |
| Declarar faixa responsiva em pixels | Escrever `sm` / `md` / `lg` em prosa — significam duas coisas |
| Auto-hospedar a fonte no repositório | Fazer qualquer requisição de rede em tempo de execução (NFR-15) |
| Usar `tabular-nums` em todo valor monetário e quantidade | Usar fonte proporcional em coluna de preço |
| Manter texto de conteúdo em 14px+ com 4,5:1 verificado | Copiar o cinza pequeno da Amazon |
| Dar à busca o peso visual da referência | Apertar o resto do layout junto |
| Escrever o nome do estado como o glossário define | Traduzir, abreviar ou iconizar `SEPARANDO`, `ENVIADO`, `ENTREGUE` |
| Espaçar a grade de Produtos com `{spacing.grid-gutter}` | Apertar a grade "porque a Amazon é assim" |
