// Package internal só existe para este teste: o mecanismo do AD-1. A regra
// "um módulo importa outro apenas pela sua interface pública" só existe de
// verdade porque `go list` a verifica de forma exaustiva, nunca amostral.
package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const modulo = "github.com/Cashnip/amazon-waddle"

// arestas é a tabela do AD-1, exaustiva: origem → destinos permitidos.
// Uma seta que não está aqui é defeito; uma seta invertida é ciclo.
var arestas = map[string][]string{
	"api":        {"identidade", "catalogo", "busca", "carrinho", "pedido", "pagamento", "plataforma", "media"},
	"cmd/azamon": {"api", "db", "pedido", "pagamento", "plataforma"}, // monta o servidor; o relógio: Varrer, Expirar, SimularEntrega e EmitirConfirmacoesDevidas
	// O AD-14 manda traduzir todo sentinela num arquivo só, e esse arquivo é
	// internal/plataforma/erro — então `plataforma` alcança a porta de cada
	// módulo cujo sentinela ele traduz, e só ela. A seta contrária continua
	// proibida: módulo de domínio não conhece `plataforma/erro` (addendum §10).
	"plataforma": {"identidade", "catalogo", "carrinho", "pedido"},
	"identidade": {},
	"catalogo":   {},
	"busca":      {"catalogo"},
	"carrinho":   {"catalogo"},
	"pedido":     {"catalogo", "carrinho", "identidade", "pagamento"},
	"pagamento":  {}, // AD-7: pagamento não conhece pedido
	"db":         {},
	"media":      {}, // só os bytes das imagens do Catálogo Semeado
}

// portaPlataforma é a única isenção universal (addendum §10): o pacote exato,
// nunca a área. `internal/plataforma/erro` conhece status HTTP e é aresta
// proibida a partir de módulo de domínio — quem quiser traduzir erro é `api/`.
const portaPlataforma = modulo + "/internal/plataforma"

// dominios são os seis módulos do NFR-2. `plataforma` não está entre eles: é a
// camada de config/log/correlação/erro, importável por todos.
var dominios = []string{"identidade", "catalogo", "busca", "carrinho", "pedido", "pagamento"}

func TestFronteiraDeModulo(t *testing.T) {
	for _, pkg := range pacotesDoModulo(t) {
		origem := area(pkg.ImportPath)
		for _, imp := range importes(pkg) {
			if !strings.HasPrefix(imp, modulo+"/") {
				continue
			}
			destino := area(imp)
			if imp == portaPlataforma || destino == origem {
				continue
			}
			if ehDominio(destino) {
				porta := modulo + "/internal/" + destino
				if imp != porta {
					t.Errorf("%s importa %s; a única porta de %s é %s (AD-1)", pkg.ImportPath, imp, destino, porta)
					continue
				}
			}
			// A tabela é exaustiva: vale para toda aresta, e não só para as
			// que chegam num módulo de domínio. Sem isto, catalogo → api
			// passaria calado, que é a inversão de camada mais fácil de fazer.
			if !permitida(origem, destino) {
				t.Errorf("%s importa %s: aresta %s → %s não está na tabela do AD-1", pkg.ImportPath, imp, origem, destino)
			}
		}
	}
}

// TestDominioNaoConheceHTTP: a tradução mora em api/, e só lá.
func TestDominioNaoConheceHTTP(t *testing.T) {
	for _, pkg := range pacotesDoModulo(t) {
		if !ehDominio(area(pkg.ImportPath)) {
			continue
		}
		for _, imp := range importes(pkg) {
			if imp == "net/http" || strings.HasPrefix(imp, "net/http/") {
				t.Errorf("%s importa net/http; módulo de domínio não conhece HTTP", pkg.ImportPath)
			}
		}
	}
}

// dirArquitetura é onde moram os dois desenhos do AD-1. O teste roda de
// internal/, então o caminho sobe um nível. Renomear ou encurtar esse
// diretório de planejamento quebra este teste: é a saída estrutural que a
// entrada da 7.1 no deferred-work.md (clone com `Filename too long` no Windows)
// deixou para o time, e quem a tomar precisa trocar esta constante junto.
//
// O teste lê os arquivos pelo repositório montado (o `-v "$PWD":/src` do
// comando de verificação). O `.dockerignore` exclui `_bmad-output` e `*.md`,
// então ele não roda dentro de uma imagem construída a partir do contexto do
// repositório: lá os dois desenhos não existem.
const dirArquitetura = "../_bmad-output/planning-artifacts/architecture/architecture-azamon-2026-09-05/"

