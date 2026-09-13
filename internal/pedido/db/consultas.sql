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

-- A leitura da tela de acompanhamento. O dono entra no WHERE, e não numa
-- checagem depois: Pedido de outro Comprador e Pedido inexistente saem os dois
-- como "nenhuma linha", que é o mesmo 404 — não vaza existência.
-- O instante é o da última transição, absoluto e vindo do servidor: o
-- navegador nunca conta duração.
-- name: BuscarPedidoDoComprador :one
SELECT p.id, p.numero, p.status, p.total_centavos,
       -- O cast é carga: sem ele o sqlc não infere o tipo da subconsulta e
       -- devolve `interface{}`, que só falharia no Scan em tempo de execução.
       (SELECT max(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id)::timestamptz AS atualizado_em
FROM pedido.pedido p
WHERE p.id = @pedido_id AND p.comprador_id = @comprador_id;

-- A leitura travada da varredura. SKIP LOCKED porque o tique que encontra o
-- Pedido já travado não tem o que esperar: o outro caminho está aplicando, e
-- insistir só serializaria a varredura inteira num Pedido.
-- O instante da última transição sai junto, e na mesma trava: a decisão de
-- avançar precisa do Status e de "desde quando" travados um com o outro. Duas
-- consultas travadas quase idênticas seriam duas oportunidades de divergirem.
-- name: TravarPedido :one
SELECT p.status,
       (SELECT max(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id)::timestamptz AS desde
FROM pedido.pedido p
WHERE p.id = @pedido_id
FOR UPDATE SKIP LOCKED;

-- Os candidatos da simulação de entrega, lidos FORA da transação: quem decide
-- é a releitura travada de TravarPedido, e selecionar já travando faria uma
-- transação por tique em vez de uma por Pedido. O avanço deriva do histórico —
-- "está neste estado desde quando" —, nunca de estado em memória, e é por isso
-- que reiniciar o contêiner retoma cada Pedido de onde parou.
--
-- ponytail: varredura sequencial de pedido.pedido a cada tique, com um
-- max(ocorrido_em) correlacionado por linha — a tabela só tem
-- pedido_comprador_id_idx, e esta épica proíbe migração nova. Na demonstração
-- são dezenas de Pedidos e não se mede. Quando o volume justificar, os índices
-- que a levantam são `pedido (status, id)` e
-- `transicao_status (pedido_id, ocorrido_em DESC)`, sem tocar na consulta.
-- name: PedidosParaAvancar :many
SELECT p.id
FROM pedido.pedido p
WHERE p.status = ANY(@status::text[])
  AND (SELECT max(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id) <= @ate
ORDER BY p.id;
