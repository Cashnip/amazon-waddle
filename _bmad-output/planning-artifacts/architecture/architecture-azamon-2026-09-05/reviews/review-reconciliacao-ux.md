# Revisão de reconciliação — espinha de arquitetura contra as duas espinhas de UX

**Documento revisado:** `ARCHITECTURE-SPINE.md` (2026-09-05)
**Insumos reconciliados:** `EXPERIENCE.md` e `DESIGN.md` (ux-azamon-2026-09-02, ambos `status: final`)
**Escopo:** o que a UX exige e a arquitetura não sustenta; regra visual com consequência de dados; restrição estrutural do `DESIGN.md`; estados enumerados e não produzíveis; contradição direta. Fora de escopo por decisão: token visual, espaçamento e tipografia — a espinha de arquitetura não é o lugar disso, e não repetir o `DESIGN.md` ali é acerto, não omissão.

## Veredito

A espinha sustenta bem a **camada de invariante** da UX — a Reserva derivada e nunca armazenada, a transição como único ponto de mutação, o efeito de Estoque na mesma transação, o arredondamento num ponto só, a autorização por dono do recurso, a rede zerada em execução. Onde ela não sustenta é a **camada de contrato**: a espinha não declara nenhuma resposta de API, e quase toda exigência específica da `EXPERIENCE.md` é uma exigência sobre o que vem na resposta. A tripla de derivação da ação disponível não tem quem a entregue; o envelope de erro `{codigo, mensagem}` não carrega dado, e oito estados da UX dependem de dado no erro; as três consultas em intervalo e o relógio derivado do servidor estão registrados no `.memlog.md` como constraint herdada e **não sobreviveram à destilação** — a espinha não menciona consulta em intervalo em lugar nenhum. Há duas contradições diretas: o Carrinho que não congela nada contra a revalidação que precisa do preço anterior, e o "nunca serial exposto" contra o "número do Pedido" que quatro superfícies exibem.

Nenhum achado abaixo pede mais arquitetura. Quatro deles pedem *um campo a mais numa resposta*; dois pedem *uma frase declarando quem faz o quê*; um pede uma coluna. O `DESIGN.md` está quase inteiramente sustentado — verde e laranja com sentido fechado estão reproduzidos literalmente na tabela de convenções, e a fonte auto-hospedada é uma invariante inteira (AD-12). O único conflito é um caminho de diretório.

---

## CRÍTICO

### C1 — O envelope de erro não carrega dado, e oito estados da UX dependem de dado no erro

**Onde.** AD-14: "cada módulo exporta erros sentinela (`pedido.ErrTransicaoInvalida`, `catalogo.ErrEstoqueInsuficiente`). Módulo **não** conhece código HTTP. `internal/plataforma/erro` faz o mapeamento, num arquivo só, e emite `{"erro":{"codigo":"…","mensagem":"…"}}`." Convenções, *Envelope de erro*: os mesmos dois campos.

**Por que quebra.** Sentinela em Go é um valor sem carga: `errors.Is` responde *qual* erro, nunca *com que números*. Um arquivo de tradução que recebe só a sentinela não tem como injetar contagem, campo ou estado. A `EXPERIENCE.md` enumera oito estados que precisam exatamente disso:

| Estado (`EXPERIENCE.md`) | O que o envelope precisaria carregar |
|---|---|
| Transição inválida (painel do Administrador) | a transição tentada **e quais seriam permitidas a partir do estado atual** — a FR-32 exige isso literalmente |
| Pedido já avançou por outro ator | o estado atual, para compor "Este Pedido já está em {estado}" |
| Cancelamento recusado porque o estado mudou | o novo Status **e a linha do tempo**, porque a tela "atualiza o selo e a linha do tempo" no lugar |
| Erro de validação administrativa | o **campo**, um erro por campo — "Nunca resumo genérico no topo" |
| E-mail já cadastrado | o campo, para o erro em linha |
| Estoque abaixo das Reservas ativas | "Há **4** unidades comprometidas em Pedidos abertos" |
| Remoção de Categoria com Produtos | "quantos Produtos estão vinculados" |
| Botão Adicionar ao Carrinho | "recusada com **o número disponível** na mensagem" |

Some-se a nona: *Erro do servidor* → "`Toast` destrutivo com **identificador de correlação visível** (NFR-9)". O AD-15 põe `X-Correlation-Id` no `context` e em toda linha de log, e nunca diz que ele volta ao navegador — nem no envelope, nem como cabeçalho de resposta. Sem isso, "deu erro" continua não sendo rastreável na apresentação, que é a única razão de a linha existir.

Compor a mensagem em Go resolve metade — a metade textual. Não resolve o erro por campo (que o *Accessibility Floor* amarra a `aria-describedby`), não resolve a correlação, e não resolve o caso em que o cliente precisa do **estado novo** para redesenhar a tela em vez de só mostrar texto.

