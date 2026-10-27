// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
)

// Documents select only configuration fields from the official WorkOS CLI catalog.
const accountTargetQuery = `query teamProjectsV2 {
  currentTeam { id projectsV2 { environments { id } } }
}`

const authkitConfigurationQuery = `query authkitConfiguration($environmentId: String!, $applicationId: ID!) {
  environment(id: $environmentId) { id webOrigins { origins } userlandSettings { isAuthkitEnabled allowSignUp isWaitlistEnabled } }
  userlandApplication(id: $applicationId) {
    id environmentId clientId isDefault initiateLoginUri
    redirectUris { id uri isDefault }
    logoutUris { id uri isDefault }
  }
  defaultUserlandApplication(environmentId: $environmentId) { id environmentId }
}`

const setRedirectURIsMutation = `mutation setRedirectUris($input: SetRedirectUrisInput!) {
  setRedirectUris(input: $input) { __typename }
}`
const setLogoutURIsMutation = `mutation setLogoutUris($input: SetLogoutUrisInput!) {
  setLogoutUris(input: $input) { __typename }
}`
const setWebOriginsMutation = `mutation updateCorsConfig($environmentId: String!, $origins: [String!]!, $dryRun: Boolean) {
  setWebOrigins(environmentId: $environmentId, origins: $origins, dryRun: $dryRun) { __typename }
}`
const updateAuthkitApplicationMutation = `mutation updateAuthkitApplication($input: UpdateUserlandApplicationInput!) {
  updateUserlandApplication(input: $input) { __typename }
}`
const updateUserlandSettingsMutation = `mutation updateUserlandSettings($input: UpdateUserlandSettingsInput!) {
  updateUserlandSettings(input: $input) { __typename }
}`

// AuthkitConfiguration owns complete URI/origin sets for the default AuthKit app.
type AuthkitConfiguration struct {
	EnvironmentID      string
	ApplicationID      string
	RedirectURIs       []string
	DefaultRedirectURI string
	LogoutURIs         []string
	DefaultLogoutURI   string
	CORSOrigins        []string
	InitiateLoginURI   string
	AuthkitEnabled     bool
	AllowSignup        bool
	WaitlistEnabled    bool
}

type configurationURI struct {
	ID        string `json:"id,omitempty"`
	URI       string `json:"uri"`
	IsDefault bool   `json:"isDefault"`
}

type authkitConfigurationData struct {
	Environment *struct {
		ID         string `json:"id"`
		WebOrigins *struct {
			Origins []string `json:"origins"`
		} `json:"webOrigins"`
		Settings *struct {
			AuthkitEnabled  *bool `json:"isAuthkitEnabled"`
			AllowSignup     *bool `json:"allowSignUp"`
			WaitlistEnabled *bool `json:"isWaitlistEnabled"`
		} `json:"userlandSettings"`
	} `json:"environment"`
	Application *struct {
		ID               string             `json:"id"`
		EnvironmentID    string             `json:"environmentId"`
		ClientID         string             `json:"clientId"`
		IsDefault        bool               `json:"isDefault"`
		InitiateLoginURI *string            `json:"initiateLoginUri"`
		RedirectURIs     []configurationURI `json:"redirectUris"`
		LogoutURIs       []configurationURI `json:"logoutUris"`
	} `json:"userlandApplication"`
	DefaultApplication *struct {
		ID            string `json:"id"`
		EnvironmentID string `json:"environmentId"`
	} `json:"defaultUserlandApplication"`
}

