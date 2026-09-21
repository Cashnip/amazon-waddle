package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Os subtestes do Endereço (2.5). Entram como subteste de TestSessaoEProduto,
// no ambiente(t) que já sobe Postgres e Redis uma vez para o pacote inteiro.
//
// As duas linhas "alheio" da matriz (PUT e DELETE sobre Endereço de outro
// Comprador) NÃO são repetidas aqui: elas são provadas por igualdade byte a
// byte em negacaoPorDono, que é onde o AD-11 mora. Uma segunda prova por status
// só diria menos.

// corpoEnderecoValido é a etiqueta completa, com o CEP na forma mascarada que a
// tela manda: o que entra com hífen tem de sair do banco com oito dígitos.
const corpoEnderecoValido = `{"destinatario":"Joana Ribeiro","cep":"01310-100",` +
	`"logradouro":"Avenida Paulista","numero":"1578","complemento":"Apto 12",` +
	`"bairro":"Bela Vista","cidade":"São Paulo","uf":"SP"}`

// Os dois limiares do Endereço, nos mesmos valores dos padrões de
// internal/plataforma/config.go. Ficam escritos aqui pelo motivo dos outros: o
// que os subtestes conferem é que o limiar configurado chega inteiro à
// mensagem, e um Config zerado recusaria todo campo de texto.
const (
	enderecoTextoMaxDeTeste        = 120
	enderecoPorCompradorMaxDeTeste = 20
)

