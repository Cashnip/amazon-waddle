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

- source_spec: `_bmad-output/implementation-artifacts/spec-5-5-entrada-no-checkout-e-confirmacao-do-preco-visto.md`
  summary: FECHA a entrada da 5.2 "quem digita `/checkout/endereco` direto chega ao passo com o Carrinho bloqueado ou com preço a confirmar".
  evidence: O passo Endereço abre por `POST /api/v1/checkout/entrada` (`escolha-de-endereco.tsx`), que revalida no Go e devolve `bloqueio` e `preco_mudou` por linha; o Continuar é travado por `podeContinuar(marcado, podeAvancar)`, com `podeAvancar` vindo de `revalidacaoDe` — fixado em `web/scripts/checkout.test.mjs`. Quem digita `/checkout/revisao` cai em `aberturaDaRevisao`, que aplica `desvioDaAberturaDaRevisao` antes de qualquer pintura. A metade de servidor está em `api/checkout_test.go`; a de tela continua sem componente montado em teste — o que há são as funções puras que o efeito chama, e uma reescrita do efeito que deixe de chamá-las passaria.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-5-entrada-no-checkout-e-confirmacao-do-preco-visto.md`
  summary: NÃO fecha a entrada da 5.4 sobre o Produto invisível: a dívida mudou de superfície, da Revisão para o passo Endereço, e continua aberta.
  evidence: A Revisão já não apresenta Carrinho com linha invisível — `desvioDaAberturaDaRevisao` a devolve ao Carrinho. Mas o `Alert` de bloqueio do passo Endereço renderiza `textoDoBloqueio(linha)`, que para linha invisível devolve exatamente `"Produto indisponível."`: sem nome e sem quantidade, que é a mesma queixa da entrada original, uma tela adiante. Um Carrinho de três Produtos desativados mostra a frase três vezes, idênticas, e o Comprador não tem como saber qual linha é qual — só o Carrinho, que tem as ações, distingue. **Condição de fechamento:** ou `textoDoBloqueio` passa a nomear a linha invisível por algo que o Comprador reconheça (a quantidade, e o nome se o sistema ainda o tiver, sem distinguir Produto desativado de Vendedor desativado — FR-12), ou a tela deixa de listar as linhas invisíveis uma a uma e diz quantas são, mandando ao Carrinho para resolver. É decisão de UX, e não cabe numa estória de servidor.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-5-entrada-no-checkout-e-confirmacao-do-preco-visto.md`
  summary: `carrinho.ConfirmarPrecoVisto` recusa a escrita parcial com um erro sem sentinela, que sai em 500 genérico.
  evidence: `internal/carrinho/carrinho.go` compara as linhas afetadas do `:execrows` com o número de vistos e devolve `fmt.Errorf` quando divergem — a transação desfaz, nada é reportado e a entrada seguinte avisa de novo, que é o desfecho seguro. Mas o Comprador vê o erro genérico, e o caso real (um Item removido noutra aba entre a leitura e a confirmação, em READ COMMITTED) mereceria "o Carrinho mudou, abra de novo". Acrescentar sentinela era `Ask First` na 5.5. A 5.6, que já traz `TOTAL_DIVERGENTE` para a mesma família de divergência-sob-transação, é onde isso cabe.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-5-entrada-no-checkout-e-confirmacao-do-preco-visto.md`
  summary: Duas entradas simultâneas do mesmo Comprador sobre os mesmos Itens podem, em tese, dar deadlock (40P01), e a borda HTTP não repete.
  evidence: `ConfirmarPrecoVisto` ordena os vistos por `ItemID` antes de montar os vetores, mas isso é determinismo do comando, não ordem de travas: num `UPDATE ... FROM (unnest ...)` quem ordena o bloqueio é o plano do executor. A ordem garantida do `AD-5` vem de `SELECT ... ORDER BY id FOR UPDATE`, construção que esta consulta não usa. O risco é baixo — as linhas são todas do mesmo dono, e duas entradas concorrentes exigem duas abas ou duplo disparo —, mas o `AD-4` prevê que a borda repita uma vez em `40P01`/`40001` e essa repetição não existe em `api/` para rota nenhuma. Fechar é adotar o `FOR UPDATE` ordenado aqui, ou implementar a repetição da borda, que serve a todas as rotas de uma vez.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: RESOLVIDO — o `INSERT` tardio em `pedido.item_pedido`, adiado na 5.1.
  evidence: A migração `20260921120000_pedido_criacao.sql` cria o gatilho `item_pedido_so_na_criacao`, que recusa com `restrict_violation` o Item de Pedido que já tem linha em `transicao_status`; `pedido.Criar` grava os Itens antes do histórico. Provado em `api/pedido_test.go` (`criacaoPeloCheckout`).

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: RESOLVIDO em parte — a escolha `azamon:checkout:endereco_id`, adiada na 5.2, é apagada quando o Pedido nasce; ao encerrar a Sessão, continua não sendo.
  evidence: A Revisão chama `limparCheckout` no desfecho `pedido` de `desfechoDaConfirmacao` (201, 200 e `CHAVE_REUTILIZADA`), que apaga a escolha e a chave juntas. Encerrar a Sessão não toca o `sessionStorage`.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: RESOLVIDO — o Confirmar Pedido desabilitado e a chave que ninguém enviava, adiados na 5.4.
  evidence: `CRIACAO_DISPONIVEL = true` em `revisao-do-pedido.tsx`; o botão envia `Idempotency-Key`, `endereco_id` e o `total_centavos` da cotação por `requisicaoDeConfirmar`, e chama `limparCheckout` e `avisarCarrinhoAlterado` quando o Pedido existe. A guarda de armazenamento continua travando o botão sozinha.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: RESOLVIDO — a divergência entre o subtotal do Carrinho e o da cotação, que a Revisão não arbitra (5.4).
  evidence: `pedido.Criar` confere o total do corpo antes do `INSERT` e, sob a trava da Reserva, relê preços e Frete e exige as duas parcelas iguais; divergir é 409 `TOTAL_DIVERGENTE`, e a tela relê a Revisão com a mesma chave. Provado em `api/pedido_test.go`, inclusive que a recusa não consome a chave.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: RESOLVIDO — o laranja duas vezes no fluxo, pelo botão do esqueleto na Página de Produto (5.4).
  evidence: `web/app/produtos/[id]/comprar.tsx` foi removido, a `CaixaDeCompra` deixou de receber `children`, e o `POST /api/v1/pedidos` não aceita mais `produto_id`.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: RESOLVIDO — a `Idempotency-Key` que sobrevive a uma alteração do Carrinho entre duas passagens pela Revisão (5.4).
  evidence: A chave é coluna do Pedido e só se prende quando ele nasce: enquanto nenhum Pedido existe, mesma chave com corpo novo é tentativa nova, e não 409. O 409 `CHAVE_REUTILIZADA` só acontece para a chave que já criou um Pedido, e a tela o trata como o Pedido que ele é — mostra e limpa —, nunca como o 409 cru.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: RESOLVIDO — a escrita parcial de `carrinho.ConfirmarPrecoVisto` saindo em 500 genérico (5.5).
  evidence: A divergência embrulha `carrinho.ErrCarrinhoMudou`, registrado em `internal/plataforma/erro/erro.go` como 409 `CARRINHO_MUDOU`.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: RESOLVIDO — a repetição em `40P01`/`40001` que o AD-4 prevê e que não existia em `api/` (5.5).
  evidence: `api/transacao.go` (`emTransacao`) repete uma vez, e só uma; `criarPedido` e `entrarNoCheckout` a usam. `api/transacao_test.go` prova que impasse e serialização repetem uma vez, que duas vezes desiste e que outro código não repete.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: `carrinho.Esvaziar` apaga por id, então o Item do mesmo Produto cuja quantidade mudou noutra aba entre a leitura e o fim é apagado com a quantidade nova, e o Pedido fica com a antiga.
  evidence: `EsvaziarItens` casa `item.id = ANY(@ids)` e o dono. Somar ao mesmo Produto noutra aba é `ON CONFLICT DO UPDATE` sobre a mesma linha, então o id não muda e a linha some; as unidades acrescentadas não entram no Pedido e não voltam ao Carrinho. A janela é de milissegundos e exige duas abas. Fechar é passar a quantidade lida junto do id e casar as duas no `WHERE` — linha a menos já é `CARRINHO_MUDOU` —, o que muda a assinatura que a spec fixou (`itemIDs`).

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: O `TOTAL_DIVERGENTE` do passo 5 — preço mudado entre o `INSERT` do Pedido e a trava dos Produtos — não tem teste determinístico.
  evidence: `api/pedido_test.go` prova o `TOTAL_DIVERGENTE` do passo 2 (preço mudado antes da confirmação). A segunda conferência, sob a trava, só é alcançável com o `UPDATE` do Administrador comitando dentro de uma janela de milissegundos, e não há gancho no meio de `pedido.Criar`. O teste de concorrência da 5.7 é onde uma corrida dessas cabe.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: Depois de um `TOTAL_DIVERGENTE`, a Revisão relida costuma desviar ao passo Endereço, e o aviso da recusa se perde na navegação.
  evidence: O preço que mudou marca `preco_mudou` no Carrinho, e `desvioDaRevisao` manda ao passo Endereço, que é quem reporta o "de X para Y" e grava a ciência (AD-17). O Comprador vê o aviso de preço do passo Endereço, e não a frase de `TOTAL_DIVERGENTE`. É o desfecho seguro; se a UX quiser a frase também, ela precisa atravessar a navegação.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: A resposta do Pedido (`saidaPedido`) não expõe subtotal, Frete nem Endereço congelados.
  evidence: As colunas existem desde a 5.6 e o teste as lê do banco, mas `GET /api/v1/pedidos/{id}` devolve só id, número, Status e total. A tela do Pedido (5.8) e o Detalhe (6.x) são os primeiros a precisar deles.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: A criação do Pedido não confere no servidor a ciência do preço visto (`preco_mudou`); só o desvio da Revisão no navegador a exige.
  evidence: `pedido.Criar` compara o total do corpo com o Carrinho de agora e nunca lê `preco_visto_centavos`. O teste de `TOTAL_DIVERGENTE` em `criacaoPeloCheckout` recota o Frete e cria o Pedido com a mesma chave sem passar por `POST /api/v1/checkout/entrada`. O Comprador viu o total que envia, então o dinheiro fecha; o que falta é o AD-17 valer no servidor e não só no `desvioDaRevisao`. Fechar pede uma recusa nova (sentinela era `Ask First` na 5.6).

- source_spec: `_bmad-output/implementation-artifacts/spec-5-6-criacao-do-pedido-com-reserva-de-estoque-atomica.md`
  summary: O contador por ano (`ProximoNumeroDoAno`) serializa toda criação de Pedido antes da trava dos Produtos, e o teste do NFR-7 da 5.7 não exercitaria o `ORDER BY id FOR UPDATE` sob concorrência real.
  evidence: pré-existente desde a 1.6 — o `INSERT ... ON CONFLICT DO UPDATE` em `pedido.contador_numero` trava uma linha por ano antes de `catalogo.Reservar`, então N criações paralelas correm em fila. A 5.7 precisa decidir: mover a numeração para depois da Reserva, ou provar a trava do AD-5 chamando `catalogo.Reservar` diretamente em transações concorrentes. Os dois caminhos de recuperação do duplo clique (`ON CONFLICT DO NOTHING` e a releitura em `recusar`) também só rodam quando a corrida acontece — a mesma bancada da 5.7 os prende.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-7-consistencia-de-estoque-sob-concorrencia.md`
  summary: PARCIAL — a 5.7 fechou a metade "provar a trava do AD-5", mas o `ProximoNumeroDoAno` continua serializando toda criação de Pedido do ano atrás de uma linha só.
  evidence: A entrada anterior dava duas saídas, e a 5.7 tomou a segunda: `travaDeReservaSerializa` em `api/concorrencia_test.go` chama `catalogo.Reservar` direto em duas transações, sem contador no caminho, e morre quando o `FOR UPDATE OF p` some (verificado por mutação). Mover a numeração para depois da Reserva não foi feito: é mudança de produção em `pedido.Criar` e no `INSERT` que reivindica a chave de idempotência, fora do escopo de uma estória cujo entregável é teste. Fica como teto de vazão, não como furo de correção — com a fila, o pior caso é lentidão sob carga, nunca venda a mais.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-7-consistencia-de-estoque-sob-concorrencia.md`
  summary: A bancada da 5.7 não prende o `TOTAL_DIVERGENTE` do passo 5 nem os dois caminhos de recuperação do duplo clique (`ON CONFLICT DO NOTHING` e a releitura em `recusar`).
  evidence: Duas entradas anteriores apontavam a bancada da 5.7 como o lugar deles. Ela não serve: pela mesma fila do contador, as criações paralelas não se sobrepõem dentro de `pedido.Criar`, então nem o `UPDATE` de preço comitando na janela entre o `INSERT` e a trava, nem a colisão de chave no índice único acontecem por disparar N requisições juntas. Os três continuam sem teste determinístico e pedem gancho dentro de `pedido.Criar` — ou a numeração movida, que reabriria a janela de verdade.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-8-tentativa-de-pagamento-e-a-tela-do-pedido-em-processamento.md`
  summary: RESOLVIDO — a entrada da 5.6 "A resposta do Pedido (`saidaPedido`) não expõe subtotal, Frete nem Endereço congelados" foi fechada pela 5.8.
  evidence: `pedido.Detalhar` substituiu `pedido.Buscar`, e `GET /api/v1/pedidos/{id}` passou a carregar `subtotal_centavos`, `frete_centavos`, `endereco` (`null` no Pedido do esqueleto), `itens[]` a preço praticado, `historico[]`, `expira_em` e `tentativas_restantes`, provados contra o que está gravado em `api/pedido_detalhe_test.go`. A lista de "Meus pedidos" continua com os quatro campos, que é decisão da 6.1.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-8-tentativa-de-pagamento-e-a-tela-do-pedido-em-processamento.md`
  summary: A superfície "recusado" da tela do Pedido não é alcançável no sistema rodando até a 5.10 (aplicar a recusa) e a 5.11 (expirar), e um total de `,90`–`,99` na primeira Tentativa fica em `AGUARDANDO_PAGAMENTO` com o relógio zerado.
  evidence: A 5.8 não emite recusa (aplicá-la é da 5.10) nem expira Tentativa (5.11); só o teste de Go chega a `PAGAMENTO_RECUSADO`, por `Transicionar` direto. O relógio zerado diz apenas que o prazo terminou, sem prometer atualização. O passeio manual da superfície recusada fica para quando a 5.10 ou a 5.11 existir; até lá ela é provada pelas funções puras de `web/lib/pedido.ts` e pela matriz de servidor.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-9-confirmacao-aprovada-por-webhook.md`
  summary: A aprovação sobre Pedido cancelado só é alcançável pelo teste até a 6.3 dar ao Comprador o cancelamento, e o sinal só fica visível na 6.5.
  evidence: `janelaDeCancelamento` inclui `AGUARDANDO_PAGAMENTO`, mas nenhuma rota leva a `CANCELADO` — o subteste cancela por `pedido.Transicionar` direto. Até a 6.3 existir, a corrida clássica da FR-26 não acontece no sistema rodando, e o registro que a varredura grava não tem leitor: o campo `pagamento_aprovado_sobre_cancelado` e as duas superfícies do Administrador são a 6.5.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-10-pagamento-recusado-e-nova-tentativa.md`
  summary: RESOLVIDO — a entrada da 5.1 "A nova Tentativa de um Pedido cujo Produto ou Vendedor foi desativado depois da compra falha como `ESTOQUE_INSUFICIENTE` com `disponivel: 0`" foi decidida pela 5.10.
  evidence: A mensagem é a mesma do esgotado, de propósito: a FR-12 manda Produto desativado e de Vendedor desativado saírem iguais ao indisponível. A tela relê o Pedido depois do 409, `disponivel` do Item vem `false`, o "Tentar pagar de novo" sai, e `impedimentosDaNovaTentativa` (`web/lib/pedido.ts`) escreve "Não há Estoque disponível de {nome} para uma nova Tentativa de Pagamento." com o nome congelado no Item de Pedido — que continua existindo mesmo com o Produto fora do Catálogo.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-10-pagamento-recusado-e-nova-tentativa.md`
  summary: RESOLVIDO EM PARTE — a entrada da 5.8 "A superfície 'recusado' da tela do Pedido não é alcançável no sistema rodando": a faixa `,90`–`,94` chega a `PAGAMENTO_RECUSADO` sozinha; a de `,95`–`,99` continua parada em `AGUARDANDO_PAGAMENTO` até a 5.11.
  evidence: `EmitirConfirmacoesDevidas` passou a emitir o `RECUSADO`, e `aplicarConfirmacao` o aplica com ator `PROVEDOR` e motivo `RECUSADO_PELO_PROVEDOR`, liberando a Reserva. A faixa que nunca confirma continua sem emissão, de propósito: é a expiração que a resolve. O passeio manual da superfície recusada pelo Roteiro B passos 1–4 continua sem ser feito (entrada abaixo).

