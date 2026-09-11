//go:build ignore

// Comando gerar escreve os dois artefatos determinísticos do Catálogo
// Semeado a partir da lista escrita à mão logo abaixo: um SVG por Produto em
// media/ e o SQL em db/semente/. Rode na raiz do repositório:
//
//	go run media/gerar.go
//
// Gerador e saída ficam os dois versionados de propósito. A construção é
// offline (AD-12) e a troca do placeholder por foto de verdade depois é
// substituir o arquivo em media/, sem tocar no banco.
//
// Os identificadores são derivados por SHA-1 do nome (o formato do uuid v5),
// e não sorteados: `DEFAULT uuidv7()` é o certo para dado nascido em execução
// e o errado para a semente, onde cada `down -v` daria identificador novo.
package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// A lista, escrita à mão. Nomes, descrições e preços em português plausível:
// o NFR-1 é uma demonstração ao vivo, e "Produto Teste 3" aparece na sala.
// ---------------------------------------------------------------------------

type categoria struct {
	nome string
	cor  string // fundo do placeholder; nunca o verde nem o laranja do DESIGN.md,
	// que carregam significado (disponibilidade e passo irreversível).
}

type produto struct {
	nome      string
	descricao string
	centavos  int64
	categoria string
	vendedor  string
}

var vendedores = []string{
	"Atlântico Importados",
	"Vale do Sol Distribuidora",
	"Sertão Livraria",
	"Casa Boa Utilidades",
	"Pampa Esportes",
}

var categorias = []categoria{
	{"Eletrônicos", "#2C3E50"},
	{"Livros", "#5B4636"},
	{"Casa e Cozinha", "#6B5B95"},
	{"Esporte e Lazer", "#3B6E8F"},
	{"Moda", "#8C3A5B"},
}