**Correção.** O envelope ganha três campos opcionais: `campo`, `correlacao` e `detalhes` (objeto livre, tipado por código de erro). O módulo continua sem conhecer HTTP: exporta um erro **tipado que embrulha a sentinela**, e a sentinela continua sendo a chave de tradução (`errors.Is` para o código HTTP, `errors.As` para os campos). Um arquivo de tradução, como o AD-14 quer — só que com o que preencher.

### C2 — `ErrTransicaoInvalida` funde três comportamentos que a UX manda distinguir

**Onde.** AD-3: "Transição com origem diferente de `esperado` devolve erro nomeado — **é assim que a corrida entre Administrador e simulação de entrega se resolve** (addendum, invariante 5)." Um erro, um caminho.

**Por que quebra.** A `EXPERIENCE.md` gastou duas linhas inteiras da tabela de *State Patterns* dizendo que esse caminho único é errado, e a segunda diz por quê:

- *Pedido já avançou por outro ator* — "`Alert` **informativo, não destrutivo**: 'Este Pedido já está em {estado}. A linha foi atualizada.' **Dizer 'transição inválida' aqui reportaria causa falsa — o Pedido não falhou, ele avançou.**"
- *Transição inválida* — "`Alert` **destrutivo** nomeando a transição tentada e qual seria permitida a partir do estado atual."

São a mesma origem-diferente-de-esperado e dois comportamentos opostos na tela. E existe um terceiro, do lado do Comprador: *Cancelamento recusado porque o estado mudou* — "Atualiza o selo e a linha do tempo e explica: 'Este Pedido foi enviado enquanto você estava nesta tela e não pode mais ser cancelado.' **Nunca cair no `Toast` genérico de erro** — seria um erro cru no clímax da UJ-3." A UX nomeou os dois clímax que essa fusão estraga: o da UJ-3 e o da UJ-4.

Há ainda um quarto ramo que o AD-3 hoje transforma em erro e a FR-31 proíbe: "Cancelar um Pedido já `CANCELADO` não produz efeito **nem erro de servidor**."

**Correção.** A regra do AD-3 distingue, dentro do próprio módulo, com o conhecimento que só ele tem — o grafo de transições:

- estado atual ≠ esperado, **mas alcançável a partir de `esperado` pelo caminho normal** → avançou por outro ator (`ErrPedidoJaAvancou`, informativo, carrega o estado atual);
- estado atual ≠ esperado e **não alcançável** → `ErrTransicaoInvalida` (destrutivo, carrega o estado atual e as transições permitidas dele);
- destino igual ao estado atual em cancelamento → sem efeito e sem erro (FR-31).

A invariante do AD-3 não muda — a transição continua condicionada ao estado esperado. Muda o que ela devolve quando perde.

### C3 — O Carrinho não congela nada, e a revalidação da FR-19 precisa do preço anterior

**Onde.** Semente Estrutural, tabela de travessias sem FK: "`item_carrinho.produto_id` | `carrinho` → `catalogo` | **Nada** — a FR-18 exibe preço atual do Catálogo".

**Por que quebra.** É a decisão certa para *exibir* e insuficiente para *revalidar*. A `EXPERIENCE.md` exige, em *State Patterns*: "Revalidação: preço mudou | Abertura do Carrinho **e** entrada do checkout | `Alert` listando **o preço anterior e o atual** por Item de Carrinho, exigindo confirmação antes de seguir (FR-19)." E em *Dinheiro e Estoque na tela*: "aparece como aviso na entrada do checkout quando preço ou disponibilidade mudaram **desde a última visita**." A FR-19 usa a mesma forma: "Produto cujo preço mudou **desde que foi adicionado**".

Sem nenhuma linha de base persistida, "mudou desde que foi adicionado" é indecidível: o sistema não tem o número anterior para comparar nem para exibir. O `Alert` de duas colunas não pode ser construído.

Isto **não é congelar preço** — congelar seria prometer o valor, e o §4.4 do PRD proíbe. É guardar o último valor visto, que não promete nada. A distinção importa porque a espinha rejeitou a coisa certa pela razão certa e levou junto uma coisa diferente.

É o único achado desta revisão que muda o ERD. Também é o mais caro de descobrir tarde: o §6.3 do PRD lista a FR-19 entre os quatro itens de "**Nunca cortar, por mais invisíveis que pareçam**".

**Correção.** Uma coluna em `carrinho.item_carrinho`: `preco_visto_centavos`, escrita na adição e reescrita quando o Comprador confirma o `Alert`. A tabela de travessias passa a dizer "o preço visto na última confirmação — base da revalidação da FR-19, não promessa de preço", e a distinção fica registrada onde alguém vai lê-la.

### C4 — A tripla da derivação não tem quem a entregue: dois dos três não têm operação de módulo, e nenhum tem contrato de resposta

**Onde.** A `EXPERIENCE.md` é explícita: "A ação disponível é derivada, e a chave da derivação é uma tripla: `(Status do Pedido, tentativas de pagamento restantes, disponibilidade dos Itens de Pedido)`. Só o Status não basta — a FR-27 esgota as tentativas em 3 (§7.1) e recusa a nova tentativa quando o Estoque disponível acabou nesse meio-tempo. **Um `acoesPara(status)` de um argumento só renderiza 'Tentar pagar de novo' para sempre.**"

