# WorkOS Terraform Provider

Terraform provider for managing [WorkOS](https://workos.com) resources including organizations, SSO connections, directory sync, webhooks, and user management.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for development)

## Installation

### From Terraform Registry (Recommended)

```hcl
terraform {
  required_providers {
    workos = {
      source  = "osodevops/workos"
      version = "~> 1.0"
    }
  }
}

provider "workos" {
  api_key = var.workos_api_key
}
```

### Local Development

```bash
# Clone the repository
git clone https://github.com/osodevops/terraform-provider-workos.git
cd terraform-provider-workos

# Build the provider
make build

# Install locally
make install
```

## Usage

### Provider Configuration

```hcl
provider "workos" {
  api_key   = var.workos_api_key   # Or set WORKOS_API_KEY env var
  client_id = var.workos_client_id # Or set WORKOS_CLIENT_ID env var (optional)
  base_url  = "https://api.workos.com" # Optional, defaults to production API
}
```

### Managing Organizations

```hcl
resource "workos_organization" "example" {
  name    = "Acme Corporation"
  domains = ["acme.com", "acmecorp.com"]

  allow_profiles_outside_organization = false
}
```

### Managing SSO Connections

```hcl
resource "workos_connection" "okta" {
  organization_id = workos_organization.example.id
  name            = "Okta SSO"
  connection_type = "OktaSAML"
}

resource "workos_connection" "google" {
  organization_id = workos_organization.example.id
  name            = "Google OAuth"
  connection_type = "GoogleOAuth"
}
```

### Managing Directory Sync

```hcl
resource "workos_directory" "okta" {
  organization_id = workos_organization.example.id
  name            = "Okta Directory"
  type            = "okta scimv2.0"
}
```

### Managing Webhooks

```hcl
resource "workos_webhook" "main" {
  url     = "https://api.example.com/webhooks/workos"
  secret  = var.webhook_secret
  enabled = true

  events = [
    "user.created",
    "user.updated",
    "dsync.user.created",
    "connection.activated",
  ]
}
```

### Managing Users

```hcl
resource "workos_user" "admin" {
  email          = "admin@example.com"
  first_name     = "Admin"
  last_name      = "User"
  email_verified = true
}

resource "workos_organization_membership" "admin" {
  user_id         = workos_user.admin.id
  organization_id = workos_organization.example.id
  role_slug       = "admin"
}
```

### Managing Roles

```hcl
resource "workos_organization_role" "billing_admin" {
  organization_id = workos_organization.example.id
  slug            = "org-billing-admin"
  name            = "Billing Admin"
  description     = "Can manage billing and invoices"
}

resource "workos_organization_role" "viewer" {
  organization_id = workos_organization.example.id
  slug            = "org-viewer"
  name            = "Viewer"
}
```

### Data Sources

```hcl
# Look up organization by ID
data "workos_organization" "by_id" {
  id = "org_01HXYZ..."
}

# Look up organization by domain
data "workos_organization" "by_domain" {
  domain = "acme.com"
}

# Look up user by email
data "workos_user" "john" {
  email = "john@example.com"
}

# Look up directory user
data "workos_directory_user" "synced" {
  directory_id = workos_directory.okta.id
  email        = "employee@acme.com"
}

# Look up organization role by slug
data "workos_organization_role" "billing" {
  organization_id = workos_organization.example.id
  slug            = "org-billing-admin"
}
```

## Resources

| Resource | Description |
|----------|-------------|
| `workos_organization` | Manages WorkOS organizations |
| `workos_connection` | Manages SSO connections (SAML, OAuth, OIDC) |
| `workos_directory` | Manages Directory Sync directories |
| `workos_webhook` | Manages webhook endpoints |
| `workos_user` | Manages AuthKit users |
| `workos_organization_membership` | Manages user-organization memberships |
| `workos_organization_role` | Manages organization authorization roles |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `workos_organization` | Retrieves organization by ID or domain |
| `workos_connection` | Retrieves SSO connection by ID or org/type |
| `workos_directory` | Retrieves directory by ID or organization |
| `workos_directory_user` | Retrieves directory-synced user |
| `workos_directory_group` | Retrieves directory-synced group |
| `workos_user` | Retrieves AuthKit user by ID or email |
| `workos_organization_role` | Retrieves organization role by slug or ID |

## Development

### Building

```bash
make build
```

### Testing

```bash
# Unit tests
make test

# Acceptance tests (requires WorkOS API credentials)
export WORKOS_API_KEY="sk_test_..."
export WORKOS_CLIENT_ID="client_..."
make testacc
```

### Generating Documentation

```bash
make docs
```

### Linting

```bash
make lint
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes (`git commit -am 'Add new feature'`)
4. Push to the branch (`git push origin feature/my-feature`)
5. Open a Pull Request

### Commit Message Format

```
feat(resource): add new attribute support
fix(organization): handle domain validation
docs(readme): update installation instructions
test(connection): add acceptance tests
```

## License

MPL-2.0 - See [LICENSE](LICENSE) for details.

## Support

- [Documentation](https://registry.terraform.io/providers/osodevops/workos/latest/docs)
- [GitHub Issues](https://github.com/osodevops/terraform-provider-workos/issues)
- [WorkOS Documentation](https://workos.com/docs)
