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
type RepositoryDroneSecretsManager struct {
	Client                  MinimalRepositoryClient
	Owner                   string
	Repository              string
	SecretHashConfiguration Argo2HashConfiguration
}

func (manager RepositoryDroneSecretsManager) List() ([]*drone.Secret, error) {
	return manager.Client.SecretList(manager.Owner, manager.Repository)
}

func (manager RepositoryDroneSecretsManager) Create(secretName string, secretValue string) (*drone.Secret, error) {
	return manager.Client.SecretCreate(manager.Owner, manager.Repository, &drone.Secret{
		Namespace: manager.Repository,
		Name:      secretName,
		Data:      secretValue,
	})
}

func (manager RepositoryDroneSecretsManager) Update(secretName string, secretValue string) (*drone.Secret, error) {
	return manager.Client.SecretUpdate(manager.Owner, manager.Repository, &drone.Secret{
		Namespace: manager.Repository,
		Name:      secretName,
		Data:      secretValue,
	})
}

func (manager RepositoryDroneSecretsManager) Delete(secretName string) error {
	return manager.Client.SecretDelete(manager.Owner, manager.Repository, secretName)
}
