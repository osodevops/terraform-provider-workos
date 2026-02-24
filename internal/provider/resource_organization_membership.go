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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &OrganizationMembershipResource{}
var _ resource.ResourceWithImportState = &OrganizationMembershipResource{}

func NewOrganizationMembershipResource() resource.Resource {
	return &OrganizationMembershipResource{}
}

// OrganizationMembershipResource defines the resource implementation.
type OrganizationMembershipResource struct {
	client *client.Client
}

// OrganizationMembershipResourceModel describes the resource data model.
type OrganizationMembershipResourceModel struct {
	ID             types.String `tfsdk:"id"`
	UserID         types.String `tfsdk:"user_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	RoleSlug       types.String `tfsdk:"role_slug"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *OrganizationMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_membership"
}

func (r *OrganizationMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS Organization Membership.",
		MarkdownDescription: `
Manages a WorkOS Organization Membership.

Organization memberships associate users with organizations and define their
role within that organization. A user can be a member of multiple organizations.

## Example Usage

### Basic Membership

` + "```hcl" + `
resource "workos_organization_membership" "example" {
  user_id         = workos_user.example.id
  organization_id = workos_organization.example.id
}
` + "```" + `

### Membership with Role

` + "```hcl" + `
resource "workos_organization_membership" "admin" {
  user_id         = workos_user.admin.id
  organization_id = workos_organization.example.id
  role_slug       = "admin"
}
` + "```" + `

## Import

Organization memberships can be imported using the membership ID:

` + "```shell" + `
terraform import workos_organization_membership.example om_01HXYZ...
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the organization membership.",
				MarkdownDescription: "The unique identifier of the organization membership (e.g., `om_01HXYZ...`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				Description:         "The ID of the user to add to the organization.",
				MarkdownDescription: "The ID of the user to add to the organization.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description:         "The ID of the organization to add the user to.",
				MarkdownDescription: "The ID of the organization to add the user to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_slug": schema.StringAttribute{
				Description:         "The slug of the role to assign to the user within the organization.",
				MarkdownDescription: "The slug of the role to assign to the user within the organization (e.g., `admin`, `member`).",
				Optional:            true,
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "The status of the membership.",
				MarkdownDescription: "The status of the membership (`active`, `inactive`, `pending`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the membership was created.",
				MarkdownDescription: "The timestamp when the membership was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the membership was last updated.",
				MarkdownDescription: "The timestamp when the membership was last updated (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *OrganizationMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationMembershipResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating organization membership", map[string]any{
		"user_id":         plan.UserID.ValueString(),
		"organization_id": plan.OrganizationID.ValueString(),
	})

	createReq := &client.OrganizationMembershipCreateRequest{
		UserID:         plan.UserID.ValueString(),
		OrganizationID: plan.OrganizationID.ValueString(),
	}

	if !plan.RoleSlug.IsNull() {
		createReq.RoleSlug = plan.RoleSlug.ValueString()
	}

	membership, err := r.client.CreateOrganizationMembership(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Organization Membership",
			"Could not create organization membership, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(membership.ID)
	plan.UserID = types.StringValue(membership.UserID)
	plan.OrganizationID = types.StringValue(membership.OrganizationID)
	// The API may not return role_slug in the response, so preserve the plan value
	// if it was set, since we know it was applied during creation.
	if membership.RoleSlug != "" {
		plan.RoleSlug = types.StringValue(membership.RoleSlug)
	} else if !plan.RoleSlug.IsNull() && plan.RoleSlug.ValueString() != "" {
		// Preserve plan value - API accepted it but didn't return it
	} else {
		plan.RoleSlug = types.StringNull()
	}
	plan.Status = types.StringValue(membership.Status)
	plan.CreatedAt = types.StringValue(membership.CreatedAt.Format(time.RFC3339))
	plan.UpdatedAt = types.StringValue(membership.UpdatedAt.Format(time.RFC3339))

	tflog.Info(ctx, "Created organization membership", map[string]any{
		"id":              membership.ID,
		"user_id":         membership.UserID,
		"organization_id": membership.OrganizationID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationMembershipResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading organization membership", map[string]any{
		"id": state.ID.ValueString(),
	})

	membership, err := r.client.GetOrganizationMembership(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Organization membership not found, removing from state", map[string]any{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Organization Membership",
			"Could not read organization membership ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state
	state.UserID = types.StringValue(membership.UserID)
	state.OrganizationID = types.StringValue(membership.OrganizationID)
	// The API may not return role_slug in the response, so preserve the
	// existing state value if the API returns empty.
	if membership.RoleSlug != "" {
		state.RoleSlug = types.StringValue(membership.RoleSlug)
	} else if !state.RoleSlug.IsNull() && state.RoleSlug.ValueString() != "" {
		// Preserve existing state value - API didn't return it but it was set
	} else {
		state.RoleSlug = types.StringNull()
	}
	state.Status = types.StringValue(membership.Status)
	state.CreatedAt = types.StringValue(membership.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(membership.UpdatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Organization memberships cannot be updated - user_id and organization_id
	// both require replacement. The only updatable field would be role_slug,
	// but WorkOS API doesn't currently support updating membership roles directly.
	// For now, we just read the current state.
	var plan OrganizationMembershipResourceModel
	var state OrganizationMembershipResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating organization membership", map[string]any{
		"id": state.ID.ValueString(),
	})

	// Read current state from API
	membership, err := r.client.GetOrganizationMembership(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Organization Membership",
			"Could not read organization membership: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = state.ID
	plan.UserID = types.StringValue(membership.UserID)
	plan.OrganizationID = types.StringValue(membership.OrganizationID)
	if membership.RoleSlug != "" {
		plan.RoleSlug = types.StringValue(membership.RoleSlug)
	}
	plan.Status = types.StringValue(membership.Status)
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(membership.UpdatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationMembershipResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting organization membership", map[string]any{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteOrganizationMembership(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Organization membership already deleted", map[string]any{
				"id": state.ID.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Organization Membership",
			"Could not delete organization membership, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted organization membership", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *OrganizationMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing organization membership", map[string]any{
		"id": req.ID,
	})

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
