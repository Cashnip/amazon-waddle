package identidade

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"

	"github.com/Cashnip/amazon-waddle/internal/identidade/db/gerado"
)

// ErrTokenInvalido cobre o token expirado, o já usado e o que nunca existiu —
// os três são a mesma coisa para quem chama, e distingui-los diria a um
// atacante que um token chegou a valer.
var ErrTokenInvalido = errors.New("Este link de redefinição não vale mais. Peça outro.")

// O token e o ponteiro reverso, cada um no seu espaço de nome, ao lado do
// prefixoSessao e do prefixoFalhas:
//
//	redefinicao:<token>  -> <comprador_id>
//	redefinicao-de:<id>  -> <token>
//
// Os dois nascem com o MESMO prazo, nativo do Redis: expirar é a chave sumir,
// e não há nada a varrer depois.
const (
	prefixoRedefinicao   = "redefinicao:"
	prefixoRedefinicaoDe = "redefinicao-de:"
)

// SolicitarRedefinicao emite o token de uso único do Comprador daquele e-mail.
// Conta inexistente devolve token vazio e erro nenhum: quem chama responde o
// mesmo para os dois casos, e uma sentinela aqui viraria enumeração de contas
// na primeira vez que alguém a traduzisse.
//
// O ponteiro reverso é o que faz "nunca dois tokens válidos" ser verdade sem
// varrer nada: a solicitação nova lê o ponteiro, apaga o token que ele aponta e
// regrava os dois. É uma chave só por Comprador, e expira junto.
func SolicitarRedefinicao(ctx context.Context, bd gerado.DBTX, rdb *redis.Client, email string, validade time.Duration) (string, error) {
	linha, err := gerado.New(bd).BuscarCompradorPorEmail(ctx, normalizarEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	id := uuidTexto(linha.ID)

	ponteiro := prefixoRedefinicaoDe + id
	anterior, err := rdb.Get(ctx, ponteiro).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("ler o token anterior: %w", err)
	}
	if anterior != "" {
		if err := rdb.Del(ctx, prefixoRedefinicao+anterior).Err(); err != nil {
			return "", fmt.Errorf("apagar o token anterior: %w", err)
		}
	}

	token := novoToken()
	if err := rdb.Set(ctx, prefixoRedefinicao+token, id, validade).Err(); err != nil {
		return "", fmt.Errorf("gravar o token de redefinição: %w", err)
	}
	if err := rdb.Set(ctx, ponteiro, token, validade).Err(); err != nil {
		return "", fmt.Errorf("gravar o ponteiro do token: %w", err)
	}
	return token, nil
}

// RedefinirSenha resolve o token, grava o Argon2id novo, gasta o token e
// encerra todas as Sessões daquele Comprador. Não abre Sessão nenhuma: quem
// redefiniu volta pelo Login, como qualquer outro.
//
// A ordem é deliberada — a senha é gravada ANTES de o token ser apagado. Ao
// contrário, uma falha do Postgres deixaria o Comprador sem token e com a senha
// velha, e sem porta de volta a não ser solicitar outro.
func RedefinirSenha(ctx context.Context, bd gerado.DBTX, rdb *redis.Client, token, senha string) error {
	// A forma é conferida antes de virar chave de Redis, pelo mesmo motivo do
	// cookie de Sessão: o token vem da barra de endereço.
	if !tokenPlausivel(token) {
		return ErrTokenInvalido
	}
	id, err := rdb.Get(ctx, prefixoRedefinicao+token).Result()
	if errors.Is(err, redis.Nil) {
		return ErrTokenInvalido
	}
	if err != nil {
		return fmt.Errorf("resolver o token de redefinição: %w", err)
	}
	var chave pgtype.UUID
	if err := chave.Scan(id); err != nil {
		return ErrTokenInvalido
	}

	hash, err := Gerar(senha)
	if err != nil {
		return err
	}
	if err := gerado.New(bd).AtualizarSenhaDoComprador(ctx, gerado.AtualizarSenhaDoCompradorParams{
		ID:        chave,
		SenhaHash: hash,
	}); err != nil {
		return fmt.Errorf("gravar a senha nova: %w", err)
	}

	// O token é de uso único: some junto com o ponteiro que o apontava.
	if err := rdb.Del(ctx, prefixoRedefinicao+token, prefixoRedefinicaoDe+id).Err(); err != nil {
		return fmt.Errorf("gastar o token de redefinição: %w", err)
	}
	return EncerrarSessoesDoComprador(ctx, rdb, id)
}
