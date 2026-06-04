# Create a first-party OAuth Connect application
resource "workos_connect_application" "oauth" {
  name             = "Customer Portal"
  application_type = "oauth"
  is_first_party   = true
  uses_pkce        = true
  scopes           = ["openid", "profile", "email"]

  redirect_uris = [
    {
      uri     = "https://app.example.com/callback"
      default = true
    },
    {
      uri = "https://app.example.com/auth/workos/callback"
    },
  ]
}

# Create a machine-to-machine Connect application
resource "workos_connect_application" "m2m" {
  name             = "Billing Worker"
  application_type = "m2m"
  organization_id  = workos_organization.example.id
  scopes           = ["billing:read", "billing:write"]
}

output "connect_application_client_id" {
  value = workos_connect_application.oauth.client_id
}
