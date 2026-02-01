# Release Guide for WorkOS Terraform Provider

This guide walks you through publishing the WorkOS Terraform Provider to the Terraform Registry.

## Prerequisites

1. GitHub repository at `github.com/osodevops/terraform-provider-workos`
2. GPG key for signing releases
3. Terraform Registry account linked to your GitHub organization

---

## Step 1: Set Up GPG Key

### Generate a new GPG key (if needed)

```bash
# Generate a new GPG key
gpg --full-generate-key

# Choose:
# - RSA and RSA (default)
# - 4096 bits
# - Key does not expire (or set expiration)
# - Your name and email

# List your keys to find the key ID
gpg --list-secret-keys --keyid-format=long

# Output looks like:
# sec   rsa4096/XXXXXXXXXXXXXXXX 2024-01-01 [SC]
#       YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY
# uid                 [ultimate] Your Name <your@email.com>
```

### Export the GPG key

```bash
# Export the private key (for GitHub secrets)
gpg --armor --export-secret-keys XXXXXXXXXXXXXXXX > private-key.asc

# Export the public key (for Terraform Registry)
gpg --armor --export XXXXXXXXXXXXXXXX > public-key.asc
```

---

## Step 2: Configure GitHub Repository Secrets

Go to your repository **Settings > Secrets and variables > Actions** and add:

| Secret Name | Value |
|-------------|-------|
| `GPG_PRIVATE_KEY` | Contents of `private-key.asc` |
| `GPG_PASSPHRASE` | Your GPG key passphrase |

---

## Step 3: Register with Terraform Registry

1. Go to [registry.terraform.io](https://registry.terraform.io)
2. Sign in with your GitHub account
3. Click **Publish** > **Provider**
4. Select `osodevops/terraform-provider-workos`
5. Add your **GPG public key** (contents of `public-key.asc`)

The registry will:
- Verify your repository structure
- Check for required files (`terraform-registry-manifest.json`, `docs/`)
- Set up webhook for automatic publishing on new releases

---

## Step 4: Push to GitHub

```bash
# Add remote (if not already configured)
git remote add origin git@github.com:osodevops/terraform-provider-workos.git

# Push main branch
git push -u origin main
```

---

## Step 5: Create and Push Release Tag

```bash
# Create the v1.0.0 tag
git tag -a v1.0.0 -m "Release v1.0.0 - Initial release"

# Push the tag
git push origin v1.0.0
```

This will trigger the GitHub Actions release workflow which:
1. Builds binaries for all platforms (Linux, macOS, Windows)
2. Creates SHA256 checksums
3. Signs the checksums with your GPG key
4. Creates a GitHub release with all artifacts

---

## Step 6: Verify Release

### Check GitHub Release

Go to your repository's **Releases** page and verify:
- [ ] All platform binaries are present
- [ ] `terraform-provider-workos_1.0.0_SHA256SUMS` exists
- [ ] `terraform-provider-workos_1.0.0_SHA256SUMS.sig` exists

### Check Terraform Registry

After a few minutes, verify at:
`https://registry.terraform.io/providers/osodevops/workos/latest`

- [ ] Version 1.0.0 is listed
- [ ] Documentation is displayed correctly
- [ ] All resources and data sources are listed

### Test Installation

```hcl
terraform {
  required_providers {
    workos = {
      source  = "osodevops/workos"
      version = "1.0.0"
    }
  }
}

provider "workos" {
  api_key = var.workos_api_key
}
```

```bash
terraform init
# Should download the provider from the registry
```

---

## Release Checklist

Before each release:

- [ ] All tests pass (`make test`)
- [ ] Documentation is up to date (`make docs`)
- [ ] CHANGELOG.md is updated
- [ ] Version number follows semver
- [ ] No sensitive data in codebase

---

## Troubleshooting

### Release workflow fails

1. Check GitHub Actions logs
2. Verify GPG secrets are correctly set
3. Ensure tag follows `v*` pattern

### Provider not appearing in registry

1. Verify repository is public
2. Check terraform-registry-manifest.json exists
3. Ensure GPG public key is registered with Terraform Registry
4. Check registry webhook is configured

### Signature verification fails

1. Ensure GPG key hasn't expired
2. Verify the same key is used in GitHub secrets and Terraform Registry
3. Check passphrase is correct

---

## Versioning Guidelines

| Change Type | Version Bump | Example |
|-------------|--------------|---------|
| Bug fixes | Patch | 1.0.0 → 1.0.1 |
| New features (backward compatible) | Minor | 1.0.0 → 1.1.0 |
| Breaking changes | Major | 1.0.0 → 2.0.0 |

### What constitutes a breaking change:
- Removing a resource or data source
- Removing or renaming an attribute
- Changing attribute type
- Changing default behavior
