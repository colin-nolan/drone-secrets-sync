package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	Value string
}

// Gets the name of the corresponding "hash" secret
func (secret Secret) HashedName() string {
	// Hashing secret value with the secret name as a salt
	hasher := sha256.Sum256([]byte(secret.Name + secret.Value))
	// Using hex to attain representations using only a-z,0-9
	hash := hex.EncodeToString(hasher[:])
	return fmt.Sprintf("%s%s", secret.HashedNamePrefix(), hash)
}

func NewSecret(name SecretName, value string) Secret {
	return Secret{MaskedSecret: MaskedSecret{Name: name}, Value: value}
}
