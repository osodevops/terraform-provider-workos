# Product Requirements Document: WorkOS Terraform Provider

**Version:** 1.0  
**Date:** January 31, 2026  
**Status:** Draft  
**Author:** Engineering Team

---

## Executive Summary

This document outlines the requirements for developing a production-ready Terraform provider for the WorkOS API. WorkOS provides enterprise-ready features including AuthKit (user management), Single Sign-On (SSO), Directory Sync (SCIM), Fine-Grained Authorization (FGA), Audit Logs, and Organizations management. This provider will enable infrastructure-as-code (IaC) management of WorkOS resources, allowing customers to version-control their enterprise authentication and authorization infrastructure alongside their application infrastructure.

### Goals
- Provide comprehensive Terraform resource coverage for all major WorkOS API endpoints
- Enable declarative management of WorkOS configurations through IaC
- Follow HashiCorp's Terraform Plugin Framework best practices
- Deliver production-ready quality with comprehensive testing and documentation
- Publish to the public Terraform Registry for community use

### Non-Goals
- Replacing the WorkOS Dashboard UI (this is a complementary tool)
- Managing runtime authentication flows (focus is on configuration management)
- Supporting legacy Terraform Plugin SDKv2 (new development uses Framework)

---

## Background & Context

### Current State
The only existing WorkOS Terraform provider (github.com/Vellanci/terraform-provider-workos) is unmaintained and incomplete. The WorkOS ecosystem has grown significantly with new features like FGA, expanded AuthKit capabilities, and API Keys management that are not covered by existing tooling.

### Why Terraform?
1. **Version Control**: WorkOS configurations can be stored in Git alongside application infrastructure
2. **Multi-Environment Management**: Staging and production environments managed through code
3. **Safety & Audit**: Changes require review and approval before application
4. **Repeatability**: Consistent configuration across multiple WorkOS organizations
5. **Compliance**: Infrastructure changes tracked and auditable

### WorkOS API Overview
The WorkOS API is a REST API that provides programmatic access to:
- **AuthKit/User Management** - User authentication, sessions, MFA, magic auth
- **Organizations** - Multi-tenant organization structure and management
- **SSO (Single Sign-On)** - SAML, OAuth connections to identity providers
- **Directory Sync** - SCIM provisioning from corporate directories (Okta, Entra ID, Google Workspace)
- **Fine-Grained Authorization (FGA)** - Zanzibar-based access control
- **Audit Logs** - Exportable audit trail of user actions
- **Events API** - Event-driven data synchronization
- **Webhooks** - Real-time notifications
- **API Keys** - Programmatic access management for end users

**Rate Limits:**
- General: 6,000 requests per 60 seconds per IP
- AuthKit Reads: 1,000 requests per 10 seconds
- AuthKit Writes: 500 requests per 10 seconds
- Specialized endpoints have additional rate limits (detailed in API docs)

---

## Technical Architecture

### Framework Selection
**Use Terraform Plugin Framework (not SDKv2)**

**Rationale:**
- HashiCorp's recommended SDK for new providers
- Better type safety with Go generics
- Improved data access and handling
- Native support for dynamic types
- Enhanced validation capabilities
- Future-proof (SDKv2 feature development has stopped)
- Protocol version 6 support

**Key Dependencies:**
```go
github.com/hashicorp/terraform-plugin-framework v1.5+
github.com/hashicorp/terraform-plugin-go v0.20+
github.com/hashicorp/terraform-plugin-testing v1.6+ (for acceptance tests)
```

### Provider Structure

```
terraform-provider-workos/
├── main.go                           # Entry point
├── internal/
│   ├── provider/
│   │   ├── provider.go               # Provider configuration
│   │   ├── provider_test.go
│   │   ├── resource_*.go             # Resource implementations
│   │   ├── resource_*_test.go        # Acceptance tests
│   │   ├── data_source_*.go          # Data source implementations
│   │   └── data_source_*_test.go
│   └── client/
│       ├── client.go                 # WorkOS API client wrapper
│       └── models.go                 # Shared data models
├── examples/
│   ├── provider/
│   ├── resources/
│   └── data-sources/
├── docs/                             # Auto-generated documentation
├── templates/                        # Documentation templates
├── .goreleaser.yml                   # Release automation
├── .github/
│   └── workflows/
│       ├── test.yml                  # CI testing
│       └── release.yml               # Automated releases
├── go.mod
├── go.sum
├── LICENSE (MPL-2.0)
└── README.md
```

### Provider Configuration

```hcl
terraform {
  required_providers {
    workos = {
      source  = "oso-sh/workos"
      version = "~> 1.0"
    }
  }
}

provider "workos" {
  api_key    = var.workos_api_key  # Or env: WORKOS_API_KEY
  client_id  = var.workos_client_id # Or env: WORKOS_CLIENT_ID (optional, for some resources)
  base_url   = "https://api.workos.com" # Optional, for testing
}
```

