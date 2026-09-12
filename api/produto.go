package api

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/Cashnip/amazon-waddle/internal/catalogo"
	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
)

// saidaProduto é o DTO da Página de Produto. O preço sai cru, em centavos
// int64 — não existe divisão no caminho monetário (AD-3), e o `R$` é o
// navegador quem escreve. A imagem sai relativa: quem serve o byte é
// GET /api/v1/media/{arquivo}, do próprio binário (AD-12).
type saidaProduto struct {
	ID            string `json:"id"`
	Nome          string `json:"nome"`
	Descricao     string `json:"descricao"`
	PrecoCentavos int64  `json:"preco_centavos"`
	ImagemURL     string `json:"imagem_url"`
	Vendedor      string `json:"vendedor"`
}

// detalheDoProduto serve só o detalhe. A listagem (GET /api/v1/produtos) é de
// `busca` e chega na Épica 3.
func (s *servidor) detalheDoProduto(w http.ResponseWriter, r *http.Request) {
	produto, err := catalogo.BuscarProduto(r.Context(), s.pool, r.PathValue("id"))
	if errors.Is(err, pgx.ErrNoRows) {
		// Identificador malformado é o mesmo 404 do Produto inexistente:
		// "abc" não é uuid, e nada disso é 500.
		erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
		return
	}
	if err != nil {
		erro.Escrever(r.Context(), w, err, nil)
		return
	}
	escreverJSON(w, saidaProduto{
		ID:            produto.ID,
		Nome:          produto.Nome,
		Descricao:     produto.Descricao,
		PrecoCentavos: produto.PrecoCentavos,
		ImagemURL:     produto.ImagemURL,
		Vendedor:      produto.VendedorNome,
	})
}
