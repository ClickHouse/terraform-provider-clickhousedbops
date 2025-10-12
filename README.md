# Clickhouse DB ops Terraform Provider

[![Docs](https://github.com/ClickHouse/terraform-provider-clickhousedbops/actions/workflows/docs.yaml/badge.svg)](https://github.com/ClickHouse/terraform-provider-clickhousedbops/actions/workflows/docs.yaml)
[![Dependabot Updates](https://github.com/ClickHouse/terraform-provider-clickhousedbops/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/ClickHouse/terraform-provider-clickhousedbops/actions/workflows/dependabot/dependabot-updates)
[![Unit tests](https://github.com/ClickHouse/terraform-provider-clickhousedbops/actions/workflows/test.yaml/badge.svg)](https://github.com/ClickHouse/terraform-provider-clickhousedbops/actions/workflows/test.yaml)

This is the official Terraform provider for ClickHouse database operations.

With this Terraform provider you can:

- Manage `databases` in a `ClickHouse` instance using the `clickhousedbops_database` resource
- Manage `users` in a `ClickHouse` instance using the `clickhousedbops_user` resource
- Manage `roles` in a `ClickHouse` instance using the `clickhousedbops_role` resource
- Manage `role grants` in a `ClickHouse` instance using the `clickhousedbops_grant_role` resource
- Manage `privilege grants` in a `ClickHouse` instance using the `clickhousedbops_grant_privilege` resource

## Getting started

The `clickhousedbops_user` resource works with both Terraform and OpenTofu. When using the `password_sha256_hash` field, it works with all versions. The `password_sha256_hash_wo` field requires at least Terraform 1.11 (write-only arguments support). All other resources work with older versions too.

You can find examples in the [examples/tests](https://github.com/ClickHouse/terraform-provider-clickhousedbops/tree/main/examples/tests) directory.

Please refer to the [official docs](https://registry.terraform.io/providers/ClickHouse/clickhousedbops/latest/docs) for more details.

## Migrating from terraform-provider-clickhouse

Please read the [Migration guide](https://github.com/ClickHouse/terraform-provider-clickhousedbops/blob/main/migrating/README.md)

## Development and contributing

Please read the [Development readme](https://github.com/ClickHouse/terraform-provider-clickhousedbops/blob/main/development/README.md)