**Provider Schema:**
- `api_key` (String, Sensitive, Required) - WorkOS API key (sk_*)
- `client_id` (String, Optional) - WorkOS Client ID for certain operations
- `base_url` (String, Optional) - API base URL, defaults to production endpoint

---

## Resource Specifications

### Priority 1: Core Resources (MVP)

#### 1. `workos_organization`
**Purpose:** Manage WorkOS Organizations (fundamental multi-tenant unit)

**Schema:**
```hcl
resource "workos_organization" "main" {
  name    = "Acme Corporation"
  domains = ["acme.com", "acmecorp.com"]
  
  # Optional
  allow_profiles_outside_organization = false
  
  # Computed
  id         = "org_01H..."
  object     = "organization"
  created_at = "2026-01-31T..."
  updated_at = "2026-01-31T..."
}
```

**API Mapping:**
- Create: `POST /organizations`
- Read: `GET /organizations/{id}`
- Update: `PUT /organizations/{id}`
- Delete: `DELETE /organizations/{id}`
- Import: By organization ID

**Attributes:**
- `name` (String, Required) - Organization name
- `domains` (Set of String, Optional) - Verified domains
- `allow_profiles_outside_organization` (Boolean, Optional, Default: false)
- `id` (String, Computed) - Organization ID
- `object` (String, Computed) - Always "organization"
- `created_at` (String, Computed) - RFC3339 timestamp
- `updated_at` (String, Computed) - RFC3339 timestamp

**Validation:**
- Name must be 1-255 characters
- Domains must be valid domain format

#### 2. `workos_connection` (SSO)
**Purpose:** Manage SSO connections to identity providers

**Schema:**
```hcl
resource "workos_connection" "okta_sso" {
  organization_id = workos_organization.main.id
  connection_type = "OktaSAML"
  name            = "Okta SSO"
  
  # Type-specific configuration
  okta_saml {
    idp_entity_id     = "http://www.okta.com/..."
    idp_sso_url       = "https://example.okta.com/app/..."
    idp_certificate   = file("${path.module}/okta-cert.pem")
  }
  
  # Computed
  id     = "conn_01H..."
  state  = "active"
  status = "linked"
}
```

**API Mapping:**
- Create: `POST /connections`
- Read: `GET /connections/{id}`
- Update: `PUT /connections/{id}`
- Delete: `DELETE /connections/{id}`
- List: `GET /connections` (for import/data source)

**Supported Connection Types:**
- SAML: `OktaSAML`, `AzureSAML`, `GoogleSAML`, `GenericSAML`
- OAuth: `GoogleOAuth`, `MicrosoftOAuth`
- OIDC: `GenericOIDC`

**Attributes:**
- `organization_id` (String, Required, ForceNew)
- `connection_type` (String, Required, ForceNew)
- `name` (String, Optional)
- `state` (String, Computed) - "active", "inactive", "validating"
- `status` (String, Computed) - "linked", "unlinked"
- Type-specific blocks (okta_saml, azure_saml, google_saml, generic_saml, etc.)

#### 3. `workos_directory`
**Purpose:** Manage Directory Sync connections

**Schema:**
```hcl
resource "workos_directory" "okta_scim" {
  organization_id = workos_organization.main.id
  name            = "Okta Directory"
  type            = "okta scim v2.0"
  
  # Computed
  id                = "directory_01H..."
  state             = "linked"
  bearer_token      = "..." # Sensitive
  endpoint          = "https://api.workos.com/scim/v2/directories/..."
  
  # Webhooks for sync events
  webhook_url = "https://api.example.com/webhooks/workos"
}
```

**API Mapping:**
- Create: `POST /directories`
- Read: `GET /directories/{id}`
- Update: `PUT /directories/{id}`
- Delete: `DELETE /directories/{id}`

**Supported Directory Types:**
- `azure scim v2.0`, `okta scim v2.0`, `generic scim v2.0`
- `google workspace`, `workday`, `rippling`, etc.

**Attributes:**
- `organization_id` (String, Required, ForceNew)
- `name` (String, Required)
- `type` (String, Required, ForceNew)
- `webhook_url` (String, Optional) - Webhook endpoint for events
- `state` (String, Computed) - "linked", "unlinked", "invalid_credentials"
- `bearer_token` (String, Computed, Sensitive) - SCIM bearer token
- `endpoint` (String, Computed) - SCIM endpoint URL

#### 4. `workos_webhook`
**Purpose:** Manage webhook endpoints for event subscriptions

**Schema:**
```hcl
resource "workos_webhook" "main" {
  url         = "https://api.example.com/webhooks/workos"
  secret      = var.webhook_secret
  enabled     = true
  
  events = [
    "dsync.activated",
    "dsync.deleted",
    "dsync.user.created",
    "dsync.user.updated",
    "dsync.user.deleted",
    "dsync.group.created",
    "dsync.group.updated",
    "dsync.group.deleted",
    "user.created",
    "user.updated",
    "user.deleted",
    "organization.created",
    "organization.updated",
    "organization.deleted",
  ]
}
```

