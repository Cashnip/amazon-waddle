---
title: Azamon
status: final
created: 2026-08-14
updated: 2026-09-06
---

# PRD: Azamon

*Título de trabalho — confirmar.*

## Sumário Executivo

*Uma página. O resto do documento detalha o que está aqui.*

**O que é.** Uma réplica funcional do núcleo de comércio eletrônico da Amazon, construída como trabalho acadêmico por 2 a 4 pessoas em um semestre. Uma pessoa entra, encontra um Produto, coloca no Carrinho, paga e acompanha a entrega até a porta.

**A tese.** O valor de uma réplica da Amazon não está na superfície coberta, e sim no **ciclo fechado com honestidade arquitetural**. Dez telas desconectadas são uma maquete; cinco que sustentam um Pedido do clique ao "entregue" — com Estoque que baixa, pagamento que pode recusar e cancelamento que devolve o item à prateleira — são um sistema.

**O MVP em seis capacidades.** Conta de usuário · Catálogo com Vendedor · Busca com filtros · Carrinho · Checkout com pagamento simulado · Pedido com máquina de estados, cancelamento e simulação de entrega. Trinta e quatro requisitos funcionais, dezesseis não funcionais.

**As quatro decisões estruturais.**
1. **Marketplace no modelo de dados, não na interface.** Todo Produto pertence a um Vendedor e o Pedido registra de quem veio cada item — mas o Vendedor não acessa o sistema. Espinha de marketplace sem portal de vendedor.
2. **Pagamento simulado, porém assíncrono.** A confirmação chega fora da requisição do checkout, com idempotência e expiração, porque é assim que um gateway real se comporta. A virada para o Stripe é troca de implementação, não reescrita.
3. **A máquina de estados do Pedido é a peça central.** Histórico, cancelamento, painel administrativo e simulação de entrega não são quatro funcionalidades — são uma máquina de estados vista de quatro ângulos.
4. **Busca é texto mais filtros, não motor de busca.** Relevância sofisticada é reconhecidamente melhor e fica para depois, atrás da mesma fronteira.

**O que fica de fora.** Portal do Vendedor, avaliações, lista de desejos, recomendações, carrinho anônimo, devolução, aplicativo móvel, publicação em nuvem. Cada omissão está justificada no §5 e no addendum §7 — com o que se perdeu em cada uma.

**Como o sucesso é medido.** Os três roteiros de demonstração do §12 executam de ponta a ponta em ambiente limpo (SM-1), e o primeiro deles já executa até a metade do semestre (SM-6). Contra-métrica declarada: nenhuma funcionalidade nova entra enquanto os roteiros não passarem.

**Mapa de leitura**

| Se você é… | Leia | Depois |
|---|---|---|
| **Integrante do time, começando** | Addendum §8 (ordem de construção, começando pelo esqueleto vertical), §3 Glossário, §4 do seu módulo | §7 NFRs antes de escrever teste |
| **Avaliador com pouco tempo** | Este sumário, §1 Visão, §11 Direção Arquitetural, §12 Roteiro de Demonstração | Addendum §7, alternativas descartadas |
| **Quem vai desenhar a arquitetura** | §3 Glossário, §4 Funcionalidades, §7 NFRs, addendum inteiro | §11 para as fronteiras que precisam ficar abertas |
| **Quem vai escrever épicas e histórias** | §4 (FRs e suas consequências), §6 Escopo, §2.3 Jornadas | §16 Suposições, antes de assumir qualquer coisa |

## 0. Propósito do Documento

Este PRD define o MVP do **Azamon**, uma réplica funcional do núcleo de comércio eletrônico da Amazon, construída como trabalho acadêmico por um time de 2 a 4 pessoas ao longo de um semestre. Ele é o documento de origem do projeto: dele saem a arquitetura, as épicas e as histórias de implementação — não há PRD, pesquisa ou especificação de UX anterior a ele.

Como ler: o §3 Glossário fixa o vocabulário e o resto do documento o usa literalmente; as funcionalidades do §4 estão agrupadas por capacidade, com Requisitos Funcionais (FR) numerados globalmente e IDs estáveis, cada um com consequências testáveis; as jornadas do §2.3 (UJ) são referenciadas por ID dentro dos FRs. Inferências que o time ainda não confirmou aparecem marcadas com `[SUPOSIÇÃO: …]` no corpo do texto e reunidas no §16.

Decisões de tecnologia **não** estão aqui. A stack imposta (Go, Postgres, Redis, Docker), o desenho da máquina de estados, o caminho de migração e o racional das alternativas descartadas vivem em `addendum.md`, no mesmo diretório. O §11 declara a *direção* arquitetural pretendida em nível de capacidade, porque ela restringe o MVP; o *como* fica no addendum e na etapa de arquitetura.

## 1. Visão

O Azamon é uma loja online completa em miniatura: uma pessoa entra, encontra um Produto, coloca no Carrinho, paga e acompanha a entrega até a porta. Nada além disso, e nada aquém disso. O ciclo fecha.

A aposta deste projeto é que o valor de uma réplica da Amazon não está na superfície coberta, e sim no **ciclo fechado com honestidade arquitetural**. É trivial produzir dez telas que não se conectam; é difícil produzir cinco que sustentam um Pedido do clique ao "entregue" — com Estoque que baixa, pagamento que pode recusar, e um cancelamento que devolve o item à prateleira. O segundo é um sistema. O primeiro é uma maquete.

Isso força um reconhecimento incomum: o Azamon tem **dois usuários reais e nenhum deles compra nada**. O primeiro é a Marina, a compradora fictícia cuja experiência precisa ser convincente. O segundo é a banca avaliadora, que julga em três eixos simultâneos — funcionalidade entregue, arquitetura e documentação, e apresentação ao vivo. Um PRD que só serve à Marina entrega um produto bonito que não defende suas escolhas. Um que só serve à banca entrega um diagrama que ninguém consegue demonstrar. Este documento tenta servir aos dois, e é por isso que ele carrega tanto um §12 Roteiro de Demonstração quanto um §7 de requisitos não funcionais com limiares numéricos.

## 2. Público-Alvo

### 2.1 Jobs To Be Done

**Do Comprador (fictício, mas com necessidades reais que o sistema precisa atender):**

- Encontrar rapidamente um Produto específico quando já sei o que quero, sem navegar por menus.
- Comparar opções dentro de uma faixa de preço que cabe no meu bolso, antes de decidir.
- Ter confiança de que o que eu paguei foi registrado — saber que deu certo, e saber imediatamente quando não deu.
- Acompanhar onde está o meu Pedido sem precisar perguntar a alguém.
- Desistir de um Pedido que ainda não saiu, sem falar com atendimento.

**Do Administrador (o operador do MVP):**

- Colocar Produtos de um Vendedor no ar sem depender de quem escreveu o código.
- Ver os Pedidos que chegaram e movê-los adiante.

**Do time (quem constrói):**

- Subir o ambiente inteiro em um comando e voltar a programar, não a configurar.
- Trabalhar em paralelo em módulos diferentes sem colidir.
- Chegar na apresentação com um caminho que já rodou dezenas de vezes.

**Da banca (quem avalia):**

- Ver o sistema funcionando de ponta a ponta, incluindo o que dá errado.
- Encontrar decisões justificadas, com o que foi abandonado dito em voz alta.

### 2.2 Não-usuários (v1)

- **O Vendedor como operador.** O Vendedor existe no modelo de dados e é dono do Produto, mas não tem acesso ao sistema no MVP. Quem cadastra é o Administrador.
- **O visitante não autenticado como comprador.** Ele navega e busca livremente, mas não monta Carrinho.
- **O operador de logística real.** A entrega é simulada; não há integração com transportadora.

### 2.3 Jornadas de Usuário

> **UJ-1. Marina compra um carregador no ônibus.**
> Marina, 34, analista, no ônibus a caminho de casa, lembra que o carregador do celular parou de funcionar. Já tem conta no Azamon e está autenticada no navegador do celular. Abre a vitrine, digita "carregador tipo c" com pressa e sem acento, e recebe uma lista de Produtos com preço e disponibilidade visíveis. Filtra por faixa de preço até R$ 100, ordena por preço, abre o terceiro resultado, confere que está disponível e adiciona ao Carrinho. No checkout, escolhe o Endereço de casa entre os dois que já cadastrou, vê o Frete calculado somar ao total, revisa e confirma. A tela mostra "processando pagamento", e segundos depois o Pedido aparece como **PAGO** com um número. Ela fecha o navegador antes do ônibus chegar no ponto. **Caso de borda:** se a Tentativa de Pagamento for recusada, o Pedido fica em **PAGAMENTO_RECUSADO** com o motivo visível e um botão para tentar de novo. Ela não precisa remontar nada: os itens ficam guardados no próprio Pedido, e é de lá que ela tenta outra vez.

> **UJ-2. Marina acompanha o Pedido até a porta.**
> No dia seguinte, Marina quer saber se o carregador saiu. Entra em "Meus pedidos", vê a lista com o Pedido de ontem no topo, e abre o detalhe: os itens que comprou, o preço que pagou (não o preço de hoje), o Endereço de entrega, o Frete, e uma linha do tempo mostrando **PAGO → SEPARANDO** com data e hora de cada transição. Ela sabe onde está sem perguntar a ninguém. Horas depois, o Status do Pedido avançou sozinho para **ENVIADO**, e depois para **ENTREGUE**.

> **UJ-3. Marina desiste antes de sair.**
> Marina percebe que comprou o carregador errado. Abre o detalhe do Pedido, que ainda está em **SEPARANDO**, e clica em cancelar. O sistema pede confirmação, muda o Status do Pedido para **CANCELADO** e devolve as unidades ao Estoque — o Produto volta a aparecer como disponível na vitrine no mesmo instante. **Caso de borda:** se o Pedido já estivesse em **ENVIADO**, o botão de cancelar não estaria lá, e o detalhe explicaria por quê.

