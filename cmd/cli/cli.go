package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/colin-nolan/drone-secrets-sync/pkg/client"
	"github.com/colin-nolan/drone-secrets-sync/pkg/secrets"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

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

func ReadSecrets(sourceFile string, hashConfiguration secrets.Argo2HashConfiguration) []secrets.Secret {
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
	log.Debug().Msgf("Input data: %s", inputData)

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

func ReadCredential() client.Credential {
	credential, err := client.GetCredentialFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting credentials from environment")
	}
	return credential
}
