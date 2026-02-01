// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import "time"

// Organization represents a WorkOS Organization
type Organization struct {
	ID                               string    `json:"id"`
	Object                           string    `json:"object"`
	Name                             string    `json:"name"`
	AllowProfilesOutsideOrganization bool      `json:"allow_profiles_outside_organization"`
	Domains                          []Domain  `json:"domains,omitempty"`
	CreatedAt                        time.Time `json:"created_at"`
	UpdatedAt                        time.Time `json:"updated_at"`
}

// Domain represents a domain associated with an organization
type Domain struct {
	ID             string `json:"id"`
	Object         string `json:"object"`
	Domain         string `json:"domain"`
	State          string `json:"state"`
	OrganizationID string `json:"organization_id"`
	VerificationType string `json:"verification_type,omitempty"`
}

// OrganizationCreateRequest represents the request to create an organization
type OrganizationCreateRequest struct {
	Name                             string   `json:"name"`
	AllowProfilesOutsideOrganization bool     `json:"allow_profiles_outside_organization,omitempty"`
	DomainData                       []DomainData `json:"domain_data,omitempty"`
}

// DomainData represents domain data for organization creation/update
type DomainData struct {
	Domain string `json:"domain"`
	State  string `json:"state,omitempty"`
}

// OrganizationUpdateRequest represents the request to update an organization
type OrganizationUpdateRequest struct {
	Name                             string       `json:"name,omitempty"`
	AllowProfilesOutsideOrganization *bool        `json:"allow_profiles_outside_organization,omitempty"`
	DomainData                       []DomainData `json:"domain_data,omitempty"`
}

// OrganizationListResponse represents the response from listing organizations
type OrganizationListResponse struct {
	Data       []Organization `json:"data"`
	ListMetadata ListMetadata `json:"list_metadata"`
}

// ListMetadata contains pagination information
type ListMetadata struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

// Connection represents a WorkOS SSO Connection
type Connection struct {
	ID               string           `json:"id"`
	Object           string           `json:"object"`
	OrganizationID   string           `json:"organization_id"`
	ConnectionType   string           `json:"connection_type"`
	Name             string           `json:"name"`
	State            string           `json:"state"`
	Status           string           `json:"status"`
	SAMLConfiguration *SAMLConfiguration `json:"saml,omitempty"`
	OIDCConfiguration *OIDCConfiguration `json:"oidc,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// SAMLConfiguration represents SAML-specific configuration
type SAMLConfiguration struct {
	IdPEntityID   string `json:"idp_entity_id"`
	IdPSSOURL     string `json:"idp_sso_url"`
	IdPCertificate string `json:"idp_certificate"`
	SPEntityID    string `json:"sp_entity_id"`
	SPACSURL      string `json:"sp_acs_url"`
}

// OIDCConfiguration represents OIDC-specific configuration
type OIDCConfiguration struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Issuer       string `json:"issuer"`
	RedirectURI  string `json:"redirect_uri"`
}

// Directory represents a WorkOS Directory Sync directory
type Directory struct {
	ID             string    `json:"id"`
	Object         string    `json:"object"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	State          string    `json:"state"`
	BearerToken    string    `json:"bearer_token,omitempty"`
	Endpoint       string    `json:"endpoint,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DirectoryUser represents a user synced from a directory
type DirectoryUser struct {
	ID             string            `json:"id"`
	Object         string            `json:"object"`
	DirectoryID    string            `json:"directory_id"`
	OrganizationID string            `json:"organization_id"`
	IdpID          string            `json:"idp_id"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	Email          string            `json:"email"`
	Username       string            `json:"username,omitempty"`
	State          string            `json:"state"`
	CustomAttributes map[string]interface{} `json:"custom_attributes,omitempty"`
	RawAttributes  map[string]interface{} `json:"raw_attributes,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// DirectoryGroup represents a group synced from a directory
type DirectoryGroup struct {
	ID             string    `json:"id"`
	Object         string    `json:"object"`
	DirectoryID    string    `json:"directory_id"`
	OrganizationID string    `json:"organization_id"`
	IdpID          string    `json:"idp_id"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Webhook represents a WorkOS Webhook
type Webhook struct {
	ID        string   `json:"id"`
	Object    string   `json:"object"`
	URL       string   `json:"url"`
	Secret    string   `json:"secret,omitempty"`
	Enabled   bool     `json:"enabled"`
	Events    []string `json:"events"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// WebhookCreateRequest represents the request to create a webhook
type WebhookCreateRequest struct {
	URL     string   `json:"url"`
	Secret  string   `json:"secret"`
	Enabled bool     `json:"enabled"`
	Events  []string `json:"events"`
}

// WebhookUpdateRequest represents the request to update a webhook
type WebhookUpdateRequest struct {
	URL     string   `json:"url,omitempty"`
	Enabled *bool    `json:"enabled,omitempty"`
	Events  []string `json:"events,omitempty"`
}

// User represents a WorkOS AuthKit User
type User struct {
	ID             string    `json:"id"`
	Object         string    `json:"object"`
	Email          string    `json:"email"`
	EmailVerified  bool      `json:"email_verified"`
	FirstName      string    `json:"first_name,omitempty"`
	LastName       string    `json:"last_name,omitempty"`
	ProfilePictureURL string `json:"profile_picture_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserCreateRequest represents the request to create a user
type UserCreateRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password,omitempty"`
	PasswordHash  string `json:"password_hash,omitempty"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
}

// UserUpdateRequest represents the request to update a user
type UserUpdateRequest struct {
	Email         string `json:"email,omitempty"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`
}

// OrganizationMembership represents a user's membership in an organization
type OrganizationMembership struct {
	ID             string    `json:"id"`
	Object         string    `json:"object"`
	UserID         string    `json:"user_id"`
	OrganizationID string    `json:"organization_id"`
	RoleSlug       string    `json:"role_slug,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// OrganizationMembershipCreateRequest represents the request to create a membership
type OrganizationMembershipCreateRequest struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	RoleSlug       string `json:"role_slug,omitempty"`
}
