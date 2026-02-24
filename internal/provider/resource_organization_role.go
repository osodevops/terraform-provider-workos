// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &OrganizationRoleResource{}
var _ resource.ResourceWithImportState = &OrganizationRoleResource{}

func NewOrganizationRoleResource() resource.Resource {
	return &OrganizationRoleResource{}
}

// OrganizationRoleResource defines the resource implementation.
type OrganizationRoleResource struct {
	client *client.Client
}

// OrganizationRoleResourceModel describes the resource data model.
type OrganizationRoleResourceModel struct {
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

func (r *OrganizationRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_role"
}

func (r *OrganizationRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS Organization Role.",
		MarkdownDescription: `
Manages a WorkOS Organization Role.

Organization roles define authorization levels within an organization and can be assigned
to organization memberships. Roles are identified by their slug within an organization.

## Example Usage

` + "```hcl" + `
resource "workos_organization_role" "billing_admin" {
  organization_id = workos_organization.example.id
  slug            = "org-billing-admin"
  name            = "Billing Admin"
  description     = "Can manage billing and invoices"
}
` + "```" + `

## Import

Organization roles can be imported using a composite key of organization ID and slug:

` + "```shell" + `
terraform import workos_organization_role.example org_01HXYZ.../org-billing-admin
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the organization role.",
				MarkdownDescription: "The unique identifier of the organization role.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description:         "The ID of the organization this role belongs to.",
				MarkdownDescription: "The ID of the organization this role belongs to (e.g., `org_01HXYZ...`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"slug": schema.StringAttribute{
				Description:         "The slug identifier for the role. Must be unique within the organization.",
				MarkdownDescription: "The slug identifier for the role. Must be unique within the organization.",
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
			"permissions": schema.ListAttribute{
				Description:         "The permissions associated with the role.",
				MarkdownDescription: "The permissions associated with the role.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
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
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *OrganizationRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *OrganizationRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationRoleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating organization role", map[string]any{
		"organization_id": plan.OrganizationID.ValueString(),
		"slug":            plan.Slug.ValueString(),
		"name":            plan.Name.ValueString(),
	})

	// Build the create request
	createReq := &client.OrganizationRoleCreateRequest{
		Slug: plan.Slug.ValueString(),
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}

	// Create the organization role
	role, err := r.client.CreateOrganizationRole(ctx, plan.OrganizationID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Organization Role",
			"Could not create organization role, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(role.ID)
	plan.Type = types.StringValue(role.Type)
	plan.Description = types.StringValue(role.Description)
	plan.CreatedAt = types.StringValue(role.CreatedAt.Format(time.RFC3339))
	plan.UpdatedAt = types.StringValue(role.UpdatedAt.Format(time.RFC3339))

	// Map permissions - always set as empty list rather than null
	if len(role.Permissions) > 0 {
		permissions, diags := types.ListValueFrom(ctx, types.StringType, role.Permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Permissions = permissions
	} else {
		plan.Permissions, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	tflog.Info(ctx, "Created organization role", map[string]any{
		"id":   role.ID,
		"slug": role.Slug,
		"name": role.Name,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationRoleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading organization role", map[string]any{
		"organization_id": state.OrganizationID.ValueString(),
		"slug":            state.Slug.ValueString(),
	})

	// Get the organization role from API
	role, err := r.client.GetOrganizationRole(ctx, state.OrganizationID.ValueString(), state.Slug.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Organization role not found, removing from state", map[string]any{
				"organization_id": state.OrganizationID.ValueString(),
				"slug":            state.Slug.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Organization Role",
			"Could not read organization role "+state.Slug.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state
	state.ID = types.StringValue(role.ID)
	state.Slug = types.StringValue(role.Slug)
	state.Name = types.StringValue(role.Name)
	state.Description = types.StringValue(role.Description)
	state.Type = types.StringValue(role.Type)
	state.CreatedAt = types.StringValue(role.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(role.UpdatedAt.Format(time.RFC3339))

	// Map permissions - always set as empty list rather than null
	if len(role.Permissions) > 0 {
		permissions, diags := types.ListValueFrom(ctx, types.StringType, role.Permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Permissions = permissions
	} else {
		state.Permissions, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationRoleResourceModel
	var state OrganizationRoleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating organization role", map[string]any{
		"organization_id": state.OrganizationID.ValueString(),
		"slug":            state.Slug.ValueString(),
		"name":            plan.Name.ValueString(),
	})

	// Skip update if no user-configurable attributes changed
	if plan.Name.Equal(state.Name) && plan.Description.Equal(state.Description) {
		plan.ID = state.ID
		plan.CreatedAt = state.CreatedAt
		plan.UpdatedAt = state.UpdatedAt
		plan.Type = state.Type
		plan.Permissions = state.Permissions
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	// Build the update request
	updateReq := &client.OrganizationRoleUpdateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	// Update the organization role
	role, err := r.client.UpdateOrganizationRole(ctx, state.OrganizationID.ValueString(), state.Slug.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Organization Role",
			"Could not update organization role, unexpected error: "+err.Error(),
		)
		return
	}

	// Update the plan with response data
	plan.ID = state.ID
	plan.CreatedAt = state.CreatedAt
	plan.Type = state.Type
	plan.Description = types.StringValue(role.Description)
	plan.UpdatedAt = types.StringValue(role.UpdatedAt.Format(time.RFC3339))

	// Map permissions - always set as empty list rather than null
	if len(role.Permissions) > 0 {
		permissions, diags := types.ListValueFrom(ctx, types.StringType, role.Permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Permissions = permissions
	} else {
		plan.Permissions, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	tflog.Info(ctx, "Updated organization role", map[string]any{
		"id":   role.ID,
		"slug": role.Slug,
		"name": role.Name,
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationRoleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting organization role", map[string]any{
		"organization_id": state.OrganizationID.ValueString(),
		"slug":            state.Slug.ValueString(),
	})

	// Delete the organization role
	err := r.client.DeleteOrganizationRole(ctx, state.OrganizationID.ValueString(), state.Slug.ValueString())
	if err != nil {
		// If the resource is already gone, that's fine
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Organization role already deleted", map[string]any{
				"organization_id": state.OrganizationID.ValueString(),
				"slug":            state.Slug.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Organization Role",
			"Could not delete organization role, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted organization role", map[string]any{
		"organization_id": state.OrganizationID.ValueString(),
		"slug":            state.Slug.ValueString(),
	})
}

func (r *OrganizationRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing organization role", map[string]any{
		"id": req.ID,
	})

	// Parse composite key: org_id/slug
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format 'organization_id/slug', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), parts[1])...)
}
