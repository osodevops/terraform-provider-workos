package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// ConnectApplicationSecret represents a WorkOS Connect application client secret.
// Secret is only present on create.
type ConnectApplicationSecret struct {
	ID         string     `json:"id"`
	Object     string     `json:"object"`
	SecretHint string     `json:"secret_hint"`
	Secret     string     `json:"secret,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (c *Client) CreateConnectApplicationClientSecret(ctx context.Context, applicationID string) (*ConnectApplicationSecret, error) {
	var secret ConnectApplicationSecret
	err := c.Post(ctx, "/connect/applications/"+url.PathEscape(applicationID)+"/client_secrets", nil, &secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create connect application client secret: %w", err)
	}
	return &secret, nil
}

func (c *Client) ListConnectApplicationClientSecrets(ctx context.Context, applicationID string) ([]ConnectApplicationSecret, error) {
	var secrets []ConnectApplicationSecret
	err := c.Get(ctx, "/connect/applications/"+url.PathEscape(applicationID)+"/client_secrets", &secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to list connect application client secrets: %w", err)
	}
	return secrets, nil
}

func (c *Client) DeleteConnectApplicationClientSecret(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/connect/client_secrets/"+url.PathEscape(id))
	if err != nil {
		return fmt.Errorf("failed to delete connect application client secret: %w", err)
	}
	return nil
}
