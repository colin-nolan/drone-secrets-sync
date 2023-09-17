#!/usr/bin/env bats

bats_require_minimum_version 1.5.0

: ${SUT:?}

SKIP_DRONE_CI_TEST_MESSAGE="tests against live Drone CI system but DRONE_TEST_SERVER and DRONE_TEST_TOKEN not set"
SKIP_REPOSITORY_TEST_MESSAGE="test requires DRONE_TEST_REPOSITORY to be set to refer to an active repository in Drone CI"
SKIP_ORGANISATION_TEST_MESSAGE="test requires DRONE_TEST_ORGANISATION to be set to refer to an active repository in Drone CI"

TEST_SECRET_NAME_1="_test-secret-1-${RANDOM}${RANDOM}-${BATS_TEST_NAME}"
TEST_SECRET_NAME_2="_test-secret-2-${RANDOM}${RANDOM}-${BATS_TEST_NAME}"
TEST_SECRET_NAME_3="_test-secret-3-${RANDOM}${RANDOM}-${BATS_TEST_NAME}"
TEST_SECRET_NAMES=( "${TEST_SECRET_NAME_1}" "${TEST_SECRET_NAME_2}" "${TEST_SECRET_NAME_3}" )
TEST_SECRET_1_INPUT="$(jq -c '.' <<< "{\"${TEST_SECRET_NAME_1}\": \"value\"}")"

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
    if "${TEST_AGAINST_DRONE}" && [[ -z ${DRONE_TEST_ORGANISATION+x} ]]; then
        return
    fi
    secrets_list="$(drone orgsecret ls --format '{{ .Name }}' --filter "${c}")"
    for secret_prefix in "${TEST_SECRET_NAMES[@]}"; do
        secrets_to_delete="$(grep "${secret_prefix}" <<< "${secrets_list}" | tr '\n' ' ' || true)"
        for secret_name in ${secrets_to_delete}; do
            drone orgsecret rm "${DRONE_TEST_ORGANISATION}" "${secret_name}"
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

skip_if_cannot_test_against_organisation() {
    skip_if_cannot_test_against_drone
    if [[ -z ${DRONE_TEST_ORGANISATION+x} ]]; then
        skip "${DRONE_TEST_ORGANISATION}"
    fi
}

add_secret() {
    local type="$1"
    local name="$2"
    local value="$3"

    if [[ "${type}" == "repository" || "${type}" == "repo" ]]; then
        drone secret add --name "${name}" --data "${value}" "${DRONE_TEST_REPOSITORY}"
    else
        drone orgsecret add "${DRONE_TEST_ORGANISATION}" "${name}" "${value}" 
    fi
}

secret_exists() {
    local type="$1"
    local name="$2"

    if [[ "${type}" == "repository" || "${type}" == "repo" ]]; then
        drone secret info --name "${name}" "${DRONE_TEST_REPOSITORY}"
    else
        drone orgsecret info "${DRONE_TEST_ORGANISATION}" "${name}"
    fi
}

@test "tool help" {
    ${SUT} --help
}

@test "invalid subcommand" {
    skip_if_cannot_test_against_drone
    run --separate-stderr ${SUT} invalid --help
    [ ${status} -ne 0 ]
}

@test "repository without namespace" {
    skip_if_cannot_test_against_drone
    run --separate-stderr ${SUT} repository missing-namespace <(echo '{}')
    [ ${status} -ne 0 ]
}

invalid_secret_json_test() {
    local subcommand="$1"
    local target="$2"

    run --separate-stderr ${SUT} "${subcommand}" "${target}" <(echo '{-}')

    [ ${status} -ne 0 ]
}

@test "repository operation with invalid JSON input" {
    skip_if_cannot_test_against_repository
    invalid_secret_json_test repository "${DRONE_TEST_REPOSITORY}"
}

@test "organisation operation with invalid JSON input" {
    skip_if_cannot_test_against_organisation
    invalid_secret_json_test organisation "${DRONE_TEST_ORGANISATION}"
}

unset_token_test() {
    local subcommand="$1"
    local target="$2"

    run --separate-stderr bash -c "DRONE_SERVER= ${SUT} -v "${subcommand}" "${target}" <<< '${TEST_SECRET_1_INPUT}'"

    [ ${status} -ne 0 ]
}

@test "repository operation without DRONE_SERVER set" {
    skip_if_cannot_test_against_repository
    unset_token_test repository "${DRONE_TEST_REPOSITORY}"
}

@test "organisation operation without DRONE_SERVER set" {
    skip_if_cannot_test_against_organisation
    unset_token_test organisation "${DRONE_TEST_ORGANISATION}"
}

incorrect_server_test() {
    local subcommand="$1"
    local target="$2"

    run --separate-stderr bash -c "DRONE_SERVER='${DRONE_SERVER}-wrong' ${SUT} -v "${subcommand}" "${target}" <<< '${TEST_SECRET_1_INPUT}'"

    [ ${status} -ne 0 ]
}

@test "repository operation with incorrect DRONE_SERVER set" {
    skip_if_cannot_test_against_repository
    incorrect_server_test repository "${DRONE_TEST_REPOSITORY}"
}

@test "organisation operation with incorrect DRONE_SERVER set" {
    skip_if_cannot_test_against_organisation
    incorrect_server_test organisation "${DRONE_TEST_ORGANISATION}"
}

unset_token_test() {
    local subcommand="$1"
    local target="$2"

    run --separate-stderr bash -c "DRONE_TOKEN= ${SUT} -v "${subcommand}" "${target}" <<< '${TEST_SECRET_1_INPUT}'"

    [ ${status} -ne 0 ]
}

