package client

import "time"

type JWTTemplate struct {
	Object    string    `json:"object"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
