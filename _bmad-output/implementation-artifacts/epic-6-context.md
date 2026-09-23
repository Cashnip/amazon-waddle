# Epic 6 Context: Pedidos — acompanhamento, cancelamento e operação

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

A máquina de estados que a Épica 5 fechou passa a ter leitores e operadores: o Comprador vê o histórico dos seus Pedidos com o mais recente no topo, abre o Detalhe com o **preço praticado** — nunca o preço de hoje — e a linha do tempo das transições, e cancela antes do envio vendo a unidade voltar à prateleira no mesmo instante; o Administrador filtra a lista e avança o Pedido pelas três transições que lhe cabem, e só por elas; a simulação de entrega leva o Pedido sozinho até `ENTREGUE` durante a apresentação; e os sete Status ganham uma única forma de exibição, igual nas três superfícies. Quatro ângulos da mesma máquina — nenhuma superfície desta épica escreve Status por conta própria, todas passam por `Transicionar`. Cobre FR-29 a FR-33, realiza a UJ-2, a UJ-3 e a UJ-4, e fecha a leitura do sinal de pagamento aprovado sobre Pedido cancelado que a Épica 5 grava sem ninguém ver.

## Stories

- Story 6.1: Meus pedidos
- Story 6.2: Detalhe do Pedido
- Story 6.3: Cancelamento pelo Comprador
- Story 6.4: Painel de Pedidos do Administrador
- Story 6.5: Pagamento aprovado sobre Pedido cancelado, visível ao Administrador
- Story 6.6: Simulação de entrega
- Story 6.7: A forma de exibição dos sete Status

## Requirements & Constraints

- **Lista do Comprador (FR-29).** Número, data, total e Status, mais recentes primeiro, **só** do Comprador autenticado, com a posse verificada no servidor na mesma consulta que carrega o dado. Comprador sem Pedido vê estado vazio com caminho único para a Vitrine. Paginação de 20 por página no envelope único de listagem, com desempate terminando em `id`.
- **Detalhe (FR-30, NFR-9).** Itens com preço praticado congelado, Endereço congelado, Frete, total e a linha do tempo das transições com data e hora. Dado um número de Pedido, a sequência de eventos que levou ao estado atual é reconstruível. A linha do tempo mostra o que aconteceu, nunca etapas futuras especulativas.
- **Autorização (NFR-6).** Pedido de outro Comprador recebe o **mesmo erro de inexistente**, inclusive por chamada direta ao identificador; não existe tela de acesso negado com dica.
- **Cancelamento (FR-31).** Permitido em `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO` e `SEPARANDO`, sempre com confirmação explícita que nomeia o Pedido. Ao cancelar, a Reserva de Estoque é liberada e as unidades voltam ao Estoque disponível, verificável na Vitrine imediatamente. Em `ENVIADO` ou `ENTREGUE` é recusado, inclusive por chamada direta à API; sobre Pedido já `CANCELADO` não produz efeito nem erro. Sem devolução e sem reembolso no MVP — cancelar é a única saída.
- **Painel do Administrador (FR-32).** Lista filtrável por Status e ordenável por data (padrão mais recentes), sem busca por número de Pedido. Exatamente três transições e mais nenhuma: `PAGO → SEPARANDO`, `SEPARANDO → ENVIADO`, `ENVIADO → ENTREGUE`. **O Administrador não cancela Pedido** — não existe rota administrativa de cancelamento. Transição inválida é recusada informando quais são permitidas a partir do estado atual. O Detalhe administrativo mostra o Comprador e os Vendedores dos Itens.
- **Simulação de entrega (FR-33).** Avança de `PAGO` a `ENTREGUE` pelas mesmas transições declaradas, com intervalo configurável (24 h padrão, 30 s em demonstração), desligável por configuração, e nunca movendo um `CANCELADO` nem um `AGUARDANDO_PAGAMENTO`. É o 4º item da ordem de corte: sem ela, o Administrador avança manualmente pela FR-32.
- **Dinheiro na tela (NFR-13).** O total exibido é a soma exata das parcelas exibidas, ao centavo, sem arredondamento na apresentação.
- **Acessibilidade (NFR-10, NFR-11).** WCAG 2.2 AA; de 360 a 1440 px sem rolagem horizontal na página e sem elemento inacessível.

