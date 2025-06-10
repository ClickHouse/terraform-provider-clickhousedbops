resource "clickhousedbops_grant_role" "role_to_user" {
  cluster_name      = "cluster"
  role_name         = "myrole"
  grantee_user_name = "myuser"
}
