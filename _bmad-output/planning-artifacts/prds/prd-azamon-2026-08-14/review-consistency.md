---
title: Revisão de Consistência Interna — PRD Azamon + Addendum
status: review
created: 2026-08-14
escopo: prd.md + addendum.md (leitura integral de ambos)
---

# Revisão de consistência interna — Azamon

Caça exclusiva a **defeitos de consistência**: contradições diretas, texto envelhecido por
edição posterior, referências cruzadas quebradas, violações da disciplina de glossário que o
próprio §3 declara, requisitos mutuamente insatisfazíveis e integridade de numeração.
Não há aqui nenhuma observação de estilo nem nenhum apontamento de "está incompleto".

**Placar: 22 achados — 1 crítico · 7 altos · 8 médios · 6 baixos.**

---

## CRÍTICO

### C-1. A Reserva de Estoque "se consolida na entrega" (§3), mas o addendum declara que a transição até `ENTREGUE` não tem efeito sobre Estoque

**Onde está afirmado (A)** — `prd.md` §3 Glossário:

> **Reserva de Estoque** — quantidade de um Produto comprometida por um Pedido ainda não concluído. Nasce com o Pedido, é liberada no cancelamento ou na recusa de pagamento, e **se consolida na entrega**.

**O que contradiz (B)** — `addendum.md` §2, tabela "Estados e quem os provoca":

> | `PAGO` → `SEPARANDO` → `ENVIADO` → `ENTREGUE` | Administrador (FR-32) **ou** simulação de entrega (FR-33) | **Nenhum sobre Estoque** |

**Por que é crítico, e não uma diferença de redação.** A FR-11 fixa a fórmula:

> Estoque disponível = Estoque total menos as Reservas de Estoque ativas.

Consolidar a Reserva significa que ela deixa de ser *ativa*. Se, como manda o addendum, nada
acontece com o Estoque nessa transição, então no instante em que o Pedido chega a `ENTREGUE`
o termo "Reservas ativas" encolhe sem que o "Estoque total" encolha junto — e o disponível
**sobe** de volta. Ou seja: as unidades entregues voltam à prateleira. É exatamente a classe de
bug que a invariante 4 do próprio addendum §2 chama de "o bug mais caro deste sistema", e o
addendum instrui o arquiteto a produzi-lo. Nenhum FR resolve o conflito: a FR-26 diz o que
acontece com a Reserva no `PAGO` ("permanece ativa; o Estoque não volta"), a FR-27 e a FR-31
dizem o que acontece na liberação, e **nenhum FR diz o que acontece na entrega**.

**Correção mínima.** Escolher um lado e deixar os dois documentos dizendo a mesma coisa. O lado
barato é o addendum: trocar a célula "Nenhum sobre Estoque" por duas linhas —
`PAGO → SEPARANDO → ENVIADO` = "Nenhum sobre Estoque"; `ENVIADO → ENTREGUE` = "Consolida a
Reserva de Estoque: baixa definitiva do Estoque total na mesma transação (invariante 4)".
Alternativa igualmente válida e ainda mais barata: remover "e se consolida na entrega" do §3 e
declarar que a Reserva permanece ativa indefinidamente após a entrega — mas então FR-11 precisa
dizer que reservas de Pedidos `ENTREGUE` contam como ativas para sempre, o que é pior de explicar
à banca.

---

## ALTO

### A-1. O auto-cadastro do Vendedor é "fase 2" no §4.2 e "fora mesmo na fase 2" no §10 — e a referência de seção está errada

**(A)** `prd.md` §4.2, Descrição:

> Isso dá ao Azamon a espinha de um marketplace sem o custo de um portal de vendedor — **o auto-cadastro e a gestão pelo próprio Vendedor ficam para a fase 2 (§11)**, e o modelo de dados já os comporta.

**(B)** `prd.md` §10, item 4 (Fase 2 de valor):

> **Portal do Vendedor** — o Vendedor entra e gerencia apenas os Produtos dele. Reaproveita as telas da FR-8 e FR-9 com autorização por dono do recurso; **auto-cadastro, moderação e repasse ficam fora mesmo na fase 2.**

**(C, reforço)** `addendum.md` §7 coloca o auto-cadastro no balde dos **descartados**, não no dos adiados:

> | **Marketplace completo** (auto-cadastro, moderação de anúncio, repasse financeiro, painel de vendas) | Dobra o escopo — é o negócio inteiro, não uma funcionalidade. […] |

Duas em uma. Primeiro, a contradição direta: o auto-cadastro do Vendedor está simultaneamente
prometido para a fase 2 e excluído dela. Segundo, a referência `(§11)` do §4.2 aponta para a
seção errada — Portal do Vendedor é fase 2 **de valor**, que vive no §10; o §11 é fase 2
**técnica** e seu próprio preâmbulo declara a separação: *"Esta seção é separada da §10 de
propósito: fase 2 de valor é o que o usuário ganha, fase 2 técnica é como o sistema evolui."*
A tabela do §11 não tem linha de Portal do Vendedor.

