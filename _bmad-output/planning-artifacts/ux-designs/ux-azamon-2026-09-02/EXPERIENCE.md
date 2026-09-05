---
name: Azamon
status: final
updated: 2026-09-04
sources:
  - ../../prds/prd-azamon-2026-08-14/prd.md
  - ../../prds/prd-azamon-2026-08-14/addendum.md
---

# Azamon — Espinha de Experiência

> Web responsiva de superfície única. shadcn/ui sobre Tailwind. `DESIGN.md` é a referência de identidade visual; esta espinha é o comportamento. Requisitos, jornadas e Arquitetura da Informação são **herdados por referência** do PRD, não duplicados aqui — esta espinha diz o que a interface faz, não o que o sistema precisa fazer.
>
> **As duas espinhas vencem em caso de conflito** com qualquer mock, wireframe ou material importado. O PRD vence sobre as espinhas em matéria de requisito; as espinhas vencem em matéria de design e comportamento.

## Foundation

Aplicação web responsiva única, executada localmente em contêineres durante a demonstração (§10 do PRD). Não há aplicativo móvel, não há instalação, não há publicação em nuvem.

shadcn/ui sobre Tailwind. A biblioteca faz a maior parte do trabalho; a disciplina é **respeitar os defaults exceto onde a camada de marca do `DESIGN.md` sobrescreve**. Acessibilidade de fábrica do shadcn é parte da razão da escolha — o piso AA deste documento seria caro demais construindo componente por componente.

Três áreas com fronteira dura, herdadas do §9 do PRD: **Público** (sem Sessão), **Comprador** (com Sessão) e **Administrador** (com Sessão). A área administrativa é separada da loja e não aparece para quem não é Administrador — nem como link desabilitado, nem como item cinza no menu.

**Os vinte termos do glossário do §3 são usados literalmente na interface.** A tela diz "Pedido", não "compra"; "Produto", não "item"; "Endereço", não "local de entrega". O glossário é vocabulário de produto, não jargão interno.

## Information Architecture

Herdada do §9 do PRD. A tabela abaixo acrescenta a camada de experiência: por onde se chega e o que a superfície resolve.

| Superfície | Área | Chega-se por | Resolve |
|---|---|---|---|
| Vitrine | Público | Wordmark / raiz | Primeira impressão; grade do Catálogo Semeado |
| Resultados de busca | Público | Busca global (toda tela) | UJ-1: termo + filtros + ordenação, estado na URL |
| Página de Produto | Público | Cartão de Produto / resultado | Decisão de compra; caixa de compra à direita |
| Login · Cadastro · Recuperação de senha | Público | Barra superior / porta de Comprador | Abre a Sessão |
| Carrinho | Comprador | Ícone na barra superior | Revisar Itens de Carrinho antes do compromisso |
| Checkout | Comprador | Botão no Carrinho | **Endereço → Revisão**; Frete e pagamento são blocos da Revisão |
| Pedido em processamento | Comprador | Confirmação do checkout | Sustenta a espera assíncrona do pagamento |
| Meus pedidos | Comprador | Menu da conta | UJ-2: lista com o mais recente no topo |
| Detalhe do Pedido | Comprador | Linha de Meus pedidos | UJ-2 e UJ-3: linha do tempo, preço praticado, cancelar |
| Meus endereços | Comprador | Menu da conta | Cadastrar, editar e remover Endereço (FR-5) |
| Perfil | Comprador | Menu da conta | Mínimo: e-mail em leitura, trocar senha via FR-3, sair — nenhuma FR pede mais |
| Vendedores · Categorias · Produtos | Administrador | Navegação administrativa | UJ-4: abastecer a loja |
| Pedidos (lista, detalhe, avançar) | Administrador | Navegação administrativa | UJ-4: operar a máquina de estados |

→ Referência de composição: [`mockups/resultados-busca.html`](mockups/resultados-busca.html) (grade, chips de filtro, ordenação, paginação) · [`mockups/pagina-de-produto.html`](mockups/pagina-de-produto.html) (Caixa de compra, preço, Estoque disponível) · [`mockups/detalhe-do-pedido.html`](mockups/detalhe-do-pedido.html) (linha do tempo e ação derivada, nos estados `SEPARANDO` e `PAGAMENTO_RECUSADO`). As outras dez superfícies da Arquitetura da Informação são construídas a partir das tabelas desta espinha, por escolha registrada — não por esquecimento.

A **busca global está em toda tela pública e de Comprador**, dentro da barra superior. É o caminho da UJ-1 e a decisão já está no §9; aqui ela é obrigação de layout, não sugestão.

`Dialog` empilha no máximo um nível. Confirmação de cancelamento de Pedido, criação de Endereço no checkout e confirmação de transição no painel do Administrador são todos `Dialog` de primeiro nível.

## Voice and Tone