**Por que quebra.** Elemento por elemento, contra a espinha:

1. **Status do Pedido** — vem de `pedido`. Sustentado.
2. **Tentativas restantes** — nenhuma operação de módulo devolve isso. O AD-8 declara o *porta* com "duas operações e mais nada" (`IniciarTentativa` e a confirmação por webhook); o AD-7 acrescenta uma operação de módulo, `pagamento.ConfirmacoesNaoAplicadas`, que serve à varredura. A superfície pública de `pagamento` declarada na espinha **não conta tentativas**, e `pedido` não pode ler `pagamento.tentativa_pagamento` direto (AD-2 proíbe consulta fora do próprio schema).
3. **Disponibilidade dos Itens de Pedido** — o AD-5 declara `catalogo.Disponivel`, que pelo nome e pelo uso é por Produto. Um Pedido tem N itens, e o Detalhe do Pedido é consultado a cada 10 s: N chamadas cruzando módulo por consulta, sem primitiva de lote declarada.

E acima dos três: **a espinha não declara nenhuma resposta de API**. Não há lista de endpoints (só `POST /api/v1/webhooks/pagamento` e a convenção de rota), não há forma de DTO, não há nada que obrigue o Detalhe do Pedido a vir numa resposta só. A pergunta que a UX levanta — a tela faz uma chamada ou três? — a espinha não responde, e a resposta que o código vai dar por inércia é três, porque três é o que a fronteira de módulo sugere.

**Correção.** Declarar o recurso, não a arquitetura: `GET /api/v1/pedidos/{id}` devolve, numa resposta, `status`, `tentativas_restantes` e `itens[].disponivel` — e é `pedido` quem compõe, chamando `pagamento.TentativasRestantes(ctx, pedidoID)` e `catalogo.DisponivelEmLote(ctx, produtoIDs)`. As duas setas já existem no grafo do AD-1 (`pedido → pagamento`, `pedido → catalogo`); o que falta é as duas operações estarem na interface pública, e a resposta estar escrita.

### C5 — As três consultas em intervalo e o relógio derivado do servidor sumiram entre o memlog e a espinha

**Onde.** O `.memlog.md` registra a constraint com os números certos: "*(constraint) HERDADO da EXPERIENCE.md (Modo demonstracao): consulta em intervalo, sem WebSocket. 3 s na tela Pedido em processamento, 10 s no Detalhe do Pedido e na Tabela de Pedidos do Administrador enquanto houver estado nao terminal. Relogio de expiracao derivado do servidor*". A `ARCHITECTURE-SPINE.md` não contém a palavra consulta nesse sentido, nem os intervalos, nem `expira_em`, nem os endpoints que os serviriam. O addendum §4 delegou explicitamente: "O mecanismo — polling curto, SSE ou WebSocket — é **decisão de arquitetura**".

**Por que quebra.** Três consequências, e a primeira é a que faz a tela mentir:

- **O relógio.** A `EXPERIENCE.md` fecha a seção *Modo demonstração* com: "**O relógio de expiração é derivado do servidor**, não contado no navegador. O addendum (invariante 6) proíbe estado alcançado por temporizador em memória; a interface segue a mesma regra, senão recarregar a página zera a contagem e mente." O AD-6 garante que a *expiração* deriva do histórico — o servidor sabe a hora certa. O que falta é o campo: sem `expira_em` (ou `segundos_restantes`) na resposta do Pedido, o `web/` só tem o instante em que a tela abriu, e o relógio volta a ser contado no navegador. O §14 do PRD nomeia essa tela como o elemento de interface mais arriscado do documento.
- **O custo da tabela do Administrador.** 10 s de intervalo sobre uma lista paginada de 20 linhas, cada linha precisando do Status (`pedido`) e do marcador da FR-26 (`pagamento`), com JOIN cruzando schema proibido pelo AD-2 — ver A5.
- **A Sessão.** A convenção diz "chave no Redis com TTL" e o §7.1 fixa 7 dias *por inatividade*. Se o TTL for absoluto, a consulta de 3 s não impede a Sessão de morrer no clímax; se for deslizante, uma aba aberta renova a Sessão para sempre e "inatividade" deixa de significar o que o §7.1 diz. Uma linha resolve — ver M7.

**Correção.** Uma invariante curta, ao lado do AD-6, com o que a espinha já decidiu no memlog: consulta em intervalo do navegador, sem WebSocket, 3 s e 10 s vindos da configuração do AD-13; a condição de parada é o estado terminal (não a permanência num estado — a UX é explícita: "parar a consulta ao virar `PAGAMENTO_RECUSADO` deixaria um relógio morto na tela"); e `expira_em` em RFC 3339 na resposta, calculado do histórico. O tique de 1 s do AD-6 já é folgado contra os 3 s da tela — vale dizer isso, porque é o que prova que a virada aparece dentro da janela.

