// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const redirectURIPrefix = "redir_"

// RedirectURI represents an AuthKit redirect URI on the User Management API.
type RedirectURI struct {
	ID        string    `json:"id"`
	Object    string    `json:"object"`
	URI       string    `json:"uri"`
	Default   bool      `json:"default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RedirectURICreateRequest represents the request to register a redirect URI.
type RedirectURICreateRequest struct {
	URI string `json:"uri"`
}

// RedirectURIListResponse represents the response from listing redirect URIs.
type RedirectURIListResponse struct {
	Data         []RedirectURI `json:"data"`
	ListMetadata ListMetadata  `json:"list_metadata"`
}

// CreateRedirectURI registers a redirect URI on the AuthKit application bound
// to the API key.
func (c *Client) CreateRedirectURI(ctx context.Context, req *RedirectURICreateRequest) (*RedirectURI, error) {
	var redirectURI RedirectURI
	err := c.Post(ctx, "/user_management/redirect_uris", req, &redirectURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create redirect URI: %w", err)
	}
	return &redirectURI, nil
}

// GetRedirectURI retrieves a redirect URI by ID by scanning the list endpoint.
// WorkOS does not document a GET-by-id for this resource.
func (c *Client) GetRedirectURI(ctx context.Context, id string) (*RedirectURI, error) {
	return c.findRedirectURI(ctx, func(item RedirectURI) bool {
		return item.ID == id
	})
}

// GetRedirectURIByURI retrieves a redirect URI by exact URI match.
func (c *Client) GetRedirectURIByURI(ctx context.Context, uri string) (*RedirectURI, error) {
	return c.findRedirectURI(ctx, func(item RedirectURI) bool {
		return item.URI == uri
	})
}

// DeleteRedirectURI deletes a redirect URI by ID.
func (c *Client) DeleteRedirectURI(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/user_management/redirect_uris/"+url.PathEscape(id))
	if err != nil {
		return fmt.Errorf("failed to delete redirect URI: %w", err)
	}
	return nil
}

// ListRedirectURIs lists AuthKit redirect URIs, following pagination.
func (c *Client) ListRedirectURIs(ctx context.Context) (*RedirectURIListResponse, error) {
	var all RedirectURIListResponse
	params := url.Values{}
	applyDefaultPagination(params)

	for {
		var page RedirectURIListResponse
		err := c.Get(ctx, pathWithQuery("/user_management/redirect_uris", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list redirect URIs: %w", err)
		}

		all.Data = append(all.Data, page.Data...)
		all.ListMetadata = page.ListMetadata
		if page.ListMetadata.After == "" {
			break
		}
		params.Set("after", page.ListMetadata.After)
	}

	return &all, nil
}

func (c *Client) findRedirectURI(ctx context.Context, match func(RedirectURI) bool) (*RedirectURI, error) {
	list, err := c.ListRedirectURIs(ctx)
	if err != nil {
		return nil, err
	}

	for i := range list.Data {
		if match(list.Data[i]) {
			return &list.Data[i], nil
		}
	}

	return nil, &APIError{
		StatusCode: http.StatusNotFound,
		Message:    "The requested resource was not found",
	}
}

// IsRedirectURIID returns true when id looks like a WorkOS redirect URI id.
func IsRedirectURIID(id string) bool {
	return strings.HasPrefix(id, redirectURIPrefix)
}

// IsExistingRedirectURI returns true when create failed because the URI is
// already registered on the AuthKit application.
func IsExistingRedirectURI(err error) bool {
	if IsConflict(err) {
		return true
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	haystack := strings.ToLower(apiErr.Code + " " + apiErr.Message)
	return strings.Contains(haystack, "already")
}
