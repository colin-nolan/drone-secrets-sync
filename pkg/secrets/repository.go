package secrets

import (
	"context"
	"fmt"
	"os"

	"github.com/drone/drone-go/drone"
	"golang.org/x/oauth2"
)

const (
	DroneHostVariable  = "DRONE_HOST"
	DroneTokenVariable = "DRONE_TOKEN"
)

type Credential struct {
	host  string
	token string
}

// Retrieves Drone credentials (host and token) from environment variables.
//
// This function reads the values of environment variables DroneHostVariable and DroneTokenVariable
// and returns them as a Credential struct. It checks that both environment variables are set and
// non-empty. If either of them is not set or empty, it returns an empty Credential and an error
// describing which environment variable is missing or empty.
//
// Returns:
//   - credential: A struct containing the host and token.
//   - error: An error describing any issues with environment variable values.
func GetCredentialFromEnv() (credential Credential, err error) {
	host := os.Getenv(DroneHostVariable)
	if host == "" {
		return Credential{}, fmt.Errorf("%s environment variables must be set and non-empty", DroneHostVariable)
	}
	token := os.Getenv(DroneTokenVariable)
	if token == "" {
		return Credential{}, fmt.Errorf("%s environment variables must be set and non-empty", DroneTokenVariable)
	}

	return Credential{
		host:  host,
		token: token,
	}, nil
}

// Creates a Drone client with the provided credential.
//
// This function takes a Credential struct as input, containing the Drone host and token.
// It creates an OAuth2 client configuration and authorizes it using the provided access token.
// Then, it constructs a Drone client using the specified host and the authorized OAuth2 client.
//
// Parameters:
//   - credential: A Credential struct containing the Drone host and access token.
//
// Returns:
//   - drone.Client: A configured Drone client for making API requests to the specified Drone instance.
func CreateClient(credential Credential) drone.Client {
	config := new(oauth2.Config)
	auther := config.Client(
		context.Background(),
		&oauth2.Token{
			AccessToken: credential.token,
		},
	)

	return drone.NewClient(credential.host, auther)
}
