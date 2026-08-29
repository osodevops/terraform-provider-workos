// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// organizationRoleMockAPI is a minimal in-memory stand-in for the WorkOS
// authorization API covering the endpoints the organization role resource, its
// permission assignments, and the organization resource they hang off exercise.
//
// Its defining behaviour is that the role update endpoint does not echo
// permissions back, exactly as the WorkOS API behaves. That omission is what
// https://github.com/osodevops/terraform-provider-workos/issues/37 trips over.
type organizationRoleMockAPI struct {
	mu            sync.Mutex
	counter       int
	organizations map[string]*client.Organization
	// roles is keyed by "<organization id>/<role slug>".
	roles map[string]*client.OrganizationRole

	// echoPermissionsOnUpdate makes the update endpoint return the role's
	// permissions, as an API that returns the full resource would. WorkOS does
	// not, so this defaults to false.
	echoPermissionsOnUpdate bool
}

func newOrganizationRoleMockAPI(t *testing.T) (*organizationRoleMockAPI, *httptest.Server) {
	t.Helper()

	api := &organizationRoleMockAPI{
		organizations: make(map[string]*client.Organization),
		roles:         make(map[string]*client.OrganizationRole),
	}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	return api, server
}

func (a *organizationRoleMockAPI) nextID(prefix string) string {
	a.counter++
	return fmt.Sprintf("%s_%08d", prefix, a.counter)
}

// seedRole registers a role directly, for tests that drive the resource methods
// without going through a create.
func (a *organizationRoleMockAPI) seedRole(orgID string, role *client.OrganizationRole) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.roles[orgID+"/"+role.Slug] = role
}

func (a *organizationRoleMockAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch {
	case r.URL.Path == "/organizations" && r.Method == http.MethodPost:
		a.createOrganization(w, r)
	case strings.HasPrefix(r.URL.Path, "/organizations/"):
		a.organizationByID(w, r, strings.TrimPrefix(r.URL.Path, "/organizations/"))
	case strings.HasPrefix(r.URL.Path, "/authorization/organizations/"):
		a.authorization(w, r, strings.TrimPrefix(r.URL.Path, "/authorization/organizations/"))
	default:
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}
}

