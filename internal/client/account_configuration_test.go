// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/osodevops/terraform-provider-workos/internal/client"
	"github.com/osodevops/terraform-provider-workos/internal/testutil"
)

func accountClient(t *testing.T, api *testutil.AccountAPI) *client.Client {
	t.Helper()
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)
	c, err := client.NewClientWithCredentials("sk_test", "client_test", server.URL, "account-test-token", "team_test")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func desiredConfiguration() *client.AuthkitConfiguration {
	return &client.AuthkitConfiguration{
		EnvironmentID: "environment_test", ApplicationID: "application_test",
		RedirectURIs: []string{"https://vigile.example/callback", "https://vigile.example/extra-callback"}, DefaultRedirectURI: "https://vigile.example/callback",
		LogoutURIs: []string{"https://vigile.example/signed-out", "https://vigile.example/extra-logout"}, DefaultLogoutURI: "https://vigile.example/signed-out",
		CORSOrigins: []string{"https://vigile.example"}, InitiateLoginURI: "https://vigile.example/login", AuthkitEnabled: true, AllowSignup: true, WaitlistEnabled: false,
	}
}

func decodeVariables(t *testing.T, request testutil.AccountRequest) map[string]any {
	t.Helper()
	var variables map[string]any
	raw, err := json.Marshal(request.Variables)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &variables); err != nil {
		t.Fatal(err)
	}
	return variables
}

func TestAuthkitConfigurationPayloadsAndConvergence(t *testing.T) {
	api := testutil.NewAccountAPI()
	c := accountClient(t, api)
	desired := desiredConfiguration()
	sort.Strings(desired.LogoutURIs)
	observed, err := c.ApplyAuthkitConfiguration(context.Background(), desired)
	if err != nil {
		t.Fatal(err)
	}
	if len(api.Writes) != 5 {
		t.Fatalf("got %d persistent writes, want 5", len(api.Writes))
	}
	if !reflect.DeepEqual(observed, desired) {
		t.Fatalf("observed configuration differs: %#v", observed)
	}
	want := []map[string]any{
		{"input": map[string]any{"environmentId": "environment_test", "dryRun": false, "redirectUris": []any{map[string]any{"uri": "https://vigile.example/callback", "isDefault": true}, map[string]any{"uri": "https://vigile.example/extra-callback", "isDefault": false}}}},
		{"input": map[string]any{"environmentId": "environment_test", "dryRun": false, "logoutUris": []any{map[string]any{"uri": "https://vigile.example/extra-logout", "isDefault": false}, map[string]any{"uri": "https://vigile.example/signed-out", "isDefault": true}}}},
		{"environmentId": "environment_test", "origins": []any{"https://vigile.example"}, "dryRun": false},
		{"input": map[string]any{"applicationId": "application_test", "initiateLoginUri": "https://vigile.example/login"}},
		{"input": map[string]any{"environmentId": "environment_test", "isAuthkitEnabled": true, "allowSignUp": true, "isWaitlistEnabled": false}},
	}
	for i, request := range api.Writes {
		if got := decodeVariables(t, request); !reflect.DeepEqual(got, want[i]) {
			t.Errorf("payload %d = %#v, want %#v", i, got, want[i])
		}
		if request.EnvironmentID != "environment_test" {
			t.Error("mutation missing exact environment header")
		}
	}
	dryRuns := 0
	for _, request := range api.Requests {
		if strings.Contains(request.Query, "query teamProjectsV2") && request.EnvironmentID != "" {
			t.Error("team lookup must be unscoped")
		}
		if !strings.HasPrefix(request.Query, "mutation") {
			continue
		}
		variables := decodeVariables(t, request)
		dryRun, _ := variables["dryRun"].(bool)
		if input, ok := variables["input"].(map[string]any); ok {
			dryRun, _ = input["dryRun"].(bool)
		}
		if dryRun {
			dryRuns++
		} else if dryRuns != 3 {
			t.Fatal("persistent write occurred before every list passed dry-run validation")
		}
	}
	if _, err := c.ApplyAuthkitConfiguration(context.Background(), desired); err != nil {
		t.Fatal(err)
	}
	if len(api.Writes) != 5 {
		t.Fatal("second reconciliation must not write")
	}
	api.Config.DefaultLogoutURI = "https://vigile.example/extra-logout"
	if _, err := c.ApplyAuthkitConfiguration(context.Background(), desired); err != nil {
		t.Fatal(err)
	}
	if len(api.Writes) != 6 || !strings.Contains(api.Writes[5].Query, "mutation setLogoutUris") {
		t.Fatal("default-only drift must reconcile the logout set once")
	}
}

