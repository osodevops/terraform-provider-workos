// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WebhookResource{}
var _ resource.ResourceWithImportState = &WebhookResource{}

func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

// WebhookResource defines the resource implementation.
type WebhookResource struct {
	client *client.Client
}

// WebhookResourceModel describes the resource data model.
type WebhookResourceModel struct {
	ID        types.String `tfsdk:"id"`
	URL       types.String `tfsdk:"url"`
	Secret    types.String `tfsdk:"secret"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	Events    types.Set    `tfsdk:"events"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *WebhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS Webhook endpoint.",
		MarkdownDescription: `
Manages a WorkOS Webhook endpoint.

Webhooks allow you to receive real-time notifications when events occur in your
WorkOS environment, such as user creation, SSO authentication, or directory sync updates.

## Example Usage

` + "```hcl" + `
resource "workos_webhook" "main" {
  url     = "https://api.example.com/webhooks/workos"
  secret  = var.webhook_secret
  enabled = true

  events = [
    "user.created",
    "user.updated",
    "user.deleted",
    "dsync.user.created",
    "dsync.user.updated",
    "dsync.user.deleted",
    "dsync.group.created",
    "dsync.group.updated",
    "dsync.group.deleted",
    "connection.activated",
    "connection.deactivated",
  ]
}
` + "```" + `

## Event Types

### User Events
- ` + "`user.created`" + `, ` + "`user.updated`" + `, ` + "`user.deleted`" + `

### Authentication Events
- ` + "`authentication.email_verification_succeeded`" + `
- ` + "`authentication.magic_auth_succeeded`" + `, ` + "`authentication.magic_auth_failed`" + `
- ` + "`authentication.mfa_succeeded`" + `
- ` + "`authentication.oauth_succeeded`" + `, ` + "`authentication.oauth_failed`" + `
- ` + "`authentication.password_succeeded`" + `, ` + "`authentication.password_failed`" + `
- ` + "`authentication.sso_succeeded`" + `, ` + "`authentication.sso_failed`" + `

### Directory Sync Events
- ` + "`dsync.activated`" + `, ` + "`dsync.deleted`" + `
- ` + "`dsync.user.created`" + `, ` + "`dsync.user.updated`" + `, ` + "`dsync.user.deleted`" + `
- ` + "`dsync.group.created`" + `, ` + "`dsync.group.updated`" + `, ` + "`dsync.group.deleted`" + `

### Connection Events
- ` + "`connection.activated`" + `, ` + "`connection.deactivated`" + `, ` + "`connection.deleted`" + `

### Organization Events
- ` + "`organization.created`" + `, ` + "`organization.updated`" + `, ` + "`organization.deleted`" + `

### Organization Membership Events
- ` + "`organization_membership.added`" + `, ` + "`organization_membership.updated`" + `, ` + "`organization_membership.removed`" + `

### Session Events
- ` + "`session.created`" + `

## Import

Webhooks can be imported using the webhook ID:

` + "```shell" + `
terraform import workos_webhook.example webhook_01HXYZ...
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the webhook.",
				MarkdownDescription: "The unique identifier of the webhook (e.g., `webhook_01HXYZ...`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Description:         "The HTTPS URL where webhook events will be sent.",
				MarkdownDescription: "The HTTPS URL where webhook events will be sent. Must be publicly accessible.",
				Required:            true,
			},
			"secret": schema.StringAttribute{
				Description:         "The secret used to sign webhook payloads.",
				MarkdownDescription: "The secret used to sign webhook payloads. Use this to verify webhook authenticity.",
				Required:            true,
				Sensitive:           true,
			},
			"enabled": schema.BoolAttribute{
				Description:         "Whether the webhook is enabled.",
				MarkdownDescription: "Whether the webhook is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"events": schema.SetAttribute{
				Description:         "The event types this webhook subscribes to.",
				MarkdownDescription: "The event types this webhook subscribes to.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the webhook was created.",
				MarkdownDescription: "The timestamp when the webhook was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the webhook was last updated.",
				MarkdownDescription: "The timestamp when the webhook was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (r *WebhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract events
	var events []string
	resp.Diagnostics.Append(plan.Events.ElementsAs(ctx, &events, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Warn about unknown event types
	for _, event := range events {
		if !client.IsKnownWebhookEvent(event) {
			tflog.Warn(ctx, "Unknown webhook event type", map[string]any{
				"event": event,
			})
		}
	}

	tflog.Debug(ctx, "Creating webhook", map[string]any{
		"url":    plan.URL.ValueString(),
		"events": events,
	})

	createReq := &client.WebhookCreateRequest{
		URL:     plan.URL.ValueString(),
		Secret:  plan.Secret.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
		Events:  events,
	}

	webhook, err := r.client.CreateWebhook(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Webhook",
			"Could not create webhook, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(webhook.ID)
	plan.CreatedAt = types.StringValue(webhook.CreatedAt)
	plan.UpdatedAt = types.StringValue(webhook.UpdatedAt)

	tflog.Info(ctx, "Created webhook", map[string]any{
		"id":  webhook.ID,
		"url": webhook.URL,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading webhook", map[string]any{
		"id": state.ID.ValueString(),
	})

	webhook, err := r.client.GetWebhook(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Webhook not found, removing from state", map[string]any{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Webhook",
			"Could not read webhook ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state
	state.URL = types.StringValue(webhook.URL)
	state.Enabled = types.BoolValue(webhook.Enabled)
	state.CreatedAt = types.StringValue(webhook.CreatedAt)
	state.UpdatedAt = types.StringValue(webhook.UpdatedAt)

	// Map events
	if len(webhook.Events) > 0 {
		events, diags := types.SetValueFrom(ctx, types.StringType, webhook.Events)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Events = events
	}

	// Note: Secret is not returned by the API, so we preserve the state value

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookResourceModel
	var state WebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract events
	var events []string
	resp.Diagnostics.Append(plan.Events.ElementsAs(ctx, &events, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Warn about unknown event types
	for _, event := range events {
		if !client.IsKnownWebhookEvent(event) {
			tflog.Warn(ctx, "Unknown webhook event type", map[string]any{
				"event": event,
			})
		}
	}

	tflog.Debug(ctx, "Updating webhook", map[string]any{
		"id":  state.ID.ValueString(),
		"url": plan.URL.ValueString(),
	})

	enabled := plan.Enabled.ValueBool()
	updateReq := &client.WebhookUpdateRequest{
		URL:     plan.URL.ValueString(),
		Enabled: &enabled,
		Events:  events,
	}

	webhook, err := r.client.UpdateWebhook(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Webhook",
			"Could not update webhook, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = state.ID
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(webhook.UpdatedAt)

	tflog.Info(ctx, "Updated webhook", map[string]any{
		"id": webhook.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting webhook", map[string]any{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteWebhook(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Webhook already deleted", map[string]any{
				"id": state.ID.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Webhook",
			"Could not delete webhook, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted webhook", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing webhook", map[string]any{
		"id": req.ID,
	})

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Note: Secret must be provided in config after import since it's not returned by API
	resp.Diagnostics.AddWarning(
		"Secret Required After Import",
		"The webhook secret is not returned by the API. You must set the 'secret' attribute "+
			"in your configuration to match the original secret, or the resource will be recreated.",
	)
}
