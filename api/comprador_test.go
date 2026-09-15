package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Os subtestes do cadastro. Entram como subteste de TestSessaoEProduto, no
// ambiente(t) que já sobe Postgres e Redis uma vez para o pacote inteiro.

// postarCadastro é o postar dos outros arquivos apontado para a rota nova.
func postarCadastro(t *testing.T, rotas http.Handler, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compradores", strings.NewReader(corpo))
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
}

// cadastroValidoAbreSessao é o caminho feliz e as duas condições de aceite que
// dependem dele: o cookie devolvido já lê a Sessão sem passar pelo login, e a
// linha gravada tem o e-mail normalizado, o PHC do AD-9 e nenhuma senha em
// claro. A mesma senha ainda autentica por POST /api/v1/sessoes.
func cadastroValidoAbreSessao(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	// Com bordas e maiúsculas de propósito: a linha "e-mail não normalizado"
	// da matriz sai do mesmo cadastro.
	const (
		email = " Ana@Exemplo.br "
		senha = "senha-da-ana-1"
		nome  = "Ana Prado"
	)

	resp := postarCadastro(t, rotas, `{"nome":"`+nome+`","email":"`+email+`","senha":"`+senha+`"}`)
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d (%s), quero 201", resp.Code, resp.Body.String())
	}
	if devolvido := decodificar(t, resp)["nome"]; devolvido != nome {
		t.Errorf("nome = %v, quero %q", devolvido, nome)
	}

	cookies := resp.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("quero exatamente um cookie, vieram %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != "azamon_sessao" || len(c.Value) != 64 {
		t.Fatalf("cookie = %q com %d caracteres; quero azamon_sessao com 64", c.Name, len(c.Value))
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.MaxAge != int(expiracaoDeTeste.Seconds()) {
		t.Errorf("cookie do cadastro difere do cookie do login: %+v", c)
	}

	// Condição de aceite: o cookie do cadastro lê a Sessão sem nenhuma chamada
	// a POST /api/v1/sessoes.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessao", nil)
	req.AddCookie(c)
	sessao := httptest.NewRecorder()
	rotas.ServeHTTP(sessao, req)
	if sessao.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/sessao = %d (%s), quero 200", sessao.Code, sessao.Body.String())
	}
	if devolvido := decodificar(t, sessao)["nome"]; devolvido != nome {
		t.Errorf("nome na Sessão = %v, quero %q", devolvido, nome)
	}

	// Condição de aceite: a linha gravada. O e-mail entrou minúsculo e sem as
	// bordas, o hash é o PHC do AD-9 e a senha em claro não está em coluna
	// nenhuma — uma consulta pelo texto cobre as três colunas de texto.
	var gravadoEmail, gravadoHash string
	err := pool.QueryRow(context.Background(),
		`SELECT email, senha_hash FROM identidade.comprador WHERE nome = $1`, nome).
		Scan(&gravadoEmail, &gravadoHash)
	if err != nil {
		t.Fatalf("ler a linha gravada: %v", err)
	}
	if gravadoEmail != "ana@exemplo.br" {
		t.Errorf("email gravado = %q, quero normalizado", gravadoEmail)
	}
	if !strings.HasPrefix(gravadoHash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("senha_hash = %q; quero o PHC do AD-9", gravadoHash)
	}
	if strings.Contains(gravadoHash, senha) {
		t.Error("a senha em claro aparece no senha_hash")
	}
	var emClaro int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM identidade.comprador WHERE nome = $1 OR email = $1 OR senha_hash = $1`,
		senha).Scan(&emClaro); err != nil {
		t.Fatalf("procurar a senha em claro: %v", err)
	}
	if emClaro != 0 {
		t.Errorf("a senha em claro aparece em %d linha(s) de identidade.comprador", emClaro)
	}

	// E a senha do cadastro autentica pelo login, que é o que prova que Gerar
	// e Verificar concordam de ponta a ponta.
	login := postar(t, rotas, `{"email":"ana@exemplo.br","senha":"`+senha+`"}`)
	if login.Code != http.StatusOK {
		t.Fatalf("login com a senha cadastrada = %d (%s), quero 200", login.Code, login.Body.String())
	}
}

// cadastroDuplicadoDa409 cobre as duas linhas de duplicidade: o e-mail da
// semente e o e-mail que só colide depois de normalizado.
func cadastroDuplicadoDa409(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	antes := contarCompradores(t, pool)

	for _, email := range []string{emailSemente, strings.ToUpper(emailSemente), "ANA@exemplo.BR"} {
		resp := postarCadastro(t, rotas, `{"nome":"Outra Pessoa","email":"`+email+`","senha":"senha-qualquer-1"}`)
		if resp.Code != http.StatusConflict {
			t.Fatalf("%s: status = %d (%s), quero 409", email, resp.Code, resp.Body.String())
		}
		envelope := decodificar(t, resp)
		if envelope["codigo"] != "EMAIL_JA_CADASTRADO" {
			t.Errorf("%s: codigo = %v", email, envelope["codigo"])
		}
		if envelope["mensagem"] != "Já existe uma conta com este e-mail." {
			t.Errorf("%s: mensagem = %v", email, envelope["mensagem"])
		}
		// O contrato do erro em linha: a tela liga a mensagem ao campo por aqui.
		dados, _ := envelope["dados"].(map[string]any)
		if dados["campo"] != "email" {
			t.Errorf("%s: dados.campo = %v, quero \"email\"", email, dados["campo"])
		}
		if len(resp.Result().Cookies()) != 0 {
			t.Errorf("%s: o 409 veio com cookie", email)
		}
	}

	if depois := contarCompradores(t, pool); depois != antes {
		t.Errorf("compradores = %d depois das recusas, eram %d; nada devia ter sido gravado", depois, antes)
	}
}

// cadastroComCampoInvalido é a parte da matriz que sai em 400 nomeando o
// campo. Nada é gravado, e a senha curta é recusada antes do Argon2id.
func cadastroComCampoInvalido(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	antes := contarCompradores(t, pool)

	casos := []struct {
		nome   string
		corpo  string
		campo  string
		contem string // trecho que a mensagem tem de nomear
	}{
		{"senha curta", `{"nome":"Bia","email":"bia@exemplo.br","senha":"1234567"}`, "senha", "8"},
		{"senha vazia", `{"nome":"Bia","email":"bia@exemplo.br","senha":""}`, "senha", "8"},
		{"senha acima do teto", `{"nome":"Bia","email":"bia@exemplo.br","senha":"` + strings.Repeat("a", senhaMaxDeTeste+1) + `"}`, "senha", "128"},
		{"e-mail sem forma de e-mail", `{"nome":"Bia","email":"ana","senha":"senha-da-bia-1"}`, "email", ""},
		{"e-mail com nome de exibição", `{"nome":"Bia","email":"Bia <bia@exemplo.br>","senha":"senha-da-bia-1"}`, "email", ""},
		{"e-mail vazio", `{"nome":"Bia","email":"","senha":"senha-da-bia-1"}`, "email", ""},
		// A RFC 5322 aceita domínio sem ponto; a mensagem que este caso dispara
		// promete "nome@dominio.com", e as duas coisas têm de concordar.
		{"domínio sem ponto", `{"nome":"Bia","email":"ana@exemplo","senha":"senha-da-bia-1"}`, "email", ""},
		{"domínio de uma letra só", `{"nome":"Bia","email":"a@b","senha":"senha-da-bia-1"}`, "email", ""},
		{"nome vazio", `{"nome":"   ","email":"bia@exemplo.br","senha":"senha-da-bia-1"}`, "nome", ""},
		{
			"nome acima do teto",
			`{"nome":"` + strings.Repeat("á", nomeMaxDeTeste+1) + `","email":"bia@exemplo.br","senha":"senha-da-bia-1"}`,
			"nome", "120",
		},
		{
			// Acima do teto do e-mail mas ainda dentro dos 4 KiB do corpo: o
			// que recusa é o limiar da Config, e não o MaxBytesReader.
			"e-mail acima do teto",
			`{"nome":"Bia","email":"` + strings.Repeat("b", emailMaxDeTeste) + `@exemplo.br","senha":"senha-da-bia-1"}`,
			"email", "254",
		},
		{
			// O teto do e-mail é medido em octetos, que é a unidade da RFC
			// 5321: são 211 runas e 411 bytes. Contado em runas, este endereço
			// passaria pelo limite.
			"e-mail acentuado acima do teto em bytes",
			`{"nome":"Bia","email":"` + strings.Repeat("é", 200) + `@exemplo.br","senha":"senha-da-bia-1"}`,
			"email", "254",
		},
	}
	for _, caso := range casos {
		resp := postarCadastro(t, rotas, caso.corpo)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d (%s), quero 400", caso.nome, resp.Code, resp.Body.String())
		}
		envelope := decodificar(t, resp)
		if envelope["codigo"] != "CAMPO_INVALIDO" {
			t.Errorf("%s: codigo = %v, quero CAMPO_INVALIDO", caso.nome, envelope["codigo"])
		}
		dados, _ := envelope["dados"].(map[string]any)
		if dados["campo"] != caso.campo {
			t.Errorf("%s: dados.campo = %v, quero %q", caso.nome, dados["campo"], caso.campo)
		}
		mensagem := fmt.Sprint(envelope["mensagem"])
		if caso.contem != "" && !strings.Contains(mensagem, caso.contem) {
			t.Errorf("%s: mensagem = %q; o limiar %q tem de ser nomeado", caso.nome, mensagem, caso.contem)
		}
		if len(resp.Result().Cookies()) != 0 {
			t.Errorf("%s: o 400 veio com cookie", caso.nome)
		}
	}

	// Corpo malformado e corpo grande demais continuam em ENTRADA_INVALIDA: é
	// o envelope genérico do AD-14, e não o erro em linha.
	for _, corpo := range []string{
		`{isso não é json`,
		`{"nome":"` + strings.Repeat("a", corpoMaximo) + `","email":"c@exemplo.br","senha":"senha-longa-1"}`,
	} {
		resp := postarCadastro(t, rotas, corpo)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, quero 400 (%.40s)", resp.Code, corpo)
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != "ENTRADA_INVALIDA" {
			t.Errorf("codigo = %v, quero ENTRADA_INVALIDA (%.40s)", codigo, corpo)
		}
	}

	if depois := contarCompradores(t, pool); depois != antes {
		t.Errorf("compradores = %d depois das recusas, eram %d; nada devia ter sido gravado", depois, antes)
	}
}

func contarCompradores(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM identidade.comprador`).Scan(&n); err != nil {
		t.Fatalf("contar compradores: %v", err)
	}
	return n
}
