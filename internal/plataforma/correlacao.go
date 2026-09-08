package plataforma

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// CabecalhoCorrelacao viaja na requisição e volta na resposta — a EXPERIENCE.md
// exige que o Toast de erro do servidor exiba a correlação (AD-15).
const CabecalhoCorrelacao = "X-Correlation-Id"

// SemCorrelacao é o valor das linhas fora de uma requisição (arranque, varredura).
const SemCorrelacao = "arranque"

// tamanhoMaxCorrelacao limita o que vem do cliente: a correlação é refletida
// no cabeçalho da resposta e escrita em toda linha de log, então é fronteira
// de confiança. Não é limiar do NFR-16 — é higiene de entrada.
const tamanhoMaxCorrelacao = 64

type chaveCorrelacao struct{}

// ComCorrelacao devolve um contexto que carrega o identificador de correlação.
func ComCorrelacao(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, chaveCorrelacao{}, id)
}

// CorrelacaoDe nunca devolve vazio: toda linha de log tem correlação (AD-15).
func CorrelacaoDe(ctx context.Context) string {
	if ctx != nil {
		if id, ok := ctx.Value(chaveCorrelacao{}).(string); ok && id != "" {
			return id
		}
	}
	return SemCorrelacao
}

// Correlacao é o middleware do AD-15: aceita o X-Correlation-Id de quem chamou
// ou gera um, põe no contexto e devolve no cabeçalho da resposta.
func Correlacao(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(CabecalhoCorrelacao)
		if !correlacaoValida(id) {
			id = novoIdentificador()
		}
		w.Header().Set(CabecalhoCorrelacao, id)
		prox.ServeHTTP(w, r.WithContext(ComCorrelacao(r.Context(), id)))
	})
}

// correlacaoValida recusa o que não serve para ser ecoado nem logado: vazio,
// longo demais, ou com qualquer coisa fora de [A-Za-z0-9._-] — quebra de linha
// inclusive, que injetaria uma linha falsa no log.
func correlacaoValida(id string) bool {
	if id == "" || len(id) > tamanhoMaxCorrelacao {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}

// novoIdentificador é um UUID v4 escrito à mão. A biblioteca padrão não traz
// gerador, e a tabela Stack não lista nenhuma dependência de UUID — o
// github.com/google/uuid que aparece no go.sum é indireto do goose, e depender
// dele daqui tornaria direto o que hoje é detalhe de outra biblioteca.
func novoIdentificador() string {
	var b [16]byte
	rand.Read(b[:]) // crypto/rand.Read nunca falha desde o Go 1.24.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
