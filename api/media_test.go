package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Cashnip/amazon-waddle/media"
)

// A imagem do Produto sai do binário, e não da rede (AD-12): a rota devolve o
// byte embutido com o Content-Type certo. O nome vem do próprio embed, então
// este teste não diz nada sobre a semente casar com os arquivos — quem liga
// imagem_url a arquivo existente é db/schema_test.go.
func TestMediaServeOArquivoEmbutido(t *testing.T) {
	nomes, err := media.Arquivos.ReadDir(".")
	if err != nil || len(nomes) == 0 {
		t.Fatalf("nenhuma imagem embutida: %v", err)
	}
	nome := nomes[0].Name()

	resp := httptest.NewRecorder()
	rotasSemDependencia().ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/media/"+nome, nil))

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, quero 200", resp.Code)
	}
	if tipo := resp.Header().Get("Content-Type"); tipo != "image/webp" {
		t.Errorf("Content-Type = %q", tipo)
	}
	// O cabeçalho RIFF do WebP: "RIFF" nos bytes 0–3 e "WEBP" nos bytes 8–11.
	if corpo := resp.Body.Bytes(); len(corpo) < 12 || string(corpo[0:4]) != "RIFF" || string(corpo[8:12]) != "WEBP" {
		t.Errorf("corpo não é o WebP embutido: %.12q", resp.Body.String())
	}
}

// Arquivo inexistente sai no envelope do AD-14, como qualquer rota — e a
// travessia percent-codificada é só mais um arquivo inexistente: o PathValue
// entrega "../../etc/passwd" já decodificado, e o fs.ValidPath de dentro do
// ReadFile do embed é quem recusa.
func TestMediaInexistenteSaiNoEnvelope(t *testing.T) {
	for _, caminho := range []string{"nao-existe.webp", "..%2F..%2Fetc%2Fpasswd"} {
		resp := httptest.NewRecorder()
		rotasSemDependencia().ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/media/"+caminho, nil))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, quero 404", caminho, resp.Code)
		}
		var env struct {
			Erro struct {
				Codigo string `json:"codigo"`
			} `json:"erro"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
			t.Fatalf("%s: corpo não é o envelope: %v", caminho, err)
		}
		if env.Erro.Codigo != "NAO_ENCONTRADO" {
			t.Errorf("%s: codigo = %q", caminho, env.Erro.Codigo)
		}
	}
}
