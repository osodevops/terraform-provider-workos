# Look up an organization by ID
data "workos_organization" "by_id" {
  id = "org_01HXYZ..."
}

output "org_name_by_id" {
  value = data.workos_organization.by_id.name
}

# Look up an organization by domain
data "workos_organization" "by_domain" {
  domain = "acme.com"
}

output "org_name_by_domain" {
  value = data.workos_organization.by_domain.name
}

# Look up an organization by external ID
data "workos_organization" "by_external_id" {
  external_id = "acme-corp-123"
}

output "org_name_by_external_id" {
  value = data.workos_organization.by_external_id.name
}
