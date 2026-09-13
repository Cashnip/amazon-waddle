# Epic 2 Context: Conta, Identidade e Endereços

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Esta épica faz o sistema saber quem está do outro lado. Um visitante cria conta, abre e encerra Sessão, redefine senha e gerencia seus Endereços; a partir daqui todo recurso de Comprador é verificado no servidor, e chamada direta à API com o identificador de outro dono é negada. É o primeiro passo depois do esqueleto vertical, e tudo o que vem depois — Catálogo administrativo, Carrinho, checkout e acompanhamento de Pedidos — depende de identidade resolvida e de posse verificável. A Épica 1 já deixou pronto o login do Comprador semeado e o cookie de Sessão atravessando a casca do front; esta épica fecha o restante do ciclo de conta e nasce com o mecanismo de autorização que cada épica posterior vai apenas estender com seus próprios recursos.

## Stories

- Story 2.1: Cadastro de Comprador
- Story 2.2: Autenticação, encerramento de Sessão e bloqueio por tentativas
- Story 2.3: Recuperação de senha
- Story 2.4: Autorização por dono do recurso e separação de papéis
- Story 2.5: Endereços do Comprador
- Story 2.6: Menu da conta e o retorno ao ponto de partida

## Requirements & Constraints

- **Cadastro:** e-mail normalizado (minúsculas, bordas aparadas) **antes** da verificação de unicidade; formato de e-mail inválido e senha com menos de 8 caracteres recusados antes de qualquer escrita no banco; ao final do cadastro o visitante já está autenticado, sem passar pelo Login.
- **Não vazar existência de conta:** credencial inválida devolve a mesma mensagem para e-mail inexistente e para senha errada; a resposta ao pedido de redefinição é idêntica exista ou não a conta; o bloqueio por tentativas conta o par (e-mail, origem) **exista ou não a conta**, senão a própria mensagem de bloqueio confirma o e-mail.
- **Bloqueio:** 5 tentativas falhas, bloqueio de 15 minutos — ambos os limiares vindos da configuração, nunca constantes no código.
- **Sessão:** expira sozinha por inatividade em 7 dias; encerrar invalida no servidor (a chave deixa de existir no cache), não só apaga o cookie; Sessão expirada em qualquer tela leva ao Login preservando o destino.
- **Redefinição de senha:** token de uso único válido por 30 minutos; nova solicitação invalida os anteriores — nunca dois tokens válidos ao mesmo tempo; redefinir encerra **todas** as Sessões ativas do Comprador; a nova senha passa pelas mesmas regras do cadastro.
- **Autorização:** recurso de outro dono devolve o **mesmo erro de inexistente**, nunca "acesso negado" com dica; teste automatizado que chama a API com identificador de Endereço alheio e espera negação nasce aqui (Carrinho e Pedido entram no mesmo teste nas Épicas 4 e 6).
- **Endereços:** CEP validado no cadastro; remover Endereço não altera nenhum Pedido existente, porque o Pedido congela o Endereço na criação; Comprador sem Endereço é levado ao cadastro e volta ao ponto em que parou.
- **Transversais que valem aqui como AC:** limite máximo declarado e verificado **no servidor** para todo campo e parâmetro; todo limiar em variável de ambiente com prefixo `AZAMON_`; correlação em toda linha de log; um módulo só é alcançado pela sua interface pública; acessibilidade WCAG 2.2 AA e faixa responsiva de 360 a 1440 px sem rolagem horizontal; os 20 termos do glossário literais na tela.
- **Ordem de corte:** a recuperação de senha é o primeiro item a cair se o escopo apertar — não aparece em nenhum roteiro de demonstração, e as contas da demo vêm do Catálogo Semeado.

## Technical Decisions

