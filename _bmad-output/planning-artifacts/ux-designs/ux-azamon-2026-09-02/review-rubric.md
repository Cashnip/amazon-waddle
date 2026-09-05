# Revisão do par de espinhas — Azamon

*Lente: rubric walker. Validação do par `DESIGN.md` + `EXPERIENCE.md` como contrato para consumidores a jusante (arquitetura, story-dev — humano ou IA). A pergunta é uma só: dá para extrair da fonte sem voltar ao PRD, com toda referência resolvendo e toda decisão que carrega peso já tomada?*

## Veredito geral

O par é utilizável como contrato. A forma está correta nos dois arquivos, os tokens resolvem sem exceção, o glossário de vinte termos do §3 é usado literalmente sem um único desvio, e as quatro jornadas chegam completas com protagonista, passos, clímax e caminho de falha. As três seções inventadas — *A máquina de estados na interface*, *Dinheiro e Estoque na tela* e *Modo demonstração* — são o melhor material do par e resolvem exatamente o que as seções padrão não alcançariam.

Os buracos estão concentrados em dois lugares e são do mesmo tipo: **superfícies que o PRD nomeia mas a espinha trata como cenário** — Login, Cadastro, Recuperação de senha, o menu de conta e a navegação administrativa — e **consequências de FR que nunca viraram estado**. Nenhum é crítico; seis são altos, e todos os seis são de omissão, não de contradição interna. Um consumidor a jusante consegue construir o fluxo de compra inteiro a partir daqui; ele não consegue construir a porta de entrada nem o chrome de conta sem inventar.

Uma agravante contextual: `mockups/`, `wireframes/` e `imports/` estão vazios (caminho rápido). Não há material visual algum, então a prosa das espinhas é a **única** fonte. Isso não é defeito do par, mas eleva o custo de cada omissão listada abaixo.

## 1. Cobertura de jornadas — forte

Extraídas as quatro UJ do §2.3 do PRD e conferidas contra `EXPERIENCE.md.Key Flows`. As quatro estão presentes com **nome verbatim** da fonte, protagonista nomeado (Marina em UJ-1/2/3, Rafael em UJ-4), passos numerados (6/3/3/4), batida de clímax marcada em todas, e caminho de falha onde a fonte tem um — `Falha:` em UJ-1 (recusa de pagamento) e `Caso de borda:` em UJ-3 e UJ-4, espelhando os casos de borda do PRD. UJ-2 não tem caminho de falha, e a fonte também não tem. Nada a corrigir na estrutura.

Varridos também os 34 FR e 16 NFR em busca de consequência com superfície de interface que não tenha casa em nenhuma das espinhas. O fluxo de compra (FR-6 a FR-34) está coberto. Os furos estão na fronteira Público → Comprador e em duas delegações explícitas do PRD.

### Achados

- **alto** O portão do visitante não autenticado não existe em lugar nenhum das espinhas. A FR-16 e a FR-7 declaram a mesma regra duas vezes — "visitante que tenta adicionar ao Carrinho é levado ao login e, ao autenticar, **retorna à página do Produto de origem**" — e é a única passagem da área Público para a área Comprador. A linha `Botão Adicionar ao Carrinho` de `EXPERIENCE.md.Component Patterns` (l. 110) descreve só o caso autenticado; a tabela de IA gesticula com "porta de Comprador" (l. 35) sem definir regra; e o estado `Sessão expirada` (l. 148) cobre um caso diferente (sessão que existia e caiu). *Correção:* uma linha em `Component Patterns` no `Botão Adicionar ao Carrinho` — sem Sessão, leva ao Login preservando o Produto de origem e a quantidade escolhida, e volta à Página de Produto após autenticar — e um estado correspondente em `State Patterns`.

- **médio** Recuperação de senha é superfície de IA (l. 35) com zero comportamento e zero estado. A FR-3 tem três consequências com efeito direto de interface: resposta idêntica para e-mail cadastrado e não cadastrado, token de uso único recusado quando usado ou expirado, e invalidação dos tokens anteriores. Nenhuma aparece. *Correção:* duas linhas em `State Patterns` — solicitação enviada (resposta neutra) e token inválido/expirado. Se a FR-3 for cortada pela ordem do §6.3, dizer isso na espinha em vez de deixar a superfície muda.

