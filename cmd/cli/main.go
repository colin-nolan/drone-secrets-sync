package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/colin-nolan/drone-secrets-manager/pkg/secrets"
)

func getUsage() string {
	return "Usage: drone-secrets-manager [--help]"
}

func main() {
	helpFlag := flag.Bool("help", false, "Print help")
	flag.Parse()

	if *helpFlag {
		fmt.Println(getUsage())
		return
	}

	syncSecrets()
}

func syncSecrets() {
	credential, err := secrets.GetCredentialFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	client := secrets.CreateClient(credential)

	repositorySecretManager := secrets.RepositorySecretManager{
		Client: client,
		Owner:  "colin-nolan",
		Name:   "drone-testing",
	}

	secrets := readSecretsFromStdin()
	synced, err := repositorySecretManager.SyncSecrets(secrets, false)
	if err != nil {
		log.Fatal(err)
	}

	data, err := json.Marshal(synced)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func readSecretsFromStdin() []secrets.Secret {
	inputData, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal("Error reading from stdin:", err)
	}

	var secretValueMap map[string]interface{}
	err = json.Unmarshal([]byte(inputData), &secretValueMap)
	if err != nil {
		log.Fatal("Error parsing JSON from stdin:", err)
	}

	var secretValuePairs []secrets.Secret
	for key, value := range secretValueMap {
		secretValuePairs = append(secretValuePairs, secrets.NewSecret(key, value.(string)))
	}
	return secretValuePairs
}
