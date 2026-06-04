package client

import "net/url"

const defaultPageLimit = "100"

func applyDefaultPagination(params url.Values) {
	if params.Get("limit") == "" {
		params.Set("limit", defaultPageLimit)
	}
}

func pathWithQuery(path string, params url.Values) string {
	if len(params) == 0 {
		return path
	}

	return path + "?" + params.Encode()
}
