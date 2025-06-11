#!/bin/bash

CLICKHOUSE_VERSION=""
TERRAFORM_IMAGE=""
TERRAFORM_VERSION=""
PROTOCOL=""
CLUSTER_NAME=""

for arg in "$@"; do
  case "$arg" in
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
    --cluster-name=*)
      CLUSTER_NAME="${arg#*=}"
      ;;
    *)
      echo "Unknown parameter: $arg"
      exit 1
      ;;
  esac
done

if [ "$CLICKHOUSE_VERSION" == "" ]
then
  echo "--clickhouse-version=<version> is required"
fi

if [ "$TERRAFORM_VERSION" == "" ]
then
  echo "--terraform-version=<version> is required"
fi

if [ "$TERRAFORM_IMAGE" == "" ]
then
  echo "--terraform-image=<image> is required"
fi

if [ "$PROTOCOL" == "" ]
then
  echo "--protocol=<http|native> is required"
fi

if [ "$CLUSTER_NAME" == "" ]
then
  echo "--cluster-name=<name> is required"
fi

#############################################################################################

cd tests/ || exit 1
export CLICKHOUSE_VERSION="$CLICKHOUSE_VERSION"
export TFVER="$TERRAFORM_VERSION"
export TFIMG="$TERRAFORM_IMAGE"
export TF_VAR_protocol="$PROTOCOL"

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

case "${CLUSTER_NAME}" in
  null)
  export TF_VAR_host=clickhouse
  ;;
  *)
  export TF_VAR_host=ch01
  export TF_VAR_cluster_name="${CLUSTER_NAME}"
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
docker compose exec ch01 clickhouse client --password "test" "select version()"
