package api

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/identidade"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// entradaEndereco é o corpo do cadastro e da edição — os dois mesmos campos,
// porque editar é reescrever a etiqueta inteira. São o mínimo de uma etiqueta
// de entrega, e `complemento` é o único opcional.
type entradaEndereco struct {
	Destinatario string `json:"destinatario"`
	CEP          string `json:"cep"`
	Logradouro   string `json:"logradouro"`
	Numero       string `json:"numero"`
	Complemento  string `json:"complemento"`
	Bairro       string `json:"bairro"`
	Cidade       string `json:"cidade"`
	UF           string `json:"uf"`
}

// saidaEndereco é a entrada mais o identificador. O CEP sai com os oito
// dígitos que estão no banco: a máscara é da tela.
type saidaEndereco struct {
	ID string `json:"id"`
	entradaEndereco
}

// formatoCEP aceita as duas formas que a tela pode mandar — com e sem hífen —
// porque a máscara é do navegador. Quem tira o hífen antes de gravar é
// identidade.CriarEndereco; o que este guarda garante é que sobram exatamente
// oito dígitos (PRD §4.2). Não há consulta a serviço de CEP pela rede (NFR-15).
var formatoCEP = regexp.MustCompile(`^[0-9]{5}-?[0-9]{3}$`)

// As 27 siglas. A validação é contra a lista, e não por comprimento: `ZZ` passa
// por qualquer teto de dois caracteres e só apareceria como etiqueta impossível
// lá na entrega.
var siglasUF = []string{
	"AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG",
	"PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RO", "RR", "RS", "SC", "SE", "SP", "TO",
}

// listarEnderecos é o "escolher" da FR-5: a lista do dono, em ordem estável.
// Quem escolhe sobre ela é o checkout da 5.2 — não há Endereço padrão.
func (s *servidor) listarEnderecos(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	enderecos, err := identidade.ListarEnderecos(r.Context(), s.pool, comprador.ID)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	// Fatia vazia, e nunca `null`: o estado vazio é da tela, e um `null` a
	// obrigaria a defender-se do formato antes de decidir o que mostrar.
	saidas := make([]saidaEndereco, 0, len(enderecos))
	for _, e := range enderecos {
		saidas = append(saidas, saidaDe(e))
	}
	escreverJSON(w, http.StatusOK, saidas)
}

