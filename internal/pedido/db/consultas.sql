-- As consultas do caso de uso "um Pedido nasce em AGUARDANDO_PAGAMENTO".
-- Todas rodam dentro da mesma transação, passada como `tx pgx.Tx` (AD-4).

-- O contador por ano resolve o número legível na própria transação: sem DDL em
-- tempo de execução e sem buraco na numeração quando a transação é revertida.
-- O `contador_numero` do SET é a linha em conflito — dentro do ON CONFLICT o
-- nome da tabela é o apelido, e qualificá-lo pelo schema seria erro de sintaxe.
-- name: ProximoNumeroDoAno :one
INSERT INTO pedido.contador_numero (ano, ultimo) VALUES ($1, 1)
ON CONFLICT (ano) DO UPDATE SET ultimo = contador_numero.ultimo + 1
RETURNING ultimo;

-- name: CriarPedido :one
INSERT INTO pedido.pedido (numero, comprador_id, status, total_centavos)
VALUES ($1, $2, $3, $4)
RETURNING id, numero, status, total_centavos;

-- O Item congela o que era verdade no instante da compra: nome, preço
-- praticado e Vendedor são cópia, e não referência.
-- name: CriarItemPedido :exec
INSERT INTO pedido.item_pedido
    (pedido_id, produto_id, nome, vendedor_nome, preco_praticado_centavos, quantidade)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: RegistrarTransicao :exec
INSERT INTO pedido.transicao_status (pedido_id, status_anterior, status_novo, autor)
VALUES ($1, $2, $3, $4);

-- O único ponto de mutação do Status (AD-6), e é compare-and-swap: o estado de
-- origem entra no WHERE, e zero linhas afetadas significa "o estado já
-- avançou", nunca "o Pedido não existe" — as duas coisas são o mesmo desfecho
-- esperado para quem chama.
-- name: AvancarStatus :execrows
UPDATE pedido.pedido SET status = @para WHERE id = @pedido_id AND status = @de;
