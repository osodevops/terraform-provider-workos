package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// ConnectApplication represents a WorkOS Connect application.
type ConnectApplication struct {
	ID                       string                          `json:"id"`
	Object                   string                          `json:"object"`
	ClientID                 string                          `json:"client_id"`
	Name                     string                          `json:"name"`
	Description              *string                         `json:"description,omitempty"`
	ApplicationType          *string                         `json:"application_type,omitempty"`
	OrganizationID           *string                         `json:"organization_id,omitempty"`
	Scopes                   []string                        `json:"scopes,omitempty"`
	RedirectURIs             []ConnectApplicationRedirectURI `json:"redirect_uris,omitempty"`
	UsesPKCE                 *bool                           `json:"uses_pkce,omitempty"`
	IsFirstParty             *bool                           `json:"is_first_party,omitempty"`
	WasDynamicallyRegistered *bool                           `json:"was_dynamically_registered,omitempty"`
	CreatedAt                time.Time                       `json:"created_at"`
	UpdatedAt                time.Time                       `json:"updated_at"`
}

// ConnectApplicationRedirectURI represents a Connect application redirect URI.
type ConnectApplicationRedirectURI struct {
	URI     string `json:"uri"`
	Default bool   `json:"default"`
}

// ConnectApplicationRedirectURIInput represents a redirect URI input.
type ConnectApplicationRedirectURIInput struct {
	URI     string `json:"uri"`
	Default *bool  `json:"default,omitempty"`
}

// ConnectApplicationCreateRequest represents the request to create a Connect application.
type ConnectApplicationCreateRequest struct {
	ApplicationType string                               `json:"application_type"`
	Name            string                               `json:"name"`
	IsFirstParty    *bool                                `json:"is_first_party,omitempty"`
	Description     string                               `json:"description,omitempty"`
	Scopes          []string                             `json:"scopes,omitempty"`
	RedirectURIs    []ConnectApplicationRedirectURIInput `json:"redirect_uris,omitempty"`
	UsesPKCE        *bool                                `json:"uses_pkce,omitempty"`
	OrganizationID  string                               `json:"organization_id,omitempty"`
}

// ConnectApplicationUpdateRequest represents the request to update a Connect application.
type ConnectApplicationUpdateRequest struct {
	Name         string                               `json:"name,omitempty"`
	Description  string                               `json:"description,omitempty"`
	Scopes       []string                             `json:"scopes,omitempty"`
	RedirectURIs []ConnectApplicationRedirectURIInput `json:"redirect_uris,omitempty"`
}

// ConnectApplicationListResponse represents the response from listing Connect applications.
type ConnectApplicationListResponse struct {
	Data         []ConnectApplication `json:"data"`
	ListMetadata ListMetadata         `json:"list_metadata"`
}

// CreateConnectApplication creates a Connect application.
func (c *Client) CreateConnectApplication(ctx context.Context, req *ConnectApplicationCreateRequest) (*ConnectApplication, error) {
	var app ConnectApplication
	err := c.Post(ctx, "/connect/applications", req, &app)
	if err != nil {
		return nil, fmt.Errorf("failed to create connect application: %w", err)
	}
	return &app, nil
}

// GetConnectApplication retrieves a Connect application by ID or client ID.
func (c *Client) GetConnectApplication(ctx context.Context, id string) (*ConnectApplication, error) {
	var app ConnectApplication
	err := c.Get(ctx, "/connect/applications/"+url.PathEscape(id), &app)
	if err != nil {
		return nil, fmt.Errorf("failed to get connect application: %w", err)
	}
	return &app, nil
}

// UpdateConnectApplication updates a Connect application.
func (c *Client) UpdateConnectApplication(ctx context.Context, id string, req *ConnectApplicationUpdateRequest) (*ConnectApplication, error) {
	var app ConnectApplication
	err := c.Put(ctx, "/connect/applications/"+url.PathEscape(id), req, &app)
	if err != nil {
		return nil, fmt.Errorf("failed to update connect application: %w", err)
	}
	return &app, nil
}

// DeleteConnectApplication deletes a Connect application.
func (c *Client) DeleteConnectApplication(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/connect/applications/"+url.PathEscape(id))
	if err != nil {
		return fmt.Errorf("failed to delete connect application: %w", err)
	}
	return nil
}

// ListConnectApplications lists Connect applications with optional organization filtering.
func (c *Client) ListConnectApplications(ctx context.Context, organizationID string) (*ConnectApplicationListResponse, error) {
	var all ConnectApplicationListResponse
	params := url.Values{}
	if organizationID != "" {
		params.Set("organization_id", organizationID)
	}
	applyDefaultPagination(params)

	for {
		var page ConnectApplicationListResponse
		err := c.Get(ctx, pathWithQuery("/connect/applications", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list connect applications: %w", err)
		}

		all.Data = append(all.Data, page.Data...)
		all.ListMetadata = page.ListMetadata
		if page.ListMetadata.After == "" {
			break
		}
		params.Set("after", page.ListMetadata.After)
	}

	return &all, nil
}
