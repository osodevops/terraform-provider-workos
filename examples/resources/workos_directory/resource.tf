# Manage a Directory Sync directory

resource "workos_organization" "example" {
  name = "Acme Corporation"
}

# Create an Okta SCIM directory
resource "workos_directory" "okta" {
  organization_id = workos_organization.example.id
  name            = "Okta Directory"
  type            = "okta scim v2.0"
}

# Output the SCIM configuration for your IdP
output "scim_endpoint" {
  description = "Configure your IdP to send SCIM requests to this URL"
  value       = workos_directory.okta.endpoint
}

output "scim_bearer_token" {
  description = "Use this token to authenticate SCIM requests"
  value       = workos_directory.okta.bearer_token
  sensitive   = true
}

output "directory_state" {
  description = "Current state of the directory"
  value       = workos_directory.okta.state
}
