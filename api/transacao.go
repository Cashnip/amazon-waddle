package api

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// iniciador é o que emTransacao usa do pool: só abrir a transação. É
// interface para o teste da repetição trocar o banco por uma transação falsa —
// o que se prova ali é quantas vezes a função roda, e não o Postgres.
type iniciador interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// emTransacao é o molde de uma transação por caso de uso (AD-4): abre, roda
// `fn` com a transação como parâmetro nomeado, e comita só quando `fn` pede.
// `fn` devolve `confirmar` falso para desfazer sem erro — é o reenvio da
// criação do Pedido, que responde 200 com o Pedido original e não pode gravar
// nada (nem o número que o contador já tinha avançado).
//
// **Repete uma vez, e só uma**, quando o banco desiste da transação por
// impasse (40P01) ou por falha de serialização (40001): é a vítima que o
// Postgres escolheu, e a segunda passada encontra as travas já soltas. A
// terceira não existe — um impasse que se repete é defeito de ordem de trava,
// e insistir só o esconderia. Qualquer outro erro sai na primeira.
//
// `fn` roda inteira de novo, então não pode ter efeito fora da transação: o
// JSON é escrito por quem chama, **depois** que emTransacao devolve.
//
// O Rollback e o Commit usam WithoutCancel: o cliente pode desistir no meio, e
// desfazer precisa de um contexto vivo; um Commit que morresse com o cliente
// deixaria o trabalho pronto por gravar e a resposta em 500.
func emTransacao(ctx context.Context, bd iniciador, fn func(tx pgx.Tx) (confirmar bool, err error)) error {
	err := tentarTransacao(ctx, bd, fn)
	if repetivel(err) {
		err = tentarTransacao(ctx, bd, fn)
	}
	return err
}

func tentarTransacao(ctx context.Context, bd iniciador, fn func(tx pgx.Tx) (bool, error)) error {
	tx, err := bd.Begin(ctx)
	if err != nil {
		return err
	}
	// Rollback depois de um Commit bem-sucedido é no-op no pgx: o que o defer
	// garante é que nenhum caminho de erro deixe a transação aberta.
	defer tx.Rollback(context.WithoutCancel(ctx))

	confirmar, err := fn(tx)
	if err != nil || !confirmar {
		return err
	}
	return tx.Commit(context.WithoutCancel(ctx))
}

// repetivel: impasse e falha de serialização, e nada mais.
func repetivel(err error) bool {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return false
	}
	return pg.Code == "40P01" || pg.Code == "40001"
}
