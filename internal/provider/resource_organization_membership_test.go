// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationMembershipResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tfacc")
	resourceName := "workos_organization_membership.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccOrganizationMembershipResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "user_id"),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOrganizationMembershipResource_withRole(t *testing.T) {
	rName := acctest.RandomWithPrefix("tfacc")
	resourceName := "workos_organization_membership.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationMembershipResourceConfig_withRole(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "role_slug", "admin"),
				),
			},
		},
	})
}

func testAccOrganizationMembershipResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = "%s-org"
}

resource "workos_user" "test" {
  email          = "%s@example.com"
  first_name     = "Test"
  last_name      = "User"
  email_verified = true
}

resource "workos_organization_membership" "test" {
  user_id         = workos_user.test.id
  organization_id = workos_organization.test.id
}
`, name, name)
}

func testAccOrganizationMembershipResourceConfig_withRole(name string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = "%s-org"
}

resource "workos_user" "test" {
  email          = "%s@example.com"
  first_name     = "Admin"
  last_name      = "User"
  email_verified = true
}

resource "workos_organization_membership" "test" {
  user_id         = workos_user.test.id
  organization_id = workos_organization.test.id
  role_slug       = "admin"
}
`, name, name)
}
