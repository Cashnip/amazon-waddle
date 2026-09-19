package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// O teto por Item, no padrão da Config.
const carrinhoUnidadesMaxDeTeste = 10

// carrinhoDoComprador é a matriz de API da 4.1 num subteste só, com conta
// própria.
func carrinhoDoComprador(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	// A contagem é relativa: o negacaoPorDono, acima, deixa um Item do dono.
	total := func() int {
		n, _ := strconv.Atoi(textoDe(t, pool, `SELECT count(*)::text FROM carrinho.item_carrinho`)[0])
		return n
	}
	antes := total()
	contar := func() int { return total() - antes }
	quantidadeNoBanco := func(itemID string) []string {
		return textoDe(t, pool, `SELECT quantidade::text FROM carrinho.item_carrinho WHERE id = $1`, itemID)
	}

	if resp := postarItem(t, rotas, corpoItem(produtoSemeado, "3"), nil); resp.Code != http.StatusUnauthorized {
		t.Fatalf("sem Sessão = %d (%s), quero 401", resp.Code, resp.Body.String())
	} else if codigo := decodificar(t, resp)["codigo"]; codigo != "SESSAO_INVALIDA" {
		t.Errorf("sem Sessão: codigo = %v, quero SESSAO_INVALIDA", codigo)
	}
	if n := contar(); n != 0 {
		t.Fatalf("sem Sessão gravou: %d Itens", n)
	}

	// O Produto das adições tem Estoque de sobra: o disponível do semeado cai a
	// cada Pedido dos subtestes acima, e a soma até 9 não caberia mais nele.
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja do Carrinho"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Categoria do Carrinho"}`, admin), http.StatusCreated)
	corpoDe := func(nome string, estoque int, ativo bool) string {
		b, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": 4990, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": estoque, "ativo": ativo,
		})
		return string(b)
	}
	amplo := idDe(t, produtoCom(t, rotas, http.MethodPost, "", corpoDe("Chaleira Ampla", 20, true), admin), http.StatusCreated)

	cookie := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Lia Carrinho","email":"lia.carrinho@exemplo.br","senha":"senha-da-lia-1"}`), http.StatusCreated)

	// Primeira adição: o Carrinho e o Item nascem, com o preço atual visto.
	primeira := postarItem(t, rotas, corpoItem(amplo, "3"), cookie)
	item := idDe(t, primeira, http.StatusCreated)
	if corpo := decodificar(t, primeira); corpo["quantidade"] != 3.0 || corpo["produto_id"] != amplo {
		t.Errorf("primeira adição = %v, quero quantidade 3 do Produto", corpo)
	}
	if v := primeira.Header().Get("Cache-Control"); v != "no-store" {
		t.Errorf("Cache-Control = %q, quero no-store", v)
	}
	visto := textoDe(t, pool, `SELECT preco_visto_centavos::text FROM carrinho.item_carrinho WHERE id = $1`, item)
	preco := textoDe(t, pool, `SELECT preco_centavos::text FROM catalogo.produto WHERE id = $1`, amplo)
	if len(visto) != 1 || visto[0] != preco[0] {
		t.Errorf("preco_visto_centavos = %v, quero o preço atual %v", visto, preco)
	}

	// Repetida: soma ao mesmo Item, uma linha só.
	repetida := postarItem(t, rotas, corpoItem(amplo, "2"), cookie)
	if id := idDe(t, repetida, http.StatusCreated); id != item {
		t.Errorf("a repetida criou outro Item: %s, quero %s", id, item)
	}
	if q := decodificar(t, repetida)["quantidade"]; q != 5.0 {
		t.Errorf("quantidade = %v, quero 5", q)
	}
	if n := contar(); n != 1 {
		t.Errorf("%d Itens no banco, quero 1", n)
	}

	// Soma acima do teto: chega a 9, e +2 não grava.
	idDe(t, postarItem(t, rotas, corpoItem(amplo, "4"), cookie), http.StatusCreated)
	acima := postarItem(t, rotas, corpoItem(amplo, "2"), cookie)
	querCampoQuantidade(t, "soma acima do teto", acima)
	if q := quantidadeNoBanco(item); len(q) != 1 || q[0] != "9" {
		t.Errorf("a soma recusada gravou: quantidade = %v, quero 9", q)
	}

	// Quantidade fora da faixa, ausente ou não inteira.
	for _, q := range []string{"0", "11", "-1", "1.5", `"3"`, "null"} {
		querCampoQuantidade(t, "quantidade "+q, postarItem(t, rotas, corpoItem(amplo, q), cookie))
	}
	querCampoQuantidade(t, "quantidade ausente",
		postarItem(t, rotas, `{"produto_id":"`+amplo+`"}`, cookie))

	// Produto invisível: desativado, inexistente e uuid malformado.
	desativado := idDe(t, produtoCom(t, rotas, http.MethodPost, "", corpoDe("Chaleira Recolhida", 3, true), admin), http.StatusCreated)
	if resp := produtoCom(t, rotas, http.MethodPut, "/"+desativado, corpoDe("Chaleira Recolhida", 3, false), admin); resp.Code != http.StatusOK {
		t.Fatalf("desativar = %d (%s)", resp.Code, resp.Body.String())
	}
	for _, id := range []string{desativado, uuidNuncaUsado, "nao-e-uuid"} {
		resp := postarItem(t, rotas, corpoItem(id, "1"), cookie)
		if resp.Code != http.StatusNotFound {
			t.Errorf("Produto %s = %d (%s), quero 404", id, resp.Code, resp.Body.String())
		} else if codigo := decodificar(t, resp)["codigo"]; codigo != "NAO_ENCONTRADO" {
			t.Errorf("Produto %s: codigo = %v, quero NAO_ENCONTRADO", id, codigo)
		}
	}
	if n := contar(); n != 1 {
		t.Errorf("a recusa gravou: %d Itens, quero 1", n)
	}

	// Remover o próprio: 204, e a linha some.
	if resp := deletarItem(t, rotas, item, cookie); resp.Code != http.StatusNoContent {
		t.Fatalf("remover o próprio = %d (%s), quero 204", resp.Code, resp.Body.String())
	}
	if q := quantidadeNoBanco(item); len(q) != 0 {
		t.Errorf("o Item removido continua no banco: %v", q)
	}
	if resp := deletarItem(t, rotas, item, cookie); resp.Code != http.StatusNotFound {
		t.Errorf("remover de novo = %d (%s), quero 404", resp.Code, resp.Body.String())
	}
}

