package client

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	exampleServer = "https://example.com/drone"
	exampleToken  = "token123"
)

func TestGetCredentialFromEnv(t *testing.T) {
	os.Setenv(DroneServerVariable, exampleServer)
	os.Setenv(DroneTokenVariable, exampleToken)
	credential, err := GetCredentialFromEnv()
	assert.Nil(t, err)
	assert.Equal(t, exampleServer, credential.Server)
	assert.Equal(t, exampleToken, credential.Token)
}

func TestGetCredentialFromEnvServerNotSet(t *testing.T) {
	os.Unsetenv(DroneServerVariable)
	os.Setenv(DroneTokenVariable, exampleToken)
	_, err := GetCredentialFromEnv()
	assert.NotNil(t, err)
}

func TestGetCredentialFromEnvTokenNotSet(t *testing.T) {
	os.Unsetenv(DroneTokenVariable)
	os.Setenv(DroneServerVariable, exampleServer)
	_, err := GetCredentialFromEnv()
	assert.NotNil(t, err)
}

func TestCreateClient(t *testing.T) {
	client := CreateClient(Credential{
		Server: exampleServer,
		Token:  exampleToken,
	})
	assert.NotNil(t, client)
}
