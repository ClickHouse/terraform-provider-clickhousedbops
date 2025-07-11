#!/bin/bash

set -euo pipefail

versions=()

# Get list of most recent clickhouse OS version.
RELEASES="$(curl -s -L -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" https://api.github.com/repos/ClickHouse/clickhouse/releases?per_page=200|jq -r '.[] |.tag_name')"

# Get major.minor of latest LTS and latest Stable releases.
LATEST_LTS="$(echo "$RELEASES" | grep lts | sort -Vr |head -n1 | cut -d "." -f 1,2 | sed 's/v//')"
LATEST_STABLE="$(echo "$RELEASES" | grep stable | sort -Vr |head -n1 | cut -d "." -f 1,2 | sed 's/v//')"

versions+=("${LATEST_LTS}")
versions+=("${LATEST_STABLE}")

# Get latest minor of previous major
latest_major="$(echo "$LATEST_LTS" | cut -d"." -f1)"
latest_major="${latest_major#v}"

for candidate in $RELEASES
do
  candidate=${candidate#v}
  candidate_major="$(echo "$candidate" | cut -d "." -f1)"
  candidate_minor="$(echo "$candidate" | cut -d "." -f2)"

  if [ "$candidate_major" != "${latest_major}" ]; then
    versions+=("${candidate_major}.${candidate_minor}")
    break
  fi
done

# Ensure no duplicate versions and turn into JSON.
json="$(printf '%s\n' "${versions[@]}" | jq -R . | uniq |jq -cs .)"
echo "versions=${json}" >> "$GITHUB_OUTPUT"
