// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &PermissionDataSource{}

func NewPermissionDataSource() datasource.DataSource {
	return &PermissionDataSource{}
}

// PermissionDataSource defines the data source implementation.
type PermissionDataSource struct {
	client *client.Client
}

// PermissionDataSourceModel describes the data source data model.
type PermissionDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Slug             types.String `tfsdk:"slug"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	System           types.Bool   `tfsdk:"system"`
	ResourceTypeSlug types.String `tfsdk:"resource_type_slug"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (d *PermissionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

func (d *PermissionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a WorkOS Permission.",
		MarkdownDescription: `
Use this data source to get information about a WorkOS Permission.

You can look up a permission by its slug.

## Example Usage

` + "```hcl" + `
data "workos_permission" "billing_read" {
  slug = "billing:read"
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the permission.",
				MarkdownDescription: "The unique identifier of the permission.",
				Computed:            true,
			},
			"slug": schema.StringAttribute{
				Description:         "The slug identifier of the permission to look up.",
				MarkdownDescription: "The slug identifier of the permission to look up.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the permission.",
				MarkdownDescription: "The display name of the permission.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				Description:         "A description of the permission.",
				MarkdownDescription: "A description of the permission.",
				Computed:            true,
			},
			"system": schema.BoolAttribute{
				Description:         "Whether this is a system-managed permission.",
				MarkdownDescription: "Whether this is a system-managed permission.",
				Computed:            true,
			},
			"resource_type_slug": schema.StringAttribute{
				Description:         "The slug of the resource type this permission applies to.",
				MarkdownDescription: "The slug of the resource type this permission applies to.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the permission was created.",
				MarkdownDescription: "The timestamp when the permission was created (RFC3339 format).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the permission was last updated.",
				MarkdownDescription: "The timestamp when the permission was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (d *PermissionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PermissionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PermissionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	slug := config.Slug.ValueString()

	tflog.Debug(ctx, "Reading permission by slug", map[string]any{
		"slug": slug,
	})

	perm, err := d.client.GetPermission(ctx, slug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Permission",
			"Could not read permission with slug "+slug+": "+err.Error(),
		)
		return
	}

	config.ID = types.StringValue(perm.ID)
	config.Slug = types.StringValue(perm.Slug)
	config.Name = types.StringValue(perm.Name)
	config.Description = types.StringValue(perm.Description)
	config.System = types.BoolValue(perm.System)
	config.ResourceTypeSlug = types.StringValue(perm.ResourceTypeSlug)
	config.CreatedAt = types.StringValue(perm.CreatedAt.Format(time.RFC3339))
	config.UpdatedAt = types.StringValue(perm.UpdatedAt.Format(time.RFC3339))

	tflog.Info(ctx, "Read permission", map[string]any{
		"id":   perm.ID,
		"slug": perm.Slug,
		"name": perm.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
