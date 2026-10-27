// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

var _ resource.Resource = &EnvironmentRoleConfigurationResource{}
var _ resource.ResourceWithConfigure = &EnvironmentRoleConfigurationResource{}
var _ resource.ResourceWithImportState = &EnvironmentRoleConfigurationResource{}

func NewEnvironmentRoleConfigurationResource() resource.Resource {
	return &EnvironmentRoleConfigurationResource{}
}

type EnvironmentRoleConfigurationResource struct{ client *client.Client }
type EnvironmentRoleConfigurationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	EnvironmentID   types.String `tfsdk:"environment_id"`
	DefaultRoleSlug types.String `tfsdk:"default_role_slug"`
}

func (r *EnvironmentRoleConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_role_configuration"
}
func (r *EnvironmentRoleConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Experimental: this resource uses the WorkOS dashboard (account) GraphQL API, which is not a public, versioned WorkOS API and may change without notice. Manages the default role for one environment. Existing workos_environment_role resources continue to own role definitions and permissions. Destroy detaches Terraform ownership with a warning and leaves the remote default unchanged.", Attributes: map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true, Description: "Singleton import ID: environment_id.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"environment_id":    configurationTargetAttribute("Exact environment ID belonging to account_team_id; checked before role configuration mutations."),
		"default_role_slug": schema.StringAttribute{Required: true, Description: "Slug of an existing unscoped environment role. Resolved to a verified role ID in the configured environment before updating the default.", Validators: []validator.String{stringvalidator.LengthAtLeast(1)}},
	}}
}
func (r *EnvironmentRoleConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
	}
}
func (data *EnvironmentRoleConfigurationResourceModel) setObserved(observed *client.EnvironmentRoleConfiguration) {
	data.ID = types.StringValue(observed.EnvironmentID)
	data.EnvironmentID = types.StringValue(observed.EnvironmentID)
	data.DefaultRoleSlug = types.StringValue(observed.DefaultRoleSlug)
}
func (r *EnvironmentRoleConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EnvironmentRoleConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	observed, err := r.client.ApplyEnvironmentRoleConfiguration(ctx, data.EnvironmentID.ValueString(), data.DefaultRoleSlug.ValueString())
	if observed != nil {
		data.setObserved(observed)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to Apply Environment Role Configuration", err.Error()+" Refresh and retry to reconcile remaining drift.")
	}
}
func (r *EnvironmentRoleConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data EnvironmentRoleConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	observed, err := r.client.ApplyEnvironmentRoleConfiguration(ctx, data.EnvironmentID.ValueString(), data.DefaultRoleSlug.ValueString())
	if observed != nil {
		data.setObserved(observed)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to Apply Environment Role Configuration", err.Error()+" Refresh and retry to reconcile remaining drift.")
	}
}
func (r *EnvironmentRoleConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentRoleConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	observed, err := r.client.ReadEnvironmentRoleConfiguration(ctx, data.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Environment Role Configuration", err.Error())
		return
	}
	data.setObserved(observed)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *EnvironmentRoleConfigurationResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Environment Role Configuration Detached", "Only Terraform ownership was removed. The default role and all role definitions and permissions remain unchanged in WorkOS. Import the singleton again to resume management.")
}
func (r *EnvironmentRoleConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if strings.TrimSpace(req.ID) == "" || strings.Contains(req.ID, "/") {
		resp.Diagnostics.AddError("Invalid Import ID", "Use the exact environment_id for the existing role configuration.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), req.ID)...)
}
