// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDirectoryDataSource_ByID(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDirectoryDataSourceConfigByID(orgName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.workos_directory.test", "id",
						"workos_directory.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.workos_directory.test", "type",
						"workos_directory.test", "type",
					),
					resource.TestCheckResourceAttrSet("data.workos_directory.test", "state"),
				),
			},
		},
	})
}

func TestAccDirectoryDataSource_ByOrganization(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDirectoryDataSourceConfigByOrg(orgName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.workos_directory.test", "id",
						"workos_directory.test", "id",
					),
				),
			},
		},
	})
}

func testAccDirectoryDataSourceConfigByID(orgName string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_directory" "test" {
  organization_id = workos_organization.test.id
  name            = "Test Directory"
  type            = "okta scim v2.0"
}

data "workos_directory" "test" {
  id = workos_directory.test.id
}
`, orgName)
}

func testAccDirectoryDataSourceConfigByOrg(orgName string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_directory" "test" {
  organization_id = workos_organization.test.id
  name            = "Test Directory"
  type            = "okta scim v2.0"
}

data "workos_directory" "test" {
  organization_id = workos_organization.test.id

  depends_on = [workos_directory.test]
}
`, orgName)
}
