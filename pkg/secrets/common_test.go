package secrets

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSecretsManager struct {
	mock.Mock
}

func (manager *MockSecretsManager) List() ([]string, error) {
	args := manager.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (manager *MockSecretsManager) Create(secretName string, secretValue string) error {
	args := manager.Called(secretName, secretValue)
	return args.Error(0)
}

func (manager *MockSecretsManager) Update(secretName string, secretValue string) error {
	args := manager.Called(secretName, secretValue)
	return args.Error(0)
}

func (manager *MockSecretsManager) Delete(secretName string) error {
	args := manager.Called(secretName)
	return args.Error(0)
}

func createMockSyncedSecretManager() (SyncedSecretManager, *MockSecretsManager) {
	manager := new(MockSecretsManager)
	return SyncedSecretManager{
		GenericSecretManager: manager,
	}, manager
}

var (
	exampleMaskedSecret1 = MaskedSecret{
		Name: "example1",
	}
	exampleMaskedSecret2 = MaskedSecret{
		Name: "example2",
	}
	exampleMaskedSecret3 = MaskedSecret{
		Name: "example3",
	}
	exampleSecret1 = Secret{
		MaskedSecret:           exampleMaskedSecret1,
		Value:                  "example-value1",
		Argo2HashConfiguration: exampleArgo2HashConfiguration,
	}
	exampleSecret2 = Secret{
		MaskedSecret:           exampleMaskedSecret2,
		Value:                  "example-value2",
		Argo2HashConfiguration: exampleArgo2HashConfiguration,
	}
	exampleSecret3 = Secret{
		MaskedSecret:           exampleMaskedSecret3,
		Value:                  "example-value3",
		Argo2HashConfiguration: exampleArgo2HashConfiguration,
	}
)

// Tests for `ListSecrets` ----------
func TestListSecrets(t *testing.T) {
	t.Run("existing-secrets", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{exampleMaskedSecret1.Name, exampleMaskedSecret2.Name}, nil).Once()

		secrets, err := repository.ListSecrets()
		assert.Nil(t, err)
		assert.ElementsMatch(t, secrets, []MaskedSecret{
			exampleMaskedSecret1,
			exampleMaskedSecret2,
		})
	})

	t.Run("no-secrets", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{}, nil).Once()

		secrets, err := repository.ListSecrets()
		assert.Nil(t, err)
		assert.Empty(t, secrets)
	})

	t.Run("err", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{}, errors.New("example")).Once()

		_, err := repository.ListSecrets()
		assert.NotNil(t, err)
	})
}

// Tests for `ListSyncedSecrets` ----------
func TestListSyncedSecrets(t *testing.T) {
	t.Run("partial-synced", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return(
			[]string{
				exampleSecret1.Name,
				exampleSecret1.HashedName(),
				exampleSecret2.Name,
				exampleSecret3.HashedName(),
				exampleSecret3.Name,
			}, nil).Once()

		secrets, err := repository.ListSyncedSecrets()
		assert.Nil(t, err)
		assert.ElementsMatch(t, secrets, []MaskedSecret{
			exampleMaskedSecret1,
			exampleMaskedSecret3,
		})
	})

	t.Run("no-secrets", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{}, nil).Once()

		secrets, err := repository.ListSyncedSecrets()
		assert.Nil(t, err)
		assert.Empty(t, secrets)
	})

	t.Run("no-synced", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return(
			[]string{
				exampleMaskedSecret1.Name,
				exampleMaskedSecret2.Name,
			}, nil).Once()

		secrets, err := repository.ListSyncedSecrets()
		assert.Nil(t, err)
		assert.Empty(t, secrets)
	})

	t.Run("err", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{}, errors.New("example")).Once()

		_, err := repository.ListSyncedSecrets()
		assert.NotNil(t, err)
	})
}

