# Trabalho adiado

Achados reais que a revisão levantou e que não pertencem à estória que os expôs.
Append-only: não edite nem remova entradas existentes.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: Não existe CI, e o AD-12 exige um passo que faça grep por `fonts.googleapis`, `@import url(http`, `//cdn.` e `next/font/google` e falhe o build.
  evidence: Três invariantes de arquitetura (fronteira do AD-1, domínio sem HTTP, config que falha alto) passaram a existir só como testes que alguém precisa lembrar de rodar. O NFR-15 depende do grep, e nada o executa.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: O `.env` versionado repete 19 das 23 variáveis com o mesmo valor do padrão de produção do `config.go`, virando segunda fonte de verdade.
  evidence: O cabeçalho do próprio arquivo diz "aqui só o que a demonstração encolhe", e só cinco variáveis divergem de fato. É o mesmo defeito que a decisão 4 do addendum §10 usa para justificar não declarar `default_transaction_isolation` no compose.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: `AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS=89` é ambígua entre "aprova até R$ 0,89" e "aprova até o resto 89 dos centavos do total".
  evidence: O §7.1 do PRD define a regra pelos dois últimos dígitos do total, mas nem o `.env`, nem o `config.go`, nem o addendum dizem qual leitura o nome carrega. A estória 1.7 vai implementar a que adivinhar.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: O `web/` não tem lint, teste, `next.config.js` nem `output: "standalone"`, e a imagem copia `node_modules` inteiro.
  evidence: O README chama `go test ./...` de suíte completa, o que é falso para o repositório. A estória 1.2 traz fonte auto-hospedada, tokens e shadcn para dentro dessa casca — é o momento mais barato de pôr o guarda-corpo.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: O goose arrasta `modernc.org/sqlite`, `libc`, `memory` e `bigfft` para o `go.sum`, e configura dialeto, logger e FS por variáveis globais de pacote.
  evidence: É uma árvore grande para aplicar `.sql` em ordem contra um único Postgres, e as globais anulam a testabilidade que a parametrização do `fs.FS` conquistou. O goose expõe um `Provider` com `WithFS`; as build tags de dialeto cortam a árvore.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: `erro.Escrever` loga `err.Error()` de todo erro não registrado, e esse texto pode conter e-mail, Endereço ou nome.
  evidence: O AD-15 proíbe dado pessoal em log. Logar só o tipo do erro preserva a regra mas perde o diagnóstico — a tensão é real e recorre em toda estória que criar sentinela, então merece uma decisão registrada e não uma escolha por handler.