## Technical Decisions

- **A transição é o único ponto de mutação do Pedido (AD-3).** Nenhum `UPDATE pedido SET status` em superfície nova: tudo por `Transicionar`, com o ator declarado (`COMPRADOR`, `ADMINISTRADOR`, `SIMULACAO`) e a tabela exaustiva consultada antes de tocar o banco. O efeito mora dentro da transição — `Liberar` incondicional no cancelamento, `Consolidar` só em `SEPARANDO → ENVIADO`, e nenhuma outra transição altera o Estoque total. Cancelamento é uma única linha com quatro origens.
- **As três recusas nomeadas não se fundem**, cada uma com tratamento próprio: corrida perdida (`ErrEstadoJaAvancado`), transição inválida (`ErrTransicaoInvalida`, cuja resposta carrega as transições permitidas para aquele ator em `dados.permitidas`) e fora da janela de cancelamento (`ErrForaDaJanelaDeCancelamento`, obtido pela releitura do Status quando o CAS de `→ CANCELADO` se perde).
- **A resposta carrega tudo que a tela deriva (AD-18).** Nenhuma tela faz mais de uma chamada para derivar uma ação: a resposta do Pedido traz a tripla completa — `status`, `tentativas_restantes`, `disponivel` por Item — mais `expira_em` e o `historico`. Todo instante é absoluto em RFC 3339, nunca duração. Consulta em intervalo existe em exatamente três superfícies: a tela do Pedido a cada 3 s enquanto `AGUARDANDO_PAGAMENTO`, a mesma a cada 10 s enquanto não for terminal, e a lista administrativa a cada 10 s enquanto listar Pedido não terminal. Terminal é `ENTREGUE` ou `CANCELADO`, por `pedido.EstadoTerminal` — a única função de terminalidade. Toda listagem usa o mesmo envelope.
- **A varredura é o único motor do tempo (AD-6).** A simulação é o terceiro passo da ordem normativa do tique e não usa temporizador em memória: o avanço deriva do histórico — tempo decorrido desde a última transição registrada —, então reiniciar o sistema retoma de onde parou. Uma transação por Pedido com `FOR UPDATE SKIP LOCKED`; um Pedido que falhe não derruba a varredura dos outros, e `ErrEstadoJaAvancado` é desfecho esperado, registrado e seguido em frente.
- **Rastreabilidade (AD-15).** A tabela de transições guarda estado anterior, novo, ator, motivo, instante e correlação — uma fonte, três usos: a linha do tempo do Detalhe, a leitura da varredura e o NFR-9. O Pedido tem `numero` curto e legível, separado do `uuid`, e é ele que as superfícies exibem. Nenhum dado pessoal entra em log.
- **O sinal de pagamento aprovado sobre Pedido cancelado é derivado na leitura**, não é coluna nem Status novo: a metade de `pagamento` (confirmação `APROVADO` em estado `NAO_APLICAVEL_SINALIZADA`) é lida por uma porta Go e juntada com `Status == CANCELADO`, sem SQL cruzando schema. O Status do Pedido não muda; é este registro que torna o estorno automático da fase 2 possível sem arqueologia de log.
- Uma transação por caso de uso, aberta e fechada na borda HTTP; sentinela do módulo traduzida para HTTP em ponto único; dinheiro em centavos `int64` ponta a ponta.

## UX & Interaction Patterns

