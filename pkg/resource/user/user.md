You can use the `clickhousedbops_user` resource to create a user in a `ClickHouse` instance.

## Password Field Options

This resource supports two mutually exclusive password fields for different Terraform versions:

- **`password_sha256_hash_wo`** (Recommended, Terraform 1.11+): Write-only field that is not stored in state for enhanced security
- **`password_sha256_hash`** (Legacy compatibility): Sensitive field stored encrypted in state for Terraform <1.11 and OpenTofu compatibility

You must specify exactly one of these fields. If both are specified, validation will fail.

## Known limitations:

- Changing password fields alone does not have any effect. In order to change the password of a user, you also need to bump `password_sha256_hash_wo_version` field.
- Changing the user's password as described above will cause the database user to be deleted and recreated.
- When importing an existing user, the `clickhousedbops_user` resource will be lacking the `password_sha256_hash_wo_version` and thus the subsequent apply will need to recreate the database User in order to set a password.
- The `password_sha256_hash_wo_version` field applies to both password field options.
