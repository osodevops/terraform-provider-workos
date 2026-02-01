# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-02-01

### Added

#### Provider
- Provider configuration with `api_key`, `client_id`, and `base_url` attributes
- Environment variable support: `WORKOS_API_KEY`, `WORKOS_CLIENT_ID`, `WORKOS_BASE_URL`
- Rate limiting with exponential backoff and Retry-After header support
- Comprehensive error handling with typed errors

#### Resources
- `workos_organization` - Manage WorkOS organizations
  - Full CRUD operations
  - Domain management
  - `allow_profiles_outside_organization` setting
  - Import support

- `workos_connection` - Manage SSO connections
  - Support for SAML, OAuth, and OIDC connection types
  - Okta, Azure AD, Google, and generic providers
  - Connection state management
  - Import support

- `workos_directory` - Manage Directory Sync directories
  - Support for Okta SCIM, Azure SCIM, and generic SCIM
  - Bearer token and endpoint configuration
  - Directory state management
  - Import support

- `workos_webhook` - Manage webhook endpoints
  - URL and secret configuration
  - Event type subscription (35+ event types)
  - Enable/disable toggle
  - Import support

- `workos_user` - Manage AuthKit users
  - Email and name management
  - Password and password hash support for authentication
  - Email verification status
  - Import support

- `workos_organization_membership` - Manage user-organization associations
  - User and organization linking
  - Role assignment support
  - Import support

#### Data Sources
- `workos_organization` - Look up organizations by ID or domain
- `workos_connection` - Look up SSO connections by ID or organization/type
- `workos_directory` - Look up directories by ID or organization
- `workos_directory_user` - Look up directory-synced users by ID or email
- `workos_directory_group` - Look up directory-synced groups by ID or name
- `workos_user` - Look up AuthKit users by ID or email

#### Documentation
- Auto-generated documentation using terraform-plugin-docs
- Comprehensive examples for all resources and data sources
- Schema descriptions with Markdown support

### Security
- API keys marked as sensitive and never logged
- Webhook secrets marked as sensitive
- User passwords marked as sensitive (write-only)
- Bearer tokens marked as sensitive

## [Unreleased]

### Added
- N/A

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
