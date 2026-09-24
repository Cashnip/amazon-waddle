# Epic 6 Context: Pedidos — acompanhamento, cancelamento e operação

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

A máquina de estados que a Épica 5 fechou ganha leitores e operadores. O Comprador vê o histórico dos seus Pedidos, o mais recente no topo. Abre o Detalhe com o **preço praticado**, nunca o preço de hoje, e com a linha do tempo das transições. E cancela antes do envio, vendo a unidade voltar à prateleira no mesmo instante. O Administrador filtra a lista e avança o Pedido pelas três transições que lhe cabem, e só por elas. A simulação de entrega leva o Pedido sozinha até `ENTREGUE` durante a apresentação, e os sete Status ganham uma única forma de exibição, igual nas três superfícies. São quatro ângulos da mesma máquina: nenhuma superfície escreve Status por conta própria, todas passam por `Transicionar`. A épica realiza a UJ-2, a UJ-3 e a UJ-4. Também dá leitor ao sinal de pagamento aprovado sobre Pedido cancelado, que a Épica 5 grava sem ninguém ver.

## Stories

- Story 6.1: Meus pedidos
- Story 6.2: Detalhe do Pedido
- Story 6.3: Cancelamento pelo Comprador
- Story 6.4: Painel de Pedidos do Administrador
- Story 6.5: Pagamento aprovado sobre Pedido cancelado, visível ao Administrador
- Story 6.6: Simulação de entrega
- Story 6.7: A forma de exibição dos sete Status

## Requirements & Constraints

- **Lista do Comprador (FR-29).** Número, data, total e Status, mais recentes primeiro, **só** do Comprador autenticado. A posse é verificada na mesma consulta que carrega o dado. Sem Pedido, estado vazio com saída única para a Vitrine. Envelope único de listagem, 20 por página, desempate terminando em `id`.
- **Detalhe (FR-30, NFR-9).** Itens com preço praticado congelado, Endereço congelado, Frete, total, e linha do tempo com data e hora. A sequência de eventos que levou ao estado atual é reconstruível a partir do número do Pedido. A linha do tempo nunca mostra etapa futura especulativa.
- **Autorização (NFR-6, AD-11).** Pedido de outro Comprador devolve o **mesmo erro de inexistente**, inclusive por chamada direta. Não existe tela de acesso negado.
- **Cancelamento (FR-31).** Permitido em `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO` e `SEPARANDO`, sempre com confirmação explícita que nomeia o Pedido. As unidades voltam ao Estoque disponível e isso se vê na Vitrine no mesmo instante. Em `ENVIADO`/`ENTREGUE` é recusado, inclusive pela API. Sobre Pedido já `CANCELADO` não produz efeito nem erro. Não há devolução nem reembolso.
- **Painel do Administrador (FR-32).** Filtrável por Status, ordenável por data (padrão: mais recentes), sem busca por número. Exatamente três transições: `PAGO → SEPARANDO`, `SEPARANDO → ENVIADO` e `ENVIADO → ENTREGUE`. **O Administrador não cancela Pedido.** Transição inválida é recusada informando as permitidas. O Detalhe administrativo mostra o Comprador e os Vendedores dos Itens.
- **Pagamento aprovado sobre cancelado (FR-26, AD-7, AD-18; invariante 7 do addendum §2).** A resposta administrativa carrega `pagamento_aprovado_sobre_cancelado`. O Detalhe mostra um `Alert` persistente e a Tabela ganha um marcador na linha. O Status do Pedido não muda. O registro existe para permitir o estorno automático da fase 2 sem arqueologia de log.
- **Simulação de entrega (FR-33).** Vai de `PAGO` a `ENTREGUE` pelas mesmas transições da tabela, com intervalo configurável (24 h padrão, 30 s em demonstração) e desligável por configuração. Nunca move `CANCELADO` nem `AGUARDANDO_PAGAMENTO`. É o 4º item da ordem de corte: sem ela, o Administrador avança manualmente.
- Total exibido = soma exata das parcelas, ao centavo (NFR-13). WCAG 2.2 AA (NFR-10, NFR-11), de 360 a 1440 px sem rolagem horizontal na página.

## Technical Decisions

