// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const environmentRoleFixture = `{
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

func TestEnvironmentRolesClientCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/authorization/roles" {
			t.Fatalf("expected /authorization/roles, got %s", r.URL.Path)
		}

		var body EnvironmentRoleCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Slug != "admin" || body.Name != "Admin" || body.Description != "Can manage all resources" || body.ResourceTypeSlug != "organization" {
			t.Fatalf("unexpected request body: %#v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(environmentRoleFixture))
	}))
	defer server.Close()

	client, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	role, err := client.CreateEnvironmentRole(context.Background(), &EnvironmentRoleCreateRequest{
		Slug:             "admin",
		Name:             "Admin",
		Description:      "Can manage all resources",
		ResourceTypeSlug: "organization",
	})
	if err != nil {
		t.Fatalf("CreateEnvironmentRole returned error: %v", err)
	}
	if role.ID != "role_01JYQ5B9Q6ZP8K4R2T1V0X9ABC" || role.Type != "EnvironmentRole" {
		t.Fatalf("unexpected role response: %#v", role)
	}
}

func TestEnvironmentRolesClientGetAndUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authorization/roles/admin" {
			t.Fatalf("expected /authorization/roles/admin, got %s", r.URL.Path)
		}

		switch r.Method {
		case http.MethodGet:
		case http.MethodPatch:
			var body EnvironmentRoleUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			if body.Name != "Administrator" || body.Description != "Updated" {
				t.Fatalf("unexpected request body: %#v", body)
			}
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(environmentRoleFixture))
	}))
	defer server.Close()

	client, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	role, err := client.GetEnvironmentRole(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetEnvironmentRole returned error: %v", err)
	}
	if role.Slug != "admin" {
		t.Fatalf("unexpected role slug: %s", role.Slug)
	}

	role, err = client.UpdateEnvironmentRole(context.Background(), "admin", &EnvironmentRoleUpdateRequest{
		Name:        "Administrator",
		Description: "Updated",
	})
	if err != nil {
		t.Fatalf("UpdateEnvironmentRole returned error: %v", err)
	}
	if role.Slug != "admin" {
		t.Fatalf("unexpected role slug: %s", role.Slug)
	}
}

func TestEnvironmentRolesClientListPaginationAndGetByID(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/authorization/roles" {
			t.Fatalf("expected /authorization/roles, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "100" {
			t.Fatalf("expected limit=100, got %s", r.URL.Query().Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			if r.URL.Query().Get("after") != "" {
				t.Fatalf("did not expect after on first page, got %s", r.URL.Query().Get("after"))
			}
			_, _ = w.Write([]byte(`{
				"data": [],
				"list_metadata": {"after": "cursor_1"}
			}`))
			return
		}

		if r.URL.Query().Get("after") != "cursor_1" {
			t.Fatalf("expected after=cursor_1, got %s", r.URL.Query().Get("after"))
		}
		_, _ = w.Write([]byte(`{
			"data": [` + environmentRoleFixture + `],
			"list_metadata": {}
		}`))
	}))
	defer server.Close()

	client, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	role, err := client.GetEnvironmentRoleByID(context.Background(), "role_01JYQ5B9Q6ZP8K4R2T1V0X9ABC")
	if err != nil {
		t.Fatalf("GetEnvironmentRoleByID returned error: %v", err)
	}
	if role.Slug != "admin" {
		t.Fatalf("unexpected role slug: %s", role.Slug)
	}
	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}
}

func TestEnvironmentRolesClientPermissions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authorization/roles/admin/permissions" {
			t.Fatalf("expected /authorization/roles/admin/permissions, got %s", r.URL.Path)
		}

		switch r.Method {
		case http.MethodPost:
			var body AddPermissionRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode add request body: %v", err)
			}
			if body.Slug != "billing:read" {
				t.Fatalf("unexpected add request body: %#v", body)
			}
		case http.MethodPut:
			var body EnvironmentRolePermissionsRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode set request body: %v", err)
			}
			if len(body.Permissions) != 2 || body.Permissions[0] != "billing:read" || body.Permissions[1] != "billing:write" {
				t.Fatalf("unexpected set request body: %#v", body)
			}
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(environmentRoleFixture))
	}))
	defer server.Close()

	client, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if _, err := client.AddEnvironmentRolePermission(context.Background(), "admin", "billing:read"); err != nil {
		t.Fatalf("AddEnvironmentRolePermission returned error: %v", err)
	}
	if _, err := client.SetEnvironmentRolePermissions(context.Background(), "admin", []string{"billing:read", "billing:write"}); err != nil {
		t.Fatalf("SetEnvironmentRolePermissions returned error: %v", err)
	}
}
