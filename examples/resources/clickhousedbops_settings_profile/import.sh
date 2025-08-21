# Settings profiles can be imported by specifying the UUID.
# Find the ID of the settings profile by checking system.settings_profiles table.
terraform import clickhousedbops_settings_profile.example xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# It's also possible to import settings profiles by name:

terraform import clickhousedbops_settings_profile.example name

# IMPORTANT: if you have a multi node cluster, you need to specify the cluster name!

terraform import clickhousedbops_settings_profile.example cluster:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
terraform import clickhousedbops_settings_profile.example cluster:name