Microcopy. A postura de marca vive em `DESIGN.md.Brand & Style`. Português do Brasil em toda a interface (§15 do PRD, questão 6 resolvida).

| Faça | Não faça |
|---|---|
| "Nenhum Produto encontrado para 'carregador tipo z'." | "Ops! Não achamos nada 😢" |
| "Limpar filtros" | "Tentar novamente" |
| "Pagamento recusado: saldo insuficiente." | "Algo deu errado. Tente mais tarde." |
| "Seu Carrinho está vazio." | "Seu carrinho está vazio! Que tal dar uma olhadinha?" |
| "Este Pedido não pode mais ser cancelado porque já foi enviado." | "Cancelamento indisponível." |
| "Restam 2 unidades." | "Últimas unidades! Corra!" |
| Nomear o motivo sempre que o sistema souber | Mensagem genérica quando o motivo existe |

**A única exceção, e ela é de segurança.** Na autenticação, nomear o motivo vaza a existência de conta. A FR-2 exige a **mesma** mensagem para e-mail inexistente e para senha errada, e o bloqueio por tentativas se aplica ao par (e-mail, origem) independentemente de a conta existir — senão a própria aparição da mensagem de bloqueio confirma a conta. Um desenvolvedor seguindo a regra geral escreve "E-mail não encontrado" e quebra o requisito; esta linha existe para impedir isso.

**Nada de urgência fabricada.** Sem contagem regressiva de escassez, sem "12 pessoas vendo agora", sem selo de oferta. O Catálogo Semeado é fictício e a banca sabe disso; teatro de urgência sobre dado inventado é a coisa que mais rápido faz a réplica parecer maquete.

**O único relógio honesto da interface** é o da expiração da Tentativa de Pagamento (FR-34), porque ele corresponde a um prazo real do sistema.

## A máquina de estados na interface

*Seção específica deste produto.* O §11 do PRD e o §2 do addendum tratam o **Status do Pedido** como a peça central do sistema — histórico, cancelamento, painel administrativo e simulação de entrega são a mesma máquina vista de quatro ângulos. A interface precisa refletir isso, ou vira quatro telas que não conversam.

**Forma de exibição.** O identificador da máquina de estados não vai cru para a tela. O mapeamento é fechado, um-para-um e igual nas três superfícies — não é tradução livre, é a única forma de exibição permitida, e por isso não viola a disciplina do glossário:

`AGUARDANDO_PAGAMENTO` → "Aguardando pagamento" · `PAGAMENTO_RECUSADO` → "Pagamento recusado" · `PAGO` → "Pago" · `SEPARANDO` → "Separando" · `ENVIADO` → "Enviado" · `ENTREGUE` → "Entregue" · `CANCELADO` → "Cancelado"

**A ação disponível é derivada, e a chave da derivação é uma tripla:** `(Status do Pedido, tentativas de pagamento restantes, disponibilidade dos Itens de Pedido)`. Só o Status não basta — a FR-27 esgota as tentativas em 3 (§7.1) e recusa a nova tentativa quando o Estoque disponível acabou nesse meio-tempo. Um `acoesPara(status)` de um argumento só renderiza "Tentar pagar de novo" para sempre.

**O Administrador não cancela Pedido.** A FR-31 é *cancelamento pelo Comprador*, a FR-32 enumera exaustivamente as três transições do Administrador e nenhuma delas é cancelar, e o addendum §2 registra o ator dessa transição como Comprador. Não existe rota administrativa de cancelamento.

| Status do Pedido | Comprador vê | Administrador vê |
|---|---|---|
| `AGUARDANDO_PAGAMENTO` | Relógio de expiração; "Cancelar Pedido" | Nenhuma ação |
| `PAGAMENTO_RECUSADO` | Motivo; "Tentar pagar de novo" com tentativas restantes, **enquanto houver tentativa e Estoque disponível**; "Cancelar Pedido" | Nenhuma ação |
| `PAGO` | "Cancelar Pedido" | "Avançar para Separando" |
| `SEPARANDO` | "Cancelar Pedido" | "Avançar para Enviado" |
| `ENVIADO` | Nenhuma ação; explicação de por que não dá mais para cancelar | "Avançar para Entregue" |
| `ENTREGUE` | Nenhuma ação | Nenhuma ação |
| `CANCELADO` | Nenhuma ação | Nenhuma ação, **salvo o marcador de pagamento aprovado sobre Pedido cancelado** (FR-26) |

**Botão ausente, não desabilitado — quando a causa é o estado.** Se o estado não permite a ação, o botão sai da tela e o detalhe explica o motivo em uma frase; botão cinza sem explicação transfere ao Comprador o trabalho de adivinhar a regra. **Durante uma operação em curso é o contrário:** o botão permanece na tela, desabilitado e com indicação de progresso, porque removê-lo produziria salto de layout e uma janela de duplo disparo justamente no passo irreversível.

## Dinheiro e Estoque na tela

