// Package media guarda as imagens do Catálogo Semeado dentro do binário. A
// imagem final é `scratch`: arquivo não embutido simplesmente não existe no
// contêiner, e nada é buscado em rede em tempo de execução (AD-12).
//
// Os arquivos e o gerador que os escreve (media/gerar.go) são os dois
// versionados: trocar placeholder por foto de verdade é substituir o arquivo
// aqui, sem tocar no banco — a URL gravada no Produto não muda.
package media

import "embed"

// Arquivos são os SVG de media/, servidos por GET /api/v1/media/{arquivo}.
//
//go:embed *.svg
var Arquivos embed.FS
