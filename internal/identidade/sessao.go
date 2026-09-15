package identidade

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NomeCookieSessao é o cookie que atravessa o rewrites() do Next (AD-10).
const NomeCookieSessao = "azamon_sessao"

// prefixoSessao mantém as Sessões num espaço de nome próprio dentro do Redis.
const prefixoSessao = "sessao:"

// tamanhoToken são os 256 bits opacos do AD-9, em hexadecimal — 64 caracteres.
const tamanhoToken = 32

// CriarSessao grava a Sessão no Redis com TTL e devolve o valor opaco do
// cookie. O cookie não carrega conteúdo nenhum: quem sabe de quem é a Sessão é
// o Redis, e expirar é apagar a chave.
func CriarSessao(ctx context.Context, rdb *redis.Client, c Comprador, ttl time.Duration) (string, error) {
	token := novoToken()

	dados, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("serializar a Sessão: %w", err)
	}
	if err := rdb.Set(ctx, prefixoSessao+token, dados, ttl).Err(); err != nil {
		return "", fmt.Errorf("gravar a Sessão: %w", err)
	}
	return token, nil
}

// novoToken são os 256 bits opacos do AD-9 em hexadecimal. Fica numa função
// porque a Sessão e o token de redefinição de senha nascem da MESMA fonte:
// duas cópias da geração divergiriam, e um token de redefinição com menos
// entropia que o cookie seria a porta mais fraca da mesma conta.
func novoToken() string {
	var b [tamanhoToken]byte
	rand.Read(b[:]) // crypto/rand.Read nunca falha desde o Go 1.24.
	return hex.EncodeToString(b[:])
}

// tokenPlausivel confere a forma do valor do cookie. O token vem do navegador
// e vira chave de Redis: conferir a forma aqui é o que impede um valor
// arbitrário de passear pelo espaço de nomes.
func tokenPlausivel(token string) bool {
	if len(token) != tamanhoToken*2 {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}

// LerSessao resolve o valor do cookie no Comprador. Cookie ausente, com forma
// errada ou sem chave no Redis são todos ErrSessaoInvalida — nada disso é 500.
//
// Ler renova o prazo: é isso que o §7.1 chama de expiração por inatividade.
// O GetEx faz as duas coisas num comando só — ler e depois expirar seriam duas
// idas ao Redis, com uma janela entre elas em que a chave pode sumir.
func LerSessao(ctx context.Context, rdb *redis.Client, token string, ttl time.Duration) (Comprador, error) {
	if !tokenPlausivel(token) {
		return Comprador{}, ErrSessaoInvalida
	}

	dados, err := rdb.GetEx(ctx, prefixoSessao+token, ttl).Bytes()
	if errors.Is(err, redis.Nil) {
		return Comprador{}, ErrSessaoInvalida
	}
	if err != nil {
		return Comprador{}, fmt.Errorf("ler a Sessão: %w", err)
	}
	var c Comprador
	if err := json.Unmarshal(dados, &c); err != nil {
		return Comprador{}, fmt.Errorf("Sessão ilegível: %w", err)
	}
	return c, nil
}

// EncerrarSessao apaga a chave no Redis — expirar o cookie sozinho não
// invalida nada, porque quem sabe da Sessão é o Redis. Token de forma errada ou
// desconhecido não é erro: sair é idempotente, e quem sai não precisa saber se
// estava dentro.
func EncerrarSessao(ctx context.Context, rdb *redis.Client, token string) error {
	if !tokenPlausivel(token) {
		return nil
	}
	if err := rdb.Del(ctx, prefixoSessao+token).Err(); err != nil {
		return fmt.Errorf("encerrar a Sessão: %w", err)
	}
	return nil
}

// EncerrarSessoesDoComprador apaga TODAS as Sessões daquele Comprador — é o
// que faz a redefinição de senha derrubar quem já estava dentro, inclusive de
// outro navegador. Expirar o cookie não serviria: quem sabe da Sessão é o
// Redis.
//
// A varredura é por SCAN, e não por um índice reverso comprador→tokens. Quem lê
// a Sessão é LerSessao, num GetEx só; um índice custaria um comando a mais em
// TODA requisição autenticada para servir uma operação que acontece uma vez por
// esquecimento de senha. O SCAN é cursor, e não KEYS: não trava o Redis.
//
// ponytail: O(chaves de Sessão) por redefinição. Índice reverso se um dia o
// volume de Sessões vivas tornar a varredura visível.
// Uma falha numa chave não interrompe a varredura: quem chama já gravou a
// senha nova, e desistir no meio deixaria vivas justamente as Sessões que a
// troca de senha existe para derrubar. O primeiro erro é guardado e devolvido
// no fim — quem chama continua sabendo que a promessa não foi cumprida por
// inteiro, depois de a varredura ter feito o que pôde.
func EncerrarSessoesDoComprador(ctx context.Context, rdb *redis.Client, compradorID string) error {
	var cursor uint64
	var primeiroErro error
	guardar := func(err error) {
		if primeiroErro == nil {
			primeiroErro = err
		}
	}
	for {
		chaves, proximo, err := rdb.Scan(ctx, cursor, prefixoSessao+"*", 100).Result()
		if err != nil {
			// Sem cursor não há como continuar: o SCAN é o que enumera.
			guardar(fmt.Errorf("varrer as Sessões: %w", err))
			return primeiroErro
		}
		for _, chave := range chaves {
			// Get puro, e não GetEx: renovar o prazo aqui daria sobrevida às
			// Sessões dos OUTROS Compradores a cada redefinição.
			dados, err := rdb.Get(ctx, chave).Bytes()
			if errors.Is(err, redis.Nil) {
				continue // expirou entre o SCAN e a leitura
			}
			if err != nil {
				guardar(fmt.Errorf("ler a Sessão na varredura: %w", err))
				continue
			}
			var c Comprador
			if err := json.Unmarshal(dados, &c); err != nil || c.ID != compradorID {
				continue
			}
			if err := rdb.Del(ctx, chave).Err(); err != nil {
				guardar(fmt.Errorf("encerrar a Sessão do Comprador: %w", err))
			}
		}
		cursor = proximo
		if cursor == 0 {
			return primeiroErro
		}
	}
}
