// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/url"
)

// CreateOrganizationRole creates a new organization role
func (c *Client) CreateOrganizationRole(ctx context.Context, orgID string, req *OrganizationRoleCreateRequest) (*OrganizationRole, error) {
	var role OrganizationRole
	err := c.Post(ctx, fmt.Sprintf("/authorization/organizations/%s/roles", url.PathEscape(orgID)), req, &role)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization role: %w", err)
	}
	return &role, nil
}

// GetOrganizationRole retrieves an organization role by slug
func (c *Client) GetOrganizationRole(ctx context.Context, orgID, slug string) (*OrganizationRole, error) {
	var role OrganizationRole
	err := c.Get(ctx, fmt.Sprintf("/authorization/organizations/%s/roles/%s", url.PathEscape(orgID), url.PathEscape(slug)), &role)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization role: %w", err)
	}
	return &role, nil
}

// UpdateOrganizationRole updates an existing organization role
func (c *Client) UpdateOrganizationRole(ctx context.Context, orgID, slug string, req *OrganizationRoleUpdateRequest) (*OrganizationRole, error) {
	var role OrganizationRole
	err := c.Patch(ctx, fmt.Sprintf("/authorization/organizations/%s/roles/%s", url.PathEscape(orgID), url.PathEscape(slug)), req, &role)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization role: %w", err)
	}
	return &role, nil
}

// DeleteOrganizationRole deletes an organization role by slug
func (c *Client) DeleteOrganizationRole(ctx context.Context, orgID, slug string) error {
	err := c.Delete(ctx, fmt.Sprintf("/authorization/organizations/%s/roles/%s", url.PathEscape(orgID), url.PathEscape(slug)))
	if err != nil {
		return fmt.Errorf("failed to delete organization role: %w", err)
	}
	return nil
}

// ListOrganizationRoles lists all roles for an organization
func (c *Client) ListOrganizationRoles(ctx context.Context, orgID string) (*OrganizationRoleListResponse, error) {
	var all OrganizationRoleListResponse
	params := url.Values{}
	applyDefaultPagination(params)
	path := fmt.Sprintf("/authorization/organizations/%s/roles", url.PathEscape(orgID))

	for {
		var page OrganizationRoleListResponse
		err := c.Get(ctx, pathWithQuery(path, params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list organization roles: %w", err)
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

// GetOrganizationRoleByID finds an organization role by its ID
func (c *Client) GetOrganizationRoleByID(ctx context.Context, orgID, roleID string) (*OrganizationRole, error) {
	resp, err := c.ListOrganizationRoles(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, role := range resp.Data {
		if role.ID == roleID {
			return &role, nil
		}
	}

	return nil, &APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("no organization role found with ID: %s", roleID),
	}
}
