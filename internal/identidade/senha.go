package identidade

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

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