**Correção mínima.** No §4.2, trocar por: "— a gestão pelo próprio Vendedor fica para a fase 2
(§10, item 4); o auto-cadastro fica fora inclusive na fase 2, e o modelo de dados já comporta os
dois."

---

### A-2. Dois itens diferentes do §6.2 são, cada um, "o primeiro a ser puxado de volta"

**(A)** `prd.md` §6.2, primeiro marcador:

> - **Portal do Vendedor** — […] `[NOTA PARA O PM: é o candidato número um a ser puxado de volta se o cronograma segurar, **à frente das avaliações** — é o único item de fase 2 que muda a natureza do sistema e não só a vitrine.]`

**(B)** `prd.md` §6.2, marcador imediatamente seguinte:

> - **Avaliações e notas de Produto** — […] `[NOTA PARA O PM: se o cronograma abrir no fim do semestre, **esta é a primeira coisa a puxar de volta** — é barata e muda a percepção da vitrine.]`

Dois marcadores adjacentes, ambos endereçados ao PM, ambos reivindicando o primeiro lugar da
mesma fila — e o primeiro deles nomeia o segundo para negar-lhe a posição. O placar geral do
documento é 3 × 1 a favor do Portal do Vendedor: além do §6.2, `addendum.md` §7 diz
*"É o **candidato número um a ser puxado para dentro** se o cronograma segurar"* e `prd.md` §10
item 4 diz *"É o item de maior efeito por esforço desta lista."* O marcador de Avaliações é o
sobrevivente de uma redação anterior.

**Correção mínima.** No marcador de Avaliações, trocar "esta é a primeira coisa a puxar de volta"
por "esta é a segunda coisa a puxar de volta, atrás do Portal do Vendedor".

---

### A-3. UJ-1 promete tolerância a erro de digitação que o documento inteiro nega

**(A)** `prd.md` §2.3, UJ-1:

> Abre a vitrine, digita "carregador tipo c" com pressa e **erro de dedo**, e recebe uma lista de Produtos relevantes com preço e disponibilidade visíveis.

**(B)** `prd.md` §4.3, Descrição:

> Relevância sofisticada e **tolerância a erro de digitação** são reconhecidamente melhores — e ficam para a fase 2, atrás da mesma fronteira (§11).

**(C)** `prd.md` §5:

> - **Não é um motor de busca.** Sem relevância por pontuação, **sem correção ortográfica**, sem facetas dinâmicas com contagem, sem sugestão enquanto digita.

**(D)** `prd.md` FR-12, única consequência sobre casamento de termo:

> O termo é comparado contra nome e descrição do Produto, sem diferenciar maiúsculas/minúsculas e sem diferenciar acentos ("cafe" encontra "café").

**(E)** `addendum.md` §7, linha do Elasticsearch, coluna "O que se perdeu":

> Relevância e **tolerância a erro de digitação**. Recuperável na fase 2 atrás da interface de busca.

Um "erro de dedo" ("carregdor", "tipo v") não é erro de acentuação. Sob a FR-12, ele devolve
zero resultados e a Marina cai no estado vazio da segunda consequência da FR-12 — o oposto do
que a jornada narra. E a UJ-1 não é decorativa: cinco seções declaram "Realiza UJ-1", o
Roteiro A do §12 declara "realiza UJ-1, UJ-2" e a NFR-11 usa "percorrer UJ-1 inteiro" como
procedimento de verificação. O próprio §12 já sabe disso e escreve o passo certo: *"Buscar um
termo com **erro de acentuação**, filtrar por faixa de preço, ordenar por preço (FR-12, FR-13,
FR-14)."*

**Correção mínima.** Na UJ-1, trocar "com pressa e erro de dedo" por "com pressa e sem acento",
alinhando com o passo 3 do Roteiro A.

---

### A-4. UJ-1 garante que o Carrinho sobrevive à recusa de pagamento; a FR-23 o esvazia antes de o pagamento acontecer

**(A)** `prd.md` §2.3, UJ-1, caso de borda:

> **Caso de borda:** se a Tentativa de Pagamento for recusada, o Pedido fica em **PAGAMENTO_RECUSADO** com o motivo visível e um botão para tentar de novo — **o Carrinho dela não é destruído no processo**.

**(B)** `prd.md` FR-23, consequência testável:

> O Carrinho é esvaziado somente após a criação bem-sucedida do Pedido.

**(C)** `prd.md` FR-25, consequência testável — o que torna (A) e (B) incompatíveis e não apenas ambíguos:

