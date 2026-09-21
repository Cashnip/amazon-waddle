package erro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cashnip/amazon-waddle/internal/carrinho"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// errTeste faz as vezes do sentinela que as próximas estórias vão registrar.
var errTeste = errors.New("Restam 2 unidades de Carregador USB-C.")

var errOutro = errors.New("Vendedor desativado.")

// registrar acrescenta traduções e devolve o registro ao estado original no
// fim do teste — estado global de pacote não vaza de um teste para o outro.
func registrar(t *testing.T, ts ...traducao) {
	t.Helper()
	original := registro
	registro = append(append([]traducao{}, registro...), ts...)
	t.Cleanup(func() { registro = original })
}

func TestErroRegistradoSaiNoEnvelope(t *testing.T) {
	registrar(t, traducao{errTeste, http.StatusConflict, "ESTOQUE_INSUFICIENTE"})
	dados := map[string]any{"disponivel": 2, "solicitado": 5}
	resp := escrever(t, fmt.Errorf("reservar: %w", errTeste), dados)

	if resp.Code != http.StatusConflict {
		t.Errorf("status = %d, quero 409", resp.Code)
	}
	corpo := decodificar(t, resp)
	if corpo["codigo"] != "ESTOQUE_INSUFICIENTE" {
		t.Errorf("codigo = %v", corpo["codigo"])
	}
	if corpo["mensagem"] != errTeste.Error() {
		t.Errorf("mensagem = %v, quero a do sentinela sem o contexto do embrulho", corpo["mensagem"])
	}
	if corpo["correlacao"] != "abc" {
		t.Errorf("correlacao = %v, quero abc", corpo["correlacao"])
	}
	d, ok := corpo["dados"].(map[string]any)
	if !ok || d["disponivel"] != float64(2) {
		t.Errorf("dados = %v", corpo["dados"])
	}
}

func TestErroDesconhecidoNaoVazaDetalhe(t *testing.T) {
	resp := escrever(t, errors.New("pq: relação inexistente em pedido.item"), map[string]any{"segredo": 1})

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, quero 500", resp.Code)
	}
	corpo := decodificar(t, resp)
	if corpo["codigo"] != CodigoInterno {
		t.Errorf("codigo = %v, quero %v", corpo["codigo"], CodigoInterno)
	}
	if corpo["mensagem"] != mensagemInterna {
		t.Errorf("mensagem = %v, quero a genérica", corpo["mensagem"])
	}
	if corpo["dados"] != nil {
		t.Errorf("dados = %v, erro desconhecido não carrega dados", corpo["dados"])
	}
	if strings.Contains(resp.Body.String(), "relação inexistente") {
		t.Error("detalhe do erro vazou no corpo")
	}
}

// A tradução não pode variar entre execuções: com dois sentinelas casando com
// o mesmo erro embrulhado, ganha sempre o primeiro registrado.
func TestTraducaoEhDeterministica(t *testing.T) {
	registrar(t,
		traducao{errTeste, http.StatusConflict, "ESTOQUE_INSUFICIENTE"},
		traducao{errOutro, http.StatusForbidden, "VENDEDOR_INATIVO"},
	)
	embrulhado := fmt.Errorf("reservar: %w e %w", errTeste, errOutro)

	for i := range 20 {
		resp := escrever(t, embrulhado, nil)
		if resp.Code != http.StatusConflict {
			t.Fatalf("execução %d: status = %d, quero 409 em todas", i, resp.Code)
		}
	}
}

// Escrever(nil) é defeito de quem chamou, e vira 500 — nunca pânico.
func TestErroNiloNaoEntraEmPanico(t *testing.T) {
	resp := escrever(t, nil, nil)
	if resp.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, quero 500", resp.Code)
	}
	if corpo := decodificar(t, resp); corpo["codigo"] != CodigoInterno {
		t.Errorf("codigo = %v, quero %v", corpo["codigo"], CodigoInterno)
	}
}

// A rota inexistente é o primeiro sentinela do registro (AD-14).
func TestRotaInexistenteTem404(t *testing.T) {
	resp := escrever(t, ErrNaoEncontrado, nil)
	if resp.Code != http.StatusNotFound {
		t.Errorf("status = %d, quero 404", resp.Code)
	}
	if corpo := decodificar(t, resp); corpo["codigo"] != "NAO_ENCONTRADO" {
		t.Errorf("codigo = %v", corpo["codigo"])
	}
}

