#!/bin/bash

set -euo pipefail

TERRAFORM_VERSIONS=""
TOFU_VERSIONS=""
CLICKHOUSE_VERSIONS=""

for arg in "$@"; do
  case "$arg" in
    --terraform-versions=*)
      TERRAFORM_VERSIONS="${arg#*=}"
      ;;
    --tofu-versions=*)
      TOFU_VERSIONS="${arg#*=}"
      ;;
    --clickhouse-versions=*)
      CLICKHOUSE_VERSIONS="${arg#*=}"
      ;;
    *)
      echo "Unknown parameter: $arg"
      exit 1
      ;;
  esac
done

if [ "$TERRAFORM_VERSIONS" == "" ]
then
  echo "--terraform-versions=<json serialization of terraform versions list> is required"
  exit 1
fi

if [ "$TOFU_VERSIONS" == "" ]
then
  echo "--tofu-versions=<json serialization of tofu versions list> is required"
  exit 1
fi

if [ "$CLICKHOUSE_VERSIONS" == "" ]
then
  echo "--clickhouse-versions=<json serialization of clickhouse versions list> is required"
  exit 1
fi

terraform=$(echo "${TERRAFORM_VERSIONS}" | jq -c 'map({binary: "terraform", image: "hashicorp/terraform", version: .})')

# Opentofu support is blocked by https://github.com/opentofu/opentofu/issues/1996
# tofu=$(echo "${TOFU_VERSIONS}" | jq -c 'map({binary: "tofu", image: "ghcr.io/opentofu/opentofu", version: .})')
tofu="[]"

both=$(jq -cn --argjson tf "$terraform" --argjson tofu "$tofu" '$tf + $tofu')

json=$(jq -c --null-input \
  --argjson terraform "${both}" \
  --argjson clickhouse_versions "${CLICKHOUSE_VERSIONS}" \
  --argjson protocols '["native", "http"]' \
  --argjson types '["single", "replicated", "localfile"]' \
  '{terraform: $terraform, clickhouse_version: $clickhouse_versions, protocol: $protocols, cluster_type: $types}')

echo "matrix=${json}" >> "$GITHUB_OUTPUT"
