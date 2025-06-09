# Databases can be imported by specifying the UUID.
# Find the UUID of the database by checking system.databases table.
terraform import clickhousedbops_database.example xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# It's also possible to import databases using the name:

terraform import clickhousedbops_database.example databasename

# IMPORTANT: if you have a multi node cluster, you need to specify the cluster name!

terraform import clickhousedbops_database.example cluster:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
terraform import clickhousedbops_database.example cluster:databasename