- **`Transicionar` é o único ponto de mutação do Status (AD-3, AD-5).** Nenhum `UPDATE … SET status` novo. O ator é declarado (`COMPRADOR`, `ADMINISTRADOR`, `SIMULACAO`…), e a tabela exaustiva é consultada antes de tocar o banco. O efeito sobre o Estoque mora dentro da transição: `Liberar` incondicional em todo cancelamento e toda recusa (sem Reserva ativa devolve `nil`), `Consolidar` só em `SEPARANDO → ENVIADO`, e nenhuma outra transição altera o Estoque total. O cancelamento é uma linha da tabela com quatro origens.
- **As três recusas nunca se fundem.** `ErrEstadoJaAvancado` sai com o Status atual em `dados.status`. `ErrTransicaoInvalida` sai com `dados.permitidas` para aquele ator. `ErrForaDaJanelaDeCancelamento` também sai com `dados.status`. As três saem em 409, cada uma com código próprio. **O campo `dados` precisa ser mapa**: struct perde `permitidas` em silêncio e quebra o contrato "nunca ausente".
- **Rota do Comprador trava o Pedido antes do CAS.** O cancelamento lê o Pedido do dono com `FOR UPDATE` **que espera**, e não com o `SKIP LOCKED` da varredura. Com a linha presa, o Status lido é o que o CAS verá. Assim, o duplo clique vira dois 200 e uma linha de histórico, e o cancelamento que espera uma aprovação ainda não comitada cancela o `PAGO`. `CANCELADO` já gravado devolve 200 sem chamar `Transicionar`. A regra mora em `pedido`, e não na tela. A rota é `POST /api/v1/pedidos/{id}/cancelamento`, sem corpo, e não existe par administrativo.
- **Rota do Administrador: o `de` vem da tela.** `POST /api/v1/admin/pedidos/{id}/transicoes` recebe `de` e `para`, e `de` é o Status que o Administrador via quando clicou. É ele a chave do CAS, e é o que separa "já avançou" de "transição inválida". Lido no servidor, a colisão com a simulação sairia como transição inválida, uma causa falsa. A rota também trava o Pedido antes do CAS, sem dono no `WHERE`, e isso dá o 404 e o `dados.status`. `{para: CANCELADO}` sai `TRANSICAO_INVALIDA` porque a tabela não tem essa linha para o Administrador, e não por um `if`.
- **Nenhuma regra de Status no Node (AD-10).** A janela de cancelamento chega pronta em `pode_cancelar` (de `pedido.Permitidas(status, COMPRADOR)`). As transições do Administrador chegam em `permitidas` na linha da Tabela e no Detalhe. O filtro valida contra `pedido.Statuses`. `terminal` vem de `pedido.EstadoTerminal`, a única função de terminalidade (`ENTREGUE` ou `CANCELADO`). Nenhum Status é copiado em JavaScript.
- **A resposta carrega o que a tela deriva, numa chamada (AD-18).** A tripla (`status`, `tentativas_restantes`, `disponivel` por Item), `expira_em`, `historico` e `pode_cancelar`, com instantes absolutos em RFC 3339. A consulta periódica acontece só em três superfícies: a tela do Pedido a 3 s em `AGUARDANDO_PAGAMENTO` e a 10 s enquanto não for terminal, e a lista administrativa a 10 s enquanto houver Pedido não terminal. Depois de toda ação, sucesso ou recusa, a tela **relê a superfície** em vez de montar o estado localmente.
- **O Detalhe administrativo é uma leitura própria (AD-1, AD-2, AD-11).** Não é o do Comprador com o dono afrouxado. Não carrega `expira_em`, `tentativas_restantes`, `pode_cancelar` nem `disponivel`. O Comprador vem pela porta de `identidade`, nunca por JOIN entre schemas, e o `vendedor_nome` vem congelado no Item.
- **A data do Pedido é `min(ocorrido_em)` do histórico**, sem coluna `criado_em`. A ordenação da Tabela usa essa data, com desempate em `id`. A contagem repete o `WHERE` do filtro palavra por palavra.
- **O sinal de aprovado sobre cancelado é derivado na leitura (AD-7, AD-1)**, sem coluna nem Status novo. Ele junta, em Go e sem SQL cruzando schema, uma confirmação `APROVADO` em `NAO_APLICAVEL_SINALIZADA` na inbox de `pagamento` (lida por porta, e presa à Tentativa por `tentativa_id`) com `Status == CANCELADO`. O caso cancelado depois de `PAGO` casa de propósito, porque é uma cobrança duplicada. A aprovação de Tentativa superada também conta. O `WarnContext` já cobre ainda a aprovação sobre Pedido expirado (`PAGAMENTO_RECUSADO` com `TEMPO_ESGOTADO`). O log não é caminho de leitura.
- **A varredura é o único motor do tempo (AD-6).** Ordem do tique: `aplicar → expirar → simular → emitir`. A simulação divide a casca com a expiração (`avancarPeloTempo`) e deriva o avanço do histórico, então reiniciar retoma de onde parou. Cada Pedido tem sua transação, com `FOR UPDATE SKIP LOCKED`. Um Pedido que falha não derruba os outros, e `ErrEstadoJaAvancado` é registrado e seguido em frente. O ator no histórico é `simulacao-entrega`.
- Histórico imutável por gatilho do banco (AD-15) (anterior, novo, ator, motivo, instante). O que as superfícies exibem é o `numero` curto. Nenhum dado pessoal em log. Uma transação por caso de uso, dinheiro em centavos `int64`.

