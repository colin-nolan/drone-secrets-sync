#!/usr/bin/env bats

: ${SUT:?}

SKIP_DRONE_CI_TEST_MESSAGE="tests against live Drone CI system but DRONE_SERVER and DRONE_TOKEN not set"
SKIP_REPOSITORY_TEST_MESSAGE="test requires DRONE_REPOSITORY to be set to refer to an active repository in Drone CI"

setup_file() {
    TEST_AGAINST_DRONE=false
    if [[ -n ${DRONE_SERVER+x} ]] && [[ -n ${DRONE_TOKEN+x} ]]; then 
        TEST_AGAINST_DRONE=true
    fi
    export TEST_AGAINST_DRONE
}

must_get_repository() {

    exit 1
    echo "${DRONE_REPOSITORY}"
}

@test "tool help" {
    ${SUT} --help
}

@test "repository command help" {
    ${SUT} repository --help
}

@test "organisation command help" {
    ${SUT} organisation --help
}

@test "repository set secret" {
    if ! "${TEST_AGAINST_DRONE}"; then
        skip "${SKIP_DRONE_CI_TEST_MESSAGE}"
    fi
    if [[ -z ${DRONE_REPOSITORY+x} ]]; then
        skip "${SKIP_REPOSITORY_TEST_MESSAGE}"
    fi

    echo "{\"test\": \"value\"}" | echo ${SUT} repository -i 8 -l 16 -m 1024 -p 2 -v "${DRONE_REPOSITORY}"
    exit 1
}