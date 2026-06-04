// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
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
	err := c.Get(ctx, "/organizations/"+url.PathEscape(id), &org)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return &org, nil
}

// UpdateOrganization updates an existing organization
func (c *Client) UpdateOrganization(ctx context.Context, id string, req *OrganizationUpdateRequest) (*Organization, error) {
	var org Organization
	err := c.Put(ctx, "/organizations/"+url.PathEscape(id), req, &org)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}
	return &org, nil
}

// DeleteOrganization deletes an organization by ID
func (c *Client) DeleteOrganization(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/organizations/"+url.PathEscape(id))
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	return nil
}

// ListOrganizations lists all organizations
func (c *Client) ListOrganizations(ctx context.Context) (*OrganizationListResponse, error) {
	var all OrganizationListResponse
	params := url.Values{}
	applyDefaultPagination(params)

	for {
		var page OrganizationListResponse
		err := c.Get(ctx, pathWithQuery("/organizations", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list organizations: %w", err)
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

// GetOrganizationByExternalID retrieves an organization by external ID
func (c *Client) GetOrganizationByExternalID(ctx context.Context, externalID string) (*Organization, error) {
	var org Organization
	err := c.Get(ctx, "/organizations/external_id/"+url.PathEscape(externalID), &org)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization by external ID: %w", err)
	}
	return &org, nil
}

// ListOrganizationsByDomain returns all organizations matching a given domain
func (c *Client) ListOrganizationsByDomain(ctx context.Context, domain string) ([]Organization, error) {
	var orgs []Organization
	params := url.Values{}
	params.Set("domains", domain)
	applyDefaultPagination(params)

	for {
		var page OrganizationListResponse
		err := c.Get(ctx, pathWithQuery("/organizations", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to search organizations by domain: %w", err)
		}

		orgs = append(orgs, page.Data...)
		if page.ListMetadata.After == "" {
			break
		}
		params.Set("after", page.ListMetadata.After)
	}

	return orgs, nil
}

// GetOrganizationByDomain finds a single organization by domain.
// Returns an error if no organizations or multiple organizations match the domain.
func (c *Client) GetOrganizationByDomain(ctx context.Context, domain string) (*Organization, error) {
	orgs, err := c.ListOrganizationsByDomain(ctx, domain)
	if err != nil {
		return nil, err
	}

	if len(orgs) == 0 {
		return nil, &APIError{
			StatusCode: 404,
			Message:    fmt.Sprintf("no organization found with domain: %s", domain),
		}
	}

	if len(orgs) > 1 {
		orgIDs := make([]string, len(orgs))
		for i, org := range orgs {
			orgIDs[i] = fmt.Sprintf("%s (%s)", org.ID, org.Name)
		}
		return nil, fmt.Errorf(
			"ambiguous domain lookup: domain %q is associated with %d organizations: [%s]. "+
				"Use the organization ID to look up a specific organization instead",
			domain, len(orgs), strings.Join(orgIDs, ", "),
		)
	}

	return &orgs[0], nil
}
