package catalogo

import (
	"testing"
)

func TestNormalizar(t *testing.T) {
	casos := []struct {
		entrada  string
		esperado string
	}{
		{"Café", "cafe"},
		{"CAFÉ", "cafe"},
		{"Açúcar Refinado", "acucar refinado"},
		{"  Tênis   de   Corrida  ", "tenis de corrida"},
		{"O Silêncio das Marés", "o silencio das mares"},
		{"Água com Gás", "agua com gas"},
		{"Óculos de Sol Polarizado", "oculos de sol polarizado"},
		{"coração, ação e emoção", "coracao, acao e emocao"},
		{"Produto 100% Original", "produto 100% original"},
	}

	for _, c := range casos {
		t.Run(c.entrada, func(t *testing.T) {
			obtido := Normalizar(c.entrada)
			if obtido != c.esperado {
				t.Errorf("Normalizar(%q) = %q, quero %q", c.entrada, obtido, c.esperado)
			}
		})
	}
}

func TestNormalizarBusca(t *testing.T) {
	nome := "Café Gourmet em Grãos"
	descricao := "Torra média, 100% arábica com notas de chocolate."
	esperado := "cafe gourmet em graos torra media, 100% arabica com notas de chocolate."

	obtido := NormalizarBusca(nome, descricao)
	if obtido != esperado {
		t.Errorf("NormalizarBusca(...) = %q, quero %q", obtido, esperado)
	}
}
