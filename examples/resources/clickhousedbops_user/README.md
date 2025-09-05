# ClickHouse User Resource Examples

This directory contains examples for using the `clickhousedbops_user` resource with dual-path password support.

## Password Field Options

The `clickhousedbops_user` resource supports two mutually exclusive password fields:

- **`password_sha256_hash_wo`** (Modern): Write-only field for Terraform 1.11+, never stored in state
- **`password_sha256_hash`** (Legacy): Sensitive field for Terraform <1.11 and OpenTofu, stored encrypted in state

## Quick Decision Guide

| Your Environment | Use This Field | 
|------------------|----------------|
| Terraform 1.11+ (Security Priority) | `password_sha256_hash_wo` |
| Terraform <1.11 | `password_sha256_hash` |
| OpenTofu (Any Version) | `password_sha256_hash` |
| Mixed Tool Environment | `password_sha256_hash` |

## Security Best Practices

### ✅ Recommended
- Use `password_sha256_hash_wo` with Terraform 1.11+
- Store passwords in external secret management systems
- Use variables instead of hardcoded values
- Implement regular password rotation

### ❌ Avoid
- Hardcoding passwords in `.tf` files
- Specifying both password fields simultaneously
- Storing sensitive passwords in version control
- Using predictable password patterns

## Import Considerations

When importing existing users:

1. **Expected Behavior**: User will be recreated on first apply (password cannot be imported)
2. **Post-Import Steps**:
   - Add password configuration to your Terraform file
   - Set `password_sha256_hash_wo_version = 1`
   - Run `terraform apply` to recreate with password
   - Subsequent applies will be stable

## Password Updates

To update a user's password:

1. Change the password hash in your configuration
2. **Important**: Increment `password_sha256_hash_wo_version`
3. Run `terraform apply`
4. The user will be deleted and recreated with the new password

## Error Resolution

### "Conflicting configuration arguments"
**Cause**: Both `password_sha256_hash_wo` and `password_sha256_hash` specified  
**Solution**: Use only one password field

### "Unsupported argument: password_sha256_hash_wo"
**Cause**: Using write-only field with Terraform <1.11 or OpenTofu  
**Solution**: Use `password_sha256_hash` instead or upgrade Terraform

### "Resource will be recreated on every apply"
**Cause**: Missing or inconsistent `password_sha256_hash_wo_version`  
**Solution**: Set consistent version value and increment only for password changes