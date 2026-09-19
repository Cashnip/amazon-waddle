# Epic 5 Context: Checkout e pagamento assíncrono

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

O Comprador escolhe o Endereço, vê o Frete e o total, confirma — e o Pedido nasce em `AGUARDANDO_PAGAMENTO` com Reserva de Estoque atômica sobre todos os Itens. O resultado do pagamento chega **depois**, por webhook; recusa, nova Tentativa e expiração são caminhos de primeira classe, não erros. A épica abre pela máquina de estados do Pedido fechada e testada (o risco nº 1 do projeto é ela atrasar tudo), e é o que separa a maquete do sistema. FR-24, FR-34 e o teste de concorrência do NFR-7 **nunca são cortados**.

## Stories

- Story 5.1: A máquina de estados do Pedido, completa e com testes
- Story 5.2: Seleção de Endereço no checkout
- Story 5.3: Cálculo do Frete
- Story 5.4: Revisão do Pedido
- Story 5.5: Entrada no checkout — revalidação e confirmação do preço visto 🔒
- Story 5.6: Criação do Pedido com Reserva de Estoque atômica 🔒
- Story 5.7: Consistência de Estoque sob concorrência 🔒
- Story 5.8: Tentativa de Pagamento e a tela do Pedido em processamento
- Story 5.9: Confirmação aprovada por webhook
- Story 5.10: Pagamento recusado e nova Tentativa
- Story 5.11: Expiração da Tentativa de Pagamento 🔒

## Requirements & Constraints

- Todo Pedido tem exatamente um Status e só transita pelas transições declaradas; transição não declarada é recusada com erro explícito **seja qual for o ator** (Comprador, Administrador, simulação, Provedor, varredura).
- Toda transição registra instante, estado anterior, novo, quem a provocou e o motivo; o histórico é consultável e **imutável**.
- O conteúdo do Pedido (Itens, preço praticado, Vendedor de cada Item, Endereço, Frete, total) é escrito uma vez, na criação, e nunca mais.
- A Reserva nasce com o Pedido, tudo ou nada sobre todos os Itens; é liberada na recusa, na expiração e no cancelamento, e **consolida** (baixa o Estoque total) na passagem para `ENVIADO` — o ponto exato em que o cancelamento deixa de existir. Nenhuma outra transição altera o Estoque total.
- Cancelamento pelo Comprador é possível a partir de `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO` e da etapa de separação.
- N criações paralelas disputando a última unidade: `pedidos_criados == estoque_inicial`, provado com PostgreSQL real.
- Confirmar duas vezes cria **um** Pedido; a mesma confirmação de pagamento duas vezes produz **um** efeito.
- Provedor Simulado decide pelos centavos do total, só na primeira Tentativa; da segunda em diante aprova. Teto de 3 Tentativas por Pedido. Nenhum dado de cartão em lugar nenhum.
- Tentativa sem confirmação no prazo (15 min padrão, 60 s em demonstração) leva o Pedido a `PAGAMENTO_RECUSADO` com `motivo = TEMPO_ESGOTADO`, liberando a Reserva — derivado do histórico, nunca de temporizador em memória.
- Frete: faixa de CEP → região → valor, CEP fora de faixa cai na região padrão, limiar de isenção de configuração, mesmo CEP e mesmo valor dão sempre o mesmo Frete, discriminado do subtotal.
- Todo FR do fluxo busca → Carrinho → checkout → Pedido tem ao menos um teste que falha se a consequência declarada quebrar.

## Technical Decisions

