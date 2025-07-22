# Roles can be imported by specifying the ID.
# Find the ID of the role by checking system.roles table.
terraform import clickhousedbops_role.example xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# It's also possible to import roles by name:

terraform import clickhousedbops_role.example rolename

# IMPORTANT: if you have a multi node cluster, you need to specify the cluster name!

terraform import clickhousedbops_role.example cluster:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
terraform import clickhousedbops_role.example cluster:rolename
