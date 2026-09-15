package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// Os subtestes da recuperação de senha (2.3). Entram como subteste de
// TestSessaoEProduto, no ambiente(t) que já sobe Postgres e Redis uma vez para
// o pacote inteiro.
//
// Duas linhas da matriz não são observáveis de fora: "a solicitação nova
// invalida o token anterior" e "só as Sessões do dono somem". A primeira só
// aparece no estado do Redis, e a segunda é justamente o que uma resposta não
// conta — por isso estes testes olham as chaves, e não só os status.

// postarRedefinicao solicita a redefinição.
func postarRedefinicao(t *testing.T, rotas http.Handler, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/redefinicoes-de-senha", strings.NewReader(corpo))
	req.RemoteAddr = origemDeTeste
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
}

// putRedefinicao consome o token com a senha nova.
func putRedefinicao(t *testing.T, rotas http.Handler, token, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/redefinicoes-de-senha/"+token, strings.NewReader(corpo))
	req.RemoteAddr = origemDeTeste
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
}

// solicitar faz a solicitação com o log capturado e devolve o token que a linha
// revelou. Não há serviço de e-mail (AD-20): o log é o único canal por onde o
// token sai, e o teste o lê pelo mesmo caminho que o Comprador da demonstração
// — `docker compose logs`. Token vazio quer dizer que nenhuma linha saiu.
//
// De carona vai a condição de aceite do AD-15: a linha tem correlação e
// caminho, e nunca o e-mail.
func solicitar(t *testing.T, rotas http.Handler, email string) (*httptest.ResponseRecorder, string) {
	t.Helper()

	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(plataforma.NovoLogger(&registro, "api"))
	resp := postarRedefinicao(t, rotas, `{"email":`+string(marcarJSON(t, email))+`}`)
	slog.SetDefault(anterior)

	saida := registro.String()
	if email != "" && strings.Contains(saida, email) {
		t.Errorf("o e-mail aparece no log (AD-15): %s", saida)
	}

	var token string
	for _, linha := range strings.Split(strings.TrimSpace(saida), "\n") {
		if linha == "" {
			continue
		}
		var campos map[string]any
		if err := json.Unmarshal([]byte(linha), &campos); err != nil {
			t.Fatalf("linha de log não é JSON: %v (%s)", err, linha)
		}
		caminho, _ := campos["caminho"].(string)
		if !strings.HasPrefix(caminho, "/redefinir-senha/") {
			continue
		}
		// O AD-15 quer correlação em toda linha: sem ela o caminho não é
		// rastreável até a requisição que o pediu.
		if correlacao, _ := campos["correlacao"].(string); correlacao == "" {
			t.Errorf("a linha do caminho saiu sem correlação: %s", linha)
		}
		token = strings.TrimPrefix(caminho, "/redefinir-senha/")
	}
	return resp, token
}

// marcarJSON escreve o e-mail como literal JSON — os casos da matriz incluem
// endereços com aspas e caracteres que quebrariam a concatenação crua.
func marcarJSON(t *testing.T, valor string) []byte {
	t.Helper()
	b, err := json.Marshal(valor)
	if err != nil {
		t.Fatalf("serializar %q: %v", valor, err)
	}
	return b
}

// cabecalhosComparaveis é a resposta sem a correlação, em ordem estável. A
// correlação é por requisição e sempre difere; todo o resto tem de bater, ou os
// dois caminhos se distinguem de fora sem ninguém ler o corpo.
func cabecalhosComparaveis(resp *httptest.ResponseRecorder) string {
	cabecalhos := resp.Header().Clone()
	cabecalhos.Del("X-Correlation-Id")
	nomes := slices.Sorted(maps.Keys(cabecalhos))
	var b strings.Builder
	for _, nome := range nomes {
		fmt.Fprintf(&b, "%s=%v;", nome, cabecalhos[nome])
	}
	return b.String()
}

// tokensDeRedefinicao são as chaves `redefinicao:<token>` vivas. O ponteiro
// reverso (`redefinicao-de:<id>`) não casa com o trecho, que é o que permite
// contar tokens sem contar ponteiros.
func tokensDeRedefinicao(t *testing.T, rdb *redis.Client) []string {
	t.Helper()
	return chavesCom(t, rdb, "redefinicao:")
}

