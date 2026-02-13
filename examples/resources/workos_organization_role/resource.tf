# Manage a WorkOS Organization Role
resource "workos_organization_role" "billing_admin" {
  organization_id = workos_organization.example.id
  slug            = "billing-admin"
  name            = "Billing Admin"
  description     = "Can manage billing and invoices"
}

# Role without a description
resource "workos_organization_role" "viewer" {
  organization_id = workos_organization.example.id
  slug            = "viewer"
  name            = "Viewer"
}

# Output the role details
output "billing_admin_role_id" {
  value = workos_organization_role.billing_admin.id
}

output "billing_admin_role_type" {
  value = workos_organization_role.billing_admin.type
}
