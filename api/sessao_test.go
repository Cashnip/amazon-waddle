package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// Nenhuma linha da matriz de I/O desta estória é observável sem Postgres e
// Redis de verdade: o hash da semente tem de autenticar, e a chave da Sessão
// tem de existir no Redis com TTL. Um par de contêineres serve aos dois
// arquivos de teste de api/, no padrão de db/schema_test.go.
//
// As migrações e a semente entram pelo disco (`os.DirFS`), e não pelo pacote
// `db`: importá-lo daria a `api` uma aresta que o AD-1 não tem na tabela.
const (
	imagemPostgres = "postgres:18.6"
	imagemRedis    = "redis:8.10.1"

	emailSemente = "comprador@azamon.test"
	senhaSemente = "azamon-comprador"
	nomeSemente  = "Joana Ribeiro"

	// O Administrador semeado desde a 1.3 (FR-4). Fica ao lado do Comprador
	// porque é a MESMA semente: são duas tabelas, e nenhuma coluna de papel.
	emailAdmin = "admin@azamon.test"
	senhaAdmin = "azamon-admin"
	nomeAdmin  = "Marcos Aleixo"

	// Identificador v5 derivado do nome em media/gerar.go — estável por
	// construção, e por isso escrevível no teste e no front.
	produtoSemeado = "3400cd00-3f5e-5433-9171-fde099a52005"
)

// expiracaoDeTeste são os mesmos 7 dias do AZAMON_SESSAO_EXPIRACAO da
// demonstração: o que o teste confere é que o valor configurado chega inteiro
// aos dois lugares que prometem prazo — o Max-Age do cookie e o TTL no Redis.
const expiracaoDeTeste = 7 * 24 * time.Hour

// segredoDeTeste é o AZAMON_WEBHOOK_SEGREDO do ambiente de teste: é ele que
// autentica o webhook do Provedor, e sem ele a rota responde 401.
const segredoDeTeste = "segredo-de-teste"

// Os dois limiares do bloqueio por tentativas, nos mesmos valores dos padrões
// de internal/plataforma/config.go. Precisam estar na Config do teste: com o
// zero-value, AuthTentativasMax=0 bloquearia a primeira tentativa da suíte
// inteira — nenhum login passaria.
const (
	tentativasDeTeste = 5
	bloqueioDeTeste   = 15 * time.Minute
)

// validadeDeTeste são os 30 minutos do AZAMON_SENHA_TOKEN_VALIDADE. Precisa
// estar na Config do teste pelo mesmo motivo dos dois acima: com o zero-value,
// o token nasceria com TTL zero e nenhuma redefinição resolveria.
const validadeDeTeste = 30 * time.Minute

// origemDeTeste é o RemoteAddr que o httptest põe em toda requisição. Fica
// escrito porque o contador de tentativas chaveia pelo par (e-mail, origem), e
// há um subteste que precisa chegar de outra origem.
const origemDeTeste = "192.0.2.1:1234"

// Os limites de campo do cadastro, nos mesmos valores dos padrões de
// internal/plataforma/config.go. Ficam escritos aqui porque o que os subtestes
// conferem é que o limiar configurado chega inteiro à mensagem de erro — um
// Config zerado deixaria toda senha acima do teto.
const (
	nomeMaxDeTeste  = 120
	emailMaxDeTeste = 254
	senhaMinDeTeste = 8
	senhaMaxDeTeste = 128
)

