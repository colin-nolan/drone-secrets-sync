package client

import (
	"context"

	"github.com/drone/drone-go/drone"
	"golang.org/x/oauth2"
)

const (
	DroneServerVariable = "DRONE_SERVER"
	DroneTokenVariable  = "DRONE_TOKEN"
)

type Credential struct {
	Server string
	Token  string
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
			AccessToken: credential.Token,
		},
	)

	return drone.NewClient(credential.Server, auther)
}
