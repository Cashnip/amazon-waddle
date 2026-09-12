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
	var b [tamanhoToken]byte
	rand.Read(b[:]) // crypto/rand.Read nunca falha desde o Go 1.24.
	token := hex.EncodeToString(b[:])

	dados, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("serializar a Sessão: %w", err)
	}
	if err := rdb.Set(ctx, prefixoSessao+token, dados, ttl).Err(); err != nil {
		return "", fmt.Errorf("gravar a Sessão: %w", err)
	}
	return token, nil
}

// LerSessao resolve o valor do cookie no Comprador. Cookie ausente, com forma
// errada ou sem chave no Redis são todos ErrSessaoInvalida — nada disso é 500.
func LerSessao(ctx context.Context, rdb *redis.Client, token string) (Comprador, error) {
	// O token vem do navegador e vira chave de Redis: conferir a forma aqui é
	// o que impede um valor arbitrário de passear pelo espaço de nomes.
	if len(token) != tamanhoToken*2 {
		return Comprador{}, ErrSessaoInvalida
	}
	if _, err := hex.DecodeString(token); err != nil {
		return Comprador{}, ErrSessaoInvalida
	}

	dados, err := rdb.Get(ctx, prefixoSessao+token).Bytes()
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
