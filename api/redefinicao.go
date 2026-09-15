package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Cashnip/amazon-waddle/internal/identidade"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

type entradaRedefinicao struct {
	Email string `json:"email"`
}

type entradaSenhaNova struct {
	Senha string `json:"senha"`
}

// criarRedefinicao emite o token e escreve o caminho no log. Não existe serviço
// de e-mail (AD-20): o log estruturado é o "e-mail" da demonstração, e é de lá
// que o caminho sai por `docker compose logs`.
//
// A resposta é o MESMO 202 de corpo vazio em todos os caminhos — conta que
// existe, conta que não existe, e-mail acima do teto e corpo que nem é JSON.
// Qualquer diferença — status, corpo, cabeçalho — enumeraria contas, que é o
// único ataque que esta rota, aberta e não autenticada, convida.
func (s *servidor) criarRedefinicao(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	var entrada entradaRedefinicao
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, corpoMaximo)).Decode(&entrada)
	entrada.Email = strings.TrimSpace(entrada.Email)
	// O teto limita o que um chamador não autenticado manda ao Postgres: sem
	// ele, cada requisição vira uma consulta com um texto do tamanho do corpo.
	// A recusa é silenciosa pelo mesmo motivo de tudo aqui.
	if err != nil || len(entrada.Email) > s.cfg.EmailMax {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	token, err := identidade.SolicitarRedefinicao(r.Context(), s.pool, s.rdb, entrada.Email, s.cfg.SenhaTokenValidade)
	if err != nil {
		// A falha não muda a resposta, como no contador de tentativas do login:
		// SolicitarRedefinicao só chega a falhar DEPOIS de achar a conta, e um
		// 500 aqui diria "esta conta existe" toda vez que o Redis tropeçasse —
		// a distinção que esta rota inteira existe para esconder. O envelope do
		// erro fica no log, que é onde ele serve para alguma coisa.
		slog.ErrorContext(r.Context(), "solicitar a redefinição de senha", "erro", err.Error())
	}
	if token != "" {
		// Só o caminho. E-mail, nome e senha não entram em log nenhum (AD-15),
		// e a linha só existe quando há conta — uma linha "sem conta" contaria
		// no log exatamente o que a resposta esconde.
		//
		// ponytail: o log carrega uma credencial viva por 30 minutos, e quem lê
		// o log entra na conta. É o preço de não haver serviço de e-mail
		// (AD-20) na demonstração. Quando o canal de e-mail existir, o token sai
		// daqui e o log fica só com o identificador da solicitação.
		slog.InfoContext(r.Context(), "redefinição de senha solicitada", "caminho", "/redefinir-senha/"+token)
	}
	w.WriteHeader(http.StatusAccepted)
}

// redefinirSenha troca a senha e derruba as Sessões. Devolve 204 e NÃO abre
// Sessão: a condição de aceite manda encerrar todas, e abrir uma aqui seria
// desfazer metade do que a rota acabou de fazer. O Comprador volta pelo Login.
func (s *servidor) redefinirSenha(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	var entrada entradaSenhaNova
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, corpoMaximo)).Decode(&entrada); err != nil {
		erro.Escrever(r.Context(), w, erro.ErrEntradaInvalida, nil)
		return
	}

	// A validação vem ANTES de resgatar o token, e é o que faz a recusa por
	// forma não gastar o link: quem digita uma senha curta demais corrige e
	// tenta de novo pelo mesmo caminho. De quebra, senha fora dos limites é
	// recusada sem gastar um Argon2id.
	if campo, mensagem := s.validarSenha(entrada.Senha); campo != "" {
		erro.EscreverCampo(r.Context(), w, campo, mensagem)
		return
	}

	if err := identidade.RedefinirSenha(r.Context(), s.pool, s.rdb, r.PathValue("token"), entrada.Senha); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
