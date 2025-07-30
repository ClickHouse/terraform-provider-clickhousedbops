resource "clickhousedbops_settingsprofile" "profile1" {
  cluster_name = var.cluster_name
  name = "profile1"
}

resource "clickhousedbops_settingsprofilesetting" "setting1" {
  settings_profile_id = clickhousedbops_settingsprofile.profile1.id
  name = "max_memory_usage"
  value = 1000
  min = "100"
  max = "2000"
  writability = "CHANGEABLE_IN_READONLY"
}

resource "clickhousedbops_settingsprofilesetting" "setting2" {
  settings_profile_id = clickhousedbops_settingsprofile.profile1.id
  name = "network_compression_method"
  value = "LZ4"
}

resource "clickhousedbops_role" "tester" {
  cluster_name = var.cluster_name
  name = "tester"
}

resource "clickhousedbops_user" "john" {
  cluster_name = var.cluster_name
  name = "john"
  password_sha256_hash_wo = sha256("test")
  password_sha256_hash_wo_version = 1
}

resource "clickhousedbops_settingsprofileassociation" "userassociation" {
  settings_profile_id = clickhousedbops_settingsprofile.profile1.id
  user_id = clickhousedbops_user.john.id
}

resource "clickhousedbops_settingsprofileassociation" "roleassociation" {
  settings_profile_id = clickhousedbops_settingsprofile.profile1.id
  role_id = clickhousedbops_role.tester.id
}
