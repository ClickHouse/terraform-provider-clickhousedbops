# Named collections can be imported by name:
terraform import clickhousedbops_named_collection.example collection_name

# IMPORTANT: if you have a multi node cluster, you need to specify the cluster name!

terraform import clickhousedbops_named_collection.example cluster:collection_name

# NOTE: values are imported as the literal '[HIDDEN]' placeholder unless the
# clickhouse user is granted 'SHOW NAMED COLLECTIONS SECRETS'.
# Write the real values in your terraform config and run one apply to converge.
