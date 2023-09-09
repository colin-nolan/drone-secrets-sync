package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/argon2"
)

// TODO: allow these to be configured by the user
const (
	hashIterations  = 32
	hashMemory      = 64 * 1024
	hashParallelism = 2
	hashLength      = 16
)

type SecretName = string

// A secret where the value is unknown (as received from the API)
type MaskedSecret struct {
	Name SecretName
}

// Gets the prefix of the name of the corresponding "hash" secret.
// The full name is only known when the value is known.
func (secret MaskedSecret) HashedNamePrefix() string {
	return fmt.Sprintf("%s___", secret.Name)
}

// A secret where the value is known
type Secret struct {
	MaskedSecret
	Value            string
	CachedHashedName string
}

// Gets the name of the corresponding "hash" secret
func (secret *Secret) HashedName() string {
	if secret.CachedHashedName == "" {
		// Salt must be derivable from only the information available, which is the secret name
		salt := sha256.Sum256([]byte(secret.Name))
		start := time.Now()
		// Creating hash using expensive argon2 algorithm to reduce the effectiveness of brute force attacks
		key := argon2.IDKey([]byte(secret.Value), salt[:], hashIterations, hashMemory, hashParallelism, hashLength)
		log.Debug().Msgf("Hash created in %s", time.Since(start))
		secret.CachedHashedName = hex.EncodeToString(key)
	}
	return fmt.Sprintf("%s%s", secret.HashedNamePrefix(), secret.CachedHashedName)
}

func NewSecret(name SecretName, value string) Secret {
	return Secret{MaskedSecret: MaskedSecret{Name: name}, Value: value}
}