*Seção específica deste produto.*

- **O total exibido é sempre a soma exata das parcelas exibidas** (NFR-13). Se a tela mostra subtotal, Frete e total, os três fecham ao centavo. Nenhum arredondamento acontece na camada de apresentação.
- **Preço praticado, nunca preço de hoje.** No Detalhe do Pedido e em Meus pedidos, o valor exibido vem do Item de Pedido congelado (glossário), não do Catálogo. Uma mudança de preço no Catálogo não pode alterar visualmente um Pedido existente — é o teste mais rápido de que o sistema não é maquete.
- **O Carrinho não congela preço nem reserva Estoque** (§4.4). A interface não pode sugerir que congela: sem "preço garantido", sem "reservado para você". A revalidação do Carrinho (FR-19) aparece como aviso na entrada do checkout quando preço ou disponibilidade mudaram desde a última visita.
- **Frete grátis acima de R$ 299,00** (§7.1) aparece no Carrinho como a distância que falta, em valor absoluto: "Faltam R$ 42,00 para o Frete grátis." Sem barra de progresso.
- **O número exibido é o Estoque disponível, nunca o Estoque.** A FR-11 define *Estoque disponível = Estoque total menos as Reservas de Estoque ativas*. São dois números diferentes, e exibir o total é o bug que a FR-11 existe para impedir: a tela promete "Restam 3 unidades" para um Produto com 3 unidades reservadas, e o Carrinho recusa logo em seguida pelo número que a página acabou de prometer. Vale nos três pontos em que um número aparece — Caixa de compra, mensagem de recusa e selo.
- **A composição monetária do Carrinho é subtotal e nada mais.** O Frete não aparece antes do passo Endereço, porque depende do CEP (FR-21) — mostrar um Frete estimado violaria a regra de que o total exibido é a soma exata das parcelas exibidas. O subtotal é recalculado a cada alteração e reflete os preços atuais do Catálogo (FR-18), que é justamente o que a revalidação da FR-19 depois confronta.
- **A Reserva de Estoque nunca é nomeada para o Comprador.** É conceito de domínio, não de interface. O que o Comprador vê é o efeito: cancelou, o Produto volta a aparecer disponível na vitrine no mesmo instante (UJ-3).

## Component Patterns

Comportamental. Especificação visual vive em `DESIGN.md.Components` ou nos defaults do shadcn.

