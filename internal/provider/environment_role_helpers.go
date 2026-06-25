// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"sort"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

func environmentRolePermissionsSet(ctx context.Context, permissions []string) (types.Set, diag.Diagnostics) {
	if permissions == nil {
		permissions = []string{}
	}
	return types.SetValueFrom(ctx, types.StringType, permissions)
}

func environmentRolePermissionsSlice(ctx context.Context, permissions types.Set) ([]string, diag.Diagnostics) {
	var values []string
	diags := permissions.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return nil, diags
	}
	sort.Strings(values)
	return values, diags
}

func applyEnvironmentRoleToResourceModel(ctx context.Context, role *client.EnvironmentRole, model *EnvironmentRoleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(role.ID)
	model.Slug = types.StringValue(role.Slug)
	model.Name = types.StringValue(role.Name)
	model.Description = types.StringValue(role.Description)
	model.Type = types.StringValue(role.Type)
	if role.ResourceTypeSlug != "" {
		model.ResourceTypeSlug = types.StringValue(role.ResourceTypeSlug)
	} else {
		model.ResourceTypeSlug = types.StringNull()
	}
	model.CreatedAt = types.StringValue(role.CreatedAt.Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(role.UpdatedAt.Format(time.RFC3339))

	permissions, permissionDiags := environmentRolePermissionsSet(ctx, role.Permissions)
	diags.Append(permissionDiags...)
	if diags.HasError() {
		return diags
	}
	model.Permissions = permissions

	return diags
}

func applyEnvironmentRoleToDataSourceModel(ctx context.Context, role *client.EnvironmentRole, model *EnvironmentRoleDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(role.ID)
	model.Slug = types.StringValue(role.Slug)
	model.Name = types.StringValue(role.Name)
	model.Description = types.StringValue(role.Description)
	model.Type = types.StringValue(role.Type)
	if role.ResourceTypeSlug != "" {
		model.ResourceTypeSlug = types.StringValue(role.ResourceTypeSlug)
	} else {
		model.ResourceTypeSlug = types.StringNull()
	}
	model.CreatedAt = types.StringValue(role.CreatedAt.Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(role.UpdatedAt.Format(time.RFC3339))

	permissions, permissionDiags := environmentRolePermissionsSet(ctx, role.Permissions)
	diags.Append(permissionDiags...)
	if diags.HasError() {
		return diags
	}
	model.Permissions = permissions

	return diags
}
