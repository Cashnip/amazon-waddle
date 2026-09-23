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
-- Subtotal, Frete e Endereço são os congelados da 5.6: a tela os exibe como
-- foram gravados, e não os recompõe (AD-9). O Endereço é todo nulo no Pedido
-- do esqueleto — o CHECK `pedido_criacao_completa` garante que nunca pela metade.
-- name: BuscarPedidoDoComprador :one
SELECT p.id, p.numero, p.status, p.subtotal_centavos, p.frete_centavos, p.total_centavos,
       p.endereco_destinatario, p.endereco_cep, p.endereco_logradouro, p.endereco_numero,
       p.endereco_complemento, p.endereco_bairro, p.endereco_cidade, p.endereco_uf,
       -- O cast é carga: sem ele o sqlc não infere o tipo da subconsulta e
       -- devolve `interface{}`, que só falharia no Scan em tempo de execução.
       (SELECT max(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id)::timestamptz AS atualizado_em
FROM pedido.pedido p
WHERE p.id = @pedido_id AND p.comprador_id = @comprador_id;

-- Os Itens de Pedido como a tela os mostra: o que foi congelado na compra,
-- nunca o preço de hoje (FR-30). Sem dono no WHERE: quem chama já leu o
-- Pedido pelo dono (AD-11). A ordem por `id` é a da criação — uuidv7().
-- O `vendedor_nome` é o congelado na criação, como o `nome` e o preço: é o
-- Vendedor que vendeu, e não o que hoje estaria ligado ao Produto. Sai nesta
-- consulta, e não numa segunda só para o Administrador, porque a coluna já
-- está na tabela e a linha já é lida — quem não o exibe (o Comprador) só não
-- o serializa.
-- name: ItensDoPedido :many
SELECT produto_id, nome, vendedor_nome, preco_praticado_centavos, quantidade
FROM pedido.item_pedido
WHERE pedido_id = @pedido_id
ORDER BY id;

-- O Detalhe administrativo (6.4). É BuscarPedidoDoComprador sem o dono no
-- WHERE e com o `comprador_id` a mais: o Administrador vê o Pedido de
-- qualquer Comprador, e é por esse identificador que `identidade` devolve o
-- nome e o e-mail de quem comprou.
--
-- Consulta própria, e não um parâmetro nulo em BuscarPedidoDoComprador: o
-- dono no WHERE é a garantia do AD-11, e um `@comprador_id IS NULL OR …`
-- deixaria a leitura do Comprador a uma linha de distância de virar leitura
-- de todo mundo.
-- name: BuscarPedidoParaAdministrador :one
SELECT p.id, p.numero, p.status, p.comprador_id,
       p.subtotal_centavos, p.frete_centavos, p.total_centavos,
       p.endereco_destinatario, p.endereco_cep, p.endereco_logradouro, p.endereco_numero,
       p.endereco_complemento, p.endereco_bairro, p.endereco_cidade, p.endereco_uf,
       -- O cast é carga, pelo mesmo motivo de BuscarPedidoDoComprador.
       (SELECT max(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id)::timestamptz AS atualizado_em
FROM pedido.pedido p
WHERE p.id = @pedido_id;

-- A Tabela de Pedidos do Administrador (6.4): todos os Compradores, filtrável
-- por Status e ordenável por data. Sem dono no WHERE — é justamente o que
-- distingue esta listagem da do Comprador.
--
-- O filtro é opcional por `IS NULL OR`, no molde de `busca`, e a data é o
-- nascimento do histórico, como em ListarPedidosDoComprador — nenhuma coluna
-- nova. A subconsulta se repete no ORDER BY porque o PostgreSQL não deixa
-- usar o apelido da lista de seleção dentro de uma expressão.
--
-- Uma chave por ordenação; a que não vale vira NULL e empata. `recentes` é
-- DESC com NULLS LAST pelo motivo de ListarPedidosDoComprador, e o desempate
-- final é sempre `id` (AD-18).
--
-- ponytail: a subconsulta correlacionada sai TRÊS vezes por linha — uma na
-- lista de seleção e uma em cada chave de ordenação —, sobre varredura
-- sequencial de `pedido.pedido` sem o dono no WHERE que estreita a do
-- Comprador. Teto: o índice `transicao_status (pedido_id, ocorrido_em)` da 5.1
-- serve as três, e o volume do MVP (dezenas de Pedidos) as torna invisíveis; o
-- que dói primeiro é a ordenação, que roda sobre a tabela inteira antes do
-- LIMIT. A saída barata é uma `CROSS JOIN LATERAL` calculando `criado_em` uma
-- vez e ordenando pela coluna dela; a cara, e que mentiria sobre os Pedidos já
-- gravados, é `criado_em` denormalizado no Pedido.
-- name: ListarPedidosAdmin :many
SELECT p.id, p.numero, p.status, p.total_centavos,
       (SELECT min(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id)::timestamptz AS criado_em
FROM pedido.pedido p
WHERE (sqlc.narg(status)::text IS NULL OR p.status = sqlc.narg(status)::text)
ORDER BY
  CASE WHEN @ordenacao::text = 'recentes'
       THEN (SELECT min(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id) END DESC NULLS LAST,
  CASE WHEN @ordenacao::text = 'antigos'
       THEN (SELECT min(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id) END ASC,
  p.id
LIMIT @limite OFFSET @deslocamento;

-- O total da mesma listagem, com o WHERE dela palavra por palavra: um filtro
-- que divergisse aqui paginaria sobre um total que não é o da lista.
-- name: ContarPedidosAdmin :one
SELECT count(*)
FROM pedido.pedido p
WHERE (sqlc.narg(status)::text IS NULL OR p.status = sqlc.narg(status)::text);

-- A leitura travada da transição pelo Administrador (6.4). É
-- TravarPedidoDoComprador sem o dono: o Administrador alcança o Pedido de
-- qualquer Comprador, e a trava serve a outra coisa — prender a linha antes
-- do compare-and-swap para que o Status lido seja o que o CAS vai ver.
--
-- FOR UPDATE que ESPERA, e não o SKIP LOCKED da varredura: o Administrador
-- clicou num botão e precisa do desfecho. É a trava que faz a corrida com a
-- simulação sair como "o Pedido já está em X", e não como transição inválida.
-- name: TravarPedidoParaAdministrador :one
SELECT p.id, p.numero, p.status, p.total_centavos
FROM pedido.pedido p
WHERE p.id = @pedido_id
FOR UPDATE;

-- A listagem da tela "Meus pedidos" (6.1). O dono entra no WHERE, e não numa
-- checagem depois (AD-11): a rota nunca lê Pedido de outro Comprador para
-- descartar depois.
--
-- A data é o nascimento já gravado no histórico: `pedido.Criar` registra a
-- primeira transição com `status_anterior` vazio na mesma transação do
-- INSERT, então `min(ocorrido_em)` É o instante de nascimento, e não uma
-- aproximação — nenhuma coluna nova, nenhuma migração. O cast é carga, pelo
-- mesmo motivo de BuscarPedidoDoComprador: sem ele o sqlc devolve
-- `interface{}` e a falha só apareceria no Scan.
--
-- A ordem é o nascimento, do mais recente para o mais antigo, terminando em
-- `id` para o OFFSET ser estável: duas transações concorrentes podem empatar
-- no instante, e aí quem desempata é a chave — uuidv7(), já cronológica por
-- construção. NULLS LAST porque DESC é NULLS FIRST no PostgreSQL: Pedido sem
-- linha em transicao_status é inalcançável pelo aplicativo (Criar grava a
-- primeira na mesma transação), mas se existisse iria ao topo da página 1 e
-- sairia sem data nenhuma.
--
-- ponytail: a subconsulta por linha na ordenação. O índice
-- `transicao_status (pedido_id, ocorrido_em)` da 5.1 já a serve; o que
-- pagaria mais é `criado_em` denormalizado no Pedido, e isso mentiria sobre
-- os Pedidos já gravados.
-- name: ListarPedidosDoComprador :many
SELECT p.id, p.numero, p.status, p.total_centavos,
       (SELECT min(t.ocorrido_em) FROM pedido.transicao_status t WHERE t.pedido_id = p.id)::timestamptz AS criado_em
FROM pedido.pedido p
WHERE p.comprador_id = @comprador_id
ORDER BY criado_em DESC NULLS LAST, p.id DESC
LIMIT @limite OFFSET @deslocamento;

-- O total da mesma listagem, com o WHERE dela palavra por palavra: um filtro
-- que divergisse aqui paginaria sobre um total que não é o da lista.
-- name: ContarPedidosDoComprador :one
SELECT count(*)
FROM pedido.pedido p
WHERE p.comprador_id = @comprador_id;

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

-- A leitura travada do cancelamento pelo Comprador (6.3). O dono entra no
-- WHERE, como em BuscarPedidoDoComprador: alheio e inexistente saem os dois
-- como "nenhuma linha", o mesmo 404 (AD-11) — e o Pedido alheio nem chega a
-- ser travado.
--
-- FOR UPDATE que ESPERA, e não o SKIP LOCKED da varredura: o Comprador que
-- cancela precisa do desfecho, e não de "tente no próximo tique". Com a linha
-- presa, o Status lido é o que o compare-and-swap de Transicionar vai ver — é
-- isso que faz o segundo clique ler CANCELADO (e sair sucesso sem efeito) em
-- vez de perder o CAS e sair FORA_DA_JANELA_DE_CANCELAMENTO de um
-- cancelamento que deu certo. É também a primeira trava do AD-4: a linha do
-- Pedido antes dos Produtos, que Liberar trava depois.
-- name: TravarPedidoDoComprador :one
SELECT p.id, p.numero, p.status, p.total_centavos
FROM pedido.pedido p
WHERE p.id = @pedido_id AND p.comprador_id = @comprador_id
FOR UPDATE;

-- Os candidatos dos dois passos do tempo — a expiração da Tentativa de
-- Pagamento (FR-34) e a simulação de entrega —, lidos FORA da transação: quem
-- decide é a releitura travada de TravarPedido, e selecionar já travando faria
-- uma transação por tique em vez de uma por Pedido. O avanço deriva do
-- histórico — "está neste estado desde quando" —, nunca de estado em memória, e
-- é por isso que reiniciar o contêiner retoma cada Pedido de onde parou.
--
-- É a mesma conta que `pedido.ExpiraEm` mostra na tela: para um Pedido em
-- AGUARDANDO_PAGAMENTO, max(ocorrido_em) É a última transição para esse Status,
-- porque qualquer linha posterior o teria tirado de lá.
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
