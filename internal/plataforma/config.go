// Package plataforma reúne o que é transversal a todos os módulos:
// configuração, log, correlação e migrações (AD-13, AD-15).
package plataforma

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config é o único lugar onde limiar do sistema mora (AD-13, NFR-16).
// Literal de limiar fora daqui é defeito.
type Config struct {
	HTTPAddr    string
	PostgresDSN string
	RedisURL    string

	AuthTentativasMax   int
	AuthBloqueioDuracao time.Duration
	SessaoExpiracao     time.Duration
	SenhaTokenValidade  time.Duration

	BuscaTermoMax    int
	PaginaTamanho    int
	PaginaTamanhoMax int

	CarrinhoUnidadesMax  int
	FreteIsencaoCentavos int64

	ProvedorAprovadoAteCentavos int64
	ProvedorRecusadoAteCentavos int64
	PagamentoTentativasMax      int
	PagamentoTentativaExpiracao time.Duration

	EntregaIntervalo      time.Duration
	EntregaSimulacaoAtiva bool

	ProdutoNomeMax      int
	ProdutoDescricaoMax int

	VarreduraIntervalo time.Duration
	ConfirmacaoAtraso  time.Duration
	WebhookBaseURL     string
}

// CarregarConfig lê o ambiente uma única vez, no arranque. Os padrões são os de
// produção (§7.1 do PRD); o `.env` versionado é quem traz a demonstração.
func CarregarConfig() (Config, error) {
	l := &leitor{}
	c := carregar(l)
	if len(l.erros) > 0 {
		return Config{}, fmt.Errorf("configuração inválida: %s", strings.Join(l.erros, "; "))
	}
	return c, nil
}

// carregar existe separada para que o teste do `.env` possa perguntar ao
// leitor exatamente quais variáveis o sistema consome.
func carregar(l *leitor) Config {
	c := Config{
		HTTPAddr:    l.texto("AZAMON_HTTP_ADDR", ":8080"),
		PostgresDSN: l.obrigatorio("AZAMON_POSTGRES_DSN"),
		RedisURL:    l.obrigatorio("AZAMON_REDIS_URL"),

		AuthTentativasMax:   l.inteiro("AZAMON_AUTH_TENTATIVAS_MAX", 5),
		AuthBloqueioDuracao: l.duracao("AZAMON_AUTH_BLOQUEIO_DURACAO", 15*time.Minute),
		SessaoExpiracao:     l.duracao("AZAMON_SESSAO_EXPIRACAO", 168*time.Hour),
		SenhaTokenValidade:  l.duracao("AZAMON_SENHA_TOKEN_VALIDADE", 30*time.Minute),

		BuscaTermoMax:    l.inteiro("AZAMON_BUSCA_TERMO_MAX", 100),
		PaginaTamanho:    l.inteiro("AZAMON_PAGINA_TAMANHO", 20),
		PaginaTamanhoMax: l.inteiro("AZAMON_PAGINA_TAMANHO_MAX", 60),

		CarrinhoUnidadesMax:  l.inteiro("AZAMON_CARRINHO_UNIDADES_MAX", 10),
		FreteIsencaoCentavos: l.centavos("AZAMON_FRETE_ISENCAO_CENTAVOS", 29900),

		ProvedorAprovadoAteCentavos: l.centavos("AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS", 89),
		ProvedorRecusadoAteCentavos: l.centavos("AZAMON_PROVEDOR_RECUSADO_ATE_CENTAVOS", 94),
		PagamentoTentativasMax:      l.inteiro("AZAMON_PAGAMENTO_TENTATIVAS_MAX", 3),
		PagamentoTentativaExpiracao: l.duracao("AZAMON_PAGAMENTO_TENTATIVA_EXPIRACAO", 15*time.Minute),

		EntregaIntervalo:      l.duracao("AZAMON_ENTREGA_INTERVALO", 24*time.Hour),
		EntregaSimulacaoAtiva: l.booleano("AZAMON_ENTREGA_SIMULACAO_ATIVA", true),

		ProdutoNomeMax:      l.inteiro("AZAMON_PRODUTO_NOME_MAX", 200),
		ProdutoDescricaoMax: l.inteiro("AZAMON_PRODUTO_DESCRICAO_MAX", 4000),

		VarreduraIntervalo: l.duracao("AZAMON_VARREDURA_INTERVALO", time.Second),
		ConfirmacaoAtraso:  l.duracao("AZAMON_CONFIRMACAO_ATRASO", 5*time.Second),
		WebhookBaseURL:     l.texto("AZAMON_WEBHOOK_BASE_URL", "http://localhost:8080"),
	}
	// Faixa, não só formato: um `-5` ou um `0s` que passa aqui só aparece como
	// comportamento absurdo lá na frente, e a configuração é fronteira de
	// confiança como qualquer entrada.
	if c.PaginaTamanho > c.PaginaTamanhoMax {
		l.erros = append(l.erros, "AZAMON_PAGINA_TAMANHO não pode passar de AZAMON_PAGINA_TAMANHO_MAX")
	}
	if c.ProvedorAprovadoAteCentavos >= c.ProvedorRecusadoAteCentavos {
		l.erros = append(l.erros, "AZAMON_PROVEDOR_APROVADO_ATE_CENTAVOS tem de vir antes de AZAMON_PROVEDOR_RECUSADO_ATE_CENTAVOS")
	}
	return c
}

