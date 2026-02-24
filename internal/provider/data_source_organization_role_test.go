// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationRoleDataSource_BySlug(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	slug := fmt.Sprintf("test-role-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationRoleDataSourceConfigBySlug(orgName, slug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.workos_organization_role.test", "id",
						"workos_organization_role.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.workos_organization_role.test", "name",
						"workos_organization_role.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.workos_organization_role.test", "slug",
						"workos_organization_role.test", "slug",
					),
					resource.TestCheckResourceAttrSet("data.workos_organization_role.test", "created_at"),
				),
			},
		},
	})
}

func TestAccOrganizationRoleDataSource_ByID(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	slug := fmt.Sprintf("test-role-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationRoleDataSourceConfigByID(orgName, slug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.workos_organization_role.test", "id",
						"workos_organization_role.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.workos_organization_role.test", "name",
						"workos_organization_role.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.workos_organization_role.test", "slug",
						"workos_organization_role.test", "slug",
					),
				),
			},
		},
	})
}

func testAccOrganizationRoleDataSourceConfigBySlug(orgName, slug string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_organization_role" "test" {
  organization_id = workos_organization.test.id
  slug            = %[2]q
  name            = "Test Role"
}

data "workos_organization_role" "test" {
  organization_id = workos_organization.test.id
  slug            = workos_organization_role.test.slug
}
`, orgName, slug)
}

func testAccOrganizationRoleDataSourceConfigByID(orgName, slug string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_organization_role" "test" {
  organization_id = workos_organization.test.id
  slug            = %[2]q
  name            = "Test Role"
}

data "workos_organization_role" "test" {
  organization_id = workos_organization.test.id
  id              = workos_organization_role.test.id
}
`, orgName, slug)
}
