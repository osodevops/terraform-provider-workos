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
	err := c.Get(ctx, "/connections/"+url.PathEscape(id), &conn)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	return &conn, nil
}

// DeleteConnection deletes a connection by ID
func (c *Client) DeleteConnection(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/connections/"+url.PathEscape(id))
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}
	return nil
}

// ListConnections lists all connections, optionally filtered by organization
func (c *Client) ListConnections(ctx context.Context, organizationID string) (*ConnectionListResponse, error) {
	var all ConnectionListResponse
	params := url.Values{}
	if organizationID != "" {
		params.Set("organization_id", organizationID)
	}
	applyDefaultPagination(params)

	for {
		var page ConnectionListResponse
		err := c.Get(ctx, pathWithQuery("/connections", params), &page)
		if err != nil {
			return nil, fmt.Errorf("failed to list connections: %w", err)
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

// GetConnectionByOrganizationAndType finds a connection by organization ID and type
func (c *Client) GetConnectionByOrganizationAndType(ctx context.Context, organizationID, connectionType string) (*Connection, error) {
	resp, err := c.ListConnections(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to search connections: %w", err)
	}

	matches := make([]Connection, 0)
	for _, conn := range resp.Data {
		if conn.ConnectionType == connectionType {
			matches = append(matches, conn)
		}
	}

	if len(matches) == 0 {
		return nil, &APIError{
			StatusCode: 404,
			Message:    fmt.Sprintf("no connection found for organization %s with type %s", organizationID, connectionType),
		}
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous connection lookup: found %d connections for organization %s with type %s", len(matches), organizationID, connectionType)
	}

	return &matches[0], nil
}