func ambiente(t *testing.T) (http.Handler, *redis.Client, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx, imagemPostgres,
		tcpostgres.WithDatabase("azamon"),
		tcpostgres.WithUsername("azamon"),
		tcpostgres.WithPassword("azamon"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(2*time.Minute)),
	)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(pg) })
	if err != nil {
		t.Fatalf("subir %s: %v", imagemPostgres, err)
	}
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("DSN do contêiner: %v", err)
	}

	raiz := os.DirFS("..")
	if err := plataforma.Migrar(ctx, dsn, raiz, "db/migracoes"); err != nil {
		t.Fatalf("migrar: %v", err)
	}
	if _, err := plataforma.Semear(ctx, dsn, raiz, "db/semente", "teste-api"); err != nil {
		t.Fatalf("semear: %v", err)
	}

	rd, err := testcontainers.Run(ctx, imagemRedis,
		testcontainers.WithExposedPorts("6379/tcp"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("6379/tcp").WithStartupTimeout(time.Minute)),
	)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(rd) })
	if err != nil {
		t.Fatalf("subir %s: %v", imagemRedis, err)
	}
	endereco, err := rd.PortEndpoint(ctx, "6379/tcp", "")
	if err != nil {
		t.Fatalf("porta do Redis: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	rdb, err := plataforma.AbrirRedis("redis://" + endereco + "/0")
	if err != nil {
		t.Fatalf("cliente Redis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	// O pool sai junto porque as linhas da matriz do Pedido são sobre o que
	// ficou no banco — histórico da transição, Item congelado e Reserva ATIVA
	// não aparecem em resposta nenhuma.
	cfg := plataforma.Config{
		SessaoExpiracao:     expiracaoDeTeste,
		AuthTentativasMax:   tentativasDeTeste,
		AuthBloqueioDuracao: bloqueioDeTeste,
		SenhaTokenValidade:  validadeDeTeste,

		WebhookSegredo:   segredoDeTeste,
		CompradorNomeMax: nomeMaxDeTeste,
		EmailMax:         emailMaxDeTeste,
		SenhaMin:         senhaMinDeTeste,
		SenhaMax:         senhaMaxDeTeste,

		EnderecoTextoMax:        enderecoTextoMaxDeTeste,
		EnderecoPorCompradorMax: enderecoPorCompradorMaxDeTeste,

		VendedorNomeMax:  vendedorNomeMaxDeTeste,
		CategoriaNomeMax: categoriaNomeMaxDeTeste,

		ProdutoNomeMax:          produtoNomeMaxDeTeste,
		ProdutoDescricaoMax:     produtoDescricaoMaxDeTeste,
		ProdutoPrecoMaxCentavos: produtoPrecoMaxDeTeste,
		ProdutoEstoqueMax:       produtoEstoqueMaxDeTeste,
		PaginaTamanho:           paginaTamanhoDeTeste,
		PaginaTamanhoMax:        paginaTamanhoMaxDeTeste,
		BuscaTermoMax:           buscaTermoMaxDeTeste,

		CarrinhoUnidadesMax:  carrinhoUnidadesMaxDeTeste,
		FreteIsencaoCentavos: freteIsencaoDeTeste,

		PagamentoTentativasMax:      tentativasMaxDeTeste,
		PagamentoTentativaExpiracao: expiracaoDaTentativaDeTeste,
	}
	return Rotas(cfg, pool, rdb), rdb, pool
}

func TestSessaoEProduto(t *testing.T) {
	rotas, rdb, pool := ambiente(t)

	var cookieValido *http.Cookie

	t.Run("login válido emite o cookie de Sessão e devolve o nome", func(t *testing.T) {
		resp := postar(t, rotas, `{"email":"`+emailSemente+`","senha":"`+senhaSemente+`"}`)
		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d (%s), quero 200", resp.Code, resp.Body.String())
		}
		if nome := decodificar(t, resp)["nome"]; nome != nomeSemente {
			t.Errorf("nome = %v, quero %q", nome, nomeSemente)
		}

		cookies := resp.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("quero exatamente um cookie, vieram %d", len(cookies))
		}
		c := cookies[0]
		cookieValido = c
		// Os mesmos atributos que a sonda da 1.2 provou atravessarem o
		// rewrites() — agora carregados pela Sessão de verdade.
		if c.Name != "azamon_sessao" {
			t.Errorf("cookie = %q", c.Name)
		}
		if len(c.Value) != 64 {
			t.Errorf("valor com %d caracteres; quero os 256 bits em hexadecimal", len(c.Value))
		}
		if strings.Contains(c.Value, emailSemente) || strings.Contains(c.Value, nomeSemente) {
			t.Error("o cookie carrega conteúdo; o valor é opaco (AD-9)")
		}
		if !c.HttpOnly {
			t.Error("cookie sem HttpOnly")
		}
		if c.SameSite != http.SameSiteLaxMode {
			t.Errorf("SameSite = %v, quero Lax", c.SameSite)
		}
		if c.Path != "/" {
			t.Errorf("Path = %q, quero /", c.Path)
		}
		if c.MaxAge != int(expiracaoDeTeste.Seconds()) {
			t.Errorf("MaxAge = %d, quero o AZAMON_SESSAO_EXPIRACAO", c.MaxAge)
		}

		// A condição de aceite: o valor do cookie é chave existente no Redis,
		// com o TTL da configuração. A chave é procurada pelo valor do cookie
		// — e não contada — para o teste não repetir o prefixo que mora em
		// identidade nem depender de quantas Sessões já foram abertas.
		ctx := context.Background()
		chaves, err := rdb.Keys(ctx, "*"+c.Value).Result()
		if err != nil || len(chaves) != 1 {
			t.Fatalf("chaves para o cookie = %v, %v; quero exatamente uma", chaves, err)
		}
		// Janela estreita: um TTL curto demais expiraria a Sessão enquanto o
		// cookie continua prometendo sete dias, e um `ttl > 0` não veria isso.
		ttl, err := rdb.TTL(ctx, chaves[0]).Result()
		if err != nil || ttl > expiracaoDeTeste || ttl < expiracaoDeTeste-time.Minute {
			t.Errorf("TTL = %v, %v; quero perto de %v", ttl, err, expiracaoDeTeste)
		}
	})

	t.Run("e-mail em maiúsculas autentica", func(t *testing.T) {
		resp := postar(t, rotas, `{"email":"COMPRADOR@Azamon.test","senha":"`+senhaSemente+`"}`)
		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d (%s), quero 200", resp.Code, resp.Body.String())
		}
	})

	// Senha errada e conta inexistente saem idênticas: duas respostas
	// distintas viram enumeração de contas.
	t.Run("senha errada e conta inexistente são o mesmo 401", func(t *testing.T) {
		var corpos []string
		// A correlação muda a cada requisição, e é a única coisa que pode
		// diferir: o resto do envelope tem de ser idêntico nos dois caminhos.
		for _, entrada := range []string{
			`{"email":"` + emailSemente + `","senha":"nao-e-a-senha"}`,
			`{"email":"ninguem@azamon.test","senha":"` + senhaSemente + `"}`,
		} {
			resp := postar(t, rotas, entrada)
			if resp.Code != http.StatusUnauthorized {
				t.Fatalf("%s: status = %d, quero 401", entrada, resp.Code)
			}
			if len(resp.Result().Cookies()) != 0 {
				t.Errorf("%s: 401 veio com cookie", entrada)
			}
			envelope := decodificar(t, resp)
			if envelope["codigo"] != "CREDENCIAL_INVALIDA" {
				t.Errorf("%s: codigo = %v", entrada, envelope["codigo"])
			}
			delete(envelope, "correlacao")
			corpos = append(corpos, fmt.Sprint(envelope))
		}
		if corpos[0] != corpos[1] {
			t.Errorf("as duas respostas diferem:\n%s\n%s", corpos[0], corpos[1])
		}
	})

	t.Run("corpo malformado ou grande demais sai em 400", func(t *testing.T) {
		for _, entrada := range []string{
			`{isso não é json`,
			`{"email":"` + strings.Repeat("a", corpoMaximo) + `","senha":"x"}`,
		} {
			resp := postar(t, rotas, entrada)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, quero 400 (%.60s)", resp.Code, entrada)
			}
			if codigo := decodificar(t, resp)["codigo"]; codigo != "ENTRADA_INVALIDA" {
				t.Errorf("codigo = %v", codigo)
			}
		}
	})

	t.Run("a Sessão do cookie é lida de volta", func(t *testing.T) {
		if cookieValido == nil {
			t.Skip("sem cookie: o login falhou antes")
		}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sessao", nil)
		req.AddCookie(cookieValido)
		resp := httptest.NewRecorder()
		rotas.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d (%s), quero 200", resp.Code, resp.Body.String())
		}
		if nome := decodificar(t, resp)["nome"]; nome != nomeSemente {
			t.Errorf("nome = %v, quero %q", nome, nomeSemente)
		}
		// A 2.6: é o mesmo envelope que o Menu da conta e o Perfil consomem —
		// sem ele Perfil não tem o que exibir.
		if email := decodificar(t, resp)["email"]; email != emailSemente {
			t.Errorf("email = %v, quero %q", email, emailSemente)
		}
	})

	// Cookie ausente e cookie fora do Redis são a mesma coisa para quem
	// chama: expirada e inexistente não se distinguem.
	t.Run("sem cookie ou com Sessão desconhecida sai em 401", func(t *testing.T) {
		for _, cookie := range []*http.Cookie{
			nil,
			{Name: "azamon_sessao", Value: strings.Repeat("0", 64)},
			{Name: "azamon_sessao", Value: "curto-demais"},
		} {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/sessao", nil)
			if cookie != nil {
				req.AddCookie(cookie)
			}
			resp := httptest.NewRecorder()
			rotas.ServeHTTP(resp, req)

			if resp.Code != http.StatusUnauthorized {
				t.Fatalf("cookie %v: status = %d, quero 401", cookie, resp.Code)
			}
			if codigo := decodificar(t, resp)["codigo"]; codigo != "SESSAO_INVALIDA" {
				t.Errorf("cookie %v: codigo = %v", cookie, codigo)
			}
		}
	})

	// A 2.1. A ordem importa entre os dois primeiros: a duplicidade por
	// normalização precisa da conta que o caminho feliz acabou de criar.
	t.Run("o cadastro grava o Comprador e já abre a Sessão", func(t *testing.T) {
		cadastroValidoAbreSessao(t, rotas, pool)
	})
	t.Run("e-mail já cadastrado sai em 409 nomeando o campo", func(t *testing.T) {
		cadastroDuplicadoDa409(t, rotas, pool)
	})
	t.Run("campo fora dos limites da Config sai em 400 nomeando o campo", func(t *testing.T) {
		cadastroComCampoInvalido(t, rotas, pool)
	})

	t.Run("detalhe do Produto semeado", func(t *testing.T) { produtoSemeadoSai(t, rotas, pool) })
	t.Run("Produto inexistente e identificador malformado", func(t *testing.T) { produtoAusenteDa404(t, rotas) })

	// A partir daqui a ordem importa: a numeração é por ano e conta do
	// primeiro Pedido, então o subteste que a confere nasce antes dos outros.
	var pedidoCriado string
	t.Run("dois Pedidos nascem em AGUARDANDO_PAGAMENTO, numerados por ano", func(t *testing.T) {
		pedidoCriado = pedidoNasceAguardandoPagamento(t, rotas, pool, cookieValido)
	})
	t.Run("sem Sessão não nasce Pedido", func(t *testing.T) { pedidoSemSessaoDa401(t, rotas, pool) })
	t.Run("sem chave, Endereço alheio, corpo inválido e corpo grande demais", func(t *testing.T) {
		pedidoComEntradaRuim(t, rotas, pool, cookieValido)
	})
	t.Run("Estoque esgotado recusa o Pedido inteiro", func(t *testing.T) {
		pedidoSemEstoqueDa409(t, rotas, pool, cookieValido)
	})
	t.Run("Transicionar sobre estado já avançado", func(t *testing.T) {
		transicaoRepetidaNaoAvanca(t, pool, pedidoCriado)
	})

	// A partir daqui é a 1.7, e a ordem continua importando: o caminho feliz
	// deixa um Pedido em PAGO, e os dois seguintes partem dele.
	t.Run("a confirmação aprovada leva o Pedido a PAGO", func(t *testing.T) {
		confirmacaoAprovadaLevaOPedidoAPago(t, rotas, pool, cookieValido)
	})
	t.Run("confirmação sobre Pedido que já avançou é sinalizada", func(t *testing.T) {
		confirmacaoForaDeAguardandoPagamento(t, rotas, pool)
	})
	t.Run("confirmação de Tentativa superada não aplica", func(t *testing.T) {
		confirmacaoDeTentativaSuperada(t, rotas, pool)
	})
	// A 5.9. Os dois criam Pedido próprio pelo checkout e não olham para os
	// dos outros subtestes, então a posição é indiferente entre os vizinhos —
	// o que não é indiferente é vir depois de quem deixa confirmação na fila:
	// `Varrer` é global, e os dois conferem o que a varredura fez ao SEU
	// Pedido depois de ela ter passado por toda a inbox.
	t.Run("aprovação sobre Pedido cancelado é registrada e não ressuscita", func(t *testing.T) {
		confirmacaoAprovadaSobrePedidoCancelado(t, rotas, pool, cookieValido)
	})
	t.Run("confirmação de Pedido travado fica pendente e é aplicada depois", func(t *testing.T) {
		confirmacaoDePedidoTravadoFicaPendente(t, rotas, pool, cookieValido)
	})
	t.Run("webhook com corpo ruim ou chave desconhecida", func(t *testing.T) {
		webhookRecusaEntradaRuim(t, rotas, pool)
	})
	t.Run("webhook sem o segredo não grava nem avança", func(t *testing.T) {
		webhookExigeOSegredo(t, rotas, pool, cookieValido)
	})
	// Por último: este subteste cria um Pedido de outro Comprador, e o número
	// dele antecede os do ano corrente na ordenação que os outros usam.
	t.Run("a tela do Pedido lê pelo dono", func(t *testing.T) {
		leituraDoPedido(t, rotas, pool, cookieValido, pedidoCriado)
	})
	// A 2.6, com conta própria: o vazio da matriz só é observável antes de
	// qualquer Pedido existir, e é por isso que não reaproveita cookieValido
	// — ele já tem Pedidos dos subtestes acima. Entra antes da simulação de
	// entrega, que é a única que conta com o `estoque_total` intacto até ali.
	t.Run("Meus pedidos: lista só os do dono, mais recente primeiro", func(t *testing.T) {
		meusPedidosListaPorDono(t, rotas)
	})
	// Depois de todos: é o único subteste que baixa o `estoque_total` de
	// produtoSemeado, e os que compram esse mesmo Produto contam com o total
	// intacto para o disponível bater.
	t.Run("a simulação leva o Pedido de PAGO a ENTREGUE", func(t *testing.T) {
		simulacaoDeEntrega(t, rotas, pool, cookieValido)
	})

	// A 2.2 fica no fim, e nesta ordem: o bloqueio suja o par (e-mail, origem)
	// pelos 15 minutos do prazo, e não há como desfazê-lo sem apagar chave do
	// Redis por fora da API. Encerrar Sessão vem antes pelo mesmo motivo — mas
	// abre a sua própria, para não matar o cookieValido de quem veio acima.
	t.Run("ler a Sessão renova o prazo", func(t *testing.T) {
		sessaoLidaRenovaOPrazo(t, rotas, rdb, cookieValido)
	})
	t.Run("encerrar apaga a chave no Redis e expira o cookie", func(t *testing.T) {
		encerrarApagaChaveECookie(t, rotas, rdb)
	})
	t.Run("encerrar sem Sessão responde o mesmo 204", func(t *testing.T) {
		encerrarSemSessaoDa204(t, rotas)
	})
	t.Run("o login válido esquece as falhas do par", func(t *testing.T) {
		loginValidoEsqueceAsFalhas(t, rotas, rdb)
	})
	t.Run("o par bloqueia no limiar, exista ou não a conta", func(t *testing.T) {
		bloqueioDepoisDoLimiar(t, rotas)
	})

	// A 2.3 depois do bloqueio: as contas são próprias, e o contador de
	// tentativas que o subteste acima deixou sujo é do par (e-mail, origem) —
	// não alcança nenhuma delas.
	t.Run("a solicitação responde o mesmo exista ou não a conta", func(t *testing.T) {
		solicitacaoNaoVazaConta(t, rotas, rdb)
	})
	t.Run("redefinir troca a senha e derruba todas as Sessões do dono", func(t *testing.T) {
		redefinicaoTrocaSenhaEDerrubaSessoes(t, rotas, rdb)
	})
	t.Run("token expirado, já usado ou inexistente sai em 404", func(t *testing.T) {
		tokenInvalidoDa404(t, rotas, rdb)
	})
	// Por último de tudo: este suja o contador de dois pares por 15 minutos, e
	// o vizinho fica bloqueado de propósito até o fim da suíte.
	t.Run("redefinir tira do bloqueio por tentativas, e só o do dono", func(t *testing.T) {
		redefinirTiraDoBloqueio(t, rotas)
	})

	// A 2.4 no fim: as contas são próprias (o segundo Comprador nasce aqui, e o
	// Administrador vem da semente), e nenhum par que os subtestes acima
	// sujaram é usado.
	t.Run("Pedido de outro dono é indistinguível de Pedido inexistente", func(t *testing.T) {
		negacaoPorDono(t, rotas, cookieValido, pedidoCriado)
	})
	t.Run("cada login consulta só a sua tabela", func(t *testing.T) {
		loginPorTabelaPropria(t, rotas)
	})
	t.Run("a guarda do prefixo administrativo devolve 404", func(t *testing.T) {
		guardaDoPrefixoAdministrativo(t, rotas, rdb, cookieValido)
	})
	t.Run("corpo sem Content-Type de JSON sai em 400", func(t *testing.T) {
		corpoSemContentTypeDa400(t, rotas)
	})

	// A 2.5 no fim, com contas próprias: o teto por Comprador sai da conta
	// cheia, e nenhum subteste acima conta com a lista de Endereços de ninguém.
	// O Pedido que a remoção não pode tocar nasce aqui — depois da simulação de
	// entrega, que é quem conta com o `estoque_total` intacto.
	t.Run("os Endereços do Comprador: lista, cadastro, edição e remoção", func(t *testing.T) {
		enderecosDoComprador(t, rotas, pool)
	})
	t.Run("as rotas de Endereço exigem Sessão de Comprador", func(t *testing.T) {
		enderecoExigeSessaoDeComprador(t, rotas)
	})

	// A 3.1 com Vendedor próprio: desativar um semeado esconderia Produtos de
	// que os subtestes acima dependem.
	t.Run("a gestão de Vendedores e o Produto de Vendedor desativado", func(t *testing.T) {
		gestaoDeVendedores(t, rotas, pool)
	})
	// A 3.2 e a 3.3, também com Vendedor, Categoria e Produtos próprios.
	t.Run("a gestão de Categorias e a recusa com Produtos vinculados", func(t *testing.T) {
		gestaoDeCategorias(t, rotas, pool)
	})
	// A 3.4, com Produto próprio: o ajuste do Estoque total, a guarda das
	// Reservas e a interface de Estoque chamada direto.
	t.Run("o ajuste do Estoque, a guarda das Reservas e o predicado de visibilidade", func(t *testing.T) {
		estoqueEReservas(t, rotas, pool)
	})
	t.Run("a gestão de Produtos, a listagem administrativa e o Produto desativado", func(t *testing.T) {
		gestaoDeProdutos(t, rotas, pool)
	})
	// A 3.5, com Produto próprio desativado: o invisível fica fora da Vitrine.
	t.Run("a Vitrine: o envelope de listagem e só Produto visível", func(t *testing.T) {
		vitrine(t, rotas, pool)
	})
	// A 3.7–3.9, com Produtos próprios de nome acentuado, `%` e `_`.
	t.Run("a busca: termo, Categoria, faixa de preço e ordenação", func(t *testing.T) {
		buscaFiltrosEOrdenacao(t, rotas, pool)
	})
	// A 4.1, com conta própria.
	t.Run("o Carrinho: adição, soma, teto, Produto invisível e remoção", func(t *testing.T) {
		carrinhoDoComprador(t, rotas, pool)
	})
	// A 4.2 e a 4.3, com contas e Produtos próprios.
	t.Run("o Carrinho: recusa por Estoque, alteração, leitura e esvaziar", func(t *testing.T) {
		carrinhoNaTela(t, rotas, pool)
	})
	// A 4.4, com conta e Produtos próprios.
	t.Run("o Carrinho: revalidação de preço, Estoque e visibilidade", func(t *testing.T) {
		carrinhoRevalidado(t, rotas, pool)
	})
	// A 5.1, com conta e Produto próprios: a máquina de estados contra o banco.
	t.Run("a máquina de estados: efeitos, recusas, histórico e imutabilidade", func(t *testing.T) {
		maquinaDeEstados(t, rotas, pool)
	})
	// A 5.3, com conta, Endereços e Produto próprios: a Regra de Frete da migração.
	t.Run("o Frete: faixa, região padrão, isenção e Endereço alheio", func(t *testing.T) {
		freteDoCarrinho(t, rotas, pool)
	})
	// A 5.5, com contas, Produtos e Endereço próprios: a entrada no checkout
	// reporta a mudança de preço e grava a ciência dela na mesma transação.
	t.Run("a entrada no checkout: reporta, confirma e trava o avanço", func(t *testing.T) {
		entradaNoCheckout(t, rotas, pool)
	})
	// A 5.6, com contas, Produtos e Endereços próprios: a criação do Pedido a
	// partir do Carrinho, a idempotência e as recusas.
	t.Run("a criação do Pedido: Carrinho, Reserva atômica, idempotência e recusas", func(t *testing.T) {
		criacaoPeloCheckout(t, rotas, pool)
	})
	// A 5.7, o critério de aceite da NFR-7: N checkouts paralelos na última
	// unidade produzem exatamente o Estoque inicial em Pedidos.
	t.Run("a consistência de Estoque sob concorrência", func(t *testing.T) {
		consistenciaSobConcorrencia(t, rotas, pool)
	})

	// A 5.8: o que a tela do Pedido lê numa chamada só (AD-18).
	t.Run("a tela do Pedido: prazo, tripla, valores, Endereço e histórico", func(t *testing.T) {
		detalheDoPedido(t, rotas, pool)
	})
	// A 5.10, com contas e Produtos próprios: a recusa aplicada e a nova
	// Tentativa. A emissão e a varredura são globais, mas o transporte entrega
	// só as Tentativas dos Pedidos dela — os dos subtestes acima não mudam.
	t.Run("a recusa do pagamento e a nova Tentativa", func(t *testing.T) {
		recusaENovaTentativa(t, rotas, pool)
	})
	// A 5.11, com contas e Produtos próprios: a Tentativa de Pagamento que
	// vence sem confirmação (FR-34). A expiração é global como a emissão, mas
	// o prazo é largo e só os Pedidos dela são envelhecidos — os dos subtestes
	// acima não vencem.
	t.Run("a expiração da Tentativa de Pagamento", func(t *testing.T) {
		expiracaoDaTentativa(t, rotas, pool)
	})
}

