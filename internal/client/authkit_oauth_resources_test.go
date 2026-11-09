package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateAuthKitOAuthResourceOmitsUnsetDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/user_management/authkit_oauth_resources" {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}

		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "default") {
			t.Errorf("expected default to be omitted, got %s", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"object": "authkit_oauth_resource",
			"id": "authkit_oauth_resource_01EHZNVPK3SFK441A1RGBFSHRT",
			"uri": "https://mcp.example.com",
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

	got, err := workosClient.CreateAuthKitOAuthResource(context.Background(), &AuthKitOAuthResourceCreateRequest{
		URI: "https://mcp.example.com",
	})
	if err != nil {
		t.Fatalf("CreateAuthKitOAuthResource: %v", err)
	}
	if !IsAuthKitOAuthResourceID(got.ID) {
		t.Fatalf("unexpected id %q", got.ID)
	}
}

func TestCreateAuthKitOAuthResourceSendsDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req AuthKitOAuthResourceCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.Default == nil || !*req.Default {
			t.Errorf("expected default true, got %v", req.Default)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"authkit_oauth_resource","id":"authkit_oauth_resource_1","uri":"https://mcp.example.com","default":true,"created_at":"2026-01-15T12:00:00.000Z","updated_at":"2026-01-15T12:00:00.000Z"}`))
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	isDefault := true
	got, err := workosClient.CreateAuthKitOAuthResource(context.Background(), &AuthKitOAuthResourceCreateRequest{
		URI:     "https://mcp.example.com",
		Default: &isDefault,
	})
	if err != nil {
		t.Fatalf("CreateAuthKitOAuthResource: %v", err)
	}
	if !got.Default {
		t.Fatal("expected default to be true")
	}
}

func TestGetAuthKitOAuthResourceFollowsPagination(t *testing.T) {
	var pages int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/user_management/authkit_oauth_resources" {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}

		pages++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("after") == "" {
			_, _ = w.Write([]byte(`{
				"object": "list",
				"data": [{"object":"authkit_oauth_resource","id":"authkit_oauth_resource_page1","uri":"https://one.example.com","default":true,"created_at":"2026-01-15T12:00:00.000Z","updated_at":"2026-01-15T12:00:00.000Z"}],
				"list_metadata": {"after": "authkit_oauth_resource_page1"}
			}`))
			return
		}

		_, _ = w.Write([]byte(`{
			"object": "list",
			"data": [{"object":"authkit_oauth_resource","id":"authkit_oauth_resource_page2","uri":"https://two.example.com","default":false,"created_at":"2026-01-15T12:00:00.000Z","updated_at":"2026-01-15T12:00:00.000Z"}],
			"list_metadata": {}
		}`))
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	got, err := workosClient.GetAuthKitOAuthResource(context.Background(), "authkit_oauth_resource_page2")
	if err != nil {
		t.Fatalf("GetAuthKitOAuthResource: %v", err)
	}
	if got.URI != "https://two.example.com" {
		t.Fatalf("unexpected uri %q", got.URI)
	}
	if pages != 2 {
		t.Fatalf("expected two list pages, got %d", pages)
	}

	byURI, err := workosClient.GetAuthKitOAuthResourceByURI(context.Background(), "https://one.example.com")
	if err != nil {
		t.Fatalf("GetAuthKitOAuthResourceByURI: %v", err)
	}
	if byURI.ID != "authkit_oauth_resource_page1" {
		t.Fatalf("unexpected id %q", byURI.ID)
	}

	_, err = workosClient.GetAuthKitOAuthResource(context.Background(), "authkit_oauth_resource_missing")
	if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDeleteAuthKitOAuthResource(t *testing.T) {
	var deleted string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		deleted = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if err := workosClient.DeleteAuthKitOAuthResource(context.Background(), "authkit_oauth_resource_1"); err != nil {
		t.Fatalf("DeleteAuthKitOAuthResource: %v", err)
	}
	if deleted != "/user_management/authkit_oauth_resources/authkit_oauth_resource_1" {
		t.Fatalf("unexpected path %q", deleted)
	}
}

func TestIsExistingAuthKitOAuthResource(t *testing.T) {
	cases := map[string]struct {
		err  error
		want bool
	}{
		"conflict":         {&APIError{StatusCode: http.StatusConflict}, true},
		"already message":  {&APIError{StatusCode: http.StatusUnprocessableEntity, Message: "Resource already exists"}, true},
		"other validation": {&APIError{StatusCode: http.StatusUnprocessableEntity, Message: "uri must be https"}, false},
		"not an api error": {io.EOF, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := IsExistingAuthKitOAuthResource(tc.err); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