// carrinhoNaTela é a matriz da 4.2 e da 4.3 num subteste só: a recusa por
// Estoque informando o disponível, a alteração, a leitura pelos preços de agora
// e o esvaziar. Os Produtos saem do admin, com o Estoque que cada linha pede, e
// os Compradores do cadastro.
func carrinhoNaTela(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	contar := func(tabela string) int {
		n, _ := strconv.Atoi(textoDe(t, pool, `SELECT count(*)::text FROM carrinho.`+tabela)[0])
		return n
	}
	quantidadeNoBanco := func(itemID string) []string {
		return textoDe(t, pool, `SELECT quantidade::text FROM carrinho.item_carrinho WHERE id = $1`, itemID)
	}
	precoVistoNoBanco := func(itemID string) string {
		return textoDe(t, pool, `SELECT preco_visto_centavos::text FROM carrinho.item_carrinho WHERE id = $1`, itemID)[0]
	}

	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja da Tela do Carrinho"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Categoria da Tela do Carrinho"}`, admin), http.StatusCreated)
	corpoDe := func(nome string, preco, estoque int, ativo bool) string {
		b, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": preco, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": estoque, "ativo": ativo,
		})
		return string(b)
	}
	novoProduto := func(nome string, preco, estoque int) string {
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", corpoDe(nome, preco, estoque, true), admin), http.StatusCreated)
	}
	chaleira := novoProduto("Chaleira de Quatro", 5000, 4)
	bule := novoProduto("Bule de Três", 3000, 3)
	vazio := novoProduto("Produto Sem Estoque", 1000, 0)

	// Sem Sessão, as três rotas novas dão 401.
	for nome, resp := range map[string]*httptest.ResponseRecorder{
		"GET":           pegarCarrinho(t, rotas, nil),
		"PATCH":         alterarItem(t, rotas, uuidNuncaUsado, `{"quantidade":1}`, nil),
		"DELETE /itens": esvaziarCarrinho(t, rotas, nil),
	} {
		if resp.Code != http.StatusUnauthorized {
			t.Errorf("%s sem Sessão = %d (%s), quero 401", nome, resp.Code, resp.Body.String())
		} else if codigo := decodificar(t, resp)["codigo"]; codigo != "SESSAO_INVALIDA" {
			t.Errorf("%s sem Sessão: codigo = %v, quero SESSAO_INVALIDA", nome, codigo)
		}
	}

	lia := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Lia da Tela","email":"lia.tela@exemplo.br","senha":"senha-da-lia-1"}`), http.StatusCreated)
	bia := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Bia da Tela","email":"bia.tela@exemplo.br","senha":"senha-da-bia-1"}`), http.StatusCreated)

	// Sem Carrinho: a lista vazia sai `[]`, e não `null`, e ler não cria nada.
	carrinhosAntes := contar("carrinho")
	vazioLido := pegarCarrinho(t, rotas, lia)
	if vazioLido.Code != http.StatusOK || !strings.Contains(vazioLido.Body.String(), `"itens":[]`) {
		t.Errorf("Carrinho sem Itens = %d (%s), quero 200 com itens []", vazioLido.Code, vazioLido.Body.String())
	}
	if corpo := decodificar(t, vazioLido); corpo["unidades"] != 0.0 || corpo["subtotal_centavos"] != 0.0 {
		t.Errorf("Carrinho sem Itens = %v, quero unidades e subtotal 0", corpo)
	}
	if v := vazioLido.Header().Get("Cache-Control"); v != "no-store" {
		t.Errorf("Cache-Control = %q, quero no-store", v)
	}

	// Acima do Estoque na primeira adição: o 409 nomeia o disponível, e nada
	// grava — nem o Carrinho.
	querEstoqueInsuficiente(t, "acima do Estoque", postarItem(t, rotas, corpoItem(bule, "4"), lia),
		"Restam 3 unidades de Bule de Três.", 3, 4)
	if n := contar("carrinho"); n != carrinhosAntes {
		t.Errorf("a recusa criou Carrinho: %d, quero %d", n, carrinhosAntes)
	}
	// Sem Estoque nenhum, o Produto está indisponível, e não "restam 0".
	querEstoqueInsuficiente(t, "sem Estoque", postarItem(t, rotas, corpoItem(vazio, "1"), lia),
		"Produto Sem Estoque está indisponível.", 0, 1)

	// A soma, e não só o acréscimo: 3 no Item e +2 pediria 5 de 4.
	itemChaleira := idDe(t, postarItem(t, rotas, corpoItem(chaleira, "3"), lia), http.StatusCreated)
	querEstoqueInsuficiente(t, "soma acima do Estoque", postarItem(t, rotas, corpoItem(chaleira, "2"), lia),
		"Restam 4 unidades de Chaleira de Quatro.", 4, 5)
	if q := quantidadeNoBanco(itemChaleira); len(q) != 1 || q[0] != "3" {
		t.Errorf("a soma recusada gravou: quantidade = %v, quero 3", q)
	}

	// O Carrinho não reserva: dois Compradores põem as 3 unidades do mesmo Produto.
	itemBule := idDe(t, postarItem(t, rotas, corpoItem(bule, "3"), lia), http.StatusCreated)
	idDe(t, postarItem(t, rotas, corpoItem(bule, "3"), bia), http.StatusCreated)
	if d := textoDe(t, pool, `SELECT estoque_disponivel::text FROM catalogo.produto_visivel WHERE id = $1`, bule); len(d) != 1 || d[0] != "3" {
		t.Errorf("disponível do Bule = %v, quero 3: o Carrinho não reserva", d)
	}

	// Ler: pelos preços de agora, e sem gravar o preço visto (AD-17).
	corpoLido := decodificar(t, pegarCarrinho(t, rotas, lia))
	itens, _ := corpoLido["itens"].([]any)
	if len(itens) != 2 || corpoLido["unidades"] != 6.0 || corpoLido["subtotal_centavos"] != 24000.0 {
		t.Fatalf("Carrinho da Lia = %v, quero 2 Itens, 6 unidades e subtotal 24000", corpoLido)
	}
	if primeiro, _ := itens[0].(map[string]any); primeiro["id"] != itemChaleira || primeiro["nome"] != "Chaleira de Quatro" ||
		primeiro["visivel"] != true || primeiro["preco_centavos"] != 5000.0 || primeiro["quantidade"] != 3.0 {
		t.Errorf("primeira linha = %v, quero a Chaleira, na ordem da adição", primeiro)
	}
	if resp := produtoCom(t, rotas, http.MethodPut, "/"+chaleira, corpoDe("Chaleira de Quatro", 6000, 4, true), admin); resp.Code != http.StatusOK {
		t.Fatalf("mudar o preço = %d (%s)", resp.Code, resp.Body.String())
	}
	if corpo := decodificar(t, pegarCarrinho(t, rotas, lia)); corpo["subtotal_centavos"] != 27000.0 {
		t.Errorf("subtotal com o preço novo = %v, quero 27000", corpo["subtotal_centavos"])
	}
	if visto := precoVistoNoBanco(itemChaleira); visto != "5000" {
		t.Errorf("preco_visto_centavos depois de ler = %s, quero 5000: ler não grava", visto)
	}

	// Alterar: a quantidade é absoluta, e o preço visto passa a ser o de agora.
	alterada := alterarItem(t, rotas, itemChaleira, `{"quantidade":2}`, lia)
	if alterada.Code != http.StatusOK {
		t.Fatalf("alterar = %d (%s), quero 200", alterada.Code, alterada.Body.String())
	}
	if corpo := decodificar(t, alterada); corpo["id"] != itemChaleira || corpo["quantidade"] != 2.0 {
		t.Errorf("Item alterado = %v, quero o mesmo id com quantidade 2", corpo)
	}
	if visto := precoVistoNoBanco(itemChaleira); visto != "6000" {
		t.Errorf("preco_visto_centavos depois de alterar = %s, quero 6000", visto)
	}
	if q := quantidadeNoBanco(itemChaleira); len(q) != 1 || q[0] != "2" {
		t.Errorf("quantidade no banco = %v, quero 2", q)
	}

	// Recusas da alteração: a quantidade fora da faixa, e acima do disponível.
	for _, q := range []string{"-1", "11", "1.5", `"2"`, "null"} {
		querCampoQuantidade(t, "alterar para "+q, alterarItem(t, rotas, itemChaleira, `{"quantidade":`+q+`}`, lia))
	}
	querCampoQuantidade(t, "alterar sem quantidade", alterarItem(t, rotas, itemChaleira, `{}`, lia))
	querEstoqueInsuficiente(t, "alterar acima do Estoque", alterarItem(t, rotas, itemChaleira, `{"quantidade":5}`, lia),
		"Restam 4 unidades de Chaleira de Quatro.", 4, 5)
	if q := quantidadeNoBanco(itemChaleira); len(q) != 1 || q[0] != "2" {
		t.Errorf("a alteração recusada gravou: quantidade = %v, quero 2", q)
	}

	// Zero remove: 204, e a linha some.
	if resp := alterarItem(t, rotas, itemBule, `{"quantidade":0}`, lia); resp.Code != http.StatusNoContent {
		t.Fatalf("alterar para zero = %d (%s), quero 204", resp.Code, resp.Body.String())
	}
	if q := quantidadeNoBanco(itemBule); len(q) != 0 {
		t.Errorf("o Item zerado continua no banco: %v", q)
	}
	if resp := alterarItem(t, rotas, itemBule, `{"quantidade":0}`, lia); resp.Code != http.StatusNotFound {
		t.Errorf("zerar de novo = %d (%s), quero 404", resp.Code, resp.Body.String())
	}
	// O uuid malformado é o mesmo 404 do inexistente, na alteração e no zero.
	for _, corpo := range []string{`{"quantidade":2}`, `{"quantidade":0}`} {
		if resp := alterarItem(t, rotas, "nao-e-uuid", corpo, lia); resp.Code != http.StatusNotFound {
			t.Errorf("alterar com uuid malformado (%s) = %d (%s), quero 404", corpo, resp.Code, resp.Body.String())
		} else if codigo := decodificar(t, resp)["codigo"]; codigo != "NAO_ENCONTRADO" {
			t.Errorf("alterar com uuid malformado (%s): codigo = %v, quero NAO_ENCONTRADO", corpo, codigo)
		}
	}

	// O Produto que saiu da visibilidade: a linha continua, sem nome nem preço e
	// fora do subtotal, e só sai por remoção. Alterar a quantidade dele é 404.
	if resp := produtoCom(t, rotas, http.MethodPut, "/"+chaleira, corpoDe("Chaleira de Quatro", 6000, 4, false), admin); resp.Code != http.StatusOK {
		t.Fatalf("desativar = %d (%s)", resp.Code, resp.Body.String())
	}
	invisivel := decodificar(t, pegarCarrinho(t, rotas, lia))
	linhas, _ := invisivel["itens"].([]any)
	if len(linhas) != 1 || invisivel["unidades"] != 2.0 || invisivel["subtotal_centavos"] != 0.0 {
		t.Fatalf("Carrinho com o Produto desativado = %v, quero 1 linha, 2 unidades e subtotal 0", invisivel)
	}
	if linha, _ := linhas[0].(map[string]any); linha["visivel"] != false || linha["nome"] != "" || linha["preco_centavos"] != 0.0 {
		t.Errorf("linha do Produto desativado = %v, quero visivel falso e sem nome nem preço", linha)
	}
	if resp := alterarItem(t, rotas, itemChaleira, `{"quantidade":1}`, lia); resp.Code != http.StatusNotFound {
		t.Errorf("alterar Produto desativado = %d (%s), quero 404", resp.Code, resp.Body.String())
	} else if codigo := decodificar(t, resp)["codigo"]; codigo != "NAO_ENCONTRADO" {
		t.Errorf("alterar Produto desativado: codigo = %v, quero NAO_ENCONTRADO", codigo)
	}
	if q := quantidadeNoBanco(itemChaleira); len(q) != 1 || q[0] != "2" {
		t.Errorf("a alteração recusada gravou: quantidade = %v, quero 2", q)
	}

	// Esvaziar: só o Carrinho do dono, e Carrinho vazio também dá 204.
	if resp := esvaziarCarrinho(t, rotas, lia); resp.Code != http.StatusNoContent {
		t.Fatalf("esvaziar = %d (%s), quero 204", resp.Code, resp.Body.String())
	}
	if corpo := decodificar(t, pegarCarrinho(t, rotas, lia)); corpo["unidades"] != 0.0 {
		t.Errorf("Carrinho da Lia depois de esvaziar = %v, quero 0 unidades", corpo)
	}
	if corpo := decodificar(t, pegarCarrinho(t, rotas, bia)); corpo["unidades"] != 3.0 {
		t.Errorf("Carrinho da Bia depois de a Lia esvaziar = %v, quero as 3 unidades dela", corpo)
	}
	if resp := esvaziarCarrinho(t, rotas, lia); resp.Code != http.StatusNoContent {
		t.Errorf("esvaziar o vazio = %d (%s), quero 204", resp.Code, resp.Body.String())
	}
}

