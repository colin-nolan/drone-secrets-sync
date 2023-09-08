package secrets

import (
	"fmt"

	"github.com/derekparker/trie"
	"github.com/drone/drone-go/drone"

	"github.com/rs/zerolog/log"
)

// Create interface that is a subset of `drone.Client` to make testing simpler
type MinimalClient interface {
	SecretList(owner string, name string) ([]*drone.Secret, error)
	SecretCreate(owner string, name string, secret *drone.Secret) (*drone.Secret, error)
	SecretUpdate(owner string, name string, secret *drone.Secret) (*drone.Secret, error)
	SecretDelete(owner string, name string, secret string) error
}

// Secret manager for a Drone CI repository. Implemented against `SecretManager` interface.
type RepositorySecretManager struct {
	Client MinimalClient
	Owner  string
	Name   string
}

func (manager RepositorySecretManager) ListSecrets() ([]MaskedSecret, error) {
	log.Info().Msgf("Getting list of secrets for %s/%s", manager.Owner, manager.Name)
	secretEntries, err := manager.Client.SecretList(manager.Owner, manager.Name)
	if err != nil {
		return nil, fmt.Errorf("Error getting secrets for %s/%s", manager.Owner, manager.Name)
	}

	var secrets []MaskedSecret
	for _, secretEntry := range secretEntries {
		secrets = append(secrets, MaskedSecret{
			Name: secretEntry.Name,
		})
	}

	return secrets, nil
}

func (manager RepositorySecretManager) ListSyncedSecrets() ([]MaskedSecret, error) {
	secretsPrefixTree, err := manager.getSecretsPrefixTree()
	if err != nil {
		return nil, err
	}

	// Storing whether a secret has been considered as metadata on nodes in the prefix tree is an idea but the
	// implementation used does not (obviously) support updating a node's metadata
	consideredSecrets := map[SecretName]struct{}{}
	managedSecrets := []MaskedSecret{}

	for _, secretName := range secretsPrefixTree.Keys() {
		if _, ok := consideredSecrets[secretName]; ok {
			continue
		}
		consideredSecrets[secretName] = struct{}{}

		secret := MaskedSecret{Name: secretName}
		matched := secretsPrefixTree.PrefixSearch(secret.HashedNamePrefix())
		if len(matched) > 0 {
			managedSecrets = append(managedSecrets, secret)
			for _, match := range matched {
				consideredSecrets[match] = struct{}{}
			}
		}
	}

	return managedSecrets, nil
}

func (manager RepositorySecretManager) SyncSecret(secret Secret, dryRun bool) (updated bool, err error) {
	secretsPrefixTree, err := manager.getSecretsPrefixTree()
	if err != nil {
		return false, err
	}
	return manager.syncSecret(secret, secretsPrefixTree, dryRun)
}

func (manager RepositorySecretManager) SyncSecrets(secrets []Secret, dryRun bool) (updated []SecretName, err error) {
	if len(secrets) == 0 {
		return []SecretName{}, nil
	}
	secretsPrefixTree, err := manager.getSecretsPrefixTree()
	if err != nil {
		return nil, err
	}

	var updatedSecretNames = make([]SecretName, 0)
	for _, secret := range secrets {
		updated, err := manager.syncSecret(secret, secretsPrefixTree, dryRun)
		if updated {
			updatedSecretNames = append(updatedSecretNames, secret.Name)
		}
		if err != nil {
			return updatedSecretNames, err
		}
	}
	return updatedSecretNames, nil
}

func (manager RepositorySecretManager) getSecretsPrefixTree() (*trie.Trie, error) {
	secrets, err := manager.ListSecrets()
	if err != nil {
		return nil, err
	}

	secretsPrefixTree := trie.New()
	for _, secret := range secrets {
		secretsPrefixTree.Add(secret.Name, secret)
	}
	return secretsPrefixTree, nil
}

func (manager RepositorySecretManager) syncSecret(secret Secret, existingSecrets *trie.Trie, dryRun bool) (updated bool, err error) {
	secretIsNew := true
	if node, _ := existingSecrets.Find(secret.Name); node != nil {
		log.Debug().Msg("Secret already exists")
		secretIsNew = false

		// Check if the secret value is already up to date based on the corresponding hash secret
		if node, _ := existingSecrets.Find(secret.HashedName()); node != nil {
			return false, nil
		}
	}

	if dryRun {
		return true, nil
	}

	// Remove old secret hashes
	matched := existingSecrets.PrefixSearch(secret.HashedNamePrefix())
	for _, match := range matched {
		log.Info().Msgf("Deleting old hash secret: %s", secret.Name)
		err = manager.Client.SecretDelete(manager.Owner, manager.Name, match)
		if err != nil {
			return true, err
		}
	}

	// Adding/Updating secret
	droneSecret := drone.Secret{
		Namespace: manager.Owner,
		Name:      secret.Name,
		Data:      secret.Value,
	}
	if secretIsNew {
		log.Info().Msgf("Adding secret: %s", secret.Name)
		_, err = manager.Client.SecretCreate(manager.Owner, manager.Name, &droneSecret)
		if err != nil {
			return true, err
		}
	} else {
		log.Info().Msgf("Updating secret: %s", secret.Name)
		_, err = manager.Client.SecretUpdate(manager.Owner, manager.Name, &droneSecret)
		if err != nil {
			return true, err
		}
	}

	// Adding secret hash
	log.Info().Msgf("Adding secret hash: %s", secret.HashedName())
	_, err = manager.Client.SecretCreate(manager.Owner, manager.Name, &drone.Secret{
		Namespace: manager.Owner,
		Name:      secret.HashedName(),
		Data:      "1", // Secret must has a non-empty value
	})
	if err != nil {
		return true, err
	}

	return true, nil
}
