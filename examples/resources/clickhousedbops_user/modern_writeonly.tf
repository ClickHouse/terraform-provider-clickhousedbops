# Modern write-only examples for Terraform 1.11+
# Use password_sha256_hash_wo for enhanced security with no state storage

# Basic modern user with write-only password field
resource "clickhousedbops_user" "modern_basic" {
  cluster_name = "production-cluster"
  name         = "modern_user"
  
  # Password hash not stored in state (enhanced security)
  password_sha256_hash_wo         = sha256("secure_password_123")
  password_sha256_hash_wo_version = 1
}

# Modern user with external password management
variable "service_account_password_hash" {
  description = "Pre-computed SHA256 hash of service account password"
  type        = string
  sensitive   = true
}

variable "password_version" {
  description = "Version number for password tracking"
  type        = number
  default     = 1
}

resource "clickhousedbops_user" "modern_external" {
  cluster_name = "production-cluster" 
  name         = "service_account"
  
  # Generate password externally and pass hash
  password_sha256_hash_wo         = var.service_account_password_hash
  password_sha256_hash_wo_version = var.password_version
}

# Integration with HashiCorp Vault
data "vault_generic_secret" "db_password" {
  path = "secret/database/users/app_user"
}

resource "clickhousedbops_user" "vault_integration" {
  cluster_name = "production-cluster"
  name         = "app_user"
  
  # Password from Vault, never stored in Terraform state
  password_sha256_hash_wo         = sha256(data.vault_generic_secret.db_password.data["password"])
  password_sha256_hash_wo_version = data.vault_generic_secret.db_password.data["version"]
}

# Modern password rotation example
locals {
  # Time-based password rotation (example: monthly)
  current_month = formatdate("YYYYMM", timestamp())
  password_version = tonumber(local.current_month) - 202400  # Relative to base year
}

resource "clickhousedbops_user" "auto_rotation" {
  cluster_name = "production-cluster"
  name         = "auto_rotating_user"
  
  # Automatic version increment based on current month
  password_sha256_hash_wo         = sha256("base_password_${local.current_month}")
  password_sha256_hash_wo_version = local.password_version
}

# Security-critical environment example
resource "clickhousedbops_user" "security_critical" {
  cluster_name = "security-cluster"
  name         = "audit_user"
  
  # Maximum security: write-only field with external secret management
  password_sha256_hash_wo         = sha256(var.audit_user_password)
  password_sha256_hash_wo_version = 1
}

variable "audit_user_password" {
  description = "Password for audit user (security-critical)"
  type        = string
  sensitive   = true
  validation {
    condition     = length(var.audit_user_password) >= 12
    error_message = "Audit user password must be at least 12 characters long."
  }
}