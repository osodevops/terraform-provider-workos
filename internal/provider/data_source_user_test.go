// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource_ByID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_ByID,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.workos_user.test", "id"),
					resource.TestCheckResourceAttrSet("data.workos_user.test", "email"),
					resource.TestCheckResourceAttrSet("data.workos_user.test", "created_at"),
				),
			},
		},
	})
}

func TestAccUserDataSource_ByEmail(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_ByEmail,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.workos_user.test", "id"),
					resource.TestCheckResourceAttr("data.workos_user.test", "email", "test@example.com"),
				),
			},
		},
	})
}

const testAccUserDataSourceConfig_ByID = `
data "workos_user" "test" {
  id = "user_01HXYZ..."
}
`

const testAccUserDataSourceConfig_ByEmail = `
data "workos_user" "test" {
  email = "test@example.com"
}
`