> **UJ-4. Rafael abastece a loja e despacha o dia.**
> Rafael, o Administrador (na vida real, o integrante do time que opera a demonstração), entra na área administrativa. Cadastra um Vendedor novo, cria três Produtos ligados a ele com Categoria, preço, Estoque e imagem, e eles aparecem na vitrine imediatamente. Depois abre a lista de Pedidos, filtra pelos que estão em **PAGO**, e move um deles para **SEPARANDO** com um clique. **Caso de borda:** ao tentar mover um Pedido **CANCELADO** para **SEPARANDO**, o sistema recusa a transição e diz qual transição é permitida a partir daquele estado.

## 3. Glossário

*Termos de domínio definidos uma vez. O restante do PRD usa cada termo literalmente; sinônimo é violação de disciplina — em particular, nunca "compra" no lugar de **Pedido**, nunca "item" no lugar de **Produto**.*

- **Comprador** — pessoa autenticada que navega, monta o Carrinho e cria Pedidos. Tem um Carrinho, zero ou mais Endereços e zero ou mais Pedidos.
- **Administrador** — papel operacional único do MVP. Gerencia Vendedores, Produtos e opera Pedidos. Não compra.
- **Vendedor** — entidade dona de um Produto. Um Vendedor tem muitos Produtos; um Produto pertence a exatamente um Vendedor. No MVP não possui acesso ao sistema.
- **Produto** — item vendável. Pertence a exatamente um Vendedor e a exatamente uma Categoria. Tem nome, descrição, preço, imagem e Estoque.
- **Categoria** — agrupamento plano de Produtos, sem hierarquia. `[SUPOSIÇÃO: lista plana é suficiente; subcategorias não entram no MVP.]`
- **Estoque** — quantidade de unidades de um Produto disponíveis para venda. É reduzido por Reserva de Estoque, não diretamente pela venda.
- **Reserva de Estoque** — quantidade de um Produto comprometida por um Pedido ainda não concluído. Nasce com o Pedido, é liberada no cancelamento, na recusa e na expiração do pagamento, e **se consolida no envio** — no exato ponto em que o cancelamento deixa de ser possível (FR-28, FR-31).
- **Carrinho** — coleção de Itens de Carrinho pertencente a exatamente um Comprador. Um Comprador tem exatamente um Carrinho, persistente entre Sessões.
- **Item de Carrinho** — referência a um Produto mais uma quantidade, dentro de um Carrinho. Não congela preço.
- **Endereço** — endereço de entrega cadastrado por um Comprador, com CEP. Um Comprador pode ter vários.
- **Regra de Frete** — política que converte o CEP de um Endereço e o valor dos Itens de Pedido em um valor de Frete.
- **Frete** — valor de entrega calculado pela Regra de Frete e congelado no Pedido.
- **Pedido** — registro de uma compra concluída ou em andamento. Contém Itens de Pedido, o Endereço escolhido, o Frete, o total e um Status do Pedido. É imutável em conteúdo: apenas o Status do Pedido muda.
- **Item de Pedido** — Produto, quantidade e **preço praticado**, congelado no momento da criação do Pedido. Mudança de preço no Catálogo não altera Pedidos existentes.
- **Status do Pedido** — estado atual do Pedido na máquina de estados: `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO`, `SEPARANDO`, `ENVIADO`, `ENTREGUE`, `CANCELADO`.
- **Tentativa de Pagamento** — cada submissão de um Pedido ao Provedor de Pagamento. Tem resultado `APROVADO` ou `RECUSADO` e um identificador próprio. Um Pedido pode ter várias.
- **Provedor de Pagamento** — fronteira que processa uma Tentativa de Pagamento e devolve o resultado de forma assíncrona. No MVP, a única implementação é o Provedor Simulado.
- **Provedor Simulado** — implementação do Provedor de Pagamento que decide `APROVADO` ou `RECUSADO` por regra controlável, sem contato com serviço externo e sem coletar dados de cartão.
- **Sessão** — vínculo autenticado entre uma pessoa (Comprador ou Administrador) e o sistema.
- **Catálogo Semeado** — conjunto de Vendedores, Categorias e Produtos fictícios carregados no ambiente para demonstração e testes.

## 4. Funcionalidades

*Requisitos numerados globalmente com IDs estáveis: reorganizar as funcionalidades não renumera nada. O índice abaixo existe para localizar um requisito sem varrer a seção — e para as etapas seguintes extraírem por ID.*

| § | Funcionalidade | Requisitos |
|---|---|---|
| 4.1 | Conta e Identidade | FR-1 Cadastro · FR-2 Autenticação e Sessão · FR-3 Recuperação de senha · FR-4 Papéis e autorização · FR-5 Endereços |
| 4.2 | Catálogo | FR-6 Vitrine · FR-7 Página de Produto · FR-8 Vendedores · FR-9 Produtos · FR-10 Categorias · FR-11 Estoque e disponibilidade |
| 4.3 | Busca e Navegação | FR-12 Busca por texto · FR-13 Filtros · FR-14 Ordenação · FR-15 Paginação |
| 4.4 | Carrinho | FR-16 Exclusivo de autenticado · FR-17 Adicionar · FR-18 Alterar e remover · FR-19 Revalidação |
| 4.5 | Checkout | FR-20 Endereço · FR-21 Frete · FR-22 Revisão · FR-23 Criação do Pedido · FR-24 Reserva de Estoque · FR-25 Tentativa de Pagamento · FR-26 Aprovação · FR-27 Recusa · **FR-34 Expiração** |
| 4.6 | Pedidos e Pós-venda | FR-28 Máquina de estados · FR-29 Histórico · FR-30 Detalhe · FR-31 Cancelamento · FR-32 Painel do Administrador · FR-33 Simulação de entrega |

### 4.1 Conta e Identidade

**Descrição:** O Azamon distingue três situações: visitante (navega e busca), Comprador autenticado (tudo que envolve Carrinho, Pedido e Endereço) e Administrador (catálogo e operação de Pedidos). A fronteira é deliberadamente rígida — não existe Carrinho anônimo, o que elimina estado de sessão anônima e todos os casos de borda de fusão de Carrinho no login. Realiza UJ-1, UJ-4.

`[SUPOSIÇÃO: conta é e-mail e senha. Sem verificação de e-mail e sem login social no MVP.]`

**Requisitos Funcionais:**

#### FR-1: Cadastro de Comprador

Um visitante pode criar uma conta informando e-mail e senha.

**Consequências (testáveis):**
- E-mail já cadastrado é recusado com mensagem específica; nenhuma conta duplicada é criada.
- O e-mail é normalizado (minúsculas, sem espaços nas bordas) antes da verificação de unicidade: `Marina@x.com` e `marina@x.com` são a mesma conta.
- E-mail sem formato válido e senha com menos de 8 caracteres são recusados antes de qualquer escrita no banco.
- A senha nunca é persistida em texto claro nem em hash reversível (ver NFR-5).
- Após o cadastro bem-sucedido, o visitante está autenticado — não precisa fazer login em seguida.

#### FR-2: Autenticação e Sessão

Um Comprador ou Administrador pode iniciar uma Sessão com e-mail e senha, e encerrá-la.

**Consequências (testáveis):**
- Credencial inválida devolve a mesma mensagem genérica para e-mail inexistente e para senha errada — sem revelar qual dos dois falhou.
- A Sessão expira após período de inatividade configurável; requisição com Sessão expirada é tratada como não autenticada.
- Encerrar a Sessão invalida a credencial de sessão no servidor, não apenas no navegador.
- Após N tentativas falhas consecutivas para o mesmo e-mail, novas tentativas são recusadas por um período determinado; a recusa continua sem revelar se a conta existe.

#### FR-3: Recuperação de Senha

Um Comprador que esqueceu a senha pode solicitar redefinição e definir uma nova.

**Consequências (testáveis):**
- A solicitação gera um token de uso único com validade limitada; token usado ou expirado é recusado.
- A resposta à solicitação é idêntica para e-mail cadastrado e não cadastrado — não confirma existência de conta.
- Redefinir a senha encerra as Sessões ativas daquele Comprador.
- Uma nova solicitação invalida os tokens anteriores do mesmo Comprador — nunca existem dois tokens válidos ao mesmo tempo.

`[SUPOSIÇÃO: o token é entregue por uma caixa de correio de desenvolvimento local, sem serviço externo de e-mail — o ambiente roda offline na demonstração.]`

#### FR-4: Papéis e Autorização

O sistema distingue Comprador e Administrador, e restringe cada recurso ao seu dono ou ao papel correto.

**Consequências (testáveis):**
- Comprador A que requisita o Pedido, o Carrinho ou o Endereço de Comprador B recebe negação de acesso — inclusive por chamada direta à API com o identificador do recurso.
- Comprador que acessa qualquer rota administrativa recebe negação de acesso.
- A verificação acontece no servidor; esconder o link na interface não é a proteção (ver NFR-6).

`[SUPOSIÇÃO: o Administrador nasce semeado por migração/seed. Não existe tela de promoção de papel nem auto-cadastro de Administrador.]`

#### FR-5: Gerenciamento de Endereços

Um Comprador pode cadastrar, editar, remover e escolher entre seus Endereços.

**Consequências (testáveis):**
- CEP com formato inválido é recusado no cadastro.
- Remover um Endereço não altera nenhum Pedido existente — o Pedido guarda o Endereço que foi usado (ver FR-23).
- Um Comprador sem nenhum Endereço é levado ao cadastro de Endereço ao entrar no checkout, e volta ao ponto onde parou.