func (a *organizationRoleMockAPI) createOrganization(w http.ResponseWriter, r *http.Request) {
	var req client.OrganizationCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	org := &client.Organization{
		ID:        a.nextID("org"),
		Object:    "organization",
		Name:      req.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	a.organizations[org.ID] = org
	writeJSON(w, http.StatusCreated, org)
}

func (a *organizationRoleMockAPI) organizationByID(w http.ResponseWriter, r *http.Request, id string) {
	org, ok := a.organizations[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, org)
	case http.MethodDelete:
		delete(a.organizations, id)
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, `{"message":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// authorization routes the paths under /authorization/organizations/, which are
// "<org id>/roles[/<slug>[/permissions[/<permission slug>]]]".
func (a *organizationRoleMockAPI) authorization(w http.ResponseWriter, r *http.Request, rest string) {
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[1] != "roles" {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	orgID := parts[0]

	switch {
	case len(parts) == 2 && r.Method == http.MethodPost:
		a.createRole(w, r, orgID)
	case len(parts) == 2 && r.Method == http.MethodGet:
		a.listRoles(w, orgID)
	case len(parts) == 3:
		a.roleBySlug(w, r, orgID, parts[2])
	case len(parts) == 4 && parts[3] == "permissions" && r.Method == http.MethodPost:
		a.addPermission(w, r, orgID, parts[2])
	case len(parts) == 5 && parts[3] == "permissions" && r.Method == http.MethodDelete:
		a.removePermission(w, orgID, parts[2], parts[4])
	default:
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}
}

func (a *organizationRoleMockAPI) createRole(w http.ResponseWriter, r *http.Request, orgID string) {
	var req client.OrganizationRoleCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	role := &client.OrganizationRole{
		ID:               a.nextID("role"),
		Object:           "role",
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      req.Description,
		Type:             "OrganizationRole",
		Permissions:      []string{},
		ResourceTypeSlug: req.ResourceTypeSlug,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	a.roles[orgID+"/"+role.Slug] = role
	writeJSON(w, http.StatusCreated, role)
}

func (a *organizationRoleMockAPI) listRoles(w http.ResponseWriter, orgID string) {
	resp := client.OrganizationRoleListResponse{Data: []client.OrganizationRole{}}
	for key, role := range a.roles {
		if strings.HasPrefix(key, orgID+"/") {
			resp.Data = append(resp.Data, *role)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *organizationRoleMockAPI) roleBySlug(w http.ResponseWriter, r *http.Request, orgID, slug string) {
	key := orgID + "/" + slug
	role, ok := a.roles[key]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, role)
	case http.MethodPatch:
		var req client.OrganizationRoleUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		if req.Name != "" {
			role.Name = req.Name
		}
		role.Description = req.Description
		role.UpdatedAt = time.Now().UTC().Truncate(time.Second)

		// WorkOS' update response omits the role's permissions. Reproduce that
		// here so the provider is exercised against the real response shape.
		response := *role
		if !a.echoPermissionsOnUpdate {
			response.Permissions = nil
		}
		writeJSON(w, http.StatusOK, response)
	case http.MethodDelete:
		delete(a.roles, key)
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, `{"message":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (a *organizationRoleMockAPI) addPermission(w http.ResponseWriter, r *http.Request, orgID, slug string) {
	role, ok := a.roles[orgID+"/"+slug]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	var req client.AddPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
		return
	}

	for _, existing := range role.Permissions {
		if existing == req.Slug {
			writeJSON(w, http.StatusOK, role)
			return
		}
	}
	role.Permissions = append(role.Permissions, req.Slug)
	writeJSON(w, http.StatusCreated, role)
}

func (a *organizationRoleMockAPI) removePermission(w http.ResponseWriter, orgID, slug, permission string) {
	role, ok := a.roles[orgID+"/"+slug]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	remaining := make([]string, 0, len(role.Permissions))
	for _, existing := range role.Permissions {
		if existing != permission {
			remaining = append(remaining, existing)
		}
	}
	role.Permissions = remaining
	w.WriteHeader(http.StatusAccepted)
}

// organizationRoleSchema returns the resource schema under test.
func organizationRoleSchema(t *testing.T) schema.Schema {
	t.Helper()

	resp := &fwresource.SchemaResponse{}
	(&OrganizationRoleResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// organizationRoleObject builds a complete object value for the resource
// schema. Attributes missing from known are filled with unknown, matching the
// plan Terraform hands a provider for Computed values it cannot predict.
func organizationRoleObject(t *testing.T, s schema.Schema, known map[string]tftypes.Value) tftypes.Value {
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
		values[name] = tftypes.NewValue(attributeType, tftypes.UnknownValue)
	}

	return tftypes.NewValue(objectType, values)
}

func organizationRolePermissionsList(permissions ...string) tftypes.Value {
	elements := make([]tftypes.Value, 0, len(permissions))
	for _, permission := range permissions {
		elements = append(elements, tftypes.NewValue(tftypes.String, permission))
	}
	return tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, elements)
}

// TestOrganizationRoleResourceUpdate_PreservesAssignedPermissions reproduces
// https://github.com/osodevops/terraform-provider-workos/issues/37: changing
// name or description on a role that has permissions assigned dropped every
// permission from state, because Update rebuilt the list from an update
// response that never carries one. permissions is Computed with
// UseStateForUnknown, so its planned value is the prior state value and
// Terraform rejects any other applied value with "element N has vanished".
func TestOrganizationRoleResourceUpdate_PreservesAssignedPermissions(t *testing.T) {
	ctx := context.Background()
	api, server := newOrganizationRoleMockAPI(t)

	const orgID = "org_00000001"
	api.seedRole(orgID, &client.OrganizationRole{
		ID:          "role_00000001",
		Object:      "role",
		Slug:        "dealer-editor",
		Name:        "Dealer Editor",
		Description: "Edits dealers",
		Type:        "OrganizationRole",
		Permissions: []string{"dealer:read", "dealer:write"},
	})

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	r := &OrganizationRoleResource{client: workosClient}

	s := organizationRoleSchema(t)
	permissions := organizationRolePermissionsList("dealer:read", "dealer:write")

	priorState := tfsdk.State{
		Schema: s,
		Raw: organizationRoleObject(t, s, map[string]tftypes.Value{
			"id":                 tftypes.NewValue(tftypes.String, "role_00000001"),
			"organization_id":    tftypes.NewValue(tftypes.String, orgID),
			"slug":               tftypes.NewValue(tftypes.String, "dealer-editor"),
			"name":               tftypes.NewValue(tftypes.String, "Dealer Editor"),
			"description":        tftypes.NewValue(tftypes.String, "Edits dealers"),
			"type":               tftypes.NewValue(tftypes.String, "OrganizationRole"),
			"resource_type_slug": tftypes.NewValue(tftypes.String, nil),
			"permissions":        permissions,
			"created_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
			"updated_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
		}),
	}

	// The plan Terraform produces for a description change: permissions keeps
	// the prior state value via UseStateForUnknown, updated_at is unknown.
	plan := tfsdk.Plan{
		Schema: s,
		Raw: organizationRoleObject(t, s, map[string]tftypes.Value{
			"id":                 tftypes.NewValue(tftypes.String, "role_00000001"),
			"organization_id":    tftypes.NewValue(tftypes.String, orgID),
			"slug":               tftypes.NewValue(tftypes.String, "dealer-editor"),
			"name":               tftypes.NewValue(tftypes.String, "Dealer Editor"),
			"description":        tftypes.NewValue(tftypes.String, "Edits dealers and their staff"),
			"type":               tftypes.NewValue(tftypes.String, "OrganizationRole"),
			"resource_type_slug": tftypes.NewValue(tftypes.String, nil),
			"permissions":        permissions,
			"created_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
		}),
	}

	resp := &fwresource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: priorState.Raw}}
	r.Update(ctx, fwresource.UpdateRequest{Plan: plan, State: priorState}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update returned errors: %v", resp.Diagnostics)
	}

	var state OrganizationRoleResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}

	if state.Description.ValueString() != "Edits dealers and their staff" {
		t.Fatalf("expected the description change to be applied, got %v", state.Description)
	}

	var applied []string
	if diags := state.Permissions.ElementsAs(ctx, &applied, false); diags.HasError() {
		t.Fatalf("failed to read permissions: %v", diags)
	}
	if len(applied) != 2 || applied[0] != "dealer:read" || applied[1] != "dealer:write" {
		t.Fatalf("expected permissions to survive the update, got %v", applied)
	}
}

