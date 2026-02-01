# Look up a user synced from a directory

# By ID
data "workos_directory_user" "by_id" {
  id = "directory_user_01HXYZ..."
}

output "user_email" {
  value = data.workos_directory_user.by_id.email
}

# By Directory and Email
data "workos_directory_user" "john" {
  directory_id = "directory_01HXYZ..."
  email        = "john@example.com"
}

output "john_full_name" {
  value = "${data.workos_directory_user.john.first_name} ${data.workos_directory_user.john.last_name}"
}
