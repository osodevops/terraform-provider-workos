// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConnectionResource_Basic(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccConnectionResourceConfig(orgName, "OktaSAML", "Test Okta Connection"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_connection.test", "connection_type", "OktaSAML"),
					resource.TestCheckResourceAttr("workos_connection.test", "name", "Test Okta Connection"),
					resource.TestCheckResourceAttrSet("workos_connection.test", "id"),
					resource.TestCheckResourceAttrSet("workos_connection.test", "organization_id"),
					resource.TestCheckResourceAttrSet("workos_connection.test", "state"),
					resource.TestCheckResourceAttrSet("workos_connection.test", "status"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "workos_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing - change name
			{
				Config: testAccConnectionResourceConfig(orgName, "OktaSAML", "Updated Okta Connection"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_connection.test", "name", "Updated Okta Connection"),
				),
			},
		},
	})
}

func TestAccConnectionResource_GoogleOAuth(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionResourceConfig(orgName, "GoogleOAuth", "Google Login"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_connection.test", "connection_type", "GoogleOAuth"),
					resource.TestCheckResourceAttr("workos_connection.test", "name", "Google Login"),
				),
			},
		},
	})
}

func TestAccConnectionResource_GenericSAML(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionResourceConfig(orgName, "GenericSAML", "Generic SAML"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_connection.test", "connection_type", "GenericSAML"),
				),
			},
		},
	})
}

func testAccConnectionResourceConfig(orgName, connectionType, connectionName string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_connection" "test" {
  organization_id = workos_organization.test.id
  connection_type = %[2]q
  name            = %[3]q
}
`, orgName, connectionType, connectionName)
}