func TestAuthkitConfigurationPartialFailureRecovers(t *testing.T) {
	api := testutil.NewAccountAPI()
	api.FailField = "setLogoutUris"
	c := accountClient(t, api)
	observed, err := c.ApplyAuthkitConfiguration(context.Background(), desiredConfiguration())
	if err == nil || observed == nil {
		t.Fatal("partial failure must return an error and observed state")
	}
	if len(api.Writes) != 1 || observed.DefaultRedirectURI != "https://vigile.example/callback" || observed.DefaultLogoutURI != "https://old.example/logout" || observed.AuthkitEnabled {
		t.Fatalf("incorrect partial state: %#v", observed)
	}
	if strings.Contains(err.Error(), "secret-token") || strings.Contains(err.Error(), "raw-sensitive") {
		t.Fatal("union error leaked raw server data")
	}
	api.FailField = ""
	if observed, err = c.ApplyAuthkitConfiguration(context.Background(), desiredConfiguration()); err != nil {
		t.Fatal(err)
	}
	if len(api.Writes) != 5 || !observed.AuthkitEnabled {
		t.Fatal("retry should converge remaining fields without repeating the completed redirect write")
	}
}

func TestAuthkitConfigurationDryRunFailurePreventsAllWrites(t *testing.T) {
	api := testutil.NewAccountAPI()
	api.FailField = "setWebOrigins"
	api.FailDryRun = true
	_, err := accountClient(t, api).ApplyAuthkitConfiguration(context.Background(), desiredConfiguration())
	if err == nil || len(api.Writes) != 0 {
		t.Fatal("all intended lists must validate before any persistent write")
	}
	if !strings.Contains(err.Error(), "setWebOrigins with ValidationFailed") || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("rejection must name the result type and withhold the server message: %v", err)
	}
}

func TestAuthkitConfigurationReadbackDetectsIgnoredMutation(t *testing.T) {
	api := testutil.NewAccountAPI()
	api.IgnoreField = "updateUserlandSettings"
	observed, err := accountClient(t, api).ApplyAuthkitConfiguration(context.Background(), desiredConfiguration())
	if err == nil || observed == nil || observed.AuthkitEnabled || observed.AllowSignup {
		t.Fatal("must reject successful-looking mutation when actual settings differ")
	}
}

func TestAuthkitConfigurationEnablementOnlyDrift(t *testing.T) {
	api := testutil.NewAccountAPI()
	desired := desiredConfiguration()
	api.Config = *desired
	api.Config.AuthkitEnabled = false
	c := accountClient(t, api)
	observed, err := c.ApplyAuthkitConfiguration(context.Background(), desired)
	if err != nil || observed == nil || !observed.AuthkitEnabled || len(api.Writes) != 1 {
		t.Fatalf("enablement-only drift must be corrected with one write: observed=%#v, error=%v", observed, err)
	}
	want := map[string]any{"input": map[string]any{"environmentId": "environment_test", "isAuthkitEnabled": true, "allowSignUp": true, "isWaitlistEnabled": false}}
	if got := decodeVariables(t, api.Writes[0]); !reflect.DeepEqual(got, want) {
		t.Fatalf("enablement payload=%#v", got)
	}
	if _, err = c.ApplyAuthkitConfiguration(context.Background(), desired); err != nil {
		t.Fatal(err)
	}
	if len(api.Writes) != 1 {
		t.Fatal("second enablement reconciliation must be a no-op")
	}
	desired.AuthkitEnabled = false
	observed, err = c.ApplyAuthkitConfiguration(context.Background(), desired)
	if err != nil || observed == nil || observed.AuthkitEnabled || len(api.Writes) != 2 {
		t.Fatalf("explicit false must disable AuthKit: observed=%#v, error=%v", observed, err)
	}
	want["input"].(map[string]any)["isAuthkitEnabled"] = false
	if got := decodeVariables(t, api.Writes[1]); !reflect.DeepEqual(got, want) {
		t.Fatalf("disablement payload=%#v", got)
	}
}