func TestSyncSecret(t *testing.T) {
	t.Run("no-existing-secrets", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{}, nil).Once()
		client.On("Create", exampleSecret1.Name, exampleSecret1.Value).Return(nil).Once()
		client.On("Create", exampleSecret1.HashedName(), mock.AnythingOfType("string")).Return(nil).Once()

		updated, err := repository.SyncSecret(exampleSecret1, false)

		assert.Nil(t, err)
		assert.True(t, updated)
		assert.ElementsMatch(t, []string{
			client.Calls[1].Arguments[0].(string),
			client.Calls[2].Arguments[0].(string),
		}, []string{exampleSecret1.Name, exampleSecret1.HashedName()})
	})

	t.Run("outdated-secret", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{
			exampleSecret1.Name,
			exampleSecret1.HashedName() + "old",
			exampleSecret1.HashedName() + "old2",
		}, nil).Once()
		client.On("Update", exampleSecret1.Name, exampleSecret1.Value).Return(nil).Once()
		client.On("Create", exampleSecret1.HashedName(), mock.AnythingOfType("string")).Return(nil).Once()
		client.On("Delete", exampleSecret1.HashedName()+"old").Return(nil).Once()
		client.On("Delete", exampleSecret1.HashedName()+"old2").Return(nil).Once()

		updated, err := repository.SyncSecret(exampleSecret1, false)

		assert.Nil(t, err)
		assert.True(t, updated)
	})

	t.Run("unsynced-secret", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{
			exampleSecret1.Name,
		}, nil).Once()
		client.On("Update", exampleSecret1.Name, exampleSecret1.Value).Return(nil).Once()
		client.On("Create", exampleSecret1.HashedName(), mock.AnythingOfType("string")).Return(nil).Once()

		updated, err := repository.SyncSecret(exampleSecret1, false)

		assert.Nil(t, err)
		assert.True(t, updated)
		assert.ElementsMatch(t, []string{
			client.Calls[1].Arguments[0].(string),
			client.Calls[2].Arguments[0].(string),
		}, []string{exampleSecret1.Name, exampleSecret1.HashedName()})
	})

	t.Run("same-secret", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{
			exampleSecret1.Name,
			exampleSecret1.HashedName(),
			exampleSecret1.HashedName() + "extra",
		}, nil).Once()

		updated, err := repository.SyncSecret(exampleSecret1, false)
		assert.Nil(t, err)
		assert.False(t, updated)
	})

	t.Run("err-secret-list", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{}, errors.New("example")).Once()

		_, err := repository.SyncSecret(exampleSecret1, false)
		assert.NotNil(t, err)
	})

	t.Run("err-secret-create", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{}, nil).Once()
		client.On("Create", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(errors.New("example")).Once()

		_, err := repository.SyncSecret(exampleSecret1, false)
		assert.NotNil(t, err)
	})

	t.Run("err-secret-update", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{
			exampleSecret1.Name,
		}, nil).Once()
		client.On("Update", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(errors.New("example")).Once()

		_, err := repository.SyncSecret(exampleSecret1, false)
		assert.NotNil(t, err)
	})

	t.Run("dry-run", func(t *testing.T) {
		t.Run("no-update-required", func(t *testing.T) {
			repository, client := createMockSyncedSecretManager()
			client.On("List").Return([]string{
				exampleSecret1.Name,
				exampleSecret1.HashedName(),
			}, nil).Once()

			updated, err := repository.SyncSecret(exampleSecret1, true)
			assert.Nil(t, err)
			assert.False(t, updated)
		})

		t.Run("create-required", func(t *testing.T) {
			repository, client := createMockSyncedSecretManager()
			client.On("List").Return([]string{}, nil).Once()

			updated, err := repository.SyncSecret(exampleSecret1, true)
			assert.Nil(t, err)
			assert.True(t, updated)
		})

		t.Run("update-required", func(t *testing.T) {
			repository, client := createMockSyncedSecretManager()
			client.On("List").Return([]string{
				exampleSecret1.Name,
			}, nil).Once()

			updated, err := repository.SyncSecret(exampleSecret1, true)
			assert.Nil(t, err)
			assert.True(t, updated)
		})
	})
}

func TestSyncSecrets(t *testing.T) {
	t.Run("no-secrets", func(t *testing.T) {
		repository, _ := createMockSyncedSecretManager()

		updatedSecrets, err := repository.SyncSecrets([]Secret{}, false)
		assert.Nil(t, err)
		assert.Len(t, updatedSecrets, 0)
	})

	t.Run("updated-secrets", func(t *testing.T) {
		repository, client := createMockSyncedSecretManager()
		client.On("List").Return([]string{
			exampleSecret1.Name,
			exampleSecret1.HashedName(),
			exampleSecret2.Name,
			exampleSecret3.Name,
			exampleSecret3.HashedName() + "old",
		}, nil).Once()
		client.On("Create", exampleSecret2.HashedName(), mock.AnythingOfType("string")).Return(nil).Once()
		client.On("Create", exampleSecret3.HashedName(), mock.AnythingOfType("string")).Return(nil).Once()
		client.On("Update", exampleSecret2.Name, exampleSecret2.Value).Return(nil).Once()
		client.On("Update", exampleSecret3.Name, exampleSecret3.Value).Return(nil).Once()
		client.On("Delete", exampleSecret3.HashedName()+"old").Return(nil).Once()

		updatedSecrets, err := repository.SyncSecrets([]Secret{
			exampleSecret1,
			exampleSecret2,
			exampleSecret3,
		}, false)
		assert.Nil(t, err)
		assert.ElementsMatch(t, updatedSecrets, []SecretName{
			exampleSecret2.Name,
			exampleSecret3.Name,
		})
	})
}
