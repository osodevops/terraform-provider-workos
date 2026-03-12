// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
)

// AddOrganizationRolePermission adds a permission to an organization role
func (c *Client) AddOrganizationRolePermission(ctx context.Context, orgID, roleSlug, permSlug string) (*OrganizationRole, error) {
	req := &AddPermissionRequest{
		Slug: permSlug,
	}
	var role OrganizationRole
	err := c.Post(ctx, fmt.Sprintf("/authorization/organizations/%s/roles/%s/permissions", orgID, roleSlug), req, &role)
	if err != nil {
		return nil, fmt.Errorf("failed to add permission to organization role: %w", err)
	}
	return &role, nil
}

// RemoveOrganizationRolePermission removes a permission from an organization role
func (c *Client) RemoveOrganizationRolePermission(ctx context.Context, orgID, roleSlug, permSlug string) error {
	err := c.Delete(ctx, fmt.Sprintf("/authorization/organizations/%s/roles/%s/permissions/%s", orgID, roleSlug, permSlug))
	if err != nil {
		return fmt.Errorf("failed to remove permission from organization role: %w", err)
	}
	return nil
}