| Componente | Onde | Regras de comportamento |
|---|---|---|
| Barra superior | Toda tela pública e de Comprador | Wordmark leva à Vitrine. Contador do Carrinho reflete a soma de unidades, não de linhas, e atualiza sem recarregar. **Sem Sessão o ícone do Carrinho não exibe contador** — a FR-16 dá Carrinho só a Comprador autenticado — e clicar nele leva ao Login. Ausente na área administrativa, que tem chrome próprio. |
| Menu da conta | Toda tela pública e de Comprador | `DropdownMenu` do shadcn. Com Sessão: Meus pedidos, Meus endereços, Perfil, Sair. Sem Sessão: Entrar, Criar conta. É a única porta para as três superfícies de conta e para encerrar a Sessão, que a FR-2 exige invalidar no servidor. |
| Navegação administrativa | Área administrativa | Barra própria com os quatro destinos (Vendedores, Categorias, Produtos, Pedidos), a identificação de quem está na Sessão e a saída. Sem busca global, sem Carrinho. Abaixo de 640px os destinos viram `Sheet`. |
| Breadcrumb | Página de Produto, Resultados | Trilha de Categoria acima do título. É o único caminho de volta da Página de Produto para uma listagem — a IA não oferece outro. |
| Faixa de Categorias | Toda tela pública e de Comprador | Lista plana, sem hierarquia. Clicar equivale a buscar com termo vazio e filtro de Categoria — cai em Resultados, com o estado na URL. Rola horizontalmente dentro de si abaixo de 640px. |
| Botão Adicionar ao Carrinho | Cartão de Produto, Caixa de compra | Espera o servidor; sem atualização otimista. Sucesso atualiza o contador da barra superior. Quantidade acima do **Estoque disponível** é recusada com o número disponível na mensagem. **Sem Sessão** (FR-7, FR-16): leva ao Login e volta com o Item de Carrinho já criado, como descrito em *State Patterns*. É a única passagem de Público para Comprador, e o primeiro clique de qualquer avaliador sem conta. |
| Botão Confirmar Pedido | Último passo do checkout | Um por fluxo. Desabilita durante o envio e não permite duplo disparo — a criação de Pedido é idempotente por chave de operação (NFR-12), e a interface não pode depender disso para se comportar. |
| Selo de disponibilidade | Cartão de Produto, Caixa de compra | Binário no cartão: "Em estoque" ou "Indisponível". Na Caixa de compra mostra o **Estoque disponível** quando ele é menor que o teto de 10 unidades por Item de Carrinho (§7.1), porque abaixo disso o número muda a decisão de quantidade. Nunca some da tela: "Indisponível" é informação, não ausência. |
| Busca global | Toda tela pública e de Comprador | Termo limitado a 100 caracteres no cliente **e** no servidor. `Enter` submete. Seletor de Categoria acoplado pré-filtra. Termo vazio devolve listagem completa paginada, não erro (FR-12). |
| Cartão de Produto | Vitrine, Resultados | Clique em qualquer ponto abre a Página de Produto. "Adicionar ao Carrinho" no cartão adiciona 1 unidade sem sair da tela. |
| Barra de filtros | Resultados | Filtros aplicados ficam visíveis como chips removíveis individualmente (FR-13). Todo estado — termo, filtros, ordenação, página — vive na URL: recarregar ou compartilhar preserva a listagem. |
| Ordenação | Resultados | Três opções e só elas (FR-14): preço crescente, preço decrescente, mais recentes. **Padrão: mais recentes.** O desempate é estável e visível na tela — Produtos de mesmo preço mantêm ordem entre requisições, e a paginação nunca repete nem pula um Produto. |
| Caixa de compra | Página de Produto | Preço, disponibilidade, quantidade (teto 10) e ação. |
| Linha de Item de Carrinho | Carrinho | Quantidade edita em linha com atualização otimista; falha reverte e explica. Remover não pede confirmação — é reversível re-adicionando. |
| Passos do checkout | Checkout | **Dois** passos nomeados: Endereço → Revisão. O Frete é inteiramente derivado do CEP (FR-21) e o pagamento não coleta dado nenhum (§8 proíbe dado de cartão até no Provedor Simulado), então nenhum dos dois merece uma tela de entrada: ambos aparecem como blocos da Revisão. Voltar preserva o preenchido, e a Revisão tem retorno explícito ao Carrinho **sem perder o Endereço escolhido** (FR-22). A Revisão é o último ponto de leitura antes do compromisso. |
| Selo de Status do Pedido | Meus pedidos, Detalhe do Pedido, painel do Administrador | Mesmo componente, mesmo texto e mesma cor nas três superfícies. O texto é a forma de exibição declarada em *A máquina de estados na interface* — nunca o identificador cru. |
| Linha do tempo do Pedido | Detalhe do Pedido | Uma linha por transição registrada, com estado anterior, novo e carimbo de tempo, em ordem cronológica (FR-30, NFR-9). Sem etapas futuras especulativas: a linha do tempo mostra o que aconteceu, não o que se espera. |
| Tabela de Pedidos | Painel do Administrador | `Table` do shadcn com rolagem horizontal dentro do contêiner. Filtro por Status do Pedido **e ordenação por data** (FR-32), padrão mais recentes primeiro. **Não há busca por número de Pedido** — dito explicitamente para que o ensaio do Roteiro C conte com o filtro e a ordenação para achar a linha, e não descubra a falta ao vivo. Avançar estado pede confirmação em `Dialog` com o nome do estado destino. |
| Paginação | Resultados, Meus pedidos, listas administrativas | `Pagination` do shadcn. 20 itens por página (§7.1). Informa página atual e total de resultados. Página além do total é estado vazio tratado, não erro (FR-15). |

## State Patterns

