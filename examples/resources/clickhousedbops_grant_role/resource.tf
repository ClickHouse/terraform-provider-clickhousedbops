resource "clickhousedbops_grant_role" "role_to_user" {
  role_name         = "myrole"
  grantee_user_name = "myuser"
}