// TestOrganizationRoleResourceUpdate_KeepsEmptyPermissions covers the role with
// no permissions assigned: the applied value must stay an empty list, never
// null, so the value still matches the planned one.
func TestOrganizationRoleResourceUpdate_KeepsEmptyPermissions(t *testing.T) {
	ctx := context.Background()
	api, server := newOrganizationRoleMockAPI(t)

	const orgID = "org_00000001"
	api.seedRole(orgID, &client.OrganizationRole{
		ID:          "role_00000001",
		Object:      "role",
		Slug:        "dealer-viewer",
		Name:        "Dealer Viewer",
		Type:        "OrganizationRole",
		Permissions: []string{},
	})

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	r := &OrganizationRoleResource{client: workosClient}

	s := organizationRoleSchema(t)
	empty := organizationRolePermissionsList()

	priorState := tfsdk.State{
		Schema: s,
		Raw: organizationRoleObject(t, s, map[string]tftypes.Value{
			"id":                 tftypes.NewValue(tftypes.String, "role_00000001"),
			"organization_id":    tftypes.NewValue(tftypes.String, orgID),
			"slug":               tftypes.NewValue(tftypes.String, "dealer-viewer"),
			"name":               tftypes.NewValue(tftypes.String, "Dealer Viewer"),
			"description":        tftypes.NewValue(tftypes.String, ""),
			"type":               tftypes.NewValue(tftypes.String, "OrganizationRole"),
			"resource_type_slug": tftypes.NewValue(tftypes.String, nil),
			"permissions":        empty,
			"created_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
			"updated_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
		}),
	}

	plan := tfsdk.Plan{
		Schema: s,
		Raw: organizationRoleObject(t, s, map[string]tftypes.Value{
			"id":                 tftypes.NewValue(tftypes.String, "role_00000001"),
			"organization_id":    tftypes.NewValue(tftypes.String, orgID),
			"slug":               tftypes.NewValue(tftypes.String, "dealer-viewer"),
			"name":               tftypes.NewValue(tftypes.String, "Dealer Viewer Renamed"),
			"description":        tftypes.NewValue(tftypes.String, ""),
			"type":               tftypes.NewValue(tftypes.String, "OrganizationRole"),
			"resource_type_slug": tftypes.NewValue(tftypes.String, nil),
			"permissions":        empty,
			"created_at":         tftypes.NewValue(tftypes.String, "2026-01-01T00:00:00Z"),
		}),
	}

	resp := &fwresource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: priorState.Raw}}
	r.Update(ctx, fwresource.UpdateRequest{Plan: plan, State: priorState}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update returned errors: %v", resp.Diagnostics)
	}

	var state OrganizationRoleResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}

	if state.Permissions.IsNull() || state.Permissions.IsUnknown() {
		t.Fatalf("expected permissions to be a known empty list, got %v", state.Permissions)
	}
	if len(state.Permissions.Elements()) != 0 {
		t.Fatalf("expected permissions to stay empty, got %v", state.Permissions)
	}
}

