// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &OrganizationDataSource{}
var _ datasource.DataSourceWithConfigValidators = &OrganizationDataSource{}

func NewOrganizationDataSource() datasource.DataSource {
	return &OrganizationDataSource{}
}

// OrganizationDataSource defines the data source implementation.
type OrganizationDataSource struct {
	client *client.Client
}

// OrganizationDataSourceModel describes the data source data model.
type OrganizationDataSourceModel struct {
	ID                               types.String `tfsdk:"id"`
	Domain                           types.String `tfsdk:"domain"`
	Name                             types.String `tfsdk:"name"`
	Domains                          types.Set    `tfsdk:"domains"`
	AllowProfilesOutsideOrganization types.Bool   `tfsdk:"allow_profiles_outside_organization"`
	CreatedAt                        types.String `tfsdk:"created_at"`
	UpdatedAt                        types.String `tfsdk:"updated_at"`
}

func (d *OrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *OrganizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a WorkOS Organization.",
		MarkdownDescription: `
Use this data source to get information about a WorkOS Organization.

You can look up an organization by its ID or by one of its domains.

## Example Usage

### By ID

` + "```hcl" + `
data "workos_organization" "example" {
  id = "org_01HXYZ..."
}
` + "```" + `

### By Domain

` + "```hcl" + `
data "workos_organization" "example" {
  domain = "acme.com"
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the organization to look up.",
				MarkdownDescription: "The unique identifier of the organization to look up (e.g., `org_01HXYZ...`).",
				Optional:            true,
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				Description:         "A domain associated with the organization to look up.",
				MarkdownDescription: "A domain associated with the organization to look up. The organization that owns this domain will be returned.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The name of the organization.",
				MarkdownDescription: "The name of the organization.",
				Computed:            true,
			},
			"domains": schema.SetAttribute{
				Description:         "The domains associated with the organization.",
				MarkdownDescription: "The domains associated with the organization.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"allow_profiles_outside_organization": schema.BoolAttribute{
				Description:         "Whether user profiles outside the organization are allowed.",
				MarkdownDescription: "Whether user profiles that don't belong to this organization are allowed.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the organization was created.",
				MarkdownDescription: "The timestamp when the organization was created (RFC3339 format).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the organization was last updated.",
				MarkdownDescription: "The timestamp when the organization was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (d *OrganizationDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("domain"),
		),
	}
}

func (d *OrganizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config OrganizationDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var org *client.Organization
	var err error

	if !config.ID.IsNull() {
		// Look up by ID
		tflog.Debug(ctx, "Reading organization by ID", map[string]any{
			"id": config.ID.ValueString(),
		})

		org, err = d.client.GetOrganization(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Organization",
				"Could not read organization ID "+config.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	} else if !config.Domain.IsNull() {
		// Look up by domain
		tflog.Debug(ctx, "Reading organization by domain", map[string]any{
			"domain": config.Domain.ValueString(),
		})

		org, err = d.client.GetOrganizationByDomain(ctx, config.Domain.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Organization",
				"Could not find organization with domain "+config.Domain.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Map response to state
	config.ID = types.StringValue(org.ID)
	config.Name = types.StringValue(org.Name)
	config.AllowProfilesOutsideOrganization = types.BoolValue(org.AllowProfilesOutsideOrganization)
	config.CreatedAt = types.StringValue(org.CreatedAt.Format("2006-01-02T15:04:05Z"))
	config.UpdatedAt = types.StringValue(org.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	// Map domains
	if len(org.Domains) > 0 {
		domainStrings := make([]string, len(org.Domains))
		for i, dom := range org.Domains {
			domainStrings[i] = dom.Domain
		}
		domains, diags := types.SetValueFrom(ctx, types.StringType, domainStrings)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		config.Domains = domains
	} else {
		config.Domains = types.SetNull(types.StringType)
	}

	tflog.Info(ctx, "Read organization", map[string]any{
		"id":   org.ID,
		"name": org.Name,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
