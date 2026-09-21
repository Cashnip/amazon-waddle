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

-- A Regra de Frete inteira (AD-17): são nove linhas, e a função pura escolhe.
-- A padrão vem por último, e a ordem por `cep_inicio` torna a escolha
-- determinística mesmo que alguém acrescente uma faixa sobreposta.
-- name: ListarFaixasDeFrete :many
SELECT cep_inicio, cep_fim, regiao, valor_centavos, padrao
FROM pedido.faixa_frete
ORDER BY padrao, cep_inicio;

-- O INSERT é a reivindicação da `Idempotency-Key` (AD-7): o gêmeo concorrente
-- espera no índice único (comprador_id, chave_idempotencia) e, quando o
-- primeiro comita, cai no DO NOTHING — zero linhas, que `pedido` traduz em
-- releitura por PedidoPorChave. O 23505 nunca chega a existir, e por isso
-- nunca chega ao navegador. Tudo que o Pedido congela está aqui, e o gatilho
-- da 5.1 impede reescrever depois.
-- name: CriarPedido :one
INSERT INTO pedido.pedido (
    numero, comprador_id, status, subtotal_centavos, frete_centavos, total_centavos,
    endereco_destinatario, endereco_cep, endereco_logradouro, endereco_numero,
    endereco_complemento, endereco_bairro, endereco_cidade, endereco_uf,
    chave_idempotencia, digest_corpo
) VALUES (
    @numero, @comprador_id, @status, @subtotal_centavos, @frete_centavos, @total_centavos,
    @endereco_destinatario, @endereco_cep, @endereco_logradouro, @endereco_numero,
    @endereco_complemento, @endereco_bairro, @endereco_cidade, @endereco_uf,
    @chave_idempotencia, @digest_corpo
)
ON CONFLICT (comprador_id, chave_idempotencia) DO NOTHING
RETURNING id, numero, status, total_centavos;

-- O Pedido que a chave já criou, deste Comprador: a chave de outro Comprador
-- não casa, porque a chave é dele (AD-11). O digest vem junto para decidir
-- entre reenvio (mesmo corpo) e chave reaproveitada (corpo diferente).
-- name: PedidoPorChave :one
SELECT id, numero, status, total_centavos, digest_corpo
FROM pedido.pedido
WHERE comprador_id = @comprador_id AND chave_idempotencia = @chave_idempotencia;

-- O Item congela o que era verdade no instante da compra: nome, preço
-- praticado e Vendedor são cópia, e não referência.
-- name: CriarItemPedido :exec
INSERT INTO pedido.item_pedido
    (pedido_id, produto_id, nome, vendedor_nome, preco_praticado_centavos, quantidade)
VALUES ($1, $2, $3, $4, $5, $6);

-- O `motivo` é anulável: nem toda transição tem um. `sqlc.narg` o deixa
-- chegar como NULL, e não como texto vazio, que seria um motivo sem conteúdo.
-- name: RegistrarTransicao :exec
INSERT INTO pedido.transicao_status (pedido_id, status_anterior, status_novo, ator, motivo)
VALUES (@pedido_id, @status_anterior, @status_novo, @ator, sqlc.narg('motivo'));

-- O único ponto de mutação do Status (AD-6), e é compare-and-swap: o estado de
-- origem entra no WHERE, e zero linhas afetadas significa "o estado já
-- avançou", nunca "o Pedido não existe" — as duas coisas são o mesmo desfecho
-- esperado para quem chama.
-- name: AvancarStatus :execrows
UPDATE pedido.pedido SET status = @para WHERE id = @pedido_id AND status = @de;

-- A releitura do Status depois de um compare-and-swap perdido: é ela que
-- separa "outro ator avançou" de "o Pedido saiu da janela de cancelamento".
-- Sem trava — quem ganhou já comitou, ou o UPDATE acima teria esperado por ele.
-- name: StatusDoPedido :one
SELECT status FROM pedido.pedido WHERE id = @pedido_id;

-- Os Itens do próprio Pedido, que é o que a nova Tentativa reserva de novo
-- (AD-3): nada é remontado a partir do Carrinho.
-- name: ItensParaReserva :many
SELECT produto_id, quantidade
FROM pedido.item_pedido
WHERE pedido_id = @pedido_id
ORDER BY produto_id;

-- O histórico em ordem de acontecimento. O `id` desempata duas transições no
-- mesmo instante: é uuidv7(), ordenado no tempo por construção.
-- name: HistoricoDoPedido :many
SELECT status_anterior, status_novo, ator, motivo, ocorrido_em
FROM pedido.transicao_status
WHERE pedido_id = @pedido_id
ORDER BY ocorrido_em, id;

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

-- A listagem da tela "Meus pedidos" (2.6). O dono entra no WHERE, e não numa
-- checagem depois (AD-11): a rota nunca lê Pedido de outro Comprador para
-- descartar depois. `id DESC` e não por data: a chave é uuidv7(), ordenada no
-- tempo por construção, então o mais recente já sai no topo sem JOIN em
-- transicao_status nem coluna nova — é o esboço que a Estória 6.1 substitui.
-- name: ListarPedidosDoComprador :many
SELECT id, numero, status, total_centavos
FROM pedido.pedido
WHERE comprador_id = @comprador_id
ORDER BY id DESC;

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
-- ponytail: varredura sequencial de pedido.pedido a cada tique. O
-- max(ocorrido_em) correlacionado já tem `transicao_status (pedido_id,
-- ocorrido_em)`, da 5.1; quando o volume justificar, o índice que falta é
-- `pedido (status, id)`, sem tocar na consulta.
-- name: PedidosParaAvancar :many
SELECT p.id
FROM pedido.pedido p
WHERE p.status = ANY(@status::text[])
  AND (SELECT max(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id) <= @ate
ORDER BY p.id;
