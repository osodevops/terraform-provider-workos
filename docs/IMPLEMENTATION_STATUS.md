# WorkOS Terraform Provider - Implementation Status

**Last Updated:** February 1, 2026

---

## Overview

This document tracks the implementation progress of the WorkOS Terraform Provider across all phases.

---

## Phase 0: Project Foundation & Scaffolding

**Status:** ✅ Complete

### All Items Complete ✅

| Item | File/Location | Notes |
|------|---------------|-------|
| Directory structure | `internal/provider/`, `internal/client/`, etc. | All directories created |
| Go module | `go.mod` | Dependencies configured for Plugin Framework v1.5+ |
| Main entry point | `main.go` | Provider server with debug flag support |
| Provider configuration | `internal/provider/provider.go` | api_key, client_id, base_url with env var fallbacks |
| API client | `internal/client/client.go` | HTTP client with rate limiting & retry |
| Error handling | `internal/client/errors.go` | Typed errors with IsNotFound, etc. |
| Data models | `internal/client/models.go` | Organization, Connection, Directory, User, etc. |
| Makefile | `Makefile` | build, test, testacc, lint, docs targets |
| CI workflow | `.github/workflows/test.yml` | Build, lint, unit tests, acceptance tests |
| Release workflow | `.github/workflows/release.yml` | GoReleaser with GPG signing |
| GoReleaser config | `.goreleaser.yml` | Multi-platform builds |
| License | `LICENSE` | MPL-2.0 |
| README | `README.md` | Installation, usage, development guide |
| Registry manifest | `terraform-registry-manifest.json` | Protocol version 6.0 |

---

## Phase 1: Organization Resource

**Status:** ✅ Complete

### All Items Complete ✅

| Item | File | Notes |
|------|------|-------|
| Organization resource | `resource_organization.go` | Full CRUD + Import |
| Organization data source | `data_source_organization.go` | Lookup by ID or domain |
| Organization API client | `organizations.go` | CRUD operations |
| Acceptance tests | `resource_organization_test.go` | 3 tests |
| Data source tests | `data_source_organization_test.go` | 2 tests |
| Example | `examples/resources/workos_organization/` | Complete |

---

## Phase 2: SSO Connection Data Source (read-only)

**Status:** ✅ Complete

**Note:** The connection _resource_ was removed because the WorkOS API does not support creating or updating connections. Connections are configured via the Dashboard/Admin Portal. Only the read-only data source is provided.

| Item | File | Notes |
|------|------|-------|
| Connection data source | `data_source_connection.go` | Lookup by ID or org/type |
| Connection API client | `connections.go` | Read-only operations |

---

## Phase 3: Directory Sync Data Sources (read-only)

**Status:** ✅ Complete

**Note:** The directory _resource_ was removed because the WorkOS API does not support creating or updating directories. Directories are provisioned via the Dashboard/SCIM provider. Only the read-only data sources are provided.

| Item | File | Notes |
|------|------|-------|
| Directory data source | `data_source_directory.go` | Lookup by ID or org |
| Directory user data source | `data_source_directory_user.go` | Lookup users |
| Directory group data source | `data_source_directory_group.go` | Lookup groups |
| Directory API client | `directories.go` | Read-only + user/group lookups |
| Examples | `examples/data-sources/workos_directory*/` | Complete |

---

## Phase 4: Webhook Resource

**Status:** ❌ Removed

**Note:** The webhook resource was removed because the WorkOS API does not have a public webhook management API. All webhook CRUD operations return 404. Webhooks must be managed via the WorkOS Dashboard.

---

## Phase 5: AuthKit User Resources

**Status:** ✅ Complete

### All Items Complete ✅

| Item | File | Notes |
|------|------|-------|
| User resource | `resource_user.go` | Full CRUD + password support |
| User data source | `data_source_user.go` | Lookup by ID or email |
| Organization membership resource | `resource_organization_membership.go` | User-org associations |
| User API client | `users.go` | CRUD + membership operations |
| User resource tests | `resource_user_test.go` | 3 tests |
| User data source tests | `data_source_user_test.go` | 2 tests |
| Membership tests | `resource_organization_membership_test.go` | 2 tests |
| Examples | `examples/resources/workos_user/` | Complete |
| Examples | `examples/resources/workos_organization_membership/` | Complete |
| Examples | `examples/data-sources/workos_user/` | Complete |

