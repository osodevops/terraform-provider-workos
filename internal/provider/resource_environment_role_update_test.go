// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// environmentRoleMockAPI stands in for the WorkOS environment role endpoints.
// Like the organization role endpoints, its update response omits the role's
// permissions.
type environmentRoleMockAPI struct {
	mu    sync.Mutex
	roles map[string]*client.EnvironmentRole
}

func newEnvironmentRoleMockAPI(t *testing.T) (*environmentRoleMockAPI, *httptest.Server) {
	t.Helper()

	api := &environmentRoleMockAPI{roles: make(map[string]*client.EnvironmentRole)}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	return api, server
}

func (a *environmentRoleMockAPI) seedRole(role *client.EnvironmentRole) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.roles[role.Slug] = role
}

func (a *environmentRoleMockAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	rest := strings.TrimPrefix(r.URL.Path, "/authorization/roles/")
	if rest == r.URL.Path {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	parts := strings.Split(rest, "/")
	role, ok := a.roles[parts[0]]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, role)
	case len(parts) == 1 && r.Method == http.MethodPatch:
		var req client.EnvironmentRoleUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		if req.Name != "" {
			role.Name = req.Name
		}
		role.Description = req.Description
		role.UpdatedAt = time.Now().UTC().Truncate(time.Second)

		// The update response omits the role's permissions.
		response := *role
		response.Permissions = nil
		writeJSON(w, http.StatusOK, response)
	case len(parts) == 2 && parts[1] == "permissions" && r.Method == http.MethodPut:
		var req client.EnvironmentRolePermissionsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		role.Permissions = req.Permissions
		role.UpdatedAt = time.Now().UTC().Truncate(time.Second)
		writeJSON(w, http.StatusOK, role)
	default:
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}
}

func environmentRoleSchema(t *testing.T) schema.Schema {
	t.Helper()

	resp := &fwresource.SchemaResponse{}
	(&EnvironmentRoleResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// environmentRoleObject builds a complete object value for the resource schema,
// filling attributes missing from known with fill.
func environmentRoleObject(t *testing.T, s schema.Schema, known map[string]tftypes.Value, fill func(tftypes.Type) tftypes.Value) tftypes.Value {
	t.Helper()

	objectType, ok := s.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatalf("expected object type for schema, got %T", s.Type().TerraformType(context.Background()))
	}

	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		if value, found := known[name]; found {
			values[name] = value
			continue
		}
		values[name] = fill(attributeType)
	}

	return tftypes.NewValue(objectType, values)
}

func unknownFill(attributeType tftypes.Type) tftypes.Value {
	return tftypes.NewValue(attributeType, tftypes.UnknownValue)
}

func nullFill(attributeType tftypes.Type) tftypes.Value {
	return tftypes.NewValue(attributeType, nil)
}

func environmentRolePermissionsValue(permissions ...string) tftypes.Value {
	elements := make([]tftypes.Value, 0, len(permissions))
	for _, permission := range permissions {
		elements = append(elements, tftypes.NewValue(tftypes.String, permission))
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, elements)
}

// TestEnvironmentRoleResourceUpdate_PreservesUnmanagedPermissions covers the
// environment role variant of
// https://github.com/osodevops/terraform-provider-workos/issues/37. When
// permissions are left out of the configuration they are Computed and planned
// from prior state, so a name or description change must not rebuild them from
// an update response that omits them.
func TestEnvironmentRoleResourceUpdate_PreservesUnmanagedPermissions(t *testing.T) {
	ctx := context.Background()
	api, server := newEnvironmentRoleMockAPI(t)

	api.seedRole(&client.EnvironmentRole{
		ID:          "role_00000001",
		Object:      "role",
		Slug:        "billing-admin",
		Name:        "Billing Admin",
		Description: "Manages billing",
		Type:        "EnvironmentRole",
		Permissions: []string{"billing:read", "billing:write"},
	})

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	r := &EnvironmentRoleResource{client: workosClient}

	s := environmentRoleSchema(t)
	permissions := environmentRolePermissionsValue("billing:read", "billing:write")

	known := map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "role_00000001"),
		"slug":               tftypes.NewValue(tftypes.String, "billing-admin"),
		"name":               tftypes.NewValue(tftypes.String, "Billing Admin"),
		"description":        tftypes.NewValue(tftypes.String, "Manages billing"),
		"type":               tftypes.NewValue(tftypes.String, "EnvironmentRole"),
		"resource_type_slug": tftypes.NewValue(tftypes.String, nil),
		"permissions":        permissions,
		"created_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
		"updated_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
	}

	priorState := tfsdk.State{Schema: s, Raw: environmentRoleObject(t, s, known, nullFill)}

	planned := make(map[string]tftypes.Value, len(known))
	for name, value := range known {
		planned[name] = value
	}
	planned["description"] = tftypes.NewValue(tftypes.String, "Manages billing and invoices")
	delete(planned, "updated_at")
	plan := tfsdk.Plan{Schema: s, Raw: environmentRoleObject(t, s, planned, unknownFill)}

	// permissions is absent from the configuration, so it is null there.
	config := tfsdk.Config{Schema: s, Raw: environmentRoleObject(t, s, map[string]tftypes.Value{
		"slug":        tftypes.NewValue(tftypes.String, "billing-admin"),
		"name":        tftypes.NewValue(tftypes.String, "Billing Admin"),
		"description": tftypes.NewValue(tftypes.String, "Manages billing and invoices"),
	}, nullFill)}

	resp := &fwresource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: priorState.Raw}}
	r.Update(ctx, fwresource.UpdateRequest{Plan: plan, State: priorState, Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update returned errors: %v", resp.Diagnostics)
	}

	var state EnvironmentRoleResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}

	if state.Description.ValueString() != "Manages billing and invoices" {
		t.Fatalf("expected the description change to be applied, got %v", state.Description)
	}

	var applied []string
	if diags := state.Permissions.ElementsAs(ctx, &applied, false); diags.HasError() {
		t.Fatalf("failed to read permissions: %v", diags)
	}
	if len(applied) != 2 {
		t.Fatalf("expected permissions to survive the update, got %v", applied)
	}
}

