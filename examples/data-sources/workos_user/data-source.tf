# Look up a user by ID
data "workos_user" "by_id" {
  id = "user_01HXYZ..."
}

output "user_email" {
  value = data.workos_user.by_id.email
}

# Look up a user by email
data "workos_user" "by_email" {
  email = "user@example.com"
}

output "user_full_name" {
  value = "${data.workos_user.by_email.first_name} ${data.workos_user.by_email.last_name}"
}

output "email_verified" {
  value = data.workos_user.by_email.email_verified
}
