resource "workos_connect_application" "m2m" {
  name             = "Billing Worker"
  application_type = "m2m"
  organization_id  = workos_organization.example.id
  scopes           = ["billing:read"]
}

# WorkOS returns the plaintext secret only once, on create, so it is stored in
# Terraform state. Protect the state file accordingly.
resource "workos_connect_application_client_secret" "m2m" {
  application_id = workos_connect_application.m2m.id

  # Changing any value here mints a replacement and revokes the old secret.
  rotate_triggers = {
    rotated_at = "2026-01-01T00:00:00Z"
  }
}

output "client_id" {
  value = workos_connect_application.m2m.client_id
}

output "client_secret" {
  value     = workos_connect_application_client_secret.m2m.secret
  sensitive = true
}
