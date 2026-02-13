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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &OrganizationRoleDataSource{}
var _ datasource.DataSourceWithConfigValidators = &OrganizationRoleDataSource{}

func NewOrganizationRoleDataSource() datasource.DataSource {
	return &OrganizationRoleDataSource{}
}

// OrganizationRoleDataSource defines the data source implementation.
type OrganizationRoleDataSource struct {
	client *client.Client
}

// OrganizationRoleDataSourceModel describes the data source data model.
type OrganizationRoleDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Slug           types.String `tfsdk:"slug"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Type           types.String `tfsdk:"type"`
	Permissions    types.List   `tfsdk:"permissions"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *OrganizationRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_role"
}

func (d *OrganizationRoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a WorkOS Organization Role.",
		MarkdownDescription: `
Use this data source to get information about a WorkOS Organization Role.

You can look up a role by its slug or ID within an organization.

## Example Usage

### By Slug

` + "```hcl" + `
data "workos_organization_role" "billing_admin" {
  organization_id = "org_01HXYZ..."
  slug            = "billing-admin"
}
` + "```" + `

### By ID

` + "```hcl" + `
data "workos_organization_role" "example" {
  organization_id = "org_01HXYZ..."
  id              = "role_01HXYZ..."
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the organization role to look up.",
				MarkdownDescription: "The unique identifier of the organization role to look up.",
				Optional:            true,
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				Description:         "The ID of the organization the role belongs to.",
				MarkdownDescription: "The ID of the organization the role belongs to (e.g., `org_01HXYZ...`).",
				Required:            true,
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
			"permissions": schema.ListAttribute{
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

func (d *OrganizationRoleDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("slug"),
		),
	}
}

func (d *OrganizationRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (d *OrganizationRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config OrganizationRoleDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgID := config.OrganizationID.ValueString()
	var role *client.OrganizationRole
	var err error

	if !config.Slug.IsNull() {
		// Look up by slug
		tflog.Debug(ctx, "Reading organization role by slug", map[string]any{
			"organization_id": orgID,
			"slug":            config.Slug.ValueString(),
		})

		role, err = d.client.GetOrganizationRole(ctx, orgID, config.Slug.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Organization Role",
				"Could not read organization role with slug "+config.Slug.ValueString()+": "+err.Error(),
			)
			return
		}
	} else if !config.ID.IsNull() {
		// Look up by ID
		tflog.Debug(ctx, "Reading organization role by ID", map[string]any{
			"organization_id": orgID,
			"id":              config.ID.ValueString(),
		})

		role, err = d.client.GetOrganizationRoleByID(ctx, orgID, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Organization Role",
				"Could not find organization role with ID "+config.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Map response to state
	config.ID = types.StringValue(role.ID)
	config.Slug = types.StringValue(role.Slug)
	config.Name = types.StringValue(role.Name)
	config.Description = types.StringValue(role.Description)
	config.Type = types.StringValue(role.Type)
	config.CreatedAt = types.StringValue(role.CreatedAt.Format("2006-01-02T15:04:05Z"))
	config.UpdatedAt = types.StringValue(role.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	// Map permissions - always set as empty list rather than null
	if len(role.Permissions) > 0 {
		permissions, diags := types.ListValueFrom(ctx, types.StringType, role.Permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		config.Permissions = permissions
	} else {
		config.Permissions, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	tflog.Info(ctx, "Read organization role", map[string]any{
		"id":   role.ID,
		"slug": role.Slug,
		"name": role.Name,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
