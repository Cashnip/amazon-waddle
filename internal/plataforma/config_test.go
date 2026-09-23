package plataforma

import (
	"testing"
	"time"
)

// Sem AZAMON_* de demonstração, valem os padrões de produção do §7.1 —
// é isso que faz o `.env` versionado ser o único dono do modo demonstração.
func TestConfigUsaPadroesDeProducao(t *testing.T) {
	t.Setenv("AZAMON_POSTGRES_DSN", "postgres://azamon@postgres:5432/azamon")
	t.Setenv("AZAMON_REDIS_URL", "redis://redis:6379/0")
	// Obrigatória como as duas acima: credencial não tem padrão no código.
	t.Setenv("AZAMON_WEBHOOK_SEGREDO", "segredo-de-teste")

	c, err := CarregarConfig()
	if err != nil {
		t.Fatalf("CarregarConfig: %v", err)
	}
	if c.PagamentoTentativaExpiracao != 15*time.Minute {
		t.Errorf("expiração da Tentativa = %v, quero 15m em produção", c.PagamentoTentativaExpiracao)
	}
	if c.EntregaIntervalo != 24*time.Hour {
		t.Errorf("intervalo da simulação = %v, quero 24h em produção", c.EntregaIntervalo)
	}
	if c.FreteIsencaoCentavos != 29900 {
		t.Errorf("isenção de Frete = %d centavos", c.FreteIsencaoCentavos)
	}
}

// Valor malformado derruba o arranque em vez de virar padrão silencioso.
func TestConfigInvalidaFalha(t *testing.T) {
	t.Setenv("AZAMON_POSTGRES_DSN", "postgres://azamon@postgres:5432/azamon")
	t.Setenv("AZAMON_REDIS_URL", "redis://redis:6379/0")
	t.Setenv("AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO", "60 segundos")

	if _, err := CarregarConfig(); err == nil {
		t.Fatal("duração malformada devia falhar")
	}
}

// Faixa cruzada: cada limiar passa sozinho pelo leitor.inteiro — os dois são
// positivos —, e o que não fecha é a relação entre eles. Sem esta checagem o
// par invertido sobe calado e o cadastro recusa toda senha, primeiro por curta
// demais e depois por longa demais.
func TestConfigComSenhaMinAcimaDaMaxFalha(t *testing.T) {
	t.Setenv("AZAMON_POSTGRES_DSN", "postgres://azamon@postgres:5432/azamon")
	t.Setenv("AZAMON_REDIS_URL", "redis://redis:6379/0")
	t.Setenv("AZAMON_WEBHOOK_SEGREDO", "segredo-de-teste")
	t.Setenv("AZAMON_SENHA_MIN", "200")

	if _, err := CarregarConfig(); err == nil {
		t.Fatal("AZAMON_SENHA_MIN acima de AZAMON_SENHA_MAX devia falhar")
	}
}

// O outro par cruzado, e o mais mudo de todos: com o atraso da confirmação
// alcançando o prazo da expiração, a janela `criada_em <= ate AND criada_em >
// desde` nunca casa — confirmação nenhuma é emitida, todo Pedido expira por
// TEMPO_ESGOTADO, e não há erro nem log em lugar nenhum. O arranque é a única
// hora em que dá para ver.
func TestConfigComAtrasoAlcancandoAExpiracaoFalha(t *testing.T) {
	t.Setenv("AZAMON_POSTGRES_DSN", "postgres://azamon@postgres:5432/azamon")
	t.Setenv("AZAMON_REDIS_URL", "redis://redis:6379/0")
	t.Setenv("AZAMON_WEBHOOK_SEGREDO", "segredo-de-teste")
	t.Setenv("AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO", "60s")
	t.Setenv("AZAMON_CONFIRMACAO_ATRASO", "60s")

	if _, err := CarregarConfig(); err == nil {
		t.Fatal("AZAMON_CONFIRMACAO_ATRASO igual à expiração devia falhar")
	}
}

// Sem DSN não há como migrar: credencial não tem padrão no código.
func TestConfigSemDSNFalha(t *testing.T) {
	t.Setenv("AZAMON_POSTGRES_DSN", "")
	t.Setenv("AZAMON_REDIS_URL", "")

	if _, err := CarregarConfig(); err == nil {
		t.Fatal("DSN ausente devia falhar")
	}
}
