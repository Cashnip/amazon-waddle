package identidade

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

// O hash literal do Comprador semeado, copiado de media/gerar.go. Se os
// parâmetros do AD-9 divergirem entre o gerador e a verificação, a
// demonstração inteira para no login — e este teste cai antes.
const hashDaSemente = "$argon2id$v=19$m=19456,t=2,p=1$YXphbW9uLWRlbW8tMDAwMQ$6Q3cmzFmai+W5Dz97kfAqozB+u5MVpUkiwZdGJXJXQ8"

func TestVerificarAceitaASenhaDaSemente(t *testing.T) {
	if err := Verificar(hashDaSemente, "azamon-comprador"); err != nil {
		t.Fatalf("a senha da semente não autentica: %v", err)
	}
}

func TestVerificarRecusaSenhaErrada(t *testing.T) {
	for _, senha := range []string{"", "azamon-comprador ", "Azamon-Comprador", "azamon-admin"} {
		if err := Verificar(hashDaSemente, senha); !errors.Is(err, ErrCredencialInvalida) {
			t.Errorf("Verificar(%q) = %v, quero ErrCredencialInvalida", senha, err)
		}
	}
}

// PHC malformado é credencial inválida, nunca pânico nem 500: o hash vem do
// banco, e uma linha estragada não pode derrubar o handler.
func TestVerificarRecusaPHCMalformado(t *testing.T) {
	for _, phc := range []string{
		"",
		"não é phc",
		"$argon2i$v=19$m=19456,t=2,p=1$YXphbW9u$YXphbW9u",      // outra variante
		"$argon2id$v=16$m=19456,t=2,p=1$YXphbW9u$YXphbW9u",     // outra versão
		"$argon2id$v=19$m=abc,t=2,p=1$YXphbW9u$YXphbW9u",       // parâmetro ilegível
		"$argon2id$v=19$m=0,t=0,p=0$YXphbW9u$YXphbW9u",         // parâmetro fora de faixa
		"$argon2id$v=19$m=19456,t=2,p=1$não-é-base64$YXphbW9u", // sal ilegível
		"$argon2id$v=19$m=19456,t=2,p=1$YXphbW9u$não-é-base64", // hash ilegível
		"$argon2id$v=19$m=19456,t=2,p=1$YXphbW9u$",             // hash vazio
		strings.TrimPrefix(hashDaSemente, "$"),                 // sem o campo vazio da frente
		hashDaSemente + "$sobrando",                            // campo a mais
	} {
		if err := Verificar(phc, "azamon-comprador"); !errors.Is(err, ErrCredencialInvalida) {
			t.Errorf("Verificar(%q) = %v, quero ErrCredencialInvalida", phc, err)
		}
	}
}

// O ciclo fechado: o hash que Gerar escreve tem de autenticar pelo mesmo
// Verificar que lê a semente. Se os parâmetros divergirem, a conta recém-criada
// não entra — e o cadastro fica verde do lado que grava e quebrado no login.
func TestGerarProduzHashQueVerificarAceita(t *testing.T) {
	const senha = "senha-de-cadastro-1"

	phc, err := Gerar(senha)
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	// O prefixo é o do AD-9, o mesmo que media/gerar.go escreveu na semente.
	if !strings.HasPrefix(phc, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("PHC = %q; quero os parâmetros do AD-9", phc)
	}
	if err := Verificar(phc, senha); err != nil {
		t.Errorf("a senha recém-gerada não autentica: %v", err)
	}
	if err := Verificar(phc, senha+"x"); !errors.Is(err, ErrCredencialInvalida) {
		t.Errorf("Verificar com senha errada = %v, quero ErrCredencialInvalida", err)
	}

	// Sal aleatório por chamada: dois hashes iguais para a mesma senha fariam
	// o banco vazado denunciar quem repetiu a senha de quem.
	outro, err := Gerar(senha)
	if err != nil {
		t.Fatalf("Gerar de novo: %v", err)
	}
	if outro == phc {
		t.Error("dois Gerar da mesma senha coincidiram; o sal não é aleatório")
	}

	// Os tamanhos do AD-9. Um sal curto demais passaria por todos os testes
	// acima — o ciclo fecha com qualquer tamanho — e só enfraqueceria o hash.
	partes := strings.Split(phc, "$")
	// Sem esta guarda, a regressão de formato que este teste existe para pegar
	// vira pânico de índice nas duas linhas abaixo, em vez de falha nomeada.
	if len(partes) != 6 {
		t.Fatalf("PHC = %q; quero as seis partes do formato", phc)
	}
	sal, err := base64.RawStdEncoding.DecodeString(partes[4])
	if err != nil || len(sal) != 16 {
		t.Errorf("sal = %d bytes, %v; quero 16", len(sal), err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(partes[5])
	if err != nil || len(hash) != 32 {
		t.Errorf("hash = %d bytes, %v; quero 32", len(hash), err)
	}
}

// O hash de descarte existe para igualar o tempo do caminho "conta
// inexistente" ao do caminho "senha errada". Se ele deixar de ser um PHC
// válido, Autenticar volta a sair barato e o tempo denuncia quais contas
// existem — sem falhar teste nenhum, a não ser este.
func TestHashDeDescarteEhValido(t *testing.T) {
	if err := Verificar(hashDeDescarte, "qualquer coisa"); !errors.Is(err, ErrCredencialInvalida) {
		t.Fatalf("hash de descarte = %v, quero ErrCredencialInvalida depois de rodar o Argon2id", err)
	}
	if !strings.Contains(hashDeDescarte, "m=19456,t=2,p=1") {
		t.Error("o hash de descarte não usa os parâmetros do AD-9; o custo difere do caminho real")
	}
	// O ErrCredencialInvalida acima sai igual quando Verificar desiste ANTES
	// do argon2.IDKey — um sal corrompido manteria o teste verde e o caminho
	// "conta inexistente" voltaria a ser barato. Conferir que as duas fatias
	// decodificam, com os tamanhos do AD-9, é o que prende o cálculo.
	sal, err := base64.RawStdEncoding.DecodeString(strings.Split(hashDeDescarte, "$")[4])
	if err != nil || len(sal) != 16 {
		t.Errorf("sal do hash de descarte = %d bytes, %v; quero 16 decodificáveis", len(sal), err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(strings.Split(hashDeDescarte, "$")[5])
	if err != nil || len(hash) != 32 {
		t.Errorf("hash de descarte = %d bytes, %v; quero 32 decodificáveis", len(hash), err)
	}
}