// enderecosDoComprador é a matriz da 2.5 num subteste só: a ordem importa entre
// as linhas (o vazio precisa vir antes do primeiro cadastro, e o teto por
// Comprador enche a conta). A conta é própria — o teto só é alcançável numa
// conta que nenhum outro subteste usa.
func enderecosDoComprador(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	cookie := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Nara Bastos","email":"nara@exemplo.br","senha":"senha-da-nara-1"}`), http.StatusCreated)

	// Listar sem nenhum: 200 com lista vazia, e `[]` e não `null` — o estado
	// vazio é da tela, e um `null` a obrigaria a defender-se do formato.
	lista := pegarEnderecos(t, rotas, cookie)
	if lista.Code != http.StatusOK {
		t.Fatalf("listar sem nenhum = %d (%s), quero 200", lista.Code, lista.Body.String())
	}
	// A lista é autenticada e por dono: guardada em cache, um intermediário
	// entregaria os Endereços de um Comprador ao seguinte.
	if v := lista.Header().Get("Cache-Control"); v != "no-store" {
		t.Errorf("Cache-Control da listagem = %q, quero no-store", v)
	}
	if corpo := strings.TrimSpace(lista.Body.String()); corpo != "[]" {
		t.Errorf("lista vazia = %s, quero []", corpo)
	}

	// Cadastrar válido: 201 com o Endereço criado, e o CEP já sem a máscara.
	criado := postarEndereco(t, rotas, corpoEnderecoValido, cookie)
	if criado.Code != http.StatusCreated {
		t.Fatalf("cadastro = %d (%s), quero 201", criado.Code, criado.Body.String())
	}
	novo := decodificar(t, criado)
	if novo["cep"] != "01310100" {
		t.Errorf("cep = %v, quero os oito dígitos sem hífen — a máscara é da tela", novo["cep"])
	}
	if novo["logradouro"] != "Avenida Paulista" || novo["uf"] != "SP" || novo["complemento"] != "Apto 12" {
		t.Errorf("o Endereço criado não veio inteiro: %v", novo)
	}
	enderecoID := idDe(t, criado, http.StatusCreated)

	// Os outros três cadastros válidos. Cada um leva o seu `numero`: é por ele
	// que a ordem estável da listagem vira asserção logo abaixo.
	for _, caso := range []struct{ nome, corpo, campo, quero string }{
		// A UF entra em minúsculas e sai maiúscula: a coluna tem
		// `CHECK (uf ~ '^[A-Z]{2}$')`, e sem a normalização o INSERT falharia
		// em 500.
		{"UF em minúsculas", variante(t, `"uf":"SP"`, `"uf":"rj"`, `"numero":"1578"`, `"numero":"2"`), "uf", "RJ"},
		// O CEP sem máscara é a forma guardada no banco, e formatoCEP a aceita:
		// quem chama a API direto não é obrigado a pôr o hífen que a tela põe.
		{"CEP sem máscara", variante(t, `"cep":"01310-100"`, `"cep":"20040002"`, `"numero":"1578"`, `"numero":"3"`), "cep", "20040002"},
		// O complemento é o único opcional: vazio entra, e volta vazio.
		{"sem complemento", variante(t, `"complemento":"Apto 12"`, `"complemento":""`, `"numero":"1578"`, `"numero":"4"`), "complemento", ""},
	} {
		resp := postarEndereco(t, rotas, caso.corpo, cookie)
		if resp.Code != http.StatusCreated {
			t.Fatalf("%s = %d (%s), quero 201", caso.nome, resp.Code, resp.Body.String())
		}
		if tem := decodificar(t, resp)[caso.campo]; tem != caso.quero {
			t.Errorf("%s: %s = %v, quero %q", caso.nome, caso.campo, tem, caso.quero)
		}
	}

	// A ordem da listagem é a de criação, e é contrato: o checkout da 5.2
	// escolhe sobre esta lista, e uma ordem que muda a cada leitura moveria o
	// item sob o cursor de quem está escolhendo.
	if tem := numerosDe(t, rotas, cookie); !slices.Equal(tem, []string{"1578", "2", "3", "4"}) {
		t.Errorf("ordem da listagem = %v, quero a de criação", tem)
	}

	// Editar o próprio: 200 com o Endereço atualizado, e a lista reflete. Sem a
	// re-listagem, o que estaria provado é o RETURNING, e não a gravação.
	editado := putEndereco(t, rotas, enderecoID, variante(t, `"numero":"1578"`, `"numero":"2000"`), cookie)
	if editado.Code != http.StatusOK {
		t.Fatalf("edição = %d (%s), quero 200", editado.Code, editado.Body.String())
	}
	if numero := decodificar(t, editado)["numero"]; numero != "2000" {
		t.Errorf("numero = %v depois do PUT, quero 2000", numero)
	}
	if tem := numerosDe(t, rotas, cookie); !slices.Equal(tem, []string{"2000", "2", "3", "4"}) {
		t.Errorf("lista depois do PUT = %v; a edição tinha de estar gravada, e a ordem de criação mantida", tem)
	}

	// Identificador malformado é o MESMO 404 do inexistente, e nunca 500: quem
	// o traduz é o uuidDe de identidade, e trocá-lo pelo erro cru do Scan
	// passaria por toda a matriz acima sem nada falhar.
	for _, resp := range []*httptest.ResponseRecorder{
		putEnderecoValido(t, rotas, "nao-e-uuid", cookie),
		deletarEndereco(t, rotas, "nao-e-uuid", cookie),
	} {
		if resp.Code != http.StatusNotFound {
			t.Fatalf("id malformado = %d (%s), quero 404", resp.Code, resp.Body.String())
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != "NAO_ENCONTRADO" {
			t.Errorf("id malformado: codigo = %v, quero NAO_ENCONTRADO", codigo)
		}
	}

	enderecoComCampoInvalido(t, rotas, cookie)
	remocaoNaoTocaPedido(t, rotas, pool, cookie, enderecoID)
	tetoPorComprador(t, rotas)
}

// enderecoComCampoInvalido é a parte da matriz que sai em 400 nomeando o campo.
// Nada é gravado: a contagem do dono é a mesma antes e depois.
func enderecoComCampoInvalido(t *testing.T, rotas http.Handler, cookie *http.Cookie) {
	antes := len(decodificarLista(t, pegarEnderecos(t, rotas, cookie)))

	// Um campo por caso, trocado sobre a etiqueta válida: o que muda é só o
	// campo em prova, e nenhum caso passa por acidente de outro campo vazio.
	casos := []struct {
		nome     string
		de, para string
		campo    string
		contem   string // trecho que a mensagem tem de nomear
	}{
		{"CEP curto demais", `"cep":"01310-100"`, `"cep":"1234"`, "cep", "8"},
		{"CEP com letra", `"cep":"01310-100"`, `"cep":"0131a100"`, "cep", "8"},
		{"CEP vazio", `"cep":"01310-100"`, `"cep":""`, "cep", ""},
		// ZZ passa por qualquer teto de dois caracteres, e só apareceria como
		// etiqueta impossível lá na entrega: a validação é contra as 27 siglas.
		{"UF inexistente", `"uf":"SP"`, `"uf":"ZZ"`, "uf", ""},
		{"UF longa demais", `"uf":"SP"`, `"uf":"SAO"`, "uf", ""},
		{"UF vazia", `"uf":"SP"`, `"uf":""`, "uf", ""},
		{"destinatário vazio", `"destinatario":"Joana Ribeiro"`, `"destinatario":"   "`, "destinatario", ""},
		{"logradouro vazio", `"logradouro":"Avenida Paulista"`, `"logradouro":""`, "logradouro", ""},
		{"número vazio", `"numero":"1578"`, `"numero":""`, "numero", ""},
		{"bairro vazio", `"bairro":"Bela Vista"`, `"bairro":""`, "bairro", ""},
		{"cidade vazia", `"cidade":"São Paulo"`, `"cidade":""`, "cidade", ""},
		{
			// Acentuado de propósito: o teto é em runas, e não em bytes. Estas
			// 121 runas são 242 bytes — contado em bytes, o limite recusaria
			// logradouros acentuados bem antes do teto configurado.
			"logradouro acima do teto",
			`"logradouro":"Avenida Paulista"`,
			`"logradouro":"` + strings.Repeat("á", enderecoTextoMaxDeTeste+1) + `"`,
			"logradouro", fmt.Sprint(enderecoTextoMaxDeTeste),
		},
		{
			// O complemento é opcional, não ilimitado.
			"complemento acima do teto",
			`"complemento":"Apto 12"`,
			`"complemento":"` + strings.Repeat("a", enderecoTextoMaxDeTeste+1) + `"`,
			"complemento", fmt.Sprint(enderecoTextoMaxDeTeste),
		},
	}
	for _, caso := range casos {
		corpo := variante(t, caso.de, caso.para)
		// O POST e o PUT validam pela MESMA função: uma cópia divergiria na
		// primeira mudança, e a edição passaria a aceitar o que o cadastro
		// recusa. Os dois entram em toda linha da matriz por isso — em fatia, e
		// não em mapa: a ordem de iteração de mapa em Go é aleatória, e qual
		// método aparece no t.Fatalf mudaria a cada corrida.
		for _, chamada := range []struct {
			rota string
			resp *httptest.ResponseRecorder
		}{
			{"POST", postarEndereco(t, rotas, corpo, cookie)},
			{"PUT", putEndereco(t, rotas, uuidNuncaUsado, corpo, cookie)},
		} {
			rota, resp := chamada.rota, chamada.resp
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("%s por %s: status = %d (%s), quero 400", caso.nome, rota, resp.Code, resp.Body.String())
			}
			envelope := decodificar(t, resp)
			if envelope["codigo"] != "CAMPO_INVALIDO" {
				t.Errorf("%s por %s: codigo = %v, quero CAMPO_INVALIDO", caso.nome, rota, envelope["codigo"])
			}
			dados, _ := envelope["dados"].(map[string]any)
			if dados["campo"] != caso.campo {
				t.Errorf("%s por %s: dados.campo = %v, quero %q", caso.nome, rota, dados["campo"], caso.campo)
			}
			mensagem := fmt.Sprint(envelope["mensagem"])
			if caso.contem != "" && !strings.Contains(mensagem, caso.contem) {
				t.Errorf("%s por %s: mensagem = %q; o limiar %q tem de ser nomeado", caso.nome, rota, mensagem, caso.contem)
			}
		}
	}

	// Corpo malformado e sem Content-Type continuam em ENTRADA_INVALIDA: é o
	// envelope genérico do AD-14, e não o erro em linha.
	for _, resp := range []*httptest.ResponseRecorder{
		postarEndereco(t, rotas, `{isso não é json`, cookie),
		postarEnderecoComTipo(t, rotas, corpoEnderecoValido, "text/plain", cookie),
	} {
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("corpo recusado = %d (%s), quero 400", resp.Code, resp.Body.String())
		}
		if codigo := decodificar(t, resp)["codigo"]; codigo != "ENTRADA_INVALIDA" {
			t.Errorf("codigo = %v, quero ENTRADA_INVALIDA", codigo)
		}
	}

	if depois := len(decodificarLista(t, pegarEnderecos(t, rotas, cookie))); depois != antes {
		t.Errorf("Endereços = %d depois das recusas, eram %d; nada devia ter sido gravado", depois, antes)
	}
}