// querEstoqueInsuficiente confere o 409 do AD-14: a mensagem nomeia o Produto e
// o disponível, e `dados` leva os dois números.
func querEstoqueInsuficiente(t *testing.T, caso string, resp *httptest.ResponseRecorder, mensagem string, disponivel, solicitado float64) {
	t.Helper()
	if resp.Code != http.StatusConflict {
		t.Errorf("%s = %d (%s), quero 409", caso, resp.Code, resp.Body.String())
		return
	}
	corpo := decodificar(t, resp)
	if corpo["codigo"] != "ESTOQUE_INSUFICIENTE" || corpo["mensagem"] != mensagem {
		t.Errorf("%s: %v, quero ESTOQUE_INSUFICIENTE com %q", caso, corpo, mensagem)
	}
	dados, _ := corpo["dados"].(map[string]any)
	if dados["disponivel"] != disponivel || dados["solicitado"] != solicitado || dados["produto_id"] == "" {
		t.Errorf("%s: dados = %v, quero disponivel %v e solicitado %v", caso, dados, disponivel, solicitado)
	}
}

func querCampoQuantidade(t *testing.T, caso string, resp *httptest.ResponseRecorder) {
	t.Helper()
	if resp.Code != http.StatusBadRequest {
		t.Errorf("%s = %d (%s), quero 400", caso, resp.Code, resp.Body.String())
		return
	}
	dados, _ := decodificar(t, resp)["dados"].(map[string]any)
	if dados["campo"] != "quantidade" {
		t.Errorf("%s: dados = %v, quero o campo quantidade", caso, dados)
	}
}

