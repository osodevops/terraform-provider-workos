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

func TestAccOrganizationRolePermissionResource_Basic(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	roleSlug := fmt.Sprintf("org-test-role-%d", time.Now().UnixNano())
	permSlug := fmt.Sprintf("tf-acc-perm-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccOrganizationRolePermissionResourceConfig(orgName, roleSlug, permSlug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("workos_organization_role_permission.test", "id"),
					resource.TestCheckResourceAttrSet("workos_organization_role_permission.test", "organization_id"),
					resource.TestCheckResourceAttr("workos_organization_role_permission.test", "role_slug", roleSlug),
					resource.TestCheckResourceAttr("workos_organization_role_permission.test", "permission", permSlug),
				),
			},
			// ImportState testing
			{
				ResourceName:      "workos_organization_role_permission.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["workos_organization_role_permission.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: workos_organization_role_permission.test")
					}
					return fmt.Sprintf("%s/%s/%s",
						rs.Primary.Attributes["organization_id"],
						rs.Primary.Attributes["role_slug"],
						rs.Primary.Attributes["permission"],
					), nil
				},
			},
		},
	})
}

func testAccOrganizationRolePermissionResourceConfig(orgName, roleSlug, permSlug string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_organization_role" "test" {
  organization_id = workos_organization.test.id
  slug            = %[2]q
  name            = "Test Role"
}

resource "workos_permission" "test" {
  slug = %[3]q
  name = "Test Permission"
}

resource "workos_organization_role_permission" "test" {
  organization_id = workos_organization.test.id
  role_slug       = workos_organization_role.test.slug
  permission      = workos_permission.test.slug
}
`, orgName, roleSlug, permSlug)
}
