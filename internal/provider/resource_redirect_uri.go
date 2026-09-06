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

var _ resource.Resource = &RedirectURIResource{}
var _ resource.ResourceWithImportState = &RedirectURIResource{}

func NewRedirectURIResource() resource.Resource {
	return &RedirectURIResource{}
}

type RedirectURIResource struct {
	client *client.Client
}

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
		Description: "Manages an AuthKit redirect URI on the User Management API. This is not workos_connect_application.redirect_uris.",
		MarkdownDescription: `
Manages an AuthKit redirect URI on the User Management API.

Redirect URIs are the login callbacks AuthKit is allowed to return users to. They belong to the
AuthKit application bound to the provider API key, not to an organization. This is a different
concept from ` + "`workos_connect_application.redirect_uris`" + `, which scopes callbacks to a single
Connect application.

WorkOS has no update endpoint for redirect URIs, so changing ` + "`uri`" + ` replaces the resource.

Registering a URI that already exists fails rather than adopting it, so that a later
` + "`terraform destroy`" + ` cannot revoke a callback this configuration did not create. Import the
existing URI instead.

## Import

Redirect URIs can be imported by ID:

` + "```shell" + `
terraform import workos_redirect_uri.example redir_01HXYZ...
` + "```" + `

They can also be imported by their exact URI, which is useful when adopting a callback that was
registered outside Terraform:

` + "```shell" + `
terraform import workos_redirect_uri.example https://acme.example.com/api/auth/callback
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the redirect URI.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uri": schema.StringAttribute{
				Description: "The HTTPS callback registered on the AuthKit application bound to the provider API key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default": schema.BoolAttribute{
				Description: "Whether WorkOS treats this as the default redirect URI.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the redirect URI was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the redirect URI was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *RedirectURIResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RedirectURIResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RedirectURIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uri := plan.URI.ValueString()
	redirectURI, err := r.client.CreateRedirectURI(ctx, &client.RedirectURICreateRequest{URI: uri})
	if err != nil {
		if client.IsExistingRedirectURI(err) {
			// Do not silently take ownership of a redirect URI this configuration
			// did not create: a later destroy would revoke a callback that other
			// systems may depend on. Terraform's import workflow is the supported
			// way to bring existing infrastructure under management.
			resp.Diagnostics.AddError(
				"Redirect URI Already Exists",
				fmt.Sprintf(
					"The redirect URI %q is already registered on this AuthKit application, so it was not created.\n\n"+
						"Import the existing redirect URI instead of creating it:\n"+
						"    terraform import workos_redirect_uri.<name> %s",
					uri, uri,
				),
			)
			return
		}
		resp.Diagnostics.AddError("Error Creating Redirect URI", "Could not create redirect URI: "+err.Error())
		return
	}

	redirectURIToState(&plan, redirectURI)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RedirectURIResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RedirectURIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	redirectURI, err := r.client.GetRedirectURI(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Redirect URI", "Could not read redirect URI: "+err.Error())
		return
	}

	redirectURIToState(&state, redirectURI)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable in practice: uri is the only configurable attribute and
// it requires replacement, and WorkOS exposes no update endpoint for redirect
// URIs. It carries prior state forward so the framework contract is satisfied.
func (r *RedirectURIResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state RedirectURIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RedirectURIResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RedirectURIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRedirectURI(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Redirect URI already deleted", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error Deleting Redirect URI", "Could not delete redirect URI: "+err.Error())
	}
}

func (r *RedirectURIResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if !client.IsRedirectURIID(id) {
		redirectURI, err := r.client.GetRedirectURIByURI(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Error Importing Redirect URI", "Could not find redirect URI "+id+": "+err.Error())
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
