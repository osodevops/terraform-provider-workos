# Manage a WorkOS Organization
resource "workos_organization" "example" {
  name    = "Acme Corporation"
  domains = ["acme.com", "acmecorp.com"]
}

# Output the organization ID for use in other resources
output "organization_id" {
  value = workos_organization.example.id
}

output "organization_created_at" {
  value = workos_organization.example.created_at
}
