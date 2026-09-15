package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Cashnip/amazon-waddle/internal/identidade"
)

// criarSessaoAdministrador é o login da área administrativa. Consulta SÓ
// `identidade.administrador`: um Comprador que use o mesmo e-mail leva o mesmo
// CREDENCIAL_INVALIDA de quem não existe, e não há promoção possível.
//
// Passa pelo mesmo `entrar` do login da loja, e por isso herda o bloqueio por
// tentativas, o teto do e-mail e o cookie de atributos idênticos.
func (s *servidor) criarSessaoAdministrador(w http.ResponseWriter, r *http.Request) {
	s.entrar(w, r, func(ctx context.Context, email, senha string) (identidade.Conta, error) {
		return identidade.AutenticarAdministrador(ctx, s.pool, email, senha)
	})
}

// lerSessaoAdministrador existe para a guarda do prefixo ter o que proteger, e
// é a rota que a Épica 3 encontra pronta. Não repete checagem nenhuma: quem já
// disse que esta Sessão é de Administrador foi `somenteAdministrador`.
func (s *servidor) lerSessaoAdministrador(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	// ponytail: duas idas ao Redis por requisição administrativa — a guarda lê
	// a Sessão e o handler a relê, no mesmo GetEx. Teto: uma ida a mais por
	// requisição, e a janela entre as duas leituras. Troca por levar a Conta no
	// contexto da requisição quando a Épica 3 trouxer o segundo handler
	// administrativo; com um só, o tipo de chave e o cast custam mais que a ida.
	conta, err := s.administradorDaRequisicao(r)
	if err != nil {
		// 404, e não o erro: a guarda JÁ deixou esta requisição passar, e uma
		// Sessão que sumiu entre as duas leituras não pode responder 401 —
		// seria a única resposta da subárvore que conta que a rota existe.
		naoEncontrado(w, r)
		return
	}
	escreverJSON(w, http.StatusOK, saidaSessao{Nome: conta.Nome})
}

// somenteAdministrador é a autorização do prefixo `/api/v1/admin/`. A recusa é
// o MESMO 404 da rota que não existe, e não um 403: a UX-DR9 pede que a área
// não apareça para quem não é Administrador, e "não aparece" do lado do
// servidor quer dizer indistinguível de inexistente. É a mesma leitura que o
// AD-11 faz para recurso de outro dono.
func (s *servidor) somenteAdministrador(proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.administradorDaRequisicao(r); err != nil {
			// Redis fora do ar sai no mesmo 404: um 500 aqui contaria que há
			// rota atrás da guarda. Mas o que não é Sessão inválida vai para o
			// log, senão a queda de infraestrutura vira um 404 mudo — a guarda
			// não passa mais por erro.Escrever, que era quem registrava.
			if !errors.Is(err, identidade.ErrSessaoInvalida) {
				slog.ErrorContext(r.Context(), "resolver a Sessão de Administrador", "erro", err.Error())
			}
			naoEncontrado(w, r)
			return
		}
		proximo.ServeHTTP(w, r)
	})
}
