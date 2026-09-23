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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/identidade/db/gerado"
)

// ErrCredencialInvalida é uma mensagem só para conta inexistente e para senha
// errada: duas mensagens distintas viram enumeração de contas.
var ErrCredencialInvalida = errors.New("E-mail ou senha inválidos.")

// ErrSessaoInvalida cobre cookie ausente, adulterado ou fora do Redis — para
// quem chama, expirada e inexistente são a mesma coisa.
var ErrSessaoInvalida = errors.New("Sessão expirada. Entre de novo.")

// ErrEmailJaCadastrado é o único caso em que o sistema confirma que uma conta
// existe. É deliberado: sem isto o visitante não tem como saber por que o
// cadastro não vai, e a tela oferece o Login em seguida.
var ErrEmailJaCadastrado = errors.New("Já existe uma conta com este e-mail.")

// hashDeDescarte existe para que o caminho "e-mail inexistente" gaste o mesmo
// Argon2id do caminho "senha errada": sem isto o tempo de resposta denuncia
// quais contas existem. Os parâmetros são os do AD-9, senão o custo difere.
const hashDeDescarte = "$argon2id$v=19$m=19456,t=2,p=1$YXphbW9uLWRlc2NhcnRlMA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

// Papel é o que distingue os dois papéis do FR-4. É campo da Conta, e nunca
// coluna: `comprador` e `administrador` continuam tabelas separadas, e o papel
// é carimbado por QUAL função autenticou — não há valor lido do banco que
// pudesse promover ninguém.
type Papel string

const (
	PapelComprador     Papel = "comprador"
	PapelAdministrador Papel = "administrador"
)

// Conta é o que a Sessão carrega — o suficiente para a saudação da casca, para
// o Pedido da 1.6 saber de quem ele é, e para a guarda de prefixo da API
// decidir o papel sem voltar ao Postgres a cada requisição.
//
// Email entra aqui, e não numa consulta à parte (2.6): as duas linhas de
// login já trazem `email` da tabela — só não estava mapeado —, e é o que a
// tela de Perfil mostra em leitura sem pagar uma ida a mais ao Postgres.
type Conta struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Papel Papel  `json:"papel"`
}

// Autenticar devolve o Comprador da semente ou ErrCredencialInvalida. O
// e-mail é normalizado para minúsculas antes da consulta: a coluna tem
// `CHECK (email = lower(email))` e não casaria de outro jeito.
// O parâmetro é a DBTX do sqlc, e não *gerado.Queries: assim `api/` passa o
// pool sem importar o pacote gerado de dentro deste módulo, que é o que o
// AD-1 proíbe — a única porta de identidade é este arquivo.
func Autenticar(ctx context.Context, bd gerado.DBTX, email, senha string) (Conta, error) {
	linha, err := gerado.New(bd).BuscarCompradorPorEmail(ctx, normalizarEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		_ = Verificar(hashDeDescarte, senha)
		return Conta{}, ErrCredencialInvalida
	}
	if err != nil {
		return Conta{}, err
	}
	if err := Verificar(linha.SenhaHash, senha); err != nil {
		return Conta{}, err
	}
	return Conta{ID: uuidTexto(linha.ID), Nome: linha.Nome, Email: linha.Email, Papel: PapelComprador}, nil
}

// AutenticarAdministrador consulta SÓ `identidade.administrador`. É uma função
// por tabela, e não um parâmetro de papel numa função só: assim a separação é
// estrutural — quem escolhe o papel é a rota que chama, e não existe ordem de
// consulta que pudesse deixar o mesmo e-mail entrar pelo lado errado.
//
// O hashDeDescarte é o mesmo do Comprador: sem ele o tempo de resposta desta
// rota diria quais e-mails são de Administrador, que é a enumeração que mais
// vale a pena fazer no sistema inteiro.
func AutenticarAdministrador(ctx context.Context, bd gerado.DBTX, email, senha string) (Conta, error) {
	linha, err := gerado.New(bd).BuscarAdministradorPorEmail(ctx, normalizarEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		_ = Verificar(hashDeDescarte, senha)
		return Conta{}, ErrCredencialInvalida
	}
	if err != nil {
		return Conta{}, err
	}
	if err := Verificar(linha.SenhaHash, senha); err != nil {
		return Conta{}, err
	}
	// Email fica de fora de propósito: nenhuma tela de Administrador desta
	// estória o lê, e `saidaSessao.Email` não tem `omitempty` — preenchê-lo
	// aqui faria `POST /api/v1/admin/sessoes` (que passa pelo `entrar()`
	// compartilhado) devolver o e-mail, enquanto `GET /api/v1/admin/sessao`
	// (api/admin.go, fora do escopo desta estória) continuaria sempre em
	// `""` — duas rotas administrativas discordando do mesmo campo.
	return Conta{ID: uuidTexto(linha.ID), Nome: linha.Nome, Papel: PapelAdministrador}, nil
}

