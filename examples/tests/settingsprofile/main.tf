resource "clickhousedbops_settings_profile" "profile1" {
  cluster_name = var.cluster_name
  name = "profile1"
}

resource "clickhousedbops_settings_profile" "profile2" {
  cluster_name = var.cluster_name
  name = "profile2"

  inherit_from = ["default", clickhousedbops_settings_profile.profile1.name]
}

resource "clickhousedbops_setting" "setting1" {
  settings_profile_id = clickhousedbops_settings_profile.profile1.id
  name = "max_memory_usage"
  value = 1000
  min = "100"
  max = "2000"
  writability = "CHANGEABLE_IN_READONLY"
}

resource "clickhousedbops_setting" "setting2" {
  settings_profile_id = clickhousedbops_settings_profile.profile1.id
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

resource "clickhousedbops_settings_profile_association" "userassociation" {
  settings_profile_id = clickhousedbops_settings_profile.profile1.id
  user_id = clickhousedbops_user.john.id
}

resource "clickhousedbops_settings_profile_association" "roleassociation" {
  settings_profile_id = clickhousedbops_settings_profile.profile1.id
  role_id = clickhousedbops_role.tester.id
}
