package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const jwtTemplateContent = `{"urn:px:roles": {{organization_membership.roles}}}`

func TestUpdateJWTTemplatePutsContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/user_management/jwt_template" {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}

		var req JWTTemplateUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.Content != jwtTemplateContent {
			t.Errorf("unexpected content %q", req.Content)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"object":     "jwt_template",
			"content":    req.Content,
			"created_at": "2026-01-15T12:00:00.000Z",
			"updated_at": "2026-01-16T12:00:00.000Z",
		})
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	got, err := workosClient.UpdateJWTTemplate(context.Background(), &JWTTemplateUpdateRequest{Content: jwtTemplateContent})
	if err != nil {
		t.Fatalf("UpdateJWTTemplate: %v", err)
	}
	if got.Content != jwtTemplateContent {
		t.Fatalf("unexpected content %q", got.Content)
	}
	if got.UpdatedAt.Day() != 16 {
		t.Fatalf("unexpected updated_at %v", got.UpdatedAt)
	}
}

func TestUpdateJWTTemplateSurfacesValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Invalid JWT template","code":"invalid_jwt_template"}`))
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = workosClient.UpdateJWTTemplate(context.Background(), &JWTTemplateUpdateRequest{Content: "{"})
	if err == nil {
		t.Fatal("expected an error for an invalid template")
	}
}

func TestGetJWTTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/user_management/jwt_template" {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"object": "jwt_template",
			"content": "{}",
			"created_at": "2026-01-15T12:00:00.000Z",
			"updated_at": "2026-01-15T12:00:00.000Z"
		}`))
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	got, err := workosClient.GetJWTTemplate(context.Background())
	if err != nil {
		t.Fatalf("GetJWTTemplate: %v", err)
	}
	if got.Content != "{}" {
		t.Fatalf("unexpected content %q", got.Content)
	}
}
