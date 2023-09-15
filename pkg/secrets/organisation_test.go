package secrets

import (
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOrganisationClient struct {
	mock.Mock
}

func (client *MockOrganisationClient) OrgSecretList(namespace string) ([]*drone.Secret, error) {
	args := client.Called(namespace)
	return args.Get(0).([]*drone.Secret), args.Error(1)
}

func (client *MockOrganisationClient) OrgSecretCreate(namespace string, secret *drone.Secret) (*drone.Secret, error) {
	args := client.Called(namespace, secret)
	return args.Get(0).(*drone.Secret), args.Error(1)
}

func (client *MockOrganisationClient) OrgSecretUpdate(namespace string, secret *drone.Secret) (*drone.Secret, error) {
	args := client.Called(namespace, secret)
	return args.Get(0).(*drone.Secret), args.Error(1)
}

func (client *MockOrganisationClient) OrgSecretDelete(namespace string, secretName string) error {
	args := client.Called(namespace, secretName)
	return args.Error(0)
}

func createRepositoryDroneSecretsManager(namespace string) (OrganisationSecretsManager, *MockOrganisationClient) {
	client := new(MockOrganisationClient)
	return OrganisationSecretsManager{
		Client:    client,
		Namespace: namespace,
	}, client
}

func TestOrganisationSecretsManager(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		manager, client := createRepositoryDroneSecretsManager(exampleNamespace)
		client.On("OrgSecretList", exampleNamespace).Return([]*drone.Secret{{Name: exampleMaskedSecret1.Name}}, nil).Once()
		secrets, err := manager.List()
		assert.Nil(t, err)
		assert.ElementsMatch(t, secrets, []string{exampleMaskedSecret1.Name})
	})

	t.Run("list-error", func(t *testing.T) {
		manager, client := createRepositoryDroneSecretsManager(exampleNamespace)
		client.On("OrgSecretList", exampleNamespace).Return([]*drone.Secret{}, errExample).Once()
		_, err := manager.List()
		assert.NotNil(t, err)
	})

	t.Run("create", func(t *testing.T) {
		manager, client := createRepositoryDroneSecretsManager(exampleNamespace)
		client.On("OrgSecretCreate", exampleNamespace, &drone.Secret{
			Namespace: exampleNamespace,
			Name:      exampleSecret1.Name,
			Data:      exampleSecret1.Value,
		}).Return(exampleSecret1, nil).Once().Return(&drone.Secret{}, nil).Once()
		err := manager.Create(exampleSecret1.Name, exampleSecret1.Value)
		assert.Nil(t, err)
		assert.Equal(t, len(client.Calls), 1)
	})

	t.Run("create-err", func(t *testing.T) {
		manager, client := createRepositoryDroneSecretsManager(exampleNamespace)
		client.On("OrgSecretCreate", exampleNamespace, mock.AnythingOfType("*drone.Secret")).Return(exampleSecret1, nil).Once().Return(&drone.Secret{}, errExample).Once()
		err := manager.Create(exampleSecret1.Name, exampleSecret1.Value)
		assert.NotNil(t, err)
	})

	t.Run("update", func(t *testing.T) {
		manager, client := createRepositoryDroneSecretsManager(exampleNamespace)
		client.On("OrgSecretUpdate", exampleNamespace, &drone.Secret{
			Namespace: exampleNamespace,
			Name:      exampleSecret1.Name,
			Data:      exampleSecret1.Value,
		}).Return(exampleSecret1, nil).Once().Return(&drone.Secret{}, nil).Once()
		err := manager.Update(exampleSecret1.Name, exampleSecret1.Value)
		assert.Nil(t, err)
		assert.Equal(t, len(client.Calls), 1)
	})

	t.Run("update-err", func(t *testing.T) {
		manager, client := createRepositoryDroneSecretsManager(exampleNamespace)
		client.On("OrgSecretUpdate", exampleNamespace, mock.AnythingOfType("*drone.Secret")).Return(exampleSecret1, nil).Once().Return(&drone.Secret{}, errExample).Once()
		err := manager.Update(exampleSecret1.Name, exampleSecret1.Value)
		assert.NotNil(t, err)
	})

	t.Run("delete", func(t *testing.T) {
		manager, client := createRepositoryDroneSecretsManager(exampleNamespace)
		client.On("OrgSecretDelete", exampleNamespace, exampleSecret1.Name).Return(nil).Once()
		err := manager.Delete(exampleSecret1.Name)
		assert.Nil(t, err)
	})

	t.Run("delete-error", func(t *testing.T) {
		manager, client := createRepositoryDroneSecretsManager(exampleNamespace)
		client.On("OrgSecretDelete", exampleNamespace, exampleSecret1.Name).Return(errExample).Once()
		err := manager.Delete(exampleSecret1.Name)
		assert.NotNil(t, err)
	})
}