func postar(t *testing.T, rotas http.Handler, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	return postarDe(t, rotas, corpo, origemDeTeste)
}

// postarDe é o postar com a origem escolhida: o contador de tentativas chaveia
// pelo par (e-mail, origem), e sem mexer no RemoteAddr não há como provar que
// a segunda origem não herda o bloqueio da primeira.
func postarDe(t *testing.T, rotas http.Handler, corpo, origem string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessoes", strings.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = origem
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
}

// chavesCom devolve as chaves do Redis que casam com o trecho. As buscas são
// por trecho, e nunca pelo prefixo literal, para o teste não repetir o espaço
// de nomes que mora em identidade.
func chavesCom(t *testing.T, rdb *redis.Client, trecho string) []string {
	t.Helper()
	chaves, err := rdb.Keys(context.Background(), "*"+trecho+"*").Result()
	if err != nil {
		t.Fatalf("procurar %q no Redis: %v", trecho, err)
	}
	return chaves
}

// sessaoLidaRenovaOPrazo é a expiração por inatividade do §7.1: toda leitura da
// Sessão devolve o prazo cheio. O prazo é encurtado à mão antes da leitura —
// sem isso o TTL já está cheio, e uma leitura que não renovasse passaria
// batida.
func sessaoLidaRenovaOPrazo(t *testing.T, rotas http.Handler, rdb *redis.Client, cookie *http.Cookie) {
	if cookie == nil {
		t.Skip("sem cookie: o login falhou antes")
	}
	ctx := context.Background()
	chaves := chavesCom(t, rdb, cookie.Value)
	if len(chaves) != 1 {
		t.Fatalf("chaves para o cookie = %v; quero exatamente uma", chaves)
	}
	if err := rdb.Expire(ctx, chaves[0], time.Minute).Err(); err != nil {
		t.Fatalf("encurtar o prazo: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessao", nil)
	req.AddCookie(cookie)
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d (%s), quero 200", resp.Code, resp.Body.String())
	}

	ttl, err := rdb.TTL(ctx, chaves[0]).Result()
	if err != nil || ttl > expiracaoDeTeste || ttl < expiracaoDeTeste-time.Minute {
		t.Errorf("TTL depois da leitura = %v, %v; quero de volta perto de %v", ttl, err, expiracaoDeTeste)
	}

	// E o outro lado: Sessão inexistente continua em 401 sem criar chave — o
	// GetEx de uma chave ausente não pode virar um SET disfarçado, ou qualquer
	// valor de 64 hexadecimais povoaria o espaço de nomes.
	desconhecido := strings.Repeat("a", 64)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessao", nil)
	req.AddCookie(&http.Cookie{Name: "azamon_sessao", Value: desconhecido})
	resp = httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("Sessão desconhecida = %d, quero 401", resp.Code)
	}
	if chaves := chavesCom(t, rdb, desconhecido); len(chaves) != 0 {
		t.Errorf("a leitura criou %v; ler Sessão inexistente não grava nada", chaves)
	}
}