- **médio** A `[NOTA PARA O PM]` do §4.6 do PRD delega explicitamente à etapa de UX uma decisão, e ela volta pela metade. A colisão entre a simulação de entrega e o painel do Administrador é resolvida no estado `Transição inválida` (l. 146) com o `Alert` destrutivo — correto. Mas *Modo demonstração* fecha a porta que faria a colisão visível: "listas administrativas atualizam por ação do usuário" (l. 158). O Administrador então olha uma linha `PAGO` obsoleta, o `Dialog` de confirmação nomeia um destino derivado de dado velho, e a espinha não diz que a tabela se atualiza depois do `Alert`. *Correção:* uma frase na linha `Tabela de Pedidos` de `Component Patterns` — o `Alert` de transição inválida recarrega a linha afetada com o estado atual do servidor.

- **baixo** FR-14 Ordenação aparece três vezes como estado de URL (l. 33, 115, 222) e nunca como controle. As três opções (preço crescente, preço decrescente, mais recentes) não são nomeadas em nenhuma espinha, e a linha `Barra de filtros` especifica chips para filtros sem dizer nada sobre a forma ou a posição do seletor de ordenação. *Correção:* nomear as três opções na linha `Barra de filtros` e dizer se o controle vive dentro do `Sheet` de filtros abaixo de `md` ou fora dele.

## 2. Completude de tokens — forte

Extraídos do frontmatter de `DESIGN.md`: 9 tokens de cor (todos com hex), 3 papéis tipográficos, 4 níveis de `rounded`, 5 tokens nomeados de `spacing`, 11 objetos em `components`. Extraídas 43 referências `{path.to.token}` — 41 em `DESIGN.md`, 2 em `EXPERIENCE.md` — cobrindo 18 caminhos distintos. **Todas resolvem.** Nenhum token de cor sem hex.

Alvos de contraste declarados para as combinações que carregam peso: `#FFD814` sobre `#0F1111` ("AA com folga larga"), `#007185` sobre branco ("4,6:1, passa AA"), foco herdado do `ring` do shadcn ("verificado em 3:1"), e o piso geral de 4,5:1 / 3:1 na `Accessibility Floor`. A regra de desvio consciente — não copiar o cinza pequeno da Amazon — está declarada nas duas espinhas e nos *Do's and Don'ts*.

### Achados

- **alto** `{colors.available}` é usado como preenchimento sem par de primeiro plano. A linha `Selo de Status do Pedido` (`DESIGN.md` l. 165) diz: os quatro estados normais usam `secondary`, `PAGAMENTO_RECUSADO` e `CANCELADO` usam `destructive`, e "`ENTREGUE` usa `{colors.available}`". Os dois primeiros nomeiam variantes de `Badge` do shadcn, que são preenchimentos com primeiro plano próprio; o terceiro nomeia uma cor solta. Não existe `available-foreground`, embora toda outra cor de preenchimento da paleta declare seu par (`chrome-foreground`, `primary-foreground`, `primary-strong-foreground`). Um consumidor tem que adivinhar se o selo `ENTREGUE` é texto verde sobre fundo neutro ou texto branco sobre verde — e, no segundo caso, qual é o texto. *Correção:* declarar `available-foreground` no frontmatter e dizer na linha se `ENTREGUE` é preenchimento ou texto, com o alvo de contraste da combinação escolhida.

- **baixo** `busca-global.background: '#FFFFFF'` é o único valor de componente com hex cru; todos os outros referenciam `{colors.*}`. *Correção:* declarar um token (`surface-input` ou equivalente) ou apontar para o token `background` do shadcn em prosa, como o resto do documento faz.

- **baixo** 1440px é o único número que as duas espinhas compartilham e é o único sem token. Aparece em `DESIGN.md.Layout & Spacing` (l. 129) e três vezes na tabela `Responsive & Platform` de `EXPERIENCE.md`. *Correção:* `spacing.max-content: 1440px` no frontmatter, e as duas espinhas o referenciam.

- **baixo** Os três papéis tipográficos da camada de marca são citados em prosa como `` `preco` ``, `` `preco-centavos` `` e `` `wordmark` `` — nunca como `{typography.preco}`. Não há uma única referência `{typography.*}` no par. *Correção:* usar a sintaxe de referência, para que o resolvedor de tokens alcance os papéis de tipografia como alcança cores e espaços.

## 3. Cobertura de componentes — adequada

