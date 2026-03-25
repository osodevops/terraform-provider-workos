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

func TestAccUserDataSource_ByExternalID(t *testing.T) {
	email := fmt.Sprintf("tf-acc-ds-extid-%d@example.com", time.Now().UnixNano())
	externalID := fmt.Sprintf("ext-ds-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_ByExternalID(email, externalID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.workos_user.test", "id"),
					resource.TestCheckResourceAttr("data.workos_user.test", "email", email),
					resource.TestCheckResourceAttr("data.workos_user.test", "external_id", externalID),
				),
			},
		},
	})
}

func testAccUserDataSourceConfig_ByExternalID(email, externalID string) string {
	return fmt.Sprintf(`
resource "workos_user" "test" {
  email       = %[1]q
  first_name  = "DS Test"
  last_name   = "ByExternalID"
  external_id = %[2]q
}

data "workos_user" "test" {
  external_id = workos_user.test.external_id
}
`, email, externalID)
}

func TestAccUserDataSource_WithMetadata(t *testing.T) {
	email := fmt.Sprintf("tf-acc-ds-meta-%d@example.com", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig_WithMetadata(email),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.workos_user.test", "id"),
					resource.TestCheckResourceAttr("data.workos_user.test", "metadata.team", "backend"),
				),
			},
		},
	})
}

func testAccUserDataSourceConfig_WithMetadata(email string) string {
	return fmt.Sprintf(`
resource "workos_user" "test" {
  email      = %[1]q
  first_name = "DS Test"
  last_name  = "Metadata"

  metadata = {
    team = "backend"
  }
}

data "workos_user" "test" {
  id = workos_user.test.id
}
`, email)
}
