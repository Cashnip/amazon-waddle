package catalogo

import (
	"strings"
)

// semAcento mapeia caracteres acentuados comuns em português para suas versões sem acento.
var semAcento = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c",
	"ñ", "n",
)

// Normalizar converte o texto para minúsculas, remove acentos/diacríticos e
// colapsa múltiplos espaços em branco para um único espaço. É usado na escrita
// para alimentar a coluna busca_normalizada (AD-16).
func Normalizar(s string) string {
	minuscula := strings.ToLower(s)
	desacentuada := semAcento.Replace(minuscula)
	campos := strings.Fields(desacentuada)
	return strings.Join(campos, " ")
}

// NormalizarBusca combina nome e descrição de um Produto em uma única string
// normalizada para busca textual rápida.
func NormalizarBusca(nome, descricao string) string {
	return Normalizar(nome + " " + descricao)
}
