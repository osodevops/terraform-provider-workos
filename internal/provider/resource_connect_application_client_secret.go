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

var _ resource.Resource = &ConnectApplicationClientSecretResource{}
var _ resource.ResourceWithImportState = &ConnectApplicationClientSecretResource{}

func NewConnectApplicationClientSecretResource() resource.Resource {
	return &ConnectApplicationClientSecretResource{}
}

type ConnectApplicationClientSecretResource struct {
	client *client.Client
}

type ConnectApplicationClientSecretResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	SecretHint    types.String `tfsdk:"secret_hint"`
	Secret        types.String `tfsdk:"secret"`
	LastUsedAt    types.String `tfsdk:"last_used_at"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *ConnectApplicationClientSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_application_client_secret"
}

func (r *ConnectApplicationClientSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a client secret for a WorkOS Connect application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the client secret.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Description: "The Connect application ID or client ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"secret_hint": schema.StringAttribute{
				Description: "A hint for the secret.",
				Computed:    true,
			},
			"secret": schema.StringAttribute{
				Description: "The plaintext client secret. Only returned on create.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_used_at": schema.StringAttribute{
				Description: "The timestamp when the secret was last used.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the secret was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the secret was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *ConnectApplicationClientSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ConnectApplicationClientSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConnectApplicationClientSecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.client.CreateConnectApplicationClientSecret(ctx, plan.ApplicationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Connect Application Client Secret", "Could not create client secret: "+err.Error())
		return
	}

	connectApplicationClientSecretToState(&plan, secret, types.StringNull())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConnectApplicationClientSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConnectApplicationClientSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.findConnectApplicationClientSecret(ctx, state.ApplicationID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Connect Application Client Secret", "Could not read client secret: "+err.Error())
		return
	}

	connectApplicationClientSecretToState(&state, secret, state.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConnectApplicationClientSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ConnectApplicationClientSecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConnectApplicationClientSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConnectApplicationClientSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteConnectApplicationClientSecret(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Connect application client secret already deleted", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error Deleting Connect Application Client Secret", "Could not delete client secret: "+err.Error())
	}
}

func (r *ConnectApplicationClientSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitCompositeID(req.ID, 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Import ID", "Expected import ID in the format 'application_id/secret_id'.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *ConnectApplicationClientSecretResource) findConnectApplicationClientSecret(ctx context.Context, applicationID, secretID string) (*client.ConnectApplicationSecret, error) {
	secrets, err := r.client.ListConnectApplicationClientSecrets(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	for i := range secrets {
		if secrets[i].ID == secretID {
			return &secrets[i], nil
		}
	}
	return nil, &client.APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("client secret %s was not found on connect application %s", secretID, applicationID),
	}
}

func connectApplicationClientSecretToState(state *ConnectApplicationClientSecretResourceModel, secret *client.ConnectApplicationSecret, priorSecret types.String) {
	state.ID = types.StringValue(secret.ID)
	state.SecretHint = types.StringValue(secret.SecretHint)
	state.CreatedAt = types.StringValue(secret.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(secret.UpdatedAt.Format(time.RFC3339))
	if secret.LastUsedAt != nil {
		state.LastUsedAt = types.StringValue(secret.LastUsedAt.Format(time.RFC3339))
	} else {
		state.LastUsedAt = types.StringNull()
	}
	if secret.Secret != "" {
		state.Secret = types.StringValue(secret.Secret)
		return
	}
	if priorSecret.IsUnknown() {
		state.Secret = types.StringNull()
		return
	}
	state.Secret = priorSecret
}
