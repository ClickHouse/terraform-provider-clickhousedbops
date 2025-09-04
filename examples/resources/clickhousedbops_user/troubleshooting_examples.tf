# Troubleshooting examples for common user configuration scenarios
# These examples demonstrate solutions to frequently encountered issues

# Example 1: Fixing "Conflicting configuration arguments" error
# WRONG: Both password fields specified (will cause validation error)
/*
resource "clickhousedbops_user" "conflicting_fields" {
  name = "error_user"
  
  # ERROR: Cannot specify both fields
  password_sha256_hash_wo = sha256("password123")
  password_sha256_hash    = sha256("password123")
  password_sha256_hash_wo_version = 1
}
*/

# CORRECT: Only one password field specified
resource "clickhousedbops_user" "single_field_modern" {
  cluster_name = "production-cluster"
  name         = "correct_modern_user"
  
  # Use only one password field
  password_sha256_hash_wo         = sha256("password123")
  password_sha256_hash_wo_version = 1
}

resource "clickhousedbops_user" "single_field_legacy" {
  cluster_name = "production-cluster"
  name         = "correct_legacy_user"
  
  # Use only one password field
  password_sha256_hash            = sha256("password123")
  password_sha256_hash_wo_version = 1
}

# Example 2: Handling Terraform version compatibility
# For environments where Terraform version is uncertain
locals {
  # Determine if write-only fields are supported (Terraform 1.11+)
  terraform_version_supports_writeonly = true  # Set based on your environment
}

# Conditional password field selection (advanced usage)
resource "clickhousedbops_user" "version_adaptive_user" {
  cluster_name = "adaptive-cluster"
  name         = "adaptive_user"
  
  # Use modern field if supported, fallback to legacy
  password_sha256_hash_wo = local.terraform_version_supports_writeonly ? sha256("adaptive_password") : null
  password_sha256_hash    = !local.terraform_version_supports_writeonly ? sha256("adaptive_password") : null
  
  password_sha256_hash_wo_version = 1
}

# Example 3: Import and recreation handling
# After importing a user, proper configuration to avoid continuous recreation
resource "clickhousedbops_user" "imported_user" {
  cluster_name = "import-cluster"
  name         = "imported_user"
  
  # Set password field and version after import
  password_sha256_hash_wo         = sha256("imported_user_password")
  password_sha256_hash_wo_version = 1  # Start with version 1 after import
}

# Example 4: Password rotation without constant recreation
variable "password_rotation_counter" {
  description = "Increment this to rotate passwords"
  type        = number
  default     = 1
}

resource "clickhousedbops_user" "rotating_user" {
  cluster_name = "production-cluster"
  name         = "rotating_user"
  
  # Password changes when counter is incremented
  password_sha256_hash_wo         = sha256("base_password_${var.password_rotation_counter}")
  password_sha256_hash_wo_version = var.password_rotation_counter
}

# Example 5: Environment-specific configuration
# Different approaches for different deployment environments

# Production: Maximum security with write-only fields
resource "clickhousedbops_user" "production_user" {
  count = var.environment == "production" ? 1 : 0
  
  cluster_name = "production-cluster"
  name         = "prod_user"
  
  # Enhanced security for production
  password_sha256_hash_wo         = sha256(var.production_password)
  password_sha256_hash_wo_version = 1
}

# Development: Legacy compatibility for mixed teams
resource "clickhousedbops_user" "development_user" {
  count = var.environment == "development" ? 1 : 0
  
  cluster_name = "development-cluster"  
  name         = "dev_user"
  
  # Legacy compatibility for development
  password_sha256_hash            = sha256(var.development_password)
  password_sha256_hash_wo_version = 1
}

# Example 6: Error recovery after failed operations
# Configuration that handles common failure scenarios gracefully

resource "clickhousedbops_user" "resilient_user" {
  cluster_name = "resilient-cluster"
  name         = "resilient_user"
  
  # Stable configuration that minimizes recreation
  password_sha256_hash_wo         = sha256("stable_password_${var.stable_version}")
  password_sha256_hash_wo_version = var.stable_version
  
  # Lifecycle management to prevent accidental deletion
  lifecycle {
    prevent_destroy = true
    ignore_changes = [
      # Ignore changes to cluster_name to prevent recreation
      cluster_name,
    ]
  }
}

# Supporting variables
variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "development"
  validation {
    condition     = contains(["development", "staging", "production"], var.environment)
    error_message = "Environment must be development, staging, or production."
  }
}

variable "production_password" {
  description = "Password for production environment"
  type        = string
  sensitive   = true
  default     = null
}

variable "development_password" {
  description = "Password for development environment"  
  type        = string
  sensitive   = true
  default     = "dev_default_password"
}

variable "stable_version" {
  description = "Stable version number for resilient user"
  type        = number
  default     = 1
}