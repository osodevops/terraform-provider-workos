// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

// Package testutil contains HTTP fixtures shared by configuration tests.
package testutil

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/osodevops/terraform-provider-workos/internal/client"
)

type AccountRequest struct {
	Query         string                     `json:"query"`
	Variables     map[string]json.RawMessage `json:"variables"`
	EnvironmentID string
}

type AccountAPI struct {
	Mu                       sync.Mutex
	Config                   client.AuthkitConfiguration
	TeamID                   string
	VisibleEnvironmentID     string
	ApplicationEnvironmentID string
	DefaultApplicationID     string
	Roles                    map[string]string
	DefaultRoleID            string
	RESTEnvironmentMismatch  bool
	RESTPaginate             bool
	RESTRepeatCursor         bool
	RESTRequests             int
	RESTMutationRequests     int
	RoleConfigID             string
	Requests                 []AccountRequest
	Writes                   []AccountRequest
	FailField                string
	FailDryRun               bool
	HTTPStatus               int
	OmitOrigins              bool
	OmitAuthkitEnabled       bool
	IgnoreField              string
}

func NewAccountAPI() *AccountAPI {
	return &AccountAPI{
		TeamID: "team_test", VisibleEnvironmentID: "environment_test", ApplicationEnvironmentID: "environment_test", DefaultApplicationID: "application_test",
		Config: client.AuthkitConfiguration{
			EnvironmentID: "environment_test", ApplicationID: "application_test",
			RedirectURIs: []string{"https://old.example/callback"}, DefaultRedirectURI: "https://old.example/callback",
			LogoutURIs: []string{"https://old.example/logout"}, DefaultLogoutURI: "https://old.example/logout",
			CORSOrigins: []string{"https://old.example"}, InitiateLoginURI: "https://old.example/login", AuthkitEnabled: false, AllowSignup: false, WaitlistEnabled: true,
		},
		Roles: map[string]string{"admin": "role_admin", "viewer": "role_viewer"}, DefaultRoleID: "role_admin", RoleConfigID: "role_config_test",
	}
}

