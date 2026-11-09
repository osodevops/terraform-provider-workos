package client

type AuthKitOAuthResourceCreateRequest struct {
	URI     string `json:"uri"`
	Default *bool  `json:"default,omitempty"`
}
