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

var _ resource.Resource = &AuthKitOAuthResourceResource{}
var _ resource.ResourceWithImportState = &AuthKitOAuthResourceResource{}

func NewAuthKitOAuthResourceResource() resource.Resource {
	return &AuthKitOAuthResourceResource{}
}

type AuthKitOAuthResourceResource struct {
	client *client.Client
}

func (r *AuthKitOAuthResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authkit_oauth_resource"
}

func (r *AuthKitOAuthResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an MCP resource indicator (RFC 8707) that AuthKit accepts as the resource, and issues as the audience, of an access token.",
		MarkdownDescription: `
Manages an MCP resource indicator (RFC 8707) on the environment bound to the provider API key.

AuthKit accepts the indicator's URI as the ` + "`resource`" + ` parameter of an OAuth request and issues it as
the ` + "`aud`" + ` claim of the resulting access token, so an MCP server or API can check that a token was
minted for it.

WorkOS has no update endpoint for resource indicators: ` + "`uri`" + ` and ` + "`default`" + ` can only be set when the
indicator is created, and changing either replaces it.

~> **Deleting an indicator also deletes every application consent granted against it.** Replacing
or destroying this resource makes every user consent again in each MCP client. Consider
` + "`lifecycle { prevent_destroy = true }`" + ` on indicators that are in use.

Registering a URI that already exists fails rather than adopting it, so that a later
` + "`terraform destroy`" + ` cannot remove an indicator, and its consents, this configuration did not create.
Import the existing indicator instead.

## Import

Resource indicators can be imported by ID:

` + "```shell" + `
terraform import workos_authkit_oauth_resource.example authkit_oauth_resource_01EHZNVPK3SFK441A1RGBFSHRT
` + "```" + `

They can also be imported by their exact URI:

` + "```shell" + `
terraform import workos_authkit_oauth_resource.example https://mcp.example.com
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the resource indicator.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uri": schema.StringAttribute{
				Description: "The resource URI AuthKit accepts as the resource parameter and issues as the token audience.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default": schema.BoolAttribute{
				Description: "Whether AuthKit uses this indicator when a request names no resource. Set only at creation; changing it replaces the indicator.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the resource indicator was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the resource indicator was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *AuthKitOAuthResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AuthKitOAuthResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthKitOAuthResourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uri := plan.URI.ValueString()
	createRequest := &client.AuthKitOAuthResourceCreateRequest{URI: uri}
	if !plan.Default.IsNull() && !plan.Default.IsUnknown() {
		isDefault := plan.Default.ValueBool()
		createRequest.Default = &isDefault
	}

	oauthResource, err := r.client.CreateAuthKitOAuthResource(ctx, createRequest)
	if err != nil {
		if client.IsExistingAuthKitOAuthResource(err) {
			resp.Diagnostics.AddError(
				"Resource Indicator Already Exists",
				fmt.Sprintf(
					"The resource indicator %q is already registered on this environment, so it was not created.\n\n"+
						"Import the existing resource indicator instead of creating it:\n"+
						"    terraform import workos_authkit_oauth_resource.<name> %s",
					uri, uri,
				),
			)
			return
		}
		resp.Diagnostics.AddError("Error Creating Resource Indicator", "Could not create resource indicator: "+err.Error())
		return
	}

	authKitOAuthResourceToState(&plan, oauthResource)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthKitOAuthResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthKitOAuthResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oauthResource, err := r.client.GetAuthKitOAuthResource(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Resource Indicator", "Could not read resource indicator: "+err.Error())
		return
	}

	authKitOAuthResourceToState(&state, oauthResource)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AuthKitOAuthResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state AuthKitOAuthResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AuthKitOAuthResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthKitOAuthResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAuthKitOAuthResource(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Resource indicator already deleted", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error Deleting Resource Indicator", "Could not delete resource indicator: "+err.Error())
	}
}

func (r *AuthKitOAuthResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if !client.IsAuthKitOAuthResourceID(id) {
		oauthResource, err := r.client.GetAuthKitOAuthResourceByURI(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Error Importing Resource Indicator", "Could not find resource indicator "+id+": "+err.Error())
			return
		}
		id = oauthResource.ID
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func authKitOAuthResourceToState(state *AuthKitOAuthResourceResourceModel, oauthResource *client.AuthKitOAuthResource) {
	state.ID = types.StringValue(oauthResource.ID)
	state.URI = types.StringValue(oauthResource.URI)
	state.Default = types.BoolValue(oauthResource.Default)
	state.CreatedAt = types.StringValue(oauthResource.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(oauthResource.UpdatedAt.Format(time.RFC3339))
}