// encerrarApagaChaveECookie é a condição de aceite da estória: a chave some do
// Redis, e não só o cookie — o mesmo cookie não volta a valer em rota nenhuma.
// A Sessão é aberta aqui mesmo para o cookieValido dos outros subtestes
// continuar vivo.
func encerrarApagaChaveECookie(t *testing.T, rotas http.Handler, rdb *redis.Client) {
	login := postar(t, rotas, `{"email":"`+emailSemente+`","senha":"`+senhaSemente+`"}`)
	if login.Code != http.StatusOK {
		t.Fatalf("login = %d (%s), quero 200", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("quero exatamente um cookie, vieram %d", len(cookies))
	}
	cookie := cookies[0]
	if chaves := chavesCom(t, rdb, cookie.Value); len(chaves) != 1 {
		t.Fatalf("chaves para o cookie = %v; quero exatamente uma antes de encerrar", chaves)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessao", nil)
	req.AddCookie(cookie)
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d (%s), quero 204", resp.Code, resp.Body.String())
	}
	if resp.Body.Len() != 0 {
		t.Errorf("204 com corpo: %q", resp.Body.String())
	}

	// O cookie volta expirado, e com Nome e Path iguais aos da emissão — o
	// navegador só apaga o cookie quando os dois batem.
	saida := resp.Result().Cookies()
	if len(saida) != 1 {
		t.Fatalf("quero exatamente um cookie na saída, vieram %d", len(saida))
	}
	if saida[0].Name != "azamon_sessao" || saida[0].Path != "/" || saida[0].Value != "" || saida[0].MaxAge >= 0 {
		t.Errorf("cookie de saída = %+v; quero o mesmo nome e caminho, vazio e expirado", saida[0])
	}

	if chaves := chavesCom(t, rdb, cookie.Value); len(chaves) != 0 {
		t.Errorf("a chave %v sobreviveu ao DELETE; o cookie sozinho não invalida nada", chaves)
	}

	// E o cookie encerrado não volta: a rota autenticada recusa.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessao", nil)
	req.AddCookie(cookie)
	resp = httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET com o cookie encerrado = %d, quero 401", resp.Code)
	}
	if codigo := decodificar(t, resp)["codigo"]; codigo != "SESSAO_INVALIDA" {
		t.Errorf("codigo = %v, quero SESSAO_INVALIDA", codigo)
	}
}

