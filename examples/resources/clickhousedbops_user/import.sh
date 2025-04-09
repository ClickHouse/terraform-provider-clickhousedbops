# Users can be imported by specifying the ID.
# Find the ID of the user by checking system.users table.
# WARNING: imported users will be recreated during first 'terraform apply' because the password cannot be imported.
terraform import clickhousedbops_user.example xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