---

## ALTO

### A1 — A VIEW da busca não declara o Estoque disponível, e o selo da Vitrine é um dos três pontos

**Onde.** AD-16: "nenhum módulo além de `busca` monta consulta para localizar Produto. `busca` lê `catalogo` por VIEW, nunca por tabela." AD-2: "`busca` não tem tabelas: lê `catalogo` por uma VIEW dedicada, que é a costura do Elasticsearch." A espinha nunca diz o que a VIEW carrega.

**Por que quebra.** A `EXPERIENCE.md` fecha a regra em três pontos: "**O número exibido é o Estoque disponível, nunca o Estoque.** […] Vale nos três pontos em que um número aparece — **Caixa de compra, mensagem de recusa e selo**." O selo binário ("Em estoque" / "Indisponível") vive no Cartão de Produto, e o Cartão de Produto vive na Vitrine e nos Resultados — servidos por `busca`. Se a VIEW expuser `estoque_total`, o bug que a FR-11 existe para impedir aparece na **primeira tela que a banca vê**, e o passo 3 do Roteiro B ("mostrar que o Estoque voltou") passa a medir o número errado. Disponível exige agregar `catalogo.reserva_estoque` dentro da VIEW; não sai de graça.

Pior, e é o lado arquitetural: essa VIEW **é a costura do Elasticsearch**. Um índice de busca guarda cópia, e a UJ-3 termina com "o Produto volta a aparecer disponível na Vitrine **no mesmo instante**". Se disponibilidade entrar no índice, o §11 quebra o clímax da UJ-3 no dia em que for exercido.

**Correção.** Duas frases no AD-16: a VIEW carrega `estoque_disponivel` derivado; e disponibilidade **nunca é indexada** — quando o Elasticsearch entrar, ele devolve identificadores e a disponibilidade continua sendo lida de `catalogo` na renderização. Vale também dizer quem serve a Vitrine (`busca` com termo vazio, que a FR-12 já define como listagem completa paginada, ou `catalogo`) — a espinha não diz, e é a tela mais vista do sistema.

### A2 — "Nunca serial exposto" contra o "número do Pedido" que quatro superfícies exibem

**Onde.** Convenções, *Chaves primárias*: "`uuid` com `DEFAULT uuidv7()` […] **Nunca serial exposto**."

**Por que quebra.** A FR-29 exige que a lista traga "**número do Pedido**, data, total e Status do Pedido". A `EXPERIENCE.md` o exibe em quatro lugares, e um deles é o clímax da UJ-1: "A tela vira Pedido em processamento, com **número do Pedido** e relógio de expiração vindo do servidor"; mais o `Dialog` de cancelamento "nomeando o Pedido"; mais Meus pedidos; mais a Tabela do Administrador — onde a UX **proibiu a busca por número de propósito**: "**Não há busca por número de Pedido** — dito explicitamente para que o ensaio do Roteiro C conte com o filtro e a ordenação para achar a linha, e não descubra a falta ao vivo."

Um UUID de 36 caracteres não é um número. Não se lê em voz alta, não se reconhece numa linha de tabela que já rola horizontalmente, e não se confere num `Dialog` de confirmação. A regra "nunca serial exposto" existe para impedir chave enumerável — e um rótulo de exibição não é chave.

**Correção.** Decidir na espinha, porque se ela não decidir o `web/` decide sozinho (e vai truncar o UUID de um jeito diferente em cada tela): ou um `numero` próprio no Pedido, sequencial e só de exibição, com o `uuid` continuando a ser a chave de rota; ou um prefixo estável do uuidv7 declarado como forma de exibição. Qualquer uma das duas, escrita uma vez.

### A3 — Nenhum limiar do §7.1 tem caminho até o navegador, e a UX exibe cinco deles

**Onde.** AD-13: "os 15 parâmetros do §7.1 são variáveis de ambiente, lidas **uma vez** para uma struct no arranque. **Constante de limiar espalhada pelo código é defeito.**"

**Por que quebra.** A regra para no binário Go. A `EXPERIENCE.md` põe cinco desses limiares na tela, com número visível:

- teto de 10 unidades por Item de Carrinho — Caixa de compra ("quantidade (teto 10)") e a condição do selo numérico ("quando ele é menor que o teto de 10 unidades");
- 100 caracteres no termo — "limitado a 100 caracteres **no cliente e no servidor**";
- R$ 299,00 do Frete grátis — "Faltam R$ 42,00 para o Frete grátis";
- 3 tentativas — "tentativas restantes de 3";
- 15 min / 60 s da expiração — "o tempo restante até a expiração (15 min padrão, 60 s em demonstração)".

Sem canal declarado, esses cinco viram constantes no `web/` — exatamente o defeito que o AD-13 nomeia, no único lugar onde o AD-13 não alcança. E o NFR-16 existe para permitir "encolher prazos para caber na apresentação sem tocar em código": com o número duplicado no front-end, encolher o prazo passa a tocar em código.

