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

resource "clickhousedbops_settingsprofile" "profile2" {
  cluster_name = var.cluster_name
  name = "profile2"
  inherit_profile = "default"

  settings = [
    {
      name = "max_memory_usage"
      value = "1000"
      min = "100"
      max = "2000"
      writability = "CHANGEABLE_IN_READONLY"
    },
  ]
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

resource "clickhousedbops_settingsprofileassociation" "tester" {
  settings_profile_name = clickhousedbops_settingsprofile.profile2.name
  # user_name = clickhousedbops_user.john.name
  role_name = clickhousedbops_role.tester.name
}
