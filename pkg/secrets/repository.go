package secrets

import (
	"log"

	"github.com/derekparker/trie"
	"github.com/drone/drone-go/drone"
)

// An interface for managing secrets in an external source.
type SecretManager interface {
	// Gets secrets - both "synced" (those with a matching hash secret) and those without
	ListSecrets() []MaskedSecret

	// Gets secrets that are "synced" (those with a matching hash secret)
	ListSyncedSecrets() []MaskedSecret

	// Synchronizes a single secret.
	//
	// `updated` is set to `true` if the secret is updated.
	//
	// Does not make actual changes if `dryRun` is `true`.
	SyncSecret(secret Secret, dryRun bool) (updated bool, err error)

	// Synchronizes a list of secrets
	//
	// `updated` is populated with the names of the secrets that were updated.
	//
	// Does not make actual changes if `dryRun` is `true`.
	SyncSecrets(secrets []Secret, dryRun bool) (updated []SecretName, err error)
}

// Secret manager for a Drone CI repository.
type RepositorySecretManager struct {
	Client drone.Client
	Owner  string
	Name   string
}

func (manager RepositorySecretManager) ListSecrets() []MaskedSecret {
	secretEntries, err := manager.Client.SecretList(manager.Owner, manager.Name)
	if err != nil {
		log.Fatalf("Error getting secrets for repository %s/%s: %s", manager.Owner, manager.Name, err)
	}

	var secrets []MaskedSecret
	for _, secretEntry := range secretEntries {
		secrets = append(secrets, MaskedSecret{
			Name: secretEntry.Name,
		})
	}

	return secrets
}

func (manager RepositorySecretManager) ListSyncedSecrets() []MaskedSecret {
	secretsPrefixTree := manager.getSecretsPrefixTree()

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

	return managedSecrets
}

func (manager RepositorySecretManager) SyncSecret(secret Secret, dryRun bool) (updated bool, err error) {
	return manager.syncSecret(secret, manager.getSecretsPrefixTree(), dryRun)
}

func (manager RepositorySecretManager) SyncSecrets(secrets []Secret, dryRun bool) (updated []SecretName, err error) {
	var updatedSecretNames []SecretName = make([]string, 0)
	for _, secret := range secrets {
		updated, err := manager.syncSecret(secret, manager.getSecretsPrefixTree(), dryRun)
		if updated {
			updatedSecretNames = append(updatedSecretNames, secret.Name)
		}
		if err != nil {
			return updatedSecretNames, err
		}
	}
	return updatedSecretNames, nil
}

func (manager RepositorySecretManager) getSecretsPrefixTree() *trie.Trie {
	secretsPrefixTree := trie.New()
	for _, secret := range manager.ListSecrets() {
		secretsPrefixTree.Add(secret.Name, secret)
	}
	return secretsPrefixTree
}

func (manager RepositorySecretManager) syncSecret(secret Secret, secrets *trie.Trie, dryRun bool) (updated bool, err error) {
	if node, _ := secrets.Find(secret.Name); node != nil {
		// Check if the secret value is already up to date based on the corresponding hash secret
		if node, _ := secrets.Find(secret.HashedName()); node != nil {
			return false, nil
		}

		if !dryRun {
			// Remove old secret (required to avoid unique constraint error)
			err = manager.Client.SecretDelete(manager.Owner, manager.Name, secret.Name)
			if err != nil {
				return true, err
			}
		}
	}

	if dryRun {
		return true, nil
	}

	// Remove old secret hashes
	matched := secrets.PrefixSearch(secret.HashedNamePrefix())
	for _, match := range matched {
		err = manager.Client.SecretDelete(manager.Owner, manager.Name, match)
		if err != nil {
			return true, err
		}
	}

	// Add new secret and secret hash
	for _, secretEntry := range []drone.Secret{
		{
			Namespace: manager.Owner,
			Name:      secret.Name,
			Data:      secret.Value,
		},
		{
			Namespace: manager.Owner,
			Name:      secret.HashedName(),
			Data:      "1", // Secret must has a non-empty value
		}} {
		_, err = manager.Client.SecretCreate(manager.Owner, manager.Name, &secretEntry)
		if err != nil {
			return true, err
		}

	}

	return true, nil
}
