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
var _ resource.Resource = &OrganizationResource{}
var _ resource.ResourceWithImportState = &OrganizationResource{}

func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

// OrganizationResource defines the resource implementation.
type OrganizationResource struct {
	client *client.Client
}

// OrganizationResourceModel describes the resource data model.
type OrganizationResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Domains   types.Set    `tfsdk:"domains"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *OrganizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *OrganizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS Organization.",
		MarkdownDescription: `
Manages a WorkOS Organization.

Organizations are the fundamental multi-tenant unit in WorkOS. They represent your customers'
companies and are used to group users, SSO connections, and directory sync configurations.

## Example Usage

` + "```hcl" + `
resource "workos_organization" "example" {
  name    = "Acme Corporation"
  domains = ["acme.com", "acmecorp.com"]
}
` + "```" + `

## Import

Organizations can be imported using the organization ID:

` + "```shell" + `
terraform import workos_organization.example org_01HXYZ...
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the organization.",
				MarkdownDescription: "The unique identifier of the organization (e.g., `org_01HXYZ...`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The name of the organization.",
				MarkdownDescription: "The name of the organization. Must be between 1 and 255 characters.",
				Required:            true,
			},
			"domains": schema.SetAttribute{
				Description:         "The domains associated with the organization.",
				MarkdownDescription: "The domains associated with the organization. These are used for domain-based SSO routing.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the organization was created.",
				MarkdownDescription: "The timestamp when the organization was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the organization was last updated.",
				MarkdownDescription: "The timestamp when the organization was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (r *OrganizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating organization", map[string]any{
		"name": plan.Name.ValueString(),
	})

	// Build the create request
	createReq := &client.OrganizationCreateRequest{
		Name: plan.Name.ValueString(),
	}

	// Add domains if specified
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		var domains []string
		resp.Diagnostics.Append(plan.Domains.ElementsAs(ctx, &domains, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, domain := range domains {
			createReq.DomainData = append(createReq.DomainData, client.DomainData{
				Domain: domain,
				State:  "verified",
			})
		}
	}

	// Create the organization
	org, err := r.client.CreateOrganization(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Organization",
			"Could not create organization, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(org.ID)
	plan.CreatedAt = types.StringValue(org.CreatedAt.Format("2006-01-02T15:04:05Z"))
	plan.UpdatedAt = types.StringValue(org.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	tflog.Info(ctx, "Created organization", map[string]any{
		"id":   org.ID,
		"name": org.Name,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading organization", map[string]any{
		"id": state.ID.ValueString(),
	})

	// Get the organization from API
	org, err := r.client.GetOrganization(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Organization not found, removing from state", map[string]any{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Organization",
			"Could not read organization ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state
	state.Name = types.StringValue(org.Name)
	state.CreatedAt = types.StringValue(org.CreatedAt.Format("2006-01-02T15:04:05Z"))
	state.UpdatedAt = types.StringValue(org.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	// Map domains
	if len(org.Domains) > 0 {
		domainStrings := make([]string, len(org.Domains))
		for i, d := range org.Domains {
			domainStrings[i] = d.Domain
		}
		domains, diags := types.SetValueFrom(ctx, types.StringType, domainStrings)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Domains = domains
	} else {
		state.Domains = types.SetNull(types.StringType)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationResourceModel
	var state OrganizationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating organization", map[string]any{
		"id":   state.ID.ValueString(),
		"name": plan.Name.ValueString(),
	})

	// Build the update request
	updateReq := &client.OrganizationUpdateRequest{
		Name: plan.Name.ValueString(),
	}

	// Add domains if specified
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		var domains []string
		resp.Diagnostics.Append(plan.Domains.ElementsAs(ctx, &domains, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, domain := range domains {
			updateReq.DomainData = append(updateReq.DomainData, client.DomainData{
				Domain: domain,
				State:  "verified",
			})
		}
	}

	// Update the organization
	org, err := r.client.UpdateOrganization(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Organization",
			"Could not update organization, unexpected error: "+err.Error(),
		)
		return
	}

	// Update the plan with response data
	plan.ID = state.ID
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(org.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	tflog.Info(ctx, "Updated organization", map[string]any{
		"id":   org.ID,
		"name": org.Name,
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting organization", map[string]any{
		"id": state.ID.ValueString(),
	})

	// Delete the organization
	err := r.client.DeleteOrganization(ctx, state.ID.ValueString())
	if err != nil {
		// If the resource is already gone, that's fine
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Organization already deleted", map[string]any{
				"id": state.ID.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Organization",
			"Could not delete organization, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted organization", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing organization", map[string]any{
		"id": req.ID,
	})

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
