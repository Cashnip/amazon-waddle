package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/pagamento"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
)

// Os dois limiares da Tentativa de Pagamento que a tela do Pedido lê da
// Config. Valores de produção, e não os de demonstração: o que se prova é que
// a resposta os usa, e não quanto valem.
const (
	expiracaoDaTentativaDeTeste = 15 * time.Minute
	tentativasMaxDeTeste        = 3
)

// detalheDoPedido é a matriz de servidor da 5.8: o que `GET /api/v1/pedidos/{id}`
// carrega para a tela derivar tudo numa chamada só (AD-18) — `expira_em` do
// histórico, a tripla, os valores e o Endereço congelados, o histórico — e que
// nenhuma chave nomeia a Reserva. Conta, Vendedor e Produto próprios: desativar
// o Produto aqui não mexe no Estoque de ninguém.
//
// Estoque 1 e Pedido de 1 unidade, de propósito: com folga de Estoque, o
// `disponivel` sairia true com ou sem a Reserva liberada, e as duas asserções
// sobre ele não teriam como falhar.
func detalheDoPedido(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	ctx := context.Background()
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	const email = "otavio.detalhe@exemplo.br"
	comprador := cookieDe(t, postarCadastro(t, rotas,
		`{"nome":"Otávio Detalhe","email":"`+email+`","senha":"senha-do-otavio-1"}`), http.StatusCreated)

	vendedor := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja do Detalhe"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Detalhe do Pedido"}`, admin), http.StatusCreated)
	const preco = 4900
	corpo, _ := json.Marshal(map[string]any{
		"nome": "Cronômetro de Pagamento", "descricao": "", "preco_centavos": preco, "imagem_url": "",
		"vendedor_id": vendedor, "categoria_id": categoria, "estoque_total": 1, "ativo": true,
	})
	produto := idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(corpo), admin), http.StatusCreated)

	lerDetalhe := func(pedidoID string) map[string]any {
		t.Helper()
		resp := pegarPedido(t, rotas, pedidoID, comprador)
		if resp.Code != http.StatusOK {
			t.Fatalf("ler o Pedido: status = %d (%s), quero 200", resp.Code, resp.Body.String())
		}
		d := decodificar(t, resp)
		for _, chave := range chavesDe(d) {
			if strings.Contains(strings.ToLower(chave), "reserv") {
				t.Errorf("a chave %q nomeia a Reserva ao Comprador", chave)
			}
		}
		return d
	}

	pedidoID := idDe(t, pedidoPeloCheckout(t, rotas, comprador, produto, 1), http.StatusCreated)

	t.Run("recém-criado: prazo do histórico, tripla, valores, Endereço e histórico", func(t *testing.T) {
		d := lerDetalhe(pedidoID)

		// O instante é o nascimento mais o prazo da Config, comparado contra o
		// histórico gravado: analisar o RFC 3339 não basta, porque devolver o
		// instante de agora mais o prazo também passaria nisso.
		quer := ultimaTransicao(t, pool, pedidoID).Add(expiracaoDaTentativaDeTeste)
		if tem := instanteDoCampo(t, d, "expira_em"); !tem.Equal(quer) {
			t.Errorf("expira_em = %v, quero %v (nascimento + prazo)", tem, quer)
		}
		if n, _ := d["tentativas_restantes"].(float64); n != tentativasMaxDeTeste-1 {
			t.Errorf("tentativas_restantes = %v, quero %d (a criação abriu uma)", d["tentativas_restantes"], tentativasMaxDeTeste-1)
		}
		subtotal, _ := d["subtotal_centavos"].(float64)
		frete, _ := d["frete_centavos"].(float64)
		total, _ := d["total_centavos"].(float64)
		if subtotal != preco || frete != freteSudeste || total != subtotal+frete {
			t.Errorf("subtotal/Frete/total = %v/%v/%v, quero %d/%d/%d", subtotal, frete, total, preco, freteSudeste, preco+freteSudeste)
		}
		if d["terminal"] != false {
			t.Errorf("terminal = %v", d["terminal"])
		}

		endereco, _ := d["endereco"].(map[string]any)
		congelado := textoDe(t, pool, `
			SELECT endereco_cep || '|' || endereco_cidade || '|' || endereco_uf
			FROM pedido.pedido WHERE id = $1::uuid`, pedidoID)
		if endereco == nil || len(congelado) != 1 ||
			fmt.Sprintf("%v|%v|%v", endereco["cep"], endereco["cidade"], endereco["uf"]) != congelado[0] {
			t.Errorf("endereco = %v, quero o congelado %v", d["endereco"], congelado)
		}

		itens, _ := d["itens"].([]any)
		if len(itens) != 1 {
			t.Fatalf("itens = %v, quero um", d["itens"])
		}
		item, _ := itens[0].(map[string]any)
		if item["produto_id"] != produto || item["nome"] != "Cronômetro de Pagamento" ||
			item["quantidade"] != float64(1) || item["preco_praticado_centavos"] != float64(preco) {
			t.Errorf("item = %v", item)
		}
		// A regra deliberada do addendum §10 (5.8): a Reserva ativa do próprio
		// Pedido conta contra ele. Ele levou a última unidade, então lê false —
		// descontar a própria Reserva faria este teste falhar, e é para isso
		// que ele existe.
		if item["disponivel"] != false {
			t.Errorf("disponivel = %v, quero false: a Reserva do próprio Pedido segura a última unidade", item["disponivel"])
		}

		historico, _ := d["historico"].([]any)
		if len(historico) != 1 {
			t.Fatalf("historico = %v, quero só o nascimento", d["historico"])
		}
		nascimento, _ := historico[0].(map[string]any)
		if de, tem := nascimento["de"]; !tem || de != nil {
			t.Errorf("de = %v (presente: %v), quero null", de, tem)
		}
		if motivo, tem := nascimento["motivo"]; !tem || motivo != nil {
			t.Errorf("motivo = %v (presente: %v), quero null", motivo, tem)
		}
		if nascimento["para"] != "AGUARDANDO_PAGAMENTO" || nascimento["ator"] != "COMPRADOR" {
			t.Errorf("nascimento = %v", nascimento)
		}
	})

	t.Run("recusado por expiração: sem prazo, motivo no histórico, Estoque de volta", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("abrir a transação: %v", err)
		}
		defer tx.Rollback(ctx)
		if err := pedido.Transicionar(ctx, tx, pedidoID, pedido.StatusAguardandoPagamento, pedido.StatusPagamentoRecusado,
			pedido.AtorVarredura, pedido.MotivoTempoEsgotado); err != nil {
			t.Fatalf("expirar: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit da expiração: %v", err)
		}

		d := lerDetalhe(pedidoID)
		if v, tem := d["expira_em"]; !tem || v != nil {
			t.Errorf("expira_em = %v (presente: %v), quero null fora de AGUARDANDO_PAGAMENTO", v, tem)
		}
		historico, _ := d["historico"].([]any)
		ultima, _ := historico[len(historico)-1].(map[string]any)
		if ultima["de"] != "AGUARDANDO_PAGAMENTO" || ultima["para"] != "PAGAMENTO_RECUSADO" || ultima["motivo"] != pedido.MotivoTempoEsgotado {
			t.Errorf("última transição = %v", ultima)
		}
		itens, _ := d["itens"].([]any)
		if item, _ := itens[0].(map[string]any); item["disponivel"] != true {
			t.Errorf("disponivel = %v, quero true: a recusa devolveu a unidade", item["disponivel"])
		}
		if n, _ := d["tentativas_restantes"].(float64); n != tentativasMaxDeTeste-1 {
			t.Errorf("tentativas_restantes = %v; a expiração não abre nem consome Tentativa", d["tentativas_restantes"])
		}
	})

	t.Run("Produto desativado depois: o Item não está disponível", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE catalogo.produto SET ativo = false WHERE id = $1::uuid`, produto); err != nil {
			t.Fatalf("desativar o Produto: %v", err)
		}
		d := lerDetalhe(pedidoID)
		itens, _ := d["itens"].([]any)
		item, _ := itens[0].(map[string]any)
		if item["disponivel"] != false {
			t.Errorf("disponivel = %v, quero false para Produto invisível", item["disponivel"])
		}
		// O que foi congelado não muda com o Produto.
		if item["nome"] != "Cronômetro de Pagamento" || item["preco_praticado_centavos"] != float64(preco) {
			t.Errorf("item = %v", item)
		}
	})

	t.Run("Pedido do esqueleto: sem Endereço; tentativas restantes nunca negativas", func(t *testing.T) {
		donos := textoDe(t, pool, `SELECT id::text FROM identidade.comprador WHERE email = $1`, email)
		if len(donos) != 1 {
			t.Fatalf("ler o Comprador: %v", donos)
		}
		esqueleto := textoDe(t, pool, `
			INSERT INTO pedido.pedido (numero, comprador_id, status, subtotal_centavos, frete_centavos, total_centavos)
			VALUES ('AZ-0000-000058', $1::uuid, 'AGUARDANDO_PAGAMENTO', 100, 0, 100)
			RETURNING id::text`, donos[0])
		if len(esqueleto) != 1 {
			t.Fatalf("criar o Pedido do esqueleto: %v", esqueleto)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO pedido.transicao_status (pedido_id, status_anterior, status_novo, ator)
			VALUES ($1::uuid, '', 'AGUARDANDO_PAGAMENTO', 'COMPRADOR')`, esqueleto[0]); err != nil {
			t.Fatalf("nascimento do Pedido do esqueleto: %v", err)
		}

		d := lerDetalhe(esqueleto[0])
		if v, tem := d["endereco"]; !tem || v != nil {
			t.Errorf("endereco = %v (presente: %v), quero null", v, tem)
		}
		// Lista vazia, e nunca null: o contrato das listas.
		if itens, ok := d["itens"].([]any); !ok || len(itens) != 0 {
			t.Errorf("itens = %#v, quero []", d["itens"])
		}
		if n, _ := d["tentativas_restantes"].(float64); n != tentativasMaxDeTeste {
			t.Errorf("tentativas_restantes = %v, quero %d sem Tentativa nenhuma", d["tentativas_restantes"], tentativasMaxDeTeste)
		}

		// As Tentativas inventadas aqui não podem sobrar para outro teste: a
		// emissão é derivada ("sem linha na inbox"), e qualquer
		// EmitirConfirmacoesDevidas posterior as emitiria.
		t.Cleanup(func() {
			if _, err := pool.Exec(ctx, `DELETE FROM pagamento.tentativa_pagamento WHERE pedido_id = $1::uuid`, esqueleto[0]); err != nil {
				t.Errorf("apagar as Tentativas do teste: %v", err)
			}
		})

		// Uma Tentativa além do teto não pode virar "-1 tentativas restantes".
		for n := 1; n <= tentativasMaxDeTeste+1; n++ {
			if _, err := pool.Exec(ctx, `
				INSERT INTO pagamento.tentativa_pagamento (pedido_id, total_centavos, id_externo, numero)
				VALUES ($1::uuid, 100, $2, $3)`, esqueleto[0], fmt.Sprintf("teste-5-8-%s-%d", esqueleto[0], n), n); err != nil {
				t.Fatalf("Tentativa %d: %v", n, err)
			}
		}
		if n, _ := lerDetalhe(esqueleto[0])["tentativas_restantes"].(float64); n != 0 {
			t.Errorf("tentativas_restantes = %v acima do teto, quero 0", n)
		}
	})

	t.Run("o Simulado decide pelos centavos só na primeira Tentativa, também na emissão", func(t *testing.T) {
		// O teste de unidade prova Decidir; este prova que o número da
		// Tentativa atravessa TentativasSemConfirmacao até ele. Total de ,95 —
		// a faixa que nunca confirma — em duas Tentativas do mesmo Pedido.
		donos := textoDe(t, pool, `SELECT id::text FROM identidade.comprador WHERE email = $1`, email)
		alvo := textoDe(t, pool, `
			INSERT INTO pedido.pedido (numero, comprador_id, status, subtotal_centavos, frete_centavos, total_centavos)
			VALUES ('AZ-0000-000059', $1::uuid, 'AGUARDANDO_PAGAMENTO', 1095, 0, 1095)
			RETURNING id::text`, donos[0])
		if len(alvo) != 1 {
			t.Fatalf("criar o Pedido: %v", alvo)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(ctx, `DELETE FROM pagamento.tentativa_pagamento WHERE pedido_id = $1::uuid`, alvo[0]); err != nil {
				t.Errorf("apagar as Tentativas do teste: %v", err)
			}
		})
		primeira, segunda := pagamento.IDExterno(alvo[0], 1), pagamento.IDExterno(alvo[0], 2)
		for n, id := range []string{primeira, segunda} {
			if _, err := pool.Exec(ctx, `
				INSERT INTO pagamento.tentativa_pagamento (pedido_id, total_centavos, id_externo, numero)
				VALUES ($1::uuid, 1095, $2, $3)`, alvo[0], id, n+1); err != nil {
				t.Fatalf("Tentativa %d: %v", n+1, err)
			}
		}

		// O transporte só anota: nada chega ao webhook, e a inbox fica como
		// estava. Outras Tentativas do banco também passam por aqui; só as
		// deste Pedido importam.
		emitidas := map[string]string{}
		enviar := func(_ context.Context, c pagamento.Confirmacao) error {
			emitidas[c.IDExterno] = c.Resultado
			return nil
		}
		simulado := pagamento.Simulado{AprovadoAteCentavos: 89, RecusadoAteCentavos: 94}
		if err := pagamento.EmitirConfirmacoesDevidas(ctx, pool, simulado, 0, enviar); err != nil {
			t.Fatalf("emitir: %v", err)
		}
		if r, tem := emitidas[primeira]; tem {
			t.Errorf("primeira Tentativa de ,95 emitiu %s; a faixa nunca confirma", r)
		}
		if r := emitidas[segunda]; r != pagamento.Aprovado {
			t.Errorf("segunda Tentativa de ,95 emitiu %q; quero %s", r, pagamento.Aprovado)
		}
	})
}

// instanteDoCampo exige que o campo seja um instante RFC 3339.
func instanteDoCampo(t *testing.T, corpo map[string]any, campo string) time.Time {
	t.Helper()
	texto, ok := corpo[campo].(string)
	if !ok {
		t.Fatalf("%s = %v; quero um instante RFC 3339", campo, corpo[campo])
	}
	instante, err := time.Parse(time.RFC3339Nano, texto)
	if err != nil {
		t.Fatalf("%s = %q não é RFC 3339: %v", campo, texto, err)
	}
	return instante
}

// chavesDe devolve todas as chaves de um JSON decodificado, em qualquer
// profundidade: é onde uma chave que nomeia a Reserva se esconderia.
func chavesDe(v any) []string {
	var chaves []string
	switch x := v.(type) {
	case map[string]any:
		for k, filho := range x {
			chaves = append(chaves, k)
			chaves = append(chaves, chavesDe(filho)...)
		}
	case []any:
		for _, filho := range x {
			chaves = append(chaves, chavesDe(filho)...)
		}
	}
	return chaves
}
