# Migrating clickhouse_database to clickhousedbops_database

Given a resource in the old provider like the following:

```
resource "clickhouse_database" "logs" {
  service_id = clickhouse_service.service.id
  name       = "logs"
  comment    = "Database for logs"
}
```

First of all remove the resource from the state file:

```
terraform state rm clickhouse_database.logs
```

Then change the resource type to `clickhousedbops_database` and remove the `service_id` field.

```
resource "clickhousedbops_database" "logs" {
  name = "logs"
  comment = "Database for logs"
}
```

Finally import the resource into the state

```
terraform import clickhousedbops_database.logs logs
```

When you run `terraform apply` there should be no changes to be applied.