---

### 4.2 Catálogo

**Descrição:** O Catálogo é a base do sistema e carrega a decisão estrutural do projeto: **todo Produto pertence a um Vendedor**. A vitrine mostra o Vendedor na página do Produto, e o Pedido registra de quem veio cada Item de Pedido. Isso dá ao Azamon a espinha de um marketplace sem o custo de um portal de vendedor — a gestão pelo próprio Vendedor fica para a fase 2 (§10), o auto-cadastro fica fora mesmo lá, e o modelo de dados já comporta os dois. Realiza UJ-1, UJ-4.

**Requisitos Funcionais:**

#### FR-6: Vitrine

Qualquer visitante pode ver a página inicial com Produtos e Categorias.

**Consequências (testáveis):**
- A vitrine lista Produtos com nome, imagem, preço e indicação de disponibilidade.
- Produto com Estoque zero aparece marcado como indisponível e não é adicionável ao Carrinho (ver FR-11, FR-17).
- A vitrine responde com Catálogo Semeado vazio sem quebrar — mostra estado vazio explicativo.

#### FR-7: Página de Produto

Qualquer visitante pode abrir a página de um Produto e ver seus detalhes.

**Consequências (testáveis):**
- A página mostra nome, descrição, imagem, preço, Categoria, **nome do Vendedor** e disponibilidade.
- Identificador de Produto inexistente devolve página de "não encontrado", não erro de servidor.
- Para um visitante não autenticado, o botão de adicionar ao Carrinho leva ao login e retorna à página do Produto depois (ver FR-17).

#### FR-8: Gestão de Vendedores pelo Administrador

Um Administrador pode criar, editar e desativar Vendedores.

**Consequências (testáveis):**
- Vendedor desativado tem seus Produtos ocultados da vitrine e da busca, sem apagar Pedidos que já os referenciam.
- Não é possível remover um Vendedor que possui Produtos — a operação oferecida é desativar.

#### FR-9: Gestão de Produtos pelo Administrador

Um Administrador pode criar, editar e desativar Produtos, sempre vinculados a um Vendedor e a uma Categoria.

**Consequências (testáveis):**
- Criar Produto sem Vendedor ou sem Categoria é recusado.
- Preço menor ou igual a zero e Estoque negativo são recusados.
- Produto criado aparece na vitrine e na busca sem necessidade de reiniciar a aplicação ou reindexar manualmente.
- Editar o preço de um Produto **não** altera o preço praticado em nenhum Item de Pedido existente (ver FR-23).

`[SUPOSIÇÃO: a imagem do Produto é referenciada por URL apontando para arquivo estático servido pelo próprio ambiente. Upload de arquivo pelo Administrador entra junto com o armazenamento de objetos na fase 2 (§11).]`

#### FR-10: Gestão de Categorias

Um Administrador pode criar e renomear Categorias.

**Consequências (testáveis):**
- Categoria com Produtos vinculados não pode ser removida.
- Nome de Categoria duplicado é recusado.

#### FR-11: Estoque e Disponibilidade

Todo Produto tem Estoque, e o sistema impede a venda além do disponível.

**Consequências (testáveis):**
- Estoque disponível = Estoque total menos as Reservas de Estoque ativas.
- Produto com disponível igual a zero é exibido como indisponível e recusa entrada no Carrinho e no checkout.
- O Administrador pode ajustar o Estoque de um Produto, e o efeito na disponibilidade é imediato.
- O Administrador não pode reduzir o Estoque total abaixo das Reservas de Estoque ativas; a tentativa é recusada informando quantas unidades estão comprometidas. O disponível nunca é negativo.
- Dois checkouts simultâneos disputando a última unidade: exatamente um cria o Pedido, o outro recebe recusa explícita de indisponibilidade (ver NFR-7).

`[SUPOSIÇÃO: o Catálogo Semeado tem entre 50 e 200 Produtos distribuídos em pelo menos 5 Categorias para a demonstração, com um conjunto ampliado de até 5.000 Produtos usado apenas para verificar o desempenho da busca (SM-4).]`

---

### 4.3 Busca e Navegação

**Descrição:** É onde a Marina do UJ-1 ganha ou perde a venda. A decisão deliberada aqui é **texto mais filtros, não motor de busca**: busca por texto sobre nome e descrição, filtro por Categoria e por faixa de preço, e ordenação. Relevância sofisticada e tolerância a erro de digitação são reconhecidamente melhores — e ficam para a fase 2, atrás da mesma fronteira (§11). Realiza UJ-1.

**Requisitos Funcionais:**

#### FR-12: Busca por Texto

Qualquer visitante pode buscar Produtos por um termo livre.

**Consequências (testáveis):**
- O termo é comparado contra nome e descrição do Produto, sem diferenciar maiúsculas/minúsculas e sem diferenciar acentos ("cafe" encontra "café").
- Busca sem resultado devolve estado vazio com sugestão de ação (limpar filtros), não uma página em branco.
- Termo vazio devolve a listagem completa paginada, não erro.
- Produtos de Vendedor desativado e Produtos desativados não aparecem nos resultados.
- Caracteres com significado especial na consulta (`%`, `_`) são tratados como texto literal: buscar "50%" procura o texto "50%", não devolve o catálogo inteiro.
- O termo tem tamanho máximo declarado; acima disso é recusado, nunca processado inteiro (ver NFR-14).

#### FR-13: Filtros

Um visitante pode restringir os resultados por Categoria e por faixa de preço.

**Consequências (testáveis):**
- Filtros são combináveis entre si e com o termo de busca.
- Faixa de preço com mínimo maior que o máximo é recusada com mensagem, sem devolver lista incoerente.
- Os filtros aplicados ficam visíveis e são removíveis individualmente.
- O estado dos filtros está na URL: recarregar a página ou compartilhar o link preserva a mesma listagem.

#### FR-14: Ordenação

Um visitante pode ordenar os resultados.

**Consequências (testáveis):**
- Estão disponíveis: preço crescente, preço decrescente e mais recentes.
- A ordenação escolhida é preservada ao paginar e ao alterar filtros.
- Toda ordenação tem critério de desempate estável: Produtos com o mesmo preço mantêm ordem consistente entre requisições, e a paginação nunca repete nem pula um Produto.

#### FR-15: Paginação

Resultados de busca e listagens de Categoria são paginados.

**Consequências (testáveis):**
- Tamanho de página fixo e definido; a navegação informa a página atual e o total de resultados.
- Página além do total devolve resultado vazio tratado, não erro.
- O tamanho de página é decidido pelo servidor; parâmetro vindo do cliente que exceda o teto é limitado ao teto, não obedecido (ver NFR-14).

---

### 4.4 Carrinho

**Descrição:** Um Carrinho por Comprador, persistente entre Sessões, exclusivo de quem está autenticado. O Carrinho **não congela preço nem reserva Estoque** — ele é uma intenção. O compromisso só acontece na criação do Pedido (FR-23, FR-24). Essa distinção é o que evita a classe de bug em que um Carrinho abandonado segura o Estoque da loja inteira. Realiza UJ-1.

`[SUPOSIÇÃO: o Carrinho é persistente no servidor e sobrevive ao logout e à troca de dispositivo.]`

**Requisitos Funcionais:**

#### FR-16: Carrinho exclusivo de Comprador autenticado

Somente um Comprador autenticado possui Carrinho.

**Consequências (testáveis):**
- Visitante não autenticado que tenta adicionar ao Carrinho é levado ao login e, ao autenticar, retorna à página do Produto de origem.
- Não existe Carrinho anônimo nem fusão de Carrinho no login — nenhum estado de Carrinho é mantido fora da conta.

#### FR-17: Adicionar Produto ao Carrinho

Um Comprador pode adicionar um Produto ao Carrinho com uma quantidade.

**Consequências (testáveis):**
- Adicionar um Produto já presente soma à quantidade do Item de Carrinho existente, em vez de criar um segundo Item de Carrinho.
- Quantidade solicitada acima do Estoque disponível é recusada com mensagem que informa o disponível.
- Produto indisponível ou desativado não pode ser adicionado.
- A quantidade é um número inteiro maior ou igual a 1 e menor ou igual a um teto por Item de Carrinho, verificado independentemente do Estoque disponível.

#### FR-18: Alterar e Remover Itens de Carrinho

Um Comprador pode mudar a quantidade de um Item de Carrinho ou removê-lo.

**Consequências (testáveis):**
- Quantidade zero remove o Item de Carrinho.
- O total do Carrinho é recalculado a cada alteração e reflete os preços atuais do Catálogo.
- Esvaziar o Carrinho é possível em uma ação, com confirmação.

#### FR-19: Revalidação do Carrinho

Ao abrir o Carrinho ou iniciar o checkout, o sistema revalida cada Item de Carrinho contra o Catálogo.

**Consequências (testáveis):**
- Produto que ficou indisponível desde que foi adicionado é sinalizado no Carrinho e bloqueia o avanço até ser removido ou ajustado.
- Produto cujo preço mudou desde que foi adicionado tem o novo preço exibido e sinalizado antes da confirmação — o Comprador nunca é cobrado por um preço que não viu.
- Produto desativado ou de Vendedor desativado é sinalizado da mesma forma.
- Se a revalidação alterar o subtotal, o Frete é recalculado antes da confirmação — inclusive quando a mudança faz o subtotal cruzar o limiar de isenção (ver FR-21).

---

### 4.5 Checkout

**Descrição:** O checkout transforma a intenção do Carrinho em um Pedido comprometido. Ele é assíncrono por decisão de projeto: a confirmação do pagamento **não** acontece dentro da requisição que cria o Pedido. Isso é mais trabalho do que um `if` retornando aprovado — e é exatamente o que permite trocar o Provedor Simulado por um gateway real sem reescrever o checkout, porque o gateway real também confirma por fora (§11). Realiza UJ-1.

