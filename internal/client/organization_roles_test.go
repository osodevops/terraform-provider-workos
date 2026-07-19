// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const organizationRoleFixture = `{
  "id": "role_01JYQ5B9Q6ZP8K4R2T1V0X9ABC",
  "object": "role",
  "slug": "org-admin",
  "name": "Admin",
  "description": "Can manage all resources",
  "type": "OrganizationRole",
  "resource_type_slug": "organization",
  "permissions": [],
  "created_at": "2026-01-15T12:00:00.000Z",
  "updated_at": "2026-01-15T12:00:00.000Z"
}`

func TestOrganizationRoleMutationsAreSerializedPerOrganization(t *testing.T) {
	var active atomic.Int32
	var overlap atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := active.Add(1); got > 1 {
			overlap.Store(true)
		}
		time.Sleep(10 * time.Millisecond)
		active.Add(-1)

		if r.Method != http.MethodDelete {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(organizationRoleFixture))
		}
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	const mutationCount = 12
	start := make(chan struct{})
	errorsCh := make(chan error, mutationCount)
	var wg sync.WaitGroup
	for i := 0; i < mutationCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start

			var mutationErr error
			switch index % 3 {
			case 0:
				_, mutationErr = workosClient.CreateOrganizationRole(context.Background(), "org_shared", &OrganizationRoleCreateRequest{
					Slug: "org-admin",
					Name: "Admin",
				})
			case 1:
				_, mutationErr = workosClient.UpdateOrganizationRole(context.Background(), "org_shared", "org-admin", &OrganizationRoleUpdateRequest{
					Name: "Admin",
				})
			case 2:
				mutationErr = workosClient.DeleteOrganizationRole(context.Background(), "org_shared", "org-admin")
			}
			if mutationErr != nil {
				errorsCh <- mutationErr
			}
		}(i)
	}

	close(start)
	wg.Wait()
	close(errorsCh)

	for mutationErr := range errorsCh {
		t.Errorf("organization role mutation failed: %v", mutationErr)
	}
	if overlap.Load() {
		t.Fatal("organization role mutations for the same organization overlapped")
	}

	workosClient.organizationRoleMutations.mu.Lock()
	defer workosClient.organizationRoleMutations.mu.Unlock()
	if got := len(workosClient.organizationRoleMutations.entries); got != 0 {
		t.Fatalf("expected lock entries to be released, got %d", got)
	}
}

func TestOrganizationRoleMutationsForDifferentOrganizationsRemainConcurrent(t *testing.T) {
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondEntered := make(chan struct{})
	var releaseFirstOnce sync.Once
	release := func() { releaseFirstOnce.Do(func() { close(releaseFirst) }) }
	defer release()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authorization/organizations/org_first/roles":
			close(firstEntered)
			<-releaseFirst
		case "/authorization/organizations/org_second/roles":
			close(secondEntered)
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(organizationRoleFixture))
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	errorsCh := make(chan error, 2)
	create := func(orgID string) {
		_, createErr := workosClient.CreateOrganizationRole(context.Background(), orgID, &OrganizationRoleCreateRequest{
			Slug: "org-admin",
			Name: "Admin",
		})
		errorsCh <- createErr
	}

	go create("org_first")
	select {
	case <-firstEntered:
	case <-time.After(time.Second):
		t.Fatal("first organization mutation did not reach the server")
	}

	go create("org_second")
	select {
	case <-secondEntered:
	case <-time.After(time.Second):
		t.Fatal("second organization mutation was blocked by a different organization")
	}
	release()

	for i := 0; i < 2; i++ {
		if createErr := <-errorsCh; createErr != nil {
			t.Errorf("CreateOrganizationRole returned error: %v", createErr)
		}
	}
}

func TestOrganizationRoleMutationLockHonorsContextCancellation(t *testing.T) {
	requestEntered := make(chan struct{})
	releaseRequest := make(chan struct{})
	var requestCount atomic.Int32
	var releaseRequestOnce sync.Once
	release := func() { releaseRequestOnce.Do(func() { close(releaseRequest) }) }
	defer release()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		close(requestEntered)
		<-releaseRequest
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(organizationRoleFixture))
	}))
	defer server.Close()

	workosClient, err := NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	firstDone := make(chan error, 1)
	go func() {
		_, createErr := workosClient.CreateOrganizationRole(context.Background(), "org_shared", &OrganizationRoleCreateRequest{
			Slug: "org-admin",
			Name: "Admin",
		})
		firstDone <- createErr
	}()

	select {
	case <-requestEntered:
	case <-time.After(time.Second):
		t.Fatal("first organization mutation did not reach the server")
	}

	ctx, cancel := context.WithCancel(context.Background())
	secondDone := make(chan error, 1)
	go func() {
		_, createErr := workosClient.CreateOrganizationRole(ctx, "org_shared", &OrganizationRoleCreateRequest{
			Slug: "org-editor",
			Name: "Editor",
		})
		secondDone <- createErr
	}()

	// Give the second request time to wait on the held organization lock.
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case createErr := <-secondDone:
		if !errors.Is(createErr, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", createErr)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled organization mutation remained blocked")
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("canceled mutation reached the server: got %d requests", got)
	}

	release()
	if createErr := <-firstDone; createErr != nil {
		t.Fatalf("first CreateOrganizationRole returned error: %v", createErr)
	}
}