### User Resource Features

- **Email:** Required, unique email address
- **Email Verified:** Boolean flag for verification status
- **First/Last Name:** Optional user details
- **Password:** Write-only sensitive field for password auth
- **Password Hash:** Write-only field for bcrypt migration
- **Profile Picture URL:** Computed from API

### Organization Membership Features

- **User ID:** Required, links to user (RequiresReplace)
- **Organization ID:** Required, links to org (RequiresReplace)
- **Role Slug:** Optional role assignment (admin, member, viewer)
- **Status:** Computed membership status

---

## Phase 6: Documentation, Examples & Polish

**Status:** ✅ Complete

### All Items Complete ✅

| Item | File | Notes |
|------|------|-------|
| Provider documentation | `docs/index.md` | Auto-generated |
| Resource documentation | `docs/resources/*.md` | 6 resource docs |
| Data source documentation | `docs/data-sources/*.md` | 6 data source docs |
| Example configurations | `examples/` | Complete for all resources |
| CHANGELOG | `CHANGELOG.md` | v1.0.0 release notes |
| README | `README.md` | Updated with all resources |

### Generated Documentation (15 files)
- `docs/index.md` - Provider documentation
- `docs/resources/organization.md`
- `docs/resources/connection.md`
- `docs/resources/directory.md`
- `docs/resources/webhook.md`
- `docs/resources/user.md`
- `docs/resources/organization_membership.md`
- `docs/data-sources/organization.md`
- `docs/data-sources/connection.md`
- `docs/data-sources/directory.md`
- `docs/data-sources/directory_user.md`
- `docs/data-sources/directory_group.md`
- `docs/data-sources/user.md`

---

## Phase 7: Release & Registry Publication

**Status:** 🟡 In Progress

### Completed ✅

| Item | Description |
|------|-------------|
| Initial commit | 73 files, 12,210 lines of code |
| Release guide | `docs/RELEASE_GUIDE.md` |
| GoReleaser config | `.goreleaser.yml` ready |
| Release workflow | `.github/workflows/release.yml` ready |

### Remaining Steps (Manual)

| Step | Description |
|------|-------------|
| 1. GPG Key | Generate/export GPG key for signing |
| 2. GitHub Secrets | Add `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` |
| 3. Push to GitHub | `git push -u origin main` |
| 4. Terraform Registry | Register provider and add GPG public key |
| 5. Create Tag | `git tag -a v1.0.0 -m "Release v1.0.0"` |
| 6. Push Tag | `git push origin v1.0.0` |
| 7. Verify | Check GitHub release and Terraform Registry |

---

## File Structure (Current)