**API Mapping:**
- Create: `POST /webhooks`
- Read: `GET /webhooks/{id}`
- Update: `PUT /webhooks/{id}`
- Delete: `DELETE /webhooks/{id}`

**Attributes:**
- `url` (String, Required) - HTTPS webhook endpoint
- `secret` (String, Required, Sensitive) - Webhook signature secret
- `enabled` (Boolean, Optional, Default: true)
- `events` (Set of String, Required) - Event types to subscribe to

**Validation:**
- URL must use HTTPS
- URL must be publicly accessible
- Events must be valid WorkOS event types

### Priority 2: AuthKit Resources

#### 5. `workos_user`
**Purpose:** Manage WorkOS users (AuthKit)

**Schema:**
```hcl
resource "workos_user" "admin" {
  email      = "[email protected]"
  first_name = "Jane"
  last_name  = "Admin"
  
  email_verified = true
  
  # Optional organization membership
  organization_memberships {
    organization_id = workos_organization.main.id
    role_slug       = "admin"
  }
  
  # Computed
  id         = "user_01H..."
  created_at = "2026-01-31T..."
  updated_at = "2026-01-31T..."
}
```

**API Mapping:**
- Create: `POST /user_management/users`
- Read: `GET /user_management/users/{id}`
- Update: `PUT /user_management/users/{id}`
- Delete: `DELETE /user_management/users/{id}`

**Attributes:**
- `email` (String, Required) - User email address
- `first_name` (String, Optional)
- `last_name` (String, Optional)
- `email_verified` (Boolean, Optional, Default: false)
- `password` (String, Optional, Sensitive, WriteOnly) - Set on creation only
- `password_hash` (String, Optional, Sensitive, WriteOnly) - Import existing hash
- `organization_memberships` (Block List, Optional) - Organization associations

**Lifecycle Considerations:**
- Password is write-only and never read back
- Deleting user revokes all sessions
- Email changes trigger verification flow

#### 6. `workos_organization_membership`
**Purpose:** Manage user membership in organizations

**Schema:**
```hcl
resource "workos_organization_membership" "jane_admin" {
  user_id         = workos_user.admin.id
  organization_id = workos_organization.main.id
  role_slug       = "admin"
}
```

**API Mapping:**
- Create: `POST /user_management/organization_memberships`
- Read: `GET /user_management/organization_memberships/{id}`
- Update: `PUT /user_management/organization_memberships/{id}`
- Delete: `DELETE /user_management/organization_memberships/{id}`

### Priority 3: Advanced Features

#### 7. `workos_fga_resource`
**Purpose:** Define FGA resource types and hierarchies

**Schema:**
```hcl
resource "workos_fga_resource" "project" {
  type        = "project"
  description = "Software project resource"
  
  # Resource hierarchy
  parent_type = "organization"
  
  # Supported roles
  roles = ["owner", "editor", "viewer"]
}
```

#### 8. `workos_audit_log_export`
**Purpose:** Configure audit log exports

**Schema:**
```hcl
resource "workos_audit_log_export" "s3" {
  organization_id = workos_organization.main.id
  
  destination {
    type   = "s3"
    bucket = "acme-audit-logs"
    region = "us-east-1"
    prefix = "workos/"
  }
  
  filters = ["user.login", "user.logout", "*.delete"]
  enabled = true
}
```

---

## Data Source Specifications

Data sources enable reading existing WorkOS resources without managing them.

### 1. `workos_organization`
```hcl
data "workos_organization" "main" {
  id = "org_01H..."
  # OR
  domain = "acme.com"
}
```

### 2. `workos_connection`
```hcl
data "workos_connection" "sso" {
  id = "conn_01H..."
  # OR
  organization_id  = workos_organization.main.id
  connection_type = "OktaSAML"
}
```

### 3. `workos_directory`
```hcl
data "workos_directory" "main" {
  id = "directory_01H..."
  # OR
  organization_id = workos_organization.main.id
}
```

### 4. `workos_directory_user`
```hcl
data "workos_directory_user" "john" {
  directory_id = data.workos_directory.main.id
  email        = "[email protected]"
}
```

### 5. `workos_directory_group`
```hcl
data "workos_directory_group" "engineering" {
  directory_id = data.workos_directory.main.id
  name         = "Engineering"
}
```

### 6. `workos_user`
```hcl
data "workos_user" "by_email" {
  email = "[email protected]"
}
```

---

## Testing Strategy

### Unit Tests
**Coverage Target:** >80% of business logic

**Focus Areas:**
- Schema validation logic
- Data transformation functions
- Error handling
- Client wrapper functions

