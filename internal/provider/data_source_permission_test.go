// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPermissionDataSource_BySlug(t *testing.T) {
	slug := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPermissionDataSourceConfigBySlug(slug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.workos_permission.test", "id",
						"workos_permission.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.workos_permission.test", "name",
						"workos_permission.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.workos_permission.test", "slug",
						"workos_permission.test", "slug",
					),
					resource.TestCheckResourceAttrSet("data.workos_permission.test", "created_at"),
				),
			},
		},
	})
}

func testAccPermissionDataSourceConfigBySlug(slug string) string {
	return fmt.Sprintf(`
resource "workos_permission" "test" {
  slug = %[1]q
  name = "Test Permission"
}

data "workos_permission" "test" {
  slug = workos_permission.test.slug
}
`, slug)
}
