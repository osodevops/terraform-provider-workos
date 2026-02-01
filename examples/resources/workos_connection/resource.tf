# Manage an SSO Connection for an organization

# First, create or reference an organization
resource "workos_organization" "example" {
  name    = "Acme Corporation"
  domains = ["acme.com"]
}

# Create an Okta SAML connection
resource "workos_connection" "okta_sso" {
  organization_id = workos_organization.example.id
  connection_type = "OktaSAML"
  name            = "Okta SSO"
}

# Create a Google OAuth connection
resource "workos_connection" "google" {
  organization_id = workos_organization.example.id
  connection_type = "GoogleOAuth"
  name            = "Google Login"
}

# Output the connection IDs
output "okta_connection_id" {
  value = workos_connection.okta_sso.id
}

output "google_connection_id" {
  value = workos_connection.google.id
}
