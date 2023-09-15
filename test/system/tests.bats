#!/usr/bin/env bats

bats_require_minimum_version 1.5.0

: ${SUT:?}

SKIP_DRONE_CI_TEST_MESSAGE="tests against live Drone CI system but DRONE_TEST_SERVER and DRONE_TEST_TOKEN not set"
SKIP_REPOSITORY_TEST_MESSAGE="test requires DRONE_TEST_REPOSITORY to be set to refer to an active repository in Drone CI"

TEST_SECRET_NAME_1="_test-secret-1-${RANDOM}${RANDOM}"
TEST_SECRET_NAME_2="_test-secret-2-${RANDOM}${RANDOM}"
TEST_SECRET_NAMES=( "${TEST_SECRET_NAME_1}" "${TEST_SECRET_NAME_2}" )
TEST_SECRET_INPUT="$(jq -c '.' <<< "{\"${TEST_SECRET_NAME_1}\": \"value\"}")"

setup_file() {
    TEST_AGAINST_DRONE=false
    if [[ -n ${DRONE_TEST_SERVER+x} ]] && [[ -n ${DRONE_TEST_TOKEN+x} ]]; then 
        TEST_AGAINST_DRONE=true
    fi
    export TEST_AGAINST_DRONE

    # Requires a test server to be set explicitly to prevent unintended usage against a production server
    export DRONE_SERVER="${DRONE_TEST_SERVER}"
    export DRONE_TOKEN="${DRONE_TEST_TOKEN}"
}

teardown() {
    cleanup_repository_secrets
    cleanup_organisation_secrets
}

cleanup_repository_secrets() {
    if "${TEST_AGAINST_DRONE}" && [[ -z ${DRONE_TEST_REPOSITORY+x} ]]; then
        return
    fi
    secrets_list="$(drone secret ls --format '{{ .Name }}' "${DRONE_TEST_REPOSITORY}")"
    for secret_prefix in "${TEST_SECRET_NAMES[@]}"; do
        secrets_to_delete="$(grep "${secret_prefix}" <<< "${secrets_list}" | tr '\n' ' ' || true)"
        for secret_name in ${secrets_to_delete}; do
            drone secret rm --name "${secret_name}" "${DRONE_TEST_REPOSITORY}"
        done
    done
}

cleanup_organisation_secrets() {
    if "${TEST_AGAINST_DRONE}" && [[ -z ${DRONE_ORGANISATION+x} ]]; then
        return
    fi
    secrets_list="$(drone orgsecret ls --format '{{ .Name }}' --filter "${c}")"
    for secret_prefix in "${TEST_SECRET_NAMES[@]}"; do
        secrets_to_delete="$(grep "${secret_prefix}" <<< "${secrets_list}" | tr '\n' ' ' || true)"
        for secret_name in ${secrets_to_delete}; do
            drone orgsecret rm "${DRONE_ORGANISATION}" "${secret_name}"
        done
    done 
}

skip_if_cannot_test_against_drone() {
    if ! "${TEST_AGAINST_DRONE}"; then
        skip "${SKIP_DRONE_CI_TEST_MESSAGE}"
    fi
}

skip_if_cannot_test_against_repository() {
    skip_if_cannot_test_against_drone
    if [[ -z ${DRONE_TEST_REPOSITORY+x} ]]; then
        skip "${SKIP_REPOSITORY_TEST_MESSAGE}"
    fi
}

@test "tool help" {
    ${SUT} --help
}

@test "repository set new secrets" {
    skip_if_cannot_test_against_repository

    run --separate-stderr ${SUT} repository -i 8 -l 16 -m 1024 -p 2 -v "${DRONE_TEST_REPOSITORY}" \
        <(jq '.' <<< "{\"${TEST_SECRET_NAME_1}\": \"value\", \"${TEST_SECRET_NAME_2}\": \"value\"}")

    [ "${status}" -eq 0 ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_1}\")" <<< "${output}") != null ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_2}\")" <<< "${output}") != null ]

    drone secret info --name "${TEST_SECRET_NAME_1}" "${DRONE_TEST_REPOSITORY}"
    drone secret info --name "${TEST_SECRET_NAME_2}" "${DRONE_TEST_REPOSITORY}"
}

@test "repository update changed secret" {
    skip_if_cannot_test_against_repository

    drone secret add --name "${TEST_SECRET_NAME_1}" --data "value1" "${DRONE_TEST_REPOSITORY}"

    run ${SUT} repository "${DRONE_TEST_REPOSITORY}" <(jq '.' <<< "{\"${TEST_SECRET_NAME_1}\": \"value2\"}")

    [ "${status}" -eq 0 ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_1}\")" <<< "${output}") != null ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_2}\")" <<< "${output}") == null ]
}

@test "repository update unchanged secret" {
    skip_if_cannot_test_against_repository

    ${SUT} repository "${DRONE_TEST_REPOSITORY}" <<< "${TEST_SECRET_INPUT}"
    run ${SUT} repository "${DRONE_TEST_REPOSITORY}" <(echo "${TEST_SECRET_INPUT}")

    [ "${status}" -eq 0 ]
    [ "${output}" == "[]" ]
}

@test "repository without namespace" {
    skip_if_cannot_test_against_drone
    run --separate-stderr ${SUT} repository missing-namespace <(echo '{}')
    [ ${status} -ne 0 ]
}

@test "repository with invalid JSON input" {
    skip_if_cannot_test_against_repository
    run --separate-stderr ${SUT} repository "${DRONE_TEST_REPOSITORY}" <(echo '{-}')
    [ ${status} -ne 0 ]
}

@test "repository without DRONE_SERVER set" {
    skip_if_cannot_test_against_repository
    run --separate-stderr bash -c "DRONE_SERVER= ${SUT} -v repository "${DRONE_TEST_REPOSITORY}" <<< '${TEST_SECRET_INPUT}'"
    [ ${status} -ne 0 ]
}

@test "repository without DRONE_TOKEN set" {
    skip_if_cannot_test_against_repository
    run --separate-stderr bash -c "DRONE_TOKEN= ${SUT} -v repository "${DRONE_TEST_REPOSITORY}" <<< '${TEST_SECRET_INPUT}'"
    [ ${status} -ne 0 ]
}