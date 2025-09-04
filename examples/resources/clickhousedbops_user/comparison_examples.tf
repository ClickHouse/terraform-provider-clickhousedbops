# Side-by-side comparison of password field approaches
# Demonstrates the differences between modern and legacy password fields

# Modern approach (Terraform 1.11+) - Recommended for enhanced security
resource "clickhousedbops_user" "modern_comparison" {
  cluster_name = "production-cluster"
  name         = "modern_user"
  
  # Write-only field - password hash never stored in state
  password_sha256_hash_wo         = sha256("comparison_password_123")
  password_sha256_hash_wo_version = 1
}

# Legacy approach (all versions) - Compatible with Terraform <1.11 and OpenTofu  
resource "clickhousedbops_user" "legacy_comparison" {
  cluster_name = "production-cluster"
  name         = "legacy_user"
  
  # Sensitive field - password hash stored encrypted in state
  password_sha256_hash            = sha256("comparison_password_123") 
  password_sha256_hash_wo_version = 1
}

# Migration example: From legacy to modern
# Step 1: Current configuration (legacy)
resource "clickhousedbops_user" "migration_user" {
  cluster_name = "production-cluster"
  name         = "migration_user"
  
  # Before: Legacy field
  password_sha256_hash            = sha256("migration_password")
  password_sha256_hash_wo_version = 1
}

# Step 2: After upgrading to Terraform 1.11+ (modern)
# Uncomment the resource below and comment out the one above for migration
/*
resource "clickhousedbops_user" "migration_user" {
  cluster_name = "production-cluster"
  name         = "migration_user"
  
  # After: Modern field
  password_sha256_hash_wo         = sha256("migration_password")
  password_sha256_hash_wo_version = 1  # Keep same version to avoid recreation
}
*/

# Decision matrix examples based on environment

# Example 1: Security-critical production environment (Terraform 1.11+)
resource "clickhousedbops_user" "security_critical_user" {
  cluster_name = "production-cluster"
  name         = "security_user"
  
  # Choose modern approach for maximum security
  password_sha256_hash_wo         = sha256(var.security_password)
  password_sha256_hash_wo_version = 1
}

# Example 2: Development environment with mixed Terraform versions
resource "clickhousedbops_user" "mixed_env_user" {
  cluster_name = "development-cluster"
  name         = "dev_user"
  
  # Choose legacy approach for compatibility
  password_sha256_hash            = sha256(var.dev_password)
  password_sha256_hash_wo_version = 1
}

# Example 3: OpenTofu environment (any version)
resource "clickhousedbops_user" "opentofu_only_user" {
  cluster_name = "opentofu-cluster"
  name         = "opentofu_user"
  
  # Must use legacy approach - write-only fields not supported in OpenTofu
  password_sha256_hash            = sha256(var.opentofu_password)
  password_sha256_hash_wo_version = 1
}

# Variables for the examples
variable "security_password" {
  description = "Password for security-critical user"
  type        = string
  sensitive   = true
}

variable "dev_password" {
  description = "Password for development user"
  type        = string
  sensitive   = true
  default     = "dev_password_123"
}

variable "opentofu_password" {
  description = "Password for OpenTofu user"
  type        = string
  sensitive   = true
  default     = "opentofu_password_456"
}