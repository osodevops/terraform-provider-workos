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
