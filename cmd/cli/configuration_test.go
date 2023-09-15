package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCredentialFromEnv(t *testing.T) {
	configuration := RepositoryConfiguration{
		Repository: "octocat/hello-world",
	}

	t.Run("RepositoryNamespace", func(t *testing.T) {
		assert.Equal(t, "octocat", configuration.RepositoryNamespace())
	})

	t.Run("RepositoryName", func(t *testing.T) {
		assert.Equal(t, "hello-world", configuration.RepositoryName())
	})
}
