// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/osodevops/terraform-provider-workos/internal/client"
	"github.com/osodevops/terraform-provider-workos/internal/testutil"
)

func accountTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &fwresource.SchemaResponse{}
	(&AuthkitConfigurationResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.Schema
}

func accountTestValues() map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"environment_id":       tftypes.NewValue(tftypes.String, "environment_test"),
		"application_id":       tftypes.NewValue(tftypes.String, "application_test"),
		"redirect_uris":        environmentRolePermissionsValue("https://vigile.example/callback"),
		"default_redirect_uri": tftypes.NewValue(tftypes.String, "https://vigile.example/callback"),
		"logout_uris":          environmentRolePermissionsValue("https://vigile.example/signed-out"),
		"default_logout_uri":   tftypes.NewValue(tftypes.String, "https://vigile.example/signed-out"),
		"cors_origins":         environmentRolePermissionsValue("https://vigile.example"),
		"initiate_login_uri":   tftypes.NewValue(tftypes.String, "https://vigile.example/login"),
		"authkit_enabled":      tftypes.NewValue(tftypes.Bool, true),
		"allow_signup":         tftypes.NewValue(tftypes.Bool, true),
		"waitlist_enabled":     tftypes.NewValue(tftypes.Bool, false),
	}
}

func TestAuthkitConfigurationSchemaValidation(t *testing.T) {
	s := accountTestSchema(t)
	if attribute, ok := s.Attributes["authkit_enabled"].(schema.BoolAttribute); !ok || !attribute.Required {
		t.Fatal("authkit_enabled must be a required boolean")
	}
	for _, name := range []string{"redirect_uris", "logout_uris", "cors_origins"} {
		attribute, ok := s.Attributes[name].(schema.SetAttribute)
		if !ok || !attribute.Required {
			t.Fatalf("%s must be a required set", name)
		}
	}
	for _, test := range []struct {
		name      string
		change    func(map[string]tftypes.Value)
		wantError bool
	}{
		{"valid", func(map[string]tftypes.Value) {}, false},
		{"default outside set", func(v map[string]tftypes.Value) {
			v["default_redirect_uri"] = tftypes.NewValue(tftypes.String, "https://other.example/callback")
		}, true},
		{"origin with path", func(v map[string]tftypes.Value) {
			v["cors_origins"] = environmentRolePermissionsValue("https://vigile.example/path")
		}, true},
		{"unknown callback", func(v map[string]tftypes.Value) {
			v["default_redirect_uri"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			known := accountTestValues()
			test.change(known)
			config := tfsdk.Config{Schema: s, Raw: environmentRoleObject(t, s, known, nullFill)}
			resp := &fwresource.ValidateConfigResponse{}
			(&AuthkitConfigurationResource{}).ValidateConfig(context.Background(), fwresource.ValidateConfigRequest{Config: config}, resp)
			if resp.Diagnostics.HasError() != test.wantError {
				t.Fatalf("diagnostics=%v", resp.Diagnostics)
			}
		})
	}
}

func TestAuthkitConfigurationCreateKeepsPartialReadback(t *testing.T) {
	api := testutil.NewAccountAPI()
	api.FailField = "setLogoutUris"
	server := httptest.NewServer(api)
	defer server.Close()
	c, err := client.NewClientWithCredentials("sk_test", "client_test", server.URL, "account-test-token", "team_test")
	if err != nil {
		t.Fatal(err)
	}
	s := accountTestSchema(t)
	plan := tfsdk.Plan{Schema: s, Raw: environmentRoleObject(t, s, accountTestValues(), unknownFill)}
	resp := &fwresource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}}
	(&AuthkitConfigurationResource{client: c}).Create(context.Background(), fwresource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected partial failure")
	}
	var data AuthkitConfigurationResourceModel
	if diagnostics := resp.State.Get(context.Background(), &data); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if data.ID.ValueString() != "environment_test/application_test" || data.DefaultRedirectURI.ValueString() != "https://vigile.example/callback" || data.DefaultLogoutURI.ValueString() != "https://old.example/logout" || data.AuthkitEnabled.IsNull() || data.AuthkitEnabled.IsUnknown() || data.AuthkitEnabled.ValueBool() {
		t.Fatalf("partial state lost: %#v", data)
	}
}

