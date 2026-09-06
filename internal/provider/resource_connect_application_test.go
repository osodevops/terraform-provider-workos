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

// connectApplicationMockAPI is a minimal in-memory stand-in for the WorkOS API
// covering the endpoints the Connect application resource (and the organization
// resource it commonly depends on) exercises. It lets the provider be driven
// end to end without a live WorkOS environment.
type connectApplicationMockAPI struct {
	mu            sync.Mutex
	counter       int
	applications  map[string]*client.ConnectApplication
	organizations map[string]*client.Organization
	secrets       map[string]*client.ConnectApplicationSecret
	secretApps    map[string]string

	// omitOptionalFields makes the mock leave optional attributes out of its
	// responses, as the WorkOS API does for fields that do not apply to a given
	// application type. The provider must keep the configured values rather than
	// nulling them out.
	omitOptionalFields bool
}

func newConnectApplicationMockAPI(t *testing.T) (*connectApplicationMockAPI, *httptest.Server) {
	t.Helper()

	api := &connectApplicationMockAPI{
		applications:  make(map[string]*client.ConnectApplication),
		organizations: make(map[string]*client.Organization),
		secrets:       make(map[string]*client.ConnectApplicationSecret),
		secretApps:    make(map[string]string),
	}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	return api, server
}

func (a *connectApplicationMockAPI) nextID(prefix string) string {
	a.counter++
	return fmt.Sprintf("%s_%08d", prefix, a.counter)
}

func (a *connectApplicationMockAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch {
	case r.URL.Path == "/organizations" && r.Method == http.MethodPost:
		a.createOrganization(w, r)
	case strings.HasPrefix(r.URL.Path, "/organizations/"):
		a.organizationByID(w, r, strings.TrimPrefix(r.URL.Path, "/organizations/"))
	case r.URL.Path == "/connect/applications" && r.Method == http.MethodPost:
		a.createApplication(w, r)
	case strings.HasPrefix(r.URL.Path, "/connect/client_secrets/"):
		a.clientSecretByID(w, r, strings.TrimPrefix(r.URL.Path, "/connect/client_secrets/"))
	case strings.HasPrefix(r.URL.Path, "/connect/applications/") && strings.HasSuffix(r.URL.Path, "/client_secrets"):
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/connect/applications/"), "/client_secrets")
		a.applicationClientSecrets(w, r, id)
	case strings.HasPrefix(r.URL.Path, "/connect/applications/"):
		a.applicationByID(w, r, strings.TrimPrefix(r.URL.Path, "/connect/applications/"))
	default:
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}
}

