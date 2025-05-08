# Migrating from terraform-provider-clickhouse

When migrating resources from the `terraform-provider-clickhouse` to the new `terraform-provider-clickhousedbops` provider, you need to declare a new provider in your terraform files:

```
terraform {
  required_providers {
    ...
    clickhousedbops = {
      # version = "<uncomment and set desired version or leave field commented to use latest available>"
      source  = "ClickHouse/clickhousedbops"
    }
  }
}

provider "clickhousedbops" {
...
}
```

For example, if you are connecting to a Clickhouse Cloud service that is defined in the same terraform file, you can set it up like this:

```
provider "clickhousedbops" {
  protocol = "nativesecure"

  host = clickhouse_service.service.endpoints.nativesecure.host
  port = clickhouse_service.service.endpoints.nativesecure.port

  auth_config = {
    strategy = "password"
    username = "default"
    password = <your service's password here>
  }
}
```

## Migrating resources

To migrate from a `clickhouse_database` to a `clickhousedbops_database` please read [Migrating database](https://github.com/ClickHouse/terraform-provider-clickhousedbops/blob/main/migrating/database.md).  
To migrate from a `clickhouse_role` to a `clickhousedbops_role` please read [Migrating roles](https://github.com/ClickHouse/terraform-provider-clickhousedbops/blob/main/migrating/role.md).  
To migrate from a `clickhouse_user` to a `clickhousedbops_user` please read [Migrating roles](https://github.com/ClickHouse/terraform-provider-clickhousedbops/blob/main/migrating/user.md).  
To migrate from a `clickhouse_grant_role` to a `clickhousedbops_grant_role` please read [Migrating roles](https://github.com/ClickHouse/terraform-provider-clickhousedbops/blob/main/migrating/grantrole.md).  
To migrate from a `clickhouse_grant_privilege` to a `clickhousedbops_grant_privilege` please read [Migrating roles](https://github.com/ClickHouse/terraform-provider-clickhousedbops/blob/main/migrating/grantprivilege.md).  