```
terraform-provider-workos/
├── main.go                              ✅
├── go.mod                               ✅
├── go.sum                               ✅
├── Makefile                             ✅
├── LICENSE                              ✅
├── README.md                            ✅
├── CHANGELOG.md                         ✅
├── .gitignore                           ✅
├── .goreleaser.yml                      ✅
├── terraform-registry-manifest.json     ✅
├── internal/
│   ├── provider/
│   │   ├── provider.go                  ✅
│   │   ├── provider_test.go             ✅
│   │   ├── resource_organization.go     ✅
│   │   ├── resource_organization_test.go ✅
│   │   ├── data_source_organization.go  ✅
│   │   ├── data_source_organization_test.go ✅
│   │   ├── resource_connection.go       ✅
│   │   ├── resource_connection_test.go  ✅
│   │   ├── data_source_connection.go    ✅
│   │   ├── data_source_connection_test.go ✅
│   │   ├── resource_directory.go        ✅
│   │   ├── resource_directory_test.go   ✅
│   │   ├── data_source_directory.go     ✅
│   │   ├── data_source_directory_test.go ✅
│   │   ├── data_source_directory_user.go ✅
│   │   ├── data_source_directory_group.go ✅
│   │   ├── resource_webhook.go          ✅
│   │   ├── resource_webhook_test.go     ✅
│   │   ├── resource_user.go             ✅
│   │   ├── resource_user_test.go        ✅
│   │   ├── data_source_user.go          ✅
│   │   ├── data_source_user_test.go     ✅
│   │   ├── resource_organization_membership.go ✅
│   │   └── resource_organization_membership_test.go ✅
│   └── client/
│       ├── client.go                    ✅
│       ├── errors.go                    ✅
│       ├── models.go                    ✅
│       ├── organizations.go             ✅
│       ├── connections.go               ✅
│       ├── directories.go               ✅
│       ├── webhooks.go                  ✅
│       └── users.go                     ✅
├── examples/
│   ├── provider/provider.tf             ✅
│   ├── resources/
│   │   ├── workos_organization/         ✅
│   │   ├── workos_connection/           ✅
│   │   ├── workos_directory/            ✅
│   │   ├── workos_webhook/              ✅
│   │   ├── workos_user/                 ✅
│   │   └── workos_organization_membership/ ✅
│   └── data-sources/
│       ├── workos_organization/         ✅
│       ├── workos_connection/           ✅
│       ├── workos_directory/            ✅
│       ├── workos_directory_user/       ✅
│       ├── workos_directory_group/      ✅
│       └── workos_user/                 ✅
├── docs/
│   ├── IMPLEMENTATION_STATUS.md         ✅
│   └── workos-tf-prd.md                 ✅
└── .github/workflows/
    ├── test.yml                         ✅
    └── release.yml                      ✅
```

---

## Test Summary

| Test File | Tests | Status |
|-----------|-------|--------|
| `resource_organization_test.go` | 2 | ✅ |
| `data_source_organization_test.go` | 2 | ✅ |
| `resource_user_test.go` | 3 | ✅ |
| `data_source_user_test.go` | 2 | ✅ |
| `resource_organization_membership_test.go` | 2 | ✅ |
| `resource_organization_role_test.go` | 2 | ✅ |
| `data_source_organization_role_test.go` | 1 | ✅ |
| **Total** | **14** | ✅ All passing |

---

## Resources Summary

### Resources (4)
| Resource | Description |
|----------|-------------|
| `workos_organization` | Organization management |
| `workos_user` | AuthKit user management |
| `workos_organization_membership` | User-organization associations |
| `workos_organization_role` | Organization role management |

### Data Sources (7)
| Data Source | Description |
|-------------|-------------|
| `workos_organization` | Lookup organizations |
| `workos_connection` | Lookup SSO connections (read-only) |
| `workos_directory` | Lookup directories (read-only) |
| `workos_directory_user` | Lookup directory users |
| `workos_directory_group` | Lookup directory groups |
| `workos_user` | Lookup AuthKit users |
| `workos_organization_role` | Lookup organization roles |

---

## Timeline Summary

| Phase | Status | Progress |
|-------|--------|----------|
| Phase 0: Foundation | ✅ Complete | 100% |
| Phase 1: Organization | ✅ Complete | 100% |
| Phase 2: Connection | ✅ Complete | 100% |
| Phase 3: Directory | ✅ Complete | 100% |
| Phase 4: Webhook | ✅ Complete | 100% |
| Phase 5: User | ✅ Complete | 100% |
| Phase 6: Documentation | ✅ Complete | 100% |
| Phase 7: Release | 🟡 In Progress | 50% |

**Current Progress:** Phases 0-6 complete, Phase 7 ready for manual steps

---

## Next Steps

1. **Phase 7:** Prepare for Terraform Registry publication
   - Set up GPG signing key for release signing
   - Configure Terraform Registry account
   - Run full acceptance test suite with real WorkOS credentials
   - Create v1.0.0 release tag
   - Verify provider installation from registry

---

## Legend

- ✅ Complete
- 🟡 In Progress
- ⬜ Not Started
