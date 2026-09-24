# Epic 6 Context: Pedidos — acompanhamento, cancelamento e operação

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

A Épica 5 fechou a máquina de estados, e esta épica lhe dá leitores e operadores. O Comprador vê o histórico dos seus Pedidos, o mais recente no topo, e abre o Detalhe com o **preço praticado** (nunca o de hoje) e a linha do tempo das transições. Também cancela antes do envio e vê a unidade voltar à prateleira no mesmo instante. O Administrador filtra a lista e avança o Pedido pelas três transições que lhe cabem, e só por elas. Ele também vê o pagamento aprovado sobre Pedido cancelado, que antes só ia ao log. A simulação de entrega leva o Pedido sozinha até `ENTREGUE` durante a apresentação. Os sete Status ganham uma única forma de exibição, igual nas três superfícies. São quatro ângulos da mesma máquina: nenhuma superfície escreve Status por conta própria, e todas passam por `Transicionar`. A épica realiza a UJ-2, a UJ-3 e a UJ-4.

## Stories

- Story 6.1: Meus pedidos
- Story 6.2: Detalhe do Pedido
- Story 6.3: Cancelamento pelo Comprador
- Story 6.4: Painel de Pedidos do Administrador
- Story 6.5: Pagamento aprovado sobre Pedido cancelado, visível ao Administrador
- Story 6.6: Simulação de entrega
- Story 6.7: A forma de exibição dos sete Status

## Requirements & Constraints

- **Lista do Comprador.** Traz número, data, total e Status, os mais recentes primeiro, e **só** os Pedidos do Comprador autenticado. A posse é verificada na mesma consulta que carrega o dado. Sem Pedido, mostra um estado vazio com saída única para a Vitrine. Usa o envelope único de listagem, com desempate terminando em `id`.
- **Detalhe.** Mostra os Itens com o preço praticado congelado, o Endereço congelado, o Frete, o total e a linha do tempo com data e hora. A sequência de eventos que levou ao estado atual se reconstrói a partir do número do Pedido. A linha do tempo nunca mostra etapa futura especulativa.
- **Autorização.** O Pedido de outro Comprador devolve o **mesmo erro de inexistente**, inclusive por chamada direta. Não existe tela de acesso negado.
- **Cancelamento.** Vale em `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO` e `SEPARANDO`, com confirmação explícita que nomeia o Pedido. As unidades voltam ao Estoque disponível, visíveis na Vitrine no mesmo instante. Em `ENVIADO` ou `ENTREGUE` é recusado, inclusive pela API. Sobre `CANCELADO` não produz efeito nem erro. Não existe devolução nem reembolso.
- **Painel do Administrador.** Filtra por Status e ordena por data (padrão: mais recentes), sem busca por número. Tem exatamente três transições: `PAGO → SEPARANDO`, `SEPARANDO → ENVIADO` e `ENVIADO → ENTREGUE`. **O Administrador não cancela Pedido.** Uma transição inválida falha alto e informa as permitidas. O Detalhe administrativo mostra o Comprador e o Vendedor de cada Item.
- **Pagamento aprovado sobre Pedido cancelado.** O Status do Pedido não muda. O registro existe para que a fase 2 faça o estorno automático sem arqueologia de log.
- **Simulação de entrega.** Vai de `PAGO` a `ENTREGUE` pelas mesmas transições da tabela, sem caminho paralelo. O intervalo é configurável (24 h por padrão, 30 s em demonstração) e a simulação desliga por configuração. Nunca move `CANCELADO` nem `AGUARDANDO_PAGAMENTO`. É o 4º a cair na ordem de corte: sem ela, o Administrador avança o Pedido à mão.
- O total exibido é a soma exata das parcelas, ao centavo. As telas cumprem WCAG 2.2 AA e funcionam de 360 a 1440 px sem rolagem horizontal na página.

## Technical Decisions