**Correção.** A mais barata não inventa endpoint de configuração: os números chegam **dentro do recurso que já é buscado**, já calculados pelo servidor — `quantidade_maxima` na Caixa de compra, `restante_para_frete_gratis_centavos` no Carrinho, `tentativas_restantes` e `expira_em` no Pedido, `total`/`por_pagina` na paginação. O navegador não conhece limiar nenhum; conhece o que o servidor já resolveu. Uma linha no AD-13 fecha o buraco.

### A4 — O motivo da recusa não tem carregador, e a porta de pagamento é a razão

**Onde.** AD-8: "a porta tem duas operações **e mais nada**: `IniciarTentativa(ctx, tx, pedidoID, totalCentavos) (idExterno, error)` e a confirmação que chega por webhook." AD-7 descreve `pagamento.confirmacao_recebida` apenas pela restrição única sobre a chave de idempotência.

**Por que quebra.** A FR-27 tem como consequência testável: "**O motivo da recusa fica visível no detalhe do Pedido.**" A FR-34 fixa um segundo motivo, de outra origem: "o Pedido transita para `PAGAMENTO_RECUSADO` com motivo 'tempo esgotado'". A `EXPERIENCE.md` põe os dois na tela — "Pagamento recusado: saldo insuficiente." em *Voice and Tone*, "Tentativa expirada | Detalhe do Pedido | […] com o motivo 'Tempo de pagamento expirado'" em *State Patterns* — e a coluna *Não faça* da mesma tabela diz "Mensagem genérica quando o motivo existe".

Nenhum campo de motivo é declarado: nem no payload do webhook, nem em `confirmacao_recebida`, nem na resposta do Pedido. E o motivo tem **duas origens diferentes** — uma vem do provedor, outra é produzida pela varredura na expiração — o que significa que o lugar de guardá-lo é a Tentativa de Pagamento, não a confirmação.

Isto também é NFR-3 e SM-5, não só UX: um gateway real manda `decline_code`. Se a porta não tiver onde recebê-lo, trocar pelo Stripe passa a exigir mexer no Pedido — precisamente o que a SM-5 mede ("zero alteração no checkout, no Pedido ou na máquina de estados").

**Correção.** `motivo` na confirmação recebida e na Tentativa de Pagamento, com valor produzido pelo adapter (recusa) ou pela varredura (`TEMPO_ESGOTADO`), e a tradução para a frase do usuário no mesmo arquivo do AD-14. A porta continua com duas operações; a confirmação é que ganha um campo.

### A5 — O marcador da FR-26 atravessa a fronteira de módulo por linha, a cada 10 segundos

**Onde.** AD-7: "Confirmação aprovada sobre Pedido cancelado é registrada e **sinalizada**, nunca aplicada (FR-26)." A espinha não diz sinalizada onde, nem por qual operação.

**Por que quebra.** A `EXPERIENCE.md` é específica: "Pagamento aprovado sobre Pedido cancelado | Painel e Detalhe do Administrador | `Alert` persistente no Detalhe **e marcador na linha da Tabela de Pedidos**." A informação vive em `pagamento.confirmacao_recebida`; a Tabela vive em `pedido`. O AD-2 proíbe o JOIN, a operação declarada em `pagamento` é `ConfirmacoesNaoAplicadas` (que serve à varredura e é justamente o conjunto complementar), e a Tabela é consultada a cada 10 s enquanto listar Pedido não terminal. Sem primitiva de lote, é N chamadas cruzando módulo a cada 10 segundos, durante a apresentação inteira — e a saída fácil que alguém vai tomar sob pressa é o JOIN que o AD-2 existe para impedir.

**Correção.** Nomear a operação de lote: `pagamento.AprovadasSobreCancelados(ctx, pedidoIDs) map[uuid]bool`, chamada uma vez por página da Tabela. É a mesma forma de `DisponivelEmLote` do C4 — o padrão "uma chamada por página, nunca uma por linha" merece uma frase no AD-2, porque a proibição de JOIN cria essa pressão em todo lugar.

### A6 — "Tentativas esgotadas" é um estado que a arquitetura não consegue produzir

**Onde.** AD-8: "O Provedor Simulado decide pelos centavos do total conforme §7.1 do PRD, **apenas na primeira Tentativa de Pagamento; da segunda em diante aprova**." A espinha reproduz o §7.1 e o addendum §4 fielmente — o problema não é infidelidade.

**Por que quebra.** Siga o caminho: tentativa 1 recusa (`,90`–`,94`) ou expira (`,95`–`,99`); tentativa 2 aprova, por regra. O contador nunca chega a 3. A `EXPERIENCE.md` enumera o estado mesmo assim — "Tentativas esgotadas | Detalhe do Pedido | Ação de tentar de novo sai da tela. Resta cancelar." — e a tabela da máquina de estados condiciona a ação a "**enquanto houver tentativa** e Estoque disponível".

