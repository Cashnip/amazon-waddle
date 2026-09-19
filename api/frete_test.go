package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/pedido"
)

// freteDoCarrinho é a matriz da 5.3 contra o banco de verdade: as faixas são as
// da migração, e o limiar é o freteIsencaoDeTeste (R$ 250,00) do ambiente. A
// regra pura, linha a linha, está em internal/pedido/frete_test.go.
func freteDoCarrinho(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	if resp := pegarFrete(t, rotas, "qualquer", nil); resp.Code != http.StatusUnauthorized {
		t.Errorf("sem Sessão = %d (%s), quero 401", resp.Code, resp.Body.String())
	}

	// Um Produto próprio, com preço de ,00: o subtotal é conhecido, e a conta
	// fecha sem depender do que os outros subtestes fizeram ao semeado.
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja do Frete"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Categoria do Frete"}`, admin), http.StatusCreated)
	b, _ := json.Marshal(map[string]any{
		"nome": "Luminária do Frete", "descricao": "", "preco_centavos": 18900, "imagem_url": "",
		"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": 50, "ativo": true,
	})
	produto := idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(b), admin), http.StatusCreated)

	cookie := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Fábio Frete","email":"fabio.frete@exemplo.br","senha":"senha-do-fabio-1"}`), http.StatusCreated)
	enderecoEm := func(cep, uf string) string {
		corpo := strings.NewReplacer(`"01310-100"`, `"`+cep+`"`, `"uf":"SP"`, `"uf":"`+uf+`"`).Replace(corpoEnderecoValido)
		return idDe(t, postarEndereco(t, rotas, corpo, cookie), http.StatusCreated)
	}
	sp, ba, semFaixa := enderecoEm("01310-100", "SP"), enderecoEm("40010-000", "BA"), enderecoEm("00000-001", "SP")

	if resp := pegarFrete(t, rotas, "", cookie); resp.Code != http.StatusBadRequest {
		t.Errorf("sem endereco_id = %d (%s), quero 400", resp.Code, resp.Body.String())
	} else if codigo := decodificar(t, resp)["codigo"]; codigo != "ENTRADA_INVALIDA" {
		t.Errorf("sem endereco_id: codigo = %v, quero ENTRADA_INVALIDA", codigo)
	}

	// Carrinho vazio: a região cobra, e o total é o próprio Frete.
	querCotacao(t, "Carrinho vazio", pegarFrete(t, rotas, sp, cookie), "Sudeste", 0, 1500)

	if resp := postarItem(t, rotas, corpoItem(produto, "1"), cookie); resp.Code != http.StatusCreated {
		t.Fatalf("adicionar = %d (%s)", resp.Code, resp.Body.String())
	}
	// Trocar o Endereço recalcula o Frete (FR-20): o mesmo Carrinho, duas regiões.
	querCotacao(t, "SP", pegarFrete(t, rotas, sp, cookie), "Sudeste", 18900, 1500)
	querCotacao(t, "BA", pegarFrete(t, rotas, ba, cookie), "Nordeste", 18900, 3000)
	querCotacao(t, "SP de novo", pegarFrete(t, rotas, sp, cookie), "Sudeste", 18900, 1500)
	// Fora de toda faixa: a região padrão, não erro (FR-21).
	querCotacao(t, "fora de faixa", pegarFrete(t, rotas, semFaixa, cookie), "Região padrão", 18900, 4000)

	// Um CEP por linha da migração, contra o banco de verdade: a tabela do
	// teste puro é cópia, e é aqui que a cópia e a migração se encontram.
	for _, c := range []struct {
		cep, uf, regiao string
		frete           float64
	}{
		{"66010-000", "PA", "Norte", 3500},
		{"70040-000", "DF", "Centro-Oeste", 2500},
		{"76801-000", "RO", "Norte", 3500},
		{"77001-000", "TO", "Norte", 3500},
		{"78005-000", "MT", "Centro-Oeste", 2500},
		{"90010-000", "RS", "Sul", 2000},
	} {
		querCotacao(t, c.uf, pegarFrete(t, rotas, enderecoEm(c.cep, c.uf), cookie), c.regiao, 18900, c.frete)
	}

	// Subtotal entre o limiar do ambiente (R$ 250,00) e o padrão do §7.1
	// (R$ 299,00): só sai isento se a rota usa o limiar configurado.
	barato := func() string {
		b, _ := json.Marshal(map[string]any{
			"nome": "Pilha do Frete", "descricao": "", "preco_centavos": 7000, "imagem_url": "",
			"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": 50, "ativo": true,
		})
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(b), admin), http.StatusCreated)
	}()
	if resp := postarItem(t, rotas, corpoItem(barato, "1"), cookie); resp.Code != http.StatusCreated {
		t.Fatalf("adicionar o segundo = %d (%s)", resp.Code, resp.Body.String())
	}
	querCotacao(t, "acima do limiar configurado", pegarFrete(t, rotas, ba, cookie), "Nordeste", 25900, 0)

	// Endereço alheio, inexistente e malformado: o mesmo 404, o mesmo corpo.
	outro := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Olga Outra","email":"olga.frete@exemplo.br","senha":"senha-da-olga-1"}`), http.StatusCreated)
	var primeiro map[string]any
	for _, caso := range []struct{ nome, id string }{
		{"alheio", sp},
		{"inexistente", "0199a000-0000-7000-8000-000000000000"},
		{"malformado", "nao-e-uuid"},
	} {
		resp := pegarFrete(t, rotas, caso.id, outro)
		if resp.Code != http.StatusNotFound {
			t.Errorf("%s = %d (%s), quero 404", caso.nome, resp.Code, resp.Body.String())
			continue
		}
		corpo := decodificar(t, resp)
		delete(corpo, "correlacao")
		if primeiro == nil {
			primeiro = corpo
		} else if !igualJSON(corpo, primeiro) {
			t.Errorf("%s = %v; difere de %v", caso.nome, corpo, primeiro)
		}
	}

	// Sem a linha padrão é erro, nunca Frete zero — nem para o CEP que cai
	// numa faixa. Dentro de uma transação revertida: o banco é do pacote.
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM pedido.faixa_frete WHERE padrao`); err != nil {
		t.Fatal(err)
	}
	contaDoFabio := textoDe(t, pool, `SELECT comprador_id::text FROM identidade.endereco WHERE id = $1`, sp)[0]
	if _, err := pedido.CotarFrete(ctx, tx, contaDoFabio, sp, freteIsencaoDeTeste); !errors.Is(err, pedido.ErrSemRegiaoPadrao) {
		t.Errorf("sem região padrão = %v, quero ErrSemRegiaoPadrao", err)
	}
	// Faixa sobreposta é recusada pelo banco (23P01), e não desempatada pela
	// ordem da consulta. Por último: o erro aborta a transação.
	if _, err := tx.Exec(ctx, `INSERT INTO pedido.faixa_frete (cep_inicio, cep_fim, regiao, valor_centavos)
		VALUES ('39000000', '40000000', 'Sobreposta', 1000)`); err == nil || !strings.Contains(err.Error(), "23P01") {
		t.Errorf("faixa sobreposta = %v, quero violação de exclusão (23P01)", err)
	}
}

func querCotacao(t *testing.T, caso string, resp *httptest.ResponseRecorder, regiao string, subtotal, frete float64) {
	t.Helper()
	if resp.Code != http.StatusOK {
		t.Errorf("%s = %d (%s), quero 200", caso, resp.Code, resp.Body.String())
		return
	}
	if v := resp.Header().Get("Cache-Control"); v != "no-store" {
		t.Errorf("%s: Cache-Control = %q, quero no-store", caso, v)
	}
	c := decodificar(t, resp)
	if c["regiao"] != regiao || c["subtotal_centavos"] != subtotal || c["frete_centavos"] != frete || c["total_centavos"] != subtotal+frete {
		t.Errorf("%s = %v; quero %s, subtotal %v, Frete %v, total %v", caso, c, regiao, subtotal, frete, subtotal+frete)
	}
}

func igualJSON(a, b map[string]any) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return string(ja) == string(jb)
}

func pegarFrete(t *testing.T, rotas http.Handler, enderecoID string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	rota := "/api/v1/frete"
	if enderecoID != "" {
		rota += "?endereco_id=" + url.QueryEscape(enderecoID)
	}
	return comCorpo(t, rotas, http.MethodGet, rota, "", "", cookie)
}
