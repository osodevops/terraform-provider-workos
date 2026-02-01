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
var _ datasource.DataSource = &ConnectionDataSource{}
var _ datasource.DataSourceWithConfigValidators = &ConnectionDataSource{}

func NewConnectionDataSource() datasource.DataSource {
	return &ConnectionDataSource{}
}

// ConnectionDataSource defines the data source implementation.
type ConnectionDataSource struct {
	client *client.Client
}

// ConnectionDataSourceModel describes the data source data model.
type ConnectionDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ConnectionType types.String `tfsdk:"connection_type"`
	Name           types.String `tfsdk:"name"`
	State          types.String `tfsdk:"state"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *ConnectionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection"
}

func (d *ConnectionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a WorkOS SSO Connection.",
		MarkdownDescription: `
Use this data source to get information about a WorkOS SSO Connection.

You can look up a connection by its ID, or by organization ID and connection type.

## Example Usage

### By ID

` + "```hcl" + `
data "workos_connection" "example" {
  id = "conn_01HXYZ..."
}
` + "```" + `

### By Organization and Type

` + "```hcl" + `
data "workos_connection" "okta" {
  organization_id = workos_organization.main.id
  connection_type = "OktaSAML"
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the connection to look up.",
				MarkdownDescription: "The unique identifier of the connection to look up (e.g., `conn_01HXYZ...`).",
				Optional:            true,
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				Description:         "The organization ID to filter connections.",
				MarkdownDescription: "The organization ID to filter connections. Must be used with `connection_type`.",
				Optional:            true,
				Computed:            true,
			},
			"connection_type": schema.StringAttribute{
				Description:         "The connection type to filter by.",
				MarkdownDescription: "The connection type to filter by (e.g., `OktaSAML`). Must be used with `organization_id`.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The name of the connection.",
				MarkdownDescription: "The friendly name of the connection.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				Description:         "The current state of the connection.",
				MarkdownDescription: "The current state of the connection (`active`, `inactive`, `validating`).",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "The configuration status of the connection.",
				MarkdownDescription: "The configuration status of the connection (`linked`, `unlinked`).",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The timestamp when the connection was created.",
				MarkdownDescription: "The timestamp when the connection was created (RFC3339 format).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				Description:         "The timestamp when the connection was last updated.",
				MarkdownDescription: "The timestamp when the connection was last updated (RFC3339 format).",
				Computed:            true,
			},
		},
	}
}

func (d *ConnectionDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("organization_id"),
		),
		datasourcevalidator.RequiredTogether(
			path.MatchRoot("organization_id"),
			path.MatchRoot("connection_type"),
		),
	}
}

func (d *ConnectionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ConnectionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var conn *client.Connection
	var err error

	if !config.ID.IsNull() {
		// Look up by ID
		tflog.Debug(ctx, "Reading connection by ID", map[string]any{
			"id": config.ID.ValueString(),
		})

		conn, err = d.client.GetConnection(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Connection",
				"Could not read connection ID "+config.ID.ValueString()+": "+err.Error(),
			)
			return
		}
	} else if !config.OrganizationID.IsNull() && !config.ConnectionType.IsNull() {
		// Look up by organization and type
		tflog.Debug(ctx, "Reading connection by organization and type", map[string]any{
			"organization_id": config.OrganizationID.ValueString(),
			"connection_type": config.ConnectionType.ValueString(),
		})

		conn, err = d.client.GetConnectionByOrganizationAndType(
			ctx,
			config.OrganizationID.ValueString(),
			config.ConnectionType.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Connection",
				fmt.Sprintf("Could not find connection for organization %s with type %s: %s",
					config.OrganizationID.ValueString(),
					config.ConnectionType.ValueString(),
					err.Error()),
			)
			return
		}
	}

	// Map response to state
	config.ID = types.StringValue(conn.ID)
	config.OrganizationID = types.StringValue(conn.OrganizationID)
	config.ConnectionType = types.StringValue(conn.ConnectionType)
	config.Name = types.StringValue(conn.Name)
	config.State = types.StringValue(conn.State)
	config.Status = types.StringValue(conn.Status)
	config.CreatedAt = types.StringValue(conn.CreatedAt.Format("2006-01-02T15:04:05Z"))
	config.UpdatedAt = types.StringValue(conn.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	tflog.Info(ctx, "Read connection", map[string]any{
		"id":              conn.ID,
		"connection_type": conn.ConnectionType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
