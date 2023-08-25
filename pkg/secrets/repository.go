package secrets

import (
	"context"

	"github.com/drone/drone-go/drone"
	"golang.org/x/oauth2"
)

func CreateClient(host string, token string) drone.Client {
	// create an http client with oauth authentication.
	config := new(oauth2.Config)
	auther := config.Client(
		context.Background(),
		&oauth2.Token{
			AccessToken: token,
		},
	)

	return drone.NewClient(host, auther)
}
