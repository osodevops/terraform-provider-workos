package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	jwtTemplateResourceName = "workos_jwt_template.this"
	rolesTemplate           = `{"urn:px:roles": {{organization_membership.roles}}}`
	emailTemplate           = `{"urn:px:email": "{{user.email}}"}`
)

func TestAccJWTTemplateResource_MockAPI(t *testing.T) {
	api, server := newJWTTemplateMockAPI(t, "{}")

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			if api.content() != emailTemplate {
				return fmt.Errorf("expected destroy to leave the template in place, got %q", api.content())
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccJWTTemplateConfig(server.URL, rolesTemplate),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(jwtTemplateResourceName, "id", "jwt_template"),
					resource.TestCheckResourceAttr(jwtTemplateResourceName, "content", rolesTemplate),
					resource.TestCheckResourceAttrSet(jwtTemplateResourceName, "updated_at"),
				),
			},
			{
				Config: testAccJWTTemplateConfig(server.URL, emailTemplate),
				Check:  resource.TestCheckResourceAttr(jwtTemplateResourceName, "content", emailTemplate),
			},
			{
				ResourceName:      jwtTemplateResourceName,
				ImportState:       true,
				ImportStateId:     "jwt_template",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccJWTTemplateResource_DetectsDrift(t *testing.T) {
	api, server := newJWTTemplateMockAPI(t, "{}")

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccJWTTemplateConfig(server.URL, rolesTemplate),
			},
			{
				PreConfig: func() {
					api.mu.Lock()
					api.template.Content = emailTemplate
					api.mu.Unlock()
				},
				Config:             testAccJWTTemplateConfig(server.URL, rolesTemplate),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccJWTTemplateResource_RejectsInvalidTemplate(t *testing.T) {
	api, server := newJWTTemplateMockAPI(t, rolesTemplate)

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccJWTTemplateConfig(server.URL, "not a template"),
				ExpectError: regexp.MustCompile(`(?s)Could not set JWT template.*422`),
			},
		},
	})

	if api.content() != rolesTemplate {
		t.Fatalf("expected the current template to stay in place, got %q", api.content())
	}
}

func testAccJWTTemplateConfig(baseURL, content string) string {
	return fmt.Sprintf(`
provider "workos" {
  api_key  = "sk_test"
  base_url = %[1]q
}

resource "workos_jwt_template" "this" {
  content = %[2]q
}
`, baseURL, content)
}
