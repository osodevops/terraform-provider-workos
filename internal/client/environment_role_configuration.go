// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
)

const environmentRoleConfigurationQuery = `query roles($id: ID!) {
  rolesForEnvironment(id: $id) { roles { id slug resourceTypeId } }
  roleConfigForEnvironmentV2(id: $id) { id defaultRole }
}`
const updateEnvironmentRoleConfigurationMutation = `mutation updateRoleConfig($input: UpdateRoleConfigInput!) {
  updateRoleConfig(input: $input) { __typename }
}`

type EnvironmentRoleConfiguration struct {
	EnvironmentID   string
	RoleConfigID    string
	DefaultRoleSlug string
}

type environmentRoleConfigurationData struct {
	Roles *struct {
		Roles []struct {
			ID             string  `json:"id"`
			Slug           string  `json:"slug"`
			ResourceTypeID *string `json:"resourceTypeId"`
		} `json:"roles"`
	} `json:"rolesForEnvironment"`
	Config *struct {
		ID          string `json:"id"`
		DefaultRole string `json:"defaultRole"`
	} `json:"roleConfigForEnvironmentV2"`
}

func (c *Client) readEnvironmentRoleConfiguration(ctx context.Context, environmentID string) (*EnvironmentRoleConfiguration, map[string]string, error) {
	if err := c.validateAccountTarget(ctx, environmentID); err != nil {
		return nil, nil, err
	}
	var data environmentRoleConfigurationData
	if err := c.accountRequest(ctx, environmentID, environmentRoleConfigurationQuery, map[string]any{"id": environmentID}, &data); err != nil {
		return nil, nil, err
	}
	if data.Config == nil || data.Config.ID == "" || data.Roles == nil || data.Roles.Roles == nil {
		return nil, nil, fmt.Errorf("account API returned incomplete environment role configuration")
	}
	ids := make(map[string]bool)
	roles := make(map[string]string)
	defaultSlug := ""
	for _, role := range data.Roles.Roles {
		if role.ResourceTypeID != nil && *role.ResourceTypeID != "" {
			continue
		}
		if role.ID == "" || role.Slug == "" || roles[role.Slug] != "" || ids[role.ID] {
			return nil, nil, fmt.Errorf("environment roles contain missing or ambiguous identifiers")
		}
		roles[role.Slug] = role.ID
		ids[role.ID] = true
		if data.Config.DefaultRole == role.ID {
			defaultSlug = role.Slug
		}
	}
	// The catalog types defaultRole as String, without ID/slug documentation.
	// Require the current value to join an actual role ID before writing an ID.
	if defaultSlug == "" {
		return nil, nil, fmt.Errorf("cannot verify defaultRole as an environment role ID from current configuration; no role mutation is allowed")
	}
	return &EnvironmentRoleConfiguration{EnvironmentID: environmentID, RoleConfigID: data.Config.ID, DefaultRoleSlug: defaultSlug}, roles, nil
}

func (c *Client) ReadEnvironmentRoleConfiguration(ctx context.Context, environmentID string) (*EnvironmentRoleConfiguration, error) {
	config, _, err := c.readEnvironmentRoleConfiguration(ctx, environmentID)
	return config, err
}

func (c *Client) ApplyEnvironmentRoleConfiguration(ctx context.Context, environmentID, defaultRoleSlug string) (*EnvironmentRoleConfiguration, error) {
	if defaultRoleSlug == "" {
		return nil, fmt.Errorf("default_role_slug is required")
	}
	observed, roles, err := c.readEnvironmentRoleConfiguration(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	roleID := roles[defaultRoleSlug]
	if roleID == "" {
		return observed, fmt.Errorf("default_role_slug must identify an existing unscoped role in the exact configured environment")
	}
	if observed.DefaultRoleSlug == defaultRoleSlug {
		return observed, nil
	}
	// Revalidate the team and environment immediately before the write.
	if err = c.validateAccountTarget(ctx, environmentID); err != nil {
		return observed, err
	}
	err = c.runConfigurationMutation(ctx, environmentID, configurationMutation{
		query: updateEnvironmentRoleConfigurationMutation, field: "updateRoleConfig", success: "RoleConfigUpdated",
		variables: map[string]any{"input": map[string]any{"roleConfigId": observed.RoleConfigID, "defaultRole": roleID}},
	})
	latest, readErr := c.ReadEnvironmentRoleConfiguration(ctx, environmentID)
	if latest != nil {
		observed = latest
	}
	if err != nil {
		return observed, err
	}
	if readErr != nil {
		return observed, fmt.Errorf("role configuration may have changed but readback failed: %w; refresh and retry", readErr)
	}
	if observed.DefaultRoleSlug != defaultRoleSlug {
		return observed, fmt.Errorf("default role readback differs from the intended role; refresh and retry")
	}
	return observed, nil
}
