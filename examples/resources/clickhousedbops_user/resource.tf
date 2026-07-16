# A user with several authentication methods combined.
resource "clickhousedbops_user" "jane" {
  name = "jane"

  auth {
    sha256_hash {
      value_wo         = sha256("changeme")
      value_wo_version = 1
    }

    ssl_certificate {
      common_name = "jane-service"
    }
  }
}

# A passwordless user. no_password cannot be combined with any other method.
resource "clickhousedbops_user" "readonly" {
  name = "readonly"

  auth {
    no_password {}
  }
}

# Legacy password field (deprecated), kept for backward compatibility.
resource "clickhousedbops_user" "john" {
  name                 = "john"
  password_sha256_hash = sha256("changeme")
}
