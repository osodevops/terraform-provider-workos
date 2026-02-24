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
var _ datasource.DataSource = &DirectoryDataSource{}
var _ datasource.DataSourceWithConfigValidators = &DirectoryDataSource{}

func NewDirectoryDataSource() datasource.DataSource {
	return &DirectoryDataSource{}
}

// DirectoryDataSource defines the data source implementation.
type DirectoryDataSource struct {
	client *client.Client
}

// DirectoryDataSourceModel describes the data source data model.
type DirectoryDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	State          types.String `tfsdk:"state"`
	Endpoint       types.String `tfsdk:"endpoint"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *DirectoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory"
}

func (d *DirectoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a WorkOS Directory.",
		MarkdownDescription: `
Use this data source to get information about a WorkOS Directory.

You can look up a directory by its ID or by organization ID.

## Example Usage

### By ID

` + "```hcl" + `
data "workos_directory" "example" {
  id = "directory_01HXYZ..."
}
` + "```" + `

### By Organization

` + "```hcl" + `
data "workos_directory" "example" {
  organization_id = workos_organization.main.id
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the directory to look up.",
				MarkdownDescription: "The unique identifier of the directory to look up (e.g., `directory_01HXYZ...`).",
				Optional:            true,
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				Description:         "The organization ID to find the directory for.",
				MarkdownDescription: "The organization ID to find the directory for.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The name of the directory.",
				MarkdownDescription: "The name of the directory.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				Description:         "The type of directory.",
				MarkdownDescription: "The type of directory (e.g., `okta scim v2.0`).",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				Description:         "The current state of the directory.",
				MarkdownDescription: "The current state of the directory (`linked`, `unlinked`, `invalid_credentials`).",
				Computed:            true,
			},
			"endpoint": schema.StringAttribute{
				Description:         "The SCIM endpoint URL for this directory.",
				MarkdownDescription: "The SCIM endpoint URL for this directory.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the directory was created.",
				MarkdownDescription: "The timestamp when the directory was created (RFC3339 format).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the directory was last updated.",
				MarkdownDescription: "The timestamp when the directory was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (d *DirectoryDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("organization_id"),
		),
	}
}

func (d *DirectoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DirectoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DirectoryDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var dir *client.Directory
	var err error

	if !config.ID.IsNull() {
		tflog.Debug(ctx, "Reading directory by ID", map[string]any{
			"id": config.ID.ValueString(),
		})

		dir, err = d.client.GetDirectory(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Directory",
				"Could not read directory ID "+config.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	} else if !config.OrganizationID.IsNull() {
		tflog.Debug(ctx, "Reading directory by organization", map[string]any{
			"organization_id": config.OrganizationID.ValueString(),
		})

		dir, err = d.client.GetDirectoryByOrganization(ctx, config.OrganizationID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Directory",
				"Could not find directory for organization "+config.OrganizationID.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Map response to state
	config.ID = types.StringValue(dir.ID)
	config.OrganizationID = types.StringValue(dir.OrganizationID)
	config.Name = types.StringValue(dir.Name)
	config.Type = types.StringValue(dir.Type)
	config.State = types.StringValue(dir.State)
	if dir.Endpoint != "" {
		config.Endpoint = types.StringValue(dir.Endpoint)
	} else {
		config.Endpoint = types.StringNull()
	}
	config.CreatedAt = types.StringValue(dir.CreatedAt.Format(time.RFC3339))
	config.UpdatedAt = types.StringValue(dir.UpdatedAt.Format(time.RFC3339))

	tflog.Info(ctx, "Read directory", map[string]any{
		"id":   dir.ID,
		"type": dir.Type,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