- **`Transicionar` é o único ponto que muda Status.** Nenhum `UPDATE … SET status` novo. A tabela exaustiva, com o ator declarado, é consultada antes de tocar o banco. O efeito sobre o Estoque mora dentro da transição:
  - `Liberar` roda incondicionalmente em todo cancelamento e em toda recusa. Sem Reserva ativa, devolve `nil`.
  - `Consolidar` roda só em `SEPARANDO → ENVIADO`. Sobre Reserva já consolidada é no-op.
  - Nenhuma outra transição altera o Estoque total. Quem chama não repete o efeito.
- **As três recusas nunca se fundem.** As três saem em 409, cada uma com código próprio, e `dados` é mapa, nunca struct:
  - `ESTADO_JA_AVANCADO` leva o Status atual em `dados.status`.
  - `TRANSICAO_INVALIDA` leva `dados.permitidas` do ator.
  - `FORA_DA_JANELA_DE_CANCELAMENTO` leva `dados.status`.
- **Toda escrita de Status que precisa de desfecho lê o Pedido travado antes do CAS.** A trava é um `FOR UPDATE` que espera, e não o `SKIP LOCKED` da varredura. Com a linha presa, o Status lido é o que o CAS vê. A ordem das travas é Pedido primeiro, Produtos dentro do efeito. São dois casos:
  - **Comprador:** a trava tem o dono no `WHERE`. `POST /api/v1/pedidos/{id}/cancelamento`, sem corpo, devolve 200 também sobre `CANCELADO`, sem linha nova no histórico e sem tocar em Reserva. O duplo clique vira dois 200 e uma linha só. O cancelamento que espera uma aprovação ainda não comitada cancela o `PAGO`.
  - **Administrador:** a trava não tem dono. Ela dá o 404 e o `dados.status`.
- **O `de` da transição administrativa vem da tela.** `POST /api/v1/admin/pedidos/{id}/transicoes` recebe `{de, para}`. O `de` é o Status que o Administrador via quando clicou, e é a chave do CAS: é ele que separa "já avançou" de "transição inválida". Se o `de` fosse lido no servidor, a colisão com a simulação sairia como transição inválida, uma causa falsa. `{para: CANCELADO}` sai `TRANSICAO_INVALIDA` porque a tabela não tem essa linha para o Administrador, e não por um `if`.
- **Nenhuma regra de Status no Node.** Cada decisão chega pronta do Go:
  - A janela de cancelamento chega em `pode_cancelar`, tirado de `Permitidas(status, COMPRADOR)`.
  - As transições do Administrador chegam em `permitidas`, tanto na linha da Tabela quanto no Detalhe.
  - `terminal` vem de `EstadoTerminal`, a única declaração de terminalidade: `ENTREGUE` ou `CANCELADO`.
  - O filtro valida contra `pedido.Statuses`.
  - Nenhum Status nem transição é copiado em JavaScript.
- **A resposta traz, numa só chamada, tudo o que a tela deriva.** Isso inclui `status`, `tentativas_restantes`, `disponivel` por Item, `expira_em`, `historico` e `pode_cancelar`, com instantes em RFC 3339. A consulta periódica fica em três lugares:
  - a tela do Pedido, a 3 s em `AGUARDANDO_PAGAMENTO`;
  - a mesma tela, a 10 s depois, enquanto o Pedido não for terminal;
  - a Tabela administrativa, a 10 s enquanto listar Pedido não terminal.

  Depois de toda ação, com sucesso ou com recusa, a tela **relê a superfície** em vez de montar o estado localmente.
