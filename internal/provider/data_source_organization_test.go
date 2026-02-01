// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationDataSource_ByID(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationDataSourceConfigByID(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.workos_organization.test", "id",
						"workos_organization.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.workos_organization.test", "name",
						"workos_organization.test", "name",
					),
					resource.TestCheckResourceAttrSet("data.workos_organization.test", "created_at"),
				),
			},
		},
	})
}

func TestAccOrganizationDataSource_ByDomain(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	domain := fmt.Sprintf("test-%d.example.com", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationDataSourceConfigByDomain(name, domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.workos_organization.test", "id",
						"workos_organization.test", "id",
					),
					resource.TestCheckResourceAttr("data.workos_organization.test", "name", name),
				),
			},
		},
	})
}

func testAccOrganizationDataSourceConfigByID(name string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}

data "workos_organization" "test" {
  id = workos_organization.test.id
}
`, name)
}

func testAccOrganizationDataSourceConfigByDomain(name, domain string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name    = %[1]q
  domains = [%[2]q]
}

data "workos_organization" "test" {
  domain = %[2]q

  depends_on = [workos_organization.test]
}
`, name, domain)
}
