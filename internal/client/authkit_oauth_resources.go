package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	authKitOAuthResourcesPath  = "/user_management/authkit_oauth_resources"
	authKitOAuthResourcePrefix = "authkit_oauth_resource_"
)

func (c *Client) CreateAuthKitOAuthResource(ctx context.Context, req *AuthKitOAuthResourceCreateRequest) (*AuthKitOAuthResource, error) {
	var resource AuthKitOAuthResource
	if err := c.Post(ctx, authKitOAuthResourcesPath, req, &resource); err != nil {
		return nil, fmt.Errorf("failed to create MCP resource indicator: %w", err)
	}
	return &resource, nil
}

func (c *Client) GetAuthKitOAuthResource(ctx context.Context, id string) (*AuthKitOAuthResource, error) {
	return c.findAuthKitOAuthResource(ctx, func(item AuthKitOAuthResource) bool {
		return item.ID == id
	})
}

func (c *Client) GetAuthKitOAuthResourceByURI(ctx context.Context, uri string) (*AuthKitOAuthResource, error) {
	return c.findAuthKitOAuthResource(ctx, func(item AuthKitOAuthResource) bool {
		return item.URI == uri
	})
}

func (c *Client) DeleteAuthKitOAuthResource(ctx context.Context, id string) error {
	if err := c.Delete(ctx, authKitOAuthResourcesPath+"/"+url.PathEscape(id)); err != nil {
		return fmt.Errorf("failed to delete MCP resource indicator: %w", err)
	}
	return nil
}

func (c *Client) ListAuthKitOAuthResources(ctx context.Context) (*AuthKitOAuthResourceListResponse, error) {
	var all AuthKitOAuthResourceListResponse
	params := url.Values{}
	applyDefaultPagination(params)

	for {
		var page AuthKitOAuthResourceListResponse
		if err := c.Get(ctx, pathWithQuery(authKitOAuthResourcesPath, params), &page); err != nil {
			return nil, fmt.Errorf("failed to list MCP resource indicators: %w", err)
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

func (c *Client) findAuthKitOAuthResource(ctx context.Context, match func(AuthKitOAuthResource) bool) (*AuthKitOAuthResource, error) {
	list, err := c.ListAuthKitOAuthResources(ctx)
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

func IsAuthKitOAuthResourceID(id string) bool {
	return strings.HasPrefix(id, authKitOAuthResourcePrefix)
}

func IsExistingAuthKitOAuthResource(err error) bool {
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
