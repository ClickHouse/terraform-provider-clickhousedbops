variable "aws_secret_access_key" {
  type      = string
  sensitive = true
}

resource "clickhousedbops_named_collection" "s3_prod" {
  name = "s3_prod"

  # Plain keys, stored in the terraform state.
  keys = {
    url           = "https://s3.amazonaws.com/bucket/"
    format        = "CSV"
    access_key_id = "AKIAEXAMPLE"
  }

  # Secrets, never written to the terraform state. Bump the version to re-apply them.
  secret_keys_wo = {
    secret_access_key = var.aws_secret_access_key
  }
  secret_keys_wo_version = 1

  overridable_keys     = ["url"]
  not_overridable_keys = ["secret_access_key"]
}
