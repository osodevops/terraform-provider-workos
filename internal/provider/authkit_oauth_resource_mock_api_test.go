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

	"github.com/osodevops/terraform-provider-workos/internal/client"
)

type authKitOAuthResourceMockAPI struct {
	mu        sync.Mutex
	counter   int
	resources map[string]*client.AuthKitOAuthResource
	deleted   []string
}

func newAuthKitOAuthResourceMockAPI(t *testing.T) (*authKitOAuthResourceMockAPI, *httptest.Server) {
	t.Helper()

	api := &authKitOAuthResourceMockAPI{
		resources: make(map[string]*client.AuthKitOAuthResource),
	}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	return api, server
}

func (a *authKitOAuthResourceMockAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	const collection = "/user_management/authkit_oauth_resources"
	switch {
	case r.URL.Path == collection && r.Method == http.MethodPost:
		a.create(w, r)
	case r.URL.Path == collection && r.Method == http.MethodGet:
		a.list(w)
	case strings.HasPrefix(r.URL.Path, collection+"/") && r.Method == http.MethodDelete:
		a.delete(w, strings.TrimPrefix(r.URL.Path, collection+"/"))
	default:
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}
}

func (a *authKitOAuthResourceMockAPI) create(w http.ResponseWriter, r *http.Request) {
	var req client.AuthKitOAuthResourceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"invalid json"}`, http.StatusBadRequest)
		return
	}

	for _, existing := range a.resources {
		if existing.URI == req.URI {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"message":"Resource indicator already exists","code":"entity_already_exists"}`))
			return
		}
	}

	a.counter++
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	item := &client.AuthKitOAuthResource{
		ID:        fmt.Sprintf("authkit_oauth_resource_%08d", a.counter),
		Object:    "authkit_oauth_resource",
		URI:       req.URI,
		Default:   req.Default != nil && *req.Default,
		CreatedAt: now,
		UpdatedAt: now,
	}
	a.resources[item.ID] = item

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(item)
}

func (a *authKitOAuthResourceMockAPI) list(w http.ResponseWriter) {
	data := make([]client.AuthKitOAuthResource, 0, len(a.resources))
	for _, item := range a.resources {
		data = append(data, *item)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(client.AuthKitOAuthResourceListResponse{Data: data})
}

func (a *authKitOAuthResourceMockAPI) delete(w http.ResponseWriter, id string) {
	if _, ok := a.resources[id]; !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	delete(a.resources, id)
	a.deleted = append(a.deleted, id)
	w.WriteHeader(http.StatusOK)
}

func (a *authKitOAuthResourceMockAPI) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.resources)
}

func (a *authKitOAuthResourceMockAPI) deletions() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.deleted)
}
