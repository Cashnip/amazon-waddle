package plataforma

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"
	"slices"
)

// Semear aplica o Catálogo Semeado depois das migrações, numa transação só, e
// devolve se foi esta chamada que o aplicou. Todo erro daqui derruba o
// arranque: semente pela metade é pior que banco vazio.
//
// A decisão de semear é a inserção do marcador, não uma consulta prévia — ler
// public.semente e depois decidir é a corrida clássica, com dois binários
// lendo vazio e semeando os dois. Com ON CONFLICT DO NOTHING dentro da mesma
// transação, quem inseriu a linha aplica o SQL e quem não inseriu segue em
// frente, sem trava separada.
func Semear(ctx context.Context, dsn string, arquivos fs.FS, dir, versao string) (bool, error) {
	nomes, err := fs.Glob(arquivos, path.Join(dir, "*.sql"))
	if err != nil {
		return false, fmt.Errorf("listar a semente: %w", err)
	}
	if len(nomes) == 0 {
		// Embed quebrado não pode passar por "catálogo vazio": marcaria a
		// versão como aplicada e o próximo arranque nem tentaria de novo.
		return false, fmt.Errorf("nenhum .sql em %s", dir)
	}
	slices.Sort(nomes) // ordem do nome do arquivo é a ordem de aplicação

	bd, err := sql.Open("pgx", dsn)
	if err != nil {
		return false, fmt.Errorf("abrir conexão: %w", err)
	}
	defer bd.Close()

	tx, err := bd.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("abrir transação da semente: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op depois do commit

	res, err := tx.ExecContext(ctx, "INSERT INTO public.semente (versao) VALUES ($1) ON CONFLICT DO NOTHING", versao)
	if err != nil {
		return false, fmt.Errorf("marcar a semente: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return false, fmt.Errorf("marcar a semente: %w", err)
	} else if n == 0 {
		return false, nil // alguém já semeou esta versão
	}

	for _, nome := range nomes {
		conteudo, err := fs.ReadFile(arquivos, nome)
		if err != nil {
			return false, fmt.Errorf("ler %s: %w", nome, err)
		}
		// Um Exec por arquivo, com o arquivo inteiro dentro: sem argumento, o
		// pgx cai no protocolo simples, que é o único que aceita vários
		// comandos numa chamada só. Passar um argumento aqui mudaria o
		// protocolo e quebraria todo arquivo com mais de um comando.
		if _, err := tx.ExecContext(ctx, string(conteudo)); err != nil {
			return false, fmt.Errorf("aplicar %s: %w", nome, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("gravar a semente: %w", err)
	}
	return true, nil
}