| Estado | Superfície | Tratamento |
|---|---|---|
| Carregamento inicial | Vitrine, Resultados | `Skeleton` do shadcn em grade, com a mesma forma dos cartões. Nunca spinner centralizado. |
| Vitrine sem Catálogo Semeado | Vitrine | "O Catálogo ainda não tem Produtos." Sem ação para o Público; com Sessão de Administrador, caminho para cadastrar. É consequência testável da FR-6 e a primeira tela que a banca vê no cenário exato que o NFR-1 e a SM-3 testam. |
| Busca sem resultado | Resultados | "Nenhum Produto encontrado para '{termo}'." Abaixo: os filtros ativos como chips e um botão "Limpar filtros" (FR-12). Nunca página em branco. |
| Faixa de preço inválida | Resultados | Mensagem em linha no filtro: "O preço mínimo não pode ser maior que o máximo." A listagem anterior permanece; nunca lista incoerente (FR-13). |
| Carregamento da Página de Produto | Página de Produto | `Skeleton` na forma da imagem, do título e da Caixa de compra. |
| Produto desativado ou inexistente | Página de Produto | "Este Produto não está disponível." Ação única: voltar à Vitrine. Produto de Vendedor desativado recebe o mesmo tratamento — a interface não distingue os dois casos (FR-12). |
| Imagem de Produto ausente | Cartão de Produto, Página de Produto | Bloco neutro na proporção declarada com o nome do Produto centralizado. **Nunca o ícone de imagem quebrada do navegador** — um caminho errado no Catálogo Semeado não pode abrir um buraco na primeira tela da apresentação. |
| Visitante tenta adicionar ao Carrinho | Página de Produto, Vitrine, Resultados | Leva ao Login guardando Produto de origem e quantidade; ao autenticar volta à Página de Produto com o Item de Carrinho já criado (FR-7, FR-16). |
| E-mail já cadastrado | Cadastro | Erro em linha no campo: "Já existe uma conta com este e-mail." Com link para o Login. |
| Credencial inválida | Login | **A mesma** mensagem para e-mail inexistente e para senha errada: "E-mail ou senha incorretos." A FR-2 exige que a recusa não revele qual dos dois falhou — ver a exceção declarada em *Voice and Tone*. |
| Conta bloqueada | Login | "Muitas tentativas. Tente novamente em 15 minutos." (§7.1) O bloqueio conta o par (e-mail, origem) **exista a conta ou não** (FR-2) — mesma exceção declarada em *Voice and Tone*. |
| Nenhum Pedido ainda | Meus pedidos | "Você ainda não fez nenhum Pedido." Ação única: voltar à Vitrine. |
| Nenhum Endereço ainda | Meus endereços | "Nenhum Endereço cadastrado." Ação única: cadastrar. |
| Carrinho vazio | Carrinho | "Seu Carrinho está vazio." Ação única: voltar à Vitrine. |
| Revalidação: preço mudou | Abertura do Carrinho **e** entrada do checkout | `Alert` listando o preço anterior e o atual por Item de Carrinho, exigindo confirmação antes de seguir (FR-19). |
| Revalidação: unidade indisponível | Abertura do Carrinho **e** entrada do checkout | `Alert` **bloqueia o avanço** até o Item ser removido ou a quantidade ajustada, com as duas ações dentro do próprio `Alert` (FR-19). Confirmar não é bloquear: deixar passar faria o Comprador chegar ao botão irreversível para receber a recusa atômica da FR-24 no único ponto do fluxo que não admite erro. |
| Nenhum Endereço cadastrado | Checkout, passo Endereço | O passo abre direto no formulário de novo Endereço, sem lista vazia intermediária. É o primeiro Pedido de todo Comprador (FR-20). |
| Aguardando pagamento | Pedido em processamento | Estado próprio, não modal e não spinner. Mostra o número do Pedido, o Status do Pedido e o tempo restante até a expiração (15 min padrão, 60 s em demonstração). **Ação persistente em todos os estados desta tela: "Ver o Pedido em Meus pedidos"** — o Comprador nunca fica preso na tela. As três saídas do estado são declaradas: `PAGO` → a tela vira o Detalhe do Pedido no lugar, sem navegação nova; `PAGAMENTO_RECUSADO` por recusa (FR-27) **ou** por expiração (FR-34) → a tela substitui o relógio pelo motivo e pela ação de tentar de novo, na mesma superfície. O §14 do PRD aponta esta tela como o elemento de interface mais arriscado do documento; um relógio que para de contar sem dizer por quê é exatamente o risco. |
| Pagamento recusado | Detalhe do Pedido | Motivo visível, tentativas restantes de 3 (§7.1) e ação de tentar de novo. Os Itens de Pedido ficam no próprio Pedido — nada é remontado (UJ-1, caso de borda). |
| Tentativa expirada | Detalhe do Pedido | Mesmo tratamento do recusado, com o motivo "Tempo de pagamento expirado" (FR-34). |
| Tentativas esgotadas | Detalhe do Pedido | Ação de tentar de novo sai da tela. Resta cancelar. |
| Nova Tentativa de Pagamento recusada por indisponibilidade | Detalhe do Pedido | O Estoque disponível acabou entre a recusa e a nova tentativa (FR-27). Mensagem explícita, o Pedido permanece em `PAGAMENTO_RECUSADO`, e a ação de tentar de novo sai da tela. Resta cancelar. |
| Cancelamento | Detalhe do Pedido | `Dialog` de confirmação nomeando o Pedido. Após confirmar, o selo muda e a linha do tempo ganha a transição (UJ-3). |
| Cancelamento recusado porque o estado mudou | Detalhe do Pedido | A consulta é de 10 s e a simulação avança a cada 30 s: existe janela em que o Comprador clica "Cancelar Pedido" numa tela `SEPARANDO` que já virou `ENVIADO`. Atualiza o selo e a linha do tempo e explica: "Este Pedido foi enviado enquanto você estava nesta tela e não pode mais ser cancelado." **Nunca cair no `Toast` genérico de erro** — seria um erro cru no clímax da UJ-3. |
| Lista administrativa vazia | Vendedores, Categorias, Produtos | "Nenhum {entidade} cadastrado." Ação única: criar. É o estado inicial da UJ-4 antes de Rafael abastecer a loja. |
| Erro de validação administrativa | Formulários do Administrador | Erro em linha por campo, com o limite nomeado — "O nome do Produto tem no máximo 200 caracteres" (§7.1). Nunca resumo genérico no topo. |
| Estoque abaixo das Reservas ativas | Produtos, ajuste de Estoque | Recusa informando **quantas unidades estão comprometidas** (FR-11). Não é limite de campo, é contagem em tempo real: "Há 4 unidades comprometidas em Pedidos abertos. O Estoque total não pode ficar abaixo disso." |
| Remoção de Vendedor com Produtos | Vendedores | Recusa e **oferece desativar no lugar** (FR-8), com a ação alternativa no próprio `Alert`. |
| Remoção de Categoria com Produtos · nome duplicado | Categorias | Recusa nomeando quantos Produtos estão vinculados; nome duplicado é recusado em linha no campo (FR-10). |
| Pedido já avançou por outro ator | Painel do Administrador | O Administrador e a simulação de entrega colidem (nota do §4.6 do PRD, que delegou esta decisão à etapa de UX). `Alert` **informativo, não destrutivo**: "Este Pedido já está em {estado}. A linha foi atualizada." Dizer "transição inválida" aqui reportaria causa falsa — o Pedido não falhou, ele avançou. |
| Transição inválida | Painel do Administrador | `Alert` destrutivo nomeando a transição tentada e qual seria permitida a partir do estado atual (UJ-4, caso de borda). O addendum exige que a transição inválida falhe alto (invariante 2). |
| Pagamento aprovado sobre Pedido cancelado | Painel e Detalhe do Administrador | `Alert` persistente no Detalhe e marcador na linha da Tabela de Pedidos. O Status do Pedido não muda (FR-26), mas a informação não pode virar perda silenciosa (invariante 7 do addendum): o §4 do addendum lista este caso como um dos dois que o mock precisa resolver porque o gateway real vai exigir. |
| Recurso de outro Comprador | Qualquer | Não existe tela de "acesso negado" com dica. O servidor nega (NFR-6) e a interface trata como não encontrado. |
| Sessão expirada | Qualquer | Redireciona ao Login preservando o destino. Após entrar, volta para onde estava. |
| Erro do servidor | Qualquer | `Toast` destrutivo com identificador de correlação visível (NFR-9). É o que transforma "deu erro" em algo rastreável durante a apresentação. |

