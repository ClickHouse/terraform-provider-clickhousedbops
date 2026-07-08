You can use the `clickhousedbops_named_collection` resource to create a [Named Collection](https://clickhouse.com/docs/operations/named-collections) in a `ClickHouse` instance.

Named collections are not available on ClickHouse Cloud. The ClickHouse user needs the `named_collection_control` privilege to manage them.

Named collections typically hold credentials. Put those values in variables marked `sensitive = true` so terraform redacts them from plan output. Values are stored in the terraform state in clear text either way, like every terraform secret.

Known limitations:

- ClickHouse hides named collection values in system tables unless the current user is granted `SHOW NAMED COLLECTIONS SECRETS`. This means the provider can only detect drift on the set of key names, not on the values or on the `OVERRIDABLE`/`NOT OVERRIDABLE` flags.
- Renaming a collection is not supported by ClickHouse, so changing `name` (or `cluster_name`) destroys and recreates the collection.
- When importing an existing collection, values are imported as the literal `[HIDDEN]` placeholder unless the user can see secrets. Write the real values in your terraform config and run one apply to converge.
