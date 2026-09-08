package plataforma

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// Sem migração nenhuma o goose não tem o que aplicar — mas o banco continua
// sendo exigido: subir com o Postgres inalcançável adiaria a descoberta.
func TestMigrarSemMigracoesAindaExigeBanco(t *testing.T) {
	vazio := fstest.MapFS{"migracoes/.gitkeep": &fstest.MapFile{}}
	ctx, cancelar := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancelar()
	if err := Migrar(ctx, "postgres://azamon@127.0.0.1:1/azamon", vazio, "migracoes"); err == nil {
		t.Fatal("embed.FS vazio com banco inalcançável devia falhar")
	}
}

// Migração quebrada derruba o arranque: nunca se serve com schema errado.
func TestMigrarComArquivoInvalidoFalha(t *testing.T) {
	quebrado := fstest.MapFS{
		"migracoes/nao_tem_versao.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}
	err := Migrar(context.Background(), "isso-nao-e-um-dsn", quebrado, "migracoes")
	if err == nil {
		t.Fatal("arquivo .sql inválido devia falhar")
	}
	if !strings.Contains(err.Error(), "migrações inválidas") {
		t.Errorf("erro = %v, esperava a falha de validação das migrações", err)
	}
}

// Banco inalcançável também derruba: o binário não sobe sem schema aplicado.
func TestMigrarSemBancoFalha(t *testing.T) {
	uma := fstest.MapFS{
		"migracoes/20260101000000_teste.sql": &fstest.MapFile{
			Data: []byte("-- +goose Up\nSELECT 1;\n"),
		},
	}
	ctx, cancelar := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancelar()
	if err := Migrar(ctx, "postgres://azamon@127.0.0.1:1/azamon", uma, "migracoes"); err == nil {
		t.Fatal("banco inalcançável devia falhar")
	}
}
