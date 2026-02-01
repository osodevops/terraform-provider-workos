// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhookResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tfacc")
	resourceName := "workos_webhook.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccWebhookResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "url", fmt.Sprintf("https://%s.example.com/webhooks", rName)),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "events.#", "3"),
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
				ImportStateVerifyIgnore: []string{"secret"}, // Secret is not returned by API
			},
			// Update and Read testing
			{
				Config: testAccWebhookResourceConfig_updated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "url", fmt.Sprintf("https://%s-updated.example.com/webhooks", rName)),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "events.#", "5"),
				),
			},
		},
	})
}

func TestAccWebhookResource_allEvents(t *testing.T) {
	rName := acctest.RandomWithPrefix("tfacc")
	resourceName := "workos_webhook.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookResourceConfig_allEvents(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func testAccWebhookResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "workos_webhook" "test" {
  url     = "https://%s.example.com/webhooks"
  secret  = "whsec_test_secret_key_12345"
  enabled = true

  events = [
    "user.created",
    "user.updated",
    "user.deleted",
  ]
}
`, name)
}

func testAccWebhookResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "workos_webhook" "test" {
  url     = "https://%s-updated.example.com/webhooks"
  secret  = "whsec_test_secret_key_12345"
  enabled = false

  events = [
    "user.created",
    "user.updated",
    "user.deleted",
    "dsync.user.created",
    "dsync.user.deleted",
  ]
}
`, name)
}

func testAccWebhookResourceConfig_allEvents(name string) string {
	return fmt.Sprintf(`
resource "workos_webhook" "test" {
  url     = "https://%s.example.com/webhooks"
  secret  = "whsec_test_secret_key_all_events"
  enabled = true

  events = [
    "authentication.email_verification_succeeded",
    "authentication.magic_auth_failed",
    "authentication.magic_auth_succeeded",
    "authentication.mfa_succeeded",
    "authentication.oauth_failed",
    "authentication.oauth_succeeded",
    "authentication.password_failed",
    "authentication.password_succeeded",
    "authentication.sso_failed",
    "authentication.sso_succeeded",
    "connection.activated",
    "connection.deactivated",
    "connection.deleted",
    "dsync.activated",
    "dsync.deleted",
    "dsync.group.created",
    "dsync.group.deleted",
    "dsync.group.updated",
    "dsync.user.created",
    "dsync.user.deleted",
    "dsync.user.updated",
    "organization.created",
    "organization.deleted",
    "organization.updated",
    "organization_domain.verification_failed",
    "organization_domain.verified",
    "organization_membership.added",
    "organization_membership.removed",
    "organization_membership.updated",
    "role.created",
    "role.deleted",
    "role.updated",
    "session.created",
    "user.created",
    "user.deleted",
    "user.updated",
  ]
}
`, name)
}
