# Legacy compatibility examples for Terraform <1.11 and OpenTofu
# Use password_sha256_hash for compatibility with older Terraform versions

# Basic legacy user with password_sha256_hash field
resource "clickhousedbops_user" "legacy_basic" {
  cluster_name = "production-cluster"
  name         = "legacy_user"
  
  # Password hash stored encrypted in state (legacy compatibility)
  password_sha256_hash            = sha256("legacy_password_123")
  password_sha256_hash_wo_version = 1
}

# Legacy user with variable for password management
variable "legacy_user_password" {
  description = "Password for legacy ClickHouse user"
  type        = string
  sensitive   = true
}

resource "clickhousedbops_user" "legacy_variable" {
  cluster_name = "production-cluster"
  name         = "legacy_variable_user"
  
  # Using sensitive variables with legacy field
  password_sha256_hash            = sha256(var.legacy_user_password)
  password_sha256_hash_wo_version = 1
}

# OpenTofu compatible user (same as legacy approach)
resource "clickhousedbops_user" "opentofu_user" {
  cluster_name = "development-cluster"
  name         = "opentofu_user"
  
  # OpenTofu requires the legacy field since write-only fields are not supported
  password_sha256_hash            = sha256("opentofu_password_456")
  password_sha256_hash_wo_version = 1
}

# Password rotation example with legacy field
resource "clickhousedbops_user" "legacy_rotation" {
  cluster_name = "production-cluster"
  name         = "rotating_user"
  
  # Increment password_sha256_hash_wo_version to trigger password update
  password_sha256_hash            = sha256("rotated_password_${var.password_rotation_version}")
  password_sha256_hash_wo_version = var.password_rotation_version
}

variable "password_rotation_version" {
  description = "Version number for password rotation"
  type        = number
  default     = 1
}