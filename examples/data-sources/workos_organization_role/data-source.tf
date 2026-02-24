# Look up an organization role by slug
data "workos_organization_role" "by_slug" {
  organization_id = "org_01HXYZ..."
  slug            = "org-billing-admin"
}

output "role_name_by_slug" {
  value = data.workos_organization_role.by_slug.name
}

# Look up an organization role by ID
data "workos_organization_role" "by_id" {
  organization_id = "org_01HXYZ..."
  id              = "role_01HXYZ..."
}

output "role_name_by_id" {
  value = data.workos_organization_role.by_id.name
}
