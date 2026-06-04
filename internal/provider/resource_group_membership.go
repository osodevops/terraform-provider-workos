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

var _ resource.Resource = &GroupMembershipResource{}
var _ resource.ResourceWithImportState = &GroupMembershipResource{}

func NewGroupMembershipResource() resource.Resource {
	return &GroupMembershipResource{}
}

type GroupMembershipResource struct {
	client *client.Client
}

type GroupMembershipResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	OrganizationID           types.String `tfsdk:"organization_id"`
	GroupID                  types.String `tfsdk:"group_id"`
	OrganizationMembershipID types.String `tfsdk:"organization_membership_id"`
	UserID                   types.String `tfsdk:"user_id"`
	Status                   types.String `tfsdk:"status"`
	CreatedAt                types.String `tfsdk:"created_at"`
	UpdatedAt                types.String `tfsdk:"updated_at"`
}

func (r *GroupMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *GroupMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Adds a WorkOS organization membership to a group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite ID for the group membership.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: "The group ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_membership_id": schema.StringAttribute{
				Description: "The organization membership ID to add to the group.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "The user ID on the organization membership.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The organization membership status.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the organization membership was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the organization membership was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *GroupMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.AddGroupMembership(ctx, plan.OrganizationID.ValueString(), plan.GroupID.ValueString(), &client.GroupMembershipCreateRequest{
		OrganizationMembershipID: plan.OrganizationMembershipID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Group Membership", "Could not add organization membership to group: "+err.Error())
		return
	}

	membership, err := r.findGroupMembership(ctx, plan.OrganizationID.ValueString(), plan.GroupID.ValueString(), plan.OrganizationMembershipID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Group Membership", "Could not confirm group membership after create: "+err.Error())
		return
	}

	groupMembershipToState(&plan, membership)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	membership, err := r.findGroupMembership(ctx, state.OrganizationID.ValueString(), state.GroupID.ValueString(), state.OrganizationMembershipID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Group Membership", "Could not read group membership: "+err.Error())
		return
	}

	groupMembershipToState(&state, membership)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteGroupMembership(ctx, state.OrganizationID.ValueString(), state.GroupID.ValueString(), state.OrganizationMembershipID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Group membership already deleted", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error Deleting Group Membership", "Could not delete group membership: "+err.Error())
	}
}

func (r *GroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitCompositeID(req.ID, 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid Import ID", "Expected import ID in the format 'organization_id/group_id/organization_membership_id'.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_membership_id"), parts[2])...)
}

func (r *GroupMembershipResource) findGroupMembership(ctx context.Context, organizationID, groupID, organizationMembershipID string) (*client.OrganizationMembership, error) {
	resp, err := r.client.ListGroupMemberships(ctx, organizationID, groupID)
	if err != nil {
		return nil, err
	}

	for _, membership := range resp.Data {
		if membership.ID == organizationMembershipID {
			return &membership, nil
		}
	}

	return nil, &client.APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("organization membership %s is not in group %s", organizationMembershipID, groupID),
	}
}

func groupMembershipToState(state *GroupMembershipResourceModel, membership *client.OrganizationMembership) {
	state.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", state.OrganizationID.ValueString(), state.GroupID.ValueString(), state.OrganizationMembershipID.ValueString()))
	state.OrganizationMembershipID = types.StringValue(membership.ID)
	state.UserID = types.StringValue(membership.UserID)
	state.Status = types.StringValue(membership.Status)
	state.CreatedAt = types.StringValue(membership.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(membership.UpdatedAt.Format(time.RFC3339))
}