// TestAccOrganizationRoleResource_UpdateWithPermissionsMockAPI drives the whole
// scenario from issue #37 through Terraform against the mock WorkOS API, so
// Terraform's own post-apply consistency check is what proves the fix.
// Requires the Terraform CLI.
func TestAccOrganizationRoleResource_UpdateWithPermissionsMockAPI(t *testing.T) {
	_, server := newOrganizationRoleMockAPI(t)

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the role and assign two permissions to it.
			{
				Config: testAccOrganizationRoleWithPermissionsMockConfig(server.URL, "Edits dealers"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization_role.dealer_editor", "slug", "dealer-editor"),
					resource.TestCheckResourceAttr("workos_organization_role.dealer_editor", "description", "Edits dealers"),
				),
			},
			// Re-apply unchanged so the refreshed state carries the assigned
			// permissions, which is the state the update then has to preserve.
			{
				Config: testAccOrganizationRoleWithPermissionsMockConfig(server.URL, "Edits dealers"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization_role.dealer_editor", "permissions.#", "2"),
				),
			},
			// Change the description. Before the fix this failed with
			// "Provider produced inconsistent result after apply ...
			// .permissions: element 0 has vanished".
			{
				Config: testAccOrganizationRoleWithPermissionsMockConfig(server.URL, "Edits dealers and their staff"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization_role.dealer_editor", "description", "Edits dealers and their staff"),
					resource.TestCheckResourceAttr("workos_organization_role.dealer_editor", "permissions.#", "2"),
					resource.TestCheckResourceAttr("workos_organization_role.dealer_editor", "permissions.0", "dealer:read"),
					resource.TestCheckResourceAttr("workos_organization_role.dealer_editor", "permissions.1", "dealer:write"),
				),
			},
		},
	})
}

func testAccOrganizationRoleWithPermissionsMockConfig(baseURL, description string) string {
	return fmt.Sprintf(`
provider "workos" {
  api_key  = "sk_test"
  base_url = %[1]q
}

resource "workos_organization" "test" {
  name = "Example Dealer Group"
}

resource "workos_organization_role" "dealer_editor" {
  organization_id    = workos_organization.test.id
  slug               = "dealer-editor"
  name               = "Dealer Editor"
  description        = %[2]q
  resource_type_slug = "dealer"
}

resource "workos_organization_role_permission" "read" {
  organization_id = workos_organization.test.id
  role_slug       = workos_organization_role.dealer_editor.slug
  permission      = "dealer:read"
}

resource "workos_organization_role_permission" "write" {
  organization_id = workos_organization.test.id
  role_slug       = workos_organization_role.dealer_editor.slug
  permission      = "dealer:write"

  depends_on = [workos_organization_role_permission.read]
}
`, baseURL, description)
}
