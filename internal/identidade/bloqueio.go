package identidade

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// prefixoFalhas mantém o contador de tentativas num espaço de nome próprio,
// ao lado do prefixoSessao.
const prefixoFalhas = "falhas:"

// chaveFalhas é o par (e-mail normalizado, origem). O e-mail passa pela mesma
// normalizarEmail do Autenticar: sem isso "Ana@x" e "ana@x " contariam em
// chaves distintas e cinco tentativas viriam de graça a cada variação de caixa.
//
// A origem entra crua — quem a escolhe é api/, e a decisão da estória é o
// RemoteAddr, nunca um cabeçalho que o próprio cliente escreve.
func chaveFalhas(email, origem string) string {
	return prefixoFalhas + normalizarEmail(email) + "|" + origem
}

// LoginBloqueado é consultado ANTES do Argon2id: o bloqueio também é o limite
// de custo da rota, e conferir depois de verificar a senha gastaria o hash que
// o bloqueio existe para poupar. Conta exista ou não a conta — se o bloqueio só
// valesse para conta existente, a própria resposta confirmaria o e-mail.
func LoginBloqueado(ctx context.Context, rdb *redis.Client, email, origem string, maximo int) (bool, error) {
	n, err := rdb.Get(ctx, chaveFalhas(email, origem)).Int()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("ler as tentativas: %w", err)
	}
	return n >= maximo, nil
}

// RegistrarFalhaDeLogin incrementa o contador e garante que ele tem prazo. O
// TTL é nativo do Redis: o contador é dado com prazo de validade, e não há
// nada a varrer depois — expirar é a chave sumir, como na Sessão.
//
// O prazo é dado por EXPIRE NX, e não só na primeira falha: INCR e EXPIRE não
// são atômicos, e um EXPIRE que falhe — ou um processo que morra entre os dois
// — deixaria o contador sem prazo nenhum, bloqueando o par para sempre. O NX
// cura a chave sem prazo na falha seguinte e não toca na que já tem: a janela
// continua contando da primeira falha, e não da última.
func RegistrarFalhaDeLogin(ctx context.Context, rdb *redis.Client, email, origem string, prazo time.Duration) error {
	chave := chaveFalhas(email, origem)
	if err := rdb.Incr(ctx, chave).Err(); err != nil {
		return fmt.Errorf("contar a falha: %w", err)
	}
	if err := rdb.ExpireNX(ctx, chave, prazo).Err(); err != nil {
		return fmt.Errorf("dar prazo às tentativas: %w", err)
	}
	return nil
}

// EsquecerFalhas apaga o contador do par. Sem isto, quem erra quatro vezes e
// acerta na quinta fica com quatro falhas penduradas até o prazo passar.
func EsquecerFalhas(ctx context.Context, rdb *redis.Client, email, origem string) error {
	if err := rdb.Del(ctx, chaveFalhas(email, origem)).Err(); err != nil {
		return fmt.Errorf("esquecer as tentativas: %w", err)
	}
	return nil
}

// EsquecerFalhasDeTodasAsOrigens apaga o contador daquele e-mail venha ele de
// onde vier. É o que a redefinição de senha chama: quem esqueceu a senha errou
// o login antes — é o caminho mais comum até aqui — e sair da redefinição para
// cair num 429 seria a recuperação não recuperar nada.
//
// Por todas as origens, e não só pela do pedido: as tentativas falhas vêm do
// navegador de onde o Comprador tentou entrar, e o link pode ser aberto de
// outro aparelho. Limpar não dá poder novo a ninguém — quem chega aqui provou
// a posse do token, e com ele já podia trocar a senha.
//
// ponytail: O(chaves do Redis) por redefinição, como a varredura de Sessões. O
// contador não tem índice por e-mail, e criar um custaria escrita a mais em
// toda tentativa falha para servir isto, que acontece uma vez por esquecimento.
func EsquecerFalhasDeTodasAsOrigens(ctx context.Context, rdb *redis.Client, email string) error {
	// O e-mail entra num padrão de glob, e a RFC 5322 aceita `*`, `?` e `[` no
	// nome — `a*b@exemplo.br` é endereço válido e cadastrável. Sem escapar, o
	// padrão varreria o contador dos vizinhos junto.
	padrao := prefixoFalhas + escaparGlob(normalizarEmail(email)) + "|*"

	var cursor uint64
	var primeiroErro error
	for {
		chaves, proximo, err := rdb.Scan(ctx, cursor, padrao, 100).Result()
		if err != nil {
			// Sem cursor não há como continuar: o SCAN é o que enumera.
			return fmt.Errorf("varrer as tentativas: %w", err)
		}
		if len(chaves) > 0 {
			// Uma falha aqui não interrompe a varredura, como na varredura de
			// Sessões: desistir no meio deixaria justamente os contadores que a
			// redefinição existe para limpar.
			if err := rdb.Del(ctx, chaves...).Err(); err != nil && primeiroErro == nil {
				primeiroErro = fmt.Errorf("esquecer as tentativas: %w", err)
			}
		}
		cursor = proximo
		if cursor == 0 {
			return primeiroErro
		}
	}
}

// escaparGlob neutraliza os metacaracteres que o MATCH do Redis entende, para
// que o texto seja procurado como texto.
func escaparGlob(s string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`*`, `\*`,
		`?`, `\?`,
		`[`, `\[`,
		`]`, `\]`,
	).Replace(s)
}