func (s *servidor) criarEndereco(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, entrada, ok := s.enderecoDaRequisicao(w, r)
	if !ok {
		return
	}
	novo, err := identidade.CriarEndereco(r.Context(), s.pool, comprador.ID, doCorpo(entrada), s.cfg.EnderecoPorCompradorMax)
	if err != nil {
		// O único ErrNoRows possível aqui é o teto por Comprador: o dono vem da
		// Sessão, e não há linha alheia nem uuid malformado neste caminho.
		if errors.Is(err, pgx.ErrNoRows) {
			erro.EscreverLimiteDeEnderecos(r.Context(), w, s.cfg.EnderecoPorCompradorMax)
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusCreated, saidaDe(novo))
}

func (s *servidor) atualizarEndereco(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, entrada, ok := s.enderecoDaRequisicao(w, r)
	if !ok {
		return
	}
	atualizado, err := identidade.AtualizarEndereco(r.Context(), s.pool, r.PathValue("id"), comprador.ID, doCorpo(entrada))
	if err != nil {
		// Endereço de outro Comprador, Endereço inexistente e uuid malformado
		// são o mesmo 404: responder diferente vazaria a existência da linha.
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusOK, saidaDe(atualizado))
}

// removerEndereco é DELETE de verdade, e nenhum Pedido é tocado: o Pedido
// congela o Endereço na criação (AD-3). Sem corpo, e por isso sem passar por
// enderecoDaRequisicao.
func (s *servidor) removerEndereco(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	if err := identidade.RemoverEndereco(r.Context(), s.pool, r.PathValue("id"), comprador.ID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
			return
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// enderecoDaRequisicao junta os três passos que o POST e o PUT fazem iguais —
// Sessão, corpo e validação — e escreve o envelope de quem falhar. Duas cópias
// divergiriam na primeira mudança de validação, e a edição passaria a aceitar
// o que o cadastro recusa.
func (s *servidor) enderecoDaRequisicao(w http.ResponseWriter, r *http.Request) (identidade.Conta, entradaEndereco, bool) {
	var entrada entradaEndereco

	comprador, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return comprador, entrada, false
	}
	if err := decodificarCorpo(w, r, &entrada); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return comprador, entrada, false
	}
	// As bordas caem antes de qualquer checagem, como no cadastro do Comprador:
	// um campo só de espaços não é campo, e " SP " é a mesma UF.
	entrada.Destinatario = strings.TrimSpace(entrada.Destinatario)
	entrada.CEP = strings.TrimSpace(entrada.CEP)
	entrada.Logradouro = strings.TrimSpace(entrada.Logradouro)
	entrada.Numero = strings.TrimSpace(entrada.Numero)
	entrada.Complemento = strings.TrimSpace(entrada.Complemento)
	entrada.Bairro = strings.TrimSpace(entrada.Bairro)
	entrada.Cidade = strings.TrimSpace(entrada.Cidade)
	entrada.UF = strings.ToUpper(strings.TrimSpace(entrada.UF))

	if campo, mensagem := s.validarEndereco(entrada); campo != "" {
		erro.EscreverCampo(r.Context(), w, campo, mensagem)
		return comprador, entrada, false
	}
	return comprador, entrada, true
}

// validarEndereco devolve o primeiro campo em falta e a mensagem que o nomeia,
// ou dois vazios — um campo por resposta, porque a tela põe o foco num campo
// só (UX-DR20c). O teto vem da Config (AD-13/NFR-16), nunca de literal aqui.
//
// Os campos são contados em runas, e não em bytes: "São João" tem oito
// caracteres para quem digita, e um limite medido em bytes recusaria
// logradouros acentuados antes da hora — o mesmo motivo do nome do Comprador.
func (s *servidor) validarEndereco(e entradaEndereco) (campo, mensagem string) {
	// A ordem é a de quem preenche o formulário, de cima para baixo: o foco
	// salta para o primeiro campo recusado, e saltar para trás confundiria.
	obrigatorios := []struct{ campo, valor, rotulo string }{
		{"destinatario", e.Destinatario, "Informe o nome de quem recebe."},
		{"cep", e.CEP, "Informe o CEP."},
		{"logradouro", e.Logradouro, "Informe a rua ou avenida."},
		{"numero", e.Numero, "Informe o número."},
		{"bairro", e.Bairro, "Informe o bairro."},
		{"cidade", e.Cidade, "Informe a cidade."},
		{"uf", e.UF, "Informe a UF."},
	}
	for _, o := range obrigatorios {
		if o.valor == "" {
			return o.campo, o.rotulo
		}
	}
	// O teto vale para todos os oito, `complemento` inclusive: ele é opcional,
	// não ilimitado.
	for _, t := range []struct{ campo, valor string }{
		{"destinatario", e.Destinatario},
		{"logradouro", e.Logradouro},
		{"numero", e.Numero},
		{"complemento", e.Complemento},
		{"bairro", e.Bairro},
		{"cidade", e.Cidade},
	} {
		if utf8.RuneCountInString(t.valor) > s.cfg.EnderecoTextoMax {
			return t.campo, fmt.Sprintf("Este campo pode ter no máximo %d caracteres.", s.cfg.EnderecoTextoMax)
		}
	}
	switch {
	case !formatoCEP.MatchString(e.CEP):
		return "cep", "Informe um CEP válido, com 8 dígitos."
	case !slices.Contains(siglasUF, e.UF):
		return "uf", "Informe uma UF válida, como SP."
	}
	return "", ""
}

func doCorpo(e entradaEndereco) identidade.Endereco {
	return identidade.Endereco{
		Destinatario: e.Destinatario,
		CEP:          e.CEP,
		Logradouro:   e.Logradouro,
		Numero:       e.Numero,
		Complemento:  e.Complemento,
		Bairro:       e.Bairro,
		Cidade:       e.Cidade,
		UF:           e.UF,
	}
}

func saidaDe(e identidade.Endereco) saidaEndereco {
	return saidaEndereco{
		ID: e.ID,
		entradaEndereco: entradaEndereco{
			Destinatario: e.Destinatario,
			CEP:          e.CEP,
			Logradouro:   e.Logradouro,
			Numero:       e.Numero,
			Complemento:  e.Complemento,
			Bairro:       e.Bairro,
			Cidade:       e.Cidade,
			UF:           e.UF,
		},
	}
}
