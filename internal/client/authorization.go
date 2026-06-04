package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// AuthorizationResource represents a WorkOS FGA authorization resource.
type AuthorizationResource struct {
	ID               string    `json:"id"`
	Object           string    `json:"object"`
	OrganizationID   string    `json:"organization_id"`
	ExternalID       string    `json:"external_id"`
	ResourceTypeSlug string    `json:"resource_type_slug"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	ParentResourceID *string   `json:"parent_resource_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// AuthorizationResourceCreateRequest represents the request to create an authorization resource.
type AuthorizationResourceCreateRequest struct {
	OrganizationID           string `json:"organization_id"`
	ExternalID               string `json:"external_id"`
	ResourceTypeSlug         string `json:"resource_type_slug"`
	Name                     string `json:"name"`
	Description              string `json:"description,omitempty"`
	ParentResourceID         string `json:"parent_resource_id,omitempty"`
	ParentResourceTypeSlug   string `json:"parent_resource_type_slug,omitempty"`
	ParentResourceExternalID string `json:"parent_resource_external_id,omitempty"`
}

// AuthorizationResourceUpdateRequest represents the request to update an authorization resource.
type AuthorizationResourceUpdateRequest struct {
	Name                     string `json:"name,omitempty"`
	Description              string `json:"description,omitempty"`
	ParentResourceID         string `json:"parent_resource_id,omitempty"`
	ParentResourceTypeSlug   string `json:"parent_resource_type_slug,omitempty"`
	ParentResourceExternalID string `json:"parent_resource_external_id,omitempty"`
}

// AuthorizationResourceListResponse represents the response from listing authorization resources.
type AuthorizationResourceListResponse struct {
	Data         []AuthorizationResource `json:"data"`
	ListMetadata ListMetadata            `json:"list_metadata"`
}

// UserRoleAssignmentResource represents the resource attached to a role assignment.
type UserRoleAssignmentResource struct {
	ID               string `json:"id"`
	ExternalID       string `json:"external_id"`
	ResourceTypeSlug string `json:"resource_type_slug"`
}

// UserRoleAssignmentRole represents the role attached to a role assignment.
type UserRoleAssignmentRole struct {
	Slug string `json:"slug"`
}

// UserRoleAssignment represents a WorkOS authorization role assignment.
type UserRoleAssignment struct {
	ID                       string                      `json:"id"`
	Object                   string                      `json:"object"`
	OrganizationMembershipID string                      `json:"organization_membership_id"`
	Role                     *UserRoleAssignmentRole     `json:"role"`
	Resource                 *UserRoleAssignmentResource `json:"resource"`
	CreatedAt                time.Time                   `json:"created_at"`
	UpdatedAt                time.Time                   `json:"updated_at"`
}

// AuthorizationRoleAssignmentCreateRequest represents the request to assign a role.
type AuthorizationRoleAssignmentCreateRequest struct {
	RoleSlug           string `json:"role_slug"`
	ResourceID         string `json:"resource_id,omitempty"`
	ResourceTypeSlug   string `json:"resource_type_slug,omitempty"`
	ResourceExternalID string `json:"resource_external_id,omitempty"`
}

// UserRoleAssignmentListResponse represents the response from listing role assignments.
type UserRoleAssignmentListResponse struct {
	Data         []UserRoleAssignment `json:"data"`
	ListMetadata ListMetadata         `json:"list_metadata"`
}

// CreateAuthorizationResource creates an authorization resource.
func (c *Client) CreateAuthorizationResource(ctx context.Context, req *AuthorizationResourceCreateRequest) (*AuthorizationResource, error) {
	var resource AuthorizationResource
	err := c.Post(ctx, "/authorization/resources", req, &resource)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization resource: %w", err)
	}
	return &resource, nil
}

// GetAuthorizationResource retrieves an authorization resource by ID.
func (c *Client) GetAuthorizationResource(ctx context.Context, id string) (*AuthorizationResource, error) {
	var resource AuthorizationResource
	err := c.Get(ctx, "/authorization/resources/"+url.PathEscape(id), &resource)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization resource: %w", err)
	}
	return &resource, nil
}

// UpdateAuthorizationResource updates an authorization resource by ID.
func (c *Client) UpdateAuthorizationResource(ctx context.Context, id string, req *AuthorizationResourceUpdateRequest) (*AuthorizationResource, error) {
	var resource AuthorizationResource
	err := c.Patch(ctx, "/authorization/resources/"+url.PathEscape(id), req, &resource)
	if err != nil {
		return nil, fmt.Errorf("failed to update authorization resource: %w", err)
	}
	return &resource, nil
}

// DeleteAuthorizationResource deletes an authorization resource by ID.
func (c *Client) DeleteAuthorizationResource(ctx context.Context, id string, cascadeDelete bool) error {
	path := "/authorization/resources/" + url.PathEscape(id)
	if cascadeDelete {
		params := url.Values{}
		params.Set("cascade_delete", "true")
		path = pathWithQuery(path, params)
	}

	err := c.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to delete authorization resource: %w", err)
	}
	return nil
}

// ListAuthorizationResources lists authorization resources with optional filters.
func (c *Client) ListAuthorizationResources(ctx context.Context, organizationID, resourceTypeSlug, externalID string) (*AuthorizationResourceListResponse, error) {
	var all AuthorizationResourceListResponse
	params := url.Values{}
	if organizationID != "" {
		params.Set("organization_id", organizationID)
	}
	if resourceTypeSlug != "" {
		params.Set("resource_type_slug", resourceTypeSlug)
	}
	if externalID != "" {
		params.Set("resource_external_id", externalID)
	}
	applyDefaultPagination(params)

	for {
		var page AuthorizationResourceListResponse
		err := c.Get(ctx, pathWithQuery("/authorization/resources", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list authorization resources: %w", err)
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

// AssignAuthorizationRole assigns a role to an organization membership on a resource.
func (c *Client) AssignAuthorizationRole(ctx context.Context, organizationMembershipID string, req *AuthorizationRoleAssignmentCreateRequest) (*UserRoleAssignment, error) {
	var assignment UserRoleAssignment
	err := c.Post(ctx, fmt.Sprintf("/authorization/organization_memberships/%s/role_assignments", url.PathEscape(organizationMembershipID)), req, &assignment)
	if err != nil {
		return nil, fmt.Errorf("failed to assign authorization role: %w", err)
	}
	return &assignment, nil
}

// ListAuthorizationRoleAssignments lists role assignments for an organization membership.
func (c *Client) ListAuthorizationRoleAssignments(ctx context.Context, organizationMembershipID string) (*UserRoleAssignmentListResponse, error) {
	var all UserRoleAssignmentListResponse
	params := url.Values{}
	applyDefaultPagination(params)
	path := fmt.Sprintf("/authorization/organization_memberships/%s/role_assignments", url.PathEscape(organizationMembershipID))

	for {
		var page UserRoleAssignmentListResponse
		err := c.Get(ctx, pathWithQuery(path, params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list authorization role assignments: %w", err)
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

// DeleteAuthorizationRoleAssignment removes a role assignment by ID.
func (c *Client) DeleteAuthorizationRoleAssignment(ctx context.Context, organizationMembershipID, roleAssignmentID string) error {
	err := c.Delete(ctx, fmt.Sprintf("/authorization/organization_memberships/%s/role_assignments/%s", url.PathEscape(organizationMembershipID), url.PathEscape(roleAssignmentID)))
	if err != nil {
		return fmt.Errorf("failed to delete authorization role assignment: %w", err)
	}
	return nil
}
