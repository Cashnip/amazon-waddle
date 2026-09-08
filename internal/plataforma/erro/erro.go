// Package erro traduz o erro sentinela do módulo em resposta HTTP — num
// arquivo só, que é o que o AD-14 exige. Módulo de domínio não conhece status.
package erro

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// ErrNaoEncontrado cobre a rota que não existe — sem ele o ServeMux
// responderia texto puro e a API teria dois contratos de erro.
var ErrNaoEncontrado = errors.New("Recurso não encontrado.")

type traducao struct {
	sentinela error
	status    int
	codigo    string
}

// registro é o mapeamento sentinela→(status, código). As estórias que criam
// sentinelas acrescentam a linha aqui, e em nenhum outro lugar. É uma fatia, e
// não um mapa, porque a ordem decide qual sentinela ganha quando dois casam com
// o mesmo erro embrulhado — e a ordem de iteração de mapa em Go é aleatória.
var registro = []traducao{
	{ErrNaoEncontrado, http.StatusNotFound, "NAO_ENCONTRADO"},
}

// CodigoInterno é o código de todo erro que não está no registro.
const CodigoInterno = "INTERNO"

const mensagemInterna = "Erro inesperado no servidor. Tente novamente."

type envelope struct {
	Erro corpo `json:"erro"`
}

type corpo struct {
	Codigo     string `json:"codigo"`
	Mensagem   string `json:"mensagem"`
	Dados      any    `json:"dados"`
	Correlacao string `json:"correlacao"`
}

// Escrever devolve o envelope do AD-14. Erro fora do registro vira 500 com
// mensagem genérica: o detalhe fica no log, nunca no corpo.
func Escrever(ctx context.Context, w http.ResponseWriter, err error, dados any) {
	status, codigo, mensagem := http.StatusInternalServerError, CodigoInterno, mensagemInterna
	conhecido := false
	for _, t := range registro {
		if err != nil && errors.Is(err, t.sentinela) {
			// A mensagem é a do sentinela, nunca a do err embrulhado: o
			// contexto que o Go acumula ("reservar: …") é detalhe de
			// desenvolvedor e não fala a Voice and Tone da UX.
			status, codigo, mensagem, conhecido = t.status, t.codigo, t.sentinela.Error(), true
			break
		}
	}
	if !conhecido {
		// Chamar Escrever sem erro é defeito de quem chamou, e vira 500 como
		// qualquer outro desconhecido — nunca um pânico dentro do handler.
		detalhe := "<nil>"
		if err != nil {
			detalhe = err.Error()
		}
		slog.ErrorContext(ctx, "erro não registrado", "erro", detalhe)
		dados = nil
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{corpo{
		Codigo:     codigo,
		Mensagem:   mensagem,
		Dados:      dados,
		Correlacao: plataforma.CorrelacaoDe(ctx),
	}})
}
