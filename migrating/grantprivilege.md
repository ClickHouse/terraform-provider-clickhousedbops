# Migrating clickhouse_grant_privilege to clickhousedbops_grant_privilege

Given a resource in the old provider like the following:

```
resource "clickhouse_grant_privilege" "grant_show_to_role" {
  service_id        = clickhouse_service.service.id
  privilege_name    = "SHOW"
  database_name     = "default"
  grantee_role_name = clickhousedbops_user.john.name
  grant_option      = false
}
```

First of all remove the resource from the state file:

```
terraform state rm clickhouse_grant_privilege.grant_show_to_role
```

Then change the resource type to `clickhousedbops_grant_privilege` and remove the `service_id` field.

```
resource "clickhousedbops_grant_privilege" "grant_show_to_role" {
  privilege_name    = "SHOW"
  database_name     = "default"
  grantee_role_name = clickhousedbops_user.john.name
  grant_option      = false
}
```

Then just run `terraform apply` without doing any import. Terraform will show there is need to create a new Privilege Grant,
but nothing will change and the state will be updated correctly.
