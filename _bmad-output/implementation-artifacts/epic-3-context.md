# Epic 3 Context: Catálogo, Estoque e Busca

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Esta épica põe a loja de pé. O Administrador abastece o Catálogo com Vendedores, Categorias, Produtos e Estoque numa área administrativa separada; qualquer visitante vê a Vitrine, encontra Produtos por termo, filtro e ordenação, e abre a Página de Produto com o nome do Vendedor e a disponibilidade real. Produto criado aparece na Vitrine e na busca sem reiniciar a aplicação. Catálogo e busca viraram uma épica só porque a Vitrine e os Resultados de busca são o mesmo endpoint: termo vazio devolve a listagem paginada. Aqui também nascem as peças de Estoque que Carrinho, checkout e Pedidos só vão consumir: o disponível derivado, a Reserva de Estoque com estado e o predicado único de "Produto visível". Três dos cinco itens da ordem de corte moram nesta épica.

## Stories

- Story 3.1: Gestão de Vendedores
- Story 3.2: Gestão de Categorias
- Story 3.3: Gestão de Produtos
- Story 3.4: Estoque disponível, guarda de Reservas e o predicado de visibilidade
- Story 3.5: A Vitrine e o envelope de listagem
- Story 3.6: Página de Produto
- Story 3.7: Busca por texto
- Story 3.8: Filtros de Categoria e faixa de preço
- Story 3.9: Ordenação
- Story 3.10: Chrome da loja: barra superior, busca global e Faixa de Categorias

## Requirements & Constraints

- **Vendedores:** criar, editar e desativar. Desativar oculta os Produtos dele na Vitrine, na busca e na Página de Produto, sem apagar nem alterar Pedido. Vendedor com Produtos não é removível, e a recusa oferece desativar no lugar.
- **Categorias:** criar e renomear. Nome duplicado é recusado. Categoria com Produtos não é removível, e a recusa diz quantos Produtos estão vinculados. A Categoria é plana: `categoria_pai_id` existe, nulo e não exposto.
- **Produtos:** sempre com Vendedor e Categoria. Preço ≤ 0 e Estoque negativo são recusados. Editar o preço não altera Item de Pedido existente. Nome com no máximo 200 caracteres e descrição com no máximo 4.000, verificados no servidor. O Produto aparece na Vitrine e na busca sem reiniciar, então não há cache de listagem.
- **Estoque:** disponível = total − Σ Reservas ativas. O ajuste do Administrador vale na hora e o disponível nunca fica negativo. Baixar o total abaixo das Reservas ativas é recusado com a contagem em tempo real: "Há N unidades comprometidas em Pedidos abertos. O Estoque total não pode ficar abaixo disso."
- **Página de Produto:** id inexistente, Produto desativado e Produto de Vendedor desativado devolvem "não encontrado", nunca erro de servidor, e a interface não distingue os três casos.
- **Busca:** termo livre sobre nome e descrição, sem diferenciar caixa nem acento ("cafe" encontra "café"). `%` e `_` são texto literal. Teto de 100 caracteres. Sem relevância por pontuação, sem correção ortográfica e sem sugestão enquanto digita. Produto e Vendedor desativados não aparecem.
- **Filtros:** Categoria e faixa de preço, combináveis entre si e com o termo. Mínimo maior que máximo é recusado em linha, e a listagem anterior continua na tela.
- **Ordenação:** preço crescente, preço decrescente e mais recentes. A ordenação se mantém ao paginar e ao filtrar, e o desempate é estável: a paginação nunca repete nem pula Produto.
- **Paginação:** 20 por página, com teto de 60 aceito do cliente e aplicado no servidor. A resposta informa página e total. Página além do total é vazio tratado, não erro.
- **Desempenho:** com 5.000 Produtos semeados, a busca com filtros responde em ≤ 500 ms no p95. Se falhar, a saída é mais índice antes de qualquer serviço: nunca cache, nunca Elasticsearch.
- **Ordem de corte:** Categorias (2º), Vendedores (3º) e Ordenação (5º). Vendedores e Categorias vêm semeados, então cortar a gestão não quebra os roteiros.
- **Transversais como AC:** limites verificados no servidor, lidos da configuração `AZAMON_`; correlação em todo log; acesso a outro módulo só pela interface pública; centavos inteiros; WCAG 2.2 AA; 360–1440 px sem rolagem horizontal; termos do glossário literais na tela.

