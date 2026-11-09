package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/osodevops/terraform-provider-workos/internal/client"
)

type jwtTemplateMockAPI struct {
	mu       sync.Mutex
	template client.JWTTemplate
	puts     int
}

func newJWTTemplateMockAPI(t *testing.T, content string) (*jwtTemplateMockAPI, *httptest.Server) {
	t.Helper()

	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	api := &jwtTemplateMockAPI{
		template: client.JWTTemplate{Object: "jwt_template", Content: content, CreatedAt: now, UpdatedAt: now},
	}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	return api, server
}

func (a *jwtTemplateMockAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if r.URL.Path != "/user_management/jwt_template" {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(a.template)
	case http.MethodPut:
		var req client.JWTTemplateUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"message":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(strings.TrimSpace(req.Content), "{") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"message":"Invalid JWT template","code":"invalid_jwt_template"}`))
			return
		}
		a.puts++
		a.template.Content = req.Content
		a.template.UpdatedAt = a.template.UpdatedAt.Add(time.Hour)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(a.template)
	default:
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}
}

func (a *jwtTemplateMockAPI) content() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.template.Content
}
