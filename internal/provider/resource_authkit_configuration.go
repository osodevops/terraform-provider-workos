// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

var _ resource.Resource = &AuthkitConfigurationResource{}
var _ resource.ResourceWithConfigure = &AuthkitConfigurationResource{}
var _ resource.ResourceWithImportState = &AuthkitConfigurationResource{}
var _ resource.ResourceWithValidateConfig = &AuthkitConfigurationResource{}

func NewAuthkitConfigurationResource() resource.Resource { return &AuthkitConfigurationResource{} }

type AuthkitConfigurationResource struct{ client *client.Client }

type AuthkitConfigurationResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	EnvironmentID      types.String `tfsdk:"environment_id"`
	ApplicationID      types.String `tfsdk:"application_id"`
	RedirectURIs       types.Set    `tfsdk:"redirect_uris"`
	DefaultRedirectURI types.String `tfsdk:"default_redirect_uri"`
	LogoutURIs         types.Set    `tfsdk:"logout_uris"`
	DefaultLogoutURI   types.String `tfsdk:"default_logout_uri"`
	CORSOrigins        types.Set    `tfsdk:"cors_origins"`
	InitiateLoginURI   types.String `tfsdk:"initiate_login_uri"`
	AuthkitEnabled     types.Bool   `tfsdk:"authkit_enabled"`
	AllowSignup        types.Bool   `tfsdk:"allow_signup"`
	WaitlistEnabled    types.Bool   `tfsdk:"waitlist_enabled"`
}

func (r *AuthkitConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authkit_configuration"
}

func configurationTargetAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{Description: description, Required: true,
		Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
}

func (r *AuthkitConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Experimental: this resource uses the WorkOS dashboard (account) GraphQL API, which is not a public, versioned WorkOS API and may change without notice. Authoritatively manages the default AuthKit application's complete redirect/logout URI sets, environment CORS origins, initiate-login URI, AuthKit enablement, signup, and waitlist settings. Use one owner per environment. Destroy only detaches Terraform state and leaves all remote settings unchanged, with a warning.",
		Attributes: map[string]schema.Attribute{
			"id":                   schema.StringAttribute{Computed: true, Description: "Singleton import ID: environment_id/application_id.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"environment_id":       configurationTargetAttribute("Exact environment ID belonging to account_team_id; checked before every configuration mutation."),
			"application_id":       configurationTargetAttribute("Exact default AuthKit application ID belonging to environment_id. This is the application ID, not its client ID."),
			"redirect_uris":        schema.SetAttribute{Required: true, ElementType: types.StringType, Description: "Complete set of absolute HTTP(S) callback URLs. Replaces the full remote set. Do not combine with workos_redirect_uri for this application.", Validators: []validator.Set{setvalidator.SizeAtLeast(1)}},
			"default_redirect_uri": schema.StringAttribute{Required: true, Description: "Default callback URL; must be a member of redirect_uris."},
			"logout_uris":          schema.SetAttribute{Required: true, ElementType: types.StringType, Description: "Complete set of absolute HTTP(S) logout URLs. Replaces the full remote set.", Validators: []validator.Set{setvalidator.SizeAtLeast(1)}},
			"default_logout_uri":   schema.StringAttribute{Required: true, Description: "Default logout URL; must be a member of logout_uris."},
			"cors_origins":         schema.SetAttribute{Required: true, ElementType: types.StringType, Description: "Complete allowed HTTP(S) web origin set without paths, queries, credentials, wildcards, or fragments. An empty set disables all allowed origins."},
			"initiate_login_uri":   schema.StringAttribute{Required: true, Description: "Absolute HTTP(S) initiate-login URL."},
			"authkit_enabled":      schema.BoolAttribute{Required: true, Description: "Whether AuthKit is enabled for the environment."},
			"allow_signup":         schema.BoolAttribute{Required: true, Description: "Whether AuthKit allows user signup."},
			"waitlist_enabled":     schema.BoolAttribute{Required: true, Description: "Whether AuthKit enables the signup waitlist."},
		},
	}
}

func (r *AuthkitConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
	}
}

