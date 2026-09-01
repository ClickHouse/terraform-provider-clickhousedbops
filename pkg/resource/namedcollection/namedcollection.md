You can use the `clickhousedbops_named_collection` resource to create a [Named Collection](https://clickhouse.com/docs/operations/named-collections) in a `ClickHouse` instance.

Named collections are not available on ClickHouse Cloud. Managing them with DDL needs the `named_collection_control` privilege, which is not granted with `GRANT`: it is enabled per user in the server config, usually a file in `/etc/clickhouse-server/users.d/`.

## Secrets

Named collections usually hold credentials, so there are two ways to write a key:

- `secret_keys_wo` is write-only: the value goes to ClickHouse and is never stored in the terraform state. It needs Terraform/OpenTofu >= 1.11. ClickHouse never returns these values, so the provider cannot tell when one changes: bump `secret_keys_wo_version` to re-apply all of them.
- `keys` is stored in the state. Values coming from variables marked `sensitive = true` are redacted from CLI output, but they are in the state file in clear text, like every terraform secret.

Both maps write into the same set of keys in ClickHouse, so a key name can only appear in one of them.

## Known limitations

- ClickHouse hides named collection values in system tables unless the current user is granted `SHOW NAMED COLLECTIONS SECRETS`. Without that grant the provider only detects drift on the set of key names, not on the values of `keys`.
- The `OVERRIDABLE`/`NOT OVERRIDABLE` flags are never returned by ClickHouse, so changing them outside terraform is not detected.
- Renaming a collection is not supported by ClickHouse, so changing `name` (or `cluster_name`) destroys and recreates the collection.
- When importing an existing collection, values are imported as the literal `[HIDDEN]` placeholder unless the user can see secrets. Write the real values in your terraform config and run one apply to converge.
