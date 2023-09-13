package secrets

import (
	"strings"
	"testing"
)

var (
	exampleMaskedSecret           = MaskedSecret{Name: "secret1"}
	exampleSecret                 = Secret{MaskedSecret: exampleMaskedSecret, Value: "value1", Argo2HashConfiguration: exampleArgo2HashConfiguration}
	exampleArgo2HashConfiguration = Argo2HashConfiguration{
		Iterations:  8,
		Memory:      1024,
		Parallelism: 1,
		Length:      1,
	}
)

func TestHashedNamePrefix(t *testing.T) {
	t.Run("type-check", func(t *testing.T) {
		var hashedNamePrefix interface{} = exampleMaskedSecret.HashedNamePrefix()

		if _, ok := hashedNamePrefix.(string); !ok {
			t.Errorf("HashedNamePrefix() should return a string")
		}
	})
}

func TestHashedName(t *testing.T) {
	t.Run("calculates", func(t *testing.T) {
		// Testing multiple times as there are different code paths due to caching
		for i := 0; i < 3; i++ {
			var hashedName interface{} = exampleSecret.HashedName()

			if _, ok := hashedName.(string); !ok {
				t.Errorf("HashedNamePrefix() should return a string")
			}
		}
	})

	t.Run("contains-prefix", func(t *testing.T) {
		hashedName := exampleSecret.HashedName()
		hashedNamePrefix := exampleSecret.HashedNamePrefix()

		if !strings.HasPrefix(hashedName, hashedNamePrefix) {
			t.Errorf("HashedName() '%s' should have prefix HashedNamePrefix() '%s'", hashedName, hashedNamePrefix)
		}
	})
}

func TestNewSecret(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		secret := NewSecret(exampleSecret.Name, exampleSecret.Value, exampleArgo2HashConfiguration)

		if secret.Name != exampleSecret.Name || secret.Value != exampleSecret.Value {
			t.Errorf("Secrets should have the same name and value: (%s, %s) != (%s, %s)", secret.Name, secret.Value, exampleSecret.Name, exampleSecret.Value)
		}
	})
}