`[SUPOSIÇÃO: valores em BRL, sem cupom de desconto e sem imposto destacado.]`

**Requisitos Funcionais:**

#### FR-20: Seleção de Endereço

No checkout, um Comprador escolhe um Endereço entre os seus, ou cadastra um novo.

**Consequências (testáveis):**
- Comprador sem Endereço é conduzido ao cadastro e retorna ao checkout no mesmo ponto (ver FR-5).
- Trocar o Endereço recalcula o Frete antes da confirmação.

#### FR-21: Cálculo do Frete

O sistema calcula o Frete pela Regra de Frete a partir do CEP do Endereço escolhido e do valor dos itens.

**Consequências (testáveis):**
- O mesmo CEP e o mesmo valor de itens produzem sempre o mesmo Frete.
- CEP fora das faixas conhecidas recebe o valor da faixa padrão, não erro.
- O Frete aparece discriminado do subtotal na revisão; o total é subtotal mais Frete.

`[SUPOSIÇÃO: a Regra de Frete do MVP é valor fixo por região derivada da faixa de CEP, com isenção acima de um limiar de valor de itens. Nenhuma integração com Correios ou transportadora.]`

#### FR-22: Revisão do Pedido

Antes de confirmar, o Comprador vê a composição completa do que vai comprar.

**Consequências (testáveis):**
- A revisão mostra cada Item de Carrinho com preço unitário e subtotal, o Endereço escolhido, o Frete e o total.
- É possível voltar ao Carrinho a partir da revisão sem perder o Endereço já escolhido.
- Carrinho vazio não permite chegar à revisão.
- A revisão declara a forma de pagamento sem coletar nenhum dado dela: o Comprador lê que o Pedido será submetido ao Provedor de Pagamento e não encontra na tela nenhum campo de cartão (ver §8, FR-25).

#### FR-23: Criação do Pedido

Confirmar o checkout cria um Pedido em `AGUARDANDO_PAGAMENTO` a partir do Carrinho.

**Consequências (testáveis):**
- Cada Item de Pedido congela o **preço praticado** no momento da criação; alterações posteriores no Catálogo não o afetam (ver FR-9).
- O Pedido congela o Endereço e o Frete — remover o Endereço depois não altera o Pedido (ver FR-5).
- O Pedido registra o Vendedor de cada Item de Pedido.
- O Carrinho é esvaziado somente após a criação bem-sucedida do Pedido.
- Confirmar duas vezes (duplo clique, reenvio do formulário) cria **um** Pedido, não dois (ver NFR-12).

`[SUPOSIÇÃO: um Pedido pode conter Produtos de Vendedores diferentes e não se divide em Pedidos separados — o Status do Pedido é do Pedido inteiro.]`

#### FR-24: Reserva de Estoque

A criação do Pedido gera Reserva de Estoque para cada Item de Pedido.

**Consequências (testáveis):**
- O Estoque disponível cai imediatamente para os demais Compradores (ver FR-11).
- Se qualquer item não tiver disponível suficiente no instante da criação, **nenhum** Pedido é criado e nenhuma Reserva de Estoque é feita — a operação é tudo ou nada.
- Duas criações simultâneas disputando a última unidade: exatamente uma sucede (ver NFR-7).

#### FR-25: Tentativa de Pagamento

O Pedido criado é submetido ao Provedor de Pagamento, que responde de forma assíncrona.

**Consequências (testáveis):**
- A resposta da requisição de checkout não espera o resultado do pagamento: o Comprador vai para a tela do Pedido em estado "processando".
- Cada submissão gera uma Tentativa de Pagamento com identificador próprio e resultado `APROVADO` ou `RECUSADO`.
- O Provedor Simulado decide o resultado por regra determinística sobre o total do Pedido, declarada no §7.1. O caminho de recusa é acionável sob demanda — o apresentador escolhe Produto e quantidade para cair na faixa que quer, sem reconfigurar o sistema, sem reiniciar contêiner e sem superfície de operador. A faixa decide a **primeira** Tentativa de Pagamento de um Pedido; da segunda em diante o resultado é `APROVADO` — sem isso um Pedido recusado nunca poderia ser pago, e a nova tentativa da FR-27 não existiria.
- Nenhum dado de cartão é coletado, transmitido ou armazenado, nem no Provedor Simulado (ver §8).

#### FR-26: Confirmação de Pagamento Aprovado

Quando a Tentativa de Pagamento resulta `APROVADO`, o Pedido passa a `PAGO`.

**Consequências (testáveis):**
- A Reserva de Estoque permanece ativa; o Estoque não volta.
- A tela do Pedido reflete o novo Status do Pedido sem que o Comprador precise recarregar manualmente.
- Receber a mesma confirmação duas vezes produz um único efeito: o Pedido não avança dois estados nem duplica registro (ver NFR-12).
- Confirmação `APROVADO` que chega para um Pedido já `CANCELADO` não altera o Status do Pedido: ela é registrada na Tentativa de Pagamento correspondente e o Pedido passa a aparecer sinalizado no painel do Administrador como pagamento aprovado sobre Pedido cancelado (ver FR-31).

#### FR-27: Tratamento de Pagamento Recusado

Quando a Tentativa de Pagamento resulta `RECUSADO`, o Pedido passa a `PAGAMENTO_RECUSADO`.

**Consequências (testáveis):**
- A Reserva de Estoque é liberada e o Estoque volta a ficar disponível para outros Compradores.
- O motivo da recusa fica visível no detalhe do Pedido.
- O Comprador pode iniciar uma nova Tentativa de Pagamento a partir do Pedido; ela cria nova Reserva de Estoque.
- Se o Estoque não estiver mais disponível na nova tentativa, ela é recusada com mensagem explícita e o Pedido permanece em `PAGAMENTO_RECUSADO`, cancelável pelo Comprador.
- Um Pedido admite no máximo N Tentativas de Pagamento; atingido o limite, a única ação restante para o Comprador é cancelar.

#### FR-34: Expiração da Tentativa de Pagamento

Uma Tentativa de Pagamento que não recebe confirmação dentro do prazo expira, e o Pedido não fica preso.

**Consequências (testáveis):**
- Passado um prazo configurável sem confirmação, a Tentativa de Pagamento é marcada como expirada e o Pedido transita para `PAGAMENTO_RECUSADO` com motivo "tempo esgotado".
- A Reserva de Estoque é liberada na mesma transação da transição — nenhum Estoque fica comprometido por um pagamento que nunca chegou.
- A expiração reutiliza o caminho de recusa: nenhum estado novo entra na máquina de estados, e o Comprador pode tentar de novo dentro do limite da FR-27.
- Nenhum Pedido permanece em `AGUARDANDO_PAGAMENTO` por mais tempo que o prazo de expiração. Verificação: varrer os Pedidos após o prazo e não encontrar nenhum.
- A expiração deriva do histórico de transições, não de temporizador em memória: reiniciar o sistema não impede que Pedidos vencidos sejam expirados (ver FR-33, NFR-9).

---

### 4.6 Pedidos e Pós-venda

**Descrição:** Esta é a peça central do sistema, e a mais subestimada da lista original. Histórico, cancelamento, painel administrativo e simulação de entrega **não são quatro funcionalidades** — são uma máquina de estados vista de quatro ângulos. Construída no dia um, as quatro saem quase de graça; enfiada depois, cada uma custa uma refatoração. Realiza UJ-2, UJ-3, UJ-4.

**Máquina de estados do Pedido** (a fonte da verdade; o addendum detalha o desenho):

```
                    ┌──────────────────────┐
                    │ AGUARDANDO_PAGAMENTO │◄── criação (FR-23)
                    └───────┬──────────┬───┘
                  aprovado  │          │  recusado
                            ▼          ▼
                     ┌──────────┐  ┌─────────────────────┐
                     │   PAGO   │  │ PAGAMENTO_RECUSADO  │
                     └────┬─────┘  └──────────┬──────────┘
                          │                   │ nova tentativa (FR-27)
                          ▼                   └──────────► AGUARDANDO_PAGAMENTO
                    ┌───────────┐
                    │ SEPARANDO │
                    └─────┬─────┘
                          ▼
                    ┌──────────┐        ┌───────────┐
                    │ ENVIADO  │───────►│ ENTREGUE  │
                    └──────────┘        └───────────┘

CANCELADO é alcançável a partir de: AGUARDANDO_PAGAMENTO, PAGAMENTO_RECUSADO, PAGO, SEPARANDO.
ENVIADO, ENTREGUE e CANCELADO são terminais para o Comprador.
AGUARDANDO_PAGAMENTO → PAGAMENTO_RECUSADO acontece também por expiração da
Tentativa de Pagamento (FR-34), pelo mesmo caminho da recusa.
```

**Requisitos Funcionais:**

#### FR-28: Máquina de Estados do Pedido

Todo Pedido tem exatamente um Status do Pedido, e só transita pelas transições declaradas acima.

**Consequências (testáveis):**
- Transição não declarada é recusada com erro explícito, seja qual for a origem (Comprador, Administrador ou simulação de entrega).
- Toda transição registra data, hora, estado anterior, estado novo e quem a provocou.
- O histórico de transições de um Pedido é consultável e imutável.
- A transição `SEPARANDO → ENVIADO` **consolida** a Reserva de Estoque: o Estoque total do Produto baixa pela quantidade reservada e a Reserva encerra, na mesma transação. É o ponto exato em que o cancelamento deixa de ser possível (FR-31), então nenhuma unidade precisa mais voltar.
- Nenhuma outra transição altera o Estoque total. Reserva ativa só existe entre a criação do Pedido e o envio, o cancelamento ou a recusa — nunca indefinidamente.

