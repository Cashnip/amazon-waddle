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

	// Identificador v5 derivado do nome em media/gerar.go — estável por
	// construção, e por isso escrevível no teste e no front.
	produtoSemeado = "3400cd00-3f5e-5433-9171-fde099a52005"
)

// expiracaoDeTeste são os mesmos 7 dias do AZAMON_SESSAO_EXPIRACAO da
// demonstração: o que o teste confere é que o valor configurado chega inteiro
// aos dois lugares que prometem prazo — o Max-Age do cookie e o TTL no Redis.
const expiracaoDeTeste = 7 * 24 * time.Hour

func ambiente(t *testing.T) (http.Handler, *redis.Client) {
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

	return Rotas(plataforma.Config{SessaoExpiracao: expiracaoDeTeste}, pool, rdb), rdb
}

func TestSessaoEProduto(t *testing.T) {
	rotas, rdb := ambiente(t)

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

	t.Run("detalhe do Produto semeado", func(t *testing.T) { produtoSemeadoSai(t, rotas) })
	t.Run("Produto inexistente e identificador malformado", func(t *testing.T) { produtoAusenteDa404(t, rotas) })
}

func postar(t *testing.T, rotas http.Handler, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessoes", strings.NewReader(corpo))
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
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
