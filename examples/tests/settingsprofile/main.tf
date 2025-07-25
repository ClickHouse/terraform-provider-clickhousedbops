resource "clickhousedbops_settingsprofile" "profile1" {
  cluster_name = var.cluster_name
  name = "profile1"
  inherit_profile = "default"

  settings = [
    {
      name = "max_memory_usage"
      value = "1000"
      min = "100"
      max = "2000"
      writability = "CHANGEABLE_IN_READONLY"
    },
    {
      name = "network_compression_method"
      value = "LZ4"
    },
  ]
}

resource "clickhousedbops_role" "tester" {
  cluster_name = var.cluster_name
  name = "tester"
  
  settings_profile = clickhousedbops_settingsprofile.profile1.name
}

resource "clickhousedbops_user" "john" {
  cluster_name = var.cluster_name
  name = "john"
  password_sha256_hash_wo = sha256("test")
  password_sha256_hash_wo_version = 1

  settings_profile = clickhousedbops_settingsprofile.profile1.name
}