Extraídos todos os nomes de componente usados em qualquer lugar das duas espinhas. **Quinze componentes nomeados têm as duas especificações** — linha visual em `DESIGN.md.Components` (ou declaração explícita de "sem delta de marca", que é uma especificação válida) e linha comportamental em `EXPERIENCE.md.Component Patterns`. Os nomes são **idênticos caractere a caractere** entre os dois arquivos e entre todas as seções onde aparecem. Nenhuma linha é descrição de uma palavra; todas carregam regra real. Os quinze componentes do shadcn herdados sem alteração estão declarados nominalmente.

O que falta são quatro elementos que as espinhas *usam* — em tabela de IA, em anatomia de componente, em seção de inspiração — sem nunca especificar.

### Achados

- **alto** "Menu da conta" é o ponto de entrada declarado de três superfícies de IA (Meus pedidos, Meus endereços, Perfil — l. 39 e 41) e não existe em mais lugar nenhum das duas espinhas. Pior: a anatomia da `Barra superior` em `DESIGN.md` (l. 153) enumera o conteúdo dela — wordmark à esquerda, busca no centro, Carrinho com contador à direita — e não inclui elemento de conta algum. A tabela de IA também diz que Login se chega pela "Barra superior". As duas espinhas se contradizem: a IA exige um afordância na barra que a especificação visual da barra exclui. *Correção:* acrescentar o elemento de conta à anatomia da `Barra superior` no `DESIGN.md` e uma linha `Menu da conta` em `Component Patterns` dizendo o que ele contém com Sessão (Meus pedidos, Meus endereços, Perfil, sair) e sem Sessão (Entrar, Criar conta).

- **alto** "Navegação administrativa" é o ponto de entrada declarado das quatro superfícies administrativas (l. 42 e 43) e é definida apenas por negação: UJ-4 passo 1 diz "sem barra superior da loja, sem busca global". Nenhuma linha visual, nenhuma regra comportamental, nenhuma forma. A área administrativa é um terço da IA e o palco inteiro da UJ-4, e um consumidor a jusante não tem por onde começar o chrome dela. *Correção:* uma linha em `DESIGN.md.Components` e uma em `Component Patterns` — lateral ou topo, cor (o `{colors.chrome}` da loja ou o neutro do shadcn), os quatro destinos, e o comportamento abaixo de `md`.

- **médio** O breadcrumb de Categoria é nomeado como uma das **três marcas herdadas da Amazon** que constituem a assinatura visual inteira do produto (`EXPERIENCE.md` l. 206, e `DESIGN.md` l. 104 lhe dá a cor `{colors.link}`) e não tem linha em nenhuma das duas tabelas de componente. Uma das três decisões estruturais de marca do projeto não tem especificação. *Correção:* linha em ambas as espinhas — em que superfícies aparece (Página de Produto? Resultados?), o que mostra, e para onde cada segmento leva.

- **médio** O rodapé aparece uma única vez, em `DESIGN.md.Colors` (l. 100), recebendo `{colors.chrome}`. Não está na IA, não tem linha de componente, não tem conteúdo especificado. Ou é componente e precisa de linha, ou não existe e não deveria receber cor. *Correção:* especificar ou remover a menção.

## 4. Cobertura de estados — adequada

Percorridas as 12 linhas da tabela de IA de `EXPERIENCE.md`, cruzadas com as três listas do §9 do PRD — todas as superfícies do §9 estão representadas, incluindo "ajustar Estoque" colapsado dentro de Produtos. Para cada superfície foram listados os estados aplicáveis (vazio, carregamento frio, erro, negação) e conferidos contra `State Patterns`.

A tabela de estados é a seção mais trabalhada do par: 24 linhas, com quatro estados globais bem escolhidos (`Sessão expirada`, `Erro do servidor` com identificador de correlação, `Recurso de outro Comprador` tratado como não-encontrado por causa do NFR-6, `Conta bloqueada`). O fluxo de compra está coberto de ponta a ponta, incluindo os quatro estados de pagamento (recusado, expirado, esgotado, cancelamento) que são o material do Roteiro B.

Os furos restantes são de dois tipos: consequências de FR que nunca viraram linha, e um caso onde a tabela de estados e a tabela de voz se contradizem.

### Achados

