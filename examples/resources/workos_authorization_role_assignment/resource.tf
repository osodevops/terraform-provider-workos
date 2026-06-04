# Assign a role to a membership on an authorization resource
resource "workos_authorization_role_assignment" "workspace_admin" {
  organization_membership_id = workos_organization_membership.admin.id
  role_slug                  = workos_organization_role.workspace_admin.slug
  resource_id                = workos_authorization_resource.workspace.id
}

# Assign a role using a resource external ID
resource "workos_authorization_role_assignment" "project_viewer" {
  organization_membership_id = workos_organization_membership.viewer.id
  role_slug                  = workos_organization_role.project_viewer.slug
  resource_type_slug         = "project"
  resource_external_id       = "project-456"
}

output "workspace_assignment_id" {
  value = workos_authorization_role_assignment.workspace_admin.id
}