// solicitacaoNaoVazaConta é a metade da estória que se prova por igualdade:
// toda entrada sai no MESMO 202 de corpo vazio, e a conta que não existe não
// deixa chave nem linha de log.
func solicitacaoNaoVazaConta(t *testing.T, rotas http.Handler, rdb *redis.Client) {
	const email = "davi@exemplo.br"
	const senha = "senha-do-davi-1"
	if resp := postarCadastro(t, rotas, `{"nome":"Davi Luz","email":"`+email+`","senha":"`+senha+`"}`); resp.Code != http.StatusCreated {
		t.Fatalf("cadastro = %d (%s), quero 201", resp.Code, resp.Body.String())
	}

	antes := len(tokensDeRedefinicao(t, rdb))

	// A conta existente vem primeiro: é a única entrada que pode criar chave e
	// escrever linha, e todas as outras são comparadas com ela.
	respExistente, token := solicitar(t, rotas, email)
	if token == "" {
		t.Fatal("a conta existente não gerou linha de log com o caminho")
	}
	// A rota é por Comprador e não pode ser guardada por intermediário nenhum.
	if cache := respExistente.Header().Get("Cache-Control"); cache != "no-store" {
		t.Errorf("Cache-Control = %q, quero no-store", cache)
	}
	if depois := len(tokensDeRedefinicao(t, rdb)); depois != antes+1 {
		t.Fatalf("tokens = %d depois da solicitação, eram %d; quero exatamente um a mais", depois, antes)
	}

	// O prazo é o da Config, e é nativo do Redis: um token sem TTL ficaria
	// válido para sempre, e um TTL curto demais expiraria antes do Comprador
	// chegar ao link.
	chaves := chavesCom(t, rdb, token)
	if len(chaves) != 1 {
		t.Fatalf("chaves para o token = %v; quero exatamente uma", chaves)
	}
	ttl, err := rdb.TTL(context.Background(), chaves[0]).Result()
	if err != nil || ttl > validadeDeTeste || ttl < validadeDeTeste-time.Minute {
		t.Errorf("TTL do token = %v, %v; quero perto de %v", ttl, err, validadeDeTeste)
	}

	depoisDaExistente := len(tokensDeRedefinicao(t, rdb))

	// As entradas que não podem produzir nada. O e-mail acima do teto e o corpo
	// que nem é JSON saem antes da consulta ao Postgres, e mesmo assim respondem
	// o mesmo que a conta existente.
	casos := []struct {
		nome  string
		corpo string
	}{
		{"conta inexistente", `{"email":"ninguem-redefine@azamon.test"}`},
		{"e-mail acima do teto", `{"email":"` + strings.Repeat("z", emailMaxDeTeste) + `@exemplo.br"}`},
		{"corpo malformado", `{isso não é json`},
		{"corpo grande demais", `{"email":"` + strings.Repeat("z", corpoMaximo) + `"}`},
		{"e-mail sem forma de e-mail", `{"email":"nem-e-e-mail"}`},
		{"corpo vazio", `{}`},
	}
	for _, caso := range casos {
		resp := postarRedefinicao(t, rotas, caso.corpo)
		if resp.Code != respExistente.Code {
			t.Errorf("%s: status = %d, quero o mesmo %d da conta existente", caso.nome, resp.Code, respExistente.Code)
		}
		if resp.Body.Len() != 0 {
			t.Errorf("%s: corpo = %q, quero vazio como o da conta existente", caso.nome, resp.Body.String())
		}
		// O cabeçalho também: um Content-Type num dos caminhos e não no outro
		// distinguiria os dois de fora sem ninguém ler o corpo. A correlação é a
		// única coisa que pode diferir — ela muda a cada requisição.
		if cabecalhosComparaveis(resp) != cabecalhosComparaveis(respExistente) {
			t.Errorf("%s: cabeçalhos = %s, quero os mesmos %s",
				caso.nome, cabecalhosComparaveis(resp), cabecalhosComparaveis(respExistente))
		}
		if len(resp.Result().Cookies()) != 0 {
			t.Errorf("%s: a solicitação veio com cookie", caso.nome)
		}
	}
	if depois := len(tokensDeRedefinicao(t, rdb)); depois != depoisDaExistente {
		t.Errorf("tokens = %d depois das recusas, eram %d; nenhuma delas podia criar chave", depois, depoisDaExistente)
	}

	// E a conta inexistente não escreve linha nenhuma: o log não pode contar o
	// que a resposta esconde.
	if _, tokenDeNinguem := solicitar(t, rotas, "ninguem-redefine@azamon.test"); tokenDeNinguem != "" {
		t.Errorf("a conta inexistente gerou o token %q no log", tokenDeNinguem)
	}

	// O e-mail é procurado pela forma NORMALIZADA, como no login: sem isso,
	// quem digita com maiúscula ou com um espaço colado nunca receberia token —
	// e a resposta, idêntica por construção, não contaria nada a ninguém.
	if _, tokenComOutraCaixa := solicitar(t, rotas, "  Davi@Exemplo.BR  "); tokenComOutraCaixa == "" {
		t.Error("o e-mail com outra caixa e com bordas não gerou token; a busca tem de ser pela forma normalizada")
	}
}