O estado é inalcançável por qualquer roteiro do §12. Isso não invalida o campo `tentativas_restantes` do C4 (a tela mostra "2 restantes" e a regra continua sendo real do lado do servidor) — invalida a tela. E o risco não é construir código morto: é alguém descobrir a inalcançabilidade durante o ensaio e "consertar" a regra do §7.1 na véspera, que é o parâmetro que o Roteiro B inteiro depende de ser estável.

**Correção.** Registrar em *Adiado*, ou numa nota do AD-8: o limite de 3 tentativas é regra de servidor verificável por teste, e é **inalcançável por demonstração** enquanto o §7.1 fizer a segunda tentativa aprovar sempre. Se o time quiser demonstrá-lo, o caminho é uma faixa de centavos que recuse em toda tentativa — mas isso é mudança de PRD, não de arquitetura, e a espinha só precisa dizer que viu.

### A7 — O pós-login que cria o Item de Carrinho não tem casa no grafo do AD-1

**Onde.** AD-1 declara o grafo **exaustivo**: "As setas abaixo são exaustivas: uma seta que não está aqui é defeito, e uma seta invertida é ciclo." Existe `carrinho → identidade`. Não existe `identidade → carrinho`. A tabela de camadas dá a `api/` "Rota, autenticação, DTO, erro→HTTP. **Nenhuma regra.**"

**Por que quebra.** A `EXPERIENCE.md` descreve o fluxo em dois lugares e o marca como o primeiro clique de qualquer avaliador: "Visitante tenta adicionar ao Carrinho | […] Leva ao Login **guardando Produto de origem e quantidade**; ao autenticar volta à Página de Produto **com o Item de Carrinho já criado** (FR-7, FR-16)"; e no componente: "**É a única passagem de Público para Comprador, e o primeiro clique de qualquer avaliador sem conta.**"

"Ao autenticar, o Item já existe" tem duas leituras arquiteturais e a espinha permite as duas: o navegador faz duas chamadas (login, depois adicionar, com a intenção viajando na URL) — o que a UX aceita, desde que a tela de retorno já mostre o Item; ou alguém orquestra as duas em `api/`, e a linha "Nenhuma regra" da tabela de camadas cede. A leitura que **não** é permitida — `identidade` chamando `carrinho` no fim do login — é a mais natural de escrever, e o grafo a chama de ciclo sem nunca mencionar este fluxo.

**Correção.** Uma linha: a intenção viaja na URL do Login (`?produto=&quantidade=`), coerente com a convenção de "estado de listagem vive na URL", e o `web/` dispara a adição após o login. Nenhuma seta nova, nenhuma regra em `api/`, e o AD-1 continua exaustivo de verdade.

---

## MÉDIO

### M1 — Não há convenção de resposta de sucesso, e três superfícies precisam do total de resultados

A espinha declara o envelope de **erro** e nada sobre listagem. A `EXPERIENCE.md` exige: "Paginação | […] Informa página atual **e total de resultados**"; o *Accessibility Floor* amarra isso a um anúncio — "Resultado de busca anuncia a contagem ao atualizar: '{N} Produtos encontrados.'"; e o estado vazio é explícito — "Página além do total é **estado vazio tratado, não erro** (FR-15)", ou seja 200 com lista vazia, nunca 404. Falta também onde vive o teto de 60 itens por página aceito do cliente (§7.1, NFR-14). Uma linha na tabela de convenções resolve os quatro: forma da resposta de listagem com `itens`, `pagina`, `por_pagina`, `total`.

### M2 — Onde mora o mapa `SEPARANDO` → "Separando", e a convenção de glossário precisa de duas exceções

A `EXPERIENCE.md` fecha a forma de exibição: "O mapeamento é fechado, um-para-um e **igual nas três superfícies** — não é tradução livre, é a única forma de exibição permitida", e o componente reforça: "Mesmo componente, mesmo texto e mesma cor nas três superfícies […] **nunca o identificador cru**."

A espinha não diz se o JSON carrega o identificador cru (e o `web/` tem um mapa só, num componente só) ou a forma de exibição (e aí o front-end passa a ramificar sobre texto traduzido — pior). A resposta certa é a primeira, e ela precisa estar escrita, porque três superfícies escritas por três pessoas produzem três mapas que divergem.

Junto: a convenção "Sinônimo é defeito, **em código como em tela**" precisa de duas exceções declaradas, ou colide com a `EXPERIENCE.md` lida ao pé da letra — (a) o Status do Pedido tem forma de exibição um-para-um, que a UX já argumenta não violar a disciplina; (b) "**A Reserva de Estoque nunca é nomeada para o Comprador.** É conceito de domínio, não de interface." Nenhum campo de resposta voltado ao Comprador pode se chamar `reserva*`, e nenhuma mensagem pode conter o termo — inclusive a da nova tentativa recusada por indisponibilidade, que é onde a palavra quase escapa. O termo continua literal em código, schema e telas do Administrador.

### M3 — Faltam duas operações no Catálogo que a UX pede pelo número

