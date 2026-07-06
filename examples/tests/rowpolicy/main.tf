resource "clickhousedbops_role" "reader" {
  cluster_name = var.cluster_name
  name         = "reader"
}

resource "clickhousedbops_user" "john" {
  cluster_name                    = var.cluster_name
  name                            = "john"
  password_sha256_hash_wo         = sha256("test")
  password_sha256_hash_wo_version = 1
}

resource "clickhousedbops_row_policy" "permissive_to_role" {
  cluster_name  = var.cluster_name
  name          = "reader_rows"
  database_name = "default"
  table_name    = "tbl1"
  select_filter = "1 = 1"
  grantee_names = [clickhousedbops_role.reader.name]
}

resource "clickhousedbops_row_policy" "restrictive_to_user" {
  cluster_name   = var.cluster_name
  name           = "john_rows"
  database_name  = "default"
  table_name     = "tbl1"
  select_filter  = "owner_id = 'john'"
  is_restrictive = true
  grantee_names  = [clickhousedbops_user.john.name]
}

resource "clickhousedbops_row_policy" "permissive_to_all_except" {
  cluster_name       = var.cluster_name
  name               = "all_except_rows"
  database_name      = "default"
  table_name         = "tbl1"
  select_filter      = "1 = 1"
  grantee_all_except = [clickhousedbops_user.john.name]
}