> A resposta da requisição de checkout não espera o resultado do pagamento: o Comprador vai para a tela do Pedido em estado "processando".

A sequência obrigatória é: checkout confirmado → Pedido criado com sucesso → **Carrinho esvaziado
(FR-23)** → tempo passa → confirmação assíncrona chega recusada (FR-27). Quando a recusa acontece,
o Carrinho já está vazio há tempo. Não existe caminho no documento em que uma Tentativa de
Pagamento seja recusada com o Carrinho ainda montado. A garantia da UJ-1 é irrealizável, e a
retomada real é outra: a FR-27 diz *"O Comprador pode iniciar uma nova Tentativa de Pagamento
**a partir do Pedido**"* — do Pedido, não do Carrinho.

**Correção mínima.** Na UJ-1, trocar o trecho final por: "— e ela retoma a partir do próprio
Pedido, sem precisar remontar nada".

---

### A-5. O §1 diz que os requisitos não funcionais estão no §9; eles estão no §7

**(A)** `prd.md` §1 Visão, último período:

> Este documento tenta servir aos dois, e é por isso que ele carrega tanto um §12 Roteiro de Demonstração quanto um **§9 de requisitos não funcionais** com limiares numéricos.

**(B)** Estrutura real do `prd.md`: o §7 é **"Requisitos Não Funcionais Transversais"** (NFR-1 a
NFR-15, cada um com limiar); o §9 é **"Arquitetura da Informação"** (lista de telas por papel,
sem um único limiar numérico).

Confirmado pelo resto do documento, que aponta corretamente: o Mapa de Leitura do Sumário cita
"§7 NFRs antes de escrever teste" e "§3 Glossário, §4 Funcionalidades, §7 NFRs". O §1 é a única
ocorrência de "§9" no documento inteiro, e é a errada.

**Correção mínima.** Trocar "§9" por "§7" no §1.

---

### A-6. `addendum.md` §5 cita "Suposição 11 do PRD" para a Regra de Frete; a Regra de Frete é a suposição nº 6

**(A)** `addendum.md` §5 Regra de Frete:

> Proposta a confirmar (**Suposição 11 do PRD**): tabela de faixas de CEP → região → valor fixo, com isenção acima de um limiar de subtotal.

**(B)** `prd.md` §16, item 11:

> 11. **§4.1 / FR-4 — O Administrador nasce semeado por migração; não há promoção de papel pela interface.** *(Confiança alta · impacto baixo.)*

**(C)** `prd.md` §16, item 6 — o alvo pretendido:

> 6. **§4.5 / FR-21 — Regra de Frete: valor fixo por região derivada da faixa de CEP, com isenção acima de um limiar.** *(Confiança média · impacto médio · reversão barata.)*

O §16 declara-se **"ordenada por risco"**; a reordenação renumerou a lista e o addendum ficou
apontando para a posição antiga. Que é erro de trânsito e não coincidência fica provado pelo
§6 do próprio addendum, que cita "(suposição 2)" e "(suposição 4)" e **acerta as duas** — só o §5
ficou para trás.

**Correção mínima.** Em `addendum.md` §5, trocar "Suposição 11 do PRD" por "Suposição 6 do PRD".
Recomendação estrutural: como o §16 se declara reordenável por risco, citar por âncora
("Suposição da Regra de Frete, §16") em vez de por número.

---

### A-7. O glossário define **Estoque** como o disponível; a FR-11 o define como o total do qual se subtraem as Reservas

**(A)** `prd.md` §3 Glossário:

> **Estoque** — quantidade de unidades de um Produto **disponíveis para venda**. É **reduzido por Reserva de Estoque**, não diretamente pela venda.

**(B)** `prd.md` FR-11, primeira consequência testável:

> Estoque disponível = **Estoque total** menos as Reservas de Estoque ativas.