**Example:**
```go
func TestOrganizationResourceSchema(t *testing.T) {
    // Test schema validation
}

func TestOrganizationCreate_ValidInput(t *testing.T) {
    // Test create logic with mock client
}
```

### Acceptance Tests
**Coverage Target:** Every resource CRUD operation + import

**Requirements:**
- Test against real WorkOS staging API
- Use `TF_ACC=1` environment variable gate
- Clean up resources after each test
- Run in parallel where safe
- Use unique names (timestamp/random suffix)

**Environment Variables:**
```bash
export TF_ACC=1
export WORKOS_API_KEY="sk_test_..."
export WORKOS_CLIENT_ID="client_test_..."
```

**Test Structure:**
```go
func TestAccOrganizationResource_Basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Create and Read
            {
                Config: testAccOrganizationResourceConfig("test-org"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("workos_organization.test", "name", "test-org"),
                    resource.TestCheckResourceAttrSet("workos_organization.test", "id"),
                ),
            },
            // ImportState
            {
                ResourceName:      "workos_organization.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
            // Update
            {
                Config: testAccOrganizationResourceConfig("test-org-updated"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("workos_organization.test", "name", "test-org-updated"),
                ),
            },
        },
    })
}
```

**Test Coverage Matrix:**

| Resource | Create | Read | Update | Delete | Import | Disappears |
|----------|--------|------|--------|--------|--------|------------|
| organization | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| connection | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| directory | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| webhook | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| user | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| membership | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

**Parallel Testing:**
- Tests for different resources can run in parallel
- Tests within same organization must run serially
- Use `t.Parallel()` appropriately

### Integration Tests
**Purpose:** Test provider against staging environment

**Scenarios:**
- Multi-organization setup
- Organization with SSO + Directory Sync
- User lifecycle (create → assign → update → delete)
- Import existing infrastructure

### Manual Testing Checklist
- [ ] Provider installation via Terraform Registry
- [ ] Provider installation via local build
- [ ] All example configurations work
- [ ] Documentation renders correctly on Registry
- [ ] Import works for all resources
- [ ] Rate limiting is handled gracefully
- [ ] Error messages are clear and actionable

---

## Documentation Requirements

### Provider Documentation
Generate using `terraform-plugin-docs` tool from templates.

**Required Files:**
```
docs/
├── index.md                    # Provider overview
├── resources/
│   ├── organization.md
│   ├── connection.md
│   ├── directory.md
│   ├── webhook.md
│   ├── user.md
│   └── organization_membership.md
└── data-sources/
    ├── organization.md
    ├── connection.md
    ├── directory.md
    ├── directory_user.md
    ├── directory_group.md
    └── user.md
```

**Documentation Standards:**
- Include complete working examples
- Document all attributes with types
- Explain computed vs. required attributes
- Provide import syntax examples
- Link to WorkOS documentation where appropriate
- Include common error scenarios and solutions

### Example Configurations

**Basic Setup:**
```hcl
# examples/resources/workos_organization/resource.tf
resource "workos_organization" "example" {
  name    = "Acme Corporation"
  domains = ["acme.com"]
}
```

**Complete SSO Setup:**
```hcl
# examples/complete-sso/main.tf
resource "workos_organization" "main" {
  name    = "Acme Corporation"
  domains = ["acme.com"]
}

resource "workos_connection" "okta" {
  organization_id = workos_organization.main.id
  connection_type = "OktaSAML"
  name            = "Okta SSO"
  
  okta_saml {
    idp_entity_id   = var.okta_entity_id
    idp_sso_url     = var.okta_sso_url
    idp_certificate = file("${path.module}/okta-cert.pem")
  }
}

output "sso_login_url" {
  value = "https://auth.example.com/sso/${workos_connection.okta.id}"
}
```

**Directory Sync Setup:**
```hcl
# examples/directory-sync/main.tf
resource "workos_organization" "main" {
  name = "Acme Corporation"
}

resource "workos_directory" "okta" {
  organization_id = workos_organization.main.id
  name            = "Okta Directory"
  type            = "okta scim v2.0"
  webhook_url     = "https://api.example.com/webhooks/workos"
}

# Use data source to read synced users
data "workos_directory_user" "all_users" {
  directory_id = workos_directory.okta.id
}

output "scim_endpoint" {
  value     = workos_directory.okta.endpoint
  sensitive = false
}

output "scim_bearer_token" {
  value     = workos_directory.okta.bearer_token
  sensitive = true
}
```

### README.md
```markdown
# WorkOS Terraform Provider

Terraform provider for managing WorkOS resources.

## Requirements
- Terraform >= 1.0
- Go >= 1.21 (for development)

## Installation
[Terraform Registry installation instructions]

## Usage
[Quick start example]

## Documentation
[Link to Registry docs]

## Development
[Build and test instructions]

## Contributing
[Contribution guidelines]

## License
MPL-2.0
```

---

