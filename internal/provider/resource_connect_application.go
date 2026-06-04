package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

var _ resource.Resource = &ConnectApplicationResource{}
var _ resource.ResourceWithImportState = &ConnectApplicationResource{}

func NewConnectApplicationResource() resource.Resource {
	return &ConnectApplicationResource{}
}

type ConnectApplicationResource struct {
	client *client.Client
}

type ConnectApplicationResourceModel struct {
	ID                       types.String                         `tfsdk:"id"`
	ClientID                 types.String                         `tfsdk:"client_id"`
	Name                     types.String                         `tfsdk:"name"`
	Description              types.String                         `tfsdk:"description"`
	ApplicationType          types.String                         `tfsdk:"application_type"`
	OrganizationID           types.String                         `tfsdk:"organization_id"`
	IsFirstParty             types.Bool                           `tfsdk:"is_first_party"`
	UsesPKCE                 types.Bool                           `tfsdk:"uses_pkce"`
	WasDynamicallyRegistered types.Bool                           `tfsdk:"was_dynamically_registered"`
	Scopes                   types.List                           `tfsdk:"scopes"`
	RedirectURIs             []ConnectApplicationRedirectURIModel `tfsdk:"redirect_uris"`
	CreatedAt                types.String                         `tfsdk:"created_at"`
	UpdatedAt                types.String                         `tfsdk:"updated_at"`
}

type ConnectApplicationRedirectURIModel struct {
	URI     types.String `tfsdk:"uri"`
	Default types.Bool   `tfsdk:"default"`
}

func (r *ConnectApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_application"
}

