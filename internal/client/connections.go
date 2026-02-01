// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/url"
)

// ConnectionCreateRequest represents the request to create a connection
type ConnectionCreateRequest struct {
	OrganizationID string `json:"organization_id"`
	ConnectionType string `json:"connection_type"`
	Name           string `json:"name,omitempty"`
}

// ConnectionUpdateRequest represents the request to update a connection
type ConnectionUpdateRequest struct {
	Name string `json:"name,omitempty"`
}

// ConnectionListResponse represents the response from listing connections
type ConnectionListResponse struct {
	Data         []Connection `json:"data"`
	ListMetadata ListMetadata `json:"list_metadata"`
}

// CreateConnection creates a new SSO connection
func (c *Client) CreateConnection(ctx context.Context, req *ConnectionCreateRequest) (*Connection, error) {
	var conn Connection
	err := c.Post(ctx, "/connections", req, &conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}
	return &conn, nil
}

// GetConnection retrieves a connection by ID
func (c *Client) GetConnection(ctx context.Context, id string) (*Connection, error) {
	var conn Connection
	err := c.Get(ctx, "/connections/"+id, &conn)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	return &conn, nil
}

// UpdateConnection updates an existing connection
func (c *Client) UpdateConnection(ctx context.Context, id string, req *ConnectionUpdateRequest) (*Connection, error) {
	var conn Connection
	err := c.Put(ctx, "/connections/"+id, req, &conn)
	if err != nil {
		return nil, fmt.Errorf("failed to update connection: %w", err)
	}
	return &conn, nil
}

// DeleteConnection deletes a connection by ID
func (c *Client) DeleteConnection(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/connections/"+id)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}
	return nil
}

// ListConnections lists all connections, optionally filtered by organization
func (c *Client) ListConnections(ctx context.Context, organizationID string) (*ConnectionListResponse, error) {
	path := "/connections"
	if organizationID != "" {
		params := url.Values{}
		params.Set("organization_id", organizationID)
		path = path + "?" + params.Encode()
	}

	var resp ConnectionListResponse
	err := c.Get(ctx, path, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}
	return &resp, nil
}

// GetConnectionByOrganizationAndType finds a connection by organization ID and type
func (c *Client) GetConnectionByOrganizationAndType(ctx context.Context, organizationID, connectionType string) (*Connection, error) {
	params := url.Values{}
	params.Set("organization_id", organizationID)
	params.Set("connection_type", connectionType)

	var resp ConnectionListResponse
	err := c.Get(ctx, "/connections?"+params.Encode(), &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to search connections: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, &APIError{
			StatusCode: 404,
			Message:    fmt.Sprintf("no connection found for organization %s with type %s", organizationID, connectionType),
		}
	}

	return &resp.Data[0], nil
}