// remocaoNaoTocaPedido é a condição de aceite da estória: o Comprador tem
// Pedido, remove o Endereço, e o Pedido continua íntegro. O AD-3 já garante
// isso — o Pedido congela o Endereço na criação —, e o que este subteste
// impede é alguém criar a dependência que tornaria o DELETE impossível.
func remocaoNaoTocaPedido(t *testing.T, rotas http.Handler, pool *pgxpool.Pool, cookie *http.Cookie, enderecoID string) {
	pedidoID := idDe(t, pedidoPeloCheckout(t, rotas, cookie, produtoSemeado, 1), http.StatusCreated)
	antes := corpoSemCorrelacao(pegarPedido(t, rotas, pedidoID, cookie))
	pedidosAntes := contarPedidos(t, pool)

	resp := deletarEndereco(t, rotas, enderecoID, cookie)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("remover o próprio = %d (%s), quero 204", resp.Code, resp.Body.String())
	}
	if resp.Body.Len() != 0 {
		t.Errorf("204 com corpo: %q", resp.Body.String())
	}

	// DELETE de verdade: a linha some da lista, e não fica desativada para a
	// tela filtrar para sempre.
	if lista := pegarEnderecos(t, rotas, cookie); strings.Contains(lista.Body.String(), enderecoID) {
		t.Errorf("o Endereço removido continua na lista: %s", lista.Body.String())
	}
	// E remover de novo é 404, como o inexistente.
	if repetido := deletarEndereco(t, rotas, enderecoID, cookie); repetido.Code != http.StatusNotFound {
		t.Errorf("remover duas vezes = %d, quero 404", repetido.Code)
	}

	if depois := corpoSemCorrelacao(pegarPedido(t, rotas, pedidoID, cookie)); depois != antes {
		t.Errorf("o Pedido mudou com a remoção do Endereço:\n%s\n%s", antes, depois)
	}
	if depois := contarPedidos(t, pool); depois != pedidosAntes {
		t.Errorf("Pedidos = %d depois da remoção, eram %d; nenhum Pedido é tocado (AD-3)", depois, pedidosAntes)
	}
}