#### FR-29: Histórico de Pedidos do Comprador

Um Comprador pode ver a lista dos seus Pedidos.

**Consequências (testáveis):**
- A lista traz número do Pedido, data, total e Status do Pedido, mais recentes primeiro.
- A lista mostra apenas Pedidos do Comprador autenticado (ver FR-4).
- Comprador sem Pedidos vê estado vazio com caminho para a vitrine.

#### FR-30: Detalhe do Pedido

Um Comprador pode abrir um Pedido e ver sua composição e sua linha do tempo.

**Consequências (testáveis):**
- O detalhe mostra Itens de Pedido com **preço praticado**, Endereço congelado, Frete, total e a linha do tempo das transições (FR-28).
- O botão de cancelar aparece se e somente se o Status do Pedido atual permite cancelamento (FR-31); quando não permite, o detalhe informa o motivo.
- Pedido de outro Comprador devolve negação de acesso, mesmo por acesso direto ao identificador.

#### FR-31: Cancelamento pelo Comprador

Um Comprador pode cancelar um Pedido enquanto ele estiver em `AGUARDANDO_PAGAMENTO`, `PAGAMENTO_RECUSADO`, `PAGO` ou `SEPARANDO`.

**Consequências (testáveis):**
- O cancelamento exige confirmação explícita.
- Ao cancelar, a Reserva de Estoque ativa é liberada e as unidades voltam ao Estoque disponível — verificável na vitrine imediatamente (realiza UJ-3).
- Cancelar Pedido em `ENVIADO` ou `ENTREGUE` é recusado, inclusive por chamada direta à API.
- Cancelar um Pedido já `CANCELADO` não produz efeito nem erro de servidor.

`[SUPOSIÇÃO: não há devolução nem reembolso no MVP. Cancelamento é a única saída, e só antes do envio.]`

#### FR-32: Painel de Pedidos do Administrador

Um Administrador pode listar todos os Pedidos e avançar o Status do Pedido manualmente.

**Consequências (testáveis):**
- A lista é filtrável por Status do Pedido e ordenável por data.
- O Administrador pode executar as transições `PAGO → SEPARANDO`, `SEPARANDO → ENVIADO` e `ENVIADO → ENTREGUE`.
- Transição inválida é recusada informando quais transições são permitidas a partir do estado atual (realiza UJ-4).
- O Administrador abre o detalhe de qualquer Pedido, incluindo a identificação do Comprador e os Vendedores dos Itens de Pedido.

#### FR-33: Simulação de Entrega

O sistema avança automaticamente os Pedidos ao longo dos estados de entrega, sem intervenção.

**Consequências (testáveis):**
- Um Pedido em `PAGO` progride sozinho por `SEPARANDO`, `ENVIADO` e `ENTREGUE`, respeitando as mesmas transições da FR-28.
- O intervalo entre etapas é configurável, com um valor curto para demonstração e um valor realista para uso normal.
- A simulação pode ser desligada por configuração, para que o Administrador conduza tudo manualmente na apresentação.
- A simulação nunca move um Pedido `CANCELADO` nem um Pedido em `AGUARDANDO_PAGAMENTO`.
- O avanço deriva do histórico de transições — o tempo decorrido desde a última transição registrada — e não de temporizador em memória: reiniciar o sistema retoma os Pedidos em andamento de onde pararam, sem congelar nenhum (ver NFR-9).

`[SUPOSIÇÃO: em modo demonstração, cada etapa avança na ordem de 30 segundos — rápido o bastante para caber na apresentação, lento o bastante para ser visto acontecendo.]`

**Notas:** `[NOTA PARA O PM]` a simulação de entrega e o painel do Administrador podem colidir — se ambos avançarem o mesmo Pedido, o segundo perde. A FR-28 já cobre isso recusando a transição inválida, mas a experiência dessa recusa na tela do Administrador precisa ser decidida na etapa de UX.

## 5. Não-Objetivos (Explícito)

- **Não é uma loja real.** Nenhum pagamento real, nenhuma entrega real, nenhum dado pessoal real. O Azamon não processa dinheiro em nenhuma fase deste MVP.
- **Não vira portal de vendedor.** O Vendedor não acessa o sistema. Nenhuma tela de auto-cadastro, moderação de anúncio, painel de vendas ou repasse financeiro entra no MVP. `[SUPOSIÇÃO: o portal do vendedor é a fase 2 mais natural, mas não constava da lista do usuário — confirmar.]`
- **Não é um motor de busca.** Sem relevância por pontuação, sem correção ortográfica, sem facetas dinâmicas com contagem, sem sugestão enquanto digita.
- **Não é aplicativo móvel.** Web responsivo apenas. Nenhum aplicativo nativo, nenhum PWA instalável.
- **Não tem sistema de conteúdo social.** Sem avaliações, notas, perguntas e respostas, comentários ou lista de desejos no MVP (fase 2, §10).
- **Não tem motor de promoções.** Sem cupom, desconto progressivo, frete promocional negociado ou programa de fidelidade.
- **Não tem pós-venda além do cancelamento.** Sem devolução, reembolso, troca ou abertura de chamado.
- **Não precisa estar publicado.** Roda local na demonstração; nenhum requisito de disponibilidade, domínio ou certificado.
- **Não é multi-idioma nem multi-moeda.** Português do Brasil e BRL.

## 6. Escopo do MVP

### 6.1 Dentro do escopo

- Conta de Comprador com cadastro, autenticação, recuperação de senha e Endereços.
- Papel de Administrador com gestão de Vendedores, Categorias, Produtos e Estoque.
- Catálogo com Produto pertencente a Vendedor, exibido na vitrine e na página de Produto.
- Busca por texto com filtro de Categoria e faixa de preço, ordenação e paginação.
- Carrinho persistente, exclusivo de Comprador autenticado, com revalidação contra o Catálogo.
- Checkout com Endereço, Frete por Regra de Frete, revisão, criação de Pedido, Reserva de Estoque e pagamento simulado assíncrono com caminho de aprovação e de recusa.
- Pedido com máquina de estados completa, histórico e detalhe para o Comprador, cancelamento com devolução ao Estoque, painel do Administrador e simulação de entrega.
- Ambiente completo em contêineres, subindo em um comando, com Catálogo Semeado.
- **Entregáveis de documentação**, porque o §1 reconhece a banca como leitora e a nota depende deles: README que leva do clone ao sistema rodando, diagrama dos seis módulos do NFR-2 com suas fronteiras e o que atravessa cada uma, e o `addendum.md` mantido vivo — toda decisão de arquitetura tomada durante a construção volta para ele.

### 6.2 Fora do escopo do MVP

- **Portal do Vendedor** — fica fora por sequenciamento, não por custo: as telas seriam as da FR-8 e FR-9 restritas por um predicado de autorização que a FR-4 já obriga a testar. `[NOTA PARA O PM: é o candidato número um a ser puxado de volta se o cronograma segurar, à frente das avaliações — é o único item de fase 2 que muda a natureza do sistema e não só a vitrine.]` → fase 2.
- **Avaliações e notas de Produto** — é a ausência que mais pesa na fidelidade visual à Amazon. `[NOTA PARA O PM: barata, e muda a percepção da vitrine mais do que qualquer outra coisa desta lista. Segunda na fila de puxar de volta, atrás do Portal do Vendedor — que muda a natureza do sistema, não só a aparência.]` → fase 2.
- **Lista de desejos e recomendações** → fase 2.
- **Carrinho anônimo com fusão no login** — decisão consciente; elimina uma classe inteira de casos de borda.
- **Upload de imagem de Produto** — entra junto com o armazenamento de objetos → fase 2 técnica.
- **Divisão de um Pedido por Vendedor** — o Pedido é único mesmo com Vendedores diferentes; dividir só faz sentido com repasse financeiro, que não existe aqui.
- **Devolução e reembolso** → sem previsão.
- **Notificação por e-mail de mudança de Status do Pedido** — a linha do tempo no detalhe do Pedido cobre a necessidade sem depender de serviço de e-mail.
- **Publicação em nuvem** → fase 2 técnica (§11).

### 6.3 Ordem de corte

O time tem entre 2 e 4 pessoas — o dobro de capacidade entre um extremo e outro. Se for o extremo menor, ou se a SM-6 não for atingida no meio do semestre, corta-se **nesta ordem**, e não pelo que estiver mais difícil na semana:

1. **FR-3 Recuperação de senha** — não aparece em nenhum roteiro do §12; as contas da demonstração vêm do Catálogo Semeado.
2. **FR-10 Gestão de Categorias** — as Categorias vêm semeadas; o CRUD nunca é demonstrado.
3. **FR-8 Gestão de Vendedores** — idem; o Roteiro C perde um passo e continua fazendo sentido.
4. **FR-33 Simulação de entrega** — o Administrador avança o Status do Pedido manualmente (FR-32). Custa um clique a mais na apresentação e devolve tempo de desenvolvimento.
5. **FR-14 Ordenação** — os filtros sozinhos sustentam o Roteiro A; ordenação é a metade menos essencial da busca.

**Nunca cortar, por mais invisíveis que pareçam:** FR-19 (revalidação do Carrinho), FR-24 (Reserva de Estoque), FR-34 (expiração da Tentativa de Pagamento) e o teste de concorrência do NFR-7. São o que separa o sistema da maquete — exatamente a tese do §1 — e nenhum deles aparece numa tela.

## 7. Requisitos Não Funcionais Transversais

*Este é o eixo que a banca lê quando avalia arquitetura. Cada item tem limiar ou critério verificável — "deve ser rápido" não é requisito.*