- **Transição é o único ponto de mutação do Pedido:** `pedido.Transicionar(ctx, tx, pedidoID, esperado, novo, ator, motivo)`, compare-and-swap literal (`UPDATE … WHERE id = $1 AND status = $esperado`); zero linhas é `ErrEstadoJaAvancado`. Ordem fixa na transação: **transição primeiro, efeito sobre o Estoque depois**. A tabela de nove transições do AD-3 é exaustiva, e cada uma carrega seu efeito (`Reservar`, `Liberar`, `Consolidar`, `carrinho.Esvaziar` só na criação).
- **Três recusas nomeadas, nunca fundidas:** `ErrEstadoJaAvancado` (outro ator chegou antes — informativo), `ErrTransicaoInvalida` (fora da tabela — carrega `dados.permitidas`), `ErrForaDaJanelaDeCancelamento` (o cancelamento reclassifica a corrida que tirou o Pedido da janela).
- `pedido.EstadoTerminal(s Status) bool` é a única noção de terminal: `ENTREGUE` ou `CANCELADO`, e mais nada.
- **Uma transação por caso de uso**, passada como parâmetro `tx pgx.Tx` (nunca no `context`). `READ COMMITTED`. Ordem de bloqueio global: linha de `pedido` primeiro, depois `catalogo.produto` ordenado por `id`. `api/` repete uma vez, e só uma, em `40P01`/`40001`. Nenhuma chamada de rede dentro de transação aberta.
- **Catálogo é dono do Estoque:** `pedido` nunca toca tabela de Estoque. Funções de mutação idempotentes: `Liberar` sem Reserva ativa devolve `nil`, `Consolidar` repetido é no-op, `Reservar` vazio é no-op. `pedido` chama `Liberar` **incondicionalmente** em toda recusa e todo cancelamento. `Reservar` trava as linhas de Produto antes de somar as Reservas, e recusa Produto invisível com `ErrEstoqueInsuficiente`.
- **Varredura é o único motor do tempo:** o tique de 1 s em `cmd/azamon` só chama, nesta ordem: aplicar confirmações → expirar Tentativas → avançar a simulação de entrega → `pagamento.EmitirConfirmacoesDevidas`. Uma transação por Pedido, `FOR UPDATE SKIP LOCKED`, e `ErrEstadoJaAvancado` é desfecho esperado.
- **Webhook + inbox:** `pagamento.confirmacao_recebida` com restrição única sobre a chave; `pagamento` não chama `pedido`. Confirmação só se aplica se é da Tentativa corrente **e** o Pedido está em `AGUARDANDO_PAGAMENTO`; senão vira `NAO_APLICAVEL_SINALIZADA`. Criar Pedido guarda chave e digest do corpo: mesma chave + mesmo corpo devolve o Pedido original; corpo diferente é 409. `23505` nunca vai ao navegador.
- **Dinheiro:** `bigint`/`int64` de centavos, sufixo `_centavos`; `total_centavos` é coluna com `CHECK (total_centavos = subtotal_centavos + frete_centavos)`; nenhuma divisão no caminho monetário.
- **Frete é dado e mora no Pedido:** `pedido.faixa_frete` + função pura; `carrinho` não calcula Frete. `carrinho.ConfirmarPrecoVisto` é a única escritora de `preco_visto_centavos`, e só `pedido` a chama, na entrada do checkout, depois de reportar. A criação recalcula o total sob o mesmo bloqueio em que reserva e recusa com `TOTAL_DIVERGENTE`.
- **Resposta do Pedido carrega a tripla** `(status, tentativas_restantes, disponivel por Item)`, `expira_em` absoluto em RFC 3339, e o histórico; nenhum campo nomeia a Reserva ao Comprador. Consulta a cada 3 s enquanto `AGUARDANDO_PAGAMENTO`.
- Status e estados são `text` com `CHECK`, não enum do Postgres. Migração goose por timestamp; `sqlc generate` depois de mudar SQL. Erro sentinela no módulo, traduzido para HTTP só em `internal/plataforma/erro`, com envelope `{erro:{codigo,mensagem,dados,correlacao}}`.
- Testes: máquina em memória; `testcontainers-go` com PostgreSQL real onde a transação é o objeto do teste.

## UX & Interaction Patterns

- Checkout em dois passos: **Endereço → Revisão**; Frete e pagamento são blocos da Revisão. De 360 a 639 px, um passo por tela. Comprador sem Endereço cai direto no formulário de novo Endereço.
- O botão Confirmar Pedido é a única aparição do laranja no fluxo. A `Idempotency-Key` é gerada no navegador ao entrar na Revisão e guardada em `sessionStorage`.
- Tela do Pedido em processamento: estado próprio (não modal, não spinner), com número, Status e tempo restante; ação persistente "Ver o Pedido em Meus pedidos"; `PAGO` vira o Detalhe no lugar; recusa e expiração trocam o relógio pelo motivo e pela ação de tentar de novo na mesma superfície.
- As três recusas da máquina têm tratamentos distintos: corrida é informativa (recarrega o estado real), transição inválida é destrutiva, fora da janela explica por que não dá mais.

## Cross-Story Dependencies

- 5.1 é a fundação: toda outra estória da épica (e as Épicas 6 — cancelamento, painel do Administrador, simulação) passa por `Transicionar` e pelas três recusas.
- 5.3 (Frete) alimenta 5.4 (Revisão), 5.5 (recalcular na entrada) e 5.6 (congelar na criação).
- 5.5 depende da revalidação do Carrinho da 4.4 (`carrinho.bloqueioDe`, `PrecoMudou`) e cria `carrinho.ConfirmarPrecoVisto`.
- 5.6 cria `carrinho.Esvaziar` e a idempotência do `POST /api/v1/pedidos`; 5.7 testa a 5.6 sob concorrência.
- 5.8 → 5.9 → 5.10 → 5.11 estendem a Tentativa e a varredura já existentes da Épica 1; o teto de Tentativas é de `pagamento` (`ErrTetoDeTentativas`).
- A sinalização de pagamento aprovado sobre Pedido cancelado é registrada em 5.9 e exibida em 6.5.
