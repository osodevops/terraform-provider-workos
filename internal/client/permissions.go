// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
)

// CreatePermission creates a new permission
func (c *Client) CreatePermission(ctx context.Context, req *PermissionCreateRequest) (*Permission, error) {
	var perm Permission
	err := c.Post(ctx, "/authorization/permissions", req, &perm)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}
	return &perm, nil
}

// GetPermission retrieves a permission by slug
func (c *Client) GetPermission(ctx context.Context, slug string) (*Permission, error) {
	var perm Permission
	err := c.Get(ctx, fmt.Sprintf("/authorization/permissions/%s", slug), &perm)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}
	return &perm, nil
}

// UpdatePermission updates an existing permission
func (c *Client) UpdatePermission(ctx context.Context, slug string, req *PermissionUpdateRequest) (*Permission, error) {
	var perm Permission
	err := c.Patch(ctx, fmt.Sprintf("/authorization/permissions/%s", slug), req, &perm)
	if err != nil {
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}
	return &perm, nil
}

// DeletePermission deletes a permission by slug
func (c *Client) DeletePermission(ctx context.Context, slug string) error {
	err := c.Delete(ctx, fmt.Sprintf("/authorization/permissions/%s", slug))
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}
	return nil
}
