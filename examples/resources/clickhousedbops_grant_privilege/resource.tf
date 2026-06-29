resource "clickhousedbops_grant_privilege" "grant" {
  privilege_name    = "SELECT"
  database_name     = "default"
  table_name        = "tbl1"
  column_name       = "count"
  grantee_user_name = "my_user_name"
  grant_option      = true
}

# Access-management privileges (CREATE/ALTER/DROP USER and ROLE) are granted
# globally, so database_name and table_name are left null (granted on *.*).
resource "clickhousedbops_grant_privilege" "user_admin" {
  privilege_name    = "CREATE USER"
  grantee_role_name = "provisioning_admin"
  grant_option      = true
}
