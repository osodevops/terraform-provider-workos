// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
)

// WebhookListResponse represents the response from listing webhooks
type WebhookListResponse struct {
	Data         []Webhook    `json:"data"`
	ListMetadata ListMetadata `json:"list_metadata"`
}

// CreateWebhook creates a new webhook
func (c *Client) CreateWebhook(ctx context.Context, req *WebhookCreateRequest) (*Webhook, error) {
	var webhook Webhook
	err := c.Post(ctx, "/webhooks", req, &webhook)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}
	return &webhook, nil
}

// GetWebhook retrieves a webhook by ID
func (c *Client) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	var webhook Webhook
	err := c.Get(ctx, "/webhooks/"+id, &webhook)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	return &webhook, nil
}

// UpdateWebhook updates an existing webhook
func (c *Client) UpdateWebhook(ctx context.Context, id string, req *WebhookUpdateRequest) (*Webhook, error) {
	var webhook Webhook
	err := c.Put(ctx, "/webhooks/"+id, req, &webhook)
	if err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}
	return &webhook, nil
}

// DeleteWebhook deletes a webhook by ID
func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/webhooks/"+id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	return nil
}

// ListWebhooks lists all webhooks
func (c *Client) ListWebhooks(ctx context.Context) (*WebhookListResponse, error) {
	var resp WebhookListResponse
	err := c.Get(ctx, "/webhooks", &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	return &resp, nil
}

// KnownWebhookEvents contains the list of known WorkOS webhook event types
var KnownWebhookEvents = map[string]bool{
	// Authentication events
	"authentication.email_verification_succeeded": true,
	"authentication.magic_auth_failed":            true,
	"authentication.magic_auth_succeeded":         true,
	"authentication.mfa_succeeded":                true,
	"authentication.oauth_failed":                 true,
	"authentication.oauth_succeeded":              true,
	"authentication.password_failed":              true,
	"authentication.password_succeeded":           true,
	"authentication.sso_failed":                   true,
	"authentication.sso_succeeded":                true,

	// Connection events
	"connection.activated":   true,
	"connection.deactivated": true,
	"connection.deleted":     true,

	// Directory sync events
	"dsync.activated":      true,
	"dsync.deleted":        true,
	"dsync.group.created":  true,
	"dsync.group.deleted":  true,
	"dsync.group.updated":  true,
	"dsync.user.created":   true,
	"dsync.user.deleted":   true,
	"dsync.user.updated":   true,

	// Organization events
	"organization.created": true,
	"organization.deleted": true,
	"organization.updated": true,

	// Organization domain events
	"organization_domain.verification_failed":   true,
	"organization_domain.verified":              true,

	// Organization membership events
	"organization_membership.added":   true,
	"organization_membership.removed": true,
	"organization_membership.updated": true,

	// Role events
	"role.created": true,
	"role.deleted": true,
	"role.updated": true,

	// Session events
	"session.created": true,

	// User events
	"user.created":              true,
	"user.deleted":              true,
	"user.updated":              true,
}

// IsKnownWebhookEvent checks if an event type is known
func IsKnownWebhookEvent(event string) bool {
	return KnownWebhookEvents[event]
}
