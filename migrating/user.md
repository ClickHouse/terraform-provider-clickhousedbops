# Migrating clickhouse_user to clickhousedbops_user

Given a resource in the old provider like the following:

```
resource "clickhouse_user" "john" {
  service_id           = clickhouse_service.service.id
  name                 = "john"
  password_sha256_hash = sha256("test")
}
```

First of all remove the resource from the state file:

```
terraform state rm clickhouse_user.john
```

Then make the following changes:
- change the resource type to `clickhousedbops_user`
- remove the `service_id` field.
- rename the `password_sha256_hash` field to `password_sha256_hash_wo`
- add a new field `password_sha256_hash_wo_version` with a value of 1.

```
resource "clickhousedbops_user" "john" {
  name = "john"
  password_sha256_hash_wo = sha256("test")
  password_sha256_hash_wo_version = 1
}
```

Note: `password_sha256_hash_wo` is a write only argument so it can use ephemeral values. See [terraform docs](https://developer.hashicorp.com/terraform/language/resources/ephemeral/write-only) for more details. 

Finally import the resource into the state:

```
terraform import clickhousedbops_user.john john
```

Please note that for security reasons ClickHouse does not expose any information about a User's password, so for terraform it's impossible to import such information during `terraform import`.
The consequence is that the first run of `terraform apply` after importing a `clickhousedbops_user` will recreate the user to ensure the password is aligned with the .tf file.
Normally this operation only takes a split second, but you might need to plan some downtime if you have a very highly used ClickHouse service.
