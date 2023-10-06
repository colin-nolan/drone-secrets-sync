package secrets

import (
	"fmt"

	"github.com/derekparker/trie"

	"github.com/rs/zerolog/log"
)

type GenericSecretsManager interface {
	// Return a list of all secret names.
	// It returns a slice of strings representing the names of the secrets and an error if any occurs.
	List() ([]string, error)

	// Creates a new secret with the given name and value.
	// It takes two parameters: secretName (name of the secret) and secretValue (value of the secret).
	// It returns an error if the creation process fails.
	Create(secretName string, secretValue string) error

	// Updates an existing secret with the given name and new value.
	// It takes two parameters: secretName (name of the secret) and secretValue (new value of the secret).
	// It returns an error if the update operation fails.
	Update(secretName string, secretValue string) error

	// Deletes an existing secret based on its name.
	// It takes one parameter: secretName (name of the secret).
	// It returns an error if the deletion process fails.
	Delete(secretName string) error
}

type SyncedSecretManager struct {
	GenericSecretManager GenericSecretsManager
}

func (manager SyncedSecretManager) ListSecrets() ([]MaskedSecret, error) {
	secretNames, err := manager.GenericSecretManager.List()
	if err != nil {
		return nil, fmt.Errorf("error getting secrets: %w", err)
	}

	var secrets []MaskedSecret
	for _, secretName := range secretNames {
		secrets = append(secrets, MaskedSecret{
			Name: secretName,
		})
	}

	return secrets, nil
}

func (manager SyncedSecretManager) ListSyncedSecrets() ([]MaskedSecret, error) {
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

func (manager SyncedSecretManager) SyncSecret(secret Secret, dryRun bool) (updated bool, err error) {
	secretsPrefixTree, err := manager.getSecretsPrefixTree()
	if err != nil {
		return false, err
	}
	return manager.syncSecret(secret, secretsPrefixTree, dryRun)
}

func (manager SyncedSecretManager) SyncSecrets(secrets []Secret, dryRun bool) (updated []SecretName, err error) {
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

func (manager SyncedSecretManager) getSecretsPrefixTree() (*trie.Trie, error) {
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

func (manager SyncedSecretManager) syncSecret(secret Secret, existingSecrets *trie.Trie, dryRun bool) (updated bool, err error) {
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
		err = manager.GenericSecretManager.Delete(match)
		if err != nil {
			return true, err
		}
	}

	// Adding/Updating secret
	if secretIsNew {
		log.Info().Msgf("Adding secret: %s", secret.Name)
		err = manager.GenericSecretManager.Create(secret.Name, secret.Value)
		if err != nil {
			return true, err
		}
	} else {
		log.Info().Msgf("Updating secret: %s", secret.Name)
		err = manager.GenericSecretManager.Update(secret.Name, secret.Value)
		if err != nil {
			return true, err
		}
	}

	// Adding secret hash
	log.Info().Msgf("Adding secret hash: %s", secret.HashedName())
	err = manager.GenericSecretManager.Create(secret.HashedName(), "1") // Secret must has a non-empty value
	if err != nil {
		return true, err
	}

	return true, nil
}
