package secrets

// An interface for managing secrets in an external source.
type SecretManager interface {
	// Gets secrets - both "synced" (those with a matching hash secret) and those without
	ListSecrets() ([]MaskedSecret, error)

	// Gets secrets that are "synced" (those with a matching hash secret)
	ListSyncedSecrets() ([]MaskedSecret, error)

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
