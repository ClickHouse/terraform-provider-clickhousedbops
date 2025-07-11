#!/bin/bash

set -euo pipefail

EXAMPLE=""
CLICKHOUSE_VERSION=""
TERRAFORM_IMAGE=""
TERRAFORM_VERSION=""
PROTOCOL=""
CLUSTER_TYPE=""

for arg in "$@"; do
  case "$arg" in
    --example=*)
      EXAMPLE="${arg#*=}"
      ;;
    --clickhouse-version=*)
      CLICKHOUSE_VERSION="${arg#*=}"
      ;;
    --terraform-version=*)
      TERRAFORM_VERSION="${arg#*=}"
      ;;
    --terraform-image=*)
      TERRAFORM_IMAGE="${arg#*=}"
      ;;
    --protocol=*)
      PROTOCOL="${arg#*=}"
      ;;
    --cluster-type=*)
      CLUSTER_TYPE="${arg#*=}"
      ;;
    *)
      echo "Unknown parameter: $arg"
      exit 1
      ;;
  esac
done

if [ "$EXAMPLE" == "" ]
then
  echo "--example=<example name> is required"
  exit 1
fi

if [ "$CLICKHOUSE_VERSION" == "" ]
then
  echo "--clickhouse-version=<version> is required"
  exit 1
fi

if [ "$TERRAFORM_VERSION" == "" ]
then
  echo "--terraform-version=<version> is required"
  exit 1
fi

if [ "$TERRAFORM_IMAGE" == "" ]
then
  echo "--terraform-image=<image> is required"
  exit 1
fi

if [ "$PROTOCOL" == "" ]
then
  echo "--protocol=<http|native> is required"
  exit 1
fi

if [ "$CLUSTER_TYPE" == "" ]
then
  echo "--cluster-type=<type> is required"
  exit 1
fi

#############################################################################################

cd tests/ || exit 1
export CLICKHOUSE_VERSION="$CLICKHOUSE_VERSION"
export TFVER="$TERRAFORM_VERSION"
export TFIMG="$TERRAFORM_IMAGE"
export TF_VAR_protocol="$PROTOCOL"
export TF_VAR_host="tests-clickhouse-1"
export CONFIGFILE="config-${CLUSTER_TYPE}.xml"

case "$TF_VAR_protocol" in
  native)
    export TF_VAR_port=9000
    export TF_VAR_auth_strategy=password
    ;;
  http)
    export TF_VAR_port=8123
    export TF_VAR_auth_strategy=basicauth
    ;;
esac

case "${CLUSTER_TYPE}" in
  single)
  export REPLICAS=1
  ;;
  replicated)
  export REPLICAS=2
  if [ "${EXAMPLE}" == "database" ]
  then
    # Even in replicated setups, the database resource need the CLUSTER_TYPE to be set.
    export TF_VAR_cluster_name="cluster1"
  fi
  ;;
  localfile)
  export REPLICAS=2
  export TF_VAR_cluster_name="cluster1"
  ;;
  *)
  echo "Invalid cluster type ${CLUSTER_TYPE}"
  exit 1
esac

# This is needed until docker compose 1.36 to avoid concurrent map write error.
docker pull "clickhouse/clickhouse-server:${CLICKHOUSE_VERSION}"

docker compose up -d
sleep 5

# Check containers are running or display logs
for svc in clickhouse shell ; do
  if [ -z "$(docker compose ps -q $svc)" ] || ! docker ps -q --no-trunc | grep -q "$(docker compose ps -q $svc)"; then
    echo "Failed running $svc"
    docker compose logs $svc
    exit 1
  fi
done

docker compose exec clickhouse clickhouse client --password "test" "select version()"
