# Epic 5 Context: Checkout e pagamento assíncrono

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

O Comprador escolhe o Endereço, vê o Frete e o total discriminados, confirma — e o Pedido nasce em `AGUARDANDO_PAGAMENTO` com Reserva de Estoque atômica sobre todos os Itens. O resultado do pagamento chega **depois**, por webhook, fora da requisição do checkout. Recusa, nova Tentativa e expiração são caminhos de primeira classe, não erros. A épica abriu pela máquina de estados fechada e testada — o risco nº 1 do projeto era ela atrasar tudo — e é o que separa a maquete do sistema. A Reserva atômica, a expiração da Tentativa e o teste de concorrência **nunca são cortados**, por mais que nenhum apareça numa tela.

## Stories

- Story 5.1: A máquina de estados do Pedido, completa e com testes ✅ `done`
- Story 5.2: Seleção de Endereço no checkout ✅ em `main`, aguardando leitura humana
- Story 5.3: Cálculo do Frete ✅ em `main`, aguardando leitura humana
- Story 5.4: Revisão do Pedido ← próxima
- Story 5.5: Entrada no checkout — revalidação e confirmação do preço visto 🔒
- Story 5.6: Criação do Pedido com Reserva de Estoque atômica 🔒
- Story 5.7: Consistência de Estoque sob concorrência 🔒
- Story 5.8: Tentativa de Pagamento e a tela do Pedido em processamento
- Story 5.9: Confirmação aprovada por webhook
- Story 5.10: Pagamento recusado e nova Tentativa
- Story 5.11: Expiração da Tentativa de Pagamento 🔒

## Requirements & Constraints

- Todo Pedido tem um Status e só percorre as transições declaradas; fora delas é erro explícito, **seja qual for o ator**. O termo do glossário é `SEPARANDO`, nunca `EM_SEPARACAO`.
- Toda transição registra instante, estado anterior, novo, ator e motivo; o histórico é consultável e **imutável**. O conteúdo do Pedido — Itens, preço praticado, Vendedor de cada Item, Endereço, Frete, total — é escrito uma vez, na criação, e nunca mais.
- A Reserva nasce com o Pedido, tudo ou nada; é liberada na recusa, na expiração e no cancelamento, e **consolida** (baixa o Estoque total) em `SEPARANDO → ENVIADO`. Nenhuma outra transição mexe no Estoque total.
- Com N criações paralelas na última unidade, `pedidos_criados == estoque_inicial`. Confirmar duas vezes cria **um** Pedido; a mesma confirmação duas vezes produz **um** efeito.
- O Provedor Simulado decide pelos centavos do total só na primeira Tentativa; depois aprova. Teto de 3 Tentativas, sem superfície de operador, e **nenhum dado de cartão em lugar nenhum** — nem pedido, nem gerado, nem guardado.
- Tentativa sem confirmação no prazo (15 min padrão, 60 s em demonstração) vai a `PAGAMENTO_RECUSADO` com `motivo = TEMPO_ESGOTADO` e libera a Reserva. O prazo deriva do histórico, nunca de temporizador em memória.
- Frete discriminado do subtotal, por faixa de CEP → região → valor, com CEP fora de faixa na região padrão e resultado sempre igual para o mesmo CEP e subtotal. Carrinho vazio nunca chega à Revisão, e voltar ao Carrinho não perde o Endereço escolhido.
- Todo FR do fluxo busca → Carrinho → checkout → Pedido tem ao menos um teste que falha se a consequência declarada quebrar.

## Technical Decisions

