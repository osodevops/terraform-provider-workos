package provider

import "strings"

func splitCompositeID(id string, parts int) []string {
	values := strings.Split(id, "/")
	if len(values) != parts {
		return nil
	}

	for _, value := range values {
		if value == "" {
			return nil
		}
	}

	return values
}