## Release & Publishing Strategy

### Version Management
**Semantic Versioning (SemVer):** MAJOR.MINOR.PATCH

**Version 1.0.0 Criteria:**
- All Priority 1 resources implemented
- Comprehensive test coverage (>80%)
- Documentation complete
- Published to Terraform Registry
- Production-ready stability

**Pre-1.0 Versions:**
- v0.1.0 - Provider scaffolding + organization resource
- v0.2.0 - SSO connections
- v0.3.0 - Directory Sync
- v0.4.0 - Webhooks + AuthKit
- v0.5.0 - Data sources
- v1.0.0 - Production ready

### Release Process

**Automated with GoReleaser:**

**.goreleaser.yml:**
```yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - env:
      - CGO_ENABLED=0
    mod_timestamp: '{{ .CommitTimestamp }}'
    flags:
      - -trimpath
    ldflags:
      - '-s -w -X main.version={{.Version}} -X main.commit={{.Commit}}'
    goos:
      - freebsd
      - windows
      - linux
      - darwin
    goarch:
      - amd64
      - '386'
      - arm
      - arm64
    ignore:
      - goos: darwin
        goarch: '386'
    binary: '{{ .ProjectName }}_v{{ .Version }}'

archives:
  - format: zip
    name_template: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}'

checksum:
  name_template: '{{ .ProjectName }}_{{ .Version }}_SHA256SUMS'
  algorithm: sha256

signs:
  - artifacts: checksum
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"

release:
  draft: false

changelog:
  skip: false
```

**GitHub Actions Workflow:**

**.github/workflows/release.yml:**
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
      
      - name: Import GPG key
        uses: crazy-max/ghaction-import-gpg@v6
        id: import_gpg
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
          passphrase: ${{ secrets.PASSPHRASE }}
      
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
```

**Release Steps:**
1. Update CHANGELOG.md with release notes
2. Create and push version tag: `git tag v1.0.0 && git push origin v1.0.0`
3. GitHub Actions automatically builds, signs, and releases
4. Terraform Registry automatically detects new release

### Terraform Registry Publishing

**Prerequisites:**
1. GitHub repository named `terraform-provider-workos`
2. Repository must be public
3. GPG key for signing releases
4. Repository structure follows Terraform conventions

**Registry Manifest:**

**terraform-registry-manifest.json:**
```json
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["6.0"]
  }
}
```

**Publishing Steps:**
1. Sign into Terraform Registry with GitHub account
2. Grant registry permissions to repository
3. Add GPG public key to registry
4. Click "Publish" → "Provider"
5. Select organization and repository
6. Registry validates and publishes

**Registry Requirements:**
- README.md in repository root
- LICENSE file (MPL-2.0)
- Documentation in docs/ directory
- At least one GitHub release with artifacts
- Signed release artifacts
- Proper naming: `terraform-provider-workos`

---

## CI/CD Pipeline

### GitHub Actions Workflows

**.github/workflows/test.yml:**
```yaml
name: Tests

on:
  pull_request:
  push:
    branches:
      - main

permissions:
  contents: read

jobs:
  build:
    name: Build
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
          cache: true
      - run: go build -v .

  generate:
    name: Generate
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
          cache: true
      - run: go generate ./...
      - name: git diff
        run: |
          git diff --compact-summary --exit-code || \
            (echo; echo "Unexpected difference in directories after code generation. Run 'go generate ./...' command and commit."; exit 1)

  test:
    name: Unit Tests
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
          cache: true
      - run: go test -v -cover -timeout=120s -parallel=4 ./...

  acceptance:
    name: Acceptance Tests
    runs-on: ubuntu-latest
    timeout-minutes: 30
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
          cache: true
      - run: go test -v -cover -timeout=20m -parallel=4 ./...
        env:
          TF_ACC: '1'
          WORKOS_API_KEY: ${{ secrets.WORKOS_STAGING_API_KEY }}
          WORKOS_CLIENT_ID: ${{ secrets.WORKOS_STAGING_CLIENT_ID }}

  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
          cache: true
      - uses: golangci/golangci-lint-action@v3
        with:
          version: latest
