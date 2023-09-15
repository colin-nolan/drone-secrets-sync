package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/colin-nolan/drone-secrets-sync/pkg/client"
	"github.com/colin-nolan/drone-secrets-sync/pkg/secrets"
)

func main() {
	configuration := ReadCliArgs()
	secretsToSync := readSecrets(configuration.SecretsFile, configuration.HashConfiguration)
	credential := readCredential()

	zerolog.SetGlobalLevel(zerolog.Level(configuration.LogLevel))

	client := client.CreateClient(credential)
	var genericSecretsManager secrets.GenericSecretsManager
	if configuration.RepositoryConfiguration != nil {
		genericSecretsManager = secrets.RepositorySecretsManager{
			Client: client,
			// XXX: use on a repository in a namespace not owned by the same user has not been tested
			Owner:     configuration.RepositoryConfiguration.RepositoryNamespace(),
			Namespace: configuration.RepositoryConfiguration.RepositoryNamespace(),
			Name:      configuration.RepositoryConfiguration.RepositoryName(),
		}
	} else {
		genericSecretsManager = secrets.OrganisationSecretsManager{
			Client:    client,
			Namespace: configuration.OrganisationConfiguration.Namespace,
		}
	}
	syncedSecretManager := secrets.SyncedSecretManager{GenericSecretManager: genericSecretsManager}

	updatedSecrets, err := syncedSecretManager.SyncSecrets(secretsToSync, false)
	if err != nil {
		log.Fatal().Err(err).Msg("Error syncing secrets")
	}

	output(updatedSecrets)
}

func readSecrets(sourceFile string, hashConfiguration secrets.Argo2HashConfiguration) []secrets.Secret {
	var inputData []byte
	var err error
	if sourceFile == "-" {
		inputData, err = io.ReadAll(os.Stdin)
	} else {
		inputData, err = os.ReadFile(sourceFile)
	}
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading from stdin")
	}

	var secretValueMap map[string]interface{}
	err = json.Unmarshal([]byte(inputData), &secretValueMap)
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing JSON from stdin")
	}

	var secretValuePairs []secrets.Secret
	for key, value := range secretValueMap {
		secretValuePairs = append(secretValuePairs, secrets.NewSecret(key, value.(string), hashConfiguration))
	}
	return secretValuePairs
}

func readCredential() client.Credential {
	credential, err := client.GetCredentialFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting credentials from environment")
	}
	return credential
}

func output(updatedSecrets []string) {
	data, err := json.Marshal(updatedSecrets)
	if err != nil {
		log.Fatal().Err(err).Msg("Error marshalling updated secrets for output")
	}
	fmt.Println(string(data))
}
