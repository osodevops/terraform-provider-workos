# Look up an environment-level role by slug
data "workos_environment_role" "admin" {
  slug = "admin"
}

output "admin_role_id" {
  value = data.workos_environment_role.admin.id
}

output "admin_role_permissions" {
  value = data.workos_environment_role.admin.permissions
}

# Look up an environment-level role by ID
data "workos_environment_role" "by_id" {
  id = "role_01HXYZ..."
}