- **NFR-1 — Ambiente em um comando.** O sistema completo (aplicação, banco, cache, Catálogo Semeado) sobe com um único comando em contêineres, a partir do repositório limpo. Verificação: um integrante que nunca rodou o projeto chega ao sistema funcionando em ≤ 15 minutos seguindo apenas o README. O Catálogo Semeado é determinístico e versionado — subir o ambiente duas vezes produz exatamente os mesmos dados, para que a demonstração seja reproduzível — e **plausível**: nomes, descrições, preços, Categorias e imagens críveis. É a primeira coisa que a banca vê, antes de qualquer funcionalidade; "Produto Teste 3" a R$ 999.999,00 desfaz meses de trabalho correto.
- **NFR-2 — Fronteiras de módulo explícitas.** O código se organiza em **seis módulos de domínio — identidade, catálogo, busca, carrinho, pedido, pagamento** — e um módulo só acessa outro pela sua interface pública. Esta lista é a decomposição canônica: o §11 e o addendum se referem a ela, e as funcionalidades do §4 mapeiam uma a uma (§4.1 → identidade, §4.2 → catálogo, §4.3 → busca, §4.4 → carrinho, §4.5 → pedido + pagamento, §4.6 → pedido). Verificação: nenhuma dependência direta entre estruturas internas de módulos diferentes, checável por inspeção ou por regra de dependência automatizada.
- **NFR-3 — Pagamento atrás de fronteira trocável.** Toda interação com pagamento passa pela interface do Provedor de Pagamento. Verificação: substituir a implementação exige alterar apenas essa implementação e a configuração — zero alteração no checkout, no Pedido ou na máquina de estados (ver SM-5).
- **NFR-4 — Desempenho da busca.** Com Catálogo Semeado de 5.000 Produtos em máquina de desenvolvimento, a busca com filtros responde em ≤ 500 ms no percentil 95.
- **NFR-5 — Armazenamento de senha.** Senhas são guardadas com função de derivação lenta e sal por usuário. Verificação: inspeção do registro no banco; nenhuma senha legível nem hash simples de uma passagem.
- **NFR-6 — Autorização no servidor.** Todo recurso pertencente a um Comprador é verificado no servidor a cada requisição. Verificação: teste automatizado que chama a API com o identificador de recurso de outro Comprador e espera negação, para Pedido, Carrinho e Endereço.
- **NFR-7 — Consistência de Estoque sob concorrência.** Duas criações de Pedido simultâneas para a última unidade de um Produto resultam em exatamente um Pedido. Verificação: teste de concorrência que dispara N requisições paralelas e verifica que o número de Pedidos criados é igual ao Estoque inicial, nunca maior.
- **NFR-8 — Cobertura do fluxo crítico por testes.** Todo FR do fluxo busca → Carrinho → checkout → Pedido tem ao menos um teste automatizado que falha se a consequência declarada do FR quebrar. A suíte roda em contêiner, sem depender da máquina de quem executa.
- **NFR-9 — Rastreabilidade de execução.** Cada requisição carrega um identificador de correlação presente nos registros de log, e toda transição de Status do Pedido é registrada. Verificação: dado um número de Pedido, é possível reconstruir a sequência de eventos que levou ao estado atual.
- **NFR-10 — Responsividade.** As telas do fluxo de compra funcionam de 360 px a 1440 px de largura sem rolagem horizontal e sem elemento inacessível.
- **NFR-11 — Acessibilidade básica.** O fluxo completo de compra é operável por teclado; todo campo de formulário tem rótulo associado; o contraste de texto atende ao nível AA. Verificação: percorrer UJ-1 inteiro sem usar o mouse.
- **NFR-12 — Idempotência.** Criar Pedido e confirmar pagamento são idempotentes por chave de operação: a mesma operação recebida duas vezes produz um único efeito. Verificação: teste que reenvia a mesma confirmação e verifica que o Pedido avançou uma vez só.
- **NFR-13 — Precisão monetária.** Valores monetários são representados como inteiros de centavos, nunca como ponto flutuante. O arredondamento acontece num único ponto declarado, e o total exibido é sempre exatamente a soma dos componentes exibidos. Verificação: teste com valores que quebram em ponto flutuante (por exemplo três unidades de R$ 33,33 mais Frete) conferindo o total ao centavo.
- **NFR-14 — Limites de entrada.** Todo campo de texto e todo parâmetro numérico vindo do cliente tem limite máximo declarado e verificado no servidor — termo de busca, descrição de Produto, quantidade, tamanho de página. Verificação: requisição com valor acima do teto é recusada ou limitada, nunca processada inteira.
- **NFR-15 — Operação offline na demonstração.** O ambiente sobe e o fluxo completo funciona sem acesso à internet: nenhuma imagem, fonte, folha de estilo, script ou dependência é buscada em rede externa em tempo de execução. Verificação: desconectar a rede e percorrer o Roteiro A (§12) inteiro, incluindo as imagens dos Produtos.
- **NFR-16 — Parâmetros declarados.** Todo limiar citado por um requisito tem valor padrão declarado na tabela abaixo, e nenhum deles é constante espalhada pelo código — todos vivem em configuração. É o que torna o NFR-14 verificável, e o que permite encolher prazos para caber na apresentação sem tocar em código.

### 7.1 Parâmetros configuráveis e valores padrão

*Valores iniciais, não verdades. A etapa de arquitetura ajusta o que precisar — mas nenhum deles pode ficar sem valor declarado.*

| Parâmetro | Requisito | Padrão | Em demonstração |
|---|---|---|---|
| Tentativas de autenticação falhas antes do bloqueio | FR-2 | 5 | — |
| Duração do bloqueio por tentativas | FR-2 | 15 min | — |
| Expiração da Sessão por inatividade | FR-2 | 7 dias | — |
| Validade do token de redefinição de senha | FR-3 | 30 min | — |
| Tamanho máximo do termo de busca | FR-12, NFR-14 | 100 caracteres | — |
| Itens por página | FR-15 | 20 | — |
| Teto de itens por página aceito do cliente | FR-15, NFR-14 | 60 | — |
| Unidades por Item de Carrinho | FR-17, NFR-14 | 10 | — |
| Isenção de Frete acima de | FR-21 | R$ 299,00 | — |
| Regra de decisão do Provedor Simulado, pelos centavos do total | FR-25 | Na **primeira** Tentativa de Pagamento: `,00`–`,89` → `APROVADO` · `,90`–`,94` → `RECUSADO` · `,95`–`,99` → confirmação nunca chega. Da segunda em diante: `APROVADO` | Idêntica — a regra não é reconfigurada |
| Tentativas de Pagamento por Pedido | FR-27 | 3 | — |
| Expiração da Tentativa de Pagamento | FR-34 | 15 min | 60 s |
| Intervalo entre etapas da simulação de entrega | FR-33 | 24 h | 30 s |
| Tamanho máximo do nome de Produto | NFR-14 | 200 caracteres | — |
| Tamanho máximo da descrição de Produto | NFR-14 | 4.000 caracteres | — |

## 8. Restrições e Salvaguardas

**Segurança**
- Nenhum dado de cartão entra no sistema — **nem no Provedor Simulado**. A simulação não pede número de cartão, não gera número fictício e não guarda nada parecido com um. A fronteira que mantém dados de pagamento fora do sistema é a mesma que torna a migração para um gateway real segura, e ela vale desde o primeiro dia.
- Entradas vindas do navegador são validadas no servidor; a validação no cliente é conveniência, nunca a defesa.
- Credenciais e segredos de configuração não vivem no repositório.

**Privacidade**
- Todos os dados de demonstração são fictícios. Nenhum nome, e-mail, CPF ou endereço de pessoa real entra no Catálogo Semeado nem nas contas de teste.
- O sistema coleta apenas o que os FRs exigem: e-mail, senha e Endereço de entrega. Sem telemetria, sem rastreamento de comportamento, sem terceiros.

**Custo**
- Zero custo de infraestrutura no MVP: tudo roda local, em contêineres, sem serviço pago e sem conta em nuvem. Qualquer item que exija cartão de crédito é, por definição, fase 2.

## 9. Arquitetura da Informação

**Público (sem Sessão)**
- Vitrine · Resultados de busca (com filtros e ordenação) · Página de Produto · Login · Cadastro · Recuperação de senha

**Comprador (com Sessão)**
- Carrinho · Checkout (Endereço → Revisão) · Tela do Pedido em processamento · Meus pedidos · Detalhe do Pedido · Meus endereços · Perfil

**Administrador (com Sessão)**
- Vendedores (lista, criar, editar) · Categorias · Produtos (lista, criar, editar, ajustar Estoque) · Pedidos (lista com filtro por Status do Pedido, detalhe, avançar status)

**O checkout tem dois passos, não quatro.** O Frete é inteiramente derivado do CEP do Endereço escolhido (FR-21) e o pagamento não coleta dado nenhum — o §8 proíbe dado de cartão em qualquer ponto do sistema, **inclusive no Provedor Simulado**. Nenhum dos dois tem dado a pedir, então ambos aparecem como blocos da Revisão, e o resultado do pagamento chega fora do checkout, na Tela do Pedido em processamento (FR-25).

A navegação principal do Comprador é a barra de busca — é o caminho da UJ-1 e deve estar presente em toda tela pública. A área administrativa é separada da loja e não aparece para quem não é Administrador.

## 10. Plataforma e Fase 2 de Valor

**Plataforma do MVP:** aplicação web responsiva, executada localmente em contêineres durante a demonstração. Sem aplicativo móvel, sem instalação, sem publicação em nuvem.

