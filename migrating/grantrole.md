# Migrating clickhouse_grant_role to clickhousedbops_grant_role

Given a resource in the old provider like the following:

```
resource "clickhouse_grant_role" "writer_to_john" {
  service_id        = clickhouse_service.service.id
  role_name         = clickhousedbops_role.writer.name
  grantee_user_name = clickhousedbops_user.john.name
  admin_option      = false
}
```

First of all remove the resource from the state file:

```
terraform state rm clickhouse_grant_role.writer_to_john
```

Then change the resource type to `clickhousedbops_grant_role` and remove the `service_id` field.

```
resource "clickhousedbops_grant_role" "writer_to_john" {
  role_name         = clickhousedbops_role.writer.name
  grantee_user_name = clickhousedbops_user.john.name
  admin_option      = false
}
```

Then just run `terraform apply` without doing any import. Terraform will show there is need to create a new Role Grant,
but nothing will change and the state will be updated correctly.
