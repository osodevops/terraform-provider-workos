# Add an additional domain to an organization
resource "workos_organization_domain" "example" {
  organization_id = workos_organization.example.id
  domain          = "example.com"
}

# Start WorkOS DNS verification for a domain
resource "workos_organization_domain" "verified" {
  organization_id = workos_organization.example.id
  domain          = "login.example.com"
  verify          = true
}

output "organization_domain_state" {
  value = workos_organization_domain.example.state
}
