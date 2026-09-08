package plataforma

import (
	"context"
	"io"
	"log/slog"
)

// manipulador garante o AD-15: `modulo` e `correlacao` em toda linha, uma vez
// só. Injetar aqui — e não com slog.With — evita a chave duplicada no JSON.
type manipulador struct {
	slog.Handler
	modulo string
}

func (m *manipulador) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(
		slog.String("modulo", m.modulo),
		slog.String("correlacao", CorrelacaoDe(ctx)),
	)
	return m.Handler.Handle(ctx, r)
}

func (m *manipulador) WithAttrs(atrs []slog.Attr) slog.Handler {
	return &manipulador{Handler: m.Handler.WithAttrs(atrs), modulo: m.modulo}
}

func (m *manipulador) WithGroup(nome string) slog.Handler {
	return &manipulador{Handler: m.Handler.WithGroup(nome), modulo: m.modulo}
}

// NovoLogger devolve o logger JSON do sistema. Nenhum dado pessoal entra aqui
// (AD-15): e-mail, Endereço e nome saem como identificador, nunca como valor.
func NovoLogger(saida io.Writer, modulo string) *slog.Logger {
	return slog.New(&manipulador{
		Handler: slog.NewJSONHandler(saida, nil),
		modulo:  modulo,
	})
}

// ComModulo deriva o logger de um módulo a partir do logger raiz.
func ComModulo(l *slog.Logger, modulo string) *slog.Logger {
	if m, ok := l.Handler().(*manipulador); ok {
		return slog.New(&manipulador{Handler: m.Handler, modulo: modulo})
	}
	return l.With("modulo", modulo)
}
