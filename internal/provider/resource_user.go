// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

// UserResource defines the resource implementation.
type UserResource struct {
	client *client.Client
}

// UserResourceModel describes the resource data model.
type UserResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Email             types.String `tfsdk:"email"`
	EmailVerified     types.Bool   `tfsdk:"email_verified"`
	FirstName         types.String `tfsdk:"first_name"`
	LastName          types.String `tfsdk:"last_name"`
	Password          types.String `tfsdk:"password"`
	PasswordHash      types.String `tfsdk:"password_hash"`
	ProfilePictureURL types.String `tfsdk:"profile_picture_url"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS AuthKit User.",
		MarkdownDescription: `
Manages a WorkOS AuthKit User.

Users are the core identity entity in WorkOS AuthKit. They can authenticate
via various methods (password, magic link, SSO, OAuth) and be members of
one or more organizations.

## Example Usage

### Basic User

` + "```hcl" + `
resource "workos_user" "example" {
  email          = "user@example.com"
  first_name     = "John"
  last_name      = "Doe"
  email_verified = true
}
` + "```" + `

### User with Password

` + "```hcl" + `
resource "workos_user" "with_password" {
  email          = "user@example.com"
  first_name     = "Jane"
  last_name      = "Smith"
  password       = var.user_password
  email_verified = true
}
` + "```" + `

## Import

Users can be imported using the user ID:

` + "```shell" + `
terraform import workos_user.example user_01HXYZ...
` + "```" + `

**Note:** The password cannot be imported and must be set in configuration
if password authentication is required.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the user.",
				MarkdownDescription: "The unique identifier of the user (e.g., `user_01HXYZ...`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Description:         "The user's email address.",
				MarkdownDescription: "The user's email address. Must be unique across all users.",
				Required:            true,
			},
			"email_verified": schema.BoolAttribute{
				Description:         "Whether the user's email address has been verified.",
				MarkdownDescription: "Whether the user's email address has been verified. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"first_name": schema.StringAttribute{
				Description:         "The user's first name.",
				MarkdownDescription: "The user's first name.",
				Optional:            true,
			},
			"last_name": schema.StringAttribute{
				Description:         "The user's last name.",
				MarkdownDescription: "The user's last name.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				Description:         "The user's password. Write-only, not returned by API.",
				MarkdownDescription: "The user's password. This is a write-only field and is not returned by the API. Only used during user creation.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password_hash": schema.StringAttribute{
				Description:         "A pre-hashed password. Write-only, not returned by API.",
				MarkdownDescription: "A pre-hashed password (bcrypt). This is a write-only field and is not returned by the API. Use this if you're migrating users with existing password hashes.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the user was last updated.",
				MarkdownDescription: "The timestamp when the user was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating user", map[string]any{
		"email": plan.Email.ValueString(),
	})

	createReq := &client.UserCreateRequest{
		Email:         plan.Email.ValueString(),
		EmailVerified: plan.EmailVerified.ValueBool(),
	}

	if !plan.FirstName.IsNull() {
		createReq.FirstName = plan.FirstName.ValueString()
	}
	if !plan.LastName.IsNull() {
		createReq.LastName = plan.LastName.ValueString()
	}
	if !plan.Password.IsNull() {
		createReq.Password = plan.Password.ValueString()
	}
	if !plan.PasswordHash.IsNull() {
		createReq.PasswordHash = plan.PasswordHash.ValueString()
	}

	user, err := r.client.CreateUser(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating User",
			"Could not create user, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(user.ID)
	plan.Email = types.StringValue(user.Email)
	plan.EmailVerified = types.BoolValue(user.EmailVerified)
	if user.FirstName != "" {
		plan.FirstName = types.StringValue(user.FirstName)
	}
	if user.LastName != "" {
		plan.LastName = types.StringValue(user.LastName)
	}
	if user.ProfilePictureURL != "" {
		plan.ProfilePictureURL = types.StringValue(user.ProfilePictureURL)
	} else {
		plan.ProfilePictureURL = types.StringNull()
	}
	plan.CreatedAt = types.StringValue(user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	plan.UpdatedAt = types.StringValue(user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	tflog.Info(ctx, "Created user", map[string]any{
		"id":    user.ID,
		"email": user.Email,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading user", map[string]any{
		"id": state.ID.ValueString(),
	})

	user, err := r.client.GetUser(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "User not found, removing from state", map[string]any{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading User",
			"Could not read user ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state
	state.Email = types.StringValue(user.Email)
	state.EmailVerified = types.BoolValue(user.EmailVerified)
	if user.FirstName != "" {
		state.FirstName = types.StringValue(user.FirstName)
	} else {
		state.FirstName = types.StringNull()
	}
	if user.LastName != "" {
		state.LastName = types.StringValue(user.LastName)
	} else {
		state.LastName = types.StringNull()
	}
	if user.ProfilePictureURL != "" {
		state.ProfilePictureURL = types.StringValue(user.ProfilePictureURL)
	} else {
		state.ProfilePictureURL = types.StringNull()
	}
	state.CreatedAt = types.StringValue(user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	state.UpdatedAt = types.StringValue(user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	// Note: Password and PasswordHash are not returned by the API, preserve state values

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserResourceModel
	var state UserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating user", map[string]any{
		"id":    state.ID.ValueString(),
		"email": plan.Email.ValueString(),
	})

	updateReq := &client.UserUpdateRequest{}

	if !plan.Email.Equal(state.Email) {
		updateReq.Email = plan.Email.ValueString()
	}
	if !plan.FirstName.Equal(state.FirstName) {
		updateReq.FirstName = plan.FirstName.ValueString()
	}
	if !plan.LastName.Equal(state.LastName) {
		updateReq.LastName = plan.LastName.ValueString()
	}
	if !plan.EmailVerified.Equal(state.EmailVerified) {
		emailVerified := plan.EmailVerified.ValueBool()
		updateReq.EmailVerified = &emailVerified
	}

	user, err := r.client.UpdateUser(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating User",
			"Could not update user, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = state.ID
	plan.Email = types.StringValue(user.Email)
	plan.EmailVerified = types.BoolValue(user.EmailVerified)
	if user.FirstName != "" {
		plan.FirstName = types.StringValue(user.FirstName)
	}
	if user.LastName != "" {
		plan.LastName = types.StringValue(user.LastName)
	}
	if user.ProfilePictureURL != "" {
		plan.ProfilePictureURL = types.StringValue(user.ProfilePictureURL)
	} else {
		plan.ProfilePictureURL = types.StringNull()
	}
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	tflog.Info(ctx, "Updated user", map[string]any{
		"id": user.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting user", map[string]any{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteUser(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "User already deleted", map[string]any{
				"id": state.ID.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting User",
			"Could not delete user, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted user", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing user", map[string]any{
		"id": req.ID,
	})

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Note: Password and PasswordHash cannot be imported
	resp.Diagnostics.AddWarning(
		"Password Not Imported",
		"The user's password cannot be imported from the API. If password authentication "+
			"is required, you must set the 'password' attribute in your configuration.",
	)
}
