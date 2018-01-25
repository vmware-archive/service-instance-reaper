#!/usr/bin/env bash

set -euo pipefail

readonly SCS_SECRETS_LAST_PASS_ID="5793183462"

secrets_file=$(mktemp)

fetch_secrets() {
  lpass show --notes "${SCS_SECRETS_LAST_PASS_ID}" > "${secrets_file}"
}

set_pipeline() {
  fly -t scs set-pipeline -p service-instance-reaper -c pipeline.yml -l "${secrets_file}"
}

cleanup() {
  rm ${secrets_file}
}

main() {
  pushd $(dirname $0) > /dev/null

  fetch_secrets
  set_pipeline

  popd > /dev/null
}

trap "cleanup" EXIT

main
