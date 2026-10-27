// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
)

// PreflightAccountConfiguration runs during provider configuration, before any
// resource can mutate either account configuration or REST environment roles.
func (c *Client) PreflightAccountConfiguration(ctx context.Context, environmentID, applicationID string) error {
	if _, err := c.ReadAuthkitConfiguration(ctx, environmentID, applicationID); err != nil {
		return err
	}
	_, accountRoles, err := c.readEnvironmentRoleConfiguration(ctx, environmentID)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		restRoles, err := c.ListEnvironmentRoles(ctx)
		if err != nil {
			return fmt.Errorf("cannot verify the REST API key's environment from its role IDs; no resources may mutate: %w", err)
		}
		matched := make(map[string]bool)
		for _, role := range restRoles.Data {
			if role.ResourceTypeSlug != "" {
				continue
			}
			if role.ID == "" || accountRoles[role.Slug] != role.ID || matched[role.Slug] {
				return fmt.Errorf("REST API key role IDs do not match the configured account environment; no resources may mutate")
			}
			matched[role.Slug] = true
		}
		if len(matched) == 0 || len(matched) != len(accountRoles) {
			return fmt.Errorf("REST API key environment cannot be proven from the complete role ID sets; no resources may mutate")
		}
	}
	c.accountEnvironmentID = environmentID
	c.accountApplicationID = applicationID
	return nil
}
