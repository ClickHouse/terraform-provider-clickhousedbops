# Example using password_sha256_hash_wo field 
resource "clickhousedbops_user" "jane" {
  cluster_name = "cluster"
  name         = "jane"
  # You'll want to generate the password and feed it here instead of hardcoding.
  password_sha256_hash_wo         = sha256("test")
  password_sha256_hash_wo_version = 4
}

# Example using the new password_sha256_hash field (recommended only for OpenTofu (version < 1.11) compatibility)
resource "clickhousedbops_user" "john" {
  cluster_name = "cluster"
  name         = "john"
  # You'll want to generate the password and feed it here instead of hardcoding.
  password_sha256_hash = sha256("test")
}

# Example using ssl_certificate authentication with the write-only auth_value_wo field
resource "clickhousedbops_user" "cert_user" {
  name                  = "cert_user"
  auth_type             = "ssl_certificate"
  auth_value_wo         = "cert-common-name"
  auth_value_wo_version = 1
}

# Example using ssl_certificate authentication with auth_value (recommended only for Terraform/OpenTofu < 1.11 compatibility)
resource "clickhousedbops_user" "legacy_cert_user" {
  name       = "legacy_cert_user"
  auth_type  = "ssl_certificate"
  auth_value = "cert-common-name"
}

# Example using no_password authentication
resource "clickhousedbops_user" "readonly" {
  name      = "readonly"
  auth_type = "no_password"
}