## Technical Decisions

- **Posse das rotas (exaustiva):** `GET /api/v1/produtos` é de `busca`, com ou sem termo, filtros, ordenação e paginação. `GET /api/v1/produtos/<id>` e `/api/v1/admin/...` (CRUD, Estoque e a listagem administrativa `GET /api/v1/admin/produtos`, que inclui inativos) são de `catalogo`. `catalogo` não registra listagem nem paginação de Produto **da loja** (emenda do AD-16 na 3.2/3.3: a proibição vale para a loja; a listagem administrativa é do `catalogo`). Se dois módulos registrarem o mesmo padrão no `ServeMux`, o binário entra em pânico no arranque.
- **`busca` não tem tabelas:** lê `catalogo` só por uma VIEW dedicada, construída sobre o predicado de visibilidade. A VIEW expõe `estoque_disponivel` derivado e nunca `estoque_total`. Nenhum outro módulo monta consulta para localizar Produto.
- **Normalização na escrita:** coluna dedicada (sem acento, minúscula) preenchida pelo Go, com índice GIN `pg_trgm`. Nada de `tsvector`. A coluna, o índice e a extensão já existem desde a Épica 1.
- **Estoque (AD-5):** `catalogo` é dono de `produto.estoque_total` e de `catalogo.reserva_estoque`. O disponível nunca é armazenado. `catalogo.Disponivel(ctx, produtoIDs) map[uuid]int` trabalha em lote. A ordem é obrigatória, em dois comandos: `SELECT ... WHERE id = ANY($1) ORDER BY id FOR UPDATE` primeiro e a soma das reservas depois. A guarda de ajuste do Administrador segue a mesma ordem. `Disponivel` orienta, mas quem decide é `Reservar` sob bloqueio.
- **Reserva de Estoque:** estados `ATIVA → LIBERADA | CONSOLIDADA`, com índice único parcial sobre `(pedido_id, produto_id) WHERE estado = 'ATIVA'`. Toda mutação é idempotente e declara isso: `Liberar` sem reserva ativa devolve `nil`, `Consolidar` repetido não faz nada, `Reservar([])` não faz nada. `Liberar` nunca escreve `estoque_total`; só `Consolidar` escreve.
- **Visibilidade (AD-19):** "Produto visível" é Produto ativo **e** Vendedor ativo, definido uma única vez em `catalogo` e exposto por `catalogo.Visiveis`. Nenhum outro lugar escreve `WHERE ativo = true`. `Disponivel` devolve `0` para Produto inexistente ou invisível, sem omitir a chave e sem erro. `Reservar` verifica visibilidade na mesma consulta `FOR UPDATE` e falha com `catalogo.ErrEstoqueInsuficiente`.
- **Envelope único de listagem:** `{ "itens": [...], "pagina": 1, "por_pagina": 20, "total": 137 }`, com desempate que sempre termina em `id` ascendente.
- **Dinheiro:** `int64` de centavos com sufixo `_centavos`, e o filtro de preço também em centavos. A formatação `R$` só acontece no navegador.
- **Dados:** um schema por módulo e nenhuma FK cruzando schema. Imagens em `media/` com URL relativa servida pelo Go, e nada vindo de rede externa. As migrações novas desta épica (Reserva de Estoque, VIEW) seguem as convenções: `uuidv7()`, `timestamptz`, `snake_case` singular.

## UX & Interaction Patterns