- source_spec: `_bmad-output/implementation-artifacts/spec-5-10-pagamento-recusado-e-nova-tentativa.md`
  summary: O "Tentar pagar de novo" nunca foi aberto num navegador, e o componente não tem teste automatizado — só as funções puras que ele chama.
  evidence: `podeTentarDeNovo`, `impedimentosDaNovaTentativa` e `desfechoDaNovaTentativa` estão presas em `web/scripts/pedido.test.mjs`, e a matriz de servidor em `api/nova_tentativa_test.go`. O `web/` continua sem bancada de montar componente, então uma reescrita de `tentarDeNovo` em `acompanhamento.tsx` que deixe de chamar a releitura, ou de barrar o segundo clique pelo `useRef`, passaria na suíte. O passeio é o Roteiro B passos 1–4 a 360 e 1440 px, só pelo teclado.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-10-pagamento-recusado-e-nova-tentativa.md`
  summary: `Corrente` é calculado na leitura da inbox, fora da trava do Pedido, então uma nova Tentativa que comite entre essa leitura e o `TravarPedido` deixa uma recusa atrasada da Tentativa 1 passar por corrente e liberar a Reserva da 2.
  evidence: `ConfirmacoesNaoAplicadas` roda sem transação e `aplicarConfirmacao` só reconfere o Status sob a trava (`internal/pedido/pedido.go`, o `if c.Corrente && …`). Hoje é inalcançável no sistema rodando: com o Provedor Simulado, uma confirmação da Tentativa 1 pendente implica o Pedido ainda em `AGUARDANDO_PAGAMENTO`, sem nova Tentativa possível. Fica alcançável com a 5.11 (expirar e tentar de novo antes de uma confirmação tardia chegar) ou com um Provedor real. A correção é reconferir a corrente sob a trava — o número da Tentativa na `Pendente` contra a contagem feita com a linha do Pedido presa, pela porta de `pagamento` —, e mexe na consulta da inbox (Ask First da 5.10).

- source_spec: `_bmad-output/implementation-artifacts/spec-5-10-pagamento-recusado-e-nova-tentativa.md`
  summary: A aprovação de uma Tentativa superada com o Pedido ainda ativo (`AGUARDANDO_PAGAMENTO` ou `PAGO` pela Tentativa nova) é sinalizada sem aviso nem leitor — é cobrança duplicada que a 6.5 não enxerga, porque o predicado dela exige `CANCELADO`.
  evidence: `aplicarConfirmacao` só avisa quando o Pedido travado está `CANCELADO`; a Tentativa superada aprovada cai em `NAO_APLICAVEL_SINALIZADA` calada. Inalcançável com o Simulado (a Tentativa 1 aprovada leva a `PAGO`, e não há nova Tentativa), alcançável com a 5.11 mais um Provedor real. Decidir junto da 6.5 se o sinal ao Administrador cobre também esse caso.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-10-pagamento-recusado-e-nova-tentativa.md`
  summary: `TentativasSemConfirmacao` devolve para sempre toda Tentativa sem linha na inbox — as da faixa `,95`–`,99`, as expiradas e as de Pedido cancelado —, e o tique de 1 s as relê todas.
  evidence: a consulta filtra só por `criada_em <= @ate` e pela ausência de confirmação; a expiração da 5.11 não grava inbox, então nada tira essas Tentativas da lista. Cresce com o histórico. A 5.11 decide a saída (marcar a Tentativa expirada, ou filtrar pela Tentativa corrente do Pedido).

