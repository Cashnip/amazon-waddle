---
title: 'Os três testes das retros da 5 e da 6: ordem da trava, ordem da suíte e o `de` do clique'
type: 'chore'
created: '2026-09-25'
status: 'done'
route: 'one-shot'
---

# Os três testes das retros da 5 e da 6: ordem da trava, ordem da suíte e o `de` do clique

## Intent

**Problem:** Três proteções não tinham teste que morresse quando elas sumissem. O `ORDER BY id` da trava do AD-5 (`epic-5-retro-item-26`). A cadeia de "por ÚLTIMO" entre as 6.4, 6.5 e 6.6 em `TestSessaoEProduto` (`epic-6-retro-item-32`, B7). E o `de` da transição administrativa capturado no clique (`epic-6-retro-item-33`, B8).

**Approach:**
- **Item 26:** um subteste novo põe o Produto de menor id depois do de maior id na tabela, proíbe índice na segunda transação e fecha o ciclo com a primeira. Sem o `ORDER BY p.id`, o Postgres acusa `40P01`. Mutação conferida.
- **Item 32:** a 6.6 desarma, num `t.Cleanup`, o Pedido que falha. As 6.4 e 6.5 já não dependiam da posição, e os comentários diziam o contrário. Dois reordenamentos temporários ficaram verdes: (a) 6.6/6.5/6.4 antes da 5.7; (b) a simulação da 1.8 depois da 6.6. A contraprova falhou com `23514`: o experimento (b) sem o `Cleanup`.
- **Item 33:** uma guarda de fonte no molde das que já existem em `pedido.test.mjs`. Três mutações a derrubam: o corpo lendo `pedido.status`, a captura movida para o confirmar e o clique sem o `de`.

## Suggested Review Order

**A ordem da trava (AD-4, AD-5)**

- Ponto de entrada: a inversão física e o ciclo que só fecha sem o `ORDER BY`.
  [`concorrencia_test.go:196`](../../api/concorrencia_test.go#L196)

- A espera é a da segunda na trava da primeira, pelo pid, e não qualquer uma.
  [`concorrencia_test.go:252`](../../api/concorrencia_test.go#L252)

- O pedido do segundo Produto pela primeira é onde a mutação vira impasse.
  [`concorrencia_test.go:269`](../../api/concorrencia_test.go#L269)

**A ordem da suíte (B7)**

- O Pedido que falha é desarmado com o Estoque da própria Reserva.
  [`simulacao_test.go:266`](../../api/simulacao_test.go#L266)

- Os comentários "por ÚLTIMO" reescritos para o que o código garante.
  [`sessao_test.go:566`](../../api/sessao_test.go#L566)

- 6.4: os Pedidos nascem logo antes da leitura, em qualquer posição.
  [`pedido_admin_test.go:45`](../../api/pedido_admin_test.go#L45)

**O `de` do clique (B8)**

- A guarda de fonte: `pedido.status` só no clique, e o POST sai do destino guardado.
  [`pedido.test.mjs:656`](../../web/scripts/pedido.test.mjs#L656)