Sob (A), Estoque **já é** o líquido de reservas. Substituindo (A) em (B), "Estoque disponível" =
(total − reservas) − reservas: a Reserva entra duas vezes. Além disso, "**Estoque total**" — o
termo que a FR-11 usa como operando, e que reaparece em *"O Administrador não pode reduzir o
Estoque total abaixo das Reservas de Estoque ativas"* — **não existe no §3**, num documento cujo
§3 abre declarando: *"Termos de domínio definidos uma vez. O restante do PRD usa cada termo
literalmente."* O resto do documento usa "Estoque" no sentido de total, contra a definição:
FR-9 (*"Estoque negativo são recusados"*), FR-11 (*"Administrador pode ajustar o Estoque de um
Produto"*), §3 Produto (*"Tem nome, descrição, preço, imagem e Estoque"*).

**Correção mínima.** No §3, trocar a entrada por duas: **Estoque** — quantidade total de unidades
de um Produto cadastradas pelo Administrador; e **Estoque disponível** — Estoque menos as Reservas
de Estoque ativas (FR-11). Depois, trocar "Estoque total" por "Estoque" na FR-11.

---

## MÉDIO

### M-1. O Sumário Executivo afirma que toda omissão tem justificativa no §5 **e** no addendum §7; seis das oito não têm

**(A)** `prd.md`, Sumário Executivo:

> **O que fica de fora.** Portal do Vendedor, avaliações, lista de desejos, recomendações, carrinho anônimo, devolução, aplicativo móvel, publicação em nuvem. **Cada omissão está justificada no §5 e no addendum §7 — com o que se perdeu em cada uma.**

**(B)** As nove linhas de `addendum.md` §7 são: Marketplace completo · Portal do Vendedor mínimo ·
Loja única sem Vendedor · Gateway real em sandbox · Pagamento síncrono simulado · Carrinho anônimo
com fusão no login · Elasticsearch no MVP · Microsserviços desde o início · Divisão do Pedido por
Vendedor.

Cruzando as duas listas:

| Omissão citada no Sumário | Tem entrada no §5? | Tem linha no addendum §7 ("o que se perdeu")? |
|---|---|---|
| Portal do Vendedor | sim | sim |
| avaliações | sim | **não** |
| lista de desejos | sim | **não** |
| recomendações | **não** (só §6.2 e §10) | **não** |
| carrinho anônimo | **não** (só §6.2) | sim |
| devolução | sim | **não** |
| aplicativo móvel | sim | **não** |
| publicação em nuvem | sim | **não** |

Duas de oito satisfazem a promessa. O "com o que se perdeu em cada uma" é ainda mais estrito e
só existe na coluna 3 da tabela do addendum §7 — logo, vale para duas das oito.

**Correção mínima.** Trocar o período por: "Cada omissão está justificada no §5 ou no §6.2; as
que envolveram uma alternativa arquitetural rejeitada têm o que se perdeu registrado no addendum
§7."

---

### M-2. O §4.1 justifica a ausência de Carrinho anônimo pelos casos de borda de fusão; o addendum §7 diz explicitamente que a fusão não é o custo

**(A)** `prd.md` §4.1, Descrição:

> A fronteira é deliberadamente rígida — não existe Carrinho anônimo, o que elimina estado de sessão anônima e **todos os casos de borda de fusão de Carrinho no login**.

**(B)** `addendum.md` §7, linha "Carrinho anônimo com fusão no login":

> **O custo não é a fusão** — a FR-19 já revalida preço e disponibilidade, então fundir é somar quantidades respeitando o Estoque. O custo é o **estado anônimo com ciclo de vida próprio**: sessão de visitante, expiração, limpeza de carrinho órfão, tudo num sistema sem visitantes reais.

O addendum reexaminou a justificativa e derrubou metade dela — a metade que o §4.1 continua
alegando. É o mesmo padrão de reexame que o addendum aplicou, e sinalizou, na linha do "Portal
do Vendedor mínimo" (*"Reexaminado — a rejeição original estava mal fundamentada"*) e na do
"Gateway real em sandbox" (*"A justificativa antiga […] não se sustenta mais"*). Aqui o reexame
aconteceu no addendum e não voltou ao PRD.

**Correção mínima.** No §4.1, trocar por: "— não existe Carrinho anônimo, o que elimina o estado
de sessão anônima com ciclo de vida próprio: expiração e limpeza de carrinho órfão num sistema sem
visitantes reais (addendum §7)."

---

### M-3. A ordem de construção do addendum §8 particiona FR-1 a FR-33 e nunca menciona a FR-34

**(A)** `addendum.md` §8, os sete passos, com suas faixas: FR-1 a FR-5 · FR-6 a FR-11 · FR-28 ·
FR-16 a FR-19 · **FR-20 a FR-27** · FR-12 a FR-15 · FR-29 a FR-33. A união é exatamente
{FR-1 … FR-33}. A FR-34 não aparece em nenhum passo, e não aparece em nenhum outro lugar do §8.

**(B)** `prd.md` §4.5, índice de funcionalidades, que coloca a FR-34 dentro do Checkout:

> | 4.5 | Checkout | FR-20 Endereço · FR-21 Frete · FR-22 Revisão · FR-23 Criação do Pedido · FR-24 Reserva de Estoque · FR-25 Tentativa de Pagamento · FR-26 Aprovação · FR-27 Recusa · **FR-34 Expiração** |

O passo 5 do addendum é nomeado "**Checkout e pagamento assíncrono**" e a FR-34 é o requisito de
expiração do pagamento assíncrono — mas a faixa "FR-20 a FR-27" a exclui, e o próprio addendum
trata a FR-34 como estruturante em outros três pontos (§2 tabela, §2 invariante 6, §4 "os dois
casos que o mock precisa resolver"). Não é um requisito ausente do documento: é um requisito que
a ordem de construção deixou órfão porque a faixa foi escrita antes de ele existir. É o efeito
colateral direto da numeração global fora de ordem que o §4 assume (*"reorganizar as
funcionalidades não renumera nada"*).

**Correção mínima.** No passo 5 do addendum §8, trocar "(FR-20 a FR-27)" por "(FR-20 a FR-27 e
FR-34)".

---

### M-4. O glossário faz a Regra de Frete operar sobre Itens de Pedido; o checkout calcula o Frete antes de o Pedido existir

**(A)** `prd.md` §3 Glossário:

> **Regra de Frete** — política que converte o CEP de um Endereço e o valor dos **Itens de Pedido** em um valor de Frete.

**(B)** `prd.md` FR-23 — o momento em que Itens de Pedido passam a existir:

> Confirmar o checkout **cria** um Pedido em `AGUARDANDO_PAGAMENTO` a partir do Carrinho.

**(C)** Os quatro requisitos que rodam a Regra de Frete **antes** da FR-23:

> FR-19: "Se a revalidação alterar o subtotal, o **Frete é recalculado antes da confirmação** — inclusive quando a mudança faz o subtotal cruzar o limiar de isenção (ver FR-21)."
> FR-20: "Trocar o Endereço **recalcula o Frete antes da confirmação**."
> FR-21: "O sistema calcula o Frete pela Regra de Frete a partir do CEP do Endereço escolhido e do **valor dos itens**."
> FR-22: "A revisão mostra cada **Item de Carrinho** com preço unitário e subtotal, o Endereço escolhido, o Frete e o total."

Em todos os quatro, o insumo da Regra de Frete é o Carrinho, não o Pedido. A definição do §3
descreve a Regra de Frete como se ela só rodasse depois da criação do Pedido — o que tornaria a
FR-22 (mostrar o Frete na revisão) e a FR-19 (recalcular ao cruzar o limiar de isenção)
impossíveis de satisfazer literalmente. A entrada seguinte do §3 já usa o vocabulário certo:
*"**Frete** — valor de entrega calculado pela Regra de Frete e **congelado no Pedido**"* — calculado
antes, congelado depois.

**Correção mínima.** No §3, trocar "o valor dos Itens de Pedido" por "o valor dos itens em
compra (Itens de Carrinho no checkout, Itens de Pedido depois de congelado)".

---

### M-5. A verificação da NFR-15 é mais larga que a própria NFR-15, e colide com a NFR-1 e com as regras de ensaio do §12

**(A)** `prd.md` NFR-15, o requisito e sua verificação, no mesmo parágrafo:

> **NFR-15 — Operação offline na demonstração.** O ambiente sobe e o fluxo completo funciona sem acesso à internet: nenhuma imagem, fonte, folha de estilo, script ou dependência é buscada em rede externa **em tempo de execução**. Verificação: **desconectar a rede e percorrer o Roteiro A (§12) inteiro**, incluindo as imagens dos Produtos.

**(B)** `prd.md` §12, Roteiro A, passo 1 — o primeiro passo do roteiro que a verificação manda
percorrer *inteiro*:

> 1. Subir o ambiente em um comando, com Catálogo Semeado (NFR-1).

**(C)** `prd.md` NFR-1 e §12, que fixam o ponto de partida como repositório sem estado local:

> NFR-1: "O sistema completo […] sobe com um único comando em contêineres, **a partir do repositório limpo**."
> §12, Regras de ensaio: "Os três roteiros são ensaiados **a partir de clone limpo do repositório**, por ao menos duas pessoas diferentes, antes da apresentação."

Compondo os três: o ensaio parte de clone limpo, a verificação da NFR-15 corta a rede e exige o
Roteiro A inteiro, e o passo 1 do Roteiro A é subir os contêineres — o que, a partir de clone
limpo, exige baixar imagens base e dependências. O requisito propriamente dito se protege com a
qualificação "em tempo de execução", que exclui construção e download de imagem; a **verificação**
não repete a qualificação e engloba o passo 1. Requisito e procedimento de verificação, no mesmo
parágrafo, cobrem escopos diferentes — e a leitura literal do procedimento é insatisfazível.

**Correção mínima.** Fechar a verificação da NFR-15 no escopo do requisito: "Verificação: com o
ambiente já construído (imagens e dependências presentes localmente), desconectar a rede, subir
os contêineres e percorrer os passos 2 a 9 do Roteiro A, incluindo as imagens dos Produtos."

---

### M-6. A verificação da FR-34 exige que nenhum Pedido esteja em `AGUARDANDO_PAGAMENTO`, estado em que a FR-25 e a FR-27 colocam Pedidos legitimamente

**(A)** `prd.md` FR-34, quarta consequência testável:

> Nenhum Pedido permanece em `AGUARDANDO_PAGAMENTO` por mais tempo que o prazo de expiração. **Verificação: varrer os Pedidos após o prazo e não encontrar nenhum.**

**(B)** `prd.md` FR-25 e FR-27, que produzem Pedidos nesse estado a qualquer momento:

> FR-25: "A resposta da requisição de checkout não espera o resultado do pagamento: o Comprador vai para a tela do Pedido em estado 'processando'."
> FR-27: "O Comprador pode **iniciar uma nova Tentativa de Pagamento** a partir do Pedido" — que, pela tabela do addendum §2, é a transição `PAGAMENTO_RECUSADO` → `AGUARDANDO_PAGAMENTO`.

A frase do requisito está certa; a frase de verificação perdeu o qualificador temporal. Uma
varredura executada "após o prazo" encontrará, corretamente, todo Pedido criado nos últimos
segundos e todo Pedido que acabou de receber nova Tentativa de Pagamento — e falhará. O teste,
como escrito, falha quando o sistema está certo. Isso é agravado pelo fato de que a FR-34 e a
FR-27 juntas criam um ciclo `AGUARDANDO_PAGAMENTO` ⇄ `PAGAMENTO_RECUSADO`, e um Pedido no meio
desse ciclo é indistinguível, para essa varredura, de um Pedido travado.

**Correção mínima.** Trocar a verificação por: "Verificação: varrer os Pedidos e não encontrar
nenhum em `AGUARDANDO_PAGAMENTO` **cuja última transição registrada seja mais antiga que o prazo
de expiração**."

---

### M-7. O balde 🟡 do §16 e o addendum §6 classificam como "alto impacto / reversão barata" suposições que o próprio §16 rotula "impacto médio / reversão média"

**(A)** `prd.md` §16, cabeçalho do segundo balde:

> **🟡 Alto impacto, reversão barata — construir de forma que a correção continue barata**

**(B)** Os cinco itens que vivem sob esse cabeçalho, com os rótulos que eles mesmos carregam:

> 2. […] *(Confiança média · **impacto alto** · reversão barata.)*
> 3. […] *(Confiança média · **impacto médio** · reversão barata.)*
> 4. […] *(Confiança média-alta · **impacto médio** · **reversão média**.)*
> 5. […] *(Confiança média-alta · **impacto médio** · reversão barata.)*
> 6. […] *(Confiança média · **impacto médio** · reversão barata.)*

Um de cinco itens satisfaz o cabeçalho; o item 4 falha nas duas dimensões.

**(C)** `addendum.md` §6, que herda o erro e o estreita para dois itens:

> A auditoria de suposições (§16 do PRD) encontrou **duas de alto impacto** cuja **reversão fica barata** se o modelo de dados nascer preparado.

— sendo as duas nomeadas logo abaixo "*Pedido não dividido por Vendedor* (suposição 2)" e
"*Categoria plana* (suposição 4)". A suposição 4 é, pelo §16, impacto **médio** e reversão
**média**. É exatamente por a reversão dela **não** ser barata que a blindagem da coluna nula
proposta pelo addendum §6 se justifica — o texto do addendum destrói o próprio argumento ao
afirmar que a reversão já é barata.

**Correção mínima.** No §16, trocar o cabeçalho por "🟡 Impacto médio a alto — construir de forma
que a correção continue barata". No addendum §6, trocar por: "encontrou duas cuja reversão só fica
barata se o modelo de dados nascer preparado".

---

### M-8. Disciplina de glossário: "item" no lugar de Produto / Item de Pedido em seis pontos, dois deles no Sumário Executivo

**A regra**, `prd.md` §3, cabeçalho:

> *Termos de domínio definidos uma vez. O restante do PRD usa cada termo literalmente; sinônimo é violação de disciplina — em particular, nunca "compra" no lugar de **Pedido**, nunca "**item**" no lugar de **Produto**.*

**As violações** (só as em que "item" substitui um termo do glossário; usos genéricos como
"cada item tem limiar" no §7 e "qualquer item que exija cartão de crédito" no §8 não contam):

| Onde | Texto | Termo correto |
|---|---|---|
| Sumário Executivo, "A tese" | "cancelamento que devolve **o item** à prateleira" | unidades do **Produto** (é o vocabulário da FR-31: "as unidades voltam ao Estoque disponível") |
| Sumário Executivo, decisão estrutural 1 | "o Pedido registra de quem veio **cada item**" | **cada Item de Pedido** — que é literalmente como o §4.2 escreve a mesma frase: "o Pedido registra de quem veio cada Item de Pedido" |
| §1 Visão | "um cancelamento que devolve **o item** à prateleira" | idem primeira linha |
| §2.3, UJ-2 | "abre o detalhe: **os itens** que comprou" | **Itens de Pedido** (é como a FR-30 escreve: "O detalhe mostra Itens de Pedido com preço praticado") |
| FR-21 (×2) | "do **valor dos itens**" / "o mesmo **valor de itens**" | subtotal dos **Itens de Carrinho** (ver M-4) |
| FR-24 | "Se **qualquer item** não tiver disponível suficiente no instante da criação" | **qualquer Item de Pedido** |

As duas primeiras são as mais graves porque estão no Sumário Executivo — a página que o §0 e o
Mapa de Leitura mandam o avaliador ler primeiro, e que foi escrita depois do glossário que ela
viola. A do §4.2 prova que o texto correto existe no documento: o Sumário parafraseou-o e perdeu
o termo.

**Correção mínima.** Seis substituições pontuais, conforme a coluna direita da tabela.

---

## BAIXO

### B-1. A Questão em Aberto 3 lista quatro suposições dependentes; só três carregam o marcador de dependência

`prd.md` §15, item 3:

> Quatro suposições do §16 dependem exclusivamente desta resposta — Portal do Vendedor fora do MVP (n.º 1), ausência de verificação de e-mail (n.º 7), ausência de devolução e reembolso (n.º 8) e **Pedido não dividido por Vendedor (n.º 2)**.

Os itens 1, 7 e 8 do §16 terminam com a frase "*Depende da Questão em Aberto 3.*"; o item 2 não —
ele termina em "*separar por Vendedor depois é agrupamento de consulta, não remodelagem.*"

**Correção mínima.** Acrescentar "Depende da Questão em Aberto 3." ao fim do item 2 do §16.

---

### B-2. "Dois dos três roteiros" dependem da máquina de estados; os três dependem

`prd.md` §14, primeiro risco: *"Ela é pré-requisito de quatro funcionalidades (§4.6) e de **dois dos
três roteiros** de demonstração."* Repetido em `addendum.md` §8, passo 3: *"É o pré-requisito de
quatro funcionalidades e de **dois roteiros de demonstração**."*

Pelo §12: o Roteiro A depende (passos 7–9, FR-23/FR-25/FR-26/FR-30/FR-33 — transições e linha do
tempo), o Roteiro B depende (passos 2 e 4, FR-27 — a transição para `PAGAMENTO_RECUSADO`) e o
Roteiro C depende (passos 2–4, FR-32/FR-31 — avanço manual, cancelamento e recusa de transição
inválida). Três de três.

**Correção mínima.** Trocar por "dos três roteiros de demonstração" nos dois lugares.

---

### B-3. O §12 declara que todo passo é sustentado por um FR; dois passos são sustentados só por NFR

`prd.md` §12, preâmbulo: *"**Todo passo aqui é sustentado por um FR**; se algum passo não tiver FR,
falta requisito."* Mas o Roteiro A, passo 1 cita apenas "(NFR-1)" e o Roteiro C, passo 5 cita
apenas "(NFR-7)".

**Correção mínima.** Trocar por "Todo passo aqui é sustentado por um FR ou por um NFR verificável".

---

### B-4. O §11 condiciona a extração de serviço à SM-C1, que fala de outra coisa

`prd.md` §11: *"É **opcional e explicitamente condicionado**: nenhuma extração começa antes de os
três roteiros do §12 passarem (**ver SM-C1**)."* A SM-C1 é *"Quantidade de telas e funcionalidades
entregues […] Nenhuma **funcionalidade nova** entra enquanto os três roteiros não passarem."* Uma
extração de serviço não é funcionalidade nova — é refatoração de fronteira; a SM-C1 literalmente
não a alcança. A métrica que enuncia o gate é a SM-1, e o próprio parágrafo do §11 já a cita duas
frases antes: *"fechado o ciclo e **passando SM-1**, extrair exatamente um serviço"*.

**Correção mínima.** Trocar "(ver SM-C1)" por "(ver SM-1)".

---

### B-5. Duas violações menores de literalidade do glossário, no §3 e no §1

(a) `prd.md` §3 define o Pedido com o sinônimo que o mesmo §3 proíbe três linhas acima:
*"**Pedido** — registro de uma **compra** concluída ou em andamento."* Uso definicional, portanto o
mais brando dos casos, mas é a única entrada do glossário que se define pelo termo banido.

(b) O §1 escreve em minúsculas os termos que o Sumário Executivo — redigido depois, sobre a mesma
frase — escreve com o vocabulário do glossário:

> §1: "uma pessoa entra, encontra um **produto**, coloca no **carrinho**, paga" / "cinco que sustentam um **pedido** do clique ao 'entregue' — com **estoque** que baixa"
> Sumário: "Uma pessoa entra, encontra um **Produto**, coloca no **Carrinho**, paga" / "cinco que sustentam um **Pedido** do clique ao 'entregue' — com **Estoque** que baixa"

**Correção mínima.** Em (a), "registro de um ato de aquisição concluído ou em andamento". Em (b),
capitalizar os quatro termos no §1 para casar com a versão do Sumário.

---

### B-6. IDs definidos e nunca referenciados em lugar nenhum dos dois documentos

**NFR-10** (Responsividade), **NFR-11** (Acessibilidade básica), **NFR-13** (Precisão monetária),
**SM-C2** e **SM-C3** aparecem exatamente uma vez cada — na própria definição. Nenhum FR, roteiro,
risco, métrica ou seção do addendum os cita.

Registrado por completude da varredura de numeração, não como contradição: NFR transversal sem
citação não é defeito. Vale um apontamento, no entanto, porque a **NFR-13** tem um citador natural
que a menciona sem nomeá-la — `addendum.md` §8, passo 0: *"quando ainda há tempo de descobrir que
o decimal virou ponto flutuante"* — e o §14 repete a mesma imagem, também sem o ID.

**Correção mínima (opcional).** Acrescentar "(NFR-13)" às duas menções ao ponto flutuante e
"(NFR-10, NFR-11)" ao passo 0 do addendum §8.

---

## Categorias sem achados

**Integridade de numeração — nenhum defeito.** FR-1 a FR-34 estão todos definidos exatamente uma
vez (34 cabeçalhos `#### FR-N`, conferido por varredura), sem lacuna e sem duplicata, e todos
constam do índice do §4; a FR-34 está fora de ordem por design declarado no §4 ("reorganizar as
funcionalidades não renumera nada"). NFR-1 a NFR-15 completos e únicos — a contagem "quinze não
funcionais" do Sumário está correta, e "trinta e quatro requisitos funcionais" também. SM-1 a SM-6
e SM-C1 a SM-C3 completos e únicos (SM-6 aparece entre as Primárias e SM-4/SM-5 entre as
Secundárias, o que é ordenação por classe, não lacuna). UJ-1 a UJ-4 completos e todos referenciados.
**Nenhum ID é citado sem estar definido** em nenhum dos dois documentos.

**Renumeração das seções do addendum — uma única vítima.** Varridas todas as citações a seções do
addendum nos dois arquivos (8 no `prd.md`, 0 auto-referências numéricas no `addendum.md`), todas
apontam corretamente para a numeração atual: §2 (máquina de estados, citada em §12), §3
(concorrência, §12), §4 (fronteira de pagamento, §12), §7 (alternativas descartadas, citada 4×) e
§8 (ordem de construção, citada 2×). Nenhuma sobrou apontando para a numeração antiga. O único
ponteiro envelhecido do addendum é para o **§16 do PRD**, não para si mesmo (A-6).

**Índice de suposições — íntegro.** As 14 marcações `[SUPOSIÇÃO: …]` do corpo do `prd.md` mapeiam
para os 13 itens do §16 (as do §5 e do §10 sobre o Portal do Vendedor convergem no item 1), e as 13
referências de seção do §16 (§3, §4.1, §4.2, §4.4, §4.5, §4.6, §5/§10 e os FRs citados) apontam
todas para o lugar certo.

**Diagrama da máquina de estados × addendum §2 × FR-28/31/32/33/34 — consistentes.** Transições,
origens de `CANCELADO`, terminais e o caminho de reentrada `PAGAMENTO_RECUSADO` →
`AGUARDANDO_PAGAMENTO` batem entre o diagrama ASCII do §4.6, a tabela do addendum §2 e os cinco FRs
envolvidos. A única divergência da tabela é a de efeito sobre Estoque, tratada em C-1.

---

## Veredicto

O corpo normativo — FRs, NFRs, máquina de estados, invariantes do addendum — é notavelmente
coerente: numeração íntegra, referências cruzadas quase todas corretas, e a renumeração do addendum
foi propagada por completo. O dano das cinco passadas de revisão concentrou-se nas **camadas
narrativas escritas por último ou reescritas por cima**: o Sumário Executivo (M-1, M-8), as jornadas
do §2.3 que não acompanharam os FRs que elas mesmas motivaram (A-3, A-4), as justificativas
reexaminadas no addendum que não voltaram ao PRD (M-2), o §16 reordenado por risco (A-6, M-7) e as
faixas de FR escritas antes de a FR-34 existir (M-3). O único defeito com consequência de
implementação direta é C-1, e ele mora justamente na costura entre os dois documentos.

**Arquivo:** `/Users/lucassung/Git/azamon/_bmad-output/planning-artifacts/prds/prd-azamon-2026-08-14/review-consistency.md`
