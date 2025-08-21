resource "clickhousedbops_setting" "setting1" {
  settings_profile_id = clickhousedbops_settings_profile.profile1.id
  name = "max_memory_usage"
  value = 1000
  min = "100"
  max = "2000"
  writability = "CHANGEABLE_IN_READONLY"
}