- **Senha:** Argon2id com `m=19456` (19 MiB), `t=2`, `p=1` e sal aleatório de 16 bytes por usuário. Nunca bcrypt de fator baixo, nunca hash de uma passagem.
- **Dado com prazo de validade vive no cache com TTL nativo:** Sessão, token de redefinição e contador de bloqueio. O cookie de Sessão é opaco de 256 bits, `HttpOnly`, `SameSite=Lax`; nenhum conteúdo viaja no cookie — o valor é apenas a chave.
- **Sem serviço de e-mail** (o ambiente roda offline): o token de redefinição é emitido para o log estruturado e lido nos logs do compose.
- **Autorização:** a camada de API resolve a Sessão e injeta a identidade; a **verificação de posse acontece dentro do módulo dono, na mesma consulta que carrega o dado** (`WHERE id = $1 AND comprador_id = $2`), nunca como condicional depois da leitura. O papel de Administrador é verificado na API **por prefixo de rota** (`/api/v1/admin/`), e a área administrativa é separada da loja.
- **Modelo de dados:** `Comprador`, `Administrador` e `Endereço` moram no schema/módulo `identidade`. Comprador e Administrador são **tabelas separadas, sem coluna de papel** — nenhuma linha de Comprador vira Administrador por engano. O Administrador nasce semeado; não existe tela de promoção nem auto-cadastro. Chave estrangeira cruzando schema é proibida; referência cruzada carrega o identificador sem FK, e quem referencia congela o que precisa exibir.
- **Convenções:** identificadores de domínio em português, literalmente os termos do glossário; tabelas e colunas em `snake_case` singular; chaves primárias UUID ordenado no tempo; datas em UTC no banco e RFC 3339 no JSON; rotas `/api/v1/<recurso-plural-em-português>`; envelope de erro com código, mensagem, dados e correlação. A aplicação dos limites de tamanho de campo acontece na decodificação do DTO na camada de API, lendo da struct de configuração carregada uma vez no arranque.
- **A migração de `identidade.endereco` pertence à estória 2.5.**

## UX & Interaction Patterns

- **Erro em linha por campo**, com o motivo ou o limite nomeado, no Cadastro e no Endereço — nunca resumo genérico no topo. O erro é associado ao campo por `aria-describedby` e recebe foco ao submeter.
- **Mensagens fechadas:** e-mail já cadastrado → "Já existe uma conta com este e-mail." com link para o Login; credencial inválida → "E-mail ou senha incorretos."; bloqueio → "Muitas tentativas. Tente novamente em 15 minutos."; sem Endereços → "Nenhum Endereço cadastrado." com ação única de cadastrar.
- **A autenticação é a única exceção declarada à regra de sempre nomear o motivo.** Um desenvolvedor seguindo a regra geral escreveria "E-mail não encontrado" e quebraria o requisito.
- **Estados que faltavam na recuperação de senha e precisam ser criados:** "solicitação enviada" com texto neutro, e "token inválido ou expirado" com caminho para solicitar outro.
- **Menu da conta:** `DropdownMenu` do shadcn na barra superior, presente em toda tela pública e de Comprador. Com Sessão: Meus pedidos, Meus endereços, Perfil, Sair. Sem Sessão: Entrar, Criar conta. É a única porta para as superfícies de conta e para encerrar a Sessão.
- **Perfil é mínimo:** e-mail em leitura, trocar senha pelo fluxo de redefinição, sair. Nenhum requisito pede mais.
- **Retorno ao ponto de partida:** visitante que tenta ação de Comprador é levado ao Login com a origem guardada e volta exatamente para onde estava ao autenticar. Vale também para Sessão expirada.
- **Disciplina visual:** componentes shadcn usados sem alteração (`Input`, `Label`, `DropdownMenu`, `Dialog`, `Alert`); verde é reservado a Estoque disponível e ao Status `ENTREGUE` — jamais para confirmação de formulário; `Dialog` de um nível só; ordem de tabulação igual à de leitura e foco visível a 3:1. A verificação mínima de acessibilidade é percorrer a jornada de compra inteira sem tocar no mouse.

## Cross-Story Dependencies

- **Da Épica 1 vem pronto:** o login do Comprador semeado, o cookie de Sessão atravessando o proxy do front, o armazenamento Argon2id e a base visual da loja. Esta épica acrescenta cadastro, encerramento de Sessão, bloqueio, redefinição e Endereços.
- **A estória 2.4 é a fundação das demais:** o mecanismo de identidade injetada e posse verificada na consulta precisa existir antes de Carrinho, Pedido e área administrativa. O teste de negação por dono nasce cobrindo Endereço e é estendido, não reescrito, nas Épicas 4 e 6.
- **As Épicas 3, 4, 5 e 6 dependem desta** para saber quem está falando; a separação de papéis por prefixo de rota é pré-requisito da área administrativa da Épica 3.
- **A estória 2.5 é pré-requisito do checkout:** a seleção de Endereço e o retorno ao ponto em que o Comprador parou são consumidos pela Épica 5, que congela o Endereço no Pedido.
- **Fica para depois:** a busca global e a Faixa de Categorias da barra superior pertencem à Épica 3; até lá as telas desta épica herdam a casca visual construída na Épica 1.
