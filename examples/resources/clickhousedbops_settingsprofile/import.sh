# Settings profiles can be imported by specifying the name.
terraform import clickhousedbops_settingsprofile.example name

# IMPORTANT: if you have a multi node cluster, you need to specify the cluster name!

terraform import clickhousedbops_settingsprofile.example cluster:name
