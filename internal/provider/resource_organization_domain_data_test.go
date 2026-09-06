// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

// organizationMockAPI is a minimal in-memory stand-in for the Organizations
// API. It exists so domain_data can be driven through a full Terraform
// lifecycle in CI, where no WorkOS credentials are available.
type organizationMockAPI struct {
	mu      sync.Mutex
	counter int
	orgs    map[string]*client.Organization
}

func newOrganizationMockAPI(t *testing.T) (*organizationMockAPI, *httptest.Server) {
	t.Helper()

	api := &organizationMockAPI{orgs: make(map[string]*client.Organization)}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	return api, server
}

func (a *organizationMockAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	id := strings.TrimPrefix(r.URL.Path, "/organizations")
	id = strings.TrimPrefix(id, "/")

	switch {
	case r.Method == http.MethodPost && id == "":
		a.create(w, r)
	case r.Method == http.MethodGet && id != "":
		a.get(w, id)
	case r.Method == http.MethodPut && id != "":
		a.update(w, r, id)
	case r.Method == http.MethodDelete && id != "":
		delete(a.orgs, id)
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}
}

// domainsFrom mirrors how WorkOS echoes domain_data back as full domain
// objects, assigning each an id and defaulting the state to pending.
func (a *organizationMockAPI) domainsFrom(orgID string, data []client.DomainData) []client.Domain {
	domains := make([]client.Domain, 0, len(data))
	for i, d := range data {
		state := d.State
		if state == "" {
			state = "pending"
		}
		domains = append(domains, client.Domain{
			ID:             fmt.Sprintf("org_domain_%s_%d", orgID, i),
			Object:         "organization_domain",
			Domain:         d.Domain,
			State:          state,
			OrganizationID: orgID,
		})
	}
	return domains
}

func (a *organizationMockAPI) create(w http.ResponseWriter, r *http.Request) {
	var req client.OrganizationCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"invalid json"}`, http.StatusBadRequest)
		return
	}

	a.counter++
	id := fmt.Sprintf("org_%08d", a.counter)
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	org := &client.Organization{
		ID:         id,
		Object:     "organization",
		Name:       req.Name,
		ExternalID: req.ExternalID,
		Domains:    a.domainsFrom(id, req.DomainData),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	a.orgs[id] = org

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(org)
}

func (a *organizationMockAPI) get(w http.ResponseWriter, id string) {
	org, ok := a.orgs[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(org)
}

func (a *organizationMockAPI) update(w http.ResponseWriter, r *http.Request, id string) {
	org, ok := a.orgs[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	var req client.OrganizationUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.Name != "" {
		org.Name = req.Name
	}
	if req.ExternalID != "" {
		org.ExternalID = req.ExternalID
	}
	if len(req.DomainData) > 0 {
		org.Domains = a.domainsFrom(id, req.DomainData)
	}
	org.UpdatedAt = org.UpdatedAt.Add(time.Second)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(org)
}

func testAccOrganizationMockConfig(baseURL, name string, domainData string) string {
	return fmt.Sprintf(`
provider "workos" {
  api_key  = "sk_test"
  base_url = %[1]q
}

resource "workos_organization" "test" {
  name = %[2]q

  domain_data = %[3]s
}
`, baseURL, name, domainData)
}

// TestAccOrganizationResource_DomainDataMockAPI drives domain_data through
// create, refresh and update against the mock API, which is the coverage the
// credential-gated acceptance tests cannot give in CI.
func TestAccOrganizationResource_DomainDataMockAPI(t *testing.T) {
	_, server := newOrganizationMockAPI(t)

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// state is omitted, so the schema default must fill in pending.
				Config: testAccOrganizationMockConfig(server.URL, "acme", `[{ domain = "acme.com" }]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization.test", "domain_data.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(
						"workos_organization.test",
						"domain_data.*",
						map[string]string{"domain": "acme.com", "state": "pending"},
					),
					resource.TestCheckNoResourceAttr("workos_organization.test", "domains.#"),
				),
			},
			{
				// Adding a second domain must update in place, not replace.
				Config: testAccOrganizationMockConfig(server.URL, "acme",
					`[{ domain = "acme.com" }, { domain = "acmecorp.com", state = "verified" }]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization.test", "domain_data.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(
						"workos_organization.test",
						"domain_data.*",
						map[string]string{"domain": "acmecorp.com", "state": "verified"},
					),
				),
			},
		},
	})
}

// TestAccOrganizationResource_LegacyDomainsStillWork guards the compatibility
// promise of deprecating domains: existing configurations must keep applying
// unchanged until the attribute is removed in a future major version.
func TestAccOrganizationResource_LegacyDomainsStillWork(t *testing.T) {
	_, server := newOrganizationMockAPI(t)

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	config := fmt.Sprintf(`
provider "workos" {
  api_key  = "sk_test"
  base_url = %[1]q
}

resource "workos_organization" "legacy" {
  name    = "legacy"
  domains = ["legacy.example.com"]
}
`, server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization.legacy", "domains.#", "1"),
					resource.TestCheckTypeSetElemAttr("workos_organization.legacy", "domains.*", "legacy.example.com"),
					resource.TestCheckNoResourceAttr("workos_organization.legacy", "domain_data.#"),
				),
			},
			{
				// A deprecation warning must not turn into a perpetual diff.
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}
