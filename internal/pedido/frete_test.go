package pedido

import (
	"errors"
	"testing"
)

// isencaoDeTeste é o padrão do §7.1 (R$ 299,00).
const isencaoDeTeste = 29900

// faixasDeTeste é a tabela da migração, na ordem em que a consulta a entrega
// (padrão por último). A regra lida do banco de verdade é provada em
// api/frete_test.go; aqui é a função pura.
func faixasDeTeste() []FaixaDeFrete {
	return []FaixaDeFrete{
		{CEPInicio: "01000000", CEPFim: "39999999", Regiao: "Sudeste", ValorCentavos: 1500},
		{CEPInicio: "40000000", CEPFim: "65999999", Regiao: "Nordeste", ValorCentavos: 3000},
		{CEPInicio: "66000000", CEPFim: "69999999", Regiao: "Norte", ValorCentavos: 3500},
		{CEPInicio: "70000000", CEPFim: "76799999", Regiao: "Centro-Oeste", ValorCentavos: 2500},
		{CEPInicio: "76800000", CEPFim: "76999999", Regiao: "Norte", ValorCentavos: 3500},
		{CEPInicio: "77000000", CEPFim: "77999999", Regiao: "Norte", ValorCentavos: 3500},
		{CEPInicio: "78000000", CEPFim: "79999999", Regiao: "Centro-Oeste", ValorCentavos: 2500},
		{CEPInicio: "80000000", CEPFim: "99999999", Regiao: "Sul", ValorCentavos: 2000},
		{Regiao: "Região padrão", ValorCentavos: 4000, Padrao: true},
	}
}

func TestCalcularFrete(t *testing.T) {
	casos := []struct {
		nome     string
		cep      string
		subtotal int64
		regiao   string
		valor    int64
	}{
		{"na faixa", "01310100", 18900, "Sudeste", 1500},
		{"fora de faixa cai na padrão, não em erro", "00000001", 18900, "Região padrão", 4000},
		{"limite exato do limiar é isento", "01310100", isencaoDeTeste, "Sudeste", 0},
		{"um centavo abaixo do limiar paga", "01310100", isencaoDeTeste - 1, "Sudeste", 1500},
		{"acima do limiar, fora de faixa, também é isento", "00000001", isencaoDeTeste + 1, "Região padrão", 0},
		{"última da faixa Sudeste", "39999999", 100, "Sudeste", 1500},
		{"primeira da faixa Nordeste", "40000000", 100, "Nordeste", 3000},
		{"Rondônia é Norte, entre duas faixas do Centro-Oeste", "76850000", 100, "Norte", 3500},
		{"Carrinho vazio paga a região", "90000000", 0, "Sul", 2000},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			frete, err := CalcularFrete(faixasDeTeste(), c.cep, c.subtotal, isencaoDeTeste)
			if err != nil {
				t.Fatalf("CalcularFrete: %v", err)
			}
			if frete.Regiao != c.regiao || frete.ValorCentavos != c.valor {
				t.Errorf("CalcularFrete(%s, %d) = %+v; quero %s, %d", c.cep, c.subtotal, frete, c.regiao, c.valor)
			}
		})
	}
}

// Mesmo CEP e mesmo valor dão sempre o mesmo Frete (FR-21).
func TestCalcularFreteEhDeterministico(t *testing.T) {
	primeiro, err := CalcularFrete(faixasDeTeste(), "51020000", 12345, isencaoDeTeste)
	if err != nil {
		t.Fatal(err)
	}
	for range 50 {
		repetido, err := CalcularFrete(faixasDeTeste(), "51020000", 12345, isencaoDeTeste)
		if err != nil || repetido != primeiro {
			t.Fatalf("segunda chamada = %+v, %v; primeira foi %+v", repetido, err, primeiro)
		}
	}
}

// Sem a linha padrão é erro de montagem — nunca Frete zero, nem para o CEP
// que cai numa faixa.
func TestCalcularFreteSemPadrao(t *testing.T) {
	faixas := faixasDeTeste()
	faixas = faixas[:len(faixas)-1]
	for _, cep := range []string{"00000001", "01310100"} {
		if _, err := CalcularFrete(faixas, cep, 100, isencaoDeTeste); !errors.Is(err, ErrSemRegiaoPadrao) {
			t.Errorf("CalcularFrete(%s) sem padrão = %v; quero ErrSemRegiaoPadrao", cep, err)
		}
	}
}

// O CEP fora da forma de oito dígitos é recusado, e não comparado: "01310-100"
// seria menor que "01310100" como texto e poderia cair noutra faixa.
func TestCalcularFreteRecusaCEPForaDaForma(t *testing.T) {
	for _, cep := range []string{"", "01310-100", "0131010", "013101000", "0131010a"} {
		if _, err := CalcularFrete(faixasDeTeste(), cep, 100, isencaoDeTeste); !errors.Is(err, ErrCEPInvalido) {
			t.Errorf("CalcularFrete(%q) = %v; quero ErrCEPInvalido", cep, err)
		}
	}
}