```

### Quality Gates
- All tests must pass
- Code coverage >80%
- No linting errors
- Documentation generation succeeds
- No uncommitted generated files

---

## Error Handling & Edge Cases

### Rate Limiting
**Strategy:** Implement exponential backoff with jitter

```go
func (c *Client) doRequestWithRetry(req *http.Request) (*http.Response, error) {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        resp, err := c.httpClient.Do(req)
        if err != nil {
            return nil, err
        }
        
        if resp.StatusCode == 429 {
            retryAfter := resp.Header.Get("Retry-After")
            if retryAfter != "" {
                // Parse and wait
            } else {
                // Exponential backoff
                time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
            }
            continue
        }
        
        return resp, nil
    }
    return nil, fmt.Errorf("max retries exceeded")
}
```

### API Error Handling

**Error Response Format:**
```json
{
  "message": "Error description",
  "code": "error_code",
  "errors": [
    {
      "field": "field_name",
      "code": "validation_error"
    }
  ]
}
```

**Provider Error Mapping:**
- 400 Bad Request → Validation error with details
- 401 Unauthorized → API key configuration error
- 403 Forbidden → Permission error (staging vs production)
- 404 Not Found → Resource not found (may have been deleted)
- 422 Unprocessable Entity → Detailed validation errors
- 429 Too Many Requests → Rate limit (retry with backoff)
- 5xx Server Error → Temporary error (retry)

**User-Friendly Error Messages:**
```go
resp.Diagnostics.AddError(
    "Unable to Create Organization",
    fmt.Sprintf(
        "An error occurred while creating the organization.\n\n"+
        "WorkOS API Error: %s\n"+
        "Please verify your API key and organization settings.",
        err.Error(),
    ),
)
```

### State Management Edge Cases

**1. Resource Deleted Outside Terraform**
- Implement graceful handling in Read
- Return nil to trigger recreation
- Don't error on 404

**2. Concurrent Modifications**
- Use ETags if available
- Detect and report conflicts
- Suggest terraform refresh

**3. Partial Failures**
- Atomic operations where possible
- Document multi-step operations
- Provide rollback guidance

### Import Edge Cases

**Handling missing optional fields:**
```go
func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
    
    // Note: Some computed fields may not match exactly after import
    resp.Diagnostics.AddWarning(
        "Import Limitations",
        "Some optional fields may not be populated after import. "+
        "Run 'terraform plan' to see any differences.",
    )
}
```

---

## Security Considerations

### API Key Management
- **Never log API keys** (even at debug level)
- Mark as `Sensitive: true` in schema
- Store in environment variables or secure vaults
- Rotate keys regularly
- Use staging keys for testing

### Secret Handling
- Webhook secrets marked sensitive
- SCIM bearer tokens marked sensitive
- Password fields write-only
- No secrets in state file debug output

### Network Security
- All API calls over HTTPS
- Validate TLS certificates
- Support custom CA certificates if needed
- Webhook URLs must be HTTPS

### Permissions
- Document required API key permissions
- Validate staging vs production environment
- Handle 403 errors with helpful messages

---

## Monitoring & Observability

### Logging Strategy
**Use Terraform's built-in logging:**

```go
import (
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    tflog.Debug(ctx, "Creating organization", map[string]any{
        "name": plan.Name.ValueString(),
    })
    
    // API call
    
    tflog.Info(ctx, "Successfully created organization", map[string]any{
        "id": org.ID,
    })
}
```

**Log Levels:**
- `TRACE`: Detailed request/response bodies (not including secrets)
- `DEBUG`: Operation flow and decisions
- `INFO`: Major lifecycle events (create, update, delete)
- `WARN`: Recoverable issues
- `ERROR`: Unrecoverable errors

**Environment Variables:**
```bash
export TF_LOG=DEBUG
export TF_LOG_PATH=terraform.log
```

### Metrics
**Track via CI/CD:**
- Test execution time
- Test success rate
- Code coverage percentage
- Build time
- Release frequency

---

## Development Workflow

### Local Development Setup

**Prerequisites:**
```bash
# Install Go 1.21+
go version

# Install Terraform
terraform version

# Clone repository
git clone https://github.com/YOUR_ORG/terraform-provider-workos
cd terraform-provider-workos

# Install dependencies
go mod download
```

**Build Provider:**
```bash
go build -o terraform-provider-workos
```

**Local Testing:**
```bash
# Create dev_overrides configuration
cat > ~/.terraformrc << EOF
provider_installation {
  dev_overrides {
    "oso-sh/workos" = "/path/to/terraform-provider-workos"
  }
  
  direct {}
}
EOF

# Test with local provider
cd examples/complete-sso
terraform init
terraform plan
```

**Run Tests:**
```bash
# Unit tests
go test -v ./...

# Acceptance tests (requires API key)
export WORKOS_API_KEY="sk_test_..."
export WORKOS_CLIENT_ID="client_test_..."
export TF_ACC=1
go test -v -timeout=20m ./...

# Single test
go test -v -run TestAccOrganizationResource_Basic ./internal/provider
```

### Code Generation
```bash
# Generate documentation
go generate ./...

# Format code
gofmt -w .
go mod tidy

