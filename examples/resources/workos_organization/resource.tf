# Manage a WorkOS Organization
resource "workos_organization" "example" {
  name        = "Acme Corporation"
  external_id = "acme-corp-123"
  domains     = ["acme.com", "acmecorp.com"]

  metadata = {
    tier   = "enterprise"
    region = "us-east-1"
  }
}

# Output the organization ID for use in other resources
output "organization_id" {
  value = workos_organization.example.id
}

output "organization_created_at" {
  value = workos_organization.example.created_at
}
