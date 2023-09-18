package main

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/colin-nolan/drone-secrets-sync/pkg/client"
	"github.com/colin-nolan/drone-secrets-sync/pkg/secrets"
)

func main() {
	configuration := ReadCliArgs()
	zerolog.SetGlobalLevel(zerolog.Level(configuration.LogLevel))

	secretsToSync := ReadSecrets(configuration.SecretsFile, configuration.HashConfiguration)

	syncedSecretManager := createSyncedSecretManager(configuration)
	updatedSecrets, err := syncedSecretManager.SyncSecrets(secretsToSync, configuration.DryRun)
	if err != nil {
		log.Fatal().Err(err).Msg("Error syncing secrets")
	}

	output(updatedSecrets)
}

func createSyncedSecretManager(configuration Configuration) secrets.SyncedSecretManager {
	client := client.CreateClient(configuration.DroneCredential)

	var genericSecretsManager secrets.GenericSecretsManager
	if configuration.RepositoryConfiguration != nil {
		genericSecretsManager = secrets.RepositorySecretsManager{
			Client:    client,
			Namespace: configuration.RepositoryConfiguration.RepositoryNamespace(),
			Name:      configuration.RepositoryConfiguration.RepositoryName(),
		}
	} else {
		genericSecretsManager = secrets.OrganisationSecretsManager{
			Client:    client,
			Namespace: configuration.OrganisationConfiguration.Namespace,
		}
	}

	return secrets.SyncedSecretManager{GenericSecretManager: genericSecretsManager}
}

func output(updatedSecrets []string) {
	data, err := json.Marshal(updatedSecrets)
	if err != nil {
		log.Fatal().Err(err).Msg("Error marshalling updated secrets for output")
	}
	fmt.Println(string(data))
}
