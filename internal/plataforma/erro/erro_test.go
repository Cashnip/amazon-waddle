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
