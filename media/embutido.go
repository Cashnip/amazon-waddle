// Package media guarda as imagens do Catálogo Semeado dentro do binário. A
// imagem final é `scratch`: arquivo não embutido simplesmente não existe no
// contêiner, e nada é buscado em rede em tempo de execução (AD-12).
//
// As fotos são WebP (desde a 8.1), o images[0] de cada Produto do DummyJSON
// como veio, e ficam versionadas ao lado do gerador que confere a presença
// delas (media/gerar.go). Créditos e licença em CREDITOS.md.
package media

import "embed"

// Arquivos são as fotos de media/, servidas por GET /api/v1/media/{arquivo}.
//
//go:embed *.webp
var Arquivos embed.FS
