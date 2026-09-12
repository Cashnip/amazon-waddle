package db

import (
	"context"
	"io/fs"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// Nenhuma condição de aceite desta estória é observável sem um Postgres de
// verdade: `DEFAULT uuidv7()` é função da 18, a transação da semente é o
// objeto do teste, e a pergunta sobre chave estrangeira cruzando schema é uma
// consulta ao information_schema. Um contêiner só serve a todos os casos.

const imagem = "postgres:18.6"

var schemasDeModulo = []string{"identidade", "catalogo", "pedido"}

func TestSchemaESemente(t *testing.T) {
	ctx := context.Background()
	dsn := subirPostgres(t, ctx)

	if err := plataforma.Migrar(ctx, dsn, Migracoes, DirMigracoes); err != nil {
		t.Fatalf("migrar: %v", err)
	}
	conexao := conectar(t, ctx, dsn)

	t.Run("existem só as dez tabelas do esqueleto", func(t *testing.T) {
		tem := textos(t, ctx, conexao, `
			SELECT table_schema || '.' || table_name
			FROM information_schema.tables
			WHERE table_schema = ANY($1) ORDER BY 1`, schemasDeModulo)
		quer := []string{
			"catalogo.categoria", "catalogo.produto", "catalogo.reserva_estoque", "catalogo.vendedor",
			"identidade.administrador", "identidade.comprador",
			"pedido.contador_numero", "pedido.item_pedido", "pedido.pedido", "pedido.transicao_status",
		}
		if !slices.Equal(tem, quer) {
			t.Errorf("tabelas = %v, quero %v", tem, quer)
		}
	})

	t.Run("pg_trgm é a primeira migração de catalogo", func(t *testing.T) {
		nomes, err := fs.Glob(Migracoes, DirMigracoes+"/*_catalogo_*.sql")
		if err != nil || len(nomes) == 0 {
			t.Fatalf("nenhuma migração de catalogo: %v", err)
		}
		slices.Sort(nomes) // versionadas por timestamp, então nome é ordem
		if !strings.Contains(nomes[0], "pg_trgm") {
			t.Errorf("primeira migração de catalogo = %s, quero a de pg_trgm (AD-16)", nomes[0])
		}
		if n := inteiro(t, ctx, conexao, `SELECT count(*) FROM pg_extension WHERE extname = 'pg_trgm'`); n != 1 {
			t.Errorf("pg_extension tem %d linha para pg_trgm, quero 1", n)
		}
	})

	t.Run("a busca tem coluna normalizada e os três índices da 1.4", func(t *testing.T) {
		// O índice GIN é a decisão "índice antes de serviço" como objeto no
		// banco; os dois B-tree fecham o adiado da 1.3, onde toda leitura por
		// Categoria ou por Vendedor era varredura sequencial.
		tem := map[string]string{}
		for _, def := range textos(t, ctx, conexao, `
			SELECT indexname || ' ' || indexdef
			FROM pg_indexes WHERE schemaname = 'catalogo' AND tablename = 'produto'`) {
			nome, corpo, _ := strings.Cut(def, " ")
			tem[nome] = corpo
		}
		for _, quer := range []struct{ indice, trecho string }{
			{"produto_busca_normalizada_idx", "USING gin (busca_normalizada gin_trgm_ops)"},
			{"produto_vendedor_id_idx", "(vendedor_id)"},
			{"produto_categoria_id_idx", "(categoria_id)"},
		} {
			corpo, ok := tem[quer.indice]
			if !ok {
				t.Errorf("falta o índice %s em catalogo.produto", quer.indice)
			} else if !strings.Contains(corpo, quer.trecho) {
				t.Errorf("%s = %q; quero conter %q", quer.indice, corpo, quer.trecho)
			}
		}
	})

	t.Run("as convenções de coluna do AD-2 valem em toda coluna", func(t *testing.T) {
		snake := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
		linhas, err := conexao.Query(ctx, `
			SELECT table_name, column_name, data_type
			FROM information_schema.columns
			WHERE table_schema = ANY($1) ORDER BY 1, 2`, schemasDeModulo)
		if err != nil {
			t.Fatalf("consultar colunas: %v", err)
		}
		defer linhas.Close()
		for linhas.Next() {
			var tabela, coluna, tipo string
			if err := linhas.Scan(&tabela, &coluna, &tipo); err != nil {
				t.Fatalf("ler coluna: %v", err)
			}
			for _, nome := range []string{tabela, coluna} {
				if !snake.MatchString(nome) {
					t.Errorf("%q não é snake_case", nome)
				}
				// Singular: os identificadores são os do glossário, e nenhum
				// termina em "s". Plural entrando aqui é desvio de convenção.
				// "Status" é invariável em português e é termo do glossário —
				// os dois identificadores isentos são nomeados, e não um
				// sufixo que absolveria qualquer nome futuro.
				if strings.HasSuffix(nome, "s") && !strings.HasSuffix(nome, "_centavos") &&
					nome != "transicao_status" && nome != "status" {
					t.Errorf("%q parece plural; a convenção é snake_case singular", nome)
				}
			}
			if strings.HasSuffix(coluna, "_centavos") && tipo != "bigint" {
				t.Errorf("%s.%s é %s; monetário é bigint (NFR-13)", tabela, coluna, tipo)
			}
			if strings.HasPrefix(tipo, "timestamp") && tipo != "timestamp with time zone" {
				t.Errorf("%s.%s é %s; data é timestamptz", tabela, coluna, tipo)
			}
		}
		if err := linhas.Err(); err != nil {
			t.Fatalf("percorrer colunas: %v", err)
		}
	})

	t.Run("toda chave primária é uuid DEFAULT uuidv7()", func(t *testing.T) {
		tem := textos(t, ctx, conexao, `
			SELECT c.table_name || '.' || c.column_name || ' ' || c.data_type || ' ' || coalesce(c.column_default, '<sem default>')
			FROM information_schema.table_constraints r
			JOIN information_schema.key_column_usage k
			  ON k.constraint_schema = r.constraint_schema AND k.constraint_name = r.constraint_name
			JOIN information_schema.columns c
			  ON c.table_schema = k.table_schema AND c.table_name = k.table_name AND c.column_name = k.column_name
			WHERE r.constraint_type = 'PRIMARY KEY' AND r.table_schema = ANY($1)
			ORDER BY 1`, schemasDeModulo)
		if len(tem) != 10 {
			t.Fatalf("%d chaves primárias, quero 10: %v", len(tem), tem)
		}
		for _, pk := range tem {
			// O contador do número por ano é a exceção declarada: não é
			// entidade com identidade própria, é a linha que serializa a
			// numeração de um ano, e a chave é o ano.
			if strings.HasPrefix(pk, "contador_numero.") {
				if pk != "contador_numero.ano integer <sem default>" {
					t.Errorf("chave do contador = %q; quero o ano sem default", pk)
				}
				continue
			}
			if !strings.HasSuffix(pk, "id uuid uuidv7()") {
				t.Errorf("chave primária %q; quero id uuid DEFAULT uuidv7()", pk)
			}
		}
	})

	t.Run("a Reserva tem o índice único parcial do AD-5 e o total tem CHECK", func(t *testing.T) {
		// O índice parcial é o AD-5 como objeto no banco: um Pedido tem no
		// máximo uma Reserva ATIVA por Produto, e LIBERADA/CONSOLIDADA podem
		// repetir. Sem o WHERE, cancelar e recomprar viraria conflito.
		indices := textos(t, ctx, conexao, `
			SELECT indexdef FROM pg_indexes
			WHERE schemaname = 'catalogo' AND indexname = 'reserva_estoque_ativa_idx'`)
		if len(indices) != 1 {
			t.Fatalf("reserva_estoque_ativa_idx = %v; quero exatamente um", indices)
		}
		for _, trecho := range []string{"CREATE UNIQUE INDEX", "(pedido_id, produto_id)", "WHERE (estado = 'ATIVA'"} {
			if !strings.Contains(indices[0], trecho) {
				t.Errorf("índice = %q; quero conter %q", indices[0], trecho)
			}
		}

		// total_centavos é coluna com CHECK, nunca derivação de leitura
		// (AD-3): a restrição existir no banco é o que impede um total zerado
		// ou negativo de entrar por caminho nenhum.
		restricoes := textos(t, ctx, conexao, `
			SELECT pg_get_constraintdef(oid) FROM pg_constraint
			WHERE contype = 'c' AND conrelid = 'pedido.pedido'::regclass`)
		if !slices.ContainsFunc(restricoes, func(d string) bool { return strings.Contains(d, "total_centavos > 0") }) {
			t.Errorf("CHECK de pedido.pedido = %v; quero um sobre total_centavos", restricoes)
		}
		if !slices.ContainsFunc(restricoes, func(d string) bool { return strings.Contains(d, "AGUARDANDO_PAGAMENTO") }) {
			t.Errorf("CHECK de pedido.pedido = %v; o Status é text com CHECK, e não enum", restricoes)
		}
	})

	t.Run("nenhuma chave estrangeira cruza schema", func(t *testing.T) {
		// A linha da matriz de I/O: zero linhas. É o AD-2 como pergunta ao
		// banco, e não como acordo entre quem escreve migração.
		n := inteiro(t, ctx, conexao, `
			SELECT count(*)
			FROM information_schema.table_constraints r
			JOIN information_schema.constraint_column_usage u
			  ON u.constraint_schema = r.constraint_schema AND u.constraint_name = r.constraint_name
			WHERE r.constraint_type = 'FOREIGN KEY' AND r.table_schema <> u.table_schema`)
		if n != 0 {
			t.Errorf("%d chaves estrangeiras cruzam schema; o AD-2 pede zero", n)
		}
	})

	// A partir daqui a ordem importa: a semente de verdade entra uma vez e os
	// casos seguintes observam o banco já semeado.
	t.Run("a semente entra uma vez e o segundo arranque a pula", func(t *testing.T) {
		aplicada, err := plataforma.Semear(ctx, dsn, Semente, DirSemente, VersaoSemente)
		if err != nil || !aplicada {
			t.Fatalf("primeiro Semear = %v, %v; quero true, nil", aplicada, err)
		}
		antes := catalogo(t, ctx, conexao)
		if len(antes) != 50 {
			t.Errorf("%d Produtos semeados; o Catálogo Semeado tem exatamente 50", len(antes))
		}
		// O catálogo só é comparado consigo mesmo, então uma imagem_url
		// apontando para arquivo que não existe passaria verde. A conferência é
		// no disco, e não no embed de media/: assim o pacote db não ganha
		// aresta de importação nova (AD-1).
		for _, url := range textos(t, ctx, conexao, `SELECT imagem_url FROM catalogo.produto ORDER BY id`) {
			if _, err := os.Stat("../media/" + path.Base(url)); err != nil {
				t.Errorf("imagem_url %q não tem arquivo: %v", url, err)
			}
		}
		// A normalização é na escrita (AD-16): o que o Go gravou tem de estar
		// em caixa baixa e sem diacrítico, ou o índice de trigrama indexa
		// texto que nenhum termo digitado vai casar.
		for _, consulta := range []struct {
			sql string
			n   int
		}{
			{`SELECT count(*) FROM catalogo.produto WHERE busca_normalizada = ''`, 0},
			{`SELECT count(*) FROM catalogo.produto WHERE busca_normalizada <> lower(busca_normalizada)`, 0},
			{`SELECT count(*) FROM catalogo.produto WHERE busca_normalizada ~ '[^[:ascii:]]'`, 0},
			{`SELECT count(*) FROM catalogo.categoria`, 5},
			{`SELECT count(*) FROM catalogo.vendedor`, 5},
			{`SELECT count(*) FROM identidade.comprador`, 1},
			{`SELECT count(*) FROM identidade.administrador`, 1},
			// A semente não cria Pedido, e nada nela consome `numero`.
			{`SELECT count(*) FROM catalogo.categoria WHERE categoria_pai_id IS NOT NULL`, 0},
			{`SELECT count(*) FROM pedido.pedido`, 0},
			// estoque_total entrou com padrão justamente para que a semente
			// não precisasse mudar: sem teto, a Reserva não teria com o que
			// comparar e nenhuma compra seria recusada.
			{`SELECT count(*) FROM catalogo.produto WHERE estoque_total <= 0`, 0},
		} {
			if n := inteiro(t, ctx, conexao, consulta.sql); n != consulta.n {
				t.Errorf("%s = %d, quero %d", consulta.sql, n, consulta.n)
			}
		}

		aplicada, err = plataforma.Semear(ctx, dsn, Semente, DirSemente, VersaoSemente)
		if err != nil || aplicada {
			t.Fatalf("segundo Semear = %v, %v; quero false, nil", aplicada, err)
		}
		if depois := catalogo(t, ctx, conexao); !slices.Equal(antes, depois) {
			t.Error("o segundo arranque mexeu no catálogo; a semente tem de ser idempotente")
		}
	})

	t.Run("semente que falha no meio não grava nada", func(t *testing.T) {
		const versao = "teste-falha"
		quebrada := fstest.MapFS{
			"semente/001_entra.sql": &fstest.MapFile{
				Data: []byte(`INSERT INTO catalogo.vendedor (nome) VALUES ('Fantasma');`),
			},
			"semente/002_quebra.sql": &fstest.MapFile{Data: []byte(`ISSO NÃO É SQL;`)},
		}
		if _, err := plataforma.Semear(ctx, dsn, quebrada, "semente", versao); err == nil {
			t.Fatal("SQL inválido no meio devia derrubar o arranque")
		}
		if n := inteiro(t, ctx, conexao, `SELECT count(*) FROM public.semente WHERE versao = 'teste-falha'`); n != 0 {
			t.Error("o marcador ficou gravado apesar da falha")
		}
		if n := inteiro(t, ctx, conexao, `SELECT count(*) FROM catalogo.vendedor WHERE nome = 'Fantasma'`); n != 0 {
			t.Error("o primeiro arquivo ficou gravado; a semente é uma transação só")
		}
	})

	t.Run("dois binários ao mesmo tempo: só um semeia", func(t *testing.T) {
		const versao = "teste-corrida"
		trivial := fstest.MapFS{"semente/001.sql": &fstest.MapFile{
			Data: []byte(`INSERT INTO catalogo.vendedor (nome) VALUES ('Corrida');`),
		}}
		resultados := make([]bool, 2)
		erros := make([]error, 2)
		var grupo sync.WaitGroup
		for i := range resultados {
			grupo.Go(func() {
				resultados[i], erros[i] = plataforma.Semear(ctx, dsn, trivial, "semente", versao)
			})
		}
		grupo.Wait()
		for i, err := range erros {
			if err != nil {
				t.Fatalf("binário %d falhou: %v", i, err)
			}
		}
		if resultados[0] == resultados[1] {
			t.Errorf("os dois devolveram %v; exatamente um tinha de semear", resultados[0])
		}
		if n := inteiro(t, ctx, conexao, `SELECT count(*) FROM catalogo.vendedor WHERE nome = 'Corrida'`); n != 1 {
			t.Errorf("%d linhas de 'Corrida'; quero 1", n)
		}
	})
}

// catalogo devolve o catálogo inteiro como texto ordenado — é a comparação da
// condição de aceite: mesmos identificadores, nomes e preços depois de um
// `down -v`, e nenhuma mudança entre dois arranques.
func catalogo(t *testing.T, ctx context.Context, conexao *pgx.Conn) []string {
	t.Helper()
	return textos(t, ctx, conexao, `
		SELECT id || ' ' || nome || ' ' || preco_centavos || ' ' || imagem_url
		FROM catalogo.produto ORDER BY id`)
}

func subirPostgres(t *testing.T, ctx context.Context) string {
	t.Helper()
	ctr, err := tcpostgres.Run(ctx, imagem,
		tcpostgres.WithDatabase("azamon"),
		tcpostgres.WithUsername("azamon"),
		tcpostgres.WithPassword("azamon"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(2*time.Minute)),
	)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(ctr) })
	if err != nil {
		t.Fatalf("subir %s: %v", imagem, err)
	}
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("DSN do contêiner: %v", err)
	}
	return dsn
}

func conectar(t *testing.T, ctx context.Context, dsn string) *pgx.Conn {
	t.Helper()
	conexao, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	t.Cleanup(func() { _ = conexao.Close(context.Background()) })
	return conexao
}

func textos(t *testing.T, ctx context.Context, conexao *pgx.Conn, sql string, args ...any) []string {
	t.Helper()
	linhas, err := conexao.Query(ctx, sql, args...)
	if err != nil {
		t.Fatalf("consultar: %v", err)
	}
	valores, err := pgx.CollectRows(linhas, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("ler resultado: %v", err)
	}
	return valores
}

func inteiro(t *testing.T, ctx context.Context, conexao *pgx.Conn, sql string) int {
	t.Helper()
	var n int
	if err := conexao.QueryRow(ctx, sql).Scan(&n); err != nil {
		t.Fatalf("consultar %q: %v", sql, err)
	}
	return n
}
