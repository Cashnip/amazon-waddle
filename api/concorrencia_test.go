package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
)

// consistenciaSobConcorrencia é o teste da NFR-7, e ele é o critério de aceite
// da 5.7: N Compradores distintos disputam a mesma última unidade ao mesmo
// tempo, e `pedidos_criados == estoque_inicial` — nunca mais, e também nunca
// menos.
//
// Não é o duplo clique da 5.6. Lá é um Comprador, uma chave de idempotência e
// um Carrinho, e o que se prova é que a chave não cria dois Pedidos; aqui cada
// requisição é uma conta, um Carrinho e uma chave própria, e o que segura a
// contagem é a trava do AD-5 sobre `catalogo.produto` mais a soma das Reservas
// ativas lida dentro dela.
//
// Roda contra o Postgres do `ambiente()` porque a transação **é** o objeto do
// teste: em memória não há trava de linha para perder.
func consistenciaSobConcorrencia(t *testing.T, rotas http.Handler, pool *pgxpool.Pool) {
	admin := cookieDe(t, postarAdmin(t, rotas, `{"email":"`+emailAdmin+`","senha":"`+senhaAdmin+`"}`), http.StatusOK)
	loja := idDe(t, vendedorCom(t, rotas, http.MethodPost, "", `{"nome":"Loja da Disputa"}`, admin), http.StatusCreated)
	categoria := idDe(t, categoriaCom(t, rotas, http.MethodPost, "", `{"nome":"Categoria da Disputa"}`, admin), http.StatusCreated)

	// Um Produto por rodada, visível e com o Estoque total que a rodada
	// disputa — nada é reaproveitado entre rodadas, senão a contagem de uma
	// dependeria do desfecho da outra.
	novoProduto := func(t *testing.T, nome string, estoque int) string {
		t.Helper()
		corpo, _ := json.Marshal(map[string]any{
			"nome": nome, "descricao": "", "preco_centavos": 1000, "imagem_url": "",
			"vendedor_id": loja, "categoria_id": categoria, "estoque_total": estoque, "ativo": true,
		})
		return idDe(t, produtoCom(t, rotas, http.MethodPost, "", string(corpo), admin), http.StatusCreated)
	}

	disputa := func(t *testing.T, apelido string, estoqueInicial, n int) {
		// A rubrica da PRD: o enunciado `pedidos_criados == estoque_inicial` só
		// vale com N maior que o Estoque inicial — com N menor o resultado
		// correto seria N, e a disputa nunca chegaria ao limite.
		if n <= estoqueInicial {
			t.Fatalf("N=%d não é maior que o Estoque inicial %d", n, estoqueInicial)
		}
		produto := novoProduto(t, "Produto da Disputa "+apelido, estoqueInicial)

		// Todo o preparo é sequencial e fica fora da disputa: o que se mede é a
		// criação do Pedido, e não o cadastro nem a cotação do Frete. Depois da
		// largada nenhuma goroutine chama `t.Fatalf`, que só vale na goroutine do
		// teste; `postarPedido` chama `t.Helper()`, e essa a documentação do
		// `testing` declara chamável de várias goroutines ao mesmo tempo.
		type participante struct {
			cookie *http.Cookie
			chave  string
			corpo  string
		}
		participantes := make([]participante, n)
		for i := range participantes {
			email := fmt.Sprintf("disputa-%s-%d@exemplo.br", apelido, i)
			cookie := cookieDe(t, postarCadastro(t, rotas, fmt.Sprintf(
				`{"nome":"Comprador %s %d","email":%q,"senha":"senha-da-disputa-1"}`, apelido, i, email)), http.StatusCreated)
			if resp := postarItem(t, rotas, corpoItem(produto, "1"), cookie); resp.Code != http.StatusCreated {
				t.Fatalf("pôr no Carrinho de %s = %d (%s)", email, resp.Code, resp.Body.String())
			}
			endereco := enderecoDeSP(t, rotas, cookie)
			participantes[i] = participante{
				cookie: cookie,
				chave:  chaveNova(t),
				corpo:  corpoPedido(endereco, totalDaRevisao(t, rotas, cookie, endereco)),
			}
		}

		pedidosAntes := contarPedidos(t, pool)

		// A largada é um canal fechado, e não o simples `go`: as N goroutines
		// já estão paradas na leitura quando ele fecha, então nenhuma sai na
		// frente por ter nascido antes.
		largada := make(chan struct{})
		respostas := make([]*httptest.ResponseRecorder, n)
		var prontas, terminadas sync.WaitGroup
		prontas.Add(n)
		terminadas.Add(n)
		for i, p := range participantes {
			go func() {
				defer terminadas.Done()
				prontas.Done()
				<-largada
				respostas[i] = postarPedido(t, rotas, p.chave, p.corpo, p.cookie)
			}()
		}
		prontas.Wait()
		close(largada)
		terminadas.Wait()

		var criados, recusados int
		for i, r := range respostas {
			switch r.Code {
			case http.StatusCreated:
				criados++
			case http.StatusConflict:
				recusados++
				// Quem perde a disputa recebe a recusa nomeada, e não um 500
				// nem o aviso genérico: é o que a tela do Pedido precisa para
				// tirar a ação de tentar de novo.
				confereErro(t, r, http.StatusConflict, "ESTOQUE_INSUFICIENTE", "")
			default:
				t.Errorf("participante %d = %d (%s), quero 201 ou 409", i, r.Code, r.Body.String())
			}
		}
		if criados != estoqueInicial {
			t.Errorf("Pedidos criados = %d, quero exatamente o Estoque inicial %d", criados, estoqueInicial)
		}
		if recusados != n-estoqueInicial {
			t.Errorf("recusas = %d, quero %d", recusados, n-estoqueInicial)
		}

		// O banco tem de contar a mesma história que as respostas: uma unidade
		// vendida por Pedido, e o Estoque fechando em zero. Se um perdedor
		// tivesse gravado alguma parte, é aqui que apareceria.
		quero := fmt.Sprintf("%d|%d", estoqueInicial, estoqueInicial)
		if vendido := textoDe(t, pool, `
			SELECT coalesce(sum(quantidade), 0)::text || '|' || count(DISTINCT pedido_id)::text
			FROM pedido.item_pedido WHERE produto_id = $1::uuid`, produto); len(vendido) != 1 || vendido[0] != quero {
			t.Errorf("unidades|Pedidos = %v, quero %s", vendido, quero)
		}

		// A sobra sai das tabelas cruas, e **não** de `catalogo.produto_visivel`:
		// a VIEW envolve a conta num `greatest(..., 0)`, então mostraria zero
		// tanto para uma unidade vendida quanto para oito — zero justamente no
		// caso que este teste existe para pegar. Negativo aqui é venda a mais.
		if sobra := textoDe(t, pool, `
			SELECT (p.estoque_total - coalesce((
				SELECT sum(r.quantidade) FROM catalogo.reserva_estoque r
				WHERE r.produto_id = p.id AND r.estado = 'ATIVA'
			), 0))::text
			FROM catalogo.produto p WHERE p.id = $1::uuid`, produto); len(sobra) != 1 || sobra[0] != "0" {
			t.Errorf("Estoque total menos Reservas ATIVAS = %v, quero 0", sobra)
		}

		// Nenhum Pedido órfão. O `INSERT` que reivindica a chave de idempotência
		// roda **antes** de `catalogo.Reservar` (5.6), então uma reversão
		// malfeita deixaria linha em `pedido.pedido` sem Item e sem Reserva —
		// invisível para as duas contagens acima, que olham pelo Produto.
		if nascidos := contarPedidos(t, pool) - pedidosAntes; nascidos != estoqueInicial {
			t.Errorf("linhas novas em pedido.pedido = %d, quero %d", nascidos, estoqueInicial)
		}
	}

	// A última unidade é o caso do enunciado da NFR-7 e do passo 5 do Roteiro
	// C; o Estoque de três prova que o invariante não é um acidente do 1 — uma
	// trava que só funcionasse para "existe ou não existe" passaria no primeiro
	// e venderia demais no segundo.
	t.Run("a última unidade: oito checkouts, um Pedido", func(t *testing.T) {
		disputa(t, "ultima", 1, 8)
	})
	t.Run("três unidades: doze checkouts, três Pedidos", func(t *testing.T) {
		disputa(t, "tres", 3, 12)
	})
	// E a trava em si, fora do caminho do contador de número — ver o comentário
	// de travaDeReservaSerializa.
	t.Run("a trava do Produto serializa duas Reservas da última unidade", func(t *testing.T) {
		travaDeReservaSerializa(t, pool, novoProduto(t, "Produto da Trava", 1))
	})
}

