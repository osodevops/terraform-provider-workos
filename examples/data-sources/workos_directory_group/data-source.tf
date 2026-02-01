# Look up a group synced from a directory

# By ID
data "workos_directory_group" "by_id" {
  id = "directory_group_01HXYZ..."
}

output "group_name" {
  value = data.workos_directory_group.by_id.name
}

# By Directory and Name
data "workos_directory_group" "engineering" {
  directory_id = "directory_01HXYZ..."
  name         = "Engineering"
}

output "engineering_group_id" {
  value = data.workos_directory_group.engineering.id
}
