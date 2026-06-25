// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/url"
)

// CreateEnvironmentRole creates a new environment-level role.
func (c *Client) CreateEnvironmentRole(ctx context.Context, req *EnvironmentRoleCreateRequest) (*EnvironmentRole, error) {
	var role EnvironmentRole
	err := c.Post(ctx, "/authorization/roles", req, &role)
	if err != nil {
		return nil, fmt.Errorf("failed to create environment role: %w", err)
	}
	return &role, nil
}

// GetEnvironmentRole retrieves an environment-level role by slug.
func (c *Client) GetEnvironmentRole(ctx context.Context, slug string) (*EnvironmentRole, error) {
	var role EnvironmentRole
	err := c.Get(ctx, fmt.Sprintf("/authorization/roles/%s", url.PathEscape(slug)), &role)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment role: %w", err)
	}
	return &role, nil
}

// UpdateEnvironmentRole updates an existing environment-level role.
func (c *Client) UpdateEnvironmentRole(ctx context.Context, slug string, req *EnvironmentRoleUpdateRequest) (*EnvironmentRole, error) {
	var role EnvironmentRole
	err := c.Patch(ctx, fmt.Sprintf("/authorization/roles/%s", url.PathEscape(slug)), req, &role)
	if err != nil {
		return nil, fmt.Errorf("failed to update environment role: %w", err)
	}
	return &role, nil
}

// ListEnvironmentRoles lists all environment-level roles.
func (c *Client) ListEnvironmentRoles(ctx context.Context) (*EnvironmentRoleListResponse, error) {
	var all EnvironmentRoleListResponse
	params := url.Values{}
	applyDefaultPagination(params)
	path := "/authorization/roles"

	for {
		var page EnvironmentRoleListResponse
		err := c.Get(ctx, pathWithQuery(path, params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list environment roles: %w", err)
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

// GetEnvironmentRoleByID finds an environment-level role by its ID.
func (c *Client) GetEnvironmentRoleByID(ctx context.Context, roleID string) (*EnvironmentRole, error) {
	resp, err := c.ListEnvironmentRoles(ctx)
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
		Message:    fmt.Sprintf("no environment role found with ID: %s", roleID),
	}
}

// AddEnvironmentRolePermission adds a permission to an environment-level role.
func (c *Client) AddEnvironmentRolePermission(ctx context.Context, roleSlug, permSlug string) (*EnvironmentRole, error) {
	req := &AddPermissionRequest{
		Slug: permSlug,
	}
	var role EnvironmentRole
	err := c.Post(ctx, fmt.Sprintf("/authorization/roles/%s/permissions", url.PathEscape(roleSlug)), req, &role)
	if err != nil {
		return nil, fmt.Errorf("failed to add permission to environment role: %w", err)
	}
	return &role, nil
}

// SetEnvironmentRolePermissions replaces all permissions on an environment-level role.
func (c *Client) SetEnvironmentRolePermissions(ctx context.Context, roleSlug string, permissions []string) (*EnvironmentRole, error) {
	req := &EnvironmentRolePermissionsRequest{
		Permissions: permissions,
	}
	var role EnvironmentRole
	err := c.Put(ctx, fmt.Sprintf("/authorization/roles/%s/permissions", url.PathEscape(roleSlug)), req, &role)
	if err != nil {
		return nil, fmt.Errorf("failed to set environment role permissions: %w", err)
	}
	return &role, nil
}
