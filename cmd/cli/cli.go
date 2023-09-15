package main

import (
	"fmt"
	"strings"

	"github.com/alexflint/go-arg"
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

// To be set on compilation (should not be `const`)
var version = "unknown"

type organisationCmd struct {
	Namespace   string `arg:"positional,required" help:"name of organisation to sync secrets for, e.g. octocat"`
	SecretsFile string `arg:"positional" default:"-" help:"location to read secrets from (default: - (stdin))"`
}

type repositoryCmd struct {
	Repository  string `arg:"positional,required" help:"repository to sync secrets for, e.g. octocat/hello-world"`
	SecretsFile string `arg:"positional" default:"-" help:"location to read secrets from (default: - (stdin))"`
}

type cliArgs struct {
	Repository            *repositoryCmd   `arg:"subcommand:repository" help:"sync secrets for a repository"`
	Organisation          *organisationCmd `arg:"subcommand:organisation" help:"sync secrets for an organisation"`
	Argon2HashIterations  uint32           `arg:"-i,--argon2-iterations" default:"32" help:"number of argon2 iterations to create corresponding hash secret name"`
	Argon2HashLength      uint32           `arg:"-l,--argon2-length" default:"32" help:"length of argon2 hash used in corresponding hash secret name"`
	Argon2HashMemory      uint32           `arg:"-m,--argon2-memory" default:"65536" help:"memory for argon2 to use when creating corresponding hash secret name"`
	Argon2HashParallelism uint8            `arg:"-p,--argon2-parallelism" default:"4" help:"parallelism used when creating argon2 hash"`
	Verbose               bool             `arg:"-v,--verbose" help:"enable verbose logging"`
}

func (cliArgs) Version() string {
	return fmt.Sprintf("drone-secrets-sync %s", version)
}

func ReadCliArgs() Configuration {
	var args cliArgs
	parser := arg.MustParse(&args)

	logLevel := zerolog.WarnLevel
	if args.Verbose {
		logLevel = zerolog.DebugLevel
	}

	configuration := Configuration{
		LogLevel: logLevel,
		HashConfiguration: secrets.Argo2HashConfiguration{
			Iterations:  args.Argon2HashIterations,
			Memory:      args.Argon2HashMemory,
			Parallelism: args.Argon2HashParallelism,
			Length:      args.Argon2HashLength,
		},
	}
	if args.Repository != nil {
		configuration.SecretsFile = args.Repository.SecretsFile
		configuration.RepositoryConfiguration = &RepositoryConfiguration{
			Repository: args.Repository.Repository,
		}
	} else if args.Organisation != nil {
		configuration.SecretsFile = args.Organisation.SecretsFile
		configuration.OrganisationConfiguration = &OrganisationConfiguration{
			Namespace: args.Organisation.Namespace,
		}
	} else {
		parser.Fail("No subcommand specified")
	}
	return configuration
}
