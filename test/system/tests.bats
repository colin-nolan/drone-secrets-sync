#!/usr/bin/env bats

bats_require_minimum_version 1.5.0

: ${SUT:?}

SKIP_DRONE_CI_TEST_MESSAGE="tests against live Drone CI system but DRONE_SERVER and DRONE_TOKEN not set"
SKIP_REPOSITORY_TEST_MESSAGE="test requires DRONE_REPOSITORY to be set to refer to an active repository in Drone CI"

TEST_SECRET_NAME_1="_test-secret-1-${RANDOM}${RANDOM}"
TEST_SECRET_NAME_2="_test-secret-2-${RANDOM}${RANDOM}"
TEST_SECRET_NAMES=( "${TEST_SECRET_NAME_1}" "${TEST_SECRET_NAME_2}" )

setup_file() {
    TEST_AGAINST_DRONE=false
    if [[ -n ${DRONE_SERVER+x} ]] && [[ -n ${DRONE_TOKEN+x} ]]; then 
        TEST_AGAINST_DRONE=true
    fi
    export TEST_AGAINST_DRONE
}

teardown() {
    cleanup_repository_secrets
    cleanup_organisation_secrets
}

cleanup_repository_secrets() {
    if "${TEST_AGAINST_DRONE}" && [[ -z ${DRONE_REPOSITORY+x} ]]; then
        return
    fi
    secrets_list="$(drone secret ls --format '{{ .Name }}' "${DRONE_REPOSITORY}")"
    for secret_prefix in "${TEST_SECRET_NAMES[@]}"; do
        secrets_to_delete="$(grep "${secret_prefix}" <<< "${secrets_list}" | tr '\n' ' ' || true)"
        for secret_name in ${secrets_to_delete}; do
            drone secret rm --name "${secret_name}" "${DRONE_REPOSITORY}"
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
    if [[ -z ${DRONE_REPOSITORY+x} ]]; then
        skip "${SKIP_REPOSITORY_TEST_MESSAGE}"
    fi
}

@test "tool help" {
    ${SUT} --help
}

@test "repository set new secrets" {
    skip_if_cannot_test_against_repository

    run --separate-stderr ${SUT} repository -i 8 -l 16 -m 1024 -p 2 -v "${DRONE_REPOSITORY}" \
        <(jq '.' <<< "{\"${TEST_SECRET_NAME_1}\": \"value\", \"${TEST_SECRET_NAME_2}\": \"value\"}")

    [ "${status}" -eq 0 ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_1}\")" <<< "${output}") != null ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_2}\")" <<< "${output}") != null ]

    drone secret info --name "${TEST_SECRET_NAME_1}" "${DRONE_REPOSITORY}"
    drone secret info --name "${TEST_SECRET_NAME_2}" "${DRONE_REPOSITORY}"
}

@test "repository update changed secret" {
    skip_if_cannot_test_against_repository

    drone secret add --name "${TEST_SECRET_NAME_1}" --data "value1" "${DRONE_REPOSITORY}"

    run ${SUT} repository "${DRONE_REPOSITORY}" <(jq '.' <<< "{\"${TEST_SECRET_NAME_1}\": \"value2\"}")

    [ "${status}" -eq 0 ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_1}\")" <<< "${output}") != null ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_2}\")" <<< "${output}") == null ]
}

@test "repository update unchanged secret" {
    skip_if_cannot_test_against_repository

    secret_input="$(jq '.' <<< "{\"${TEST_SECRET_NAME_1}\": \"value\"}")"
    ${SUT} repository "${DRONE_REPOSITORY}" <<< "${secret_input}"
    run ${SUT} repository "${DRONE_REPOSITORY}" <(echo "${secret_input}")

    [ "${status}" -eq 0 ]
    [ "${output}" == "[]" ]
}