// desenhos são os dois arquivos que carregam o grafo do AD-1: a espinha, que é
// o substrato que os agentes leem, e o DIAGRAMA-MODULOS, que se lê sozinho e é
// o entregável do §6.1 do PRD. O bloco é um só, copiado byte a byte.
var desenhos = []string{"ARCHITECTURE-SPINE.md", "DIAGRAMA-MODULOS.md"}

// marcaDoAD1 abre o bloco mermaid que este teste confere: a primeira linha do
// bloco é a marca exata, ou a marca seguida de espaço. `%% AD-12` não é o
// AD-1. Os outros blocos mermaid da espinha (topologia, entidades) não a
// carregam e ficam de fora.
const marcaDoAD1 = "%% AD-1"

// foraDaConferencia são os nós desenhados que não têm área na tabela: `web` é
// o Next.js, e a porta e o Provedor Simulado moram dentro de `pagamento`. Uma
// seta que toque um deles não é conferida — e só elas.
var foraDaConferencia = []string{"web", "porta", "simulado"}

// noDoRelogio é o id do nó de cmd/azamon: barra não cabe em id de mermaid.
const noDoRelogio = "relogio"

// A sintaxe mermaid que a guarda aceita é pequena de propósito, e por linha:
//   - vazia, comentário `%%`, `graph …` ou `flowchart …`;
//   - declaração de nó: id["rótulo"], id{{"rótulo"}} ou id("rótulo");
//   - seta entre dois ids já declarados: a --> b, a -.-> b, a -->|"rótulo"| b
//     ou a -.->|"rótulo"| b.
//
// Todo o resto falha alto, e não é ignorado: `style`, `classDef`, `linkStyle`,
// `subgraph`/`end`, seta grossa `==>`, encadeamento com `&` e nó declarado na
// própria linha da seta. Aceitar o que não se sabe ler seria deixar uma seta
// passar sem conferência.
var (
	// a --> b, a -.-> b e a -->|"rótulo"| b, com os nós já declarados antes.
	reSeta = regexp.MustCompile(`^([A-Za-z_]\w*)\s+(?:-->|-\.->)(?:\|[^|]*\|)?\s+([A-Za-z_]\w*)$`)
	// a["…"], a{{"…"}}, a("…"): a declaração do nó, que carrega o rótulo.
	reNo = regexp.MustCompile(`^([A-Za-z_]\w*)\s*(?:\[|\{\{|\()`)
)

// TestDiagramaEhATabela faz de "`fronteira_test.go` passando é a prova" algo
// literal (SM-C4): o desenho do AD-1 tem o mesmo conjunto de setas que a
// tabela `arestas`, nos dois sentidos, e os dois arquivos que o carregam têm o
// mesmo bloco. Sem isto, o teste de fronteira conferia a tabela que carrega, e
// o desenho envelhecia calado — foi o que aconteceu entre 2026-09-06 e a 7.2.
// Os rótulos das setas não são conferidos aqui: nomeiam símbolos exportados, e
// quem os confere é `grep`.
func TestDiagramaEhATabela(t *testing.T) {
	blocos := make([]string, len(desenhos))
	for i, nome := range desenhos {
		blocos[i] = blocoDoAD1(t, nome)
	}
	for _, p := range problemasDosDesenhos(blocos, arestas) {
		t.Error(p)
	}
}

