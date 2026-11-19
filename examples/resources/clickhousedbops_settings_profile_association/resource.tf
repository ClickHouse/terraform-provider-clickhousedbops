resource "clickhousedbops_settings_profile_association" "roleassociation" {
  settings_profile_id = clickhousedbops_settings_profile.profile1.id
  role_id             = clickhousedbops_role.role1.id
}

# A user that ALTER USER cannot change, such as a JWT ephemeral user.
resource "clickhousedbops_settings_profile_association" "jwtuserassociation" {
  settings_profile_id = clickhousedbops_settings_profile.profile1.id
  user_name           = "JWT::jane.doe::<claims hash>"
}
