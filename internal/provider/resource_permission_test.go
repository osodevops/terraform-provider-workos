// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPermissionResource_Basic(t *testing.T) {
	slug := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPermissionResourceConfig(slug, "Test Permission", "A test permission"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_permission.test", "slug", slug),
					resource.TestCheckResourceAttr("workos_permission.test", "name", "Test Permission"),
					resource.TestCheckResourceAttr("workos_permission.test", "description", "A test permission"),
					resource.TestCheckResourceAttrSet("workos_permission.test", "id"),
					resource.TestCheckResourceAttrSet("workos_permission.test", "created_at"),
					resource.TestCheckResourceAttrSet("workos_permission.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "workos_permission.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     slug,
			},
			// Update testing
			{
				Config: testAccPermissionResourceConfig(slug, "Updated Permission", "An updated test permission"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_permission.test", "name", "Updated Permission"),
					resource.TestCheckResourceAttr("workos_permission.test", "description", "An updated test permission"),
					resource.TestCheckResourceAttr("workos_permission.test", "slug", slug),
				),
			},
		},
	})
}

func TestAccPermissionResource_NoDescription(t *testing.T) {
	slug := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without description
			{
				Config: testAccPermissionResourceConfigNoDescription(slug, "Test Permission"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_permission.test", "slug", slug),
					resource.TestCheckResourceAttr("workos_permission.test", "name", "Test Permission"),
					resource.TestCheckResourceAttrSet("workos_permission.test", "id"),
				),
			},
		},
	})
}

func testAccPermissionResourceConfig(slug, name, description string) string {
	return fmt.Sprintf(`
resource "workos_permission" "test" {
  slug        = %[1]q
  name        = %[2]q
  description = %[3]q
}
`, slug, name, description)
}

func testAccPermissionResourceConfigNoDescription(slug, name string) string {
	return fmt.Sprintf(`
resource "workos_permission" "test" {
  slug = %[1]q
  name = %[2]q
}
`, slug, name)
}