func TestAccountProviderPreflightStopsAllResources(t *testing.T) {
	for _, test := range []struct {
		name      string
		change    func(*testutil.AccountAPI)
		wantError bool
	}{
		{"matching environment", func(*testutil.AccountAPI) {}, false},
		{"wrong REST key", func(api *testutil.AccountAPI) { api.RESTEnvironmentMismatch = true }, true},
		{"OAuth gate", func(api *testutil.AccountAPI) { api.HTTPStatus = 403 }, true},
		{"expired OAuth", func(api *testutil.AccountAPI) { api.HTTPStatus = 401 }, true},
		{"stale target", func(api *testutil.AccountAPI) { api.VisibleEnvironmentID = "stale" }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := testutil.NewAccountAPI()
			test.change(api)
			server := httptest.NewServer(api)
			defer server.Close()
			p := &WorkOSProvider{}
			schemaResp := &fwprovider.SchemaResponse{}
			p.Schema(context.Background(), fwprovider.SchemaRequest{}, schemaResp)
			s := schemaResp.Schema
			object := s.Type().TerraformType(context.Background()).(tftypes.Object)
			values := make(map[string]tftypes.Value)
			for name, kind := range object.AttributeTypes {
				values[name] = tftypes.NewValue(kind, nil)
			}
			for name, value := range map[string]string{"api_key": "sk_test", "client_id": "client_test", "base_url": server.URL, "account_access_token": "account-test-token", "account_team_id": "team_test", "environment_id": "environment_test", "application_id": "application_test"} {
				values[name] = tftypes.NewValue(tftypes.String, value)
			}
			resp := &fwprovider.ConfigureResponse{}
			p.Configure(context.Background(), fwprovider.ConfigureRequest{Config: tfsdk.Config{Schema: s, Raw: tftypes.NewValue(object, values)}}, resp)
			if resp.Diagnostics.HasError() != test.wantError {
				t.Fatalf("diagnostics=%v", resp.Diagnostics)
			}
			if test.wantError && (resp.ResourceData != nil || resp.DataSourceData != nil) {
				t.Fatal("unverified client must not reach resources")
			}
			if len(api.Writes) != 0 {
				t.Fatal("provider preflight must not mutate")
			}
		})
	}
}

func accountTerraformConfig(baseURL string) string {
	return fmt.Sprintf(`
provider "workos" {
  api_key = "sk_test"
  client_id = "client_test"
  base_url = %q
  account_access_token = "account-test-token"
  account_team_id = "team_test"
  environment_id = "environment_test"
  application_id = "application_test"
}
resource "workos_authkit_configuration" "test" {
  environment_id = "environment_test"
  application_id = "application_test"
  redirect_uris = ["https://vigile.example/callback"]
  default_redirect_uri = "https://vigile.example/callback"
  logout_uris = ["https://vigile.example/signed-out"]
  default_logout_uri = "https://vigile.example/signed-out"
  cors_origins = ["https://vigile.example"]
  initiate_login_uri = "https://vigile.example/login"
  authkit_enabled = true
  allow_signup = true
  waitlist_enabled = false
}
resource "workos_environment_role_configuration" "test" {
  environment_id = "environment_test"
  default_role_slug = "viewer"
}
`, baseURL)
}

