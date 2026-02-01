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
var _ resource.Resource = &DirectoryResource{}
var _ resource.ResourceWithImportState = &DirectoryResource{}

func NewDirectoryResource() resource.Resource {
	return &DirectoryResource{}
}

// DirectoryResource defines the resource implementation.
type DirectoryResource struct {
	client *client.Client
}

// DirectoryResourceModel describes the resource data model.
type DirectoryResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	State          types.String `tfsdk:"state"`
	BearerToken    types.String `tfsdk:"bearer_token"`
	Endpoint       types.String `tfsdk:"endpoint"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *DirectoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory"
}

func (r *DirectoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WorkOS Directory Sync directory.",
		MarkdownDescription: `
Manages a WorkOS Directory Sync directory.

Directories enable SCIM-based user and group synchronization from identity providers
like Okta, Azure AD, Google Workspace, and others.

## Example Usage

` + "```hcl" + `
resource "workos_directory" "okta" {
  organization_id = workos_organization.main.id
  name            = "Okta Directory"
  type            = "okta scim v2.0"
}

output "scim_endpoint" {
  value = workos_directory.okta.endpoint
}

output "scim_bearer_token" {
  value     = workos_directory.okta.bearer_token
  sensitive = true
}
` + "```" + `

## Supported Directory Types

- ` + "`azure scim v2.0`" + ` - Azure AD SCIM
- ` + "`okta scim v2.0`" + ` - Okta SCIM
- ` + "`generic scim v2.0`" + ` - Generic SCIM 2.0
- ` + "`google workspace`" + ` - Google Workspace
- ` + "`workday`" + ` - Workday
- ` + "`bamboohr`" + ` - BambooHR
- ` + "`breathehr`" + ` - BreatheHR
- ` + "`cezannehr`" + ` - CezanneHR
- ` + "`cyberark scim v2.0`" + ` - CyberArk SCIM
- ` + "`fourth hr`" + ` - Fourth HR
- ` + "`hibob`" + ` - HiBob
- ` + "`jump cloud scim v2.0`" + ` - JumpCloud SCIM
- ` + "`onelogin scim v2.0`" + ` - OneLogin SCIM
- ` + "`peopleforce`" + ` - PeopleForce
- ` + "`personio`" + ` - Personio
- ` + "`pingfederate scim v2.0`" + ` - PingFederate SCIM
- ` + "`rippling scim v2.0`" + ` - Rippling SCIM

## Import

Directories can be imported using the directory ID:

` + "```shell" + `
terraform import workos_directory.example directory_01HXYZ...
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the directory.",
				MarkdownDescription: "The unique identifier of the directory (e.g., `directory_01HXYZ...`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description:         "The ID of the organization this directory belongs to.",
				MarkdownDescription: "The ID of the organization this directory belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "A friendly name for the directory.",
				MarkdownDescription: "A friendly name for the directory.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				Description:         "The type of directory.",
				MarkdownDescription: "The type of directory (e.g., `okta scim v2.0`, `azure scim v2.0`). Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"state": schema.StringAttribute{
				Description:         "The current state of the directory.",
				MarkdownDescription: "The current state of the directory (`linked`, `unlinked`, `invalid_credentials`, `deleting`).",
				Computed:            true,
			},
			"bearer_token": schema.StringAttribute{
				Description:         "The SCIM bearer token for this directory.",
				MarkdownDescription: "The SCIM bearer token for this directory. Use this token to authenticate SCIM requests.",
				Computed:            true,
				Sensitive:           true,
			},
			"endpoint": schema.StringAttribute{
				Description:         "The SCIM endpoint URL for this directory.",
				MarkdownDescription: "The SCIM endpoint URL for this directory. Configure your IdP to send SCIM requests to this URL.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the directory was created.",
				MarkdownDescription: "The timestamp when the directory was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the directory was last updated.",
				MarkdownDescription: "The timestamp when the directory was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (r *DirectoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DirectoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DirectoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating directory", map[string]any{
		"organization_id": plan.OrganizationID.ValueString(),
		"type":            plan.Type.ValueString(),
	})

	createReq := &client.DirectoryCreateRequest{
		OrganizationID: plan.OrganizationID.ValueString(),
		Name:           plan.Name.ValueString(),
		Type:           plan.Type.ValueString(),
	}

	dir, err := r.client.CreateDirectory(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Directory",
			"Could not create directory, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(dir.ID)
	plan.State = types.StringValue(dir.State)
	if dir.BearerToken != "" {
		plan.BearerToken = types.StringValue(dir.BearerToken)
	} else {
		plan.BearerToken = types.StringNull()
	}
	if dir.Endpoint != "" {
		plan.Endpoint = types.StringValue(dir.Endpoint)
	} else {
		plan.Endpoint = types.StringNull()
	}
	plan.CreatedAt = types.StringValue(dir.CreatedAt.Format("2006-01-02T15:04:05Z"))
	plan.UpdatedAt = types.StringValue(dir.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	tflog.Info(ctx, "Created directory", map[string]any{
		"id":   dir.ID,
		"type": dir.Type,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DirectoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DirectoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading directory", map[string]any{
		"id": state.ID.ValueString(),
	})

	dir, err := r.client.GetDirectory(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Directory not found, removing from state", map[string]any{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Directory",
			"Could not read directory ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state
	state.OrganizationID = types.StringValue(dir.OrganizationID)
	state.Name = types.StringValue(dir.Name)
	state.Type = types.StringValue(dir.Type)
	state.State = types.StringValue(dir.State)
	if dir.BearerToken != "" {
		state.BearerToken = types.StringValue(dir.BearerToken)
	} else {
		state.BearerToken = types.StringNull()
	}
	if dir.Endpoint != "" {
		state.Endpoint = types.StringValue(dir.Endpoint)
	} else {
		state.Endpoint = types.StringNull()
	}
	state.CreatedAt = types.StringValue(dir.CreatedAt.Format("2006-01-02T15:04:05Z"))
	state.UpdatedAt = types.StringValue(dir.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DirectoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DirectoryResourceModel
	var state DirectoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating directory", map[string]any{
		"id":   state.ID.ValueString(),
		"name": plan.Name.ValueString(),
	})

	updateReq := &client.DirectoryUpdateRequest{
		Name: plan.Name.ValueString(),
	}

	dir, err := r.client.UpdateDirectory(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Directory",
			"Could not update directory, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = state.ID
	plan.State = types.StringValue(dir.State)
	if dir.BearerToken != "" {
		plan.BearerToken = types.StringValue(dir.BearerToken)
	} else {
		plan.BearerToken = state.BearerToken
	}
	if dir.Endpoint != "" {
		plan.Endpoint = types.StringValue(dir.Endpoint)
	} else {
		plan.Endpoint = state.Endpoint
	}
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(dir.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	tflog.Info(ctx, "Updated directory", map[string]any{
		"id": dir.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DirectoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DirectoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting directory", map[string]any{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteDirectory(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Directory already deleted", map[string]any{
				"id": state.ID.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Directory",
			"Could not delete directory, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted directory", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *DirectoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing directory", map[string]any{
		"id": req.ID,
	})

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