## UX & Interaction Patterns

- **Os sete Status têm uma forma de exibição fechada e um-para-um**, com o mesmo componente, texto e cor em Meus pedidos, no Detalhe e na Tabela do Administrador: "Aguardando pagamento", "Pagamento recusado", "Pago", "Separando", "Enviado", "Entregue", "Cancelado". Nunca aparece o identificador cru nem ícone. O rótulo já está centralizado em `rotuloDoStatus`, e à 6.7 resta o selo. O selo é `Badge` em `rounded.full`: `secondary` para progresso, `destructive` para `PAGAMENTO_RECUSADO`/`CANCELADO`, e verde **só** em `ENTREGUE`.
- **A ação vem da resposta, não do Status sozinho.** Quando o estado não permite, o botão fica **ausente, não desabilitado**, com uma frase na região viva do Status. Durante a operação, o botão permanece desabilitado e com progresso.
- **"Cancelar Pedido"** é `outline` e fica no Card do topo. Abre um `Dialog` controlado, de primeiro nível, que nomeia o Pedido pelo número. O `Dialog` não fecha durante o envio, desiste em 30 s e fecha **antes** de reler, para que a mudança seja anunciada. Nenhum texto nomeia a Reserva de Estoque nem promete reembolso.
- **As recusas nunca viram `Toast` genérico.** Pedido já avançado mostra um `Alert` informativo: "Este Pedido já está em {estado}. A linha foi atualizada.". Transição inválida mostra um `Alert` destrutivo que nomeia a tentada e as permitidas. Cancelamento fora da janela atualiza o selo e diz: "Este Pedido foi enviado enquanto você estava nesta tela e não pode mais ser cancelado.".
- **No Administrador, um único componente de ação** (botões e os dois `Alert`s) serve a linha da Tabela e o Detalhe. O desfecho não mora na linha, porque a releitura sob filtro a desmonta. As regiões vivas existem desde o primeiro render e só trocam de texto, e a Tabela anuncia por uma única região, nomeando o Pedido que mudou. Avançar pede um `Dialog` com o estado destino. O estado da Tabela vive na URL (`?status=&ordenacao=&pagina=`).
- Mudança de Status é anunciada por `aria-live="polite"`. As listas carregam com `Skeleton`. A Tabela é desktop-first e rola dentro do próprio contêiner.

## Cross-Story Dependencies

- A 6.1 a 6.4 estão em `main`. Deixaram prontos `pode_cancelar`, `permitidas`, as travas antes do CAS e a Tabela e o Detalhe administrativos.
- **6.5:** o sinal gravado pela 5.9 é alcançável desde a 6.3. O `Alert` persistente entra no Detalhe administrativo e o marcador entra na linha da Tabela da 6.4, sem tela nova. Falta decidir se o sinal também cobre a Tentativa superada aprovada com o Pedido ainda ativo, e a aprovação sobre Pedido expirado, que hoje só vai ao log.
- **6.6:** endurece a simulação que a Épica 1 deixou na varredura. A colisão com a 6.4 está agendada dentro da apresentação, e o desfecho dela é o `Alert` informativo, não erro.
- **6.7:** fecha o selo sobre `rotuloDoStatus` nas três superfícies.
- A Épica 7 depende desta para os roteiros de ensaio: acompanhamento até `ENTREGUE`, cancelamento com o Estoque voltando, e operação pelo painel.
