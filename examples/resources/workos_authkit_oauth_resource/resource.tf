# Let MCP clients request tokens for this server: AuthKit accepts the URI as the
# OAuth resource parameter and issues it as the token's aud claim.
resource "workos_authkit_oauth_resource" "mcp" {
  uri = "https://mcp.example.com"

  lifecycle {
    prevent_destroy = true
  }
}
