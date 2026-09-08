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

// Sem DSN não há como migrar: credencial não tem padrão no código.
func TestConfigSemDSNFalha(t *testing.T) {
	t.Setenv("AZAMON_POSTGRES_DSN", "")
	t.Setenv("AZAMON_REDIS_URL", "")

	if _, err := CarregarConfig(); err == nil {
		t.Fatal("DSN ausente devia falhar")
	}
}
