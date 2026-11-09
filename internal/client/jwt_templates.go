package client

import (
	"context"
	"fmt"
)

const jwtTemplatePath = "/user_management/jwt_template"

func (c *Client) GetJWTTemplate(ctx context.Context) (*JWTTemplate, error) {
	var template JWTTemplate
	if err := c.Get(ctx, jwtTemplatePath, &template); err != nil {
		return nil, fmt.Errorf("failed to get JWT template: %w", err)
	}
	return &template, nil
}

func (c *Client) UpdateJWTTemplate(ctx context.Context, req *JWTTemplateUpdateRequest) (*JWTTemplate, error) {
	var template JWTTemplate
	if err := c.Put(ctx, jwtTemplatePath, req, &template); err != nil {
		return nil, fmt.Errorf("failed to update JWT template: %w", err)
	}
	return &template, nil
}
