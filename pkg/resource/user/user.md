You can use the `clickhousedbops_user` resource to create a user in a `ClickHouse` instance.

Known limitations:

- Changing the `password_sha256_hash_wo` field alone does not have any effect. In order to change the password of a user, you also need to bump `password_sha256_hash_wo_version` field.
- Changing the user's password as described above will cause the database user to be deleted and recreated.
- When importing an existing user, the `clickhousedbops_user` resource will be lacking the `password_sha256_hash_wo_version` and thus the subsequent apply will need to recreate the database User in order to set a password.
