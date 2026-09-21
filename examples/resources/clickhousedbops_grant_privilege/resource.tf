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

# Granting ALL on a database directly fails on ClickHouse Cloud with error 497;
# current_grants copies it from the admin instead.
resource "clickhousedbops_grant_privilege" "full_db_access" {
  privilege_name    = "ALL"
  database_name     = "mydb"
  grantee_role_name = "my_role"
  current_grants    = true
}

# Access-management privileges (CREATE/ALTER/DROP USER and ROLE) are granted
# globally, so database_name and table_name are left null (granted on *.*).
resource "clickhousedbops_grant_privilege" "user_admin" {
  privilege_name    = "CREATE USER"
  grantee_role_name = "provisioning_admin"
  grant_option      = true
}

# USER_NAME/DEFINER-scoped privileges can be restricted to a specific object or a
# "prefix*" pattern via access_object, granting ON <object> instead of ON *.*.
resource "clickhousedbops_grant_privilege" "team_user_admin" {
  privilege_name    = "CREATE USER"
  access_object     = "team_*"
  grantee_role_name = "team_provisioner"
  grant_option      = true
}

# Parameterized privileges use access_object for targets that are not in the
# database/table hierarchy, including table engines and external sources.
resource "clickhousedbops_grant_privilege" "distributed_engine" {
  privilege_name    = "TABLE ENGINE"
  access_object     = "Distributed"
  grantee_role_name = "ddl_role"
}

resource "clickhousedbops_grant_privilege" "read_s3" {
  privilege_name    = "READ"
  access_object     = "S3"
  grantee_role_name = "reader_role"
}

# ClickHouse 25.8+ can restrict source grants to a URI regexp. The provider
# quotes the filter as a SQL string; this field does not accept raw SQL.
resource "clickhousedbops_grant_privilege" "read_public_exports" {
  privilege_name       = "READ"
  access_object        = "URL"
  access_object_filter = "https://example\\.com/public/.*"
  grantee_role_name    = "reader_role"
}