func (data *AuthkitConfigurationResourceModel) configuration(ctx context.Context) (*client.AuthkitConfiguration, diag.Diagnostics) {
	config := &client.AuthkitConfiguration{
		EnvironmentID: data.EnvironmentID.ValueString(), ApplicationID: data.ApplicationID.ValueString(),
		DefaultRedirectURI: data.DefaultRedirectURI.ValueString(), DefaultLogoutURI: data.DefaultLogoutURI.ValueString(),
		InitiateLoginURI: data.InitiateLoginURI.ValueString(), AuthkitEnabled: data.AuthkitEnabled.ValueBool(), AllowSignup: data.AllowSignup.ValueBool(), WaitlistEnabled: data.WaitlistEnabled.ValueBool(),
	}
	var diagnostics diag.Diagnostics
	diagnostics.Append(data.RedirectURIs.ElementsAs(ctx, &config.RedirectURIs, false)...)
	diagnostics.Append(data.LogoutURIs.ElementsAs(ctx, &config.LogoutURIs, false)...)
	diagnostics.Append(data.CORSOrigins.ElementsAs(ctx, &config.CORSOrigins, false)...)
	return config, diagnostics
}

func (r *AuthkitConfigurationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data AuthkitConfigurationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for _, value := range []types.String{data.EnvironmentID, data.ApplicationID, data.DefaultRedirectURI, data.DefaultLogoutURI, data.InitiateLoginURI} {
		if value.IsUnknown() || value.IsNull() {
			return
		}
	}
	for _, value := range []types.Set{data.RedirectURIs, data.LogoutURIs, data.CORSOrigins} {
		if value.IsUnknown() || value.IsNull() {
			return
		}
		for _, element := range value.Elements() {
			if element.IsUnknown() {
				return
			}
		}
	}
	config, diagnostics := data.configuration(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.ValidateAuthkitConfiguration(config); err != nil {
		resp.Diagnostics.AddError("Invalid AuthKit Configuration", err.Error())
	}
}

func (data *AuthkitConfigurationResourceModel) setObserved(ctx context.Context, observed *client.AuthkitConfiguration) diag.Diagnostics {
	data.ID = types.StringValue(observed.EnvironmentID + "/" + observed.ApplicationID)
	data.EnvironmentID = types.StringValue(observed.EnvironmentID)
	data.ApplicationID = types.StringValue(observed.ApplicationID)
	data.DefaultRedirectURI = types.StringValue(observed.DefaultRedirectURI)
	data.DefaultLogoutURI = types.StringValue(observed.DefaultLogoutURI)
	data.InitiateLoginURI = types.StringValue(observed.InitiateLoginURI)
	data.AuthkitEnabled = types.BoolValue(observed.AuthkitEnabled)
	data.AllowSignup = types.BoolValue(observed.AllowSignup)
	data.WaitlistEnabled = types.BoolValue(observed.WaitlistEnabled)
	var diagnostics, current diag.Diagnostics
	data.RedirectURIs, current = types.SetValueFrom(ctx, types.StringType, observed.RedirectURIs)
	diagnostics.Append(current...)
	data.LogoutURIs, current = types.SetValueFrom(ctx, types.StringType, observed.LogoutURIs)
	diagnostics.Append(current...)
	data.CORSOrigins, current = types.SetValueFrom(ctx, types.StringType, observed.CORSOrigins)
	diagnostics.Append(current...)
	return diagnostics
}

func (r *AuthkitConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AuthkitConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	desired, diagnostics := data.configuration(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	observed, err := r.client.ApplyAuthkitConfiguration(ctx, desired)
	if observed != nil {
		resp.Diagnostics.Append(data.setObserved(ctx, observed)...)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to Apply AuthKit Configuration", err.Error()+" Existing settings are preserved in observed state where readback was available; refresh and retry.")
	}
}

func (r *AuthkitConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AuthkitConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	desired, diagnostics := data.configuration(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	observed, err := r.client.ApplyAuthkitConfiguration(ctx, desired)
	if observed != nil {
		resp.Diagnostics.Append(data.setObserved(ctx, observed)...)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to Apply AuthKit Configuration", err.Error()+" Refresh and retry to reconcile remaining drift.")
	}
}

func (r *AuthkitConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AuthkitConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	observed, err := r.client.ReadAuthkitConfiguration(ctx, data.EnvironmentID.ValueString(), data.ApplicationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read AuthKit Configuration", err.Error())
		return
	}
	resp.Diagnostics.Append(data.setObserved(ctx, observed)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AuthkitConfigurationResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("AuthKit Configuration Detached", "Only Terraform ownership was removed. WorkOS redirect/logout URIs, CORS origins, initiate-login URI, AuthKit enablement, signup, and waitlist settings remain unchanged. Import the singleton again to resume management.")
}

func (r *AuthkitConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Use environment_id/application_id for the existing default AuthKit application.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), parts[1])...)
}
