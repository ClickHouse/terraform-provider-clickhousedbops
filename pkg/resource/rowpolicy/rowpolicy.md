Manages a ClickHouse row policy, which restricts the rows a user or role can access when querying a table.

~> **Important:** When a row policy is created on a table, **all** users and roles without a matching policy will see zero rows from that table. You must create an additional permissive policy (e.g. `USING 1` to `ALL EXCEPT ...`) for any users or roles that should retain full access.

Resource can be imported by `id` or `<database>.<table>.<short_name>` triple.

## Grantees

A policy applies either to a specific set of grantees or to everyone. Set exactly one of:

- `grantee_names`: a list of user and role names. ClickHouse stores these as one untyped list and resolves each name to a user before a role, so users and roles are not distinguished here.
- `grantee_all_except`: apply to all users and roles, excluding the ones listed. An empty set (`[]`) applies to everyone with no exclusions.

## Example Usage

```hcl
resource "clickhousedbops_user" "user_a" {
  name                 = "user-a"
  password_sha256_hash = var.user_a_password_hash
}

resource "clickhousedbops_grant_privilege" "user_a_select" {
  privilege_name    = "SELECT"
  database_name     = "logs"
  table_name        = "example_table"
  grantee_user_name = clickhousedbops_user.user_a.name
}

resource "clickhousedbops_row_policy" "user_a_policy" {
  name          = "user_a_policy"
  database_name = "logs"
  table_name    = "example_table"
  select_filter = "user_id = 'a'"
  grantee_names = [clickhousedbops_user.user_a.name]
}

# Permissive policy for admin users to retain full access
resource "clickhousedbops_row_policy" "admin_full_access" {
  name          = "admin_full_access"
  database_name = "logs"
  table_name    = "example_table"
  select_filter = "1"
  grantee_names = ["admin"]
}

# Apply to everyone except user_a (an empty grantee_all_except set applies to everyone)
resource "clickhousedbops_row_policy" "all_except_user_a" {
  name               = "all_except_user_a"
  database_name      = "logs"
  table_name         = "example_table"
  select_filter      = "1"
  grantee_all_except = [clickhousedbops_user.user_a.name]
}
```
