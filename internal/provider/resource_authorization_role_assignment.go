package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

var _ resource.Resource = &AuthorizationRoleAssignmentResource{}
var _ resource.ResourceWithConfigValidators = &AuthorizationRoleAssignmentResource{}
var _ resource.ResourceWithImportState = &AuthorizationRoleAssignmentResource{}

func NewAuthorizationRoleAssignmentResource() resource.Resource {
	return &AuthorizationRoleAssignmentResource{}
}

type AuthorizationRoleAssignmentResource struct {
	client *client.Client
}

type AuthorizationRoleAssignmentResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	OrganizationMembershipID types.String `tfsdk:"organization_membership_id"`
	RoleSlug                 types.String `tfsdk:"role_slug"`
	ResourceID               types.String `tfsdk:"resource_id"`
	ResourceTypeSlug         types.String `tfsdk:"resource_type_slug"`
	ResourceExternalID       types.String `tfsdk:"resource_external_id"`
	CreatedAt                types.String `tfsdk:"created_at"`
	UpdatedAt                types.String `tfsdk:"updated_at"`
}

func (r *AuthorizationRoleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authorization_role_assignment"
}

func (r *AuthorizationRoleAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a WorkOS authorization role to an organization membership on a resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the role assignment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_membership_id": schema.StringAttribute{
				Description: "The organization membership ID receiving the role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_slug": schema.StringAttribute{
				Description: "The role slug to assign.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: "The resource ID to assign the role on.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_type_slug": schema.StringAttribute{
				Description: "The resource type slug when identifying the resource by external ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_external_id": schema.StringAttribute{
				Description: "The resource external ID when identifying the resource by external ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the role assignment was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the role assignment was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *AuthorizationRoleAssignmentResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("resource_id"),
			path.MatchRoot("resource_external_id"),
		),
		resourcevalidator.RequiredTogether(
			path.MatchRoot("resource_type_slug"),
			path.MatchRoot("resource_external_id"),
		),
	}
}

func (r *AuthorizationRoleAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AuthorizationRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthorizationRoleAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.AuthorizationRoleAssignmentCreateRequest{
		RoleSlug: plan.RoleSlug.ValueString(),
	}
	if !plan.ResourceID.IsNull() && !plan.ResourceID.IsUnknown() {
		createReq.ResourceID = plan.ResourceID.ValueString()
	} else {
		createReq.ResourceTypeSlug = plan.ResourceTypeSlug.ValueString()
		createReq.ResourceExternalID = plan.ResourceExternalID.ValueString()
	}

	assignment, err := r.client.AssignAuthorizationRole(ctx, plan.OrganizationMembershipID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Authorization Role Assignment", "Could not create role assignment: "+err.Error())
		return
	}

	authorizationRoleAssignmentToState(&plan, assignment)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthorizationRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthorizationRoleAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	assignment, err := r.findAuthorizationRoleAssignment(ctx, state.OrganizationMembershipID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Authorization Role Assignment", "Could not read role assignment: "+err.Error())
		return
	}

	authorizationRoleAssignmentToState(&state, assignment)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AuthorizationRoleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuthorizationRoleAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthorizationRoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthorizationRoleAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAuthorizationRoleAssignment(ctx, state.OrganizationMembershipID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Authorization role assignment already deleted", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error Deleting Authorization Role Assignment", "Could not delete role assignment: "+err.Error())
	}
}

func (r *AuthorizationRoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitCompositeID(req.ID, 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Import ID", "Expected import ID in the format 'organization_membership_id/role_assignment_id'.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_membership_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *AuthorizationRoleAssignmentResource) findAuthorizationRoleAssignment(ctx context.Context, organizationMembershipID, assignmentID string) (*client.UserRoleAssignment, error) {
	resp, err := r.client.ListAuthorizationRoleAssignments(ctx, organizationMembershipID)
	if err != nil {
		return nil, err
	}

	for _, assignment := range resp.Data {
		if assignment.ID == assignmentID {
			return &assignment, nil
		}
	}

	return nil, &client.APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("role assignment %s not found for organization membership %s", assignmentID, organizationMembershipID),
	}
}

func authorizationRoleAssignmentToState(state *AuthorizationRoleAssignmentResourceModel, assignment *client.UserRoleAssignment) {
	state.ID = types.StringValue(assignment.ID)
	state.OrganizationMembershipID = types.StringValue(assignment.OrganizationMembershipID)
	if assignment.Role != nil {
		state.RoleSlug = types.StringValue(assignment.Role.Slug)
	}
	if assignment.Resource != nil {
		state.ResourceID = types.StringValue(assignment.Resource.ID)
		state.ResourceExternalID = types.StringValue(assignment.Resource.ExternalID)
		state.ResourceTypeSlug = types.StringValue(assignment.Resource.ResourceTypeSlug)
	}
	state.CreatedAt = types.StringValue(assignment.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(assignment.UpdatedAt.Format(time.RFC3339))
}
