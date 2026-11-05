You can use the `clickhousedbops_database` resource to create a database in a `ClickHouse` instance.

## Engines

`engine` sets the [database engine](https://clickhouse.com/docs/engines/database-engines). When omitted, ClickHouse uses its default engine. `engine_arguments` and `engine_settings` are passed to the engine as written, so the resource supports every engine without knowing their signatures: arguments are SQL expressions and setting values are literals, so quote string values.

## Secrets

Engine arguments and settings often hold credentials. Reference them as query parameters, such as `{secret_access_key:String}`, and set their values in `engine_parameters_wo`:

```terraform
resource "clickhousedbops_database" "bucket" {
  name   = "bucket"
  engine = "S3"
  engine_arguments = [
    "'https://bucket.s3.amazonaws.com/data/'",
    "{access_key_id:String}",
    "{secret_access_key:String}",
  ]

  engine_parameters_wo = {
    access_key_id     = var.access_key_id
    secret_access_key = var.secret_access_key
  }
  engine_parameters_wo_version = 1
}
```

- `engine_parameters_wo` is write-only: ClickHouse binds the values server-side, so they are not part of the SQL the provider sends or logs, and they are never stored in the terraform state. It needs Terraform/OpenTofu >= 1.11.
- Engine arguments accept query parameters on every supported ClickHouse version, engine settings only on ClickHouse 26.7 or newer.
- ClickHouse itself may still record the values, for example in the distributed DDL queue of an `ON CLUSTER` query.
- ClickHouse never returns these values, so the provider cannot tell when one changes: bump `engine_parameters_wo_version` to re-apply them. This recreates the database.

Known limitations:

- Changing the comment on a `database` resource is unsupported and will cause the database to be destroyed and recreated. WARNING: you will lose any content of the database if you do so!
- Changing `name`, `engine`, `engine_arguments`, `engine_settings` or `engine_parameters_wo_version` destroys and recreates the database as well.
- The provider compares only the engine name with the database, so it detects drift of `engine`, but not of `engine_arguments` or `engine_settings`.
- ClickHouse reports `HDFS` databases as `S3`, so the provider can't manage databases with `engine = "HDFS"`: their engine never matches the configuration.
- After an import, or with state from a provider version without engine support, the state has no `engine_arguments`, `engine_settings` or `engine_parameters_wo_version`. The configured values are recorded with a warning, without checking them against the database or recreating it: make sure they match the database.