// redefinicaoTrocaSenhaEDerrubaSessoes é o resto da matriz, na ordem em que os
// estados dependem uns dos outros: a segunda solicitação invalida a primeira, a
// senha inválida não gasta o token, e a redefinição derruba TODAS as Sessões do
// dono — e só as dele.
func redefinicaoTrocaSenhaEDerrubaSessoes(t *testing.T, rotas http.Handler, rdb *redis.Client) {
	const (
		email     = "elisa@exemplo.br"
		senha     = "senha-da-elisa-1"
		senhaNova = "nova-senha-da-elisa-2"

		outroEmail = "fabio@exemplo.br"
		outraSenha = "senha-do-fabio-1"
	)

	// Duas Sessões do mesmo Comprador: o cadastro abre uma, e o login abre
	// outra. Uma Sessão só não provaria "todas" — provaria "a que o cookie
	// nomeia", que é o que a 2.2 já fazia.
	primeira := cookieDe(t, postarCadastro(t, rotas, `{"nome":"Elisa Faro","email":"`+email+`","senha":"`+senha+`"}`), http.StatusCreated)
	segunda := cookieDe(t, postar(t, rotas, `{"email":"`+email+`","senha":"`+senha+`"}`), http.StatusOK)
	// E a Sessão de outro Comprador, que tem de sobreviver intacta.
	doOutro := cookieDe(t, postarCadastro(t, rotas, `{"nome":"Fábio Reis","email":"`+outroEmail+`","senha":"`+outraSenha+`"}`), http.StatusCreated)

	_, primeiroToken := solicitar(t, rotas, email)
	if primeiroToken == "" {
		t.Fatal("a primeira solicitação não gerou token")
	}

	// A segunda solicitação: o token anterior deixa de resolver, e o Comprador
	// fica com um token só. Sem isto, cada "esqueci a senha" deixaria mais uma
	// chave válida atrás de si.
	_, segundoToken := solicitar(t, rotas, email)
	if segundoToken == "" || segundoToken == primeiroToken {
		t.Fatalf("segundo token = %q; quero um token novo, diferente de %q", segundoToken, primeiroToken)
	}
	if chaves := chavesCom(t, rdb, primeiroToken); len(chaves) != 0 {
		t.Errorf("a chave %v do primeiro token sobreviveu à segunda solicitação", chaves)
	}
	if resp := putRedefinicao(t, rotas, primeiroToken, `{"senha":"`+senhaNova+`"}`); resp.Code != http.StatusNotFound {
		t.Errorf("o primeiro token ainda redefine: status = %d, quero 404", resp.Code)
	}

	// Senha fora dos limites: 400 nomeando o campo, e o token CONTINUA válido —
	// recusar a forma não pode gastar o link que chegou por e-mail.
	for _, caso := range []struct{ nome, corpo, contem string }{
		{"curta demais", `{"senha":"1234567"}`, "8"},
		{"vazia", `{"senha":""}`, "8"},
		{"acima do teto", `{"senha":"` + strings.Repeat("a", senhaMaxDeTeste+1) + `"}`, "128"},
	} {
		resp := putRedefinicao(t, rotas, segundoToken, caso.corpo)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d (%s), quero 400", caso.nome, resp.Code, resp.Body.String())
		}
		envelope := decodificar(t, resp)
		if envelope["codigo"] != "CAMPO_INVALIDO" {
			t.Errorf("%s: codigo = %v, quero CAMPO_INVALIDO", caso.nome, envelope["codigo"])
		}
		dados, _ := envelope["dados"].(map[string]any)
		if dados["campo"] != "senha" {
			t.Errorf("%s: dados.campo = %v, quero \"senha\"", caso.nome, dados["campo"])
		}
		if mensagem := fmt.Sprint(envelope["mensagem"]); !strings.Contains(mensagem, caso.contem) {
			t.Errorf("%s: mensagem = %q; o limiar %q tem de ser nomeado", caso.nome, mensagem, caso.contem)
		}
	}
	if chaves := chavesCom(t, rdb, segundoToken); len(chaves) != 1 {
		t.Fatalf("chaves do token = %v depois das recusas; a recusa por forma não pode gastar o token", chaves)
	}
	// Corpo malformado continua em ENTRADA_INVALIDA e também não gasta nada.
	if resp := putRedefinicao(t, rotas, segundoToken, `{isso não é json`); resp.Code != http.StatusBadRequest {
		t.Errorf("corpo malformado = %d, quero 400", resp.Code)
	}

	// O ponteiro reverso do token, guardado antes de o token ser gasto: o nome
	// dele não carrega o token, e sem guardá-lo aqui não há como cobrar depois
	// que ele sumiu. Um ponteiro órfão seguraria o Comprador sem token válido
	// até o prazo passar — a solicitação seguinte apagaria um token inexistente.
	ponteiro := ponteiroDoToken(t, rdb, segundoToken)

	// E o enxame de Sessões: o SCAN varre por páginas, e com três chaves ele
	// termina na primeira. Com estas, não — uma varredura que lesse só a
	// primeira página deixaria viva a maioria das Sessões do dono.
	enxame := semearSessoes(t, rdb, primeira, 300)

	// O caminho feliz.
	resp := putRedefinicao(t, rotas, segundoToken, `{"senha":"`+senhaNova+`"}`)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d (%s), quero 204", resp.Code, resp.Body.String())
	}
	if resp.Body.Len() != 0 {
		t.Errorf("204 com corpo: %q", resp.Body.String())
	}
	// Redefinir NÃO abre Sessão: a condição de aceite manda encerrar todas, e o
	// Comprador volta pelo Login.
	if len(resp.Result().Cookies()) != 0 {
		t.Errorf("a redefinição emitiu cookie: %v", resp.Result().Cookies())
	}
	if cache := resp.Header().Get("Cache-Control"); cache != "no-store" {
		t.Errorf("Cache-Control = %q, quero no-store", cache)
	}

	// O token é de uso único, e o ponteiro some junto com ele.
	if chaves := chavesCom(t, rdb, segundoToken); len(chaves) != 0 {
		t.Errorf("a chave %v sobreviveu ao uso do token", chaves)
	}
	if existe, err := rdb.Exists(context.Background(), ponteiro).Result(); err != nil || existe != 0 {
		t.Errorf("o ponteiro %q sobreviveu ao uso do token (%v, %v)", ponteiro, existe, err)
	}
	if chaves := chavesCom(t, rdb, "enxame-"); len(chaves) != 0 {
		t.Errorf("%d das %d Sessões do enxame sobreviveram; a varredura parou na primeira página",
			len(chaves), enxame)
	}
	if resp := putRedefinicao(t, rotas, segundoToken, `{"senha":"`+senhaNova+`3"}`); resp.Code != http.StatusNotFound {
		t.Errorf("o token usado ainda redefine: status = %d, quero 404", resp.Code)
	}

	// As duas Sessões do dono somem — as CHAVES, e não só os cookies.
	for i, cookie := range []*http.Cookie{primeira, segunda} {
		if chaves := chavesCom(t, rdb, cookie.Value); len(chaves) != 0 {
			t.Errorf("Sessão %d: a chave %v sobreviveu à redefinição", i+1, chaves)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sessao", nil)
		req.AddCookie(cookie)
		leitura := httptest.NewRecorder()
		rotas.ServeHTTP(leitura, req)
		if leitura.Code != http.StatusUnauthorized {
			t.Errorf("Sessão %d depois da redefinição = %d, quero 401", i+1, leitura.Code)
		}
	}
	// E a do outro Comprador fica: a varredura apaga pelo dono, não pelo prefixo.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessao", nil)
	req.AddCookie(doOutro)
	leitura := httptest.NewRecorder()
	rotas.ServeHTTP(leitura, req)
	if leitura.Code != http.StatusOK {
		t.Errorf("a Sessão do outro Comprador caiu junto: status = %d, quero 200", leitura.Code)
	}

	// A senha nova autentica, e a antiga não.
	if resp := postar(t, rotas, `{"email":"`+email+`","senha":"`+senhaNova+`"}`); resp.Code != http.StatusOK {
		t.Errorf("login com a senha nova = %d (%s), quero 200", resp.Code, resp.Body.String())
	}
	if resp := postar(t, rotas, `{"email":"`+email+`","senha":"`+senha+`"}`); resp.Code != http.StatusUnauthorized {
		t.Errorf("login com a senha antiga = %d, quero 401", resp.Code)
	}
}

