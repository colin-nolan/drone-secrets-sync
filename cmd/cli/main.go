package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/colin-nolan/drone-secrets-manager/pkg/secrets"
)

func getUsage() string {
	return "Usage: drone-secrets-manager [--help] key value"
}

func main() {
	helpFlag := flag.Bool("help", false, "Print help")
	if *helpFlag {
		fmt.Print(getUsage())
		return
	}

	if len(os.Args) != 3 {
		log.Fatal(fmt.Sprintf("Invalid number of arguments\n%s", getUsage()))
	}
	key := os.Args[1]
	value := os.Args[2]

	syncSecret(key, value)
}

func syncSecret(key string, value string) {
	credential, err := secrets.GetCredentialFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	client := secrets.CreateClient(credential)

	user, err := client.Self()
	fmt.Println(user, err)

	// client.Secret()

	repo, err := client.Repo("drone", "drone-go")
	fmt.Println(repo, err)
}