var produtos = []produto{
	// Eletrônicos
	{"Fone de Ouvido Bluetooth Aurora", "Fone sem fio com cancelamento de ruído e até 30 horas de reprodução com o estojo de carga.", 24990, "Eletrônicos", "Atlântico Importados"},
	{"Caixa de Som Portátil Maré", "Caixa Bluetooth resistente a respingos, com 12 horas de bateria e alça de transporte.", 18900, "Eletrônicos", "Atlântico Importados"},
	{"Teclado Mecânico Compacto Tucano", "Teclado 65% com switches lineares, estrutura de alumínio e cabo removível.", 32900, "Eletrônicos", "Atlântico Importados"},
	{"Mouse Sem Fio Ergonômico Curió", "Mouse vertical de seis botões, sensor de 4000 DPI e conexão por receptor USB.", 13990, "Eletrônicos", "Atlântico Importados"},
	{"Monitor 24 Polegadas Horizonte", "Monitor IPS Full HD de 75 Hz com ajuste de altura e entrada HDMI dupla.", 89900, "Eletrônicos", "Vale do Sol Distribuidora"},
	{"Carregador Rápido 65W Relâmpago", "Carregador de parede com duas saídas USB-C e uma USB-A, compatível com notebooks leves.", 15900, "Eletrônicos", "Vale do Sol Distribuidora"},
	{"Webcam Full HD Retrato", "Webcam de 1080p a 30 quadros por segundo, com microfone estéreo e tampa de privacidade.", 21900, "Eletrônicos", "Atlântico Importados"},
	{"Roteador Wi-Fi 6 Alcance", "Roteador de banda dupla com quatro antenas e porta gigabit para rede cabeada.", 45900, "Eletrônicos", "Vale do Sol Distribuidora"},
	{"Smartwatch Passo Certo", "Relógio com medição de passos, frequência cardíaca e sono, à prova d'água até 50 metros.", 39900, "Eletrônicos", "Atlântico Importados"},
	{"Power Bank 20000mAh Reserva", "Bateria portátil com carga rápida de 22,5 W e visor de porcentagem restante.", 17990, "Eletrônicos", "Vale do Sol Distribuidora"},

	// Livros
	{"O Silêncio das Marés", "Romance sobre três gerações de uma família de pescadores no litoral sul.", 4990, "Livros", "Sertão Livraria"},
	{"Cartografia do Cotidiano", "Ensaios curtos sobre os mapas invisíveis que organizam a vida de quem mora na cidade.", 5490, "Livros", "Sertão Livraria"},
	{"Manual Prático de Horta Urbana", "Guia ilustrado para cultivar temperos e hortaliças em varandas e quintais pequenos.", 6790, "Livros", "Sertão Livraria"},
	{"A Última Estação do Vale", "Narrativa de um maquinista no ramal ferroviário desativado do interior mineiro.", 4590, "Livros", "Sertão Livraria"},
	{"Introdução à Lógica de Programação", "Livro-texto com exercícios resolvidos, do primeiro algoritmo às estruturas de repetição.", 8990, "Livros", "Sertão Livraria"},
	{"Receitas de Domingo", "Sessenta receitas de almoço de família, com tempos de preparo e variações regionais.", 7490, "Livros", "Sertão Livraria"},
	{"Breve História dos Rios", "Como os grandes rios brasileiros moldaram fronteiras, cidades e economias.", 5290, "Livros", "Sertão Livraria"},
	{"O Relojoeiro de Ouro Preto", "Policial histórico ambientado nas ladeiras da cidade mineira em 1902.", 4890, "Livros", "Sertão Livraria"},
	{"Anotações de um Andarilho", "Diário de viagem a pé por sete estados, escrito em fragmentos.", 3990, "Livros", "Sertão Livraria"},
	{"Física para Curiosos", "Explicações sem fórmulas para perguntas que todo mundo já fez sobre o mundo físico.", 6990, "Livros", "Sertão Livraria"},

	// Casa e Cozinha
	{"Jogo de Panelas Antiaderente 5 Peças", "Conjunto com três panelas, frigideira e caçarola, com cabos que não esquentam.", 32900, "Casa e Cozinha", "Casa Boa Utilidades"},
	{"Liquidificador Turbo 1200W", "Liquidificador de doze velocidades com copo de vidro de 2 litros e lâmina de quatro pontas.", 18900, "Casa e Cozinha", "Casa Boa Utilidades"},
	{"Cafeteira Italiana de Alumínio", "Cafeteira para seis xícaras, feita para fogão a gás ou elétrico.", 9900, "Casa e Cozinha", "Casa Boa Utilidades"},
	{"Conjunto de Potes Herméticos 10 Peças", "Potes de plástico livre de BPA, com trava nas quatro laterais e tampa colorida.", 12900, "Casa e Cozinha", "Casa Boa Utilidades"},
	{"Tábua de Corte em Bambu", "Tábua de 38 por 28 centímetros com canaleta para líquidos e alça embutida.", 5900, "Casa e Cozinha", "Casa Boa Utilidades"},
	{"Air Fryer 4 Litros", "Fritadeira sem óleo com cesto removível, timer de 60 minutos e controle de temperatura.", 39900, "Casa e Cozinha", "Casa Boa Utilidades"},
	{"Faqueiro Inox 24 Peças", "Faqueiro para seis pessoas em aço inoxidável escovado, próprio para lava-louças.", 16900, "Casa e Cozinha", "Casa Boa Utilidades"},
	{"Jogo de Toalhas de Banho", "Quatro toalhas de algodão penteado, duas de banho e duas de rosto, alta absorção.", 11900, "Casa e Cozinha", "Casa Boa Utilidades"},
	{"Luminária de Mesa Articulada", "Luminária com braço articulado, três níveis de intensidade e base antiderrapante.", 8900, "Casa e Cozinha", "Vale do Sol Distribuidora"},
	{"Organizador de Gavetas Modular", "Seis divisórias encaixáveis para roupa íntima, meias e acessórios pequenos.", 6900, "Casa e Cozinha", "Casa Boa Utilidades"},

	// Esporte e Lazer
	{"Tênis de Corrida Leve Passada", "Tênis de amortecimento neutro com cabedal em malha respirável e solado de borracha.", 27900, "Esporte e Lazer", "Pampa Esportes"},
	{"Bola de Futebol Campo Costurada", "Bola tamanho oficial, costurada à mão, com câmara de butil e miolo reposicionável.", 12900, "Esporte e Lazer", "Pampa Esportes"},
	{"Kit de Halteres Ajustáveis 20kg", "Par de halteres com anilhas rosqueáveis e barras emborrachadas para treino em casa.", 34900, "Esporte e Lazer", "Pampa Esportes"},
	{"Tapete de Yoga Antiderrapante", "Tapete de 6 milímetros com superfície texturizada e alça para transporte.", 9900, "Esporte e Lazer", "Pampa Esportes"},
	{"Garrafa Térmica 750ml", "Garrafa de parede dupla em aço inoxidável, mantém a bebida gelada por 24 horas.", 7900, "Esporte e Lazer", "Pampa Esportes"},
	{"Corda de Pular Profissional", "Corda de aço revestido com rolamentos nos punhos e comprimento regulável.", 4900, "Esporte e Lazer", "Pampa Esportes"},
	{"Mochila de Trilha 40 Litros", "Mochila cargueira com estrutura ventilada, capa de chuva e bolso para hidratação.", 24900, "Esporte e Lazer", "Pampa Esportes"},
	{"Barraca Iglu para 2 Pessoas", "Barraca de montagem rápida, coluna d'água de 2000 milímetros e sobreteto duplo.", 29900, "Esporte e Lazer", "Pampa Esportes"},
	{"Luvas de Treino Palma Reforçada", "Luvas com palma em couro sintético e fecho ajustável no punho.", 6900, "Esporte e Lazer", "Pampa Esportes"},
	{"Bicicleta Ergométrica Vertical", "Bicicleta com oito níveis de carga, banco regulável e painel de tempo e distância.", 89900, "Esporte e Lazer", "Pampa Esportes"},

	// Moda
	{"Camiseta Algodão Pima Lisa", "Camiseta de malha fina, gola careca reforçada e caimento reto.", 7900, "Moda", "Vale do Sol Distribuidora"},
	{"Calça Jeans Reta Masculina", "Jeans de lavagem média com elastano leve e cinco bolsos.", 15900, "Moda", "Vale do Sol Distribuidora"},
	{"Vestido Midi de Viscose", "Vestido de manga curta, cintura marcada por cordão e comprimento abaixo do joelho.", 18900, "Moda", "Vale do Sol Distribuidora"},
	{"Jaqueta Corta-Vento Impermeável", "Jaqueta leve com capuz embutido, punho elástico e costura selada.", 25900, "Moda", "Vale do Sol Distribuidora"},
	{"Tênis Casual de Couro", "Tênis de couro legítimo com forro têxtil e solado de borracha costurado.", 29900, "Moda", "Vale do Sol Distribuidora"},
	{"Cinto de Couro Fivela Fosca", "Cinto de 3,5 centímetros em couro legítimo, com cinco furos de ajuste.", 8900, "Moda", "Vale do Sol Distribuidora"},
	{"Meia Cano Alto Kit 5 Pares", "Cinco pares de meia em algodão com punho sem marca e reforço no calcanhar.", 4900, "Moda", "Vale do Sol Distribuidora"},
	{"Bolsa Transversal Compacta", "Bolsa com alça regulável, bolso interno com zíper e fecho magnético.", 13900, "Moda", "Vale do Sol Distribuidora"},
	{"Óculos de Sol Polarizado", "Óculos com lente polarizada, proteção UV400 e armação em acetato.", 19900, "Moda", "Vale do Sol Distribuidora"},
	{"Boné Aba Curva Bordado", "Boné de sarja com seis gomos, aba pré-curvada e fecho traseiro regulável.", 6900, "Moda", "Vale do Sol Distribuidora"},
}

