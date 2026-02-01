# Basic user with email verification
resource "workos_user" "basic" {
  email          = "user@example.com"
  first_name     = "John"
  last_name      = "Doe"
  email_verified = true
}

# User with password authentication
resource "workos_user" "with_password" {
  email          = "jane@example.com"
  first_name     = "Jane"
  last_name      = "Smith"
  password       = var.user_password
  email_verified = true
}

# Minimal user (email only)
resource "workos_user" "minimal" {
  email = "minimal@example.com"
}

# User for SSO-only authentication (no password)
resource "workos_user" "sso_user" {
  email          = "sso-user@example.com"
  first_name     = "SSO"
  last_name      = "User"
  email_verified = true
}

# User with pre-hashed password (for migration)
resource "workos_user" "migrated" {
  email          = "migrated@example.com"
  first_name     = "Migrated"
  last_name      = "User"
  password_hash  = var.bcrypt_password_hash
  email_verified = true
}

# Variables
variable "user_password" {
  type        = string
  description = "Password for the user"
  sensitive   = true
}

variable "bcrypt_password_hash" {
  type        = string
  description = "Pre-hashed bcrypt password for migration"
  sensitive   = true
}

# Outputs
output "user_id" {
  value       = workos_user.basic.id
  description = "The ID of the basic user"
}

output "user_email" {
  value       = workos_user.basic.email
  description = "The email of the basic user"
}
