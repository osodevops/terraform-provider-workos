// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// redirectURIMockAPI is a minimal in-memory stand-in for the User Management
// redirect URI endpoints. It lets the provider be driven end to end without a
// live WorkOS environment.
type redirectURIMockAPI struct {
	mu      sync.Mutex
	counter int
	uris    map[string]*client.RedirectURI
}

func newRedirectURIMockAPI(t *testing.T) (*redirectURIMockAPI, *httptest.Server) {
	t.Helper()

	api := &redirectURIMockAPI{
		uris: make(map[string]*client.RedirectURI),
	}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	return api, server
}

func (a *redirectURIMockAPI) nextID() string {
	a.counter++
	return fmt.Sprintf("redir_%08d", a.counter)
}

func (a *redirectURIMockAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch {
	case r.URL.Path == "/user_management/redirect_uris" && r.Method == http.MethodPost:
		a.create(w, r)
	case r.URL.Path == "/user_management/redirect_uris" && r.Method == http.MethodGet:
		a.list(w)
	case strings.HasPrefix(r.URL.Path, "/user_management/redirect_uris/") && r.Method == http.MethodDelete:
		a.delete(w, strings.TrimPrefix(r.URL.Path, "/user_management/redirect_uris/"))
	default:
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}
}

func (a *redirectURIMockAPI) create(w http.ResponseWriter, r *http.Request) {
	var req client.RedirectURICreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"invalid json"}`, http.StatusBadRequest)
		return
	}

	for _, existing := range a.uris {
		if existing.URI == req.URI {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"message":"Redirect URI already exists","code":"entity_already_exists"}`))
			return
		}
	}

	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	item := &client.RedirectURI{
		ID:        a.nextID(),
		Object:    "redirect_uri",
		URI:       req.URI,
		Default:   len(a.uris) == 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	a.uris[item.ID] = item

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(item)
}

func (a *redirectURIMockAPI) list(w http.ResponseWriter) {
	data := make([]client.RedirectURI, 0, len(a.uris))
	for _, item := range a.uris {
		data = append(data, *item)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(client.RedirectURIListResponse{
		Data: data,
	})
}

func (a *redirectURIMockAPI) delete(w http.ResponseWriter, id string) {
	if _, ok := a.uris[id]; !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	delete(a.uris, id)
	w.WriteHeader(http.StatusOK)
}

func redirectURISchema(t *testing.T) schema.Schema {
	t.Helper()

	resp := &fwresource.SchemaResponse{}
	(&RedirectURIResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func redirectURIPlan(t *testing.T, s schema.Schema, known map[string]tftypes.Value) tfsdk.Plan {
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

func redirectURINullState(t *testing.T, s schema.Schema) tfsdk.State {
	t.Helper()

	return tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(s.Type().TerraformType(context.Background()), nil),
	}
}

func TestRedirectURIResourceCreate_RefusesToAdoptExisting(t *testing.T) {
	ctx := context.Background()
	api, server := newRedirectURIMockAPI(t)

	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	existing := &client.RedirectURI{
		ID:        "redir_existing",
		Object:    "redirect_uri",
		URI:       "https://acme.example.com/api/auth/callback",
		Default:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	api.uris[existing.ID] = existing

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	r := &RedirectURIResource{client: workosClient}

	s := redirectURISchema(t)
	plan := redirectURIPlan(t, s, map[string]tftypes.Value{
		"uri": tftypes.NewValue(tftypes.String, existing.URI),
	})

	resp := &fwresource.CreateResponse{State: redirectURINullState(t, s)}
	r.Create(ctx, fwresource.CreateRequest{Plan: plan}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Create to fail when the redirect URI already exists")
	}

	var detail string
	for _, d := range resp.Diagnostics.Errors() {
		detail += d.Summary() + " " + d.Detail()
	}
	if !strings.Contains(detail, "already registered") {
		t.Fatalf("expected an already-registered error, got: %s", detail)
	}
	if !strings.Contains(detail, "terraform import") {
		t.Fatalf("expected the error to point at terraform import, got: %s", detail)
	}
	if len(api.uris) != 1 {
		t.Fatalf("expected the pre-existing URI to be left alone, have %d records", len(api.uris))
	}
}

func TestAccRedirectURIResource_MockAPI(t *testing.T) {
	_, server := newRedirectURIMockAPI(t)

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedirectURIMockConfig(server.URL, "https://acme.example.com/api/auth/callback"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_redirect_uri.tenant_callback", "uri", "https://acme.example.com/api/auth/callback"),
					resource.TestCheckResourceAttrSet("workos_redirect_uri.tenant_callback", "id"),
					resource.TestCheckResourceAttrSet("workos_redirect_uri.tenant_callback", "created_at"),
					resource.TestCheckResourceAttrSet("workos_redirect_uri.tenant_callback", "updated_at"),
				),
			},
			{
				ResourceName:      "workos_redirect_uri.tenant_callback",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "workos_redirect_uri.tenant_callback",
				ImportState:       true,
				ImportStateId:     "https://acme.example.com/api/auth/callback",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRedirectURIResource_ImportsExisting(t *testing.T) {
	api, server := newRedirectURIMockAPI(t)

	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	existing := &client.RedirectURI{
		ID:        "redir_existing",
		Object:    "redirect_uri",
		URI:       "https://acme.example.com/api/auth/callback",
		Default:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	api.uris[existing.ID] = existing

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Creating a URI that already exists must fail and point at import.
				Config:      testAccRedirectURIMockConfig(server.URL, existing.URI),
				ExpectError: regexp.MustCompile(`(?s)already\s+registered.*terraform\s+import`),
			},
			{
				// The documented recovery: import the existing URI by its URI.
				Config:        testAccRedirectURIMockConfig(server.URL, existing.URI),
				ResourceName:  "workos_redirect_uri.tenant_callback",
				ImportState:   true,
				ImportStateId: existing.URI,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported instance, got %d", len(states))
					}
					if got := states[0].ID; got != existing.ID {
						return fmt.Errorf("expected imported id %q, got %q", existing.ID, got)
					}
					if got := states[0].Attributes["uri"]; got != existing.URI {
						return fmt.Errorf("expected imported uri %q, got %q", existing.URI, got)
					}
					return nil
				},
			},
		},
	})
}

func testAccRedirectURIMockConfig(baseURL, uri string) string {
	return fmt.Sprintf(`
provider "workos" {
  api_key  = "sk_test"
  base_url = %[1]q
}

resource "workos_redirect_uri" "tenant_callback" {
  uri = %[2]q
}
`, baseURL, uri)
}
