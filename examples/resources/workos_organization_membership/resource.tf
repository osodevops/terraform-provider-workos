# Create an organization
resource "workos_organization" "example" {
  name    = "Acme Corporation"
  domains = ["acme.com"]
}

# Create users
resource "workos_user" "admin" {
  email          = "admin@acme.com"
  first_name     = "Admin"
  last_name      = "User"
  email_verified = true
}

resource "workos_user" "member" {
  email          = "member@acme.com"
  first_name     = "Regular"
  last_name      = "Member"
  email_verified = true
}

resource "workos_user" "viewer" {
  email          = "viewer@acme.com"
  first_name     = "View"
  last_name      = "Only"
  email_verified = true
}

# Basic membership (default role)
resource "workos_organization_membership" "member" {
  user_id         = workos_user.member.id
  organization_id = workos_organization.example.id
}

# Admin membership
resource "workos_organization_membership" "admin" {
  user_id         = workos_user.admin.id
  organization_id = workos_organization.example.id
  role_slug       = "admin"
}

# Viewer membership
resource "workos_organization_membership" "viewer" {
  user_id         = workos_user.viewer.id
  organization_id = workos_organization.example.id
  role_slug       = "viewer"
}

# Outputs
output "admin_membership_id" {
  value       = workos_organization_membership.admin.id
  description = "The ID of the admin membership"
}

output "membership_status" {
  value       = workos_organization_membership.member.status
  description = "The status of the member's membership"
}
