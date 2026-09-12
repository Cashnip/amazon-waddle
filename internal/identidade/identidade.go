// Package identidade é dono de Comprador, Administrador, Sessão, Endereço e Argon2id.
//
// Este arquivo é a interface pública do módulo: o ÚNICO que outro módulo
// importa (AD-1). O que mora aqui na Épica 1 é o mínimo que a Estória 1.5
// pede — autenticar o Comprador semeado e carregar a Sessão. Cadastro,
// bloqueio por tentativas e recuperação de senha são da Épica 2.
package identidade

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/identidade/db/gerado"
)

// ErrCredencialInvalida é uma mensagem só para conta inexistente e para senha
// errada: duas mensagens distintas viram enumeração de contas.
var ErrCredencialInvalida = errors.New("E-mail ou senha inválidos.")

// ErrSessaoInvalida cobre cookie ausente, adulterado ou fora do Redis — para
// quem chama, expirada e inexistente são a mesma coisa.
var ErrSessaoInvalida = errors.New("Sessão expirada. Entre de novo.")

// hashDeDescarte existe para que o caminho "e-mail inexistente" gaste o mesmo
// Argon2id do caminho "senha errada": sem isto o tempo de resposta denuncia
// quais contas existem. Os parâmetros são os do AD-9, senão o custo difere.
const hashDeDescarte = "$argon2id$v=19$m=19456,t=2,p=1$YXphbW9uLWRlc2NhcnRlMA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

// Comprador é o que a Sessão carrega — o suficiente para a saudação da casca
// e para o Pedido da 1.6 saber de quem ele é.
type Comprador struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
}

// Autenticar devolve o Comprador da semente ou ErrCredencialInvalida. O
// e-mail é normalizado para minúsculas antes da consulta: a coluna tem
// `CHECK (email = lower(email))` e não casaria de outro jeito.
// O parâmetro é a DBTX do sqlc, e não *gerado.Queries: assim `api/` passa o
// pool sem importar o pacote gerado de dentro deste módulo, que é o que o
// AD-1 proíbe — a única porta de identidade é este arquivo.
func Autenticar(ctx context.Context, bd gerado.DBTX, email, senha string) (Comprador, error) {
	linha, err := gerado.New(bd).BuscarCompradorPorEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, pgx.ErrNoRows) {
		_ = Verificar(hashDeDescarte, senha)
		return Comprador{}, ErrCredencialInvalida
	}
	if err != nil {
		return Comprador{}, err
	}
	if err := Verificar(linha.SenhaHash, senha); err != nil {
		return Comprador{}, err
	}
	return Comprador{ID: uuidTexto(linha.ID), Nome: linha.Nome}, nil
}

// uuidTexto escreve o uuid do pgtype na forma canônica. O pgx guarda os 16
// bytes crus, e o módulo não depende de google/uuid por causa disso — a mesma
// escolha de plataforma.novoIdentificador.
func uuidTexto(u pgtype.UUID) string {
	h := hex.EncodeToString(u.Bytes[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
