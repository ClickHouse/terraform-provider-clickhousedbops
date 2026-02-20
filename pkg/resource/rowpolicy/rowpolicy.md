Manages a ClickHouse row policy, which restricts the rows a user or role can access when querying a table.

~> **Important:** When a row policy is created on a table, **all** users and roles without a matching policy will see zero rows from that table. You must create an additional permissive policy (e.g. `USING 1` to `ALL EXCEPT ...`) for any users or roles that should retain full access.

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
  name              = "user_a_policy"
  database_name     = "logs"
  table_name        = "example_table"
  select_filter     = "user_id = 'a'"
  grantee_user_name = clickhousedbops_user.user_a.name
}

# Permissive policy for admin users to retain full access
resource "clickhousedbops_row_policy" "admin_full_access" {
  name              = "admin_full_access"
  database_name     = "logs"
  table_name        = "example_table"
  select_filter     = "1"
  grantee_role_name = "admin"
}
```