type leitor struct {
	erros []string
	nomes []string
}

// valor é o único ponto que toca o ambiente: registrar aqui o nome lido é o
// que permite ao teste comparar o `.env` com o que o sistema de fato consome.
func (l *leitor) valor(nome string) string {
	l.nomes = append(l.nomes, nome)
	return os.Getenv(nome)
}

func (l *leitor) texto(nome, padrao string) string {
	if v := l.valor(nome); v != "" {
		return v
	}
	return padrao
}

// obrigatorio não tem padrão porque credencial não mora no código (AD-13).
func (l *leitor) obrigatorio(nome string) string {
	v := l.valor(nome)
	if v == "" {
		l.erros = append(l.erros, nome+" é obrigatória")
	}
	return v
}

// Todo limiar do sistema é positivo — página de zero item, teto de zero
// unidade e varredura de intervalo zero são configuração que sobe e só falha
// como comportamento absurdo. A checagem mora aqui, e não em vinte lugares.
func (l *leitor) inteiro(nome string, padrao int) int {
	v := l.valor(nome)
	if v == "" {
		return padrao
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		l.erros = append(l.erros, fmt.Sprintf("%s=%q não é inteiro", nome, v))
		return padrao
	}
	if n <= 0 {
		l.erros = append(l.erros, fmt.Sprintf("%s=%d tem de ser positivo", nome, n))
		return padrao
	}
	return n
}

func (l *leitor) centavos(nome string, padrao int64) int64 {
	v := l.valor(nome)
	if v == "" {
		return padrao
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		l.erros = append(l.erros, fmt.Sprintf("%s=%q não é inteiro de centavos", nome, v))
		return padrao
	}
	if n <= 0 {
		l.erros = append(l.erros, fmt.Sprintf("%s=%d tem de ser positivo", nome, n))
		return padrao
	}
	return n
}

func (l *leitor) duracao(nome string, padrao time.Duration) time.Duration {
	v := l.valor(nome)
	if v == "" {
		return padrao
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		l.erros = append(l.erros, fmt.Sprintf("%s=%q não é duração", nome, v))
		return padrao
	}
	if d <= 0 {
		l.erros = append(l.erros, fmt.Sprintf("%s=%s tem de ser positiva", nome, d))
		return padrao
	}
	return d
}

func (l *leitor) booleano(nome string, padrao bool) bool {
	v := l.valor(nome)
	if v == "" {
		return padrao
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		l.erros = append(l.erros, fmt.Sprintf("%s=%q não é booleano", nome, v))
		return padrao
	}
	return b
}
