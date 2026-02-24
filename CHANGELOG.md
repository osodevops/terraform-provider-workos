# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-02-24

### Added
- `workos_organization_role` resource - Manage organization authorization roles
- `workos_organization_role` data source - Look up organization roles by slug or ID

### Removed
- **BREAKING:** `workos_connection` resource - WorkOS API does not support creating/updating connections via API; use the Dashboard instead. The read-only data source is still available.
- **BREAKING:** `workos_directory` resource - WorkOS API does not support creating/updating directories via API; use the Dashboard instead. The read-only data source is still available.
- **BREAKING:** `workos_webhook` resource - WorkOS has no public webhook management API; use the Dashboard instead.
- **BREAKING:** `allow_profiles_outside_organization` attribute on `workos_organization` resource and data source - WorkOS API no longer accepts this parameter.

### Fixed
- `workos_user` resource: `email_verified` is now always sent on updates, preventing drift when email changes reset verification status
- `workos_organization_membership` resource: `role_slug` is preserved from plan/state when the API omits it from responses
- `workos_user` data source tests: replaced hardcoded placeholder IDs with dynamically created resources
- `workos_organization_role` resource: slug is now prefixed with `org-` per WorkOS API requirement

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
- `workos_connection` - Look up SSO connections by ID or organization/type (read-only)
- `workos_directory` - Look up directories by ID or organization (read-only)
- `workos_directory_user` - Look up directory-synced users by ID or email
- `workos_directory_group` - Look up directory-synced groups by ID or name
- `workos_user` - Look up AuthKit users by ID or email

#### Documentation
- Auto-generated documentation using terraform-plugin-docs
- Comprehensive examples for all resources and data sources
- Schema descriptions with Markdown support

### Security
- API keys marked as sensitive and never logged
- User passwords marked as sensitive (write-only)