## Modo demonstração

*Seção específica deste produto.* O §7.1 troca dois prazos em demonstração: a expiração da Tentativa de Pagamento cai de 15 min para 60 s, e o intervalo da simulação de entrega de 24 h para 30 s. Isso muda o comportamento da interface, não só a configuração.

- **A tela Pedido em processamento consulta o servidor a cada 3 segundos até o Status do Pedido sair de `AGUARDANDO_PAGAMENTO`.** A condição é a saída do estado, não a permanência nele: parar a consulta ao virar `PAGAMENTO_RECUSADO` deixaria um relógio morto na tela. `[ASSUMPTION: 3 s; nenhum requisito fixa o intervalo. Precisa ser bem menor que os 60 s da demonstração para a virada parecer imediata na banca.]`
- **O Detalhe do Pedido consulta a cada 10 segundos** enquanto o estado não for terminal (`ENTREGUE` ou `CANCELADO`), para que a UJ-2 mostre `PAGO → SEPARANDO → ENVIADO → ENTREGUE` avançando sozinho durante a apresentação. `[ASSUMPTION: 10 s.]`
- **A Tabela de Pedidos do Administrador consulta a cada 10 segundos enquanto listar algum Pedido não terminal.** Sem isso, a cada 30 s um processo de fundo muda exatamente as linhas que a tabela exibe, e o Roteiro A deixa a simulação de entrega rodando durante o Roteiro C — a colisão está agendada dentro da apresentação. Qualquer transição também recarrega a linha afetada. `[ASSUMPTION: 10 s.]`
- **Nenhuma outra superfície consulta em intervalo.** Vitrine, Resultados, Carrinho e as demais listas administrativas atualizam por ação do usuário.
- **Sem WebSocket.** Consulta em intervalo resolve o caso e não adiciona dependência nem fronteira nova ao sistema.
- **O caminho de recusa é acionado por regra determinística sobre o total do Pedido, sem tela e sem reconfiguração.** A FR-25 exige que a recusa seja acionável sob demanda, não por sorte, e o passo 5 do Roteiro B pede um Pedido cuja confirmação nunca chega. Os centavos do total decidem: o apresentador escolhe Produto e quantidade para cair no caminho que quer, e nenhuma superfície de operador entra na IA. A faixa exata de centavos por desfecho é decisão de arquitetura e vive na configuração do NFR-16 — e precisa estar no roteiro de ensaio, porque sem saber a regra ninguém consegue provocar a recusa ao vivo.
- **O relógio de expiração é derivado do servidor**, não contado no navegador. O addendum (invariante 6) proíbe estado alcançado por temporizador em memória; a interface segue a mesma regra, senão recarregar a página zera a contagem e mente.

