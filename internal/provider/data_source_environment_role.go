// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

var _ datasource.DataSource = &EnvironmentRoleDataSource{}
var _ datasource.DataSourceWithConfigValidators = &EnvironmentRoleDataSource{}

func NewEnvironmentRoleDataSource() datasource.DataSource {
	return &EnvironmentRoleDataSource{}
}

// EnvironmentRoleDataSource defines the data source implementation.
type EnvironmentRoleDataSource struct {
	client *client.Client
}

// EnvironmentRoleDataSourceModel describes the data source data model.
type EnvironmentRoleDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Slug             types.String `tfsdk:"slug"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Type             types.String `tfsdk:"type"`
	ResourceTypeSlug types.String `tfsdk:"resource_type_slug"`
	Permissions      types.Set    `tfsdk:"permissions"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (d *EnvironmentRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_role"
}

func (d *EnvironmentRoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a WorkOS environment-level role.",
		MarkdownDescription: `
Use this data source to get information about a WorkOS environment-level role.

You can look up a role by its slug or ID.

## Example Usage

### By Slug

` + "```hcl" + `
data "workos_environment_role" "admin" {
  slug = "admin"
}
` + "```" + `

### By ID

` + "```hcl" + `
data "workos_environment_role" "example" {
  id = "role_01HXYZ..."
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the environment role to look up.",
				MarkdownDescription: "The unique identifier of the environment role to look up.",
				Optional:            true,
				Computed:            true,
			},
			"slug": schema.StringAttribute{
				Description:         "The slug identifier of the role to look up.",
				MarkdownDescription: "The slug identifier of the role to look up.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the role.",
				MarkdownDescription: "The display name of the role.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				Description:         "A description of the role.",
				MarkdownDescription: "A description of the role.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				Description:         "The type of the role.",
				MarkdownDescription: "The type of the role.",
				Computed:            true,
			},
			"resource_type_slug": schema.StringAttribute{
				Description:         "The resource type slug this role is scoped to.",
				MarkdownDescription: "The resource type slug this role is scoped to.",
				Computed:            true,
			},
			"permissions": schema.SetAttribute{
				Description:         "The permissions associated with the role.",
				MarkdownDescription: "The permissions associated with the role.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the role was created.",
				MarkdownDescription: "The timestamp when the role was created (RFC3339 format).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the role was last updated.",
				MarkdownDescription: "The timestamp when the role was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (d *EnvironmentRoleDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("slug"),
		),
	}
}

func (d *EnvironmentRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *EnvironmentRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config EnvironmentRoleDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var role *client.EnvironmentRole
	var err error

	if !config.Slug.IsNull() {
		tflog.Debug(ctx, "Reading environment role by slug", map[string]any{
			"slug": config.Slug.ValueString(),
		})

		role, err = d.client.GetEnvironmentRole(ctx, config.Slug.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Environment Role",
				"Could not read environment role with slug "+config.Slug.ValueString()+": "+err.Error(),
			)
			return
		}
	} else if !config.ID.IsNull() {
		tflog.Debug(ctx, "Reading environment role by ID", map[string]any{
			"id": config.ID.ValueString(),
		})

		role, err = d.client.GetEnvironmentRoleByID(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Environment Role",
				"Could not find environment role with ID "+config.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	resp.Diagnostics.Append(applyEnvironmentRoleToDataSourceModel(ctx, role, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Read environment role", map[string]any{
		"id":   role.ID,
		"slug": role.Slug,
		"name": role.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
