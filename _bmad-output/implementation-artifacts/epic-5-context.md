# Epic 5 Context: Checkout e pagamento assíncrono

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

O Comprador escolhe o Endereço, vê o Frete e o total, confirma, e o Pedido nasce em `AGUARDANDO_PAGAMENTO` com Reserva de Estoque atômica sobre todos os Itens. O resultado do pagamento chega **depois**, por webhook. Recusa, nova Tentativa e expiração são caminhos de primeira classe, não erros. A épica abre pela máquina de estados do Pedido fechada e testada (o risco nº 1 do projeto é ela atrasar tudo), e é o que separa a maquete do sistema. FR-24, FR-34 e o teste de concorrência do NFR-7 **nunca são cortados**.

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

- Todo Pedido tem exatamente um Status e só transita pelas transições declaradas. Uma transição não declarada é recusada com erro explícito **seja qual for o ator** (Comprador, Administrador, simulação, Provedor, varredura).
- Os Status são `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO`, `SEPARANDO`, `ENVIADO`, `ENTREGUE` e `CANCELADO`. O nome é `SEPARANDO`, que é o termo do glossário, e nunca `EM_SEPARACAO`.
- Toda transição registra o instante, o estado anterior e o novo, o ator e o motivo. O histórico é consultável e **imutável**.
- O conteúdo do Pedido (Itens, preço praticado, Vendedor de cada Item, Endereço, Frete, total) é escrito uma vez, na criação, e nunca mais.
- A Reserva nasce com o Pedido, tudo ou nada sobre todos os Itens. É liberada na recusa, na expiração e no cancelamento, e **consolida** (baixa o Estoque total) em `SEPARANDO → ENVIADO`, o ponto exato em que o cancelamento deixa de existir. Nenhuma outra transição altera o Estoque total.
- O Comprador pode cancelar a partir de `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO` e `SEPARANDO`.
- Com N criações paralelas disputando a última unidade, `pedidos_criados == estoque_inicial`, provado com PostgreSQL real.
- Confirmar duas vezes cria **um** Pedido, e a mesma confirmação de pagamento recebida duas vezes produz **um** efeito.
- O Provedor Simulado decide pelos centavos do total só na primeira Tentativa. Da segunda em diante, aprova. O teto é de 3 Tentativas por Pedido, e nenhum dado de cartão existe em lugar nenhum.
- Uma Tentativa sem confirmação no prazo (15 min por padrão, 60 s em demonstração) leva o Pedido a `PAGAMENTO_RECUSADO` com `motivo = TEMPO_ESGOTADO` e libera a Reserva. O prazo é derivado do histórico, nunca de um temporizador em memória.
- O Frete segue faixa de CEP → região → valor. Um CEP fora de faixa cai na região padrão, o limiar de isenção vem da configuração, o mesmo CEP com o mesmo valor dá sempre o mesmo Frete, e o Frete aparece discriminado do subtotal.
- Todo FR do fluxo busca → Carrinho → checkout → Pedido tem ao menos um teste que falha se a consequência declarada quebrar.

## Technical Decisions

