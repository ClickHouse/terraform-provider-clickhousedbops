# Basic user creation example with modern write-only password field (Terraform 1.11+)
resource "clickhousedbops_user" "john" {
  cluster_name = "cluster"
  name         = "john"
  
  # Modern approach: Write-only password field (recommended for Terraform 1.11+)
  # Password hash is never stored in Terraform state for enhanced security
  password_sha256_hash_wo         = sha256("secure_password_123")
  password_sha256_hash_wo_version = 1
}

# Alternative: Legacy compatibility example for Terraform <1.11 and OpenTofu
# Uncomment this resource and comment the one above if using older Terraform or OpenTofu
/*
resource "clickhousedbops_user" "john_legacy" {
  cluster_name = "cluster"
  name         = "john"
  
  # Legacy approach: Sensitive field stored encrypted in state
  # Use this for Terraform <1.11 or OpenTofu compatibility
  password_sha256_hash            = sha256("secure_password_123")
  password_sha256_hash_wo_version = 1
}
*/

# Best practice: Use variables for password management
variable "user_password" {
  description = "Password for ClickHouse user"
  type        = string
  sensitive   = true
  # Set via environment variable: TF_VAR_user_password or terraform.tfvars
}

resource "clickhousedbops_user" "variable_example" {
  cluster_name = "cluster"
  name         = "variable_user"
  
  # Use variable instead of hardcoding password
  password_sha256_hash_wo         = sha256(var.user_password)
  password_sha256_hash_wo_version = 1
}
