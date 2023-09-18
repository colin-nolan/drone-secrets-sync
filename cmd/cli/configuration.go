package main

import (
	"strings"

	"github.com/colin-nolan/drone-secrets-sync/pkg/secrets"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Configuration struct {
	SecretsFile               string
	LogLevel                  zerolog.Level
	HashConfiguration         secrets.Argo2HashConfiguration
	RepositoryConfiguration   *RepositoryConfiguration
	OrganisationConfiguration *OrganisationConfiguration
	DryRun                    bool
}

type RepositoryConfiguration struct {
	Repository string
}

func (configuration *RepositoryConfiguration) RepositoryNamespace() string {
	namespace, _ := parseRepository(configuration.Repository)
	return namespace
}

func (configuration *RepositoryConfiguration) RepositoryName() string {
	_, name := parseRepository(configuration.Repository)
	return name
}

type OrganisationConfiguration struct {
	Namespace string
}

func parseRepository(repository string) (namespace string, name string) {
	repositorySplit := strings.Split(repository, "/")
	if len(repositorySplit) != 2 {
		log.Fatal().Msg("Repository must be in the format <namespace>/<name>")
	}
	return repositorySplit[0], repositorySplit[1]
}
