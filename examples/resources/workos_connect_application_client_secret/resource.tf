resource "workos_connect_application" "m2m" {
  name             = "Billing Worker"
  application_type = "m2m"
  organization_id  = workos_organization.example.id
  scopes           = ["billing:read"]
}

resource "workos_connect_application_client_secret" "m2m" {
  application_id = workos_connect_application.m2m.id
}

output "client_id" {
  value = workos_connect_application.m2m.client_id
}

output "client_secret" {
  value     = workos_connect_application_client_secret.m2m.secret
  sensitive = true
}
