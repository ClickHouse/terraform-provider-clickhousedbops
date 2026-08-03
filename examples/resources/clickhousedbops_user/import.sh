# Users can be imported by specifying the ID.
# Find the ID of the user by checking system.users table.
terraform import clickhousedbops_user.example xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# It's also possible to import users using the username:

terraform import clickhousedbops_user.example username

# IMPORTANT: if you have a multi node cluster, you need to specify the cluster name!

terraform import clickhousedbops_user.example cluster:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
terraform import clickhousedbops_user.example cluster:username