// O 429 do bloqueio por tentativas não passa pelo registro: a mensagem nomeia
// o limiar da Config, e por isso é aqui que os minutos são contados. O caso de
// 30 s prende o arredondamento para cima — truncado, ele mandaria o Comprador
// tentar de novo "em 0 minutos".
func TestBloqueioNomeiaOsMinutos(t *testing.T) {
	casos := []struct {
		prazo    time.Duration
		mensagem string
	}{
		{15 * time.Minute, "Muitas tentativas. Tente novamente em 15 minutos."},
		{time.Minute, "Muitas tentativas. Tente novamente em 1 minuto."},
		{30 * time.Second, "Muitas tentativas. Tente novamente em 1 minuto."},
		{90 * time.Second, "Muitas tentativas. Tente novamente em 2 minutos."},
	}
	for _, caso := range casos {
		resp := httptest.NewRecorder()
		EscreverBloqueio(plataforma.ComCorrelacao(context.Background(), "abc"), resp, caso.prazo)

		if resp.Code != http.StatusTooManyRequests {
			t.Errorf("%v: status = %d, quero 429", caso.prazo, resp.Code)
		}
		corpo := decodificar(t, resp)
		if corpo["codigo"] != "MUITAS_TENTATIVAS" {
			t.Errorf("%v: codigo = %v", caso.prazo, corpo["codigo"])
		}
		if corpo["mensagem"] != caso.mensagem {
			t.Errorf("%v: mensagem = %v, quero %q", caso.prazo, corpo["mensagem"], caso.mensagem)
		}
	}
}

// O 409 do Carrinho (4.2) nomeia o Produto e o disponível, e o zero não vira
// "Restam 0": o Produto está indisponível. Os números viajam também em `dados`.
func TestEstoqueInsuficienteNomeiaODisponivel(t *testing.T) {
	casos := []struct {
		disponivel int
		mensagem   string
	}{
		{0, "Chaleira está indisponível."},
		{1, "Resta 1 unidade de Chaleira."},
		{3, "Restam 3 unidades de Chaleira."},
		{1500, "Restam 1.500 unidades de Chaleira."},
	}
	for _, caso := range casos {
		resp := httptest.NewRecorder()
		EscreverEstoqueInsuficiente(plataforma.ComCorrelacao(context.Background(), "abc"), resp, "p-1", "Chaleira", caso.disponivel, 4000)

		if resp.Code != http.StatusConflict {
			t.Errorf("%d: status = %d, quero 409", caso.disponivel, resp.Code)
		}
		corpo := decodificar(t, resp)
		if corpo["codigo"] != "ESTOQUE_INSUFICIENTE" {
			t.Errorf("%d: codigo = %v", caso.disponivel, corpo["codigo"])
		}
		if corpo["mensagem"] != caso.mensagem {
			t.Errorf("%d: mensagem = %v, quero %q", caso.disponivel, corpo["mensagem"], caso.mensagem)
		}
		dados, _ := corpo["dados"].(map[string]any)
		if dados["produto_id"] != "p-1" || dados["disponivel"] != float64(caso.disponivel) || dados["solicitado"] != float64(4000) {
			t.Errorf("%d: dados = %v", caso.disponivel, corpo["dados"])
		}
	}
}