@test "repository operation without DRONE_TOKEN set" {
    skip_if_cannot_test_against_repository
    unset_token_test repository "${DRONE_TEST_REPOSITORY}"
}

@test "organisation operation without DRONE_TOKEN set" {
    skip_if_cannot_test_against_organisation
    unset_token_test organisation "${DRONE_TEST_ORGANISATION}"
}

incorrect_token_test() {
    local subcommand="$1"
    local target="$2"

    run --separate-stderr bash -c "DRONE_TOKEN='${DRONE_TOKEN}-wrong' ${SUT} -v "${subcommand}" "${target}" <<< '${TEST_SECRET_1_INPUT}'"

    [ ${status} -ne 0 ]
}

@test "repository operation with incorrect DRONE_TOKEN set" {
    skip_if_cannot_test_against_repository
    incorrect_token_test repository "${DRONE_TEST_REPOSITORY}"
}

@test "organisation operation with incorrect DRONE_TOKEN set" {
    skip_if_cannot_test_against_organisation
    incorrect_token_test organisation "${DRONE_TEST_ORGANISATION}"
}

set_new_secret_test() {
    local subcommand="$1"
    local target="$2"

    run --separate-stderr ${SUT} "${subcommand}" -i 8 -l 16 -m 1024 -p 2 -v "${target}" \
        <(jq '.' <<< "{\"${TEST_SECRET_NAME_1}\": \"value\", \"${TEST_SECRET_NAME_2}\": \"value\"}")

    [ "${status}" -eq 0 ]
    [ $(jq '. | length' <<< "${output}") -eq 2 ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_1}\")" <<< "${output}") != null ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_2}\")" <<< "${output}") != null ]

    secret_exists "${subcommand}" "${TEST_SECRET_NAME_1}"
    secret_exists "${subcommand}" "${TEST_SECRET_NAME_2}"
}

@test "repository set new secrets" {
    skip_if_cannot_test_against_repository
    set_new_secret_test repository "${DRONE_TEST_REPOSITORY}"
}

@test "repo set new secrets" {
    skip_if_cannot_test_against_repository
    set_new_secret_test repo "${DRONE_TEST_REPOSITORY}"
}

@test "organisation set new secrets" {
    skip_if_cannot_test_against_organisation
    set_new_secret_test organisation "${DRONE_TEST_ORGANISATION}"
}

@test "org set new secrets" {
    skip_if_cannot_test_against_organisation
    set_new_secret_test org "${DRONE_TEST_ORGANISATION}"
}

update_changed_secret_test() {
    local subcommand="$1"
    local target="$2"
   
    add_secret "${subcommand}" "${TEST_SECRET_NAME_1}" value1

    run --separate-stderr ${SUT} "${subcommand}" "${target}" <(jq '.' <<< "{\"${TEST_SECRET_NAME_1}\": \"value2\"}")

    [ "${status}" -eq 0 ]
    [ $(jq '. | length' <<< "${output}") -eq 1 ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_1}\")" <<< "${output}") != null ]
    secret_exists "${subcommand}" "${TEST_SECRET_NAME_1}"
}

@test "repository update changed secret" {
    skip_if_cannot_test_against_repository
    update_changed_secret_test repository "${DRONE_TEST_REPOSITORY}"
}

@test "organisation update changed secret" {
    skip_if_cannot_test_against_organisation
    update_changed_secret_test organisation "${DRONE_TEST_ORGANISATION}"
}

update_unchanged_secret_test() {
    local subcommand="$1"
    local target="$2"
   
    ${SUT} "${subcommand}" "${target}" <<< "${TEST_SECRET_1_INPUT}"
    run --separate-stderr ${SUT} "${subcommand}" "${target}" <(echo "${TEST_SECRET_1_INPUT}")

    [ "${status}" -eq 0 ]
    [ "${output}" == "[]" ]
    secret_exists "${subcommand}" "${TEST_SECRET_NAME_1}"
}

@test "repository update unchanged secret" {
    skip_if_cannot_test_against_repository
    update_unchanged_secret_test repository "${DRONE_TEST_REPOSITORY}"
}

@test "organisation update unchanged secret" {
    skip_if_cannot_test_against_organisation
    update_unchanged_secret_test organisation "${DRONE_TEST_ORGANISATION}"
}

mixed_secrets_update_test() {
    local subcommand="$1"
    local target="$2"

    add_secret "${subcommand}" "${TEST_SECRET_NAME_1}" value
    ${SUT} "${subcommand}" "${target}" <(jq '.' <<< "{\"${TEST_SECRET_NAME_2}\": \"value\"}")
    ${SUT} "${subcommand}" "${target}" <(jq '.' <<< "{\"${TEST_SECRET_NAME_3}\": \"value\"}")

    run --separate-stderr ${SUT} "${subcommand}" "${target}" \
        <(jq '.' <<< "{\"${TEST_SECRET_NAME_1}\": \"next-value\", \"${TEST_SECRET_NAME_2}\": \"value\"}")
    
    [ "${status}" -eq 0 ]
    [ $(jq '. | length' <<< "${output}") -eq 1 ]
    [ $(jq ". | index(\"${TEST_SECRET_NAME_1}\")" <<< "${output}") != null ]
    secret_exists "${subcommand}" "${TEST_SECRET_NAME_1}"
    secret_exists "${subcommand}" "${TEST_SECRET_NAME_2}"
    secret_exists "${subcommand}" "${TEST_SECRET_NAME_3}"
}

@test "repository mixed secrets update" {
    skip_if_cannot_test_against_repository
    mixed_secrets_update_test repository "${DRONE_TEST_REPOSITORY}"
}

@test "organisation mixed secrets update" {
    skip_if_cannot_test_against_organisation
    mixed_secrets_update_test organisation "${DRONE_TEST_ORGANISATION}"
}