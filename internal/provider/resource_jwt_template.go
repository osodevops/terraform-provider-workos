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
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

const jwtTemplateID = "jwt_template"

var _ resource.Resource = &JWTTemplateResource{}
var _ resource.ResourceWithImportState = &JWTTemplateResource{}

func NewJWTTemplateResource() resource.Resource {
	return &JWTTemplateResource{}
}

type JWTTemplateResource struct {
	client *client.Client
}

func (r *JWTTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jwt_template"
}

func (r *JWTTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the JWT template that adds custom claims to the access tokens of the environment bound to the API key.",
		MarkdownDescription: `
Manages the JWT template that adds custom claims to the access tokens AuthKit issues for the
environment bound to the provider API key.

The template is a singleton per environment: creating this resource replaces whatever template the
environment already has, so declare it at most once per environment. WorkOS validates the content
and rejects an invalid template with HTTP 422, which fails the apply and leaves the current template in place.

WorkOS has no endpoint to delete the template, so destroying this resource only removes it from
Terraform state and leaves the template in place. Set ` + "`content`" + ` to ` + "`\"{}\"`" + ` first to clear it.

## Import

The template has no ID of its own. Import it with any ID, for example:

` + "```shell" + `
terraform import workos_jwt_template.this jwt_template
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Always \"jwt_template\": the template is a singleton per environment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"content": schema.StringAttribute{
				Description: "The template, a JSON object whose values may use WorkOS template expressions such as {{user.email}} or {{organization_membership.roles}}.",
				Required:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the template was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the template was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *JWTTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *JWTTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan JWTTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.UpdateJWTTemplate(ctx, &client.JWTTemplateUpdateRequest{Content: plan.Content.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error Setting JWT Template", "Could not set JWT template: "+err.Error())
		return
	}

	jwtTemplateToState(&plan, template)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *JWTTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state JWTTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.GetJWTTemplate(ctx)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading JWT Template", "Could not read JWT template: "+err.Error())
		return
	}

	jwtTemplateToState(&state, template)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *JWTTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan JWTTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.UpdateJWTTemplate(ctx, &client.JWTTemplateUpdateRequest{Content: plan.Content.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error Updating JWT Template", "Could not update JWT template: "+err.Error())
		return
	}

	jwtTemplateToState(&plan, template)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *JWTTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"JWT Template Detached",
		"WorkOS has no endpoint to delete the JWT template, so only Terraform ownership was removed and the template remains in place. Apply content = \"{}\" before destroying to clear it.",
	)
}

func (r *JWTTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), jwtTemplateID)...)
}

func jwtTemplateToState(state *JWTTemplateResourceModel, template *client.JWTTemplate) {
	state.ID = types.StringValue(jwtTemplateID)
	state.Content = types.StringValue(template.Content)
	state.CreatedAt = types.StringValue(template.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(template.UpdatedAt.Format(time.RFC3339))
}