// tokenInvalidoDa404: expirado, já usado e inexistente são a mesma resposta —
// distingui-los diria a quem chama que um token chegou a valer.
//
// O expirado é um token de verdade, com a chave vencida à mão: com o prazo
// cheio não há como esperar os 30 minutos, e é o mesmo truque que a 2.2 usa
// para provar a renovação do prazo da Sessão.
func tokenInvalidoDa404(t *testing.T, rotas http.Handler, rdb *redis.Client) {
	const email = "gilda@exemplo.br"
	if resp := postarCadastro(t, rotas, `{"nome":"Gilda Nunes","email":"`+email+`","senha":"senha-da-gilda-1"}`); resp.Code != http.StatusCreated {
		t.Fatalf("cadastro = %d (%s), quero 201", resp.Code, resp.Body.String())
	}
	_, expirado := solicitar(t, rotas, email)
	if expirado == "" {
		t.Fatal("a solicitação não gerou token")
	}
	chaves := chavesCom(t, rdb, expirado)
	if len(chaves) != 1 {
		t.Fatalf("chaves para o token = %v; quero exatamente uma", chaves)
	}
	// Vencer é a chave sumir: é o que o TTL nativo faria sozinho em 30 minutos.
	if err := rdb.Del(context.Background(), chaves[0]).Err(); err != nil {
		t.Fatalf("vencer o token: %v", err)
	}

	for _, token := range []string{
		expirado,                // emitido de verdade, e vencido
		strings.Repeat("0", 64), // forma certa, nunca emitido
		"curto-demais",
		strings.Repeat("z", 64), // 64 caracteres que não são hexadecimais
	} {
		resp := putRedefinicao(t, rotas, token, `{"senha":"senha-qualquer-1"}`)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("token %.12s: status = %d (%s), quero 404", token, resp.Code, resp.Body.String())
		}
		envelope := decodificar(t, resp)
		if envelope["codigo"] != "TOKEN_INVALIDO" {
			t.Errorf("token %.12s: codigo = %v, quero TOKEN_INVALIDO", token, envelope["codigo"])
		}
		if len(resp.Result().Cookies()) != 0 {
			t.Errorf("token %.12s: o 404 veio com cookie", token)
		}
	}
}

