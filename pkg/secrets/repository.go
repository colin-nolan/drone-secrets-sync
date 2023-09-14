package secrets

import (
	"github.com/drone/drone-go/drone"
)

// Create interface that is a subset of `drone.Client` to make testing simpler
type MinimalRepositoryClient interface {
	SecretList(owner string, name string) ([]*drone.Secret, error)
	SecretCreate(owner string, name string, secret *drone.Secret) (*drone.Secret, error)
	SecretUpdate(owner string, name string, secret *drone.Secret) (*drone.Secret, error)
	SecretDelete(owner string, name string, secret string) error
}

// Secret manager for a Drone CI repository. Implemented against `DroneSecretsManager` interface.
type RepositorySecretsManager struct {
	Client     MinimalRepositoryClient
	Owner      string
	Repository string
}

func (manager RepositorySecretsManager) List() ([]string, error) {
	secrets, err := manager.Client.SecretList(manager.Owner, manager.Repository)
	if err != nil {
		return nil, err
	}
	secretNames := make([]string, len(secrets))
	for i, secret := range secrets {
		secretNames[i] = secret.Name
	}
	return secretNames, nil
}

func (manager RepositorySecretsManager) Create(secretName string, secretValue string) error {
	_, err := manager.Client.SecretCreate(manager.Owner, manager.Repository, &drone.Secret{
		Namespace: manager.Repository,
		Name:      secretName,
		Data:      secretValue,
	})
	return err
}

func (manager RepositorySecretsManager) Update(secretName string, secretValue string) error {
	_, err := manager.Client.SecretUpdate(manager.Owner, manager.Repository, &drone.Secret{
		Namespace: manager.Repository,
		Name:      secretName,
		Data:      secretValue,
	})
	return err
}

func (manager RepositorySecretsManager) Delete(secretName string) error {
	return manager.Client.SecretDelete(manager.Owner, manager.Repository, secretName)
}