func (r *ConnectApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS Connect application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the Connect application.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID of the Connect application.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The Connect application name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "An optional Connect application description.",
				Optional:    true,
				Computed:    true,
			},
			"application_type": schema.StringAttribute{
				Description: "The Connect application type: oauth or m2m.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID for organization-scoped applications.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_first_party": schema.BoolAttribute{
				Description: "Whether an OAuth application is first-party.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"uses_pkce": schema.BoolAttribute{
				Description: "Whether an OAuth application uses PKCE.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"was_dynamically_registered": schema.BoolAttribute{
				Description: "Whether the application was dynamically registered.",
				Computed:    true,
			},
			"scopes": schema.ListAttribute{
				Description: "OAuth scopes granted to the application.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"redirect_uris": schema.ListNestedAttribute{
				Description: "Redirect URIs configured for an OAuth application.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uri": schema.StringAttribute{
							Description: "The redirect URI.",
							Required:    true,
						},
						"default": schema.BoolAttribute{
							Description: "Whether this redirect URI is the default.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the application was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the application was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *ConnectApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConnectApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConnectApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !validateConnectApplicationPlan(&plan, &resp.Diagnostics) {
		return
	}

	scopes, diags := stringListFromTerraform(ctx, plan.Scopes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.ConnectApplicationCreateRequest{
		ApplicationType: plan.ApplicationType.ValueString(),
		Name:            plan.Name.ValueString(),
		Scopes:          scopes,
		RedirectURIs:    redirectURIInputs(plan.RedirectURIs),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}
	if !plan.OrganizationID.IsNull() && !plan.OrganizationID.IsUnknown() {
		createReq.OrganizationID = plan.OrganizationID.ValueString()
	}
	if !plan.IsFirstParty.IsNull() && !plan.IsFirstParty.IsUnknown() {
		value := plan.IsFirstParty.ValueBool()
		createReq.IsFirstParty = &value
	} else if plan.ApplicationType.ValueString() == "oauth" {
		value := true
		createReq.IsFirstParty = &value
	}
	if !plan.UsesPKCE.IsNull() && !plan.UsesPKCE.IsUnknown() {
		value := plan.UsesPKCE.ValueBool()
		createReq.UsesPKCE = &value
	}

	app, err := r.client.CreateConnectApplication(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Connect Application", "Could not create Connect application: "+err.Error())
		return
	}

	connectApplicationToState(ctx, &plan, app, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConnectApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConnectApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetConnectApplication(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Connect Application", "Could not read Connect application: "+err.Error())
		return
	}

	connectApplicationToState(ctx, &state, app, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConnectApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ConnectApplicationResourceModel
	var state ConnectApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !validateConnectApplicationPlan(&plan, &resp.Diagnostics) {
		return
	}

	scopes, diags := stringListFromTerraform(ctx, plan.Scopes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.ConnectApplicationUpdateRequest{
		Name:         plan.Name.ValueString(),
		Scopes:       scopes,
		RedirectURIs: redirectURIInputs(plan.RedirectURIs),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		updateReq.Description = plan.Description.ValueString()
	}

	app, err := r.client.UpdateConnectApplication(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Connect Application", "Could not update Connect application: "+err.Error())
		return
	}

	connectApplicationToState(ctx, &plan, app, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConnectApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConnectApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteConnectApplication(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Connect application already deleted", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error Deleting Connect Application", "Could not delete Connect application: "+err.Error())
	}
}

func (r *ConnectApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func validateConnectApplicationPlan(plan *ConnectApplicationResourceModel, diags *diag.Diagnostics) bool {
	applicationType := plan.ApplicationType.ValueString()
	if applicationType != "oauth" && applicationType != "m2m" {
		diags.AddAttributeError(path.Root("application_type"), "Invalid Application Type", "application_type must be either 'oauth' or 'm2m'.")
		return false
	}

	if applicationType == "m2m" && (plan.OrganizationID.IsNull() || plan.OrganizationID.IsUnknown() || plan.OrganizationID.ValueString() == "") {
		diags.AddAttributeError(path.Root("organization_id"), "Missing Organization ID", "organization_id is required for m2m Connect applications.")
		return false
	}

	if applicationType == "m2m" && len(plan.RedirectURIs) > 0 {
		diags.AddAttributeError(path.Root("redirect_uris"), "Unsupported Redirect URIs", "redirect_uris can only be configured for oauth Connect applications.")
		return false
	}
	if applicationType == "oauth" &&
		!plan.IsFirstParty.IsNull() &&
		!plan.IsFirstParty.IsUnknown() &&
		!plan.IsFirstParty.ValueBool() &&
		(plan.OrganizationID.IsNull() || plan.OrganizationID.IsUnknown() || plan.OrganizationID.ValueString() == "") {
		diags.AddAttributeError(path.Root("organization_id"), "Missing Organization ID", "organization_id is required when is_first_party is false.")
		return false
	}

	return true
}

func stringListFromTerraform(ctx context.Context, value types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return nil, diags
	}

	var values []string
	diags.Append(value.ElementsAs(ctx, &values, false)...)
	return values, diags
}

func redirectURIInputs(values []ConnectApplicationRedirectURIModel) []client.ConnectApplicationRedirectURIInput {
	if len(values) == 0 {
		return nil
	}

	inputs := make([]client.ConnectApplicationRedirectURIInput, 0, len(values))
	for _, value := range values {
		input := client.ConnectApplicationRedirectURIInput{
			URI: value.URI.ValueString(),
		}
		if !value.Default.IsNull() && !value.Default.IsUnknown() {
			defaultValue := value.Default.ValueBool()
			input.Default = &defaultValue
		}
		inputs = append(inputs, input)
	}
	return inputs
}

func connectApplicationToState(ctx context.Context, state *ConnectApplicationResourceModel, app *client.ConnectApplication, diags *diag.Diagnostics) {
	state.ID = types.StringValue(app.ID)
	state.ClientID = types.StringValue(app.ClientID)
	state.Name = types.StringValue(app.Name)
	state.Description = optionalString(app.Description)
	state.ApplicationType = optionalString(app.ApplicationType)
	state.OrganizationID = optionalString(app.OrganizationID)
	state.IsFirstParty = optionalBool(app.IsFirstParty)
	state.UsesPKCE = optionalBool(app.UsesPKCE)
	state.WasDynamicallyRegistered = optionalBool(app.WasDynamicallyRegistered)

	if len(app.Scopes) > 0 {
		scopes, scopeDiags := types.ListValueFrom(ctx, types.StringType, app.Scopes)
		diags.Append(scopeDiags...)
		state.Scopes = scopes
	} else {
		state.Scopes, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	state.RedirectURIs = make([]ConnectApplicationRedirectURIModel, 0, len(app.RedirectURIs))
	for _, redirectURI := range app.RedirectURIs {
		state.RedirectURIs = append(state.RedirectURIs, ConnectApplicationRedirectURIModel{
			URI:     types.StringValue(redirectURI.URI),
			Default: types.BoolValue(redirectURI.Default),
		})
	}

	state.CreatedAt = types.StringValue(app.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(app.UpdatedAt.Format(time.RFC3339))
}

func optionalBool(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}
