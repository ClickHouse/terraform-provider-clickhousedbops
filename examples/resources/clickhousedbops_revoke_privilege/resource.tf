resource "clickhousedbops_grant_privilege" "table_reader" {
  privilege_name    = "SELECT"
  database_name     = "analytics"
  table_name        = "events"
  grantee_role_name = "analyst"
}

resource "clickhousedbops_revoke_privilege" "hide_payload" {
  privilege_name    = "SELECT"
  database_name     = "analytics"
  table_name        = "events"
  column_name       = "secret_payload"
  grantee_role_name = "analyst"

  depends_on = [clickhousedbops_grant_privilege.table_reader]

  lifecycle {
    replace_triggered_by = [clickhousedbops_grant_privilege.table_reader]
  }
}