// As três recusas da máquina de estados saem com três códigos, nunca um só
// (AD-3), e a transição inválida leva os destinos legais em
// `dados.permitidas` sem que o handler precise extraí-los. Um Status sem saída
// serializa em `[]`, e não em `null`.
func TestRecusasDaMaquinaDeEstados(t *testing.T) {
	casos := []struct {
		err        error
		codigo     string
		permitidas []any
	}{
		{fmt.Errorf("avançar: %w", pedido.ErrEstadoJaAvancado), "ESTADO_JA_AVANCADO", nil},
		{pedido.ErrForaDaJanelaDeCancelamento, "FORA_DA_JANELA_DE_CANCELAMENTO", nil},
		{pedido.TransicaoInvalida{De: pedido.StatusPago, Para: pedido.StatusEntregue,
			Permitidas: []pedido.Status{pedido.StatusSeparando}}, "TRANSICAO_INVALIDA", []any{"SEPARANDO"}},
		{pedido.TransicaoInvalida{De: pedido.StatusCancelado, Para: pedido.StatusSeparando,
			Permitidas: []pedido.Status{}}, "TRANSICAO_INVALIDA", []any{}},
		// Montada à mão, sem Permitidas: sai `[]`, nunca `null`.
		{pedido.TransicaoInvalida{De: pedido.StatusCancelado, Para: pedido.StatusSeparando}, "TRANSICAO_INVALIDA", []any{}},
		// Embrulhada: as permitidas saem do mesmo jeito.
		{fmt.Errorf("avançar: %w", pedido.TransicaoInvalida{De: pedido.StatusPago, Para: pedido.StatusEntregue,
			Permitidas: []pedido.Status{pedido.StatusSeparando}}), "TRANSICAO_INVALIDA", []any{"SEPARANDO"}},
	}
	for _, caso := range casos {
		resp := escrever(t, caso.err, nil)
		if resp.Code != http.StatusConflict {
			t.Errorf("%s: status = %d, quero 409", caso.codigo, resp.Code)
		}
		corpo := decodificar(t, resp)
		if corpo["codigo"] != caso.codigo {
			t.Errorf("codigo = %v, quero %s", corpo["codigo"], caso.codigo)
		}
		if caso.permitidas == nil {
			if corpo["dados"] != nil {
				t.Errorf("%s: dados = %v, quero null", caso.codigo, corpo["dados"])
			}
			continue
		}
		dados, _ := corpo["dados"].(map[string]any)
		permitidas, ok := dados["permitidas"].([]any)
		if !ok || fmt.Sprint(permitidas) != fmt.Sprint(caso.permitidas) {
			t.Errorf("%s: dados = %v, quero permitidas %v", caso.codigo, corpo["dados"], caso.permitidas)
		}
	}
}

// Quem já passa `dados` não perde as permitidas: elas entram no mesmo mapa.
func TestPermitidasSomamAosDadosDeQuemChama(t *testing.T) {
	err := pedido.TransicaoInvalida{De: pedido.StatusPago, Para: pedido.StatusEntregue,
		Permitidas: []pedido.Status{pedido.StatusSeparando}}
	corpo := decodificar(t, escrever(t, err, map[string]any{"pedido": "AZ-2026-000001"}))
	dados, _ := corpo["dados"].(map[string]any)
	if dados["pedido"] != "AZ-2026-000001" || fmt.Sprint(dados["permitidas"]) != "[SEPARANDO]" {
		t.Errorf("dados = %v; quero o campo de quem chamou e as permitidas", corpo["dados"])
	}
}

func escrever(t *testing.T, err error, dados any) *httptest.ResponseRecorder {
	t.Helper()
	resp := httptest.NewRecorder()
	Escrever(plataforma.ComCorrelacao(context.Background(), "abc"), resp, err, dados)
	return resp
}

func decodificar(t *testing.T, resp *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var env struct {
		Erro map[string]any `json:"erro"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("corpo não é o envelope: %v", err)
	}
	if env.Erro == nil {
		t.Fatal("corpo sem a chave \"erro\"")
	}
	return env.Erro
}

// TestRecusasDaCriacaoDoPedido: as quatro recusas da 5.6 estão no registro,
// todas 409 com o código que a tela lê — inclusive embrulhadas com %w, que é
// como `carrinho.Esvaziar` e `ConfirmarPrecoVisto` devolvem ErrCarrinhoMudou.
func TestRecusasDaCriacaoDoPedido(t *testing.T) {
	for _, caso := range []struct {
		err    error
		codigo string
	}{
		{pedido.ErrTotalDivergente, "TOTAL_DIVERGENTE"},
		{pedido.ErrCarrinhoVazio, "CARRINHO_VAZIO"},
		{pedido.ErrChaveReutilizada, "CHAVE_REUTILIZADA"},
		{carrinho.ErrCarrinhoMudou, "CARRINHO_MUDOU"},
		{fmt.Errorf("esvaziar o Carrinho: 1 Itens apagados de 2 lidos: %w", carrinho.ErrCarrinhoMudou), "CARRINHO_MUDOU"},
	} {
		resp := escrever(t, caso.err, nil)
		if resp.Code != http.StatusConflict {
			t.Errorf("%v: status = %d, quero 409", caso.err, resp.Code)
		}
		corpo := decodificar(t, resp)
		if corpo["codigo"] != caso.codigo {
			t.Errorf("%v: codigo = %v, quero %s", caso.err, corpo["codigo"], caso.codigo)
		}
		// A mensagem é a do sentinela, nunca a do embrulho (AD-14).
		if strings.Contains(fmt.Sprint(corpo["mensagem"]), "lidos") {
			t.Errorf("%v: a mensagem levou o contexto do embrulho: %v", caso.err, corpo["mensagem"])
		}
	}
}