// TestEnvironmentRoleResourceUpdate_AppliesManagedPermissions guards the other
// direction: when permissions are configured they are authoritative, and the
// replace-all endpoint's response is what must land in state.
func TestEnvironmentRoleResourceUpdate_AppliesManagedPermissions(t *testing.T) {
	ctx := context.Background()
	api, server := newEnvironmentRoleMockAPI(t)

	api.seedRole(&client.EnvironmentRole{
		ID:          "role_00000001",
		Object:      "role",
		Slug:        "billing-admin",
		Name:        "Billing Admin",
		Description: "Manages billing",
		Type:        "EnvironmentRole",
		Permissions: []string{"billing:read"},
	})

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	r := &EnvironmentRoleResource{client: workosClient}

	s := environmentRoleSchema(t)

	priorState := tfsdk.State{Schema: s, Raw: environmentRoleObject(t, s, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "role_00000001"),
		"slug":               tftypes.NewValue(tftypes.String, "billing-admin"),
		"name":               tftypes.NewValue(tftypes.String, "Billing Admin"),
		"description":        tftypes.NewValue(tftypes.String, "Manages billing"),
		"type":               tftypes.NewValue(tftypes.String, "EnvironmentRole"),
		"resource_type_slug": tftypes.NewValue(tftypes.String, nil),
		"permissions":        environmentRolePermissionsValue("billing:read"),
		"created_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
		"updated_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
	}, nullFill)}

	newPermissions := environmentRolePermissionsValue("billing:read", "billing:write")

	plan := tfsdk.Plan{Schema: s, Raw: environmentRoleObject(t, s, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "role_00000001"),
		"slug":               tftypes.NewValue(tftypes.String, "billing-admin"),
		"name":               tftypes.NewValue(tftypes.String, "Billing Admin"),
		"description":        tftypes.NewValue(tftypes.String, "Manages billing"),
		"type":               tftypes.NewValue(tftypes.String, "EnvironmentRole"),
		"resource_type_slug": tftypes.NewValue(tftypes.String, nil),
		"permissions":        newPermissions,
		"created_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
	}, unknownFill)}

	config := tfsdk.Config{Schema: s, Raw: environmentRoleObject(t, s, map[string]tftypes.Value{
		"slug":        tftypes.NewValue(tftypes.String, "billing-admin"),
		"name":        tftypes.NewValue(tftypes.String, "Billing Admin"),
		"description": tftypes.NewValue(tftypes.String, "Manages billing"),
		"permissions": newPermissions,
	}, nullFill)}

	resp := &fwresource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: priorState.Raw}}
	r.Update(ctx, fwresource.UpdateRequest{Plan: plan, State: priorState, Config: config}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update returned errors: %v", resp.Diagnostics)
	}

	var state EnvironmentRoleResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}

	var applied []string
	if diags := state.Permissions.ElementsAs(ctx, &applied, false); diags.HasError() {
		t.Fatalf("failed to read permissions: %v", diags)
	}
	if len(applied) != 2 {
		t.Fatalf("expected the configured permissions to be applied, got %v", applied)
	}
}
