package secrets

import (
	"errors"
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepositoryClient struct {
	mock.Mock
}

func (client *MockRepositoryClient) SecretList(owner string, name string) ([]*drone.Secret, error) {
	args := client.Called(owner, name)
	return args.Get(0).([]*drone.Secret), args.Error(1)
}

func (client *MockRepositoryClient) SecretCreate(owner string, name string, secret *drone.Secret) (*drone.Secret, error) {
	args := client.Called(owner, name, secret)
	return args.Get(0).(*drone.Secret), args.Error(1)
}

func (client *MockRepositoryClient) SecretUpdate(owner string, name string, secret *drone.Secret) (*drone.Secret, error) {
	args := client.Called(owner, name, secret)
	return args.Get(0).(*drone.Secret), args.Error(1)
}

func (client *MockRepositoryClient) SecretDelete(owner string, name string, secretName string) error {
	args := client.Called(owner, name, secretName)
	return args.Error(0)
}

func createRepositorySecretsManager(namespace string, name string) (RepositorySecretsManager, *MockRepositoryClient) {
	client := new(MockRepositoryClient)
	return RepositorySecretsManager{
		Client:    client,
		Namespace: namespace,
		Name:      name,
	}, client
}

const (
	exampleNamespace = "octocat"
	exampleName      = "hello-world"
)

var (
	errExample = errors.New("example")
)

func TestRepositorySecretsManager(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		manager, client := createRepositorySecretsManager(exampleNamespace, exampleName)
		client.On("SecretList", exampleNamespace, exampleName).Return([]*drone.Secret{{Name: exampleMaskedSecret1.Name}}, nil).Once()
		secrets, err := manager.List()
		assert.Nil(t, err)
		assert.ElementsMatch(t, secrets, []string{exampleMaskedSecret1.Name})
	})

	t.Run("list-error", func(t *testing.T) {
		manager, client := createRepositorySecretsManager(exampleNamespace, exampleName)
		client.On("SecretList", exampleNamespace, exampleName).Return([]*drone.Secret{}, errExample).Once()
		_, err := manager.List()
		assert.NotNil(t, err)
	})

	t.Run("create", func(t *testing.T) {
		manager, client := createRepositorySecretsManager(exampleNamespace, exampleName)
		client.On("SecretCreate", exampleNamespace, exampleName, &drone.Secret{
			Namespace: exampleNamespace,
			Name:      exampleSecret1.Name,
			Data:      exampleSecret1.Value,
		}).Return(exampleSecret1, nil).Once().Return(&drone.Secret{}, nil).Once()
		err := manager.Create(exampleSecret1.Name, exampleSecret1.Value)
		assert.Nil(t, err)
		assert.Equal(t, len(client.Calls), 1)
	})

	t.Run("create-err", func(t *testing.T) {
		manager, client := createRepositorySecretsManager(exampleNamespace, exampleName)
		client.On("SecretCreate", exampleNamespace, exampleName, mock.AnythingOfType("*drone.Secret")).Return(exampleSecret1, nil).Once().Return(&drone.Secret{}, errExample).Once()
		err := manager.Create(exampleSecret1.Name, exampleSecret1.Value)
		assert.NotNil(t, err)
	})

	t.Run("update", func(t *testing.T) {
		manager, client := createRepositorySecretsManager(exampleNamespace, exampleName)
		client.On("SecretUpdate", exampleNamespace, exampleName, &drone.Secret{
			Namespace: exampleNamespace,
			Name:      exampleSecret1.Name,
			Data:      exampleSecret1.Value,
		}).Return(exampleSecret1, nil).Once().Return(&drone.Secret{}, nil).Once()
		err := manager.Update(exampleSecret1.Name, exampleSecret1.Value)
		assert.Nil(t, err)
		assert.Equal(t, len(client.Calls), 1)
	})

	t.Run("update-err", func(t *testing.T) {
		manager, client := createRepositorySecretsManager(exampleNamespace, exampleName)
		client.On("SecretUpdate", exampleNamespace, exampleName, mock.AnythingOfType("*drone.Secret")).Return(exampleSecret1, nil).Once().Return(&drone.Secret{}, errExample).Once()
		err := manager.Update(exampleSecret1.Name, exampleSecret1.Value)
		assert.NotNil(t, err)
	})

	t.Run("delete", func(t *testing.T) {
		manager, client := createRepositorySecretsManager(exampleNamespace, exampleName)
		client.On("SecretDelete", exampleNamespace, exampleName, exampleSecret1.Name).Return(nil).Once()
		err := manager.Delete(exampleSecret1.Name)
		assert.Nil(t, err)
	})

	t.Run("delete-error", func(t *testing.T) {
		manager, client := createRepositorySecretsManager(exampleNamespace, exampleName)
		client.On("SecretDelete", exampleNamespace, exampleName, exampleSecret1.Name).Return(errExample).Once()
		err := manager.Delete(exampleSecret1.Name)
		assert.NotNil(t, err)
	})
}
