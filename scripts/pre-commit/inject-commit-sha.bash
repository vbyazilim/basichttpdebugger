#!/usr/bin/env bash

set -e
set -o pipefail
set -o errexit
set -o nounset

target_file="internal/release/release.go"
latest_sha=$(git rev-parse HEAD)

sed -i '' "s/var BuildInformation = \".*\"/var BuildInformation = \"${latest_sha}\"/" "${target_file}"

if git diff --quiet "${target_file}"; then
    exit 0
else
    git add "${target_file}"
fi
