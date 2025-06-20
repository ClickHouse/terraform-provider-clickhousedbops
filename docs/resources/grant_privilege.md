---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "clickhousedbops_grant_privilege Resource - clickhousedbops"
subcategory: ""
description: |-
  You can use the clickhousedbops_grant_privilege resource to grant privileges on databases and tables to either a clickhousedbops_user or a clickhousedbops_role.
  Please note that in order to grant privileges to all database and/or all tables, the database and/or table fields must be set to null, and not to "*".
  Known limitations:
  Only a subset of privileges can be granted on ClickHouse cloud. For example the ALL privilege can't be granted. See https://clickhouse.com/docs/en/sql-reference/statements/grant#allIt's not possible to grant privileges using their alias name. The canonical name must be used.It's not possible to grant group of privileges. Please grant each member of the group individually instead.It's not possible to grant the same clickhousedbops_grant_privilege to both a clickhousedbops_user and a clickhousedbops_role using a single clickhousedbops_grant_privilege stanza. You can do that using two different stanzas, one with grantee_user_name and the other with grantee_role_name fields set.It's not possible to grant the same privilege (example 'SELECT') to multiple entities (for example tables) with a single stanza. You can do that my creating one stanza for each entity you want to grant privileges on.Importing clickhousedbops_grant_privilege resources into terraform is not supported.
---

# clickhousedbops_grant_privilege (Resource)

You can use the `clickhousedbops_grant_privilege` resource to grant privileges on databases and tables to either a `clickhousedbops_user` or a `clickhousedbops_role`.

Please note that in order to grant privileges to all database and/or all tables, the `database` and/or `table` fields must be set to null, and not to "*".

Known limitations:

- Only a subset of privileges can be granted on ClickHouse cloud. For example the `ALL` privilege can't be granted. See https://clickhouse.com/docs/en/sql-reference/statements/grant#all
- It's not possible to grant privileges using their alias name. The canonical name must be used.
- It's not possible to grant group of privileges. Please grant each member of the group individually instead.
- It's not possible to grant the same `clickhousedbops_grant_privilege` to both a `clickhousedbops_user` and a `clickhousedbops_role` using a single `clickhousedbops_grant_privilege` stanza. You can do that using two different stanzas, one with `grantee_user_name` and the other with `grantee_role_name` fields set.
- It's not possible to grant the same privilege (example 'SELECT') to multiple entities (for example tables) with a single stanza. You can do that my creating one stanza for each entity you want to grant privileges on.
- Importing `clickhousedbops_grant_privilege` resources into terraform is not supported.

## Example Usage

```terraform
resource "clickhousedbops_grant_privilege" "grant" {
  privilege_name    = "SELECT"
  database_name     = "default"
  table_name        = "tbl1"
  column_name       = "count"
  grantee_user_name = "my_user_name"
  grant_option      = true
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `privilege_name` (String) The privilege to grant, such as `CREATE DATABASE`, `SELECT`, etc. See https://clickhouse.com/docs/en/sql-reference/statements/grant#privileges.

### Optional

- `cluster_name` (String) Name of the cluster to create the resource into. If omitted, resource will be created on the replica hit by the query.
This field must be left null when using a ClickHouse Cloud cluster.
When using a self hosted ClickHouse instance, this field should only be set when there is more than one replica and you are not using 'replicated' storage for user_directory.
- `column_name` (String) The name of the column in `table_name` to grant privilege on.
- `database_name` (String) The name of the database to grant privilege on. Defaults to all databases if left null
- `grant_option` (Boolean) If true, the grantee will be able to grant the same privileges to others.
- `grantee_role_name` (String) Name of the `role` to grant privileges to.
- `grantee_user_name` (String) Name of the `user` to grant privileges to.
- `table_name` (String) The name of the table to grant privilege on.