- **Forma de exibição fechada e um-para-um** dos sete Status, no mesmo componente, texto e cor nas três superfícies (Meus pedidos, Detalhe do Pedido, Tabela do Administrador): "Aguardando pagamento", "Pagamento recusado", "Pago", "Separando", "Enviado", "Entregue", "Cancelado". Nunca o identificador cru, nunca ícone. Selo é `Badge` em `rounded.full` — um dos quatro usos declarados desse raio; progresso normal em `secondary`, `PAGAMENTO_RECUSADO` e `CANCELADO` em `destructive`, e o verde de disponibilidade **só** em `ENTREGUE`, jamais como sucesso de operação, confirmação de formulário ou aviso.
- **A ação disponível é derivada da tripla**, não do Status sozinho. Quando o estado não permite, o botão fica **ausente, não desabilitado**, com uma frase que explica o motivo; durante operação em curso é o contrário — permanece na tela, desabilitado e com progresso, para não abrir janela de duplo disparo no passo irreversível.
- **Cancelar abre `Dialog` de primeiro nível nomeando o Pedido**; ao confirmar, o selo muda e a linha do tempo ganha a transição. Nenhuma superfície nomeia a Reserva de Estoque ao Comprador — ele vê o efeito na Vitrine.
- **As recusas têm tratamentos distintos e nenhuma cai no aviso genérico.** Pedido que já avançou por outro ator é `Alert` **informativo** ("já está em {estado}; a linha foi atualizada") — dizer "transição inválida" aqui reportaria causa falsa. Transição inválida é `Alert` **destrutivo** nomeando a tentada e as permitidas. Cancelamento fora da janela atualiza o selo e a linha do tempo e explica que o Pedido foi enviado enquanto a tela estava aberta.
- Mudança de Status durante a consulta em intervalo é anunciada por `aria-live="polite"`. Listas (Meus pedidos e administrativas) carregam com `Skeleton`, nunca spinner centralizado — a lacuna da UX, que declarava `Skeleton` só para Vitrine, Resultados e Página de Produto, é fechada aqui. A Tabela de Pedidos é desktop-first e rola **dentro do próprio contêiner**; avançar estado pede confirmação em `Dialog` com o nome do estado destino. `mockups/detalhe-do-pedido.html` ilustra a linha do tempo e a ação derivada em `SEPARANDO` e `PAGAMENTO_RECUSADO`.

## Cross-Story Dependencies

- Tudo aqui colhe a 5.1: a tabela com ator, o CAS, as três recusas e a janela de cancelamento já existem. A 6.2 herda `pedido.Detalhar` da 5.8 (leitura que monta a resposta do AD-18 com `expira_em`, `tentativas_restantes`, `historico` e disponibilidade por Item) e a tela de acompanhamento que já consulta sozinha; falta o Detalhe pleno com Endereço e Frete e a paginação declarada da lista, que é decisão da 6.1.
- A 6.3 é pré-requisito real da 6.5: hoje nenhuma rota leva um Pedido a `CANCELADO`, então a corrida clássica da FR-26 não acontece no sistema rodando e o sinal gravado pela 5.9 não tem leitor. A 6.3 também precisa resolver a releitura de `CANCELADO` na corrida perdida — sem isso a tela informa recusa de um cancelamento que deu certo. A 6.5 decide junto se o sinal ao Administrador cobre também a Tentativa superada aprovada com o Pedido ainda ativo, hoje sinalizada calada.
- A 6.4 monta os botões a partir das transições permitidas que o erro carrega; o campo `dados` precisa ser mapa, porque struct perde `permitidas` em silêncio e quebra o contrato "nunca ausente".
- A 6.6 endurece o esqueleto de simulação que a Épica 1 deixou de pé na varredura: mesma ordem normativa do tique, mesmas transições da tabela, `Consolidar` só no envio. Ela e a 6.4 colidem por construção — a colisão está agendada dentro da apresentação —, e o desfecho dessa colisão é o `Alert` informativo, não erro.
- A 6.7 extrai os mapas de rótulo hoje duplicados à mão em duas telas para um ponto único testado; sem isso uma renomeação de Status deixa uma superfície mostrando o identificador cru.
- A Épica 7 depende desta para os três roteiros de ensaio: o acompanhamento até `ENTREGUE`, o cancelamento com o Estoque voltando e a operação pelo painel são o que a banca vê rodando.
