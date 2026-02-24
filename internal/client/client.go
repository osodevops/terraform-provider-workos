// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const (
	// DefaultBaseURL is the default WorkOS API base URL
	DefaultBaseURL = "https://api.workos.com"

	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second

	// MaxRetries is the maximum number of retry attempts for rate-limited requests
	MaxRetries = 3

	// BaseRetryDelay is the base delay for exponential backoff
	BaseRetryDelay = 1 * time.Second

	// MaxRetryDelay is the maximum delay between retries
	MaxRetryDelay = 30 * time.Second
)

// Client is the WorkOS API client
type Client struct {
	httpClient *http.Client
	apiKey     string
	clientID   string
	baseURL    string
}

// NewClient creates a new WorkOS API client
func NewClient(apiKey, clientID, baseURL string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		apiKey:   apiKey,
		clientID: clientID,
		baseURL:  baseURL,
	}, nil
}

// doRequest performs an HTTP request with automatic retry on rate limiting
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		// Reset body reader for retries
		if body != nil {
			jsonBody, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(jsonBody)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "terraform-provider-workos")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		// Handle rate limiting (429)
		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt == MaxRetries {
				return resp, nil // Return the 429 response on final attempt
			}

			// Calculate retry delay
			delay := c.calculateRetryDelay(resp, attempt)

			// Close the response body before retrying
			resp.Body.Close()

			// Wait before retrying
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// calculateRetryDelay determines how long to wait before retrying
func (c *Client) calculateRetryDelay(resp *http.Response, attempt int) time.Duration {
	// Check for Retry-After header
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		// Try to parse as seconds
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}
		// Try to parse as HTTP date
		if t, err := http.ParseTime(retryAfter); err == nil {
			return time.Until(t)
		}
	}

	// Exponential backoff with jitter
	delay := time.Duration(math.Pow(2, float64(attempt))) * BaseRetryDelay
	if delay > MaxRetryDelay {
		delay = MaxRetryDelay
	}

	// Add jitter (up to 25% of delay)
	jitter := time.Duration(rand.Int63n(int64(delay / 4)))
	return delay + jitter
}

// parseResponse parses an HTTP response into the target struct
func (c *Client) parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error responses
	if resp.StatusCode >= 400 {
		return parseAPIError(resp.StatusCode, bodyBytes)
	}

	// Parse successful response
	if target != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, target); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return c.parseResponse(resp, result)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.parseResponse(resp, result)
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	return c.parseResponse(resp, result)
}

// Patch performs a PATCH request
func (c *Client) Patch(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPatch, path, body)
	if err != nil {
		return err
	}
	return c.parseResponse(resp, result)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.parseResponse(resp, nil)
}