// tetoPorComprador enche uma conta até o limiar da Config. É conta própria, e a
// última do arquivo, porque ela sai daqui cheia — e o 409 tem de nomear o
// número configurado, senão o Comprador não sabe quantos pode ter.
func tetoPorComprador(t *testing.T, rotas http.Handler) {
	cookie := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Otávio Lins","email":"otavio@exemplo.br","senha":"senha-do-otavio-1"}`), http.StatusCreated)

	for i := 1; i <= enderecoPorCompradorMaxDeTeste; i++ {
		resp := postarEndereco(t, rotas, corpoEnderecoValido, cookie)
		if resp.Code != http.StatusCreated {
			t.Fatalf("Endereço %d = %d (%s), quero 201 até o teto", i, resp.Code, resp.Body.String())
		}
	}

	resp := postarEndereco(t, rotas, corpoEnderecoValido, cookie)
	if resp.Code != http.StatusConflict {
		t.Fatalf("acima do teto = %d (%s), quero 409", resp.Code, resp.Body.String())
	}
	envelope := decodificar(t, resp)
	if envelope["codigo"] != "LIMITE_DE_ENDERECOS" {
		t.Errorf("codigo = %v, quero LIMITE_DE_ENDERECOS", envelope["codigo"])
	}
	if mensagem := fmt.Sprint(envelope["mensagem"]); !strings.Contains(mensagem, fmt.Sprint(enderecoPorCompradorMaxDeTeste)) {
		t.Errorf("mensagem = %q; o limiar configurado tem de aparecer", mensagem)
	}
	if lista := decodificarLista(t, pegarEnderecos(t, rotas, cookie)); len(lista) != enderecoPorCompradorMaxDeTeste {
		t.Errorf("Endereços = %d depois do 409, quero %d — o recusado não podia ter entrado",
			len(lista), enderecoPorCompradorMaxDeTeste)
	}

	// E o teto não é uma porta fechada: remover um abre a vaga de volta.
	primeiro := fmt.Sprint(decodificarLista(t, pegarEnderecos(t, rotas, cookie))[0]["id"])
	if removido := deletarEndereco(t, rotas, primeiro, cookie); removido.Code != http.StatusNoContent {
		t.Fatalf("abrir vaga = %d (%s), quero 204", removido.Code, removido.Body.String())
	}
	if resp := postarEndereco(t, rotas, corpoEnderecoValido, cookie); resp.Code != http.StatusCreated {
		t.Errorf("depois de remover um = %d (%s), quero 201", resp.Code, resp.Body.String())
	}
}

// enderecoExigeSessaoDeComprador são as duas últimas linhas da matriz: sem
// Sessão e com Sessão de Administrador, as quatro rotas saem em 401. O
// Administrador não tem Endereço — é a loja, e o dono é sempre um Comprador.
func enderecoExigeSessaoDeComprador(t *testing.T, rotas http.Handler) {
	cookieAdmin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)

	for _, quem := range []struct {
		nome   string
		cookie *http.Cookie
	}{
		{"sem Sessão", nil},
		{"Sessão de Administrador", cookieAdmin},
	} {
		for _, rota := range []struct {
			nome string
			resp *httptest.ResponseRecorder
		}{
			{"GET", pegarEnderecos(t, rotas, quem.cookie)},
			{"POST", postarEndereco(t, rotas, corpoEnderecoValido, quem.cookie)},
			{"PUT", putEnderecoValido(t, rotas, uuidNuncaUsado, quem.cookie)},
			{"DELETE", deletarEndereco(t, rotas, uuidNuncaUsado, quem.cookie)},
		} {
			if rota.resp.Code != http.StatusUnauthorized {
				t.Fatalf("%s, %s: status = %d (%s), quero 401",
					quem.nome, rota.nome, rota.resp.Code, rota.resp.Body.String())
			}
			if codigo := decodificar(t, rota.resp)["codigo"]; codigo != "SESSAO_INVALIDA" {
				t.Errorf("%s, %s: codigo = %v, quero SESSAO_INVALIDA", quem.nome, rota.nome, codigo)
			}
		}
	}
}