// Cadastrar cria o Comprador e devolve o mesmo Comprador de Autenticar — quem
// chama abre a Sessão em seguida sem passar de novo pelo login.
//
// O e-mail é normalizado aqui, e não por disciplina de quem chama: a coluna
// tem `CHECK (email = lower(email))` e o UNIQUE só vale sobre a forma
// normalizada. A duplicidade é decidida pelo próprio UNIQUE (SQLSTATE 23505),
// nunca por um SELECT antes do INSERT — dois cadastros simultâneos do mesmo
// e-mail passariam pelo teste-e-depois-grava.
func Cadastrar(ctx context.Context, bd gerado.DBTX, nome, email, senha string) (Conta, error) {
	hash, err := Gerar(senha)
	if err != nil {
		return Conta{}, err
	}
	linha, err := gerado.New(bd).CriarComprador(ctx, gerado.CriarCompradorParams{
		Nome:      nome,
		Email:     normalizarEmail(email),
		SenhaHash: hash,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Conta{}, ErrEmailJaCadastrado
	}
	if err != nil {
		return Conta{}, err
	}
	// O e-mail não volta no RETURNING de CriarComprador (só id, nome): a forma
	// gravada já é normalizarEmail(email), a mesma que acabou de entrar no
	// INSERT — reconsultar por ela seria uma ida ao Postgres pelo dado que
	// esta função já tem na mão.
	return Conta{ID: uuidTexto(linha.ID), Nome: linha.Nome, Email: normalizarEmail(email), Papel: PapelComprador}, nil
}

// BuscarComprador devolve o Comprador pelo identificador — a porta que o
// painel do Administrador (6.4) usa para pôr nome e e-mail no Detalhe de um
// Pedido que não é dele. Não autentica ninguém: quem chama já provou o papel
// na guarda do prefixo administrativo.
//
// O identificador malformado sai como pgx.ErrNoRows, e não como erro próprio:
// para quem chama, "não é uuid" e "não existe" são a mesma ausência, e o
// `api/` já traduz ErrNoRows no 404 de sempre (AD-11).
func BuscarComprador(ctx context.Context, bd gerado.DBTX, id string) (Conta, error) {
	var chave pgtype.UUID
	if err := chave.Scan(id); err != nil {
		return Conta{}, pgx.ErrNoRows
	}
	linha, err := gerado.New(bd).BuscarCompradorPorID(ctx, chave)
	if err != nil {
		return Conta{}, err
	}
	return Conta{ID: uuidTexto(linha.ID), Nome: linha.Nome, Email: linha.Email, Papel: PapelComprador}, nil
}

// normalizarEmail é a forma canônica do e-mail dentro do módulo: a coluna tem
// `CHECK (email = lower(email))` e o UNIQUE só vale sobre ela. Fica numa função
// porque o contador de tentativas do bloqueio chaveia pela MESMA forma — duas
// cópias da expressão divergiriam e o bloqueio passaria a contar por caixa.
func normalizarEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// uuidTexto escreve o uuid do pgtype na forma canônica. O pgx guarda os 16
// bytes crus, e o módulo não depende de google/uuid por causa disso — a mesma
// escolha de plataforma.novoIdentificador.
func uuidTexto(u pgtype.UUID) string {
	h := hex.EncodeToString(u.Bytes[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
