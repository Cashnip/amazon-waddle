// Package db carrega as migrações dentro do binário. O Catálogo Semeado mora
// em db/semente/ e ainda não é embutido — é a estória 1.3 que o aplica.
//
// O embed mora aqui, e não em internal/plataforma, porque //go:embed não
// atravessa ".." — a diretiva só enxerga a árvore do próprio pacote.
package db

import "embed"

// Migracoes é a árvore de db/migracoes aplicada no arranque (goose, embed.FS).
// O prefixo "all:" inclui o .gitkeep, o que mantém o embed válido enquanto
// nenhuma migração existe.
//
//go:embed all:migracoes
var Migracoes embed.FS

// DirMigracoes é o caminho das migrações dentro de Migracoes.
const DirMigracoes = "migracoes"
