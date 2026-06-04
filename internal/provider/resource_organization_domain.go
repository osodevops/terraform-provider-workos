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

var _ resource.Resource = &OrganizationDomainResource{}
var _ resource.ResourceWithImportState = &OrganizationDomainResource{}

func NewOrganizationDomainResource() resource.Resource {
	return &OrganizationDomainResource{}
}

type OrganizationDomainResource struct {
	client *client.Client
}

type OrganizationDomainResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	OrganizationID       types.String `tfsdk:"organization_id"`
	Domain               types.String `tfsdk:"domain"`
	Verify               types.Bool   `tfsdk:"verify"`
	State                types.String `tfsdk:"state"`
	VerificationPrefix   types.String `tfsdk:"verification_prefix"`
	VerificationToken    types.String `tfsdk:"verification_token"`
	VerificationStrategy types.String `tfsdk:"verification_strategy"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func (r *OrganizationDomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_domain"
}

func (r *OrganizationDomainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS Organization Domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the organization domain.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The ID of the organization to add the domain to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "The domain name to add to the organization.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"verify": schema.BoolAttribute{
				Description: "Whether to initiate WorkOS domain verification after create or when toggled to true.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				Description: "The domain verification state.",
				Computed:    true,
			},
			"verification_prefix": schema.StringAttribute{
				Description: "The DNS verification prefix.",
				Computed:    true,
			},
			"verification_token": schema.StringAttribute{
				Description: "The DNS verification token.",
				Computed:    true,
			},
			"verification_strategy": schema.StringAttribute{
				Description: "The verification strategy for the domain.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the organization domain was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the organization domain was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *OrganizationDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.CreateOrganizationDomain(ctx, &client.OrganizationDomainCreateRequest{
		Domain:         plan.Domain.ValueString(),
		OrganizationID: plan.OrganizationID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Organization Domain", "Could not create organization domain: "+err.Error())
		return
	}

	if !plan.Verify.IsNull() && !plan.Verify.IsUnknown() && plan.Verify.ValueBool() {
		domain, err = r.client.VerifyOrganizationDomain(ctx, domain.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error Verifying Organization Domain", "Could not verify organization domain: "+err.Error())
			return
		}
	}

	organizationDomainToState(&plan, domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.GetOrganizationDomain(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Organization Domain", "Could not read organization domain: "+err.Error())
		return
	}

	organizationDomainToState(&state, domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationDomainResourceModel
	var state OrganizationDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var domain *client.OrganizationDomain
	var err error
	if !plan.Verify.Equal(state.Verify) && !plan.Verify.IsNull() && plan.Verify.ValueBool() {
		domain, err = r.client.VerifyOrganizationDomain(ctx, state.ID.ValueString())
	} else {
		domain, err = r.client.GetOrganizationDomain(ctx, state.ID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Organization Domain", "Could not update organization domain: "+err.Error())
		return
	}

	organizationDomainToState(&plan, domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrganizationDomain(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Organization domain already deleted", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error Deleting Organization Domain", "Could not delete organization domain: "+err.Error())
	}
}

func (r *OrganizationDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func organizationDomainToState(state *OrganizationDomainResourceModel, domain *client.OrganizationDomain) {
	state.ID = types.StringValue(domain.ID)
	state.OrganizationID = types.StringValue(domain.OrganizationID)
	state.Domain = types.StringValue(domain.Domain)
	if state.Verify.IsUnknown() {
		state.Verify = types.BoolValue(false)
	}
	state.State = optionalString(domain.State)
	state.VerificationPrefix = optionalString(domain.VerificationPrefix)
	state.VerificationToken = optionalString(domain.VerificationToken)
	state.VerificationStrategy = optionalString(domain.VerificationStrategy)
	state.CreatedAt = types.StringValue(domain.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(domain.UpdatedAt.Format(time.RFC3339))
}

func optionalString(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}
	return types.StringValue(*value)
}