## Interaction Primitives

**A busca é a navegação.** Não há menu de departamentos, não há navegação lateral na loja. O caminho do Comprador é: busca → filtro → Produto → Carrinho → checkout.

- `Enter` na busca global submete de qualquer tela.
- Todo estado de listagem vive na URL. Voltar no navegador funciona; compartilhar o link reproduz a tela.
- Atualização otimista apenas em quantidade de Item de Carrinho. Toda operação que toca Pedido, Estoque ou pagamento espera o servidor — são as operações que o NFR-12 exige idempotentes e o NFR-7 exige consistentes.
- Confirmação em `Dialog` só onde a ação é irreversível: cancelar Pedido, avançar Status do Pedido, desativar Produto ou Vendedor.
- `Esc` fecha o `Dialog` ou `Sheet` mais acima.

**Proibido em toda parte:** rolagem infinita (a paginação da FR-15 é o contrato), afordância que só aparece em `hover` abaixo de `md`, `Dialog` empilhado além de um nível, e qualquer animação de celebração ao concluir Pedido.

## Accessibility Floor

Comportamental. Contraste visual vive em `DESIGN.md`.

O piso é **WCAG 2.2 AA completo**, acima do "básico" do NFR-11. A verificação do NFR-11 continua valendo como teste mínimo: **percorrer a UJ-1 inteira sem tocar no mouse.**

- Todo campo de formulário tem `label` associado programaticamente — não `placeholder` fazendo as vezes de rótulo.
- Foco visível em tudo que recebe teclado, herdado do token `ring` do shadcn, verificado em 3:1 contra o fundo.
- Ordem de tabulação igual à ordem de leitura em toda superfície. Na Página de Produto isso significa que a caixa de compra vem depois do título e do preço, nunca antes.
- Mudança de Status do Pedido durante a consulta em intervalo é anunciada por `aria-live="polite"` — a virada para `PAGO` não pode ser só visual.
- Resultado de busca anuncia a contagem ao atualizar: "{N} Produtos encontrados."
- Erro de formulário é associado ao campo por `aria-describedby` e recebe foco ao submeter.
- Imagem de Produto tem texto alternativo com o nome do Produto. Imagem decorativa não existe no MVP.
- Contraste de 4,5:1 em todo texto de conteúdo, 3:1 em componente de interface. O cinza pequeno da referência não é copiado.

## Responsive & Platform

De **360px a 1440px sem rolagem horizontal e sem elemento inacessível** (NFR-10). O teste é o requisito.

| Faixa | Comportamento |
|---|---|
| 360–639px | Grade de Produtos em 1 coluna. Busca ocupa a linha inteira abaixo do wordmark. Faixa de Categorias rola horizontalmente dentro de si. Caixa de compra empilha abaixo da descrição. Checkout em um passo por tela. |
| 640–1023px | Grade em 2 colunas. Busca volta à linha do wordmark. Filtros viram `Sheet` acionado por botão. |
| 1024–1439px | Grade em 3 colunas. Filtros em coluna à esquerda, fixos. Caixa de compra à direita na Página de Produto. |
| ≥1440px | Grade em 4 colunas, conteúdo limitado a 1440px e centralizado. |

O painel do Administrador é **desktop-first**: a UJ-4 é operação, não navegação de vitrine. Funciona abaixo de `md` com a tabela rolando dentro do contêiner, mas não é o alvo. O fluxo do Comprador é **mobile-first** — a UJ-1 acontece no navegador do celular, no ônibus.

## Inspiration & Anti-patterns

- **Herdado da Amazon — a cromática do chrome.** Barra superior quase-preta azulada, ação em amarelo, link em azul-petróleo, disponibilidade em verde. É o que faz a tela ser reconhecida antes de ser lida.
- **Herdado da Amazon — a busca dominante.** Largura quase total, seletor de Categoria acoplado, presente em toda tela — o §9 do PRD já a elegeu navegação principal.
- **Herdado da Amazon — os padrões de preço e compra.** Centavos sobrescritos, caixa de compra separada à direita, breadcrumb de Categoria.
- **Herdado do shadcn — todo o resto do vocabulário de superfície.** A marca do Azamon é *o que se acrescenta ao shadcn*, não um sistema do zero. Postura deliberada.
- **Rejeitado — a densidade da Amazon.** Facetas com contagem, estrelas, seletores de variação, tarjas de patrocinado e menu de departamentos existem porque a Amazon cobre livros a pranchas de surf. O Catálogo Semeado não tem essa largura; copiar a densidade seria encenar um catálogo que não existe.
- **Rejeitado — o cinza pequeno da referência.** Reprova em AA. Desvio consciente da réplica.
- **Rejeitado — os frameworks de "taste" anti-genérico.** Avaliados e descartados: o próprio `design-taste-frontend` se declara fora de escopo para painel administrativo, tabela de dados e formulário multi-passo — os três presentes aqui. E um skill cujo trabalho declarado é fazer a interface **não** parecer com nada existente desfaz a premissa de uma réplica assumida.
- **Rejeitado — urgência fabricada.** Sem contagem de escassez, sem "X pessoas vendo agora", sem selo de oferta — ver *Voice and Tone*.
- **Rejeitado — celebração ao concluir Pedido.** Confete e animação de sucesso escondem a informação que importa: o número do Pedido e o Status do Pedido.

