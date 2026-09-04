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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RedirectURIResource{}
var _ resource.ResourceWithImportState = &RedirectURIResource{}

func NewRedirectURIResource() resource.Resource {
	return &RedirectURIResource{}
}

// RedirectURIResource defines the resource implementation.
type RedirectURIResource struct {
	client *client.Client
}

// RedirectURIResourceModel describes the resource data model.
type RedirectURIResourceModel struct {
	ID        types.String `tfsdk:"id"`
	URI       types.String `tfsdk:"uri"`
	Default   types.Bool   `tfsdk:"default"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *RedirectURIResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redirect_uri"
}

func (r *RedirectURIResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an AuthKit redirect URI on the User Management API.",
		MarkdownDescription: `
Manages an AuthKit login callback URI.

This is not the same as ` + "`workos_connect_application.redirect_uris`" + `. Connect
applications (` + "`connect_app_…`" + `, ` + "`POST /connect/applications`" + `) are OAuth or
M2M clients. AuthKit callbacks live on the environment's AuthKit application
(` + "`app_…`" + `) and are registered with ` + "`POST /user_management/redirect_uris`" + `.
The target application is the API-key context; the create body has no
` + "`application_id`" + `.

## Import

Redirect URIs can be imported by WorkOS id (` + "`redir_…`" + `) or by exact URI:

` + "```shell" + `
terraform import workos_redirect_uri.example redir_01EHZNVPK3SFK441A1RGBFSHRT
terraform import workos_redirect_uri.example https://acme.example.com/api/auth/callback
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the redirect URI.",
				MarkdownDescription: "The unique identifier of the redirect URI (`redir_…`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uri": schema.StringAttribute{
				Description: "The HTTPS callback registered on the AuthKit application bound to the provider API key.",
				MarkdownDescription: "The HTTPS callback registered on the AuthKit application bound to " +
					"the provider API key. Changing this value forces replacement.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default": schema.BoolAttribute{
				Description:         "Whether WorkOS treats this as the default redirect URI.",
				MarkdownDescription: "Whether WorkOS treats this as the default redirect URI.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the redirect URI was created.",
				MarkdownDescription: "The timestamp when the redirect URI was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the redirect URI was last updated.",
				MarkdownDescription: "The timestamp when the redirect URI was last updated (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RedirectURIResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	workosClient, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = workosClient
}

func (r *RedirectURIResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RedirectURIResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	uri := plan.URI.ValueString()
	tflog.Debug(ctx, "Creating redirect URI", map[string]any{
		"uri": uri,
	})

	redirectURI, err := r.client.CreateRedirectURI(ctx, &client.RedirectURICreateRequest{
		URI: uri,
	})
	if err != nil {
		if client.IsExistingRedirectURI(err) {
			tflog.Info(ctx, "Redirect URI already exists, adopting", map[string]any{
				"uri": uri,
			})
			redirectURI, err = r.client.GetRedirectURIByURI(ctx, uri)
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Redirect URI",
				"Could not create redirect URI, unexpected error: "+err.Error(),
			)
			return
		}
	}

	redirectURIToState(&plan, redirectURI)

	tflog.Info(ctx, "Created redirect URI", map[string]any{
		"id":  redirectURI.ID,
		"uri": redirectURI.URI,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RedirectURIResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RedirectURIResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading redirect URI", map[string]any{
		"id": state.ID.ValueString(),
	})

	redirectURI, err := r.client.GetRedirectURI(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Redirect URI not found, removing from state", map[string]any{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Redirect URI",
			"Could not read redirect URI "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	redirectURIToState(&state, redirectURI)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RedirectURIResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RedirectURIResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	redirectURI, err := r.client.GetRedirectURI(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Redirect URI",
			"Could not refresh redirect URI "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	redirectURIToState(&plan, redirectURI)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RedirectURIResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RedirectURIResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting redirect URI", map[string]any{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteRedirectURI(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Redirect URI already deleted", map[string]any{
				"id": state.ID.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Redirect URI",
			"Could not delete redirect URI, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted redirect URI", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *RedirectURIResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing redirect URI", map[string]any{
		"id": req.ID,
	})

	id := req.ID
	if !client.IsRedirectURIID(id) {
		redirectURI, err := r.client.GetRedirectURIByURI(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Importing Redirect URI",
				"Could not find redirect URI "+id+": "+err.Error(),
			)
			return
		}
		id = redirectURI.ID
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func redirectURIToState(state *RedirectURIResourceModel, redirectURI *client.RedirectURI) {
	state.ID = types.StringValue(redirectURI.ID)
	state.URI = types.StringValue(redirectURI.URI)
	state.Default = types.BoolValue(redirectURI.Default)
	state.CreatedAt = types.StringValue(redirectURI.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(redirectURI.UpdatedAt.Format(time.RFC3339))
}
