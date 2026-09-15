package identidade

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Os parâmetros do AD-9. São constantes de escrita, e não de leitura: Verificar
// lê os dele do próprio PHC, senão trocar o custo aqui invalidaria toda senha
// já gravada. Mudar qualquer um destes números faz as contas novas divergirem
// da semente, que media/gerar.go escreveu com exatamente estes.
const (
	argonMemoria     = 19456
	argonTempo       = 2
	argonParalelismo = 1
	argonTamanhoSal  = 16
	argonTamanhoHash = 32
)

// Gerar devolve o PHC Argon2id de uma senha, com sal aleatório por chamada —
// duas contas com a mesma senha não podem compartilhar hash, senão o banco
// vazado denuncia quem repetiu a senha de quem.
func Gerar(senha string) (string, error) {
	sal := make([]byte, argonTamanhoSal)
	if _, err := rand.Read(sal); err != nil {
		return "", fmt.Errorf("sal do Argon2id: %w", err)
	}
	hash := argon2.IDKey([]byte(senha), sal, argonTempo, argonMemoria, argonParalelismo, argonTamanhoHash)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemoria, argonTempo, argonParalelismo,
		base64.RawStdEncoding.EncodeToString(sal),
		base64.RawStdEncoding.EncodeToString(hash)), nil
}

// Verificar confere a senha contra o hash Argon2id no formato PHC
// (`$argon2id$v=19$m=19456,t=2,p=1$sal$hash`), que é como a semente grava e
// como o AD-9 manda. Os parâmetros vêm do próprio hash — é para isso que o
// PHC os carrega — e não de constante daqui, senão trocar o custo no futuro
// invalidaria toda senha já gravada.
//
// O base64 é raw-std, sem preenchimento: é o que o formato PHC especifica e o
// que media/gerar.go escreve.
func Verificar(phc, senha string) error {
	partes := strings.Split(phc, "$")
	// "" $ argon2id $ v=19 $ m=…,t=…,p=… $ sal $ hash
	if len(partes) != 6 || partes[0] != "" || partes[1] != "argon2id" || partes[2] != "v=19" {
		return fmt.Errorf("hash PHC malformado: %w", ErrCredencialInvalida)
	}
	var memoria, tempo uint32
	var paralelismo uint8
	if _, err := fmt.Sscanf(partes[3], "m=%d,t=%d,p=%d", &memoria, &tempo, &paralelismo); err != nil {
		return fmt.Errorf("parâmetros do PHC: %w", ErrCredencialInvalida)
	}
	if memoria == 0 || tempo == 0 || paralelismo == 0 {
		return fmt.Errorf("parâmetros do PHC fora de faixa: %w", ErrCredencialInvalida)
	}
	sal, err := base64.RawStdEncoding.DecodeString(partes[4])
	if err != nil {
		return fmt.Errorf("sal do PHC: %w", ErrCredencialInvalida)
	}
	esperado, err := base64.RawStdEncoding.DecodeString(partes[5])
	if err != nil || len(esperado) == 0 {
		return fmt.Errorf("hash do PHC: %w", ErrCredencialInvalida)
	}

	calculado := argon2.IDKey([]byte(senha), sal, tempo, memoria, paralelismo, uint32(len(esperado)))
	// Comparação em tempo constante: `bytes.Equal` sai no primeiro byte
	// diferente e transforma o hash em oráculo.
	if subtle.ConstantTimeCompare(calculado, esperado) != 1 {
		return ErrCredencialInvalida
	}
	return nil
}