- **O Detalhe administrativo é uma leitura própria.** Não é a leitura do Comprador com o dono afrouxado. Não carrega `expira_em`, `tentativas_restantes`, `pode_cancelar` nem `disponivel`. O Comprador chega pela porta de `identidade`, nunca por JOIN entre schemas. O `vendedor_nome` já vem congelado no Item.
- **A data do Pedido é o `min(ocorrido_em)` do histórico**, sem coluna `criado_em`. As duas ordenações usam essa data e desempatam em `id`. A contagem repete o `WHERE` do filtro palavra por palavra.
- **`pagamento_aprovado_sobre_cancelado` é derivado na leitura.** Não existe coluna, Status nem migração para ele.
  - **A condição.** O sinal acende quando o Pedido está `CANCELADO` **e** uma Tentativa de Pagamento dele tem confirmação `APROVADO` em `NAO_APLICAVEL_SINALIZADA`.
  - **A leitura, em lote.** A porta `pagamento.PedidosComAprovacaoSinalizada` responde para vários Pedidos de uma vez. A junção com o Status acontece em Go, em `pedido`, sem SQL entre schemas. A Tabela faz uma consulta por página, só com os ids `CANCELADO`. O Detalhe lê dentro da transação `REPEATABLE READ` da rota.
  - **O escopo.** A Tentativa superada conta. `RECUSADO` não conta. O Pedido cancelado depois de `PAGO` casa de propósito, porque é cobrança duplicada. A Tentativa superada aprovada com o Pedido ativo e a aprovação sobre Pedido expirado ficam **fora**: com o Provedor Simulado nenhuma das duas acontece, e hoje as duas só chegam ao `WarnContext`.
  - **Onde o campo aparece.** Está sempre presente nas duas respostas administrativas, `false` quando não se aplica, e nunca aparece na resposta do Comprador. O 200 da transição administrativa o leva `false` sem ler a inbox, porque o destino do Administrador nunca é `CANCELADO`.
- **A varredura é o único motor do tempo.** O tique roda `aplicar → expirar → simular → emitir`.
  - **Casca comum.** A simulação divide a casca `avancarPeloTempo` com a expiração e deriva o avanço do histórico (`max(ocorrido_em)` anterior a `agora - AZAMON_ENTREGA_INTERVALO`), então reiniciar retoma de onde parou.
  - **Um Pedido por transação.** Os candidatos são lidos fora da transação e reconferidos dentro dela, travados com `FOR UPDATE SKIP LOCKED`. Um Pedido que falha não derruba os outros, e `ErrEstadoJaAvancado` é registrado em nível informativo.
  - **Configuração.** O ator no histórico é `SIMULACAO` (o `simulacao-entrega` da 1.8 só sobrevive no backfill da migração da 5.1). Um mapa único de três passagens gera tanto os candidatos quanto o próximo Status. O interruptor é `AZAMON_ENTREGA_SIMULACAO_ATIVA` e cobre só a simulação. A expiração não tem interruptor.
- O histórico é imutável por gatilho do banco e registra anterior, novo, ator, motivo e instante. As telas exibem o `numero` curto do Pedido. Nenhum dado pessoal vai ao log. Cada caso de uso tem uma transação, e o dinheiro é `int64` em centavos.

## UX & Interaction Patterns

- **Os sete Status têm uma forma de exibição fechada, um-para-um.** O mesmo componente, texto e cor aparecem em Meus pedidos, no Detalhe e na Tabela do Administrador:

  | Status | Texto exibido |
  |---|---|
  | `AGUARDANDO_PAGAMENTO` | "Aguardando pagamento" |
  | `PAGAMENTO_RECUSADO` | "Pagamento recusado" |
  | `PAGO` | "Pago" |
  | `SEPARANDO` | "Separando" |
  | `ENVIADO` | "Enviado" |
  | `ENTREGUE` | "Entregue" |
  | `CANCELADO` | "Cancelado" |

  Nunca aparece o identificador cru nem ícone. O rótulo já é único, em `rotuloDoStatus`, e à 6.7 resta o selo: `Badge` em `rounded.full`. O selo usa `secondary` nos Status de progresso e `destructive` em `PAGAMENTO_RECUSADO` e `CANCELADO`. O verde aparece **só** em `ENTREGUE`.
