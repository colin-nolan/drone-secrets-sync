#!/usr/bin/env bash

set -euf -o pipefail

output_directory="$1"
shift
image_tarball_locations=$@

manifest_location="${output_directory}/manifest.json"

rm -rf "${output_directory}"
mkdir -p "${output_directory}"

temp_directory="$(mktemp -d)"
trap "rm -r ${temp_directory}" EXIT

jq '.' <<< '{
    "schemaVersion": 2,
    "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
    "manifests": []
}' > "${manifest_location}"

for container_image in ${image_tarball_locations}; do \
    extract_directory="${temp_directory}/$(basename ${container_image} .tar)"
    
    skopeo copy "docker-archive:${container_image}" "dir:${extract_directory}"

    skopeo_manifest=$(skopeo inspect "dir:${extract_directory}")
    manifest_digest=$(jq -r '.Digest' <<< "${skopeo_manifest}" | cut -d: -f2)
    # TODO: might need to so something with "size"
    manifest=$(jq -s '{
            "mediaType": .[1].mediaType,
            "digest": .[0].Digest,
            "platform": {
                "architecture": .[0].Architecture,
                "os": .[0].Os
            }
        }' <(echo "${skopeo_manifest}") <(cat "${extract_directory}/manifest.json"))
    old_manifest="$(cat "${manifest_location}")"
    jq ".manifests += [${manifest}]" <<< "${old_manifest}" > "${manifest_location}"

    mv "${extract_directory}/manifest.json" "${extract_directory}/${manifest_digest}.manifest.json"
    find "${extract_directory}" -type f ! -name version -exec mv {} "${output_directory}" \; 
done

>&2 echo "Complete!"
>&2 echo "You can now publish the multiarch image using: skopeo copy --all dir:${output_directory} docker://colinnolan/drone-secrets-sync"