// travaDeReservaSerializa isola a trava do AD-5, que o teste de ponta a ponta
// acima **não** consegue observar: toda criação de Pedido passa antes pelo
// `ProximoNumeroDoAno`, um `ON CONFLICT DO UPDATE` sobre a única linha do ano,
// e essa linha serializa as criações inteiras. Com ela no caminho, o invariante
// da NFR-7 continua de pé mesmo que a trava dos Produtos suma — foi o que uma
// mutação mostrou: removido o `FOR UPDATE OF p` de `TravarProdutosParaReserva`,
// a disputa de oito checkouts continuou verde.
//
// Aqui `catalogo.Reservar` é chamada direto, em duas transações abertas à mão:
// não há contador no caminho, então o que segura a segunda é a trava do
// Produto e nada mais. Sem ela, a segunda Reserva passaria na hora e venderia
// a mesma unidade duas vezes.
func travaDeReservaSerializa(t *testing.T, pool *pgxpool.Pool, produto string) {
	ctx := context.Background()

	primeira, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir a primeira transação: %v", err)
	}
	defer primeira.Rollback(ctx)
	if err := catalogo.Reservar(ctx, primeira, uuidFalso(1), []catalogo.ItemReserva{{ProdutoID: produto, Quantidade: 1}}); err != nil {
		t.Fatalf("a primeira Reserva da última unidade: %v", err)
	}

	// A segunda transação para na trava da linha do Produto e só sai quando a
	// primeira comitar. É o bloqueio que se prova primeiro: sem a trava,
	// `desfecho` receberia `nil` antes de a espera terminar.
	desfecho := make(chan error, 1)
	go func() {
		segunda, err := pool.Begin(ctx)
		if err != nil {
			desfecho <- err
			return
		}
		defer segunda.Rollback(ctx)
		desfecho <- catalogo.Reservar(ctx, segunda, uuidFalso(2), []catalogo.ItemReserva{{ProdutoID: produto, Quantidade: 1}})
	}()

	// Meio segundo é folga larga para uma consulta que, destravada, responde em
	// microssegundos. É espera fixa, e não sondagem: o bloqueio dá para observar
	// em `pg_locks` com `granted = false`, e é para lá que isto vai se um dia
	// flacar numa bancada carregada.
	select {
	case err := <-desfecho:
		t.Fatalf("a segunda Reserva respondeu %v com a primeira transação aberta: a linha do Produto não está travada", err)
	case <-time.After(500 * time.Millisecond):
	}

	if err := primeira.Commit(ctx); err != nil {
		t.Fatalf("comitar a primeira: %v", err)
	}
	select {
	case err := <-desfecho:
		var falta catalogo.EstoqueInsuficiente
		if !errors.As(err, &falta) || falta.ProdutoID != produto || falta.Disponivel != 0 {
			t.Errorf("a segunda Reserva = %v, quero EstoqueInsuficiente do Produto com disponível 0", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a segunda Reserva não saiu depois do commit da primeira")
	}
}

// uuidFalso é um Pedido que não existe: `catalogo.reserva_estoque.pedido_id`
// não tem chave estrangeira, e o que esta prova precisa é só de dois donos
// distintos para a Reserva.
func uuidFalso(n int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", n)
}