func (a *connectApplicationMockAPI) createOrganization(w http.ResponseWriter, r *http.Request) {
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

func (a *connectApplicationMockAPI) organizationByID(w http.ResponseWriter, r *http.Request, id string) {
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

func (a *connectApplicationMockAPI) createApplication(w http.ResponseWriter, r *http.Request) {
	var req client.ConnectApplicationCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	app := &client.ConnectApplication{
		ID:              a.nextID("app"),
		Object:          "connect_application",
		ClientID:        a.nextID("client"),
		Name:            req.Name,
		ApplicationType: stringPtrOrNil(req.ApplicationType),
		OrganizationID:  stringPtrOrNil(req.OrganizationID),
		Scopes:          req.Scopes,
		UsesPKCE:        req.UsesPKCE,
		IsFirstParty:    req.IsFirstParty,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if req.Description != "" {
		description := req.Description
		app.Description = &description
	}
	app.RedirectURIs = redirectURIsFromInputs(req.RedirectURIs)

	if a.omitOptionalFields {
		app.ApplicationType = nil
		app.OrganizationID = nil
		app.Description = nil
		app.IsFirstParty = nil
		app.UsesPKCE = nil
	}

	a.applications[app.ID] = app
	writeJSON(w, http.StatusCreated, app)
}

func (a *connectApplicationMockAPI) applicationByID(w http.ResponseWriter, r *http.Request, id string) {
	app, ok := a.applications[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, app)
	case http.MethodPut:
		var req client.ConnectApplicationUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		if req.Name != "" {
			app.Name = req.Name
		}
		if req.Description != "" {
			description := req.Description
			app.Description = &description
		}
		app.Scopes = req.Scopes
		app.RedirectURIs = redirectURIsFromInputs(req.RedirectURIs)
		app.UpdatedAt = time.Now().UTC().Truncate(time.Second)
		writeJSON(w, http.StatusOK, app)
	case http.MethodDelete:
		delete(a.applications, id)
		for secretID, applicationID := range a.secretApps {
			if applicationID == id {
				delete(a.secrets, secretID)
				delete(a.secretApps, secretID)
			}
		}
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, `{"message":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (a *connectApplicationMockAPI) findApplication(id string) (*client.ConnectApplication, bool) {
	if app, ok := a.applications[id]; ok {
		return app, true
	}
	for _, app := range a.applications {
		if app.ClientID == id {
			return app, true
		}
	}
	return nil, false
}

func (a *connectApplicationMockAPI) applicationClientSecrets(w http.ResponseWriter, r *http.Request, applicationID string) {
	app, ok := a.findApplication(applicationID)
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodPost:
		now := time.Now().UTC().Truncate(time.Second)
		id := a.nextID("secret")
		plaintext := "plaintext-" + id
		hint := plaintext
		if len(hint) > 6 {
			hint = hint[:6]
		}
		secret := &client.ConnectApplicationSecret{
			ID:         id,
			Object:     "connect_application_secret",
			SecretHint: hint,
			Secret:     plaintext,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		stored := *secret
		stored.Secret = ""
		a.secrets[id] = &stored
		a.secretApps[id] = app.ID
		writeJSON(w, http.StatusCreated, secret)
	case http.MethodGet:
		listed := make([]client.ConnectApplicationSecret, 0)
		for secretID, ownerID := range a.secretApps {
			if ownerID == app.ID {
				listed = append(listed, *a.secrets[secretID])
			}
		}
		writeJSON(w, http.StatusOK, listed)
	default:
		http.Error(w, `{"message":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (a *connectApplicationMockAPI) clientSecretByID(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := a.secrets[id]; !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		delete(a.secrets, id)
		delete(a.secretApps, id)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, `{"message":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func redirectURIsFromInputs(inputs []client.ConnectApplicationRedirectURIInput) []client.ConnectApplicationRedirectURI {
	if len(inputs) == 0 {
		return nil
	}

	uris := make([]client.ConnectApplicationRedirectURI, 0, len(inputs))
	for _, input := range inputs {
		uri := client.ConnectApplicationRedirectURI{URI: input.URI}
		if input.Default != nil {
			uri.Default = *input.Default
		}
		uris = append(uris, uri)
	}
	return uris
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// connectApplicationSchema returns the resource schema under test.
func connectApplicationSchema(t *testing.T) schema.Schema {
	t.Helper()

	resp := &fwresource.SchemaResponse{}
	(&ConnectApplicationResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// connectApplicationPlan builds the plan Terraform hands to Create: every
// attribute the practitioner did not configure is unknown, exactly as it is for
// an Optional+Computed attribute with no prior state.
func connectApplicationPlan(t *testing.T, s schema.Schema, known map[string]tftypes.Value) tfsdk.Plan {
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

	return tfsdk.Plan{
		Schema: s,
		Raw:    tftypes.NewValue(objectType, values),
	}
}

func connectApplicationNullState(t *testing.T, s schema.Schema) tfsdk.State {
	t.Helper()

	return tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(s.Type().TerraformType(context.Background()), nil),
	}
}

// TestConnectApplicationResourceCreate_M2MUnknownComputedAttributes reproduces
// https://github.com/osodevops/terraform-provider-workos/issues/34: creating an
// m2m Connect application without redirect_uris/scopes failed with a
// "Value Conversion Error" before any API request was made, because the plan
// carries unknown values for every unconfigured Optional+Computed attribute.
func TestConnectApplicationResourceCreate_M2MUnknownComputedAttributes(t *testing.T) {
	ctx := context.Background()
	api, server := newConnectApplicationMockAPI(t)

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	r := &ConnectApplicationResource{client: workosClient}

	s := connectApplicationSchema(t)
	plan := connectApplicationPlan(t, s, map[string]tftypes.Value{
		"name":             tftypes.NewValue(tftypes.String, "example-widget"),
		"application_type": tftypes.NewValue(tftypes.String, "m2m"),
		"organization_id":  tftypes.NewValue(tftypes.String, "org_00000001"),
	})

	resp := &fwresource.CreateResponse{State: connectApplicationNullState(t, s)}
	r.Create(ctx, fwresource.CreateRequest{Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create returned errors: %v", resp.Diagnostics)
	}

	if len(api.applications) != 1 {
		t.Fatalf("expected 1 application to be created, got %d", len(api.applications))
	}

	var state ConnectApplicationResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		t.Fatalf("expected id to be set in state, got %v", state.ID)
	}
	if state.ClientID.IsNull() || state.ClientID.IsUnknown() {
		t.Fatalf("expected client_id to be set in state, got %v", state.ClientID)
	}
	if state.OrganizationID.ValueString() != "org_00000001" {
		t.Fatalf("expected organization_id org_00000001, got %q", state.OrganizationID.ValueString())
	}
	if state.RedirectURIs.IsUnknown() {
		t.Fatal("expected redirect_uris to be known after create")
	}
	if state.Scopes.IsUnknown() {
		t.Fatal("expected scopes to be known after create")
	}
}

// TestConnectApplicationResourceCreate_OAuthWithRedirectURIs covers the
// configured-list path so the unknown-value fix does not regress the case where
// redirect_uris and scopes are supplied.
func TestConnectApplicationResourceCreate_OAuthWithRedirectURIs(t *testing.T) {
	ctx := context.Background()
	api, server := newConnectApplicationMockAPI(t)

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	r := &ConnectApplicationResource{client: workosClient}

	s := connectApplicationSchema(t)
	redirectURIType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"uri":     tftypes.String,
		"default": tftypes.Bool,
	}}
	plan := connectApplicationPlan(t, s, map[string]tftypes.Value{
		"name":             tftypes.NewValue(tftypes.String, "example-oauth"),
		"application_type": tftypes.NewValue(tftypes.String, "oauth"),
		"scopes": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "openid"),
			tftypes.NewValue(tftypes.String, "profile"),
		}),
		"redirect_uris": tftypes.NewValue(tftypes.List{ElementType: redirectURIType}, []tftypes.Value{
			tftypes.NewValue(redirectURIType, map[string]tftypes.Value{
				"uri":     tftypes.NewValue(tftypes.String, "https://example.com/callback"),
				"default": tftypes.NewValue(tftypes.Bool, true),
			}),
		}),
	})

	resp := &fwresource.CreateResponse{State: connectApplicationNullState(t, s)}
	r.Create(ctx, fwresource.CreateRequest{Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create returned errors: %v", resp.Diagnostics)
	}

	var created *client.ConnectApplication
	for _, app := range api.applications {
		created = app
	}
	if created == nil {
		t.Fatal("expected an application to be created")
	}
	if len(created.RedirectURIs) != 1 || created.RedirectURIs[0].URI != "https://example.com/callback" || !created.RedirectURIs[0].Default {
		t.Fatalf("unexpected redirect URIs sent to API: %#v", created.RedirectURIs)
	}
	if len(created.Scopes) != 2 {
		t.Fatalf("unexpected scopes sent to API: %#v", created.Scopes)
	}

	var state ConnectApplicationResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}
	if got := len(state.RedirectURIs.Elements()); got != 1 {
		t.Fatalf("expected 1 redirect URI in state, got %d", got)
	}
}

// TestConnectApplicationResourceCreate_M2MRejectsRedirectURIs keeps the
// validation behaviour for m2m applications intact.
func TestConnectApplicationResourceCreate_M2MRejectsRedirectURIs(t *testing.T) {
	ctx := context.Background()
	_, server := newConnectApplicationMockAPI(t)

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	r := &ConnectApplicationResource{client: workosClient}

	s := connectApplicationSchema(t)
	redirectURIType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"uri":     tftypes.String,
		"default": tftypes.Bool,
	}}
	plan := connectApplicationPlan(t, s, map[string]tftypes.Value{
		"name":             tftypes.NewValue(tftypes.String, "example-widget"),
		"application_type": tftypes.NewValue(tftypes.String, "m2m"),
		"organization_id":  tftypes.NewValue(tftypes.String, "org_00000001"),
		"redirect_uris": tftypes.NewValue(tftypes.List{ElementType: redirectURIType}, []tftypes.Value{
			tftypes.NewValue(redirectURIType, map[string]tftypes.Value{
				"uri":     tftypes.NewValue(tftypes.String, "https://example.com/callback"),
				"default": tftypes.NewValue(tftypes.Bool, true),
			}),
		}),
	})

	resp := &fwresource.CreateResponse{State: connectApplicationNullState(t, s)}
	r.Create(ctx, fwresource.CreateRequest{Plan: plan}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when redirect_uris are set on an m2m application")
	}
}

// TestConnectApplicationResourceCreate_SparseAPIResponse asserts that
// attributes the practitioner configured survive an API response that omits
// them. Nulling them out would make Terraform reject the apply with
// "Provider produced inconsistent result after apply".
func TestConnectApplicationResourceCreate_SparseAPIResponse(t *testing.T) {
	ctx := context.Background()
	api, server := newConnectApplicationMockAPI(t)
	api.omitOptionalFields = true

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	r := &ConnectApplicationResource{client: workosClient}

	s := connectApplicationSchema(t)
	plan := connectApplicationPlan(t, s, map[string]tftypes.Value{
		"name":             tftypes.NewValue(tftypes.String, "example-widget"),
		"application_type": tftypes.NewValue(tftypes.String, "m2m"),
		"organization_id":  tftypes.NewValue(tftypes.String, "org_00000001"),
		"description":      tftypes.NewValue(tftypes.String, "a widget"),
	})

	resp := &fwresource.CreateResponse{State: connectApplicationNullState(t, s)}
	r.Create(ctx, fwresource.CreateRequest{Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create returned errors: %v", resp.Diagnostics)
	}

	var state ConnectApplicationResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}

	if state.ApplicationType.ValueString() != "m2m" {
		t.Fatalf("expected application_type to be preserved as m2m, got %v", state.ApplicationType)
	}
	if state.OrganizationID.ValueString() != "org_00000001" {
		t.Fatalf("expected organization_id to be preserved, got %v", state.OrganizationID)
	}
	if state.Description.ValueString() != "a widget" {
		t.Fatalf("expected description to be preserved, got %v", state.Description)
	}
	// Unconfigured optional attributes have no value worth keeping, so an
	// unknown must collapse to null rather than stay unknown.
	if state.UsesPKCE.IsUnknown() || !state.UsesPKCE.IsNull() {
		t.Fatalf("expected uses_pkce to be null, got %v", state.UsesPKCE)
	}
}

// TestAccConnectApplicationResource_M2MSparseAPIResponse runs the same scenario
// through Terraform, which independently enforces that the applied state
// matches the plan.
func TestAccConnectApplicationResource_M2MSparseAPIResponse(t *testing.T) {
	api, server := newConnectApplicationMockAPI(t)
	api.omitOptionalFields = true

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectApplicationMockConfig(server.URL, "example-widget"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_connect_application.m2m", "application_type", "m2m"),
					resource.TestCheckResourceAttrPair(
						"workos_connect_application.m2m", "organization_id",
						"workos_organization.public", "id",
					),
				),
			},
		},
	})
}

// TestAccConnectApplicationResource_M2MMockAPI drives the full Terraform
// lifecycle for the configuration reported in issue #34 against the mock WorkOS
// API, so it runs without live credentials. Requires the Terraform CLI.
func TestAccConnectApplicationResource_M2MMockAPI(t *testing.T) {
	_, server := newConnectApplicationMockAPI(t)

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectApplicationMockConfig(server.URL, "example-widget"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_connect_application.m2m", "name", "example-widget"),
					resource.TestCheckResourceAttr("workos_connect_application.m2m", "application_type", "m2m"),
					resource.TestCheckResourceAttrSet("workos_connect_application.m2m", "id"),
					resource.TestCheckResourceAttrSet("workos_connect_application.m2m", "client_id"),
					resource.TestCheckResourceAttrPair(
						"workos_connect_application.m2m", "organization_id",
						"workos_organization.public", "id",
					),
					resource.TestCheckResourceAttr("workos_connect_application.m2m", "redirect_uris.#", "0"),
					resource.TestCheckResourceAttr("workos_connect_application.m2m", "scopes.#", "0"),
				),
			},
			{
				ResourceName:      "workos_connect_application.m2m",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccConnectApplicationMockConfig(server.URL, "example-widget-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_connect_application.m2m", "name", "example-widget-renamed"),
				),
			},
		},
	})
}

// TestAccConnectApplicationResource_OAuthMockAPI covers an OAuth application
// with redirect URIs end to end against the mock WorkOS API.
func TestAccConnectApplicationResource_OAuthMockAPI(t *testing.T) {
	_, server := newConnectApplicationMockAPI(t)

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectApplicationOAuthMockConfig(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_connect_application.oauth", "application_type", "oauth"),
					resource.TestCheckResourceAttr("workos_connect_application.oauth", "redirect_uris.#", "1"),
					resource.TestCheckResourceAttr("workos_connect_application.oauth", "redirect_uris.0.uri", "https://example.com/callback"),
					resource.TestCheckResourceAttr("workos_connect_application.oauth", "redirect_uris.0.default", "true"),
					resource.TestCheckResourceAttr("workos_connect_application.oauth", "scopes.#", "2"),
				),
			},
		},
	})
}

func testAccConnectApplicationMockConfig(baseURL, name string) string {
	return fmt.Sprintf(`
provider "workos" {
  api_key  = "sk_test"
  base_url = %[1]q
}

resource "workos_organization" "public" {
  name = "Example Public"
}

resource "workos_connect_application" "m2m" {
  name             = %[2]q
  application_type = "m2m"
  organization_id  = workos_organization.public.id
}
`, baseURL, name)
}

func testAccConnectApplicationOAuthMockConfig(baseURL string) string {
	return fmt.Sprintf(`
provider "workos" {
  api_key  = "sk_test"
  base_url = %[1]q
}

resource "workos_connect_application" "oauth" {
  name             = "example-oauth"
  application_type = "oauth"
  scopes           = ["openid", "profile"]

  redirect_uris = [
    {
      uri     = "https://example.com/callback"
      default = true
    },
  ]
}
`, baseURL)
}
