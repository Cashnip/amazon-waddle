package plataforma

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

// AbrirRedis monta o cliente a partir da AZAMON_REDIS_URL. Sem ping de
// propósito: a conexão do go-redis é preguiçosa, e exigir Redis no arranque
// derrubaria `/api/v1/saude` e o teste do arranque por um serviço que só a
// Sessão usa.
func AbrirRedis(url string) (*redis.Client, error) {
	opcoes, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("AZAMON_REDIS_URL inválida: %w", err)
	}
	return redis.NewClient(opcoes), nil
}