// TestGuardaDoDiagrama prova que a guarda acima morde: cada linha da matriz da
// 7.2 é uma mutação do desenho real ou da tabela real, e cada uma tem de
// produzir o problema previsto. Uma guarda que passa sempre seria a mesma boa
// vontade que ela veio substituir.
func TestGuardaDoDiagrama(t *testing.T) {
	bloco := blocoDoAD1(t, desenhos[1])
	comAresta := map[string][]string{}
	for origem, destinos := range arestas {
		comAresta[origem] = slices.Clone(destinos)
	}
	comAresta["busca"] = append(comAresta["busca"], "identidade")
	// Tirar uma seta CONFERIDA da cópia da espinha: `simulado -.-> api`, a
	// última linha, fica fora da conferência e provaria só a diferença.
	const setaConferida = "\n    pedido -->|\"BuscarEndereco · BuscarComprador\"| identidade"
	if !strings.Contains(bloco, setaConferida) {
		t.Fatalf("o desenho real não tem mais a linha %q; atualize este caso", setaConferida)
	}
	soAEspinha := strings.Replace(bloco, setaConferida, "", 1)
	setaAMais := bloco + "\n    carrinho --> identidade"
	noDesconhecido := bloco + "\n    xpto --> api"
	setaIlegivel := bloco + "\n    a[\"x\"] --> b"
	linhaEstranha := bloco + "\n    classDef x fill:#fff"

	casos := []struct {
		nome   string
		blocos []string
		tabela map[string][]string
		quer   []string // trechos que algum problema tem de conter; nenhum = nenhum problema
	}{
		{"em dia", []string{bloco, bloco}, arestas, nil},
		{"aresta nova no código", []string{bloco, bloco}, comAresta,
			[]string{"a aresta busca → identidade está na tabela arestas e falta no desenho"}},
		{"desenho com seta a mais", []string{setaAMais, setaAMais}, arestas,
			[]string{"a seta carrinho → identidade está no desenho e não está na tabela"}},
		{"só a espinha editada", []string{soAEspinha, bloco}, arestas,
			[]string{"são diferentes", "ARCHITECTURE-SPINE.md: a aresta pedido → identidade está na tabela arestas e falta no desenho"}},
		{"nó desconhecido", []string{noDesconhecido, noDesconhecido}, arestas,
			[]string{`o nó "xpto" não é área`}},
		{"seta com nó declarado na linha", []string{setaIlegivel, setaIlegivel}, arestas,
			[]string{"seta ilegível"}},
		{"linha que a guarda não sabe ler", []string{linhaEstranha, linhaEstranha}, arestas,
			[]string{"linha não reconhecida"}},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			problemas := problemasDosDesenhos(c.blocos, c.tabela)
			if len(c.quer) == 0 {
				if len(problemas) != 0 {
					t.Errorf("quero nenhum problema, veio %q", problemas)
				}
				return
			}
			for _, quer := range c.quer {
				if !slices.ContainsFunc(problemas, func(p string) bool { return strings.Contains(p, quer) }) {
					t.Errorf("quero um problema com %q, veio %q", quer, problemas)
				}
			}
		})
	}
}

// TestExtracaoDoAD1 prende a leitura do bloco: marca exata na primeira linha,
// um bloco só por arquivo, e bloco que não fecha é erro, nunca silêncio.
func TestExtracaoDoAD1(t *testing.T) {
	const ad1 = "```mermaid\n%% AD-1 — conferido\ngraph TD\n    a --> b\n```\n"
	const ad12 = "```mermaid\n%% AD-12\ngraph TD\n    c --> d\n```\n"
	const outro = "```mermaid\ngraph LR\n    e --> f\n```\n"
	casos := []struct {
		nome  string
		texto string
		quer  string // corpo esperado; vazio = erro esperado
		erro  string // trecho do erro
	}{
		{"um bloco, com CRLF", strings.ReplaceAll("# t\n\n"+ad1, "\n", "\r\n"), "%% AD-1 — conferido\ngraph TD\n    a --> b", ""},
		{"marca exata sem sufixo", "```mermaid\n%% AD-1\ngraph TD\n```\n", "%% AD-1\ngraph TD", ""},
		{"AD-12 não é AD-1", ad12 + ad1 + outro, "%% AD-1 — conferido\ngraph TD\n    a --> b", ""},
		{"nenhum bloco", "# nada\n" + outro, "", "0 blocos mermaid"},
		{"só o AD-12", ad12, "", "0 blocos mermaid"},
		{"dois blocos do AD-1", ad1 + ad12 + ad1, "", "2 blocos mermaid"},
		{"bloco que não fecha", "```mermaid\n%% AD-1\ngraph TD\n    a --> b\n", "", "não fecha"},
		{"bloco que não fecha antes do próximo", "```mermaid\n%% AD-1\ngraph TD\n" + outro, "", "não fecha"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			bloco, err := blocoUnicoDoAD1("x.md", c.texto)
			if c.erro != "" {
				if err == nil || !strings.Contains(err.Error(), c.erro) {
					t.Errorf("quero erro com %q, veio bloco %q e erro %v", c.erro, bloco, err)
				}
				return
			}
			if err != nil || bloco != c.quer {
				t.Errorf("quero %q, veio %q e erro %v", c.quer, bloco, err)
			}
		})
	}
}

// problemasDosDesenhos confere os blocos dos dois arquivos, na ordem de
// `desenhos`: iguais entre si, e com as setas da tabela. Quando divergem, os
// dois são conferidos, para o problema nomear qual dos lados envelheceu.
func problemasDosDesenhos(blocos []string, tabela map[string][]string) []string {
	if blocos[0] == blocos[1] {
		return problemasDoBloco(desenhos[1], blocos[1], tabela)
	}
	problemas := []string{fmt.Sprintf("os blocos %q de %s e %s são diferentes; o desenho do AD-1 é um só, copiado byte a byte",
		marcaDoAD1, desenhos[0], desenhos[1])}
	for i, nome := range desenhos {
		problemas = append(problemas, problemasDoBloco(nome, blocos[i], tabela)...)
	}
	return problemas
}