**Fase 2 de valor** (funcionalidades que o Comprador percebe; fora do MVP):
1. **Avaliações e notas de Produto** — só quem comprou avalia; a média aparece na vitrine e na página de Produto.
2. **Lista de desejos** — salvar Produto para depois.
3. **Recomendações** — "quem comprou isto também comprou", ainda que por regra simples de Categoria.
4. **Portal do Vendedor** — o Vendedor entra e gerencia apenas os Produtos dele. Reaproveita as telas da FR-8 e FR-9 com autorização por dono do recurso; auto-cadastro, moderação e repasse ficam fora mesmo na fase 2. É o item de maior efeito por esforço desta lista. `[SUPOSIÇÃO: pertence à fase 2; não constava da lista do usuário.]`

## 11. Direção Arquitetural Pretendida

*Esta seção é separada do §10 de propósito: fase 2 de valor é o que o usuário ganha, fase 2 técnica é como o sistema evolui. Misturar as duas numa lista só é o erro que faz um roadmap parecer coerente enquanto esconde que metade dele não muda nada para quem usa. Aqui estão as evoluções técnicas pretendidas e — mais importante — **o que o MVP precisa fazer hoje para que elas continuem baratas amanhã.***

O princípio é único e se repete: **implementação simples hoje, atrás de uma fronteira explícita; troca depois sem reescrever quem chama.**

| Evolução pretendida | Como o MVP mantém a porta aberta | Custo se ignorarmos hoje |
|---|---|---|
| **Elasticsearch** para busca | A busca é consumida por uma interface própria (NFR-2). Nenhum outro módulo monta consulta de banco para buscar Produto. | Consultas de busca espalhadas por controladores e telas; a troca vira caça ao `SELECT`. |
| **Stripe** para pagamento | Provedor de Pagamento como interface, confirmação assíncrona e idempotência desde o dia um (NFR-3, NFR-12). | Checkout síncrono acoplado ao resultado do pagamento; a troca vira reescrita do fluxo inteiro. |
| **S3** para imagens de Produto | Imagem referenciada por URL, nunca por caminho de disco embutido na aplicação (FR-9). | Caminhos locais espalhados pelo código e pelo banco. |
| **Lambda / processamento assíncrono** | A simulação de entrega e a confirmação de pagamento já são disparadas fora da requisição do usuário (FR-25, FR-33). | Lógica de longa duração dentro do ciclo de requisição, impossível de extrair. |
| **Microsserviços** | Monólito modular com fronteiras nos seis módulos do NFR-2 — identidade, catálogo, busca, carrinho, pedido, pagamento. | Módulos entrelaçados; a separação exige reescrever, não mover. |

**Microsserviços é destino, não ponto de partida.** Começar distribuído neste projeto consome o semestre em orquestração e entrega três serviços que não conversam. Um monólito modular com fronteiras corretas entrega o ciclo fechado *e* torna a migração um exercício de mover pacote e trocar chamada de função por chamada de rede. O trade-off aceito explicitamente: o MVP não demonstra comunicação entre serviços, escalabilidade horizontal independente nem tolerância a falha parcial — e não deve fingir que demonstra.

**Marco opcional de fim de semestre: extrair um serviço.** O trade-off acima tem um custo escondido que vale nomear — em trabalho acadêmico, a fase 2 costuma nunca acontecer. Se o objetivo de aprender sistemas distribuídos for real, um monólito que nunca é dividido o cumpre tão mal quanto um distribuído que nunca fecha o ciclo. A saída honesta não é nenhum dos extremos: **fechado o ciclo e passando SM-1, extrair exatamente um serviço** — busca ou pagamento, os dois já atrás de interface. Uma única extração exercita fronteira de rede, serialização, versionamento de contrato e o comportamento do sistema quando a dependência cai. Custa uma fração de começar distribuído e entrega o aprendizado inteiro. É **opcional e explicitamente condicionado**: nenhuma extração começa antes de os três roteiros do §12 passarem (ver SM-C1).

## 12. Roteiro de Demonstração

*A apresentação é um dos três eixos da avaliação, então ela é requisito, não improviso. Estes três roteiros precisam rodar em ambiente limpo, do zero, sem intervenção no banco. Todo passo aqui é sustentado por um FR; se algum passo não tiver FR, falta requisito.*

**Roteiro A — o caminho feliz (realiza UJ-1, UJ-2)**
1. Subir o ambiente em um comando, com Catálogo Semeado (NFR-1).
2. Cadastrar uma conta nova e entrar (FR-1, FR-2).
3. Buscar um termo com erro de acentuação, filtrar por faixa de preço, ordenar por preço (FR-12, FR-13, FR-14).
4. Abrir a página de Produto e mostrar o Vendedor — é aqui que se explica a espinha de marketplace (FR-7).
5. Adicionar ao Carrinho, ajustar quantidade (FR-17, FR-18).
6. Checkout em dois passos: cadastrar Endereço e chegar à Revisão; voltar ao passo Endereço, trocar para outra região e mostrar o Frete recalculado na Revisão (FR-20, FR-21, FR-22).
7. Confirmar; mostrar o Pedido em processamento virar `PAGO` sem recarregar a página (FR-23, FR-25, FR-26).
8. Abrir "Meus pedidos" e o detalhe, mostrando o preço praticado e a linha do tempo (FR-29, FR-30).
9. Deixar a simulação de entrega avançar até `ENTREGUE` durante o resto da apresentação (FR-33).

**Roteiro B — o caminho triste (o que separa maquete de sistema)**
1. Repetir o checkout com um Pedido cujo total cai na faixa de recusa da regra do §7.1 — nada é reconfigurado entre os roteiros (FR-25).
2. Mostrar o Pedido em `PAGAMENTO_RECUSADO` com o motivo (FR-27).
3. Abrir a página do Produto em outra aba e mostrar que o Estoque **voltou** (FR-11, FR-27).
4. Tentar de novo com aprovação e mostrar o Pedido seguindo adiante (FR-27).
5. Com o prazo de expiração em 60 s (§7.1), criar um Pedido cujo total cai na faixa em que a confirmação **nunca chega**: mostrar o Pedido saindo sozinho de `AGUARDANDO_PAGAMENTO` por tempo esgotado e o Estoque voltando (FR-34). É a única forma de a SM-1 exercitar a expiração.

**Roteiro C — operação e reversão (realiza UJ-3, UJ-4)**
1. Entrar como Administrador, cadastrar Vendedor e Produto, e vê-lo aparecer na vitrine (FR-8, FR-9).
2. Mover um Pedido de `PAGO` para `SEPARANDO` pelo painel (FR-32).
3. Como Comprador, cancelar esse Pedido e mostrar o Estoque voltando na vitrine (FR-31).
4. Tentar cancelar um Pedido em `ENVIADO` e mostrar a recusa (FR-31).
5. Se houver tempo: disparar dois checkouts simultâneos na última unidade, mostrando que só um Pedido nasce (NFR-7).

**Regras de ensaio**
- Os três roteiros são ensaiados **a partir de clone limpo do repositório**, por ao menos duas pessoas diferentes, antes da apresentação.
- Nenhum passo entra na apresentação sem ter passado em ensaio. Em particular, o passo 5 do Roteiro C só é demonstrado se o teste de concorrência do NFR-7 estiver passando — não se demonstra ao vivo o que nunca passou automatizado.
- A verificação do NFR-1 e da SM-3 é recorrente, não única: a cada semana, um integrante diferente sobe o ambiente do zero.

**Perguntas prováveis da banca, e onde mora a resposta**

A nota de arquitetura costuma ser decidida na sequência de perguntas, não na demonstração. Cada pergunta tem um responsável nomeado antes da apresentação, e ninguém responde por uma parte que não construiu sem antes ler a fonte.

| Pergunta | Onde está a resposta |
|---|---|
| Por que o pagamento é assíncrono, se é simulado? | §11 e addendum §4 |
| Isso é um marketplace ou uma loja? | §4.2 e addendum §7 |
| Como vocês impedem que dois Compradores levem a última unidade? | NFR-7, FR-24, addendum §3 |
| Por que não usaram microsserviços? | §11, incluindo o marco opcional de extração |
| E se o pagamento nunca confirmar? | FR-34 e addendum §2, invariante 6 |
| Por que não tem carrinho para visitante? | addendum §7 |

## 13. Métricas de Sucesso

*Este produto não tem usuários reais, então métricas de uso seriam ficção. As métricas abaixo medem o que efetivamente determina o resultado do trabalho.*

**Primárias**
- **SM-1** — Os três roteiros do §12 executam de ponta a ponta em ambiente limpo, sem intervenção manual no banco e sem erro visível. Alvo: 3 de 3. Valida o conjunto dos FRs pelos caminhos que os atravessam.
- **SM-2** — Todo FR do fluxo crítico (busca → Carrinho → checkout → Pedido) tem ao menos um teste automatizado que falha se a consequência declarada do FR quebrar. Alvo: 100% desses FRs. Valida NFR-8.
- **SM-3** — Um integrante novo sobe o ambiente completo com Catálogo Semeado em ≤ 15 minutos a partir do repositório limpo, seguindo apenas o README. Valida NFR-1.
- **SM-6** — **Marco intermediário:** o Roteiro A executa de ponta a ponta até a metade do semestre, ainda que com telas cruas e catálogo mínimo. É a métrica que detecta cedo o único modo de falha capaz de anular o trabalho inteiro — chegar em dezembro com o ciclo aberto. Se SM-6 não for atingida no prazo, a resposta é cortar escopo, não acelerar.

