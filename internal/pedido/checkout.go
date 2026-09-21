package pedido

import (
	"context"

	"github.com/Cashnip/amazon-waddle/internal/carrinho"
	"github.com/Cashnip/amazon-waddle/internal/pedido/db/gerado"
)

// A entrada no checkout (FR-19, AD-17). É de `pedido`, e não de `carrinho`,
// porque é o Pedido que precisa da ciência: ler o Carrinho continua sendo
// leitura pura, e a única escrita de `preco_visto_centavos` que o Comprador
// não provocou sai daqui.

// EntrarNoCheckout revalida o Carrinho do Comprador, devolve o que mudou e só
// então persiste a ciência dos preços que acabou de devolver.
//
// **A ordem é a razão de ser da função** (AD-17): ler → montar o relatório →
// confirmar os preços reportados. Confirmar antes de ler apagaria a mudança
// antes de ela aparecer; confirmar uma releitura gravaria um preço que o
// Comprador não viu. Quem chama fecha a transação **depois**, e escreve o JSON
// só depois do Commit — se o Commit falhar, nada foi reportado e o aviso volta
// na entrada seguinte.
//
// Confirma só as linhas visíveis com PrecoMudou: a invisível não tem preço a
// confirmar, e o visto dela fica intacto. Confirmar **não** desbloqueia — as
// duas listas são independentes —, então uma linha com preço mudado e bloqueio
// tem o preço confirmado e o bloqueio mantido.
//
// A regra de bloqueio e de mudança é a de `carrinho.Itens`, reaproveitada e
// não reescrita: o que a tela do Carrinho mostra e o que o checkout decide são
// o mesmo cálculo (AD-10).
//
// A transação vem de quem chama (AD-4) e é a mesma para a leitura e para a
// confirmação. `carrinho.Itens` a aceita por tipagem estrutural, como em
// CotarFrete.
func EntrarNoCheckout(ctx context.Context, bd gerado.DBTX, compradorID string) (carrinho.Conteudo, error) {
	conteudo, err := carrinho.Itens(ctx, bd, compradorID)
	if err != nil {
		return carrinho.Conteudo{}, err
	}
	// O relatório é este `conteudo`, e é ele que sai na resposta. A lista de
	// confirmação sai dele, e não de uma segunda leitura.
	vistos := make([]carrinho.PrecoVisto, 0, len(conteudo.Itens))
	for _, item := range conteudo.Itens {
		if item.Visivel && item.PrecoMudou {
			vistos = append(vistos, carrinho.PrecoVisto{ItemID: item.ID, PrecoCentavos: item.PrecoCentavos})
		}
	}
	if err := carrinho.ConfirmarPrecoVisto(ctx, bd, compradorID, vistos); err != nil {
		return carrinho.Conteudo{}, err
	}
	return conteudo, nil
}