## Key Flows

As quatro jornadas são herdadas do §2.3 do PRD, com os nomes preservados literalmente. Os passos abaixo acrescentam a camada de experiência — superfície, estado e interação — sem repetir a narrativa de produto.

### UJ-1 — Marina compra um carregador no ônibus

1. Marina abre a Vitrine no navegador do celular, com Sessão ativa. Grade em 1 coluna; a busca ocupa a linha inteira.
2. Digita "carregador tipo c" e submete com `Enter`. Resultados carregam com `Skeleton` na forma dos cartões; a contagem é anunciada por `aria-live`.
3. Aplica faixa de preço até R$ 100 pelo `Sheet` de filtros e ordena por preço crescente. Termo, filtro, ordenação e página entram na URL.
4. Abre o terceiro resultado. A caixa de compra empilha abaixo da descrição, com o selo de disponibilidade em `{colors.available}`. Adiciona ao Carrinho; o contador da barra superior sobe.
5. No checkout percorre **Endereço → Revisão**. Escolhe o Endereço de casa entre os dois que já cadastrou; na Revisão o Frete aparece calculado a partir do CEP, e subtotal, Frete e total fecham ao centavo.
6. **Clímax:** confirma no botão `{colors.primary-strong}` — o único laranja do fluxo, e o único ponto irreversível. A tela vira Pedido em processamento, com número do Pedido e relógio de expiração vindo do servidor. Três segundos depois a consulta em intervalo traz `PAGO`, o selo muda e `aria-live` anuncia. Ela fecha o navegador antes do ponto: o Pedido existe fora da aba.

Falha: a Tentativa de Pagamento é recusada → o Status do Pedido vai para `PAGAMENTO_RECUSADO`, o Detalhe do Pedido mostra o motivo nomeado e as tentativas restantes. Os Itens de Pedido continuam no Pedido; nada é remontado.

### UJ-2 — Marina acompanha o Pedido até a porta

1. Entra em Meus pedidos. Lista paginada de 20, o Pedido de ontem no topo, cada linha com o selo de Status do Pedido.
2. Abre o Detalhe do Pedido: Itens de Pedido com o **preço praticado**, Endereço, Frete, total, e a linha do tempo com `PAGO → SEPARANDO` carimbados.
3. **Clímax:** ela não recarrega nada. A consulta de 10 s traz `ENVIADO` e depois `ENTREGUE`; a linha do tempo ganha as transições na frente dela e `aria-live` anuncia cada uma. A superfície que responde "onde está meu Pedido" é a mesma que mostra a resposta mudando — ninguém precisa perguntar.

### UJ-3 — Marina desiste antes de sair

1. Abre o Detalhe do Pedido, ainda em `SEPARANDO`. O botão "Cancelar Pedido" está presente porque o estado o permite.
2. Clica; um `Dialog` de primeiro nível pede confirmação nomeando o Pedido.
3. **Clímax:** confirma. O selo vira `CANCELADO`, a linha do tempo ganha a transição, e o Produto volta a aparecer disponível na Vitrine no mesmo instante — o efeito da Reserva de Estoque liberada, sem que a palavra apareça em tela.

Caso de borda: se o Pedido estivesse em `ENVIADO`, o botão **não estaria lá** — ausente, não cinza — e o detalhe explicaria em uma frase por quê.

### UJ-4 — Rafael abastece a loja e despacha o dia

1. Rafael entra na área administrativa, separada da loja: sem barra superior da loja, sem busca global.
2. Cadastra um Vendedor, cria três Produtos com Categoria, preço, Estoque e imagem. Nome limitado a 200 caracteres, descrição a 4.000 (§7.1). Eles aparecem na Vitrine imediatamente.
3. Abre a lista de Pedidos e filtra por `PAGO`. `Table` do shadcn rolando dentro do contêiner.
4. **Clímax:** avança um Pedido para `SEPARANDO`. O `Dialog` de confirmação nomeia o estado destino; ao confirmar, o selo muda na tabela sem recarregar a página, e a mesma transição já está na linha do tempo que Marina vê do outro lado. É o momento em que as duas superfícies revelam ser a mesma máquina de estados.

Caso de borda: tentar mover um Pedido `CANCELADO` para `SEPARANDO` → `Alert` destrutivo nomeando a transição tentada e dizendo qual seria permitida a partir daquele estado.
