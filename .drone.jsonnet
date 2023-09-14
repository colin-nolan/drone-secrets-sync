// --------- Common ---------
local make_commands_fail_on_error(commands) =
  // Not adding `-o pipefail` because it is not supported by sh
  ['set -euf'] + commands;

local bypass_git_ownership_protection_command = 'git config --global --add safe.directory "$${PWD}"';

// Using arm64 not because it's required but due to CI resourcing - would ideally be "any" (https://github.com/jnohlgard/drone-yaml/tree/arch-any)
local build_arch = 'arm64';

// --------- Lint Pipeline ---------
local lint_pipeline = {
  kind: 'pipeline',
  type: 'docker',
  name: 'lint',
  platform: {
    arch: build_arch,
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
      commands: make_commands_fail_on_error([
        'apk add --update-cache git go make',
        'pip install mdformat-gfm',
        bypass_git_ownership_protection_command,
        'make lint-markdown',
      ]),
      depends_on: [],
    },
    {
      name: 'lint-jsonnet',
      // Could not  use `bitnami/jsonnet` as it has the user set to non-root
      image: 'alpine',
      commands: make_commands_fail_on_error([
        'apk add --update-cache git go make',
        'GOBIN=/usr/local/bin/ go install github.com/google/go-jsonnet/cmd/jsonnetfmt@latest',
        bypass_git_ownership_protection_command,
        'make lint-jsonnet',
      ]),
      depends_on: [],
    },
  ],
};

// --------- Test Pipeline ---------
local test_pipeline = {
  kind: 'pipeline',
  type: 'docker',
  name: 'test',
  platform: {
    arch: build_arch,
  },
  steps: [
    {
      name: 'unit-test',
      image: 'golang:alpine',
      commands: make_commands_fail_on_error([
        'apk add --update-cache gcc git libc-dev make',
        bypass_git_ownership_protection_command,
        'make test',
      ]),
    },
    {
      name: 'publish-coverage',
      image: 'alpine',
      commands: make_commands_fail_on_error([
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
      depends_on: ['unit-test'],
    },
  ],
};

// --------- Build Pipeline ---------
local supported_os_list = ['linux'];
local supported_arch_list = ['arm', 'arm64', 'amd64'];
local supported_os_arch_pairs = [
  [os, arch]
  for os in supported_os_list
  for arch in supported_arch_list
];

local tag_if_not_build_architecture(architecture) = if architecture != build_arch then { when: { event: ['tag'] } } else {};

local binary_build_step_name_prefix = 'build-binary_';
local binary_build_step(os, architecture) = {
  name: '%s%s-%s' % [binary_build_step_name_prefix, os, architecture],
  image: 'golang:alpine',
  commands: make_commands_fail_on_error([
    'apk add --update-cache git make',
    bypass_git_ownership_protection_command,
    'make build GOOS=%s GOARCH=%s' % [os, architecture],
  ]),
  depends_on: [],
} + tag_if_not_build_architecture(architecture);

local image_build_step_name_prefix = 'build-image_';
local image_build_step(os, architecture) = {
  name: '%s%s-%s' % [image_build_step_name_prefix, os, architecture],
  image: 'golang:alpine',
  commands: make_commands_fail_on_error([
    'apk add --update-cache git make',
    bypass_git_ownership_protection_command,
    'make build-image GOOS=%s GOARCH=%s KANIKO_EXECUTOR=build/third-party/kaniko/out/executor' % [os, architecture],
  ]),
  depends_on: ['build-kaniko-tool'],
} + tag_if_not_build_architecture(architecture);

local create_image_publish_step(name_postfix, tag_expression) = {
  name: 'publish-image-%s' % name_postfix,
  image: 'alpine',
  commands: make_commands_fail_on_error([
    'apk --update-cache add git go make skopeo',
    bypass_git_ownership_protection_command,
    'echo "$${DOCKER_TOKEN}" | docker login --password-stdin --username "$${DOCKER_USERNAME}"',
    'skopeo copy --all dir:build/release/$$(make version)/multiarch docker://colinnolan/drone-secrets-sync:%s' % tag_expression,
  ]),
  environment: {
    DOCKER_USERNAME: {
      from_secret: 'dockerhub_username',
    },
    DOCKER_TOKEN: {
      from_secret: 'dockerhub_token',
    },
  },
  when: {
    event: ['tag'],
  },
  depends_on: ['build-multiarch-image'],
};

local find_build_steps(step_name_prefix, steps) = std.filter(function(name) std.startsWith(name, step_name_prefix), std.map(function(step) step.name, steps));

local build_pipeline = {
  kind: 'pipeline',
  type: 'docker',
  name: 'build',
  platform: {
    arch: build_arch,
  },
  steps:
    [binary_build_step(x[0], x[1]) for x in supported_os_arch_pairs] +
    [{
      // Unfortunately, we cannot use the official kaniko image for the image builds because Drone CI converts commands
      // into a shell script and the kaniko image does not have a shell (https://docs.drone.io/pipeline/docker/syntax/steps/#commands)
      name: 'build-kaniko-tool',
      image: 'golang:alpine',
      commands: make_commands_fail_on_error([
        'apk add --update-cache bash git make',
        'if [[ ! -d build/third-party/kaniko ]]; then git clone --depth=1 --branch=main https://github.com/GoogleContainerTools/kaniko.git build/third-party/kaniko; fi',
        'cd build/third-party/kaniko',
        'make out/executor',
      ]),
      depends_on: [],
    }] +
    [image_build_step(x[0], x[1]) for x in supported_os_arch_pairs] +
    [
      {
        name: 'build-multiarch-image',
        image: 'alpine',
        commands: make_commands_fail_on_error([
          'apk --update-cache add bash git go jq make skopeo',
          bypass_git_ownership_protection_command,
          'make build-image-multiarch',
        ]),
        depends_on: find_build_steps(image_build_step_name_prefix, build_pipeline.steps),
        when: {
          event: ['tag'],
        },
      },
      create_image_publish_step('latest', 'latest'),
      create_image_publish_step('release', '$$(make version)'),
      {
        name: 'link-latest',
        image: 'alpine',
        commands: make_commands_fail_on_error([
          'apk add --update-cache git go make',
          'echo git config --global --add safe.directory "$${PWD}"',
          bypass_git_ownership_protection_command,
          'mkdir -p build/release',
          'version="$(make version)"; cd build/release && ln -f -s "$${version}" latest && cd -',
        ]),
        when: {
          event: ['tag'],
        },
        depends_on: [],
      },
      {
        name: 'publish-github-release',
        image: 'plugins/github-release:latest',
        settings: {
          api_key: {
            from_secret: 'github_release_token',
          },
          files: [
            'build/release/latest/drone-secrets-sync*',
          ],
        },
        when: {
          event: ['tag'],
        },
        depends_on: find_build_steps(binary_build_step_name_prefix, build_pipeline.steps) + find_build_steps(image_build_step_name_prefix, build_pipeline.steps) + ['link-latest'],
      },
    ],
};

// --------- Finalise ---------
[
  lint_pipeline,
  test_pipeline,
  build_pipeline,
]
