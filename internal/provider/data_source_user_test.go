// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource_ByID(t *testing.T) {
	email := fmt.Sprintf("tf-acc-ds-id-%d@example.com", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_ByID(email),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.workos_user.test", "id"),
					resource.TestCheckResourceAttr("data.workos_user.test", "email", email),
					resource.TestCheckResourceAttrSet("data.workos_user.test", "created_at"),
				),
			},
		},
	})
}

func TestAccUserDataSource_ByEmail(t *testing.T) {
	email := fmt.Sprintf("tf-acc-ds-email-%d@example.com", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_ByEmail(email),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.workos_user.test", "id"),
					resource.TestCheckResourceAttr("data.workos_user.test", "email", email),
				),
			},
		},
	})
}

func testAccUserDataSourceConfig_ByID(email string) string {
	return fmt.Sprintf(`
resource "workos_user" "test" {
  email      = %[1]q
  first_name = "DS Test"
  last_name  = "ByID"
}

data "workos_user" "test" {
  id = workos_user.test.id
}
`, email)
}

func testAccUserDataSourceConfig_ByEmail(email string) string {
	return fmt.Sprintf(`
resource "workos_user" "test" {
  email      = %[1]q
  first_name = "DS Test"
  last_name  = "ByEmail"
}

data "workos_user" "test" {
  email = workos_user.test.email
}
`, email)
}
