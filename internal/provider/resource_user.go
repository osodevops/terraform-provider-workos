// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
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
	PasswordHashType  types.String `tfsdk:"password_hash_type"`
	ExternalID        types.String `tfsdk:"external_id"`
	Metadata          types.Map    `tfsdk:"metadata"`
	Locale            types.String `tfsdk:"locale"`
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

### User with External ID and Metadata

` + "```hcl" + `
resource "workos_user" "with_metadata" {
  email          = "user@example.com"
  first_name     = "Jane"
  last_name      = "Smith"
  external_id    = "ext-12345"
  email_verified = true

  metadata = {
    department = "Engineering"
    timezone   = "America/New_York"
  }
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
				MarkdownDescription: "A pre-hashed password (bcrypt or argon2). This is a write-only field and is not returned by the API. Use this if you're migrating users with existing password hashes.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password_hash_type": schema.StringAttribute{
				Description:         "The type of password hash. Write-only, used with password_hash.",
				MarkdownDescription: "The type of password hash (e.g., `bcrypt`, `argon2`). This is a write-only field used only during creation alongside `password_hash`.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_id": schema.StringAttribute{
				Description:         "An external identifier for the user.",
				MarkdownDescription: "An external identifier for the user. Useful for mapping to identifiers in external systems.",
				Optional:            true,
				Computed:            true,
			},
			"metadata": schema.MapAttribute{
				Description:         "Custom metadata for the user.",
				MarkdownDescription: "Custom metadata for the user as key-value string pairs.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"locale": schema.StringAttribute{
				Description:         "The user's locale.",
				MarkdownDescription: "The user's locale (e.g., `en-US`). Set by the system based on user activity.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_picture_url": schema.StringAttribute{
				Description:         "URL of the user's profile picture.",
				MarkdownDescription: "URL of the user's profile picture.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					useStateForUnknownIfConfigUnchanged{
						configAttributes: []path.Path{
							path.Root("email"),
							path.Root("email_verified"),
							path.Root("first_name"),
							path.Root("last_name"),
							path.Root("external_id"),
							path.Root("metadata"),
						},
					},
				},
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
	if !plan.PasswordHashType.IsNull() {
		createReq.PasswordHashType = plan.PasswordHashType.ValueString()
	}
	if !plan.ExternalID.IsNull() {
		createReq.ExternalID = plan.ExternalID.ValueString()
	}
	if !plan.Metadata.IsNull() {
		metadata := make(map[string]string)
		resp.Diagnostics.Append(plan.Metadata.ElementsAs(ctx, &metadata, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Metadata = metadata
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
	if user.ExternalID != "" {
		plan.ExternalID = types.StringValue(user.ExternalID)
	} else {
		plan.ExternalID = types.StringNull()
	}
	if len(user.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, user.Metadata)
		resp.Diagnostics.Append(diags...)
		plan.Metadata = metadataMap
	} else {
		plan.Metadata = types.MapNull(types.StringType)
	}
	if user.Locale != "" {
		plan.Locale = types.StringValue(user.Locale)
	} else {
		plan.Locale = types.StringNull()
	}
	plan.CreatedAt = types.StringValue(user.CreatedAt.Format(time.RFC3339))
	plan.UpdatedAt = types.StringValue(user.UpdatedAt.Format(time.RFC3339))

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
	if user.ExternalID != "" {
		state.ExternalID = types.StringValue(user.ExternalID)
	} else {
		state.ExternalID = types.StringNull()
	}
	if len(user.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, user.Metadata)
		resp.Diagnostics.Append(diags...)
		state.Metadata = metadataMap
	} else {
		state.Metadata = types.MapNull(types.StringType)
	}
	if user.Locale != "" {
		state.Locale = types.StringValue(user.Locale)
	} else {
		state.Locale = types.StringNull()
	}
	state.CreatedAt = types.StringValue(user.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(user.UpdatedAt.Format(time.RFC3339))

	// Note: Password, PasswordHash, and PasswordHashType are not returned by the API, preserve state values

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

	// Skip update if no user-configurable attributes changed
	if plan.Email.Equal(state.Email) &&
		plan.EmailVerified.Equal(state.EmailVerified) &&
		plan.FirstName.Equal(state.FirstName) &&
		plan.LastName.Equal(state.LastName) &&
		plan.ExternalID.Equal(state.ExternalID) &&
		plan.Metadata.Equal(state.Metadata) {
		plan.ID = state.ID
		plan.CreatedAt = state.CreatedAt
		plan.UpdatedAt = state.UpdatedAt
		plan.ProfilePictureURL = state.ProfilePictureURL
		plan.Locale = state.Locale
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	updateReq := &client.UserUpdateRequest{}

	// Always include email_verified — the API may reset it when the email changes.
	emailVerified := plan.EmailVerified.ValueBool()
	updateReq.EmailVerified = &emailVerified

	if !plan.Email.Equal(state.Email) {
		updateReq.Email = plan.Email.ValueString()
	}
	if !plan.FirstName.Equal(state.FirstName) {
		updateReq.FirstName = plan.FirstName.ValueString()
	}
	if !plan.LastName.Equal(state.LastName) {
		updateReq.LastName = plan.LastName.ValueString()
	}
	if !plan.ExternalID.Equal(state.ExternalID) {
		updateReq.ExternalID = plan.ExternalID.ValueString()
	}
	if !plan.Metadata.Equal(state.Metadata) {
		metadata := make(map[string]string)
		resp.Diagnostics.Append(plan.Metadata.ElementsAs(ctx, &metadata, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Metadata = metadata
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
	if user.ExternalID != "" {
		plan.ExternalID = types.StringValue(user.ExternalID)
	} else {
		plan.ExternalID = types.StringNull()
	}
	if len(user.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, user.Metadata)
		resp.Diagnostics.Append(diags...)
		plan.Metadata = metadataMap
	} else {
		plan.Metadata = types.MapNull(types.StringType)
	}
	if user.Locale != "" {
		plan.Locale = types.StringValue(user.Locale)
	} else {
		plan.Locale = types.StringNull()
	}
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(user.UpdatedAt.Format(time.RFC3339))

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
		"Write-Only Fields Not Imported",
		"The user's password, password_hash, and password_hash_type cannot be imported from the API. "+
			"If password authentication is required, you must set these attributes in your configuration.",
	)
}