func TestAuthkitConfigurationEnablementReadbackRejectsIgnoredMutation(t *testing.T) {
	api := testutil.NewAccountAPI()
	desired := desiredConfiguration()
	api.Config = *desired
	api.Config.AuthkitEnabled = false
	api.IgnoreField = "updateUserlandSettings"
	observed, err := accountClient(t, api).ApplyAuthkitConfiguration(context.Background(), desired)
	if err == nil || observed == nil || observed.AuthkitEnabled || len(api.Writes) != 1 {
		t.Fatal("enablement must be verified independently of unchanged signup and waitlist settings")
	}
}

func TestAccountTargetsFailClosed(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*testutil.AccountAPI, *client.AuthkitConfiguration)
	}{
		{"wrong team", func(api *testutil.AccountAPI, _ *client.AuthkitConfiguration) { api.TeamID = "other_team" }},
		{"stale environment", func(api *testutil.AccountAPI, _ *client.AuthkitConfiguration) {
			api.VisibleEnvironmentID = "other_environment"
		}},
		{"foreign application", func(api *testutil.AccountAPI, _ *client.AuthkitConfiguration) {
			api.ApplicationEnvironmentID = "other_environment"
		}},
		{"wrong default application", func(api *testutil.AccountAPI, _ *client.AuthkitConfiguration) {
			api.DefaultApplicationID = "other_application"
		}},
		{"missing environment", func(_ *testutil.AccountAPI, c *client.AuthkitConfiguration) { c.EnvironmentID = "" }},
		{"wrong environment", func(_ *testutil.AccountAPI, c *client.AuthkitConfiguration) { c.EnvironmentID = "other_environment" }},
		{"missing application", func(_ *testutil.AccountAPI, c *client.AuthkitConfiguration) { c.ApplicationID = "" }},
		{"wrong application", func(_ *testutil.AccountAPI, c *client.AuthkitConfiguration) { c.ApplicationID = "other_application" }},
		{"incomplete configuration", func(api *testutil.AccountAPI, _ *client.AuthkitConfiguration) { api.OmitOrigins = true }},
		{"missing enablement state", func(api *testutil.AccountAPI, _ *client.AuthkitConfiguration) { api.OmitAuthkitEnabled = true }},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := testutil.NewAccountAPI()
			desired := desiredConfiguration()
			test.change(api, desired)
			_, err := accountClient(t, api).ApplyAuthkitConfiguration(context.Background(), desired)
			if err == nil || len(api.Writes) != 0 {
				t.Fatal("invalid target must fail before writing")
			}
		})
	}
}

func TestAuthkitConfigurationRevalidatesTargetBetweenWrites(t *testing.T) {
	api := testutil.NewAccountAPI()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.ServeHTTP(w, r)
		api.Mu.Lock()
		defer api.Mu.Unlock()
		if len(api.Writes) > 0 {
			api.VisibleEnvironmentID = "deleted_environment"
		}
	}))
	defer server.Close()
	c, _ := client.NewClientWithCredentials("sk_test", "client_test", server.URL, "account-test-token", "team_test")
	_, err := c.ApplyAuthkitConfiguration(context.Background(), desiredConfiguration())
	if err == nil || len(api.Writes) != 1 {
		t.Fatal("stale target must prevent every subsequent write")
	}
}

func TestConfigurationReadsCompleteArrays(t *testing.T) {
	api := testutil.NewAccountAPI()
	for i := 0; i < 225; i++ {
		api.Config.RedirectURIs = append(api.Config.RedirectURIs, fmt.Sprintf("https://vigile.example/callback/%d", i))
		api.Config.LogoutURIs = append(api.Config.LogoutURIs, fmt.Sprintf("https://vigile.example/logout/%d", i))
	}
	observed, err := accountClient(t, api).ReadAuthkitConfiguration(context.Background(), "environment_test", "application_test")
	if err != nil {
		t.Fatal(err)
	}
	if len(observed.RedirectURIs) != 226 || len(observed.LogoutURIs) != 226 {
		t.Fatal("complete application arrays were truncated")
	}
	for _, request := range api.Requests {
		if strings.Contains(request.Query, "limit:") {
			t.Fatal("full-list ownership must not use capped list queries")
		}
	}
}

