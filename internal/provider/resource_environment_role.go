// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

var _ resource.Resource = &EnvironmentRoleResource{}
var _ resource.ResourceWithImportState = &EnvironmentRoleResource{}
var _ resource.ResourceWithModifyPlan = &EnvironmentRoleResource{}

func NewEnvironmentRoleResource() resource.Resource {
	return &EnvironmentRoleResource{}
}

// EnvironmentRoleResource defines the resource implementation.
type EnvironmentRoleResource struct {
	client *client.Client
}

// EnvironmentRoleResourceModel describes the resource data model.
type EnvironmentRoleResourceModel struct {
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

func (r *EnvironmentRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_role"
}

func (r *EnvironmentRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS environment-level role.",
		MarkdownDescription: `
Manages a WorkOS environment-level role.

Environment roles are global to a WorkOS environment and can be assigned across
organizations. They are identified by an immutable slug.

The WorkOS public API does not currently expose environment-role deletion. Terraform
can create, import, read, and update environment roles, but destroying this resource
returns a diagnostic instead of silently leaving the role behind. Use ` + "`terraform state rm`" + `
or ` + "`tofu state rm`" + ` if you need Terraform to stop managing an existing role.

When ` + "`permissions`" + ` is configured, the provider manages the complete permission set for
the role by using WorkOS' replace-all permissions endpoint.

## Example Usage

` + "```hcl" + `
resource "workos_environment_role" "billing_admin" {
  slug        = "billing-admin"
  name        = "Billing Admin"
  description = "Can manage billing across organizations"

  permissions = [
    workos_permission.billing_read.slug,
    workos_permission.billing_write.slug,
  ]
}
` + "```" + `

## Import

Environment roles can be imported using their slug:

` + "```shell" + `
terraform import workos_environment_role.example billing-admin
` + "```" + `

OpenTofu uses the same import ID format:

` + "```shell" + `
tofu import workos_environment_role.example billing-admin
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the environment role.",
				MarkdownDescription: "The unique identifier of the environment role.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Description:         "The slug identifier for the role. Must be unique within the environment.",
				MarkdownDescription: "The slug identifier for the role. Must be unique within the environment. Changing this value after creation is not supported because WorkOS does not currently expose a public delete API for environment roles.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the role.",
				MarkdownDescription: "The display name of the role.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "A description of the role.",
				MarkdownDescription: "A description of the role.",
				Optional:            true,
				Computed:            true,
			},
			"type": schema.StringAttribute{
				Description:         "The type of the role.",
				MarkdownDescription: "The type of the role.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type_slug": schema.StringAttribute{
				Description:         "The resource type slug this role is scoped to.",
				MarkdownDescription: "The resource type slug this role is scoped to. Changing this value after creation is not supported because WorkOS does not currently expose a public delete API for environment roles.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"permissions": schema.SetAttribute{
				Description:         "The complete set of permissions associated with the role. When configured, this set is authoritative.",
				MarkdownDescription: "The complete set of permissions associated with the role. When configured, this set is authoritative.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the role was created.",
				MarkdownDescription: "The timestamp when the role was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the role was last updated.",
				MarkdownDescription: "The timestamp when the role was last updated (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					useStateForUnknownIfConfigUnchanged{
						configAttributes: []path.Path{
							path.Root("name"),
							path.Root("description"),
							path.Root("resource_type_slug"),
							path.Root("permissions"),
						},
					},
				},
			},
		},
	}
}

func (r *EnvironmentRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EnvironmentRoleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() && !req.State.Raw.IsNull() {
		resp.Diagnostics.AddError(
			"WorkOS Environment Role Deletion Is Not Supported",
			"The WorkOS public API does not currently expose deletion for environment roles. "+
				"Terraform will not silently remove this resource from state while leaving the remote role behind. "+
				"Remove the resource from configuration only after deleting it outside Terraform, then run `terraform state rm` or `tofu state rm` for this address.",
		)
	}

	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var plan EnvironmentRoleResourceModel
	var state EnvironmentRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	slugChanged := !plan.Slug.IsUnknown() && !plan.Slug.Equal(state.Slug)
	resourceTypeChanged := !plan.ResourceTypeSlug.IsUnknown() && !plan.ResourceTypeSlug.Equal(state.ResourceTypeSlug)
	if slugChanged || resourceTypeChanged {
		resp.Diagnostics.AddError(
			"WorkOS Environment Role Replacement Is Not Supported",
			"The requested change would require replacing the environment role, but the WorkOS public API does not currently expose deletion for environment roles. "+
				"Create a new Terraform resource for the new role slug or resource type, and use `terraform state rm` or `tofu state rm` if Terraform should stop managing the existing role.",
		)
	}
}

func (r *EnvironmentRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentRoleResourceModel
	var config EnvironmentRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating environment role", map[string]any{
		"slug": plan.Slug.ValueString(),
		"name": plan.Name.ValueString(),
	})

	createReq := &client.EnvironmentRoleCreateRequest{
		Slug: plan.Slug.ValueString(),
		Name: plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}
	if !plan.ResourceTypeSlug.IsNull() && !plan.ResourceTypeSlug.IsUnknown() {
		createReq.ResourceTypeSlug = plan.ResourceTypeSlug.ValueString()
	}

	role, err := r.client.CreateEnvironmentRole(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Environment Role",
			"Could not create environment role, unexpected error: "+err.Error(),
		)
		return
	}

	if !config.Permissions.IsNull() {
		permissions, diags := environmentRolePermissionsSlice(ctx, plan.Permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		role, err = r.client.SetEnvironmentRolePermissions(ctx, plan.Slug.ValueString(), permissions)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Setting Environment Role Permissions",
				"Could not set environment role permissions after creating the role, unexpected error: "+err.Error(),
			)
			return
		}
	}

	resp.Diagnostics.Append(applyEnvironmentRoleToResourceModel(ctx, role, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Created environment role", map[string]any{
		"id":   role.ID,
		"slug": role.Slug,
		"name": role.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EnvironmentRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading environment role", map[string]any{
		"slug": state.Slug.ValueString(),
	})

	role, err := r.client.GetEnvironmentRole(ctx, state.Slug.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Environment role not found, removing from state", map[string]any{
				"slug": state.Slug.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Environment Role",
			"Could not read environment role "+state.Slug.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(applyEnvironmentRoleToResourceModel(ctx, role, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EnvironmentRoleResourceModel
	var state EnvironmentRoleResourceModel
	var config EnvironmentRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating environment role", map[string]any{
		"slug": state.Slug.ValueString(),
		"name": plan.Name.ValueString(),
	})

	roleChanged := !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description)
	permissionsChanged := !config.Permissions.IsNull() && !plan.Permissions.Equal(state.Permissions)

	if !roleChanged && !permissionsChanged {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	role := &client.EnvironmentRole{}

	if roleChanged {
		updateReq := &client.EnvironmentRoleUpdateRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		}

		var err error
		role, err = r.client.UpdateEnvironmentRole(ctx, state.Slug.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Environment Role",
				"Could not update environment role, unexpected error: "+err.Error(),
			)
			return
		}
	}

	if !config.Permissions.IsNull() && !plan.Permissions.Equal(state.Permissions) {
		permissions, diags := environmentRolePermissionsSlice(ctx, plan.Permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var err error
		role, err = r.client.SetEnvironmentRolePermissions(ctx, state.Slug.ValueString(), permissions)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Setting Environment Role Permissions",
				"Could not set environment role permissions, unexpected error: "+err.Error(),
			)
			return
		}
	}

	resp.Diagnostics.Append(applyEnvironmentRoleToResourceModel(ctx, role, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updated environment role", map[string]any{
		"id":   role.ID,
		"slug": role.Slug,
		"name": role.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"WorkOS Environment Role Deletion Is Not Supported",
		"The WorkOS public API does not currently expose deletion for environment roles. "+
			"Terraform will not silently remove this resource from state while leaving the remote role behind. "+
			"Remove the resource from configuration only after deleting it outside Terraform, then run `terraform state rm` or `tofu state rm` for this address.",
	)
}

func (r *EnvironmentRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing environment role", map[string]any{
		"id": req.ID,
	})

	if req.ID == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected import ID in the format 'slug', got an empty string.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), req.ID)...)
}
