package secrets

import (
	"fmt"

	"github.com/drone/drone-go/drone"
	"github.com/rs/zerolog/log"
)

// Create interface that is a subset of `drone.Client` to make testing simpler
type RepositoryClient interface {
	// SecretList returns a list of all repository secrets.
	SecretList(owner, name string) ([]*drone.Secret, error)

	// SecretCreate creates a registry.
	SecretCreate(owner, name string, secret *drone.Secret) (*drone.Secret, error)

	// SecretUpdate updates a registry.
	SecretUpdate(owner, name string, secret *drone.Secret) (*drone.Secret, error)

	// SecretDelete deletes a secret.
	SecretDelete(owner, name, secret string) error
}

// Secret manager for a Drone CI repository. Implemented against `GenericSecretsManager` interface.
type RepositorySecretsManager struct {
	Client    RepositoryClient
	Namespace string
	Name      string
}

func (manager RepositorySecretsManager) Repository() string {
	return fmt.Sprintf("%s/%s", manager.Namespace, manager.Name)
}

func (manager RepositorySecretsManager) List() ([]string, error) {
	log.Debug().Msgf("Getting list of secrets for repository: %s", manager.Repository())
	secrets, err := manager.Client.SecretList(manager.Namespace, manager.Name)
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
	log.Debug().Msgf("Creating secret in repository: %s:%s", manager.Repository(), secretName)
	_, err := manager.Client.SecretCreate(manager.Namespace, manager.Name, &drone.Secret{
		Namespace: manager.Namespace,
		Name:      secretName,
		Data:      secretValue,
	})
	return err
}

func (manager RepositorySecretsManager) Update(secretName string, secretValue string) error {
	log.Debug().Msgf("Updating secret in repository: %s:%s", manager.Repository(), secretName)
	_, err := manager.Client.SecretUpdate(manager.Namespace, manager.Name, &drone.Secret{
		Namespace: manager.Namespace,
		Name:      secretName,
		Data:      secretValue,
	})
	return err
}

func (manager RepositorySecretsManager) Delete(secretName string) error {
	log.Debug().Msgf("Deleting secret in repository: %s:%s", manager.Repository(), secretName)
	return manager.Client.SecretDelete(manager.Namespace, manager.Name, secretName)
}