**Secundárias**
- **SM-4** — Busca com filtros responde em ≤ 500 ms (p95) com 5.000 Produtos semeados. Valida FR-12 a FR-15 e NFR-4.
- **SM-5** — Trocar o Provedor Simulado por outra implementação exige alterar apenas a implementação e a configuração, sem tocar checkout, Pedido ou máquina de estados. Verificado por inspeção antes da entrega. Valida NFR-3 e a §11.
- **SM-7** — Os entregáveis de documentação (§6.1) existem e **não mentem** no dia da entrega: o README leva alguém de fora do clone ao sistema rodando sem ajuda (mesma verificação da SM-3), o diagrama de módulos corresponde à decomposição real do código, e nenhuma decisão registrada no addendum é contradita pelo que foi construído. Verificação: percorrer as decisões do addendum contra o código antes da entrega. Valida o eixo "arquitetura e documentação" declarado no §1.

**Contra-métricas (não otimizar)**
- **SM-C1** — Quantidade de telas e funcionalidades entregues. Contrabalança SM-1: mais funcionalidades com o ciclo quebrado vale menos que o ciclo fechado. Nenhuma funcionalidade nova entra enquanto os três roteiros não passarem.
- **SM-C2** — Percentual de cobertura de testes. Contrabalança SM-2: cobertura alta com o fluxo crítico sem asserção sobre consequência de FR é teatro. O que conta é o teste que falha quando o comportamento quebra.
- **SM-C3** — Tamanho do Catálogo Semeado. Contrabalança SM-4: 5.000 Produtos servem para medir a busca, não para impressionar. A demonstração usa um catálogo pequeno e crível.
- **SM-C4** — Volume de documentação. Contrabalança SM-7: páginas escritas não são prova de nada, e documentação que descreve um sistema diferente do construído é pior que documentação nenhuma. O que conta é o README funcionar e o diagrama bater com o código.

## 14. Riscos

- **A máquina de estados atrasa tudo.** Ela é pré-requisito de quatro funcionalidades (§4.6) e de dois dos três roteiros de demonstração. *Mitigação:* construir primeiro, com testes, antes das telas que dependem dela.
- **A busca vira projeto de busca.** A tentação de melhorar relevância é constante e sem fim natural. *Mitigação:* FR-12 a FR-15 são o teto do MVP; qualquer melhoria vai para a fase 2 (§11) e só depois de SM-1 passar.
- **O time se distribui cedo demais.** Com microsserviços declarados como destino, existe risco de alguém começar por ali. *Mitigação:* §11 declara explicitamente que o MVP é monólito modular, e NFR-2 dá o critério do que é fronteira suficiente.
- **A demonstração depende de algo que só funciona na máquina de um integrante.** *Mitigação:* NFR-1 e SM-3 tornam isso verificável antes da apresentação, não durante — e a verificação é semanal, por pessoas diferentes (§12, regras de ensaio), porque verificada uma vez em setembro ela não prova nada em dezembro.
- **Construção horizontal: nada funciona até o fim.** Terminar cada camada antes de passar à seguinte produz um sistema que só fecha o ciclo em dezembro — e dezembro é onde o tempo acaba. É o modo de falha mais provável do projeto e o único que anula a tese do §1. *Mitigação:* esqueleto vertical antes de alargar qualquer camada (addendum §8, passo 0) e SM-6 como marco de meio de semestre.
- **Integração entre front e back deixada para o fim.** Formato de data, decimal virando ponto flutuante, cookie de sessão, CORS — e, pior, a tela do Pedido em processamento, que é o elemento de interface mais arriscado do documento e o clímax da UJ-1. *Mitigação:* o esqueleto vertical força a integração na semana 2 ou 3; a tela de processamento é o primeiro protótipo, não o último.
- **Ninguém consegue defender as decisões na banca.** As respostas existem no PRD e no addendum, mas um documento que a equipe não abre é um documento que não existe — e quem apresenta pode não ter construído a parte perguntada. *Mitigação:* tabela de perguntas prováveis com responsável nomeado (§12) e ensaio de perguntas, não só de cliques.

## 15. Questões em Aberto

*Triadas no fechamento do PRD: nenhuma delas impede este documento de ser usado, mas duas travam a etapa seguinte. Cada uma com dono e condição de revisão.*

1. ✅ **Front-end: Next.js ou React puro?** Resolvida em 2026-09-06 por `bmad-architecture`: **Next.js 16 sobre React 19**, escolhido pelo time por familiaridade — num semestre, aprender do zero custa mais que a economia de um contêiner. Como este PRD antecipava, nada do §4 mudou. A renderização no servidor foi recusada pelo mesmo motivo que esta questão já nomeava: não há usuário real nem indexação a otimizar. O front-end é casca de apresentação sem regra de negócio (`AD-10` da espinha de arquitetura).
2. 🟡 **Portal do Vendedor é mesmo fase 2?** Ele é a consequência natural do modelo escolhido, mas não apareceu na lista de fase 2 do usuário. Corte consciente ou omissão? *Adiada, condicionada à questão 3.* Segura de adiar porque a reversão é barata: o Item de Pedido já carrega o Vendedor e as telas seriam as da FR-8 e FR-9 restritas por autorização (addendum §6).
3. 🔴 **Existe rubrica escrita do professor ou enunciado formal do trabalho?** **Bloqueadora.** Quatro suposições do §16 dependem exclusivamente desta resposta — Portal do Vendedor fora do MVP (n.º 1), ausência de verificação de e-mail (n.º 7), ausência de devolução e reembolso (n.º 8) e Pedido não dividido por Vendedor (n.º 2). Não são quatro riscos independentes: são um só, com quatro sintomas. Nenhuma análise adicional os resolve; o enunciado resolve os quatro de uma vez.
4. 🟢 **Como o time se divide?** Os seis módulos do NFR-2 são as costuras naturais, e o addendum §8 dá a ordem de construção. *Adiada para `bmad-sprint-planning`. Dono: o time.* Não afeta o conteúdo deste PRD.
5. 🟢 **A entrega inclui documentação formal além do código?** *Resolvida pela metade:* os entregáveis passaram a existir e a ser medidos (§6.1, SM-7, SM-C4). O que resta — se o professor exige formato ou artefato específico — cai na questão 3.
6. ✅ **Idioma.** Resolvida: interface, documentação e apresentação em português do Brasil.

## 16. Índice de Suposições

*Toda `[SUPOSIÇÃO]` do documento, reunida para confirmação explícita e **ordenada por risco** — confiança de que está certa, cruzada com o impacto de estar errada. Reversibilidade é o custo de corrigir depois de construído; suposição de alto impacto mas reversão barata não precisa ser decidida agora.*

**🔴 Decidir antes de construir**

1. **§5 / §10 — O Portal do Vendedor pertence à fase 2.** *(Confiança baixa · impacto alto · reversão cara.)* Não constava da lista de fase 2 do usuário: pode ter sido corte consciente ou omissão. Contra-argumento a favor de mantê-la: o portal não aparece em nenhum dos três roteiros do §12, logo não muda o que a banca vê. Depende da Questão em Aberto 3.

**🟡 Alto impacto, reversão barata — construir de forma que a correção continue barata**

2. **§4.5 / FR-23 — Um Pedido pode conter Produtos de Vendedores diferentes e não se divide.** *(Confiança média · impacto alto · reversão barata.)* O Pedido já registra o Vendedor de cada Item de Pedido, então separar por Vendedor depois é agrupamento de consulta, não remodelagem. Depende da Questão em Aberto 3.
3. **§4.2 / FR-9 — Imagem de Produto referenciada por URL de arquivo estático; upload é fase 2.** *(Confiança média · impacto médio · reversão barata.)* Escondia um requisito não declarado, agora explícito no NFR-15: a demonstração precisa funcionar sem internet.
4. **§3 — Categoria é lista plana, sem subcategorias.** *(Confiança média-alta · impacto médio · reversão média.)* Hierarquia é o que faz a navegação parecer a Amazon. Blindagem registrada no addendum.
5. **§4.2 / FR-11 — Catálogo Semeado de 50 a 200 Produtos em ao menos 5 Categorias; até 5.000 apenas para medir a busca.** *(Confiança média-alta · impacto médio · reversão barata.)* O NFR-1 passou a exigir que o Catálogo Semeado seja determinístico e versionado.
6. **§4.5 / FR-21 — Regra de Frete: valor fixo por região derivada da faixa de CEP, com isenção acima de um limiar.** *(Confiança média · impacto médio · reversão barata.)* Sustenta o passo 6 do Roteiro A — simplificar para frete fixo esvazia esse momento da demonstração.

**🟢 Baixo risco — confirmar por conveniência**

7. **§4.1 — Conta é e-mail e senha; sem verificação de e-mail e sem login social.** *(Confiança alta · impacto médio · reversão barata: a FR-3 já paga o mecanismo de token de uso único.)* Depende da Questão em Aberto 3.
8. **§4.6 / FR-31 — Sem devolução nem reembolso; cancelamento é a única saída, só antes do envio.** *(Confiança alta · impacto baixo-médio.)* Depende da Questão em Aberto 3.
9. **§4.4 — O Carrinho é persistente no servidor e sobrevive ao logout e à troca de dispositivo.** *(Confiança alta · impacto baixo.)*
10. **§4.1 / FR-3 — O token de redefinição é entregue por caixa de correio de desenvolvimento local, sem serviço externo.** *(Confiança alta · impacto baixo.)*
11. **§4.1 / FR-4 — O Administrador nasce semeado por migração; não há promoção de papel pela interface.** *(Confiança alta · impacto baixo.)*
12. **§4.5 — Valores em BRL, sem cupom e sem imposto destacado.** *(Confiança alta · impacto baixo.)*
13. **§4.6 / FR-33 — Em modo demonstração, cada etapa da simulação de entrega avança na ordem de 30 segundos.** *(Confiança alta · impacto baixo · é um valor de configuração.)*