- **A transição é o único ponto de mutação do Pedido.** `pedido.Transicionar(ctx, tx, pedidoID, esperado, novo, ator, motivo)` faz um compare-and-swap literal (`UPDATE … WHERE id = $1 AND status = $esperado`), e zero linhas afetadas é `ErrEstadoJaAvancado`.
- **A tabela do AD-3 já é dado, entregue na 5.1** (`internal/pedido/maquina.go`). São nove linhas: o nascimento é `INSERT` em `Criar`, fora do CAS, e o cancelamento é uma linha só com quatro origens. Cada linha diz qual ator pode provocá-la (`COMPRADOR`, `PROVEDOR`, `VARREDURA`, `ADMINISTRADOR`, `SIMULACAO`). A expiração exige `motivo = TEMPO_ESGOTADO`. `Transicionar` valida contra a tabela **antes** de tocar o banco.
- **O efeito sobre o Estoque mora dentro de `Transicionar`, e não em quem o chama.** Primeiro vêm o CAS e a linha do histórico, depois o efeito, tudo na mesma transação: `Liberar` em toda recusa e todo cancelamento, `Reservar` com os Itens do próprio Pedido na nova Tentativa, `Consolidar` em `SEPARANDO → ENVIADO`. As estórias seguintes não chamam esses efeitos por conta própria. `carrinho.Esvaziar` só acontece na criação.
- **Três recusas nomeadas, nunca fundidas, todas em 409 com códigos distintos.** `ErrEstadoJaAvancado` quer dizer que outro ator chegou antes, e é informativa. `TransicaoInvalida` é uma transição fora da tabela e carrega as permitidas para aquele ator, que `erro.Escrever` põe em `dados.permitidas`. `ErrForaDaJanelaDeCancelamento` sai quando um cancelamento perde o CAS: o Pedido é relido, e se quem chegou antes o tirou da janela a recusa é reclassificada assim.
- `pedido.EstadoTerminal(s)` é a única noção de terminal: `ENTREGUE` ou `CANCELADO`, e mais nada. A resposta carrega `terminal`, e o JavaScript não redeclara a regra.
- **A imutabilidade é garantida por gatilho no banco (erro 23001).** `pedido.pedido` recusa `UPDATE` de qualquer coluna que não seja `status`, e também `DELETE`. Por isso as colunas que a 5.3 e a 5.6 acrescentarem já nascem protegidas e só podem ser escritas no `INSERT`. `item_pedido` e `transicao_status` recusam `UPDATE` e `DELETE`. Uma migração que precise reescrever essas linhas desliga o gatilho dentro dela mesma. A coluna `correlacao` no histórico foi adiada: `pedido` não importa `plataforma`.
- **Uma transação por caso de uso**, passada como parâmetro `tx pgx.Tx` e nunca no `context`, em `READ COMMITTED`. A ordem de bloqueio é global: primeiro a linha do Pedido, depois `catalogo.produto` com `ORDER BY id FOR UPDATE`, e **só então** a soma das Reservas ativas. `api/` repete uma vez, e só uma, em `40P01`/`40001`. Nenhuma chamada de rede acontece com transação aberta.
- **O Catálogo é dono do Estoque**, e `pedido` nunca toca tabela de Estoque. As mutações são idempotentes: `Liberar` sem Reserva ativa devolve `nil`, `Consolidar` repetido é no-op e `Reservar` vazio é no-op. `Reservar` recusa Produto invisível com `ErrEstoqueInsuficiente`.
- **A varredura é o único motor do tempo.** O tique de 1 s segue a ordem aplicar confirmações → expirar Tentativas (o lugar da 5.11, hoje reservado em comentário) → simular entrega → `pagamento.EmitirConfirmacoesDevidas`. São candidatos lidos fora da transação e reconferidos dentro dela, com uma transação por Pedido e `FOR UPDATE SKIP LOCKED`. `ErrEstadoJaAvancado` é um desfecho esperado.
- **Webhook com inbox.** `pagamento.confirmacao_recebida` tem restrição única sobre a chave, e `pagamento` não chama `pedido`. Uma confirmação só se aplica se for da Tentativa corrente **e** o Pedido estiver em `AGUARDANDO_PAGAMENTO`. Fora disso, vira `NAO_APLICAVEL_SINALIZADA`.
- **Criar um Pedido é idempotente.** A criação guarda a chave e o digest do corpo. A mesma chave com o mesmo corpo devolve o Pedido original, e com um corpo diferente devolve 409. `23505` nunca chega ao navegador.
- **Dinheiro é `bigint`/`int64` de centavos**, com sufixo `_centavos`. Um `CHECK` garante `total_centavos = subtotal_centavos + frete_centavos`, e o caminho monetário não tem nenhuma divisão.
- **O Frete é dado e mora no módulo `pedido`**, como `pedido.faixa_frete` mais uma função pura. `carrinho` não calcula Frete.
- **Preço visto:** `carrinho.ConfirmarPrecoVisto` é a única escritora de `preco_visto_centavos`, e só `pedido` a chama, na entrada do checkout. A criação recalcula o total sob o mesmo bloqueio da Reserva e recusa com `TOTAL_DIVERGENTE`.
- **A resposta do Pedido carrega a tripla** `(status, tentativas_restantes, disponivel por Item)`, além do `expira_em` absoluto em RFC 3339 e do histórico. Nenhum campo nomeia a Reserva ao Comprador.
- Status e estados são `text` com `CHECK`, não enum. As migrações goose são versionadas por timestamp, e `sqlc generate` roda depois de mudar SQL. O erro sentinela nasce no módulo e só vira HTTP em `internal/plataforma/erro`, no envelope `{erro:{codigo,mensagem,dados,correlacao}}`.
- **Testes:** a máquina é testada em memória, e com `testcontainers-go` sobre PostgreSQL real onde a transação é o objeto do teste.

## UX & Interaction Patterns

- O checkout tem dois passos, **Endereço → Revisão**. Frete e pagamento são blocos da Revisão. De 360 a 639 px, cada passo ocupa uma tela. Um Comprador sem Endereço cai direto no formulário de novo Endereço.
- O botão Confirmar Pedido é a única aparição do laranja no fluxo. A `Idempotency-Key` é gerada no navegador ao entrar na Revisão e guardada em `sessionStorage`.
- **A tela do Pedido em processamento é um estado próprio**, não um modal nem um spinner. Mostra o número, o Status e o tempo restante, com a ação persistente "Ver o Pedido em Meus pedidos". A consulta acontece a cada 3 s enquanto `AGUARDANDO_PAGAMENTO`. `PAGO` vira o Detalhe no mesmo lugar. Na recusa ou na expiração, o relógio dá lugar ao motivo e à ação de tentar de novo, na mesma superfície.
- As três recusas da máquina têm tratamentos distintos. A corrida é informativa e recarrega o estado real. A transição inválida é destrutiva. A recusa fora da janela explica por que o cancelamento não é mais possível.

## Cross-Story Dependencies

- A 5.1 está feita e é a fundação. Toda outra estória da épica, e também a Épica 6 (cancelamento, painel do Administrador, simulação), passa por `Transicionar`, pela tabela com ator e pelas três recusas.
- A 5.3 (Frete) alimenta a 5.4 (Revisão), a 5.5 (recalcular na entrada) e a 5.6 (congelar na criação). As colunas novas de Frete e Endereço só podem ser escritas no `INSERT`, por causa do gatilho.
- A 5.5 depende da revalidação do Carrinho feita na 4.4 (`carrinho.bloqueioDe`, `PrecoMudou`) e cria `carrinho.ConfirmarPrecoVisto`.
- A 5.6 cria `carrinho.Esvaziar` e a idempotência de `POST /api/v1/pedidos`. A 5.7 testa a 5.6 sob concorrência.
- As estórias 5.8 → 5.9 → 5.10 → 5.11 estendem a Tentativa e a varredura que a Épica 1 já entregou. O teto de Tentativas é de `pagamento` (`ErrTetoDeTentativas`). Na 5.10, o `Reservar` da nova Tentativa já sai de `Transicionar`.
- A sinalização de pagamento aprovado sobre um Pedido cancelado é registrada na 5.9 e exibida na 6.5.
