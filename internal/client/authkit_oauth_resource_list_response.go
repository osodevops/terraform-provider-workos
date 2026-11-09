package client

type AuthKitOAuthResourceListResponse struct {
	Data         []AuthKitOAuthResource `json:"data"`
	ListMetadata ListMetadata           `json:"list_metadata"`
}