// variante troca pares "de"→"para" sobre a etiqueta válida: o que muda é só o
// campo em prova, e o resto do corpo continua sendo o mesmo de sempre. Recusar
// a substituição que não casou é o que impede um caso de "passar" porque o
// trecho procurado mudou e ele acabou postando a etiqueta válida inteira.
func variante(t *testing.T, pares ...string) string {
	t.Helper()
	corpo := strings.NewReplacer(pares...).Replace(corpoEnderecoValido)
	if corpo == corpoEnderecoValido {
		t.Fatalf("nenhum dos trechos %v está no corpo válido — o caso não prova nada", pares)
	}
	return corpo
}

// numerosDe é o `numero` de cada Endereço da listagem, na ordem em que a rota
// os devolveu — é assim que a ordem estável do ORDER BY vira asserção.
func numerosDe(t *testing.T, rotas http.Handler, cookie *http.Cookie) []string {
	t.Helper()
	var numeros []string
	for _, e := range decodificarLista(t, pegarEnderecos(t, rotas, cookie)) {
		numeros = append(numeros, fmt.Sprint(e["numero"]))
	}
	return numeros
}

// decodificarLista é o `decodificar` da resposta que é um array — a listagem é
// a única rota do sistema cujo corpo de sucesso não é objeto.
func decodificarLista(t *testing.T, resp *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	if resp.Code != http.StatusOK {
		t.Fatalf("listar = %d (%s), quero 200", resp.Code, resp.Body.String())
	}
	var lista []map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &lista); err != nil {
		t.Fatalf("a lista não é JSON de array: %v (%s)", err, resp.Body.String())
	}
	return lista
}

// idDe é o cookieDe dos recursos com identificador: confere o status e extrai o
// `id`, para o chamador não carregar um vazio difícil de ler adiante.
func idDe(t *testing.T, resp *httptest.ResponseRecorder, querido int) string {
	t.Helper()
	if resp.Code != querido {
		t.Fatalf("status = %d (%s), quero %d", resp.Code, resp.Body.String(), querido)
	}
	id := fmt.Sprint(decodificar(t, resp)["id"])
	if id == "" || id == "<nil>" {
		t.Fatalf("a resposta não trouxe identificador: %s", resp.Body.String())
	}
	return id
}

func pegarEnderecos(t *testing.T, rotas http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return pegarCom(t, rotas, "/api/v1/enderecos", cookie)
}

func postarEndereco(t *testing.T, rotas http.Handler, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return postarEnderecoComTipo(t, rotas, corpo, "application/json", cookie)
}

func postarEnderecoComTipo(t *testing.T, rotas http.Handler, corpo, tipo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodPost, "/api/v1/enderecos", corpo, tipo, cookie)
}

func putEndereco(t *testing.T, rotas http.Handler, id, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodPut, "/api/v1/enderecos/"+id, corpo, "application/json", cookie)
}

// putEnderecoValido tem a assinatura de pegarPedido — é o que deixa a tabela de
// recursos de negacaoPorDono misturar GET, PUT e DELETE sem um caso por método.
func putEnderecoValido(t *testing.T, rotas http.Handler, id string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return putEndereco(t, rotas, id, corpoEnderecoValido, cookie)
}

func deletarEndereco(t *testing.T, rotas http.Handler, id string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodDelete, "/api/v1/enderecos/"+id, "", "", cookie)
}

// comCorpo é o postarComTipo com método e cookie escolhidos: o PUT e o DELETE
// desta estória não cabiam nos ajudantes de POST que já existiam.
func comCorpo(t *testing.T, rotas http.Handler, metodo, rota, corpo, tipo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(metodo, rota, strings.NewReader(corpo))
	req.RemoteAddr = origemDeTeste
	if tipo != "" {
		req.Header.Set("Content-Type", tipo)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp := httptest.NewRecorder()
	rotas.ServeHTTP(resp, req)
	return resp
}
