package secrets

import (
	"github.com/drone/drone-go/drone"
)

// Create interface that is a subset of `drone.Client` to make testing simpler
type OrganisationClient interface {
	// OrgSecretList returns a list of all repository secrets.
	OrgSecretList(namespace string) ([]*drone.Secret, error)

	// OrgSecretCreate creates a registry.
	OrgSecretCreate(namespace string, secret *drone.Secret) (*drone.Secret, error)

	// OrgSecretUpdate updates a registry.
	OrgSecretUpdate(namespace string, secret *drone.Secret) (*drone.Secret, error)

	// OrgSecretDelete deletes a secret.
	OrgSecretDelete(namespace, name string) error
}

// Secret manager for a Drone CI repository. Implemented against `GenericSecretsManager` interface.
type OrganisationSecretsManager struct {
	Client    OrganisationClient
	Namespace string
}

func (manager OrganisationSecretsManager) List() ([]string, error) {
	secrets, err := manager.Client.OrgSecretList(manager.Namespace)
	if err != nil {
		return nil, err
	}
	secretNames := make([]string, len(secrets))
	for i, secret := range secrets {
		secretNames[i] = secret.Name
	}
	return secretNames, nil
}

func (manager OrganisationSecretsManager) Create(secretName string, secretValue string) error {
	_, err := manager.Client.OrgSecretCreate(manager.Namespace, &drone.Secret{
		Namespace: manager.Namespace,
		Name:      secretName,
		Data:      secretValue,
	})
	return err
}

func (manager OrganisationSecretsManager) Update(secretName string, secretValue string) error {
	_, err := manager.Client.OrgSecretUpdate(manager.Namespace, &drone.Secret{
		Namespace: manager.Namespace,
		Name:      secretName,
		Data:      secretValue,
	})
	return err
}

func (manager OrganisationSecretsManager) Delete(secretName string) error {
	return manager.Client.OrgSecretDelete(manager.Namespace, secretName)
}
