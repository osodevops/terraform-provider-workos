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
	APIKey             types.String `tfsdk:"api_key"`
	ClientID           types.String `tfsdk:"client_id"`
	BaseURL            types.String `tfsdk:"base_url"`
	AccountAccessToken types.String `tfsdk:"account_access_token"`
	AccountTeamID      types.String `tfsdk:"account_team_id"`
	EnvironmentID      types.String `tfsdk:"environment_id"`
	ApplicationID      types.String `tfsdk:"application_id"`
}

func (p *WorkOSProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "workos"
	resp.Version = p.version
}

func (p *WorkOSProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The WorkOS provider allows you to manage WorkOS resources including organizations, " +
			"users, organization memberships, and roles through Terraform.",
		MarkdownDescription: `
The WorkOS provider allows you to manage WorkOS resources through Terraform.

## Authentication

REST resources require an environment-scoped WorkOS API key. You can provide this in three ways:

1. Set the ` + "`api_key`" + ` attribute in the provider configuration
2. Set the ` + "`WORKOS_API_KEY`" + ` environment variable
3. Use a combination of both (attribute takes precedence)

The account configuration resources additionally use a separate account OAuth access token
(` + "`account_access_token`" + ` or ` + "`WORKOS_ACCOUNT_ACCESS_TOKEN`" + `), an expected team
(` + "`account_team_id`" + ` or ` + "`WORKOS_ACCOUNT_TEAM_ID`" + `), and explicit provider
` + "`environment_id`" + ` and ` + "`application_id`" + ` attributes. This token is only for
Terraform configuration, never application runtime. Account-plane access is gated by a WorkOS
team feature flag. Missing access is an error, with no fallback to another credential or environment.

Before exposing a client to resources, the provider verifies the exact account target and, when
an API key is also configured, matches the complete unscoped role ID sets returned by REST and
GraphQL. Empty or inconsistent evidence prevents every resource operation. Account-only use is
supported; REST resources still require their own API key.

~> **Experimental.** The account configuration resources call the WorkOS dashboard ("account
plane") GraphQL API through the same operations as the official WorkOS CLI. That API is not a
documented, versioned public API: WorkOS gates it per team and may change or remove operations
without notice, so pin the provider version. The account access token is the short-lived OAuth
access token that ` + "`workos auth login`" + ` stores for the CLI. The CLI refreshes it in the
background; this provider does not. Supply a current token for each run and expect an HTTP 401
diagnostic once it expires. It is not a long-lived CI secret.

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
			"environment_id": schema.StringAttribute{Optional: true, Description: "Explicit account configuration environment. Required with account_access_token and checked against the account team and REST key before any resource can mutate."},
			"application_id": schema.StringAttribute{Optional: true, Description: "Explicit default AuthKit application ID. Required with account_access_token. All configuration resource targets must match the provider target."},
			"account_access_token": schema.StringAttribute{
				Description: "Account OAuth access token for configuration resources only. Can also be set via WORKOS_ACCOUNT_ACCESS_TOKEN. Never used for REST resources or application runtime. Experimental: this is the short-lived token from the WorkOS CLI session (workos auth login) and is not refreshed by the provider.",
				Optional:    true, Sensitive: true,
			},
			"account_team_id": schema.StringAttribute{
				Description: "Expected WorkOS account team ID. Required with account_access_token and verified before configuration reads and writes. Can also be set via WORKOS_ACCOUNT_TEAM_ID.",
				Optional:    true,
			},
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
	accountAccessToken := os.Getenv("WORKOS_ACCOUNT_ACCESS_TOKEN")
	accountTeamID := os.Getenv("WORKOS_ACCOUNT_TEAM_ID")

	for name, value := range map[string]types.String{"api_key": config.APIKey, "client_id": config.ClientID, "base_url": config.BaseURL, "account_access_token": config.AccountAccessToken, "account_team_id": config.AccountTeamID, "environment_id": config.EnvironmentID, "application_id": config.ApplicationID} {
		if value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Unknown WorkOS Configuration", "Provider configuration values must be known before configuring the client.")
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if !config.AccountAccessToken.IsNull() {
		accountAccessToken = config.AccountAccessToken.ValueString()
	}
	if !config.AccountTeamID.IsNull() {
		accountTeamID = config.AccountTeamID.ValueString()
	}

	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if !config.ClientID.IsNull() {
		clientID = config.ClientID.ValueString()
	}

	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	if apiKey == "" && accountAccessToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing WorkOS API Key",
			"The provider cannot create the WorkOS API client as there is a missing or empty value for the WorkOS API key. "+
				"Set api_key or WORKOS_API_KEY for REST resources, or account_access_token and account_team_id for account configuration resources. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if accountAccessToken != "" && accountTeamID == "" {
		resp.Diagnostics.AddAttributeError(path.Root("account_team_id"), "Missing WorkOS Account Team", "Set account_team_id or WORKOS_ACCOUNT_TEAM_ID to the exact team owning the configured environment.")
	}
	if accountAccessToken != "" && (config.EnvironmentID.ValueString() == "" || config.ApplicationID.ValueString() == "") {
		resp.Diagnostics.AddError("Missing WorkOS Account Target", "Set explicit provider environment_id and application_id with account OAuth; production fallback is forbidden.")
	}
	if accountAccessToken == "" && (!config.EnvironmentID.IsNull() || !config.ApplicationID.IsNull()) {
		resp.Diagnostics.AddAttributeError(path.Root("account_access_token"), "Missing WorkOS Account Credential", "An explicit account configuration target requires account_access_token or WORKOS_ACCOUNT_ACCESS_TOKEN before any resources can run.")
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
	workosClient, err := client.NewClientWithCredentials(apiKey, clientID, baseURL, accountAccessToken, accountTeamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create WorkOS API Client",
			"An unexpected error occurred when creating the WorkOS API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"WorkOS Client Error: "+err.Error(),
		)
		return
	}
	if accountAccessToken != "" {
		if err := workosClient.PreflightAccountConfiguration(ctx, config.EnvironmentID.ValueString(), config.ApplicationID.ValueString()); err != nil {
			resp.Diagnostics.AddError("WorkOS Account Preflight Failed", err.Error())
			return
		}
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
		NewOrganizationDomainResource,
		NewUserResource,
		NewOrganizationMembershipResource,
		NewGroupResource,
		NewGroupMembershipResource,
		NewConnectApplicationResource,
		NewRedirectURIResource,
		NewConnectApplicationClientSecretResource,
		NewEnvironmentRoleResource,
		NewAuthkitConfigurationResource,
		NewEnvironmentRoleConfigurationResource,
		NewOrganizationRoleResource,
		NewPermissionResource,
		NewOrganizationRolePermissionResource,
		NewAuthorizationResourceResource,
		NewAuthorizationRoleAssignmentResource,
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
		NewEnvironmentRoleDataSource,
		NewOrganizationRoleDataSource,
		NewPermissionDataSource,
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
