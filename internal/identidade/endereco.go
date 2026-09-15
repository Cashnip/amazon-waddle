package identidade

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cashnip/amazon-waddle/internal/identidade/db/gerado"
)

// Endereco é o mínimo de uma etiqueta de entrega. Sem apelido e sem coluna de
// "padrão": a lista do checkout (5.2) distingue um Endereço do outro pelo
// logradouro, e um rótulo livre seria campo que nenhuma FR pede.
//
// O CEP sai daqui como os oito dígitos que estão no banco. A máscara é da
// tela — uma forma só no banco é o que deixa a Faixa de Frete da 5.3 comparar
// por intervalo de texto sem normalizar a cada leitura.
type Endereco struct {
	ID           string
	Destinatario string
	CEP          string
	Logradouro   string
	Numero       string
	Complemento  string
	Bairro       string
	Cidade       string
	UF           string
}

// ListarEnderecos devolve os Endereços do dono em ordem estável — é ela o
// "escolher" da FR-5: quem escolhe é o checkout da 5.2, sobre esta lista.
// Comprador sem nenhum devolve fatia vazia, e não erro.
func ListarEnderecos(ctx context.Context, bd gerado.DBTX, compradorID string) ([]Endereco, error) {
	dono, err := uuidDe(compradorID)
	if err != nil {
		return nil, err
	}
	linhas, err := gerado.New(bd).ListarEnderecos(ctx, dono)
	if err != nil {
		return nil, err
	}
	enderecos := make([]Endereco, 0, len(linhas))
	for _, l := range linhas {
		enderecos = append(enderecos, Endereco{
			ID:           uuidTexto(l.ID),
			Destinatario: l.Destinatario,
			CEP:          l.Cep,
			Logradouro:   l.Logradouro,
			Numero:       l.Numero,
			Complemento:  l.Complemento,
			Bairro:       l.Bairro,
			Cidade:       l.Cidade,
			UF:           l.Uf,
		})
	}
	return enderecos, nil
}

// CriarEndereco grava o Endereço do dono, até o teto por Comprador.
//
// ALCANÇAR O TETO SAI COMO pgx.ErrNoRows, e é o único caminho desta função que
// o produz: o dono vem da Sessão, então não há linha alheia nem uuid
// malformado por aqui. O teto é conferido dentro do próprio INSERT, e não num
// SELECT count antes — uma ida ao banco só, e o mesmo idioma de "zero linhas
// afetadas" do compare-and-swap do Pedido.
//
// A normalização do CEP e da UF mora aqui, como normalizarEmail, porque as
// colunas têm os CHECKs correspondentes e não aceitariam a forma digitada.
//
// O teto é conferido em READ COMMITTED, então dois cadastros simultâneos do
// MESMO Comprador podem ambos ver a contagem abaixo do limiar e passar — o
// desvio máximo é um Endereço a mais numa conta, sem nada quebrado. Apertar
// custaria um `FOR UPDATE` sobre as linhas do dono a cada cadastro, e o teto
// não é regra de dinheiro nem de Estoque: se um dia precisar ser exato, é aí
// que ele fica.
func CriarEndereco(ctx context.Context, bd gerado.DBTX, compradorID string, e Endereco, maximo int) (Endereco, error) {
	dono, err := uuidDe(compradorID)
	if err != nil {
		return Endereco{}, err
	}
	linha, err := gerado.New(bd).CriarEndereco(ctx, gerado.CriarEnderecoParams{
		CompradorID:  dono,
		Destinatario: e.Destinatario,
		Cep:          normalizarCEP(e.CEP),
		Logradouro:   e.Logradouro,
		Numero:       e.Numero,
		Complemento:  e.Complemento,
		Bairro:       e.Bairro,
		Cidade:       e.Cidade,
		Uf:           normalizarUF(e.UF),
		Maximo:       int64(maximo),
	})
	if err != nil {
		return Endereco{}, err
	}
	return Endereco{
		ID:           uuidTexto(linha.ID),
		Destinatario: linha.Destinatario,
		CEP:          linha.Cep,
		Logradouro:   linha.Logradouro,
		Numero:       linha.Numero,
		Complemento:  linha.Complemento,
		Bairro:       linha.Bairro,
		Cidade:       linha.Cidade,
		UF:           linha.Uf,
	}, nil
}