func corpoItem(produtoID, quantidade string) string {
	return `{"produto_id":"` + produtoID + `","quantidade":` + quantidade + `}`
}

func postarItem(t *testing.T, rotas http.Handler, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodPost, "/api/v1/carrinho/itens", corpo, "application/json", cookie)
}

func pegarCarrinho(t *testing.T, rotas http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodGet, "/api/v1/carrinho", "", "", cookie)
}

func alterarItem(t *testing.T, rotas http.Handler, id, corpo string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodPatch, "/api/v1/carrinho/itens/"+id, corpo, "application/json", cookie)
}

// alterarItemPara5 tem a assinatura de pegarPedido, para entrar na tabela de
// negacaoPorDono. O 5 cabe no teto, e nunca chega ao Estoque: o Item alheio é
// 404 antes disso.
func alterarItemPara5(t *testing.T, rotas http.Handler, id string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return alterarItem(t, rotas, id, `{"quantidade":5}`, cookie)
}

func esvaziarCarrinho(t *testing.T, rotas http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodDelete, "/api/v1/carrinho/itens", "", "", cookie)
}

// deletarItem tem a assinatura de pegarPedido, para entrar na tabela de
// negacaoPorDono.
func deletarItem(t *testing.T, rotas http.Handler, id string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodDelete, "/api/v1/carrinho/itens/"+id, "", "", cookie)
}

