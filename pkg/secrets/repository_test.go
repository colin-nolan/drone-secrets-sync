package secrets

import (
	"fmt"
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/stretchr/testify/assert"
)

type MockClient struct {
	drone.Client

	SecretListOkReturn  []*drone.Secret
	SecretListErrReturn error

	SecretCreateOkReturn  *drone.Secret
	SecretCreateErrReturn error

	SecretDeleteReturn error
}

func (client MockClient) SecretList(owner, name string) ([]*drone.Secret, error) {
	return client.SecretListOkReturn, client.SecretListErrReturn
}

func (client MockClient) SecretCreate(owner, name string, secret *drone.Secret) (*drone.Secret, error) {
	return client.SecretCreateOkReturn, client.SecretCreateErrReturn
}

func (client MockClient) SecretDelete(owner, name, secret string) error {
	return client.SecretDeleteReturn
}

func createRepositorySecretManager(client MinimalClient) RepositorySecretManager {
	return RepositorySecretManager{
		Client: client,
		Owner:  "example-owner",
		Name:   "example-name",
	}
}

func TestListSecrets(t *testing.T) {
	repository := createRepositorySecretManager(MockClient{
		SecretListOkReturn: []*drone.Secret{
			{Name: "example1"},
			{Name: "example2"},
		},
	})
	secrets, err := repository.ListSecrets()
	assert.Nil(t, err)
	assert.ElementsMatch(t, secrets, []MaskedSecret{
		{Name: "example1"},
		{Name: "example2"},
	})
}

func TestListSecretsWhenNone(t *testing.T) {
	repository := createRepositorySecretManager(MockClient{
		SecretListOkReturn: []*drone.Secret{},
	})
	secrets, err := repository.ListSecrets()
	assert.Nil(t, err)
	assert.Empty(t, secrets)
}

func TestListSecretsWhenErr(t *testing.T) {
	repository := createRepositorySecretManager(MockClient{
		SecretListErrReturn: fmt.Errorf("example-error"),
	})
	_, err := repository.ListSecrets()
	assert.NotNil(t, err)
}
