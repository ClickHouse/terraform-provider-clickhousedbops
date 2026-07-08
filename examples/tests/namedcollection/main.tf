resource "clickhousedbops_named_collection" "s3_prod" {
  cluster_name = var.cluster_name
  name         = "s3_prod"

  keys = {
    url               = "https://s3.amazonaws.com/bucket/"
    format            = "CSV"
    access_key_id     = "AKIAEXAMPLE"
    secret_access_key = "topsecret"
  }

  overridable_keys     = ["url"]
  not_overridable_keys = ["secret_access_key"]
}

resource "clickhousedbops_named_collection" "mysql" {
  cluster_name = var.cluster_name
  name         = "mysql_conn"

  keys = {
    host     = "127.0.0.1"
    port     = "3306"
    database = "test"
    password = "secret"
  }
}