func TestAccountPreflightVerifiesRESTEnvironmentAndPagination(t *testing.T) {
	for _, test := range []struct {
		name                       string
		mismatch, paginate, repeat bool
		wantError                  bool
	}{
		{"matching", false, false, false, false}, {"matching paginated", false, true, false, false}, {"wrong REST environment", true, false, false, true}, {"repeating cursor", false, true, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := testutil.NewAccountAPI()
			api.RESTEnvironmentMismatch = test.mismatch
			api.RESTPaginate = test.paginate
			api.RESTRepeatCursor = test.repeat
			c := accountClient(t, api)
			err := c.PreflightAccountConfiguration(context.Background(), "environment_test", "application_test")
			if (err != nil) != test.wantError {
				t.Fatalf("preflight error=%v", err)
			}
			if len(api.Writes) != 0 {
				t.Fatal("preflight must be read-only")
			}
			if test.paginate && api.RESTRequests != 2 {
				t.Fatalf("REST requests=%d, want two pages", api.RESTRequests)
			}
			if !test.wantError {
				if _, err = c.ReadAuthkitConfiguration(context.Background(), "other_environment", "application_test"); err == nil {
					t.Fatal("resource must match preflighted provider environment")
				}
				if _, err = c.ReadAuthkitConfiguration(context.Background(), "environment_test", "other_application"); err == nil {
					t.Fatal("resource must match preflighted provider application")
				}
			}
		})
	}
}

func TestEnvironmentRoleConfigurationResolvesSlugToVerifiedID(t *testing.T) {
	api := testutil.NewAccountAPI()
	c := accountClient(t, api)
	observed, err := c.ApplyEnvironmentRoleConfiguration(context.Background(), "environment_test", "viewer")
	if err != nil {
		t.Fatal(err)
	}
	if observed.DefaultRoleSlug != "viewer" || len(api.Writes) != 1 {
		t.Fatal("default viewer role not applied")
	}
	want := map[string]any{"input": map[string]any{"roleConfigId": "role_config_test", "defaultRole": "role_viewer"}}
	if got := decodeVariables(t, api.Writes[0]); !reflect.DeepEqual(got, want) {
		t.Fatalf("role payload=%#v", got)
	}
	if _, err = c.ApplyEnvironmentRoleConfiguration(context.Background(), "environment_test", "viewer"); err != nil {
		t.Fatal(err)
	}
	if len(api.Writes) != 1 {
		t.Fatal("second default-role reconciliation must be a no-op")
	}
	if len(api.Roles) != 2 {
		t.Fatal("configuration must preserve role definitions")
	}
}

func TestEnvironmentRoleConfigurationRejectsUnprovenIdentifiers(t *testing.T) {
	for _, test := range []struct{ name, defaultID, desiredSlug string }{
		{"slug is not ID", "admin", "viewer"}, {"missing current role", "unknown_id", "viewer"}, {"missing desired role", "role_admin", "unknown"},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := testutil.NewAccountAPI()
			api.DefaultRoleID = test.defaultID
			_, err := accountClient(t, api).ApplyEnvironmentRoleConfiguration(context.Background(), "environment_test", test.desiredSlug)
			if err == nil || len(api.Writes) != 0 {
				t.Fatal("role identifier must be proven before writes")
			}
		})
	}
}

func TestAccountErrorsAreRedacted(t *testing.T) {
	for _, status := range []int{401, 403, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			api := testutil.NewAccountAPI()
			api.HTTPStatus = status
			_, err := accountClient(t, api).ReadAuthkitConfiguration(context.Background(), "environment_test", "application_test")
			if err == nil || strings.Contains(err.Error(), "secret-token") || strings.Contains(err.Error(), "raw-sensitive") {
				t.Fatalf("unsafe error=%v", err)
			}
			if !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("HTTP status absent: %v", err)
			}
		})
	}
	for _, body := range []string{
		`{"errors":[{"message":"account-test-token raw-sensitive-server-message","extensions":{"code":"UNAUTHENTICATED"}}]}`,
		`{"errors":[{"message":"account-test-token raw-sensitive-server-message"}]}`,
		`{"data":null}`, `{"data":{"currentTeam":42}}`, `invalid account-test-token`,
	} {
		t.Run(body[:8], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			c, _ := client.NewClientWithCredentials("sk_test", "", server.URL, "account-test-token", "team_test")
			_, err := c.ReadAuthkitConfiguration(context.Background(), "environment_test", "application_test")
			if err == nil || strings.Contains(err.Error(), "account-test-token") || strings.Contains(err.Error(), "raw-sensitive") {
				t.Fatalf("unsafe error=%v", err)
			}
		})
	}
}

