package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// entradaNoCheckout é a matriz da 5.5 contra o banco de verdade: a rota que
// revalida o Carrinho, **reporta** o que mudou e só então grava a ciência dos
// preços reportados (AD-17, FR-19). O que se prova aqui é a ordem — a resposta
// traz o preço anterior e o atual, e o banco, depois da mesma chamada, já
// guarda o atual — e que a segunda entrada silencia.
func entradaNoCheckout(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	if resp := postarEntrada(t, rotas, nil); resp.Code != http.StatusUnauthorized {
		t.Errorf("sem Sessão = %d (%s), quero 401", resp.Code, resp.Body.String())
	}

	precoVistoNoBanco := func(itemID string) string {
		t.Helper()
		return textoDe(t, pool, `SELECT preco_visto_centavos::text FROM carrinho.item_carrinho WHERE id = $1`, itemID)[0]
	}

	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	loja := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja da Entrada"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Categoria da Entrada"}`, admin), http.StatusCreated)
	corpoDe := func(nome string, preco, estoque int, ativo bool) string {
		b, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": preco, "imagem_url": "",
			"vendedor_id": loja, "categoria_id": categoria, "estoque_total": estoque, "ativo": ativo,
		})
		return string(b)
	}
	novoProduto := func(nome string, preco, estoque int) string {
		t.Helper()
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", corpoDe(nome, preco, estoque, true), admin), http.StatusCreated)
	}
	// O Administrador mexe no Produto: preço, Estoque total e visibilidade.
	// É a mudança que o Comprador NÃO provocou — a única que ConfirmarPrecoVisto
	// tem para persistir.
	publicar := func(id, nome string, preco, estoque int, ativo bool) {
		t.Helper()
		if resp := produtoCom(t, rotas, http.MethodPut, "/"+id, corpoDe(nome, preco, estoque, ativo), admin); resp.Code != http.StatusOK {
			t.Fatalf("republicar %s = %d (%s)", nome, resp.Code, resp.Body.String())
		}
	}
	ajustarEstoqueTotal := func(id string, total int) {
		t.Helper()
		corpo := `{"estoque_total":` + strconv.Itoa(total) + `}`
		if resp := produtoCom(t, rotas, http.MethodPut, "/"+id+"/estoque", corpo, admin); resp.Code != http.StatusOK {
			t.Fatalf("ajustar o Estoque de %s = %d (%s)", id, resp.Code, resp.Body.String())
		}
	}

	// As linhas da resposta, por Produto. Serve para a entrada e para o
	// `GET /api/v1/carrinho`: o envelope é o mesmo, de propósito.
	linhasDe := func(caso string, resp *httptest.ResponseRecorder) map[string]map[string]any {
		t.Helper()
		if resp.Code != http.StatusOK {
			t.Fatalf("%s = %d (%s), quero 200", caso, resp.Code, resp.Body.String())
		}
		if v := resp.Header().Get("Cache-Control"); v != "no-store" {
			t.Errorf("%s: Cache-Control = %q, quero no-store", caso, v)
		}
		corpo := decodificar(t, resp)
		if corpo["frete_isencao_centavos"] != float64(freteIsencaoDeTeste) {
			t.Errorf("%s: frete_isencao_centavos = %v, quero %d — o envelope é o do Carrinho",
				caso, corpo["frete_isencao_centavos"], freteIsencaoDeTeste)
		}
		saida := map[string]map[string]any{}
		itens, _ := corpo["itens"].([]any)
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

	chaleira := novoProduto("Chaleira da Entrada", 5000, 10)
	bule := novoProduto("Bule da Entrada", 3000, 10)
	caneca := novoProduto("Caneca da Entrada", 800, 10)
	panela := novoProduto("Panela da Entrada", 9000, 10)

	clara := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Clara do Checkout","email":"clara.checkout@exemplo.br","senha":"senha-da-clara-1"}`), http.StatusCreated)
	itemChaleira := idDe(t, postarItem(t, rotas, corpoItem(chaleira, "2"), clara), http.StatusCreated)
	itemBule := idDe(t, postarItem(t, rotas, corpoItem(bule, "4"), clara), http.StatusCreated)
	itemCaneca := idDe(t, postarItem(t, rotas, corpoItem(caneca, "1"), clara), http.StatusCreated)
	itemPanela := idDe(t, postarItem(t, rotas, corpoItem(panela, "1"), clara), http.StatusCreated)

	// O Carrinho de outro Comprador, montado antes de tudo mudar: nenhum preço
	// visto dele pode se mexer quando Clara entra no checkout (AD-11, NFR-6).
	otavio := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Otávio de Fora","email":"otavio.checkout@exemplo.br","senha":"senha-do-otavio-1"}`), http.StatusCreated)
	itemAlheio := idDe(t, postarItem(t, rotas, corpoItem(chaleira, "1"), otavio), http.StatusCreated)

	// Entrada limpa: nada mudou, nada bloqueia, e o visto é o preço de agora.
	limpo := linhasDe("entrada limpa", postarEntrada(t, rotas, clara))
	for _, id := range []string{chaleira, bule, caneca, panela} {
		quer("entrada limpa", limpo[id], false, "")
		if limpo[id]["preco_visto_centavos"] != limpo[id]["preco_centavos"] {
			t.Errorf("entrada limpa: linha = %v, quero o visto igual ao atual", limpo[id])
		}
	}

	// **A linha central da estória.** O preço muda pelas costas do Comprador; a
	// resposta traz o "de 5000 para 6000", e o banco, depois da MESMA chamada,
	// já guarda 6000. Reportar e confirmar acontecem na mesma transação, nessa
	// ordem — invertida, o 5000 nunca chegaria à tela.
	publicar(chaleira, "Chaleira da Entrada", 6000, 10, true)
	mudou := linhasDe("preço mudou", postarEntrada(t, rotas, clara))
	quer("preço mudou", mudou[chaleira], true, "")
	if mudou[chaleira]["preco_visto_centavos"] != 5000.0 || mudou[chaleira]["preco_centavos"] != 6000.0 {
		t.Errorf("preço mudou: linha = %v, quero visto 5000 e atual 6000", mudou[chaleira])
	}
	if visto := precoVistoNoBanco(itemChaleira); visto != "6000" {
		t.Errorf("preco_visto_centavos depois da entrada = %s, quero 6000: a resposta reporta e a transação grava", visto)
	}
	quer("o outro Produto não muda", mudou[bule], false, "")

	// Entrar de novo não avisa mais nada, e o `GET /api/v1/carrinho` concorda:
	// a ciência ficou no banco, e não na tela.
	denovo := linhasDe("entrar de novo", postarEntrada(t, rotas, clara))
	quer("entrar de novo", denovo[chaleira], false, "")
	if denovo[chaleira]["preco_visto_centavos"] != 6000.0 {
		t.Errorf("entrar de novo: linha = %v, quero visto 6000", denovo[chaleira])
	}
	quer("o Carrinho concorda", linhasDe("o Carrinho concorda", pegarCarrinho(t, rotas, clara))[chaleira], false, "")

	// Ler não grava: o `GET` sobre um preço que mudou continua avisando.
	publicar(caneca, "Caneca da Entrada", 900, 10, true)
	quer("ler não grava", linhasDe("ler não grava", pegarCarrinho(t, rotas, clara))[caneca], true, "")
	if visto := precoVistoNoBanco(itemCaneca); visto != "800" {
		t.Errorf("preco_visto_centavos depois do GET = %s, quero 800: ler nunca grava (AD-17)", visto)
	}

	// Acima do Estoque: há disponível, mas menos que a quantidade do Item.
	ajustarEstoqueTotal(bule, 2)
	acima := linhasDe("acima do Estoque", postarEntrada(t, rotas, clara))
	quer("acima do Estoque", acima[bule], false, "acima_do_estoque")
	if acima[bule]["estoque_disponivel"] != 2.0 || acima[bule]["quantidade"] != 4.0 {
		t.Errorf("acima do Estoque: linha = %v, quero disponível 2 e quantidade 4", acima[bule])
	}
	// E a mesma entrada reportou a Caneca — o GET anterior avisara e não
	// gravara — e só depois a confirmou: a resposta traz o "de 800 para 900",
	// e o banco já guarda 900.
	quer("a Caneca é reportada", acima[caneca], true, "")
	if acima[caneca]["preco_visto_centavos"] != 800.0 || acima[caneca]["preco_centavos"] != 900.0 {
		t.Errorf("a Caneca: linha = %v, quero visto 800 e atual 900", acima[caneca])
	}
	if visto := precoVistoNoBanco(itemCaneca); visto != "900" {
		t.Errorf("preco_visto_centavos da Caneca = %s, quero 900", visto)
	}

	// As duas coisas na mesma linha: o preço é confirmado e o bloqueio fica.
	// Confirmar preço não desbloqueia — as duas listas são independentes.
	publicar(bule, "Bule da Entrada", 3500, 2, true)
	duas := linhasDe("preço mudado em linha bloqueada", postarEntrada(t, rotas, clara))
	quer("preço mudado em linha bloqueada", duas[bule], true, "acima_do_estoque")
	if duas[bule]["preco_visto_centavos"] != 3000.0 || duas[bule]["preco_centavos"] != 3500.0 {
		t.Errorf("linha bloqueada: linha = %v, quero visto 3000 e atual 3500", duas[bule])
	}
	if visto := precoVistoNoBanco(itemBule); visto != "3500" {
		t.Errorf("preco_visto_centavos da linha bloqueada = %s, quero 3500: o preço é confirmado mesmo assim", visto)
	}
	ficou := linhasDe("o bloqueio fica", postarEntrada(t, rotas, clara))
	quer("o bloqueio fica", ficou[bule], false, "acima_do_estoque")

	// Invisível com preço antigo: a linha não tem preço a confirmar, e o visto
	// dela não é tocado — nem agora, nem em nenhuma entrada seguinte.
	publicar(panela, "Panela da Entrada", 9500, 10, false)
	fora := linhasDe("Produto invisível", postarEntrada(t, rotas, clara))
	quer("Produto invisível", fora[panela], false, "indisponivel")
	// A linha invisível sai sem nome, preço nem visto — é o mesmo envelope que
	// o Carrinho (4.4) já devolvia, e não há "de X para Y" a mostrar.
	if fora[panela]["visivel"] != false || fora[panela]["nome"] != "" ||
		fora[panela]["preco_centavos"] != 0.0 || fora[panela]["preco_visto_centavos"] != 0.0 {
		t.Errorf("Produto invisível: linha = %v, quero invisível e sem nome, preço nem visto", fora[panela])
	}
	if visto := precoVistoNoBanco(itemPanela); visto != "9000" {
		t.Errorf("preco_visto_centavos do invisível = %s, quero 9000 intacto", visto)
	}

	// O Carrinho alheio ficou intacto o tempo todo: o preço da Chaleira mudou
	// de 5000 para 6000, e o Item do Otávio nunca foi confirmado.
	if visto := precoVistoNoBanco(itemAlheio); visto != "5000" {
		t.Errorf("preco_visto_centavos do Carrinho alheio = %s, quero 5000 intacto (AD-11)", visto)
	}
	quer("o alheio ainda avisa", linhasDe("o alheio ainda avisa", pegarCarrinho(t, rotas, otavio))[chaleira], true, "")

	// Carrinho só de Produtos invisíveis: não é vazio, e todas as linhas
	// bloqueiam. É por aqui que a Revisão devolve ao Carrinho.
	abajur := novoProduto("Abajur da Entrada", 4000, 10)
	vera := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Vera do Invisível","email":"vera.checkout@exemplo.br","senha":"senha-da-vera-1"}`), http.StatusCreated)
	idDe(t, postarItem(t, rotas, corpoItem(abajur, "1"), vera), http.StatusCreated)
	publicar(abajur, "Abajur da Entrada", 4000, 10, false)
	soInvisivel := decodificar(t, postarEntrada(t, rotas, vera))
	itensDaVera, _ := soInvisivel["itens"].([]any)
	if len(itensDaVera) != 1 || soInvisivel["subtotal_centavos"] != 0.0 {
		t.Fatalf("só invisíveis = %v, quero uma linha e subtotal 0 — não é Carrinho vazio", soInvisivel)
	}
	if linha, _ := itensDaVera[0].(map[string]any); linha["bloqueio"] != "indisponivel" {
		t.Errorf("só invisíveis: linha = %v, quero bloqueio indisponivel em todas", linha)
	}

	// Carrinho vazio: a entrada responde o envelope vazio, e quem devolve ao
	// Carrinho é a tela. `itens` é `[]`, e nunca `null`.
	zeca := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Zeca sem Carrinho","email":"zeca.checkout@exemplo.br","senha":"senha-do-zeca-1"}`), http.StatusCreated)
	vazio := decodificar(t, postarEntrada(t, rotas, zeca))
	if itens, ok := vazio["itens"].([]any); !ok || len(itens) != 0 || vazio["unidades"] != 0.0 {
		t.Errorf("Carrinho vazio = %v, quero itens [] e zero unidades", vazio)
	}

	// Cruzar o limiar de isenção, nos dois sentidos: a entrada não calcula
	// Frete nenhum, mas o preço que ela confirma é o que a cotação seguinte
	// soma. O limiar é o freteIsencaoDeTeste (R$ 250,00) do ambiente.
	escrivaninha := novoProduto("Escrivaninha da Entrada", 26000, 10)
	livia := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Lívia do Limiar","email":"livia.checkout@exemplo.br","senha":"senha-da-livia-1"}`), http.StatusCreated)
	idDe(t, postarItem(t, rotas, corpoItem(escrivaninha, "1"), livia), http.StatusCreated)
	enderecoDaLivia := idDe(t, postarEndereco(t, rotas, corpoEnderecoValido, livia), http.StatusCreated)
	linhasDe("limiar: entrada limpa", postarEntrada(t, rotas, livia))
	querCotacao(t, "acima do limiar: Grátis", pegarFrete(t, rotas, enderecoDaLivia, livia), "Sudeste", 26000, 0)

	publicar(escrivaninha, "Escrivaninha da Entrada", 24000, 10, true)
	abaixo := linhasDe("limiar: o preço caiu", postarEntrada(t, rotas, livia))
	quer("limiar: o preço caiu", abaixo[escrivaninha], true, "")
	querCotacao(t, "abaixo do limiar: o Frete volta", pegarFrete(t, rotas, enderecoDaLivia, livia), "Sudeste", 24000, 1500)

	// E de volta para cima: o mesmo mecanismo nos dois sentidos.
	publicar(escrivaninha, "Escrivaninha da Entrada", 26000, 10, true)
	acimaDeNovo := linhasDe("limiar: o preço subiu", postarEntrada(t, rotas, livia))
	quer("limiar: o preço subiu", acimaDeNovo[escrivaninha], true, "")
	querCotacao(t, "de volta acima do limiar: Grátis", pegarFrete(t, rotas, enderecoDaLivia, livia), "Sudeste", 26000, 0)
}

// postarEntrada pede a entrada no checkout. Sem corpo e sem Content-Type de
// propósito: a rota não lê corpo nenhum — o dono vem da Sessão, e o cookie é
// `SameSite=Lax`, que é o que dispensa o corpo (o precedente é `DELETE
// /api/v1/carrinho/itens`).
func postarEntrada(t *testing.T, rotas http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return comCorpo(t, rotas, http.MethodPost, "/api/v1/checkout/entrada", "", "", cookie)
}