// ponteiroDoToken acha a chave do ponteiro reverso daquele token. O nome do
// ponteiro é pelo identificador do Comprador, e não pelo token, e por isso ele
// é procurado pelo VALOR — que é justamente o motivo de o teste precisar
// guardá-lo antes: depois de o token ser gasto, não há mais por onde achá-lo.
func ponteiroDoToken(t *testing.T, rdb *redis.Client, token string) string {
	t.Helper()
	ctx := context.Background()
	for _, chave := range chavesCom(t, rdb, "redefinicao-de:") {
		if valor, err := rdb.Get(ctx, chave).Result(); err == nil && valor == token {
			return chave
		}
	}
	t.Fatalf("nenhum ponteiro reverso aponta para o token %.12s…", token)
	return ""
}

// semearSessoes copia a Sessão do cookie em `quantas` chaves novas do mesmo
// espaço de nomes. O valor é o mesmo, então todas pertencem ao mesmo
// Comprador — o que o SCAN de EncerrarSessoesDoComprador tem de varrer é uma
// página só ou várias, e com três chaves ele nunca passa da primeira.
//
// O prefixo é derivado da chave existente, e não escrito aqui: o espaço de
// nomes mora em identidade, e o teste não o repete.
func semearSessoes(t *testing.T, rdb *redis.Client, cookie *http.Cookie, quantas int) int {
	t.Helper()
	ctx := context.Background()
	chaves := chavesCom(t, rdb, cookie.Value)
	if len(chaves) != 1 {
		t.Fatalf("chaves para o cookie = %v; quero exatamente uma para copiar", chaves)
	}
	valor, err := rdb.Get(ctx, chaves[0]).Result()
	if err != nil {
		t.Fatalf("ler a Sessão a copiar: %v", err)
	}
	prefixo := chaves[0][:strings.LastIndex(chaves[0], ":")+1]
	for i := range quantas {
		if err := rdb.Set(ctx, fmt.Sprintf("%senxame-%d", prefixo, i), valor, expiracaoDeTeste).Err(); err != nil {
			t.Fatalf("semear a Sessão %d: %v", i, err)
		}
	}
	return quantas
}

