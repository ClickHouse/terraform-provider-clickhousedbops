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