- **A ação vem da resposta, e não do Status sozinho.** Quando o estado não permite a ação, o botão fica **ausente, e não desabilitado**, e uma frase na região viva do Status explica o motivo. Durante uma operação em curso, o botão continua na tela, desabilitado e com progresso.
- **"Cancelar Pedido"** é `outline` e fica no Card do topo. Ele abre um `Dialog` controlado, de primeiro nível, que nomeia o Pedido pelo número. O `Dialog` não fecha durante o envio, desiste em 30 s e fecha **antes** de reler, para que a mudança seja anunciada. Nenhum texto nomeia a Reserva de Estoque nem promete reembolso.
- **As recusas nunca viram `Toast` genérico.** Cada uma tem a sua saída:
  - **Pedido já avançado:** `Alert` informativo, "Este Pedido já está em {estado}. A linha foi atualizada."
  - **Transição inválida:** `Alert` destrutivo, que nomeia a transição tentada e as permitidas.
  - **Cancelamento fora da janela:** o selo é atualizado, e a tela diz "Este Pedido foi enviado enquanto você estava nesta tela e não pode mais ser cancelado."
- **No Administrador, um único componente de ação serve a linha da Tabela e o Detalhe.** Ele reúne os botões e os dois `Alert`s. O desfecho não mora na linha, porque a releitura sob filtro desmonta a linha. As regiões vivas existem desde o primeiro render e só trocam de texto. A Tabela anuncia por uma região única, nomeando o Pedido que mudou. Avançar pede um `Dialog` com o estado de destino. O estado da Tabela vive na URL (`?status=&ordenacao=&pagina=`), e um valor fora das listas cai no padrão.
- **Pagamento aprovado sobre Pedido cancelado aparece de duas formas.** No Detalhe administrativo, um `Alert` neutro, persistente e sem botão de fechar fica abaixo do Status, com o texto "O Provedor de Pagamento aprovou uma Tentativa de Pagamento deste Pedido cancelado." Ele não diz "depois do cancelamento", porque a aprovação pode ter chegado antes. Na Tabela, um `Badge` `outline` "Pagamento aprovado" fica na célula do Status. Os dois usam texto, sem verde nem laranja, e não prometem estorno. "Persistente" quer dizer que o registro não some, e não que chega ao vivo: `CANCELADO` é terminal e a consulta periódica para.
- A mudança de Status é anunciada por `aria-live="polite"`. As listas carregam com `Skeleton`. A Tabela é desktop-first e rola dentro do próprio contêiner.

## Cross-Story Dependencies

- As estórias 6.1 a 6.5 estão em `main`. Elas deixaram prontos `pode_cancelar`, `permitidas` e `terminal` nas respostas, as duas leituras travadas antes do CAS, a Tabela e o Detalhe administrativos, e o sinal de aprovação sobre cancelado com as suas duas marcas na tela.
- **6.6.** Endurece a simulação que existe na varredura desde a Épica 1, com teste de cada condição da FR-33, e não só do caminho feliz. A colisão com o Administrador já está provada pelo lado dele: o Administrador trava e espera, e a simulação pula o Pedido. O desfecho dessa colisão é o `Alert` informativo, e não erro. A simulação não interage com o sinal da 6.5, porque nunca move `CANCELADO`. Um teste do binário que lê o Status corrente com a simulação a 200 ms é instável e está anotado nos adiados.
- **6.7.** Troca o rótulo pelo selo nas três superfícies, sem tirar da célula de Status da Tabela o marcador "Pagamento aprovado", que é outro `Badge`. Não toca em nenhuma regra.
- **Épica 7.** Depende desta para os roteiros de ensaio: o acompanhamento até `ENTREGUE`, o cancelamento com o Estoque voltando e a operação pelo painel. Nenhuma tela desta épica foi aberta num navegador até agora.
