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

func TestCreateRedirectURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/user_management/redirect_uris" {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}

		var req RedirectURICreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.URI != "https://acme.example.com/api/auth/callback" {
			t.Errorf("unexpected uri %q", req.URI)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"object": "redirect_uri",
			"id": "redir_01EHZNVPK3SFK441A1RGBFSHRT",
			"uri": "https://acme.example.com/api/auth/callback",
			"default": false,
			"created_at": "2026-01-15T12:00:00.000Z",
			"updated_at": "2026-01-15T12:00:00.000Z"
		}`))
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	got, err := workosClient.CreateRedirectURI(context.Background(), &RedirectURICreateRequest{
		URI: "https://acme.example.com/api/auth/callback",
	})
	if err != nil {
		t.Fatalf("CreateRedirectURI: %v", err)
	}
	if got.ID != "redir_01EHZNVPK3SFK441A1RGBFSHRT" {
		t.Fatalf("unexpected id %q", got.ID)
	}
	if got.Default {
		t.Fatal("expected default to be false")
	}
}

func TestGetRedirectURIUsesList(t *testing.T) {
	var pages int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/user_management/redirect_uris" {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}

		pages++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("after") == "" {
			_, _ = w.Write([]byte(`{
				"object": "list",
				"data": [{
					"object": "redirect_uri",
					"id": "redir_page1",
					"uri": "https://one.example.com/callback",
					"default": true,
					"created_at": "2026-01-15T12:00:00.000Z",
					"updated_at": "2026-01-15T12:00:00.000Z"
				}],
				"list_metadata": {"after": "redir_page1"}
			}`))
			return
		}

		_, _ = w.Write([]byte(`{
			"object": "list",
			"data": [{
				"object": "redirect_uri",
				"id": "redir_page2",
				"uri": "https://two.example.com/callback",
				"default": false,
				"created_at": "2026-01-15T12:00:00.000Z",
				"updated_at": "2026-01-15T12:00:00.000Z"
			}],
			"list_metadata": {}
		}`))
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	got, err := workosClient.GetRedirectURI(context.Background(), "redir_page2")
	if err != nil {
		t.Fatalf("GetRedirectURI: %v", err)
	}
	if got.URI != "https://two.example.com/callback" {
		t.Fatalf("unexpected uri %q", got.URI)
	}
	if pages != 2 {
		t.Fatalf("expected two list pages, got %d", pages)
	}

	byURI, err := workosClient.GetRedirectURIByURI(context.Background(), "https://one.example.com/callback")
	if err != nil {
		t.Fatalf("GetRedirectURIByURI: %v", err)
	}
	if byURI.ID != "redir_page1" {
		t.Fatalf("unexpected id %q", byURI.ID)
	}

	_, err = workosClient.GetRedirectURI(context.Background(), "redir_missing")
	if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDeleteRedirectURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/user_management/redirect_uris/redir_01EHZNVPK3SFK441A1RGBFSHRT" {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if err := workosClient.DeleteRedirectURI(context.Background(), "redir_01EHZNVPK3SFK441A1RGBFSHRT"); err != nil {
		t.Fatalf("DeleteRedirectURI: %v", err)
	}
}

func TestIsExistingRedirectURI(t *testing.T) {
	conflict := &APIError{StatusCode: http.StatusConflict, Message: "conflict"}
	if !IsExistingRedirectURI(conflict) {
		t.Fatal("expected conflict to be treated as existing")
	}

	already := &APIError{StatusCode: http.StatusBadRequest, Code: "entity_already_exists", Message: "Redirect URI already exists"}
	if !IsExistingRedirectURI(already) {
		t.Fatal("expected already-exists message to be treated as existing")
	}

	invalid := &APIError{StatusCode: http.StatusBadRequest, Message: "invalid uri"}
	if IsExistingRedirectURI(invalid) {
		t.Fatal("did not expect invalid uri to be treated as existing")
	}

	if !IsRedirectURIID("redir_01ABC") {
		t.Fatal("expected redir_ prefix to match")
	}
	if IsRedirectURIID("https://example.com/callback") {
		t.Fatal("did not expect a URI to match a redirect id")
	}
}
