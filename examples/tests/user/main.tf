resource "clickhousedbops_user" "john" {
  cluster_name = var.cluster_name
  name         = "john"
  # You'll want to generate the password and feed it here instead of hardcoding.
  password_sha256_hash_wo         = sha256("test")
  password_sha256_hash_wo_version = 1
}

resource "clickhousedbops_user" "jane" {
  cluster_name = var.cluster_name
  name         = "jane"

  auth {
    sha256_hash {
      # You'll want to generate the password and feed it here instead of hardcoding.
      value_wo = sha256("test")
      value_wo_version = 4
    }
  }
}

resource "clickhousedbops_user" "anyone" {
  name = "anyone"

  auth {
    no_password {}
  }
}

resource "clickhousedbops_user" "multiple" {
  name = "multiple"

  auth {
    sha256_hash {
      value_wo         = sha256("changeme")
      value_wo_version = 1
    }

    sha256_hash {
      # You'll want to generate the password and feed it here instead of hardcoding.
      value_wo = sha256("test")
      value_wo_version = 4
    }
  }
}