# Run linter
golangci-lint run
```

### Git Workflow

**Branch Strategy:**
- `main` - stable, production-ready
- `feature/*` - new features
- `bugfix/*` - bug fixes
- `release/*` - release preparation

**Commit Messages:**
```
feat(organization): add domain verification support
fix(connection): handle certificate validation errors
docs(directory): update SCIM configuration examples
test(webhook): add acceptance tests for event filtering
```

**Pull Request Checklist:**
- [ ] Tests pass locally
- [ ] Documentation updated
- [ ] Changelog entry added
- [ ] Examples provided for new resources
- [ ] Acceptance tests included

---

## Maintenance & Support

### Versioning Policy
- **Patch releases** (x.x.N): Bug fixes, documentation updates
- **Minor releases** (x.N.0): New resources, new features (backwards compatible)
- **Major releases** (N.0.0): Breaking changes

### Deprecation Policy
1. Announce deprecation in release notes
2. Maintain deprecated feature for 2 minor versions
3. Log warnings when deprecated features are used
4. Remove in next major version

### Breaking Changes
- Avoid in minor/patch releases
- Document migration path
- Provide state upgraders where possible
- Announce in advance (major version)

### Support Channels
1. GitHub Issues - bug reports, feature requests
2. GitHub Discussions - questions, usage help
3. Documentation - comprehensive guides
4. Examples - working code samples

### Issue Triage
**Labels:**
- `bug` - Something isn't working
- `enhancement` - New feature request
- `documentation` - Documentation improvements
- `good first issue` - Good for newcomers
- `help wanted` - Extra attention needed
- `priority: high` - Critical issues
- `wontfix` - Not planned

**Response SLA:**
- Critical bugs: 24 hours
- Regular bugs: 7 days
- Feature requests: 14 days
- Questions: 3 days

---

## Success Metrics

### MVP Success Criteria
- [ ] Provider published to Terraform Registry
- [ ] All Priority 1 resources implemented
- [ ] Test coverage >80%
- [ ] Documentation complete with examples
- [ ] 5+ community users/stars on GitHub
- [ ] Zero critical bugs in first month

### Long-Term Metrics
- **Adoption:** Downloads per month from Registry
- **Quality:** Bug report rate
- **Community:** GitHub stars, forks, contributors
- **Reliability:** Acceptance test pass rate
- **Coverage:** % of WorkOS API covered

### User Satisfaction
- Survey users after 3 months
- Track GitHub issue sentiment
- Monitor feature request volume
- Measure documentation effectiveness (bounce rate, time on page)

---

## Risks & Mitigations

### Risk 1: WorkOS API Changes
**Impact:** High  
**Probability:** Medium  
**Mitigation:**
- Subscribe to WorkOS changelog
- Version API client separately
- Comprehensive acceptance tests catch breaking changes
- Maintain backwards compatibility where possible

### Risk 2: Rate Limiting in CI
**Impact:** Medium  
**Probability:** Medium  
**Mitigation:**
- Implement exponential backoff
- Run acceptance tests serially or with rate limit awareness
- Use staging environment with higher limits if available
- Cache read operations where appropriate

### Risk 3: Complex State Management
**Impact:** High  
**Probability:** Medium  
**Mitigation:**
- Follow Terraform best practices religiously
- Comprehensive acceptance tests including import
- Clear documentation on state management
- Regular testing against real infrastructure

### Risk 4: Community Adoption
**Impact:** Medium  
**Probability:** Low  
**Mitigation:**
- High-quality documentation with examples
- Responsive issue triage
- Promote in WorkOS community/documentation
- Present at HashiCorp events/webinars

---

## Future Enhancements (Post-MVP)

### Phase 2 Features
- Multi-factor authentication resource configuration
- Advanced FGA policy management
- Magic Auth configuration
- Password policies
- Session management
- IP allowlisting

### Phase 3 Features
- Terraform Cloud integration examples
- Automated compliance policies
- Advanced data sources (aggregations, filters)
- Bulk import tooling
- Migration helper from other providers

### Ecosystem Integration
- Terraform Cloud sentinel policies
- Atlantis integration examples
- Terragrunt patterns
- Terraform CDK support

---

## References

### WorkOS Documentation
- API Reference: https://workos.com/docs/reference
- AuthKit: https://workos.com/docs/authkit
- SSO: https://workos.com/docs/sso
- Directory Sync: https://workos.com/docs/directory-sync
- FGA: https://workos.com/docs/fga
- Events API: https://workos.com/docs/events
- Rate Limits: https://workos.com/docs/reference#rate-limits

### Terraform Documentation
- Plugin Framework: https://developer.hashicorp.com/terraform/plugin/framework
- Provider Development: https://developer.hashicorp.com/terraform/plugin/best-practices
- Testing: https://developer.hashicorp.com/terraform/plugin/sdkv2/testing/acceptance-tests
- Registry Publishing: https://developer.hashicorp.com/terraform/registry/providers/publishing

### Example Providers (Reference)
- AWS Provider: https://github.com/hashicorp/terraform-provider-aws
- Auth0 Provider: https://github.com/auth0/terraform-provider-auth0
- Okta Provider: https://github.com/okta/terraform-provider-okta

### Tools
- GoReleaser: https://goreleaser.com/
- terraform-plugin-docs: https://github.com/hashicorp/terraform-plugin-docs
- golangci-lint: https://golangci-lint.run/

---

## Appendix A: Resource Priority Matrix

| Resource | Priority | MVP | Complexity | Dependencies |
|----------|----------|-----|------------|--------------|
| organization | P1 | ✓ | Low | None |
| connection (SSO) | P1 | ✓ | Medium | Organization |
| directory | P1 | ✓ | Medium | Organization |
| webhook | P1 | ✓ | Low | None |
| user | P2 | ✓ | Medium | None |
| organization_membership | P2 | ✓ | Low | User, Organization |
| fga_resource | P3 | ✗ | High | Organization |
| fga_assignment | P3 | ✗ | High | FGA Resource, User |
| audit_log_export | P3 | ✗ | Medium | Organization |
| mfa_config | P3 | ✗ | Medium | Organization |

---

## Appendix B: API Endpoint Coverage

### Organizations
- ✓ `POST /organizations` - Create
- ✓ `GET /organizations/{id}` - Read
- ✓ `PUT /organizations/{id}` - Update
- ✓ `DELETE /organizations/{id}` - Delete
- ✓ `GET /organizations` - List

### SSO Connections
- ✓ `POST /connections` - Create
- ✓ `GET /connections/{id}` - Read
- ✓ `PUT /connections/{id}` - Update
- ✓ `DELETE /connections/{id}` - Delete
- ✓ `GET /connections` - List

### Directory Sync
- ✓ `POST /directories` - Create
- ✓ `GET /directories/{id}` - Read
- ✓ `PUT /directories/{id}` - Update
- ✓ `DELETE /directories/{id}` - Delete
- ✓ `GET /directory_users` - List Users (data source)
- ✓ `GET /directory_groups` - List Groups (data source)

### User Management (AuthKit)
- ✓ `POST /user_management/users` - Create
- ✓ `GET /user_management/users/{id}` - Read
- ✓ `PUT /user_management/users/{id}` - Update
- ✓ `DELETE /user_management/users/{id}` - Delete
- ✓ `POST /user_management/organization_memberships` - Create Membership
- ✓ `GET /user_management/organization_memberships/{id}` - Read Membership
- ✓ `DELETE /user_management/organization_memberships/{id}` - Delete Membership

### Webhooks
- ✓ `POST /webhooks` - Create
- ✓ `GET /webhooks/{id}` - Read
- ✓ `PUT /webhooks/{id}` - Update
- ✓ `DELETE /webhooks/{id}` - Delete

### Fine-Grained Authorization
- ○ `POST /authorization/resources` - Create Resource
- ○ `GET /authorization/resources/{id}` - Read Resource
- ○ `POST /authorization/check` - Check Access (data source)

### Audit Logs
- ○ `POST /audit_logs/exports` - Create Export
- ○ `GET /audit_logs/exports/{id}` - Read Export

**Legend:** ✓ MVP, ○ Post-MVP

---

## Appendix C: Testing Checklist

### Pre-Release Checklist
- [ ] All acceptance tests pass
- [ ] Unit test coverage >80%
- [ ] Documentation generated successfully
- [ ] All examples tested manually
- [ ] Import works for all resources
- [ ] goreleaser build succeeds
- [ ] GPG signing configured
- [ ] CHANGELOG.md updated
- [ ] Version bumped appropriately

### Manual Testing Scenarios
- [ ] Fresh install from Terraform Registry
- [ ] Create organization with all fields
- [ ] Add SSO connection
- [ ] Configure Directory Sync
- [ ] Create and manage users
- [ ] Import existing infrastructure
- [ ] Update all resource types
- [ ] Delete resources in correct order
- [ ] Handle API errors gracefully
- [ ] Rate limiting works correctly

### Edge Case Testing
- [ ] Empty/null optional fields
- [ ] Very long strings (within limits)
- [ ] Special characters in names
- [ ] Concurrent resource creation
- [ ] Network timeouts
- [ ] Malformed API responses
- [ ] 404 on deleted resources
- [ ] Invalid credentials
- [ ] Cross-organization dependencies

---

## Changelog Template

```markdown
## [Unreleased]

## [1.0.0] - 2026-XX-XX
### Added
- Initial release
- Resources: organization, connection, directory, webhook, user, organization_membership
- Data sources: organization, connection, directory, directory_user, directory_group, user
- Comprehensive documentation and examples
- Published to Terraform Registry

### Changed
- N/A

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- N/A
```

---

**Document End**

**Next Steps:**
1. Review and approve PRD
2. Set up repository structure
3. Implement provider scaffolding
4. Begin Priority 1 resource development
5. Set up CI/CD pipeline
6. Write comprehensive tests
7. Generate documentation
8. Publish to Terraform Registry

**Estimated Timeline:** 8-12 weeks for MVP (Priority 1 + Priority 2 resources)

**Team Size:** 1-2 developers