// blocoDoAD1 lê o arquivo de `dirArquitetura` e devolve o seu bloco do AD-1.
func blocoDoAD1(t *testing.T, nome string) string {
	t.Helper()
	bruto, err := os.ReadFile(dirArquitetura + nome)
	if err != nil {
		t.Fatalf("ler %s: %v", nome, err)
	}
	bloco, err := blocoUnicoDoAD1(nome, string(bruto))
	if err != nil {
		t.Fatal(err)
	}
	return bloco
}

// blocoUnicoDoAD1 exige exatamente um bloco do AD-1 no texto.
func blocoUnicoDoAD1(nome, texto string) (string, error) {
	blocos, err := blocosDoAD1(texto)
	if err != nil {
		return "", fmt.Errorf("%s: %w", nome, err)
	}
	if len(blocos) != 1 {
		return "", fmt.Errorf("%s: %d blocos mermaid começam pela marca %q; quero exatamente um", nome, len(blocos), marcaDoAD1)
	}
	return blocos[0], nil
}

// blocosDoAD1 devolve o corpo de cada bloco mermaid cuja primeira linha é a
// marca do AD-1 (exata, ou seguida de espaço), com CRLF normalizado: o
// autocrlf do Windows não é diferença entre os dois desenhos. Bloco da marca
// que não fecha é erro.
func blocosDoAD1(texto string) ([]string, error) {
	texto = strings.ReplaceAll(texto, "\r\n", "\n")
	var achados []string
	for _, trecho := range strings.Split(texto, "```mermaid\n")[1:] {
		primeira, _, _ := strings.Cut(trecho, "\n")
		if primeira != marcaDoAD1 && !strings.HasPrefix(primeira, marcaDoAD1+" ") {
			continue
		}
		fim := strings.Index(trecho, "\n```")
		if fim < 0 {
			return nil, fmt.Errorf("o bloco %q não fecha", primeira)
		}
		achados = append(achados, trecho[:fim])
	}
	return achados, nil
}

// TestTabelaEhOCodigo é o sentido que faltava: a tabela ⊆ o código. Toda
// aresta de `arestas` precisa de ao menos um importe real da área de origem
// para a de destino. Sem isto, uma aresta que o código deixou de usar
// continuaria permitida, e desenhada, para sempre — a tabela diria que existe
// uma dependência que não existe.
func TestTabelaEhOCodigo(t *testing.T) {
	observadas := arestasObservadas(pacotesDoModulo(t))
	for _, p := range arestasSemImporte(observadas, arestas) {
		t.Error(p)
	}
	t.Run("aresta sem importe é apontada", func(t *testing.T) {
		comAresta := map[string][]string{}
		for origem, destinos := range arestas {
			comAresta[origem] = slices.Clone(destinos)
		}
		comAresta["busca"] = append(comAresta["busca"], "identidade")
		problemas := arestasSemImporte(observadas, comAresta)
		if !slices.ContainsFunc(problemas, func(p string) bool { return strings.Contains(p, "busca → identidade") }) {
			t.Errorf("quero um problema com busca → identidade, veio %q", problemas)
		}
	})
}

// arestasObservadas reduz o grafo de importação às arestas entre áreas. A
// raiz isenta `internal/plataforma` conta aqui como `→ plataforma`: é ela que
// sustenta `cmd/azamon → plataforma`, e a isenção do TestFronteiraDeModulo
// diz que ela é sempre permitida, não que ela não é aresta.
func arestasObservadas(pacotes []pacote) map[string]bool {
	observadas := map[string]bool{}
	for _, pkg := range pacotes {
		origem := area(pkg.ImportPath)
		for _, imp := range importes(pkg) {
			if !strings.HasPrefix(imp, modulo+"/") {
				continue
			}
			if destino := area(imp); destino != origem {
				observadas[origem+" → "+destino] = true
			}
		}
	}
	return observadas
}

// arestasSemImporte devolve, em ordem, as arestas da tabela que nenhum
// importe observado sustenta.
func arestasSemImporte(observadas map[string]bool, tabela map[string][]string) []string {
	var problemas []string
	for _, origem := range slices.Sorted(maps.Keys(tabela)) {
		for _, destino := range tabela[origem] {
			if a := origem + " → " + destino; !observadas[a] {
				problemas = append(problemas, fmt.Sprintf("a aresta %s está na tabela arestas e nenhum importe do código a sustenta", a))
			}
		}
	}
	return problemas
}

