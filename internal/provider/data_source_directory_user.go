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
var _ datasource.DataSource = &DirectoryUserDataSource{}
var _ datasource.DataSourceWithConfigValidators = &DirectoryUserDataSource{}

func NewDirectoryUserDataSource() datasource.DataSource {
	return &DirectoryUserDataSource{}
}

// DirectoryUserDataSource defines the data source implementation.
type DirectoryUserDataSource struct {
	client *client.Client
}

// DirectoryUserDataSourceModel describes the data source data model.
type DirectoryUserDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	DirectoryID    types.String `tfsdk:"directory_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Email          types.String `tfsdk:"email"`
	FirstName      types.String `tfsdk:"first_name"`
	LastName       types.String `tfsdk:"last_name"`
	Username       types.String `tfsdk:"username"`
	State          types.String `tfsdk:"state"`
	IdpID          types.String `tfsdk:"idp_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *DirectoryUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory_user"
}

func (d *DirectoryUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a user synced from a WorkOS Directory.",
		MarkdownDescription: `
Use this data source to get information about a user synced from a WorkOS Directory.

You can look up a user by ID or by directory ID and email.

## Example Usage

### By ID

` + "```hcl" + `
data "workos_directory_user" "example" {
  id = "directory_user_01HXYZ..."
}
` + "```" + `

### By Directory and Email

` + "```hcl" + `
data "workos_directory_user" "john" {
  directory_id = data.workos_directory.main.id
  email        = "john@example.com"
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the directory user.",
				MarkdownDescription: "The unique identifier of the directory user (e.g., `directory_user_01HXYZ...`).",
				Optional:            true,
				Computed:            true,
			},
			"directory_id": schema.StringAttribute{
				Description:         "The ID of the directory to search in.",
				MarkdownDescription: "The ID of the directory to search in. Required when looking up by email.",
				Optional:            true,
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				Description:         "The organization ID the user belongs to.",
				MarkdownDescription: "The organization ID the user belongs to.",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				Description:         "The email address of the user.",
				MarkdownDescription: "The email address of the user. Required when looking up by directory.",
				Optional:            true,
				Computed:            true,
			},
			"first_name": schema.StringAttribute{
				Description:         "The user's first name.",
				MarkdownDescription: "The user's first name.",
				Computed:            true,
			},
			"last_name": schema.StringAttribute{
				Description:         "The user's last name.",
				MarkdownDescription: "The user's last name.",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				Description:         "The user's username.",
				MarkdownDescription: "The user's username (if available).",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				Description:         "The state of the directory user.",
				MarkdownDescription: "The state of the directory user (`active`, `suspended`).",
				Computed:            true,
			},
			"idp_id": schema.StringAttribute{
				Description:         "The user's ID in the identity provider.",
				MarkdownDescription: "The user's ID in the identity provider.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the user was synced.",
				MarkdownDescription: "The timestamp when the user was synced (RFC3339 format).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the user was last updated.",
				MarkdownDescription: "The timestamp when the user was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (d *DirectoryUserDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("directory_id"),
		),
		datasourcevalidator.RequiredTogether(
			path.MatchRoot("directory_id"),
			path.MatchRoot("email"),
		),
	}
}

func (d *DirectoryUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DirectoryUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DirectoryUserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user *client.DirectoryUser
	var err error

	if !config.ID.IsNull() {
		tflog.Debug(ctx, "Reading directory user by ID", map[string]any{
			"id": config.ID.ValueString(),
		})

		user, err = d.client.GetDirectoryUser(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Directory User",
				"Could not read directory user ID "+config.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	} else if !config.DirectoryID.IsNull() && !config.Email.IsNull() {
		tflog.Debug(ctx, "Reading directory user by email", map[string]any{
			"directory_id": config.DirectoryID.ValueString(),
			"email":        config.Email.ValueString(),
		})

		user, err = d.client.GetDirectoryUserByEmail(
			ctx,
			config.DirectoryID.ValueString(),
			config.Email.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Directory User",
				fmt.Sprintf("Could not find user with email %s in directory %s: %s",
					config.Email.ValueString(),
					config.DirectoryID.ValueString(),
					err.Error()),
			)
			return
		}
	}

	// Map response to state
	config.ID = types.StringValue(user.ID)
	config.DirectoryID = types.StringValue(user.DirectoryID)
	config.OrganizationID = types.StringValue(user.OrganizationID)
	config.Email = types.StringValue(user.Email)
	config.FirstName = types.StringValue(user.FirstName)
	config.LastName = types.StringValue(user.LastName)
	if user.Username != "" {
		config.Username = types.StringValue(user.Username)
	} else {
		config.Username = types.StringNull()
	}
	config.State = types.StringValue(user.State)
	config.IdpID = types.StringValue(user.IdpID)
	config.CreatedAt = types.StringValue(user.CreatedAt.Format(time.RFC3339))
	config.UpdatedAt = types.StringValue(user.UpdatedAt.Format(time.RFC3339))

	tflog.Info(ctx, "Read directory user", map[string]any{
		"id":    user.ID,
		"email": user.Email,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
