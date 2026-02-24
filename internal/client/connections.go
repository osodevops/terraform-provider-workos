// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/url"
)

// ConnectionListResponse represents the response from listing connections
type ConnectionListResponse struct {
	Data         []Connection `json:"data"`
	ListMetadata ListMetadata `json:"list_metadata"`
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
