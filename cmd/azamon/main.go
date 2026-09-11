// Command azamon é o binário único do sistema: servidor HTTP e, a partir da
// estória 1.8, a varredura que é o único motor do tempo (AD-6).
package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Cashnip/amazon-waddle/api"
	"github.com/Cashnip/amazon-waddle/db"
	"github.com/Cashnip/amazon-waddle/internal/plataforma"
)

// tempoDeCabecalho fecha a conexão que abre e não termina de mandar o
// cabeçalho. Não é limiar do NFR-16 — é higiene de servidor, como o orçamento
// de espera pelo Postgres, e por isso não vira variável AZAMON_.
const tempoDeCabecalho = 10 * time.Second

func main() {
	// O healthcheck do compose roda o próprio binário: a imagem é `scratch`,
	// sem shell e sem curl.
	if len(os.Args) > 1 && os.Args[1] == "-saude" {
		os.Exit(sondarSaude())
	}

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	if err := executar(ctx, os.Stdout); err != nil {
		os.Exit(1)
	}
}

// executar é a sequência de arranque — config → migrações → rotas → servir —
// separada de main para que um teste possa observá-la. Todo erro já sai
// logado; main só o traduz em código de saída.
func executar(ctx context.Context, saida io.Writer) error {
	raiz := plataforma.NovoLogger(saida, "plataforma")
	slog.SetDefault(raiz)
	logger := plataforma.ComModulo(raiz, "arranque")

	cfg, err := plataforma.CarregarConfig()
	if err != nil {
		logger.ErrorContext(ctx, "configuração inválida", "erro", err.Error())
		return err
	}

	// Migração antes de servir: nunca se serve com schema errado.
	if err := plataforma.Migrar(ctx, cfg.PostgresDSN, db.Migracoes, db.DirMigracoes); err != nil {
		logger.ErrorContext(ctx, "migração falhou; o processo não sobe", "erro", err.Error())
		return err
	}

	// A semente vem depois, e nunca antes: semear sobre schema que falhou
	// mascararia a falha de migração (AD-16).
	aplicada, err := plataforma.Semear(ctx, cfg.PostgresDSN, db.Semente, db.DirSemente, db.VersaoSemente)
	if err != nil {
		logger.ErrorContext(ctx, "semente falhou; o processo não sobe", "erro", err.Error())
		return err
	}
	if aplicada {
		logger.InfoContext(ctx, "Catálogo Semeado aplicado", "versao", db.VersaoSemente)
	} else {
		logger.InfoContext(ctx, "Catálogo Semeado pulado; o marcador já existe", "versao", db.VersaoSemente)
	}

	// O conjunto de medição do NFR-4 (Estória 1.4) é ligado só por
	// configuração e entra depois do Catálogo Semeado, porque multiplica os
	// 50 Produtos dele. O catálogo da demonstração nunca vira 5.000 (SM-C3).
	if cfg.SementeGrande {
		aplicada, err := plataforma.Semear(ctx, cfg.PostgresDSN, db.SementeGrande, db.DirSementeGrande, db.VersaoSementeGrande)
		if err != nil {
			logger.ErrorContext(ctx, "conjunto de medição falhou; o processo não sobe", "erro", err.Error())
			return err
		}
		logger.InfoContext(ctx, "conjunto de medição do NFR-4", "versao", db.VersaoSementeGrande, "aplicado", aplicada)
	}

	servidor := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Rotas(),
		ReadHeaderTimeout: tempoDeCabecalho,
	}
	encerrado := make(chan struct{})
	go func() {
		<-ctx.Done()
		logger.Info("encerrando")
		// WithoutCancel: o desligamento não pode herdar o cancelamento que o
		// disparou, senão as requisições em voo morrem no meio.
		_ = servidor.Shutdown(context.WithoutCancel(ctx))
		close(encerrado)
	}()

	logger.InfoContext(ctx, "servindo", "endereco", cfg.HTTPAddr)
	if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.ErrorContext(ctx, "servidor encerrou", "erro", err.Error())
		return err
	}
	<-encerrado
	return nil
}

// sondarSaude é o `/azamon -saude` do healthcheck: um GET em si mesmo.
func sondarSaude() int {
	cfg, err := plataforma.CarregarConfig()
	if err != nil {
		return 1
	}
	_, porta, err := net.SplitHostPort(cfg.HTTPAddr)
	if err != nil {
		return 1
	}
	resp, err := http.Get("http://127.0.0.1:" + porta + "/api/v1/saude")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
