package api

import (
	"net/http"

	"github.com/Cashnip/amazon-waddle/internal/plataforma/erro"
	"github.com/Cashnip/amazon-waddle/media"
)

// midia serve a imagem do Produto do sistema de arquivos embutido. Fica sob
// /api/v1 porque é o único prefixo que o rewrites() do Next atravessa (AD-10),
// e é por isso que a URL gravada no Produto é relativa: o navegador pede à
// mesma origem, sem rede externa em ponto nenhum (AD-12).
func midia(w http.ResponseWriter, r *http.Request) {
	// Quem barra travessia de diretório é o fs.ValidPath de dentro do ReadFile
	// do embed: ele recusa qualquer caminho com ".." ou com "/" à frente. Não é
	// o {arquivo} do ServeMux — o padrão casa o caminho escapado, e o
	// PathValue entrega o segmento já decodificado, então "%2F" e "%2E" chegam
	// aqui como "/" e ".".
	nome := r.PathValue("arquivo")
	conteudo, err := media.Arquivos.ReadFile(nome)
	if err != nil {
		erro.Escrever(r.Context(), w, erro.ErrNaoEncontrado, nil)
		return
	}
	// Literal, e não mime.TypeByExtension: aquele lê a tabela MIME do sistema
	// operacional, e o embed guarda um tipo de arquivo só.
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write(conteudo)
}
