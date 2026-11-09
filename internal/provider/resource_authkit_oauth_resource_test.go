package provider

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

const authKitOAuthResourceName = "workos_authkit_oauth_resource.mcp"

func TestAccAuthKitOAuthResourceResource_MockAPI(t *testing.T) {
	api, server := newAuthKitOAuthResourceMockAPI(t)

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			if api.count() != 0 {
				return fmt.Errorf("expected destroy to remove the resource indicator, %d remain", api.count())
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccAuthKitOAuthResourceConfig(server.URL, "https://mcp.example.com", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(authKitOAuthResourceName, "uri", "https://mcp.example.com"),
					resource.TestCheckResourceAttr(authKitOAuthResourceName, "default", "false"),
					resource.TestCheckResourceAttrSet(authKitOAuthResourceName, "id"),
					resource.TestCheckResourceAttrSet(authKitOAuthResourceName, "created_at"),
				),
			},
			{
				ResourceName:      authKitOAuthResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      authKitOAuthResourceName,
				ImportState:       true,
				ImportStateId:     "https://mcp.example.com",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAuthKitOAuthResourceResource_DefaultIsSetAtCreation(t *testing.T) {
	api, server := newAuthKitOAuthResourceMockAPI(t)

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAuthKitOAuthResourceConfig(server.URL, "https://mcp.example.com", "true"),
				Check:  resource.TestCheckResourceAttr(authKitOAuthResourceName, "default", "true"),
			},
			{
				Config: testAccAuthKitOAuthResourceConfig(server.URL, "https://mcp.example.com", "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(authKitOAuthResourceName, "default", "false"),
					func(*terraform.State) error {
						if api.deletions() != 1 {
							return fmt.Errorf("expected changing default to replace the indicator, saw %d deletions", api.deletions())
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccAuthKitOAuthResourceResource_RefusesToAdoptExisting(t *testing.T) {
	api, server := newAuthKitOAuthResourceMockAPI(t)

	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	existing := &client.AuthKitOAuthResource{
		ID:        "authkit_oauth_resource_existing",
		Object:    "authkit_oauth_resource",
		URI:       "https://mcp.example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}
	api.resources[existing.ID] = existing

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAuthKitOAuthResourceConfig(server.URL, existing.URI, ""),
				ExpectError: regexp.MustCompile(`(?s)already\s+registered.*terraform\s+import`),
			},
			{
				Config:        testAccAuthKitOAuthResourceConfig(server.URL, existing.URI, ""),
				ResourceName:  authKitOAuthResourceName,
				ImportState:   true,
				ImportStateId: existing.URI,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 || states[0].ID != existing.ID {
						return fmt.Errorf("expected to import %q, got %v", existing.ID, states)
					}
					return nil
				},
			},
		},
	})

	if api.deletions() != 0 {
		t.Fatalf("expected the pre-existing indicator to be left alone, saw %d deletions", api.deletions())
	}
}

func testAccAuthKitOAuthResourceConfig(baseURL, uri, isDefault string) string {
	defaultLine := ""
	if isDefault != "" {
		defaultLine = "default = " + isDefault
	}

	return fmt.Sprintf(`
provider "workos" {
  api_key  = "sk_test"
  base_url = %[1]q
}

resource "workos_authkit_oauth_resource" "mcp" {
  uri = %[2]q
  %[3]s
}
`, baseURL, uri, defaultLine)
}