// redefinirTiraDoBloqueio é o caminho mais comum até a recuperação de senha:
// quem esqueceu a senha errou o login antes. Sem isto a estória entrega uma
// senha nova que só funciona quinze minutos depois, e a recuperação não
// recupera nada.
//
// O par de e-mails é escolhido a dedo: `ana*cruz@` é endereço válido pela RFC
// 5322 (o `*` é atext) e, sem escapar, o padrão de MATCH `falhas:ana*cruz@…|*`
// varreria também o contador de `anaxcruz@`. É o que separa "limpar o contador
// deste Comprador" de "limpar o contador dos vizinhos".
func redefinirTiraDoBloqueio(t *testing.T, rotas http.Handler) {
	const (
		glob      = "ana*cruz@exemplo.br"
		vizinho   = "anaxcruz@exemplo.br"
		senha     = "senha-da-ana-1"
		senhaNova = "nova-senha-da-ana-2"
	)

	for _, email := range []string{glob, vizinho} {
		if resp := postarCadastro(t, rotas, `{"nome":"Ana Cruz","email":`+string(marcarJSON(t, email))+`,"senha":"`+senha+`"}`); resp.Code != http.StatusCreated {
			t.Fatalf("%s: cadastro = %d (%s), quero 201", email, resp.Code, resp.Body.String())
		}
		// Até o limiar, e mais uma para confirmar que o par está bloqueado.
		for i := 1; i <= tentativasDeTeste; i++ {
			if resp := postar(t, rotas, `{"email":`+string(marcarJSON(t, email))+`,"senha":"nao-e-a-senha"}`); resp.Code != http.StatusUnauthorized {
				t.Fatalf("%s: tentativa %d = %d, quero 401 até o limiar", email, i, resp.Code)
			}
		}
		if resp := postar(t, rotas, `{"email":`+string(marcarJSON(t, email))+`,"senha":"`+senha+`"}`); resp.Code != http.StatusTooManyRequests {
			t.Fatalf("%s: depois de %d falhas o status = %d, quero 429", email, tentativasDeTeste, resp.Code)
		}
	}

	_, token := solicitar(t, rotas, glob)
	if token == "" {
		t.Fatal("a solicitação não gerou token")
	}
	if resp := putRedefinicao(t, rotas, token, `{"senha":"`+senhaNova+`"}`); resp.Code != http.StatusNoContent {
		t.Fatalf("redefinir = %d (%s), quero 204", resp.Code, resp.Body.String())
	}

	// A condição de aceite: a senha nova entra na hora, sem esperar o prazo.
	if resp := postar(t, rotas, `{"email":`+string(marcarJSON(t, glob))+`,"senha":"`+senhaNova+`"}`); resp.Code != http.StatusOK {
		t.Errorf("login logo depois de redefinir = %d (%s), quero 200 — redefinir tem de tirar do bloqueio",
			resp.Code, resp.Body.String())
	}
	// E o vizinho continua bloqueado: o padrão de varredura é texto, não glob.
	if resp := postar(t, rotas, `{"email":"`+vizinho+`","senha":"`+senha+`"}`); resp.Code != http.StatusTooManyRequests {
		t.Errorf("o vizinho %s saiu do bloqueio junto: status = %d, quero 429 — o e-mail entra escapado no MATCH",
			vizinho, resp.Code)
	}
}

// cookieDe extrai o cookie de Sessão da resposta, conferindo o status antes —
// os três casos deste arquivo precisam de uma Sessão de verdade, e um cookie
// ausente aqui viraria um nil difícil de ler vinte linhas adiante.
func cookieDe(t *testing.T, resp *httptest.ResponseRecorder, querido int) *http.Cookie {
	t.Helper()
	if resp.Code != querido {
		t.Fatalf("status = %d (%s), quero %d", resp.Code, resp.Body.String(), querido)
	}
	cookies := resp.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("quero exatamente um cookie, vieram %d", len(cookies))
	}
	return cookies[0]
}