O AD-5 declara `Disponivel`, `Reservar`, `Liberar`, `Consolidar`. A `EXPERIENCE.md` pede um número que nenhuma delas devolve: "Estoque abaixo das Reservas ativas | Produtos, ajuste de Estoque | Recusa informando **quantas unidades estão comprometidas** (FR-11). Não é limite de campo, é contagem em tempo real: 'Há 4 unidades comprometidas em Pedidos abertos.'" Σ reservas ativas não é o disponível — é a outra parcela da subtração. Mais a forma de lote do `Disponivel` (C4, A5). Duas assinaturas a acrescentar no AD-5.

### M4 — O Frete precisa existir antes do Pedido, e o Carrinho precisa do limiar sem poder importar `pedido`

O AD-3 diz que o conteúdo do Pedido — incluindo o Frete — "é escrito uma vez, na criação". Mas a Revisão do checkout mostra o Frete **antes** de confirmar (FR-22, UJ-1 passo 5: "na Revisão o Frete aparece calculado a partir do CEP, e subtotal, Frete e total fecham ao centavo"). Existe portanto um caminho de leitura "calcular Frete para este CEP e este subtotal" que a espinha não declara e cujo dono não aparece no *Mapa Funcionalidade → Arquitetura*.

E o Carrinho exibe "Faltam R$ 42,00 para o Frete grátis" — mas `carrinho → pedido` é seta invertida no AD-1. Resolve-se porque o limiar é configuração (AD-13) e os dois módulos leem a mesma struct; mas isso precisa estar dito, senão alguém cria a seta para chegar à Regra de Frete.

### M5 — O estado vazio da Vitrine pode ser inalcançável

A `EXPERIENCE.md` declara: "Vitrine sem Catálogo Semeado | Vitrine | 'O Catálogo ainda não tem Produtos.' […] É consequência testável da FR-6 e **a primeira tela que a banca vê no cenário exato que o NFR-1 e a SM-3 testam**." A espinha separa `db/migracoes/` (goose, aplicadas no arranque) de `db/semente/` e nunca diz quando a semente entra — nem como **não** entrar. Com "modo demonstração é o padrão" (AD-13), o estado vazio pode não ter caminho. Uma linha: a semente é um passo separado do arranque, governado por um parâmetro do AD-13.

### M6 — O papel de Administrador é verificado só por prefixo de rota, e a casca precisa saber antes de desenhar

AD-11: "O papel de Administrador é verificado em `api/`, por prefixo de rota." Correto como autorização. Mas a `EXPERIENCE.md` exige que a área administrativa "não apareça para quem não é Administrador — **nem como link desabilitado, nem como item cinza no menu**", e que a Vitrine vazia ofereça o caminho de cadastro "com Sessão de Administrador" — duas decisões de renderização em superfícies **públicas**, que nunca passam pelo prefixo `/api/v1/admin/`. Não há recurso de Sessão declarado que devolva papel e identidade ao `web/`. A autorização continua no servidor; o que falta é o que a casca desenha.

### M7 — TTL da Sessão contra a consulta em intervalo

A convenção diz "chave no Redis com TTL"; o §7.1 fixa "Expiração da Sessão por inatividade: 7 dias". Não está declarado se o TTL é deslizante. Com consulta de 3 s e 10 s, uma aba aberta renova a Sessão indefinidamente e "inatividade" deixa de significar o que o §7.1 diz; com TTL absoluto, a consulta não impede a Sessão de morrer no meio do clímax da UJ-1, caindo em "Sessão expirada | Qualquer | Redireciona ao Login preservando o destino" na pior hora possível. Uma linha de decisão.

---

## BAIXO

### B1 — `web/public/fontes/` contra `/fonts`

AD-12: "fonte auto-hospedada em `web/public/fontes/`". `DESIGN.md`, *Typography*: "um único arquivo WOFF2 auto-hospedado em `/fonts`". O `@font-face` escrito conforme o `DESIGN.md` pede um caminho que a árvore da arquitetura não serve — 404 silencioso e queda para a pilha de sistema, que é o modo de falha mais difícil de notar (a tela continua funcionando, só deixa de ser a marca). Uma das duas grafias cede; a convenção de português da própria espinha aponta `fontes`.

### B2 — O `grep` de CI do AD-12 não pega as duas formas que o Next.js oferece por padrão

O AD-12 declara três padrões (Google Fonts, `@import url(http`, `//cdn.`). Não pegam `next/font/google` nem `<link rel="preconnect">` — as duas construções que um desenvolvedor de Next.js escreve sem pensar, e que o NFR-15 proíbe. Custa acrescentar dois padrões à mesma linha de CI.

### B3 — A justificativa da fonte auto-hospedada no `DESIGN.md` envelheceu, e a espinha não registra que a envelheceu

O `DESIGN.md` justifica a fonte por arquivo assim: "Amarrar o token tipográfico ao Geist fecharia de lado a **Questão em Aberto 1 do §15** (Next.js vs React) […] Com o arquivo auto-hospedado, a espinha funciona nos dois caminhos". A arquitetura fechou a Questão 1 em Next.js 16 e não diz que a fechou. A decisão de auto-hospedar continua certa — pelo NFR-15, que não depende do framework — mas o próximo leitor vai encontrar um `DESIGN.md` argumentando por uma questão que já tem resposta. Uma linha no *Mapa* ou em *Adiado* fecha o laço.

