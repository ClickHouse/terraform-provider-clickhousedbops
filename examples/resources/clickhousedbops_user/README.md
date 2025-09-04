# ClickHouse User Resource Examples

This directory contains comprehensive examples for using the `clickhousedbops_user` resource with dual-path password support.

## Password Field Options

The `clickhousedbops_user` resource supports two mutually exclusive password fields:

- **`password_sha256_hash_wo`** (Modern): Write-only field for Terraform 1.11+, never stored in state
- **`password_sha256_hash`** (Legacy): Sensitive field for Terraform <1.11 and OpenTofu, stored encrypted in state

## Quick Decision Guide

| Your Environment | Use This Field | Example File |
|------------------|----------------|--------------|
| Terraform 1.11+ (Security Priority) | `password_sha256_hash_wo` | `modern_writeonly.tf` |
| Terraform <1.11 | `password_sha256_hash` | `legacy_compatibility.tf` |
| OpenTofu (Any Version) | `password_sha256_hash` | `legacy_compatibility.tf` |
| Mixed Tool Environment | `password_sha256_hash` | `legacy_compatibility.tf` |

## Example Files

### 📁 `resource.tf`
- Basic user creation examples
- Shows both modern and legacy approaches
- Includes variable-based password management
- **Start here** for simple use cases

### 📁 `modern_writeonly.tf` 
- **Terraform 1.11+** examples with enhanced security
- External secret management integration (HashiCorp Vault)
- Advanced password rotation patterns
- Security-critical environment configurations

### 📁 `legacy_compatibility.tf`
- **Terraform <1.11** and **OpenTofu** compatible examples
- Variable-based password management
- Password rotation with legacy fields
- Universal compatibility patterns

### 📁 `comparison_examples.tf`
- **Side-by-side** comparison of modern vs legacy approaches
- Migration examples between field types
- Environment-specific decision patterns
- Decision matrix implementations

### 📁 `troubleshooting_examples.tf`
- Solutions for common configuration errors
- Import and recreation handling
- Environment-specific error recovery
- Advanced lifecycle management

## Common Use Cases

### New Project (Terraform 1.11+)
```bash
# Use modern approach for new projects
cp modern_writeonly.tf your_project/
# or start with resource.tf for basic needs
```

### Existing Project (Terraform <1.11)
```bash
# Use legacy approach for compatibility
cp legacy_compatibility.tf your_project/
```

### Migration Project
```bash
# Use comparison examples for migration guidance
cp comparison_examples.tf your_project/
```

### Troubleshooting Issues
```bash
# Use troubleshooting examples for error resolution
cp troubleshooting_examples.tf your_project/
```

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

## Additional Resources

- [Terraform Write-Only Arguments Documentation](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments)
- [ClickHouse User Management Documentation](https://clickhouse.com/docs/en/operations/access-rights/)
- [Terraform Sensitive Variables](https://developer.hashicorp.com/terraform/tutorials/configuration-language/sensitive-variables)