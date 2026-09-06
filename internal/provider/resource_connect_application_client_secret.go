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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
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
	ID             types.String `tfsdk:"id"`
	ApplicationID  types.String `tfsdk:"application_id"`
	RotateTriggers types.Map    `tfsdk:"rotate_triggers"`
	SecretHint     types.String `tfsdk:"secret_hint"`
	Secret         types.String `tfsdk:"secret"`
	LastUsedAt     types.String `tfsdk:"last_used_at"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *ConnectApplicationClientSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_application_client_secret"
}

func (r *ConnectApplicationClientSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a client secret for a WorkOS Connect application.",
		MarkdownDescription: `
Mints and revokes a client secret for a WorkOS Connect application.

WorkOS returns the plaintext secret only once, in the create response, so the provider stores it in
Terraform state. **Anyone who can read your state file can read this secret.** Use an encrypted
remote backend with access controls, and treat the state as the credential itself. WorkOS allows at
most five secrets per application.

There is no update endpoint. To rotate a secret, change ` + "`rotate_triggers`" + ` (or run
` + "`terraform apply -replace`" + `), which mints a replacement and revokes the old one. Destroying the
resource revokes the secret immediately, so anything still authenticating with it will start
failing.

An imported secret has a null ` + "`secret`" + `, because WorkOS never returns the plaintext again.
Rotate to obtain a usable value.

## Import

Client secrets are imported using the application ID and the secret ID:

` + "```shell" + `
terraform import workos_connect_application_client_secret.example connect_app_01HXYZ.../connect_app_secret_01HXYZ...
` + "```" + `
`,
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
			"rotate_triggers": schema.MapAttribute{
				Description:         "Arbitrary key/value pairs that force a new secret to be minted when they change.",
				MarkdownDescription: "Arbitrary key/value pairs that force a new secret to be minted when any value changes. Use this to rotate on a schedule or alongside another resource, for example `{ rotated_at = time_rotating.quarterly.rotation_rfc3339 }`. Changing it replaces the resource: WorkOS has no update endpoint for client secrets.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"secret_hint": schema.StringAttribute{
				Description: "A hint for the secret.",
				Computed:    true,
			},
			"secret": schema.StringAttribute{
				Description:         "The plaintext client secret. Only returned on create and stored in Terraform state.",
				MarkdownDescription: "The plaintext client secret. WorkOS returns it only in the create response, so it is stored in Terraform state and is null for imported secrets.",
				Computed:            true,
				Sensitive:           true,
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

// Update is unreachable in practice: every configurable attribute requires
// replacement and WorkOS exposes no update endpoint for client secrets. It
// carries prior state forward so the framework contract is satisfied.
func (r *ConnectApplicationClientSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state ConnectApplicationClientSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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