func TestAccountCredentialDoesNotAuthenticateREST(t *testing.T) {
	api := testutil.NewAccountAPI()
	server := httptest.NewServer(api)
	defer server.Close()
	c, err := client.NewClientWithCredentials("", "client_test", server.URL, "account-test-token", "team_test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.ListEnvironmentRoles(context.Background()); err == nil {
		t.Fatal("REST must require its own key")
	}
	if api.RESTRequests != 0 {
		t.Fatal("OAuth token must never be sent to REST")
	}
	if _, err = client.NewClientWithCredentials("", "", server.URL, "account-test-token", ""); err == nil {
		t.Fatal("OAuth must require explicit team")
	}
}

func TestValidateAuthkitConfigurationBeforeRequests(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*client.AuthkitConfiguration)
	}{
		{"missing callback default", func(c *client.AuthkitConfiguration) { c.DefaultRedirectURI = "https://other.example/callback" }},
		{"missing logout default", func(c *client.AuthkitConfiguration) { c.DefaultLogoutURI = "" }},
		{"empty callbacks", func(c *client.AuthkitConfiguration) { c.RedirectURIs = nil }},
		{"duplicate callbacks", func(c *client.AuthkitConfiguration) { c.RedirectURIs = append(c.RedirectURIs, c.RedirectURIs[0]) }},
		{"invalid origin", func(c *client.AuthkitConfiguration) { c.CORSOrigins = []string{"https://vigile.example/path"} }},
		{"origin query", func(c *client.AuthkitConfiguration) { c.CORSOrigins = []string{"https://vigile.example?x"} }},
		{"misplaced wildcard", func(c *client.AuthkitConfiguration) { c.CORSOrigins = []string{"https://sub.*.example.com"} }},
		{"two wildcards", func(c *client.AuthkitConfiguration) {
			c.RedirectURIs = append(c.RedirectURIs, "https://*-*.vigile.example/callback")
		}},
		{"wildcard path", func(c *client.AuthkitConfiguration) {
			c.RedirectURIs = append(c.RedirectURIs, "https://vigile.example/*")
		}},
		{"wildcard port on public host", func(c *client.AuthkitConfiguration) {
			c.LogoutURIs = append(c.LogoutURIs, "https://vigile.example:*/signed-out")
		}},
		{"wildcard default", func(c *client.AuthkitConfiguration) {
			c.RedirectURIs = append(c.RedirectURIs, "https://*.vigile.example/callback")
			c.DefaultRedirectURI = "https://*.vigile.example/callback"
		}},
		{"wildcard initiate login", func(c *client.AuthkitConfiguration) { c.InitiateLoginURI = "https://*.vigile.example/login" }},
		{"credentials", func(c *client.AuthkitConfiguration) { c.InitiateLoginURI = "https://user:secret@vigile.example/login" }},
		{"fragment", func(c *client.AuthkitConfiguration) { c.InitiateLoginURI = "https://vigile.example/login#fragment" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := testutil.NewAccountAPI()
			desired := desiredConfiguration()
			test.change(desired)
			_, err := accountClient(t, api).ApplyAuthkitConfiguration(context.Background(), desired)
			if err == nil || len(api.Requests) != 0 {
				t.Fatal("invalid intended settings must fail before any request")
			}
		})
	}
	valid := desiredConfiguration()
	valid.CORSOrigins = []string{}
	valid.InitiateLoginURI = "http://localhost:3000/login"
	if err := client.ValidateAuthkitConfiguration(valid); err != nil {
		t.Fatal(err)
	}
	// WorkOS allows one wildcard in the left-most host label, or in place of
	// the port on localhost/loopback hosts, as long as it is not the default.
	wildcards := desiredConfiguration()
	wildcards.RedirectURIs = append(wildcards.RedirectURIs, "https://*.preview.vigile.example/callback", "https://prefix-*-suffix.vigile.example/callback", "http://localhost:*/callback", "http://127.0.0.1:*/callback", "http://[::1]:*/callback")
	wildcards.LogoutURIs = append(wildcards.LogoutURIs, "https://*.preview.vigile.example/signed-out")
	wildcards.CORSOrigins = []string{"https://*.preview.vigile.example", "http://localhost:*", "https://vigile.example:8443"}
	if err := client.ValidateAuthkitConfiguration(wildcards); err != nil {
		t.Fatal(err)
	}
}