- **alto** Login com credencial inválida não tem estado, e a regra de voz vigente leva o consumidor a violar a FR-2. A FR-2 exige "a mesma mensagem genérica para e-mail inexistente e para senha errada — sem revelar qual dos dois falhou". A tabela `Voice and Tone` (l. 61) estabelece o oposto como princípio: "Nomear o motivo sempre que o sistema souber" contra "Mensagem genérica quando o motivo existe". O estado `Conta bloqueada` acerta ao dizer "Sem revelar se o e-mail existe", mas cobre só o bloqueio. Um desenvolvedor seguindo a espinha escreve "E-mail não encontrado" e quebra um requisito de segurança. *Correção:* linha em `State Patterns` para credencial inválida, com a mensagem literal, **e** uma exceção declarada na tabela `Voice and Tone` — nomear o motivo, exceto onde nomear vaza existência de conta.

- **alto** Vitrine com Catálogo Semeado vazio não tem estado. É consequência testável explícita da FR-6 ("a vitrine responde com Catálogo Semeado vazio sem quebrar — mostra estado vazio explicativo"), a Vitrine é a primeira linha da tabela de IA, e o NFR-1 diz que o Catálogo Semeado "é a primeira coisa que a banca vê". A tabela cobre o carregamento frio da Vitrine e o vazio de busca, mas não o vazio da própria vitrine. *Correção:* uma linha — a UJ-4 passo 2 já descreve o momento em que Rafael abastece uma loja vazia, então o estado tem dono narrativo.

- **médio** A lista de Pedidos do Administrador filtrada sem resultado não tem estado. A linha `Lista administrativa vazia` (l. 137) nomeia só "Vendedores, Categorias, Produtos" e sua ação única é "criar" — que não faz sentido para Pedidos. Filtrar por `PAGO` é literalmente o passo 3 da UJ-4. *Correção:* linha própria, com ação de limpar o filtro em vez de criar.

- **médio** A guarda de Estoque da FR-11 não tem estado. "O Administrador não pode reduzir o Estoque total abaixo das Reservas de Estoque ativas; a tentativa é recusada **informando quantas unidades estão comprometidas**." A mensagem tem conteúdo obrigatório e a linha `Erro de validação administrativa` (l. 138) cobre limites de campo do §7.1, não esta regra de domínio. *Correção:* linha própria, e é uma boa ocasião para a única exceção à regra de *Dinheiro e Estoque na tela* de nunca nomear a Reserva de Estoque — para o Administrador ela é vocabulário legítimo.

- **médio** A aprovação tardia sobre Pedido cancelado não aparece no painel do Administrador. A FR-26 e a invariante 7 do addendum exigem que o Pedido "passe a aparecer **sinalizado no painel do Administrador** como pagamento aprovado sobre Pedido cancelado" — é o único requisito do PRD que pede um marcador visual específico no painel, e o addendum o chama de risco de perda silenciosa de informação. A seção *A máquina de estados na interface* trata `CANCELADO` como "Nenhuma ação" nas duas colunas. *Correção:* linha em `State Patterns` e uma célula na tabela de estado/ação, dizendo como o Pedido é sinalizado na `Tabela de Pedidos`.

- **baixo** Perfil é linha de IA (l. 41) e nunca mais é mencionado — sem estado, sem componente, sem conteúdo. *Correção:* dizer o que a superfície contém, ou fundir com Meus endereços se não contém nada além.

- **baixo** Validação de campo do lado do Comprador não tem padrão. `Erro de validação administrativa` declara erro em linha por campo com o limite nomeado, mas só para "Formulários do Administrador". Cadastro (senha < 8 caracteres, e-mail sem formato válido — FR-1) e Endereço (CEP inválido — FR-5) ficam sem padrão declarado, embora `E-mail já cadastrado` mostre a forma pretendida. *Correção:* generalizar a linha para todo formulário, em vez de restringi-la à área administrativa.

- **baixo** FR-22 "Carrinho vazio não permite chegar à revisão" não tem tratamento. O estado `Carrinho vazio` cobre a superfície Carrinho; o bloqueio de entrada no checkout não está declarado. *Correção:* uma cláusula na linha `Passos do checkout`.

- **baixo** Carregamento frio só é tratado em Vitrine, Resultados e Página de Produto. Meus pedidos, Detalhe do Pedido e as listas administrativas são todas paginadas e carregadas do servidor, e nenhuma declara `Skeleton`. *Correção:* estender a linha `Carregamento inicial` às listas, ou declarar explicitamente que só a loja pública recebe esqueleto e por quê.

## 5. Cobertura de referências visuais — forte

