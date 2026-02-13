// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// Ensure WorkOSProvider satisfies various provider interfaces.
var _ provider.Provider = &WorkOSProvider{}

// WorkOSProvider defines the provider implementation.
type WorkOSProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and run locally, and "test" when running acceptance
	// testing.
	version string
}

// WorkOSProviderModel describes the provider data model.
type WorkOSProviderModel struct {
	APIKey   types.String `tfsdk:"api_key"`
	ClientID types.String `tfsdk:"client_id"`
	BaseURL  types.String `tfsdk:"base_url"`
}

func (p *WorkOSProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "workos"
	resp.Version = p.version
}

func (p *WorkOSProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The WorkOS provider allows you to manage WorkOS resources including organizations, " +
			"SSO connections, directory sync, webhooks, and user management through Terraform.",
		MarkdownDescription: `
The WorkOS provider allows you to manage WorkOS resources through Terraform.

## Authentication

The provider requires a WorkOS API key for authentication. You can provide this in three ways:

1. Set the ` + "`api_key`" + ` attribute in the provider configuration
2. Set the ` + "`WORKOS_API_KEY`" + ` environment variable
3. Use a combination of both (attribute takes precedence)

## Example Usage

` + "```hcl" + `
provider "workos" {
  api_key = var.workos_api_key
}

resource "workos_organization" "example" {
  name    = "Acme Corporation"
  domains = ["acme.com"]
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "The WorkOS API key (starts with sk_). " +
					"Can also be set via the WORKOS_API_KEY environment variable.",
				MarkdownDescription: "The WorkOS API key (starts with `sk_`). " +
					"Can also be set via the `WORKOS_API_KEY` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"client_id": schema.StringAttribute{
				Description: "The WorkOS Client ID. Required for certain operations. " +
					"Can also be set via the WORKOS_CLIENT_ID environment variable.",
				MarkdownDescription: "The WorkOS Client ID. Required for certain operations. " +
					"Can also be set via the `WORKOS_CLIENT_ID` environment variable.",
				Optional: true,
			},
			"base_url": schema.StringAttribute{
				Description: "The WorkOS API base URL. Defaults to https://api.workos.com. " +
					"Can also be set via the WORKOS_BASE_URL environment variable.",
				MarkdownDescription: "The WorkOS API base URL. Defaults to `https://api.workos.com`. " +
					"Can also be set via the `WORKOS_BASE_URL` environment variable.",
				Optional: true,
			},
		},
	}
}

func (p *WorkOSProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring WorkOS client")

	var config WorkOSProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	apiKey := os.Getenv("WORKOS_API_KEY")
	clientID := os.Getenv("WORKOS_CLIENT_ID")
	baseURL := os.Getenv("WORKOS_BASE_URL")

	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if !config.ClientID.IsNull() {
		clientID = config.ClientID.ValueString()
	}

	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	// If API key is not configured, return an error
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing WorkOS API Key",
			"The provider cannot create the WorkOS API client as there is a missing or empty value for the WorkOS API key. "+
				"Set the api_key value in the configuration or use the WORKOS_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default base URL if not set
	if baseURL == "" {
		baseURL = "https://api.workos.com"
	}

	ctx = tflog.SetField(ctx, "workos_base_url", baseURL)
	ctx = tflog.SetField(ctx, "workos_client_id", clientID)
	// Intentionally not logging API key even masked

	tflog.Debug(ctx, "Creating WorkOS client")

	// Create a new WorkOS client using the configuration values
	workosClient, err := client.NewClient(apiKey, clientID, baseURL)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create WorkOS API Client",
			"An unexpected error occurred when creating the WorkOS API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"WorkOS Client Error: "+err.Error(),
		)
		return
	}

	// Make the WorkOS client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = workosClient
	resp.ResourceData = workosClient

	tflog.Info(ctx, "Configured WorkOS client", map[string]any{"success": true})
}

func (p *WorkOSProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOrganizationResource,
		NewConnectionResource,
		NewDirectoryResource,
		NewWebhookResource,
		NewUserResource,
		NewOrganizationMembershipResource,
		NewOrganizationRoleResource,
	}
}

func (p *WorkOSProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOrganizationDataSource,
		NewConnectionDataSource,
		NewDirectoryDataSource,
		NewDirectoryUserDataSource,
		NewDirectoryGroupDataSource,
		NewUserDataSource,
		NewOrganizationRoleDataSource,
	}
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &WorkOSProvider{
			version: version,
		}
	}
}
