package secrets

import (
	"strings"
	"testing"
)

var (
	exampleMaskedSecret = MaskedSecret{Name: "secret1"}
	exampleSecret       = Secret{MaskedSecret: exampleMaskedSecret, Value: "value1"}
)

func TestHashedNamePrefixType(t *testing.T) {
	var hashedNamePrefix interface{} = exampleMaskedSecret.HashedNamePrefix()
	if _, ok := hashedNamePrefix.(string); !ok {
		t.Errorf("HashedNamePrefix() should return a string")
	}
}

func TestHashedNameType(t *testing.T) {
	var hashedName interface{} = exampleSecret.HashedName()
	if _, ok := hashedName.(string); !ok {
		t.Errorf("HashedNamePrefix() should return a string")
	}
}

func TestHashedNameHasPrefix(t *testing.T) {
	hashedName := exampleSecret.HashedName()
	hashedNamePrefix := exampleSecret.HashedNamePrefix()

	if !strings.HasPrefix(hashedName, hashedNamePrefix) {
		t.Errorf("HashedName() '%s' should have prefix HashedNamePrefix() '%s'", hashedName, hashedNamePrefix)
	}
}

func TestNewSecret(t *testing.T) {
	secret := NewSecret(exampleSecret.Name, exampleSecret.Value)
	if secret.Name != exampleSecret.Name || secret.Value != exampleSecret.Value {
		t.Errorf("Secrets should have the same name and value: %s != %s", secret, exampleSecret)
	}
}
