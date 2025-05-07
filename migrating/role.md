# Migrating clickhouse_role to clickhousedbops_role

Given a resource in the old provider like the following:

```
resource "clickhouse_role" "writer" {
  service_id = clickhouse_service.service.id
  name       = "writer"
}
```

First of all remove the resource from the state file:

```
terraform state rm clickhouse_role.writer
```

Then change the resource type to `clickhousedbops_role` and remove the `service_id` field.

```
resource "clickhousedbops_role" "writer" {
  name = "writer"
}
```

Finally import the resource into the state

```
terraform import clickhousedbops_role.writer writer
```

When you run `terraform apply` there should be no changes to be applied.
