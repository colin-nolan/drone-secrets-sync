[![Build Status](https://nobadkitty.tplinkdns.com:8900/api/badges/colin-nolan/drone-secrets-sync/status.svg)](https://nobadkitty.tplinkdns.com:8900/colin-nolan/drone-secrets-sync)
[![Code Coverage](https://codecov.io/gh/colin-nolan/drone-secrets-sync/graph/badge.svg?token=MS3WG5C1W5)](https://codecov.io/gh/colin-nolan/drone-secrets-sync)

## About

`drone-secrets-sync` is able to _idempotently_ synchronise [Drone CI](https://www.drone.io) repository and organisation secrets.

```shell
# Synchronise repository secrets from JSON file
drone-secrets-sync repo octocat/hello-world secrets.json

# Synchronise organisation secrets from JSON file
drone-secrets-sync org octocat secrets.json
```

```shell
# Synchronise multiple repository secrets from JSON map on stdin
echo '{"SOME_SECRET": "example", "OTHER_SECRET": "value"}' \
    | drone-secrets-sync repo octocat/hello-world

# Synchronise multiple organisation secrets from JSON map on stdin
echo '{"SOME_SECRET": "example", "OTHER_SECRET": "value"}' \
    | drone-secrets-sync org octocat
```

The tool will output what secrets have changed, e.g.

```json
["SOME_SECRET","OTHER_SECRET"]
```

The Drone CI API does not provide access to secret values. Therefore, to allow the determination as to whether a secret already contains the required value, two secrets are created:

1. The requested secret with the name, and value supplied.
1. A corresponding "hash secret", with a name that contains a salted hash of the secret value.

```shell
drone secret ls octocat/hello-world
```

```text
SECRET 
Pull Request Read:  false
Pull Request Write: false

SECRET___e861b26001c00803bb492889c1cf3faaf5a093ebc59f2c6838c7e10edfae4d0a 
Pull Request Read:  false
Pull Request Write: false
```

Be aware that exposing hashes makes it possible for an attacker that has gained access to the Drone API to brute force secret values offline. Hashes are generated using [Argon2](https://github.com/P-H-C/phc-winner-argon2/blob/master/argon2-specs.pdf) to make attacks more difficult. The memory and compute required to generate hashes can be configured.

## Installation

```shell
make install
```

## Usage

The tool uses the [Drone API](https://docs.drone.io/api/overview) via the official [drone-go](https://github.com/drone/drone-go) library. It requires `DRONE_TOKEN` and `DRONE_SERVER` environment variables to be set, e.g.

```shell
# Configure environment - see: https://docs.drone.io/cli/configure
export DRONE_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
export DRONE_SERVER=http://drone.mycompany.com
```

```text
Usage: drone-secrets-sync [--argon2-iterations ARGON2-ITERATIONS] [--argon2-length ARGON2-LENGTH] [--argon2-memory ARGON2-MEMORY] [--argon2-parallelism ARGON2-PARALLELISM] [--dry-run] [--droneserver DRONESERVER] [--dronetoken DRONETOKEN] [--verbose] <command> [<args>]

Options:
  --argon2-iterations ARGON2-ITERATIONS, -i ARGON2-ITERATIONS
                         number of argon2 iterations to create corresponding hash secret name [default: 32]
  --argon2-length ARGON2-LENGTH, -l ARGON2-LENGTH
                         length of argon2 hash used in corresponding hash secret name [default: 32]
  --argon2-memory ARGON2-MEMORY, -m ARGON2-MEMORY
                         memory for argon2 to use when creating corresponding hash secret name [default: 65536]
  --argon2-parallelism ARGON2-PARALLELISM, -p ARGON2-PARALLELISM
                         parallelism used when creating argon2 hash [default: 4]
  --dry-run, -d          indicate only what secrets would be updated; does not update secrets
  --droneserver DRONESERVER [env: DRONE_SERVER]
  --dronetoken DRONETOKEN [env: DRONE_TOKEN]
  --verbose, -v          enable verbose logging
  --help, -h             display this help and exit
  --version              display version and exit

Commands:
  repository             sync secrets for a repository
  repo                   sync secrets for a repository
  organisation           sync secrets for an organisation
  org                    sync secrets for an organisation
```

```text
Usage: drone-secrets-sync repo REPOSITORY [SECRETSFILE]

Positional arguments:
  REPOSITORY             repository to sync secrets for, e.g. octocat/hello-world
  SECRETSFILE            location to read secrets from (default: - (stdin))

Global options:
  --argon2-iterations ARGON2-ITERATIONS, -i ARGON2-ITERATIONS
                         number of argon2 iterations to create corresponding hash secret name [default: 32]
  --argon2-length ARGON2-LENGTH, -l ARGON2-LENGTH
                         length of argon2 hash used in corresponding hash secret name [default: 32]
  --argon2-memory ARGON2-MEMORY, -m ARGON2-MEMORY
                         memory for argon2 to use when creating corresponding hash secret name [default: 65536]
  --argon2-parallelism ARGON2-PARALLELISM, -p ARGON2-PARALLELISM
                         parallelism used when creating argon2 hash [default: 4]
  --dry-run, -d          indicate only what secrets would be updated; does not update secrets
  --droneserver DRONESERVER [env: DRONE_SERVER]
  --dronetoken DRONETOKEN [env: DRONE_TOKEN]
  --verbose, -v          enable verbose logging
  --help, -h             display this help and exit
  --version              display version and exit
```

```text
Usage: drone-secrets-sync org NAMESPACE [SECRETSFILE]

Positional arguments:
  NAMESPACE              name of organisation to sync secrets for, e.g. octocat
  SECRETSFILE            location to read secrets from (default: - (stdin))

Global options:
  --argon2-iterations ARGON2-ITERATIONS, -i ARGON2-ITERATIONS
                         number of argon2 iterations to create corresponding hash secret name [default: 32]
  --argon2-length ARGON2-LENGTH, -l ARGON2-LENGTH
                         length of argon2 hash used in corresponding hash secret name [default: 32]
  --argon2-memory ARGON2-MEMORY, -m ARGON2-MEMORY
                         memory for argon2 to use when creating corresponding hash secret name [default: 65536]
  --argon2-parallelism ARGON2-PARALLELISM, -p ARGON2-PARALLELISM
                         parallelism used when creating argon2 hash [default: 4]
  --dry-run, -d          indicate only what secrets would be updated; does not update secrets
  --droneserver DRONESERVER [env: DRONE_SERVER]
  --dronetoken DRONETOKEN [env: DRONE_TOKEN]
  --verbose, -v          enable verbose logging
  --help, -h             display this help and exit
  --version              display version and exit
```

## Development

### Build and Run

#### Executable

```shell
make build
```

To run after building:

```shell
./bin/drone-secrets-sync --help
```

#### Docker Image

```shell
make build-image-and-load
```

To run after building:

```shell
docker run --rm --pull never -e DRONE_SERVER -e DRONE_TOKEN "colin-nolan/drone-secrets-sync:$(make version)" --help
```

### Test

```shell
make test
```

### Linting

```shell
make lint
```

Requires:

- [golangci-lint](https://github.com/golangci/golangci-lint)
- [mdformat-gfm](https://github.com/executablebooks/mdformat)
- [jsonnetfmt](https://pkg.go.dev/github.com/google/go-jsonnet@v0.20.0/cmd/jsonnetfmt)

#### Apply Format

```shell
make format
```

Requires: (see Linting)

#### CI

To run a Drone CI step manually:

```shell
drone exec --pipeline=lint <(drone jsonnet --stream --stdout)
```

### Clear Secrets

When testing against a Drone CI installation, to clear all secrets on a repository:

```shell
repository=octocat/hello-world
drone secret ls --format '{{ .Name }}' "${repository}" \
    | xargs -I {} drone secret rm --name {} "${repository}"
```

Or on an organisation:

```shell
namespace=octocat
drone orgsecret ls --format '{{ .Name }}' "${namespace}" \
    | xargs -n 1 drone orgsecret rm "${namespace}"
```

Requires:

- [drone-cli](https://docs.drone.io/quickstart/cli/)

## Alternatives

- [drone-secret-sync](https://github.com/appleboy/drone-secret-sync) can synchronise secrets across multiple orgs/repositories. It is not idempotent though, meaning it will update all secrets, every time it is ran.

## Legal

GPL v3 (contact for other licencing). Copyright 2023 Colin Nolan.

This work is in no way related to any company that I may work for.
