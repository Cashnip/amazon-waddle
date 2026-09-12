// Package internal só existe para este teste: o mecanismo do AD-1. A regra
// "um módulo importa outro apenas pela sua interface pública" só existe de
// verdade porque `go list` a verifica de forma exaustiva, nunca amostral.
package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

const modulo = "github.com/Cashnip/amazon-waddle"

// arestas é a tabela do AD-1, exaustiva: origem → destinos permitidos.
// Uma seta que não está aqui é defeito; uma seta invertida é ciclo.
var arestas = map[string][]string{
	"api":        {"identidade", "catalogo", "busca", "carrinho", "pedido", "pagamento", "plataforma", "media"},
	"cmd/azamon": {"api", "db", "pedido", "pagamento", "plataforma"}, // monta o servidor; o relógio: Varrer() e EmitirDevidas()
	// O AD-14 manda traduzir todo sentinela num arquivo só, e esse arquivo é
	// internal/plataforma/erro — então `plataforma` alcança a porta de cada
	// módulo cujo sentinela ele traduz, e só ela. A seta contrária continua
	// proibida: módulo de domínio não conhece `plataforma/erro` (addendum §10).
	"plataforma": {"identidade"},
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
