# Register an AuthKit login callback on the application bound to the API key.
# This is not workos_connect_application.redirect_uris.
resource "workos_redirect_uri" "tenant_callback" {
  uri = "https://acme.example.com/api/auth/callback"
}

output "redirect_uri_id" {
  value = workos_redirect_uri.tenant_callback.id
}
