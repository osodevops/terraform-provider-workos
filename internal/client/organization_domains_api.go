package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// OrganizationDomain represents a WorkOS organization domain.
type OrganizationDomain struct {
	ID                   string    `json:"id"`
	Object               string    `json:"object"`
	OrganizationID       string    `json:"organization_id"`
	Domain               string    `json:"domain"`
	State                *string   `json:"state,omitempty"`
	VerificationPrefix   *string   `json:"verification_prefix,omitempty"`
	VerificationToken    *string   `json:"verification_token,omitempty"`
	VerificationStrategy *string   `json:"verification_strategy,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// OrganizationDomainCreateRequest represents the request to add a domain to an organization.
type OrganizationDomainCreateRequest struct {
	Domain         string `json:"domain"`
	OrganizationID string `json:"organization_id"`
}

// CreateOrganizationDomain creates a new organization domain.
func (c *Client) CreateOrganizationDomain(ctx context.Context, req *OrganizationDomainCreateRequest) (*OrganizationDomain, error) {
	var domain OrganizationDomain
	err := c.Post(ctx, "/organization_domains", req, &domain)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization domain: %w", err)
	}
	return &domain, nil
}

// GetOrganizationDomain retrieves an organization domain by ID.
func (c *Client) GetOrganizationDomain(ctx context.Context, id string) (*OrganizationDomain, error) {
	var domain OrganizationDomain
	err := c.Get(ctx, "/organization_domains/"+url.PathEscape(id), &domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization domain: %w", err)
	}
	return &domain, nil
}

// DeleteOrganizationDomain deletes an organization domain by ID.
func (c *Client) DeleteOrganizationDomain(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/organization_domains/"+url.PathEscape(id))
	if err != nil {
		return fmt.Errorf("failed to delete organization domain: %w", err)
	}
	return nil
}

// VerifyOrganizationDomain starts verification for an organization domain.
func (c *Client) VerifyOrganizationDomain(ctx context.Context, id string) (*OrganizationDomain, error) {
	var domain OrganizationDomain
	err := c.Post(ctx, "/organization_domains/"+url.PathEscape(id)+"/verify", nil, &domain)
	if err != nil {
		return nil, fmt.Errorf("failed to verify organization domain: %w", err)
	}
	return &domain, nil
}