// AtualizarEndereco escreve sobre o Endereço do dono. O dono entra no WHERE, e
// não numa checagem depois (AD-11): Endereço de outro Comprador, Endereço
// inexistente e uuid malformado saem os três como pgx.ErrNoRows, que `api/`
// traduz no MESMO 404 — responder diferente vazaria a existência da linha.
func AtualizarEndereco(ctx context.Context, bd gerado.DBTX, enderecoID, compradorID string, e Endereco) (Endereco, error) {
	chave, err := uuidDe(enderecoID)
	if err != nil {
		return Endereco{}, err
	}
	dono, err := uuidDe(compradorID)
	if err != nil {
		return Endereco{}, err
	}
	linha, err := gerado.New(bd).AtualizarEndereco(ctx, gerado.AtualizarEnderecoParams{
		ID:           chave,
		CompradorID:  dono,
		Destinatario: e.Destinatario,
		Cep:          normalizarCEP(e.CEP),
		Logradouro:   e.Logradouro,
		Numero:       e.Numero,
		Complemento:  e.Complemento,
		Bairro:       e.Bairro,
		Cidade:       e.Cidade,
		Uf:           normalizarUF(e.UF),
	})
	if err != nil {
		return Endereco{}, err
	}
	return Endereco{
		ID:           uuidTexto(linha.ID),
		Destinatario: linha.Destinatario,
		CEP:          linha.Cep,
		Logradouro:   linha.Logradouro,
		Numero:       linha.Numero,
		Complemento:  linha.Complemento,
		Bairro:       linha.Bairro,
		Cidade:       linha.Cidade,
		UF:           linha.Uf,
	}, nil
}

// RemoverEndereco apaga a linha do dono — DELETE de verdade, e nenhum Pedido é
// tocado: o Pedido congela o Endereço na criação (AD-3), então não há Pedido
// que dependa desta linha continuar existindo.
//
// Zero linhas afetadas é pgx.ErrNoRows pelo mesmo motivo de AtualizarEndereco:
// alheio e inexistente têm de sair idênticos.
func RemoverEndereco(ctx context.Context, bd gerado.DBTX, enderecoID, compradorID string) error {
	chave, err := uuidDe(enderecoID)
	if err != nil {
		return err
	}
	dono, err := uuidDe(compradorID)
	if err != nil {
		return err
	}
	linhas, err := gerado.New(bd).RemoverEndereco(ctx, gerado.RemoverEnderecoParams{ID: chave, CompradorID: dono})
	if err != nil {
		return err
	}
	if linhas == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// uuidDe é o par de uuidTexto: o identificador malformado vira pgx.ErrNoRows, e
// não um 500 — para quem chama, "não é um uuid" e "não existe" são a mesma
// coisa (o molde é o de pedido.Buscar).
func uuidDe(texto string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(texto); err != nil {
		return u, pgx.ErrNoRows
	}
	return u, nil
}

// normalizarCEP guarda só os dígitos: a coluna tem `CHECK (cep ~ '^[0-9]{8}$')`
// e a forma com hífen não entraria. Quem já recusou o que não é CEP é a
// validação de `api/` — aqui o que sobra é tirar a máscara da tela.
func normalizarCEP(cep string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cep)
}

// normalizarUF é o mesmo contrato de normalizarEmail, do outro lado do CHECK:
// a coluna tem `CHECK (uf = upper(uf))`. Quais são as 27 siglas é regra de
// quem valida, não de quem grava.
func normalizarUF(uf string) string {
	return strings.ToUpper(strings.TrimSpace(uf))
}