`mockups/` e `wireframes/` não existem no workspace. `imports/` existe e está vazia, como o `.memlog.md` registra em `[ASSUMPTION]` (entrada 12). `.working/` está vazia. Não há órfãos porque não há arquivos, e não há referência inespecífica porque não há referência.

A regra de precedência está declarada **uma vez**, no bloco de citação de abertura de `EXPERIENCE.md` (l. 14), e está declarada melhor do que nos exemplos de referência: ela separa os dois eixos — "o PRD vence sobre as espinhas em matéria de requisito; as espinhas vencem em matéria de design e comportamento" — o que resolve antecipadamente a única ambiguidade real do arranjo. `DESIGN.md` não repete, o que é correto.

Nada a corrigir. A consequência a registrar é que as espinhas são a fonte visual **única**: cada omissão das seções 3 e 4 acima custa mais do que custaria se houvesse um mock para consultar.

## 6. Excesso e superespecificação — forte

O par é denso e quase todo o texto é carregado. `DESIGN.md` usa voz editorial, que a especificação permite, e cada parágrafo termina numa decisão. `EXPERIENCE.md` é predominantemente tabelar, e a voz que sobra está nas seções onde os exemplos de referência também a têm (clímax de Key Flows, `Inspiration & Anti-patterns`). Não há restatement de personas, FR ou escopo; as citações do §7.1 são ponteiros com número, não duplicação. Nenhuma medida em pixel onde um token cobriria — as poucas literais (`1px` de borda, marcador de `8px`, 60% de opacidade) não têm token correspondente na escala declarada.

### Achados

- **baixo** A rejeição dos frameworks de "taste" ocupa espaço nas **duas** espinhas — `DESIGN.md.Brand & Style` (l. 92) e um item longo em `EXPERIENCE.md.Inspiration` (l. 210) com o argumento sobre a seção 13 do skill. É nota de processo sobre a ferramentaria desta execução, não decisão que arquitetura ou story-dev leiam, e já está registrada em três entradas do `.memlog.md` (15–17). *Correção:* cortar de `DESIGN.md` e reduzir o item de `EXPERIENCE.md` a uma linha sobre a postura de réplica assumida, sem a arqueologia do skill.

- **baixo** Voz editorial vazando para fora de Key Flows e Inspiration em `EXPERIENCE.md`: "o único relógio honesto da interface" (l. 65), "é o teste mais rápido de que o sistema não é maquete" (l. 96), "ou vira quatro telas que não conversam" (l. 69). As regras que essas frases justificam já se sustentam sozinhas. *Correção:* manter a regra, cortar a justificativa — ela pertence ao `DESIGN.md` ou ao memlog.

## 7. Disciplina de herança — forte

`sources` resolve nos dois arquivos: `../../prds/prd-azamon-2026-08-14/prd.md` e `addendum.md` existem a partir do workspace. Os nomes das quatro UJ são verbatim da fonte. Os nomes de componente são idênticos entre as duas espinhas e entre todas as seções de cada uma. As duas referências de token de `EXPERIENCE.md` resolvem para tokens de `DESIGN.md` por nome. As referências de seção (`DESIGN.md.Components`, `DESIGN.md.Brand & Style`) apontam para seções existentes.

**Glossário: nenhum desvio.** Varridos os vinte termos do §3 e os dois sinônimos que o PRD proíbe nominalmente. "Item" aparece 6 vezes e sempre dentro de **Item de Carrinho** ou **Item de Pedido**, exceto "20 itens por página", que é o nome literal do parâmetro no §7.1. "Compra" aparece em "decisão de compra", "caixa de compra" e "fluxo de compra" — usos de substantivo genérico que o próprio PRD faz (a definição de **Pedido** no §3 diz "uma compra concluída ou em andamento"; o NFR-10 e o NFR-11 dizem "fluxo de compra"), nunca substituindo uma instância de Pedido. "Usuário" aparece uma vez (l. 158), num enunciado que abrange as duas áreas e onde nenhum termo do glossário serviria. Nomes de superfície ("Vitrine", "Meus pedidos", "Meus endereços", "Perfil") são verbatim do §9. Os treze parâmetros do §7.1 citados nas espinhas foram conferidos um a um contra a tabela: todos batem, inclusive os dois valores de demonstração.

Também conferido: `EXPERIENCE.md` nomeia tokens do shadcn (`ring`, `foreground`, `muted-foreground`, `secondary`, `destructive`, `border`) sem sintaxe de chave, o que está correto — não são tokens de `DESIGN.md` e a especificação prevê essa forma de herança de sistema de UI.

