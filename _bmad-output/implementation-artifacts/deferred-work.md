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

