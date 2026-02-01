// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConnectionDataSource_ByID(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionDataSourceConfigByID(orgName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.workos_connection.test", "id",
						"workos_connection.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.workos_connection.test", "connection_type",
						"workos_connection.test", "connection_type",
					),
					resource.TestCheckResourceAttrSet("data.workos_connection.test", "state"),
				),
			},
		},
	})
}

func TestAccConnectionDataSource_ByOrganizationAndType(t *testing.T) {
	orgName := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionDataSourceConfigByOrgAndType(orgName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.workos_connection.test", "id",
						"workos_connection.test", "id",
					),
					resource.TestCheckResourceAttr("data.workos_connection.test", "connection_type", "OktaSAML"),
				),
			},
		},
	})
}

func testAccConnectionDataSourceConfigByID(orgName string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_connection" "test" {
  organization_id = workos_organization.test.id
  connection_type = "OktaSAML"
  name            = "Test Connection"
}

data "workos_connection" "test" {
  id = workos_connection.test.id
}
`, orgName)
}

func testAccConnectionDataSourceConfigByOrgAndType(orgName string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

resource "workos_connection" "test" {
  organization_id = workos_organization.test.id
  connection_type = "OktaSAML"
  name            = "Test Connection"
}

data "workos_connection" "test" {
  organization_id = workos_organization.test.id
  connection_type = "OktaSAML"

  depends_on = [workos_connection.test]
}
`, orgName)
}