// As duas contas de demonstração do NFR-1. O hash é Argon2id no formato PHC,
// com os parâmetros do AD-9 (m=19456, t=2, p=1, sal de 16 bytes) — literal
// porque a semente é determinística, e o sal é fixo pelo mesmo motivo.
// Reproduzir: argon2.IDKey([]byte(senha), []byte(sal), 2, 19456, 1, 32).
var contas = []struct{ tabela, nome, email, senha, hash string }{
	{
		"identidade.comprador", "Joana Ribeiro", "comprador@azamon.test", "azamon-comprador",
		"$argon2id$v=19$m=19456,t=2,p=1$YXphbW9uLWRlbW8tMDAwMQ$6Q3cmzFmai+W5Dz97kfAqozB+u5MVpUkiwZdGJXJXQ8",
	},
	{
		"identidade.administrador", "Marcos Aleixo", "admin@azamon.test", "azamon-admin",
		"$argon2id$v=19$m=19456,t=2,p=1$YXphbW9uLWRlbW8tMDAwMg$rQXJzEZBGDbaO34QvCTVhxFu87lcvsuu4T6Oa85rw8k",
	},
}

// ---------------------------------------------------------------------------

const (
	dirMedia   = "media"
	arquivoSQL = "db/semente/001_catalogo_semeado.sql"
	baseURL    = "/api/v1/media/" // relativa: quem serve é o Go, no compose ou fora dele
)