- source_spec: `_bmad-output/implementation-artifacts/spec-5-11-expiracao-da-tentativa-de-pagamento.md`
  summary: RESOLVIDO — as entradas da 5.8 e da 5.10 sobre a superfície "recusado" não alcançável: com a expiração, as duas faixas chegam a `PAGAMENTO_RECUSADO` sozinhas.
  evidence: `pedido.Expirar` entrou no tique entre `aplicar` e `simular`, sem interruptor, e leva a Tentativa vencida a `PAGAMENTO_RECUSADO` com ator `VARREDURA` e motivo `TEMPO_ESGOTADO`, liberando a Reserva pelo efeito da linha do AD-3. A faixa `,95`–`,99` deixa de ficar parada em `AGUARDANDO_PAGAMENTO` com o relógio zerado: `api/expiracao_test.go` prende o vencido, o não vencido, a nova Tentativa recomeçando o prazo e o `PAGO` que nunca é candidato, e `cmd/azamon/main_test.go` prende a FR-34 no binário inteiro. O passeio manual do Roteiro B passo 5 continua sem ser feito (entrada abaixo).

- source_spec: `_bmad-output/implementation-artifacts/spec-5-11-expiracao-da-tentativa-de-pagamento.md`
  summary: RESOLVIDO — a entrada da 5.10 "`TentativasSemConfirmacao` devolve para sempre toda Tentativa sem linha na inbox": a janela de emissão passou a ter os dois lados.
  evidence: A consulta ganhou o piso `criada_em > @desde` e `EmitirConfirmacoesDevidas` recebe o prazo da expiração por parâmetro, como já recebia o atraso — passado o prazo a Tentativa está morta, porque a varredura já encerrou o Pedido, e emitir sobre ela não teria efeito. Sem migração e sem coluna: a emissão continua derivada, e não marcada (AD-7). O subteste "a Tentativa passada do prazo sai da lista de emissão" morre quando o piso sai.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-11-expiracao-da-tentativa-de-pagamento.md`
  summary: ATUALIZADO, e não fechado — a entrada da 5.10 sobre o `Corrente` calculado fora da trava do Pedido: continua adiado, e a leitura da 5.11 estreita ainda mais a janela em vez de abri-la.
  evidence: A 5.11 tornava o caso alcançável em tese (expirar, tentar de novo, e uma confirmação tardia da Tentativa 1 chegar depois). Na prática segue inalcançável com o Provedor Simulado — uma confirmação pendente da Tentativa 1 implica o Pedido ainda aguardando — e o piso novo da janela de emissão estreita mais o intervalo: passado o prazo, a Tentativa 1 nem sequer é relida para emissão. A correção continua sendo reconferir a corrente sob a trava, e mexe na consulta da inbox.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-11-expiracao-da-tentativa-de-pagamento.md`
  summary: A reconferência inteira de `avancarUm` — as DUAS metades, `!travado.Desde.Valid` e `travado.Desde.Time.After(ate.Time)` — não tem teste que morra quando ela sai, nem na expiração (5.11) nem na simulação de entrega (1.8, de onde a linha veio).
  evidence: Verificado por mutação nesta estória: apagar a comparação e rodar `go test ./...` inteiro fica verde. A razão é estrutural e vale para as duas metades — `PedidosParaAvancar` já filtra por `max(ocorrido_em) <= @ate`, que exclui tanto o Pedido dentro do prazo quanto o Pedido sem histórico nenhum (o `max` é NULL e a comparação não é verdadeira), então nenhum dos dois chega à releitura travada. A guarda protege só a corrida entre a seleção e a trava, que nenhum teste de uma goroutine alcança; prendê-la pede gancho dentro de `avancarPeloTempo` ou duas goroutines com a linha travada de propósito. A seção Verification da spec da 5.11 esperava que essa mutação derrubasse o subteste do prazo por vencer, e ela não derruba.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-11-expiracao-da-tentativa-de-pagamento.md`
  summary: O Roteiro B passo 5 nunca foi aberto num navegador: a troca do relógio por "Tempo de pagamento expirado." sem recarregar não tem prova de tela.
  evidence: Nenhuma linha do `web/` mudou na 5.11 — `frasesDoMotivo` já tinha `TEMPO_ESGOTADO` e `podeTentarDeNovo` já não olha o motivo —, e a superfície é provada pelas funções puras de `web/lib/pedido.ts` mais a matriz de servidor. Falta o passeio a 360 e 1440 px, só pelo teclado: Produto que feche em `,95`, o relógio de 60 s chegando a zero, outra aba com o Estoque de volta, e "Tentar pagar de novo" trazendo relógio novo.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-11-expiracao-da-tentativa-de-pagamento.md`
  summary: RESOLVIDO — achado novo, fechado na própria estória: a suíte de `api/` piscava no subteste da emissão da faixa `,95`, por deriva entre o relógio do host e o do contêiner do Postgres.
  evidence: Os chamadores de teste de `EmitirConfirmacoesDevidas` passavam `atraso = 0`, então o corte saía do relógio do host e `criada_em`, do relógio do contêiner: com o contêiner um instante à frente, a Tentativa recém-criada caía fora da janela. Reproduzido no código de antes desta estória (uma falha em quatro execuções do baseline). Trocado por `atrasoDeTeste = -time.Minute` em `api/pedido_detalhe_test.go`, com o porquê escrito na constante; quatro execuções seguidas da suíte inteira ficaram verdes depois.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-11-expiracao-da-tentativa-de-pagamento.md`
  summary: Os cortes de tempo da varredura e da emissão saem do relógio do processo Go e são comparados com colunas que o Postgres escreveu com `now()` — em produção, não só no teste.
  evidence: `avancarPeloTempo` calcula `ate` com `time.Now()` e o compara com `max(ocorrido_em)`; `EmitirConfirmacoesDevidas` faz o mesmo com `ate` e `desde` contra `criada_em`. Com o prazo de 15 min de produção, segundos de deriva são ruído; com os **60 s da demonstração**, são ~10% do prazo, e deslocam para os dois lados o instante em que o Pedido expira contra o que a tela contou. A 5.11 corrigiu **só os chamadores de teste** (`atrasoDeTeste`), e o código de produção continua como estava. A saída seria o corte sair do próprio banco (`now() - $1::interval` na consulta), o que tiraria o relógio do Go do caminho e valeria também para `TentativasSemConfirmacao`.

- source_spec: `_bmad-output/implementation-artifacts/spec-5-11-expiracao-da-tentativa-de-pagamento.md`
  summary: `Expirar` não tem teto de lote por tique: depois de uma parada longa, o primeiro tique expira **todo** Pedido vencido, um por transação e em sequência, segurando `aplicar` e `emitir` do mesmo `time.Ticker`.
  evidence: `avancarPeloTempo` percorre `PedidosParaAvancar` inteiro sem `LIMIT`, e `varrer()` chama os quatro passos em série na mesma goroutine — um tique longo atrasa os outros três, inclusive a aplicação de confirmações que já chegaram. Vale igual para `SimularEntrega`, que tem a mesma casca desde a 1.8, mas a FR-34 é a que acumula candidatos durante a parada. Na demonstração e no teste é invisível (uma dúzia de Pedidos); a saída é um `LIMIT` na consulta de candidatos, com o resto ficando para o tique seguinte — o que já é o comportamento correto, porque nada aqui exige lote completo.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-1-meus-pedidos.md`
  summary: `acompanhamento.tsx` mantém a própria cópia do mapa dos sete Status, então o "ponto único de rótulo" de `web/lib/pedido.ts` ainda não é único.
  evidence: A 6.1 apagou a cópia de `meus-pedidos.tsx` e passou a importar `rotuloDoStatus`, mas o bloco congelado desta estória decidiu (escolha humana, opção (a)) não tocar `web/app/pedidos/[id]/*`, que está em review pela 5.8 — a segunda cópia, em `acompanhamento.tsx:43-51`, continua viva e é usada em `:315`. Renomear um rótulo hoje muda "Meus pedidos" e deixa a tela de acompanhamento com o texto velho; um Status chamado `constructor` renderiza o herdado de `Object.prototype` lá, que é justamente o caso que o `Object.hasOwn` novo bloqueia. Nenhum teste observa `acompanhamento.tsx` — `npm test` roda só `scripts/*.test.mjs`, sobre `lib/`. Settlement: a 6.7, que fecha os sete Status num componente único nas três superfícies, apaga o mapa local e passa a chamar `rotuloDoStatus`.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-3-cancelamento-pelo-comprador.md`
  summary: RESOLVIDO — a entrada da 5.1 "O cancelamento repetido (duplo clique, o segundo chegando depois do commit do primeiro) sai como `FORA_DA_JANELA_DE_CANCELAMENTO`, e não como sucesso".
  evidence: Decidido no servidor, e não só no botão: `pedido.Cancelar` lê o Pedido do dono com `FOR UPDATE` que espera (`TravarPedidoDoComprador`) e trata `CANCELADO` como sucesso sem efeito — 200, nenhuma linha no histórico, nenhuma Reserva tocada. O segundo clique, simultâneo ou depois do commit, espera na trava e lê `CANCELADO`. A tela também barra o segundo envio por `useRef`. `api/cancelamento_test.go` prende o duplo clique preso na trava (dois 200, uma linha `CANCELADO`), e a mutação que tira o `FOR UPDATE` o derruba.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-3-cancelamento-pelo-comprador.md`
  summary: RESOLVIDO EM PARTE — a entrada da 5.9 "A aprovação sobre Pedido cancelado só é alcançável pelo teste até a 6.3 dar ao Comprador o cancelamento": a corrida da FR-26 acontece no sistema rodando; o sinal continua sem leitor até a 6.5.
  evidence: `POST /api/v1/pedidos/{id}/cancelamento` leva o Pedido de `AGUARDANDO_PAGAMENTO` a `CANCELADO` pela rota, e o subteste "a aprovação que chega depois do cancelamento não ressuscita o Pedido" (`api/cancelamento_test.go`) cancela um Pedido de `,00` pela rota, emite e varre: segue `CANCELADO`, Reserva `LIBERADA`, confirmação `NAO_APLICAVEL_SINALIZADA`. O campo `pagamento_aprovado_sobre_cancelado` e as superfícies do Administrador continuam sendo da 6.5.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-3-cancelamento-pelo-comprador.md`
  summary: A UJ-3 nunca foi percorrida num navegador: o "Cancelar Pedido", o `Dialog` (foco preso, Esc e clique fora durante o envio, 360 px) e a frase da corrida perdida só têm prova nas funções puras e na matriz de servidor.
  evidence: `podeCancelar`, `desfechoDoCancelamento`, `fraseSemCancelamento` e `textosDoCancelamento` estão presas em `web/scripts/pedido.test.mjs`, e a rota em `api/cancelamento_test.go`; o `web/` continua sem bancada de montar componente, então uma reescrita de `cancelar` em `acompanhamento.tsx` que deixe de reler, de barrar o segundo clique pelo `useRef` ou de fechar o `Dialog` antes da releitura passaria na suíte. Falta o passeio a 360 e 1440 px só pelo teclado: Pedido em `SEPARANDO`, "Cancelar Pedido", confirmar, "Cancelado" na região viva e na linha do tempo, foco no título, e outra aba com o Estoque de volta na Página de Produto.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-3-cancelamento-pelo-comprador.md`
  summary: `TestExecutarDeixaOCatalogoSemeado` (`cmd/azamon/main_test.go:135`) piscou uma vez com a suíte inteira em paralelo: "o Pedido pago no prazo esta em SEPARANDO; quero PAGO".
  evidence: Visto em uma de três execuções de `go test ./...` durante a 6.3, que não toca nada do que o teste exercita; o pacote sozinho passou três vezes seguidas, e a suíte inteira ficou verde na execução seguinte. O teste depende de tempo real — tique de 1 s, `AZAMON_ENTREGA_INTERVALO` de 200 ms —, e sob a carga de quatro pacotes subindo Postgres ao mesmo tempo a simulação de entrega chega a avançar o Pedido pago antes de a asserção lê-lo. Não investigado a fundo; a saída é alargar o intervalo da entrega no teste, ou ler o Status pelo histórico em vez de pelo Status corrente.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-4-painel-de-pedidos-do-administrador.md`
  summary: As duas telas do painel (`/admin/pedidos` e `/admin/pedidos/{id}`) nunca foram abertas num navegador: o filtro, a ordenação, o `Skeleton`, a tabela rolando no próprio contêiner e os dois `Alert`s só têm prova nas funções puras e na matriz de servidor.
  evidence: `rotuloDaAcao`, `desfechoDaTransicao`, `filtroDaURL`/`consultaDaTabela` e `intervaloDaTabela` estão presos em `web/scripts/pedido.test.mjs`, e as três rotas em `api/pedido_admin_test.go`; o `web/` continua sem bancada de montar componente, então uma reescrita de `acoes-do-pedido.tsx` que deixasse de reler, de barrar o segundo clique pelo `useRef` ou de trocar a variante do `Alert` passaria na suíte inteira. Falta o passeio da seção Verification a 1440 e 360 px, só pelo teclado: filtrar por `PAGO`, inverter a ordenação, recarregar (o estado se mantém na URL) e avançar um Pedido pelos três botões até `ENTREGUE`, vendo o Estoque total baixar no `ENVIADO`.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-4-painel-de-pedidos-do-administrador.md`
  summary: O subteste "a Tabela lista Pedido de todo Comprador" depende de rodar por ÚLTIMO em `TestSessaoEProduto`, porque a listagem administrativa é global e a asserção de ordem confere que os Pedidos da estória são os mais recentes.
  evidence: `painelDePedidosDoAdministrador` está registrado no fim de `api/sessao_test.go` com o porquê escrito ali, e a asserção compara as posições de dois Pedidos criados nele, sem exigir que sejam os primeiros da página. Ainda assim, um subteste novo registrado depois — que crie Pedido — não quebra nada, mas um que os reordene sim; e um `-shuffle` na suíte derrubaria a comparação. A saída, se incomodar, é filtrar a Tabela por um Status que só os Pedidos da estória tenham, ou comparar as posições contra a consulta equivalente no banco em vez de contra a expectativa de recência.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-4-painel-de-pedidos-do-administrador.md`
  summary: RESOLVIDO na própria estória — a consulta de 10 s da Tabela e do Detalhe administrativo podia sobrescrever a releitura forçada do clique com uma resposta mais velha.
  evidence: As duas telas ganharam o contador `ultima` que `acompanhamento.tsx` usa desde a 5.8: cada consulta pega um número, e só a mais recente escreve no estado. Sem ele, a consulta do intervalo que saiu antes do clique voltaria depois da releitura e traria o Status velho de volta por um intervalo inteiro, logo depois de o `Alert` dizer que a linha foi atualizada. Sem teste próprio: o `web/` não monta componente em teste, e a corrida está na mesma lacuna de bancada da entrada acima.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-4-painel-de-pedidos-do-administrador.md`
  summary: O `de` que a tela envia no POST da transição — a base inteira da distinção entre `ESTADO_JA_AVANCADO` e `TRANSICAO_INVALIDA` — não tem teste que morra quando ele muda.
  evidence: `desfechoDaTransicao` recebe a transição tentada por parâmetro, então `web/scripts/pedido.test.mjs` prende o tratamento da resposta, mas não o que o componente põe no corpo; e o subteste da corrida em `api/pedido_admin_test.go` fornece o `de` velho ele mesmo, provando o servidor e não o cliente. Trocar `de: pedido.status` por uma releitura, ou mandar só `para`, faria a corrida com a simulação sair com o `Alert` destrutivo — a causa falsa que a estória existe para evitar — com a suíte inteira verde. Fechar exige a bancada de montar componente que o `web/` não tem, e que é a mesma lacuna das entradas acima.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-5-pagamento-aprovado-sobre-pedido-cancelado.md`
  summary: RESOLVIDO — a entrada da 5.9 "A aprovação sobre Pedido cancelado só é alcançável pelo teste até a 6.3 dar ao Comprador o cancelamento, e o sinal só fica visível na 6.5", e a parte que a 6.3 deixou aberta: o sinal ganhou leitor.
  evidence: `pagamento_aprovado_sobre_cancelado` é derivado na leitura — `pagamento.PedidosComAprovacaoSinalizada` (porta nova, em lote) diz quais Pedidos têm confirmação `APROVADO` em `NAO_APLICAVEL_SINALIZADA`, e `pedido` a junta com `Status == CANCELADO` em Go, sem SQL cruzando schema. As duas respostas administrativas o carregam sempre (`false` quando não se aplica), a do Comprador nunca; o Detalhe administrativo mostra um `Alert` persistente abaixo do Status e a linha da Tabela um `Badge` `outline` "Pagamento aprovado". `api/aprovado_sobre_cancelado_test.go` prende a matriz inteira pela rota — corrida clássica, `PAGO` cancelado sem sinal, recusa sinalizada, segunda aprovação sinalizada com e sem cancelamento — e morre com as duas mutações do predicado.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-5-pagamento-aprovado-sobre-pedido-cancelado.md`
  summary: DECIDIDO — a entrada da 5.10 "A aprovação de uma Tentativa superada com o Pedido ainda ativo é sinalizada sem aviso nem leitor": o sinal da 6.5 NÃO acende nesses casos enquanto o Pedido está ativo ou expirado (`PAGAMENTO_RECUSADO` por `TEMPO_ESGOTADO`); os dois seguem abertos, à espera de um Provedor real. Se o Comprador depois cancelar o Pedido, o sinal acende — de propósito: é dinheiro aprovado sobre Pedido cancelado, e por isso o texto do `Alert` não afirma que a aprovação veio depois do cancelamento.
  evidence: O predicado da 6.5 exige `CANCELADO` (spec, Never e Ask First). Os dois casos são inalcançáveis com o Provedor Simulado: a Tentativa 1 ou recebe a sua única confirmação — a recusa, que a supera — ou nenhuma (a faixa que expira), e da segunda em diante o Simulado aprova dentro do prazo, com `Expirar` adiado enquanto houver confirmação pendente. A aprovação sobre expirado continua coberta só pelo `WarnContext` de `aplicarConfirmacao` (5.11), e a de Tentativa superada com Pedido ativo nem isso. Estender o sinal é mudar o nome e o escopo do campo — `pagamento_aprovado_sobre_cancelado` deixaria de dizer o que diz — e fica para quando houver Provedor real e estorno (fase 2); a porta nova já devolve a aprovação sinalizada por Pedido, então a mudança seria a condição em `aprovadosSobreCancelado` e o texto da tela.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-5-pagamento-aprovado-sobre-pedido-cancelado.md`
  summary: As duas superfícies do sinal nunca foram abertas num navegador, e o `Alert` e o marcador não têm teste de componente — só os textos, presos como constantes.
  evidence: `web/scripts/pedido.test.mjs` prende os três textos e que nenhum promete estorno; a matriz de servidor prende o campo. Um componente que deixe de ler `pagamento_aprovado_sobre_cancelado`, ou que ponha o `Badge` fora da célula do Status, passaria na suíte — é a mesma lacuna de bancada das entradas da 6.4. O passeio é o da Verification da spec: a 1440 px, só pelo teclado, Produto de `,00`, cancelar em `AGUARDANDO_PAGAMENTO`, esperar ~5 s, abrir `/admin/pedidos?status=CANCELADO` e o Detalhe.

- source_spec: `_bmad-output/implementation-artifacts/spec-6-6-simulacao-de-entrega.md`
  summary: RESOLVIDO — a entrada da 6.3 "`TestExecutarDeixaOCatalogoSemeado` (`cmd/azamon/main_test.go:135`) piscou uma vez com a suíte inteira em paralelo" (a asserção do Pedido pago no prazo, que o achou em `SEPARANDO`).
  evidence: A causa era a lida na entrada: a asserção do `noPrazo` em `oTiqueExpiraNaOrdemDoAD6` lia o Status corrente, e a simulação de entrega do mesmo binário, a 200 ms, podia levá-lo adiante de `PAGO` antes da leitura. Das duas saídas anotadas, ficou a segunda. O intervalo da entrega não mudou, porque a `simulacaoLevaOPedidoAEntregue` precisa dele curto. A asserção passou a ler o histórico: a primeira saída de `AGUARDANDO_PAGAMENTO` é para `PAGO`, e nenhuma linha vai a `PAGAMENTO_RECUSADO`. Ela passa esteja o Pedido em `PAGO` ou já adiante, e continua morrendo se a expiração vencer a aprovação. Nenhuma linha de produção mudou.