func TestAccAccountConfiguration_MockAPI(t *testing.T) {
	api := testutil.NewAccountAPI()
	server := httptest.NewServer(api)
	defer server.Close()
	config := accountTerraformConfig(server.URL)
	checkWrites := func(want int) resource.TestCheckFunc {
		return func(*terraform.State) error {
			api.Mu.Lock()
			defer api.Mu.Unlock()
			if len(api.Writes) != want {
				return fmt.Errorf("got %d writes, want %d", len(api.Writes), want)
			}
			return nil
		}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			api.Mu.Lock()
			defer api.Mu.Unlock()
			if api.Config.DefaultRedirectURI != "https://vigile.example/callback" || !api.Config.AuthkitEnabled || api.DefaultRoleID != "role_viewer" || len(api.Writes) != 9 {
				return fmt.Errorf("destroy must only detach and preserve all remote settings")
			}
			return nil
		},
		Steps: []resource.TestStep{
			{Config: config, Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("workos_authkit_configuration.test", "id", "environment_test/application_test"),
				resource.TestCheckResourceAttr("workos_authkit_configuration.test", "authkit_enabled", "true"),
				resource.TestCheckResourceAttr("workos_authkit_configuration.test", "allow_signup", "true"),
				resource.TestCheckResourceAttr("workos_authkit_configuration.test", "waitlist_enabled", "false"),
				resource.TestCheckResourceAttr("workos_environment_role_configuration.test", "default_role_slug", "viewer"), checkWrites(6),
			)},
			{ResourceName: "workos_authkit_configuration.test", ImportState: true, ImportStateVerify: true},
			{ResourceName: "workos_environment_role_configuration.test", ImportState: true, ImportStateVerify: true},
			{Config: config, PlanOnly: true, Check: checkWrites(6)},
			{PreConfig: func() {
				api.Mu.Lock()
				defer api.Mu.Unlock()
				api.Config.InitiateLoginURI = "https://drift.example/login"
				api.Config.AuthkitEnabled = false
				api.DefaultRoleID = "role_admin"
			}, Config: config, Check: resource.ComposeAggregateTestCheckFunc(checkWrites(9), resource.TestCheckResourceAttr("workos_authkit_configuration.test", "authkit_enabled", "true"))},
			{Config: config, PlanOnly: true, Check: checkWrites(9)},
		},
	})
}

func TestAccAccountConfiguration_InvalidDefault(t *testing.T) {
	api := testutil.NewAccountAPI()
	server := httptest.NewServer(api)
	defer server.Close()
	config := strings.Replace(accountTerraformConfig(server.URL), `default_redirect_uri = "https://vigile.example/callback"`, `default_redirect_uri = "https://unregistered.example/callback"`, 1)
	missingEnabled := strings.Replace(accountTerraformConfig(server.URL), "  authkit_enabled = true\n", "", 1)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("default URI must be a member")},
		{Config: missingEnabled, ExpectError: regexp.MustCompile("Missing required argument")},
	}})
	if len(api.Writes) != 0 {
		t.Fatal("invalid default must be rejected without writes")
	}
}

func TestAccAccountConfiguration_PreflightPreventsParallelRESTWrites(t *testing.T) {
	api := testutil.NewAccountAPI()
	api.RESTEnvironmentMismatch = true
	server := httptest.NewServer(api)
	defer server.Close()
	config := accountTerraformConfig(server.URL) + `
resource "workos_environment_role" "new_role" {
  slug = "new-role"
  name = "New role"
  permissions = []
}
`
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile("REST API key role IDs do not match")}}})
	if len(api.Writes) != 0 || api.RESTMutationRequests != 0 {
		t.Fatal("failed provider preflight must prevent account and parallel REST mutations")
	}
}

func TestAccAccountConfiguration_PartialCreateRetry(t *testing.T) {
	api := testutil.NewAccountAPI()
	api.FailField = "setLogoutUris"
	server := httptest.NewServer(api)
	defer server.Close()
	config := accountTerraformConfig(server.URL)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("Unable to Apply AuthKit Configuration")},
		{PreConfig: func() { api.Mu.Lock(); defer api.Mu.Unlock(); api.FailField = "" }, Config: config, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("workos_authkit_configuration.test", "default_logout_uri", "https://vigile.example/signed-out"),
			resource.TestCheckResourceAttr("workos_authkit_configuration.test", "authkit_enabled", "true"),
		)},
		{Config: config, PlanOnly: true},
	}})
	if len(api.Writes) != 6 {
		t.Fatalf("retry must converge without repeating successful writes; got %d writes", len(api.Writes))
	}
}
