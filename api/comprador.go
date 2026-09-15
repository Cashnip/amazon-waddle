package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/Cashnip/amazon-waddle/internal/identidade"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// entradaComprador é o corpo do cadastro. Nome, e-mail e senha: Endereço é da
// 2.5 e não é pedido aqui.
type entradaComprador struct {
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Senha string `json:"senha"`
}

// criarComprador cadastra e já abre a Sessão: o visitante sai autenticado, sem
// digitar as mesmas credenciais outra vez. A resposta é a mesma saidaSessao do
// login, e por isso a casca não precisa saber por onde a Sessão nasceu.
func (s *servidor) criarComprador(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	var entrada entradaComprador
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, corpoMaximo)).Decode(&entrada); err != nil {
		erro.Escrever(r.Context(), w, erro.ErrEntradaInvalida, nil)
		return
	}
	// As bordas caem antes de qualquer checagem: " ana@exemplo.br " é o mesmo
	// endereço, e um nome só de espaços não é nome.
	entrada.Nome = strings.TrimSpace(entrada.Nome)
	entrada.Email = strings.TrimSpace(entrada.Email)

	// A validação vem antes do Cadastrar de propósito: senha curta é recusada
	// sem gastar um Argon2id, que é a operação cara desta rota.
	if campo, mensagem := s.validarComprador(entrada); campo != "" {
		erro.EscreverCampo(r.Context(), w, campo, mensagem)
		return
	}

	comprador, err := identidade.Cadastrar(r.Context(), s.pool, entrada.Nome, entrada.Email, entrada.Senha)
	if err != nil {
		// O 409 também é erro em linha: a tela o liga ao campo do e-mail pelo
		// mesmo `dados.campo` do 400, numa leitura só.
		if errors.Is(err, identidade.ErrEmailJaCadastrado) {
			erro.Escrever(r.Context(), w, err, map[string]string{"campo": "email"})
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	if !s.abrirSessao(w, r, comprador) {
		return
	}
	escreverJSON(w, http.StatusCreated, saidaSessao{Nome: comprador.Nome})
}

// validarComprador devolve o primeiro campo em falta e a mensagem que o
// nomeia, ou dois vazios. Um campo por resposta porque a tela põe o foco num
// campo só — e os limiares vêm todos da Config (AD-13/NFR-16), nunca de
// literal aqui.
//
// Nome e senha são contados em runas, e não em bytes: "José" tem quatro
// caracteres para quem digita, e um limite medido em bytes recusaria nomes
// acentuados antes da hora. O e-mail é a exceção e é contado em bytes: o 254
// vem da RFC 5321, cuja unidade é o octeto — medido em runas, um endereço
// acentuado de 411 bytes passaria pelo teto.
func (s *servidor) validarComprador(e entradaComprador) (campo, mensagem string) {
	switch {
	case e.Nome == "":
		return "nome", "Informe o seu nome."
	case utf8.RuneCountInString(e.Nome) > s.cfg.CompradorNomeMax:
		return "nome", fmt.Sprintf("O nome pode ter no máximo %d caracteres.", s.cfg.CompradorNomeMax)
	case !enderecoPlausivel(e.Email):
		return "email", "Informe um e-mail válido, no formato nome@dominio.com."
	case len(e.Email) > s.cfg.EmailMax:
		return "email", fmt.Sprintf("O e-mail pode ter no máximo %d caracteres.", s.cfg.EmailMax)
	}
	return s.validarSenha(e.Senha)
}

// validarSenha são as regras de senha da 2.1, numa função só porque a
// redefinição promete "as mesmas regras" — e isso só é verdade se for a mesma
// função. Duas cópias divergiriam na primeira mudança de limiar, e o Comprador
// escolheria na redefinição uma senha que o cadastro recusaria.
//
// A senha é contada em runas, e não em bytes: "señ@-de-8" tem nove caracteres
// para quem digita, e um limite medido em bytes recusaria senhas acentuadas
// antes da hora.
func (s *servidor) validarSenha(senha string) (campo, mensagem string) {
	switch {
	case utf8.RuneCountInString(senha) < s.cfg.SenhaMin:
		return "senha", fmt.Sprintf("A senha precisa de pelo menos %d caracteres.", s.cfg.SenhaMin)
	case utf8.RuneCountInString(senha) > s.cfg.SenhaMax:
		return "senha", fmt.Sprintf("A senha pode ter no máximo %d caracteres.", s.cfg.SenhaMax)
	}
	return "", ""
}

// enderecoPlausivel usa o net/mail da biblioteca padrão em vez de uma expressão
// regular escrita à mão — endereço de e-mail é a RFC 5322, e toda regex curta
// para isso erra dos dois lados. A igualdade com o texto original é o que
// recusa a forma com nome de exibição (`Ana <ana@exemplo.br>`), que o
// ParseAddress aceita e que não é o que o visitante quis digitar.
//
// O ponto no domínio é conferido à mão porque a RFC 5322 não o exige: sem ele
// `ana@exemplo`, `a@b` e `ana@localhost` entram, e a mensagem que este guarda
// produz promete "no formato nome@dominio.com" — validador e mensagem têm de
// dizer a mesma coisa.
func enderecoPlausivel(email string) bool {
	endereco, err := mail.ParseAddress(email)
	if err != nil || endereco.Address != email {
		return false
	}
	_, dominio, _ := strings.Cut(email, "@")
	return strings.Contains(dominio, ".")
}