- **A transição é o único ponto de mutação do Pedido (AD-3).** `Transicionar` valida contra a tabela de nove linhas, com o ator de cada uma, **antes** de tocar o banco; o `UPDATE` é compare-and-swap sobre o Status esperado e zero linhas é corrida perdida.
- **O efeito sobre o Estoque mora dentro de `Transicionar`**, não em quem chama: CAS, histórico e então liberar/reservar/consolidar, na mesma transação — as estórias seguintes não os invocam por conta própria.
- **Três recusas nomeadas, nunca fundidas, todas 409 com códigos distintos:** corrida perdida, transição fora da tabela (carregando as permitidas para aquele ator) e cancelamento fora da janela, este reclassificado pela releitura do Status.
- **Imutabilidade é gatilho de banco.** A linha do Pedido só aceita `UPDATE` de `status`; Itens e histórico recusam `UPDATE` e `DELETE`. As colunas de Frete, Endereço e total que a criação acrescentar nascem protegidas: só o `INSERT` as escreve.
- **Uma transação por caso de uso (AD-4)**, passada como parâmetro e nunca pelo `context`, em `READ COMMITTED`. Ordem de bloqueio global: linha do Pedido → Produtos com `ORDER BY id FOR UPDATE` → **só então** a soma das Reservas ativas (AD-5). A borda HTTP repete uma vez, e só uma, em deadlock ou falha de serialização.
- **O Catálogo é dono do Estoque (AD-5, AD-19)**; `pedido` nunca toca tabela de Estoque. Liberar sem Reserva ativa devolve `nil`, consolidar repetido é no-op, e reservar Produto invisível falha como Estoque insuficiente.
- **Dinheiro é inteiro de centavos ponta a ponta (AD-9)**, com `CHECK` amarrando total = subtotal + frete, sem divisão no caminho monetário e sem soma no navegador.
- **A Regra de Frete é dado e mora em `pedido` (AD-17):** tabela de faixas semeada **na própria migração** (não na semente do Catálogo, que roda uma vez por marcador de versão), mais uma função pura. Valores por região em reais inteiros — múltiplos de 100 por `CHECK`, porque os centavos do total são o que aciona o desfecho do Provedor Simulado. Região padrão é uma linha única, e tabela sem ela é erro nomeado, **nunca Frete zero**; faixas sobrepostas são recusadas pelo banco. Isenção é `subtotal >= limiar` vindo da configuração, o mesmo ponto em que o Carrinho para de dizer quanto falta. `carrinho` não calcula Frete.
- **A cotação é leitura de `pedido`**, por Endereço escolhido, com o dono no `WHERE` e o total somado no Go; Endereço alheio, inexistente e malformado dão a mesma resposta. Congelar o Frete é da criação.
- **Preço visto:** uma única função do Carrinho o atualiza, e só `pedido` a chama, na entrada do checkout, **depois** de reportar a diferença. A criação recalcula o total sob o mesmo bloqueio da Reserva e recusa com `TOTAL_DIVERGENTE`.
- **Criar Pedido é idempotente (AD-7):** chave mais digest do corpo; mesmo corpo devolve o Pedido original, corpo diferente é 409, e a violação de unicidade do banco nunca chega ao navegador.
- **A varredura é o único motor do tempo (AD-6):** aplicar confirmações → expirar Tentativas (lugar reservado, é da 5.11) → simular entrega → emitir confirmações devidas, com candidatos lidos fora da transação e reconferidos dentro, uma transação por Pedido e `FOR UPDATE SKIP LOCKED`.
- **Webhook com inbox (AD-7):** restrição única sobre a chave, `pagamento` não chama `pedido`, e a confirmação só se aplica se for da Tentativa corrente **e** o Pedido estiver em `AGUARDANDO_PAGAMENTO`; fora disso vira sinal de não-aplicável, sem perda silenciosa.
- **A resposta carrega tudo que a tela deriva (AD-18):** Status, tentativas restantes, disponibilidade por Item, `terminal` (só `ENTREGUE` ou `CANCELADO`), expiração como instante absoluto em RFC 3339 e o histórico. Nenhum campo nomeia a Reserva ao Comprador.
- Migrações versionadas por timestamp, sqlc regenerado após mudar SQL, sentinela nascendo no módulo e virando HTTP num ponto único (AD-14). Testes: máquina em memória; PostgreSQL real onde a transação é o objeto do teste.

## UX & Interaction Patterns

- Checkout em **dois passos, Endereço → Revisão**; Frete e pagamento são blocos da Revisão, porque nenhum tem dado a pedir. De 360 a 639 px, um passo por tela; o cadastro de Endereço no checkout é `Dialog` de primeiro nível.
- A Revisão mostra cada Item com preço unitário e subtotal, mais Endereço, Frete e total, e **declara a forma de pagamento sem coletar nada**. Pergunta o Frete ao servidor **a cada abertura** e não guarda cotação no navegador — é isso que faz trocar o Endereço recalcular o Frete. O navegador guarda só o identificador do Endereço escolhido.
- O botão Confirmar Pedido é a **única aparição do laranja no fluxo inteiro**, desabilita durante o envio e não admite duplo disparo. A chave de idempotência é gerada pelo navegador ao entrar na Revisão e guardada em `sessionStorage`, para sobreviver ao recarregamento que ela existe para proteger.
- **O Pedido em processamento é estado próprio**, não modal nem spinner: número, Status e tempo restante, com a ação persistente "Ver o Pedido em Meus pedidos" em todos os estados. Consulta a cada 3 s enquanto `AGUARDANDO_PAGAMENTO` e a cada 10 s depois, enquanto não for terminal. `PAGO` vira o Detalhe no mesmo lugar; recusa e expiração trocam o relógio pelo motivo e pela ação de tentar de novo, na mesma superfície.
- As três recusas têm tratamentos distintos e nenhuma cai no aviso genérico de erro: a corrida é informativa e recarrega o estado real, a inválida é destrutiva, e a fora da janela explica por que o cancelamento já não é possível.
- Verde só para Estoque disponível e `ENTREGUE`; laranja uma vez por fluxo, no passo irreversível.

## Cross-Story Dependencies

- A 5.1 é a fundação: toda outra estória da épica — e a Épica 6 inteira — passa por `Transicionar`, pela tabela com ator e pelas três recusas.
- A 5.2 entregou o passo Endereço e a escolha guardada no navegador; a 5.3, a Regra de Frete e a cotação. A 5.4 **substitui o esboço da Revisão** e herda três dívidas nomeadas: manter a pergunta do Frete a cada abertura sem guardar cotação, recusar Carrinho vazio na Revisão e apagar a escolha de Endereço ao confirmar. Falta também o teste de tela da troca de Endereço — a metade de servidor já está provada.
- A 5.5 depende da revalidação do Carrinho da Épica 4, cria a confirmação do preço visto e fecha o furo de quem digita a URL da Revisão por cima do avanço bloqueado.
- A 5.6 congela Frete, Endereço e total em colunas novas, esvazia o Carrinho dentro da mesma transação e traz a idempotência da criação. A 5.7 testa a 5.6 sob concorrência.
- As 5.8 → 5.9 → 5.10 → 5.11 estendem a Tentativa e a varredura que a Épica 1 já entregou; o teto de Tentativas é de `pagamento`, e a nova Reserva da nova Tentativa já sai de `Transicionar`.
- A sinalização de pagamento aprovado sobre Pedido cancelado é registrada na 5.9 e exibida na 6.5.
