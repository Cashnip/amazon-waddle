package api

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// txFalsa conta Commit e Rollback; o resto de pgx.Tx fica nil, porque a
// função de teste não o toca — o que se prova é quantas vezes emTransacao roda
// a função e quando comita, e não o Postgres.
type txFalsa struct {
	pgx.Tx
	contas *contagem
}

type contagem struct {
	begins, commits, rollbacks int
	// errosDoCommit é o erro de cada Commit, em ordem; faltando, é sucesso.
	errosDoCommit []error
	// erroDoBegin faz toda abertura falhar.
	erroDoBegin error
}

func (t txFalsa) Commit(context.Context) error {
	t.contas.commits++
	if n := t.contas.commits - 1; n < len(t.contas.errosDoCommit) {
		return t.contas.errosDoCommit[n]
	}
	return nil
}
func (t txFalsa) Rollback(context.Context) error { t.contas.rollbacks++; return nil }

type poolFalso struct{ contas *contagem }

func (p poolFalso) Begin(context.Context) (pgx.Tx, error) {
	p.contas.begins++
	if p.contas.erroDoBegin != nil {
		return nil, p.contas.erroDoBegin
	}
	return txFalsa{contas: p.contas}, nil
}

// TestEmTransacaoRepeteUmaVez é a regra do AD-4 na borda HTTP: impasse ou
// falha de serialização repete uma vez, e só uma; qualquer outro erro sai na
// primeira; e `confirmar` falso desfaz sem erro.
func TestEmTransacaoRepeteUmaVez(t *testing.T) {
	impasse := &pgconn.PgError{Code: "40P01"}
	serializacao := &pgconn.PgError{Code: "40001"}
	unica := &pgconn.PgError{Code: "23505"}
	outro := errors.New("qualquer")

	for _, caso := range []struct {
		nome      string
		erros     []error // o erro de cada passada, em ordem; nil é sucesso
		confirmar bool
		execucoes int
		commits   int
		erro      error
	}{
		{"sucesso na primeira comita uma vez", []error{nil}, true, 1, 1, nil},
		{"40P01 uma vez repete e comita", []error{impasse, nil}, true, 2, 1, nil},
		{"40P01 embrulhado com %w repete", []error{fmt.Errorf("travar os Produtos: %w", impasse), nil}, true, 2, 1, nil},
		{"40001 uma vez repete e comita", []error{serializacao, nil}, true, 2, 1, nil},
		{"40P01 duas vezes desiste", []error{impasse, impasse, nil}, true, 2, 0, impasse},
		{"outro código do banco não repete", []error{unica, nil}, true, 1, 0, unica},
		{"erro que não é do banco não repete", []error{outro, nil}, true, 1, 0, outro},
		{"confirmar falso desfaz sem erro", []error{nil}, false, 1, 0, nil},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			contas := &contagem{}
			execucoes := 0
			err := emTransacao(context.Background(), poolFalso{contas}, func(pgx.Tx) (bool, error) {
				err := caso.erros[execucoes]
				execucoes++
				return err == nil && caso.confirmar, err
			})
			if !errors.Is(err, caso.erro) {
				t.Errorf("erro = %v, quero %v", err, caso.erro)
			}
			if execucoes != caso.execucoes || contas.begins != caso.execucoes {
				t.Errorf("a função rodou %d vezes em %d transações, quero %d", execucoes, contas.begins, caso.execucoes)
			}
			if contas.commits != caso.commits {
				t.Errorf("commits = %d, quero %d", contas.commits, caso.commits)
			}
			// Toda transação aberta é fechada: o Rollback do defer roda sempre,
			// e depois de um Commit é no-op no pgx.
			if contas.rollbacks != contas.begins {
				t.Errorf("rollbacks = %d para %d transações", contas.rollbacks, contas.begins)
			}
		})
	}
}

// O Commit também pode ser a vítima: a falha de serialização chega nele, e a
// transação inteira roda de novo — uma vez.
func TestEmTransacaoRepeteOCommitQueFalha(t *testing.T) {
	serializacao := &pgconn.PgError{Code: "40001"}
	for _, caso := range []struct {
		nome      string
		erros     []error
		execucoes int
		erro      error
	}{
		{"40001 no primeiro Commit repete e comita", []error{serializacao}, 2, nil},
		{"40001 nos dois Commits desiste", []error{serializacao, serializacao}, 2, serializacao},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			contas := &contagem{errosDoCommit: caso.erros}
			execucoes := 0
			err := emTransacao(context.Background(), poolFalso{contas}, func(pgx.Tx) (bool, error) {
				execucoes++
				return true, nil
			})
			if !errors.Is(err, caso.erro) {
				t.Errorf("erro = %v, quero %v", err, caso.erro)
			}
			if execucoes != caso.execucoes || contas.commits != caso.execucoes {
				t.Errorf("rodou %d vezes com %d Commits, quero %d", execucoes, contas.commits, caso.execucoes)
			}
		})
	}
}

// Begin que falha devolve o erro sem repetir e sem rodar a função — mesmo que
// o erro fosse repetível, não há transação que o Postgres tenha escolhido
// como vítima.
func TestEmTransacaoBeginQueFalhaNaoRepete(t *testing.T) {
	falha := errors.New("pool fechado")
	contas := &contagem{erroDoBegin: falha}
	execucoes := 0
	err := emTransacao(context.Background(), poolFalso{contas}, func(pgx.Tx) (bool, error) {
		execucoes++
		return true, nil
	})
	if !errors.Is(err, falha) {
		t.Errorf("erro = %v, quero %v", err, falha)
	}
	if contas.begins != 1 || execucoes != 0 {
		t.Errorf("Begin chamado %d vezes e função %d; quero 1 e 0", contas.begins, execucoes)
	}
}
