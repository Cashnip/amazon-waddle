// Package db carrega as migrações e o Catálogo Semeado dentro do binário.
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

// Semente é o Catálogo Semeado, aplicado depois das migrações e uma vez só.
//
//go:embed semente
var Semente embed.FS

// DirSemente é o caminho da semente dentro de Semente.
const DirSemente = "semente"

// VersaoSemente é o que public.semente guarda: mudou o conteúdo de
// db/semente/, muda esta constante e o próximo arranque semeia de novo.
// É texto, e não número, porque quem lê a tabela quer saber *qual* semente
// está lá, não quantas vieram antes.
const VersaoSemente = "2026-09-11-catalogo-inicial"

// SementeGrande é o conjunto de medição do NFR-4: 5.000 Produtos que só
// entram com AZAMON_SEMENTE_GRANDE=true. Fica fora de Semente de propósito —
// o catálogo da demonstração continua com 50 Produtos (SM-C3).
//
//go:embed semente-grande
var SementeGrande embed.FS

// DirSementeGrande é o caminho do conjunto de medição dentro de SementeGrande.
const DirSementeGrande = "semente-grande"

// VersaoSementeGrande é o marcador próprio do conjunto de medição: ele e o
// Catálogo Semeado são semeados e versionados de forma independente.
const VersaoSementeGrande = "2026-09-11-medicao-nfr4"