// problemasDoBloco lê as setas do bloco, reduz cada nó à sua área e compara o
// conjunto com a tabela nos dois sentidos.
func problemasDoBloco(nome, bloco string, tabela map[string][]string) []string {
	var problemas []string
	areas := map[string]bool{}
	permitidas := map[string]bool{}
	for origem, destinos := range tabela {
		areas[origem] = true
		for _, d := range destinos {
			areas[d] = true
			permitidas[origem+" → "+d] = true
		}
	}
	desconhecidos := map[string]bool{}
	// areaDoNo devolve a área do nó e se a seta que o toca é conferida.
	areaDoNo := func(id string) (string, bool) {
		switch {
		case slices.Contains(foraDaConferencia, id):
			return "", false
		case id == noDoRelogio:
			return "cmd/azamon", true
		case areas[id]:
			return id, true
		}
		desconhecidos[id] = true
		return "", false
	}

	desenhadas := map[string]bool{}
	for i, linha := range strings.Split(bloco, "\n") {
		linha = strings.TrimSpace(linha)
		switch {
		case linha == "", strings.HasPrefix(linha, "%%"),
			strings.HasPrefix(linha, "graph "), strings.HasPrefix(linha, "flowchart "):
			continue
		}
		if m := reSeta.FindStringSubmatch(linha); m != nil {
			origem, conta := areaDoNo(m[1])
			destino, contaDestino := areaDoNo(m[2])
			if conta && contaDestino {
				desenhadas[origem+" → "+destino] = true
			}
			continue
		}
		if strings.Contains(linha, "-->") || strings.Contains(linha, "-.->") {
			problemas = append(problemas, fmt.Sprintf("%s, linha %d do bloco: seta ilegível %q; declare os nós antes e use a --> b, a -.-> b ou a -->|\"…\"| b", nome, i+1, linha))
			continue
		}
		if m := reNo.FindStringSubmatch(linha); m != nil {
			areaDoNo(m[1])
			continue
		}
		problemas = append(problemas, fmt.Sprintf("%s, linha %d do bloco: linha não reconhecida %q", nome, i+1, linha))
	}

	for _, id := range slices.Sorted(maps.Keys(desconhecidos)) {
		problemas = append(problemas, fmt.Sprintf("%s: o nó %q não é área da tabela do AD-1 (%q é cmd/azamon) nem um de %v", nome, id, noDoRelogio, foraDaConferencia))
	}
	for _, a := range slices.Sorted(maps.Keys(permitidas)) {
		if !desenhadas[a] {
			problemas = append(problemas, fmt.Sprintf("%s: a aresta %s está na tabela arestas e falta no desenho", nome, a))
		}
	}
	for _, a := range slices.Sorted(maps.Keys(desenhadas)) {
		if !permitidas[a] {
			problemas = append(problemas, fmt.Sprintf("%s: a seta %s está no desenho e não está na tabela arestas (AD-1)", nome, a))
		}
	}
	return problemas
}

type pacote struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
}

// importes junta os três: `go list` põe o que só o teste importa em
// TestImports e XTestImports, e uma fronteira que o teste pode atravessar não
// é fronteira — é onde o atalho acontece primeiro.
func importes(p pacote) []string {
	return slices.Concat(p.Imports, p.TestImports, p.XTestImports)
}

func pacotesDoModulo(t *testing.T) []pacote {
	t.Helper()
	saida, err := exec.Command("go", "list", "-deps", "-json", modulo+"/...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	dec := json.NewDecoder(bytes.NewReader(saida))
	var pacotes []pacote
	for {
		var p pacote
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("decodificar go list: %v", err)
		}
		if strings.HasPrefix(p.ImportPath, modulo+"/") {
			pacotes = append(pacotes, p)
		}
	}
	if len(pacotes) == 0 {
		t.Fatal("go list não devolveu nenhum pacote do módulo")
	}
	return pacotes
}

// area reduz o caminho do pacote à sua área na tabela do AD-1:
// internal/pedido/db/gerado → pedido; api/dto → api; db → db.
func area(caminho string) string {
	resto := strings.TrimPrefix(caminho, modulo+"/")
	resto = strings.TrimPrefix(resto, "internal/")
	partes := strings.Split(resto, "/")
	if partes[0] == "cmd" && len(partes) > 1 {
		return "cmd/" + partes[1]
	}
	return partes[0]
}

func ehDominio(nome string) bool { return slices.Contains(dominios, nome) }

func permitida(origem, destino string) bool { return slices.Contains(arestas[origem], destino) }
