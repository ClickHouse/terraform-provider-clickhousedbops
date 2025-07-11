#!/bin/bash

set -euo pipefail

MIN=""
WANT=""
REPO=""

for arg in "$@"; do
  case "$arg" in
    --min=*)
      MIN="${arg#*=}"
      ;;
    --want=*)
      WANT="${arg#*=}"
      ;;
    --repo=*)
      REPO="${arg#*=}"
      ;;
    *)
      echo "Unknown parameter: $arg"
      exit 1
      ;;
  esac
done

if [ "$REPO" == "" ]
then
  echo "--repo=<repository name> is required"
  exit 1
fi

if [ "$WANT" == "" ]
then
  echo "--want=<count> is required"
  exit 1
fi

all="$(curl -s -L -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" "https://api.github.com/repos/${REPO}/releases?per_page=100"| jq -r '.[]|.name'|sort -V -r)"

min="${MIN}"
min="${min#v}"
want=${WANT}

# gets 2 arguments, representing a semver string such as 1.2.3
# exits with 1 if first argument is >= second argument
vergte() {
  printf '%s\n' "$1" "$2" | sort -C -V -r;
}

versions=()
current_nopatch=""
for candidate in $all
do
  # only keep final releases such as x.y.z
  if [[ $candidate =~ ^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
    candidate=${candidate#v}

    if [ "${min}" != "" ]
    then
      # Breaks the for loop at first occourence of a candidate older than the min.
      vergte "$candidate" "$min" || break
    fi

    # Keep major.minor for each release to check if it changes since last iteration
    nopatch="$(echo "$candidate" | cut -d "." -f1).$(echo "$candidate" | cut -d "." -f2)"

    if [ "$nopatch" != "${current_nopatch}" ]; then
      # First time we see this major.minor, this is a good candidate
      versions+=("${candidate}")
      current_nopatch=${nopatch}
    fi
  fi
  [ ${#versions[@]} -ge "$want" ] && break
done

json="$(printf '%s\n' "${versions[@]}" | jq -R . | jq -cs .)"
echo "versions=${json}" >> "$GITHUB_OUTPUT"
