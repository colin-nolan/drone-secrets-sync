package main

import (
	"fmt"

	"github.com/colin-nolan/drone-secrets-manager/pkg/secrets"
)

const (
	token = "example"
	host  = "http://drone.company.com"
)

func main() {
	client := secrets.CreateClient(host, token)

	// gets the current user
	user, err := client.Self()
	fmt.Println(user, err)

	// gets the named repository information
	repo, err := client.Repo("drone", "drone-go")
	fmt.Println(repo, err)
}
