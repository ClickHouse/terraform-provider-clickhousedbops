resource "clickhousedbops_settings_profile_association" "roleassociation" {
  settings_profile_id = clickhousedbops_settings_profile.profile1.id
  role_id = clickhousedbops_role.role1.id
}
