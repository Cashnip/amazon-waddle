package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// Os subtestes da 2.4. Entram como subteste de TestSessaoEProduto, no
// ambiente(t) que já sobe Postgres e Redis uma vez para o pacote inteiro.
//
// É aqui que o NFR-6 ganha prova automatizada: até agora a posse do Pedido era
// precedente da 1.8, e nada obrigava o próximo recurso a repetir a forma. A 2.5
// estende `negacaoPorDono` para Endereço, e as Épicas 4 e 6 para Carrinho e
// Pedido, em vez de reescrever a igualdade.

// uuidNuncaUsado é um uuid v7 bem-formado que não nomeia linha nenhuma: é o
// lado "inexistente" de toda igualdade desta prova.
const uuidNuncaUsado = "00000000-0000-7000-8000-000000000000"

// negacaoPorDono é o AD-11 provado por IGUALDADE, e não por status: o recurso de
// outro dono e o uuid que nunca existiu produzem a mesma resposta, byte a byte,
// menos a correlação. Conferir só o 404 deixaria passar um envelope com
// mensagem própria — e uma mensagem própria já conta que aquele recurso existe.
//
// O segundo Comprador sai do cadastro, e não da semente: a semente tem uma
// conta só, e o que se prova aqui é a Sessão do outro dono batendo na rota.
//
// A 2.5 estende a prova para o segundo recurso com dono — o Endereço, por PUT e
// por DELETE — sem reescrever a forma: o que mudou foi a tabela de recursos, e
// a Épica 4 acrescenta Carrinho na mesma linha.
func negacaoPorDono(t *testing.T, rotas http.Handler, cookieDono *http.Cookie, pedidoID string) {
	if cookieDono == nil || pedidoID == "" {
		t.Fatal("sem Pedido ou sem Sessão: a pré-condição sai dos subtestes acima, e pular esconderia a prova da 2.4 do relatório")
	}

	// O dono lê o próprio Pedido: sem esta linha, uma regressão que negasse
	// tudo passaria pelas outras duas.
	if resp := pegarPedido(t, rotas, pedidoID, cookieDono); resp.Code != http.StatusOK {
		t.Fatalf("o dono = %d (%s), quero 200", resp.Code, resp.Body.String())
	}
	// E o Endereço do dono, criado aqui: sem uma linha de A não há "alheio"
	// para B tentar, e o subteste passaria provando só o inexistente.
	enderecoDoDono := idDe(t, postarEndereco(t, rotas, corpoEnderecoValido, cookieDono), http.StatusCreated)

	outro := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Helena Prado","email":"helena@exemplo.br","senha":"senha-da-helena-1"}`), http.StatusCreated)

	for _, recurso := range []struct {
		nome string
		id   string
		bate func(*testing.T, http.Handler, string, *http.Cookie) *httptest.ResponseRecorder
	}{
		{"Pedido por GET", pedidoID, pegarPedido},
		{"Endereço por PUT", enderecoDoDono, putEnderecoValido},
		// O DELETE por último: se ele apagasse, o PUT acima já teria rodado.
		{"Endereço por DELETE", enderecoDoDono, deletarEndereco},
	} {
		alheio := recurso.bate(t, rotas, recurso.id, outro)
		inexistente := recurso.bate(t, rotas, uuidNuncaUsado, outro)

		if alheio.Code != http.StatusNotFound {
			t.Fatalf("%s alheio = %d (%s), quero 404", recurso.nome, alheio.Code, alheio.Body.String())
		}
		if codigo := decodificar(t, alheio)["codigo"]; codigo != "NAO_ENCONTRADO" {
			t.Errorf("%s alheio: codigo = %v, quero NAO_ENCONTRADO", recurso.nome, codigo)
		}
		if corpoSemCorrelacao(alheio) != corpoSemCorrelacao(inexistente) {
			t.Errorf("o %s alheio se distingue do inexistente:\n%s\n%s",
				recurso.nome, corpoSemCorrelacao(alheio), corpoSemCorrelacao(inexistente))
		}
		if cabecalhosComparaveis(alheio) != cabecalhosComparaveis(inexistente) {
			t.Errorf("%s: os cabeçalhos distinguem os dois:\n%s\n%s",
				recurso.nome, cabecalhosComparaveis(alheio), cabecalhosComparaveis(inexistente))
		}
	}

	// O 404 não pode ser só a resposta: o Endereço de A continua lá, inteiro,
	// depois do PUT e do DELETE de B. Um handler que negasse a resposta e
	// escrevesse assim mesmo passaria por toda a igualdade acima.
	lista := pegarEnderecos(t, rotas, cookieDono)
	if lista.Code != http.StatusOK {
		t.Fatalf("listar os Endereços do dono = %d (%s), quero 200", lista.Code, lista.Body.String())
	}
	if !strings.Contains(lista.Body.String(), enderecoDoDono) {
		t.Errorf("o Endereço %s do dono não sobreviveu ao PUT e ao DELETE de outro Comprador: %s",
			enderecoDoDono, lista.Body.String())
	}
}

