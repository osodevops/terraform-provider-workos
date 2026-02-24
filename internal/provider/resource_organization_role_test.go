// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccOrganizationRoleResource_Basic(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	slug := fmt.Sprintf("org-test-role-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccOrganizationRoleResourceConfig(orgName, slug, "Test Role", "A test role"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization_role.test", "slug", slug),
					resource.TestCheckResourceAttr("workos_organization_role.test", "name", "Test Role"),
					resource.TestCheckResourceAttr("workos_organization_role.test", "description", "A test role"),
					resource.TestCheckResourceAttrSet("workos_organization_role.test", "id"),
					resource.TestCheckResourceAttrSet("workos_organization_role.test", "organization_id"),
					resource.TestCheckResourceAttrSet("workos_organization_role.test", "type"),
					resource.TestCheckResourceAttrSet("workos_organization_role.test", "created_at"),
					resource.TestCheckResourceAttrSet("workos_organization_role.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "workos_organization_role.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["workos_organization_role.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: workos_organization_role.test")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["organization_id"], rs.Primary.Attributes["slug"]), nil
				},
			},
			// Update testing
			{
				Config: testAccOrganizationRoleResourceConfig(orgName, slug, "Updated Role", "An updated test role"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization_role.test", "name", "Updated Role"),
					resource.TestCheckResourceAttr("workos_organization_role.test", "description", "An updated test role"),
					resource.TestCheckResourceAttr("workos_organization_role.test", "slug", slug),
				),
			},
		},
	})
}

func TestAccOrganizationRoleResource_NoDescription(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	slug := fmt.Sprintf("org-test-role-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without description
			{
				Config: testAccOrganizationRoleResourceConfigNoDescription(orgName, slug, "Test Role"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization_role.test", "slug", slug),
					resource.TestCheckResourceAttr("workos_organization_role.test", "name", "Test Role"),
					resource.TestCheckResourceAttrSet("workos_organization_role.test", "id"),
				),
			},
		},
	})
}

func testAccOrganizationRoleResourceConfig(orgName, slug, name, description string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_organization_role" "test" {
  organization_id = workos_organization.test.id
  slug            = %[2]q
  name            = %[3]q
  description     = %[4]q
}
`, orgName, slug, name, description)
}

func testAccOrganizationRoleResourceConfigNoDescription(orgName, slug, name string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_organization_role" "test" {
  organization_id = workos_organization.test.id
  slug            = %[2]q
  name            = %[3]q
}
`, orgName, slug, name)
}
