package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Group represents a WorkOS group within an organization.
type Group struct {
	ID             string    `json:"id"`
	Object         string    `json:"object"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Description    *string   `json:"description,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// GroupCreateRequest represents the request to create a group.
type GroupCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// GroupUpdateRequest represents the request to update a group.
type GroupUpdateRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// GroupMembershipCreateRequest represents the request to add a membership to a group.
type GroupMembershipCreateRequest struct {
	OrganizationMembershipID string `json:"organization_membership_id"`
}

// GroupListResponse represents the response from listing groups.
type GroupListResponse struct {
	Data         []Group      `json:"data"`
	ListMetadata ListMetadata `json:"list_metadata"`
}

// GroupMembershipListResponse represents the response from listing group memberships.
type GroupMembershipListResponse struct {
	Data         []OrganizationMembership `json:"data"`
	ListMetadata ListMetadata             `json:"list_metadata"`
}

// CreateGroup creates a group in an organization.
func (c *Client) CreateGroup(ctx context.Context, organizationID string, req *GroupCreateRequest) (*Group, error) {
	var group Group
	err := c.Post(ctx, fmt.Sprintf("/organizations/%s/groups", url.PathEscape(organizationID)), req, &group)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	return &group, nil
}

// GetGroup retrieves a group by ID within an organization.
func (c *Client) GetGroup(ctx context.Context, organizationID, groupID string) (*Group, error) {
	var group Group
	err := c.Get(ctx, fmt.Sprintf("/organizations/%s/groups/%s", url.PathEscape(organizationID), url.PathEscape(groupID)), &group)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	return &group, nil
}

// UpdateGroup updates a group within an organization.
func (c *Client) UpdateGroup(ctx context.Context, organizationID, groupID string, req *GroupUpdateRequest) (*Group, error) {
	var group Group
	err := c.Patch(ctx, fmt.Sprintf("/organizations/%s/groups/%s", url.PathEscape(organizationID), url.PathEscape(groupID)), req, &group)
	if err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}
	return &group, nil
}

// DeleteGroup deletes a group from an organization.
func (c *Client) DeleteGroup(ctx context.Context, organizationID, groupID string) error {
	err := c.Delete(ctx, fmt.Sprintf("/organizations/%s/groups/%s", url.PathEscape(organizationID), url.PathEscape(groupID)))
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	return nil
}

// ListGroups lists all groups in an organization.
func (c *Client) ListGroups(ctx context.Context, organizationID string) (*GroupListResponse, error) {
	var all GroupListResponse
	params := url.Values{}
	applyDefaultPagination(params)
	path := fmt.Sprintf("/organizations/%s/groups", url.PathEscape(organizationID))

	for {
		var page GroupListResponse
		err := c.Get(ctx, pathWithQuery(path, params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list groups: %w", err)
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

// ListGroupMemberships lists all organization memberships in a group.
func (c *Client) ListGroupMemberships(ctx context.Context, organizationID, groupID string) (*GroupMembershipListResponse, error) {
	var all GroupMembershipListResponse
	params := url.Values{}
	applyDefaultPagination(params)
	path := fmt.Sprintf("/organizations/%s/groups/%s/organization-memberships", url.PathEscape(organizationID), url.PathEscape(groupID))

	for {
		var page GroupMembershipListResponse
		err := c.Get(ctx, pathWithQuery(path, params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list group memberships: %w", err)
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

// AddGroupMembership adds an organization membership to a group.
func (c *Client) AddGroupMembership(ctx context.Context, organizationID, groupID string, req *GroupMembershipCreateRequest) (*Group, error) {
	var group Group
	err := c.Post(ctx, fmt.Sprintf("/organizations/%s/groups/%s/organization-memberships", url.PathEscape(organizationID), url.PathEscape(groupID)), req, &group)
	if err != nil {
		return nil, fmt.Errorf("failed to add group membership: %w", err)
	}
	return &group, nil
}

// DeleteGroupMembership removes an organization membership from a group.
func (c *Client) DeleteGroupMembership(ctx context.Context, organizationID, groupID, organizationMembershipID string) error {
	err := c.Delete(ctx, fmt.Sprintf("/organizations/%s/groups/%s/organization-memberships/%s", url.PathEscape(organizationID), url.PathEscape(groupID), url.PathEscape(organizationMembershipID)))
	if err != nil {
		return fmt.Errorf("failed to delete group membership: %w", err)
	}
	return nil
}
