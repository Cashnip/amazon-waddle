package plataforma

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// O modo demonstração morre calado: um typo numa chave do `.env` faz
// CarregarConfig devolver o padrão de produção — sem erro, sem log, com a suíte
// verde — e a entrega passa a levar 24h em vez de 30s na frente da banca.
// Estes dois testes são o único mecanismo que impede isso.

func TestEnvTemExatamenteAsChavesQueOSistemaConsome(t *testing.T) {
	doArquivo := lerEnv(t)

	l := &leitor{}
	carregar(l)
	consumidas := l.nomes

	for _, nome := range consumidas {
		if _, ok := doArquivo[nome]; !ok {
			t.Errorf("%s é lida pelo sistema e não está no .env — a demonstração cairia no padrão de produção", nome)
		}
	}
	for nome := range doArquivo {
		if !slices.Contains(consumidas, nome) {
			t.Errorf("%s está no .env e ninguém lê — chave órfã, provavelmente um typo", nome)
		}
	}
}

func TestEnvProduzOsValoresDeDemonstracao(t *testing.T) {
	for nome, valor := range lerEnv(t) {
		t.Setenv(nome, valor)
	}

	c, err := CarregarConfig()
	if err != nil {
		t.Fatalf("o .env versionado tem de carregar limpo: %v", err)
	}
	casos := []struct {
		nome string
		tem  any
		quer any
	}{
		{"AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO", c.PagamentoTentativaExpiracao, 60 * time.Second},
		{"AZAMON_ENTREGA_INTERVALO", c.EntregaIntervalo, 30 * time.Second},
		{"AZAMON_VARREDURA_INTERVALO", c.VarreduraIntervalo, time.Second},
		{"AZAMON_CONFIRMACAO_ATRASO", c.ConfirmacaoAtraso, 5 * time.Second},
		{"AZAMON_WEBHOOK_BASE_URL", c.WebhookBaseURL, "http://azamon:8080"},
	}
	for _, caso := range casos {
		if caso.tem != caso.quer {
			t.Errorf("%s = %v, quero %v em demonstração", caso.nome, caso.tem, caso.quer)
		}
	}
}

// lerEnv devolve as chaves AZAMON_* do `.env` versionado na raiz do repositório.
func lerEnv(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", ".env"))
	if err != nil {
		t.Fatalf("abrir o .env da raiz: %v", err)
	}
	defer f.Close()

	valores := map[string]string{}
	linhas := bufio.NewScanner(f)
	for linhas.Scan() {
		linha := strings.TrimSpace(linhas.Text())
		if !strings.HasPrefix(linha, "AZAMON_") {
			continue
		}
		nome, valor, ok := strings.Cut(linha, "=")
		if !ok {
			t.Fatalf("linha sem '=' no .env: %q", linha)
		}
		valores[nome] = valor
	}
	if err := linhas.Err(); err != nil {
		t.Fatalf("ler o .env: %v", err)
	}
	if len(valores) == 0 {
		t.Fatal("o .env não tem nenhuma variável AZAMON_")
	}
	return valores
}
