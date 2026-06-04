# Add an organization membership to a group
resource "workos_group_membership" "engineering_member" {
  organization_id            = workos_organization.example.id
  group_id                   = workos_group.engineering.id
  organization_membership_id = workos_organization_membership.member.id
}

output "engineering_member_status" {
  value = workos_group_membership.engineering_member.status
}