// loginPorTabelaPropria é a separação estrutural: cada login consulta SÓ a sua
// tabela. Não há ordem de consulta para o mesmo e-mail disputar — quem decide o
// papel é a rota que foi chamada.
func loginPorTabelaPropria(t *testing.T, rotas http.Handler) {
	admin := postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`)
	if admin.Code != http.StatusOK {
		t.Fatalf("login de Administrador = %d (%s), quero 200", admin.Code, admin.Body.String())
	}
	if nome := decodificar(t, admin)["nome"]; nome != nomeAdmin {
		t.Errorf("nome = %v, quero %q", nome, nomeAdmin)
	}
	if len(admin.Result().Cookies()) != 1 {
		t.Fatalf("quero exatamente um cookie, vieram %d", len(admin.Result().Cookies()))
	}

	// As duas travessias cruzadas. A resposta é a de credencial inválida, e não
	// uma mensagem própria: contar "esta conta é do outro lado" enumeraria
	// Administradores a partir da loja.
	for _, caso := range []struct {
		nome  string
		posta func(*testing.T, http.Handler, string) *httptest.ResponseRecorder
		corpo string
	}{
		{"Administrador na loja", postar, `{"email":"` + emailAdmin + `","senha":"` + senhaAdmin + `"}`},
		{"Comprador no admin", postarAdmin, `{"email":"` + emailSemente + `","senha":"` + senhaSemente + `"}`},
	} {
		resp := caso.posta(t, rotas, caso.corpo)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d (%s), quero 401", caso.nome, resp.Code, resp.Body.String())
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != "CREDENCIAL_INVALIDA" {
			t.Errorf("%s: codigo = %v, quero CREDENCIAL_INVALIDA", caso.nome, codigo)
		}
		if len(resp.Result().Cookies()) != 0 {
			t.Errorf("%s: o 401 veio com cookie", caso.nome)
		}
	}
}

// guardaDoPrefixoAdministrativo cobre os dois lados do papel: a Sessão de
// Administrador não vale na loja (401), e a de Comprador não vale no admin
// (404). E prova o mecanismo, não só a rota que existe hoje — uma rota nova,
// com um handler sem checagem nenhuma, já responde 404 ao Comprador.
func guardaDoPrefixoAdministrativo(t *testing.T, rotas http.Handler, rdb *redis.Client, cookieComprador *http.Cookie) {
	if cookieComprador == nil {
		t.Fatal("sem Sessão de Comprador: a pré-condição sai dos subtestes acima, e pular esconderia a prova da 2.4 do relatório")
	}
	cookieAdmin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)

	if resp := pegarCom(t, rotas, "/api/v1/admin/sessao", cookieAdmin); resp.Code != http.StatusOK {
		t.Fatalf("o Administrador na própria área = %d (%s), quero 200", resp.Code, resp.Body.String())
	} else if nome := decodificar(t, resp)["nome"]; nome != nomeAdmin {
		t.Errorf("nome = %v, quero %q", nome, nomeAdmin)
	}

	// 404, e não 403: a UX-DR9 pede que a área não apareça para quem não é
	// Administrador, e a igualdade com a rota inexistente é o que "não aparece"
	// quer dizer do lado do servidor. Um 403 contaria que a rota existe.
	inexistente := pegarCom(t, rotas, "/api/v1/nao-existe", cookieComprador)
	for _, caso := range []struct {
		nome   string
		rota   string
		cookie *http.Cookie
	}{
		{"Comprador no admin", "/api/v1/admin/sessao", cookieComprador},
		{"sem Sessão no admin", "/api/v1/admin/sessao", nil},
		{"Comprador em rota administrativa qualquer", "/api/v1/admin/qualquer-coisa", cookieComprador},
		// Sem a barra final: o ServeMux redirecionaria com 307 e corpo HTML
		// antes da guarda, e o redirecionamento sozinho já confirmaria a
		// subárvore a quem nunca entrou.
		{"prefixo sem a barra final", "/api/v1/admin", cookieComprador},
		{"prefixo sem a barra final, sem Sessão", "/api/v1/admin", nil},
	} {
		resp := pegarCom(t, rotas, caso.rota, caso.cookie)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d (%s), quero 404", caso.nome, resp.Code, resp.Body.String())
		}
		if corpoSemCorrelacao(resp) != corpoSemCorrelacao(inexistente) {
			t.Errorf("%s: a resposta se distingue da rota inexistente:\n%s\n%s",
				caso.nome, corpoSemCorrelacao(resp), corpoSemCorrelacao(inexistente))
		}
	}

	// O outro lado do papel: a Sessão de Administrador não abre nada da loja.
	for _, rota := range []string{"/api/v1/sessao", "/api/v1/pedidos/" + "00000000-0000-7000-8000-000000000000"} {
		resp := pegarCom(t, rotas, rota, cookieAdmin)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("%s com Sessão de Administrador = %d (%s), quero 401", rota, resp.Code, resp.Body.String())
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != "SESSAO_INVALIDA" {
			t.Errorf("%s: codigo = %v, quero SESSAO_INVALIDA", rota, codigo)
		}
	}

	// A condição de aceite do mecanismo: uma rota administrativa NOVA, cujo
	// handler não confere nada, já está protegida. Se a guarda vivesse em cada
	// handler, esta seria exatamente a rota que alguém esqueceria.
	//
	// Só o prazo da Sessão importa para a guarda, e por isso a Config aqui é
	// mínima — montar um `servidor` é o que permite pendurar um handler que
	// ainda não existe em rota nenhuma.
	s := &servidor{cfg: plataforma.Config{SessaoExpiracao: expiracaoDeTeste}, rdb: rdb}
	var alcancado bool
	nova := s.somenteAdministrador(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		alcancado = true
		escreverJSON(w, http.StatusOK, saidaSessao{Nome: "rota nova"})
	}))
	for _, caso := range []struct {
		nome   string
		cookie *http.Cookie
		quero  int
		passa  bool
	}{
		{"Comprador", cookieComprador, http.StatusNotFound, false},
		{"Administrador", cookieAdmin, http.StatusOK, true},
	} {
		alcancado = false
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/rota-nova", nil)
		req.AddCookie(caso.cookie)
		resp := httptest.NewRecorder()
		nova.ServeHTTP(resp, req)
		if resp.Code != caso.quero {
			t.Errorf("rota nova, %s: status = %d (%s), quero %d", caso.nome, resp.Code, resp.Body.String(), caso.quero)
		}
		if alcancado != caso.passa {
			t.Errorf("rota nova, %s: handler alcançado = %v, quero %v", caso.nome, alcancado, caso.passa)
		}
	}

	// O papel é carga em mais dois lugares, e os dois entram aqui porque a
	// Sessão de Administrador já está aberta.
	t.Run("Sessão sem papel não vale de lado nenhum", func(t *testing.T) {
		sessaoSemPapelNaoVale(t, rotas, rdb, cookieComprador)
	})
	t.Run("a Sessão de Administrador sobrevive à redefinição de um Comprador", func(t *testing.T) {
		sessaoDeAdminSobreviveARedefinicao(t, rotas, cookieAdmin)
	})
}

// corpoSemContentTypeDa400 fecha a fixação de Sessão por formulário cross-site:
// um <form> só emite text/plain, x-www-form-urlencoded ou multipart/form-data, e
// nenhum dos três decodifica mais. Os chamadores legítimos não mudam — o web/ e
// o Provedor Simulado já mandam o cabeçalho.
//
// A varredura é sobre as TRÊS rotas que emitem cookie por `abrirSessao`, e não
// só sobre o login: cobrir uma delas deixaria reverter as outras duas ao
// decodificador antigo com a suíte verde, e a fixação de Sessão voltaria por
// onde ninguém está olhando.
func corpoSemContentTypeDa400(t *testing.T, rotas http.Handler) {
	login := `{"email":"` + emailSemente + `","senha":"` + senhaSemente + `"}`
	// O corpo do cadastro é de uma conta que NÃO existe: se a guarda cair, a
	// rota responde 201 com cookie, e é isso que o subteste tem de ver.
	cadastro := `{"nome":"Jonas Vidal","email":"jonas@exemplo.br","senha":"senha-do-jonas-1"}`
	admin := `{"email":"` + emailAdmin + `","senha":"` + senhaAdmin + `"}`

	for _, rota := range []struct{ caminho, corpo string }{
		{"/api/v1/sessoes", login},
		{"/api/v1/compradores", cadastro},
		{"/api/v1/admin/sessoes", admin},
	} {
		for _, tipo := range []string{
			"", // nenhum cabeçalho
			"text/plain",
			"text/plain;charset=UTF-8",
			"application/x-www-form-urlencoded",
			"multipart/form-data; boundary=x",
			"application/jsonp",
		} {
			resp := postarComTipo(t, rotas, rota.caminho, rota.corpo, tipo)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("%s com Content-Type %q: status = %d (%s), quero 400",
					rota.caminho, tipo, resp.Code, resp.Body.String())
			}
			if codigo := decodificar(t, resp)["codigo"]; codigo != "ENTRADA_INVALIDA" {
				t.Errorf("%s com Content-Type %q: codigo = %v, quero ENTRADA_INVALIDA", rota.caminho, tipo, codigo)
			}
			// O que a guarda existe para impedir: nenhuma Sessão nasce daí.
			if len(resp.Result().Cookies()) != 0 {
				t.Errorf("%s com Content-Type %q: o corpo cross-site fixou Sessão", rota.caminho, tipo)
			}
		}
	}

	// E o parâmetro do tipo não pode recusar ninguém: `fetch` com um corpo de
	// texto manda `application/json;charset=utf-8`, e é o mesmo tipo.
	if resp := postarComTipo(t, rotas, "/api/v1/sessoes", login, "application/json; charset=utf-8"); resp.Code != http.StatusOK {
		t.Errorf("application/json com charset = %d (%s), quero 200", resp.Code, resp.Body.String())
	}
}

// sessaoSemPapelNaoVale é o formato anterior a esta estória: a chave do Redis
// gravada sem o campo `papel`. Ela não pode valer de lado nenhum — afrouxar a
// checagem para "papel vazio passa" entregaria o prefixo administrativo a todo
// navegador que estivesse dentro antes da subida.
//
// A chave é gravada crua, com o prefixo derivado de uma chave existente: o
// espaço de nomes mora em identidade, e o teste não o repete (o mesmo truque de
// semearSessoes).
func sessaoSemPapelNaoVale(t *testing.T, rotas http.Handler, rdb *redis.Client, cookieVivo *http.Cookie) {
	ctx := context.Background()
	chaves := chavesCom(t, rdb, cookieVivo.Value)
	if len(chaves) != 1 {
		t.Fatalf("chaves para o cookie = %v; quero exatamente uma para derivar o prefixo", chaves)
	}
	prefixo := chaves[0][:strings.LastIndex(chaves[0], ":")+1]

	antigo := strings.Repeat("b", 64)
	// O valor é o da 2.3: id e nome, e nenhum papel.
	if err := rdb.Set(ctx, prefixo+antigo,
		`{"id":"fce3404b-df7d-59cd-84f9-553ad0f4ae0d","nome":"`+nomeSemente+`"}`,
		expiracaoDeTeste).Err(); err != nil {
		t.Fatalf("gravar a Sessão sem papel: %v", err)
	}
	cookie := &http.Cookie{Name: "azamon_sessao", Value: antigo}

	loja := pegarCom(t, rotas, "/api/v1/sessao", cookie)
	if loja.Code != http.StatusUnauthorized {
		t.Errorf("Sessão sem papel na loja = %d (%s), quero 401", loja.Code, loja.Body.String())
	}
	if codigo := decodificar(t, loja)["codigo"]; codigo != "SESSAO_INVALIDA" {
		t.Errorf("Sessão sem papel na loja: codigo = %v, quero SESSAO_INVALIDA", codigo)
	}
	if adm := pegarCom(t, rotas, "/api/v1/admin/sessao", cookie); adm.Code != http.StatusNotFound {
		t.Errorf("Sessão sem papel no admin = %d (%s), quero 404", adm.Code, adm.Body.String())
	}
}

// sessaoDeAdminSobreviveARedefinicao: a varredura de EncerrarSessoesDoComprador
// apaga pelo DONO, e o dono é um Comprador — a Sessão de Administrador não entra
// na conta. Sem a checagem de papel na varredura, uma redefinição de senha
// poderia derrubar quem não é dono de nada daquilo.
func sessaoDeAdminSobreviveARedefinicao(t *testing.T, rotas http.Handler, cookieAdmin *http.Cookie) {
	const email = "isaura@exemplo.br"
	if resp := postarCadastro(t, rotas, `{"nome":"Isaura Melo","email":"`+email+`","senha":"senha-da-isaura-1"}`); resp.Code != http.StatusCreated {
		t.Fatalf("cadastro = %d (%s), quero 201", resp.Code, resp.Body.String())
	}
	_, token := solicitar(t, rotas, email)
	if token == "" {
		t.Fatal("a solicitação não gerou token")
	}
	if resp := putRedefinicao(t, rotas, token, `{"senha":"nova-senha-da-isaura-2"}`); resp.Code != http.StatusNoContent {
		t.Fatalf("redefinir = %d (%s), quero 204", resp.Code, resp.Body.String())
	}

	if resp := pegarCom(t, rotas, "/api/v1/admin/sessao", cookieAdmin); resp.Code != http.StatusOK {
		t.Errorf("a Sessão de Administrador caiu com a redefinição de um Comprador: status = %d (%s), quero 200",
			resp.Code, resp.Body.String())
	}
}

// postarAdmin é o login da área administrativa, com a mesma assinatura de
// postar — é o que permite as duas travessias cruzadas numa tabela só.
func postarAdmin(t *testing.T, rotas http.Handler, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	return postarComTipo(t, rotas, "/api/v1/admin/sessoes", corpo, "application/json")
}

// postarComTipo é o POST com o Content-Type escolhido: sem mexer no cabeçalho
// não há como provar a guarda que fecha a fixação de Sessão.
func postarComTipo(t *testing.T, rotas http.Handler, rota, corpo, tipo string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, rota, strings.NewReader(corpo))
	req.RemoteAddr = origemDeTeste
	if tipo != "" {
		req.Header.Set("Content-Type", tipo)
	}
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
}

// pegarCom é o GET com cookie opcional — o `nil` é a linha "sem Sessão" da matriz.
func pegarCom(t *testing.T, rotas http.Handler, rota string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, rota, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
}

// corpoSemCorrelacao é o corpo cru com o identificador de correlação apagado —
// a única coisa que pode diferir entre duas respostas que precisam ser
// indistinguíveis. Comparar o corpo cru, e não um mapa decodificado, é o que
// torna a igualdade literal: ordem de campo e espaço em branco contam.
func corpoSemCorrelacao(resp *httptest.ResponseRecorder) string {
	correlacao := resp.Header().Get("X-Correlation-Id")
	if correlacao == "" {
		return resp.Body.String()
	}
	return strings.ReplaceAll(resp.Body.String(), correlacao, "")
}