// encerrarSemSessaoDa204: sair é idempotente. Um 401 aqui diria ao chamador
// que aquele cookie valia alguma coisa.
func encerrarSemSessaoDa204(t *testing.T, rotas http.Handler) {
	for _, cookie := range []*http.Cookie{
		nil,
		{Name: "azamon_sessao", Value: strings.Repeat("0", 64)},
		{Name: "azamon_sessao", Value: "curto-demais"},
	} {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessao", nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		resp := httptest.NewRecorder()
		rotas.ServeHTTP(resp, req)

		if resp.Code != http.StatusNoContent {
			t.Errorf("cookie %v: status = %d (%s), quero 204", cookie, resp.Code, resp.Body.String())
		}
		if resp.Body.Len() != 0 {
			t.Errorf("cookie %v: 204 com corpo %q", cookie, resp.Body.String())
		}
	}
}

// loginValidoEsqueceAsFalhas: quem erra quatro vezes e acerta na quinta não
// pode ficar com quatro falhas penduradas. A conta é criada aqui para o
// contador partir limpo — emailSemente já acumulou falha nos subtestes acima.
func loginValidoEsqueceAsFalhas(t *testing.T, rotas http.Handler, rdb *redis.Client) {
	const (
		email = "carlos@exemplo.br"
		senha = "senha-do-carlos-1"
	)
	if resp := postarCadastro(t, rotas, `{"nome":"Carlos Dias","email":"`+email+`","senha":"`+senha+`"}`); resp.Code != http.StatusCreated {
		t.Fatalf("cadastro = %d (%s), quero 201", resp.Code, resp.Body.String())
	}

	errada := `{"email":"` + email + `","senha":"nao-e-a-senha"}`
	if resp := postar(t, rotas, errada); resp.Code != http.StatusUnauthorized {
		t.Fatalf("primeira falha = %d (%s), quero 401", resp.Code, resp.Body.String())
	}
	chaves := chavesCom(t, rdb, email)
	if len(chaves) != 1 {
		t.Fatalf("contador do par = %v; quero exatamente um depois da falha", chaves)
	}

	// O contador tem de nascer com prazo: sem o EXPIRE ele fica para sempre, e
	// o par nunca mais sai do bloqueio.
	ctx := context.Background()
	ttl, err := rdb.TTL(ctx, chaves[0]).Result()
	if err != nil || ttl > bloqueioDeTeste || ttl < bloqueioDeTeste-time.Minute {
		t.Fatalf("TTL do contador = %v, %v; quero perto de %v", ttl, err, bloqueioDeTeste)
	}

	// E a janela não desliza: o prazo conta da primeira falha, e não da última.
	// Encurtá-lo à mão é o que torna isso observável — as falhas acontecem em
	// milissegundos, e o TTL em segundos não distinguiria as duas leituras.
	if err := rdb.Expire(ctx, chaves[0], time.Minute).Err(); err != nil {
		t.Fatalf("encurtar o prazo do contador: %v", err)
	}
	for i := 2; i < tentativasDeTeste; i++ {
		if resp := postar(t, rotas, errada); resp.Code != http.StatusUnauthorized {
			t.Fatalf("falha %d = %d (%s), quero 401 abaixo do limiar", i, resp.Code, resp.Body.String())
		}
	}
	if ttl, err := rdb.TTL(ctx, chaves[0]).Result(); err != nil || ttl > time.Minute {
		t.Errorf("TTL depois das outras falhas = %v, %v; a janela voltou ao cheio e passou a deslizar", ttl, err)
	}

	resp := postar(t, rotas, `{"email":"`+email+`","senha":"`+senha+`"}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("login válido = %d (%s), quero 200", resp.Code, resp.Body.String())
	}
	if len(resp.Result().Cookies()) != 1 {
		t.Errorf("o login válido não emitiu cookie")
	}
	if chaves := chavesCom(t, rdb, email); len(chaves) != 0 {
		t.Errorf("o contador %v sobreviveu ao login válido", chaves)
	}
}

// bloqueioDepoisDoLimiar cobre as três linhas do bloqueio: a resposta depois do
// limiar, a igualdade entre conta existente e inexistente, e a origem que não
// herda. A conta de carlos vem do subteste anterior, e sai daqui bloqueada —
// por isso este é o último.
func bloqueioDepoisDoLimiar(t *testing.T, rotas http.Handler) {
	const existente = "carlos@exemplo.br"
	const inexistente = "ninguem-bloqueado@azamon.test"

	// A tentativa que ultrapassa o limiar chega com outra caixa e com bordas: a
	// chave do contador é pelo e-mail NORMALIZADO, e sem isso bastaria mudar
	// uma maiúscula para ganhar mais cinco tentativas.
	var corpos []string
	for _, caso := range []struct{ email, comOutraCaixa string }{
		{existente, "  CARLOS@Exemplo.BR  "},
		{inexistente, " Ninguem-Bloqueado@Azamon.TEST "},
	} {
		email := caso.email
		entrada := `{"email":"` + email + `","senha":"nao-e-a-senha"}`
		for i := 1; i <= tentativasDeTeste; i++ {
			if resp := postar(t, rotas, entrada); resp.Code != http.StatusUnauthorized {
				t.Fatalf("%s: tentativa %d = %d, quero 401 até o limiar", email, i, resp.Code)
			}
		}

		resp := postar(t, rotas, `{"email":"`+caso.comOutraCaixa+`","senha":"nao-e-a-senha"}`)
		if resp.Code != http.StatusTooManyRequests {
			t.Fatalf("%s: depois de %d falhas o status = %d (%s), quero 429",
				email, tentativasDeTeste, resp.Code, resp.Body.String())
		}
		if len(resp.Result().Cookies()) != 0 {
			t.Errorf("%s: o 429 veio com cookie", email)
		}
		envelope := decodificar(t, resp)
		if envelope["codigo"] != "MUITAS_TENTATIVAS" {
			t.Errorf("%s: codigo = %v", email, envelope["codigo"])
		}
		// O limiar vem da Config e tem de ser nomeado na mensagem: um texto
		// sem os minutos deixaria o Comprador sem saber quando tentar de novo.
		if mensagem := fmt.Sprint(envelope["mensagem"]); !strings.Contains(mensagem, "15") {
			t.Errorf("%s: mensagem = %q; os minutos configurados têm de aparecer", email, mensagem)
		}
		delete(envelope, "correlacao")
		corpos = append(corpos, fmt.Sprint(envelope))
	}
	// Se o bloqueio só valesse para conta existente, a própria resposta
	// confirmaria o e-mail — a correlação é a única coisa que pode diferir.
	if corpos[0] != corpos[1] {
		t.Errorf("o bloqueio distingue conta existente de inexistente:\n%s\n%s", corpos[0], corpos[1])
	}

	// A senha certa também leva 429: é o que prova que a consulta vem antes do
	// Autenticar, e que o bloqueio é também o limite de custo da rota.
	if resp := postar(t, rotas, `{"email":"`+existente+`","senha":"senha-do-carlos-1"}`); resp.Code != http.StatusTooManyRequests {
		t.Errorf("senha certa sob bloqueio = %d, quero 429 sem gastar Argon2id", resp.Code)
	}

	// Outra origem não herda o bloqueio: o contador é do par.
	resp := postarDe(t, rotas, `{"email":"`+existente+`","senha":"nao-e-a-senha"}`, "198.51.100.7:4321")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("outra origem = %d (%s), quero 401 — o bloqueio é do par", resp.Code, resp.Body.String())
	}
	if codigo := decodificar(t, resp)["codigo"]; codigo != "CREDENCIAL_INVALIDA" {
		t.Errorf("outra origem: codigo = %v, quero CREDENCIAL_INVALIDA", codigo)
	}
}

// decodificar devolve o corpo do sucesso ou o miolo do envelope de erro —
// nunca os dois, porque o envelope tem sempre a chave "erro".
func decodificar(t *testing.T, resp *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var corpo map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo não é JSON: %v (%s)", err, resp.Body.String())
	}
	if envelope, ok := corpo["erro"].(map[string]any); ok {
		return envelope
	}
	return corpo
}
