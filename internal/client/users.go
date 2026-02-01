// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/url"
)

// UserListResponse represents the response from listing users
type UserListResponse struct {
	Data         []User       `json:"data"`
	ListMetadata ListMetadata `json:"list_metadata"`
}

// OrganizationMembershipListResponse represents the response from listing memberships
type OrganizationMembershipListResponse struct {
	Data         []OrganizationMembership `json:"data"`
	ListMetadata ListMetadata             `json:"list_metadata"`
}

// CreateUser creates a new user
func (c *Client) CreateUser(ctx context.Context, req *UserCreateRequest) (*User, error) {
	var user User
	err := c.Post(ctx, "/user_management/users", req, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// GetUser retrieves a user by ID
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var user User
	err := c.Get(ctx, "/user_management/users/"+id, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// UpdateUser updates an existing user
func (c *Client) UpdateUser(ctx context.Context, id string, req *UserUpdateRequest) (*User, error) {
	var user User
	err := c.Put(ctx, "/user_management/users/"+id, req, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return &user, nil
}

// DeleteUser deletes a user by ID
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/user_management/users/"+id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListUsers lists all users with optional filters
func (c *Client) ListUsers(ctx context.Context, email string, organizationID string) (*UserListResponse, error) {
	path := "/user_management/users"
	params := url.Values{}
	if email != "" {
		params.Set("email", email)
	}
	if organizationID != "" {
		params.Set("organization_id", organizationID)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var resp UserListResponse
	err := c.Get(ctx, path, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return &resp, nil
}

// GetUserByEmail retrieves a user by email
func (c *Client) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	resp, err := c.ListUsers(ctx, email, "")
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, &APIError{StatusCode: 404, Message: "user not found"}
	}
	return &resp.Data[0], nil
}

// CreateOrganizationMembership creates a new organization membership
func (c *Client) CreateOrganizationMembership(ctx context.Context, req *OrganizationMembershipCreateRequest) (*OrganizationMembership, error) {
	var membership OrganizationMembership
	err := c.Post(ctx, "/user_management/organization_memberships", req, &membership)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization membership: %w", err)
	}
	return &membership, nil
}

// GetOrganizationMembership retrieves an organization membership by ID
func (c *Client) GetOrganizationMembership(ctx context.Context, id string) (*OrganizationMembership, error) {
	var membership OrganizationMembership
	err := c.Get(ctx, "/user_management/organization_memberships/"+id, &membership)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization membership: %w", err)
	}
	return &membership, nil
}

// DeleteOrganizationMembership deletes an organization membership by ID
func (c *Client) DeleteOrganizationMembership(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/user_management/organization_memberships/"+id)
	if err != nil {
		return fmt.Errorf("failed to delete organization membership: %w", err)
	}
	return nil
}

// ListOrganizationMemberships lists memberships with optional filters
func (c *Client) ListOrganizationMemberships(ctx context.Context, userID string, organizationID string) (*OrganizationMembershipListResponse, error) {
	path := "/user_management/organization_memberships"
	params := url.Values{}
	if userID != "" {
		params.Set("user_id", userID)
	}
	if organizationID != "" {
		params.Set("organization_id", organizationID)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var resp OrganizationMembershipListResponse
	err := c.Get(ctx, path, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list organization memberships: %w", err)
	}
	return &resp, nil
}

// DeactivateOrganizationMembership deactivates a membership
func (c *Client) DeactivateOrganizationMembership(ctx context.Context, id string) (*OrganizationMembership, error) {
	var membership OrganizationMembership
	err := c.Put(ctx, "/user_management/organization_memberships/"+id+"/deactivate", nil, &membership)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate organization membership: %w", err)
	}
	return &membership, nil
}

// ReactivateOrganizationMembership reactivates a membership
func (c *Client) ReactivateOrganizationMembership(ctx context.Context, id string) (*OrganizationMembership, error) {
	var membership OrganizationMembership
	err := c.Put(ctx, "/user_management/organization_memberships/"+id+"/reactivate", nil, &membership)
	if err != nil {
		return nil, fmt.Errorf("failed to reactivate organization membership: %w", err)
	}
	return &membership, nil
}
