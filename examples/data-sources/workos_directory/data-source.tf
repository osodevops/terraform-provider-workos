# Look up an existing directory

# By ID
data "workos_directory" "by_id" {
  id = "directory_01HXYZ..."
}

output "directory_type" {
  value = data.workos_directory.by_id.type
}

# By Organization
data "workos_directory" "by_org" {
  organization_id = "org_01HXYZ..."
}

output "directory_endpoint" {
  value = data.workos_directory.by_org.endpoint
}
