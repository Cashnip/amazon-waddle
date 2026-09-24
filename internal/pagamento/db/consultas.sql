-- As consultas da porta do Provedor de Pagamento. A Tentativa nasce dentro da
-- transação de `pedido.Criar`; a confirmação entra pelo webhook, fora dela; e
-- a varredura lê as duas listas sem transação aberta.

-- name: CriarTentativa :exec
INSERT INTO pagamento.tentativa_pagamento (pedido_id, total_centavos, id_externo, numero)
VALUES (@pedido_id, @total_centavos, @id_externo, @numero);

-- A chave desconhecida não é erro do sistema: é uma confirmação sobre
-- Tentativa que não existe, e `api/` a traduz em 404 sem gravar nada.
-- name: BuscarTentativa :one
SELECT id FROM pagamento.tentativa_pagamento WHERE id_externo = @id_externo;

-- A restrição única sobre chave_idempotencia é quem decide: o 23505 devolvido
-- daqui é sinal de "esta confirmação já entrou", e vira no-op no Go.
-- name: GravarConfirmacao :exec
INSERT INTO pagamento.confirmacao_recebida (tentativa_id, chave_idempotencia, resultado, estado)
VALUES (@tentativa_id, @chave_idempotencia, @resultado, 'PENDENTE');

-- O que a varredura de `pedido` lê. `corrente` diz se a confirmação pertence à
-- Tentativa corrente do Pedido — a que tem o maior número. A que não pertence
-- também sai daqui, para virar terminal em vez de ficar PENDENTE para sempre.
-- name: ConfirmacoesNaoAplicadas :many
SELECT c.id, t.pedido_id, c.resultado, (t.numero = m.maior)::boolean AS corrente
FROM pagamento.confirmacao_recebida c
JOIN pagamento.tentativa_pagamento t ON t.id = c.tentativa_id
JOIN (
    SELECT pedido_id, max(numero) AS maior
    FROM pagamento.tentativa_pagamento GROUP BY pedido_id
) m ON m.pedido_id = t.pedido_id
WHERE c.estado = 'PENDENTE'
ORDER BY c.recebida_em;

-- Terminal em compare-and-swap, como o Status do Pedido: só sai de PENDENTE
-- quem ainda está lá, e zero linhas é outro tique que chegou primeiro.
-- name: MarcarConfirmacao :exec
UPDATE pagamento.confirmacao_recebida
SET estado = @estado
WHERE id = @id AND estado = 'PENDENTE';

-- A emissão é derivada, e não marcada (AD-7): Tentativa cujo atraso venceu e
-- que não tem linha na inbox. Marcar "já emiti" exigiria escrever depois do
-- POST, e a queda entre uma coisa e outra deixaria a Tentativa órfã para
-- sempre; derivando, reemitir é o comportamento normal e a chave única
-- transforma o reenvio em no-op.
-- O `numero` sai junto porque o Simulado decide pelos centavos só na primeira
-- Tentativa (§7.1, AD-8): da segunda em diante aprova.
--
-- A janela tem os dois lados. O piso `criada_em > @desde` é o prazo da
-- expiração (FR-34): passado ele a Tentativa está morta — a varredura já
-- encerrou o Pedido —, e emitir sobre ela não teria efeito nenhum. Sem o piso,
-- a faixa que nunca confirma seria relida a cada tique para sempre. Continua
-- derivada, e não marcada (AD-7): nada é escrito para sair da lista.
-- name: TentativasSemConfirmacao :many
SELECT t.id_externo, t.total_centavos, t.numero
FROM pagamento.tentativa_pagamento t
WHERE t.criada_em <= @ate
  AND t.criada_em > @desde
  AND NOT EXISTS (
      SELECT 1 FROM pagamento.confirmacao_recebida c WHERE c.tentativa_id = t.id
  )
ORDER BY t.criada_em;

-- A porta da FR-34: existe confirmação ainda PENDENTE para este Pedido?
-- A expiração a chama com a linha do Pedido já presa, e desiste se houver uma:
-- a ordem do tique sozinha não basta — `aplicar` pode ter falhado, o SKIP LOCKED
-- pode ter pulado a linha, e a confirmação pode ter chegado entre os dois passos.
-- Em todos, o Comprador pagou no prazo e perderia a Reserva de Estoque.
-- name: TemConfirmacaoPendente :one
SELECT EXISTS (
    SELECT 1
    FROM pagamento.confirmacao_recebida c
    JOIN pagamento.tentativa_pagamento t ON t.id = c.tentativa_id
    WHERE t.pedido_id = @pedido_id AND c.estado = 'PENDENTE'
)::boolean;

-- Quantas Tentativas o Pedido já teve. O teto é de `pagamento`, dono da
-- entidade (AD-8): `pedido` recebe as restantes prontas, e não conta sozinho.
-- name: ContarTentativas :one
SELECT count(*)::integer FROM pagamento.tentativa_pagamento WHERE pedido_id = @pedido_id;

-- A metade de `pagamento` do sinal da FR-26 (6.5): quais destes Pedidos têm uma
-- aprovação que chegou e não pôde ser aplicada. A outra metade — o Pedido
-- estar CANCELADO — é de `pedido`, e a junção é em Go (AD-1, AD-2): esta
-- consulta não conhece Status nenhum. Tentativa superada conta, porque aprovação
-- de Tentativa superada também é dinheiro aprovado; RECUSADO não conta, porque
-- é dinheiro que nunca entrou. Uma ida só para a página inteira da Tabela.
-- name: PedidosComAprovacaoSinalizada :many
SELECT DISTINCT t.pedido_id
FROM pagamento.confirmacao_recebida c
JOIN pagamento.tentativa_pagamento t ON t.id = c.tentativa_id
WHERE t.pedido_id = ANY(@pedido_ids::uuid[])
  AND c.resultado = 'APROVADO'
  AND c.estado = 'NAO_APLICAVEL_SINALIZADA';