// carrinhoRevalidado é a matriz da 4.4 num subteste só: cada linha do Carrinho
// aberto sai com o que mudou (`preco_mudou`) e o que a impede (`bloqueio`), e
// ler nunca grava o preço visto (AD-17, FR-19).
func carrinhoRevalidado(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	precoVistoNoBanco := func(itemID string) string {
		return textoDe(t, pool, `SELECT preco_visto_centavos::text FROM carrinho.item_carrinho WHERE id = $1`, itemID)[0]
	}

	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	lojaA := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja A da Revalidação"}`, admin), http.StatusCreated)
	lojaB := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja B da Revalidação"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Categoria da Revalidação"}`, admin), http.StatusCreated)
	corpoDe := func(nome, vendedor string, preco, estoque int, ativo bool) string {
		b, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": preco, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": estoque, "ativo": ativo,
		})
		return string(b)
	}
	novoProduto := func(nome, vendedor string, preco, estoque int) string {
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", corpoDe(nome, vendedor, preco, estoque, true), admin), http.StatusCreated)
	}
	mudarPreco := func(id, nome string, preco int) {
		t.Helper()
		if resp := produtoCom(t, rotas, http.MethodPut, "/"+id, corpoDe(nome, lojaA, preco, 10, true), admin); resp.Code != http.StatusOK {
			t.Fatalf("mudar o preço de %s = %d (%s)", nome, resp.Code, resp.Body.String())
		}
	}
	ajustarEstoque := func(id string, total int) {
		t.Helper()
		corpo := `{"estoque_total":` + strconv.Itoa(total) + `}`
		if resp := produtoCom(t, rotas, http.MethodPut, "/"+id+"/estoque", corpo, admin); resp.Code != http.StatusOK {
			t.Fatalf("ajustar o Estoque de %s = %d (%s)", id, resp.Code, resp.Body.String())
		}
	}

	chaleira := novoProduto("Chaleira Revalidada", lojaA, 5000, 10)
	bule := novoProduto("Bule Revalidado", lojaA, 3000, 10)
	caneca := novoProduto("Caneca Revalidada", lojaA, 800, 10)
	panela := novoProduto("Panela Revalidada", lojaA, 9000, 10)
	garrafa := novoProduto("Garrafa Revalidada", lojaB, 2000, 10)

	duda := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Duda da Revalidação","email":"duda.revalidacao@exemplo.br","senha":"senha-da-duda-1"}`), http.StatusCreated)
	itemChaleira := idDe(t, postarItem(t, rotas, corpoItem(chaleira, "2"), duda), http.StatusCreated)
	itemBule := idDe(t, postarItem(t, rotas, corpoItem(bule, "4"), duda), http.StatusCreated)
	idDe(t, postarItem(t, rotas, corpoItem(caneca, "1"), duda), http.StatusCreated)
	idDe(t, postarItem(t, rotas, corpoItem(panela, "1"), duda), http.StatusCreated)
	idDe(t, postarItem(t, rotas, corpoItem(garrafa, "1"), duda), http.StatusCreated)

	// A linha de cada Produto, lida do Carrinho aberto agora.
	linhas := func() map[string]map[string]any {
		t.Helper()
		resp := pegarCarrinho(t, rotas, duda)
		if resp.Code != http.StatusOK {
			t.Fatalf("abrir o Carrinho = %d (%s)", resp.Code, resp.Body.String())
		}
		saida := map[string]map[string]any{}
		itens, _ := decodificar(t, resp)["itens"].([]any)
		for _, bruto := range itens {
			linha, _ := bruto.(map[string]any)
			saida[linha["produto_id"].(string)] = linha
		}
		return saida
	}
	quer := func(caso string, linha map[string]any, precoMudou bool, bloqueio string) {
		t.Helper()
		if linha["preco_mudou"] != precoMudou || linha["bloqueio"] != bloqueio {
			t.Errorf("%s: preco_mudou = %v e bloqueio = %q, quero %v e %q (linha %v)",
				caso, linha["preco_mudou"], linha["bloqueio"], precoMudou, bloqueio, linha)
		}
	}

	// Limpo: nada mudou, e o disponível é o do Catálogo.
	limpo := linhas()
	for _, id := range []string{chaleira, bule, caneca, panela, garrafa} {
		quer("limpo", limpo[id], false, "")
		if limpo[id]["preco_visto_centavos"] != limpo[id]["preco_centavos"] || limpo[id]["estoque_disponivel"] != 10.0 {
			t.Errorf("limpo: linha = %v, quero o preço visto igual ao atual e disponível 10", limpo[id])
		}
	}

	// Preço mudou: o "de X para Y" tem os dois números, e ler não grava.
	mudarPreco(chaleira, "Chaleira Revalidada", 6000)
	mudado := linhas()
	quer("preço mudou", mudado[chaleira], true, "")
	if mudado[chaleira]["preco_visto_centavos"] != 5000.0 || mudado[chaleira]["preco_centavos"] != 6000.0 {
		t.Errorf("preço mudou: linha = %v, quero visto 5000 e atual 6000", mudado[chaleira])
	}
	quer("o outro Produto não muda", mudado[bule], false, "")
	linhas()
	if visto := precoVistoNoBanco(itemChaleira); visto != "5000" {
		t.Errorf("preco_visto_centavos depois de abrir o Carrinho = %s, quero 5000: ler não grava", visto)
	}

	// Mudou e voltou: o preço visto é o atual, e não há o que confirmar.
	mudarPreco(chaleira, "Chaleira Revalidada", 5000)
	quer("preço de volta", linhas()[chaleira], false, "")

	// Acima do Estoque: há disponível, mas menos que a quantidade do Item.
	ajustarEstoque(bule, 2)
	acima := linhas()[bule]
	quer("acima do Estoque", acima, false, "acima_do_estoque")
	if acima["estoque_disponivel"] != 2.0 || acima["quantidade"] != 4.0 {
		t.Errorf("acima do Estoque: linha = %v, quero disponível 2 e quantidade 4", acima)
	}

	// Sem Estoque: visível, mas indisponível — só remover resolve.
	ajustarEstoque(caneca, 0)
	sem := linhas()[caneca]
	quer("sem Estoque", sem, false, "indisponivel")
	if sem["visivel"] != true || sem["estoque_disponivel"] != 0.0 {
		t.Errorf("sem Estoque: linha = %v, quero visível com disponível 0", sem)
	}

	// Produto desativado e Vendedor desativado saem iguais: invisíveis, sem nome
	// nem preço, indisponíveis, e o preço não conta como mudado.
	if resp := produtoCom(t, rotas, http.MethodPut, "/"+panela, corpoDe("Panela Revalidada", lojaA, 9000, 10, false), admin); resp.Code != http.StatusOK {
		t.Fatalf("desativar o Produto = %d (%s)", resp.Code, resp.Body.String())
	}
	if resp := vendedorCom(t, rotas, http.MethodPut, "/"+lojaB, `{"nome":"Loja B da Revalidação","ativo":false}`, admin); resp.Code != http.StatusOK {
		t.Fatalf("desativar o Vendedor = %d (%s)", resp.Code, resp.Body.String())
	}
	fora := linhas()
	for caso, id := range map[string]string{"Produto desativado": panela, "Vendedor desativado": garrafa} {
		quer(caso, fora[id], false, "indisponivel")
		if fora[id]["visivel"] != false || fora[id]["nome"] != "" || fora[id]["preco_centavos"] != 0.0 || fora[id]["estoque_disponivel"] != 0.0 {
			t.Errorf("%s: linha = %v, quero invisível e sem nome, preço nem Estoque", caso, fora[id])
		}
	}

	// Duas mudanças na mesma linha: o preço e o Estoque. Ajustar para o
	// disponível é a alteração da 4.3, que dá ciência do preço daquela linha.
	mudarPreco(chaleira, "Chaleira Revalidada", 6000)
	ajustarEstoque(chaleira, 1)
	duas := linhas()[chaleira]
	quer("duas mudanças", duas, true, "acima_do_estoque")
	if resp := alterarItem(t, rotas, itemChaleira, `{"quantidade":1}`, duda); resp.Code != http.StatusOK {
		t.Fatalf("ajustar para o disponível = %d (%s)", resp.Code, resp.Body.String())
	}
	quer("depois de ajustar", linhas()[chaleira], false, "")
	if visto := precoVistoNoBanco(itemChaleira); visto != "6000" {
		t.Errorf("preco_visto_centavos depois de ajustar = %s, quero 6000", visto)
	}

	// Ajustar o Bule para o disponível libera a linha.
	if resp := alterarItem(t, rotas, itemBule, `{"quantidade":2}`, duda); resp.Code != http.StatusOK {
		t.Fatalf("ajustar o Bule = %d (%s)", resp.Code, resp.Body.String())
	}
	quer("Bule ajustado", linhas()[bule], false, "")
}
