// Package erro traduz o erro sentinela do módulo em resposta HTTP — num
// arquivo só, que é o que o AD-14 exige. Módulo de domínio não conhece status.
package erro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/identidade"
	"github.com/Cashnip/amazon-waddle/internal/pedido"
	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// ErrNaoEncontrado cobre a rota que não existe — sem ele o ServeMux
// responderia texto puro e a API teria dois contratos de erro.
var ErrNaoEncontrado = errors.New("Recurso não encontrado.")

// ErrEntradaInvalida cobre o corpo que nem chega ao domínio: JSON malformado
// ou acima do limite do http.MaxBytesReader. Mora aqui, e não num módulo,
// porque é falha de tradução — o mesmo motivo de ErrNaoEncontrado.
var ErrEntradaInvalida = errors.New("Requisição inválida.")

// ErrNaoAutorizado cobre quem chama sem a credencial que a rota exige e não
// tem Sessão para exibir — hoje só o webhook do Provedor, autenticado por
// segredo compartilhado. Mora aqui pelo mesmo motivo de ErrEntradaInvalida:
// é falha de tradução, e não de domínio.
var ErrNaoAutorizado = errors.New("Requisição não autorizada.")

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
	{ErrEntradaInvalida, http.StatusBadRequest, "ENTRADA_INVALIDA"},
	{ErrNaoAutorizado, http.StatusUnauthorized, "NAO_AUTORIZADO"},
	{identidade.ErrCredencialInvalida, http.StatusUnauthorized, "CREDENCIAL_INVALIDA"},
	{identidade.ErrSessaoInvalida, http.StatusUnauthorized, "SESSAO_INVALIDA"},
	// 404 e não 401: o link de redefinição não é credencial de ninguém, e o
	// recurso que ele nomeia de fato não existe mais. Expirado, já usado e
	// inexistente saem os três por aqui — a tela oferece solicitar outro.
	{identidade.ErrTokenInvalido, http.StatusNotFound, "TOKEN_INVALIDO"},
	// 409 e não 400: o corpo veio correto, o que não comporta é o estado do
	// mundo. O campo em falta viaja em `dados`, como em todo erro em linha.
	{identidade.ErrEmailJaCadastrado, http.StatusConflict, "EMAIL_JA_CADASTRADO"},
	// 409 nos dois: a requisição está correta, o estado do mundo é que não
	// comporta. O disponível viaja em `dados`, e não na mensagem.
	{catalogo.ErrEstoqueInsuficiente, http.StatusConflict, "ESTOQUE_INSUFICIENTE"},
	{pedido.ErrEstadoJaAvancado, http.StatusConflict, "ESTADO_JA_AVANCADO"},
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
	envelopar(ctx, w, status, codigo, mensagem, dados)
}

// EscreverCampo é o erro em linha: 400 CAMPO_INVALIDO com o nome do campo em
// `dados`, que é o contrato que a tela usa para ligar a mensagem ao campo e
// pôr o foco nele (UX-DR16). Não passa pelo registro porque a mensagem nomeia
// o limiar — e limiar vem da Config, nunca de um sentinela fixo daqui.
func EscreverCampo(ctx context.Context, w http.ResponseWriter, campo, mensagem string) {
	envelopar(ctx, w, http.StatusBadRequest, "CAMPO_INVALIDO", mensagem, map[string]string{"campo": campo})
}

// EscreverLimiteDeEnderecos é o 409 do teto por Comprador da 2.5. Não é
// sentinela nova, e não passa pelo registro, pelo mesmo motivo de EscreverCampo
// e EscreverBloqueio: a mensagem nomeia o limiar, e limiar vem da Config
// (AD-13/NFR-16) — um sentinela fixo aqui teria de repetir o número.
//
// 409 e não 400: o corpo veio correto, o que não comporta é o estado da conta.
// A saída é remover um Endereço, e a mensagem diz isso.
func EscreverLimiteDeEnderecos(ctx context.Context, w http.ResponseWriter, maximo int) {
	envelopar(ctx, w, http.StatusConflict, "LIMITE_DE_ENDERECOS",
		fmt.Sprintf("Você já tem %d Endereços cadastrados. Remova um para cadastrar outro.", maximo), nil)
}

// EscreverBloqueio é o 429 do login bloqueado por tentativas. Não passa pelo
// registro pelo mesmo motivo do EscreverCampo: a mensagem nomeia o limiar, e
// limiar vem da Config (AD-13/NFR-16), nunca de um sentinela fixo daqui.
//
// O corpo não diz se a conta existe — é o mesmo para e-mail cadastrado e para
// e-mail que nunca existiu, senão a própria resposta enumeraria contas.
// O prazo entra como duração, e não em minutos já contados: arredondar para
// cima é responsabilidade de quem escreve a mensagem, e truncar faria um
// AZAMON_AUTH_BLOQUEIO_DURACAO=30s — que a Config aceita — mandar tentar de
// novo "em 0 minutos".
func EscreverBloqueio(ctx context.Context, w http.ResponseWriter, prazo time.Duration) {
	minutos := int((prazo + time.Minute - 1) / time.Minute)
	mensagem := fmt.Sprintf("Muitas tentativas. Tente novamente em %d minutos.", minutos)
	if minutos == 1 {
		mensagem = "Muitas tentativas. Tente novamente em 1 minuto."
	}
	envelopar(ctx, w, http.StatusTooManyRequests, "MUITAS_TENTATIVAS", mensagem, nil)
}

// envelopar é o único ponto que serializa o envelope do AD-14.
func envelopar(ctx context.Context, w http.ResponseWriter, status int, codigo, mensagem string, dados any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{corpo{
		Codigo:     codigo,
		Mensagem:   mensagem,
		Dados:      dados,
		Correlacao: plataforma.CorrelacaoDe(ctx),
	}})
}
