// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DirectoryGroupDataSource{}
var _ datasource.DataSourceWithConfigValidators = &DirectoryGroupDataSource{}

func NewDirectoryGroupDataSource() datasource.DataSource {
	return &DirectoryGroupDataSource{}
}

// DirectoryGroupDataSource defines the data source implementation.
type DirectoryGroupDataSource struct {
	client *client.Client
}

// DirectoryGroupDataSourceModel describes the data source data model.
type DirectoryGroupDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	DirectoryID    types.String `tfsdk:"directory_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	IdpID          types.String `tfsdk:"idp_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *DirectoryGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory_group"
}

func (d *DirectoryGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a group synced from a WorkOS Directory.",
		MarkdownDescription: `
Use this data source to get information about a group synced from a WorkOS Directory.

You can look up a group by ID or by directory ID and name.

## Example Usage

### By ID

` + "```hcl" + `
data "workos_directory_group" "example" {
  id = "directory_group_01HXYZ..."
}
` + "```" + `

### By Directory and Name

` + "```hcl" + `
data "workos_directory_group" "engineering" {
  directory_id = data.workos_directory.main.id
  name         = "Engineering"
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the directory group.",
				MarkdownDescription: "The unique identifier of the directory group (e.g., `directory_group_01HXYZ...`).",
				Optional:            true,
				Computed:            true,
			},
			"directory_id": schema.StringAttribute{
				Description:         "The ID of the directory to search in.",
				MarkdownDescription: "The ID of the directory to search in. Required when looking up by name.",
				Optional:            true,
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				Description:         "The organization ID the group belongs to.",
				MarkdownDescription: "The organization ID the group belongs to.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The name of the group.",
				MarkdownDescription: "The name of the group. Required when looking up by directory.",
				Optional:            true,
				Computed:            true,
			},
			"idp_id": schema.StringAttribute{
				Description:         "The group's ID in the identity provider.",
				MarkdownDescription: "The group's ID in the identity provider.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the group was synced.",
				MarkdownDescription: "The timestamp when the group was synced (RFC3339 format).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the group was last updated.",
				MarkdownDescription: "The timestamp when the group was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (d *DirectoryGroupDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("directory_id"),
		),
		datasourcevalidator.RequiredTogether(
			path.MatchRoot("directory_id"),
			path.MatchRoot("name"),
		),
	}
}

func (d *DirectoryGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DirectoryGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DirectoryGroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group *client.DirectoryGroup
	var err error

	if !config.ID.IsNull() {
		tflog.Debug(ctx, "Reading directory group by ID", map[string]any{
			"id": config.ID.ValueString(),
		})

		group, err = d.client.GetDirectoryGroup(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Directory Group",
				"Could not read directory group ID "+config.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	} else if !config.DirectoryID.IsNull() && !config.Name.IsNull() {
		tflog.Debug(ctx, "Reading directory group by name", map[string]any{
			"directory_id": config.DirectoryID.ValueString(),
			"name":         config.Name.ValueString(),
		})

		group, err = d.client.GetDirectoryGroupByName(
			ctx,
			config.DirectoryID.ValueString(),
			config.Name.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Directory Group",
				fmt.Sprintf("Could not find group with name %s in directory %s: %s",
					config.Name.ValueString(),
					config.DirectoryID.ValueString(),
					err.Error()),
			)
			return
		}
	}

	// Map response to state
	config.ID = types.StringValue(group.ID)
	config.DirectoryID = types.StringValue(group.DirectoryID)
	config.OrganizationID = types.StringValue(group.OrganizationID)
	config.Name = types.StringValue(group.Name)
	config.IdpID = types.StringValue(group.IdpID)
	config.CreatedAt = types.StringValue(group.CreatedAt.Format(time.RFC3339))
	config.UpdatedAt = types.StringValue(group.UpdatedAt.Format(time.RFC3339))

	tflog.Info(ctx, "Read directory group", map[string]any{
		"id":   group.ID,
		"name": group.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
