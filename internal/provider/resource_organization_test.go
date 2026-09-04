// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationResource_Basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccOrganizationResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization.test", "name", name),
					resource.TestCheckResourceAttrSet("workos_organization.test", "id"),
					resource.TestCheckResourceAttrSet("workos_organization.test", "created_at"),
					resource.TestCheckResourceAttrSet("workos_organization.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "workos_organization.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing
			{
				Config: testAccOrganizationResourceConfig(name + "-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization.test", "name", name+"-updated"),
				),
			},
		},
	})
}

func TestAccOrganizationResource_WithDomains(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	domain := fmt.Sprintf("test-%d.example.com", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with domains
			{
				Config: testAccOrganizationResourceConfigWithDomains(name, domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization.test", "name", name),
					resource.TestCheckResourceAttr("workos_organization.test", "domains.#", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "workos_organization.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOrganizationResource_WithDomainData(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	domain := fmt.Sprintf("test-%d.example.com", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationResourceConfigWithDomainData(name, domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization.test", "name", name),
					resource.TestCheckResourceAttr("workos_organization.test", "domain_data.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(
						"workos_organization.test",
						"domain_data.*",
						map[string]string{
							"domain": domain,
							"state":  "pending",
						},
					),
				),
			},
		},
	})
}

func TestAccOrganizationResource_DomainDataConflictsWithDomains(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	domain := fmt.Sprintf("test-%d.example.com", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccOrganizationResourceConfigWithConflictingDomains(name, domain),
				ExpectError: regexp.MustCompile(`(?i)cannot be configured together`),
			},
		},
	})
}

func TestAccOrganizationResource_DomainDataRejectsInvalidState(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	domain := fmt.Sprintf("test-%d.example.com", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccOrganizationResourceConfigWithDomainDataState(name, domain, "invalid"),
				ExpectError: regexp.MustCompile(`(?i)value must be one of`),
			},
		},
	})
}

func TestAccOrganizationResource_WithMetadataAndExternalID(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%d", time.Now().UnixNano())
	externalID := fmt.Sprintf("ext-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with external_id and metadata
			{
				Config: testAccOrganizationResourceConfigWithMetadata(name, externalID, "env", "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization.test", "name", name),
					resource.TestCheckResourceAttr("workos_organization.test", "external_id", externalID),
					resource.TestCheckResourceAttr("workos_organization.test", "metadata.env", "test"),
					resource.TestCheckResourceAttrSet("workos_organization.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "workos_organization.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update external_id and metadata
			{
				Config: testAccOrganizationResourceConfigWithMetadata(name, externalID+"-updated", "env", "production"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("workos_organization.test", "external_id", externalID+"-updated"),
					resource.TestCheckResourceAttr("workos_organization.test", "metadata.env", "production"),
				),
			},
		},
	})
}

func testAccOrganizationResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q
}
`, name)
}

func testAccOrganizationResourceConfigWithDomains(name, domain string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name    = %[1]q
  domains = [%[2]q]
}
`, name, domain)
}

func testAccOrganizationResourceConfigWithDomainData(name, domain string) string {
	return testAccOrganizationResourceConfigWithDomainDataState(name, domain, "")
}

func testAccOrganizationResourceConfigWithDomainDataState(name, domain, state string) string {
	stateConfig := ""
	if state != "" {
		stateConfig = fmt.Sprintf("\n    state  = %q", state)
	}

	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name = %[1]q

  domain_data {
    domain = %[2]q%[3]s
  }
}
`, name, domain, stateConfig)
}

func testAccOrganizationResourceConfigWithConflictingDomains(name, domain string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name    = %[1]q
  domains = [%[2]q]

  domain_data {
    domain = %[2]q
  }
}
`, name, domain)
}

func testAccOrganizationResourceConfigWithMetadata(name, externalID, metadataKey, metadataValue string) string {
	return fmt.Sprintf(`
resource "workos_organization" "test" {
  name        = %[1]q
  external_id = %[2]q
  metadata = {
    %[3]s = %[4]q
  }
}
`, name, externalID, metadataKey, metadataValue)
}
