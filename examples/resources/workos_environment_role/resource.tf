# Manage an environment-level role
resource "workos_environment_role" "billing_admin" {
  slug        = "billing-admin"
  name        = "Billing Admin"
  description = "Can manage billing across organizations"

  permissions = [
    workos_permission.billing_read.slug,
    workos_permission.billing_write.slug,
  ]
}

resource "workos_permission" "billing_read" {
  slug = "billing:read"
  name = "Read Billing"
}

resource "workos_permission" "billing_write" {
  slug = "billing:write"
  name = "Write Billing"
}

output "billing_admin_role_id" {
  value = workos_environment_role.billing_admin.id
}
