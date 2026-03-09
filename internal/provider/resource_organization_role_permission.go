// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &OrganizationRolePermissionResource{}
var _ resource.ResourceWithImportState = &OrganizationRolePermissionResource{}

func NewOrganizationRolePermissionResource() resource.Resource {
	return &OrganizationRolePermissionResource{}
}

// OrganizationRolePermissionResource defines the resource implementation.
type OrganizationRolePermissionResource struct {
	client *client.Client
}

// OrganizationRolePermissionResourceModel describes the resource data model.
type OrganizationRolePermissionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	RoleSlug       types.String `tfsdk:"role_slug"`
	Permission     types.String `tfsdk:"permission"`
}

func (r *OrganizationRolePermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_role_permission"
}

func (r *OrganizationRolePermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a permission to a WorkOS Organization Role.",
		MarkdownDescription: `
Assigns a permission to a WorkOS Organization Role.

This resource manages the assignment of a permission to an organization role. All fields
are immutable — changing any field will destroy and recreate the assignment.

## Example Usage

` + "```hcl" + `
resource "workos_organization_role_permission" "assign" {
  organization_id = workos_organization.acme.id
  role_slug       = workos_organization_role.billing.slug
  permission      = workos_permission.billing_read.slug
}
` + "```" + `

## Import

Organization role permissions can be imported using a composite key of organization ID, role slug, and permission slug:

` + "```shell" + `
terraform import workos_organization_role_permission.example org_01HXYZ.../billing-admin/billing:read
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The composite identifier of the organization role permission (organization_id/role_slug/permission).",
				MarkdownDescription: "The composite identifier of the organization role permission (`organization_id/role_slug/permission`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description:         "The ID of the organization the role belongs to.",
				MarkdownDescription: "The ID of the organization the role belongs to (e.g., `org_01HXYZ...`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_slug": schema.StringAttribute{
				Description:         "The slug of the organization role to assign the permission to.",
				MarkdownDescription: "The slug of the organization role to assign the permission to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission": schema.StringAttribute{
				Description:         "The slug of the permission to assign to the role.",
				MarkdownDescription: "The slug of the permission to assign to the role.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *OrganizationRolePermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationRolePermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationRolePermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrganizationID.ValueString()
	roleSlug := plan.RoleSlug.ValueString()
	permSlug := plan.Permission.ValueString()

	tflog.Debug(ctx, "Adding permission to organization role", map[string]any{
		"organization_id": orgID,
		"role_slug":       roleSlug,
		"permission":      permSlug,
	})

	_, err := r.client.AddOrganizationRolePermission(ctx, orgID, roleSlug, permSlug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Adding Permission to Organization Role",
			"Could not add permission to organization role, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", orgID, roleSlug, permSlug))

	tflog.Info(ctx, "Added permission to organization role", map[string]any{
		"organization_id": orgID,
		"role_slug":       roleSlug,
		"permission":      permSlug,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationRolePermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationRolePermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	roleSlug := state.RoleSlug.ValueString()
	permSlug := state.Permission.ValueString()

	tflog.Debug(ctx, "Reading organization role permission", map[string]any{
		"organization_id": orgID,
		"role_slug":       roleSlug,
		"permission":      permSlug,
	})

	// Get the role and check if the permission is present
	role, err := r.client.GetOrganizationRole(ctx, orgID, roleSlug)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Organization role not found, removing permission assignment from state", map[string]any{
				"organization_id": orgID,
				"role_slug":       roleSlug,
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Organization Role Permission",
			"Could not read organization role "+roleSlug+": "+err.Error(),
		)
		return
	}

	// Check if the permission is in the role's permissions list
	found := false
	for _, p := range role.Permissions {
		if p == permSlug {
			found = true
			break
		}
	}

	if !found {
		tflog.Info(ctx, "Permission not found on role, removing from state", map[string]any{
			"organization_id": orgID,
			"role_slug":       roleSlug,
			"permission":      permSlug,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationRolePermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields have RequiresReplace, so Update should never be called.
	resp.Diagnostics.AddError(
		"Unexpected Update",
		"This resource does not support in-place updates. All changes require replacement.",
	)
}

func (r *OrganizationRolePermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationRolePermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	roleSlug := state.RoleSlug.ValueString()
	permSlug := state.Permission.ValueString()

	tflog.Debug(ctx, "Removing permission from organization role", map[string]any{
		"organization_id": orgID,
		"role_slug":       roleSlug,
		"permission":      permSlug,
	})

	err := r.client.RemoveOrganizationRolePermission(ctx, orgID, roleSlug, permSlug)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Organization role permission already removed", map[string]any{
				"organization_id": orgID,
				"role_slug":       roleSlug,
				"permission":      permSlug,
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error Removing Permission from Organization Role",
			"Could not remove permission from organization role, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Removed permission from organization role", map[string]any{
		"organization_id": orgID,
		"role_slug":       roleSlug,
		"permission":      permSlug,
	})
}

func (r *OrganizationRolePermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing organization role permission", map[string]any{
		"id": req.ID,
	})

	// Parse composite key: organization_id/role_slug/permission
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format 'organization_id/role_slug/permission', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_slug"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
