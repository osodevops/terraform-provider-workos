// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/url"
)

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

// GetDirectory retrieves a directory by ID
func (c *Client) GetDirectory(ctx context.Context, id string) (*Directory, error) {
	var dir Directory
	err := c.Get(ctx, "/directories/"+url.PathEscape(id), &dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory: %w", err)
	}
	return &dir, nil
}

// DeleteDirectory deletes a directory by ID
func (c *Client) DeleteDirectory(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/directories/"+url.PathEscape(id))
	if err != nil {
		return fmt.Errorf("failed to delete directory: %w", err)
	}
	return nil
}

// ListDirectories lists all directories, optionally filtered by organization
func (c *Client) ListDirectories(ctx context.Context, organizationID string) (*DirectoryListResponse, error) {
	var all DirectoryListResponse
	params := url.Values{}
	if organizationID != "" {
		params.Set("organization_id", organizationID)
	}
	applyDefaultPagination(params)

	for {
		var page DirectoryListResponse
		err := c.Get(ctx, pathWithQuery("/directories", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list directories: %w", err)
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
	return c.listDirectoryUsers(ctx, directoryID, "")
}

func (c *Client) listDirectoryUsers(ctx context.Context, directoryID, email string) (*DirectoryUserListResponse, error) {
	var all DirectoryUserListResponse
	params := url.Values{}
	params.Set("directory", directoryID)
	if email != "" {
		params.Set("emails", email)
	}
	applyDefaultPagination(params)

	for {
		var page DirectoryUserListResponse
		err := c.Get(ctx, pathWithQuery("/directory_users", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list directory users: %w", err)
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

// GetDirectoryUser retrieves a directory user by ID
func (c *Client) GetDirectoryUser(ctx context.Context, id string) (*DirectoryUser, error) {
	var user DirectoryUser
	err := c.Get(ctx, "/directory_users/"+url.PathEscape(id), &user)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory user: %w", err)
	}
	return &user, nil
}

// GetDirectoryUserByEmail finds a directory user by email
func (c *Client) GetDirectoryUserByEmail(ctx context.Context, directoryID, email string) (*DirectoryUser, error) {
	resp, err := c.listDirectoryUsers(ctx, directoryID, email)
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
	var all DirectoryGroupListResponse
	params := url.Values{}
	params.Set("directory", directoryID)
	applyDefaultPagination(params)

	for {
		var page DirectoryGroupListResponse
		err := c.Get(ctx, pathWithQuery("/directory_groups", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list directory groups: %w", err)
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

// GetDirectoryGroup retrieves a directory group by ID
func (c *Client) GetDirectoryGroup(ctx context.Context, id string) (*DirectoryGroup, error) {
	var group DirectoryGroup
	err := c.Get(ctx, "/directory_groups/"+url.PathEscape(id), &group)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory group: %w", err)
	}
	return &group, nil
}

// GetDirectoryGroupByName finds a directory group by name
func (c *Client) GetDirectoryGroupByName(ctx context.Context, directoryID, name string) (*DirectoryGroup, error) {
	resp, err := c.ListDirectoryGroups(ctx, directoryID)
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
