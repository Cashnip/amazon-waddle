package api

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/Cashnip/amazon-waddle/internal/identidade"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// corpoMaximo é higiene de entrada (NFR-14), não limiar de negócio: o login
// tem dois campos curtos, e nada justifica ler mais do que isto de um corpo
// que ainda não foi autenticado.
const corpoMaximo = 4 << 10

type entradaSessao struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

// saidaSessao é o que a casca precisa para saudar quem entrou — Comprador ou
// Administrador. O identificador e o papel ficam na Sessão, dentro do Redis: o
// navegador não tem o que fazer com nenhum dos dois, e um papel que viajasse no
// corpo seria papel que a casca pode mentir.
type saidaSessao struct {
	Nome string `json:"nome"`
}

// criarSessao autentica na loja e emite o cookie opaco. Só consulta
// `identidade.comprador`: o mesmo e-mail pode existir como Administrador, e
// quem decide o papel é a rota, nunca uma ordem de consulta.
func (s *servidor) criarSessao(w http.ResponseWriter, r *http.Request) {
	s.entrar(w, r, func(ctx context.Context, email, senha string) (identidade.Conta, error) {
		return identidade.Autenticar(ctx, s.pool, email, senha)
	})
}

// autenticador é a assinatura comum das duas autenticações, com o pool já
// fechado dentro do fecho: `api` não nomeia a DBTX do pacote gerado — a única
// porta de identidade é identidade.go (AD-1).
type autenticador func(ctx context.Context, email, senha string) (identidade.Conta, error)

// entrar é o corpo dos dois logins — o da loja e o da área administrativa. O
// que muda entre eles é só a tabela consultada, e por isso a autenticação chega
// como parâmetro: duas cópias deste fluxo divergiriam no bloqueio por
// tentativas, e o login de Administrador ficaria com um Argon2id sem teto de
// custo — a conta mais valiosa atrás da porta mais fraca.
func (s *servidor) entrar(w http.ResponseWriter, r *http.Request, autenticar autenticador) {
	semCache(w)

	var entrada entradaSessao
	if err := decodificarCorpo(w, r, &entrada); err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}

	// O e-mail vira parte de chave de Redis com prazo de 15 minutos. Sem teto,
	// um chamador não autenticado cunha chaves de 4 KiB — o tamanho do corpo —
	// a cada requisição. O limiar é o mesmo do cadastro, e a recusa é o 401 de
	// sempre: um e-mail de 300 caracteres não existe em conta nenhuma, e uma
	// mensagem própria aqui contaria ao chamador onde ele bateu.
	if len(entrada.Email) > s.cfg.EmailMax {
		erro.Escrever(r.Context(), w, identidade.ErrCredencialInvalida, nil)
		return
	}

	// A consulta vem ANTES do Autenticar: o bloqueio é também o limite de
	// custo da rota, e conferir depois gastaria o Argon2id que ele existe para
	// poupar. A origem é o RemoteAddr, e não o X-Forwarded-For: o cabeçalho
	// chega do navegador, e quem quisesse escapar do bloqueio só teria de
	// variá-lo.
	origem := origemDe(r)
	bloqueado, err := identidade.LoginBloqueado(r.Context(), s.rdb, entrada.Email, origem, s.cfg.AuthTentativasMax)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	if bloqueado {
		erro.EscreverBloqueio(r.Context(), w, s.cfg.AuthBloqueioDuracao)
		return
	}

	conta, err := autenticar(r.Context(), entrada.Email, entrada.Senha)
	if err != nil {
		// Só credencial inválida conta: um Postgres fora do ar é 500, e contar
		// isso como tentativa bloquearia o Comprador por uma falha nossa.
		//
		// A falha de contagem não muda a resposta — o 401 já está decidido, e
		// virá-lo em 500 por causa do contador diria ao atacante que a rota
		// tropeçou. O envelope do erro de Redis fica no log do Escrever quando
		// for a vez dele.
		if errors.Is(err, identidade.ErrCredencialInvalida) {
			if falha := identidade.RegistrarFalhaDeLogin(r.Context(), s.rdb, entrada.Email, origem, s.cfg.AuthBloqueioDuracao); falha != nil {
				slog.ErrorContext(r.Context(), "registrar a falha de login", "erro", falha.Error())
			}
		}
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	// O login válido esquece as falhas daquele par; senão quem erra quatro
	// vezes e acerta fica com quatro falhas penduradas até o prazo passar.
	if falha := identidade.EsquecerFalhas(r.Context(), s.rdb, entrada.Email, origem); falha != nil {
		slog.ErrorContext(r.Context(), "esquecer as falhas de login", "erro", falha.Error())
	}

	if !s.abrirSessao(w, r, conta) {
		return
	}
	escreverJSON(w, http.StatusOK, saidaSessao{Nome: conta.Nome})
}

