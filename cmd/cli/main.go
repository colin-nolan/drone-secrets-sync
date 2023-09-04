package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/colin-nolan/drone-secrets-manager/pkg/client"
	"github.com/colin-nolan/drone-secrets-manager/pkg/secrets"

	"github.com/alexflint/go-arg"
)

func main() {
	// TODO: CLI function
	var args struct {
		Repository string `arg:"positional"`
		// Log      string    `arg:"positional,required"`
		// Debug    bool      `arg:"-d" help:"turn on debug mode"`
		// RealMode bool      `arg:"--real"`
		Wr io.Writer `arg:"-"`
	}
	arg.MustParse(&args)

	syncSecrets(args.Repository)
}

func syncSecrets(repository string) {
	credential, err := client.GetCredentialFromEnv()
	if err != nil {
		log.Fatal().Err(err)
	}
	client := client.CreateClient(credential)

	repositoryOwner, repositoryName := parseRepository(repository)
	repositorySecretManager := secrets.RepositorySecretManager{
		Client: client,
		Owner:  repositoryOwner,
		Name:   repositoryName,
	}

	secrets := readSecretsFromStdin()
	synced, err := repositorySecretManager.SyncSecrets(secrets, false)
	if err != nil {
		log.Fatal().Err(err)
	}

	data, err := json.Marshal(synced)
	if err != nil {
		log.Fatal().Err(err)
	}
	fmt.Println(string(data))
}

func parseRepository(repository string) (owner string, name string) {
	repositorySplit := strings.Split(repository, "/")
	if len(repositorySplit) != 2 {
		log.Fatal().Msg("Repository must be in the format <owner>/<name>")
	}
	return repositorySplit[0], repositorySplit[1]
}

func readSecretsFromStdin() []secrets.Secret {
	inputData, err := io.ReadAll(os.Stdin)
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
		secretValuePairs = append(secretValuePairs, secrets.NewSecret(key, value.(string)))
	}
	return secretValuePairs
}
