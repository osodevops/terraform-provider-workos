// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

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
var _ resource.Resource = &ConnectionResource{}
var _ resource.ResourceWithImportState = &ConnectionResource{}

func NewConnectionResource() resource.Resource {
	return &ConnectionResource{}
}

// ConnectionResource defines the resource implementation.
type ConnectionResource struct {
	client *client.Client
}

// ConnectionResourceModel describes the resource data model.
type ConnectionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ConnectionType types.String `tfsdk:"connection_type"`
	Name           types.String `tfsdk:"name"`
	State          types.String `tfsdk:"state"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *ConnectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection"
}

func (r *ConnectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS SSO Connection.",
		MarkdownDescription: `
Manages a WorkOS SSO Connection.

Connections represent the link between your application and an identity provider (IdP)
for Single Sign-On authentication. WorkOS supports SAML, OAuth, and OIDC connections.

~> **Note:** Connection configuration (SAML certificates, OIDC client credentials) is typically
done through the WorkOS Dashboard or Admin Portal after the connection is created via Terraform.

## Example Usage

### Basic Connection

` + "```hcl" + `
resource "workos_connection" "okta_sso" {
  organization_id = workos_organization.main.id
  connection_type = "OktaSAML"
  name            = "Okta SSO"
}
` + "```" + `

### Google OAuth Connection

` + "```hcl" + `
resource "workos_connection" "google" {
  organization_id = workos_organization.main.id
  connection_type = "GoogleOAuth"
  name            = "Google Login"
}
` + "```" + `

## Supported Connection Types

### SAML
- ` + "`OktaSAML`" + ` - Okta SAML
- ` + "`AzureSAML`" + ` - Azure AD SAML
- ` + "`GoogleSAML`" + ` - Google Workspace SAML
- ` + "`OneLoginSAML`" + ` - OneLogin SAML
- ` + "`PingFederateSAML`" + ` - PingFederate SAML
- ` + "`PingOneSAML`" + ` - PingOne SAML
- ` + "`JumpCloudSAML`" + ` - JumpCloud SAML
- ` + "`GenericSAML`" + ` - Generic SAML 2.0

### OAuth
- ` + "`GoogleOAuth`" + ` - Google OAuth
- ` + "`MicrosoftOAuth`" + ` - Microsoft OAuth

### OIDC
- ` + "`GenericOIDC`" + ` - Generic OpenID Connect

## Import

Connections can be imported using the connection ID:

` + "```shell" + `
terraform import workos_connection.example conn_01HXYZ...
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the connection.",
				MarkdownDescription: "The unique identifier of the connection (e.g., `conn_01HXYZ...`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description:         "The ID of the organization this connection belongs to.",
				MarkdownDescription: "The ID of the organization this connection belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"connection_type": schema.StringAttribute{
				Description:         "The type of SSO connection.",
				MarkdownDescription: "The type of SSO connection (e.g., `OktaSAML`, `GoogleOAuth`, `GenericOIDC`). Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "A friendly name for the connection.",
				MarkdownDescription: "A friendly name for the connection. This is displayed in the WorkOS Dashboard.",
				Optional:            true,
				Computed:            true,
			},
			"state": schema.StringAttribute{
				Description:         "The current state of the connection.",
				MarkdownDescription: "The current state of the connection (`active`, `inactive`, `validating`).",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "The configuration status of the connection.",
				MarkdownDescription: "The configuration status of the connection (`linked`, `unlinked`).",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the connection was created.",
				MarkdownDescription: "The timestamp when the connection was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the connection was last updated.",
				MarkdownDescription: "The timestamp when the connection was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (r *ConnectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConnectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating connection", map[string]any{
		"organization_id": plan.OrganizationID.ValueString(),
		"connection_type": plan.ConnectionType.ValueString(),
	})

	createReq := &client.ConnectionCreateRequest{
		OrganizationID: plan.OrganizationID.ValueString(),
		ConnectionType: plan.ConnectionType.ValueString(),
		Name:           plan.Name.ValueString(),
	}

	conn, err := r.client.CreateConnection(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Connection",
			"Could not create connection, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(conn.ID)
	plan.State = types.StringValue(conn.State)
	plan.Status = types.StringValue(conn.Status)
	if conn.Name != "" {
		plan.Name = types.StringValue(conn.Name)
	}
	plan.CreatedAt = types.StringValue(conn.CreatedAt.Format("2006-01-02T15:04:05Z"))
	plan.UpdatedAt = types.StringValue(conn.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	tflog.Info(ctx, "Created connection", map[string]any{
		"id":              conn.ID,
		"connection_type": conn.ConnectionType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConnectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading connection", map[string]any{
		"id": state.ID.ValueString(),
	})

	conn, err := r.client.GetConnection(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Connection not found, removing from state", map[string]any{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Connection",
			"Could not read connection ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state
	state.OrganizationID = types.StringValue(conn.OrganizationID)
	state.ConnectionType = types.StringValue(conn.ConnectionType)
	state.Name = types.StringValue(conn.Name)
	state.State = types.StringValue(conn.State)
	state.Status = types.StringValue(conn.Status)
	state.CreatedAt = types.StringValue(conn.CreatedAt.Format("2006-01-02T15:04:05Z"))
	state.UpdatedAt = types.StringValue(conn.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ConnectionResourceModel
	var state ConnectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating connection", map[string]any{
		"id":   state.ID.ValueString(),
		"name": plan.Name.ValueString(),
	})

	updateReq := &client.ConnectionUpdateRequest{
		Name: plan.Name.ValueString(),
	}

	conn, err := r.client.UpdateConnection(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Connection",
			"Could not update connection, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = state.ID
	plan.State = types.StringValue(conn.State)
	plan.Status = types.StringValue(conn.Status)
	plan.Name = types.StringValue(conn.Name)
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(conn.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	tflog.Info(ctx, "Updated connection", map[string]any{
		"id": conn.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConnectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting connection", map[string]any{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteConnection(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Connection already deleted", map[string]any{
				"id": state.ID.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Connection",
			"Could not delete connection, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted connection", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *ConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing connection", map[string]any{
		"id": req.ID,
	})

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
