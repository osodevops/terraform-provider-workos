// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/url"
)

// DirectoryCreateRequest represents the request to create a directory
type DirectoryCreateRequest struct {
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
}

// DirectoryUpdateRequest represents the request to update a directory
type DirectoryUpdateRequest struct {
	Name string `json:"name,omitempty"`
}

// DirectoryListResponse represents the response from listing directories
type DirectoryListResponse struct {
	Data         []Directory  `json:"data"`
	ListMetadata ListMetadata `json:"list_metadata"`
}

// DirectoryUserListResponse represents the response from listing directory users
type DirectoryUserListResponse struct {
	Data         []DirectoryUser `json:"data"`
	ListMetadata ListMetadata    `json:"list_metadata"`
}

// DirectoryGroupListResponse represents the response from listing directory groups
type DirectoryGroupListResponse struct {
	Data         []DirectoryGroup `json:"data"`
	ListMetadata ListMetadata     `json:"list_metadata"`
}

// CreateDirectory creates a new directory
func (c *Client) CreateDirectory(ctx context.Context, req *DirectoryCreateRequest) (*Directory, error) {
	var dir Directory
	err := c.Post(ctx, "/directories", req, &dir)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	return &dir, nil
}

// GetDirectory retrieves a directory by ID
func (c *Client) GetDirectory(ctx context.Context, id string) (*Directory, error) {
	var dir Directory
	err := c.Get(ctx, "/directories/"+id, &dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory: %w", err)
	}
	return &dir, nil
}

// UpdateDirectory updates an existing directory
func (c *Client) UpdateDirectory(ctx context.Context, id string, req *DirectoryUpdateRequest) (*Directory, error) {
	var dir Directory
	err := c.Put(ctx, "/directories/"+id, req, &dir)
	if err != nil {
		return nil, fmt.Errorf("failed to update directory: %w", err)
	}
	return &dir, nil
}

// DeleteDirectory deletes a directory by ID
func (c *Client) DeleteDirectory(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/directories/"+id)
	if err != nil {
		return fmt.Errorf("failed to delete directory: %w", err)
	}
	return nil
}

// ListDirectories lists all directories, optionally filtered by organization
func (c *Client) ListDirectories(ctx context.Context, organizationID string) (*DirectoryListResponse, error) {
	path := "/directories"
	if organizationID != "" {
		params := url.Values{}
		params.Set("organization_id", organizationID)
		path = path + "?" + params.Encode()
	}

	var resp DirectoryListResponse
	err := c.Get(ctx, path, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list directories: %w", err)
	}
	return &resp, nil
}

// GetDirectoryByOrganization finds a directory by organization ID
func (c *Client) GetDirectoryByOrganization(ctx context.Context, organizationID string) (*Directory, error) {
	resp, err := c.ListDirectories(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, &APIError{
			StatusCode: 404,
			Message:    fmt.Sprintf("no directory found for organization %s", organizationID),
		}
	}

	return &resp.Data[0], nil
}

// ListDirectoryUsers lists users in a directory
func (c *Client) ListDirectoryUsers(ctx context.Context, directoryID string) (*DirectoryUserListResponse, error) {
	params := url.Values{}
	params.Set("directory", directoryID)

	var resp DirectoryUserListResponse
	err := c.Get(ctx, "/directory_users?"+params.Encode(), &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory users: %w", err)
	}
	return &resp, nil
}

// GetDirectoryUser retrieves a directory user by ID
func (c *Client) GetDirectoryUser(ctx context.Context, id string) (*DirectoryUser, error) {
	var user DirectoryUser
	err := c.Get(ctx, "/directory_users/"+id, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory user: %w", err)
	}
	return &user, nil
}

// GetDirectoryUserByEmail finds a directory user by email
func (c *Client) GetDirectoryUserByEmail(ctx context.Context, directoryID, email string) (*DirectoryUser, error) {
	params := url.Values{}
	params.Set("directory", directoryID)
	params.Set("emails", email)

	var resp DirectoryUserListResponse
	err := c.Get(ctx, "/directory_users?"+params.Encode(), &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to search directory users: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, &APIError{
			StatusCode: 404,
			Message:    fmt.Sprintf("no user found with email %s in directory %s", email, directoryID),
		}
	}

	return &resp.Data[0], nil
}

// ListDirectoryGroups lists groups in a directory
func (c *Client) ListDirectoryGroups(ctx context.Context, directoryID string) (*DirectoryGroupListResponse, error) {
	params := url.Values{}
	params.Set("directory", directoryID)

	var resp DirectoryGroupListResponse
	err := c.Get(ctx, "/directory_groups?"+params.Encode(), &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory groups: %w", err)
	}
	return &resp, nil
}

// GetDirectoryGroup retrieves a directory group by ID
func (c *Client) GetDirectoryGroup(ctx context.Context, id string) (*DirectoryGroup, error) {
	var group DirectoryGroup
	err := c.Get(ctx, "/directory_groups/"+id, &group)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory group: %w", err)
	}
	return &group, nil
}

// GetDirectoryGroupByName finds a directory group by name
func (c *Client) GetDirectoryGroupByName(ctx context.Context, directoryID, name string) (*DirectoryGroup, error) {
	params := url.Values{}
	params.Set("directory", directoryID)

	var resp DirectoryGroupListResponse
	err := c.Get(ctx, "/directory_groups?"+params.Encode(), &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to search directory groups: %w", err)
	}

	// Filter by name since API doesn't support name filter
	for _, group := range resp.Data {
		if group.Name == name {
			return &group, nil
		}
	}

	return nil, &APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("no group found with name %s in directory %s", name, directoryID),
	}
}