### Achados

Nenhum que altere o comportamento de um consumidor. Ver as notas mecânicas.

## 8. Encaixe de forma — forte

`DESIGN.md`: as oito seções canônicas estão presentes, na ordem travada — Brand & Style → Colors → Typography → Layout & Spacing → Elevation & Depth → Shapes → Components → Do's and Don'ts. Nenhuma omissão, nenhuma inversão, nenhuma seção estranha no corpo.

`EXPERIENCE.md`: as oito seções obrigatórias estão presentes (Foundation, Information Architecture, Voice and Tone, Component Patterns, State Patterns, Interaction Primitives, Accessibility Floor, Key Flows). As duas condicionais estão presentes porque disparam: `Inspiration & Anti-patterns` (existe produto de referência declarado) e `Responsive & Platform` (o NFR-10 fixa a faixa 360–1440px). A ordem relativa das obrigatórias é preservada.

As três seções inventadas **ganham o lugar**, e são o que separa este par de uma espinha genérica:

- *A máquina de estados na interface* converte FR-28, FR-31 e FR-32 numa tabela estado → ação por papel, e trava a regra que nenhuma seção padrão carregaria: a ação é derivada do estado, e botão indisponível **sai da tela** em vez de ficar cinza. É a peça central do PRD virando peça central da interface.
- *Dinheiro e Estoque na tela* traduz NFR-13, FR-19, FR-21 e a decisão do §4.4 de o Carrinho não congelar preço em regras verificáveis de tela — inclusive a mais fina delas, que a Reserva de Estoque nunca é nomeada para o Comprador.
- *Modo demonstração* é insumo direto de arquitetura: os dois intervalos de consulta, a recusa do WebSocket com justificativa, e o relógio derivado do servidor seguindo a invariante 6 do addendum. Os dois valores assumidos estão marcados com `[ASSUMPTION]` no lugar certo.

### Achados

Nenhum.

## Notas mecânicas

- **Referências cruzadas quebradas:** duas, ambas da tabela de IA para o vazio — "Menu da conta" (l. 39, 41) e "Navegação administrativa" (l. 42, 43) são pontos de entrada de sete superfícies e não existem em nenhuma outra seção das duas espinhas. Detalhadas na seção 3.
- **Contradição interna:** uma. A anatomia da `Barra superior` em `DESIGN.md` (l. 153) exclui elemento de conta; a tabela de IA de `EXPERIENCE.md` (l. 35, 39, 41) exige um. Seção 3.
- **Tensão entre seções:** uma. "Nomear o motivo sempre que o sistema souber" (`Voice and Tone`, l. 61) contra a FR-2, que exige mensagem genérica no Login. Seção 4.
- **Completude do frontmatter:** `DESIGN.md` carrega `status`, `updated` e `sources` além das chaves previstas na tabela de frontmatter do `design-md-spec.md` (que lista apenas `name`, `description`, `colors`, `typography`, `rounded`, `spacing`, `components`) e além do que os dois exemplos de DESIGN.md trazem. É extensão inofensiva e útil — o par fica auto-rastreável —, registrada aqui só para não ser lida como divergência acidental. Os dois arquivos declaram `status: draft`; promover a `final` quando os achados altos fecharem.
- **Mapeamento de nomes:** as chaves kebab-case do frontmatter (`botao-adicionar-carrinho`, `linha-do-tempo-pedido`, `caixa-de-compra`) não têm mapeamento declarado para os nomes em prosa ("Botão Adicionar ao Carrinho", "Linha do tempo do Pedido", "Caixa de compra"). É inferível por leitura e os exemplos de referência têm a mesma folga, mas um resolvedor que case por nome não fecha. Se a ideia é que o código espelhe a espinha, vale uma coluna de chave na lista de `Components`.
- **Idioma dos títulos de seção:** as seções canônicas estão em inglês nos dois arquivos (Foundation, Information Architecture, Colors, Shapes…) e as três inventadas em português, num par cujo `document_output_language` é pt-BR. Está **correto** — os nomes canônicos são identificadores estruturais e a ordem é travada por eles —, mas registrado porque a assimetria parece deslize e não é.
- **Declaração morta:** `RadioGroup` consta da lista de componentes shadcn herdados (`DESIGN.md` l. 149) e não é usado em nenhuma superfície de nenhuma espinha. Provável sobra da seleção de Endereço no checkout, que hoje não tem componente próprio.
