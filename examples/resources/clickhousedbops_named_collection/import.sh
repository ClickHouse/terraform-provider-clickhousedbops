# Named collections can be imported by name:
terraform import clickhousedbops_named_collection.example collection_name

# IMPORTANT: if you have a multi node cluster, you need to specify the cluster name!

terraform import clickhousedbops_named_collection.example cluster:collection_name

# NOTE: every key lands in 'keys', with the literal '[HIDDEN]' placeholder as its
# value unless the clickhouse user is granted 'SHOW NAMED COLLECTIONS SECRETS'.
# Write the real values in your terraform config, moving the secrets to
# 'secret_keys_wo', and run one apply to converge.
