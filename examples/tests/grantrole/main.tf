resource "clickhousedbops_role" "reader" {
  name = "reader"
}

resource "clickhousedbops_role" "writer" {
  name = "writer"
}

resource "clickhousedbops_grant_role" "role_to_role" {
  role_name         = clickhousedbops_role.reader.name
  grantee_role_name = clickhousedbops_role.writer.name
  admin_option      = true
}

resource "clickhousedbops_user" "user" {
  name                           = "user"
  password_sha256_hash_wo         = sha256("test")
  password_sha256_hash_wo_version = 1
}

resource "clickhousedbops_grant_role" "role_to_user" {
 role_name         = clickhousedbops_role.reader.name
 grantee_user_name = clickhousedbops_user.user.name
}
