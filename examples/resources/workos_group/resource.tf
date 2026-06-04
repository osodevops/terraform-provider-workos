# Create a group within an organization
resource "workos_group" "engineering" {
  organization_id = workos_organization.example.id
  name            = "Engineering"
  description     = "Engineering team members"
}

resource "workos_group" "billing" {
  organization_id = workos_organization.example.id
  name            = "Billing"
}

output "engineering_group_id" {
  value = workos_group.engineering.id
}
