// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

func TestEnvironmentRoleDataSource_BySlug(t *testing.T) {
	server := httptest.NewServer(environmentRoleDataSourceHandler(t))
	defer server.Close()

	state := readEnvironmentRoleDataSource(t, server.URL, EnvironmentRoleDataSourceModel{
		ID:               types.StringNull(),
		Slug:             types.StringValue("admin"),
		Name:             types.StringNull(),
		Description:      types.StringNull(),
		Type:             types.StringNull(),
		ResourceTypeSlug: types.StringNull(),
		Permissions:      types.SetNull(types.StringType),
		CreatedAt:        types.StringNull(),
		UpdatedAt:        types.StringNull(),
	})

	if state.ID.ValueString() != "role_01JYQ5B9Q6ZP8K4R2T1V0X9ABC" {
		t.Fatalf("unexpected ID: %s", state.ID.ValueString())
	}
	if state.Slug.ValueString() != "admin" {
		t.Fatalf("unexpected slug: %s", state.Slug.ValueString())
	}
	if state.Name.ValueString() != "Admin" {
		t.Fatalf("unexpected name: %s", state.Name.ValueString())
	}
	if state.Type.ValueString() != "EnvironmentRole" {
		t.Fatalf("unexpected type: %s", state.Type.ValueString())
	}
	if len(state.Permissions.Elements()) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(state.Permissions.Elements()))
	}
	if state.CreatedAt.ValueString() == "" {
		t.Fatal("expected created_at to be set")
	}
}

func TestEnvironmentRoleDataSource_ByID(t *testing.T) {
	server := httptest.NewServer(environmentRoleDataSourceHandler(t))
	defer server.Close()

	state := readEnvironmentRoleDataSource(t, server.URL, EnvironmentRoleDataSourceModel{
		ID:               types.StringValue("role_01JYQ5B9Q6ZP8K4R2T1V0X9ABC"),
		Slug:             types.StringNull(),
		Name:             types.StringNull(),
		Description:      types.StringNull(),
		Type:             types.StringNull(),
		ResourceTypeSlug: types.StringNull(),
		Permissions:      types.SetNull(types.StringType),
		CreatedAt:        types.StringNull(),
		UpdatedAt:        types.StringNull(),
	})

	if state.ID.ValueString() != "role_01JYQ5B9Q6ZP8K4R2T1V0X9ABC" {
		t.Fatalf("unexpected ID: %s", state.ID.ValueString())
	}
	if state.Slug.ValueString() != "admin" {
		t.Fatalf("unexpected slug: %s", state.Slug.ValueString())
	}
	if state.Name.ValueString() != "Admin" {
		t.Fatalf("unexpected name: %s", state.Name.ValueString())
	}
}

func environmentRoleDataSourceHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test" {
			t.Fatalf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/authorization/roles/admin":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			_, _ = w.Write([]byte(testEnvironmentRoleJSON()))
		case "/authorization/roles":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"data":[` + testEnvironmentRoleJSON() + `],"list_metadata":{}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	})
}

func readEnvironmentRoleDataSource(t *testing.T, baseURL string, config EnvironmentRoleDataSourceModel) EnvironmentRoleDataSourceModel {
	t.Helper()

	ctx := context.Background()
	workosClient, err := client.NewClient("sk_test", "", baseURL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	dataSource := &EnvironmentRoleDataSource{}
	configureResp := &datasource.ConfigureResponse{}
	dataSource.Configure(ctx, datasource.ConfigureRequest{ProviderData: workosClient}, configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %v", configureResp.Diagnostics)
	}

	schemaResp := &datasource.SchemaResponse{}
	dataSource.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	configState := tfsdk.State{Schema: schemaResp.Schema}
	diags := configState.Set(ctx, &config)
	if diags.HasError() {
		t.Fatalf("failed to build config state: %v", diags)
	}

	readResp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	dataSource.Read(ctx, datasource.ReadRequest{
		Config: tfsdk.Config{
			Raw:    configState.Raw,
			Schema: schemaResp.Schema,
		},
	}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %v", readResp.Diagnostics)
	}

	var state EnvironmentRoleDataSourceModel
	diags = readResp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("failed to read state: %v", diags)
	}

	return state
}

func testEnvironmentRoleJSON() string {
	return `{
  "id": "role_01JYQ5B9Q6ZP8K4R2T1V0X9ABC",
  "object": "role",
  "slug": "admin",
  "name": "Admin",
  "description": "Can manage all resources",
  "type": "EnvironmentRole",
  "resource_type_slug": "organization",
  "permissions": ["billing:read", "billing:write"],
  "created_at": "2026-01-15T12:00:00.000Z",
  "updated_at": "2026-01-15T12:00:00.000Z"
}`
}
