// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/url"
)

// CreateOrganization creates a new organization
func (c *Client) CreateOrganization(ctx context.Context, req *OrganizationCreateRequest) (*Organization, error) {
	var org Organization
	err := c.Post(ctx, "/organizations", req, &org)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}
	return &org, nil
}

// GetOrganization retrieves an organization by ID
func (c *Client) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	var org Organization
	err := c.Get(ctx, "/organizations/"+id, &org)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return &org, nil
}

// UpdateOrganization updates an existing organization
func (c *Client) UpdateOrganization(ctx context.Context, id string, req *OrganizationUpdateRequest) (*Organization, error) {
	var org Organization
	err := c.Put(ctx, "/organizations/"+id, req, &org)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}
	return &org, nil
}

// DeleteOrganization deletes an organization by ID
func (c *Client) DeleteOrganization(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/organizations/"+id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	return nil
}

// ListOrganizations lists all organizations
func (c *Client) ListOrganizations(ctx context.Context) (*OrganizationListResponse, error) {
	var resp OrganizationListResponse
	err := c.Get(ctx, "/organizations", &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	return &resp, nil
}

// GetOrganizationByDomain finds an organization by domain
func (c *Client) GetOrganizationByDomain(ctx context.Context, domain string) (*Organization, error) {
	params := url.Values{}
	params.Set("domains", domain)

	var resp OrganizationListResponse
	err := c.Get(ctx, "/organizations?"+params.Encode(), &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to search organizations by domain: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, &APIError{
			StatusCode: 404,
			Message:    fmt.Sprintf("no organization found with domain: %s", domain),
		}
	}

	return &resp.Data[0], nil
}
