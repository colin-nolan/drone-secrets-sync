local makeCommandsFailOnError(commands) =
  // Not adding `-o pipefail` because it is not supported by sh
  ['set -euf'] + commands;

local lintPipeline = {
  kind: 'pipeline',
  type: 'docker',
  name: 'lint',
  platform: {
    // Using arm64 not because it's required but due to CI resourcing - would ideally be "any" (https://github.com/jnohlgard/drone-yaml/tree/arch-any)
    arch: 'arm64',
  },
  steps: [
    {
      name: 'lint-code',
      image: 'golangci/golangci-lint',
      commands: [
        'make lint-code',
      ],
      depends_on: [],
    },
    {
      name: 'lint-markdown',
      image: 'python:3-alpine',
      commands: makeCommandsFailOnError([
        'apk add --update-cache git go make',
        'pip install mdformat-gfm',
        'git config --global --add safe.directory "$(pwd)"',
        'make lint-markdown',
      ]),
      depends_on: [],
    },
    {
      name: 'lint-jsonnet',
      // Could not  use `bitnami/jsonnet` as it has the user set to non-root
      image: 'alpine',
      commands: makeCommandsFailOnError([
        'apk add --update-cache git go make',
        'GOBIN=/usr/local/bin/ go install github.com/google/go-jsonnet/cmd/jsonnetfmt@latest',
        'git config --global --add safe.directory "$(pwd)"',
        'make lint-jsonnet',
      ]),
      depends_on: [],
    },
  ],
};

local testPipeline = {
  kind: 'pipeline',
  type: 'docker',
  name: 'test',
  platform: {
    arch: 'arm64',
  },
  steps: [
    {
      name: 'test',
      image: 'golang:alpine',
      commands: makeCommandsFailOnError([
        'apk add --update-cache gcc git libc-dev make',
        'git config --global --add safe.directory "$(pwd)"',
        'make test',
      ]),
    },
    {
      name: 'publish-coverage',
      image: 'alpine',
      commands: makeCommandsFailOnError([
        'apk add --update-cache curl',
        // XXX: This is an arch specific binary
        'curl -fsL https://uploader.codecov.io/latest/aarch64/codecov > /usr/local/bin/codecov',
        'chmod +x /usr/local/bin/codecov',
        'codecov',
      ]),
      environment: {
        CODECOV_TOKEN: {
          from_secret: 'codecov_token',
        },
      },
      depends_on: ['test'],
    },
  ],
};

local supportedOsList = ['linux'];
local supportedArchList = ['arm', 'arm64', 'amd64'];
local supportedOsArchPairs = [
  [os, arch]
  for os in supportedOsList
  for arch in supportedArchList
];

local binary_build_step(os, architecture) = {
  name: 'build-binary_%s-%s' % [os, architecture],
  image: 'golang:alpine',
  commands: makeCommandsFailOnError([
    'apk add --update-cache git make',
    'git config --global --add safe.directory "$(pwd)"',
    'make build GOOS=%s GOARCH=%s' % [os, architecture],
  ]),
  depends_on: [],
};

local container_build_step_name_prefix = "build-container_";
local container_build_step(os, architecture) = {
  name: '%s%s-%s' % [container_build_step_name_prefix, os, architecture],
  image: 'golang:alpine',
  commands: makeCommandsFailOnError([
    'apk add --update-cache git make',
    'git config --global --add safe.directory "$(pwd)"',
    'make build-container GOOS=%s GOARCH=%s KANIKO_EXECUTOR=build/third-party/kaniko/out/executor' % [os, architecture],
  ]),
  depends_on: ['build-kaniko-tool'],
};


local buildPipeline = {
  kind: 'pipeline',
  type: 'docker',
  name: 'build',
  platform: {
    arch: 'arm64',
  },
  steps:
    [binary_build_step(x[0], x[1]) for x in supportedOsArchPairs] +
    [{
      // Unfortunately, we cannot use the official kaniko image for the container builds because Drone CI converts commands
      // into a shell script and the kaniko image does not have a shell (https://docs.drone.io/pipeline/docker/syntax/steps/#commands)
      name: 'build-kaniko-tool',
      image: 'golang:alpine',
      commands: makeCommandsFailOnError([
        'apk add --update-cache bash git make',
        'if [[ ! -d build/third-party/kaniko ]]; then git clone --depth=1 --branch=main https://github.com/GoogleContainerTools/kaniko.git build/third-party/kaniko; fi',
        'cd build/third-party/kaniko',
        'make out/executor',
      ]),
      depends_on: [],
    }] +
    [container_build_step(x[0], x[1]) for x in supportedOsArchPairs] +
    [
      {
        name: 'link-latest',
        image: 'alpine',
        commands: makeCommandsFailOnError([
          'apk add --update-cache git go make',
          'echo git config --global --add safe.directory "$${PWD}"',
          'git config --global --add safe.directory "$${PWD}"',
          'mkdir -p build/release',
          'version="$(make version)"; cd build/release && ln -f -s "$${version}" latest && cd -',
        ]),
        depends_on: [],
      },
      // FIXME: dependecny
      // {
      //   name: 'publish-github-release',
      //   image: 'plugins/github-release:latest',
      //   settings: {
      //     api_key: {
      //       from_secret: 'github_release_token',
      //     },
      //     files: ['build/release/latest/*'],
      //   },
      //   when: {
      //     event: ['tag'],
      //   },
      //   depends_on: std.filter(function(name) name != self.name, std.map(function(step) step.name, buildPipeline.steps)),
      // },
      // TODO: make a common build multiarch step
      {
        name: 'publish-dockerhub-latest',
        image: 'alpine',
        commands: makeCommandsFailOnError([
          'apk --update-cache add bash git go jq make skopeo',
          'git config --global --add safe.directory "$${PWD}"',
          'make build-container-multiarch',
          'skopeo copy --all dir:build/release/$$(make version)/multiarch docker://colinnolan/drone-secrets-sync:latest'
        ]),
        depends_on: std.filter(function(name) std.startsWith(name, container_build_step_name_prefix), std.map(function(step) step.name, buildPipeline.steps))
      }
    ],
};

[
  lintPipeline,
  testPipeline,
  buildPipeline,
]
