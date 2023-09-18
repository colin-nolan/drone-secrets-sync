package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	exampleServer = "https://example.com/drone"
	exampleToken  = "token123"
)

func TestCreateClient(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		client := CreateClient(Credential{
			Server: exampleServer,
			Token:  exampleToken,
		})
		assert.NotNil(t, client)
	})
}
