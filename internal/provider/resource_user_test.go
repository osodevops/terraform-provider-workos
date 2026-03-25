// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tfacc")
	resourceName := "workos_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccUserResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s@example.com", rName)),
					resource.TestCheckResourceAttr(resourceName, "first_name", "Test"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "User"),
					resource.TestCheckResourceAttr(resourceName, "email_verified", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "password_hash", "password_hash_type"},
			},
			// Update and Read testing
			{
				Config: testAccUserResourceConfig_updated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s-updated@example.com", rName)),
					resource.TestCheckResourceAttr(resourceName, "first_name", "Updated"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "Name"),
				),
			},
		},
	})
}

func TestAccUserResource_withPassword(t *testing.T) {
	rName := acctest.RandomWithPrefix("tfacc")
	resourceName := "workos_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig_withPassword(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s@example.com", rName)),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccUserResource_withExternalIDAndMetadata(t *testing.T) {
	rName := acctest.RandomWithPrefix("tfacc")
	resourceName := "workos_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with external_id and metadata
			{
				Config: testAccUserResourceConfig_withExternalIDAndMetadata(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s@example.com", rName)),
					resource.TestCheckResourceAttr(resourceName, "external_id", fmt.Sprintf("ext-%s", rName)),
					resource.TestCheckResourceAttr(resourceName, "metadata.department", "Engineering"),
					resource.TestCheckResourceAttr(resourceName, "metadata.team", "Platform"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "password_hash", "password_hash_type"},
			},
			// Update metadata
			{
				Config: testAccUserResourceConfig_withExternalIDAndMetadataUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "external_id", fmt.Sprintf("ext-%s-updated", rName)),
					resource.TestCheckResourceAttr(resourceName, "metadata.department", "Product"),
					resource.TestCheckResourceAttr(resourceName, "metadata.role", "Lead"),
				),
			},
		},
	})
}

func TestAccUserResource_minimal(t *testing.T) {
	rName := acctest.RandomWithPrefix("tfacc")
	resourceName := "workos_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig_minimal(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s@example.com", rName)),
					resource.TestCheckResourceAttr(resourceName, "email_verified", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func testAccUserResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "workos_user" "test" {
  email          = "%s@example.com"
  first_name     = "Test"
  last_name      = "User"
  email_verified = true
}
`, name)
}

func testAccUserResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "workos_user" "test" {
  email          = "%s-updated@example.com"
  first_name     = "Updated"
  last_name      = "Name"
  email_verified = true
}
`, name)
}

func testAccUserResourceConfig_withPassword(name string) string {
	return fmt.Sprintf(`
resource "workos_user" "test" {
  email          = "%s@example.com"
  first_name     = "Password"
  last_name      = "User"
  password       = "SecureP@ssw0rd123!"
  email_verified = true
}
`, name)
}

func testAccUserResourceConfig_minimal(name string) string {
	return fmt.Sprintf(`
resource "workos_user" "test" {
  email = "%s@example.com"
}
`, name)
}

func testAccUserResourceConfig_withExternalIDAndMetadata(name string) string {
	return fmt.Sprintf(`
resource "workos_user" "test" {
  email          = "%s@example.com"
  first_name     = "Test"
  last_name      = "User"
  external_id    = "ext-%s"
  email_verified = true

  metadata = {
    department = "Engineering"
    team       = "Platform"
  }
}
`, name, name)
}

func testAccUserResourceConfig_withExternalIDAndMetadataUpdated(name string) string {
	return fmt.Sprintf(`
resource "workos_user" "test" {
  email          = "%s@example.com"
  first_name     = "Test"
  last_name      = "User"
  external_id    = "ext-%s-updated"
  email_verified = true

  metadata = {
    department = "Product"
    role       = "Lead"
  }
}
`, name, name)
}
