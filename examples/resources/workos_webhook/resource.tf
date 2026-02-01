# Basic webhook configuration
resource "workos_webhook" "main" {
  url     = "https://api.example.com/webhooks/workos"
  secret  = var.webhook_secret
  enabled = true

  events = [
    "user.created",
    "user.updated",
    "user.deleted",
  ]
}

# Webhook for directory sync events
resource "workos_webhook" "dsync" {
  url     = "https://api.example.com/webhooks/dsync"
  secret  = var.dsync_webhook_secret
  enabled = true

  events = [
    "dsync.activated",
    "dsync.deleted",
    "dsync.user.created",
    "dsync.user.updated",
    "dsync.user.deleted",
    "dsync.group.created",
    "dsync.group.updated",
    "dsync.group.deleted",
  ]
}

# Webhook for SSO connection events
resource "workos_webhook" "sso" {
  url     = "https://api.example.com/webhooks/sso"
  secret  = var.sso_webhook_secret
  enabled = true

  events = [
    "connection.activated",
    "connection.deactivated",
    "connection.deleted",
    "authentication.sso_succeeded",
    "authentication.sso_failed",
  ]
}

# Comprehensive webhook for all authentication events
resource "workos_webhook" "auth" {
  url     = "https://api.example.com/webhooks/auth"
  secret  = var.auth_webhook_secret
  enabled = true

  events = [
    "authentication.email_verification_succeeded",
    "authentication.magic_auth_succeeded",
    "authentication.magic_auth_failed",
    "authentication.mfa_succeeded",
    "authentication.oauth_succeeded",
    "authentication.oauth_failed",
    "authentication.password_succeeded",
    "authentication.password_failed",
    "authentication.sso_succeeded",
    "authentication.sso_failed",
    "session.created",
  ]
}

# Variables for webhook secrets (use sensitive input)
variable "webhook_secret" {
  type        = string
  description = "Secret for the main webhook"
  sensitive   = true
}

variable "dsync_webhook_secret" {
  type        = string
  description = "Secret for the directory sync webhook"
  sensitive   = true
}

variable "sso_webhook_secret" {
  type        = string
  description = "Secret for the SSO webhook"
  sensitive   = true
}

variable "auth_webhook_secret" {
  type        = string
  description = "Secret for the authentication webhook"
  sensitive   = true
}

# Outputs
output "main_webhook_id" {
  value       = workos_webhook.main.id
  description = "The ID of the main webhook"
}