func main() {
	if err := gerar(); err != nil {
		fmt.Fprintln(os.Stderr, "gerar:", err)
		os.Exit(1)
	}
}

func gerar() error {
	corDe := map[string]string{}
	for _, c := range categorias {
		corDe[c.nome] = c.cor
	}

	var sql strings.Builder
	sql.WriteString("-- Catálogo Semeado — GERADO por media/gerar.go. Não edite à mão:\n")
	sql.WriteString("-- a lista mora no gerador, e `go run media/gerar.go` reescreve este arquivo.\n")
	sql.WriteString("--\n")
	sql.WriteString("-- Todo uuid é literal. A semente roda depois das migrações, uma vez só,\n")
	sql.WriteString("-- dentro da transação que insere o marcador em public.semente.\n\n")

	sql.WriteString("INSERT INTO catalogo.vendedor (id, nome, ativo) VALUES\n")
	linhas := make([]string, 0, len(produtos))
	for _, v := range vendedores {
		linhas = append(linhas, fmt.Sprintf("  (%s, %s, true)", literal(id("vendedor", v)), literal(v)))
	}
	sql.WriteString(strings.Join(linhas, ",\n") + ";\n\n")

	sql.WriteString("-- categoria_pai_id fica nulo no MVP (addendum §6): a coluna existe para\n")
	sql.WriteString("-- que hierarquia depois não seja migração de dados.\n")
	sql.WriteString("INSERT INTO catalogo.categoria (id, nome, categoria_pai_id) VALUES\n")
	linhas = linhas[:0]
	for _, c := range categorias {
		linhas = append(linhas, fmt.Sprintf("  (%s, %s, NULL)", literal(id("categoria", c.nome)), literal(c.nome)))
	}
	sql.WriteString(strings.Join(linhas, ",\n") + ";\n\n")

	sql.WriteString("INSERT INTO catalogo.produto (id, nome, descricao, preco_centavos, imagem_url, vendedor_id, categoria_id) VALUES\n")
	linhas = linhas[:0]
	for _, p := range produtos {
		cor, ok := corDe[p.categoria]
		if !ok {
			return fmt.Errorf("produto %q aponta para categoria inexistente %q", p.nome, p.categoria)
		}
		arquivo := apelido(p.nome) + ".svg"
		if err := os.WriteFile(filepath.Join(dirMedia, arquivo), []byte(svg(p.nome, p.categoria, cor)), 0o644); err != nil {
			return err
		}
		linhas = append(linhas, fmt.Sprintf("  (%s, %s, %s, %d, %s, %s, %s)",
			literal(id("produto", p.nome)), literal(p.nome), literal(p.descricao), p.centavos,
			literal(baseURL+arquivo), literal(id("vendedor", p.vendedor)), literal(id("categoria", p.categoria))))
	}
	sql.WriteString(strings.Join(linhas, ",\n") + ";\n\n")

	sql.WriteString("-- Contas de demonstração (FR-4): duas tabelas separadas, nunca uma coluna\n")
	sql.WriteString("-- de papel. As credenciais estão no README.\n")
	for _, c := range contas {
		sql.WriteString(fmt.Sprintf("INSERT INTO %s (id, nome, email, senha_hash) VALUES\n  (%s, %s, %s, %s);\n",
			c.tabela, literal(id("conta", c.email)), literal(c.nome), literal(c.email), literal(c.hash)))
	}

	if err := os.WriteFile(arquivoSQL, []byte(sql.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("%d SVG em %s/ e %s\n", len(produtos), dirMedia, arquivoSQL)
	return nil
}

// id deriva um uuid estável do nome (formato v5, SHA-1). Estável é o ponto:
// `docker compose down -v` seguido de `up` devolve os mesmos identificadores.
func id(tipo, nome string) string {
	s := sha1.Sum([]byte("azamon:" + tipo + ":" + nome))
	b := s[:16]
	b[6] = b[6]&0x0f | 0x50 // versão 5
	b[8] = b[8]&0x3f | 0x80 // variante RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// literal cita um texto para o SQL. A aspa simples dobrada é o único escape
// que o Postgres pede em literal padrão, e toda entrada daqui é do repositório.
func literal(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// apelido reduz o nome do Produto a um nome de arquivo estável.
var semAcento = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o",
	"ú", "u", "ü", "u", "ç", "c",
)

func apelido(nome string) string {
	var b strings.Builder
	for _, r := range semAcento.Replace(strings.ToLower(nome)) {
		switch {
		case r >= 'a' && r <= 'z', unicode.IsDigit(r):
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// svg desenha o placeholder: fundo na cor da Categoria, nome do Produto no
// meio. Sem fonte externa e sem imagem embutida — a pilha do sistema resolve
// no navegador, e nada é buscado em rede (AD-12).
func svg(nome, categoria, cor string) string {
	linhas := quebrar(nome, 18)
	topo := 420 - (len(linhas)-1)*30
	var texto strings.Builder
	for i, l := range linhas {
		texto.WriteString(fmt.Sprintf(
			"\n    <tspan x=\"400\" y=\"%d\">%s</tspan>", topo+i*60, xml(l)))
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="800" height="800" viewBox="0 0 800 800" role="img" aria-label="%s">
  <title>%s</title>
  <rect width="800" height="800" fill="%s"/>
  <rect x="48" y="48" width="704" height="704" fill="none" stroke="#ffffff" stroke-opacity="0.28" stroke-width="4"/>
  <text x="400" y="150" text-anchor="middle" font-family="system-ui, -apple-system, Segoe UI, Roboto, sans-serif" font-size="26" font-weight="600" letter-spacing="6" fill="#ffffff" fill-opacity="0.72">%s</text>
  <text text-anchor="middle" font-family="system-ui, -apple-system, Segoe UI, Roboto, sans-serif" font-size="48" font-weight="700" fill="#ffffff">%s
  </text>
</svg>
`, xml(nome), xml(nome), cor, xml(strings.ToUpper(categoria)), texto.String())
}

func quebrar(texto string, largura int) []string {
	var linhas []string
	atual := ""
	for _, palavra := range strings.Fields(texto) {
		if atual == "" {
			atual = palavra
		} else if len([]rune(atual+" "+palavra)) <= largura {
			atual += " " + palavra
		} else {
			linhas = append(linhas, atual)
			atual = palavra
		}
	}
	return append(linhas, atual)
}

var xmlEscape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func xml(s string) string { return xmlEscape.Replace(s) }
