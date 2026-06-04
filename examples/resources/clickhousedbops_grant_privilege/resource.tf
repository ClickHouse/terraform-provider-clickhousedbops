resource "clickhousedbops_grant_privilege" "grant" {
  privilege_name    = "SELECT"
  database_name     = "default"
  table_name        = "tbl1"
  column_name       = "count"
  grantee_user_name = "my_user_name"
  grant_option      = true
}

# On ClickHouse Cloud, broad grants the default admin holds but cannot transfer
# directly (e.g. SELECT on every database) must be copied with CURRENT GRANTS.
resource "clickhousedbops_grant_privilege" "read_everything" {
  privilege_name    = "SELECT"
  grantee_user_name = "my_monitoring_user"
  current_grants    = true
}
