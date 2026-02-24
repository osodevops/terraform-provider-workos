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
var _ datasource.DataSource = &UserDataSource{}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

// UserDataSource defines the data source implementation.
type UserDataSource struct {
	client *client.Client
}

// UserDataSourceModel describes the data source data model.
type UserDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Email             types.String `tfsdk:"email"`
	EmailVerified     types.Bool   `tfsdk:"email_verified"`
	FirstName         types.String `tfsdk:"first_name"`
	LastName          types.String `tfsdk:"last_name"`
	ProfilePictureURL types.String `tfsdk:"profile_picture_url"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a WorkOS AuthKit User.",
		MarkdownDescription: `
Use this data source to get information about a WorkOS AuthKit User.

## Example Usage

### Lookup by ID

` + "```hcl" + `
data "workos_user" "by_id" {
  id = "user_01HXYZ..."
}

output "user_email" {
  value = data.workos_user.by_id.email
}
` + "```" + `

### Lookup by Email

` + "```hcl" + `
data "workos_user" "by_email" {
  email = "user@example.com"
}

output "user_name" {
  value = "${data.workos_user.by_email.first_name} ${data.workos_user.by_email.last_name}"
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the user.",
				MarkdownDescription: "The unique identifier of the user (e.g., `user_01HXYZ...`). Either `id` or `email` must be specified.",
				Optional:            true,
				Computed:            true,
			},
			"email": schema.StringAttribute{
				Description:         "The user's email address.",
				MarkdownDescription: "The user's email address. Either `id` or `email` must be specified.",
				Optional:            true,
				Computed:            true,
			},
			"email_verified": schema.BoolAttribute{
				Description:         "Whether the user's email address has been verified.",
				MarkdownDescription: "Whether the user's email address has been verified.",
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
			"profile_picture_url": schema.StringAttribute{
				Description:         "URL of the user's profile picture.",
				MarkdownDescription: "URL of the user's profile picture.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the user was created.",
				MarkdownDescription: "The timestamp when the user was created (RFC3339 format).",
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

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user *client.User
	var err error

	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		tflog.Debug(ctx, "Looking up user by ID", map[string]any{
			"id": data.ID.ValueString(),
		})
		user, err = d.client.GetUser(ctx, data.ID.ValueString())
	} else if !data.Email.IsNull() && data.Email.ValueString() != "" {
		tflog.Debug(ctx, "Looking up user by email", map[string]any{
			"email": data.Email.ValueString(),
		})
		user, err = d.client.GetUserByEmail(ctx, data.Email.ValueString())
	} else {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'email' must be specified to look up a user.",
		)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User",
			"Could not read user: "+err.Error(),
		)
		return
	}

	// Map response to state
	data.ID = types.StringValue(user.ID)
	data.Email = types.StringValue(user.Email)
	data.EmailVerified = types.BoolValue(user.EmailVerified)
	if user.FirstName != "" {
		data.FirstName = types.StringValue(user.FirstName)
	} else {
		data.FirstName = types.StringNull()
	}
	if user.LastName != "" {
		data.LastName = types.StringValue(user.LastName)
	} else {
		data.LastName = types.StringNull()
	}
	if user.ProfilePictureURL != "" {
		data.ProfilePictureURL = types.StringValue(user.ProfilePictureURL)
	} else {
		data.ProfilePictureURL = types.StringNull()
	}
	data.CreatedAt = types.StringValue(user.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(user.UpdatedAt.Format(time.RFC3339))

	tflog.Info(ctx, "Read user", map[string]any{
		"id":    user.ID,
		"email": user.Email,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
