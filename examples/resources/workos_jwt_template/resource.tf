# Add custom claims to every access token AuthKit issues for the environment
# bound to the API key. Keep custom claim names namespaced.
resource "workos_jwt_template" "this" {
  content = jsonencode({
    "urn:example:full_name" = "{{user.first_name}} {{user.last_name}}"
    "urn:example:email"     = "{{user.email}}"
  })
}