func (api *AccountAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.Mu.Lock()
	defer api.Mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet && strings.HasPrefix(r.URL.Path, "/authorization/roles") {
		api.RESTMutationRequests++
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/authorization/roles" {
		api.RESTRequests++
		if r.Header.Get("Authorization") != "Bearer sk_test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		roles := []map[string]any{}
		for _, slug := range []string{"admin", "viewer"} {
			id := api.Roles[slug]
			if api.RESTEnvironmentMismatch {
				id = "other_" + id
			}
			if id != "" {
				roles = append(roles, map[string]any{"id": id, "slug": slug, "type": "EnvironmentRole"})
			}
		}
		after := ""
		if api.RESTPaginate && r.URL.Query().Get("after") == "" {
			roles = roles[:1]
			after = "next_page"
		} else if api.RESTPaginate {
			roles = roles[1:]
		}
		if api.RESTRepeatCursor {
			after = "next_page"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": roles, "list_metadata": map[string]any{"after": after}})
		return
	}
	if api.HTTPStatus != 0 {
		w.WriteHeader(api.HTTPStatus)
		_, _ = w.Write([]byte(`{"message":"secret-token raw-sensitive-server-message"}`))
		return
	}
	if r.Method != http.MethodPost || r.URL.Path != "/graphql" || r.Header.Get("Authorization") != "Bearer account-test-token" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var request AccountRequest
	if json.NewDecoder(r.Body).Decode(&request) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	request.EnvironmentID = r.Header.Get("x-url-environment-id")
	api.Requests = append(api.Requests, request)
	write := func(data any) { _ = json.NewEncoder(w).Encode(map[string]any{"data": data}) }
	if strings.Contains(request.Query, "query teamProjectsV2") {
		write(map[string]any{"currentTeam": map[string]any{"id": api.TeamID, "projectsV2": []any{map[string]any{"environments": []any{map[string]any{"id": api.VisibleEnvironmentID}}}}}})
		return
	}
	if request.EnvironmentID != api.Config.EnvironmentID {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if strings.Contains(request.Query, "query authkitConfiguration") {
		origins := any(map[string]any{"origins": api.Config.CORSOrigins})
		if api.OmitOrigins {
			origins = nil
		}
		uris := func(values []string, defaultURI string) []any {
			items := make([]any, 0, len(values))
			for _, value := range values {
				items = append(items, map[string]any{"id": "uri_id", "uri": value, "isDefault": value == defaultURI})
			}
			return items
		}
		settings := map[string]any{"isAuthkitEnabled": api.Config.AuthkitEnabled, "allowSignUp": api.Config.AllowSignup, "isWaitlistEnabled": api.Config.WaitlistEnabled}
		if api.OmitAuthkitEnabled {
			delete(settings, "isAuthkitEnabled")
		}
		write(map[string]any{
			"environment":                map[string]any{"id": api.Config.EnvironmentID, "webOrigins": origins, "userlandSettings": settings},
			"userlandApplication":        map[string]any{"id": api.Config.ApplicationID, "clientId": "client_test", "environmentId": api.ApplicationEnvironmentID, "isDefault": true, "initiateLoginUri": api.Config.InitiateLoginURI, "redirectUris": uris(api.Config.RedirectURIs, api.Config.DefaultRedirectURI), "logoutUris": uris(api.Config.LogoutURIs, api.Config.DefaultLogoutURI)},
			"defaultUserlandApplication": map[string]any{"id": api.DefaultApplicationID, "environmentId": api.Config.EnvironmentID},
		})
		return
	}
	if strings.Contains(request.Query, "query roles(") {
		roles := make([]any, 0, len(api.Roles))
		for slug, id := range api.Roles {
			roles = append(roles, map[string]any{"id": id, "slug": slug, "resourceTypeId": nil})
		}
		write(map[string]any{"rolesForEnvironment": map[string]any{"roles": roles}, "roleConfigForEnvironmentV2": map[string]any{"id": api.RoleConfigID, "defaultRole": api.DefaultRoleID}})
		return
	}
	var input struct {
		EnvironmentID string `json:"environmentId"`
		ApplicationID string `json:"applicationId"`
		DryRun        bool   `json:"dryRun"`
		RedirectURIs  []struct {
			URI       string `json:"uri"`
			IsDefault bool   `json:"isDefault"`
		} `json:"redirectUris"`
		LogoutURIs []struct {
			URI       string `json:"uri"`
			IsDefault bool   `json:"isDefault"`
		} `json:"logoutUris"`
		InitiateLoginURI string `json:"initiateLoginUri"`
		AuthkitEnabled   bool   `json:"isAuthkitEnabled"`
		AllowSignup      bool   `json:"allowSignUp"`
		WaitlistEnabled  bool   `json:"isWaitlistEnabled"`
		DefaultRole      string `json:"defaultRole"`
		RoleConfigID     string `json:"roleConfigId"`
	}
	_ = json.Unmarshal(request.Variables["input"], &input)
	var dryRun bool
	_ = json.Unmarshal(request.Variables["dryRun"], &dryRun)
	dryRun = dryRun || input.DryRun
	field, success := "", ""
	switch {
	case strings.Contains(request.Query, "mutation setRedirectUris"):
		field, success = "setRedirectUris", "RedirectUrisSet"
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(request.Variables["input"], &fields)
		// applicationId selects an IDP app, not the default AuthKit userland app.
		if _, provided := fields["applicationId"]; provided {
			write(map[string]any{field: map[string]any{"__typename": "ApplicationNotFound"}})
			return
		}
	case strings.Contains(request.Query, "mutation setLogoutUris"):
		field, success = "setLogoutUris", "LogoutUrisSet"
	case strings.Contains(request.Query, "mutation updateCorsConfig"):
		field, success = "setWebOrigins", "WebOriginsSet"
	case strings.Contains(request.Query, "mutation updateAuthkitApplication"):
		field, success = "updateUserlandApplication", "UserlandApplicationUpdated"
	case strings.Contains(request.Query, "mutation updateUserlandSettings"):
		field, success = "updateUserlandSettings", "UserlandSettingsUpdated"
	case strings.Contains(request.Query, "mutation updateRoleConfig"):
		field, success = "updateRoleConfig", "RoleConfigUpdated"
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if api.FailField == field && (!dryRun || api.FailDryRun) {
		write(map[string]any{field: map[string]any{"__typename": "ValidationFailed", "message": "secret-token raw-sensitive-server-message"}})
		return
	}
	if !dryRun {
		api.Writes = append(api.Writes, request)
		if api.IgnoreField != field {
			switch field {
			case "setRedirectUris":
				api.Config.RedirectURIs = []string{}
				api.Config.DefaultRedirectURI = ""
				for _, item := range input.RedirectURIs {
					api.Config.RedirectURIs = append(api.Config.RedirectURIs, item.URI)
					if item.IsDefault {
						api.Config.DefaultRedirectURI = item.URI
					}
				}
			case "setLogoutUris":
				api.Config.LogoutURIs = []string{}
				api.Config.DefaultLogoutURI = ""
				for _, item := range input.LogoutURIs {
					api.Config.LogoutURIs = append(api.Config.LogoutURIs, item.URI)
					if item.IsDefault {
						api.Config.DefaultLogoutURI = item.URI
					}
				}
			case "setWebOrigins":
				_ = json.Unmarshal(request.Variables["origins"], &api.Config.CORSOrigins)
			case "updateUserlandApplication":
				api.Config.InitiateLoginURI = input.InitiateLoginURI
			case "updateUserlandSettings":
				api.Config.AuthkitEnabled = input.AuthkitEnabled
				api.Config.AllowSignup = input.AllowSignup
				api.Config.WaitlistEnabled = input.WaitlistEnabled
			case "updateRoleConfig":
				api.DefaultRoleID = input.DefaultRole
			}
		}
	}
	write(map[string]any{field: map[string]any{"__typename": success}})
}
