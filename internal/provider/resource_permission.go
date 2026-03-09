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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PermissionResource{}
var _ resource.ResourceWithImportState = &PermissionResource{}

func NewPermissionResource() resource.Resource {
	return &PermissionResource{}
}

// PermissionResource defines the resource implementation.
type PermissionResource struct {
	client *client.Client
}

// PermissionResourceModel describes the resource data model.
type PermissionResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Slug             types.String `tfsdk:"slug"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	System           types.Bool   `tfsdk:"system"`
	ResourceTypeSlug types.String `tfsdk:"resource_type_slug"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *PermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

func (r *PermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS Permission.",
		MarkdownDescription: `
Manages a WorkOS Permission.

Permissions are environment-level resources that can be assigned to both environment roles
and organization roles. Permissions are identified by their slug.

## Example Usage

` + "```hcl" + `
resource "workos_permission" "billing_read" {
  slug        = "billing:read"
  name        = "Read Billing"
  description = "Allows reading billing data"
}
` + "```" + `

## Import

Permissions can be imported using their slug:

` + "```shell" + `
terraform import workos_permission.example billing:read
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the permission.",
				MarkdownDescription: "The unique identifier of the permission.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Description:         "The slug identifier for the permission. Must be unique within the environment.",
				MarkdownDescription: "The slug identifier for the permission. Must be unique within the environment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the permission.",
				MarkdownDescription: "The display name of the permission.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "A description of the permission.",
				MarkdownDescription: "A description of the permission.",
				Optional:            true,
				Computed:            true,
			},
			"system": schema.BoolAttribute{
				Description:         "Whether this is a system-managed permission.",
				MarkdownDescription: "Whether this is a system-managed permission.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type_slug": schema.StringAttribute{
				Description:         "The slug of the resource type this permission applies to.",
				MarkdownDescription: "The slug of the resource type this permission applies to.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the permission was created.",
				MarkdownDescription: "The timestamp when the permission was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the permission was last updated.",
				MarkdownDescription: "The timestamp when the permission was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (r *PermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *PermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating permission", map[string]any{
		"slug": plan.Slug.ValueString(),
		"name": plan.Name.ValueString(),
	})

	createReq := &client.PermissionCreateRequest{
		Slug: plan.Slug.ValueString(),
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}

	if !plan.ResourceTypeSlug.IsNull() && !plan.ResourceTypeSlug.IsUnknown() {
		createReq.ResourceTypeSlug = plan.ResourceTypeSlug.ValueString()
	}

	perm, err := r.client.CreatePermission(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Permission",
			"Could not create permission, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(perm.ID)
	plan.Description = types.StringValue(perm.Description)
	plan.System = types.BoolValue(perm.System)
	plan.ResourceTypeSlug = types.StringValue(perm.ResourceTypeSlug)
	plan.CreatedAt = types.StringValue(perm.CreatedAt.Format(time.RFC3339))
	plan.UpdatedAt = types.StringValue(perm.UpdatedAt.Format(time.RFC3339))

	tflog.Info(ctx, "Created permission", map[string]any{
		"id":   perm.ID,
		"slug": perm.Slug,
		"name": perm.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading permission", map[string]any{
		"slug": state.Slug.ValueString(),
	})

	perm, err := r.client.GetPermission(ctx, state.Slug.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Permission not found, removing from state", map[string]any{
				"slug": state.Slug.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Permission",
			"Could not read permission "+state.Slug.ValueString()+": "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(perm.ID)
	state.Slug = types.StringValue(perm.Slug)
	state.Name = types.StringValue(perm.Name)
	state.Description = types.StringValue(perm.Description)
	state.System = types.BoolValue(perm.System)
	state.ResourceTypeSlug = types.StringValue(perm.ResourceTypeSlug)
	state.CreatedAt = types.StringValue(perm.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(perm.UpdatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PermissionResourceModel
	var state PermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating permission", map[string]any{
		"slug": state.Slug.ValueString(),
		"name": plan.Name.ValueString(),
	})

	// Skip update if no user-configurable attributes changed
	if plan.Name.Equal(state.Name) && plan.Description.Equal(state.Description) {
		plan.ID = state.ID
		plan.System = state.System
		plan.ResourceTypeSlug = state.ResourceTypeSlug
		plan.CreatedAt = state.CreatedAt
		plan.UpdatedAt = state.UpdatedAt
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	updateReq := &client.PermissionUpdateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	perm, err := r.client.UpdatePermission(ctx, state.Slug.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Permission",
			"Could not update permission, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID
	plan.System = state.System
	plan.ResourceTypeSlug = state.ResourceTypeSlug
	plan.CreatedAt = state.CreatedAt
	plan.Description = types.StringValue(perm.Description)
	plan.UpdatedAt = types.StringValue(perm.UpdatedAt.Format(time.RFC3339))

	tflog.Info(ctx, "Updated permission", map[string]any{
		"id":   perm.ID,
		"slug": perm.Slug,
		"name": perm.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting permission", map[string]any{
		"slug": state.Slug.ValueString(),
	})

	err := r.client.DeletePermission(ctx, state.Slug.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Permission already deleted", map[string]any{
				"slug": state.Slug.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Permission",
			"Could not delete permission, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted permission", map[string]any{
		"slug": state.Slug.ValueString(),
	})
}

func (r *PermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing permission", map[string]any{
		"id": req.ID,
	})

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), req.ID)...)
}
