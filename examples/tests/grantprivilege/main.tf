resource "clickhousedbops_role" "reader" {
  name = "reader"
}

resource "clickhousedbops_user" "john" {
  name                           = "john"
  password_sha256_hash_wo         = sha256("test")
  password_sha256_hash_wo_version = 1
}

resource "clickhousedbops_grant_privilege" "grant_show_to_role" {
  privilege_name    = "SHOW"
  database_name     = "default"
  grantee_role_name = clickhousedbops_role.reader.name
  grant_option      = false
}

resource "clickhousedbops_grant_privilege" "grant_global_privilege" {
  privilege_name    = "REMOTE"
  grantee_role_name = clickhousedbops_role.reader.name
  grant_option      = false
}

resource "clickhousedbops_grant_privilege" "grant_dictget_to_role" {
  privilege_name    = "dictGet"
  database_name     = "default"
  grantee_role_name = clickhousedbops_role.reader.name
  grant_option      = false
}

resource "clickhousedbops_grant_privilege" "grant_insert_on_table_to_user" {
  privilege_name    = "INSERT"
  database_name     = "default"
  table_name        = "tbl1"
  grantee_user_name = clickhousedbops_user.john.name
  grant_option      = true
}

resource "clickhousedbops_grant_privilege" "grant_select_on_single_column_on_table_to_user" {
  privilege_name    = "SELECT"
  database_name     = "default"
  table_name        = "tbl1"
  column_name       = "count"
  grantee_user_name = clickhousedbops_user.john.name
  grant_option      = true
}
