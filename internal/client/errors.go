// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Common error types for WorkOS API responses
var (
	ErrNotFound       = errors.New("resource not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrBadRequest     = errors.New("bad request")
	ErrConflict       = errors.New("conflict")
	ErrRateLimited    = errors.New("rate limited")
	ErrInternalServer = errors.New("internal server error")
)

// APIError represents a WorkOS API error response
type APIError struct {
	StatusCode int
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Errors     []ValidationError `json:"errors,omitempty"`
}

// ValidationError represents a field-level validation error
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("WorkOS API error (HTTP %d)", e.StatusCode))

	if e.Code != "" {
		sb.WriteString(fmt.Sprintf(" [%s]", e.Code))
	}

	if e.Message != "" {
		sb.WriteString(": ")
		sb.WriteString(e.Message)
	}

	if len(e.Errors) > 0 {
		sb.WriteString("\nValidation errors:")
		for _, ve := range e.Errors {
			sb.WriteString(fmt.Sprintf("\n  - %s", ve.Field))
			if ve.Code != "" {
				sb.WriteString(fmt.Sprintf(" [%s]", ve.Code))
			}
			if ve.Message != "" {
				sb.WriteString(fmt.Sprintf(": %s", ve.Message))
			}
		}
	}

	return sb.String()
}

// Unwrap returns the underlying error type for errors.Is() support
func (e *APIError) Unwrap() error {
	switch e.StatusCode {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusBadRequest:
		return ErrBadRequest
	case http.StatusConflict:
		return ErrConflict
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		if e.StatusCode >= 500 {
			return ErrInternalServer
		}
		return nil
	}
}

// parseAPIError parses an error response from the WorkOS API
func parseAPIError(statusCode int, body []byte) error {
	apiErr := &APIError{
		StatusCode: statusCode,
	}

	// Try to parse JSON error response
	if len(body) > 0 {
		if err := json.Unmarshal(body, apiErr); err != nil {
			// If JSON parsing fails, use raw body as message
			apiErr.Message = string(body)
		}
	}

	// Set default message if none provided
	if apiErr.Message == "" {
		switch statusCode {
		case http.StatusNotFound:
			apiErr.Message = "The requested resource was not found"
		case http.StatusUnauthorized:
			apiErr.Message = "Invalid API key or authentication failed"
		case http.StatusForbidden:
			apiErr.Message = "Access denied to this resource"
		case http.StatusBadRequest:
			apiErr.Message = "The request was invalid or malformed"
		case http.StatusConflict:
			apiErr.Message = "The resource already exists or conflicts with existing data"
		case http.StatusTooManyRequests:
			apiErr.Message = "Rate limit exceeded, please retry later"
		case http.StatusUnprocessableEntity:
			apiErr.Message = "The request was well-formed but contained invalid data"
		default:
			if statusCode >= 500 {
				apiErr.Message = "WorkOS service encountered an internal error"
			} else {
				apiErr.Message = fmt.Sprintf("Unexpected error (HTTP %d)", statusCode)
			}
		}
	}

	return apiErr
}

// IsNotFound returns true if the error indicates a resource was not found
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized returns true if the error indicates an authentication failure
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden returns true if the error indicates an authorization failure
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsBadRequest returns true if the error indicates a bad request
func IsBadRequest(err error) bool {
	return errors.Is(err, ErrBadRequest)
}

// IsConflict returns true if the error indicates a conflict
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsRateLimited returns true if the error indicates rate limiting
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

// IsInternalServerError returns true if the error indicates a server error
func IsInternalServerError(err error) bool {
	return errors.Is(err, ErrInternalServer)
}
