You can use the `clickhousedbops_user` resource to create a user in a `ClickHouse` instance.

## Password Field Options

This resource supports two approaches for setting passwords:

- **`password_sha256_hash_wo` and `password_sha256_hash_wo_version`**:  field uses the write-only pattern (not stored in state), so you must bump `password_sha256_hash_wo_version` to trigger password updates.
- **`password_sha256_hash`**: Use this field for OpenTofu (version < 1.11) compatibility. This field uses the standard `Sensitive` attribute and is stored in state, so OpenTofu can automatically detect password changes. Any change to this field will trigger resource replacement.

You must use either `password_sha256_hash_wo`/`password_sha256_hash_wo_version` pair
OR `password_sha256_hash`, but not both.

Known limitations:

- Changing the password will cause the database user to be deleted and recreated.
- Changing `password_sha256_hash_wo` alone does not trigger an update. You must also bump `password_sha256_hash_wo_version`.
- When importing an existing user, the `clickhousedbops_user` resource will be lacking the password or the `password_sha256_hash_wo_version`, and thus the subsequent apply will need to recreate the database User in order to set a password.
