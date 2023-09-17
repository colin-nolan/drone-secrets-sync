// --------- Common ---------
local make_commands_fail_on_error(commands) =
  // Not adding `-o pipefail` because it is not supported by sh
  ['set -euf'] + commands;

// Using arm64 not because it's required but due to CI resourcing - would ideally be "any" (https://github.com/jnohlgard/drone-yaml/tree/arch-any)
local build_arch = 'arm64';

local create_setup_commands(extra_apk_packages=[]) = make_commands_fail_on_error([
  'apk add --update-cache git go make %s' % std.join(' ', extra_apk_packages),
  'git config --global --add safe.directory "$${PWD}"',
]);

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
      commands: make_commands_fail_on_error([
        'make lint-code',
      ]),
      depends_on: [],
    },
    {
      name: 'lint-markdown',
      image: 'python:3-alpine',
      commands: create_setup_commands() + [
        'pip install mdformat-gfm',
        'make lint-markdown',
      ],
      depends_on: [],
    },
    {
      name: 'lint-jsonnet',
      // Was not able tp use `bitnami/jsonnet` as it has the user set to non-root
      image: 'alpine',
      commands: create_setup_commands() + [
        'GOBIN=/usr/local/bin/ go install github.com/google/go-jsonnet/cmd/jsonnetfmt@latest',
        'make lint-jsonnet',
      ],
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
      name: 'unit-tests',
      image: 'golang:alpine',
      commands: create_setup_commands(['gcc', 'libc-dev']) + [
        'make test-unit',
      ],
      depends_on: [],
    },
    {
      name: 'system-tests',
      image: 'golang:alpine',
      commands: create_setup_commands(['bash', 'gcc', 'jq', 'parallel', 'libc-dev']) + [
        // XXX: It would be better to `go install` from github but I could not get it to work for drone-cli
        |||
          git clone --depth=1 --branch master https://github.com/harness/drone-cli.git /tmp/drone-cli
          cd /tmp/drone-cli
          go install ./...
          cd -
        |||,
        'git submodule update --init --recursive',
        'make test-system',
      ],
      environment: {
        DRONE_TEST_SERVER: {
          from_secret: 'drone_test_server',
        },
        DRONE_TEST_TOKEN: {
          from_secret: 'drone_test_token',
        },
        DRONE_TEST_REPOSITORY: {
          from_secret: 'drone_test_repository',
        },
        DRONE_TEST_ORGANISATION: {
          from_secret: 'drone_test_organisation',
        },
      },
      depends_on: [],
    },
    {
      name: 'compile-coverage-report',
      image: 'golang:alpine',
      commands: create_setup_commands() + [
        'make test-coverage-report',
      ],
      depends_on: ['unit-tests', 'system-tests'],
    },
    {
      // Installing codecov uploader from source to support any runner arch
      name: 'codecov-builder',
      image: 'node:16-alpine',
      commands: make_commands_fail_on_error([
        'apk add --update-cache curl git',
        'repository_directory="$${PWD}"',
        'git clone --depth=1 --branch=main https://github.com/codecov/uploader.git /tmp/uploader',
        'cd /tmp/uploader',
        'npm install',
        'npm run build',
        'npx pkg . --targets alpine --output "$${repository_directory}/build/third-party/codecov"',
      ]),
      depends_on: [],
    },
    {
      name: 'publish-coverage',
      image: 'alpine',
      commands: [
        'codecov',
      ],
      environment: {
        CODECOV_TOKEN: {
          from_secret: 'codecov_token',
        },
      },
      depends_on: ['codecov-builder', 'compile-coverage-report'],
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
  commands: create_setup_commands() + [
    'make build GOOS=%s GOARCH=%s' % [os, architecture],
  ],
  depends_on: [],
} + tag_if_not_build_architecture(architecture);

local image_build_step_name_prefix = 'build-image_';
local image_build_step(os, architecture) = {
  name: '%s%s-%s' % [image_build_step_name_prefix, os, architecture],
  image: 'golang:alpine',
  commands: create_setup_commands() + [
    'make build-image GOOS=%s GOARCH=%s KANIKO_EXECUTOR=build/third-party/kaniko/out/executor' % [os, architecture],
  ],
  depends_on: ['build-kaniko-tool'],
} + tag_if_not_build_architecture(architecture);

local create_image_publish_step(name_postfix, tag_expression) = {
  name: 'publish-image-%s' % name_postfix,
  image: 'alpine',
  commands: create_setup_commands(['docker', 'skopeo']) + [
    'echo "$${DOCKER_TOKEN}" | docker login --password-stdin --username "$${DOCKER_USERNAME}"',
    'skopeo copy --all dir:build/release/$$(make version)/multiarch docker://colinnolan/drone-secrets-sync:%s' % tag_expression,
  ],
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
      commands: create_setup_commands(['bash']) + [
        |||
          if [[ ! -d build/third-party/kaniko ]]; then 
            git clone --depth=1 --branch=main https://github.com/GoogleContainerTools/kaniko.git build/third-party/kaniko
          fi
        |||,
        'cd build/third-party/kaniko',
        'make out/executor',
      ],
      depends_on: [],
    }] +
    [image_build_step(x[0], x[1]) for x in supported_os_arch_pairs] +
    [
      {
        name: 'build-multiarch-image',
        image: 'alpine',
        commands: create_setup_commands(['bash', 'jq', 'skopeo']) + [
          'make build-image-multiarch',
        ],
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
        commands: create_setup_commands() + [
          'mkdir -p build/release',
          |||
            version="$(make version)"
            cd build/release
            ln -f -s "$${version}" latest
          |||,
        ],
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
            'CHANGELOG.md',
            'README.md',
            'LICENCE.txt',
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
