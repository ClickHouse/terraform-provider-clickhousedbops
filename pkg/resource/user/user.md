You can use the `clickhousedbops_user` resource to create a user in a `ClickHouse` instance.

## Authentication Options

This resource supports two approaches for authenticating users:

### Option 1: `auth_type` and `auth_value` (recommended for new configurations)

Use `auth_type` to specify the ClickHouse authentication method, and either `auth_value` or the write-only `auth_value_wo` (together with `auth_value_wo_version`) to provide the corresponding credential or identifier.

- **`auth_value_wo` and `auth_value_wo_version`**: Write-only pattern (not stored in state), so you must bump `auth_value_wo_version` to trigger auth value updates. Requires Terraform/OpenTofu >= 1.11.
- **`auth_value`**: Uses the standard `Sensitive` attribute and is stored in state. Use this for Terraform/OpenTofu < 1.11.

Supported `auth_type` values:
- **`sha256_hash`**: Authenticate with a SHA256 password hash. The auth value is the hash.
- **`ssl_certificate`**: Authenticate with a TLS client certificate. The auth value is the Common Name (CN) from the certificate.
- **`plaintext_password`**: Authenticate with a plaintext password. The auth value is the password (will be hashed server-side).
- **`bcrypt_hash`**: Authenticate with a bcrypt password hash. The auth value is the hash.
- **`double_sha1_hash`**: Authenticate with a double SHA1 hash. The auth value is the hash.
- **`no_password`**: No authentication required. Neither `auth_value` nor `auth_value_wo` must be set.

### Option 2: `password_sha256_hash` or `password_sha256_hash_wo` (legacy)

- **`password_sha256_hash_wo` and `password_sha256_hash_wo_version`**: Write-only pattern (not stored in state), so you must bump `password_sha256_hash_wo_version` to trigger password updates.
- **`password_sha256_hash`**: Use this field for OpenTofu (version < 1.11) compatibility. This field uses the standard `Sensitive` attribute and is stored in state, so OpenTofu can automatically detect password changes. Any change to this field will trigger resource replacement.

You must use either `auth_type` with `auth_value` or `auth_value_wo`/`auth_value_wo_version`, or one of
`password_sha256_hash_wo`/`password_sha256_hash_wo_version` or `password_sha256_hash`. These options are mutually exclusive.

Known limitations:

- Changing the password or authentication will cause the database user to be deleted and recreated.
- Changing `password_sha256_hash_wo` alone does not trigger an update. You must also bump `password_sha256_hash_wo_version`.
- Changing `auth_value_wo` alone does not trigger an update. You must also bump `auth_value_wo_version`.
- When importing an existing user, the `clickhousedbops_user` resource will be lacking the password or the `password_sha256_hash_wo_version`, and thus the subsequent apply will need to recreate the database User in order to set a password.
