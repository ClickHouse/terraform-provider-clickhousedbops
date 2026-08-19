# Databases can be imported by specifying a unique UUID.
# Some database engines use the shared zero UUID; import those databases by name.
terraform import clickhousedbops_database.example xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# It's also possible to import databases using the name:

terraform import clickhousedbops_database.example databasename

# IMPORTANT: if you have a multi node cluster, you need to specify the cluster name!
# Name import is recommended for clustered databases because UUIDs can differ
# between replicas (for example, an Atomic database created independently on each replica).

terraform import clickhousedbops_database.example cluster:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
terraform import clickhousedbops_database.example cluster:databasename
