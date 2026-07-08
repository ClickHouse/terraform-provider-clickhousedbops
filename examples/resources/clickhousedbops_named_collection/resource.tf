# Mark credential variables as sensitive so terraform redacts them from plan output.
variable "aws_secret_access_key" {
  type      = string
  sensitive = true
}

resource "clickhousedbops_named_collection" "s3_prod" {
  name = "s3_prod"

  keys = {
    url               = "https://s3.amazonaws.com/bucket/"
    format            = "CSV"
    access_key_id     = "AKIAEXAMPLE"
    secret_access_key = var.aws_secret_access_key
  }

  overridable_keys     = ["url"]
  not_overridable_keys = ["secret_access_key"]
}
