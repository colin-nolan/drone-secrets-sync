// --------- Common ---------
local make_commands_fail_on_error(commands) =
  // Not adding `-o pipefail` because it is not supported by sh
  ['set -euf'] + commands;

// Using arm64 not because it's required but due to CI resourcing - would ideally be "any" (https://github.com/jnohlgard/drone-yaml/tree/arch-any)
local build_arch = 'arm64';

local create_setup_commands(extra_apk_packages=[]) = make_commands_fail_on_error([
  'apk add --update-cache bash git go make %s' % std.join(' ', extra_apk_packages),
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
        'build/third-party/codecov',
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
local target_builds = ['linux/arm', 'linux/arm64', 'linux/amd64', 'darwin/amd64', 'darwin/arm64'];
local target_platforms = ['linux/arm', 'linux/arm64', 'linux/amd64'];

local run_only_on_tag_if_not_build_architecture(architecture) = if architecture != build_arch then { when: { event: ['tag'] } } else {};

local binary_build_step_name_prefix = 'build-binary_';
local binary_build_step(target) =
  local architecture = std.split(target, '/')[1];
  {
    name: '%s%s' % [binary_build_step_name_prefix, std.strReplace(target, '/', '-')],
    image: 'golang:alpine',
    commands: create_setup_commands() + [
      'make build TARGET_BUILD=%s' % [target],
    ],
    depends_on: [],
  } + run_only_on_tag_if_not_build_architecture(architecture);

local image_build_step_name_prefix = 'build-image_';
local image_build_step(platform) =
  local architecture = std.split(platform, '/')[1];
  {
    name: '%s%s' % [image_build_step_name_prefix, std.strReplace(platform, '/', '-')],
    image: 'golang:alpine',
    commands: create_setup_commands() + [
      'make build-image TARGET_PLATFORM=%s KANIKO_EXECUTOR=build/third-party/kaniko/out/executor' % [platform],
    ],
    depends_on: ['build-kaniko-tool'],
  } + run_only_on_tag_if_not_build_architecture(architecture);

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
    [binary_build_step(target_build) for target_build in target_builds] +
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
    [image_build_step(target_platform) for target_platform in target_platforms] +
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
