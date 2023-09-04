package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type SecretName = string

type MaskedSecret struct {
	Name SecretName
}

func (secret MaskedSecret) HashedNamePrefix() string {
	return fmt.Sprintf("%s___", secret.Name)
}

type Secret struct {
	MaskedSecret
	Value string
}

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
