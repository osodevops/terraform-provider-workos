package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
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

var _ resource.Resource = &AuthorizationResourceResource{}
var _ resource.ResourceWithConfigValidators = &AuthorizationResourceResource{}
var _ resource.ResourceWithImportState = &AuthorizationResourceResource{}

func NewAuthorizationResourceResource() resource.Resource {
	return &AuthorizationResourceResource{}
}

type AuthorizationResourceResource struct {
	client *client.Client
}

type AuthorizationResourceResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	OrganizationID           types.String `tfsdk:"organization_id"`
	ExternalID               types.String `tfsdk:"external_id"`
	ResourceTypeSlug         types.String `tfsdk:"resource_type_slug"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	ParentResourceID         types.String `tfsdk:"parent_resource_id"`
	ParentResourceTypeSlug   types.String `tfsdk:"parent_resource_type_slug"`
	ParentResourceExternalID types.String `tfsdk:"parent_resource_external_id"`
	CascadeDelete            types.Bool   `tfsdk:"cascade_delete"`
	CreatedAt                types.String `tfsdk:"created_at"`
	UpdatedAt                types.String `tfsdk:"updated_at"`
}

func (r *AuthorizationResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authorization_resource"
}

func (r *AuthorizationResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS authorization resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the authorization resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID that owns the resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"external_id": schema.StringAttribute{
				Description: "The external identifier for the resource in your system.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_type_slug": schema.StringAttribute{
				Description: "The resource type slug.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The display name for the resource.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "An optional resource description.",
				Optional:    true,
				Computed:    true,
			},
			"parent_resource_id": schema.StringAttribute{
				Description: "The parent resource ID.",
				Optional:    true,
				Computed:    true,
			},
			"parent_resource_type_slug": schema.StringAttribute{
				Description: "The parent resource type slug when identifying the parent by external ID.",
				Optional:    true,
			},
			"parent_resource_external_id": schema.StringAttribute{
				Description: "The parent resource external ID when identifying the parent by external ID.",
				Optional:    true,
			},
			"cascade_delete": schema.BoolAttribute{
				Description: "Whether delete should cascade to descendant resources and role assignments.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the resource was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the resource was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *AuthorizationResourceResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("parent_resource_id"),
			path.MatchRoot("parent_resource_type_slug"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("parent_resource_id"),
			path.MatchRoot("parent_resource_external_id"),
		),
		resourcevalidator.RequiredTogether(
			path.MatchRoot("parent_resource_type_slug"),
			path.MatchRoot("parent_resource_external_id"),
		),
	}
}

func (r *AuthorizationResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AuthorizationResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthorizationResourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.AuthorizationResourceCreateRequest{
		OrganizationID:   plan.OrganizationID.ValueString(),
		ExternalID:       plan.ExternalID.ValueString(),
		ResourceTypeSlug: plan.ResourceTypeSlug.ValueString(),
		Name:             plan.Name.ValueString(),
	}
	applyAuthorizationResourceParentToCreate(&plan, createReq)
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}

	resource, err := r.client.CreateAuthorizationResource(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Authorization Resource", "Could not create authorization resource: "+err.Error())
		return
	}

	authorizationResourceToState(&plan, resource)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthorizationResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthorizationResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resource, err := r.client.GetAuthorizationResource(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Authorization Resource", "Could not read authorization resource: "+err.Error())
		return
	}

	authorizationResourceToState(&state, resource)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AuthorizationResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuthorizationResourceResourceModel
	var state AuthorizationResourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.AuthorizationResourceUpdateRequest{Name: plan.Name.ValueString()}
	applyAuthorizationResourceParentToUpdate(&plan, updateReq)
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		updateReq.Description = plan.Description.ValueString()
	}

	resource, err := r.client.UpdateAuthorizationResource(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Authorization Resource", "Could not update authorization resource: "+err.Error())
		return
	}

	authorizationResourceToState(&plan, resource)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthorizationResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthorizationResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cascadeDelete := !state.CascadeDelete.IsNull() && state.CascadeDelete.ValueBool()
	err := r.client.DeleteAuthorizationResource(ctx, state.ID.ValueString(), cascadeDelete)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Authorization resource already deleted", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error Deleting Authorization Resource", "Could not delete authorization resource: "+err.Error())
	}
}

func (r *AuthorizationResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyAuthorizationResourceParentToCreate(plan *AuthorizationResourceResourceModel, req *client.AuthorizationResourceCreateRequest) {
	if !plan.ParentResourceID.IsNull() && !plan.ParentResourceID.IsUnknown() {
		req.ParentResourceID = plan.ParentResourceID.ValueString()
		return
	}
	if !plan.ParentResourceTypeSlug.IsNull() && !plan.ParentResourceTypeSlug.IsUnknown() {
		req.ParentResourceTypeSlug = plan.ParentResourceTypeSlug.ValueString()
		req.ParentResourceExternalID = plan.ParentResourceExternalID.ValueString()
	}
}

func applyAuthorizationResourceParentToUpdate(plan *AuthorizationResourceResourceModel, req *client.AuthorizationResourceUpdateRequest) {
	if !plan.ParentResourceID.IsNull() && !plan.ParentResourceID.IsUnknown() {
		req.ParentResourceID = plan.ParentResourceID.ValueString()
		return
	}
	if !plan.ParentResourceTypeSlug.IsNull() && !plan.ParentResourceTypeSlug.IsUnknown() {
		req.ParentResourceTypeSlug = plan.ParentResourceTypeSlug.ValueString()
		req.ParentResourceExternalID = plan.ParentResourceExternalID.ValueString()
	}
}

func authorizationResourceToState(state *AuthorizationResourceResourceModel, resource *client.AuthorizationResource) {
	state.ID = types.StringValue(resource.ID)
	state.OrganizationID = types.StringValue(resource.OrganizationID)
	state.ExternalID = types.StringValue(resource.ExternalID)
	state.ResourceTypeSlug = types.StringValue(resource.ResourceTypeSlug)
	state.Name = types.StringValue(resource.Name)
	state.Description = optionalString(resource.Description)
	state.ParentResourceID = optionalString(resource.ParentResourceID)
	if state.CascadeDelete.IsNull() || state.CascadeDelete.IsUnknown() {
		state.CascadeDelete = types.BoolValue(false)
	}
	state.CreatedAt = types.StringValue(resource.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(resource.UpdatedAt.Format(time.RFC3339))
}
