resource "workos_organization_role_permission" "billing_admin_read" {
  organization_id = workos_organization.example.id
  role_slug       = workos_organization_role.billing_admin.slug
  permission      = workos_permission.billing_read.slug
}