### B4 — Imagem de Produto ausente: o 404 do Go vira o ícone quebrado que a UX proíbe

A `EXPERIENCE.md`: "**Nunca o ícone de imagem quebrada do navegador** — um caminho errado no Catálogo Semeado não pode abrir um buraco na primeira tela da apresentação." O AD-12 põe as imagens em `media/` servidas pelo Go com URL relativa; um caminho errado na semente vira 404 e o navegador desenha o ícone. O tratamento no `web/` é a metade fácil; a metade barata é que a semente é determinística e versionada (NFR-1), então verificar no arranque que todo caminho semeado existe em `media/` custa dez linhas e falha cedo, não na sala.

### B5 — A linha do tempo é projeção da tabela, não a tabela

AD-15: "`pedido.transicao_status` guarda estado anterior, novo, **ator**, instante **e a correlação** — é a mesma tabela que a linha do tempo da FR-30 renderiza." A `EXPERIENCE.md` pede três campos: "uma linha por transição registrada, com estado anterior, novo e carimbo de tempo". Ator e correlação não devem sair na resposta do Comprador. "Uma fonte, três usos" continua verdade; o que falta é dizer que o uso da tela é uma projeção.

### B6 — Cancelar Pedido já `CANCELADO` devolve erro que a FR-31 proíbe

Terceiro ramo do C2, registrado à parte porque é o único que vem do PRD e não da UX: "Cancelar um Pedido já `CANCELADO` **não produz efeito nem erro de servidor**" (FR-31), contra o AD-3, que devolve erro nomeado sempre que a origem difere de `esperado`.

---

## O que a espinha já sustenta — e que não deve ser mexido ao corrigir o resto

Registrado porque metade das correções acima toca ADs que estão certos no essencial, e a tentação de reescrever o AD junto é real.

- **Verde e laranja com sentido fechado** — a tabela de convenções reproduz o `DESIGN.md` literalmente: "Verde só para Estoque disponível e `ENTREGUE`; laranja uma vez por fluxo, no passo irreversível." É a única regra visual com consequência de comportamento que o `DESIGN.md` tem, e é a única que a espinha carrega. Escolha certa.
- **AD-12 e o NFR-15** — a fonte auto-hospedada é uma invariante inteira, com verificação de CI. O `DESIGN.md` pedia isso em três parágrafos; a espinha entrega em uma regra.
- **AD-11 = "Recurso de outro Comprador"** — "Recurso de outro Comprador devolve o mesmo erro de inexistente" e "a interface trata como não encontrado" são a mesma frase escrita duas vezes. Não há tela de acesso negado a construir.
- **AD-9** — arredondamento num ponto só e formatação apenas no navegador é exatamente o que "o total exibido é sempre a soma exata das parcelas exibidas" precisa, e o que faz `tabular-nums` ler como verdade em vez de enfeite.
- **AD-10 e a renderização no cliente** — casa com o `Skeleton` de carregamento inicial, com "sem WebSocket" e com o estado de listagem na URL. Um caminho de dado, e a UX não pede outro.
- **`uuidv7` ordenado no tempo** — entrega de graça a ordenação padrão "mais recentes" e o desempate estável que a FR-14 e a `EXPERIENCE.md` exigem ("a paginação nunca repete nem pula um Produto"). Vale dizer isso em voz alta no AD-16, porque é um acerto que ninguém vai enxergar sozinho.
- **Cache de vitrine recusado** — é o que faz o clímax da UJ-3 ("o Produto volta a aparecer disponível na Vitrine **no mesmo instante**") ser verdade. A linha de *Adiado* justifica pela "segunda cópia da verdade sem medição"; a razão mais forte é a UJ-3, e citá-la protege a decisão de ser revertida por alguém medindo só o NFR-4.
- **Idempotência de `POST /pedidos`** — casa com "Um por fluxo. Desabilita durante o envio e não permite duplo disparo […] e a interface não pode depender disso para se comportar". Servidor e tela protegem o mesmo passo por dois caminhos, que é o desenho certo para o único ponto irreversível do fluxo.

---

## Ordem sugerida de correção

1. **C3** — é o único que muda o ERD; quanto mais tarde, mais migração.
2. **C1 + C2** — mesmo arquivo (`internal/plataforma/erro`), mesma decisão; separados custam duas rodadas.
3. **C4 + C5 + A3** — todos são "o que vem na resposta"; resolvidos juntos viram uma seção de contrato de recurso, que é o que falta na espinha.
4. **A1, A4, A5, M3** — quatro assinaturas de interface pública e uma coluna.
5. **A2, A6, A7, M2, M4, M5, M6, M7** — uma linha cada, sem código.
6. **B1..B6** — limpeza.
