-- A consulta da Página de Produto (FR-9). Lê da VIEW do AD-19: Produto de
-- Vendedor desativado sai daqui como zero linhas, o mesmo 404 do inexistente.
-- `busca_normalizada` fica de fora: é dado de índice. A Categoria entra por
-- JOIN (3.6), e não pela VIEW: o contrato que `busca` lê não muda por causa
-- de uma tela só. O número é sempre o disponível, nunca o total.
-- name: BuscarProdutoComVendedor :one
SELECT pv.id, pv.nome, pv.descricao, pv.preco_centavos, pv.imagem_url, pv.vendedor_nome,
       pv.categoria_id, c.nome AS categoria_nome, pv.estoque_disponivel
FROM catalogo.produto_visivel pv
JOIN catalogo.categoria c ON c.id = pv.categoria_id
WHERE pv.id = $1;

-- As duas consultas abaixo são o AD-5, e são duas de propósito: é a ordem
-- entre elas que protege o Estoque. Uma consulta só não teria como errá-la —
-- e errá-la é vender a mesma última unidade duas vezes.
--
-- Primeiro a trava. O `ORDER BY id` não é estilo: sem ele, dois Pedidos com os
-- mesmos dois Produtos em ordem inversa travam um no outro.
--
-- A visibilidade entra na mesma consulta, por subconsulta à VIEW, e a trava é
-- `OF p`: `FOR UPDATE` sobre a VIEW travaria também a linha do Vendedor, e
-- Pedidos de Produtos diferentes do mesmo Vendedor esperariam um pelo outro.
-- name: TravarProdutosParaReserva :many
SELECT p.id, p.estoque_total
FROM catalogo.produto p
WHERE p.id = ANY(@ids::uuid[])
  AND p.id IN (SELECT id FROM catalogo.produto_visivel)
ORDER BY p.id
FOR UPDATE OF p;

-- Só depois a soma, em lote. Com a trava segura, nenhuma Reserva nova entra
-- entre a soma e o INSERT que a sucede — nem, no ajuste do Administrador,
-- entre a soma e o UPDATE do total. Produto sem Reserva ativa não aparece.
-- name: SomarReservasAtivas :many
SELECT produto_id, sum(quantidade)::bigint AS reservado
FROM catalogo.reserva_estoque
WHERE produto_id = ANY(@ids::uuid[]) AND estado = 'ATIVA'
GROUP BY produto_id;

-- name: CriarReservaAtiva :exec
INSERT INTO catalogo.reserva_estoque (produto_id, pedido_id, quantidade, estado)
VALUES ($1, $2, $3, 'ATIVA');

-- A liberação (AD-5): muda só o estado, nunca `estoque_total`. O
-- `estado = 'ATIVA'` é compare-and-swap: repetir não encontra nada, e é no-op.
-- name: LiberarReservasDoPedido :exec
UPDATE catalogo.reserva_estoque
SET estado = 'LIBERADA'
WHERE pedido_id = @pedido_id AND estado = 'ATIVA';

-- O disponível de `Disponivel` e `Visiveis` (AD-19): lido da VIEW, que é o
-- único lugar do predicado de visibilidade. Id invisível ou inexistente não
-- volta, e quem preenche o zero é o Go.
-- name: DisponivelDosVisiveis :many
SELECT id, estoque_disponivel
FROM catalogo.produto_visivel
WHERE id = ANY(@ids::uuid[]);

-- O que o Carrinho mostra de cada Produto (4.3), lido da mesma VIEW e em lote:
-- um Carrinho de N Itens é uma ida, e não N. Invisível ou inexistente não volta.
-- name: ResumosDosVisiveis :many
SELECT id, nome, preco_centavos, imagem_url, estoque_disponivel
FROM catalogo.produto_visivel
WHERE id = ANY(@ids::uuid[]);

-- A trava do ajuste do Administrador, primeiro dos dois comandos do AD-5.
-- Sem a VIEW: o ajuste vale para Produto inativo, e travar pela VIEW faria do
-- inativo um 404.
-- name: TravarProdutoParaAjuste :one
SELECT id FROM catalogo.produto
WHERE id = ANY(@ids::uuid[])
ORDER BY id
FOR UPDATE;

-- name: AjustarEstoqueTotal :exec
UPDATE catalogo.produto SET estoque_total = @estoque_total WHERE id = @id;

-- As duas consultas da consolidação (Estória 1.8), a única passagem do Pedido
-- em que o Estoque total muda (a outra é o ajuste do Administrador, da 3.4).
-- O `estado = 'ATIVA'` no WHERE é compare-and-swap como o
-- do Status: zero linhas devolvidas é Reserva já consolidada, que é no-op e
-- nunca erro.
-- name: ConsolidarReservasDoPedido :many
UPDATE catalogo.reserva_estoque
SET estado = 'CONSOLIDADA'
WHERE pedido_id = @pedido_id AND estado = 'ATIVA'
RETURNING produto_id, quantidade;

