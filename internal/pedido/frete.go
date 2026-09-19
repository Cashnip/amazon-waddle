package pedido

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/Cashnip/amazon-waddle/internal/carrinho"
	"github.com/Cashnip/amazon-waddle/internal/identidade"
	"github.com/Cashnip/amazon-waddle/internal/pedido/db/gerado"
)

// A Regra de Frete (AD-17) é a tabela pedido.faixa_frete mais a função pura
// CalcularFrete, e mora aqui porque é o Pedido que congela o Frete. `carrinho`
// não calcula Frete: a tela do Carrinho só mostra a distância até o limiar.

// ErrSemRegiaoPadrao é a tabela sem a linha padrão. É erro de montagem, e não
// Frete zero: um zero silencioso daria Frete grátis a todo CEP fora de faixa.
var ErrSemRegiaoPadrao = errors.New("a Regra de Frete não tem região padrão")

// ErrCEPInvalido é o CEP fora da forma do banco. A comparação por intervalo de
// texto só vale com oito dígitos: com hífen ou mais curto, cairia em silêncio
// na faixa errada ou na padrão. O CHECK de identidade.endereco já o garante;
// isto é a mesma regra na porta da função pura.
var ErrCEPInvalido = errors.New("CEP fora da forma de oito dígitos")

var formaDoCEP = regexp.MustCompile(`^[0-9]{8}$`)

// FaixaDeFrete é uma linha de pedido.faixa_frete. A padrão não tem faixa: é o
// valor de todo CEP que nenhuma outra linha contém (FR-21).
type FaixaDeFrete struct {
	CEPInicio     string
	CEPFim        string
	Regiao        string
	ValorCentavos int64
	Padrao        bool
}

// Frete é o resultado da Regra: a região e o valor. Isento é valor zero, com a
// região ainda dita — o Comprador vê de onde o Frete sairia.
type Frete struct {
	Regiao        string
	ValorCentavos int64
}

// Cotacao é o que a Revisão mostra: as três parcelas, com o total somado aqui.
// O navegador não soma nada (NFR-13).
type Cotacao struct {
	Regiao           string
	SubtotalCentavos int64
	FreteCentavos    int64
	TotalCentavos    int64
}

// CalcularFrete é a Regra de Frete inteira, pura: mesmo CEP, mesmo subtotal e
// mesmo limiar dão sempre o mesmo Frete. O CEP chega com os oito dígitos do
// banco, e o intervalo compara por texto — com o mesmo comprimento, a ordem do
// texto é a ordem numérica.
//
// Subtotal igual ou acima do limiar é isento: é o mesmo ponto em que o
// Carrinho (4.5) para de dizer "Faltam R$ X". Nenhuma divisão, nenhum rateio
// por Item (AD-9) — o valor é o da linha, ou zero.
//
// A primeira faixa que contém o CEP decide, na ordem recebida; a consulta
// entrega por `cep_inicio`, o que mantém a escolha determinística.
func CalcularFrete(faixas []FaixaDeFrete, cep string, subtotalCentavos, isencaoCentavos int64) (Frete, error) {
	if !formaDoCEP.MatchString(cep) {
		return Frete{}, ErrCEPInvalido
	}
	var escolhida, padrao *FaixaDeFrete
	for i := range faixas {
		f := &faixas[i]
		if f.Padrao {
			if padrao == nil {
				padrao = f
			}
			continue
		}
		if escolhida == nil && f.CEPInicio <= cep && cep <= f.CEPFim {
			escolhida = f
		}
	}
	// A padrão é exigida mesmo quando o CEP cai numa faixa: a Regra sem ela
	// está mal montada, e o defeito tem de aparecer no primeiro cálculo, não
	// no primeiro CEP fora de faixa.
	if padrao == nil {
		return Frete{}, ErrSemRegiaoPadrao
	}
	if escolhida == nil {
		escolhida = padrao
	}
	frete := Frete{Regiao: escolhida.Regiao, ValorCentavos: escolhida.ValorCentavos}
	if subtotalCentavos >= isencaoCentavos {
		frete.ValorCentavos = 0
	}
	return frete, nil
}

// CotarFrete é a leitura da Revisão: o CEP do Endereço do dono, o subtotal do
// Carrinho pelos preços de agora e a Regra de Frete. Leitura pura — nada é
// gravado; congelar o Frete é da criação do Pedido.
//
// Endereço de outro Comprador, inexistente ou de uuid malformado sai como
// pgx.ErrNoRows, que `api/` traduz no mesmo 404.
func CotarFrete(ctx context.Context, bd gerado.DBTX, compradorID, enderecoID string, isencaoCentavos int64) (Cotacao, error) {
	endereco, err := identidade.BuscarEndereco(ctx, bd, enderecoID, compradorID)
	if err != nil {
		return Cotacao{}, err
	}
	conteudo, err := carrinho.Itens(ctx, bd, compradorID)
	if err != nil {
		return Cotacao{}, err
	}
	faixas, err := faixasDeFrete(ctx, bd)
	if err != nil {
		return Cotacao{}, err
	}
	frete, err := CalcularFrete(faixas, endereco.CEP, conteudo.SubtotalCentavos, isencaoCentavos)
	if err != nil {
		return Cotacao{}, err
	}
	return Cotacao{
		Regiao:           frete.Regiao,
		SubtotalCentavos: conteudo.SubtotalCentavos,
		FreteCentavos:    frete.ValorCentavos,
		TotalCentavos:    conteudo.SubtotalCentavos + frete.ValorCentavos,
	}, nil
}

// faixasDeFrete lê a tabela inteira; são nove linhas.
func faixasDeFrete(ctx context.Context, bd gerado.DBTX) ([]FaixaDeFrete, error) {
	linhas, err := gerado.New(bd).ListarFaixasDeFrete(ctx)
	if err != nil {
		return nil, fmt.Errorf("ler a Regra de Frete: %w", err)
	}
	faixas := make([]FaixaDeFrete, 0, len(linhas))
	for _, l := range linhas {
		faixas = append(faixas, FaixaDeFrete{
			CEPInicio:     l.CepInicio.String,
			CEPFim:        l.CepFim.String,
			Regiao:        l.Regiao,
			ValorCentavos: l.ValorCentavos,
			Padrao:        l.Padrao,
		})
	}
	return faixas, nil
}