// abrirSessao grava a Sessão no Redis e emite o cookie opaco. O cookie é
// `HttpOnly` para que script nenhum o leia, e `SameSite=Lax` porque a casca do
// Next e o Go vivem na mesma origem — quem atravessa é o rewrites(), não o
// navegador. Login e cadastro passam os dois por aqui: duplicar o SetCookie
// perderia um atributo num dos dois lados na primeira mudança.
//
// Devolve false depois de já ter escrito o envelope de erro — quem chama só
// precisa voltar.
func (s *servidor) abrirSessao(w http.ResponseWriter, r *http.Request, conta identidade.Conta) bool {
	token, err := identidade.CriarSessao(r.Context(), s.rdb, conta, s.cfg.SessaoExpiracao)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return false
	}
	http.SetCookie(w, cookieSessao(token, int(s.cfg.SessaoExpiracao.Seconds())))
	return true
}

// cookieSessao monta o cookie com os atributos do AD-9. Emitir e expirar
// passam os dois por aqui: o navegador só apaga o cookie quando Nome e Path
// batem com os da emissão, e duas cópias divergiriam na primeira mudança.
func cookieSessao(valor string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     identidade.NomeCookieSessao,
		Value:    valor,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// origemDe é o host do RemoteAddr, sem a porta. A porta é efêmera: mantê-la
// faria cada conexão TCP virar uma chave nova de contador, e o bloqueio nunca
// dispararia. Endereço sem porta (o que não acontece num servidor HTTP, mas
// acontece num teste) passa cru — a chave continua estável, que é o que importa.
func origemDe(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// semCache: as respostas por Comprador não podem ser guardadas. Sem a
// diretiva, um intermediário pode fazê-lo por heurística e servir o nome — ou
// o Pedido — de um a outro.
func semCache(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }

// compradorDaRequisicao e administradorDaRequisicao resolvem o cookie de Sessão
// exigindo o papel de cada lado. Cookie ausente, adulterado, fora do Redis ou
// de papel errado saem todos como ErrSessaoInvalida — para quem chama, expirada,
// inexistente e "não é sua" são a mesma coisa. O papel vem de dentro da Sessão,
// e por isso conferi-lo não custa uma ida ao Postgres.
func (s *servidor) compradorDaRequisicao(r *http.Request) (identidade.Conta, error) {
	return s.contaDaRequisicao(r, identidade.PapelComprador)
}

func (s *servidor) administradorDaRequisicao(r *http.Request) (identidade.Conta, error) {
	return s.contaDaRequisicao(r, identidade.PapelAdministrador)
}

func (s *servidor) contaDaRequisicao(r *http.Request, papel identidade.Papel) (identidade.Conta, error) {
	cookie, err := r.Cookie(identidade.NomeCookieSessao)
	if err != nil {
		return identidade.Conta{}, identidade.ErrSessaoInvalida
	}
	// O TTL vai junto porque ler renova o prazo: é a expiração por inatividade
	// do §7.1, e quem sabe o prazo configurado é esta camada.
	conta, err := identidade.LerSessao(r.Context(), s.rdb, cookie.Value, s.cfg.SessaoExpiracao)
	if err != nil {
		return identidade.Conta{}, err
	}
	// Papel vazio cai aqui também: é a Sessão gravada antes desta estória, e
	// quem não sabe dizer o que é entra de novo.
	if conta.Papel != papel {
		return identidade.Conta{}, identidade.ErrSessaoInvalida
	}
	return conta, nil
}

func (s *servidor) lerSessao(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	conta, err := s.compradorDaRequisicao(r)
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, http.StatusOK, saidaSessao{Nome: conta.Nome})
}

// encerrarSessao apaga a chave no Redis e expira o cookie. Sem cookie, com
// valor desconhecido ou malformado a resposta é o mesmo 204: sair é
// idempotente, e quem sai não precisa saber se estava dentro. Um 401 aqui
// diria ao chamador que aquele cookie valia alguma coisa.
func (s *servidor) encerrarSessao(w http.ResponseWriter, r *http.Request) {
	semCache(w)

	if cookie, err := r.Cookie(identidade.NomeCookieSessao); err == nil {
		if err := identidade.EncerrarSessao(r.Context(), s.rdb, cookie.Value); err != nil {
			// O Redis fora do ar é o único caso em que não dá para prometer que
			// a Sessão morreu — e prometer sem apagar seria pior que o 500.
			erro.Escrever(r.Context(), w, err, nil)
			return
		}
	}
	// MaxAge negativo é o `Max-Age=0` que manda o navegador apagar o cookie.
	http.SetCookie(w, cookieSessao("", -1))
	w.WriteHeader(http.StatusNoContent)
}