-- A baixa é uma por Produto, e quem as ordena por identificador é o Go: o
-- UPDATE … RETURNING não aceita ORDER BY, e a ordem é a mesma disciplina do
-- `ORDER BY id FOR UPDATE` do Reservar — dois Pedidos com os mesmos Produtos
-- em ordem inversa travariam um no outro. O CHECK (estoque_total >= 0) da
-- migração é quem barra a baixa a mais.
-- name: BaixarEstoqueTotal :exec
UPDATE catalogo.produto
SET estoque_total = estoque_total - @quantidade
WHERE id = @produto_id;

-- A gestão de Vendedores (3.1). A lista traz ativos e inativos — é a tela do
-- Administrador, e não a loja — por nome, com desempate estável por id.
-- name: ListarVendedores :many
SELECT id, nome, ativo FROM catalogo.vendedor ORDER BY nome, id;

-- Nome duplicado é decidido pelo UNIQUE (23505), nunca por SELECT antes.
-- name: CriarVendedor :one
INSERT INTO catalogo.vendedor (nome) VALUES (@nome)
RETURNING id, nome, ativo;

-- Desativar é este UPDATE de `ativo`: nenhum Pedido é tocado, e quem esconde
-- os Produtos é a VIEW produto_visivel.
-- name: AtualizarVendedor :one
UPDATE catalogo.vendedor SET nome = @nome, ativo = @ativo
WHERE id = @id
RETURNING id, nome, ativo;

-- Vendedor com Produto é barrado pela FK `produto.vendedor_id` (23503), sem
-- contagem antes.
-- name: RemoverVendedor :execrows
DELETE FROM catalogo.vendedor WHERE id = @id;

-- A gestão de Categorias (3.2). `categoria_pai_id` não sai daqui: a Categoria
-- é plana, e a coluna existe nula e não exposta.
-- name: ListarCategorias :many
SELECT id, nome FROM catalogo.categoria ORDER BY nome, id;

-- Nome duplicado é decidido pelo UNIQUE (23505), nunca por SELECT antes.
-- name: CriarCategoria :one
INSERT INTO catalogo.categoria (nome) VALUES (@nome)
RETURNING id, nome;

-- name: AtualizarCategoria :one
UPDATE catalogo.categoria SET nome = @nome
WHERE id = @id
RETURNING id, nome;

-- Categoria com Produto é barrada pela FK `produto.categoria_id` (23503).
-- name: RemoverCategoria :execrows
DELETE FROM catalogo.categoria WHERE id = @id;

-- Só depois do 23503: a recusa diz quantos Produtos estão vinculados, e o
-- caminho comum (remover Categoria vazia) continua sendo uma ida ao banco.
-- name: ContarProdutosDaCategoria :one
SELECT count(*) FROM catalogo.produto WHERE categoria_id = @categoria_id;

-- A listagem administrativa de Produto (3.3, AD-16 emendado): é do
-- `catalogo`, e traz ativos e inativos — a VIEW de `busca` só enxerga Produto
-- visível. Desempate sempre em id (AD-18).
-- name: ListarProdutosAdmin :many
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url,
       p.estoque_total, p.ativo,
       v.id AS vendedor_id, v.nome AS vendedor_nome,
       c.id AS categoria_id, c.nome AS categoria_nome
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
JOIN catalogo.categoria c ON c.id = p.categoria_id
ORDER BY p.nome, p.id
LIMIT @limite OFFSET @deslocamento;

-- name: ContarProdutos :one
SELECT count(*) FROM catalogo.produto;

-- A linha que o criar e o editar devolvem, com os nomes de Vendedor e
-- Categoria da listagem.
-- name: BuscarProdutoAdmin :one
SELECT p.id, p.nome, p.descricao, p.preco_centavos, p.imagem_url,
       p.estoque_total, p.ativo,
       v.id AS vendedor_id, v.nome AS vendedor_nome,
       c.id AS categoria_id, c.nome AS categoria_nome
FROM catalogo.produto p
JOIN catalogo.vendedor v ON v.id = p.vendedor_id
JOIN catalogo.categoria c ON c.id = p.categoria_id
WHERE p.id = @id;

-- Vendedor ou Categoria inexistente é decidido pelas FKs (23503), e o nome da
-- constraint diz qual campo errou. `busca_normalizada` vem do Go.
-- name: CriarProduto :one
INSERT INTO catalogo.produto
    (nome, descricao, preco_centavos, imagem_url, vendedor_id, categoria_id, estoque_total, busca_normalizada)
VALUES (@nome, @descricao, @preco_centavos, @imagem_url, @vendedor_id, @categoria_id, @estoque_total, @busca_normalizada)
RETURNING id;

-- O Estoque total não está aqui: o ajuste tem rota própria, com a guarda das
-- Reservas (3.4) — a tela reenvia a linha lida, e isto regravaria um total
-- velho por cima de uma consolidação. Desativar é este UPDATE de `ativo`, e quem esconde o
-- Produto é a VIEW produto_visivel.
-- name: AtualizarProduto :one
UPDATE catalogo.produto
SET nome = @nome, descricao = @descricao, preco_centavos = @preco_centavos,
    imagem_url = @imagem_url, vendedor_id = @vendedor_id, categoria_id = @categoria_id,
    ativo = @ativo, busca_normalizada = @busca_normalizada
WHERE id = @id
RETURNING id;
