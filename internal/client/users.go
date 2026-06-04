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
	err := c.Get(ctx, "/user_management/users/"+url.PathEscape(id), &user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// UpdateUser updates an existing user
func (c *Client) UpdateUser(ctx context.Context, id string, req *UserUpdateRequest) (*User, error) {
	var user User
	err := c.Put(ctx, "/user_management/users/"+url.PathEscape(id), req, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return &user, nil
}

// DeleteUser deletes a user by ID
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/user_management/users/"+url.PathEscape(id))
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListUsers lists all users with optional filters
func (c *Client) ListUsers(ctx context.Context, email string, organizationID string) (*UserListResponse, error) {
	var all UserListResponse
	params := url.Values{}
	if email != "" {
		params.Set("email", email)
	}
	if organizationID != "" {
		params.Set("organization_id", organizationID)
	}
	applyDefaultPagination(params)

	for {
		var page UserListResponse
		err := c.Get(ctx, pathWithQuery("/user_management/users", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
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

// GetUserByExternalID retrieves a user by external ID
func (c *Client) GetUserByExternalID(ctx context.Context, externalID string) (*User, error) {
	var user User
	err := c.Get(ctx, "/user_management/users/external_id/"+url.PathEscape(externalID), &user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by external ID: %w", err)
	}
	return &user, nil
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
	err := c.Get(ctx, "/user_management/organization_memberships/"+url.PathEscape(id), &membership)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization membership: %w", err)
	}
	return &membership, nil
}

// UpdateOrganizationMembership updates an organization membership by ID
func (c *Client) UpdateOrganizationMembership(ctx context.Context, id string, req *OrganizationMembershipUpdateRequest) (*OrganizationMembership, error) {
	var membership OrganizationMembership
	err := c.Put(ctx, "/user_management/organization_memberships/"+url.PathEscape(id), req, &membership)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization membership: %w", err)
	}
	return &membership, nil
}

// DeleteOrganizationMembership deletes an organization membership by ID
func (c *Client) DeleteOrganizationMembership(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/user_management/organization_memberships/"+url.PathEscape(id))
	if err != nil {
		return fmt.Errorf("failed to delete organization membership: %w", err)
	}
	return nil
}

// ListOrganizationMemberships lists memberships with optional filters
func (c *Client) ListOrganizationMemberships(ctx context.Context, userID string, organizationID string) (*OrganizationMembershipListResponse, error) {
	var all OrganizationMembershipListResponse
	params := url.Values{}
	if userID != "" {
		params.Set("user_id", userID)
	}
	if organizationID != "" {
		params.Set("organization_id", organizationID)
	}
	applyDefaultPagination(params)

	for {
		var page OrganizationMembershipListResponse
		err := c.Get(ctx, pathWithQuery("/user_management/organization_memberships", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list organization memberships: %w", err)
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

// DeactivateOrganizationMembership deactivates a membership
func (c *Client) DeactivateOrganizationMembership(ctx context.Context, id string) (*OrganizationMembership, error) {
	var membership OrganizationMembership
	err := c.Put(ctx, "/user_management/organization_memberships/"+url.PathEscape(id)+"/deactivate", nil, &membership)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate organization membership: %w", err)
	}
	return &membership, nil
}

// ReactivateOrganizationMembership reactivates a membership
func (c *Client) ReactivateOrganizationMembership(ctx context.Context, id string) (*OrganizationMembership, error) {
	var membership OrganizationMembership
	err := c.Put(ctx, "/user_management/organization_memberships/"+url.PathEscape(id)+"/reactivate", nil, &membership)
	if err != nil {
		return nil, fmt.Errorf("failed to reactivate organization membership: %w", err)
	}
	return &membership, nil
}