- source_spec: `spec-1-1-andaime-do-servico.md`
  summary: `api/` não tem middleware de recuperação de pânico nem `http.MaxBytesReader`, apesar de o doc do pacote citar "os limites do NFR-14".
  evidence: Um pânico em qualquer handler futuro derruba a conexão, imprime stack trace não estruturado em stderr (furando o AD-15) e escapa do envelope do AD-14. Os limites de corpo do NFR-14 só ganham consumidor quando existir o primeiro DTO.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-2-casca-do-navegador.md`
  summary: Nenhum teste exercita o caminho `:3000 → rewrites() → :8080` de ponta a ponta, então a linha do `Dockerfile` que copia o `next.config.ts` para o estágio de execução pode ser removida sem que nada fique vermelho.
  evidence: A camada de revisão de lacunas reverteu essa linha, construiu a imagem e rodou o contêiner — `/` devolveu 200 e `/api` voltou ao 404 do Next, com `npm test` e a suíte Go verdes. Fechar exige um contêiner de fumaça, e o repositório não tem CI nem arreio de teste de integração onde pendurá-lo.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-2-casca-do-navegador.md`
  summary: O `Badge` do shadcn usa texto de 12px, e a regra de acessibilidade do `DESIGN.md` manda nada abaixo de 14px — as duas regras vêm da mesma espinha de UX, que também proíbe editar o componente.
  evidence: Confirmado em `web/components/ui/badge.tsx` (`text-xs`). O `DESIGN.md` diz ao mesmo tempo "usados do shadcn sem alteração: … Badge" e "todo texto de conteúdo … em no mínimo 14px", e o selo de Status do Pedido é um `Badge` que carrega texto de conteúdo. A tensão é da espinha, não desta estória; quem a resolve é o dono da UX, decidindo se rótulo de selo conta como texto de conteúdo.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-2-casca-do-navegador.md`
  summary: A imagem de execução do `web` carrega 505 MB de `node_modules` porque o estágio de execução copia tudo sem podar.
  evidence: Medido pela camada de revisão de lacunas dentro da imagem construída. Reclassificar as ferramentas para `devDependencies` já foi feito, mas `npm prune --omit=dev` ainda não é seguro: o `next start` lê `next.config.ts` em tempo de execução e precisa do TypeScript instalado. Resolver exige ou converter a configuração para `.js` na imagem, ou adotar `output: "standalone"`.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-3-primeira-migracao-e-catalogo-semeado.md`
  summary: `catalogo.produto.vendedor_id` e `categoria_id` nascem sem índice, e a 1.4 é quem mede p95 com 5.000 Produtos.
  evidence: O Postgres cria índice só no lado referenciado da chave estrangeira, então toda leitura por Categoria ou por Vendedor é varredura sequencial. Com 50 Produtos isso não aparece; a 1.4 é a estória que mede o limiar do NFR-4 e já é dona do índice GIN da busca, então é ela quem deve decidir estes dois no mesmo movimento. Fecha com dois `CREATE INDEX` numa migração.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-3-primeira-migracao-e-catalogo-semeado.md`
  summary: O mesmo e-mail pode existir ao mesmo tempo como Comprador e como Administrador, e nada define a precedência na autenticação.
  evidence: `identidade.comprador` e `identidade.administrador` são tabelas separadas por decisão humana registrada na 1.3, e `UNIQUE` vale por tabela. Hoje não há leitor: a semente usa dois endereços distintos e nenhuma estória autentica. A estória que construir autenticação precisa decidir de qualquer forma — ou a sobreposição é proibida por restrição compartilhada dentro do schema `identidade`, ou é permitida e a ordem de consulta vira regra escrita no addendum.

- source_spec: `spec-1-4-a-medicao-do-nfr-4-com-5-000-produtos.md`
  summary: RESOLVIDO — os índices de `catalogo.produto(vendedor_id)` e `(categoria_id)`, adiados na 1.3, entraram na migração `20260911140000_catalogo_busca_e_indices.sql`.
  evidence: A medição do NFR-4 mostrou que `produto_categoria_id_idx` é o índice que o planejador de fato escolhe para a consulta com termo, Categoria e faixa de preço — o adiamento não era higiene, era o índice que paga. `db/schema_test.go` passou a verificar os três índices pelo `pg_indexes`.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-6-um-pedido-nasce-em-aguardando-pagamento.md`
  summary: A trava da linha do contador de número serializa a criação de Pedidos antes de a Reserva travar o Produto, e com isso a ordem trava-depois-soma do AD-5 fica inalcançável — a prova de concorrência do NFR-7 passaria sem testar nada.
  evidence: `ProximoNumeroDoAno` faz `INSERT … ON CONFLICT (ano) DO UPDATE` e segura a trava da linha do ano até o commit, antes de `catalogo.Reservar` ser chamado. Duas criações do mesmo ano nunca estão dentro de `Reservar` ao mesmo tempo. Nada está errado hoje e ninguém vende duas vezes; o que fica sem valor é a prova. Quem construir o teste de concorrência do NFR-7 precisa primeiro tirar o contador da frente da trava do Produto — travar o Produto antes de numerar, ou numerar fora da transação do caso de uso.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-6-um-pedido-nasce-em-aguardando-pagamento.md`
  summary: `catalogo.Reservar` recebe um Produto só, então o `ORDER BY id FOR UPDATE` ordena uma linha e não protege nada; com Carrinho, a única forma de usar a API é laço por Produto, que é o entrelaçamento que o AD-5 existe para impedir.
  evidence: A assinatura é `Reservar(ctx, tx, produtoID, pedidoID string, quantidade int32)` e monta uma fatia de um elemento para `TravarProdutosParaReserva`. A Épica 4 traz o Carrinho com vários Produtos por Pedido e precisa da assinatura em fatia, com um único comando de trava. Vale conferir junto, contra o plano de execução real, se o `ORDER BY id` de fato determina a ordem de travamento — o Postgres trava durante a varredura, e a ordenação pode ser aplicada depois.

- source_spec: `spec-1-7-provedor-simulado-confirma-por-webhook.md`
  summary: RESOLVIDO — a ambiguidade de `AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS`, levantada na 1.1, foi decidida: a faixa é lida sobre `total_centavos % 100`.
  evidence: `pagamento.Simulado.Decidir` compara o resto dos centavos do total com `AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS` e `AZAMON_PROVEDOR_RECUSADO_ATE_CENTAVOS`, nessa ordem, e `internal/pagamento/pagamento_test.go` prova as fronteiras das três faixas (`,89`/`,90` e `,94`/`,95`). A leitura "aprova até R$ 0,89" está descartada: só a leitura pelos centavos torna a faixa acionável por escolha de Produto, que é o que o §7.1 e a SM-1 exigem. A decisão está registrada no addendum §10, na entrada da estória 1.7. O nome da variável fica como está — renomeá-la quebraria a paridade do `.env` sem acrescentar clareza que o addendum não dê.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-7-provedor-simulado-confirma-por-webhook.md`
  summary: `pgx.ErrNoRows` é o contrato de "não encontrado" atravessando fronteira de módulo, em vez do sentinela que o AD-14 pede.
  evidence: `pedido.Buscar` e `pagamento.RegistrarConfirmacao` devolvem erro do driver como sinal de ausência, e `api/pedido.go` e `api/webhook.go` importam pgx para repetir o mesmo `errors.Is(err, pgx.ErrNoRows)`. É pré-existente — a 1.6 já o fazia em `pedido.Criar` sobre `catalogo.BuscarProduto` —, então não foi causado por esta estória. Um sentinela por módulo, registrado em `internal/plataforma/erro/erro.go`, tira o driver dos handlers.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-7-provedor-simulado-confirma-por-webhook.md`
  summary: A tela de acompanhamento e a navegação pós-compra não têm verificação nenhuma; a parada do polling e o destino do `router.push` podem regredir com toda a suíte verde.
  evidence: `web/package.json` roda `node --test` sobre `scripts/*.mjs`, e o único arquivo lá testa o `next.config.ts` e a guarda offline — nenhuma bancada renderiza componente. Apagar o `clearInterval` da parada, ou trocar `corpo.id` por `corpo.numero` no `router.push`, não deixa nada vermelho. Fechar isto é decidir por uma pilha de teste de DOM, ou extrair as duas decisões para ajudantes puros testáveis sob `node:test` — escolha maior que esta estória.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-8-a-varredura-leva-o-pedido-ate-entregue.md`
  summary: A releitura travada de `Desde` em `avancarEntrega` não tem teste que a alcance.
  evidence: Só dois processos sobre o mesmo Postgres entram no ramo; `internal/pedido` não mantém contêiner de propósito e a demonstração roda um contêiner só. Settlement: um teste que dispare dois `SimularEntrega` concorrentes, ou dois binários, contra o mesmo banco.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-8-a-varredura-leva-o-pedido-ate-entregue.md`
  summary: O critério de parada e os dois ritmos da tela de acompanhamento não têm verificação automatizada.
  evidence: `web/` verifica só com `node --test` sobre `scripts/`, sem jsdom, vitest ou playwright — inverter a guarda do `reprogramar` ou apagar o `clearInterval` do terminal mantém tudo verde. Settlement: extrair a decisão de "próximo ritmo ou parar" para um módulo puro coberto por `node --test`.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-9-clone-limpo-rede-desconectada-readme-de-15-minutos.md`
  summary: O único link de Produto da casca leva ao Fone de R$ 249,90, cujos centavos caem na faixa que o Provedor Simulado recusa — e recusar ainda não existe, então quem segue o caminho óbvio vê o Pedido parado em `AGUARDANDO_PAGAMENTO` para sempre, sem nada na tela que explique.
  evidence: Reproduzido na pilha em execução durante o ensaio da 1.9: 45 s depois da compra o Pedido do Fone segue em `AGUARDANDO_PAGAMENTO`, enquanto um Produto de centavos `,00` chega a `ENTREGUE` em 1 min 41 s. A 1.9 fechou por documentação — o README manda trocar o identificador na URL —, porque acrescentar comportamento era proibido nela. Fecha de verdade quando a 5.11 trouxer a expiração da Tentativa (o Pedido passaria a `PAGAMENTO_RECUSADO` com `TEMPO_ESGOTADO` em vez de ficar parado) ou quando a Épica 3 entregar a busca e remover os dois links temporários da casca. Quem chegar primeiro resolve.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-1-cadastro-de-comprador.md`
  summary: O contrato `dados.campo`, de que depende todo erro em linha do Cadastro, não tem verificação nenhuma do lado do navegador.
  evidence: `npm test` roda só `web/scripts/casca.test.mjs` (rewrite e guarda offline); renomear a chave em qualquer das duas pontas deixa `go test ./...` verde e apaga silenciosamente a mensagem por campo, o foco e o link para o Login. Fecha quando `web/` ganhar uma superfície de teste de componente — a mesma lacuna registrada na 1.8 para a tela de acompanhamento.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-1-cadastro-de-comprador.md`
  summary: `POST /api/v1/compradores` não tem limite por origem e roda 19 MiB de Argon2id por chamada não autenticada.
  evidence: É a única rota pública que paga o custo do hash incondicionalmente; a 2.2 traz bloqueio para tentativa de login, não para criação de conta. Fecha quando o mecanismo de bloqueio da 2.2 existir e puder envolver esta rota também.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-1-cadastro-de-comprador.md`
  summary: Nenhuma guarda contra POST cross-site nas rotas que emitem Sessão — um cadastro forjado fixa a Sessão do atacante na vítima.
  evidence: Um formulário cross-site com `enctype=text/plain` forja um corpo que a decodificação JSON aceita, e o `SameSite=Lax` não impede o navegador de guardar o cookie que volta. `POST /api/v1/sessoes` tem o mesmo furo desde a 1.5, então o padrão é anterior a esta estória. Settlement: conferir `Sec-Fetch-Site` ou exigir o `Content-Type` nas rotas que escrevem, junto com a autorização da 2.4.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-1-cadastro-de-comprador.md`
  summary: O e-mail é único por tabela, então um Comprador pode se cadastrar com o e-mail de um Administrador.
  evidence: `identidade.comprador` e `identidade.administrador` têm `UNIQUE` independentes, por decisão de AD (duas tabelas, sem coluna de papel). Hoje o login só consulta `comprador`, mas a recuperação de senha da 2.3 fica ambígua. Fecha na 2.4, que é dona da separação de papéis.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-1-cadastro-de-comprador.md`
  summary: O cookie de Sessão não tem `Secure`, agora num único `http.SetCookie` compartilhado por login e cadastro.
  evidence: Anterior a esta estória — a 1.5 emitiu o primeiro cookie sem o atributo. A demonstração roda em http e `Secure` a quebraria, então o conserto é um interruptor na Config, não a constante. A extração de `abrirSessao` deixou o lugar pronto para recebê-lo.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-1-cadastro-de-comprador.md`
  summary: A varredura de teclado de 360 a 1440 px na tela `/cadastrar` não foi percorrida em navegador.
  evidence: A extensão do Chrome não estava conectada na sessão que implementou a estória. A metade estrutural do AC está conferida na fonte (rótulo associado, `aria-describedby`, foco no campo recusado) e o contêiner é o mesmo de `/entrar`, que passou na 1.5; falta a metade visual — foco visível a 3:1 e ausência de rolagem horizontal. Settlement: uma passagem manual, ou a mesma superfície de teste que fecharia a primeira entrada desta estória.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-2-autenticacao-encerramento-de-sessao-e-bloqueio.md`
  summary: A navegação para `/entrar?destino=…` no 401 de `comprar.tsx` e `acompanhamento.tsx` não tem teste automatizado.
  evidence: O guarda do destino saiu para `web/lib/destino.ts` e ganhou teste em `node --test`, mas o ramo que dispara a navegação vive dentro de componente React, e `web/` não tem jsdom nem biblioteca de render. Verificado por leitura e pela verificação manual da estória. Settlement: a mesma superfície de teste de componente que fecharia o contrato de `dados.campo` da 2.1.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-2-autenticacao-encerramento-de-sessao-e-bloqueio.md`
  summary: A varredura de teclado de 360 a 1440 px em `/entrar` e no "Sair" da casca não foi percorrida em navegador.
  evidence: A extensão do Chrome não está conectada nesta máquina. A metade estrutural está conferida na fonte (o "Sair" é `<button>` alcançável por `Tab`, `buttonVariants` traz `focus-visible:ring-3`, e `/entrar` não mudou de contêiner); falta a metade visual — foco visível a 3:1 e ausência de rolagem horizontal. Settlement: uma passagem manual.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-2-autenticacao-encerramento-de-sessao-e-bloqueio.md`
  summary: Com a origem sendo o `RemoteAddr`, em compose o par (e-mail, origem) degenera em só-e-mail e qualquer um bloqueia a conta de outro por 15 minutos.
  evidence: Decisão consciente da estória (o `X-Forwarded-For` chega do navegador e quem quisesse escapar do bloqueio só teria de variá-lo). Settlement: uma lista de proxies confiáveis, ou o bloqueio passando a esperar antes de recusar em vez de recusar de vez.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-2-autenticacao-encerramento-de-sessao-e-bloqueio.md`
  summary: A tela `/cadastrar` descarta o `destino`, e o link "Criar conta" de `/entrar` perde a busca — quem cai no 401 e escolhe criar conta não volta ao ponto em que parou.
  evidence: `web/app/cadastrar/page.tsx` empurra `router.push("/")` fixo, e o link para `/cadastrar` em `web/app/entrar/page.tsx` é `href` sem query. A condição de aceite da 2.2 fala do Login preservando o destino, e a porta do cadastro nunca teve destino nenhum — por isso ficou de fora. Settlement: passar o `destino` adiante no link e consumi-lo no sucesso do cadastro, com o mesmo `destinoSeguro` de `web/lib/destino.ts`.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-3-recuperacao-de-senha.md`
  summary: RESOLVIDO — redefinir a senha passou a limpar o contador de bloqueio por tentativas daquele e-mail, em todas as origens.
  evidence: Decidido pela leitura (a) das tres: o contador protegia uma senha que deixou de existir, e limpa-lo nao da poder novo a ninguem — quem chega ao resgate provou a posse do token, e com ele ja podia trocar a senha. `AtualizarSenhaDoComprador` virou `:one` com `RETURNING email` (sem SELECT a mais, e o `pgx.ErrNoRows` passou a distinguir o UPDATE que nao achou linha), e `identidade.EsquecerFalhasDeTodasAsOrigens` varre `falhas:<e-mail>|*` com os metacaracteres de glob escapados. Coberto por `redefinirTiraDoBloqueio` em `api/redefinicao_test.go`, com as duas metades verificadas por mutacao.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-3-recuperacao-de-senha.md`
  summary: A solicitacao de redefinicao distingue conta existente de inexistente pelo tempo de resposta — a existente paga um GET, ate um DEL e dois SET no Redis que a outra nao paga.
  evidence: `SolicitarRedefinicao` devolve cedo em `pgx.ErrNoRows`, antes do primeiro comando Redis. O login fechou esse canal de proposito com o `hashDeDescarte` (`internal/identidade/identidade.go`), e esta rota nao tem equivalente. Nao foi corrigido porque o sinal aqui e ~3 idas ao Redis em soquete local contra o ~50 ms do Argon2id que justificou o `hashDeDescarte` — ordens de grandeza menor, e igualar exige trabalho ficticio no caminho da conta inexistente. Settlement: medir a diferenca sob carga; se for distinguivel, gastar os mesmos comandos no ramo sem conta.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-3-recuperacao-de-senha.md`
  summary: `POST /api/v1/redefinicoes-de-senha` nao tem limite de taxa, e como cada pedido invalida o token anterior, quem souber um e-mail pode derrubar indefinidamente o link que o Comprador legitimo esta segurando.
  evidence: A rota e aberta, e o contador da 2.2 e so do login. E consequencia direta do invariante que a propria FR-3 exige ("nova solicitacao invalida os tokens anteriores"), entao toda implementacao correta numa rota nao autenticada tem essa propriedade. Settlement: contador proprio por (e-mail, origem) nesta rota, ou aceitar e registrar no addendum.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-3-recuperacao-de-senha.md`
  summary: Se a varredura de Sessoes falhar depois do UPDATE, a resposta e 500 e a tela diz "nao foi possivel redefinir" para uma redefinicao que ja gravou a senha e gastou o token.
  evidence: `RedefinirSenha` termina em `EncerrarSessoesDoComprador`, cujo erro sobe ate `erro.Escrever`. A varredura passou a ser resiliente a erro por chave nesta estoria, mas o dilema fica: 204 com Sessoes vivas quebra a condicao de aceite, e 500 mente ao contrario. Settlement: decidir qual das duas mentiras e preferivel, ou dar a rota uma mensagem propria para o sucesso parcial.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-3-recuperacao-de-senha.md`
  summary: O teto de `EmailMax` em `criarRedefinicao` nao e observavel de fora — remove-lo deixa a suite verde, porque um e-mail gigante tambem nao acha conta nenhuma.
  evidence: O caso da matriz afirma status, corpo, cabecalhos e ausencia de chave nova, e todos valem igual para um endereco desconhecido. O dano e limitado pelo `corpoMaximo` de 4 KiB. Settlement: so com um gancho que observe a ida ao Postgres, que o arnes nao tem.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-3-recuperacao-de-senha.md`
  summary: A verificacao de acessibilidade das duas telas novas (foco visivel a 3:1 e ausencia de rolagem horizontal de 360 a 1440 px) nao foi percorrida em navegador.
  evidence: Mesmo desfecho da 2.1 e da 2.2 — a extensao do Chrome nao esta conectada nesta maquina. Na fonte, as telas reusam a casca `max-w-md` de `/entrar` e `/cadastrar`, sem largura fixa. Settlement: percorrer as duas telas so pelo teclado antes de aceitar a Epica 2.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-4-autorizacao-por-dono-e-separacao-de-papeis.md`
  summary: RESOLVIDO — a sobreposição de e-mail entre Comprador e Administrador fica permitida, e a precedência deixa de ser ambígua porque não existe ordem de consulta.
  evidence: São duas funções, uma por tabela: `identidade.Autenticar` só olha `identidade.comprador` e `identidade.AutenticarAdministrador` só olha `identidade.administrador`. Quem escolhe o papel é a rota — `POST /api/v1/sessoes` contra `POST /api/v1/admin/sessoes` —, e a redefinição de senha continua sendo de Comprador. Proibir a sobreposição exigiria restrição compartilhada dentro do schema, que é migração nova. Fecha as duas entradas da 1.3 e da 2.1 sobre o mesmo defeito, e `api/autorizacao_test.go` prova as duas travessias cruzadas em 401.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-4-autorizacao-por-dono-e-separacao-de-papeis.md`
  summary: RESOLVIDO — o POST cross-site que fixava Sessão não decodifica mais: toda rota que lê corpo JSON exige `Content-Type: application/json`.
  evidence: `api.decodificarCorpo` é o único ponto que decodifica corpo, e recusa com `ErrEntradaInvalida` o que não vem como `application/json` — um `<form>` só consegue emitir `text/plain`, `x-www-form-urlencoded` ou `multipart/form-data`, e `application/json` exigiria `fetch`, com o preflight do CORS na frente. Nenhum chamador legítimo mudou: o `web/` e o Provedor Simulado já mandavam o cabeçalho. `Secure` no cookie continua aberto, numa entrada própria.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-5-enderecos-do-comprador.md`
  summary: A casca de cabecalho (header `bg-chrome` + wordmark + `Saudacao`) esta duplicada literalmente em seis paginas do `web/app`.
  evidence: `cadastrar`, `entrar`, `esqueci-a-senha`, `pedidos/[id]`, `produtos/[id]` e agora `enderecos`. O padrao e anterior a esta estoria, que so acrescenta a sexta copia; o import `@/app/produtos/[id]/saudacao`, alcancando a pasta de outra rota, e o mesmo cheiro. Settlement: a 2.6 (Menu da conta) edita as seis de qualquer forma — extrair um componente de casca la custa menos que agora.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-5-enderecos-do-comprador.md`
  summary: Nao ha estado de carregamento entre a montagem da tela e a primeira resposta do servidor.
  evidence: Em `web/app/enderecos/meus-enderecos.tsx`, enquanto `enderecos` e `null` a tela mostra so o `h1` — sem lista, sem vazio, sem esqueleto. O comentario explica corretamente por que "Nenhum Endereco cadastrado." nao pode aparecer ainda, mas nao diz o que aparece. `acompanhamento.tsx` tem o mesmo buraco, entao e padrao do repo: consertar so aqui deixaria as duas telas diferentes. O `Skeleton` do shadcn ja esta em `web/components/ui`.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-5-enderecos-do-comprador.md`
  summary: O CEP e guardado em oito digitos sem faixa conhecida, e a Faixa de Frete da 5.3 vai compara-lo por intervalo de texto.
  evidence: A coluna tem `CHECK (cep ~ '^[0-9]{8}$')` e nada mais: nao existe tabela de faixas nem validacao de CEP existente (NFR-15 proibe consulta a servico pela rede). Um CEP sintaticamente valido mas inexistente entra hoje e cai na regiao padrao da FR-21 quando a 5.3 chegar, que e o comportamento declarado — registrado para que a 5.3 confirme, e nao redescubra, que a validacao e so de forma.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-6-menu-da-conta-e-o-retorno-ao-ponto-de-partida.md`
  summary: A UJ-1 inteira (achar um Produto, comprar, ver o Pedido) não foi percorrida só pelo teclado com o `DropdownMenu` novo no caminho.
  evidence: A extensão do Chrome não está conectada nesta máquina, o mesmo desfecho das estórias 2.1-2.3. Na fonte, o Trigger é o `<button>` que o Radix `DropdownMenuPrimitive.Trigger` renderiza por padrão (Enter/Espaço abre, setas navegam, Esc fecha — mecanismo do próprio shadcn, UX-DR1), e os dois links "Entrar"/"Criar conta" ganharam foco visível escrito à mão (`focus-visible:ring-2 focus-visible:ring-chrome-foreground`) porque o anel padrão do design system, pensado para superfície clara, não bate 3:1 sobre `bg-chrome`. Falta a metade visual: abrir e navegar o Menu só pelo teclado, e conferir o contraste do anel a olho nu de 360 a 1440 px. Settlement: uma passagem manual, ou a mesma superfície de teste de componente que fecharia as lacunas equivalentes das estórias 2.1-2.3.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-6-menu-da-conta-e-o-retorno-ao-ponto-de-partida.md`
  summary: `meus-pedidos.tsx` e `perfil.tsx` não têm estado de carregamento antes da primeira resposta do servidor.
  evidence: Mesmo buraco documentado para `meus-enderecos.tsx`/`acompanhamento.tsx` na 2.5 — `pedidos` fica `null` e a maior parte de `perfil.tsx` fica sem indicação até a primeira resposta, sem texto, spinner ou `Skeleton`. A 2.6 estende o mesmo padrão do repo para a terceira e a quarta tela, sem registrar as novas instâncias.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-6-menu-da-conta-e-o-retorno-ao-ponto-de-partida.md`
  summary: `GET /api/v1/pedidos` (`pedido.Listar`) não tem `LIMIT` nem teto algum — devolve o histórico inteiro do Comprador.
  evidence: Diferente de Endereço, onde `AZAMON_ENDERECO_POR_COMPRADOR_MAX` limita a 20 pela própria regra de negócio, Pedido não tem teto estrutural nenhum. A paginação é da 6.1 por decisão da espinha, mas nada impede hoje uma consulta e um payload sem tamanho declarado enquanto ela não chega.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-6-menu-da-conta-e-o-retorno-ao-ponto-de-partida.md`
  summary: `casca.tsx`, `menu-da-conta.tsx`, `perfil.tsx` e `meus-pedidos.tsx` não têm verificação automatizada nenhuma.
  evidence: Mesma lacuna estrutural das 2.1-2.3 (sem jsdom nem harness de componente em `web/`) — `npm test` roda só `web/scripts/*.test.mjs`, que não toca nenhum dos quatro arquivos novos desta estória. O branch autenticado/sem-Sessão de `MenuDaConta` e o uso de `paraLogin()` nas duas telas novas ficam sem cobertura nenhuma.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-6-menu-da-conta-e-o-retorno-ao-ponto-de-partida.md`
  summary: `identidade.Conta.Email` passa a viajar para dentro de toda Sessão gravada no Redis (`CriarSessao`), não só na resposta HTTP.
  evidence: A Design Notes da 2.6 justifica o campo só pelo custo de uma consulta a mais evitada; a espinha define Sessão como "dado com prazo de validade" mas não decide explicitamente se o e-mail pode viver lá pelos 7 dias de TTL. Antes desta estória, a Sessão carregava só id/nome/papel — é uma categoria nova de dado pessoal em repouso que ninguém decidiu por escrito.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-6-menu-da-conta-e-o-retorno-ao-ponto-de-partida.md`
  summary: Uma Sessão do Redis criada antes desta estória subir, ainda válida dentro dos 7 dias, desserializa com `Conta.Email` vazio.
  evidence: `identidade.Conta` ganhou o campo `Email`, mas Sessões antigas gravadas antes da mudança não têm essa chave — o zero-value ("") entra sem erro nenhum, e o Perfil mostraria o e-mail em branco, sem explicação, até a Sessão expirar ou o Comprador entrar de novo.

- source_spec: `_bmad-output/implementation-artifacts/spec-2-6-menu-da-conta-e-o-retorno-ao-ponto-de-partida.md`
  summary: `MenuDaConta` pisca o estado "sem Sessão" (Entrar + Criar conta) antes de resolver para o `DropdownMenu`, mesmo para quem já está autenticado.
  evidence: A `Saudacao` antiga tinha o mesmo lampejo com um único link "Entrar"; o `MenuDaConta` da 2.6 dobra para dois elementos interativos (Entrar e Criar conta) no mesmo instante, nas oito telas, enquanto `GET /api/v1/sessao` não responde — sem menção na spec nem registro anterior.

- source_spec: `_bmad-output/implementation-artifacts/spec-3-5-a-vitrine-e-o-envelope-de-listagem.md`
  summary: O `web/` não tem `error.tsx`; a Vitrine (e as outras telas Server Component) cai na página de erro genérica do Next quando a API não responde 2xx.
  evidence: `web/app/page.tsx` lança em `!resposta.ok`, e não existe fronteira de erro em `web/app/`; a EXPERIENCE.md não define o texto desse estado.

- source_spec: `_bmad-output/implementation-artifacts/spec-3-10-chrome-da-loja.md`
  summary: A tela `/redefinir-senha/[token]` tem cabeçalho próprio e não usa a `Casca`, então fica sem a busca global, sem a Faixa de Categorias e sem o Menu da conta.
  evidence: `web/app/redefinir-senha/[token]/page.tsx:19` monta o `<header>` à mão desde a 2.3; a UX-DR9 pede a barra em toda tela pública. Não foi mudada na 3.10 porque a spec listou as telas que já usam a `Casca`.

- source_spec: `_bmad-output/implementation-artifacts/spec-4-2-4-3-adicionar-alterar-e-esvaziar-o-carrinho.md`
  summary: A linha "Tela" da matriz (edição otimista com reversão, `Dialog` de esvaziar fechado por `Esc`, estado vazio, volta do Login) não tem teste automatizado, e a tela `/carrinho` não foi aberta num navegador.
  evidence: o `web/` só roda `node --test` sobre `lib/`, sem bancada de componente; `comQuantidade` e `quantidadeDoCampo` estão cobertas em `web/scripts/carrinho.test.mjs`, mas o comportamento de `meu-carrinho.tsx` só foi verificado por `next build` e pelas duas rotas de API que ele chama.

- source_spec: `_bmad-output/implementation-artifacts/spec-4-4-revalidacao-do-carrinho-na-abertura.md`
  summary: Os dois `Alert` da revalidação (bloqueio com Ajustar/Remover dentro, e preço com confirmação) não têm teste automatizado, e nunca foram abertos num navegador.
  evidence: `meu-carrinho.tsx` só passou por `next build`; `revalidacaoDe`, `textoDoBloqueio` e os campos `bloqueio`/`preco_mudou` da API têm teste, mas a renderização condicional, a cor dos botões `outline` dentro do `Alert` e o desaparecer do `Alert` de preço ao confirmar só se veem no passeio a 360 e 1440 px.

- source_spec: `_bmad-output/implementation-artifacts/spec-4-5-a-tela-do-carrinho-nao-promete-o-que-o-sistema-nao-faz.md`
  summary: A frase "Faltam R$ X para o Frete grátis." não tem teste automatizado de renderização, e o Carrinho vazio sem frase vale por construção, não por teste.
  evidence: `faltaParaFreteGratis` está coberta em `web/scripts/carrinho.test.mjs` nas cinco fronteiras da matriz, mas o JSX de `meu-carrinho.tsx` só passou por `next build`; a ausência da frase no Carrinho vazio depende do retorno antecipado da tela, e só o passeio a 360 e 1440 px mostra o alinhamento da frase sob o subtotal.

- source_spec: `_bmad-output/implementation-artifacts/spec-3-5-a-vitrine-e-o-envelope-de-listagem.md`
  summary: A passada de largura do passeio no navegador foi feita a 500 px, e não a 360, porque o Chrome no Windows não reduz a janela abaixo de ~500 px de viewport.
  evidence: `resize_window` para 380x950 devolve `innerWidth` 500; a faixa abaixo do breakpoint `sm:` (640) é a mesma nas duas larguras, então o ramo de layout exercitado é o correto — o que fica sem verificação é truncamento e transbordo específicos de 360 px. Um passeio a 360 exige emulação de dispositivo no DevTools.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-1-a-maquina-de-estados-do-pedido-completa-e-com-testes.md`
  summary: `pedido.transicao_status` ainda não grava a `correlacao` da requisição que o AD-15 pede em cada transição.
  evidence: a correlação vive no `context` por `plataforma.CorrelacaoDe`, e `pedido` não importa `plataforma` (`internal/fronteira_test.go`, AD-1). As saídas — `api/` passar a correlação como parâmetro de `Transicionar`, ou a chave do `context` descer para um pacote que os módulos possam importar — mudam a assinatura do AD-3 ou a tabela do AD-1. A coluna, quando vier, é anulável e não pede preenchimento retroativo; o registro está no `addendum.md` §10, entrada da 5.1.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-1-a-maquina-de-estados-do-pedido-completa-e-com-testes.md`
  summary: Nenhum teste confere que os dois mapas de rótulo de Status do `web/` cobrem exatamente os sete Status do Go.
  evidence: `rotulo` em `web/app/pedidos/meus-pedidos.tsx:23` e em `web/app/pedidos/[id]/acompanhamento.tsx:33` são cópias à mão, e o `?? pedido.status` mostraria o enum cru se um deles ficasse para trás numa renomeação como a de `EM_SEPARACAO` → `SEPARANDO`; o `web/` só testa `lib/` com `node --test`. A 6.7 (forma de exibição dos sete Status) é o lugar natural para extrair os mapas para `lib/` e testá-los.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-1-a-maquina-de-estados-do-pedido-completa-e-com-testes.md`
  summary: O compare-and-swap de `pedido.Transicionar` não tem teste com duas transações concorrendo de verdade pelo mesmo Pedido.
  evidence: `api/maquina_test.go` prova a corrida perdida e a reclassificação em sequência (o segundo chamador chega depois do commit do primeiro); a espera do `UPDATE` pela linha travada por outra transação em andamento — Comprador cancelando contra a simulação consolidando — só é afirmada, não exercitada. É o par do teste do NFR-7 (5.7) para a máquina de estados.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-1-a-maquina-de-estados-do-pedido-completa-e-com-testes.md`
  summary: `pedido.item_pedido` ainda aceita `INSERT` depois da criação do Pedido, então "escrito uma vez" vale para alterar e apagar Item, mas não para acrescentar.
  evidence: os gatilhos da migração `20260919120000_pedido_maquina_de_estados` cobrem `UPDATE`, `DELETE` e `TRUNCATE`; recusar o `INSERT` tardio pede uma regra como "não existe linha de histórico para o Pedido", que depende da ordem Item → histórico dentro de `pedido.Criar` e deve ser decidida junto com a criação multi-Item da 5.6.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-1-a-maquina-de-estados-do-pedido-completa-e-com-testes.md`
  summary: O cancelamento repetido (duplo clique, o segundo chegando depois do commit do primeiro) sai como `FORA_DA_JANELA_DE_CANCELAMENTO`, e não como sucesso.
  evidence: leitura humana da 5.1 (2026-09-19). `corridaPerdida` em `internal/pedido/maquina.go:249` relê `CANCELADO`, que está fora da janela, e a tela diria "Este Pedido não pode mais ser cancelado." de um cancelamento que deu certo. A 6.3 decide: tratar o código como sucesso quando o Status relido é `CANCELADO`, ou travar o botão enquanto a requisição está em voo.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-1-a-maquina-de-estados-do-pedido-completa-e-com-testes.md`
  summary: A nova Tentativa de um Pedido cujo Produto ou Vendedor foi desativado depois da compra falha como `ESTOQUE_INSUFICIENTE` com `disponivel: 0`, e não como indisponível.
  evidence: leitura humana da 5.1 (2026-09-19). `reservarDeNovo` (`internal/pedido/maquina.go:269`) chama `catalogo.Reservar`, que só trava Produtos da VIEW `produto_visivel` (`internal/catalogo/db/consultas.sql:27`). O Pedido fica corretamente em `PAGAMENTO_RECUSADO`; o que a 5.10 decide é a mensagem.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-1-a-maquina-de-estados-do-pedido-completa-e-com-testes.md`
  summary: `erro.Escrever` só acrescenta `dados.permitidas` quando `dados` é `nil` ou `map[string]any`, e muta o mapa de quem chamou.
  evidence: leitura humana da 5.1 (2026-09-19), `internal/plataforma/erro/erro.go:117`. Um handler da 6.4 que passe struct em `dados` perde `permitidas` em silêncio e quebra o contrato "nunca ausente". Nenhum chamador passa `dados` com esse erro hoje.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-1-a-maquina-de-estados-do-pedido-completa-e-com-testes.md`
  summary: A imutabilidade de `pedido` no banco é proteção contra engano, não contra o próprio serviço: o `azamon` conecta como superusuário.
  evidence: leitura humana da 5.1 (2026-09-19). `POSTGRES_USER: azamon` em `docker-compose.yml:34` cria superusuário, e qualquer sessão dele pode `SET session_replication_role = replica`, como faz `envelhecerHistorico`. Basta para o trabalho; não afirmar mais que isso na apresentação, ou criar um papel sem superusuário para o serviço.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-2-selecao-de-endereco-no-checkout.md`
  summary: A consequência "trocar o Endereço recalcula o Frete" (FR-22) ainda não tem teste que a prove: a 5.2 só garante a causa, que o navegador guarda o `endereco_id` e nada mais.
  evidence: `web/lib/checkout.ts` guarda só o `id` em `sessionStorage`, e `web/scripts/checkout.test.mjs` prova que nenhum Frete, CEP ou total atravessa a ida ao Carrinho. O Frete não existe até a 5.3, e a Revisão que o pede ao Go é a 5.4 — é lá que um teste precisa trocar o Endereço e ver o Frete mudar.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-2-selecao-de-endereco-no-checkout.md`
  summary: O `RadioGroupItem` do shadcn estiliza o marcado com `data-checked:`, mas o Radix instalado marca com `data-state="checked"`, então o fundo e a borda de marcado nunca aplicam — só o ponto do `Indicator` aparece.
  evidence: `web/components/ui/radio-group.tsx` usa `data-checked:bg-primary` e `data-checked:border-primary`; `node_modules/@radix-ui/react-radio-group` só emite `data-state`. O checkout da 5.2 contorna no próprio cartão com `has-[[data-state=checked]]:border-foreground`, sem editar o componente. Quem decide é o dono da UX: o `DESIGN.md` manda usar o shadcn sem alteração.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-2-selecao-de-endereco-no-checkout.md`
  summary: Quem digita `/checkout/endereco` direto chega ao passo com o Carrinho bloqueado ou com preço a confirmar — o passo só recusa o Carrinho vazio, e o `podeAvancar` vale só para o botão "Fechar o Pedido".
  evidence: `escolha-de-endereco.tsx` confere só `itens.length === 0`; a confirmação de preço é estado da tela do Carrinho (AD-17) e se perde na navegação. A revalidação no servidor na entrada do checkout é da 5.5, que deve fechar este caminho.
- source_spec: `_bmad-output/implementation-artifacts/spec-5-2-selecao-de-endereco-no-checkout.md`
  summary: A escolha `azamon:checkout:endereco_id` no `sessionStorage` nunca é apagada — nem na criação do Pedido, nem ao encerrar a Sessão.
  evidence: Um segundo checkout na mesma aba volta com a escolha antiga marcada; outro Comprador na mesma aba herda a chave (inofensivo, porque `enderecoMarcado` cai no primeiro da própria lista). Apagar na confirmação é da 5.4/5.6.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-3-calculo-do-frete.md`
  summary: Metade da consequência "trocar o Endereço recalcula o Frete" (FR-20/FR-22) está provada na 5.3; falta a tela, que é da 5.4.
  evidence: `api/frete_test.go` pede `GET /api/v1/frete` para o mesmo Carrinho com um Endereço de SP, um da BA e de volta o de SP, e vê R$ 15,00, R$ 30,00 e R$ 15,00, com o total acompanhando. O esboço da Revisão já pergunta a cada abertura, mas não há teste de tela: a 5.4, ao substituir o esboço, precisa manter a pergunta a cada abertura e nunca guardar a cotação no navegador.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-4-revisao-do-pedido.md`
  summary: O Confirmar Pedido da Revisão está desabilitado: a chave de idempotência é gerada e guardada, mas ninguém a envia, e `limparCheckout` não é chamada por ninguém.
  evidence: `web/app/checkout/revisao/revisao-do-pedido.tsx` declara o botão em `{colors.primary-strong}` e o desabilita pelas razões acumuladas, das quais a constante `CRIACAO_DISPONIVEL = false` é uma; `api/pedido.go:17` ainda recebe `entradaPedido{ProdutoID}`, uma unidade, sem Endereço nem Frete. A 5.6 refaz o `POST /api/v1/pedidos` com chave, digest do corpo, congelamento e `TOTAL_DIVERGENTE`, vira **só** `CRIACAO_DISPONIVEL`, envia o cabeçalho `Idempotency-Key` e chama `limparCheckout` depois de o Pedido nascer. A outra razão — `sessionStorage` que não respondeu — segue travando o botão por conta própria depois disso.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-4-revisao-do-pedido.md`
  summary: O subtotal do `GET /api/v1/carrinho` e o da cotação de `GET /api/v1/frete` são duas leituras separadas e podem discordar; a Revisão não arbitra a divergência.
  evidence: `revisao-do-pedido.tsx` exibe as parcelas das linhas do Carrinho e o bloco Subtotal/Frete/Total da cotação, sem comparar os dois subtotais — a tela não soma nada (NFR-13). Um preço alterado entre as duas requisições deixa a soma das parcelas diferente do Subtotal exibido. Quem recusa é a criação do Pedido, sob a mesma trava da Reserva, com `TOTAL_DIVERGENTE` (5.6).

- source_spec: `_bmad-output/implementation-artifacts/spec-5-4-revisao-do-pedido.md`
  summary: O teste de tela da troca de Endereço (FR-20/FR-22) continua sem bancada: o `web/` só tem `node --test` sobre módulos de `lib/`, e nenhum componente React é montado em teste.
  evidence: `web/package.json` roda `node --test "scripts/*.test.mjs"`, e os oito arquivos de teste importam só `web/lib/*.ts`. `checkout.test.mjs` prova a chave, a rota do Frete e a escolha de Endereço; a consequência "trocar o Endereço recalcula o Frete" está provada no servidor por `api/frete_test.go`, mas a metade de tela — o `useEffect` que pergunta a cotação a cada abertura e não guarda nada — segue conferida só à mão. Montar componente pede Testing Library e um ambiente de DOM, que o repositório não tem e que nenhuma estória da Épica 5 orça.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-4-revisao-do-pedido.md`
  summary: O laranja aparece duas vezes no fluxo do Comprador enquanto o botão do esqueleto continuar na Página de Produto.
  evidence: `web/app/produtos/[id]/comprar.tsx:63` é o "Confirmar o Pedido" da 1.6, em `primary-strong` e `rounded-full`, ainda montado dentro da `CaixaDeCompra` por `web/app/produtos/[id]/page.tsx:98`; a Revisão da 5.4 traz o segundo, que é o do `DESIGN.md`. Os dois chamam o mesmo `POST /api/v1/pedidos` do esqueleto — um de verdade, um desabilitado. Quem fecha é a 5.6, ao refazer a criação do Pedido: o caminho de um Produto e uma unidade some junto com o botão.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-4-revisao-do-pedido.md`
  summary: A `Idempotency-Key` sobrevive a uma alteração do Carrinho entre duas passagens pela Revisão, e a 5.6 precisa decidir se isso é a mesma tentativa de checkout ou outra.
  evidence: `chaveDeIdempotencia` só devolve chave nova depois de `limparCheckout`, que a 5.6 chama na criação do Pedido. O Comprador que entra na Revisão, volta ao Carrinho, muda a quantidade e retorna confirma sob a **mesma** chave descrevendo outro Carrinho. Pela AC da 5.6, mesma chave com corpo diferente é `409` — recusa para um Comprador legítimo. A 5.6 decide entre apagar a chave quando o conteúdo do Carrinho muda, ou tratar o digest divergente como tentativa nova; o que não serve é o 409 cru chegar à tela.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-4-revisao-do-pedido.md`
  summary: Um Carrinho só de Produtos invisíveis não é vazio, chega à Revisão e é apresentado com linhas "Produto indisponível." sem quantidade nem parcela.
  evidence: `revisao-do-pedido.tsx` recusa só `itens.length === 0`, e `LinhaDaRevisao` cala nome, quantidade e parcela da linha invisível. O subtotal e o total não mentem — `carrinho.Itens` soma só as visíveis, no Go —, mas o último ponto de leitura antes do compromisso lista Itens que não somam nada. Quem fecha o caminho é a 5.5, que revalida na entrada do checkout e bloqueia o avanço com `bloqueio` ativo (FR-19); a 5.4 tem a revalidação em `Never`. Se a 5.5 mudar de forma, a linha invisível ainda deve mostrar quantidade, ou a Revisão não deve apresentar um total que não consegue itemizar.
