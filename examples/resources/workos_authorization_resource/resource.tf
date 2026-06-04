# Create a top-level authorization resource
resource "workos_authorization_resource" "workspace" {
  organization_id    = workos_organization.example.id
  resource_type_slug = "workspace"
  external_id        = "workspace-123"
  name               = "Acme Workspace"
  description        = "Primary workspace for Acme"
  cascade_delete     = false
}

# Create a child authorization resource
resource "workos_authorization_resource" "project" {
  organization_id    = workos_organization.example.id
  resource_type_slug = "project"
  external_id        = "project-456"
  name               = "Launch Project"
  parent_resource_id = workos_authorization_resource.workspace.id
}

output "workspace_resource_id" {
  value = workos_authorization_resource.workspace.id
}
