package client

import "time"

type AuthKitOAuthResource struct {
	ID        string    `json:"id"`
	Object    string    `json:"object"`
	URI       string    `json:"uri"`
	Default   bool      `json:"default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