// accountRequest never exposes response bodies, GraphQL messages, or credentials.
func (c *Client) accountRequest(ctx context.Context, environmentID, query string, variables any, result any) error {
	if c.accountAccessToken == "" || c.accountTeamID == "" {
		return fmt.Errorf("account configuration requires account_access_token and account_team_id")
	}
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return fmt.Errorf("cannot encode account configuration request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+"/graphql", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("invalid account API URL")
	}
	req.Header.Set("Authorization", "Bearer "+c.accountAccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "terraform-provider-workos")
	if environmentID != "" {
		req.Header.Set("x-url-environment-id", environmentID)
	}
	httpClient := *c.httpClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("account API request failed or timed out; retry after checking connectivity")
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("account OAuth credential was rejected or expired (HTTP 401); supply a current account_access_token")
	case http.StatusForbidden:
		return fmt.Errorf("account API access denied (HTTP 403); verify team access and the WorkOS account-plane feature flag")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("account API returned HTTP %d", resp.StatusCode)
	}
	const maxResponseSize = 4 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil || len(data) > maxResponseSize {
		return fmt.Errorf("account API response is unreadable or exceeds the safe size limit; no complete configuration was accepted")
	}
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Extensions struct {
				Code string `json:"code"`
			} `json:"extensions"`
		} `json:"errors"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return fmt.Errorf("account API returned invalid JSON")
	}
	if len(envelope.Errors) != 0 {
		for _, e := range envelope.Errors {
			if e.Extensions.Code == "UNAUTHENTICATED" {
				return fmt.Errorf("account OAuth credential was rejected or expired; supply a current account_access_token")
			}
			if e.Extensions.Code == "FORBIDDEN" {
				return fmt.Errorf("account API access denied; verify team access and the WorkOS account-plane feature flag")
			}
		}
		return fmt.Errorf("account GraphQL operation failed; server messages are withheld to protect sensitive data")
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return fmt.Errorf("account API returned no configuration data")
	}
	if json.Unmarshal(envelope.Data, result) != nil {
		return fmt.Errorf("account API returned an invalid configuration shape")
	}
	return nil
}

func (c *Client) validateAccountTarget(ctx context.Context, environmentID string) error {
	if c.accountEnvironmentID != "" && environmentID != c.accountEnvironmentID {
		return fmt.Errorf("resource environment_id must match the preflighted provider environment_id")
	}
	if strings.TrimSpace(environmentID) == "" {
		return fmt.Errorf("an explicit environment_id is required; production fallback is forbidden")
	}
	var data struct {
		Team *struct {
			ID       string `json:"id"`
			Projects []struct {
				Environments []struct {
					ID string `json:"id"`
				} `json:"environments"`
			} `json:"projectsV2"`
		} `json:"currentTeam"`
	}
	if err := c.accountRequest(ctx, "", accountTargetQuery, map[string]any{}, &data); err != nil {
		return err
	}
	if data.Team == nil || data.Team.ID != c.accountTeamID {
		return fmt.Errorf("account team does not match account_team_id; no configuration operation is allowed")
	}
	count := 0
	for _, project := range data.Team.Projects {
		for _, environment := range project.Environments {
			if environment.ID == environmentID {
				count++
			}
		}
	}
	if count != 1 {
		return fmt.Errorf("environment_id is missing, stale, or ambiguous in the configured team; production fallback is forbidden")
	}
	return nil
}

// ReadAuthkitConfiguration validates ownership before accepting any scoped data.
func (c *Client) ReadAuthkitConfiguration(ctx context.Context, environmentID, applicationID string) (*AuthkitConfiguration, error) {
	if c.accountApplicationID != "" && applicationID != c.accountApplicationID {
		return nil, fmt.Errorf("resource application_id must match the preflighted provider application_id")
	}
	if strings.TrimSpace(applicationID) == "" {
		return nil, fmt.Errorf("an explicit application_id is required")
	}
	if err := c.validateAccountTarget(ctx, environmentID); err != nil {
		return nil, err
	}
	var data authkitConfigurationData
	if err := c.accountRequest(ctx, environmentID, authkitConfigurationQuery, map[string]any{"environmentId": environmentID, "applicationId": applicationID}, &data); err != nil {
		return nil, err
	}
	if data.Environment == nil || data.Environment.ID != environmentID || data.Application == nil || data.Application.ID != applicationID || data.Application.EnvironmentID != environmentID || !data.Application.IsDefault || data.DefaultApplication == nil || data.DefaultApplication.ID != applicationID || data.DefaultApplication.EnvironmentID != environmentID {
		return nil, fmt.Errorf("application_id must be the default AuthKit application of the exact configured environment; no configuration operation is allowed")
	}
	if c.clientID != "" && data.Application.ClientID != c.clientID {
		return nil, fmt.Errorf("provider client_id does not match the explicitly configured AuthKit application")
	}
	settings := data.Environment.Settings
	if settings == nil || settings.AuthkitEnabled == nil || settings.AllowSignup == nil || settings.WaitlistEnabled == nil || data.Environment.WebOrigins == nil || data.Environment.WebOrigins.Origins == nil || data.Application.RedirectURIs == nil || data.Application.LogoutURIs == nil {
		return nil, fmt.Errorf("account API returned incomplete configuration; full URI and origin sets are required")
	}
	redirects, defaultRedirect, err := readConfigurationURIs(data.Application.RedirectURIs)
	if err != nil {
		return nil, err
	}
	logouts, defaultLogout, err := readConfigurationURIs(data.Application.LogoutURIs)
	if err != nil {
		return nil, err
	}
	initiateLoginURI := ""
	if data.Application.InitiateLoginURI != nil {
		initiateLoginURI = *data.Application.InitiateLoginURI
	}
	return &AuthkitConfiguration{
		EnvironmentID: environmentID, ApplicationID: applicationID,
		RedirectURIs: redirects, DefaultRedirectURI: defaultRedirect,
		LogoutURIs: logouts, DefaultLogoutURI: defaultLogout,
		CORSOrigins:      sortedStrings(data.Environment.WebOrigins.Origins),
		InitiateLoginURI: initiateLoginURI, AuthkitEnabled: *settings.AuthkitEnabled, AllowSignup: *settings.AllowSignup, WaitlistEnabled: *settings.WaitlistEnabled,
	}, nil
}

func readConfigurationURIs(items []configurationURI) ([]string, string, error) {
	values := make([]string, 0, len(items))
	seen := make(map[string]bool)
	defaultURI := ""
	for _, item := range items {
		if item.URI == "" || seen[item.URI] {
			return nil, "", fmt.Errorf("account API returned an invalid or duplicate URI; full configuration was not accepted")
		}
		seen[item.URI] = true
		values = append(values, item.URI)
		if item.IsDefault {
			if defaultURI != "" {
				return nil, "", fmt.Errorf("account API returned multiple default URIs")
			}
			defaultURI = item.URI
		}
	}
	return sortedStrings(values), defaultURI, nil
}

func sortedStrings(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	return result
}

// ValidateAuthkitConfiguration checks every intended value before any mutation.
func ValidateAuthkitConfiguration(config *AuthkitConfiguration) error {
	if config.EnvironmentID == "" || config.ApplicationID == "" {
		return fmt.Errorf("environment_id and application_id are required")
	}
	for _, list := range []struct {
		name       string
		values     []string
		defaultURI string
	}{
		{"redirect_uris", config.RedirectURIs, config.DefaultRedirectURI}, {"logout_uris", config.LogoutURIs, config.DefaultLogoutURI},
	} {
		if len(list.values) == 0 {
			return fmt.Errorf("%s must contain at least one URI", list.name)
		}
		found := false
		seen := make(map[string]bool)
		for _, value := range list.values {
			if !validConfigurationURL(value, false) || seen[value] {
				return fmt.Errorf("%s must contain unique absolute HTTP(S) URLs without credentials or fragments; a single wildcard is only allowed in the left-most host label or as the port of a localhost/loopback URL", list.name)
			}
			seen[value] = true
			if value == list.defaultURI {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("the default URI must be a member of %s", list.name)
		}
		if strings.Contains(list.defaultURI, "*") {
			return fmt.Errorf("the default URI in %s cannot contain a wildcard", list.name)
		}
	}
	for _, origin := range config.CORSOrigins {
		if !validConfigurationURL(origin, true) {
			return fmt.Errorf("cors_origins must contain HTTP(S) origins without paths, queries, credentials, or fragments; a single wildcard is only allowed in the left-most host label or as the port of a localhost/loopback origin")
		}
	}
	if strings.Contains(config.InitiateLoginURI, "*") || !validConfigurationURL(config.InitiateLoginURI, false) {
		return fmt.Errorf("initiate_login_uri must be an absolute HTTP(S) URL without credentials, wildcards, or fragments")
	}
	return nil
}

// validConfigurationURL accepts absolute HTTP(S) URLs without credentials or
// fragments. One wildcard is accepted where WorkOS documents support for it:
// in the left-most host label (https://*.example.com,
// https://prefix-*-suffix.example.com) or in place of the port on localhost
// and loopback hosts (http://localhost:*/callback). The account API dry-runs
// every URI and origin list before it is written, so server-side rules such
// as the public-suffix and per-environment wildcard limits stay authoritative.
func validConfigurationURL(value string, origin bool) bool {
	if strings.TrimSpace(value) != value || strings.ContainsAny(value, " \r\n\t") || strings.Count(value, "*") > 1 {
		return false
	}
	parsable, ok := substituteWildcard(value)
	if !ok {
		return false
	}
	u, err := url.Parse(parsable)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return false
	}
	if origin && (u.Path != "" || u.RawQuery != "" || u.ForceQuery) {
		return false
	}
	return true
}

// substituteWildcard replaces a permitted wildcard with a parsable stand-in
// and reports whether the wildcard sits in a position WorkOS allows.
func substituteWildcard(value string) (string, bool) {
	star := strings.IndexByte(value, '*')
	if star < 0 {
		return value, true
	}
	schemeEnd := strings.Index(value, "://")
	if schemeEnd < 0 || star < schemeEnd+3 {
		return "", false
	}
	authorityStart := schemeEnd + 3
	authorityEnd := len(value)
	if i := strings.IndexAny(value[authorityStart:], "/?#"); i >= 0 {
		authorityEnd = authorityStart + i
	}
	if star >= authorityEnd {
		return "", false
	}
	authority := value[authorityStart:authorityEnd]
	if strings.HasSuffix(authority, ":*") {
		switch strings.TrimSuffix(authority, ":*") {
		case "localhost", "127.0.0.1", "[::1]":
			return value[:star] + "1" + value[star+1:], true
		}
		return "", false
	}
	host := authority
	if i := strings.LastIndexByte(authority, ':'); i >= 0 && !strings.Contains(authority[i:], "]") {
		host = authority[:i]
	}
	offset := star - authorityStart
	if offset >= len(host) || strings.Contains(host[:offset], ".") {
		return "", false
	}
	return value[:star] + "wildcard" + value[star+1:], true
}

func configurationURIInputs(values []string, defaultURI string) []configurationURI {
	items := make([]configurationURI, 0, len(values))
	for _, value := range sortedStrings(values) {
		items = append(items, configurationURI{URI: value, IsDefault: value == defaultURI})
	}
	return items
}

type configurationMutation struct {
	query, field, success string
	variables             map[string]any
}

func (c *Client) runConfigurationMutation(ctx context.Context, environmentID string, mutation configurationMutation) error {
	var result map[string]struct {
		Type string `json:"__typename"`
	}
	if err := c.accountRequest(ctx, environmentID, mutation.query, mutation.variables, &result); err != nil {
		return err
	}
	actual := result[mutation.field].Type
	if actual == "" {
		return fmt.Errorf("account API returned no result for %s", mutation.field)
	}
	// Result type names such as InvalidWildcardRedirectUri are safe to surface;
	// server messages are withheld because they may echo sensitive values.
	rejected := mutation.success != "" && actual != mutation.success
	// The catalog does not enumerate this result union; independent readback verifies it.
	rejected = rejected || (mutation.success == "" && (strings.Contains(actual, "Error") || strings.Contains(actual, "Failed") || strings.Contains(actual, "NotFound")))
	if rejected {
		return fmt.Errorf("account API rejected %s with %s; check the configured values against the WorkOS dashboard rules for this environment", mutation.field, actual)
	}
	return nil
}

// ApplyAuthkitConfiguration returns observed state even after a partial failure.
// Retrying reconciles only the settings that still differ.
func (c *Client) ApplyAuthkitConfiguration(ctx context.Context, desired *AuthkitConfiguration) (*AuthkitConfiguration, error) {
	if err := ValidateAuthkitConfiguration(desired); err != nil {
		return nil, err
	}
	observed, err := c.ReadAuthkitConfiguration(ctx, desired.EnvironmentID, desired.ApplicationID)
	if err != nil {
		return nil, err
	}
	var mutations []configurationMutation
	if !reflect.DeepEqual(sortedStrings(desired.RedirectURIs), observed.RedirectURIs) || desired.DefaultRedirectURI != observed.DefaultRedirectURI {
		// applicationId selects an IDP app; environmentId selects the default AuthKit app.
		mutations = append(mutations, configurationMutation{setRedirectURIsMutation, "setRedirectUris", "RedirectUrisSet", map[string]any{"input": map[string]any{"environmentId": desired.EnvironmentID, "redirectUris": configurationURIInputs(desired.RedirectURIs, desired.DefaultRedirectURI)}}})
	}
	if !reflect.DeepEqual(sortedStrings(desired.LogoutURIs), observed.LogoutURIs) || desired.DefaultLogoutURI != observed.DefaultLogoutURI {
		mutations = append(mutations, configurationMutation{setLogoutURIsMutation, "setLogoutUris", "LogoutUrisSet", map[string]any{"input": map[string]any{"environmentId": desired.EnvironmentID, "logoutUris": configurationURIInputs(desired.LogoutURIs, desired.DefaultLogoutURI)}}})
	}
	if !reflect.DeepEqual(sortedStrings(desired.CORSOrigins), observed.CORSOrigins) {
		mutations = append(mutations, configurationMutation{setWebOriginsMutation, "setWebOrigins", "WebOriginsSet", map[string]any{"environmentId": desired.EnvironmentID, "origins": sortedStrings(desired.CORSOrigins)}})
	}
	if desired.InitiateLoginURI != observed.InitiateLoginURI {
		mutations = append(mutations, configurationMutation{updateAuthkitApplicationMutation, "updateUserlandApplication", "UserlandApplicationUpdated", map[string]any{"input": map[string]any{"applicationId": desired.ApplicationID, "initiateLoginUri": desired.InitiateLoginURI}}})
	}
	if desired.AuthkitEnabled != observed.AuthkitEnabled || desired.AllowSignup != observed.AllowSignup || desired.WaitlistEnabled != observed.WaitlistEnabled {
		mutations = append(mutations, configurationMutation{updateUserlandSettingsMutation, "updateUserlandSettings", "", map[string]any{"input": map[string]any{"environmentId": desired.EnvironmentID, "isAuthkitEnabled": desired.AuthkitEnabled, "allowSignUp": desired.AllowSignup, "isWaitlistEnabled": desired.WaitlistEnabled}}})
	}
	// Validate every list on the server before making any persistent change.
	for _, mutation := range mutations {
		if mutation.field != "setRedirectUris" && mutation.field != "setLogoutUris" && mutation.field != "setWebOrigins" {
			continue
		}
		if _, err = c.ReadAuthkitConfiguration(ctx, desired.EnvironmentID, desired.ApplicationID); err != nil {
			return observed, err
		}
		if input, ok := mutation.variables["input"].(map[string]any); ok {
			input["dryRun"] = true
		} else {
			mutation.variables["dryRun"] = true
		}
		err = c.runConfigurationMutation(ctx, desired.EnvironmentID, mutation)
		if input, ok := mutation.variables["input"].(map[string]any); ok {
			input["dryRun"] = false
		} else {
			mutation.variables["dryRun"] = false
		}
		if err != nil {
			return observed, err
		}
	}
	for _, mutation := range mutations {
		if _, err = c.ReadAuthkitConfiguration(ctx, desired.EnvironmentID, desired.ApplicationID); err != nil {
			return observed, err
		}
		mutationErr := c.runConfigurationMutation(ctx, desired.EnvironmentID, mutation)
		latest, readErr := c.ReadAuthkitConfiguration(ctx, desired.EnvironmentID, desired.ApplicationID)
		if latest != nil {
			observed = latest
		}
		if mutationErr != nil {
			return observed, mutationErr
		}
		if readErr != nil {
			return observed, fmt.Errorf("configuration may have changed but readback failed: %w; refresh and retry", readErr)
		}
		matches := false
		switch mutation.field {
		case "setRedirectUris":
			matches = reflect.DeepEqual(sortedStrings(desired.RedirectURIs), observed.RedirectURIs) && desired.DefaultRedirectURI == observed.DefaultRedirectURI
		case "setLogoutUris":
			matches = reflect.DeepEqual(sortedStrings(desired.LogoutURIs), observed.LogoutURIs) && desired.DefaultLogoutURI == observed.DefaultLogoutURI
		case "setWebOrigins":
			matches = reflect.DeepEqual(sortedStrings(desired.CORSOrigins), observed.CORSOrigins)
		case "updateUserlandApplication":
			matches = desired.InitiateLoginURI == observed.InitiateLoginURI
		case "updateUserlandSettings":
			matches = desired.AuthkitEnabled == observed.AuthkitEnabled && desired.AllowSignup == observed.AllowSignup && desired.WaitlistEnabled == observed.WaitlistEnabled
		}
		if !matches {
			return observed, fmt.Errorf("configuration readback did not confirm %s; refresh and retry", mutation.field)
		}
	}
	normalized := *desired
	normalized.RedirectURIs = sortedStrings(desired.RedirectURIs)
	normalized.LogoutURIs = sortedStrings(desired.LogoutURIs)
	normalized.CORSOrigins = sortedStrings(desired.CORSOrigins)
	if !reflect.DeepEqual(&normalized, observed) {
		return observed, fmt.Errorf("configuration readback differs from the intended settings; refresh and retry to reconcile remaining drift")
	}
	return observed, nil
}
