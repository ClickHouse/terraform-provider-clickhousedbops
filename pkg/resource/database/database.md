Use the *clickhousedbops_database* resource to create a database in a ClickHouse instance.

Known limitations:

- Changing the comment on a `database` resource is unsupported and will cause the database to be destroyed and recreated. WARNING: you will lose any content of the database if you do so!
- Changing the `engine` name (e.g. `Atomic` to `Replicated`) destroys and recreates the database. WARNING: you will lose any content of the database if you do so!
- Only the engine name is compared, because ClickHouse normalizes the engine arguments it reports. Changing only the arguments (or their formatting) is an in-place update of Terraform state that leaves the database untouched. Use `terraform apply -replace` to recreate the database with new arguments.
- After importing a database, `engine` is set to `engine_full` from `system.databases`. A configured `engine` with the same name shows as an in-place update on the next plan, which only updates state.

