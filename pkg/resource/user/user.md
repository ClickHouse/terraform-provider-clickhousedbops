You can use the `clickhousedbops_user` resource to create a user in a `ClickHouse` instance.

## Password Field Options

This resource supports two mutually exclusive password fields designed for different Terraform/OpenTofu environments:

### Modern Approach (Recommended)
- **`password_sha256_hash_wo`**: Write-only field for Terraform 1.11+
- **Enhanced Security**: Password hash never stored in state file
- **Better Privacy**: No password data persisted or transmitted in state operations
- **Version Requirement**: Requires Terraform 1.11.0 or later

### Legacy Compatibility 
- **`password_sha256_hash`**: Sensitive field for Terraform <1.11 and OpenTofu
- **State Storage**: Password hash stored encrypted in state file
- **Compatibility**: Works with all Terraform and OpenTofu versions
- **Use Case**: Required for Terraform <1.11 or when using OpenTofu

## Decision Criteria

Choose your password field based on:

| Environment | Recommended Field | Reason |
|-------------|------------------|---------|
| Terraform 1.11+ | `password_sha256_hash_wo` | Enhanced security, no state storage |
| Terraform <1.11 | `password_sha256_hash` | Write-only fields not supported |
| OpenTofu (any version) | `password_sha256_hash` | Write-only fields not supported |
| Security-critical environments | `password_sha256_hash_wo` | Password never persisted |
| Mixed tool environments | `password_sha256_hash` | Universal compatibility |

**Important**: You must specify exactly one of these fields. Specifying both will cause validation failure.

## Known limitations:

- **Password changes require version bump**: Changing password fields alone has no effect. You must increment `password_sha256_hash_wo_version`.
- **User recreation on password change**: Password updates cause the database user to be deleted and recreated.
- **Import limitations**: Imported users lack `password_sha256_hash_wo_version`, requiring recreation on first apply.
- **Version field applies to both**: The `password_sha256_hash_wo_version` field controls both password field types.