- **Área administrativa:** chrome próprio em `chrome-muted`, sem busca e sem Carrinho, desktop-first. A navegação tem quatro destinos (Vendedores, Categorias, Produtos, Pedidos) e vira `Sheet` abaixo de 640 px. Lista vazia mostra "Nenhum {entidade} cadastrado." com a ação única de criar. Erro aparece em linha por campo, nomeando o limite, nunca como resumo no topo.
- **Vitrine:** grade de 1 a 4 colunas nas faixas 360–639, 640–1023, 1024–1439 e ≥1440 px. O carregamento usa `Skeleton` em grade com a forma dos cartões, nunca spinner. Catálogo vazio mostra "O Catálogo ainda não tem Produtos.", com caminho para cadastrar só para Sessão de Administrador.
- **Cartão de Produto:** imagem em 4/3, nome em `link`, preço nos papéis `preco`/`preco-centavos`, selo de disponibilidade e "Adicionar ao Carrinho" (1 unidade). Clicar em qualquer ponto abre a Página de Produto. Imagem ausente vira um bloco neutro com o nome centralizado, nunca o ícone de imagem quebrada.
- **Selo de disponibilidade:** texto em `available`, sem fundo e sem ícone, "Em estoque" ou "Indisponível". Na Caixa de compra, mostra o número quando o disponível é menor que 10. O número exibido é sempre o Estoque disponível. A Reserva nunca é nomeada para o Comprador; o termo aparece só na guarda do Administrador. O verde fica restrito a disponibilidade.
- **Página de Produto:** Caixa de compra à direita a partir de 1024 px e empilhada abaixo disso. Breadcrumb de Categoria acima do título. Produto não encontrado mostra "Este Produto não está disponível." com a ação única de voltar à Vitrine.
- **Resultados:** sem resultado, mostra "Nenhum Produto encontrado para '{termo}'." com chips dos filtros e "Limpar filtros", nunca página em branco. A contagem é anunciada por `aria-live="polite"`. Os filtros ficam em `Sheet` de 640 a 1023 px e fixos à esquerda a partir de 1024 px. Chips ativos em `rounded.full`, removíveis um a um.
- **Estado na URL:** termo, filtros, ordenação e página vivem na URL; recarregar, compartilhar e voltar no navegador reproduzem a listagem. Sem rolagem infinita e sem consulta em intervalo.
- **Chrome da loja:** barra superior, busca global, Faixa de Categorias e Menu da conta aparecem em toda tela pública e de Comprador. `Enter` submete a busca de qualquer tela. A busca global é o único elemento fora do ritmo folgado. A Faixa de Categorias é plana; clicar nela equivale a buscar com termo vazio e aquela Categoria, e abaixo de 640 px ela rola dentro de si. `rounded.full` tem exatamente quatro usos: busca, botões de ação, selos de Status e chips.
- **Referência visual:** `mockups/resultados-busca.html` e `mockups/pagina-de-produto.html` ilustram a composição, mas as espinhas vencem em conflito.

## Cross-Story Dependencies

- **Da Épica 1 já vêm prontos:** as tabelas `vendedor`, `categoria` e `produto`, o `pg_trgm`, a coluna normalizada com índice GIN, a semente determinística (com o conjunto de 5.000 ligado por configuração), a rota de detalhe crua e a base visual.
- **Da Épica 2 já vêm prontos:** a separação de papéis por prefixo `/api/v1/admin/`, o Menu da conta e o retorno ao ponto de partida pelo Login.
- **Dentro da épica:** 3.4 (VIEW sobre o predicado de visibilidade e o disponível derivado) é a base de 3.5 a 3.9. A 3.5 fixa rota, VIEW e envelope; depois dela, termo (3.7), filtros (3.8) e ordenação (3.9) podem correr em paralelo. A 3.1 depende do predicado da 3.4 para ocultar os Produtos. A 3.10 depende das Categorias (3.2 ou semente) e da busca (3.7).
- **Para as épicas seguintes:** `Disponivel`, `Visiveis`, `ObterVisivel`, `Reservar`, `Liberar` e `Consolidar` são a interface que Carrinho (Épica 4), criação de Pedido (Épica 5) e cancelamento (Épica 6) consomem. As assinaturas e a idempotência fixadas aqui não mudam depois. O caminho "visitante adiciona → Login → volta com o Item criado" da 3.6 só se completa com o Carrinho da Épica 4.
