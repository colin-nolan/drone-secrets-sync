package secrets

import (
	"fmt"

	"github.com/derekparker/trie"
	"github.com/drone/drone-go/drone"

	"github.com/rs/zerolog/log"
)

type DroneSecretsManager interface {
	List() ([]*drone.Secret, error)
	Create(secretName string, secretValue string) (*drone.Secret, error)
	Update(secretName string, secretValue string) (*drone.Secret, error)
	Delete(secretName string) error
}

type SecretManager struct {
	DroneSecretManager DroneSecretsManager
}

func (manager SecretManager) ListSecrets() ([]MaskedSecret, error) {
	secretEntries, err := manager.DroneSecretManager.List()
	if err != nil {
		return nil, fmt.Errorf("error getting secrets for")
	}

	var secrets []MaskedSecret
	for _, secretEntry := range secretEntries {
		secrets = append(secrets, MaskedSecret{
			Name: secretEntry.Name,
		})
	}

	return secrets, nil
}

func (manager SecretManager) ListSyncedSecrets() ([]MaskedSecret, error) {
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

func (manager SecretManager) SyncSecret(secret Secret, dryRun bool) (updated bool, err error) {
	secretsPrefixTree, err := manager.getSecretsPrefixTree()
	if err != nil {
		return false, err
	}
	return manager.syncSecret(secret, secretsPrefixTree, dryRun)
}

func (manager SecretManager) SyncSecrets(secrets []Secret, dryRun bool) (updated []SecretName, err error) {
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

func (manager SecretManager) getSecretsPrefixTree() (*trie.Trie, error) {
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

func (manager SecretManager) syncSecret(secret Secret, existingSecrets *trie.Trie, dryRun bool) (updated bool, err error) {
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
		err = manager.DroneSecretManager.Delete(match)
		if err != nil {
			return true, err
		}
	}

	// Adding/Updating secret
	if secretIsNew {
		log.Info().Msgf("Adding secret: %s", secret.Name)
		_, err = manager.DroneSecretManager.Create(secret.Name, secret.Value)
		if err != nil {
			return true, err
		}
	} else {
		log.Info().Msgf("Updating secret: %s", secret.Name)
		_, err = manager.DroneSecretManager.Update(secret.Name, secret.Value)
		if err != nil {
			return true, err
		}
	}

	// Adding secret hash
	log.Info().Msgf("Adding secret hash: %s", secret.HashedName())
	_, err = manager.DroneSecretManager.Create(secret.HashedName(), "1") // Secret must has a non-empty value
	if err != nil {
		return true, err
	}

	return true, nil
}
