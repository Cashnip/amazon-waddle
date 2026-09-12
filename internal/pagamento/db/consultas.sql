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
-- name: TentativasSemConfirmacao :many
SELECT t.id_externo, t.total_centavos
FROM pagamento.tentativa_pagamento t
WHERE t.criada_em <= @ate
  AND NOT EXISTS (
      SELECT 1 FROM pagamento.confirmacao_recebida c WHERE c.tentativa_id = t.id
  )
ORDER BY t.criada_em;
