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
