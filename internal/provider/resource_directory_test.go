// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDirectoryResource_Basic(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDirectoryResourceConfig(orgName, "Test Directory", "okta scim v2.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_directory.test", "name", "Test Directory"),
					resource.TestCheckResourceAttr("workos_directory.test", "type", "okta scim v2.0"),
					resource.TestCheckResourceAttrSet("workos_directory.test", "id"),
					resource.TestCheckResourceAttrSet("workos_directory.test", "organization_id"),
					resource.TestCheckResourceAttrSet("workos_directory.test", "state"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "workos_directory.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"bearer_token"},
			},
			// Update testing - change name
			{
				Config: testAccDirectoryResourceConfig(orgName, "Updated Directory", "okta scim v2.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_directory.test", "name", "Updated Directory"),
				),
			},
		},
	})
}

func TestAccDirectoryResource_AzureSCIM(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDirectoryResourceConfig(orgName, "Azure Directory", "azure scim v2.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_directory.test", "type", "azure scim v2.0"),
				),
			},
		},
	})
}

func TestAccDirectoryResource_GenericSCIM(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDirectoryResourceConfig(orgName, "Generic SCIM Directory", "generic scim v2.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_directory.test", "type", "generic scim v2.0"),
				),
			},
		},
	})
}

func testAccDirectoryResourceConfig(orgName, dirName, dirType string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_directory" "test" {
  organization_id = workos_organization.test.id
  name            = %[2]q
  type            = %[3]q
}
`, orgName, dirName, dirType)
}
