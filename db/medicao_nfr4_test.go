package db

import (
	"context"
	"encoding/json"
	"maps"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// TestMedicaoNFR4 é a verificação 4 do passo 0: com 5.000 Produtos, a consulta
// com termo, Categoria e faixa de preço tem de responder em p95 ≤ 500 ms
// (NFR-4, SM-4). O número que sai daqui é o que fecha a decisão "índice antes
// de serviço" no addendum §10 — sem ele, a escolha entre índice, cache e
// Elasticsearch seria adivinhação em novembro.
//
// A consulta é sonda de medição, não funcionalidade: a busca de verdade, com
// paginação e rota, é da Épica 3.

const (
	tetoP95ms  = 500 // NFR-4
	repeticoes = 20  // × 5 casos = 100 execuções, o bastante para um p95 honesto
)

// sonda é a forma que a Épica 3 vai implementar: LIKE sobre a coluna
// normalizada (o que o índice GIN pg_trgm serve), mais os dois filtros.
const sonda = `
	SELECT id, nome, preco_centavos
	FROM catalogo.produto
	WHERE busca_normalizada LIKE '%' || $1::text || '%'
	  AND categoria_id = (SELECT id FROM catalogo.categoria WHERE nome = $2::text)
	  AND preco_centavos BETWEEN $3::bigint AND $4::bigint
	ORDER BY preco_centavos, id
	LIMIT 20`

// sondaSoTermo é a mesma busca sem Categoria escolhida — a Vitrine com termo
// digitado e nenhum filtro. É onde o índice GIN pg_trgm tem chance de ser a
// única via de acesso, e por isso é ela que diz se ele serve para alguma
// coisa nesta escala.
const sondaSoTermo = `
	SELECT id, nome, preco_centavos
	FROM catalogo.produto
	WHERE busca_normalizada LIKE '%' || $1::text || '%'
	ORDER BY preco_centavos, id
	LIMIT 20`

func TestMedicaoNFR4(t *testing.T) {
	ctx := context.Background()
	dsn := subirPostgres(t, ctx)

	if err := plataforma.Migrar(ctx, dsn, Migracoes, DirMigracoes); err != nil {
		t.Fatalf("migrar: %v", err)
	}
	if _, err := plataforma.Semear(ctx, dsn, Semente, DirSemente, VersaoSemente); err != nil {
		t.Fatalf("Catálogo Semeado: %v", err)
	}
	if _, err := plataforma.Semear(ctx, dsn, SementeGrande, DirSementeGrande, VersaoSementeGrande); err != nil {
		t.Fatalf("conjunto de medição: %v", err)
	}
	conexao := conectar(t, ctx, dsn)

	if n := inteiro(t, ctx, conexao, `SELECT count(*) FROM catalogo.produto`); n != 5050 {
		t.Fatalf("%d Produtos; quero 5050 — 50 do Catálogo Semeado e 5.000 do conjunto de medição", n)
	}
	// Sem ANALYZE o planejador ainda acha que a tabela tem uma página, e o
	// plano medido não seria o que a demonstração roda.
	if _, err := conexao.Exec(ctx, `ANALYZE catalogo.produto`); err != nil {
		t.Fatalf("ANALYZE: %v", err)
	}

	casos := []struct {
		termo, categoria string
		min, max         int64
	}{
		{"fone", "Eletrônicos", 1000, 200000},
		{"livro", "Livros", 1000, 200000},
		{"panela", "Casa e Cozinha", 1000, 200000},
		{"tenis", "Esporte e Lazer", 1000, 200000},
		// A linha da matriz: termo de dois caracteres não extrai trigrama e
		// degrada para varredura. Com 5.000 linhas isso ainda cabe no teto.
		{"ca", "Moda", 1000, 200000},
	}

	planos := map[string]string{}
	var comFiltro, soTermo []float64
	for range repeticoes {
		for _, c := range casos {
			ms, plano := explicar(t, ctx, conexao, sonda, c.termo, c.categoria, c.min, c.max)
			comFiltro = append(comFiltro, ms)
			planos["termo+categoria+preço "+c.termo] = plano

			ms, plano = explicar(t, ctx, conexao, sondaSoTermo, c.termo)
			soTermo = append(soTermo, ms)
			planos["só termo "+c.termo] = plano
		}
	}

	p95 := percentil95(comFiltro)
	t.Logf("NFR-4, termo + Categoria + faixa de preço sobre 5.050 Produtos: %d execuções · mediana %.2f ms · p95 %.2f ms · máximo %.2f ms",
		len(comFiltro), mediana(comFiltro), p95, slices.Max(comFiltro))
	t.Logf("NFR-4, só termo: %d execuções · mediana %.2f ms · p95 %.2f ms · máximo %.2f ms",
		len(soTermo), mediana(soTermo), percentil95(soTermo), slices.Max(soTermo))
	for _, chave := range slices.Sorted(maps.Keys(planos)) {
		t.Logf("plano de %s: %s", chave, planos[chave])
	}
	if p95 > tetoP95ms {
		t.Errorf("p95 = %.2f ms; o NFR-4 pede ≤ %d ms", p95, tetoP95ms)
	}
}

func percentil95(v []float64) float64 {
	ordenado := slices.Sorted(slices.Values(v))
	return ordenado[int(math.Ceil(0.95*float64(len(ordenado))))-1]
}

func mediana(v []float64) float64 {
	ordenado := slices.Sorted(slices.Values(v))
	return ordenado[len(ordenado)/2]
}

// explicar roda a sonda sob EXPLAIN ANALYZE e devolve o tempo de execução
// medido pelo próprio Postgres e a árvore do plano achatada. O tempo vem do
// banco, e não de um cronômetro em Go, para não medir junto a ida e volta da
// rede. A árvore inteira, e não só o nó de topo: o que interessa registrar no
// addendum é por onde o Postgres chegou em catalogo.produto, e esse nó está
// três níveis abaixo do Limit.
func explicar(t *testing.T, ctx context.Context, conexao *pgx.Conn, consulta string, args ...any) (float64, string) {
	t.Helper()
	var bruto []byte
	if err := conexao.QueryRow(ctx, `EXPLAIN (ANALYZE, FORMAT JSON)`+consulta, args...).Scan(&bruto); err != nil {
		t.Fatalf("EXPLAIN ANALYZE com %v: %v", args, err)
	}
	var saida []struct {
		TempoDeExecucao float64        `json:"Execution Time"`
		Plano           map[string]any `json:"Plan"`
	}
	if err := json.Unmarshal(bruto, &saida); err != nil || len(saida) == 0 {
		t.Fatalf("ler o EXPLAIN de %v: %v", args, err)
	}
	return saida[0].TempoDeExecucao, strings.Join(achatar(saida[0].Plano), " · ")
}

// achatar percorre o plano em profundidade e devolve um nó por elemento, com
// o índice usado quando há um.
func achatar(no map[string]any) []string {
	tipo, _ := no["Node Type"].(string)
	if indice, ok := no["Index Name"].(string); ok {
		tipo += " em " + indice
	}
	if relacao, ok := no["Relation Name"].(string); ok {
		tipo += " sobre " + relacao
	}
	nos := []string{tipo}
	filhos, _ := no["Plans"].([]any)
	for _, filho := range filhos {
		if m, ok := filho.(map[string]any); ok {
			nos = append(nos, achatar(m)...)
		}
	}
	return nos
}
